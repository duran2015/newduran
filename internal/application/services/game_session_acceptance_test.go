package services

import (
	"errors"
	"testing"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/gamepackage"
	"control-panel/internal/domain/gameproject"
	rundomain "control-panel/internal/domain/run"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type gameSessionFixture struct {
	service  *GameSessionService
	projects *GameProjectService
	db       *gorm.DB
	project  *GameProjectDetail
	director GameProjectNPC
	npc      GameProjectNPC
}

func newGameSessionAcceptanceFixture(t *testing.T, events []GameProjectEvent) *gameSessionFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&agent.AgentConfig{}, &gamepackage.Package{}, &gameproject.Project{}, &gameproject.Release{}, &gameproject.Session{},
		&rundomain.Run{}, &rundomain.RunAgent{}, &rundomain.CapabilityBinding{}, &rundomain.StateSchema{},
		&rundomain.RunState{}, &rundomain.RunStateChange{}, &rundomain.RunActivity{},
	); err != nil {
		t.Fatal(err)
	}
	directorAgent := agent.AgentConfig{Name: "session-director", TenantID: "tenant-a", ContentHash: "director-v1", SystemPrompt: "direct"}
	npcAgent := agent.AgentConfig{Name: "session-cfo", TenantID: "tenant-a", ContentHash: "cfo-v1", SystemPrompt: "finance"}
	if err := db.Create(&directorAgent).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&npcAgent).Error; err != nil {
		t.Fatal(err)
	}
	packages := NewGamePackageService(db)
	projects := NewGameProjectService(db, packages)
	project, err := projects.Create("tenant-a", "creator", CreateGameProjectInput{Key: "session.test", Name: "正式试玩验收", Description: "用于检查回合与留痕", Genre: "社会策略", PlayGoal: "稳定推进"})
	if err != nil {
		t.Fatal(err)
	}
	director := GameProjectNPC{ID: "director", AgentID: directorAgent.ID, DisplayName: "闻峥", Role: "导演", CharacterType: GameCharacterDirector, Permissions: []string{"advance_turn"}}
	npc := GameProjectNPC{ID: "cfo", AgentID: npcAgent.ID, DisplayName: "莫云岑", Role: "财务负责人", CharacterType: GameCharacterNPC}
	draft := GameProjectDraft{
		Metrics: []GameMetricDefinition{{ID: "pressure", Name: "监管压力", Type: "number", Initial: float64(60)}, {ID: "trust", Name: "公众信任", Type: "number", Initial: float64(50)}},
		Actions: []GameActionDefinition{{ID: "wait", Name: "等待", Effects: []GameEffectDefinition{}, Emits: ""}},
		Events:  events,
		Endings: []GameEndingDefinition{{ID: "done", Name: "结束"}},
		NPCs:    []GameProjectNPC{director, npc},
		Scenes:  []GameProjectScene{},
	}
	project, err = projects.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	if err != nil {
		t.Fatal(err)
	}
	return &gameSessionFixture{service: NewGameSessionService(db, projects, NewRunService(db)), projects: projects, db: db, project: project, director: director, npc: npc}
}

func alwaysEvent(id string, priority int, group string, once bool, cooldown int, delta float64) GameProjectEvent {
	return GameProjectEvent{
		ID: id, Name: id, Category: "story", DisplayContent: "事件 " + id,
		ParticipantCharacterIDs: []string{"cfo"}, Effects: []GameEffectDefinition{{Operation: "increment", Target: "pressure", Value: delta}},
		AdvancesTime: true, TimeAdvance: 1, Priority: priority, Once: once, CooldownTurns: cooldown, ExclusiveGroup: group,
	}
}

func TestGameSessionPinsDraftAndKeepsTenantBoundary(t *testing.T) {
	f := newGameSessionAcceptanceFixture(t, []GameProjectEvent{alwaysEvent("opening", 10, "", false, 0, 5)})
	session, err := f.service.Create("tenant-a", f.project.Project.ID, CreateGameSessionInput{CreatedBy: "pm"})
	if err != nil {
		t.Fatal(err)
	}
	if session.CurrentTurn != 0 || session.Status != rundomain.StatusRunning || session.State["pressure"] != float64(60) {
		t.Fatalf("试玩初始状态不正确: %#v", session)
	}
	if _, err := f.service.Get("tenant-b", f.project.Project.ID, session.ID); !errors.Is(err, ErrGameSessionNotFound) {
		t.Fatalf("其他租户不应读取试玩局: %v", err)
	}

	latest, err := f.projects.Get("tenant-a", f.project.Project.ID)
	if err != nil {
		t.Fatal(err)
	}
	draft := latest.Draft
	draft.Metrics[0].Initial = float64(999)
	draft.Events[0].Effects[0].Value = float64(100)
	if _, err := f.projects.Update("tenant-a", latest.Project.ID, "editor", UpdateGameProjectInput{Revision: latest.Project.Revision, Name: latest.Project.Name, Description: latest.Project.Description, Genre: latest.Project.Genre, PlayGoal: latest.Project.PlayGoal, Draft: &draft}); err != nil {
		t.Fatal(err)
	}
	advanced, err := f.service.Advance("tenant-a", f.project.Project.ID, session.ID, AdvanceGameSessionInput{RequestKey: "turn-1", ActorCharacterID: f.director.ID})
	if err != nil {
		t.Fatal(err)
	}
	if advanced.State["pressure"] != float64(65) {
		t.Fatalf("已开始试玩应继续使用开始时的规则，got=%v", advanced.State["pressure"])
	}
}

