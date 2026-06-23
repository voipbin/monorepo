package analysishandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"monorepo/bin-timeline-manager/models/event"
)

// reduceEvents applies deterministic reduction when the assembled list exceeds
// the reduce target. Priority (review H2-input):
//  1. ALWAYS keep error-class events (their frozen index must be stable).
//  2. keep a bounded number of non-error events, dropping low-signal repeats.
//
// Returns (reduced, didReduce). Ordering is preserved so indexing stays
// deterministic across re-analysis.
func reduceEvents(events []*canonicalEvent) ([]*canonicalEvent, bool) {
	if approxSize(events) <= analysisReduceTargetBytes {
		return events, false
	}

	// keep all error events; sample non-error events until we approach target.
	kept := make([]*canonicalEvent, 0, len(events))
	var size int
	for _, ce := range events {
		ceSize := len(ce.Summary) + len(ce.EventType) + len(ce.ResourceID) + 64
		if ce.isError {
			kept = append(kept, ce)
			size += ceSize
			continue
		}
		if size+ceSize > analysisReduceTargetBytes {
			// over target: drop this low-signal non-error event.
			continue
		}
		kept = append(kept, ce)
		size += ceSize
	}

	return kept, len(kept) < len(events)
}

func approxSize(events []*canonicalEvent) int {
	var n int
	for _, ce := range events {
		n += len(ce.Summary) + len(ce.EventType) + len(ce.ResourceID) + 64
	}
	return n
}

// summarizeData produces a compact, LLM-facing summary of an event payload.
// It does NOT dump the full raw JSON (token control); it extracts a few
// high-signal keys when present and otherwise truncates.
func summarizeData(e *event.Event) string {
	if len(e.Data) == 0 {
		return ""
	}

	var m map[string]any
	if err := json.Unmarshal(e.Data, &m); err != nil {
		// not an object; return a truncated raw string.
		return truncateRunes(string(e.Data), 240)
	}

	// pull a few high-signal fields when present.
	keys := []string{"status", "reason", "hangup_reason", "direction", "type", "message", "error"}
	parts := []string{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	if len(parts) == 0 {
		return truncateRunes(string(e.Data), 240)
	}
	return truncateRunes(strings.Join(parts, " "), 240)
}

// resourceIDOf best-effort extracts a resource id from the event payload.
func resourceIDOf(e *event.Event) string {
	if len(e.Data) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(e.Data, &m); err != nil {
		return ""
	}
	if v, ok := m["id"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

// collectTranscripts fetches transcript text for the activeflow's transcribe
// sessions (best-effort; missing content is non-fatal). Each returned string is
// one resource's joined transcript, capped per resource.
func (h *analysisHandler) collectTranscripts(ctx context.Context, customerID, activeflowID uuid.UUID) []string {
	log := logrus.WithField("func", "collectTranscripts")

	if activeflowID == uuid.Nil {
		return nil
	}

	transcribes, err := h.reqHandler.TranscribeV1TranscribeList(ctx, "", 100, map[tmtranscribe.Field]any{
		tmtranscribe.FieldCustomerID:   customerID,
		tmtranscribe.FieldActiveflowID: activeflowID,
		tmtranscribe.FieldDeleted:      false,
	})
	if err != nil {
		log.Debugf("could not list transcribes (non-fatal). err: %v", err)
		return nil
	}

	out := []string{}
	for _, tr := range transcribes {
		texts, err := h.reqHandler.TranscribeV1TranscriptList(ctx, "", 1000, map[tmtranscript.Field]any{
			tmtranscript.FieldTranscribeID: tr.ID,
			tmtranscript.FieldDeleted:      false,
		})
		if err != nil {
			log.Debugf("could not list transcripts for transcribe %s (non-fatal). err: %v", tr.ID, err)
			continue
		}
		joined := joinTranscript(texts)
		if joined != "" {
			out = append(out, truncateRunes(joined, analysisMaxTranscriptRunesPerResource))
		}
	}
	return out
}

func joinTranscript(texts []tmtranscript.Transcript) string {
	parts := make([]string, 0, len(texts))
	for _, t := range texts {
		if t.Message == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("[%s] %s", t.Direction, t.Message))
	}
	return strings.Join(parts, "\n")
}
