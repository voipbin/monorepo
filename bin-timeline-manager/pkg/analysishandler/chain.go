package analysishandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	amanalysis "monorepo/bin-ai-manager/models/analysis"

	"monorepo/bin-timeline-manager/models/verdict"
)

// runChain executes the full analysis pipeline and returns the persisted verdict
// JSON plus the model used for the diagnostic stage. Any error fails the chain.
func (h *analysisHandler) runChain(ctx context.Context, customerID, activeflowID uuid.UUID) ([]byte, string, error) {
	// (1)-(4) input collection, reduction, freeze+index, pre-extraction.
	input, err := h.collectInput(ctx, customerID, activeflowID)
	if err != nil {
		return nil, "", fmt.Errorf("runChain: could not collect input. err: %v", err)
	}

	// (5) adaptive staging decision.
	var rawVerdict json.RawMessage
	var modelUsed string
	var interactions []verdict.Interaction
	staged := !h.useSingleCall(input)
	if staged {
		rawVerdict, modelUsed, interactions, err = h.runStaged(ctx, input)
	} else {
		rawVerdict, modelUsed, err = h.runCombined(ctx, input)
	}
	if err != nil {
		return nil, "", err
	}

	// (6) raw-output validation (enum / evidence / index-range) on the LLM JSON.
	raw, err := verdict.ValidateRaw(rawVerdict, len(input.events))
	if err != nil {
		return nil, "", fmt.Errorf("runChain: raw verdict validation failed. err: %v", err)
	}

	// (7) resolution + (c) deterministic overwrite of resources_used.
	// Interactions source is selected by PATH (structural), not by len()/nil:
	// the staged path carries the Stage 2 content forward; the single-call path
	// reads them from the combined emission (raw.Interactions).
	if !staged {
		interactions = raw.Interactions
	}
	final := h.buildFinalVerdict(raw, interactions, input)

	out, err := json.Marshal(final)
	if err != nil {
		return nil, "", fmt.Errorf("runChain: could not marshal final verdict. err: %v", err)
	}
	return out, modelUsed, nil
}

// useSingleCall decides single combined call vs full 3-stage chain (review #6).
func (h *analysisHandler) useSingleCall(input *collectedInput) bool {
	if len(input.events) >= analysisStageThresholdEvents {
		return false
	}
	total := 0
	for _, t := range input.transcripts {
		total += len([]rune(t))
	}
	return total < analysisShortTranscriptRunes
}

// runCombined runs a single gateway call producing the full verdict directly.
// Uses the stage3 (best/diagnostic) model so it is covered by the allow-set.
func (h *analysisHandler) runCombined(ctx context.Context, input *collectedInput) (json.RawMessage, string, error) {
	prompt := combinedPrompt
	data := h.buildCombinedData(input)

	resp, err := h.callGateway(ctx, prompt, data, verdictSchema, "timeline_verdict", h.models.Stage3)
	if err != nil {
		return nil, "", fmt.Errorf("runCombined: %v", err)
	}
	return resp.Result, resp.Model, nil
}

// runStaged runs the full 3-stage chain. Each stage consumes the prior stage's
// STRUCTURED output (not raw events again). It also returns the Stage 2 content
// interactions, carried forward into the final verdict (VOIP-1200): Stage 3 runs
// on stage3VerdictSchema (no interactions) so it cannot paraphrase/lose them.
func (h *analysisHandler) runStaged(ctx context.Context, input *collectedInput) (json.RawMessage, string, []verdict.Interaction, error) {
	// Stage 1 — Inventory.
	stage1Data := h.buildStage1Data(input)
	resp1, err := h.callGateway(ctx, stage1Prompt, stage1Data, stage1Schema, "timeline_stage1", h.models.Stage1)
	if err != nil {
		return nil, "", nil, fmt.Errorf("runStaged: stage1: %v", err)
	}

	// Stage 2 — Content.
	stage2Data := h.buildStage2Data(resp1.Result, input)
	resp2, err := h.callGateway(ctx, stage2Prompt, stage2Data, stage2Schema, "timeline_stage2", h.models.Stage2)
	if err != nil {
		return nil, "", nil, fmt.Errorf("runStaged: stage2: %v", err)
	}

	// Parse the Stage 2 content. Its schema is the OBJECT
	// {interactions:[{resource_type,summary}], overall_narrative}; unmarshaling
	// into a bare slice would fail. We keep the interactions; overall_narrative
	// is consumed by the Stage 3 prompt, not persisted here.
	var s2 stage2Result
	if err := json.Unmarshal(resp2.Result, &s2); err != nil {
		return nil, "", nil, fmt.Errorf("runStaged: could not parse stage2 content. err: %v", err)
	}

	// Stage 3 — Diagnosis (produces the final verdict, WITHOUT interactions).
	stage3Data := h.buildStage3Data(resp1.Result, resp2.Result, input)
	resp3, err := h.callGateway(ctx, stage3Prompt, stage3Data, stage3VerdictSchema, "timeline_verdict", h.models.Stage3)
	if err != nil {
		return nil, "", nil, fmt.Errorf("runStaged: stage3: %v", err)
	}

	return resp3.Result, resp3.Model, s2.Interactions, nil
}

// stage2Result is the parse target for the Stage 2 content pass. Its schema is
// an object; only the interactions are carried forward (overall_narrative feeds
// the Stage 3 prompt and is not persisted).
type stage2Result struct {
	Interactions     []verdict.Interaction `json:"interactions"`
	OverallNarrative string                `json:"overall_narrative"`
}

