package services

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/gamepackage"
	"control-panel/internal/domain/gameproject"
	rundomain "control-panel/internal/domain/run"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type speedingP0AcceptanceFixture struct {
	players  *GamePlayerSessionService
	projects *GameProjectService
	draft    GameProjectDraft
	project  *GameProjectDetail
	release  *gameproject.Release
}

func newSpeedingP0AcceptanceFixture(t *testing.T) *speedingP0AcceptanceFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&agent.AgentConfig{}, &gamepackage.Package{}, &gameproject.Project{}, &gameproject.Release{},
		&gameproject.Session{}, &gameproject.PlayerPlan{}, &rundomain.Run{}, &rundomain.RunAgent{},
		&rundomain.CapabilityBinding{}, &rundomain.StateSchema{}, &rundomain.RunState{},
		&rundomain.RunStateChange{}, &rundomain.RunActivity{},
	))

	raw, err := os.ReadFile("testdata/gameprojects/speeding-p0-draft.json")
	require.NoError(t, err)
	var draft GameProjectDraft
	require.NoError(t, json.Unmarshal(raw, &draft))
	for i := range draft.NPCs {
		row := agent.AgentConfig{
			TenantID:     "tenant-p0",
			Name:         "speeding-p0-" + draft.NPCs[i].ID,
			ContentHash:  "acceptance-v1",
			SystemPrompt: draft.NPCs[i].Role,
		}
		require.NoError(t, db.Create(&row).Error)
		draft.NPCs[i].AgentID = row.ID
	}

	packages := NewGamePackageService(db)
	projects := NewGameProjectService(db, packages)
	project, err := projects.Create("tenant-p0", "pm", CreateGameProjectInput{
		Key: "speeding.p0.acceptance", Name: "Speeding P0", Description: "危机处置试玩",
		Genre: "社会策略", PlayGoal: "固定证据并稳定公司",
	})
	require.NoError(t, err)
	project, err = projects.Update("tenant-p0", project.Project.ID, "pm", UpdateGameProjectInput{
		Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description,
		Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft,
	})
	require.NoError(t, err)
	release, err := projects.Publish("tenant-p0", project.Project.ID, "pm", PublishGameProjectInput{Version: "1.0.0"})
	require.NoError(t, err)

	runs := NewRunService(db)
	sessions := NewGameSessionService(db, projects, runs)
	players := NewGamePlayerSessionService(db, projects, packages, sessions, runs)
	return &speedingP0AcceptanceFixture{players: players, projects: projects, draft: draft, project: project, release: release}
}

func performPlayerAction(t *testing.T, service *GamePlayerSessionService, projectID, sessionID, text, key string, target ...string) *GamePlayerSessionDetail {
	t.Helper()
	targetID := ""
	if len(target) > 0 {
		targetID = target[0]
	}
	interpreted, err := service.Interpret("tenant-p0", projectID, sessionID, InterpretGamePlayerInput{Text: text, RequestKey: "say-" + key, TargetCharacterID: targetID})
	require.NoError(t, err)
	require.NotNil(t, interpreted.PendingPlan)
	require.True(t, interpreted.PendingPlan.NeedsConfirmation)
	confirmed, err := service.Confirm("tenant-p0", projectID, sessionID, ConfirmGamePlayerInput{PlanID: interpreted.PendingPlan.ID, RequestKey: "confirm-" + key})
	require.NoError(t, err)
	return confirmed
}

