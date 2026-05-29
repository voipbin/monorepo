package geminiproposalhandler

import (
	"strings"
	"testing"
)

func TestParseProposalResponse_Valid(t *testing.T) {
	h := &geminiProposalHandler{}
	in := []byte(`{"proposed_prompt":"You are a polite assistant.","rationale":"Improved tone."}`)
	out, err := h.ParseProposalResponse(in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.ProposedPrompt != "You are a polite assistant." {
		t.Errorf("ProposedPrompt mismatch: %q", out.ProposedPrompt)
	}
	if out.Rationale != "Improved tone." {
		t.Errorf("Rationale mismatch: %q", out.Rationale)
	}
}

func TestParseProposalResponse_MalformedJSON(t *testing.T) {
	h := &geminiProposalHandler{}
	_, err := h.ParseProposalResponse([]byte(`{not json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseProposalResponse_EmptyProposedPrompt(t *testing.T) {
	h := &geminiProposalHandler{}
	in := []byte(`{"proposed_prompt":"","rationale":"hi"}`)
	_, err := h.ParseProposalResponse(in)
	if err == nil {
		t.Fatal("expected error for empty proposed_prompt")
	}
}

func TestParseProposalResponse_EmptyRationale(t *testing.T) {
	h := &geminiProposalHandler{}
	in := []byte(`{"proposed_prompt":"valid","rationale":""}`)
	_, err := h.ParseProposalResponse(in)
	if err == nil {
		t.Fatal("expected error for empty rationale")
	}
}

func TestParseProposalResponse_ProposedPromptTooLong(t *testing.T) {
	h := &geminiProposalHandler{}
	long := strings.Repeat("a", maxProposedPromptChars+1)
	in := []byte(`{"proposed_prompt":"` + long + `","rationale":"ok"}`)
	_, err := h.ParseProposalResponse(in)
	if err == nil {
		t.Fatal("expected error for proposed_prompt over cap")
	}
}

func TestParseProposalResponse_RationaleTooLong(t *testing.T) {
	h := &geminiProposalHandler{}
	long := strings.Repeat("a", maxRationaleChars+1)
	in := []byte(`{"proposed_prompt":"ok","rationale":"` + long + `"}`)
	_, err := h.ParseProposalResponse(in)
	if err == nil {
		t.Fatal("expected error for rationale over cap")
	}
}

func TestBuildPrompt_IncludesEveryAuditBlock(t *testing.T) {
	h := &geminiProposalHandler{}
	audits := []AuditBlock{
		{Index: 1, OverallScore: 3, HelpfulnessR: "h1", AccuracyR: "a1", ToneR: "t1", GoalCompletionR: "g1", Summary: "s1", Transcript: "T1"},
		{Index: 2, OverallScore: 4, HelpfulnessR: "h2", AccuracyR: "a2", ToneR: "t2", GoalCompletionR: "g2", Summary: "s2", Transcript: "T2"},
	}
	out := h.BuildPrompt("ORIG", audits, "en-US")
	if !strings.Contains(out, "ORIG") {
		t.Error("missing original prompt")
	}
	if !strings.Contains(out, "AUDIT 1 / 2") || !strings.Contains(out, "AUDIT 2 / 2") {
		t.Error("missing audit headers")
	}
	if !strings.Contains(out, "T1") || !strings.Contains(out, "T2") {
		t.Error("missing transcripts")
	}
	if !strings.Contains(out, `"en-US"`) {
		t.Error("missing language directive")
	}
}

func TestBuildPrompt_SanitizesTripleDash(t *testing.T) {
	h := &geminiProposalHandler{}
	out := h.BuildPrompt("hi --- there", nil, "en-US")
	if strings.Contains(out, "hi --- there") {
		t.Error("triple-dash was not sanitized")
	}
}

func TestBuildPrompt_OmitsToolUsageBlockWhenEmpty(t *testing.T) {
	h := &geminiProposalHandler{}
	out := h.BuildPrompt("orig", []AuditBlock{{Index: 1, OverallScore: 4, ToolUsageR: ""}}, "en-US")
	if strings.Contains(out, "tool_usage:") {
		t.Error("expected tool_usage line to be omitted when ToolUsageR is empty")
	}
}
