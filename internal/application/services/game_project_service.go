package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"control-panel/internal/domain/agent"
	"control-panel/internal/domain/agentrelation"
	"control-panel/internal/domain/gameasset"
	"control-panel/internal/domain/gameproject"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrGameProjectNotFound = errors.New("game project not found")
	ErrGameProjectConflict = errors.New("game project has been changed; reload it before saving")
	projectKeyPattern      = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*$`)
)

type GameProjectNPC struct {
	ID                  string   `json:"id"`
	AgentID             uint64   `json:"agentId"`
	DisplayName         string   `json:"displayName"`
	Role                string   `json:"role"`
	Responsibilities    string   `json:"responsibilities,omitempty"`
	CharacterType       string   `json:"characterType"`
	VisibleMetricIDs    []string `json:"visibleMetricIds,omitempty"`
	ExecutableActionIDs []string `json:"executableActionIds,omitempty"`
	AdvisoryActionIDs   []string `json:"advisoryActionIds,omitempty"`
	KnownInformation    []string `json:"knownInformation,omitempty"`
	InformationScope    string   `json:"informationScope,omitempty"`
	KnowledgeRefs       []string `json:"knowledgeRefs,omitempty"`
	RelationIDs         []uint64 `json:"relationIds,omitempty"`
	Permissions         []string `json:"permissions,omitempty"`
}

const (
	GameCharacterDirector = "director"
	GameCharacterNPC      = "npc"
	GameCharacterObserver = "observer"
)

type GameCharacterContextPreview struct {
	CharacterID       string   `json:"characterId"`
	DisplayName       string   `json:"displayName"`
	Identity          string   `json:"identity"`
	Responsibilities  string   `json:"responsibilities,omitempty"`
	AgentName         string   `json:"agentName"`
	AgentAvailable    bool     `json:"agentAvailable"`
	AvailableSummary  string   `json:"availableSummary"`
	VisibleWorldState []string `json:"visibleWorldState"`
	CanExecute        []string `json:"canExecute"`
	CanSuggest        []string `json:"canSuggest"`
	KnownInformation  []string `json:"knownInformation"`
	KnowledgeSummary  string   `json:"knowledgeSummary"`
	Relationships     []string `json:"relationships"`
	Limits            []string `json:"limits"`
}

type GameProjectEvent struct {
	SceneID                 string                    `json:"sceneId,omitempty"`
	ID                      string                    `json:"id"`
	Name                    string                    `json:"name"`
	Description             string                    `json:"description,omitempty"`
	Category                string                    `json:"category,omitempty"`
	DisplayContent          string                    `json:"displayContent,omitempty"`
	EventType               string                    `json:"eventType,omitempty"`
	Content                 string                    `json:"content,omitempty"`
	Trigger                 map[string]any            `json:"trigger,omitempty"` // 兼容早期草稿；新内容使用 Conditions。
	ConditionMode           string                    `json:"conditionMode,omitempty"`
	Conditions              []GameConditionDefinition `json:"conditions,omitempty"`
	ParticipantCharacterIDs []string                  `json:"participantCharacterIds,omitempty"`
	ParticipantIDs          []string                  `json:"participantIds,omitempty"` // 兼容早期草稿
	Effects                 []GameEffectDefinition    `json:"effects,omitempty"`
	AdvancesTime            bool                      `json:"advancesTime,omitempty"`
	TimeAdvance             int                       `json:"timeAdvance,omitempty"`
	AdvanceTurns            int                       `json:"advanceTurns,omitempty"`
	Priority                int                       `json:"priority,omitempty"`
	Once                    bool                      `json:"once,omitempty"`
	CooldownTurns           int                       `json:"cooldownTurns,omitempty"`
	ExclusiveGroup          string                    `json:"exclusiveGroup,omitempty"`
	MutexGroup              string                    `json:"mutexGroup,omitempty"`
	NextEventIDs            []string                  `json:"nextEventIds,omitempty"`
	EndGame                 bool                      `json:"endGame,omitempty"`
	EndsGame                bool                      `json:"endsGame,omitempty"` // 兼容早期草稿
	EndReason               string                    `json:"endReason,omitempty"`
}

type GameProjectScene struct {
	ID                  string               `json:"id"`
	Name                string               `json:"name"`
	Description         string               `json:"description,omitempty"`
	Order               int                  `json:"order,omitempty"`
	BackgroundAssetID   string               `json:"backgroundAssetId,omitempty"`
	MusicAssetID        string               `json:"musicAssetId,omitempty"`
	SoundEffectAssetIDs []string             `json:"soundEffectAssetIds,omitempty"`
	Characters          []GameSceneCharacter `json:"characters,omitempty"`
	// Deprecated E0 compatibility fields. New drafts use typed asset IDs and
	// character layers above.
	BackgroundRef   string            `json:"backgroundRef,omitempty"`
	PortraitRefs    map[string]string `json:"portraitRefs,omitempty"`
	MusicRef        string            `json:"musicRef,omitempty"`
	SoundEffectRefs []string          `json:"soundEffectRefs,omitempty"`
}

type GameSceneCharacter struct {
	ID                      string            `json:"id"`
	CharacterID             string            `json:"characterId"`
	PortraitAssetID         string            `json:"portraitAssetId,omitempty"`
	EmotionPortraitAssetIDs map[string]string `json:"emotionPortraitAssetIds,omitempty"`
	X                       float64           `json:"x"`
	Y                       float64           `json:"y"`
	Scale                   float64           `json:"scale"`
	Layer                   int               `json:"layer"`
	Visible                 bool              `json:"visible"`
}

type SaveGameSceneInput struct {
	Revision int              `json:"revision"`
	Scene    GameProjectScene `json:"scene"`
}

type GameProjectDraft struct {
	Metrics    []GameMetricDefinition    `json:"metrics"`
	Actions    []GameActionDefinition    `json:"actions"`
	Endings    []GameEndingDefinition    `json:"endings"`
	Objectives []GameObjectiveDefinition `json:"objectives,omitempty"`
	NPCs       []GameProjectNPC          `json:"npcs"`
	Events     []GameProjectEvent        `json:"events"`
	Scenes     []GameProjectScene        `json:"scenes"`
}

type CreateGameProjectInput struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Genre       string `json:"genre"`
	PlayGoal    string `json:"playGoal"`
}

type UpdateGameProjectInput struct {
	Revision    int               `json:"revision"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Genre       string            `json:"genre"`
	PlayGoal    string            `json:"playGoal"`
	Draft       *GameProjectDraft `json:"draft,omitempty"`
}

type PublishGameProjectInput struct {
	Version      string `json:"version"`
	ReleaseNotes string `json:"releaseNotes"`
}

type RestoreGameReleaseInput struct {
	Mode     string `json:"mode"` // replace-draft or new-project
	Revision int    `json:"revision,omitempty"`
	Key      string `json:"key,omitempty"`
	Name     string `json:"name,omitempty"`
}

type GameReleaseDetail struct {
	Release     gameproject.Release `json:"release"`
	ProjectName string              `json:"projectName"`
	Summary     map[string]int      `json:"summary"`
}

// GamePublishCheck is the server-side, creator-facing release gate. The
// frontend renders these plain-language results but never decides whether a
// draft is publishable on its own.
type GamePublishCheck struct {
	CanPublish bool                   `json:"canPublish"`
	Items      []GamePublishCheckItem `json:"items"`
}

type GamePublishCheckItem struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Ready    bool   `json:"ready"`
	Blocking bool   `json:"blocking"`
	Detail   string `json:"detail"`
	Step     string `json:"step,omitempty"`
}

// PreviewGameActionInput runs one action against an isolated state snapshot.
// It never writes game state. RequestKey is returned for UI correlation; it
// becomes an idempotency key only when a future formal settlement endpoint is
// introduced.
type PreviewGameActionInput struct {
	ActionID   string         `json:"actionId"`
	State      map[string]any `json:"state,omitempty"`
	Input      map[string]any `json:"input,omitempty"`
	RequestKey string         `json:"requestKey,omitempty"`
}

type GameConditionCheck struct {
	Target   string `json:"target"`
	Operator string `json:"operator"`
	Expected any    `json:"expected"`
	Actual   any    `json:"actual"`
	Matched  bool   `json:"matched"`
	Reason   string `json:"reason"`
}

type PreviewGameActionResult struct {
	RequestKey string               `json:"requestKey,omitempty"`
	ActionID   string               `json:"actionId"`
	Allowed    bool                 `json:"allowed"`
	Reasons    []string             `json:"reasons"`
	Checks     []GameConditionCheck `json:"checks"`
	Before     map[string]any       `json:"before"`
	After      map[string]any       `json:"after"`
	StateDiff  []GameStateDiff      `json:"stateDiff"`
}

