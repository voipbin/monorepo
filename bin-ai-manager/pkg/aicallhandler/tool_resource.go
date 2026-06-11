package aicallhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
)

// msgResourceNotFound is the single masked response for every path where the
// caller is not allowed to see the resource: absent, cross-customer, or any
// state in between. All such paths MUST return this byte-identical string so
// the tool is not an existence oracle.
const msgResourceNotFound = "Resource not found."

// maxResourceSummaryRunes caps the WHOLE rendered message (metadata + body +
// truncation marker) of every resource summary, in runes. Raising it requires
// a code change.
const maxResourceSummaryRunes = 4000

// resourceListPageSize is the page size used for enrichment list fetches
// (transcripts, aicall messages). One extra row is requested to detect that
// more rows exist beyond the rendered page.
const resourceListPageSize = 100

// ErrResourceNotAccessible signals that the resource is absent or not owned by
// the caller. The caller masks it behind msgResourceNotFound. Transient
// failures are returned as wrapped ordinary errors and must NOT be masked
// (existence is unknown; masking would make the LLM assert a false
// "not found").
var ErrResourceNotAccessible = stderrors.New("resource not accessible")

// resourceFetchResult carries the owner of a fetched resource and a render
// closure. ownerID is valid only when err == nil. render is invoked by the
// caller ONLY after ownership has been validated; any enrichment fetch
// (e.g. transcript list, aicall message list) happens inside render, never
// before.
type resourceFetchResult struct {
	ownerID uuid.UUID
	render  func(ctx context.Context) string
}

// resourceFetcher fetches the primary resource for a given id and returns its
// owner plus a deferred renderer.
type resourceFetcher func(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error)

// mapResourceFetchers is the source of truth for the supported resource
// types. The tool definition's JSON-schema enum is asserted equal to the
// sorted keys of this map by a unit test so the two cannot drift.
var mapResourceFetchers = map[string]resourceFetcher{
	"call":           fetchResourceCall,
	"groupcall":      fetchResourceGroupcall,
	"recording":      fetchResourceRecording,
	"transcribe":     fetchResourceTranscribe,
	"summary":        fetchResourceSummary,
	"aicall":         fetchResourceAIcall,
	"conferencecall": fetchResourceConferencecall,
	"queuecall":      fetchResourceQueuecall,
}

