package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

var ErrGameActionModelUnavailable = errors.New("game action language model unavailable")

// GameActionInterpretation is only a proposal. It must never be used to mutate
// game state directly; the player-session service must persist it as pending and
// require an explicit confirmation before invoking the deterministic rule engine.
type GameActionInterpretation struct {
	ActionID           string         `json:"actionId,omitempty"`
	Target             string         `json:"target,omitempty"`
	Parameters         map[string]any `json:"parameters"`
	Steps              []string       `json:"steps"`
	Confidence         float64        `json:"confidence"`
	Ambiguous          bool           `json:"ambiguous"`
	NeedsClarification bool           `json:"needsClarification"`
	ClarifyingQuestion string         `json:"clarifyingQuestion,omitempty"`
	Summary            string         `json:"summary"`
	Degraded           bool           `json:"degraded"`
	DegradedReason     string         `json:"degradedReason,omitempty"`
}

type GameActionInterpretationRequest struct {
	PlayerText string
	Actions    []GameActionInterpretationAction
	Context    GameActionInterpretationContext
}

type GameActionInterpretationAction struct {
	ID          string
	Name        string
	Description string
	Allowed     bool
	Targets     []string
}

type GameActionInterpretationContext struct {
	VisibleState map[string]any
	PlayerRole   string
}

// GameActionLanguageModel is deliberately narrower than a provider SDK. A
// provider-backed adapter can use the tenant's existing provider/model config,
// but only returns JSON for a candidate plan; it receives no state-writing tool.
type GameActionLanguageModel interface {
	InterpretGameAction(ctx context.Context, prompt string) ([]byte, error)
}

type GameActionInterpreter struct {
	model GameActionLanguageModel
}

func NewGameActionInterpreter(model GameActionLanguageModel) *GameActionInterpreter {
	return &GameActionInterpreter{model: model}
}

func (s *GameActionInterpreter) Interpret(ctx context.Context, req GameActionInterpretationRequest) (GameActionInterpretation, error) {
	text := strings.TrimSpace(req.PlayerText)
	if text == "" {
		return clarification("请说说你想做什么。"), nil
	}
	if len([]rune(text)) > 1000 {
		return clarification("这段话有点长，请先说最想做的一件事。"), nil
	}
	// Most game actions describe their intent clearly in the editor. Resolve a
	// strong, unique local match before paying the latency cost of an LLM call.
	// Ambiguous/free-form input still falls through to the model.
	fast := conservativeGameActionFallback(req, nil)
	if fast.ActionID != "" && !fast.NeedsClarification {
		fast.Degraded = false
		fast.DegradedReason = ""
		fast.Summary = strings.Replace(fast.Summary, "当前为降级识别，请确认后再执行。", "请确认后再执行。", 1)
		return fast, nil
	}
	if s.model == nil {
		return conservativeGameActionFallback(req, ErrGameActionModelUnavailable), nil
	}
	raw, err := s.model.InterpretGameAction(ctx, buildGameActionPrompt(req))
	if err != nil {
		return conservativeGameActionFallback(req, err), nil
	}
	var result GameActionInterpretation
	if err := json.Unmarshal(raw, &result); err != nil {
		return conservativeGameActionFallback(req, fmt.Errorf("invalid model response: %w", err)), nil
	}
	return validateGameActionInterpretation(result, req), nil
}

func buildGameActionPrompt(req GameActionInterpretationRequest) string {
	type promptAction struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description,omitempty"`
		Allowed     bool     `json:"allowed"`
		Targets     []string `json:"targets,omitempty"`
	}
	actions := make([]promptAction, 0, len(req.Actions))
	for _, action := range req.Actions {
		actions = append(actions, promptAction{ID: action.ID, Name: action.Name, Description: action.Description, Allowed: action.Allowed, Targets: action.Targets})
	}
	payload, _ := json.Marshal(map[string]any{"playerText": req.PlayerText, "playerRole": req.Context.PlayerRole, "visibleState": req.Context.VisibleState, "actions": actions})
	return "你只负责理解玩家意图，不得执行行动或修改状态。只能从 actions 中选择 allowed=true 的 actionId。" +
		"如果玩家表达含糊、无关、包含多个互相独立的行动，或要求执行无权限行动，必须 needsClarification=true。" +
		"只返回 JSON：actionId,target,parameters,steps,confidence,ambiguous,needsClarification,clarifyingQuestion,summary。\n" + string(payload)
}

func validateGameActionInterpretation(result GameActionInterpretation, req GameActionInterpretationRequest) GameActionInterpretation {
	result.Parameters = cloneAnyMap(result.Parameters)
	result.Steps = compactStrings(result.Steps, 8)
	if result.Confidence < 0 {
		result.Confidence = 0
	}
	if result.Confidence > 1 {
		result.Confidence = 1
	}
	var selected *GameActionInterpretationAction
	for i := range req.Actions {
		if req.Actions[i].ID == result.ActionID {
			selected = &req.Actions[i]
			break
		}
	}
	if result.ActionID == "" || selected == nil {
		result.ActionID = ""
		result.Ambiguous = true
		result.NeedsClarification = true
		result.ClarifyingQuestion = defaultString(result.ClarifyingQuestion, "我还不能确定你想采取哪项行动，可以再具体一点吗？")
		return result
	}
	if !selected.Allowed {
		result.ActionID = ""
		result.Ambiguous = false
		result.NeedsClarification = true
		result.ClarifyingQuestion = "当前身份不能执行这项行动，请换一种符合角色权限的做法。"
		return result
	}
	if result.Target != "" && len(selected.Targets) > 0 && !containsExactString(selected.Targets, result.Target) {
		result.ActionID = ""
		result.Ambiguous = true
		result.NeedsClarification = true
		result.ClarifyingQuestion = "行动对象不明确，请从当前可接触的人物或对象中重新说明。"
		return result
	}
	if result.Confidence < 0.65 {
		result.NeedsClarification = true
		result.ClarifyingQuestion = defaultString(result.ClarifyingQuestion, "我不太确定你的意思，是要“"+selected.Name+"”吗？")
	}
	result.Summary = defaultString(result.Summary, "准备“"+selected.Name+"”，确认前不会改变游戏状态。")
	return result
}

