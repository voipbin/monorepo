package analysishandler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/gofrs/uuid"

	"monorepo/bin-timeline-manager/models/event"
)

// canonicalEvent is one entry of the frozen canonical event list. Its index in
// the slice is its evidence_index and is assigned ONCE, after reduction. Nothing
// renumbers it (review C1).
type canonicalEvent struct {
	Index      int
	Timestamp  string
	EventType  string
	Publisher  string
	ResourceID string
	// summary is a compact, LLM-facing description of the event payload.
	Summary string
	// isError marks deterministic error-class events that must survive reduction.
	isError bool
	// raw is kept for resolution / inventory; not all of it is sent to the LLM.
	raw *event.Event
}

// collectedInput is everything the chain assembles before talking to the LLM.
type collectedInput struct {
	events       []*canonicalEvent
	inventory    []resourceCount
	errorSignals []*canonicalEvent
	transcripts  []string
	inputReduced bool
}

type resourceCount struct {
	Type  string
	Count int
}

// collectInput runs steps (1) collect -> (2) reduce -> (3) freeze+index ->
// (4) deterministic pre-extraction. The returned canonical list is frozen: its
// indices are final.
func (h *analysisHandler) collectInput(ctx context.Context, customerID, activeflowID uuid.UUID) (*collectedInput, error) {
	// (1) collect all aggregated events, paginated and bounded.
	raw, reducedByCap, err := h.collectEvents(ctx, activeflowID)
	if err != nil {
		return nil, fmt.Errorf("collectInput: could not collect events. err: %v", err)
	}

	// (2) reduce: drop low-signal repetitive non-error events if over target.
	reduced, reducedBySize := reduceEvents(raw)
	inputReduced := reducedByCap || reducedBySize

	// (3) freeze + index the FINAL list.
	frozen := make([]*canonicalEvent, len(reduced))
	for i, ce := range reduced {
		ce.Index = i
		frozen[i] = ce
	}

	// (4) deterministic pre-extraction: inventory + error signals.
	inventory := buildInventory(frozen)
	errorSignals := []*canonicalEvent{}
	for _, ce := range frozen {
		if ce.isError {
			errorSignals = append(errorSignals, ce)
		}
	}

	// content (best-effort): transcripts for the customer's resources.
	transcripts := h.collectTranscripts(ctx, customerID, activeflowID)

	return &collectedInput{
		events:       frozen,
		inventory:    inventory,
		errorSignals: errorSignals,
		transcripts:  transcripts,
		inputReduced: inputReduced,
	}, nil
}

// collectEvents loops AggregatedList pages, bounded by analysisMaxEvents /
// analysisMaxPages, de-duplicating same-timestamp boundary rows by a stable
// composite key (review M2). Returns (events, reducedByCap).
func (h *analysisHandler) collectEvents(ctx context.Context, activeflowID uuid.UUID) ([]*canonicalEvent, bool, error) {
	seen := map[string]struct{}{}
	out := []*canonicalEvent{}
	pageToken := ""
	reducedByCap := false

	for page := 0; page < analysisMaxPages; page++ {
		resp, err := h.eventHandler.AggregatedList(ctx, activeflowID, pageToken, analysisEventPageSize)
		if err != nil {
			return nil, false, fmt.Errorf("collectEvents: could not list events. err: %v", err)
		}
		if resp == nil || len(resp.Result) == 0 {
			break
		}

		for _, e := range resp.Result {
			key := dedupKey(e)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, toCanonical(e))

			if len(out) >= analysisMaxEvents {
				reducedByCap = true
				return out, reducedByCap, nil
			}
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return out, reducedByCap, nil
}

// dedupKey is a stable composite key for a boundary row (timestamp,event_type,
// resource_id,hash(data)).
func dedupKey(e *event.Event) string {
	sum := sha256.Sum256(e.Data)
	return strings.Join([]string{
		e.Timestamp.Format("2006-01-02T15:04:05.000000Z07:00"),
		e.EventType,
		resourceIDOf(e),
		hex.EncodeToString(sum[:8]),
	}, "|")
}

func toCanonical(e *event.Event) *canonicalEvent {
	return &canonicalEvent{
		Timestamp:  e.Timestamp.Format("2006-01-02T15:04:05.000000Z07:00"),
		EventType:  e.EventType,
		Publisher:  string(e.Publisher),
		ResourceID: resourceIDOf(e),
		Summary:    summarizeData(e),
		isError:    isErrorEvent(e.EventType),
		raw:        e,
	}
}

// buildInventory computes per-publisher-type counts (Go-authoritative — review M1).
func buildInventory(events []*canonicalEvent) []resourceCount {
	counts := map[string]int{}
	for _, ce := range events {
		t := ce.Publisher
		if t == "" {
			t = "unknown"
		}
		counts[t]++
	}

	out := make([]resourceCount, 0, len(counts))
	for t, c := range counts {
		out = append(out, resourceCount{Type: t, Count: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

// isErrorEvent flags error-class event types that must survive reduction and be
// surfaced as deterministic signals.
func isErrorEvent(eventType string) bool {
	lt := strings.ToLower(eventType)
	return strings.Contains(lt, "_error") ||
		strings.Contains(lt, "_failed") ||
		strings.Contains(lt, "_hangup") ||
		strings.Contains(lt, "_timeout")
}