func TestGameSessionAdvanceIsDeterministicAtomicAndIdempotent(t *testing.T) {
	f := newGameSessionAcceptanceFixture(t, []GameProjectEvent{
		alwaysEvent("lower", 10, "opening", false, 0, 50),
		alwaysEvent("higher", 20, "opening", false, 0, 5),
	})
	session, err := f.service.Create("tenant-a", f.project.Project.ID, CreateGameSessionInput{CreatedBy: "pm"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := f.service.Advance("tenant-a", f.project.Project.ID, session.ID, AdvanceGameSessionInput{RequestKey: "same-request", ActorCharacterID: f.director.ID})
	if err != nil {
		t.Fatal(err)
	}
	if first.CurrentTurn != 1 || first.State["pressure"] != float64(65) || first.LatestEvent == nil || first.LatestEvent.EventID != "higher" {
		t.Fatalf("互斥组应稳定选择高优先级事件: %#v", first)
	}
	replayed, err := f.service.Advance("tenant-a", f.project.Project.ID, session.ID, AdvanceGameSessionInput{RequestKey: "same-request", ActorCharacterID: f.director.ID})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.CurrentTurn != first.CurrentTurn || replayed.State["pressure"] != first.State["pressure"] {
		t.Fatalf("同一推进请求不应重复结算: first=%#v replayed=%#v", first, replayed)
	}
	var changes, turns int64
	f.db.Model(&rundomain.RunStateChange{}).Where("run_id = ? AND reason = ?", session.RunID, "导演推进回合").Count(&changes)
	f.db.Model(&rundomain.RunActivity{}).Where("run_id = ? AND kind = ?", session.RunID, "game.turn.advanced").Count(&turns)
	if changes != 1 || turns != 1 {
		t.Fatalf("一次推进必须只留下一个状态版本和一条时间线，changes=%d turns=%d", changes, turns)
	}
}

func TestGameSessionComposesEffectsFromMultipleSelectedEvents(t *testing.T) {
	f := newGameSessionAcceptanceFixture(t, []GameProjectEvent{
		alwaysEvent("first", 20, "", false, 0, 5),
		alwaysEvent("second", 10, "", false, 0, 7),
	})
	session, err := f.service.Create("tenant-a", f.project.Project.ID, CreateGameSessionInput{CreatedBy: "pm"})
	if err != nil {
		t.Fatal(err)
	}
	advanced, err := f.service.Advance("tenant-a", f.project.Project.ID, session.ID, AdvanceGameSessionInput{RequestKey: "both-events", ActorCharacterID: f.director.ID})
	if err != nil {
		t.Fatal(err)
	}
	// 两条都被本回合选中时，后一条影响必须基于前一条结算后的状态，
	// 不能各自从回合开始值计算后互相覆盖。
	if advanced.State["pressure"] != float64(72) {
		t.Fatalf("同回合多事件影响没有累积结算，got=%v want=72", advanced.State["pressure"])
	}
}

func TestGameSessionOnceCooldownAndMutexSurviveAcrossTurns(t *testing.T) {
	t.Run("整局一次", func(t *testing.T) {
		f := newGameSessionAcceptanceFixture(t, []GameProjectEvent{alwaysEvent("once", 10, "", true, 0, 5)})
		s, err := f.service.Create("tenant-a", f.project.Project.ID, CreateGameSessionInput{CreatedBy: "pm"})
		if err != nil {
			t.Fatal(err)
		}
		first, err := f.service.Advance("tenant-a", f.project.Project.ID, s.ID, AdvanceGameSessionInput{RequestKey: "one", ActorCharacterID: f.director.ID})
		if err != nil {
			t.Fatal(err)
		}
		second, err := f.service.Advance("tenant-a", f.project.Project.ID, s.ID, AdvanceGameSessionInput{RequestKey: "two", ActorCharacterID: f.director.ID})
		if err != nil {
			t.Fatal(err)
		}
		if first.State["pressure"] != float64(65) || second.State["pressure"] != float64(65) {
			t.Fatalf("整局一次事件不应重复结算: first=%v second=%v", first.State, second.State)
		}
	})

	t.Run("冷却回合", func(t *testing.T) {
		f := newGameSessionAcceptanceFixture(t, []GameProjectEvent{alwaysEvent("cooldown", 10, "", false, 2, 5)})
		s, err := f.service.Create("tenant-a", f.project.Project.ID, CreateGameSessionInput{CreatedBy: "pm"})
		if err != nil {
			t.Fatal(err)
		}
		want := []float64{65, 65, 65, 70}
		for i, expected := range want {
			got, err := f.service.Advance("tenant-a", f.project.Project.ID, s.ID, AdvanceGameSessionInput{RequestKey: string(rune('a' + i)), ActorCharacterID: f.director.ID})
			if err != nil {
				t.Fatal(err)
			}
			if got.State["pressure"] != expected {
				t.Fatalf("第 %d 次推进冷却结果=%v，want=%v", i+1, got.State["pressure"], expected)
			}
		}
	})
}

func TestGameSessionLifecycleDirectorGuardAndTimeline(t *testing.T) {
	f := newGameSessionAcceptanceFixture(t, []GameProjectEvent{alwaysEvent("event", 10, "", false, 0, 5)})
	session, err := f.service.Create("tenant-a", f.project.Project.ID, CreateGameSessionInput{CreatedBy: "pm"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.Advance("tenant-a", f.project.Project.ID, session.ID, AdvanceGameSessionInput{RequestKey: "npc", ActorCharacterID: f.npc.ID}); !errors.Is(err, ErrGameDirectorRequired) {
		t.Fatalf("普通 NPC 不应推进试玩: %v", err)
	}
	paused, err := f.service.Pause("tenant-a", f.project.Project.ID, session.ID, f.director.ID)
	if err != nil || paused.Status != rundomain.StatusPaused {
		t.Fatalf("暂停失败: %#v err=%v", paused, err)
	}
	if _, err := f.service.Advance("tenant-a", f.project.Project.ID, session.ID, AdvanceGameSessionInput{RequestKey: "while-paused", ActorCharacterID: f.director.ID}); err == nil {
		t.Fatal("暂停后仍然推进了试玩")
	}
	resumed, err := f.service.Resume("tenant-a", f.project.Project.ID, session.ID, f.director.ID)
	if err != nil || resumed.Status != rundomain.StatusRunning {
		t.Fatalf("继续失败: %#v err=%v", resumed, err)
	}
	if _, err := f.service.Advance("tenant-a", f.project.Project.ID, session.ID, AdvanceGameSessionInput{RequestKey: "after-resume", ActorCharacterID: f.director.ID}); err != nil {
		t.Fatal(err)
	}
	ended, err := f.service.End("tenant-a", f.project.Project.ID, session.ID, f.director.ID)
	if err != nil || ended.Status != rundomain.StatusCompleted {
		t.Fatalf("结束失败: %#v err=%v", ended, err)
	}
	if _, err := f.service.Advance("tenant-a", f.project.Project.ID, session.ID, AdvanceGameSessionInput{RequestKey: "after-end", ActorCharacterID: f.director.ID}); err == nil {
		t.Fatal("结束后仍然推进了试玩")
	}
	final, err := f.service.Get("tenant-a", f.project.Project.ID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, activity := range final.Timeline {
		kinds[activity.Kind]++
	}
	for _, kind := range []string{"game.session.started", "game.session.paused", "game.session.running", "game.turn.advanced", "game.session.completed"} {
		if kinds[kind] != 1 {
			t.Fatalf("全过程时间线缺少 %s: %#v", kind, kinds)
		}
	}
}

func TestGameSessionEndingEventCompletesAndExplainsTheSession(t *testing.T) {
	ending := alwaysEvent("ending", 10, "", false, 0, 5)
	ending.EndsGame = true
	ending.EndReason = "监管压力触发结局"
	f := newGameSessionAcceptanceFixture(t, []GameProjectEvent{ending})
	session, err := f.service.Create("tenant-a", f.project.Project.ID, CreateGameSessionInput{CreatedBy: "pm"})
	if err != nil {
		t.Fatal(err)
	}
	finished, err := f.service.Advance("tenant-a", f.project.Project.ID, session.ID, AdvanceGameSessionInput{RequestKey: "ending-turn", ActorCharacterID: f.director.ID})
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != rundomain.StatusCompleted {
		t.Fatalf("结局事件没有结束试玩: status=%s", finished.Status)
	}
	found := false
	for _, activity := range finished.Timeline {
		if activity.Kind == "game.session.completed" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("结局事件虽然结束了试玩，但时间线没有留下结束记录")
	}
}