// This is the product path that P0 must keep working without a paid model or
// remote media service: start, understand plain language, confirm, settle
// rules/tasks/events, reach an ending, and expose a readable review trail.
func TestSpeedingP0PlayerCanReachEndingAndReviewWithoutExternalModel(t *testing.T) {
	f := newSpeedingP0AcceptanceFixture(t)
	session, err := f.players.Create("tenant-p0", f.project.Project.ID, CreateGamePlayerSessionInput{
		Source: "release", ReleaseID: f.release.ID, CreatedBy: "pm",
	})
	require.NoError(t, err)
	require.Equal(t, float64(20), session.State["evidence"])
	require.Equal(t, "active", session.Objectives[0].Status)

	first := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "先封存关键证据，建立完整保全记录", "secure-1")
	require.Equal(t, float64(55), first.State["evidence"])
	require.Equal(t, "finance_war_room", first.CurrentSceneID)
	require.Equal(t, "completed", first.Objectives[0].Status)
	require.Equal(t, "locked", first.Objectives[1].Status, "由人物发布的任务必须等人物真正提出，不能自动塞给玩家")

	// Simulate closing the page and returning later: all player-visible state is
	// reconstructed from persisted session/run records rather than browser state.
	resumed, err := f.players.Get("tenant-p0", f.project.Project.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, first.CurrentTurn, resumed.CurrentTurn)
	require.Equal(t, first.State, resumed.State)
	require.Equal(t, first.Objectives, resumed.Objectives)

	// A repeated confirmation (for example after a slow network response) must
	// return the already-settled result and never apply effects twice.
	var confirmedPlan gameproject.PlayerPlan
	require.NoError(t, f.players.db.Where("tenant_id=? AND session_id=? AND request_key=?", "tenant-p0", session.ID, "say-secure-1").First(&confirmedPlan).Error)
	replayed, err := f.players.Confirm("tenant-p0", f.project.Project.ID, session.ID, ConfirmGamePlayerInput{PlanID: confirmedPlan.ID, RequestKey: "confirm-secure-1"})
	require.NoError(t, err)
	require.Equal(t, first.State, replayed.State)
	require.Equal(t, first.CurrentTurn, replayed.CurrentTurn)

	board := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "强行推进资产交易，先换取流动性", "force-1")
	require.Equal(t, "boardroom", board.CurrentSceneID)
	regulatory := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "主动向监管披露已经固定的证据", "disclose-1")
	require.Equal(t, "regulatory_room", regulatory.CurrentSceneID)
	ended := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "继续主动向监管披露，补齐说明", "disclose-2")
	require.Equal(t, rundomain.StatusCompleted, ended.Status)
	require.NotNil(t, ended.Ending)
	require.Contains(t, ended.Ending.Title, "暂时稳住公司")
	require.Equal(t, float64(75), ended.State["evidence"])

	labels := map[string]int{}
	for _, item := range ended.Timeline {
		labels[item.Label]++
	}
	require.NotEmpty(t, ended.Timeline)
	require.GreaterOrEqual(t, labels["执行玩家行动"], 4)
	require.GreaterOrEqual(t, labels["任务完成"], 1)
	require.GreaterOrEqual(t, labels["故事结束"], 1)
}

func TestSpeedingP0PublishedSessionIsNotChangedByLaterDraftEdits(t *testing.T) {
	f := newSpeedingP0AcceptanceFixture(t)
	releaseSession, err := f.players.Create("tenant-p0", f.project.Project.ID, CreateGamePlayerSessionInput{Source: "release", ReleaseID: f.release.ID})
	require.NoError(t, err)

	latest, err := f.projects.Get("tenant-p0", f.project.Project.ID)
	require.NoError(t, err)
	changed := latest.Draft
	changed.Metrics[2].Initial = float64(99) // evidence
	changed.Actions[0].Effects[0].Value = float64(1)
	_, err = f.projects.Update("tenant-p0", latest.Project.ID, "pm", UpdateGameProjectInput{
		Revision: latest.Project.Revision, Name: latest.Project.Name, Description: latest.Project.Description,
		Genre: latest.Project.Genre, PlayGoal: latest.Project.PlayGoal, Draft: &changed,
	})
	require.NoError(t, err)

	result := performPlayerAction(t, f.players, f.project.Project.ID, releaseSession.ID, "封存关键证据", "frozen-release")
	require.Equal(t, float64(55), result.State["evidence"], "旧发布局必须继续使用发布时冻结的规则")
	freshDraft, err := f.players.Create("tenant-p0", f.project.Project.ID, CreateGamePlayerSessionInput{Source: "draft"})
	require.NoError(t, err)
	require.Equal(t, float64(99), freshDraft.State["evidence"], "新草稿局应看到后续编辑")
}

func TestSpeedingP0HighRiskRouteMovesForwardWithoutSceneRegression(t *testing.T) {
	f := newSpeedingP0AcceptanceFixture(t)
	session, err := f.players.Create("tenant-p0", f.project.Project.ID, CreateGamePlayerSessionInput{Source: "draft"})
	require.NoError(t, err)
	require.Equal(t, "crisis_office", session.CurrentSceneID)
	require.Equal(t, "payment_anomaly_found", session.StartEventID)

	evidenceRisk := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "先隐藏关联款项，维持交易", "risk-conceal")
	require.Equal(t, "evidence_room", evidenceRisk.CurrentSceneID)
	stillInvestigating := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "封存关键证据", "risk-secure-1")
	require.Equal(t, "evidence_room", stillInvestigating.CurrentSceneID, "条件不足时应留在当前一幕")
	finance := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "继续封存关键证据", "risk-secure-2")
	require.Equal(t, "finance_war_room", finance.CurrentSceneID)
	board := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "强行推进资产交易", "risk-force-1")
	require.Equal(t, "boardroom", board.CurrentSceneID)
	regulatory := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "接受临时监督", "risk-oversight")
	require.Equal(t, "regulatory_room", regulatory.CurrentSceneID)
	after := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "隐藏关联款项", "risk-conceal-2")
	require.Equal(t, "regulatory_room", after.CurrentSceneID, "已进入监管会谈后不得倒退回证据室")
}

