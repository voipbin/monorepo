package eval

import (
	"context"
	"errors"
	"fmt"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/pkg/builderhandler"
)

// SyntheticCase is one scenario-15 case: a hand-built history that injects the
// code-computed session facts (user_turns, draft_exists, checkpoint) directly,
// with no simulator. A human checks whether the first reply handles the
// checkpoint sentence as rule 3 says. No Korean text is matched automatically.
type SyntheticCase struct {
	Name     string
	Turns    int  // user turns
	HasDraft bool // current_draft present
	Expect   string
	// Summarising marks the case whose history has all four dimensions filled
	// and no summary yet, so this turn is a summary turn and the checkpoint
	// sentence must be omitted.
	Summarising bool
}

// Fixed pet-grooming conversation (a domain used by neither the few-shot
// examples nor the other scenarios). Answer i is the user's i-th message.
var synthAnswers = []string{
	"반려견 미용실 예약을 전화로 받는 봇을 만들고 싶어요",
	"음성 전화로 받고 손님은 한국어를 써요",
	"주로 단골 보호자들이고 대부분 정기 미용 예약이에요",
	"미용 중 건강 이상 상담은 절대 하지 않고 수의사 연결도 안 해요",
	"예약 변경은 하루 전까지만 받아요",
	"공격성이 있는 아이는 사람이 직접 상담해요",
	"대형견은 시간이 두 배 걸려요",
	"요금은 전화로 말하지 않아요",
	"예약금은 없어요",
	"당일 취소는 다음 예약에 불이익이 있어요",
}

var synthQuestions = []string{
	"손님이 전화로 예약하나요, 채팅으로 하나요?",
	"주로 어떤 보호자가 전화하나요?",
	"예약을 잡은 뒤 변경이나 취소는 어떻게 처리하나요?",
	"이 봇이 절대 해서는 안 되는 일이 있나요?",
	"사람이 직접 받아야 하는 경우가 있나요?",
	"견종이나 크기에 따라 달라지는 것이 있나요?",
	"요금은 어떻게 안내하나요?",
	"예약금이나 선결제가 있나요?",
	"당일 취소는 어떻게 처리하나요?",
}

// SyntheticCases returns the five scenario-15 cases in design order.
func SyntheticCases() []SyntheticCase {
	return []SyntheticCase{
		{Name: "6 turns, no draft, checkpoint", Turns: 6, HasDraft: false, Expect: "ends with one sentence offering to write the draft now or keep refining"},
		{Name: "6 turns, draft exists, checkpoint", Turns: 6, HasDraft: true, Expect: "ends with one sentence asking whether to refine further"},
		{Name: "8 turns, no draft, no checkpoint", Turns: 8, HasDraft: false, Expect: "no checkpoint sentence at all"},
		{Name: "10 turns, draft exists, checkpoint", Turns: 10, HasDraft: true, Expect: "ends with one sentence asking whether to refine further"},
		{Name: "6 turns, summary turn overlaps the checkpoint", Turns: 6, HasDraft: false, Summarising: true, Expect: "summarises and asks to write the draft; the checkpoint sentence is omitted"},
	}
}

// Request builds the case's request. It is always valid input.
func (c SyntheticCase) Request() *builder.ChatRequest {
	var msgs []builder.Message
	for i := 0; i < c.Turns; i++ {
		msgs = append(msgs, builder.Message{Role: builder.RoleUser, Content: synthAnswers[i]})
		if i < c.Turns-1 {
			q := synthQuestions[i%len(synthQuestions)]
			msgs = append(msgs, builder.Message{Role: builder.RoleAssistant, Content: q})
		}
	}
	// The last user message decides what this turn should be. A summary-turn
	// case ends with an answer that closes every dimension, so the model is
	// expected to summarise. Every other case ends with an answer that opens a
	// new fork, so the model is expected to keep digging.
	last := &msgs[len(msgs)-1]
	if c.Summarising {
		last.Content = "네 그 정도예요. 더 정해야 할 건 없어요"
	} else {
		last.Content = "그런데 사람에게 넘기는 기준은 아직 정하지 못했어요"
	}
	req := &builder.ChatRequest{Messages: msgs}
	if c.HasDraft {
		req.CurrentDraft = &builder.Draft{
			Name:       "반려견 미용 예약",
			Detail:     "반려견 미용 예약을 받는 음성 AI",
			InitPrompt: "# Grooming Reservation Assistant\n\n## Identity & Purpose\nTake grooming reservations by phone in Korean.\n",
			ToolNames:  []string{"connect_call"},
		}
	}
	return req
}

// RunSynthetic runs every scenario-15 case once.
func RunSynthetic(ctx context.Context, sender builderhandler.Sender, cfg builderhandler.Config) []RunResult {
	var out []RunResult
	for i, c := range SyntheticCases() {
		id := fmt.Sprintf("s15-%d", i+1)
		r := RunResult{RunID: id + "#1", ScenarioID: id, Group: "s15", Title: c.Name}
		req := c.Request()
		turn, err := builderhandler.RunTurn(ctx, sender, cfg, req)
		r.BuilderCalls = 1
		r.Turns = append(r.Turns, req.Messages...)
		if turn != nil {
			r.BuilderPromptTokens = turn.Usage.PromptTokens
			r.BuilderCompletionTokens = turn.Usage.CompletionTokens
		}
		if err != nil {
			if errors.Is(err, builderhandler.ErrInvalidResponse) || errors.Is(err, builderhandler.ErrTruncated) {
				r.ParseFailures = 1
			}
			r.Aborted = err.Error()
			out = append(out, r)
			continue
		}
		r.Turns = append(r.Turns, builder.Message{Role: builder.RoleAssistant, Content: turn.Parsed.Message})
		if turn.Parsed.Draft != nil {
			r.FinalDraft = turn.Parsed.Draft
			r.FirstResponseHasDraft = true
		}
		out = append(out, r)
	}
	return out
}
