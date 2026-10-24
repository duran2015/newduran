package services

import (
	"testing"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/agentrelation"
)

// migrateGameCharacterDependencies keeps these acceptance tests explicit about
// the Agent Hub records a game character may reference. The game project owns
// none of these records; it only stores checked references to them.
func migrateGameCharacterDependencies(t *testing.T, service *GameProjectService) {
	t.Helper()
	if err := service.db.AutoMigrate(&agent.AgentKnowledgeDataset{}, &agent.DeploymentSnapshot{}, &agentrelation.AgentRelation{}); err != nil {
		t.Fatal(err)
	}
}

func TestGameCharacterCannotReferenceAgentFromAnotherTenant(t *testing.T) {
	service, db := gameProjectTestService(t)
	migrateGameCharacterDependencies(t, service)
	foreign := agent.AgentConfig{Name: "other-cfo", TenantID: "tenant-b", ContentHash: "foreign-v1", SystemPrompt: "secret"}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	project := createGameProjectForTest(t, service)
	draft := emptyGameProjectDraft()
	draft.NPCs = []GameProjectNPC{{ID: "cfo", AgentID: foreign.ID, Role: "财务负责人", CharacterType: GameCharacterNPC}}
	_, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: 1, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	if err == nil {
		t.Fatal("expected a cross-tenant Agent reference to be rejected")
	}
}

func TestGameCharacterOnlyDirectorMayAdvanceTime(t *testing.T) {
	service, db := gameProjectTestService(t)
	migrateGameCharacterDependencies(t, service)
	cfg := agent.AgentConfig{Name: "speeding-cfo", TenantID: "tenant-a", ContentHash: "cfo-v1", SystemPrompt: "CFO"}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	project := createGameProjectForTest(t, service)
	draft := emptyGameProjectDraft()
	draft.NPCs = []GameProjectNPC{{ID: "cfo", AgentID: cfg.ID, Role: "财务负责人", CharacterType: GameCharacterNPC, Permissions: []string{"advance_turn"}}}
	_, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: 1, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	if err == nil {
		t.Fatal("expected a normal NPC with advance_turn permission to be rejected")
	}

	draft.NPCs[0].CharacterType = GameCharacterDirector
	if _, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: 1, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	}); err != nil {
		t.Fatalf("expected a director to be allowed to advance time: %v", err)
	}
}

func TestGameProjectRejectsTwoDirectorsAndUnrelatedRelationship(t *testing.T) {
	service, db := gameProjectTestService(t)
	migrateGameCharacterDependencies(t, service)
	agents := []agent.AgentConfig{
		{Name: "speeding-director", TenantID: "tenant-a", ContentHash: "director-v1", SystemPrompt: "director"},
		{Name: "speeding-cfo", TenantID: "tenant-a", ContentHash: "cfo-v1", SystemPrompt: "cfo"},
		{Name: "speeding-outsider", TenantID: "tenant-a", ContentHash: "outsider-v1", SystemPrompt: "outsider"},
	}
	for i := range agents {
		if err := db.Create(&agents[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	project := createGameProjectForTest(t, service)
	draft := emptyGameProjectDraft()
	draft.NPCs = []GameProjectNPC{
		{ID: "director", AgentID: agents[0].ID, Role: "导演", CharacterType: GameCharacterDirector},
		{ID: "cfo", AgentID: agents[1].ID, Role: "财务负责人", CharacterType: GameCharacterDirector},
	}
	_, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: 1, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	if err == nil {
		t.Fatal("expected a second director to be rejected")
	}

	relation := agentrelation.AgentRelation{TenantID: "tenant-a", Scope: "global", SourceAgentID: agents[1].ID, TargetAgentID: agents[2].ID, RelationType: "advisor", AllowedActions: []string{"consult"}, ContextPolicy: "summary_only", DeliveryPolicy: "async", Enabled: true}
	if err := db.Create(&relation).Error; err != nil {
		t.Fatal(err)
	}
	draft.NPCs[1].CharacterType = GameCharacterNPC
	draft.NPCs[1].RelationIDs = []uint64{relation.ID}
	_, err = service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: 1, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	if err == nil {
		t.Fatal("expected a relationship to a character outside this game to be rejected")
	}
}

func TestGameCharacterContextOnlyShowsConfiguredWorldStateAndActions(t *testing.T) {
	service, db := gameProjectTestService(t)
	migrateGameCharacterDependencies(t, service)
	cfg := agent.AgentConfig{Name: "speeding-cfo", TenantID: "tenant-a", ContentHash: "cfo-v1", SystemPrompt: "CFO", DeploymentStatus: "running"}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	project := createGameProjectForTest(t, service)
	draft := emptyGameProjectDraft()
	draft.Metrics = []GameMetricDefinition{{ID: "wealth", Name: "财富", Type: "number", Initial: float64(100)}, {ID: "pressure", Name: "监管压力", Type: "number", Initial: float64(20)}}
	draft.Actions = []GameActionDefinition{{ID: "disclose", Name: "主动披露"}, {ID: "hide", Name: "隐藏款项"}}
	draft.NPCs = []GameProjectNPC{{ID: "cfo", AgentID: cfg.ID, DisplayName: "莫云岑", Role: "首席财务官", CharacterType: GameCharacterNPC, VisibleMetricIDs: []string{"wealth"}, AdvisoryActionIDs: []string{"disclose"}, KnownInformation: []string{"现金流明细"}}}
	updated, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{
		Revision: 1, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.PreviewCharacterContext("tenant-a", updated.Project.ID, "cfo")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.VisibleWorldState) != 1 || preview.VisibleWorldState[0] != "财富" {
		t.Fatalf("unexpected visible world state: %#v", preview.VisibleWorldState)
	}
	if len(preview.CanSuggest) != 1 || preview.CanSuggest[0] != "主动披露" || len(preview.CanExecute) != 0 {
		t.Fatalf("unexpected action boundary: execute=%#v suggest=%#v", preview.CanExecute, preview.CanSuggest)
	}
	if len(preview.KnownInformation) != 1 || preview.KnownInformation[0] != "现金流明细" {
		t.Fatalf("unexpected known information: %#v", preview.KnownInformation)
	}
	if len(preview.Limits) == 0 || !preview.AgentAvailable {
		t.Fatalf("expected normal NPC limits and running availability: %#v", preview)
	}
}