// callGateway invokes the ai-manager generic analysis gateway and guards against
// output truncation (length finish_reason makes the JSON unreliable — §6.5).
func (h *analysisHandler) callGateway(ctx context.Context, prompt string, data, schema json.RawMessage, schemaName, model string) (*amanalysis.Response, error) {
	req := &amanalysis.Request{
		Prompt:     prompt,
		Data:       data,
		Schema:     schema,
		SchemaName: schemaName,
		Model:      model,
	}

	resp, err := h.reqHandler.AIV1ServiceTypeAnalysisRun(ctx, req, analysisGatewayTimeoutMS)
	if err != nil {
		return nil, fmt.Errorf("gateway call failed. err: %v", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("gateway returned nil response")
	}
	if resp.Truncated || resp.FinishReason == "length" {
		return nil, fmt.Errorf("gateway output truncated (finish_reason=%s); JSON unreliable", resp.FinishReason)
	}
	if len(resp.Result) == 0 {
		return nil, fmt.Errorf("gateway returned empty result")
	}
	return resp, nil
}

// buildFinalVerdict resolves evidence indices to concrete tuples and overwrites
// resources_used with the Go-computed inventory (authoritative — review M1). The
// interactions argument is the content summary selected by the caller per path
// (Stage 2 carry on the staged path, combined emission on the single-call path);
// it is normalized to a non-nil slice so every v2 record serializes
// "interactions":[] rather than null (VOIP-1200, uniform wire shape).
func (h *analysisHandler) buildFinalVerdict(raw *verdict.RawVerdict, interactions []verdict.Interaction, input *collectedInput) *verdict.Verdict {
	issues := make([]verdict.Issue, 0, len(raw.Issues))
	for _, ri := range raw.Issues {
		evidence := make([]verdict.Evidence, 0, len(ri.EvidenceIndex))
		for _, idx := range ri.EvidenceIndex {
			// idx already range-checked in ValidateRaw.
			ce := input.events[idx]
			evidence = append(evidence, verdict.Evidence{
				EvidenceIndex: idx,
				EventType:     ce.EventType,
				Timestamp:     ce.Timestamp,
				ResourceID:    ce.ResourceID,
			})
		}
		issues = append(issues, verdict.Issue{
			Severity: ri.Severity,
			Area:     ri.Area,
			Summary:  ri.Summary,
			Evidence: evidence,
		})
	}

	resources := make([]verdict.ResourceUsed, 0, len(input.inventory))
	for _, rc := range input.inventory {
		resources = append(resources, verdict.ResourceUsed{Type: rc.Type, Count: rc.Count})
	}

	if interactions == nil {
		interactions = []verdict.Interaction{}
	}

	return &verdict.Verdict{
		Version:       verdict.CurrentVersion,
		OverallStatus: raw.OverallStatus,
		InputReduced:  input.inputReduced,
		ResourcesUsed: resources,
		Interactions:  interactions,
		Narrative:     raw.Narrative,
		Issues:        issues,
	}
}

// --- prompt data builders ---

// indexedEventLines renders the frozen canonical list as the LLM-facing outline.
func indexedEventLines(events []*canonicalEvent) string {
	var b strings.Builder
	for _, ce := range events {
		fmt.Fprintf(&b, "%d\t%s\t%s\t%s\t%s\n", ce.Index, ce.Timestamp, ce.EventType, ce.Publisher, ce.Summary)
	}
	return b.String()
}

func errorSignalLines(events []*canonicalEvent) string {
	var b strings.Builder
	for _, ce := range events {
		fmt.Fprintf(&b, "%d\t%s\t%s\t%s\n", ce.Index, ce.Timestamp, ce.EventType, ce.Summary)
	}
	return b.String()
}

func inventoryLines(inv []resourceCount) string {
	var b strings.Builder
	for _, rc := range inv {
		fmt.Fprintf(&b, "%s\t%d\n", rc.Type, rc.Count)
	}
	return b.String()
}

func (h *analysisHandler) buildCombinedData(input *collectedInput) json.RawMessage {
	payload := map[string]any{
		"inventory":     inventoryLines(input.inventory),
		"events":        indexedEventLines(input.events),
		"error_signals": errorSignalLines(input.errorSignals),
		"transcripts":   input.transcripts,
		"event_count":   len(input.events),
	}
	b, _ := json.Marshal(payload)
	return b
}

func (h *analysisHandler) buildStage1Data(input *collectedInput) json.RawMessage {
	payload := map[string]any{
		"inventory":   inventoryLines(input.inventory),
		"events":      indexedEventLines(input.events),
		"event_count": len(input.events),
	}
	b, _ := json.Marshal(payload)
	return b
}

func (h *analysisHandler) buildStage2Data(stage1 json.RawMessage, input *collectedInput) json.RawMessage {
	payload := map[string]any{
		"stage1":      stage1,
		"transcripts": input.transcripts,
	}
	b, _ := json.Marshal(payload)
	return b
}

func (h *analysisHandler) buildStage3Data(stage1, stage2 json.RawMessage, input *collectedInput) json.RawMessage {
	payload := map[string]any{
		"stage1":        stage1,
		"stage2":        stage2,
		"error_signals": errorSignalLines(input.errorSignals),
		"event_count":   len(input.events),
	}
	b, _ := json.Marshal(payload)
	return b
}