// PreviewGameEventsInput is a disposable turn snapshot. It deliberately does
// not identify a formal session: E3's editor preview must never mutate a run.
type PreviewGameEventsInput struct {
	State             map[string]any `json:"state,omitempty"`
	CurrentTurn       int            `json:"currentTurn,omitempty"`
	TriggeredAtTurn   map[string]int `json:"triggeredAtTurn,omitempty"`
	OccurredEventIDs  []string       `json:"occurredEventIds,omitempty"`
	CandidateEventIDs []string       `json:"candidateEventIds,omitempty"`
}

type GameEventPreviewItem struct {
	EventID        string               `json:"eventId"`
	Name           string               `json:"name"`
	Category       string               `json:"category"`
	Eligible       bool                 `json:"eligible"`
	EventType      string               `json:"eventType"`
	Triggered      bool                 `json:"triggered"`
	Selected       bool                 `json:"selected"`
	Reasons        []string             `json:"reasons"`
	Checks         []GameConditionCheck `json:"checks"`
	Before         map[string]any       `json:"before"`
	After          map[string]any       `json:"after"`
	StateDiff      []GameStateDiff      `json:"stateDiff"`
	AdvanceTurns   int                  `json:"advanceTurns"`
	TimeAdvance    int                  `json:"timeAdvance"`
	NextEventIDs   []string             `json:"nextEventIds,omitempty"`
	EndGame        bool                 `json:"endGame"`
	EndsGame       bool                 `json:"endsGame"`
	Priority       int                  `json:"priority"`
	MutexGroup     string               `json:"mutexGroup,omitempty"`
	ExclusiveGroup string               `json:"exclusiveGroup,omitempty"`
}

type PreviewGameEventsResult struct {
	CurrentTurn int                    `json:"currentTurn"`
	Items       []GameEventPreviewItem `json:"items"`
	SelectedIDs []string               `json:"selectedIds"`
	Summary     string                 `json:"summary"`
}

type GameProjectDetail struct {
	Project  gameproject.Project   `json:"project"`
	Draft    GameProjectDraft      `json:"draft"`
	Releases []gameproject.Release `json:"releases"`
}

type GameProjectService struct {
	db       *gorm.DB
	packages *GamePackageService
}

func NewGameProjectService(db *gorm.DB, packages *GamePackageService) *GameProjectService {
	return &GameProjectService{db: db, packages: packages}
}

func emptyGameProjectDraft() GameProjectDraft {
	return GameProjectDraft{Metrics: []GameMetricDefinition{}, Actions: []GameActionDefinition{}, Endings: []GameEndingDefinition{}, Objectives: []GameObjectiveDefinition{}, NPCs: []GameProjectNPC{}, Events: []GameProjectEvent{}, Scenes: []GameProjectScene{}}
}

func encodeProjectDraft(draft GameProjectDraft) map[string]any {
	raw, _ := json.Marshal(draft)
	value := map[string]any{}
	_ = json.Unmarshal(raw, &value)
	return value
}

func decodeProjectDraft(value map[string]any) (GameProjectDraft, error) {
	if len(value) == 0 {
		return emptyGameProjectDraft(), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return GameProjectDraft{}, err
	}
	var draft GameProjectDraft
	if err := json.Unmarshal(raw, &draft); err != nil {
		return GameProjectDraft{}, err
	}
	return draft, nil
}

func validateProjectOverview(key, name, genre, playGoal string) error {
	if !projectKeyPattern.MatchString(strings.TrimSpace(key)) {
		return fmt.Errorf("game key may contain lowercase letters, numbers, dots, dashes and underscores")
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("game name is required")
	}
	if strings.TrimSpace(genre) == "" {
		return fmt.Errorf("genre is required")
	}
	if strings.TrimSpace(playGoal) == "" {
		return fmt.Errorf("play goal is required")
	}
	return nil
}

func (s *GameProjectService) Create(tenantID, actor string, in CreateGameProjectInput) (*GameProjectDetail, error) {
	if err := validateProjectOverview(in.Key, in.Name, in.Genre, in.PlayGoal); err != nil {
		return nil, err
	}
	row := gameproject.Project{ID: uuid.NewString(), TenantID: tenantID, Key: strings.TrimSpace(in.Key), Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description), Genre: strings.TrimSpace(in.Genre), PlayGoal: strings.TrimSpace(in.PlayGoal), Status: gameproject.StatusDraft, Revision: 1, Draft: encodeProjectDraft(emptyGameProjectDraft()), CreatedBy: actor, UpdatedBy: actor}
	if err := s.db.Create(&row).Error; err != nil {
		if isDuplicate(err) {
			return nil, fmt.Errorf("a game with this key already exists")
		}
		return nil, err
	}
	return &GameProjectDetail{Project: row, Draft: emptyGameProjectDraft(), Releases: []gameproject.Release{}}, nil
}

func (s *GameProjectService) List(tenantID string) ([]gameproject.Project, error) {
	rows := []gameproject.Project{}
	err := s.db.Where("tenant_id=?", tenantID).Order("updated_at DESC").Find(&rows).Error
	return rows, err
}

func (s *GameProjectService) Get(tenantID, id string) (*GameProjectDetail, error) {
	var row gameproject.Project
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGameProjectNotFound
		}
		return nil, err
	}
	draft, err := decodeProjectDraft(row.Draft)
	if err != nil {
		return nil, err
	}
	releases := []gameproject.Release{}
	if err := s.db.Where("tenant_id=? AND project_id=?", tenantID, id).Order("published_at DESC").Find(&releases).Error; err != nil {
		return nil, err
	}
	return &GameProjectDetail{Project: row, Draft: draft, Releases: releases}, nil
}

func (s *GameProjectService) GetRelease(tenantID, projectID, releaseID string) (*GameReleaseDetail, error) {
	var release gameproject.Release
	if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, releaseID).First(&release).Error; err != nil {
		return nil, ErrGameProjectNotFound
	}
	pkg, err := s.packages.Get(tenantID, release.PackageID)
	if err != nil {
		return nil, err
	}
	name := pkg.Package.DisplayName
	summary := map[string]int{"worldState": len(pkg.Manifest.Registry.Metrics), "actions": len(pkg.Manifest.Registry.Actions), "events": len(pkg.Manifest.Registry.Events), "endings": len(pkg.Manifest.Registry.Endings), "characters": 0, "scenes": 0, "assets": 0}
	if pkg.Manifest.Content != nil {
		summary["characters"] = len(pkg.Manifest.Content.NPCs)
		summary["scenes"] = len(pkg.Manifest.Content.Scenes)
		summary["assets"] = len(pkg.Manifest.Content.Assets)
	}
	return &GameReleaseDetail{Release: release, ProjectName: name, Summary: summary}, nil
}

// RestoreRelease never changes an immutable package. It either replaces the
// editable draft through optimistic locking or creates a separate project.
func (s *GameProjectService) RestoreRelease(tenantID, projectID, releaseID, actor string, in RestoreGameReleaseInput) (*GameProjectDetail, error) {
	var release gameproject.Release
	if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, releaseID).First(&release).Error; err != nil {
		return nil, ErrGameProjectNotFound
	}
	pkg, err := s.packages.Get(tenantID, release.PackageID)
	if err != nil {
		return nil, err
	}
	draft := draftFromPackage(pkg.Manifest)
	switch in.Mode {
	case "replace-draft":
		current, err := s.Get(tenantID, projectID)
		if err != nil {
			return nil, err
		}
		if in.Revision != current.Project.Revision {
			return nil, ErrGameProjectConflict
		}
		return s.Update(tenantID, projectID, actor, UpdateGameProjectInput{Revision: in.Revision, Name: current.Project.Name, Description: current.Project.Description, Genre: current.Project.Genre, PlayGoal: current.Project.PlayGoal, Draft: &draft})
	case "new-project":
		key, name := strings.TrimSpace(in.Key), strings.TrimSpace(in.Name)
		if name == "" {
			name = pkg.Package.DisplayName + "（基于历史版本）"
		}
		if key == "" {
			return nil, fmt.Errorf("请填写新游戏的唯一标识")
		}
		content := pkg.Manifest.Content
		genre, goal := "历史版本复用", "继续完善这个历史版本"
		if content != nil {
			genre = defaultString(content.Genre, genre)
			goal = defaultString(content.PlayGoal, goal)
		}
		created, err := s.Create(tenantID, actor, CreateGameProjectInput{Key: key, Name: name, Description: pkg.Package.Description, Genre: genre, PlayGoal: goal})
		if err != nil {
			return nil, err
		}
		result, err := s.Update(tenantID, created.Project.ID, actor, UpdateGameProjectInput{Revision: created.Project.Revision, Name: name, Description: created.Project.Description, Genre: genre, PlayGoal: goal, Draft: &draft})
		if err != nil {
			_ = s.db.Where("tenant_id=? AND id=?", tenantID, created.Project.ID).Delete(&gameproject.Project{}).Error
			return nil, err
		}
		return result, nil
	default:
		return nil, fmt.Errorf("请选择替换当前草稿或创建新游戏")
	}
}