// SupportedResourceTypes returns the sorted list of resource types supported
// by the get_resource tool. Exported so the toolhandler definition test can
// assert the JSON-schema enum matches.
func SupportedResourceTypes() []string {
	res := make([]string, 0, len(mapResourceFetchers))
	for k := range mapResourceFetchers {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

// supportedResourceTypes is the human-readable list used in error messages.
var supportedResourceTypes = strings.Join(SupportedResourceTypes(), ", ")

// toolHandleGetResource retrieves the content of a single VoIPBin resource by
// id and returns a curated, LLM-readable summary.
//
// Ownership validation and existence-oracle masking are delegated to
// resolveResource. Every "cannot see this" path collapses to the single
// msgResourceNotFound emission site below.
//
// CRITICAL: both branches inside the `if err` block MUST return. Falling
// through would (a) for a cross-customer error expose a foreign resource
// summary (IDOR), and (b) for the transient case render an invalid summary.
// The returns are load-bearing.
func (h *aicallHandler) toolHandleGetResource(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleGetResource",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool get_resource.")

	res := newToolResult(tc.ID)

	var args struct {
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		// Pass the unmarshal error through (sibling precedent:
		// toolHandleSearchKnowledge) so the LLM can self-correct its arguments.
		fillFailed(res, errors.Wrap(err, "invalid arguments"))
		return res
	}

	if args.ResourceType == "" {
		fillFailed(res, fmt.Errorf("resource_type is required. supported: %s", supportedResourceTypes))
		return res
	}
	fetcher, ok := mapResourceFetchers[args.ResourceType]
	if !ok {
		fillFailed(res, fmt.Errorf("unsupported resource_type: %s. supported: %s", args.ResourceType, supportedResourceTypes))
		return res
	}

	resourceID, err := uuid.FromString(args.ResourceID)
	if err != nil || resourceID == uuid.Nil {
		fillFailed(res, fmt.Errorf("invalid resource_id"))
		return res
	}

	summary, err := h.resolveResource(ctx, c.CustomerID, fetcher, resourceID)
	if err != nil {
		if stderrors.Is(err, ErrResourceNotAccessible) {
			// Single masking site for ALL not-accessible paths.
			// CRITICAL: this return is load-bearing (IDOR if fallen through).
			fillSuccess(res, args.ResourceType, resourceID.String(), msgResourceNotFound)
			return res
		}
		// Transient/infra failure: existence unknown, report honest tool
		// failure. The cause is logged for operators but never reaches the LLM.
		// CRITICAL: this return is load-bearing.
		log.Errorf("Resource lookup failed. err: %v", err)
		fillFailed(res, fmt.Errorf("resource lookup failed"))
		return res
	}

	fillSuccess(res, args.ResourceType, resourceID.String(), summary)
	return res
}

// resolveResource fetches the primary resource, validates ownership, and only
// then renders the summary (running any enrichment fetch). Mirrors the
// resolveCorrelation two-tier error contract: ErrResourceNotAccessible for
// absent/cross-customer (masked by the caller), wrapped ordinary errors for
// transient failures (honest tool failure).
func (h *aicallHandler) resolveResource(ctx context.Context, callerCustomerID uuid.UUID, fetcher resourceFetcher, resourceID uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "resolveResource",
		"customer_id": callerCustomerID,
		"resource_id": resourceID,
	})

	r, err := fetcher(ctx, h, resourceID)
	if err != nil {
		if stderrors.Is(err, requesthandler.ErrNotFound) {
			// Absent. Mask.
			return "", ErrResourceNotAccessible
		}
		// Transient: honest failure, no masking.
		return "", errors.Wrap(err, "could not fetch resource")
	}

	if r.ownerID != callerCustomerID || r.ownerID == uuid.Nil {
		// ownerID == uuid.Nil is treated as not-accessible (fail closed): a
		// row with unset customer_id must never match any caller.
		log.Warnf("Cross-customer resource access blocked. resource_owner: %s", r.ownerID)
		return "", ErrResourceNotAccessible
	}

	// Ownership validated: only now render (and run any enrichment fetch).
	return r.render(ctx), nil
}

// ---- rendering helpers ----

// summaryBuilder accumulates labeled metadata lines for a resource summary.
type summaryBuilder struct {
	lines []string
}

func (b *summaryBuilder) add(label, value string) {
	if value == "" {
		return
	}
	b.lines = append(b.lines, fmt.Sprintf("%s: %s", label, value))
}

func (b *summaryBuilder) addUUID(label string, id uuid.UUID) {
	if id == uuid.Nil {
		return
	}
	b.add(label, id.String())
}

func (b *summaryBuilder) addTime(label string, t *time.Time) {
	if t == nil || t.IsZero() {
		return
	}
	b.add(label, t.UTC().Format(time.RFC3339))
}

func (b *summaryBuilder) String() string {
	return strings.Join(b.lines, "\n")
}

// capSummaryRunes enforces the whole-message rune cap on a rendered summary.
// Free-text overflow is hard-truncated mid-text with a generic marker.
func capSummaryRunes(s string) string {
	runes := []rune(s)
	if len(runes) <= maxResourceSummaryRunes {
		return s
	}
	marker := "\n...(truncated)"
	keep := maxResourceSummaryRunes - len([]rune(marker))
	if keep < 0 {
		keep = 0
	}
	return string(runes[:keep]) + marker
}

