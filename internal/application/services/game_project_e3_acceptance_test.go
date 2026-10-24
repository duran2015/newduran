package services

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/gamepackage"
	"control-panel/internal/domain/gameproject"
)

func e3Draft() GameProjectDraft {
	return GameProjectDraft{
		Metrics: []GameMetricDefinition{
			{ID: "pressure", Name: "监管压力", Type: "number", Initial: float64(65)},
			{ID: "reputation", Name: "公众信任", Type: "number", Initial: float64(50)},
		},
		Actions: []GameActionDefinition{{
			ID: "respond", Name: "回应事件", Effects: []GameEffectDefinition{{Operation: "increment", Target: "reputation", Value: float64(1)}}, Emits: "leak",
		}},
		Events: []GameProjectEvent{
			{ID: "leak", Name: "资料曝光", Category: "route", DisplayContent: "内部材料被媒体掌握", ConditionMode: "all", Conditions: []GameConditionDefinition{{Target: "pressure", Operator: "gte", Value: float64(60)}}, Effects: []GameEffectDefinition{{Operation: "decrement", Target: "reputation", Value: float64(10)}}, AdvancesTime: true, TimeAdvance: 1, Priority: 20, Once: true, ExclusiveGroup: "opening", NextEventIDs: []string{"inquiry"}},
			{ID: "inquiry", Name: "监管问询", Category: "system", DisplayContent: "监管机构要求说明", ConditionMode: "all", Conditions: []GameConditionDefinition{{Target: "pressure", Operator: "gte", Value: float64(50)}}, Effects: []GameEffectDefinition{{Operation: "increment", Target: "pressure", Value: float64(5)}}, Priority: 10, CooldownTurns: 2, ExclusiveGroup: "opening"},
		},
		Endings: []GameEndingDefinition{{ID: "survive", Name: "维持经营"}},
		NPCs:    []GameProjectNPC{},
		Scenes:  []GameProjectScene{},
	}
}

func saveE3Draft(t *testing.T, service *GameProjectService, draft GameProjectDraft) *GameProjectDetail {
	t.Helper()
	project := createGameProjectForTest(t, service)
	updated, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	return updated
}

