package services

import (
	"context"
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

var ErrGamePlayerPlanNotFound = errors.New("player plan not found")

type CreateGamePlayerSessionInput struct {
	Source          string `json:"source"`
	ProjectRevision int    `json:"projectRevision,omitempty"`
	ReleaseID       string `json:"releaseId,omitempty"`
	StartTurn       int    `json:"startTurn,omitempty"`
	StartEventID    string `json:"startEventId,omitempty"`
	CreatedBy       string `json:"-"`
}

type InterpretGamePlayerInput struct {
	Text              string `json:"text"`
	RequestKey        string `json:"requestKey"`
	TargetCharacterID string `json:"targetCharacterId,omitempty"`
}

type ConfirmGamePlayerInput struct {
	PlanID     string `json:"planId"`
	RequestKey string `json:"requestKey"`
}

type RejectGamePlayerInput struct {
	PlanID string `json:"planId"`
}

type DecideGameObjectiveInput struct {
	ObjectiveID string `json:"objectiveId"`
	RequestKey  string `json:"requestKey"`
}

type GamePlayerEffect struct {
	MetricID string `json:"metricId"`
	Before   any    `json:"before"`
	After    any    `json:"after"`
}

type GamePlayerPlan struct {
	NeedsClarification bool               `json:"needsClarification"`
	ClarifyingQuestion string             `json:"clarifyingQuestion,omitempty"`
	Degraded           bool               `json:"degraded"`
	ID                 string             `json:"id"`
	InputText          string             `json:"inputText"`
	Summary            string             `json:"summary"`
	ActionID           string             `json:"actionId,omitempty"`
	Effects            []GamePlayerEffect `json:"effects"`
	Warnings           []string           `json:"warnings"`
	NeedsConfirmation  bool               `json:"needsConfirmation"`
	Status             string             `json:"status"`
}

const gameFreeformActionID = "__freeform_interaction__"

type GamePlayerTimelineItem struct {
	SceneID           string             `json:"sceneId,omitempty"`
	CharacterID       string             `json:"characterId,omitempty"`
	TargetCharacterID string             `json:"targetCharacterId,omitempty"`
	Label             string             `json:"label"`
	Content           string             `json:"content"`
	Speaker           string             `json:"speaker,omitempty"`
	OccurredAt        time.Time          `json:"occurredAt"`
	StateDiff         []GamePlayerEffect `json:"stateDiff,omitempty"`
}

type GamePlayerInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type GamePlayerSceneCharacter struct {
	CharacterID string  `json:"characterId"`
	Name        string  `json:"name"`
	PortraitURL string  `json:"portraitUrl,omitempty"`
	Mood        string  `json:"mood,omitempty"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Scale       float64 `json:"scale"`
	Layer       int     `json:"layer"`
	Visible     bool    `json:"visible"`
}
type GamePlayerScene struct {
	ID            string                     `json:"id"`
	Name          string                     `json:"name"`
	Description   string                     `json:"description,omitempty"`
	Order         int                        `json:"order"`
	BackgroundURL string                     `json:"backgroundUrl,omitempty"`
	MusicURL      string                     `json:"musicUrl,omitempty"`
	SoundURLs     []string                   `json:"soundUrls"`
	Characters    []GamePlayerSceneCharacter `json:"characters"`
}

type GamePlayerSessionDetail struct {
	CurrentSceneID  string                   `json:"currentSceneId,omitempty"`
	StateLabels     map[string]string        `json:"stateLabels"`
	Ending          *GamePlayerEnding        `json:"ending,omitempty"`
	RunID           string                   `json:"-"`
	LatestEvent     *GameSessionLatestEvent  `json:"latestEvent,omitempty"`
	ID              string                   `json:"id"`
	Game            GamePlayerInfo           `json:"game"`
	ProjectID       string                   `json:"projectId"`
	Source          string                   `json:"source"`
	ProjectRevision int                      `json:"projectRevision"`
	ReleaseID       string                   `json:"releaseId,omitempty"`
	Status          string                   `json:"status"`
	CurrentTurn     int                      `json:"currentTurn"`
	StartTurn       int                      `json:"startTurn"`
	StartEventID    string                   `json:"startEventId,omitempty"`
	State           map[string]any           `json:"state"`
	Objectives      []GamePlayerObjective    `json:"objectives"`
	PendingPlan     *GamePlayerPlan          `json:"pendingPlan,omitempty"`
	Timeline        []GamePlayerTimelineItem `json:"timeline"`
	Scenes          []GamePlayerScene        `json:"scenes"`
	CreatedAt       time.Time                `json:"createdAt"`
	UpdatedAt       time.Time                `json:"updatedAt"`
}
type GamePlayerObjective struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Kind         string `json:"kind"`
	Status       string `json:"status"`
	Issuer       string `json:"issuer,omitempty"`
	DeadlineTurn int    `json:"deadlineTurn,omitempty"`
	PromiseText  string `json:"promiseText,omitempty"`
}
type GamePlayerEnding struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}
type GamePlayerReleaseDetail struct {
	Game            GamePlayerInfo    `json:"game"`
	ReleaseID       string            `json:"releaseId"`
	Version         string            `json:"version"`
	ProjectRevision int               `json:"projectRevision"`
	InitialState    map[string]any    `json:"initialState"`
	Scenes          []GamePlayerScene `json:"scenes"`
}

type GamePlayerSessionService struct {
	actionModelFactory func(string) GameActionLanguageModel
	db                 *gorm.DB
	projects           *GameProjectService
	packages           *GamePackageService
	sessions           *GameSessionService
	runs               *RunService
	director           *GameDirectorBridge
}

func (s *GamePlayerSessionService) SetActionModelFactory(factory func(string) GameActionLanguageModel) {
	s.actionModelFactory = factory
}

func NewGamePlayerSessionService(db *gorm.DB, projects *GameProjectService, packages *GamePackageService, sessions *GameSessionService, runs *RunService) *GamePlayerSessionService {
	return &GamePlayerSessionService{db: db, projects: projects, packages: packages, sessions: sessions, runs: runs}
}

func (s *GamePlayerSessionService) ProjectForRelease(tenantID, releaseID string) (string, error) {
	var row gameproject.Release
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, releaseID).First(&row).Error; err != nil {
		return "", ErrGameSessionNotFound
	}
	return row.ProjectID, nil
}

func (s *GamePlayerSessionService) ValidateReleaseSession(tenantID, releaseID, sessionID string) error {
	var count int64
	if err := s.db.Model(&gameproject.Session{}).Where("tenant_id=? AND id=? AND source='release' AND release_id=?", tenantID, sessionID, releaseID).Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return ErrGameSessionNotFound
	}
	return nil
}

func (s *GamePlayerSessionService) ReleaseDetail(tenantID, releaseID string) (*GamePlayerReleaseDetail, error) {
	var release gameproject.Release
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, releaseID).First(&release).Error; err != nil {
		return nil, ErrGameSessionNotFound
	}
	pkg, err := s.packages.Get(tenantID, release.PackageID)
	if err != nil {
		return nil, err
	}
	draft := draftFromPackage(pkg.Manifest)
	var project gameproject.Project
	_ = s.db.Where("tenant_id=? AND id=?", tenantID, release.ProjectID).First(&project).Error
	state := map[string]any{}
	for _, m := range draft.Metrics {
		state[m.ID] = m.Initial
	}
	session := gameproject.Session{Source: "release", ReleaseID: release.ID, ProjectID: release.ProjectID}
	return &GamePlayerReleaseDetail{Game: GamePlayerInfo{Name: project.Name, Description: project.Description}, ReleaseID: release.ID, Version: release.Version, ProjectRevision: release.ProjectRevision, InitialState: state, Scenes: playerScenes(session, draft)}, nil
}

func (s *GamePlayerSessionService) Create(tenantID, projectID string, in CreateGamePlayerSessionInput) (*GamePlayerSessionDetail, error) {
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = "draft"
	}
	if source != "draft" && source != "release" {
		return nil, fmt.Errorf("请选择试玩当前草稿或已发布版本")
	}
	if source == "release" && strings.TrimSpace(in.ReleaseID) == "" {
		return nil, fmt.Errorf("请选择要游玩的发布版本")
	}
	if source == "release" {
		var release gameproject.Release
		if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, in.ReleaseID).First(&release).Error; err != nil {
			return nil, fmt.Errorf("选择的发布版本不存在")
		}
		pkg, err := s.packages.Get(tenantID, release.PackageID)
		if err != nil || pkg.Manifest.Content == nil {
			return nil, fmt.Errorf("发布版本内容不可用")
		}
		draft := draftFromPackage(pkg.Manifest)
		detail, err := s.createFromSnapshot(tenantID, projectID, release.ProjectRevision, in.ReleaseID, source, draft, in.StartTurn, in.StartEventID, in.CreatedBy)
		if err != nil {
			return nil, err
		}
		return s.Get(tenantID, projectID, detail.ID)
	}
	project, err := s.projects.Get(tenantID, projectID)
	if err != nil {
		return nil, err
	}
	if in.ProjectRevision > 0 && in.ProjectRevision != project.Project.Revision {
		return nil, ErrGameProjectConflict
	}
	detail, err := s.createFromSnapshot(tenantID, projectID, project.Project.Revision, "", source, project.Draft, in.StartTurn, in.StartEventID, in.CreatedBy)
	if err != nil {
		return nil, err
	}
	return s.Get(tenantID, projectID, detail.ID)
}

func draftFromPackage(m GamePackageManifest) GameProjectDraft {
	d := emptyGameProjectDraft()
	d.Metrics, d.Actions, d.Endings = m.Registry.Metrics, m.Registry.Actions, m.Registry.Endings
	if m.Content == nil {
		return d
	}
	d.Objectives = append([]GameObjectiveDefinition{}, m.Content.Objectives...)
	for _, n := range m.Content.NPCs {
		d.NPCs = append(d.NPCs, GameProjectNPC{ID: n.ID, AgentID: n.AgentID, DisplayName: n.DisplayName, Role: n.Role, Responsibilities: n.Responsibilities, CharacterType: n.CharacterType, VisibleMetricIDs: n.VisibleMetricIDs, ExecutableActionIDs: n.ExecutableActionIDs, AdvisoryActionIDs: n.AdvisoryActionIDs, KnownInformation: n.KnownInformation, InformationScope: n.InformationScope, KnowledgeRefs: n.KnowledgeRefs, RelationIDs: n.RelationIDs, Permissions: n.Permissions})
	}
	for _, e := range m.Content.StoryEvents {
		d.Events = append(d.Events, GameProjectEvent{SceneID: e.SceneID, ID: e.ID, Name: e.Name, Description: e.Description, Category: e.Category, DisplayContent: e.DisplayContent, ConditionMode: e.ConditionMode, Conditions: e.Conditions, ParticipantCharacterIDs: e.ParticipantCharacterIDs, Effects: e.Effects, TimeAdvance: e.TimeAdvance, Priority: e.Priority, Once: e.Once, CooldownTurns: e.CooldownTurns, ExclusiveGroup: e.ExclusiveGroup, NextEventIDs: e.NextEventIDs, EndsGame: e.EndsGame, EndReason: e.EndReason})
	}
	for _, sc := range m.Content.Scenes {
		d.Scenes = append(d.Scenes, GameProjectScene{ID: sc.ID, Name: sc.Name, Description: sc.Description, Order: sc.Order, BackgroundAssetID: sc.BackgroundAssetID, MusicAssetID: sc.MusicAssetID, SoundEffectAssetIDs: sc.SoundEffectAssetIDs, Characters: sc.Characters})
	}
	return d
}

func eventIncomingIDs(events []GameProjectEvent) map[string]bool {
	incoming := make(map[string]bool)
	for _, event := range events {
		for _, nextID := range event.NextEventIDs {
			incoming[nextID] = true
		}
	}
	return incoming
}

// applyEventFlowEligibility keeps authored story chains monotonic. Conditions
// answer "may this happen in this situation?" while nextEventIds answers "may
// this happen at this point in the story?". An event that is part of a chain
// must be explicitly opened by an event that already happened. Standalone
// ambient events remain condition-driven for backwards compatibility.
func applyEventFlowEligibility(item *GameEventPreviewItem, event GameProjectEvent, events []GameProjectEvent, occurred map[string]bool, emittedEventID string) {
	if item == nil || !eventIncomingIDs(events)[event.ID] {
		return
	}
	// Player actions are authored commands that may deliberately jump to a
	// branch (for example, hiding funds opens the evidence-risk route while
	// preserving evidence opens the financial-review route).
	if strings.TrimSpace(emittedEventID) == event.ID {
		return
	}
	unlocked := false
	for _, previous := range events {
		if !occurred[previous.ID] {
			continue
		}
		for _, nextID := range previous.NextEventIDs {
			if nextID == event.ID {
				unlocked = true
				break
			}
		}
		if unlocked {
			break
		}
	}
	if unlocked {
		return
	}
	item.Triggered = false
	item.Eligible = false
	item.Reasons = append(item.Reasons, "故事还没有推进到这一幕")
}

func (s *GamePlayerSessionService) createFromSnapshot(tenantID, projectID string, revision int, releaseID, source string, draft GameProjectDraft, startTurn int, startEventID, actor string) (*gameproject.Session, error) {
	var director *GameProjectNPC
	for i := range draft.NPCs {
		if draft.NPCs[i].CharacterType == GameCharacterDirector {
			director = &draft.NPCs[i]
			break
		}
	}
	if director == nil {
		return nil, fmt.Errorf("请先设置一位导演")
	}
	if len(draft.Metrics) == 0 || len(draft.Actions) == 0 {
		return nil, fmt.Errorf("请先完成世界状态和玩家行动")
	}
	if startTurn < 0 {
		return nil, fmt.Errorf("起始回合不能小于零")
	}
	var opening *GameProjectEvent
	if strings.TrimSpace(startEventID) != "" {
		for i := range draft.Events {
			if draft.Events[i].ID == startEventID {
				opening = &draft.Events[i]
				break
			}
		}
		if opening == nil {
			return nil, fmt.Errorf("选择的起始事件不存在")
		}
	} else {
		// “从头开始”应该真正进入故事开场，而不是只加载一组没有
		// 上下文的初始数值。根事件（没有被其他事件指向）中优先级最高的
		// 一个就是默认开场；没有编排事件的旧项目仍可以按原方式开始。
		incoming := eventIncomingIDs(draft.Events)
		for i := range draft.Events {
			candidate := &draft.Events[i]
			// Only scene-backed events can serve as a visible opening. Older
			// projects often use root events as turn-one rules; consuming one at
			// session creation would change hard state before the player's first
			// confirmed action.
			if strings.TrimSpace(candidate.SceneID) == "" {
				continue
			}
			if incoming[candidate.ID] {
				continue
			}
			if opening == nil || candidate.Priority > opening.Priority {
				opening = candidate
			}
		}
		if opening != nil {
			startEventID = opening.ID
		}
	}
	run, err := s.runs.Create(tenantID, CreateRunInput{Name: "玩家试玩", Description: "玩家使用自然语言推进故事", CreatedBy: actor, Metadata: map[string]any{"kind": "game-player-session", "projectId": projectID, "source": source, "releaseId": releaseID}})
	if err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			s.sessions.cleanupRun(tenantID, run.ID)
		}
	}()
	if err = s.sessions.ensureStateSchema(tenantID); err != nil {
		return nil, err
	}
	metrics := map[string]any{}
	for _, m := range draft.Metrics {
		metrics[m.ID] = m.Initial
	}
	occurred := []string{}
	triggered := map[string]int{}
	if opening != nil {
		for _, effect := range opening.Effects {
			def, ok := metricDefinitions(draft.Metrics)[effect.Target]
			if !ok {
				continue
			}
			after, e := applyGameEffect(def, metrics[effect.Target], effect.Value, effect.Operation)
			if e != nil {
				return nil, e
			}
			metrics[effect.Target] = after
		}
		occurred = append(occurred, opening.ID)
		triggered[opening.ID] = startTurn
	}
	objectiveStatuses := initialObjectiveStatuses(draft.Objectives)
	objectiveActivatedTurns := initialObjectiveActivatedTurns(draft.Objectives, startTurn)
	if _, err = s.runs.InitializeState(tenantID, run.ID, InitializeStateInput{Namespace: gameSessionNamespace, SchemaName: gameSessionSchema, SchemaVersion: gameSessionVersion, SubjectType: "game", SubjectID: projectID, Data: normalizeJSONMap(map[string]any{"currentTurn": startTurn, "metrics": metrics, "occurredEventIds": occurred, "triggeredAtTurn": triggered, "objectiveStatuses": objectiveStatuses, "objectiveActivatedTurns": objectiveActivatedTurns}), IdempotencyKey: "player-session:" + run.ID + ":initialize", Reason: "开始玩家试玩", Source: "player-session"}); err != nil {
		return nil, err
	}
	row := &gameproject.Session{ID: uuid.NewString(), TenantID: tenantID, RunID: run.ID, ProjectID: projectID, ProjectRevision: revision, Source: source, ReleaseID: releaseID, StartTurn: startTurn, StartEventID: startEventID, DirectorCharacterID: director.ID, Snapshot: encodeProjectDraft(draft), CreatedBy: actor}
	if err = s.db.Create(row).Error; err != nil {
		return nil, err
	}
	if _, err = s.runs.Transition(tenantID, run.ID, rundomain.StatusRunning); err != nil {
		return nil, err
	}
	_, _ = s.runs.AppendActivity(tenantID, run.ID, AppendRunActivityInput{Kind: "game.player.started", Status: "completed", Name: "开始游戏", Output: map[string]any{"summary": "游戏已经开始，请用自然语言说出你想做什么。"}})
	if opening != nil {
		_, _ = s.runs.AppendActivity(tenantID, run.ID, AppendRunActivityInput{Kind: "game.event.opening", Status: "completed", Name: "从指定事件开始", Output: map[string]any{"summary": "从“" + opening.Name + "”开始试玩，事件影响已计入初始局面。"}})
	}
	cleanup = false
	return row, nil
}

func (s *GamePlayerSessionService) Get(tenantID, projectID, sessionID string) (*GamePlayerSessionDetail, error) {
	var row gameproject.Session
	if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, sessionID).First(&row).Error; err != nil {
		return nil, ErrGameSessionNotFound
	}
	var run rundomain.Run
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, row.RunID).First(&run).Error; err != nil {
		return nil, err
	}
	var state rundomain.RunState
	if err := s.db.Where("tenant_id=? AND run_id=? AND namespace=?", tenantID, row.RunID, gameSessionNamespace).First(&state).Error; err != nil {
		return nil, err
	}
	draft, err := decodeProjectDraft(row.Snapshot)
	if err != nil {
		return nil, err
	}
	activities, err := s.runs.ListActivities(tenantID, row.RunID, 500)
	if err != nil {
		return nil, err
	}
	timeline := make([]GamePlayerTimelineItem, 0, len(activities))
	currentSceneID := ""
	if len(draft.Scenes) > 0 {
		currentSceneID = draft.Scenes[0].ID
	}
	for _, event := range draft.Events {
		if event.ID == row.StartEventID && event.SceneID != "" {
			currentSceneID = event.SceneID
		}
	}
	for _, a := range activities {
		label, content := playerActivityCopy(a)
		speaker := ""
		characterID := ""
		targetCharacterID := ""
		if a.Kind == "game.player.interpreted" {
			if text, ok := a.Input["text"].(string); ok {
				label = "你说"
				content = text
				speaker = "你"
			}
			targetCharacterID, _ = a.Input["targetCharacterId"].(string)
		}
		if a.Kind == "game.character.response" {
			characterID = a.ActorID
			for _, character := range draft.NPCs {
				if character.ID == a.ActorID {
					speaker = character.DisplayName
					break
				}
			}
		}
		sceneID, _ := a.Output["sceneId"].(string)
		if sceneID != "" {
			currentSceneID = sceneID
		}
		if a.Kind == "game.event.triggered" {
			if story, ok := a.Output["content"].(string); ok && story != "" {
				content = story
			}
		}
		timeline = append(timeline, GamePlayerTimelineItem{SceneID: sceneID, CharacterID: characterID, TargetCharacterID: targetCharacterID, Label: label, Content: content, Speaker: speaker, OccurredAt: a.OccurredAt, StateDiff: playerActivityDiff(a)})
	}
	var pending gameproject.PlayerPlan
	var plan *GamePlayerPlan
	if err = s.db.Where("tenant_id=? AND session_id=? AND status='pending'", tenantID, row.ID).Order("created_at DESC").First(&pending).Error; err == nil {
		p := playerPlanDTO(pending)
		plan = &p
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	metrics, _ := state.Data["metrics"].(map[string]any)
	var latest *GameSessionLatestEvent
	if value := state.Data["latestEvent"]; value != nil {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &latest); err != nil {
			return nil, err
		}
	}
	var project gameproject.Project
	_ = s.db.Where("tenant_id=? AND id=?", tenantID, row.ProjectID).First(&project).Error
	labels := map[string]string{}
	if value, ok := state.Data["currentSceneId"].(string); ok && value != "" {
		currentSceneID = value
	}
	for _, metric := range draft.Metrics {
		labels[metric.ID] = metric.Name
	}
	var ending *GamePlayerEnding
	if run.Status == rundomain.StatusCompleted {
		ending = &GamePlayerEnding{Title: "本局结束", Summary: "本局的行动和结果已保存在过程回顾中。"}
		if latest != nil && latest.EndsGame {
			ending.Title = latest.Name
			ending.Summary = latest.DisplayContent
		}
		if value := state.Data["ending"]; value != nil {
			raw, _ := json.Marshal(value)
			_ = json.Unmarshal(raw, ending)
		}
	}
	moods := map[string]string{}
	if s.director != nil && s.director.persona.emotion != nil {
		for _, character := range draft.NPCs {
			if character.AgentID == 0 {
				continue
			}
			if status, moodErr := s.director.persona.emotion.Status(tenantID, row.RunID, character.AgentID); moodErr == nil {
				moods[character.ID] = status.Mood
			}
		}
	}
	return &GamePlayerSessionDetail{CurrentSceneID: currentSceneID, StateLabels: labels, Ending: ending, RunID: row.RunID, LatestEvent: latest, ID: row.ID, Game: GamePlayerInfo{Name: project.Name, Description: project.Description}, ProjectID: row.ProjectID, Source: defaultString(row.Source, "draft"), ProjectRevision: row.ProjectRevision, ReleaseID: row.ReleaseID, Status: run.Status, CurrentTurn: mapInt(state.Data["currentTurn"]), StartTurn: row.StartTurn, StartEventID: row.StartEventID, State: metrics, Objectives: playerObjectives(draft, stringMap(state.Data["objectiveStatuses"]), intMap(state.Data["objectiveActivatedTurns"])), PendingPlan: plan, Timeline: timeline, Scenes: playerScenes(row, draft, moods), CreatedAt: row.CreatedAt, UpdatedAt: run.UpdatedAt}, nil
}

// Reset creates a fresh play-through from the exact same locked source. The
// old session remains available for comparison and audit.
func (s *GamePlayerSessionService) Reset(tenantID, projectID, sessionID, actor string) (*GamePlayerSessionDetail, error) {
	var row gameproject.Session
	if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, sessionID).First(&row).Error; err != nil {
		return nil, ErrGameSessionNotFound
	}
	draft, err := decodeProjectDraft(row.Snapshot)
	if err != nil {
		return nil, err
	}
	created, err := s.createFromSnapshot(tenantID, projectID, row.ProjectRevision, row.ReleaseID, defaultString(row.Source, "draft"), draft, row.StartTurn, row.StartEventID, actor)
	if err != nil {
		return nil, err
	}
	return s.Get(tenantID, projectID, created.ID)
}

func playerScenes(session gameproject.Session, draft GameProjectDraft, moodSets ...map[string]string) []GamePlayerScene {
	moods := map[string]string{}
	if len(moodSets) > 0 && moodSets[0] != nil {
		moods = moodSets[0]
	}
	names := map[string]string{}
	for _, n := range draft.NPCs {
		names[n.ID] = n.DisplayName
	}
	assetURL := func(id string) string {
		if id == "" {
			return ""
		}
		if session.Source == "release" {
			return "/api/v1/games/" + session.ReleaseID + "/assets/" + id
		}
		return "/api/v1/admin/game-projects/" + session.ProjectID + "/assets/" + id + "/content"
	}
	out := []GamePlayerScene{}
	for _, sc := range draft.Scenes {
		item := GamePlayerScene{ID: sc.ID, Name: sc.Name, Description: sc.Description, Order: sc.Order, BackgroundURL: assetURL(sc.BackgroundAssetID), MusicURL: assetURL(sc.MusicAssetID), SoundURLs: []string{}, Characters: []GamePlayerSceneCharacter{}}
		for _, id := range sc.SoundEffectAssetIDs {
			item.SoundURLs = append(item.SoundURLs, assetURL(id))
		}
		for _, c := range sc.Characters {
			mood := moods[c.CharacterID]
			portraitID := c.PortraitAssetID
			if variantID := c.EmotionPortraitAssetIDs[mood]; variantID != "" {
				portraitID = variantID
			}
			item.Characters = append(item.Characters, GamePlayerSceneCharacter{CharacterID: c.CharacterID, Name: names[c.CharacterID], PortraitURL: assetURL(portraitID), Mood: mood, X: c.X, Y: c.Y, Scale: c.Scale, Layer: c.Layer, Visible: c.Visible})
		}
		out = append(out, item)
	}
	return out
}

func playerActivityCopy(a rundomain.RunActivity) (string, string) {
	if v, ok := a.Output["summary"].(string); ok && v != "" {
		return defaultString(a.Name, "游戏进展"), v
	}
	switch a.Kind {
	case "game.player.started":
		return "游戏开始", "世界状态已载入。"
	case "game.player.interpreted":
		return "等待确认", "系统已把玩家输入整理成待确认行动，尚未改变游戏状态。"
	case "game.player.confirmed":
		return "行动已执行", "玩家确认后，系统才完成规则结算。人物尚未回应。"
	case "game.player.rejected":
		return "已放弃行动", "玩家放弃了这次行动，世界状态没有变化。"
	}
	return "游戏进展", "系统记录了一次真实发生的游戏进展。"
}

func playerActivityDiff(a rundomain.RunActivity) []GamePlayerEffect {
	raw, ok := a.Output["stateDiff"].([]any)
	if !ok {
		return nil
	}
	out := []GamePlayerEffect{}
	for _, v := range raw {
		if m, ok := v.(map[string]any); ok {
			out = append(out, GamePlayerEffect{MetricID: fmt.Sprint(m["metricId"]), Before: m["before"], After: m["after"]})
		}
	}
	return out
}

func playerPlanDTO(row gameproject.PlayerPlan) GamePlayerPlan {
	effects := []GamePlayerEffect{}
	if raw, ok := row.Effects["items"].([]any); ok {
		for _, v := range raw {
			if m, ok := v.(map[string]any); ok {
				effects = append(effects, GamePlayerEffect{MetricID: fmt.Sprint(m["metricId"]), Before: m["before"], After: m["after"]})
			}
		}
	}
	warnings := []string{}
	if row.ActionID == "" {
		warnings = append(warnings, "没有找到明确对应的玩家行动，请换一种说法或返回编辑器补充行动。")
	}
	degraded, _ := row.Effects["degraded"].(bool)
	question, _ := row.Effects["clarifyingQuestion"].(string)
	return GamePlayerPlan{ID: row.ID, InputText: row.InputText, Summary: row.Summary, ActionID: row.ActionID, Effects: effects, Warnings: warnings, NeedsConfirmation: row.ActionID != "", NeedsClarification: row.ActionID == "", ClarifyingQuestion: question, Degraded: degraded, Status: row.Status}
}

func (s *GamePlayerSessionService) Interpret(tenantID, projectID, sessionID string, in InterpretGamePlayerInput) (*GamePlayerSessionDetail, error) {
	if strings.TrimSpace(in.Text) == "" || strings.TrimSpace(in.RequestKey) == "" {
		return nil, fmt.Errorf("请填写玩家想做的事情")
	}
	if len([]rune(strings.TrimSpace(in.Text))) > 1000 {
		return nil, fmt.Errorf("一次最多输入 1000 个字")
	}
	var session gameproject.Session
	if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, sessionID).First(&session).Error; err != nil {
		return nil, ErrGameSessionNotFound
	}
	var existing gameproject.PlayerPlan
	if err := s.db.Where("tenant_id=? AND session_id=? AND request_key=?", tenantID, sessionID, in.RequestKey).First(&existing).Error; err == nil {
		return s.Get(tenantID, projectID, sessionID)
	}
	var pendingCount int64
	if err := s.db.Model(&gameproject.PlayerPlan{}).Where("tenant_id=? AND session_id=? AND status='pending'", tenantID, sessionID).Count(&pendingCount).Error; err != nil {
		return nil, err
	}
	if pendingCount > 0 {
		return nil, fmt.Errorf("请先确认或放弃上一个行动")
	}
	draft, err := decodeProjectDraft(session.Snapshot)
	if err != nil {
		return nil, err
	}
	if in.TargetCharacterID != "" {
		validTarget := false
		for _, character := range draft.NPCs {
			if character.ID == in.TargetCharacterID && character.CharacterType == GameCharacterNPC && character.AgentID != 0 {
				validTarget = true
				break
			}
		}
		if !validTarget {
			return nil, fmt.Errorf("这个人物当前无法对话")
		}
	}
	var state rundomain.RunState
	if err = s.db.Where("tenant_id=? AND run_id=? AND namespace=?", tenantID, session.RunID, gameSessionNamespace).First(&state).Error; err != nil {
		return nil, err
	}
	metrics, _ := state.Data["metrics"].(map[string]any)
	actions := make([]GameActionInterpretationAction, 0, len(draft.Actions))
	for _, a := range draft.Actions {
		preview, previewErr := calculatePlayerAction(a, metricDefinitions(draft.Metrics), metrics)
		if previewErr != nil {
			return nil, previewErr
		}
		actions = append(actions, GameActionInterpretationAction{ID: a.ID, Name: a.Name, Description: a.Description, Allowed: preview.Allowed})
	}
	request := GameActionInterpretationRequest{PlayerText: in.Text, Actions: actions, Context: GameActionInterpretationContext{VisibleState: metrics, PlayerRole: "玩家"}}
	// Rules determine authoritative state changes; they must not limit what a
	// player may attempt. Resolve strong rule matches locally and route every
	// other expression to the director/NPC layer as a free interaction.
	interpretation, err := NewGameActionInterpreter(nil).Interpret(context.Background(), request)
	if err != nil {
		return nil, err
	}
	var chosen *GameActionDefinition
	for i := range draft.Actions {
		if !interpretation.NeedsClarification && !interpretation.Ambiguous && draft.Actions[i].ID == interpretation.ActionID {
			chosen = &draft.Actions[i]
			break
		}
	}
	summary := defaultString(interpretation.ClarifyingQuestion, interpretation.Summary)
	actionID := ""
	items := []map[string]any{}
	if chosen != nil {
		preview, err := calculatePlayerAction(*chosen, metricDefinitions(draft.Metrics), metrics)
		if err != nil {
			return nil, err
		}
		actionID = chosen.ID
		summary = "准备执行“" + chosen.Name + "”。确认前不会改变游戏状态。"
		for _, d := range preview.StateDiff {
			items = append(items, map[string]any{"metricId": d.MetricID, "before": d.Before, "after": d.After})
		}
	} else {
		actionID = gameFreeformActionID
		summary = "尝试自由互动：“" + strings.TrimSpace(in.Text) + "”。人物会根据身份、关系和当前处境作出回应；这不会自动改写世界数值。"
		interpretation.ActionID = gameFreeformActionID
		interpretation.Ambiguous = false
		interpretation.NeedsClarification = false
		interpretation.ClarifyingQuestion = ""
		interpretation.Summary = summary
	}
	row := gameproject.PlayerPlan{ID: uuid.NewString(), TenantID: tenantID, SessionID: sessionID, RequestKey: in.RequestKey, InputText: strings.TrimSpace(in.Text), Summary: summary, ActionID: actionID, Effects: map[string]any{"items": items, "degraded": interpretation.Degraded, "clarifyingQuestion": interpretation.ClarifyingQuestion, "interpretation": interpretation, "targetCharacterId": in.TargetCharacterID}, Status: "pending"}
	if err = s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	_, _ = s.runs.AppendActivity(tenantID, session.RunID, AppendRunActivityInput{Kind: "game.player.interpreted", Status: "completed", Name: "理解玩家行动", Input: map[string]any{"text": row.InputText, "targetCharacterId": in.TargetCharacterID}, Output: map[string]any{"summary": summary}})
	return s.Get(tenantID, projectID, sessionID)
}

func (s *GamePlayerSessionService) Confirm(tenantID, projectID, sessionID string, in ConfirmGamePlayerInput) (*GamePlayerSessionDetail, error) {
	if in.PlanID == "" || in.RequestKey == "" {
		return nil, fmt.Errorf("请选择要确认的行动")
	}
	key := "player-confirm:" + strings.TrimSpace(in.RequestKey)
	var participation *GameDirectorTurnInput
	shouldComplete := false
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var session gameproject.Session
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, sessionID).First(&session).Error; err != nil {
			return ErrGameSessionNotFound
		}
		var prior rundomain.RunStateChange
		if err := tx.Where("tenant_id=? AND idempotency_key=?", tenantID, key).First(&prior).Error; err == nil {
			return nil
		}
		var plan gameproject.PlayerPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND session_id=? AND id=?", tenantID, sessionID, in.PlanID).First(&plan).Error; err != nil {
			return ErrGamePlayerPlanNotFound
		}
		if plan.Status == "confirmed" {
			return nil
		}
		if plan.Status != "pending" {
			return fmt.Errorf("这个行动已经处理")
		}
		if plan.ActionID == "" {
			return fmt.Errorf("当前内容还不能执行，请重新描述玩家行动")
		}
		var run rundomain.Run
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND id=?", tenantID, session.RunID).First(&run).Error; err != nil {
			return err
		}
		if run.Status != rundomain.StatusRunning {
			return fmt.Errorf("只有进行中的游戏可以执行行动")
		}
		var state rundomain.RunState
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND run_id=? AND namespace=?", tenantID, session.RunID, gameSessionNamespace).First(&state).Error; err != nil {
			return err
		}
		draft, err := decodeProjectDraft(session.Snapshot)
		if err != nil {
			return err
		}
		var action *GameActionDefinition
		if plan.ActionID == gameFreeformActionID {
			action = &GameActionDefinition{ID: gameFreeformActionID, Name: "自由互动", Description: "由人物根据身份、关系和局势回应", TimeCost: 1}
		} else {
			for i := range draft.Actions {
				if draft.Actions[i].ID == plan.ActionID {
					action = &draft.Actions[i]
					break
				}
			}
		}
		if action == nil {
			return fmt.Errorf("这个行动已不可用")
		}
		metrics, _ := state.Data["metrics"].(map[string]any)
		if metrics == nil {
			metrics = map[string]any{}
		}
		preview, err := calculatePlayerAction(*action, metricDefinitions(draft.Metrics), metrics)
		if err != nil {
			return err
		}
		if !preview.Allowed {
			reasons := make([]string, 0, len(preview.Checks))
			for _, check := range preview.Checks {
				if !check.Matched && strings.TrimSpace(check.Reason) != "" {
					reasons = append(reasons, check.Reason)
				}
			}
			if len(reasons) > 0 {
				return fmt.Errorf("现在还不能执行“%s”：%s", action.Name, strings.Join(reasons, "；"))
			}
			return fmt.Errorf("现在还不能执行“%s”，请先推进当前局面", action.Name)
		}
		metrics = cloneMap(preview.After)
		turn := mapInt(state.Data["currentTurn"]) + max(1, action.TimeCost)
		occurred := stringSet(state.Data["occurredEventIds"])
		triggered := intMap(state.Data["triggeredAtTurn"])
		defs := metricDefinitions(draft.Metrics)
		items := []GameEventPreviewItem{}
		for _, event := range draft.Events {
			item, e := previewProjectEvent(event, defs, metrics, turn, occurred, triggered)
			if e != nil {
				return e
			}
			applyEventFlowEligibility(&item, event, draft.Events, occurred, action.Emits)
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
		ends := false
		eventAdvance := 0
		var latest *GameSessionLatestEvent
		for i := range selected {
			item := &selected[i]
			occurred[item.EventID] = true
			triggered[item.EventID] = turn
			if item.TimeAdvance > eventAdvance {
				eventAdvance = item.TimeAdvance
			}
			var source *GameProjectEvent
			for j := range draft.Events {
				if draft.Events[j].ID == item.EventID {
					source = &draft.Events[j]
					break
				}
			}
			if source == nil {
				continue
			}
			item.StateDiff = []GameStateDiff{}
			for _, effect := range source.Effects {
				def := defs[effect.Target]
				before := metrics[effect.Target]
				after, e := applyGameEffect(def, before, effect.Value, effect.Operation)
				if e != nil {
					return e
				}
				metrics[effect.Target] = after
				item.StateDiff = append(item.StateDiff, GameStateDiff{MetricID: effect.Target, Before: before, After: after})
			}
			latest = &GameSessionLatestEvent{EventID: item.EventID, Name: item.Name, DisplayContent: eventContent(*source), Participants: participantDetails(draft, eventParticipantIDs(*source)), StateDiff: item.StateDiff, TimeAdvance: item.TimeAdvance, EndsGame: item.EndsGame, Reasons: item.Reasons}
			if item.EndsGame {
				ends = true
				selected = selected[:i+1]
				break
			}
		}
		// A story-closing event can be unlocked by an event that happened in
		// this same action (for example crisis_resolution opening an ending).
		// The first preview above intentionally evaluates a stable pre-event
		// snapshot, so run one focused ending pass after event effects have been
		// applied. Players should not have to spend a meaningless extra turn
		// merely to let an already-decided ending appear.
		if !ends {
			endingItems := make([]GameEventPreviewItem, 0)
			for _, event := range draft.Events {
				if !eventEndsGame(event) || occurred[event.ID] {
					continue
				}
				item, e := previewProjectEvent(event, defs, metrics, turn, occurred, triggered)
				if e != nil {
					return e
				}
				applyEventFlowEligibility(&item, event, draft.Events, occurred, action.Emits)
				if item.Triggered {
					endingItems = append(endingItems, item)
				}
			}
			sort.SliceStable(endingItems, func(i, j int) bool {
				if endingItems[i].Priority != endingItems[j].Priority {
					return endingItems[i].Priority > endingItems[j].Priority
				}
				return endingItems[i].EventID < endingItems[j].EventID
			})
			if len(endingItems) > 0 {
				ending := endingItems[0]
				occurred[ending.EventID] = true
				triggered[ending.EventID] = turn
				selected = append(selected, ending)
				ends = true
				for i := range draft.Events {
					if draft.Events[i].ID != ending.EventID {
						continue
					}
					latest = &GameSessionLatestEvent{EventID: ending.EventID, Name: ending.Name, DisplayContent: eventContent(draft.Events[i]), Participants: participantDetails(draft, eventParticipantIDs(draft.Events[i])), EndsGame: true, Reasons: ending.Reasons}
					break
				}
			}
		}
		turn += eventAdvance
		occurredIDs := make([]string, 0, len(occurred))
		for id := range occurred {
			occurredIDs = append(occurredIDs, id)
		}
		sort.Strings(occurredIDs)
		next := normalizeJSONMap(state.Data)
		next["metrics"] = metrics
		next["currentTurn"] = turn
		next["occurredEventIds"] = occurredIDs
		next["triggeredAtTurn"] = triggered
		objectiveStatuses, activatedTurns, objectiveChanges, objectiveDiffs, objectiveErr := settleObjectivesAtTurn(draft.Objectives, stringMap(state.Data["objectiveStatuses"]), intMap(state.Data["objectiveActivatedTurns"]), metrics, turn, defs)
		if objectiveErr != nil {
			return objectiveErr
		}
		next["objectiveStatuses"] = objectiveStatuses
		next["objectiveActivatedTurns"] = activatedTurns
		next["metrics"] = metrics
		if latest != nil {
			next["latestEvent"] = latest
		}
		for _, item := range selected {
			for _, event := range draft.Events {
				if event.ID != item.EventID {
					continue
				}
				if event.SceneID != "" {
					next["currentSceneId"] = event.SceneID
				}
				if item.EndsGame {
					next["ending"] = map[string]any{"title": event.Name, "summary": defaultString(event.EndReason, eventContent(event))}
				}
			}
		}
		now := time.Now().UTC()
		for _, change := range objectiveChanges {
			kind, name, summary := "game.objective.activated", "新任务", "新的任务已可以开始。"
			if change.Status == "completed" {
				kind, name, summary = "game.objective.completed", "任务完成", "你已经完成“"+change.Name+"”。"
			} else if change.Status == "failed" {
				kind, name, summary = "game.objective.failed", "任务失败", "任务“"+change.Name+"”已经失败。"
				if change.PromiseText != "" {
					kind, name, summary = "game.objective.promise_broken", "承诺未兑现", fmt.Sprintf("你没有兑现承诺：“%s”。", change.PromiseText)
				}
			}
			if err = tx.Create(&rundomain.RunActivity{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, Kind: kind, Status: "completed", Name: name, Output: map[string]any{"summary": summary, "objectiveId": change.ID, "stateDiff": gameDiffMaps(objectiveDiffs[change.ID])}, OccurredAt: now}).Error; err != nil {
				return err
			}
		}
		encoded, err := json.Marshal(next)
		if err != nil {
			return err
		}
		revision := state.Revision + 1
		res := tx.Model(&rundomain.RunState{}).Where("tenant_id=? AND id=? AND revision=?", tenantID, state.ID, state.Revision).Updates(map[string]any{"data": string(encoded), "revision": revision, "updated_at": time.Now().UTC()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return rundomain.ErrConflict
		}
		if err = tx.Create(&rundomain.RunStateChange{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, RunStateID: state.ID, RevisionBefore: state.Revision, RevisionAfter: revision, Before: state.Data, After: next, Reason: "确认玩家行动并推进故事", Source: "player-session", IdempotencyKey: key}).Error; err != nil {
			return err
		}
		actionDiff := gameDiffMaps(preview.StateDiff)
		confirmedSummary := "已执行“" + action.Name + "”。"
		if plan.ActionID == gameFreeformActionID {
			confirmedSummary = "你尝试了：“" + plan.InputText + "”。现在由在场人物作出回应。"
		}
		if err = tx.Create(&rundomain.RunActivity{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, Kind: "game.player.confirmed", Status: "completed", Name: "执行玩家行动", Output: map[string]any{"summary": confirmedSummary, "stateDiff": actionDiff, "currentTurn": turn}, OccurredAt: now}).Error; err != nil {
			return err
		}
		for _, item := range selected {
			output := map[string]any{"summary": "故事事件“" + item.Name + "”已经发生。", "eventId": item.EventID, "reasons": item.Reasons, "stateDiff": gameDiffMaps(item.StateDiff), "currentTurn": turn}
			for _, event := range draft.Events {
				if event.ID == item.EventID {
					output["sceneId"] = event.SceneID
					output["content"] = eventContent(event)
					output["summary"] = defaultString(eventContent(event), output["summary"].(string))
				}
			}
			if err = tx.Create(&rundomain.RunActivity{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, Kind: "game.event.triggered", Status: "completed", Name: item.Name, Output: output, OccurredAt: now}).Error; err != nil {
				return err
			}
		}
		if err = tx.Create(&rundomain.RunActivity{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, Kind: "game.turn.advanced", Status: "completed", Name: fmt.Sprintf("推进到第 %d 回合", turn), Output: map[string]any{"summary": fmt.Sprintf("行动和事件结算完成，进入第 %d 回合。", turn), "currentTurn": turn}, OccurredAt: now}).Error; err != nil {
			return err
		}
		// Ending is finalized only after the director and NPCs have responded.
		// RunService intentionally rejects new activities on a completed run, so
		// completing it here would make the final character scene disappear from
		// the player timeline.
		shouldComplete = ends
		result := tx.Model(&gameproject.PlayerPlan{}).Where("tenant_id=? AND id=? AND status='pending'", tenantID, plan.ID).Updates(map[string]any{"status": "confirmed", "confirmed_by_key": in.RequestKey})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return rundomain.ErrConflict
		}
		targetCharacterID, _ := plan.Effects["targetCharacterId"].(string)
		participation = &GameDirectorTurnInput{TenantID: tenantID, RunID: session.RunID, SessionID: session.ID, InteractionID: plan.ID, PlayerText: plan.InputText, ActionID: plan.ActionID, TargetCharacterID: targetCharacterID, State: cloneMap(metrics), ObjectiveStatuses: objectiveStatuses, Draft: draft}
		if len(selected) > 0 {
			combined := &GameProjectEvent{Name: "本轮发生的故事事件"}
			for _, item := range selected {
				for _, event := range draft.Events {
					if event.ID == item.EventID {
						combined.ParticipantCharacterIDs = append(combined.ParticipantCharacterIDs, eventParticipantIDs(event)...)
					}
				}
			}
			participation.Event = combined
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if participation != nil {
		s.participateAfterConfirmation(*participation)
	}
	if shouldComplete && participation != nil {
		now := time.Now().UTC()
		if err := s.db.Transaction(func(tx *gorm.DB) error {
			var run rundomain.Run
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND id=?", tenantID, participation.RunID).First(&run).Error; err != nil {
				return err
			}
			if run.Status == rundomain.StatusCompleted {
				return nil
			}
			if err := tx.Create(&rundomain.RunActivity{ID: uuid.NewString(), TenantID: tenantID, RunID: participation.RunID, Kind: "game.session.completed", Status: "completed", Name: "故事结束", Output: map[string]any{"summary": "人物已完成本轮回应，本局游戏结束。"}, OccurredAt: now}).Error; err != nil {
				return err
			}
			return tx.Model(&rundomain.Run{}).Where("tenant_id=? AND id=?", tenantID, participation.RunID).Updates(map[string]any{"status": rundomain.StatusCompleted, "completed_at": now, "updated_at": now}).Error
		}); err != nil {
			return nil, err
		}
	}
	return s.Get(tenantID, projectID, sessionID)
}

func gameDiffMaps(items []GameStateDiff) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, d := range items {
		out = append(out, map[string]any{"metricId": d.MetricID, "before": d.Before, "after": d.After})
	}
	return out
}

func initialObjectiveStatuses(defs []GameObjectiveDefinition) map[string]string {
	statuses := make(map[string]string, len(defs))
	for _, def := range defs {
		if def.StartsActive {
			statuses[def.ID] = "active"
		} else {
			statuses[def.ID] = "locked"
		}
	}
	return statuses
}

func initialObjectiveActivatedTurns(defs []GameObjectiveDefinition, turn int) map[string]int {
	result := map[string]int{}
	for _, def := range defs {
		if def.StartsActive {
			result[def.ID] = turn
		}
	}
	return result
}

func stringMap(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for key, item := range typed {
			out[key] = item
		}
	case map[string]any:
		for key, item := range typed {
			if text, ok := item.(string); ok {
				out[key] = text
			}
		}
	}
	return out
}

func settleObjectives(defs []GameObjectiveDefinition, current map[string]string, metrics map[string]any) (map[string]string, []GamePlayerObjective) {
	statuses, _, changes, _, _ := settleObjectivesAtTurn(defs, current, nil, metrics, 0, nil)
	return statuses, changes
}

// settleObjectivesAtTurn is the authoritative task state machine. Model output
// can activate an authored task, but completion, failure, deadlines and their
// consequences are evaluated here from persisted state and declared rules.
func settleObjectivesAtTurn(defs []GameObjectiveDefinition, current map[string]string, activated map[string]int, metrics map[string]any, turn int, metricDefs map[string]GameMetricDefinition) (map[string]string, map[string]int, []GamePlayerObjective, map[string][]GameStateDiff, error) {
	statuses := map[string]string{}
	for key, value := range current {
		statuses[key] = value
	}
	if len(statuses) == 0 {
		statuses = initialObjectiveStatuses(defs)
	}
	activationTurns := map[string]int{}
	for key, value := range activated {
		activationTurns[key] = value
	}
	for _, def := range defs {
		if statuses[def.ID] == "active" {
			if _, ok := activationTurns[def.ID]; !ok {
				activationTurns[def.ID] = turn
			}
		}
	}
	changes := []GamePlayerObjective{}
	diffs := map[string][]GameStateDiff{}
	for pass := 0; pass < len(defs)+1; pass++ {
		changed := false
		for _, def := range defs {
			status := statuses[def.ID]
			if status == "" {
				status = "locked"
				statuses[def.ID] = status
			}
			if status == "locked" && def.IssuerCharacterID == "" && len(def.DependsOn) > 0 && objectiveDependenciesCompleted(def.DependsOn, statuses) {
				statuses[def.ID] = "active"
				activationTurns[def.ID] = turn
				status, changed = "active", true
				changes = append(changes, GamePlayerObjective{ID: def.ID, Name: def.Name, Status: "active", PromiseText: def.PromiseText})
			}
			if status == "active" && objectiveConditionsMatched(def, metrics) {
				statuses[def.ID] = "completed"
				changed = true
				change := GamePlayerObjective{ID: def.ID, Name: def.Name, Status: "completed", PromiseText: def.PromiseText}
				objectiveDiff, err := applyObjectiveEffects(def.CompletedEffects, metricDefs, metrics)
				if err != nil {
					return nil, nil, nil, nil, err
				}
				diffs[def.ID] = objectiveDiff
				changes = append(changes, change)
				continue
			}
			if status == "active" && objectiveFailed(def, metrics, turn, activationTurns[def.ID]) {
				statuses[def.ID] = "failed"
				changed = true
				objectiveDiff, err := applyObjectiveEffects(def.FailedEffects, metricDefs, metrics)
				if err != nil {
					return nil, nil, nil, nil, err
				}
				diffs[def.ID] = objectiveDiff
				changes = append(changes, GamePlayerObjective{ID: def.ID, Name: def.Name, Status: "failed", PromiseText: def.PromiseText})
			}
		}
		if !changed {
			break
		}
	}
	return statuses, activationTurns, changes, diffs, nil
}

func objectiveFailed(def GameObjectiveDefinition, metrics map[string]any, turn, activatedTurn int) bool {
	if len(def.FailureConditions) > 0 && conditionsMatched(def.FailureConditions, def.FailureConditionMode, metrics) {
		return true
	}
	return def.DeadlineTurns > 0 && turn > activatedTurn+def.DeadlineTurns
}

func applyObjectiveEffects(effects []GameEffectDefinition, defs map[string]GameMetricDefinition, metrics map[string]any) ([]GameStateDiff, error) {
	result := []GameStateDiff{}
	for _, effect := range effects {
		def, ok := defs[effect.Target]
		if !ok {
			// Old snapshots may carry no consequences or definitions. A declared
			// consequence must never silently invent a world-state field.
			return nil, fmt.Errorf("任务后果引用了不存在的世界状态 %s", effect.Target)
		}
		before := metrics[effect.Target]
		after, err := applyGameEffect(def, before, effect.Value, effect.Operation)
		if err != nil {
			return nil, err
		}
		metrics[effect.Target] = after
		result = append(result, GameStateDiff{MetricID: effect.Target, Before: before, After: after})
	}
	return result, nil
}

func objectiveDependenciesCompleted(ids []string, statuses map[string]string) bool {
	if len(ids) == 0 {
		return true
	}
	for _, id := range ids {
		if statuses[id] != "completed" {
			return false
		}
	}
	return true
}

func objectiveConditionsMatched(def GameObjectiveDefinition, metrics map[string]any) bool {
	return conditionsMatched(def.Conditions, def.ConditionMode, metrics)
}

func conditionsMatched(conditions []GameConditionDefinition, mode string, metrics map[string]any) bool {
	if len(conditions) == 0 {
		return false
	}
	matched := 0
	for _, condition := range conditions {
		ok, err := matchGameCondition(metrics[condition.Target], condition.Value, condition.Operator)
		if err == nil && ok {
			matched++
		}
	}
	if mode == "any" {
		return matched > 0
	}
	return matched == len(conditions)
}

func playerObjectives(draft GameProjectDraft, statuses map[string]string, activatedTurns map[string]int) []GamePlayerObjective {
	characterNames := map[string]string{}
	for _, character := range draft.NPCs {
		characterNames[character.ID] = character.DisplayName
	}
	out := make([]GamePlayerObjective, 0, len(draft.Objectives))
	for _, def := range draft.Objectives {
		status := statuses[def.ID]
		if status == "" {
			status = "locked"
		}
		deadlineTurn := 0
		if activatedTurn, ok := activatedTurns[def.ID]; def.DeadlineTurns > 0 && ok && status == "active" {
			deadlineTurn = activatedTurn + def.DeadlineTurns
		}
		out = append(out, GamePlayerObjective{ID: def.ID, Name: def.Name, Description: def.Description, Kind: defaultString(def.Kind, "main"), Status: status, Issuer: characterNames[def.IssuerCharacterID], DeadlineTurn: deadlineTurn, PromiseText: def.PromiseText})
	}
	return out
}

func (s *GamePlayerSessionService) Reject(tenantID, projectID, sessionID string, in RejectGamePlayerInput) (*GamePlayerSessionDetail, error) {
	var session gameproject.Session
	if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, sessionID).First(&session).Error; err != nil {
		return nil, ErrGameSessionNotFound
	}
	res := s.db.Model(&gameproject.PlayerPlan{}).Where("tenant_id=? AND session_id=? AND id=? AND status='pending'", tenantID, sessionID, in.PlanID).Update("status", "rejected")
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected != 1 {
		return nil, ErrGamePlayerPlanNotFound
	}
	_, _ = s.runs.AppendActivity(tenantID, session.RunID, AppendRunActivityInput{Kind: "game.player.rejected", Status: "completed", Name: "放弃行动", Output: map[string]any{"summary": "玩家放弃了这次行动，世界状态没有变化。"}})
	return s.Get(tenantID, projectID, sessionID)
}

func (s *GamePlayerSessionService) AcceptObjective(tenantID, projectID, sessionID string, in DecideGameObjectiveInput) (*GamePlayerSessionDetail, error) {
	return s.decideObjective(tenantID, projectID, sessionID, in, true)
}

func (s *GamePlayerSessionService) RejectObjective(tenantID, projectID, sessionID string, in DecideGameObjectiveInput) (*GamePlayerSessionDetail, error) {
	return s.decideObjective(tenantID, projectID, sessionID, in, false)
}

func (s *GamePlayerSessionService) decideObjective(tenantID, projectID, sessionID string, in DecideGameObjectiveInput, accept bool) (*GamePlayerSessionDetail, error) {
	objectiveID, requestKey := strings.TrimSpace(in.ObjectiveID), strings.TrimSpace(in.RequestKey)
	if objectiveID == "" || requestKey == "" {
		return nil, fmt.Errorf("请选择任务并提供请求标识")
	}
	decision := "reject"
	if accept {
		decision = "accept"
	}
	idempotencyKey := "player-objective:" + sessionID + ":" + objectiveID + ":" + decision + ":" + requestKey
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var session gameproject.Session
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, sessionID).First(&session).Error; err != nil {
			return ErrGameSessionNotFound
		}
		var prior rundomain.RunStateChange
		if err := tx.Where("tenant_id=? AND idempotency_key=?", tenantID, idempotencyKey).First(&prior).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		draft, err := decodeProjectDraft(session.Snapshot)
		if err != nil {
			return err
		}
		var objective *GameObjectiveDefinition
		for i := range draft.Objectives {
			if draft.Objectives[i].ID == objectiveID {
				objective = &draft.Objectives[i]
				break
			}
		}
		if objective == nil {
			return fmt.Errorf("这个任务不存在")
		}
		var state rundomain.RunState
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id=? AND run_id=? AND namespace=?", tenantID, session.RunID, gameSessionNamespace).First(&state).Error; err != nil {
			return err
		}
		statuses := stringMap(state.Data["objectiveStatuses"])
		if statuses[objectiveID] != "offered" {
			return fmt.Errorf("这个任务当前不在等待选择")
		}
		activatedTurns := intMap(state.Data["objectiveActivatedTurns"])
		status := "rejected"
		if accept {
			status = "active"
			activatedTurns[objectiveID] = mapInt(state.Data["currentTurn"])
		} else {
			delete(activatedTurns, objectiveID)
		}
		statuses[objectiveID] = status
		next := normalizeJSONMap(state.Data)
		next["objectiveStatuses"] = statuses
		next["objectiveActivatedTurns"] = activatedTurns
		encoded, err := json.Marshal(next)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		revision := state.Revision + 1
		res := tx.Model(&rundomain.RunState{}).Where("tenant_id=? AND id=? AND revision=?", tenantID, state.ID, state.Revision).Updates(map[string]any{"data": string(encoded), "revision": revision, "updated_at": now})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return rundomain.ErrConflict
		}
		if err := tx.Create(&rundomain.RunStateChange{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, RunStateID: state.ID, RevisionBefore: state.Revision, RevisionAfter: revision, Before: state.Data, After: next, Reason: "玩家选择任务", Source: "player-session", IdempotencyKey: idempotencyKey}).Error; err != nil {
			return err
		}
		kind, name, summary := "game.objective.rejected", "拒绝任务", "你拒绝了任务“"+objective.Name+"”。任务发布人之后仍可重新提出。"
		output := map[string]any{"summary": summary, "objectiveId": objective.ID, "decision": status}
		if accept {
			kind, name, summary = "game.objective.accepted", "接受任务", "你接受了任务“"+objective.Name+"”。"
			if objective.PromiseText != "" {
				kind, name, summary = "game.objective.promised", "作出承诺", fmt.Sprintf("你作出了承诺：“%s”。", objective.PromiseText)
				output["promise"] = objective.PromiseText
			}
			if objective.DeadlineTurns > 0 {
				output["deadlineTurn"] = activatedTurns[objectiveID] + objective.DeadlineTurns
			}
		}
		output["summary"] = summary
		return tx.Create(&rundomain.RunActivity{ID: uuid.NewString(), TenantID: tenantID, RunID: session.RunID, Kind: kind, Status: "completed", Name: name, Output: output, OccurredAt: now}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.Get(tenantID, projectID, sessionID)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func calculatePlayerAction(action GameActionDefinition, metrics map[string]GameMetricDefinition, state map[string]any) (*PreviewGameActionResult, error) {
	result := &PreviewGameActionResult{ActionID: action.ID, Before: cloneMap(state), After: cloneMap(state), Reasons: []string{}, Checks: []GameConditionCheck{}, StateDiff: []GameStateDiff{}}
	mode := defaultString(action.ConditionMode, "all")
	matchedCount := 0
	for _, condition := range action.Conditions {
		metric, ok := metrics[condition.Target]
		if !ok {
			return nil, fmt.Errorf("行动检查了未定义的世界状态")
		}
		actual := state[condition.Target]
		matched, err := matchGameCondition(actual, condition.Value, condition.Operator)
		if err != nil {
			return nil, err
		}
		if matched {
			matchedCount++
		}
		result.Checks = append(result.Checks, GameConditionCheck{Target: condition.Target, Operator: condition.Operator, Expected: condition.Value, Actual: actual, Matched: matched, Reason: conditionReason(metric, condition, actual, matched)})
	}
	result.Allowed = len(action.Conditions) == 0 || (mode == "all" && matchedCount == len(action.Conditions)) || (mode == "any" && matchedCount > 0)
	if !result.Allowed {
		return result, nil
	}
	for _, effect := range action.Effects {
		metric, ok := metrics[effect.Target]
		if !ok {
			return nil, fmt.Errorf("行动修改了未定义的世界状态")
		}
		before := result.After[effect.Target]
		after, err := applyGameEffect(metric, before, effect.Value, effect.Operation)
		if err != nil {
			return nil, err
		}
		result.After[effect.Target] = after
		result.StateDiff = append(result.StateDiff, GameStateDiff{MetricID: effect.Target, Before: before, After: after})
	}
	return result, nil
}
