package services

import (
	"errors"
	"testing"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/agentrelation"
	"control-panel/internal/domain/capability"
	"control-panel/internal/domain/gamepackage"
	"control-panel/internal/domain/gameproject"
	rundomain "control-panel/internal/domain/run"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newGameSessionUnitFixture(t *testing.T) (*GameSessionService, *GameProjectService, *gorm.DB, string, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&agent.AgentConfig{}, &agent.AgentKnowledgeDataset{}, &agent.DeploymentSnapshot{}, &agentrelation.AgentRelation{}, &capability.Package{}, &gamepackage.Package{}, &gameproject.Project{}, &gameproject.Release{}, &gameproject.Session{}, &rundomain.Run{}, &rundomain.RunAgent{}, &rundomain.CapabilityBinding{}, &rundomain.StateSchema{}, &rundomain.RunState{}, &rundomain.RunStateChange{}, &rundomain.RunActivity{}))
	director := agent.AgentConfig{TenantID: "tenant-a", Name: "director", ContentHash: "director-v1", SystemPrompt: "推进故事"}
	require.NoError(t, db.Create(&director).Error)
	projects := NewGameProjectService(db, NewGamePackageService(db))
	project, err := projects.Create("tenant-a", "pm", CreateGameProjectInput{Key: "session.test", Name: "试玩", Description: "test", Genre: "策略", PlayGoal: "存活"})
	require.NoError(t, err)
	draft := GameProjectDraft{
		Metrics: []GameMetricDefinition{{ID: "pressure", Name: "压力", Type: "number", Initial: float64(0)}},
		NPCs:    []GameProjectNPC{{ID: "director", AgentID: director.ID, DisplayName: "导演", Role: "导演", CharacterType: GameCharacterDirector}},
		Events: []GameProjectEvent{
			{ID: "first", Name: "首次警报", DisplayContent: "警报出现", Priority: 20, Once: true, Conditions: []GameConditionDefinition{{Target: "pressure", Operator: "eq", Value: float64(0)}}, Effects: []GameEffectDefinition{{Target: "pressure", Operation: "increment", Value: float64(10)}}, ParticipantCharacterIDs: []string{"director"}, AdvancesTime: true, TimeAdvance: 1},
			{ID: "cool", Name: "持续压力", Priority: 10, CooldownTurns: 1, Conditions: []GameConditionDefinition{{Target: "pressure", Operator: "gte", Value: float64(10)}}, Effects: []GameEffectDefinition{{Target: "pressure", Operation: "increment", Value: float64(5)}}, ParticipantCharacterIDs: []string{"director"}, AdvancesTime: true, TimeAdvance: 1},
		},
		Actions: []GameActionDefinition{}, Endings: []GameEndingDefinition{}, Scenes: []GameProjectScene{},
	}
	_, err = projects.Update("tenant-a", project.Project.ID, "pm", UpdateGameProjectInput{Revision: 1, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	require.NoError(t, err)
	runs := NewRunService(db)
	return NewGameSessionService(db, projects, runs), projects, db, project.Project.ID, "director"
}

func TestGameSessionUsesRunStateAndAdvancesAtomicallyIdempotently(t *testing.T) {
	s, projects, db, projectID, director := newGameSessionUnitFixture(t)
	session, err := s.Create("tenant-a", projectID, CreateGameSessionInput{CreatedBy: "pm"})
	require.NoError(t, err)
	require.Equal(t, rundomain.StatusRunning, session.Status)
	require.Equal(t, 0, session.CurrentTurn)
	// Later draft edits do not alter this play-through's pinned revision.
	detail, err := projects.Get("tenant-a", projectID)
	require.NoError(t, err)
	draft := detail.Draft
	draft.Metrics[0].Initial = float64(99)
	_, err = projects.Update("tenant-a", projectID, "pm", UpdateGameProjectInput{Revision: detail.Project.Revision, Name: detail.Project.Name, Description: detail.Project.Description, Genre: detail.Project.Genre, PlayGoal: detail.Project.PlayGoal, Draft: &draft})
	require.NoError(t, err)
	a, err := s.Advance("tenant-a", projectID, session.ID, AdvanceGameSessionInput{RequestKey: "turn-1", ActorCharacterID: director})
	require.NoError(t, err)
	require.Equal(t, 1, a.CurrentTurn)
	require.Equal(t, float64(10), a.State["pressure"])
	require.Equal(t, "first", a.LatestEvent.EventID)
	b, err := s.Advance("tenant-a", projectID, session.ID, AdvanceGameSessionInput{RequestKey: "turn-1", ActorCharacterID: director})
	require.NoError(t, err)
	require.Equal(t, 1, b.CurrentTurn)
	require.Equal(t, float64(10), b.State["pressure"])
	var changes int64
	require.NoError(t, db.Model(&rundomain.RunStateChange{}).Where("run_id=?", session.RunID).Count(&changes).Error)
	require.Equal(t, int64(2), changes) // initialization + one turn
	c, err := s.Advance("tenant-a", projectID, session.ID, AdvanceGameSessionInput{RequestKey: "turn-2", ActorCharacterID: director})
	require.NoError(t, err)
	require.Equal(t, 2, c.CurrentTurn)
	require.Equal(t, float64(15), c.State["pressure"])
}

func TestGameSessionDirectorLifecycleAndTenantIsolation(t *testing.T) {
	s, _, _, projectID, director := newGameSessionUnitFixture(t)
	session, err := s.Create("tenant-a", projectID, CreateGameSessionInput{CreatedBy: "pm"})
	require.NoError(t, err)
	_, err = s.Advance("tenant-a", projectID, session.ID, AdvanceGameSessionInput{RequestKey: "bad", ActorCharacterID: "npc"})
	require.ErrorIs(t, err, ErrGameDirectorRequired)
	_, err = s.Get("tenant-b", projectID, session.ID)
	require.ErrorIs(t, err, ErrGameSessionNotFound)
	p, err := s.Pause("tenant-a", projectID, session.ID, director)
	require.NoError(t, err)
	require.Equal(t, rundomain.StatusPaused, p.Status)
	_, err = s.Advance("tenant-a", projectID, session.ID, AdvanceGameSessionInput{RequestKey: "paused", ActorCharacterID: director})
	require.Error(t, err)
	r, err := s.Resume("tenant-a", projectID, session.ID, director)
	require.NoError(t, err)
	require.Equal(t, rundomain.StatusRunning, r.Status)
	e, err := s.End("tenant-a", projectID, session.ID, director)
	require.NoError(t, err)
	require.Equal(t, rundomain.StatusCompleted, e.Status)
	_, err = s.Resume("tenant-a", projectID, session.ID, director)
	require.True(t, errors.Is(err, rundomain.ErrInvalidTransition))
}