func conservativeGameActionFallback(req GameActionInterpretationRequest, cause error) GameActionInterpretation {
	normalized := normalizePlayerIntent(req.PlayerText)
	mentioned := make([]GameActionInterpretationAction, 0, 2)
	for _, action := range req.Actions {
		if !action.Allowed {
			continue
		}
		name := normalizePlayerIntent(action.Name)
		if name != "" && strings.Contains(normalized, name) {
			mentioned = append(mentioned, action)
		}
	}
	result := GameActionInterpretation{Parameters: map[string]any{}, Confidence: 0, Degraded: true, DegradedReason: "自然语言理解服务暂时不可用"}
	if len(mentioned) == 1 {
		result.ActionID = mentioned[0].ID
		result.Summary = "识别到明确行动“" + mentioned[0].Name + "”。当前为降级识别，请确认后再执行。"
		result.Steps = []string{mentioned[0].Name}
		result.Confidence = 0.72
		return result
	}
	// Keep a playable local game when the language model is temporarily
	// unavailable. Chinese player input rarely repeats an editor-defined action
	// name verbatim, so compare short phrases from both the action name and its
	// plain-language description. This only proposes one existing allowed action;
	// confirmation and all state changes still go through the deterministic engine.
	if len(mentioned) == 0 {
		if matched, score, separated := bestFallbackAction(req, normalized); matched != nil && separated && score >= 3 {
			result.ActionID = matched.ID
			result.Summary = "理解为“" + matched.Name + "”。请确认后再执行。"
			result.Steps = []string{matched.Name}
			result.Confidence = min(0.78, 0.58+float64(score)*0.025)
			return result
		}
	}
	result.Ambiguous = len(mentioned) > 1
	result.NeedsClarification = true
	if len(mentioned) > 1 {
		names := make([]string, 0, len(mentioned))
		for _, match := range mentioned {
			names = append(names, match.Name)
		}
		sort.Strings(names)
		result.ClarifyingQuestion = "你同时提到了“" + strings.Join(names, "”和“") + "”，请先选择一件要做的事。"
	} else {
		result.ClarifyingQuestion = "自然语言理解服务暂时不可用，且这句话没有明确对应已有行动。请直接说出一个行动名称后重试。"
	}
	_ = cause // Keep provider errors out of player-facing copy.
	return result
}

func bestFallbackAction(req GameActionInterpretationRequest, normalizedText string) (*GameActionInterpretationAction, int, bool) {
	type candidate struct {
		index int
		score int
	}
	candidates := make([]candidate, 0, len(req.Actions))
	playerPhrases := intentPhrases(normalizedText)
	for i, action := range req.Actions {
		if !action.Allowed {
			continue
		}
		actionPhrases := intentPhrases(normalizePlayerIntent(action.Name + action.Description))
		score := 0
		for phrase := range playerPhrases {
			if _, ok := actionPhrases[phrase]; ok {
				score++
			}
		}
		candidates = append(candidates, candidate{index: i, score: score})
	}
	if len(candidates) == 0 {
		return nil, 0, false
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	second := 0
	if len(candidates) > 1 {
		second = candidates[1].score
	}
	// Two independently meaningful matches are a multi-step request, even when
	// one happens to share more words with the player's sentence. Do not silently
	// discard the second instruction; ask the player which step should happen now.
	separated := candidates[0].score >= second+2 && second < 3
	return &req.Actions[candidates[0].index], candidates[0].score, separated
}

func intentPhrases(value string) map[string]struct{} {
	runes := []rune(value)
	phrases := make(map[string]struct{}, len(runes))
	for size := 2; size <= 4; size++ {
		for i := 0; i+size <= len(runes); i++ {
			phrases[string(runes[i:i+size])] = struct{}{}
		}
	}
	return phrases
}

func isExplicitFallbackIntent(text, actionName string) bool {
	for _, candidate := range []string{actionName, "我要" + actionName, "我想" + actionName, "先" + actionName} {
		if text == candidate {
			return true
		}
	}
	return false
}

func clarification(question string) GameActionInterpretation {
	return GameActionInterpretation{Parameters: map[string]any{}, Ambiguous: true, NeedsClarification: true, ClarifyingQuestion: question}
}

func normalizePlayerIntent(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || strings.ContainsRune("，。！？、；：,.!?;:\"'（）()", r) {
			return -1
		}
		return unicode.ToLower(r)
	}, strings.TrimSpace(value))
}

func compactStrings(values []string, limit int) []string {
	result := make([]string, 0, min(len(values), limit))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
			if len(result) == limit {
				break
			}
		}
	}
	return result
}

func cloneAnyMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func containsExactString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