// renderBodyLines appends a line-structured body (transcript lines, aicall
// message lines) to a metadata header under the whole-message rune cap.
// lines must be in chronological order. Truncation keeps the MOST RECENT
// lines and drops the oldest; the marker is placed at the top of the block
// (where the omitted earlier lines were). pagedOut indicates the source list
// had more rows than the fetched page (page cap), which also forces the
// marker. noun is "transcripts" or "messages".
func renderBodyLines(header string, lines []string, pagedOut bool, noun string) string {
	budget := maxResourceSummaryRunes - len([]rune(header)) - 1 // newline joining header/body

	// Walk backwards keeping the newest lines that fit, reserving marker room.
	marker := fmt.Sprintf("...(earlier %s omitted; showing the most recent %%d)", noun)
	markerRoom := len([]rune(fmt.Sprintf(marker, len(lines)))) + 1

	kept := make([]string, 0, len(lines))
	used := 0
	for i := len(lines) - 1; i >= 0; i-- {
		lineLen := len([]rune(lines[i])) + 1
		if used+lineLen > budget-markerRoom {
			break
		}
		kept = append([]string{lines[i]}, kept...)
		used += lineLen
	}

	truncated := pagedOut || len(kept) < len(lines)

	if len(kept) == 0 && len(lines) > 0 {
		// Degenerate case: even the newest single line cannot fit. Hard-cut
		// it mid-text and still show it.
		newest := []rune(lines[len(lines)-1])
		room := budget - markerRoom
		if room < 0 {
			room = 0
		}
		if len(newest) > room {
			newest = newest[:room]
		}
		degenerate := fmt.Sprintf("...(earlier %s omitted; showing the most recent 1, truncated)", noun)
		return header + "\n" + degenerate + "\n" + string(newest)
	}

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n")
	if truncated {
		fmt.Fprintf(&sb, marker+"\n", len(kept))
	}
	sb.WriteString(strings.Join(kept, "\n"))
	return sb.String()
}

// ---- per-type fetchers ----

func fetchResourceCall(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error) {
	r, err := h.reqHandler.CallV1CallGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &resourceFetchResult{
		ownerID: r.CustomerID,
		render: func(ctx context.Context) string {
			return renderCall(r)
		},
	}, nil
}

func renderCall(r *cmcall.Call) string {
	b := &summaryBuilder{}
	b.add("status", string(r.Status))
	b.add("direction", string(r.Direction))
	b.add("source", r.Source.Target)
	b.add("destination", r.Destination.Target)
	b.addTime("created", r.TMCreate)
	b.addTime("ringing", r.TMRinging)
	b.addTime("progressing", r.TMProgressing)
	b.addTime("hangup", r.TMHangup)
	b.add("hangup_by", string(r.HangupBy))
	b.add("hangup_reason", string(r.HangupReason))
	if len(r.RecordingIDs) > 0 {
		b.add("recordings", fmt.Sprintf("%d", len(r.RecordingIDs)))
	}
	b.addUUID("groupcall_id", r.GroupcallID)
	return capSummaryRunes(b.String())
}

func fetchResourceGroupcall(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error) {
	r, err := h.reqHandler.CallV1GroupcallGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &resourceFetchResult{
		ownerID: r.CustomerID,
		render: func(ctx context.Context) string {
			return renderGroupcall(r)
		},
	}, nil
}

func renderGroupcall(r *cmgroupcall.Groupcall) string {
	b := &summaryBuilder{}
	b.add("status", string(r.Status))
	b.add("ring_method", string(r.RingMethod))
	b.add("answer_method", string(r.AnswerMethod))
	if r.Source != nil {
		b.add("source", r.Source.Target)
	}
	b.add("destinations", fmt.Sprintf("%d", len(r.Destinations)))
	b.addUUID("answered_call_id", r.AnswerCallID)
	if len(r.CallIDs) > 0 {
		ids := make([]string, len(r.CallIDs))
		for i, cid := range r.CallIDs {
			ids[i] = cid.String()
		}
		b.add("call_ids", strings.Join(ids, ", "))
	}
	return capSummaryRunes(b.String())
}

func fetchResourceRecording(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error) {
	r, err := h.reqHandler.CallV1RecordingGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &resourceFetchResult{
		ownerID: r.CustomerID,
		render: func(ctx context.Context) string {
			return renderRecording(r)
		},
	}, nil
}

func renderRecording(r *cmrecording.Recording) string {
	b := &summaryBuilder{}
	b.add("status", string(r.Status))
	b.add("format", string(r.Format))
	b.add("reference_type", string(r.ReferenceType))
	b.addUUID("reference_id", r.ReferenceID)
	b.add("recording_name", r.RecordingName)
	b.addTime("started", r.TMStart)
	b.addTime("ended", r.TMEnd)
	return capSummaryRunes(b.String())
}