func (s *GameProjectService) validateDraft(tenantID, projectID string, draft GameProjectDraft, publishing bool) error {
	// Drafts are intentionally permissive: creators must be able to save an
	// unfinished section and return later. Cross-reference and completeness
	// checks run at publish time, when the snapshot becomes executable.
	if err := s.validateCharacters(tenantID, draft); err != nil {
		return err
	}
	objectiveIDs := map[string]bool{}
	for i, objective := range draft.Objectives {
		if !gamePackageIDPattern.MatchString(objective.ID) || objectiveIDs[objective.ID] {
			return fmt.Errorf("任务 %d 需要一个不重复的标识", i+1)
		}
		if strings.TrimSpace(objective.Name) == "" {
			return fmt.Errorf("任务 %s 需要填写名称", objective.ID)
		}
		if objective.ConditionMode != "" && objective.ConditionMode != "all" && objective.ConditionMode != "any" {
			return fmt.Errorf("任务 %s 的完成条件组合方式无效", objective.ID)
		}
		if objective.FailureConditionMode != "" && objective.FailureConditionMode != "all" && objective.FailureConditionMode != "any" {
			return fmt.Errorf("任务 %s 的失败条件组合方式无效", objective.ID)
		}
		if objective.DeadlineTurns < 0 {
			return fmt.Errorf("任务 %s 的截止回合不能小于零", objective.ID)
		}
		for _, condition := range objective.Conditions {
			metric, ok := metricDefinitions(draft.Metrics)[condition.Target]
			if !ok {
				return fmt.Errorf("任务 %s 检查了不存在的世界状态 %s", objective.ID, condition.Target)
			}
			if err := validateGameCondition(metric, condition); err != nil {
				return fmt.Errorf("任务 %s 的完成条件无效: %w", objective.ID, err)
			}
		}
		for _, condition := range objective.FailureConditions {
			metric, ok := metricDefinitions(draft.Metrics)[condition.Target]
			if !ok {
				return fmt.Errorf("任务 %s 的失败条件检查了不存在的世界状态 %s", objective.ID, condition.Target)
			}
			if err := validateGameCondition(metric, condition); err != nil {
				return fmt.Errorf("任务 %s 的失败条件无效: %w", objective.ID, err)
			}
		}
		for _, effects := range [][]GameEffectDefinition{objective.CompletedEffects, objective.FailedEffects} {
			for _, effect := range effects {
				metric, ok := metricDefinitions(draft.Metrics)[effect.Target]
				if !ok {
					return fmt.Errorf("任务 %s 的后果修改了不存在的世界状态 %s", objective.ID, effect.Target)
				}
				if _, err := applyGameEffect(metric, metric.Initial, effect.Value, effect.Operation); err != nil {
					return fmt.Errorf("任务 %s 的后果无效: %w", objective.ID, err)
				}
			}
		}
		objectiveIDs[objective.ID] = true
	}
	for _, objective := range draft.Objectives {
		for _, dependency := range objective.DependsOn {
			if !objectiveIDs[dependency] || dependency == objective.ID {
				return fmt.Errorf("任务 %s 的前置任务无效", objective.ID)
			}
		}
	}
	seenScenes := map[string]bool{}
	for _, scene := range draft.Scenes {
		if strings.TrimSpace(scene.ID) == "" {
			continue
		}
		if seenScenes[scene.ID] {
			return fmt.Errorf("场景标识不能重复")
		}
		seenScenes[scene.ID] = true
		if err := s.validateSceneDraft(tenantID, projectID, draft, scene); err != nil {
			return err
		}
	}
	if !publishing {
		return nil
	}
	manifest := GamePackageManifest{SchemaVersion: GamePackageSchemaV1, Package: GamePackageMetadata{Key: "draft.validation", Version: "0.0.0", DisplayName: "draft"}, Registry: GameRegistryManifest{Metrics: draft.Metrics, Actions: draft.Actions, Events: make([]GameEventDefinition, 0, len(draft.Events)), Endings: draft.Endings}}
	npcIDs := map[string]bool{}
	for i, npc := range draft.NPCs {
		if !gamePackageIDPattern.MatchString(npc.ID) || npcIDs[npc.ID] {
			return fmt.Errorf("character %d needs a unique key", i+1)
		}
		if strings.TrimSpace(npc.Role) == "" || npc.AgentID == 0 {
			return fmt.Errorf("character %s must select an Agent and describe its role", npc.ID)
		}
		var count int64
		if err := s.db.Model(&agent.AgentConfig{}).Where("tenant_id=? AND id=?", tenantID, npc.AgentID).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("character %s references an Agent that is unavailable", npc.ID)
		}
		npcIDs[npc.ID] = true
	}
	metricIDs := map[string]bool{}
	for _, metric := range draft.Metrics {
		metricIDs[metric.ID] = true
	}
	eventIDs := map[string]bool{}
	for i, event := range draft.Events {
		if !gamePackageIDPattern.MatchString(event.ID) || eventIDs[event.ID] {
			return fmt.Errorf("事件 %d 需要一个不重复的标识", i+1)
		}
		eventIDs[event.ID] = true
		manifest.Registry.Events = append(manifest.Registry.Events, GameEventDefinition{ID: event.ID, Name: event.Name, Description: event.Description})
		for _, participant := range eventParticipantIDs(event) {
			if !npcIDs[participant] {
				return fmt.Errorf("事件 %s 引用了不存在的游戏角色 %s", event.ID, participant)
			}
		}
		if strings.TrimSpace(event.Name) == "" {
			return fmt.Errorf("事件 %d 需要填写名称", i+1)
		}
		if !validEventType(eventCategory(event)) {
			return fmt.Errorf("事件 %s 的类别无效", event.ID)
		}
		if event.ConditionMode != "" && event.ConditionMode != "all" && event.ConditionMode != "any" {
			return fmt.Errorf("事件 %s 的条件组合方式无效", event.ID)
		}
		if eventAdvanceTurns(event) < 0 || event.CooldownTurns < 0 {
			return fmt.Errorf("事件 %s 的推进回合和冷却回合不能小于零", event.ID)
		}
		for _, condition := range event.Conditions {
			metric, ok := metricDefinitions(draft.Metrics)[condition.Target]
			if !ok {
				return fmt.Errorf("事件 %s 检查了不存在的世界状态 %s", event.ID, condition.Target)
			}
			if err := validateGameCondition(metric, condition); err != nil {
				return fmt.Errorf("事件 %s 的触发条件无效: %w", event.ID, err)
			}
		}
		for _, effect := range event.Effects {
			if !metricIDs[effect.Target] {
				return fmt.Errorf("event %s changes an unknown world state %s", event.ID, effect.Target)
			}
			switch effect.Operation {
			case "set", "increment", "decrement":
			default:
				return fmt.Errorf("event %s has an unsupported state change", event.ID)
			}
		}
	}
	for _, event := range draft.Events {
		if event.SceneID != "" {
			found := false
			for _, scene := range draft.Scenes {
				if scene.ID == event.SceneID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("事件 %s 关联的场景不存在，请重新选择", event.Name)
			}
		}
		for _, next := range event.NextEventIDs {
			if !eventIDs[next] {
				return fmt.Errorf("事件 %s 的后续事件 %s 不存在", event.ID, next)
			}
			if next == event.ID {
				return fmt.Errorf("事件 %s 不能把自己设为后续事件", event.ID)
			}
		}
	}
	sceneIDs := map[string]bool{}
	assetKinds := map[string]string{}
	if len(draft.Scenes) > 0 && s.db.Migrator().HasTable(&gameasset.Asset{}) {
		var assets []gameasset.Asset
		if err := s.db.Where("tenant_id=? AND project_id=?", tenantID, projectID).Find(&assets).Error; err != nil {
			return err
		}
		for _, asset := range assets {
			assetKinds[asset.ID] = asset.Kind
		}
	}
	for i, scene := range draft.Scenes {
		if !gamePackageIDPattern.MatchString(scene.ID) || sceneIDs[scene.ID] || strings.TrimSpace(scene.Name) == "" {
			return fmt.Errorf("scene %d needs a unique key and a name", i+1)
		}
		sceneIDs[scene.ID] = true
		if scene.Order < 0 {
			return fmt.Errorf("场景 %s 的排序不能小于零", scene.ID)
		}
		if scene.BackgroundAssetID != "" && assetKinds[scene.BackgroundAssetID] != gameasset.KindBackground {
			return fmt.Errorf("场景 %s 的背景图已不可用", scene.ID)
		}
		if scene.MusicAssetID != "" && assetKinds[scene.MusicAssetID] != gameasset.KindMusic {
			return fmt.Errorf("场景 %s 的背景音乐已不可用", scene.ID)
		}
		for _, id := range scene.SoundEffectAssetIDs {
			if assetKinds[id] != gameasset.KindSound {
				return fmt.Errorf("场景 %s 的音效已不可用", scene.ID)
			}
		}
		layerIDs := map[string]bool{}
		for j, layer := range scene.Characters {
			if strings.TrimSpace(layer.ID) == "" || layerIDs[layer.ID] {
				return fmt.Errorf("场景 %s 的人物摆放 %d 需要不重复的标识", scene.ID, j+1)
			}
			layerIDs[layer.ID] = true
			if !npcIDs[layer.CharacterID] {
				return fmt.Errorf("场景 %s 引用了不存在的游戏人物", scene.ID)
			}
			if layer.PortraitAssetID != "" && assetKinds[layer.PortraitAssetID] != gameasset.KindPortrait {
				return fmt.Errorf("场景 %s 的人物立绘已不可用", scene.ID)
			}
			for _, assetID := range layer.EmotionPortraitAssetIDs {
				if assetID != "" && assetKinds[assetID] != gameasset.KindPortrait {
					return fmt.Errorf("场景 %s 的人物情绪立绘已不可用", scene.ID)
				}
			}
			if layer.X < 0 || layer.X > 100 || layer.Y < 0 || layer.Y > 100 {
				return fmt.Errorf("场景 %s 的人物位置超出画面", scene.ID)
			}
			if layer.Scale <= 0 || layer.Scale > 4 {
				return fmt.Errorf("场景 %s 的人物缩放必须在 0–4 之间", scene.ID)
			}
		}
	}
	if len(draft.Metrics) == 0 || len(draft.Actions) == 0 || len(draft.Events) == 0 {
		return fmt.Errorf("before publishing, add at least one world state, one player action and one event")
	}
	return validateGamePackageManifest(&manifest)
}

