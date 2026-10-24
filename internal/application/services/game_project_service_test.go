package services

import (
	"fmt"
	"testing"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/agentrelation"
	"control-panel/internal/domain/gamepackage"
	"control-panel/internal/domain/gameproject"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func gameProjectTestService(t *testing.T) (*GameProjectService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&agent.AgentConfig{}, &agent.AgentKnowledgeDataset{}, &agent.DeploymentSnapshot{}, &agentrelation.AgentRelation{}, &gamepackage.Package{}, &gameproject.Project{}, &gameproject.Release{}); err != nil {
		t.Fatal(err)
	}
	packages := NewGamePackageService(db)
	return NewGameProjectService(db, packages), db
}

func createGameProjectForTest(t *testing.T, service *GameProjectService) *GameProjectDetail {
	t.Helper()
	project, err := service.Create("tenant-a", "creator", CreateGameProjectInput{Key: "speeding.demo", Name: "Speeding", Description: "财富、权力与社会系统的互动游戏", Genre: "社会策略", PlayGoal: "在三十天内完成财富处置并保持控制权"})
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func completeGameDraft(agentID uint64) GameProjectDraft {
	return GameProjectDraft{
		Metrics: []GameMetricDefinition{{ID: "wealth", Name: "可支配财富", Type: "number", Initial: float64(100)}},
		Actions: []GameActionDefinition{{ID: "sell_asset", Name: "出售资产", Effects: []GameEffectDefinition{{Operation: "decrement", Target: "wealth", Value: float64(10)}}, Emits: "asset_sold"}},
		Endings: []GameEndingDefinition{{ID: "survived", Name: "维持控制权"}},
		NPCs:    []GameProjectNPC{{ID: "cfo", AgentID: agentID, Role: "财务负责人", Permissions: []string{"propose"}, KnownInformation: []string{"真实现金余额"}}},
		Events:  []GameProjectEvent{{ID: "asset_sold", Name: "资产出售完成", ParticipantIDs: []string{"cfo"}, AdvancesTime: true}},
		Scenes:  []GameProjectScene{{ID: "boardroom", Name: "董事会议室", BackgroundRef: "asset://background/boardroom", MusicRef: "asset://music/tension"}},
	}
}

func TestGameProjectCreateEditAndPublishImmutableRelease(t *testing.T) {
	service, db := gameProjectTestService(t)
	agentRow := agent.AgentConfig{Name: "speeding-cfo", TenantID: "tenant-a", ContentHash: "test", SystemPrompt: "act as CFO"}
	if err := db.Create(&agentRow).Error; err != nil {
		t.Fatal(err)
	}
	project := createGameProjectForTest(t, service)
	if project.Project.Revision != 1 || len(project.Draft.Metrics) != 0 {
		t.Fatalf("unexpected new project: %#v", project)
	}
	updated, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 1, Name: "Speeding", Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: ptrDraft(completeGameDraft(agentRow.ID))})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Project.Revision != 2 || updated.Draft.NPCs[0].AgentID != agentRow.ID {
		t.Fatalf("draft not saved: %#v", updated)
	}
	release, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.packages.Get("tenant-a", release.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	if published.Manifest.Content == nil || published.Manifest.Content.ProjectRevision != 2 || published.Manifest.Content.NPCs[0].AgentID != agentRow.ID {
		t.Fatalf("package missing project snapshot: %#v", published.Manifest.Content)
	}

	// Editing continues in the project and must not mutate the published package.
	draft := updated.Draft
	draft.Metrics[0].Initial = float64(80)
	_, err = service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 2, Name: "Speeding", Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	if err != nil {
		t.Fatal(err)
	}
	publishedAgain, err := service.packages.Get("tenant-a", release.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	if publishedAgain.Manifest.Registry.Metrics[0].Initial != float64(100) {
		t.Fatalf("published package was changed through project editing: %#v", publishedAgain.Manifest.Registry.Metrics[0].Initial)
	}
}

func TestHistoricalReleaseCanReplaceDraftOrCreateSeparateProjectWithoutMutatingRelease(t *testing.T) {
	service, db := gameProjectTestService(t)
	agentRow := agent.AgentConfig{Name: "director", TenantID: "tenant-a", ContentHash: "v1", SystemPrompt: "director"}
	require.NoError(t, db.Create(&agentRow).Error)
	project := createGameProjectForTest(t, service)
	draft := completeGameDraft(agentRow.ID)
	draft.NPCs[0].CharacterType = GameCharacterDirector
	updated, err := service.Update("tenant-a", project.Project.ID, "pm", UpdateGameProjectInput{Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	require.NoError(t, err)
	release, err := service.Publish("tenant-a", project.Project.ID, "pm", PublishGameProjectInput{Version: "1.0.0", ReleaseNotes: "首个稳定版本"})
	require.NoError(t, err)
	changed := updated.Draft
	changed.Metrics[0].Initial = float64(999)
	current, err := service.Update("tenant-a", project.Project.ID, "pm", UpdateGameProjectInput{Revision: updated.Project.Revision, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &changed})
	require.NoError(t, err)
	restored, err := service.RestoreRelease("tenant-a", project.Project.ID, release.ID, "pm", RestoreGameReleaseInput{Mode: "replace-draft", Revision: current.Project.Revision})
	require.NoError(t, err)
	require.Equal(t, float64(100), restored.Draft.Metrics[0].Initial)
	copy, err := service.RestoreRelease("tenant-a", project.Project.ID, release.ID, "pm", RestoreGameReleaseInput{Mode: "new-project", Key: "speeding.history", Name: "历史版本副本"})
	require.NoError(t, err)
	require.NotEqual(t, project.Project.ID, copy.Project.ID)
	require.Equal(t, float64(100), copy.Draft.Metrics[0].Initial)
	published, err := service.packages.Get("tenant-a", release.PackageID)
	require.NoError(t, err)
	require.Equal(t, float64(100), published.Manifest.Registry.Metrics[0].Initial)
}

func TestGameProjectRejectsUnknownAgentAsSoonAsItIsSelected(t *testing.T) {
	service, _ := gameProjectTestService(t)
	project := createGameProjectForTest(t, service)
	draft := completeGameDraft(999)
	if _, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 1, Name: "Speeding", Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft}); err == nil {
		t.Fatal("expected an unavailable Agent reference to be rejected immediately")
	}
	draft.NPCs[0].AgentID = 0
	if _, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 1, Name: "Speeding", Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft}); err != nil {
		t.Fatalf("an unfinished character without a selected Agent should remain saveable: %v", err)
	}
}