func fetchResourceTranscribe(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error) {
	r, err := h.reqHandler.TranscribeV1TranscribeGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &resourceFetchResult{
		ownerID: r.CustomerID,
		render: func(ctx context.Context) string {
			return h.renderTranscribe(ctx, r)
		},
	}, nil
}

// renderTranscribe renders the transcribe metadata followed by the transcript
// body. It runs only after ownership has been validated; the transcript fetch
// never happens for a foreign transcribe.
func (h *aicallHandler) renderTranscribe(ctx context.Context, r *tmtranscribe.Transcribe) string {
	b := &summaryBuilder{}
	b.add("status", string(r.Status))
	b.add("language", r.Language)
	b.add("direction", string(r.Direction))
	b.add("reference_type", string(r.ReferenceType))
	b.addUUID("reference_id", r.ReferenceID)
	b.add("provider", string(r.Provider))
	header := b.String()

	// page_size+1 to detect that more rows exist beyond the rendered page.
	transcripts, err := h.reqHandler.TranscribeV1TranscriptList(ctx, "", resourceListPageSize+1, map[tmtranscript.Field]any{
		tmtranscript.FieldTranscribeID: r.ID,
		tmtranscript.FieldDeleted:      false,
	})
	if err != nil {
		logrus.WithField("func", "renderTranscribe").Errorf("Could not list transcripts. transcribe_id: %s, err: %v", r.ID, err)
		return capSummaryRunes(header + "\n(transcripts unavailable)")
	}

	pagedOut := len(transcripts) > resourceListPageSize
	if pagedOut {
		transcripts = transcripts[:resourceListPageSize]
	}
	if len(transcripts) == 0 {
		return capSummaryRunes(header + "\n(no transcripts)")
	}

	// The dbhandler returns tm_create DESC (most recent first); reverse to
	// chronological order for rendering.
	lines := make([]string, 0, len(transcripts))
	for i := len(transcripts) - 1; i >= 0; i-- {
		lines = append(lines, renderTranscriptLine(&transcripts[i]))
	}

	return renderBodyLines(header, lines, pagedOut, "transcripts")
}

