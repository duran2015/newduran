package services

import (
	"strings"
	"testing"

	"control-panel/internal/domain/agent"
)

func saveEventPreviewDraft(t *testing.T, service *GameProjectService, project *GameProjectDetail, events []GameProjectEvent) {
	t.Helper()
	draft := emptyGameProjectDraft()
	draft.Metrics = []GameMetricDefinition{
		{ID: "pressure", Name: "外部压力", Type: "number", Initial: float64(20)},
		{ID: "trust", Name: "公众信任", Type: "number", Initial: float64(50)},
	}
	draft.Actions = []GameActionDefinition{{ID: "wait", Name: "等待", Effects: []GameEffectDefinition{}, Emits: "waited"}}
	draft.Events = events
	if _, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: project.Project.Revision, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft}); err != nil {
		t.Fatal(err)
	}
}

func TestPreviewEventsExplainsConditionsAndEffectWithoutWriting(t *testing.T) {
	service, _ := gameProjectTestService(t)
	project := createGameProjectForTest(t, service)
	events := []GameProjectEvent{{
		ID: "public_crisis", Name: "舆情危机", EventType: "system", Content: "外部质疑集中出现",
		ConditionMode: "all", Conditions: []GameConditionDefinition{{Target: "pressure", Operator: "gte", Value: float64(30)}},
		Effects: []GameEffectDefinition{{Operation: "decrement", Target: "trust", Value: float64(10)}}, AdvanceTurns: 1,
	}}
	saveEventPreviewDraft(t, service, project, events)

	blocked, err := service.PreviewEvents("tenant-a", project.Project.ID, "public_crisis", PreviewGameEventsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Items[0].Triggered || blocked.Items[0].After["trust"] != float64(50) || len(blocked.Items[0].Reasons) == 0 {
		t.Fatalf("expected explained non-match without effects: %#v", blocked.Items[0])
	}
	matched, err := service.PreviewEvents("tenant-a", project.Project.ID, "public_crisis", PreviewGameEventsInput{State: map[string]any{"pressure": float64(35)}, CurrentTurn: 2})
	if err != nil {
		t.Fatal(err)
	}
	item := matched.Items[0]
	if !item.Triggered || !item.Selected || item.After["trust"] != float64(40) || item.AdvanceTurns != 1 || len(item.StateDiff) != 1 {
		t.Fatalf("unexpected matched preview: %#v", item)
	}
	stored, err := service.Get("tenant-a", project.Project.ID)
	if err != nil || stored.Draft.Metrics[1].Initial != float64(50) {
		t.Fatalf("preview must not persist state: %#v, %v", stored, err)
	}
}

func TestPreviewEventsHonorsOnceCooldownMutexAndStablePriority(t *testing.T) {
	service, _ := gameProjectTestService(t)
	project := createGameProjectForTest(t, service)
	events := []GameProjectEvent{
		{ID: "route_low", Name: "次要路线", EventType: "route", Priority: 10, MutexGroup: "route", Once: true},
		{ID: "route_high", Name: "主要路线", EventType: "route", Priority: 20, MutexGroup: "route"},
		{ID: "follow_up", Name: "后续回访", EventType: "personal", Priority: 5, CooldownTurns: 2},
	}
	saveEventPreviewDraft(t, service, project, events)
	result, err := service.PreviewEvents("tenant-a", project.Project.ID, "", PreviewGameEventsInput{CurrentTurn: 4, OccurredEventIDs: []string{"route_low", "follow_up"}, TriggeredAtTurn: map[string]int{"follow_up": 3}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 3 || result.Items[0].EventID != "route_high" || !result.Items[0].Selected {
		t.Fatalf("priority order or winner is wrong: %#v", result)
	}
	byID := map[string]GameEventPreviewItem{}
	for _, item := range result.Items {
		byID[item.EventID] = item
	}
	if byID["route_low"].Triggered || byID["follow_up"].Triggered {
		t.Fatalf("once and cooldown must block: %#v", byID)
	}

	// Without history, both route events are eligible but only the stable,
	// higher-priority event is selected from their mutual-exclusion group.
	clean, err := service.PreviewEvents("tenant-a", project.Project.ID, "", PreviewGameEventsInput{CurrentTurn: 10})
	if err != nil {
		t.Fatal(err)
	}
	byID = map[string]GameEventPreviewItem{}
	for _, item := range clean.Items {
		byID[item.EventID] = item
	}
	if !byID["route_low"].Triggered || byID["route_low"].Selected || !strings.Contains(strings.Join(byID["route_low"].Reasons, " "), "互斥") {
		t.Fatalf("lower priority mutex event should remain explainable but unselected: %#v", byID["route_low"])
	}
}

func TestGameProjectPublishRejectsBrokenEventReferences(t *testing.T) {
	service, db := gameProjectTestService(t)
	agentRow := agent.AgentConfig{Name: "director", TenantID: "tenant-a", ContentHash: "director-v1", SystemPrompt: "direct"}
	if err := db.Create(&agentRow).Error; err != nil {
		t.Fatal(err)
	}
	project := createGameProjectForTest(t, service)
	draft := completeGameDraft(agentRow.ID)
	draft.Events = []GameProjectEvent{{ID: "broken", Name: "错误事件", EventType: "ambient", ParticipantCharacterIDs: []string{"missing"}, NextEventIDs: []string{"unknown"}, Effects: []GameEffectDefinition{{Operation: "increment", Target: "missing_metric", Value: float64(1)}}}}
	if _, err := service.Update("tenant-a", project.Project.ID, "editor", UpdateGameProjectInput{Revision: 1, Name: project.Project.Name, Description: project.Project.Description, Genre: project.Project.Genre, PlayGoal: project.Project.PlayGoal, Draft: &draft}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Publish("tenant-a", project.Project.ID, "publisher", PublishGameProjectInput{Version: "0.1.0"}); err == nil {
		t.Fatal("expected broken event references to prevent publishing")
	}
}