func TestGameProjectCannotPublishIncompleteDraft(t *testing.T) {
	service, _ := gameProjectTestService(t)
	project := createGameProjectForTest(t, service)
	if _, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "0.1.0"}); err == nil {
		t.Fatal("expected incomplete project to be rejected")
	}
}

func TestGameProjectCharacterBindingPreviewAndFrozenReferences(t *testing.T) {
	service, db := gameProjectTestService(t)
	director := agent.AgentConfig{Name: "speeding-director", TenantID: "tenant-a", ContentHash: "director-v3", SystemPrompt: "direct", DeploymentStatus: "running", PersonalityTemplateName: "strategist", PersonalityTemplateVersion: 3}
	cfo := agent.AgentConfig{Name: "speeding-cfo", TenantID: "tenant-a", ContentHash: "cfo-v2", SystemPrompt: "finance"}
	if err := db.Create(&director).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&cfo).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&agent.AgentKnowledgeDataset{AgentID: cfo.ID, DatasetID: "finance-ledger"}).Error; err != nil {
		t.Fatal(err)
	}
	relation := agentrelation.AgentRelation{TenantID: "tenant-a", Scope: "global", SourceAgentID: director.ID, TargetAgentID: cfo.ID, RelationType: "delegates_to", AllowedActions: []string{"delegate"}, ContextPolicy: "summary_only", DeliveryPolicy: "async", Enabled: true}
	if err := db.Create(&relation).Error; err != nil {
		t.Fatal(err)
	}
	project := createGameProjectForTest(t, service)
	draft := completeGameDraft(cfo.ID)
	draft.NPCs = []GameProjectNPC{
		{ID: "director", AgentID: director.ID, DisplayName: "闻峥", Role: "危机总指挥", CharacterType: GameCharacterDirector, VisibleMetricIDs: []string{"wealth"}, ExecutableActionIDs: []string{"sell_asset"}, RelationIDs: []uint64{relation.ID}, Permissions: []string{"advance_turn"}},
		{ID: "cfo", AgentID: cfo.ID, DisplayName: "莫云岑", Role: "首席财务官", Responsibilities: "评估资金安全", CharacterType: GameCharacterNPC, VisibleMetricIDs: []string{"wealth"}, AdvisoryActionIDs: []string{"sell_asset"}, KnownInformation: []string{"真实现金余额"}, KnowledgeRefs: []string{"finance-ledger"}, RelationIDs: []uint64{relation.ID}},
	}
	updated, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 1, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.PreviewCharacterContext("tenant-a", project.Project.ID, "cfo")
	if err != nil {
		t.Fatal(err)
	}
	if preview.DisplayName != "莫云岑" || preview.AgentAvailable || len(preview.VisibleWorldState) != 1 || len(preview.Relationships) != 1 {
		t.Fatalf("unexpected character preview: %#v", preview)
	}
	release, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "0.2.0"})
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.packages.Get("tenant-a", release.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	got := published.Manifest.Content.NPCs[1]
	if got.AgentVersion == nil || got.AgentVersion.ContentHash != "cfo-v2" || len(got.AgentVersion.Relations) != 1 || got.KnowledgeRefs[0] != "finance-ledger" {
		t.Fatalf("character references were not frozen: %#v", got)
	}
	if updated.Project.Revision != 2 {
		t.Fatalf("unexpected revision: %d", updated.Project.Revision)
	}
}

