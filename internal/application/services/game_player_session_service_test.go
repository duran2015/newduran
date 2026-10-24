package services

import (
	"context"
	"testing"

	"control-panel/internal/domain/gameproject"
	rundomain "control-panel/internal/domain/run"
	"github.com/stretchr/testify/require"
)

func newPlayerSessionFixture(t *testing.T) (*GamePlayerSessionService, string) {
	t.Helper()
	sessions, projects, db, projectID, _ := newGameSessionUnitFixture(t)
	require.NoError(t, db.AutoMigrate(&gameproject.PlayerPlan{}))
	detail, err := projects.Get("tenant-a", projectID)
	require.NoError(t, err)
	draft := detail.Draft
	draft.Actions = []GameActionDefinition{{
		ID: "calm-down", Name: "降低压力", TimeCost: 1,
		Effects: []GameEffectDefinition{{Target: "pressure", Operation: "increment", Value: float64(-3)}},
	}}
	_, err = projects.Update("tenant-a", projectID, "pm", UpdateGameProjectInput{
		Revision: detail.Project.Revision, Name: detail.Project.Name, Description: detail.Project.Description,
		Genre: detail.Project.Genre, PlayGoal: detail.Project.PlayGoal, Draft: &draft,
	})
	require.NoError(t, err)
	runs := NewRunService(db)
	return NewGamePlayerSessionService(db, projects, projects.packages, sessions, runs), projectID
}

func TestPlayerActionOnlyChangesStateAfterConfirmationAndOnlyOnce(t *testing.T) {
	service, projectID := newPlayerSessionFixture(t)
	service.SetActionModelFactory(func(string) GameActionLanguageModel {
		return playerLanguageStub{`{"actionId":"calm-down","confidence":0.95,"summary":"先缓和局面"}`}
	})
	session, err := service.Create("tenant-a", projectID, CreateGamePlayerSessionInput{Source: "draft", CreatedBy: "pm"})
	require.NoError(t, err)
	require.Equal(t, float64(0), session.State["pressure"])

	interpreted, err := service.Interpret("tenant-a", projectID, session.ID, InterpretGamePlayerInput{
		Text: "我想先降低压力，再观察情况", RequestKey: "say-1",
	})
	require.NoError(t, err)
	require.NotNil(t, interpreted.PendingPlan)
	require.Equal(t, float64(0), interpreted.State["pressure"], "理解玩家输入时不能提前修改世界状态")

	confirmed, err := service.Confirm("tenant-a", projectID, session.ID, ConfirmGamePlayerInput{
		PlanID: interpreted.PendingPlan.ID, RequestKey: "confirm-1",
	})
	require.NoError(t, err)
	require.Equal(t, float64(-3), confirmed.State["pressure"])

	replayed, err := service.Confirm("tenant-a", projectID, session.ID, ConfirmGamePlayerInput{
		PlanID: interpreted.PendingPlan.ID, RequestKey: "confirm-1",
	})
	require.NoError(t, err)
	require.Equal(t, float64(-3), replayed.State["pressure"], "重复确认不能重复结算")
}

type playerLanguageStub struct{ response string }

func (m playerLanguageStub) InterpretGameAction(context.Context, string) ([]byte, error) {
	return []byte(m.response), nil
}