func characterType(value string) string {
	if strings.TrimSpace(value) == "" {
		return GameCharacterNPC
	}
	return strings.TrimSpace(value)
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func validEventType(value string) bool {
	switch strings.TrimSpace(value) {
	case "", "system", "systemic", "route", "personal", "environment", "ambient":
		return true
	default:
		return false
	}
}

func eventParticipantIDs(event GameProjectEvent) []string {
	if len(event.ParticipantIDs) > 0 {
		return event.ParticipantIDs
	}
	return event.ParticipantCharacterIDs
}

func eventCategory(event GameProjectEvent) string {
	if event.Category != "" {
		return event.Category
	}
	return event.EventType
}

func eventContent(event GameProjectEvent) string {
	if event.DisplayContent != "" {
		return event.DisplayContent
	}
	return event.Content
}

func eventAdvanceTurns(event GameProjectEvent) int {
	if event.TimeAdvance != 0 {
		return event.TimeAdvance
	}
	if event.AdvanceTurns != 0 {
		return event.AdvanceTurns
	}
	if event.AdvancesTime {
		return 1
	}
	return 0
}

func eventMutexGroup(event GameProjectEvent) string {
	if event.ExclusiveGroup != "" {
		return event.ExclusiveGroup
	}
	return event.MutexGroup
}

func eventEndsGame(event GameProjectEvent) bool {
	return event.EndsGame || event.EndGame
}

// validateCharacters enforces the game boundary without creating another Agent
// model. A character only references an Agent and the Agent's existing assets.
func (s *GameProjectService) validateCharacters(tenantID string, draft GameProjectDraft) error {
	metricIDs, actionIDs := map[string]bool{}, map[string]bool{}
	for _, item := range draft.Metrics {
		metricIDs[item.ID] = true
	}
	for _, item := range draft.Actions {
		actionIDs[item.ID] = true
	}
	characterIDs, agentIDs := map[string]bool{}, map[uint64]bool{}
	directors := 0
	for i, npc := range draft.NPCs {
		if strings.TrimSpace(npc.ID) == "" {
			continue
		} // unfinished row may be saved
		if !gamePackageIDPattern.MatchString(npc.ID) || characterIDs[npc.ID] {
			return fmt.Errorf("角色 %d 需要一个不重复的标识", i+1)
		}
		characterIDs[npc.ID] = true
		kind := characterType(npc.CharacterType)
		if kind != GameCharacterDirector && kind != GameCharacterNPC && kind != GameCharacterObserver {
			return fmt.Errorf("角色 %s 的类型无效", npc.ID)
		}
		if kind == GameCharacterDirector {
			directors++
		}
		if kind != GameCharacterDirector && (containsString(npc.Permissions, "advance_turn") || containsString(npc.ExecutableActionIDs, "advance_turn")) {
			return fmt.Errorf("只有导演角色可以推进游戏时间")
		}
		if npc.AgentID == 0 {
			continue
		}
		if agentIDs[npc.AgentID] {
			return fmt.Errorf("同一个 Agent 不能重复绑定为多个游戏角色")
		}
		agentIDs[npc.AgentID] = true
		var cfg agent.AgentConfig
		if err := s.db.Where("tenant_id=? AND id=?", tenantID, npc.AgentID).First(&cfg).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("角色 %s 选择的 Agent 当前不可用", npc.ID)
			}
			return err
		}
		for _, id := range npc.VisibleMetricIDs {
			if !metricIDs[id] {
				return fmt.Errorf("角色 %s 引用了不存在的世界状态", npc.ID)
			}
		}
		for _, id := range append(append([]string{}, npc.ExecutableActionIDs...), npc.AdvisoryActionIDs...) {
			if !actionIDs[id] {
				return fmt.Errorf("角色 %s 引用了不存在的行动", npc.ID)
			}
		}
		if len(npc.KnowledgeRefs) > 0 {
			var rows []agent.AgentKnowledgeDataset
			if err := s.db.Where("agent_id=?", npc.AgentID).Find(&rows).Error; err != nil {
				return err
			}
			bound := map[string]bool{}
			for _, row := range rows {
				bound[row.DatasetID] = true
			}
			for _, ref := range npc.KnowledgeRefs {
				if !bound[ref] {
					return fmt.Errorf("角色 %s 引用了这个 Agent 未绑定的资料", npc.ID)
				}
			}
		}
	}
	if directors > 1 {
		return fmt.Errorf("一个游戏只能设置一名导演")
	}
	for _, npc := range draft.NPCs {
		if npc.AgentID == 0 {
			continue
		}
		for _, relationID := range npc.RelationIDs {
			var rel agentrelation.AgentRelation
			if err := s.db.Where("tenant_id=? AND id=?", tenantID, relationID).First(&rel).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("角色 %s 引用的关系已不可用", npc.ID)
				}
				return err
			}
			if rel.SourceAgentID != npc.AgentID && rel.TargetAgentID != npc.AgentID {
				return fmt.Errorf("角色 %s 不能引用与自己无关的关系", npc.ID)
			}
			other := rel.SourceAgentID
			if other == npc.AgentID {
				other = rel.TargetAgentID
			}
			if !agentIDs[other] {
				return fmt.Errorf("角色 %s 的关系对象尚未加入当前游戏", npc.ID)
			}
		}
	}
	return nil
}

