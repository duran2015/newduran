package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"control-panel/internal/domain/gameproject"
	rundomain "control-panel/internal/domain/run"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	gameSessionNamespace = "io.zerone.game-session"
	gameSessionSchema    = "world-state"
	gameSessionVersion   = "1.0.0"
)

var (
	ErrGameSessionNotFound  = errors.New("game session not found")
	ErrGameDirectorRequired = errors.New("only the game director may control this session")
)

type GameSessionParticipant struct {
	CharacterID string `json:"characterId"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

type GameSessionLatestEvent struct {
	EventID        string                   `json:"eventId"`
	Name           string                   `json:"name"`
	DisplayContent string                   `json:"displayContent,omitempty"`
	Participants   []GameSessionParticipant `json:"participants"`
	StateDiff      []GameStateDiff          `json:"stateDiff"`
	TimeAdvance    int                      `json:"timeAdvance"`
	EndsGame       bool                     `json:"endsGame"`
	Reasons        []string                 `json:"reasons,omitempty"`
}

type GameSessionDetail struct {
	ID                  string                   `json:"id"`
	RunID               string                   `json:"runId"`
	ProjectID           string                   `json:"projectId"`
	ProjectRevision     int                      `json:"projectRevision"`
	Status              string                   `json:"status"`
	DirectorCharacterID string                   `json:"directorCharacterId"`
	CurrentTurn         int                      `json:"currentTurn"`
	State               map[string]any           `json:"state"`
	LatestEvent         *GameSessionLatestEvent  `json:"latestEvent,omitempty"`
	Participants        []GameSessionParticipant `json:"participants"`
	Timeline            []rundomain.RunActivity  `json:"timeline"`
	CreatedAt           time.Time                `json:"createdAt"`
	UpdatedAt           time.Time                `json:"updatedAt"`
}

type CreateGameSessionInput struct {
	CreatedBy string
}

type AdvanceGameSessionInput struct {
	RequestKey       string `json:"requestKey"`
	ActorCharacterID string `json:"actorCharacterId"`
}

type GameSessionService struct {
	db       *gorm.DB
	projects *GameProjectService
	runs     *RunService
}

func NewGameSessionService(db *gorm.DB, projects *GameProjectService, runs *RunService) *GameSessionService {
	return &GameSessionService{db: db, projects: projects, runs: runs}
}

func (s *GameSessionService) Create(tenantID, projectID string, in CreateGameSessionInput) (*GameSessionDetail, error) {
	detail, err := s.projects.Get(tenantID, projectID)
	if err != nil {
		return nil, err
	}
	var director *GameProjectNPC
	for i := range detail.Draft.NPCs {
		if detail.Draft.NPCs[i].CharacterType == GameCharacterDirector {
			if director != nil {
				return nil, fmt.Errorf("试玩只能设置一位导演")
			}
			director = &detail.Draft.NPCs[i]
		}
	}
	if director == nil || director.AgentID == 0 {
		return nil, fmt.Errorf("请先绑定一位导演 Agent")
	}
	if len(detail.Draft.Metrics) == 0 {
		return nil, fmt.Errorf("请先设置世界状态")
	}

	snapshot := encodeProjectDraft(detail.Draft)
	run, err := s.runs.Create(tenantID, CreateRunInput{
		Name:        detail.Project.Name + " · 试玩",
		Description: detail.Project.PlayGoal,
		CreatedBy:   in.CreatedBy,
		Metadata:    map[string]any{"kind": "game-session", "projectId": projectID, "projectRevision": detail.Project.Revision},
	})
	if err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			s.cleanupRun(tenantID, run.ID)
		}
	}()
	seenAgents := map[uint64]bool{}
	for _, npc := range detail.Draft.NPCs {
		if npc.AgentID == 0 || seenAgents[npc.AgentID] {
			continue
		}
		if _, err = s.runs.AddAgent(tenantID, run.ID, npc.AgentID, npc.CharacterType); err != nil {
			return nil, err
		}
		seenAgents[npc.AgentID] = true
	}
	if err = s.ensureStateSchema(tenantID); err != nil {
		return nil, err
	}
	metrics := map[string]any{}
	for _, metric := range detail.Draft.Metrics {
		metrics[metric.ID] = metric.Initial
	}
	initialData := normalizeJSONMap(map[string]any{"currentTurn": 0, "metrics": metrics, "occurredEventIds": []string{}, "triggeredAtTurn": map[string]int{}})
	state, err := s.runs.InitializeState(tenantID, run.ID, InitializeStateInput{
		Namespace: gameSessionNamespace, SchemaName: gameSessionSchema, SchemaVersion: gameSessionVersion,
		SubjectType: "game", SubjectID: projectID,
		Data:           initialData,
		IdempotencyKey: "game-session:" + run.ID + ":initialize", Reason: "创建试玩局", Source: "game-session",
	})
	if err != nil {
		return nil, err
	}
	_ = state
	session := &gameproject.Session{ID: uuid.NewString(), TenantID: tenantID, RunID: run.ID, ProjectID: projectID, ProjectRevision: detail.Project.Revision, DirectorCharacterID: director.ID, Snapshot: snapshot, CreatedBy: in.CreatedBy}
	if err = s.db.Create(session).Error; err != nil {
		return nil, err
	}
	if _, err = s.runs.Transition(tenantID, run.ID, rundomain.StatusRunning); err != nil {
		return nil, err
	}
	_, _ = s.runs.AppendActivity(tenantID, run.ID, AppendRunActivityInput{Kind: "game.session.started", Status: "completed", ActorType: "character", ActorID: director.ID, Name: "开始试玩", Output: map[string]any{"currentTurn": 0}})
	cleanup = false
	return s.Get(tenantID, projectID, session.ID)
}

func (s *GameSessionService) cleanupRun(tenantID, runID string) {
	_ = s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("tenant_id=? AND run_id=?", tenantID, runID).Delete(&gameproject.Session{})
		tx.Where("tenant_id=? AND run_id=?", tenantID, runID).Delete(&rundomain.RunActivity{})
		tx.Where("tenant_id=? AND run_id=?", tenantID, runID).Delete(&rundomain.RunStateChange{})
		tx.Where("tenant_id=? AND run_id=?", tenantID, runID).Delete(&rundomain.RunState{})
		tx.Where("tenant_id=? AND run_id=?", tenantID, runID).Delete(&rundomain.RunAgent{})
		return tx.Where("tenant_id=? AND id=?", tenantID, runID).Delete(&rundomain.Run{}).Error
	})
}

func (s *GameSessionService) ensureStateSchema(tenantID string) error {
	var existing rundomain.StateSchema
	if err := s.db.Where("tenant_id=? AND namespace=? AND name=? AND version=?", tenantID, gameSessionNamespace, gameSessionSchema, gameSessionVersion).First(&existing).Error; err == nil {
		return nil
	}
	_, err := s.runs.RegisterStateSchema(tenantID, RegisterStateSchemaInput{
		Namespace: gameSessionNamespace, Name: gameSessionSchema, Version: gameSessionVersion,
		ScopeTypes: []string{"run"}, SubjectTypes: []string{"game"},
		Schema: map[string]any{"type": "object", "required": []any{"currentTurn", "metrics", "occurredEventIds", "triggeredAtTurn"}, "properties": map[string]any{"currentTurn": map[string]any{"type": "integer", "minimum": 0}, "metrics": map[string]any{"type": "object"}, "occurredEventIds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "triggeredAtTurn": map[string]any{"type": "object"}, "latestEvent": map[string]any{"type": []any{"object", "null"}}}, "additionalProperties": true},
	})
	if errors.Is(err, rundomain.ErrDuplicate) {
		return nil
	}
	return err
}

func (s *GameSessionService) List(tenantID, projectID string) ([]GameSessionDetail, error) {
	rows := []gameproject.Session{}
	if err := s.db.Where("tenant_id=? AND project_id=?", tenantID, projectID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]GameSessionDetail, 0, len(rows))
	for _, row := range rows {
		d, err := s.Get(tenantID, projectID, row.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, *d)
	}
	return result, nil
}

func (s *GameSessionService) Get(tenantID, projectID, sessionID string) (*GameSessionDetail, error) {
	var session gameproject.Session
	if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, sessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGameSessionNotFound
		}
		return nil, err
	}
	var run rundomain.Run
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, session.RunID).First(&run).Error; err != nil {
		return nil, err
	}
	var state rundomain.RunState
	if err := s.db.Where("tenant_id=? AND run_id=? AND namespace=?", tenantID, session.RunID, gameSessionNamespace).First(&state).Error; err != nil {
		return nil, err
	}
	activities, err := s.runs.ListActivities(tenantID, session.RunID, 500)
	if err != nil {
		return nil, err
	}
	draft, err := decodeProjectDraft(session.Snapshot)
	if err != nil {
		return nil, err
	}
	participants := make([]GameSessionParticipant, 0, len(draft.NPCs))
	for _, npc := range draft.NPCs {
		participants = append(participants, GameSessionParticipant{CharacterID: npc.ID, DisplayName: npc.DisplayName, Role: npc.Role, Status: "等待参与"})
	}
	metrics, _ := state.Data["metrics"].(map[string]any)
	result := &GameSessionDetail{ID: session.ID, RunID: session.RunID, ProjectID: session.ProjectID, ProjectRevision: session.ProjectRevision, Status: run.Status, DirectorCharacterID: session.DirectorCharacterID, CurrentTurn: mapInt(state.Data["currentTurn"]), State: metrics, Participants: participants, Timeline: activities, CreatedAt: session.CreatedAt, UpdatedAt: run.UpdatedAt}
	if raw, ok := state.Data["latestEvent"].(map[string]any); ok {
		result.LatestEvent = decodeLatestEvent(raw)
		active := map[string]bool{}
		for _, participant := range result.LatestEvent.Participants {
			active[participant.CharacterID] = true
		}
		for i := range result.Participants {
			if active[result.Participants[i].CharacterID] {
				result.Participants[i].Status = "本回合参与"
			}
		}
	}
	return result, nil
}

func (s *GameSessionService) Advance(tenantID, projectID, sessionID string, in AdvanceGameSessionInput) (*GameSessionDetail, error) {
	if strings.TrimSpace(in.RequestKey) == "" {
		return nil, fmt.Errorf("requestKey is required")
	}
	key := "game-session:" + sessionID + ":advance:" + strings.TrimSpace(in.RequestKey)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var session gameproject.Session
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, sessionID).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGameSessionNotFound
			}
			return err
		}
		if in.ActorCharacterID != session.DirectorCharacterID {
			return ErrGameDirectorRequired
		}
		var prior rundomain.RunStateChange
		if err := tx.Where("tenant_id=? AND idempotency_key=?", tenantID, key).First(&prior).Error; err == nil {
			return nil
		}
		var run rundomain.Run
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND id=?", tenantID, session.RunID).First(&run).Error; err != nil {
			return err
		}
		if run.Status != rundomain.StatusRunning {
			return fmt.Errorf("只有进行中的试玩可以推进")
		}
		var state rundomain.RunState
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND run_id=? AND namespace=?", tenantID, session.RunID, gameSessionNamespace).First(&state).Error; err != nil {
			return err
		}
		draft, err := decodeProjectDraft(session.Snapshot)
		if err != nil {
			return err
		}
		metrics, _ := state.Data["metrics"].(map[string]any)
		if metrics == nil {
			metrics = map[string]any{}
		}
		turn := mapInt(state.Data["currentTurn"])
		occurred := stringSet(state.Data["occurredEventIds"])
		triggered := intMap(state.Data["triggeredAtTurn"])
		defs := metricDefinitions(draft.Metrics)
		items := []GameEventPreviewItem{}
		for _, event := range draft.Events {
			item, e := previewProjectEvent(event, defs, metrics, turn, occurred, triggered)
			if e != nil {
				return e
			}
			items = append(items, item)
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].Priority != items[j].Priority {
				return items[i].Priority > items[j].Priority
			}
			return items[i].EventID < items[j].EventID
		})
		claimed := map[string]bool{}
		selected := []GameEventPreviewItem{}
		for _, item := range items {
			if !item.Triggered {
				continue
			}
			if item.MutexGroup != "" && claimed[item.MutexGroup] {
				continue
			}
			if item.MutexGroup != "" {
				claimed[item.MutexGroup] = true
			}
			selected = append(selected, item)
		}
		before := cloneMap(metrics)
		latest := (*GameSessionLatestEvent)(nil)
		maxAdvance := 1
		ends := false
		participantSet := map[string]bool{}
		for selectedIndex := range selected {
			item := &selected[selectedIndex]
			occurred[item.EventID] = true
			triggered[item.EventID] = turn
			if item.TimeAdvance > maxAdvance {
				maxAdvance = item.TimeAdvance
			}
			if item.EndsGame {
				ends = true
			}
			var source *GameProjectEvent
			for i := range draft.Events {
				if draft.Events[i].ID == item.EventID {
					source = &draft.Events[i]
					break
				}
			}
			partIDs := []string{}
			content := ""
			if source != nil {
				partIDs = eventParticipantIDs(*source)
				content = eventContent(*source)
				// Conditions are evaluated against the start-of-turn snapshot, but
				// selected event effects settle in stable priority order. This keeps
				// two independent increments additive instead of letting the later
				// event overwrite the earlier event's result.
				item.StateDiff = []GameStateDiff{}
				for _, effect := range source.Effects {
					metric := defs[effect.Target]
					beforeValue := metrics[effect.Target]
					afterValue, applyErr := applyGameEffect(metric, beforeValue, effect.Value, effect.Operation)
					if applyErr != nil {
						return applyErr
					}
					metrics[effect.Target] = afterValue
					item.StateDiff = append(item.StateDiff, GameStateDiff{MetricID: effect.Target, Before: beforeValue, After: afterValue})
				}
				for _, p := range partIDs {
					participantSet[p] = true
				}
			}
			parts := participantDetails(draft, partIDs)
			latest = &GameSessionLatestEvent{EventID: item.EventID, Name: item.Name, DisplayContent: content, Participants: parts, StateDiff: item.StateDiff, TimeAdvance: item.TimeAdvance, EndsGame: item.EndsGame, Reasons: item.Reasons}
			if item.EndsGame {
				selected = selected[:selectedIndex+1]
				break
			}
		}
		newTurn := turn + maxAdvance
		occurredIDs := make([]string, 0, len(occurred))
		for id := range occurred {
			occurredIDs = append(occurredIDs, id)
		}
		sort.Strings(occurredIDs)
		data := map[string]any{"currentTurn": newTurn, "metrics": metrics, "occurredEventIds": occurredIDs, "triggeredAtTurn": triggered}
		if latest != nil {
			data["latestEvent"] = latest
		}
		next := state.Revision + 1
		encoded, err := json.Marshal(data)
		if err != nil {
			return err
		}
		res := tx.Model(&rundomain.RunState{}).Where("tenant_id=? AND id=? AND revision=?", tenantID, state.ID, state.Revision).Updates(map[string]any{"data": string(encoded), "revision": next, "updated_at": time.Now().UTC()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return rundomain.ErrConflict
		}
		change := rundomain.RunStateChange{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, RunStateID: state.ID, RevisionBefore: state.Revision, RevisionAfter: next, Before: state.Data, After: data, Reason: "导演推进回合", Source: "game-session", IdempotencyKey: key}
		if err := tx.Create(&change).Error; err != nil {
			return err
		}
		selectedNames := []string{}
		for _, v := range selected {
			selectedNames = append(selectedNames, v.Name)
		}
		parts := []string{}
		for p := range participantSet {
			parts = append(parts, p)
		}
		sort.Strings(parts)
		activity := rundomain.RunActivity{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, Kind: "game.turn.advanced", Status: "completed", ActorType: "character", ActorID: in.ActorCharacterID, Name: fmt.Sprintf("推进到第 %d 回合", newTurn), Input: map[string]any{"requestKey": in.RequestKey, "turn": turn}, Output: map[string]any{"before": before, "after": metrics, "events": selectedNames, "participants": parts, "selected": selected, "currentTurn": newTurn}, OccurredAt: time.Now().UTC()}
		if err := tx.Create(&activity).Error; err != nil {
			return err
		}
		if ends {
			now := time.Now().UTC()
			if err := tx.Model(&rundomain.Run{}).Where("tenant_id=? AND id=?", tenantID, session.RunID).Updates(map[string]any{"status": rundomain.StatusCompleted, "completed_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
			completed := rundomain.RunActivity{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, Kind: "game.session.completed", Status: "completed", ActorType: "character", ActorID: in.ActorCharacterID, Name: "事件触发游戏结束", Output: map[string]any{"currentTurn": newTurn}, OccurredAt: now}
			if err := tx.Create(&completed).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(tenantID, projectID, sessionID)
}

func (s *GameSessionService) Pause(t, p, id, actor string) (*GameSessionDetail, error) {
	return s.transition(t, p, id, actor, rundomain.StatusPaused, "暂停试玩")
}
func (s *GameSessionService) Resume(t, p, id, actor string) (*GameSessionDetail, error) {
	return s.transition(t, p, id, actor, rundomain.StatusRunning, "继续试玩")
}
func (s *GameSessionService) End(t, p, id, actor string) (*GameSessionDetail, error) {
	return s.transition(t, p, id, actor, rundomain.StatusCompleted, "结束试玩")
}
func (s *GameSessionService) transition(tenantID, projectID, sessionID, actor, target, name string) (*GameSessionDetail, error) {
	var session gameproject.Session
	if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, sessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGameSessionNotFound
		}
		return nil, err
	}
	if actor != session.DirectorCharacterID {
		return nil, ErrGameDirectorRequired
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var run rundomain.Run
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND id=?", tenantID, session.RunID).First(&run).Error; err != nil {
			return err
		}
		if !rundomain.ValidTransitions[run.Status][target] {
			return rundomain.ErrInvalidTransition
		}
		now := time.Now().UTC()
		updates := map[string]any{"status": target, "updated_at": now}
		switch target {
		case rundomain.StatusRunning:
			if run.StartedAt == nil {
				updates["started_at"] = now
			}
			updates["paused_at"] = nil
		case rundomain.StatusPaused:
			updates["paused_at"] = now
		case rundomain.StatusCompleted:
			updates["completed_at"] = now
		}
		res := tx.Model(&rundomain.Run{}).Where("tenant_id=? AND id=? AND status=?", tenantID, session.RunID, run.Status).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return rundomain.ErrConflict
		}
		activity := rundomain.RunActivity{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, Kind: "game.session." + target, Status: "completed", ActorType: "character", ActorID: actor, Name: name, OccurredAt: now}
		return tx.Create(&activity).Error
	})
	if err != nil {
		return nil, err
	}
	return s.Get(tenantID, projectID, sessionID)
}

func mapInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}
func stringSet(v any) map[string]bool {
	r := map[string]bool{}
	switch x := v.(type) {
	case []string:
		for _, s := range x {
			r[s] = true
		}
	case []any:
		for _, s := range x {
			if q, ok := s.(string); ok {
				r[q] = true
			}
		}
	}
	return r
}
func intMap(v any) map[string]int {
	r := map[string]int{}
	if x, ok := v.(map[string]any); ok {
		for k, n := range x {
			r[k] = mapInt(n)
		}
	}
	if x, ok := v.(map[string]int); ok {
		for k, n := range x {
			r[k] = n
		}
	}
	return r
}
func decodeLatestEvent(raw map[string]any) *GameSessionLatestEvent {
	e := &GameSessionLatestEvent{}
	encoded, _ := json.Marshal(raw)
	_ = json.Unmarshal(encoded, e)
	return e
}

func participantDetails(draft GameProjectDraft, ids []string) []GameSessionParticipant {
	out := make([]GameSessionParticipant, 0, len(ids))
	for _, id := range ids {
		for _, npc := range draft.NPCs {
			if npc.ID == id {
				out = append(out, GameSessionParticipant{CharacterID: id, DisplayName: npc.DisplayName, Role: npc.Role, Status: "本回合参与"})
				break
			}
		}
	}
	return out
}
func normalizeJSONMap(value map[string]any) map[string]any {
	raw, _ := json.Marshal(value)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}
