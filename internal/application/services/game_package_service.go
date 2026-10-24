package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"control-panel/internal/domain/gamepackage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const GamePackageSchemaV1 = "game.zerone.io/v1"

var gamePackageKeyPattern = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)+$`)
var gamePackageIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)
var semanticVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

type GamePackageManifest struct {
	SchemaVersion       string                   `json:"schemaVersion"`
	Package             GamePackageMetadata      `json:"package"`
	Registry            GameRegistryManifest     `json:"registry"`
	Content             *GamePackageContent      `json:"content,omitempty"`
	AcceptanceScenarios []GameAcceptanceScenario `json:"acceptanceScenarios,omitempty"`
}

// GamePackageContent is the immutable presentation/story snapshot produced by
// the game-project editor. The machine registry remains generic; this content
// tells higher layers which existing Agents and media assets take part.
type GamePackageContent struct {
	ProjectID       string                    `json:"projectId"`
	ProjectRevision int                       `json:"projectRevision"`
	Genre           string                    `json:"genre,omitempty"`
	PlayGoal        string                    `json:"playGoal,omitempty"`
	Objectives      []GameObjectiveDefinition `json:"objectives,omitempty"`
	NPCs            []GameNPCReference        `json:"npcs,omitempty"`
	StoryEvents     []GameStoryEvent          `json:"storyEvents,omitempty"`
	Scenes          []GameSceneDefinition     `json:"scenes,omitempty"`
	Assets          []GameAssetSnapshot       `json:"assets,omitempty"`
}

type GameNPCReference struct {
	ID                  string                    `json:"id"`
	AgentID             uint64                    `json:"agentId"`
	DisplayName         string                    `json:"displayName,omitempty"`
	Role                string                    `json:"role"`
	Responsibilities    string                    `json:"responsibilities,omitempty"`
	CharacterType       string                    `json:"characterType,omitempty"`
	VisibleMetricIDs    []string                  `json:"visibleMetricIds,omitempty"`
	ExecutableActionIDs []string                  `json:"executableActionIds,omitempty"`
	AdvisoryActionIDs   []string                  `json:"advisoryActionIds,omitempty"`
	KnownInformation    []string                  `json:"knownInformation,omitempty"`
	InformationScope    string                    `json:"informationScope,omitempty"`
	KnowledgeRefs       []string                  `json:"knowledgeRefs,omitempty"`
	RelationIDs         []uint64                  `json:"relationIds,omitempty"`
	Permissions         []string                  `json:"permissions,omitempty"`
	AgentVersion        *GameAgentVersionSnapshot `json:"agentVersion,omitempty"`
}

// GameAgentVersionSnapshot freezes the boundary used by an immutable release.
// It stores hashes and IDs, not a second copy of the Agent prompt or knowledge.
type GameAgentVersionSnapshot struct {
	ContentHash                string                     `json:"contentHash"`
	PersonalityTemplateName    string                     `json:"personalityTemplateName,omitempty"`
	PersonalityTemplateVersion int                        `json:"personalityTemplateVersion,omitempty"`
	KnowledgeRefs              []string                   `json:"knowledgeRefs,omitempty"`
	ToolHashes                 map[string]string          `json:"toolHashes,omitempty"`
	SkillHashes                map[string]string          `json:"skillHashes,omitempty"`
	Relations                  []GameRelationVersionEntry `json:"relations,omitempty"`
}

type GameRelationVersionEntry struct {
	ID                          uint64   `json:"id"`
	SourceAgentID               uint64   `json:"sourceAgentId"`
	TargetAgentID               uint64   `json:"targetAgentId"`
	RelationType                string   `json:"relationType"`
	RelationTypeTemplateName    string   `json:"relationTypeTemplateName,omitempty"`
	RelationTypeTemplateVersion int      `json:"relationTypeTemplateVersion,omitempty"`
	AllowedActions              []string `json:"allowedActions,omitempty"`
	ContextPolicy               string   `json:"contextPolicy,omitempty"`
	DeliveryPolicy              string   `json:"deliveryPolicy,omitempty"`
	Enabled                     bool     `json:"enabled"`
}

type GameStoryEvent struct {
	SceneID                 string                    `json:"sceneId,omitempty"`
	ID                      string                    `json:"id"`
	Name                    string                    `json:"name"`
	Description             string                    `json:"description,omitempty"`
	Category                string                    `json:"category,omitempty"`
	DisplayContent          string                    `json:"displayContent,omitempty"`
	ParticipantIDs          []string                  `json:"participantIds,omitempty"`
	TimeAdvance             int                       `json:"timeAdvance,omitempty"`
	ExclusiveGroup          string                    `json:"exclusiveGroup,omitempty"`
	EndsGame                bool                      `json:"endsGame,omitempty"`
	EventType               string                    `json:"eventType,omitempty"`
	Content                 string                    `json:"content,omitempty"`
	ConditionMode           string                    `json:"conditionMode,omitempty"`
	Conditions              []GameConditionDefinition `json:"conditions,omitempty"`
	ParticipantCharacterIDs []string                  `json:"participantCharacterIds,omitempty"`
	Effects                 []GameEffectDefinition    `json:"effects,omitempty"`
	AdvanceTurns            int                       `json:"advanceTurns,omitempty"`
	Priority                int                       `json:"priority,omitempty"`
	Once                    bool                      `json:"once,omitempty"`
	CooldownTurns           int                       `json:"cooldownTurns,omitempty"`
	MutexGroup              string                    `json:"mutexGroup,omitempty"`
	NextEventIDs            []string                  `json:"nextEventIds,omitempty"`
	EndGame                 bool                      `json:"endGame,omitempty"`
	EndReason               string                    `json:"endReason,omitempty"`
}

type GameAssetSnapshot struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	OriginalName    string `json:"originalName"`
	MimeType        string `json:"mimeType"`
	SizeBytes       int64  `json:"sizeBytes"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
	DurationMs      int64  `json:"durationMs,omitempty"`
	ContentHash     string `json:"contentHash"`
	StorageProvider string `json:"storageProvider"`
	StorageKey      string `json:"storageKey"`
}