func TestE3PreviewIsTenantScopedDeterministicAndDoesNotPersist(t *testing.T) {
	service, db := gameProjectTestService(t)
	project := saveE3Draft(t, service, e3Draft())

	var before gameproject.Project
	if err := db.First(&before, "id = ?", project.Project.ID).Error; err != nil {
		t.Fatal(err)
	}
	beforeJSON, _ := json.Marshal(before.Draft)
	var packageCountBefore, releaseCountBefore int64
	db.Model(&gamepackage.Package{}).Count(&packageCountBefore)
	db.Model(&gameproject.Release{}).Count(&releaseCountBefore)

	input := PreviewGameEventsInput{State: map[string]any{"pressure": float64(65), "reputation": float64(50)}, CurrentTurn: 4}
	first, err := service.PreviewEvents("tenant-a", project.Project.ID, "", input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.PreviewEvents("tenant-a", project.Project.ID, "", input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("相同输入必须得到相同试演结果\nfirst=%#v\nsecond=%#v", first, second)
	}
	if len(first.SelectedIDs) != 1 || first.SelectedIDs[0] != "leak" {
		t.Fatalf("互斥组应稳定选择高优先级事件: %#v", first.SelectedIDs)
	}
	if _, err := service.PreviewEvents("tenant-b", project.Project.ID, "", input); err == nil {
		t.Fatal("其他租户不应能读取或试演该游戏")
	}

	var after gameproject.Project
	if err := db.First(&after, "id = ?", project.Project.ID).Error; err != nil {
		t.Fatal(err)
	}
	afterJSON, _ := json.Marshal(after.Draft)
	var packageCountAfter, releaseCountAfter int64
	db.Model(&gamepackage.Package{}).Count(&packageCountAfter)
	db.Model(&gameproject.Release{}).Count(&releaseCountAfter)
	if before.Revision != after.Revision || string(beforeJSON) != string(afterJSON) || packageCountBefore != packageCountAfter || releaseCountBefore != releaseCountAfter {
		t.Fatalf("事件试演不得写入草稿、发布版本或运行记录")
	}
}

func TestE3OnceAndCooldownPreventDuplicateTrigger(t *testing.T) {
	service, _ := gameProjectTestService(t)
	project := saveE3Draft(t, service, e3Draft())

	once, err := service.PreviewEvents("tenant-a", project.Project.ID, "leak", PreviewGameEventsInput{CurrentTurn: 6, OccurredEventIDs: []string{"leak"}, TriggeredAtTurn: map[string]int{"leak": 2}})
	if err != nil {
		t.Fatal(err)
	}
	if len(once.Items) != 1 || once.Items[0].Triggered || once.Items[0].Selected {
		t.Fatalf("整局一次事件不应重复触发: %#v", once.Items)
	}

	for _, tc := range []struct {
		turn int
		want bool
	}{{4, false}, {5, false}, {6, true}} {
		result, err := service.PreviewEvents("tenant-a", project.Project.ID, "inquiry", PreviewGameEventsInput{CurrentTurn: tc.turn, OccurredEventIDs: []string{"inquiry"}, TriggeredAtTurn: map[string]int{"inquiry": 3}})
		if err != nil {
			t.Fatal(err)
		}
		if got := result.Items[0].Selected; got != tc.want {
			t.Fatalf("回合 %d 冷却判断=%v, want %v (%#v)", tc.turn, got, tc.want, result.Items[0].Reasons)
		}
	}
}

func TestE3PublishRejectsOutOfGameParticipantsAndUnknownEffects(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*GameProjectDraft)
		want   string
	}{
		{"参与人物不在本游戏", func(d *GameProjectDraft) { d.Events[0].ParticipantCharacterIDs = []string{"outsider"} }, "不存在的游戏角色"},
		{"影响未声明数值", func(d *GameProjectDraft) { d.Events[0].Effects[0].Target = "secret_metric" }, "unknown world state"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, _ := gameProjectTestService(t)
			draft := e3Draft()
			tc.mutate(&draft)
			project := saveE3Draft(t, service, draft)
			_, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "0.1.0"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("发布应拒绝越界事件，err=%v", err)
			}
		})
	}
}

func TestE3NormalNPCCannotGainTurnAdvancePermission(t *testing.T) {
	service, db := gameProjectTestService(t)
	npc := agent.AgentConfig{Name: "cfo", TenantID: "tenant-a", ContentHash: "v1", SystemPrompt: "finance"}
	if err := db.Create(&npc).Error; err != nil {
		t.Fatal(err)
	}
	draft := e3Draft()
	draft.NPCs = []GameProjectNPC{{ID: "cfo", AgentID: npc.ID, Role: "财务负责人", CharacterType: GameCharacterNPC, Permissions: []string{"advance_turn"}}}
	project := createGameProjectForTest(t, service)
	_, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 1, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	if err == nil {
		t.Fatal("普通 NPC 不应获得推进回合权限")
	}
}

func TestE3PublishedEventSnapshotDoesNotChangeWithDraft(t *testing.T) {
	service, _ := gameProjectTestService(t)
	project := saveE3Draft(t, service, e3Draft())
	release, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.packages.Get("tenant-a", release.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	if published.Manifest.Content == nil || published.Manifest.Content.StoryEvents[0].DisplayContent != "内部材料被媒体掌握" {
		t.Fatalf("发布版本没有冻结完整事件: %#v", published.Manifest.Content)
	}

	draft := project.Draft
	draft.Events[0].DisplayContent = "草稿中的新文案"
	draft.Events[0].Effects[0].Value = float64(30)
	if _, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft}); err != nil {
		t.Fatal(err)
	}
	publishedAgain, err := service.packages.Get("tenant-a", release.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	got := publishedAgain.Manifest.Content.StoryEvents[0]
	if got.DisplayContent != "内部材料被媒体掌握" || got.Effects[0].Value != float64(10) {
		t.Fatalf("修改草稿不应污染旧发布版本: %#v", got)
	}
}
