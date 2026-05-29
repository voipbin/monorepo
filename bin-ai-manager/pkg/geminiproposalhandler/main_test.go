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