func (s *GameProjectService) Update(tenantID, id, actor string, in UpdateGameProjectInput) (*GameProjectDetail, error) {
	current, err := s.Get(tenantID, id)
	if err != nil {
		return nil, err
	}
	if in.Revision <= 0 || in.Revision != current.Project.Revision {
		return nil, ErrGameProjectConflict
	}
	draft := current.Draft
	if in.Draft != nil {
		draft = *in.Draft
	}
	if err := validateProjectOverview(current.Project.Key, in.Name, in.Genre, in.PlayGoal); err != nil {
		return nil, err
	}
	if err := s.validateDraft(tenantID, id, draft, false); err != nil {
		return nil, err
	}
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, err
	}
	// Map Updates bypasses GORM's serializer hook, so persist canonical JSON
	// rather than passing a Go map to the SQL driver.
	updates := map[string]any{"name": strings.TrimSpace(in.Name), "description": strings.TrimSpace(in.Description), "genre": strings.TrimSpace(in.Genre), "play_goal": strings.TrimSpace(in.PlayGoal), "draft_json": string(draftJSON), "updated_by": actor, "revision": in.Revision + 1}
	result := s.db.Model(&gameproject.Project{}).Where("tenant_id=? AND id=? AND revision=?", tenantID, id, in.Revision).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrGameProjectConflict
	}
	return s.Get(tenantID, id)
}

func (s *GameProjectService) validateSceneDraft(tenantID, projectID string, draft GameProjectDraft, scene GameProjectScene) error {
	if !gamePackageIDPattern.MatchString(scene.ID) {
		return fmt.Errorf("场景标识无效")
	}
	if strings.TrimSpace(scene.Name) == "" {
		return fmt.Errorf("请填写场景名称")
	}
	if scene.Order < 0 {
		return fmt.Errorf("场景排序不能小于零")
	}
	characters := map[string]bool{}
	for _, npc := range draft.NPCs {
		characters[npc.ID] = true
	}
	expectedAssets := map[string]string{}
	if scene.BackgroundAssetID != "" {
		expectedAssets[scene.BackgroundAssetID] = gameasset.KindBackground
	}
	if scene.MusicAssetID != "" {
		expectedAssets[scene.MusicAssetID] = gameasset.KindMusic
	}
	for _, assetID := range scene.SoundEffectAssetIDs {
		expectedAssets[assetID] = gameasset.KindSound
	}
	layerIDs := map[string]bool{}
	for _, item := range scene.Characters {
		if strings.TrimSpace(item.ID) == "" || layerIDs[item.ID] {
			return fmt.Errorf("人物摆放需要不重复的标识")
		}
		layerIDs[item.ID] = true
		if !characters[item.CharacterID] {
			return fmt.Errorf("场景引用了不存在的游戏人物")
		}
		if item.PortraitAssetID != "" {
			expectedAssets[item.PortraitAssetID] = gameasset.KindPortrait
		}
		for _, assetID := range item.EmotionPortraitAssetIDs {
			if assetID != "" {
				expectedAssets[assetID] = gameasset.KindPortrait
			}
		}
		if item.X < 0 || item.X > 100 || item.Y < 0 || item.Y > 100 {
			return fmt.Errorf("人物位置必须在画面内")
		}
		if item.Scale <= 0 || item.Scale > 4 {
			return fmt.Errorf("人物缩放必须在 0–4 之间")
		}
	}
	if len(expectedAssets) > 0 {
		for assetID, kind := range expectedAssets {
			var asset gameasset.Asset
			if err := s.db.Where("tenant_id=? AND project_id=? AND id=?", tenantID, projectID, assetID).First(&asset).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("场景引用的资源已不可用")
				}
				return err
			}
			if asset.Kind != kind {
				return fmt.Errorf("场景引用的资源类型不正确")
			}
		}
	}
	return nil
}

func (s *GameProjectService) CreateScene(tenantID, projectID, actor string, in SaveGameSceneInput) (*GameProjectDetail, error) {
	detail, err := s.Get(tenantID, projectID)
	if err != nil {
		return nil, err
	}
	if in.Revision != detail.Project.Revision {
		return nil, ErrGameProjectConflict
	}
	if strings.TrimSpace(in.Scene.ID) == "" {
		in.Scene.ID = "scene-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	for _, current := range detail.Draft.Scenes {
		if current.ID == in.Scene.ID {
			return nil, fmt.Errorf("场景标识已存在")
		}
	}
	if err := s.validateSceneDraft(tenantID, projectID, detail.Draft, in.Scene); err != nil {
		return nil, err
	}
	detail.Draft.Scenes = append(detail.Draft.Scenes, in.Scene)
	return s.Update(tenantID, projectID, actor, UpdateGameProjectInput{Revision: in.Revision, Name: detail.Project.Name, Description: detail.Project.Description, Genre: detail.Project.Genre, PlayGoal: detail.Project.PlayGoal, Draft: &detail.Draft})
}