type GameSceneDefinition struct {
	ID                  string               `json:"id"`
	Name                string               `json:"name"`
	Description         string               `json:"description,omitempty"`
	Order               int                  `json:"order,omitempty"`
	BackgroundAssetID   string               `json:"backgroundAssetId,omitempty"`
	MusicAssetID        string               `json:"musicAssetId,omitempty"`
	SoundEffectAssetIDs []string             `json:"soundEffectAssetIds,omitempty"`
	Characters          []GameSceneCharacter `json:"characters,omitempty"`
	BackgroundRef       string               `json:"backgroundRef,omitempty"`
	PortraitRefs        map[string]string    `json:"portraitRefs,omitempty"`
	MusicRef            string               `json:"musicRef,omitempty"`
	SoundEffectRefs     []string             `json:"soundEffectRefs,omitempty"`
}

type GamePackageMetadata struct {
	Key         string `json:"key"`
	Version     string `json:"version"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	License     string `json:"license,omitempty"`
}

type GameRegistryManifest struct {
	Metrics []GameMetricDefinition `json:"metrics"`
	Actions []GameActionDefinition `json:"actions"`
	Events  []GameEventDefinition  `json:"events"`
	Endings []GameEndingDefinition `json:"endings"`
}

type GameMetricDefinition struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	DisplayUnit string   `json:"displayUnit,omitempty"`
	Type        string   `json:"type"`
	Initial     any      `json:"initial"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
}

type GameActionDefinition struct {
	ID            string                    `json:"id"`
	Name          string                    `json:"name,omitempty"`
	Description   string                    `json:"description,omitempty"`
	TimeCost      int                       `json:"timeCost,omitempty"`
	ConditionMode string                    `json:"conditionMode,omitempty"`
	Conditions    []GameConditionDefinition `json:"conditions,omitempty"`
	InputSchema   map[string]any            `json:"inputSchema,omitempty"`
	ExampleInput  map[string]any            `json:"exampleInput,omitempty"`
	Effects       []GameEffectDefinition    `json:"effects"`
	Emits         string                    `json:"emits"`
}