func TestPlayerFreeformInteractionCanProceedWithoutChangingHardState(t *testing.T) {
	service, projectID := newPlayerSessionFixture(t)
	project, err := service.projects.Get("tenant-a", projectID)
	require.NoError(t, err)
	draft := project.Draft
	draft.Events = nil
	_, err = service.projects.Update("tenant-a", projectID, "pm", UpdateGameProjectInput{Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	require.NoError(t, err)
	session, err := service.Create("tenant-a", projectID, CreateGamePlayerSessionInput{Source: "draft"})
	require.NoError(t, err)
	result, err := service.Interpret("tenant-a", projectID, session.ID, InterpretGamePlayerInput{Text: "我想试探她是否愿意站在我这边", RequestKey: "freeform"})
	require.NoError(t, err)
	require.False(t, result.PendingPlan.NeedsClarification)
	require.True(t, result.PendingPlan.NeedsConfirmation)
	require.Equal(t, gameFreeformActionID, result.PendingPlan.ActionID)
	confirmed, err := service.Confirm("tenant-a", projectID, session.ID, ConfirmGamePlayerInput{PlanID: result.PendingPlan.ID, RequestKey: "confirm-freeform"})
	require.NoError(t, err)
	require.Equal(t, float64(0), confirmed.State["pressure"], "自由互动不能自行改写硬数值")
	require.Equal(t, 1, confirmed.CurrentTurn)
}

func TestPlayerCanRejectPlanAndOnlyOnePlanMayWaitForConfirmation(t *testing.T) {
	service, projectID := newPlayerSessionFixture(t)
	session, err := service.Create("tenant-a", projectID, CreateGamePlayerSessionInput{Source: "draft", CreatedBy: "pm"})
	require.NoError(t, err)

	first, err := service.Interpret("tenant-a", projectID, session.ID, InterpretGamePlayerInput{Text: "降低压力", RequestKey: "say-1"})
	require.NoError(t, err)
	_, err = service.Interpret("tenant-a", projectID, session.ID, InterpretGamePlayerInput{Text: "再次降低压力", RequestKey: "say-2"})
	require.Error(t, err, "已有待确认行动时不能悄悄创建第二个分支")

	rejected, err := service.Reject("tenant-a", projectID, session.ID, RejectGamePlayerInput{PlanID: first.PendingPlan.ID})
	require.NoError(t, err)
	require.Nil(t, rejected.PendingPlan)
	require.Equal(t, float64(0), rejected.State["pressure"], "放弃行动不能改变世界状态")
}

func TestPlayerExplicitlyAcceptsOrRejectsOfferedObjective(t *testing.T) {
	service, projectID := newPlayerSessionFixture(t)
	project, err := service.projects.Get("tenant-a", projectID)
	require.NoError(t, err)
	draft := project.Draft
	draft.Objectives = []GameObjectiveDefinition{{ID: "submit-report", Name: "提交报告", DeadlineTurns: 2, PromiseText: "两回合内提交报告", Conditions: []GameConditionDefinition{{Target: "pressure", Operator: "lte", Value: -10}}}}
	_, err = service.projects.Update("tenant-a", projectID, "pm", UpdateGameProjectInput{Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	require.NoError(t, err)
	session, err := service.Create("tenant-a", projectID, CreateGamePlayerSessionInput{Source: "draft", CreatedBy: "pm"})
	require.NoError(t, err)

	offer := func() {
		service.activateOfferedObjectives(GameDirectorTurnInput{TenantID: "tenant-a", RunID: session.RunID, Draft: draft}, &GameDirectorTurnResult{Responses: []GameAgentTurnResponse{{CharacterID: "issuer", DisplayName: "发布人", ObjectiveID: "submit-report"}}})
	}
	offer()
	offered, err := service.Get("tenant-a", projectID, session.ID)
	require.NoError(t, err)
	require.Equal(t, "offered", offered.Objectives[0].Status)
	require.Zero(t, offered.Objectives[0].DeadlineTurn, "未接受前不能启动截止回合")

	rejected, err := service.RejectObjective("tenant-a", projectID, session.ID, DecideGameObjectiveInput{ObjectiveID: "submit-report", RequestKey: "reject-1"})
	require.NoError(t, err)
	require.Equal(t, "rejected", rejected.Objectives[0].Status)
	require.Zero(t, rejected.Objectives[0].DeadlineTurn)
	var broken int64
	require.NoError(t, service.db.Model(&rundomain.RunActivity{}).Where("run_id=? AND kind=?", session.RunID, "game.objective.promise_broken").Count(&broken).Error)
	require.Zero(t, broken, "拒绝任务不是违约")

	offer()
	reoffered, err := service.Get("tenant-a", projectID, session.ID)
	require.NoError(t, err)
	require.Equal(t, "offered", reoffered.Objectives[0].Status, "拒绝后允许NPC重新交付")
	accepted, err := service.AcceptObjective("tenant-a", projectID, session.ID, DecideGameObjectiveInput{ObjectiveID: "submit-report", RequestKey: "accept-1"})
	require.NoError(t, err)
	require.Equal(t, "active", accepted.Objectives[0].Status)
	require.Equal(t, accepted.CurrentTurn+2, accepted.Objectives[0].DeadlineTurn)
	require.Equal(t, "两回合内提交报告", accepted.Objectives[0].PromiseText)

	replayed, err := service.AcceptObjective("tenant-a", projectID, session.ID, DecideGameObjectiveInput{ObjectiveID: "submit-report", RequestKey: "accept-1"})
	require.NoError(t, err)
	require.Equal(t, accepted.Objectives[0].DeadlineTurn, replayed.Objectives[0].DeadlineTurn, "同一请求重试不能重置期限")
	_, err = service.AcceptObjective("tenant-b", projectID, session.ID, DecideGameObjectiveInput{ObjectiveID: "submit-report", RequestKey: "cross-tenant"})
	require.ErrorIs(t, err, ErrGameSessionNotFound)
}

func TestPlayerSessionCanStartAtTurnAndEventThenResetWithoutChangingHistory(t *testing.T) {
	service, projectID := newPlayerSessionFixture(t)
	project, err := service.projects.Get("tenant-a", projectID)
	require.NoError(t, err)
	require.NotEmpty(t, project.Draft.Events)
	eventID := project.Draft.Events[0].ID
	started, err := service.Create("tenant-a", projectID, CreateGamePlayerSessionInput{Source: "draft", ProjectRevision: project.Project.Revision, StartTurn: 4, StartEventID: eventID, CreatedBy: "pm"})
	require.NoError(t, err)
	require.Equal(t, 4, started.CurrentTurn)
	require.Equal(t, eventID, started.StartEventID)
	reset, err := service.Reset("tenant-a", projectID, started.ID, "pm")
	require.NoError(t, err)
	require.NotEqual(t, started.ID, reset.ID)
	require.Equal(t, started.ProjectRevision, reset.ProjectRevision)
	require.Equal(t, 4, reset.CurrentTurn)
	require.Equal(t, eventID, reset.StartEventID)
	original, err := service.Get("tenant-a", projectID, started.ID)
	require.NoError(t, err)
	require.Equal(t, started.ID, original.ID, "重置不能覆盖原试玩记录")
}

func TestPlayerSessionRejectsStaleDraftRevisionAndUnknownStartingEvent(t *testing.T) {
	service, projectID := newPlayerSessionFixture(t)
	_, err := service.Create("tenant-a", projectID, CreateGamePlayerSessionInput{Source: "draft", ProjectRevision: 999, CreatedBy: "pm"})
	require.ErrorIs(t, err, ErrGameProjectConflict)
	_, err = service.Create("tenant-a", projectID, CreateGamePlayerSessionInput{Source: "draft", StartEventID: "missing", CreatedBy: "pm"})
	require.Error(t, err)
}

func TestPlayerConfirmSettlesPriorityMutexTurnAndEndingInOneTransaction(t *testing.T) {
	service, projectID := newPlayerSessionFixture(t)
	project, err := service.projects.Get("tenant-a", projectID)
	require.NoError(t, err)
	draft := project.Draft
	draft.Events = []GameProjectEvent{
		{ID: "low", Name: "较弱的后果", Priority: 10, ExclusiveGroup: "result", Conditions: []GameConditionDefinition{{Target: "pressure", Operator: "lte", Value: float64(-3)}}, Effects: []GameEffectDefinition{{Target: "pressure", Operation: "increment", Value: float64(-100)}}},
		{ID: "high", Name: "决定性的后果", Priority: 20, ExclusiveGroup: "result", Once: true, TimeAdvance: 2, EndGame: true, Conditions: []GameConditionDefinition{{Target: "pressure", Operator: "lte", Value: float64(-3)}}, Effects: []GameEffectDefinition{{Target: "pressure", Operation: "increment", Value: float64(-7)}}},
	}
	_, err = service.projects.Update("tenant-a", projectID, "pm", UpdateGameProjectInput{Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft})
	require.NoError(t, err)

	session, err := service.Create("tenant-a", projectID, CreateGamePlayerSessionInput{Source: "draft", CreatedBy: "pm"})
	require.NoError(t, err)
	interpreted, err := service.Interpret("tenant-a", projectID, session.ID, InterpretGamePlayerInput{Text: "降低压力", RequestKey: "say-settle"})
	require.NoError(t, err)
	confirmed, err := service.Confirm("tenant-a", projectID, session.ID, ConfirmGamePlayerInput{PlanID: interpreted.PendingPlan.ID, RequestKey: "confirm-settle"})
	require.NoError(t, err)
	require.Equal(t, float64(-10), confirmed.State["pressure"], "只应结算互斥组中优先级最高的事件")
	require.Equal(t, 3, confirmed.CurrentTurn, "行动耗时与事件推进应在同次结算中完成")
	require.Equal(t, rundomain.StatusCompleted, confirmed.Status)
	require.NotNil(t, confirmed.LatestEvent)
	require.Equal(t, "high", confirmed.LatestEvent.EventID)

	replayed, err := service.Confirm("tenant-a", projectID, session.ID, ConfirmGamePlayerInput{PlanID: interpreted.PendingPlan.ID, RequestKey: "confirm-settle"})
	require.NoError(t, err)
	require.Equal(t, float64(-10), replayed.State["pressure"])
	require.Equal(t, 3, replayed.CurrentTurn)
	var eventActivities int64
	require.NoError(t, service.db.Model(&rundomain.RunActivity{}).Where("run_id=? AND kind=?", session.RunID, "game.event.triggered").Count(&eventActivities).Error)
	require.Equal(t, int64(1), eventActivities, "重复确认不能再次写入事件留痕")
}

func TestPlayerConfirmFailureRollsBackStatePlanAndTimeline(t *testing.T) {
	service, projectID := newPlayerSessionFixture(t)
	session, err := service.Create("tenant-a", projectID, CreateGamePlayerSessionInput{Source: "draft", CreatedBy: "pm"})
	require.NoError(t, err)
	interpreted, err := service.Interpret("tenant-a", projectID, session.ID, InterpretGamePlayerInput{Text: "降低压力", RequestKey: "say-rollback"})
	require.NoError(t, err)
	require.NoError(t, service.db.Exec(`CREATE TRIGGER fail_player_turn BEFORE INSERT ON run_activities WHEN NEW.kind = 'game.turn.advanced' BEGIN SELECT RAISE(ABORT, 'forced rollback'); END`).Error)

	_, err = service.Confirm("tenant-a", projectID, session.ID, ConfirmGamePlayerInput{PlanID: interpreted.PendingPlan.ID, RequestKey: "confirm-rollback"})
	require.Error(t, err)
	after, err := service.Get("tenant-a", projectID, session.ID)
	require.NoError(t, err)
	require.Equal(t, float64(0), after.State["pressure"], "结算失败不能留下部分状态")
	require.Equal(t, 0, after.CurrentTurn)
	require.NotNil(t, after.PendingPlan)
	require.Equal(t, "pending", after.PendingPlan.Status)
	var changes int64
	require.NoError(t, service.db.Model(&rundomain.RunStateChange{}).Where("run_id=? AND idempotency_key=?", session.RunID, "player-confirm:confirm-rollback").Count(&changes).Error)
	require.Zero(t, changes)
	var confirmedActivities int64
	require.NoError(t, service.db.Model(&rundomain.RunActivity{}).Where("run_id=? AND kind=?", session.RunID, "game.player.confirmed").Count(&confirmedActivities).Error)
	require.Zero(t, confirmedActivities, "失败事务不能留下玩家已执行的假留痕")
}