func TestGameProjectRejectsSecondDirectorAndNPCAdvanceTurn(t *testing.T) {
	service, db := gameProjectTestService(t)
	a := agent.AgentConfig{Name: "a", TenantID: "tenant-a", ContentHash: "a", SystemPrompt: "a"}
	b := agent.AgentConfig{Name: "b", TenantID: "tenant-a", ContentHash: "b", SystemPrompt: "b"}
	_ = db.Create(&a).Error
	_ = db.Create(&b).Error
	project := createGameProjectForTest(t, service)
	draft := completeGameDraft(a.ID)
	draft.NPCs = []GameProjectNPC{{ID: "a", AgentID: a.ID, Role: "A", CharacterType: GameCharacterDirector}, {ID: "b", AgentID: b.ID, Role: "B", CharacterType: GameCharacterDirector}}
	if _, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 1, Name: project.Project.Name, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft}); err == nil {
		t.Fatal("expected two directors to be rejected")
	}
	draft.NPCs = []GameProjectNPC{{ID: "a", AgentID: a.ID, Role: "A", CharacterType: GameCharacterNPC, Permissions: []string{"advance_turn"}}}
	if _, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 1, Name: project.Project.Name, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft}); err == nil {
		t.Fatal("expected npc advance permission to be rejected")
	}
}

func TestGameProjectPreviewActionExplainsConditionsAndDoesNotPersistState(t *testing.T) {
	service, _ := gameProjectTestService(t)
	project := createGameProjectForTest(t, service)
	maximum := float64(100)
	draft := emptyGameProjectDraft()
	draft.Metrics = []GameMetricDefinition{
		{ID: "pressure", Name: "监管压力", Type: "number", Initial: float64(20), Maximum: &maximum},
		{ID: "reputation", Name: "公众信任", Type: "number", Initial: float64(40), Maximum: &maximum},
	}
	draft.Actions = []GameActionDefinition{{
		ID: "disclose", Name: "主动披露", ConditionMode: "all",
		Conditions: []GameConditionDefinition{{Target: "pressure", Operator: "gte", Value: float64(30)}},
		Effects: []GameEffectDefinition{
			{Operation: "decrement", Target: "pressure", Value: float64(10)},
			{Operation: "increment", Target: "reputation", Value: float64(15)},
		}, Emits: "disclosed",
	}}
	updated, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 1, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	if err != nil {
		t.Fatal(err)
	}

	blocked, err := service.PreviewAction("tenant-a", project.Project.ID, PreviewGameActionInput{ActionID: "disclose", RequestKey: "preview-1"})
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Allowed || len(blocked.Reasons) != 1 || blocked.After["pressure"] != float64(20) {
		t.Fatalf("expected an explained rejection without state changes, got %#v", blocked)
	}

	input := PreviewGameActionInput{ActionID: "disclose", State: map[string]any{"pressure": float64(35)}, RequestKey: "preview-2"}
	first, err := service.PreviewAction("tenant-a", project.Project.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.PreviewAction("tenant-a", project.Project.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Allowed || first.After["pressure"] != float64(25) || first.After["reputation"] != float64(55) {
		t.Fatalf("unexpected preview result: %#v", first)
	}
	if fmt.Sprint(first.After) != fmt.Sprint(second.After) {
		t.Fatalf("same preview request must be deterministic: %#v %#v", first.After, second.After)
	}
	stored, err := service.Get("tenant-a", project.Project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Project.Revision != updated.Project.Revision || stored.Draft.Metrics[0].Initial != float64(20) {
		t.Fatalf("preview must not persist game state: %#v", stored)
	}
}

func TestGameProjectPreviewActionAnyConditionAndBounds(t *testing.T) {
	service, _ := gameProjectTestService(t)
	project := createGameProjectForTest(t, service)
	minimum, maximum := float64(0), float64(100)
	draft := emptyGameProjectDraft()
	draft.Metrics = []GameMetricDefinition{{ID: "control", Name: "控制权", Type: "number", Initial: float64(90), Minimum: &minimum, Maximum: &maximum}}
	draft.Actions = []GameActionDefinition{{ID: "secure", ConditionMode: "any", Conditions: []GameConditionDefinition{{Target: "control", Operator: "lt", Value: float64(20)}, {Target: "control", Operator: "gte", Value: float64(80)}}, Effects: []GameEffectDefinition{{Operation: "increment", Target: "control", Value: float64(30)}}, Emits: "secured"}}
	_, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 1, Name: project.Project.Name, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.PreviewAction("tenant-a", project.Project.ID, PreviewGameActionInput{ActionID: "secure"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed || result.After["control"] != float64(100) {
		t.Fatalf("any condition should allow and clamp to maximum: %#v", result)
	}
}

func ptrDraft(value GameProjectDraft) *GameProjectDraft { return &value }