// GameConditionDefinition is shared by editable projects and immutable game
// packages. Keeping it beside the existing action/effect definitions avoids a
// second, incompatible rule format in the editor.
type GameConditionDefinition struct {
	Target   string `json:"target"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// GameObjectiveDefinition is a rule-checked player goal. Models may discuss
// it, but only its declared conditions can complete it.
type GameObjectiveDefinition struct {
	ID                   string                    `json:"id"`
	Name                 string                    `json:"name"`
	Description          string                    `json:"description,omitempty"`
	Kind                 string                    `json:"kind,omitempty"`
	IssuerCharacterID    string                    `json:"issuerCharacterId,omitempty"`
	StartsActive         bool                      `json:"startsActive,omitempty"`
	DependsOn            []string                  `json:"dependsOn,omitempty"`
	ConditionMode        string                    `json:"conditionMode,omitempty"`
	Conditions           []GameConditionDefinition `json:"conditions,omitempty"`
	FailureConditionMode string                    `json:"failureConditionMode,omitempty"`
	FailureConditions    []GameConditionDefinition `json:"failureConditions,omitempty"`
	DeadlineTurns        int                       `json:"deadlineTurns,omitempty"`
	CompletedEffects     []GameEffectDefinition    `json:"completedEffects,omitempty"`
	FailedEffects        []GameEffectDefinition    `json:"failedEffects,omitempty"`
	PromiseText          string                    `json:"promiseText,omitempty"`
}

type GameEffectDefinition struct {
	Operation    string `json:"operation"`
	Target       string `json:"target"`
	Value        any    `json:"value,omitempty"`
	ValueFrom    string `json:"valueFrom,omitempty"`
	DefaultValue any    `json:"defaultValue,omitempty"`
}

type GameEventDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// Endings are metadata only in M0. Their deterministic evaluation belongs to M1.
type GameEndingDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type GameAcceptanceScenario struct {
	ID           string            `json:"id"`
	Name         string            `json:"name,omitempty"`
	Description  string            `json:"description,omitempty"`
	InitialState map[string]any    `json:"initialState,omitempty"`
	Commands     []GameTestCommand `json:"commands"`
}

type GameRegistrySummary struct {
	MetricCount int `json:"metricCount"`
	ActionCount int `json:"actionCount"`
	EventCount  int `json:"eventCount"`
	EndingCount int `json:"endingCount"`
}

type GamePackageListItem struct {
	ID          string              `json:"id"`
	Key         string              `json:"key"`
	Version     string              `json:"version"`
	DisplayName string              `json:"displayName"`
	Description string              `json:"description"`
	Status      string              `json:"status"`
	Enabled     bool                `json:"enabled"`
	ContentHash string              `json:"contentHash"`
	Registry    GameRegistrySummary `json:"registry"`
}

type GamePackageListResult struct {
	Items []GamePackageListItem `json:"items"`
	Total int                   `json:"total"`
}

type GamePackageDetail struct {
	Package  GamePackageListItem `json:"package"`
	Manifest GamePackageManifest `json:"manifest"`
}

type GamePackageService struct{ db *gorm.DB }

func NewGamePackageService(db *gorm.DB) *GamePackageService { return &GamePackageService{db: db} }

func validateGamePackageManifest(m *GamePackageManifest) error {
	if m.SchemaVersion != GamePackageSchemaV1 {
		return fmt.Errorf("schemaVersion must be %q", GamePackageSchemaV1)
	}
	if !gamePackageKeyPattern.MatchString(m.Package.Key) {
		return fmt.Errorf("package.key must be a stable reverse-domain style key")
	}
	if !semanticVersionPattern.MatchString(m.Package.Version) {
		return fmt.Errorf("package.version must be semantic version x.y.z")
	}
	if strings.TrimSpace(m.Package.DisplayName) == "" {
		return fmt.Errorf("package.displayName is required")
	}
	metricIDs, actionIDs, eventIDs, endingIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	metricsByID := metricDefinitions(m.Registry.Metrics)
	for i, metric := range m.Registry.Metrics {
		if !gamePackageIDPattern.MatchString(metric.ID) || metricIDs[metric.ID] {
			return fmt.Errorf("registry.metrics[%d].id is invalid or duplicated", i)
		}
		metricIDs[metric.ID] = true
		switch metric.Type {
		case "integer", "number", "string", "boolean":
		default:
			return fmt.Errorf("metric %s has unsupported type %q", metric.ID, metric.Type)
		}
		if err := validateMetricValue(metric, metric.Initial); err != nil {
			return fmt.Errorf("metric %s initial value: %w", metric.ID, err)
		}
		if metric.Minimum != nil && metric.Maximum != nil && *metric.Minimum > *metric.Maximum {
			return fmt.Errorf("metric %s minimum exceeds maximum", metric.ID)
		}
	}
	for i, event := range m.Registry.Events {
		if !gamePackageIDPattern.MatchString(event.ID) || eventIDs[event.ID] {
			return fmt.Errorf("registry.events[%d].id is invalid or duplicated", i)
		}
		eventIDs[event.ID] = true
	}
	for i, ending := range m.Registry.Endings {
		if !gamePackageIDPattern.MatchString(ending.ID) || endingIDs[ending.ID] {
			return fmt.Errorf("registry.endings[%d].id is invalid or duplicated", i)
		}
		endingIDs[ending.ID] = true
	}
	for i, action := range m.Registry.Actions {
		if !gamePackageIDPattern.MatchString(action.ID) || actionIDs[action.ID] {
			return fmt.Errorf("registry.actions[%d].id is invalid or duplicated", i)
		}
		actionIDs[action.ID] = true
		if action.TimeCost < 0 {
			return fmt.Errorf("action %s timeCost cannot be negative", action.ID)
		}
		if action.ConditionMode != "" && action.ConditionMode != "all" && action.ConditionMode != "any" {
			return fmt.Errorf("action %s has unsupported condition mode %q", action.ID, action.ConditionMode)
		}
		for _, condition := range action.Conditions {
			metric, ok := metricsByID[condition.Target]
			if !ok {
				return fmt.Errorf("action %s checks unregistered metric %q", action.ID, condition.Target)
			}
			if err := validateGameCondition(metric, condition); err != nil {
				return fmt.Errorf("action %s condition on %s: %w", action.ID, condition.Target, err)
			}
		}
		if !eventIDs[action.Emits] {
			return fmt.Errorf("action %s emits unregistered event %q", action.ID, action.Emits)
		}
		if len(action.Effects) == 0 {
			return fmt.Errorf("action %s must declare at least one effect", action.ID)
		}
		for _, effect := range action.Effects {
			if !metricIDs[effect.Target] {
				return fmt.Errorf("action %s targets unregistered metric %q", action.ID, effect.Target)
			}
			switch effect.Operation {
			case "set", "increment", "decrement":
			default:
				return fmt.Errorf("action %s has unsupported effect operation %q", action.ID, effect.Operation)
			}
		}
	}
	for _, scenario := range m.AcceptanceScenarios {
		if !gamePackageIDPattern.MatchString(scenario.ID) {
			return fmt.Errorf("acceptance scenario id %q is invalid", scenario.ID)
		}
		for _, command := range scenario.Commands {
			if !actionIDs[command.ActionID] {
				return fmt.Errorf("scenario %s references unregistered action %q", scenario.ID, command.ActionID)
			}
		}
	}
	return nil
}

func metricDefinitions(metrics []GameMetricDefinition) map[string]GameMetricDefinition {
	result := make(map[string]GameMetricDefinition, len(metrics))
	for _, metric := range metrics {
		result[metric.ID] = metric
	}
	return result
}

func validateGameCondition(metric GameMetricDefinition, condition GameConditionDefinition) error {
	switch condition.Operator {
	case "eq", "ne":
		return validateMetricValue(metric, condition.Value)
	case "gt", "gte", "lt", "lte":
		if metric.Type != "integer" && metric.Type != "number" {
			return fmt.Errorf("comparison %s requires a numeric world state", condition.Operator)
		}
		if _, ok := gameNumber(condition.Value); !ok {
			return fmt.Errorf("comparison value must be a number")
		}
		return nil
	default:
		return fmt.Errorf("unsupported operator %q", condition.Operator)
	}
}

func validateMetricValue(metric GameMetricDefinition, value any) error {
	switch metric.Type {
	case "integer":
		n, ok := gameNumber(value)
		if !ok || math.Trunc(n) != n {
			return fmt.Errorf("must be an integer")
		}
	case "number":
		if _, ok := gameNumber(value); !ok {
			return fmt.Errorf("must be a number")
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("must be a boolean")
		}
	}
	return nil
}

func (s *GamePackageService) Register(tenantID, createdBy string, raw json.RawMessage) (*GamePackageDetail, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("tenant is required")
	}
	var manifest GamePackageManifest
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid GamePackageManifest: %w", err)
	}
	if err := validateGamePackageManifest(&manifest); err != nil {
		return nil, err
	}
	canonical, _ := json.Marshal(manifest)
	sum := sha256.Sum256(canonical)
	hash := hex.EncodeToString(sum[:])
	manifestMap := map[string]any{}
	_ = json.Unmarshal(canonical, &manifestMap)
	row := gamepackage.Package{ID: uuid.NewString(), TenantID: tenantID, Key: manifest.Package.Key, Version: manifest.Package.Version, DisplayName: manifest.Package.DisplayName, Description: manifest.Package.Description, Status: gamepackage.StatusDisabled, Manifest: manifestMap, ContentHash: hash, CreatedBy: strings.TrimSpace(createdBy)}
	if err := s.db.Create(&row).Error; err != nil {
		if isDuplicate(err) {
			var existing gamepackage.Package
			if qerr := s.db.Where("tenant_id=? AND `key`=? AND version=?", tenantID, row.Key, row.Version).First(&existing).Error; qerr != nil {
				return nil, qerr
			}
			if existing.ContentHash != hash {
				return nil, fmt.Errorf("package %s@%s already exists with different content", row.Key, row.Version)
			}
			row = existing
		} else {
			return nil, err
		}
	}
	return packageDetail(row, manifest), nil
}

func packageItem(row gamepackage.Package, manifest GamePackageManifest) GamePackageListItem {
	return GamePackageListItem{ID: row.ID, Key: row.Key, Version: row.Version, DisplayName: row.DisplayName, Description: row.Description, Status: row.Status, Enabled: row.Status == gamepackage.StatusEnabled, ContentHash: row.ContentHash, Registry: GameRegistrySummary{MetricCount: len(manifest.Registry.Metrics), ActionCount: len(manifest.Registry.Actions), EventCount: len(manifest.Registry.Events), EndingCount: len(manifest.Registry.Endings)}}
}

func packageDetail(row gamepackage.Package, manifest GamePackageManifest) *GamePackageDetail {
	return &GamePackageDetail{Package: packageItem(row, manifest), Manifest: manifest}
}

func decodeGameManifest(row gamepackage.Package) (GamePackageManifest, error) {
	raw, err := json.Marshal(row.Manifest)
	if err != nil {
		return GamePackageManifest{}, err
	}
	var m GamePackageManifest
	err = json.Unmarshal(raw, &m)
	return m, err
}

func (s *GamePackageService) List(tenantID string, enabled *bool) (*GamePackageListResult, error) {
	q := s.db.Where("tenant_id=?", tenantID)
	if enabled != nil {
		status := gamepackage.StatusDisabled
		if *enabled {
			status = gamepackage.StatusEnabled
		}
		q = q.Where("status=?", status)
	}
	var rows []gamepackage.Package
	if err := q.Order("`key`, version").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]GamePackageListItem, 0, len(rows))
	for _, row := range rows {
		m, err := decodeGameManifest(row)
		if err != nil {
			return nil, err
		}
		items = append(items, packageItem(row, m))
	}
	return &GamePackageListResult{Items: items, Total: len(items)}, nil
}

func (s *GamePackageService) Get(tenantID, id string) (*GamePackageDetail, error) {
	var row gamepackage.Package
	if err := s.db.Where("tenant_id=? AND id=?", tenantID, id).First(&row).Error; err != nil {
		return nil, err
	}
	m, err := decodeGameManifest(row)
	if err != nil {
		return nil, err
	}
	return packageDetail(row, m), nil
}

func (s *GamePackageService) SetEnabled(tenantID, id string, enabled bool) (*GamePackageListItem, error) {
	status := gamepackage.StatusDisabled
	if enabled {
		status = gamepackage.StatusEnabled
	}
	res := s.db.Model(&gamepackage.Package{}).Where("tenant_id=? AND id=?", tenantID, id).Update("status", status)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	detail, err := s.Get(tenantID, id)
	if err != nil {
		return nil, err
	}
	return &detail.Package, nil
}

type GameTestCommand struct {
	ActionID string         `json:"actionId"`
	Input    map[string]any `json:"input,omitempty"`
}
type GameTestRunInput struct {
	InitialState map[string]any    `json:"initialState,omitempty"`
	Commands     []GameTestCommand `json:"commands"`
}
type GameStateDiff struct {
	MetricID string `json:"metricId"`
	Before   any    `json:"before"`
	After    any    `json:"after"`
}
type GameTestEvent struct {
	ID       string         `json:"id"`
	Sequence int            `json:"sequence"`
	ActionID string         `json:"actionId"`
	Payload  map[string]any `json:"payload,omitempty"`
}
type GameTestStep struct {
	Sequence  int             `json:"sequence"`
	Command   GameTestCommand `json:"command"`
	Event     GameTestEvent   `json:"event"`
	StateDiff []GameStateDiff `json:"stateDiff"`
	State     map[string]any  `json:"state"`
}
type GameTestRunResult struct {
	PackageID      string         `json:"packageId"`
	PackageVersion string         `json:"packageVersion"`
	InitialState   map[string]any `json:"initialState"`
	FinalState     map[string]any `json:"finalState"`
	Steps          []GameTestStep `json:"steps"`
	Passed         bool           `json:"passed"`
}

func (s *GamePackageService) TestRun(tenantID, id string, in GameTestRunInput) (*GameTestRunResult, error) {
	detail, err := s.Get(tenantID, id)
	if err != nil {
		return nil, err
	}
	if !detail.Package.Enabled {
		return nil, fmt.Errorf("game package is disabled")
	}
	state := map[string]any{}
	defs := map[string]GameMetricDefinition{}
	for _, metric := range detail.Manifest.Registry.Metrics {
		state[metric.ID] = metric.Initial
		defs[metric.ID] = metric
	}
	for key, value := range in.InitialState {
		metric, ok := defs[key]
		if !ok {
			return nil, fmt.Errorf("initialState contains unregistered metric %q", key)
		}
		if err := validateMetricValue(metric, value); err != nil {
			return nil, fmt.Errorf("initialState.%s: %w", key, err)
		}
		state[key] = value
	}
	initial := cloneMap(state)
	actions := map[string]GameActionDefinition{}
	for _, a := range detail.Manifest.Registry.Actions {
		actions[a.ID] = a
	}
	steps := make([]GameTestStep, 0, len(in.Commands))
	for index, cmd := range in.Commands {
		action, ok := actions[cmd.ActionID]
		if !ok {
			return nil, fmt.Errorf("command %d references unregistered action %q", index+1, cmd.ActionID)
		}
		matchedCount := 0
		for _, condition := range action.Conditions {
			matched, matchErr := matchGameCondition(state[condition.Target], condition.Value, condition.Operator)
			if matchErr != nil {
				return nil, fmt.Errorf("command %d action %s condition: %w", index+1, action.ID, matchErr)
			}
			if matched {
				matchedCount++
			}
		}
		conditionMode := action.ConditionMode
		if conditionMode == "" {
			conditionMode = "all"
		}
		allowed := len(action.Conditions) == 0 || (conditionMode == "all" && matchedCount == len(action.Conditions)) || (conditionMode == "any" && matchedCount > 0)
		if !allowed {
			return nil, fmt.Errorf("command %d action %s does not meet its conditions", index+1, action.ID)
		}
		diffs := []GameStateDiff{}
		for _, effect := range action.Effects {
			before := state[effect.Target]
			value := effect.Value
			if effect.ValueFrom != "" {
				value = cmd.Input[effect.ValueFrom]
				if value == nil {
					value = effect.DefaultValue
				}
			}
			after, err := applyGameEffect(defs[effect.Target], before, value, effect.Operation)
			if err != nil {
				return nil, fmt.Errorf("command %d action %s: %w", index+1, action.ID, err)
			}
			state[effect.Target] = after
			diffs = append(diffs, GameStateDiff{MetricID: effect.Target, Before: before, After: after})
		}
		sort.Slice(diffs, func(i, j int) bool { return diffs[i].MetricID < diffs[j].MetricID })
		seq := index + 1
		event := GameTestEvent{ID: action.Emits, Sequence: seq, ActionID: action.ID, Payload: cloneMap(cmd.Input)}
		steps = append(steps, GameTestStep{Sequence: seq, Command: cmd, Event: event, StateDiff: diffs, State: cloneMap(state)})
	}
	return &GameTestRunResult{PackageID: id, PackageVersion: detail.Package.Version, InitialState: initial, FinalState: cloneMap(state), Steps: steps, Passed: true}, nil
}

func applyGameEffect(metric GameMetricDefinition, before, value any, operation string) (any, error) {
	var after any
	if operation == "set" {
		after = value
	} else {
		left, ok := gameNumber(before)
		if !ok {
			return nil, fmt.Errorf("%s requires numeric target %s", operation, metric.ID)
		}
		right, ok := gameNumber(value)
		if !ok {
			return nil, fmt.Errorf("%s requires numeric value", operation)
		}
		if operation == "decrement" {
			right = -right
		}
		after = left + right
		if metric.Type == "integer" {
			after = int64(math.Round(after.(float64)))
		}
	}
	if n, ok := gameNumber(after); ok {
		if metric.Minimum != nil && n < *metric.Minimum {
			n = *metric.Minimum
		}
		if metric.Maximum != nil && n > *metric.Maximum {
			n = *metric.Maximum
		}
		if metric.Type == "integer" {
			after = int64(math.Round(n))
		} else {
			after = n
		}
	}
	if err := validateMetricValue(metric, after); err != nil {
		return nil, err
	}
	return after, nil
}

func gameNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	raw, _ := json.Marshal(in)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}