func (s *GameProjectService) UpdateScene(tenantID, projectID, sceneID, actor string, in SaveGameSceneInput) (*GameProjectDetail, error) {
	detail, err := s.Get(tenantID, projectID)
	if err != nil {
		return nil, err
	}
	if in.Revision != detail.Project.Revision {
		return nil, ErrGameProjectConflict
	}
	if in.Scene.ID != "" && in.Scene.ID != sceneID {
		return nil, fmt.Errorf("场景标识与地址不一致")
	}
	in.Scene.ID = sceneID
	if err := s.validateSceneDraft(tenantID, projectID, detail.Draft, in.Scene); err != nil {
		return nil, err
	}
	found := false
	for i := range detail.Draft.Scenes {
		if detail.Draft.Scenes[i].ID == sceneID {
			detail.Draft.Scenes[i], found = in.Scene, true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("场景不存在")
	}
	return s.Update(tenantID, projectID, actor, UpdateGameProjectInput{Revision: in.Revision, Name: detail.Project.Name, Description: detail.Project.Description, Genre: detail.Project.Genre, PlayGoal: detail.Project.PlayGoal, Draft: &detail.Draft})
}

func (s *GameProjectService) DeleteScene(tenantID, projectID, sceneID, actor string, revision int) (*GameProjectDetail, error) {
	detail, err := s.Get(tenantID, projectID)
	if err != nil {
		return nil, err
	}
	if revision != detail.Project.Revision {
		return nil, ErrGameProjectConflict
	}
	next, found := make([]GameProjectScene, 0, len(detail.Draft.Scenes)), false
	for _, current := range detail.Draft.Scenes {
		if current.ID == sceneID {
			found = true
		} else {
			next = append(next, current)
		}
	}
	if !found {
		return nil, fmt.Errorf("场景不存在")
	}
	detail.Draft.Scenes = next
	return s.Update(tenantID, projectID, actor, UpdateGameProjectInput{Revision: revision, Name: detail.Project.Name, Description: detail.Project.Description, Genre: detail.Project.Genre, PlayGoal: detail.Project.PlayGoal, Draft: &detail.Draft})
}

func (s *GameProjectService) PublishCheck(tenantID, id string) (*GamePublishCheck, error) {
	detail, err := s.Get(tenantID, id)
	if err != nil {
		return nil, err
	}
	items := []GamePublishCheckItem{}
	add := func(key, label string, ready, blocking bool, detailText, step string) {
		items = append(items, GamePublishCheckItem{Key: key, Label: label, Ready: ready, Blocking: blocking, Detail: detailText, Step: step})
	}
	overviewReady := strings.TrimSpace(detail.Project.Name) != "" && strings.TrimSpace(detail.Project.Genre) != "" && strings.TrimSpace(detail.Project.PlayGoal) != ""
	add("overview", "游戏资料", overviewReady, true, map[bool]string{true: "名称、题材和玩法目标已填写。", false: "请补全游戏名称、题材和玩法目标。"}[overviewReady], "overview")
	rulesReady := len(detail.Draft.Metrics) > 0 && len(detail.Draft.Actions) > 0
	add("rules", "规则与世界状态", rulesReady, true, map[bool]string{true: "已经设置世界状态和玩家行动。", false: "至少需要一个世界状态和一个玩家行动。"}[rulesReady], "world")
	directors := 0
	for _, npc := range detail.Draft.NPCs {
		if characterType(npc.CharacterType) == GameCharacterDirector && npc.AgentID > 0 {
			directors++
		}
	}
	charactersReady := len(detail.Draft.NPCs) > 0 && directors == 1
	add("characters", "游戏人物", charactersReady, true, map[bool]string{true: "人物已绑定，并且只有一位导演。", false: "请绑定游戏人物，并设置一位导演。"}[charactersReady], "characters")
	eventsReady := len(detail.Draft.Events) > 0
	add("events", "事件与推进", eventsReady, true, map[bool]string{true: "故事事件已经设置。", false: "至少需要一个可以推进故事的事件。"}[eventsReady], "events")
	scenesReady := len(detail.Draft.Scenes) > 0
	add("scenes", "场景与视听", scenesReady, true, map[bool]string{true: "至少有一个可展示场景。", false: "请先建立一个游戏场景。"}[scenesReady], "scenes")
	mediaReady := scenesReady
	if mediaReady {
		for _, scene := range detail.Draft.Scenes {
			if scene.BackgroundAssetID == "" {
				mediaReady = false
				break
			}
		}
	}
	add("media", "画面资源", mediaReady, true, map[bool]string{true: "每个场景都有背景图。", false: "请为每个场景选择背景图，避免玩家看到空白画面。"}[mediaReady], "scenes")
	integrityReady := true
	integrityDetail := "所有人物、事件、场景和资源引用都可以正常使用。"
	if err := s.validateDraft(tenantID, id, detail.Draft, true); err != nil {
		integrityReady = false
		// The lower-level validator also protects package imports and therefore
		// uses developer-facing wording. The creator page should not leak those
		// internal messages; the checks above already identify the exact editing
		// step that needs attention.
		integrityDetail = "还有必填内容或引用关系没有完成，请按上面的检查逐项修正。"
	}
	add("integrity", "完整性检查", integrityReady, true, integrityDetail, "")
	can := true
	for _, item := range items {
		if item.Blocking && !item.Ready {
			can = false
			break
		}
	}
	return &GamePublishCheck{CanPublish: can, Items: items}, nil
}

func (s *GameProjectService) Publish(tenantID, id, actor string, in PublishGameProjectInput) (*gameproject.Release, error) {
	if !semanticVersionPattern.MatchString(in.Version) {
		return nil, fmt.Errorf("version must use x.y.z format")
	}
	detail, err := s.Get(tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.validateDraft(tenantID, id, detail.Draft, true); err != nil {
		return nil, err
	}
	storyEvents := make([]GameStoryEvent, 0, len(detail.Draft.Events))
	registryEvents := make([]GameEventDefinition, 0, len(detail.Draft.Events))
	for _, event := range detail.Draft.Events {
		registryEvents = append(registryEvents, GameEventDefinition{ID: event.ID, Name: event.Name, Description: event.Description})
		category, content := eventCategory(event), eventContent(event)
		participants, advanceTurns := eventParticipantIDs(event), eventAdvanceTurns(event)
		group, ends := eventMutexGroup(event), eventEndsGame(event)
		storyEvents = append(storyEvents, GameStoryEvent{SceneID: event.SceneID, ID: event.ID, Name: event.Name, Description: event.Description, Category: category, DisplayContent: content, ParticipantIDs: participants, TimeAdvance: advanceTurns, ExclusiveGroup: group, EndsGame: ends, EventType: category, Content: content, ConditionMode: event.ConditionMode, Conditions: event.Conditions, ParticipantCharacterIDs: participants, Effects: event.Effects, AdvanceTurns: advanceTurns, Priority: event.Priority, Once: event.Once, CooldownTurns: event.CooldownTurns, MutexGroup: group, NextEventIDs: event.NextEventIDs, EndGame: ends, EndReason: event.EndReason})
	}
	npcs := make([]GameNPCReference, 0, len(detail.Draft.NPCs))
	for _, npc := range detail.Draft.NPCs {
		version, err := s.snapshotAgentVersion(tenantID, npc)
		if err != nil {
			return nil, err
		}
		npcs = append(npcs, GameNPCReference{ID: npc.ID, AgentID: npc.AgentID, DisplayName: npc.DisplayName, Role: npc.Role, Responsibilities: npc.Responsibilities, CharacterType: characterType(npc.CharacterType), VisibleMetricIDs: npc.VisibleMetricIDs, ExecutableActionIDs: npc.ExecutableActionIDs, AdvisoryActionIDs: npc.AdvisoryActionIDs, Permissions: npc.Permissions, KnownInformation: npc.KnownInformation, InformationScope: npc.InformationScope, KnowledgeRefs: npc.KnowledgeRefs, RelationIDs: npc.RelationIDs, AgentVersion: version})
	}
	scenes := make([]GameSceneDefinition, 0, len(detail.Draft.Scenes))
	for _, scene := range detail.Draft.Scenes {
		scenes = append(scenes, GameSceneDefinition{ID: scene.ID, Name: scene.Name, Description: scene.Description, Order: scene.Order, BackgroundAssetID: scene.BackgroundAssetID, MusicAssetID: scene.MusicAssetID, SoundEffectAssetIDs: append([]string{}, scene.SoundEffectAssetIDs...), Characters: append([]GameSceneCharacter{}, scene.Characters...), BackgroundRef: scene.BackgroundRef, PortraitRefs: scene.PortraitRefs, MusicRef: scene.MusicRef, SoundEffectRefs: scene.SoundEffectRefs})
	}
	var assetRows []gameasset.Asset
	if s.db.Migrator().HasTable(&gameasset.Asset{}) {
		if err := s.db.Where("tenant_id=? AND project_id=?", tenantID, id).Order("created_at ASC").Find(&assetRows).Error; err != nil {
			return nil, err
		}
	}
	assets := make([]GameAssetSnapshot, 0, len(assetRows))
	referencedAssets := map[string]bool{}
	for _, scene := range detail.Draft.Scenes {
		if scene.BackgroundAssetID != "" {
			referencedAssets[scene.BackgroundAssetID] = true
		}
		if scene.MusicAssetID != "" {
			referencedAssets[scene.MusicAssetID] = true
		}
		for _, assetID := range scene.SoundEffectAssetIDs {
			referencedAssets[assetID] = true
		}
		for _, layer := range scene.Characters {
			if layer.PortraitAssetID != "" {
				referencedAssets[layer.PortraitAssetID] = true
			}
			for _, assetID := range layer.EmotionPortraitAssetIDs {
				if assetID != "" {
					referencedAssets[assetID] = true
				}
			}
		}
	}
	for _, item := range assetRows {
		if !referencedAssets[item.ID] {
			continue
		}
		assets = append(assets, GameAssetSnapshot{ID: item.ID, Name: item.Name, Kind: item.Kind, OriginalName: item.OriginalName, MimeType: item.MimeType, SizeBytes: item.SizeBytes, Width: item.Width, Height: item.Height, DurationMs: item.DurationMs, ContentHash: item.ContentHash, StorageProvider: item.StorageProvider, StorageKey: item.StorageKey})
	}
	manifest := GamePackageManifest{SchemaVersion: GamePackageSchemaV1, Package: GamePackageMetadata{Key: "game." + detail.Project.Key, Version: in.Version, DisplayName: detail.Project.Name, Description: detail.Project.Description, Author: actor}, Registry: GameRegistryManifest{Metrics: detail.Draft.Metrics, Actions: detail.Draft.Actions, Events: registryEvents, Endings: detail.Draft.Endings}, Content: &GamePackageContent{ProjectID: detail.Project.ID, ProjectRevision: detail.Project.Revision, Genre: detail.Project.Genre, PlayGoal: detail.Project.PlayGoal, Objectives: detail.Draft.Objectives, NPCs: npcs, StoryEvents: storyEvents, Scenes: scenes, Assets: assets}}
	raw, _ := json.Marshal(manifest)
	pkg, err := s.packages.Register(tenantID, actor, raw)
	if err != nil {
		return nil, err
	}
	release := &gameproject.Release{ID: uuid.NewString(), TenantID: tenantID, ProjectID: id, Version: in.Version, ProjectRevision: detail.Project.Revision, PackageID: pkg.Package.ID, ContentHash: pkg.Package.ContentHash, ReleaseNotes: strings.TrimSpace(in.ReleaseNotes), PublishedBy: actor, PublishedAt: time.Now().UTC()}
	if err := s.db.Create(release).Error; err != nil {
		if isDuplicate(err) {
			var existing gameproject.Release
			if qerr := s.db.Where("tenant_id=? AND project_id=? AND version=?", tenantID, id, in.Version).First(&existing).Error; qerr == nil && existing.ContentHash == release.ContentHash {
				return &existing, nil
			}
			return nil, fmt.Errorf("this version was already published with different content")
		}
		return nil, err
	}
	return release, nil
}

func (s *GameProjectService) snapshotAgentVersion(tenantID string, npc GameProjectNPC) (*GameAgentVersionSnapshot, error) {
	var cfg agent.AgentConfig
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, npc.AgentID).First(&cfg).Error; err != nil {
		return nil, err
	}
	result := &GameAgentVersionSnapshot{ContentHash: cfg.ContentHash, PersonalityTemplateName: cfg.PersonalityTemplateName, PersonalityTemplateVersion: cfg.PersonalityTemplateVersion, KnowledgeRefs: append([]string{}, npc.KnowledgeRefs...), ToolHashes: map[string]string{}, SkillHashes: map[string]string{}, Relations: []GameRelationVersionEntry{}}
	if s.db.Migrator().HasTable(&agent.DeploymentSnapshot{}) {
		var deployment agent.DeploymentSnapshot
		if err := s.db.Where("tenant_id=? AND agent_id=?", tenantID, npc.AgentID).First(&deployment).Error; err == nil {
			for key, value := range deployment.ToolHashes {
				result.ToolHashes[key] = value
			}
			for key, value := range deployment.SkillHashes {
				result.SkillHashes[key] = value
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	for _, id := range npc.RelationIDs {
		var rel agentrelation.AgentRelation
		if err := s.db.Where("tenant_id=? AND id=?", tenantID, id).First(&rel).Error; err != nil {
			return nil, err
		}
		result.Relations = append(result.Relations, GameRelationVersionEntry{ID: rel.ID, SourceAgentID: rel.SourceAgentID, TargetAgentID: rel.TargetAgentID, RelationType: rel.RelationType, RelationTypeTemplateName: rel.RelationTypeTemplateName, RelationTypeTemplateVersion: rel.RelationTypeTemplateVersion, AllowedActions: append([]string{}, rel.AllowedActions...), ContextPolicy: rel.ContextPolicy, DeliveryPolicy: rel.DeliveryPolicy, Enabled: rel.Enabled})
	}
	return result, nil
}

// PreviewCharacterContext gives creators a plain-language view of the
// information boundary that will be applied to one game character.
func (s *GameProjectService) PreviewCharacterContext(tenantID, projectID, characterID string) (*GameCharacterContextPreview, error) {
	detail, err := s.Get(tenantID, projectID)
	if err != nil {
		return nil, err
	}
	var character *GameProjectNPC
	for i := range detail.Draft.NPCs {
		if detail.Draft.NPCs[i].ID == characterID {
			character = &detail.Draft.NPCs[i]
			break
		}
	}
	if character == nil {
		return nil, fmt.Errorf("找不到这个游戏角色")
	}
	var cfg agent.AgentConfig
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, character.AgentID).First(&cfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("这个角色引用的 Agent 已不可用")
		}
		return nil, err
	}
	metricNames, actionNames := map[string]string{}, map[string]string{}
	for _, item := range detail.Draft.Metrics {
		metricNames[item.ID] = item.Name
	}
	for _, item := range detail.Draft.Actions {
		actionNames[item.ID] = item.Name
	}
	readNames := func(ids []string, names map[string]string) []string {
		out := []string{}
		for _, id := range ids {
			if name := names[id]; name != "" {
				out = append(out, name)
			}
		}
		return out
	}
	available := cfg.DeploymentStatus == "running"
	availability := "Agent 已绑定，但需要启动后才能参与试玩"
	if available {
		availability = "Agent 已启动，可以参与试玩"
	}
	knowledgeSummary := "不额外开放资料库，只使用角色已掌握的信息"
	if len(character.KnowledgeRefs) > 0 {
		knowledgeSummary = fmt.Sprintf("可使用已绑定的 %d 份资料", len(character.KnowledgeRefs))
	}
	limits := []string{}
	if characterType(character.CharacterType) != GameCharacterDirector {
		limits = append(limits, "不能推进游戏时间")
	}
	if characterType(character.CharacterType) == GameCharacterObserver {
		limits = append(limits, "只能观察和提出建议，不能直接执行行动")
	}
	relationships := []string{}
	for _, relationID := range character.RelationIDs {
		var rel agentrelation.AgentRelation
		if err := s.db.Where("tenant_id=? AND id=?", tenantID, relationID).First(&rel).Error; err != nil {
			continue
		}
		otherID, direction := rel.TargetAgentID, "可联系"
		if rel.TargetAgentID == character.AgentID {
			otherID, direction = rel.SourceAgentID, "可接收来自"
		}
		var other agent.AgentConfig
		if s.db.Where("tenant_id=? AND id=?", tenantID, otherID).First(&other).Error == nil {
			relationships = append(relationships, fmt.Sprintf("%s %s（%s）", direction, other.Name, rel.RelationType))
		}
	}
	displayName := strings.TrimSpace(character.DisplayName)
	if displayName == "" {
		displayName = cfg.Name
	}
	return &GameCharacterContextPreview{CharacterID: character.ID, DisplayName: displayName, Identity: character.Role, Responsibilities: character.Responsibilities, AgentName: cfg.Name, AgentAvailable: available, AvailableSummary: availability, VisibleWorldState: readNames(character.VisibleMetricIDs, metricNames), CanExecute: readNames(character.ExecutableActionIDs, actionNames), CanSuggest: readNames(character.AdvisoryActionIDs, actionNames), KnownInformation: append([]string{}, character.KnownInformation...), KnowledgeSummary: knowledgeSummary, Relationships: relationships, Limits: limits}, nil
}

// PreviewAction is deliberately pure: it reads the current project draft and
// returns a calculated result without persisting world state or consuming an
// idempotency key. Formal settlement will be a separate command boundary.
func (s *GameProjectService) PreviewAction(tenantID, id string, in PreviewGameActionInput) (*PreviewGameActionResult, error) {
	detail, err := s.Get(tenantID, id)
	if err != nil {
		return nil, err
	}
	metrics := metricDefinitions(detail.Draft.Metrics)
	state := make(map[string]any, len(metrics))
	for key, metric := range metrics {
		state[key] = metric.Initial
	}
	for key, value := range in.State {
		metric, ok := metrics[key]
		if !ok {
			return nil, fmt.Errorf("当前状态包含未定义的项目 %s", key)
		}
		if err := validateMetricValue(metric, value); err != nil {
			return nil, fmt.Errorf("当前状态 %s 的值无效: %w", key, err)
		}
		state[key] = value
	}
	actions := make(map[string]GameActionDefinition, len(detail.Draft.Actions))
	for _, action := range detail.Draft.Actions {
		actions[action.ID] = action
	}
	action, ok := actions[in.ActionID]
	if !ok {
		return nil, fmt.Errorf("找不到这个玩家行动")
	}
	result := &PreviewGameActionResult{RequestKey: strings.TrimSpace(in.RequestKey), ActionID: action.ID, Before: cloneMap(state), After: cloneMap(state), Reasons: []string{}, Checks: []GameConditionCheck{}, StateDiff: []GameStateDiff{}}
	mode := action.ConditionMode
	if mode == "" {
		mode = "all"
	}
	matchedCount := 0
	for _, condition := range action.Conditions {
		metric, exists := metrics[condition.Target]
		if !exists {
			return nil, fmt.Errorf("行动检查了未定义的世界状态 %s", condition.Target)
		}
		if err := validateGameCondition(metric, condition); err != nil {
			return nil, fmt.Errorf("行动条件无效: %w", err)
		}
		actual := state[condition.Target]
		matched, err := matchGameCondition(actual, condition.Value, condition.Operator)
		if err != nil {
			return nil, err
		}
		if matched {
			matchedCount++
		}
		reason := conditionReason(metric, condition, actual, matched)
		result.Checks = append(result.Checks, GameConditionCheck{Target: condition.Target, Operator: condition.Operator, Expected: condition.Value, Actual: actual, Matched: matched, Reason: reason})
	}
	result.Allowed = len(action.Conditions) == 0 || (mode == "all" && matchedCount == len(action.Conditions)) || (mode == "any" && matchedCount > 0)
	if !result.Allowed {
		for _, check := range result.Checks {
			if !check.Matched {
				result.Reasons = append(result.Reasons, check.Reason)
			}
		}
		return result, nil
	}
	result.Reasons = append(result.Reasons, "当前条件允许执行这个行动")
	for _, effect := range action.Effects {
		metric, exists := metrics[effect.Target]
		if !exists {
			return nil, fmt.Errorf("行动修改了未定义的世界状态 %s", effect.Target)
		}
		before := result.After[effect.Target]
		value := effect.Value
		if effect.ValueFrom != "" {
			value = in.Input[effect.ValueFrom]
			if value == nil {
				value = effect.DefaultValue
			}
		}
		after, err := applyGameEffect(metric, before, value, effect.Operation)
		if err != nil {
			return nil, fmt.Errorf("无法计算 %s 的变化: %w", metric.Name, err)
		}
		result.After[effect.Target] = after
		result.StateDiff = append(result.StateDiff, GameStateDiff{MetricID: effect.Target, Before: before, After: after})
	}
	sort.Slice(result.StateDiff, func(i, j int) bool { return result.StateDiff[i].MetricID < result.StateDiff[j].MetricID })
	return result, nil
}

// PreviewEvents explains event eligibility and the stable scheduler choice
// without creating a session, advancing a turn, or persisting any state.
func (s *GameProjectService) PreviewEvents(tenantID, projectID, eventID string, in PreviewGameEventsInput) (*PreviewGameEventsResult, error) {
	detail, err := s.Get(tenantID, projectID)
	if err != nil {
		return nil, err
	}
	metrics := metricDefinitions(detail.Draft.Metrics)
	state := make(map[string]any, len(metrics))
	for key, metric := range metrics {
		state[key] = metric.Initial
	}
	for key, value := range in.State {
		metric, ok := metrics[key]
		if !ok {
			return nil, fmt.Errorf("当前状态包含未定义的项目 %s", key)
		}
		if err := validateMetricValue(metric, value); err != nil {
			return nil, fmt.Errorf("当前状态 %s 的值无效: %w", key, err)
		}
		state[key] = value
	}
	if in.CurrentTurn < 0 {
		return nil, fmt.Errorf("当前回合不能小于零")
	}
	occurred := map[string]bool{}
	for _, id := range in.OccurredEventIDs {
		occurred[id] = true
	}
	for id := range in.TriggeredAtTurn {
		occurred[id] = true
	}
	candidates := map[string]bool{}
	for _, id := range in.CandidateEventIDs {
		candidates[id] = true
	}
	if eventID != "" {
		candidates = map[string]bool{eventID: true}
	}
	result := &PreviewGameEventsResult{CurrentTurn: in.CurrentTurn, Items: []GameEventPreviewItem{}, SelectedIDs: []string{}}
	found := eventID == ""
	for _, event := range detail.Draft.Events {
		if len(candidates) > 0 && !candidates[event.ID] {
			continue
		}
		if event.ID == eventID {
			found = true
		}
		item, err := previewProjectEvent(event, metrics, state, in.CurrentTurn, occurred, in.TriggeredAtTurn)
		if err != nil {
			return nil, err
		}
		result.Items = append(result.Items, item)
	}
	if !found {
		return nil, fmt.Errorf("找不到这个事件")
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		if result.Items[i].Priority != result.Items[j].Priority {
			return result.Items[i].Priority > result.Items[j].Priority
		}
		return result.Items[i].EventID < result.Items[j].EventID
	})
	claimedGroups := map[string]string{}
	for i := range result.Items {
		item := &result.Items[i]
		if !item.Triggered {
			continue
		}
		if item.MutexGroup != "" {
			if winner := claimedGroups[item.MutexGroup]; winner != "" {
				item.Reasons = append(item.Reasons, fmt.Sprintf("与事件“%s”互斥，本次优先呈现优先级更高的事件", winner))
				continue
			}
			claimedGroups[item.MutexGroup] = item.Name
		}
		item.Selected = true
		result.SelectedIDs = append(result.SelectedIDs, item.EventID)
	}
	if len(result.SelectedIDs) == 0 {
		result.Summary = "当前没有符合条件的事件"
	} else {
		result.Summary = fmt.Sprintf("当前有 %d 个事件可以进入下一步", len(result.SelectedIDs))
	}
	return result, nil
}

func previewProjectEvent(event GameProjectEvent, metrics map[string]GameMetricDefinition, state map[string]any, currentTurn int, occurred map[string]bool, triggeredAt map[string]int) (GameEventPreviewItem, error) {
	advanceTurns := eventAdvanceTurns(event)
	category, group, ends := eventCategory(event), eventMutexGroup(event), eventEndsGame(event)
	item := GameEventPreviewItem{EventID: event.ID, Name: event.Name, Category: category, Eligible: true, EventType: category, Triggered: true, Reasons: []string{}, Checks: []GameConditionCheck{}, Before: cloneMap(state), After: cloneMap(state), StateDiff: []GameStateDiff{}, AdvanceTurns: advanceTurns, TimeAdvance: advanceTurns, NextEventIDs: append([]string{}, event.NextEventIDs...), EndGame: ends, EndsGame: ends, Priority: event.Priority, MutexGroup: group, ExclusiveGroup: group}
	if event.Once && occurred[event.ID] {
		item.Triggered = false
		item.Eligible = false
		item.Reasons = append(item.Reasons, "这个事件只能发生一次，本局已经发生过")
	}
	if event.CooldownTurns > 0 && occurred[event.ID] {
		lastTurn, known := triggeredAt[event.ID]
		if !known {
			lastTurn = currentTurn
		}
		remaining := event.CooldownTurns - (currentTurn - lastTurn)
		if remaining >= 0 {
			item.Triggered = false
			item.Eligible = false
			item.Reasons = append(item.Reasons, fmt.Sprintf("这个事件仍在冷却中，还需等待 %d 个回合", remaining+1))
		}
	}
	mode := event.ConditionMode
	if mode == "" {
		mode = "all"
	}
	matchedCount := 0
	for _, condition := range event.Conditions {
		metric, exists := metrics[condition.Target]
		if !exists {
			return item, fmt.Errorf("事件 %s 检查了未定义的世界状态 %s", event.ID, condition.Target)
		}
		if err := validateGameCondition(metric, condition); err != nil {
			return item, fmt.Errorf("事件 %s 的触发条件无效: %w", event.ID, err)
		}
		actual := state[condition.Target]
		matched, err := matchGameCondition(actual, condition.Value, condition.Operator)
		if err != nil {
			return item, err
		}
		if matched {
			matchedCount++
		}
		item.Checks = append(item.Checks, GameConditionCheck{Target: condition.Target, Operator: condition.Operator, Expected: condition.Value, Actual: actual, Matched: matched, Reason: conditionReason(metric, condition, actual, matched)})
	}
	conditionsMatched := len(event.Conditions) == 0 || (mode == "all" && matchedCount == len(event.Conditions)) || (mode == "any" && matchedCount > 0)
	if !conditionsMatched {
		item.Triggered = false
		item.Eligible = false
		for _, check := range item.Checks {
			if !check.Matched {
				item.Reasons = append(item.Reasons, check.Reason)
			}
		}
	}
	if !item.Triggered {
		return item, nil
	}
	item.Reasons = append(item.Reasons, "当前条件允许触发这个事件")
	for _, effect := range event.Effects {
		metric, exists := metrics[effect.Target]
		if !exists {
			return item, fmt.Errorf("事件 %s 修改了未定义的世界状态 %s", event.ID, effect.Target)
		}
		before := item.After[effect.Target]
		after, err := applyGameEffect(metric, before, effect.Value, effect.Operation)
		if err != nil {
			return item, fmt.Errorf("无法计算 %s 的变化: %w", metric.Name, err)
		}
		item.After[effect.Target] = after
		item.StateDiff = append(item.StateDiff, GameStateDiff{MetricID: effect.Target, Before: before, After: after})
	}
	sort.Slice(item.StateDiff, func(i, j int) bool { return item.StateDiff[i].MetricID < item.StateDiff[j].MetricID })
	return item, nil
}

func matchGameCondition(actual, expected any, operator string) (bool, error) {
	if operator == "eq" || operator == "ne" {
		equal := false
		if left, leftOK := gameNumber(actual); leftOK {
			if right, rightOK := gameNumber(expected); rightOK {
				equal = left == right
			} else {
				equal = false
			}
		} else {
			left, _ := json.Marshal(actual)
			right, _ := json.Marshal(expected)
			equal = string(left) == string(right)
		}
		if operator == "ne" {
			return !equal, nil
		}
		return equal, nil
	}
	left, leftOK := gameNumber(actual)
	right, rightOK := gameNumber(expected)
	if !leftOK || !rightOK {
		return false, fmt.Errorf("条件比较需要数字")
	}
	switch operator {
	case "gt":
		return left > right, nil
	case "gte":
		return left >= right, nil
	case "lt":
		return left < right, nil
	case "lte":
		return left <= right, nil
	default:
		return false, fmt.Errorf("不支持的条件比较方式")
	}
}

func conditionReason(metric GameMetricDefinition, condition GameConditionDefinition, actual any, matched bool) string {
	name := strings.TrimSpace(metric.Name)
	if name == "" {
		name = metric.ID
	}
	words := map[string]string{"eq": "等于", "ne": "不等于", "gt": "大于", "gte": "不低于", "lt": "小于", "lte": "不高于"}
	if matched {
		return fmt.Sprintf("%s当前为 %v，符合%s %v 的要求", name, actual, words[condition.Operator], condition.Value)
	}
	return fmt.Sprintf("%s当前为 %v，不符合%s %v 的要求", name, actual, words[condition.Operator], condition.Value)
}
