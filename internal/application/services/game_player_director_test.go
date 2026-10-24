package services

import (
	"control-panel/internal/domain/agent"
	rundomain "control-panel/internal/domain/run"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfirmedPlayerActionInvokesNPCOnceAndPersistsReply(t *testing.T) {
	s, projectID := newPlayerSessionFixture(t)
	npc := agent.AgentConfig{TenantID: "tenant-a", Name: "finance", ContentHash: "npc", SystemPrompt: "财务"}
	require.NoError(t, s.db.Create(&npc).Error)
	project, err := s.projects.Get("tenant-a", projectID)
	require.NoError(t, err)
	draft := project.Draft
	draft.NPCs = append(draft.NPCs, GameProjectNPC{ID: "finance", AgentID: npc.ID, DisplayName: "财务", Role: "财务负责人", CharacterType: GameCharacterNPC, VisibleMetricIDs: []string{"pressure"}})
	_, err = s.projects.Update("tenant-a", projectID, "pm", UpdateGameProjectInput{Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	require.NoError(t, err)
	runner := &gameBridgeRunner{replies: map[string]string{"director": `{"selectedCharacterIds":["finance"],"instruction":"secret director instruction"}`, "finance": `{"kind":"reply","content":"我已了解目前局势"}`}}
	s.SetDirectorBridge(NewGameDirectorBridge(s.db, runner, s.runs))
	session, err := s.Create("tenant-a", projectID, CreateGamePlayerSessionInput{Source: "draft"})
	require.NoError(t, err)
	plan, err := s.Interpret("tenant-a", projectID, session.ID, InterpretGamePlayerInput{Text: "降低压力", RequestKey: "input"})
	require.NoError(t, err)
	confirm := ConfirmGamePlayerInput{PlanID: plan.PendingPlan.ID, RequestKey: "confirm"}
	result, err := s.Confirm("tenant-a", projectID, session.ID, confirm)
	require.NoError(t, err)
	require.Equal(t, float64(-3), result.State["pressure"])
	require.Len(t, runner.calls, 2)
	require.NotContains(t, runner.calls[1].prompt, "secret director instruction")
	_, err = s.Confirm("tenant-a", projectID, session.ID, confirm)
	require.NoError(t, err)
	confirm.RequestKey = "different-retry-key"
	_, err = s.Confirm("tenant-a", projectID, session.ID, confirm)
	require.NoError(t, err)
	require.Len(t, runner.calls, 2)
	var rows []rundomain.RunActivity
	require.NoError(t, s.db.Where("run_id=? AND kind=?", result.RunID, "game.character.response").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "我已了解目前局势", rows[0].Output["content"])
}