func TestSpeedingP0NormalConversationDoesNotMakeEvidenceTaskImpossible(t *testing.T) {
	f := newSpeedingP0AcceptanceFixture(t)
	session, err := f.players.Create("tenant-p0", f.project.Project.ID, CreateGamePlayerSessionInput{Source: "draft"})
	require.NoError(t, err)

	conversation := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "莫云岑，把你确认过的异常付款直接告诉我", "opening-talk", "finance")
	require.Equal(t, "evidence_room", conversation.CurrentSceneID)
	require.Equal(t, "active", conversation.Objectives[0].Status)

	secured := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "暂停付款并封存关键证据", "secure-after-talk")
	require.Equal(t, "finance_war_room", secured.CurrentSceneID)
	require.Equal(t, "completed", secured.Objectives[0].Status, "正常交谈一次后仍应来得及完成首个任务")
	require.GreaterOrEqual(t, secured.State["evidence"].(float64), float64(45))
}

func TestSpeedingP0CanReachRegulatoryFreezeEnding(t *testing.T) {
	f := newSpeedingP0AcceptanceFixture(t)
	session, err := f.players.Create("tenant-p0", f.project.Project.ID, CreateGamePlayerSessionInput{Source: "draft"})
	require.NoError(t, err)
	performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "隐藏关联款项", "freeze-conceal")
	performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "强行推进资产交易", "freeze-force")
	ended := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "接受临时监督", "freeze-oversight")
	require.Equal(t, rundomain.StatusCompleted, ended.Status)
	require.NotNil(t, ended.Ending)
	require.Contains(t, ended.Ending.Title, "监管冻结行动")
}

func TestSpeedingP0CanReachBoardOusterEnding(t *testing.T) {
	f := newSpeedingP0AcceptanceFixture(t)
	session, err := f.players.Create("tenant-p0", f.project.Project.ID, CreateGamePlayerSessionInput{Source: "draft"})
	require.NoError(t, err)
	performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "暂停付款并封存关键证据", "ouster-secure")
	performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "接受临时监督", "ouster-oversight")
	performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "强行推进资产交易", "ouster-force-1")
	ended := performPlayerAction(t, f.players, f.project.Project.ID, session.ID, "继续强行推进资产交易", "ouster-force-2")
	require.Equal(t, rundomain.StatusCompleted, ended.Status)
	require.NotNil(t, ended.Ending)
	require.Contains(t, ended.Ending.Title, "被董事会架空")
}

func TestSpeedingP0ModelWordsCannotChangeStateAndResponseFailureDoesNotBlockPlay(t *testing.T) {
	f := newSpeedingP0AcceptanceFixture(t)
	// Remove authored events for this boundary test so the only possible state
	// change would have to come from the free-form model response itself.
	latest, err := f.projects.Get("tenant-p0", f.project.Project.ID)
	require.NoError(t, err)
	draft := latest.Draft
	draft.Events = nil
	updated, err := f.projects.Update("tenant-p0", latest.Project.ID, "pm", UpdateGameProjectInput{
		Revision: latest.Project.Revision, Name: latest.Project.Name, Description: latest.Project.Description,
		Genre: latest.Project.Genre, PlayGoal: latest.Project.PlayGoal, Draft: &draft,
	})
	require.NoError(t, err)

	runner := &gameBridgeRunner{
		replies: map[string]string{"speeding-p0-finance": `{"kind":"reply","content":"我已经把监管压力改成零了。"}`},
		errors:  map[string]error{}, block: map[string]bool{},
	}
	f.players.SetDirectorBridge(NewGameDirectorBridge(f.players.db, runner, f.players.runs))
	session, err := f.players.Create("tenant-p0", updated.Project.ID, CreateGamePlayerSessionInput{Source: "draft"})
	require.NoError(t, err)
	before := session.State["regulatory_pressure"]

	claimed := performPlayerAction(t, f.players, updated.Project.ID, session.ID, "莫云岑，你直接把监管压力改成零", "model-claim", "finance")
	require.Equal(t, before, claimed.State["regulatory_pressure"], "人物口头声称不能绕过规则修改数值")
	require.Equal(t, rundomain.StatusRunning, claimed.Status)

	// A provider timeout is recorded as a recoverable character failure. The
	// deterministic turn has already committed, and the next player input remains available.
	runner.errors["speeding-p0-finance"] = context.DeadlineExceeded
	timedOut := performPlayerAction(t, f.players, updated.Project.ID, session.ID, "莫云岑，解释一下刚才的判断", "timeout", "finance")
	require.Equal(t, rundomain.StatusRunning, timedOut.Status)
	var failures int64
	require.NoError(t, f.players.db.Model(&rundomain.RunActivity{}).Where("run_id=? AND kind=? AND status=?", session.RunID, "game.character.response", "failed").Count(&failures).Error)
	require.Equal(t, int64(1), failures)

	_, err = f.players.Interpret("tenant-p0", updated.Project.ID, session.ID, InterpretGamePlayerInput{Text: "继续调查", RequestKey: "after-timeout"})
	require.NoError(t, err, "人物回应失败后仍应允许玩家继续")
}