// renderTranscriptLine renders one transcript as "[in 00:00:03] hello".
// TMTranscript encodes an offset from the zero time (0001-01-01 00:00:00 =
// transcribe start).
func renderTranscriptLine(t *tmtranscript.Transcript) string {
	offset := "--:--:--"
	if t.TMTranscript != nil && !t.TMTranscript.IsZero() {
		d := t.TMTranscript.UTC().Sub(time.Time{})
		offset = fmt.Sprintf("%02d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
	}
	return fmt.Sprintf("[%s %s] %s", t.Direction, offset, t.Message)
}

func fetchResourceSummary(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error) {
	r, err := h.reqHandler.AIV1SummaryGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &resourceFetchResult{
		ownerID: r.CustomerID,
		render: func(ctx context.Context) string {
			b := &summaryBuilder{}
			b.add("status", string(r.Status))
			b.add("language", r.Language)
			b.add("reference_type", string(r.ReferenceType))
			b.addUUID("reference_id", r.ReferenceID)
			b.add("content", r.Content)
			return capSummaryRunes(b.String())
		},
	}, nil
}

func fetchResourceAIcall(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error) {
	r, err := h.reqHandler.AIV1AIcallGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &resourceFetchResult{
		ownerID: r.CustomerID,
		render: func(ctx context.Context) string {
			return h.renderAIcall(ctx, r)
		},
	}, nil
}

// renderAIcall renders the aicall metadata followed by the curated
// conversation history. It runs only after ownership has been validated; the
// message fetch never happens for a foreign aicall.
func (h *aicallHandler) renderAIcall(ctx context.Context, r *aicall.AIcall) string {
	b := &summaryBuilder{}
	b.add("status", string(r.Status))
	b.add("reference_type", string(r.ReferenceType))
	b.addUUID("reference_id", r.ReferenceID)
	b.add("engine_model", string(r.AIEngineModel))
	b.add("tts_type", string(r.AITTSType))
	b.add("stt_type", string(r.AISTTType))
	b.addTime("created", r.TMCreate)
	b.addTime("ended", r.TMEnd)
	header := b.String()

	// page_size+1 to detect that more rows exist beyond the rendered page.
	messages, err := h.messageHandler.List(ctx, resourceListPageSize+1, "", map[message.Field]any{
		message.FieldAIcallID: r.ID,
		message.FieldDeleted:  false,
	})
	if err != nil {
		logrus.WithField("func", "renderAIcall").Errorf("Could not list aicall messages. aicall_id: %s, err: %v", r.ID, err)
		return capSummaryRunes(header + "\n(messages unavailable)")
	}

	pagedOut := len(messages) > resourceListPageSize
	if pagedOut {
		messages = messages[:resourceListPageSize]
	}

	// The dbhandler returns tm_create DESC (most recent first); reverse to
	// chronological order, then render via the role allowlist.
	lines := make([]string, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		lines = append(lines, renderAIcallMessageLines(messages[i])...)
	}
	if len(lines) == 0 {
		return capSummaryRunes(header + "\n(no messages)")
	}

	return renderBodyLines(header, lines, pagedOut, "messages")
}

// renderAIcallMessageLines renders one message into zero or more lines via an
// ALLOWLIST with explicit precedence. Code fact: ToolHandle persists assistant
// tool-call rows with Content == "", so the ToolCalls check must run before
// the empty-content drop. Everything not explicitly allowed (system prompt
// snapshots, notification, tool, function, empty role, empty content without
// tool calls) is dropped. The Direction field is deliberately unused; role
// labels carry the speaker.
func renderAIcallMessageLines(m *message.Message) []string {
	if m == nil {
		return nil
	}

	switch m.Role {
	case message.RoleAssistant:
		var lines []string
		if m.Content != "" {
			lines = append(lines, fmt.Sprintf("[assistant] %s", m.Content))
		}
		for _, tcall := range m.ToolCalls {
			lines = append(lines, fmt.Sprintf("[assistant called %s]", tcall.Function.Name))
		}
		return lines

	case message.RoleUser:
		if m.Content == "" {
			return nil
		}
		return []string{fmt.Sprintf("[user] %s", m.Content)}

	default:
		return nil
	}
}

func fetchResourceConferencecall(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error) {
	r, err := h.reqHandler.ConferenceV1ConferencecallGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &resourceFetchResult{
		ownerID: r.CustomerID,
		render: func(ctx context.Context) string {
			return renderConferencecall(r)
		},
	}, nil
}

func renderConferencecall(r *cfconferencecall.Conferencecall) string {
	b := &summaryBuilder{}
	b.add("status", string(r.Status))
	b.addUUID("conference_id", r.ConferenceID)
	b.add("reference_type", string(r.ReferenceType))
	b.addUUID("reference_id", r.ReferenceID)
	b.addTime("created", r.TMCreate)
	b.addTime("updated", r.TMUpdate)
	return capSummaryRunes(b.String())
}

func fetchResourceQueuecall(ctx context.Context, h *aicallHandler, id uuid.UUID) (*resourceFetchResult, error) {
	r, err := h.reqHandler.QueueV1QueuecallGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return &resourceFetchResult{
		ownerID: r.CustomerID,
		render: func(ctx context.Context) string {
			return renderQueuecall(r)
		},
	}, nil
}

func renderQueuecall(r *qmqueuecall.Queuecall) string {
	b := &summaryBuilder{}
	b.add("status", string(r.Status))
	b.addUUID("queue_id", r.QueueID)
	b.add("routing_method", string(r.RoutingMethod))
	if r.DurationWaiting > 0 {
		b.add("duration_waiting_ms", fmt.Sprintf("%d", r.DurationWaiting))
	}
	if r.DurationService > 0 {
		b.add("duration_service_ms", fmt.Sprintf("%d", r.DurationService))
	}
	b.addUUID("service_agent_id", r.ServiceAgentID)
	b.addTime("created", r.TMCreate)
	b.addTime("serviced", r.TMService)
	b.addTime("ended", r.TMEnd)
	return capSummaryRunes(b.String())
}
