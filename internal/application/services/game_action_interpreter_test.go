package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type gameActionModelStub struct {
	response string
	err      error
}

type gameActionModelPanicStub struct{}

func (gameActionModelPanicStub) InterpretGameAction(context.Context, string) ([]byte, error) {
	panic("a strong local match must not call the model")
}

func (s gameActionModelStub) InterpretGameAction(context.Context, string) ([]byte, error) {
	return []byte(s.response), s.err
}

func gameInterpretRequest(text string) GameActionInterpretationRequest {
	return GameActionInterpretationRequest{PlayerText: text, Actions: []GameActionInterpretationAction{
		{ID: "investigate", Name: "调查账目", Description: "核对财务记录", Allowed: true, Targets: []string{"财务部"}},
		{ID: "negotiate", Name: "私下谈判", Description: "与董事协商", Allowed: true, Targets: []string{"董事长"}},
		{ID: "seize", Name: "强制接管", Description: "董事长专属行动", Allowed: false},
	}}
}

func TestGameActionInterpreterDirectIntent(t *testing.T) {
	svc := NewGameActionInterpreter(gameActionModelStub{response: `{"actionId":"investigate","target":"财务部","parameters":{"depth":"full"},"steps":["调取流水","交叉核验"],"confidence":0.94,"summary":"全面调查财务部账目"}`})
	got, err := svc.Interpret(context.Background(), gameInterpretRequest("先调取流水，再核对异常付款"))
	require.NoError(t, err)
	require.Equal(t, "investigate", got.ActionID)
	require.False(t, got.NeedsClarification)
	require.Equal(t, "财务部", got.Target)
	require.Equal(t, []string{"调取流水", "交叉核验"}, got.Steps)
}

func TestGameActionInterpreterAmbiguousIntent(t *testing.T) {
	svc := NewGameActionInterpreter(gameActionModelStub{response: `{"actionId":"investigate","confidence":0.41,"ambiguous":true,"needsClarification":true,"clarifyingQuestion":"你要查哪一笔账？"}`})
	got, err := svc.Interpret(context.Background(), gameInterpretRequest("查一下那个事情"))
	require.NoError(t, err)
	require.True(t, got.NeedsClarification)
	require.Equal(t, "你要查哪一笔账？", got.ClarifyingQuestion)
}

func TestGameActionInterpreterRejectsUnrelatedText(t *testing.T) {
	svc := NewGameActionInterpreter(gameActionModelStub{response: `{"confidence":0.1,"ambiguous":true,"needsClarification":true,"clarifyingQuestion":"这句话还没有对应到游戏行动。"}`})
	got, err := svc.Interpret(context.Background(), gameInterpretRequest("今天天气真不错"))
	require.NoError(t, err)
	require.Empty(t, got.ActionID)
	require.True(t, got.NeedsClarification)
}

func TestGameActionInterpreterSplitsCompositeIntent(t *testing.T) {
	svc := NewGameActionInterpreter(gameActionModelStub{response: `{"confidence":0.55,"ambiguous":true,"needsClarification":true,"steps":["调查账目","私下谈判"],"clarifyingQuestion":"你想先做哪一件？"}`})
	got, err := svc.Interpret(context.Background(), gameInterpretRequest("先调查账目，再找董事长私下谈判"))
	require.NoError(t, err)
	require.Empty(t, got.ActionID)
	require.True(t, got.Ambiguous)
	require.Equal(t, []string{"调查账目", "私下谈判"}, got.Steps)
}

func TestGameActionInterpreterBlocksUnauthorizedAction(t *testing.T) {
	svc := NewGameActionInterpreter(gameActionModelStub{response: `{"actionId":"seize","confidence":0.99,"summary":"强制接管公司"}`})
	got, err := svc.Interpret(context.Background(), gameInterpretRequest("跳过董事会，我直接接管公司"))
	require.NoError(t, err)
	require.Empty(t, got.ActionID)
	require.True(t, got.NeedsClarification)
	require.Contains(t, got.ClarifyingQuestion, "不能执行")
}

func TestGameActionInterpreterConservativeFallback(t *testing.T) {
	svc := NewGameActionInterpreter(gameActionModelStub{err: errors.New("provider timeout")})
	direct, err := svc.Interpret(context.Background(), gameInterpretRequest("我要调查账目"))
	require.NoError(t, err)
	require.Equal(t, "investigate", direct.ActionID)
	require.False(t, direct.Degraded)

	unrelated, err := svc.Interpret(context.Background(), gameInterpretRequest("帮我随便处理一下"))
	require.NoError(t, err)
	require.Empty(t, unrelated.ActionID)
	require.True(t, unrelated.NeedsClarification)
	require.Contains(t, unrelated.ClarifyingQuestion, "没有明确对应")

	composite, err := svc.Interpret(context.Background(), gameInterpretRequest("我要调查账目再私下谈判"))
	require.NoError(t, err)
	require.Empty(t, composite.ActionID)
	require.True(t, composite.Ambiguous)
}

func TestGameActionInterpreterFallbackUnderstandsNaturalChineseDescription(t *testing.T) {
	svc := NewGameActionInterpreter(gameActionModelStub{err: errors.New("provider timeout")})
	req := GameActionInterpretationRequest{PlayerText: "先别对外解释，让财务把原始付款记录全部封存并交叉核验", Actions: []GameActionInterpretationAction{
		{ID: "secure-evidence", Name: "封存关键证据", Description: "让财务与法务共同核对付款记录并封存原始材料。", Allowed: true},
		{ID: "call-board", Name: "召开紧急董事会", Description: "向关键董事公开已核验事实，争取阶段性授权。", Allowed: true},
		{ID: "disclose", Name: "向监管主动披露", Description: "提交材料并说明补救计划。", Allowed: true},
	}}

	got, err := svc.Interpret(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "secure-evidence", got.ActionID)
	require.False(t, got.NeedsClarification)
	require.False(t, got.Degraded)
}

func TestGameActionInterpreterUsesFastLocalMatchBeforeModel(t *testing.T) {
	svc := NewGameActionInterpreter(gameActionModelPanicStub{})
	req := GameActionInterpretationRequest{PlayerText: "让财务把原始付款记录封存并交叉核验", Actions: []GameActionInterpretationAction{
		{ID: "secure-evidence", Name: "封存关键证据", Description: "让财务与法务共同核对付款记录并封存原始材料。", Allowed: true},
		{ID: "call-board", Name: "召开紧急董事会", Description: "向关键董事公开已核验事实，争取阶段性授权。", Allowed: true},
	}}

	got, err := svc.Interpret(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "secure-evidence", got.ActionID)
	require.False(t, got.Degraded)
}
