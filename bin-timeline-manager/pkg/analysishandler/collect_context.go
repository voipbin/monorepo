package analysishandler

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"monorepo/bin-timeline-manager/models/verdict"
)

// canonicalTimestampLayout matches the format used by toCanonical (collect.go).
const canonicalTimestampLayout = "2006-01-02T15:04:05.000000Z07:00"

// enrichMaxChaseDepth caps transcribe/recording origin re-chase. The only legal
// re-chase is transcribe->recording->{call,confbridge} (depth 1, terminal); a
// recording never references a transcribe, so no real cycle exists. The guard is
// belt-and-suspenders (design F9/#4).
const enrichMaxChaseDepth = 1

// sessionEnrichment bundles the three Go-authoritative verdict blocks. Any
// field may be nil; a nil block is an omitted key on the wire (never null/zero).
type sessionEnrichment struct {
	ctx     *verdict.SessionContext
	outcome *verdict.SessionOutcome
	metrics *verdict.SessionMetrics
}

// enrich computes the channel-neutral context/outcome/metrics for an analysis,
// dispatching on the activeflow's reference_type. It is best-effort and never
// fatal: a failed resolution leaves a block nil and the analysis still completes
// (design §5 last bullet). The activeflow is THREADED IN from Start (not re-Got).
func (h *analysisHandler) enrich(ctx context.Context, input *collectedInput, customerID uuid.UUID, af *fmactiveflow.Activeflow) *sessionEnrichment {
	sc, outcome, chased := h.enrichRef(ctx, customerID, af.ReferenceType, af.ReferenceID, af.FlowID, 0)
	if sc == nil {
		// nothing resolved (e.g. deleted reference); leave all blocks nil.
		h.metricEnrichmentOutcome(af.ReferenceType, "unresolved")
		return &sessionEnrichment{}
	}

	var metrics *verdict.SessionMetrics
	if !chased {
		// AIHandled/HumanInvolved/Metrics are derived from THIS activeflow's
		// event stream and are only meaningful for a DIRECT reference. For a
		// chased transcribe/recording card the origin lives in a DIFFERENT
		// activeflow whose events were not loaded, so they stay suppressed (F1).
		sc.AIHandled = aiHandled(input.allEvents)
		sc.HumanInvolved = agentLegConnected(input.allEvents)

		// Metrics is VOICE/AI ONLY. conversation/api/none -> nil so a chat does
		// not get a misleading zero-value voice metrics block (#2/#3).
		switch af.ReferenceType {
		case fmactiveflow.ReferenceTypeCall, fmactiveflow.ReferenceTypeAI:
			metrics = aggregateMetrics(input.allEvents)
		}
	}

	h.metricEnrichmentOutcome(af.ReferenceType, "resolved")
	return &sessionEnrichment{ctx: sc, outcome: outcome, metrics: metrics}
}

// enrichRef resolves ONE reference into a SessionContext + Outcome. It does NOT
// set AIHandled/HumanInvolved/Metrics (those are activeflow-stream-derived; see
// enrich). The third return is chased=true ONLY for a chased transcribe/
// recording. Dispatch is TOTAL over the 8 activeflow reference_types PLUS
// confbridge (a transcribe/recording origin type, not an activeflow type).
func (h *analysisHandler) enrichRef(ctx context.Context, customerID uuid.UUID, refType fmactiveflow.ReferenceType, refID, flowID uuid.UUID, depth int) (*verdict.SessionContext, *verdict.SessionOutcome, bool) {
	switch refType {
	case fmactiveflow.ReferenceTypeTranscribe, fmactiveflow.ReferenceTypeRecording:
		// Chase BEFORE computing FlowName: a chased card omits FlowName anyway
		// (origin's flow is a different activeflow, F2/#5), so skip the wasted
		// FlowV1FlowGet on this path.
		return h.chaseOrigin(ctx, customerID, refType, refID, depth)

	case fmactiveflow.ReferenceTypeCall:
		sc := &verdict.SessionContext{ReferenceType: string(refType), Channel: channelOf(refType)}
		sc.FlowName = h.scopedFlowName(ctx, customerID, flowID)
		outcome := h.enrichCall(ctx, customerID, refID, sc)
		return sc, outcome, false

	case fmactiveflow.ReferenceTypeConversation:
		sc := &verdict.SessionContext{ReferenceType: string(refType), Channel: channelOf(refType)}
		sc.FlowName = h.scopedFlowName(ctx, customerID, flowID)
		outcome := h.enrichConversation(ctx, customerID, refID, sc)
		return sc, outcome, false

	case fmactiveflow.ReferenceTypeAI:
		sc := &verdict.SessionContext{ReferenceType: string(refType), Channel: channelOf(refType)}
		sc.FlowName = h.scopedFlowName(ctx, customerID, flowID)
		outcome := h.enrichAIcall(ctx, customerID, refID, sc)
		return sc, outcome, false

	case fmactiveflow.ReferenceTypeCampaign, fmactiveflow.ReferenceTypeAPI, fmactiveflow.ReferenceTypeNone:
		// header-only (campaign is a P1 stub; api/none have no resource).
		sc := &verdict.SessionContext{ReferenceType: string(refType), Channel: channelOf(refType)}
		sc.FlowName = h.scopedFlowName(ctx, customerID, flowID)
		return sc, nil, false

	default:
		// Unknown reference_type: header-only, no leak, never panic.
		sc := &verdict.SessionContext{ReferenceType: string(refType), Channel: channelOf(refType)}
		sc.FlowName = h.scopedFlowName(ctx, customerID, flowID)
		return sc, nil, false
	}
}

// chaseOrigin resolves a transcribe/recording to its underlying call/
// conversation and renders the ORIGIN's participants/direction/outcome, stamping
// OriginKind/OriginType on the OUTER (transcribe/recording) reference. The card's
// reference_type STAYS the real activeflow value (F3). headerOnly ALWAYS stamps
// OriginKind so the "transcription/recording of" signal survives a chase miss
// (R5-L1).
func (h *analysisHandler) chaseOrigin(ctx context.Context, customerID uuid.UUID, refType fmactiveflow.ReferenceType, refID uuid.UUID, depth int) (*verdict.SessionContext, *verdict.SessionOutcome, bool) {
	if depth > enrichMaxChaseDepth {
		return headerOnly(refType), nil, true
	}

	originType, originID, ok := h.chaseRecord(ctx, customerID, refType, refID)
	if !ok {
		// record missing or not owned -> header-only + marker, never leak.
		return headerOnly(refType), nil, true
	}

	// Resolve the origin with the SAME call/conversation provider (no duplicate
	// logic). flowID="" because the origin's flow is a different activeflow and
	// FlowName is suppressed on chased cards anyway (#5).
	sc, outcome, _ := h.enrichRef(ctx, customerID, fmactiveflow.ReferenceType(originType), originID, uuid.Nil, depth+1)
	if sc == nil {
		// origin unresolvable -> header-only + marker.
		return headerOnly(refType), nil, true
	}

	// Build the card from the ORIGIN's body but keep the OUTER reference_type.
	sc.ReferenceType = string(refType)            // stays "transcribe"/"recording" (F3)
	sc.OriginKind = markerFor(refType)            // "transcription" | "recording"
	sc.OriginType = originType                    // immediate origin: call|conversation|confbridge (#6)
	sc.Channel = channelOf(fmactiveflow.ReferenceType(originType)) // channel of the underlying medium (F5)
	sc.FlowName = ""                              // origin's flow is a different activeflow; omit (F2)
	return sc, outcome, true                      // chased -> enrich() suppresses metrics/flags
}

// chaseRecord fetches the transcribe/recording record itself, ownership-checks
// it BEFORE its ReferenceID is trusted (F10), and returns its origin reference
// (type-string, id). The origin reference_type strings are the channel manager
// enums, which channelOf understands (call|confbridge for recording; call|
// confbridge|recording|unknown for transcribe).
func (h *analysisHandler) chaseRecord(ctx context.Context, customerID uuid.UUID, refType fmactiveflow.ReferenceType, refID uuid.UUID) (string, uuid.UUID, bool) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "chaseRecord",
		"customer_id":    customerID,
		"reference_type": refType,
		"reference_id":   refID,
	})

	switch refType {
	case fmactiveflow.ReferenceTypeTranscribe:
		tr, err := h.reqHandler.TranscribeV1TranscribeGet(ctx, refID)
		if err != nil || tr == nil {
			log.Warnf("could not get transcribe for origin chase. err: %v", err)
			return "", uuid.Nil, false
		}
		if tr.CustomerID != customerID {
			log.Warnf("transcribe customer mismatch; refusing chase.")
			return "", uuid.Nil, false
		}
		return string(tr.ReferenceType), tr.ReferenceID, true

	case fmactiveflow.ReferenceTypeRecording:
		rec, err := h.reqHandler.CallV1RecordingGet(ctx, refID)
		if err != nil || rec == nil {
			log.Warnf("could not get recording for origin chase. err: %v", err)
			return "", uuid.Nil, false
		}
		if rec.CustomerID != customerID {
			log.Warnf("recording customer mismatch; refusing chase.")
			return "", uuid.Nil, false
		}
		return string(rec.ReferenceType), rec.ReferenceID, true

	default:
		return "", uuid.Nil, false
	}
}

// enrichCall resolves a call reference into participants/direction + a call
// outcome (the highest-value reference). Lists by activeflow_id so multi-leg is
// detectable; falls back to a single Get. Customer-scoped (never leak).
func (h *analysisHandler) enrichCall(ctx context.Context, customerID, refID uuid.UUID, sc *verdict.SessionContext) *verdict.SessionOutcome {
	log := logrus.WithFields(logrus.Fields{"func": "enrichCall", "customer_id": customerID, "reference_id": refID})

	calls, err := h.reqHandler.CallV1CallList(ctx, "", 100, map[cmcall.Field]any{
		cmcall.FieldActiveflowID: refID,
		cmcall.FieldCustomerID:   customerID,
		cmcall.FieldDeleted:      false,
	})
	if err != nil || len(calls) == 0 {
		// fall back to a direct Get on the reference id.
		c, gerr := h.reqHandler.CallV1CallGet(ctx, refID)
		if gerr != nil || c == nil {
			log.Warnf("could not resolve call. list_err: %v, get_err: %v", err, gerr)
			return nil
		}
		calls = []cmcall.Call{*c}
	}

	sc.MultiLeg = len(calls) > 1

	// primary = earliest TMCreate (entry leg).
	primary := calls[0]
	for _, c := range calls[1:] {
		if tmBefore(c.TMCreate, primary.TMCreate) {
			primary = c
		}
	}
	if primary.CustomerID != customerID {
		log.Warnf("resolved call customer mismatch; refusing.")
		return nil
	}

	sc.DirectionRaw = string(primary.Direction)
	sc.Direction = normalizeDir(string(primary.Direction))
	sc.Participants = []verdict.Participant{
		{Role: "source", Address: primary.Source.Target},
		{Role: "destination", Address: primary.Destination.Target},
	}
	sc.StartedAt = rfc3339OrEmpty(firstNonNil(primary.TMProgressing, primary.TMCreate))

	durSec := callDurationSec(&primary)
	detail := map[string]string{}
	if durSec >= 0 {
		detail["duration_sec"] = strconv.Itoa(durSec)
	}
	if primary.HangupReason != "" {
		detail["hangup_reason"] = string(primary.HangupReason)
	}

	return &verdict.SessionOutcome{
		Result:  mapCallResult(primary.HangupReason, primary.Status),
		EndedBy: string(primary.HangupBy),
		Reason:  string(primary.HangupReason),
		Detail:  detailOrNil(detail),
	}
}

// enrichConversation resolves a conversation thread into Self/Peer participants
// and a dialogue-flow outcome (NOT hangup). Self = the VoIPBin customer's
// (business) address, Peer = the end-user (code-verified). INVARIANT:
// outgoing == business reply (self), incoming == end-user message (peer). A
// refactor must not silently invert this.
func (h *analysisHandler) enrichConversation(ctx context.Context, customerID, refID uuid.UUID, sc *verdict.SessionContext) *verdict.SessionOutcome {
	log := logrus.WithFields(logrus.Fields{"func": "enrichConversation", "customer_id": customerID, "reference_id": refID})

	cv, err := h.reqHandler.ConversationV1ConversationGet(ctx, refID)
	if err != nil || cv == nil {
		log.Warnf("could not get conversation. err: %v", err)
		return nil
	}
	if cv.CustomerID != customerID {
		log.Warnf("conversation customer mismatch; refusing.")
		return nil
	}

	sc.Channel = "chat"
	sc.Participants = []verdict.Participant{
		{Role: "self", Address: cv.Self.Target},
		{Role: "peer", Address: cv.Peer.Target},
	}
	sc.StartedAt = rfc3339OrEmpty(cv.TMCreate)
	// Thread-level Direction stays EMPTY in P1: direction is per-message, a
	// single representative is low-value and contradicts the per-message reality
	// (F7). Dialogue-flow lives in the outcome Detail.

	// MessageList returns tm_create DESC, capped at 1000 (no pagination, F6).
	msgs, err := h.reqHandler.ConversationV1MessageList(ctx, "", 1000, map[cvmessage.Field]any{
		cvmessage.FieldConversationID: refID,
		cvmessage.FieldDeleted:        false,
	})
	if err != nil {
		log.Warnf("could not list conversation messages. err: %v", err)
		// participants still useful; emit in_progress with chat_platform only.
		return &verdict.SessionOutcome{
			Result: "in_progress",
			Detail: detailOrNil(map[string]string{"chat_platform": string(cv.Type)}),
		}
	}
	if len(msgs) == 0 {
		return &verdict.SessionOutcome{
			Result: "in_progress",
			Detail: detailOrNil(map[string]string{"chat_platform": string(cv.Type)}),
		}
	}

	// Sort ASC by tm_create locally before first/last/span math (F6).
	sort.SliceStable(msgs, func(i, j int) bool { return tmBefore(msgs[i].TMCreate, msgs[j].TMCreate) })
	first := msgs[0]
	last := msgs[len(msgs)-1]

	turnsSelf := 0 // business replies (outgoing)
	turnsPeer := 0 // end-user messages (incoming)
	failures := 0
	for i := range msgs {
		switch msgs[i].Direction {
		case cvmessage.DirectionOutgoing:
			turnsSelf++
		case cvmessage.DirectionIncoming:
			turnsPeer++
		}
		if msgs[i].Status == cvmessage.StatusFailed {
			failures++
		}
	}

	lastActivityBy := "self"
	unanswered := "false"
	if last.Direction == cvmessage.DirectionIncoming {
		lastActivityBy = "peer"
		unanswered = "true" // end-user spoke last, business did not reply
	}

	// Result reflects the LAST message status, NOT any-failed (F12). A single
	// failed delivery among many is surfaced as a delivery_failures count.
	result := "in_progress"
	switch last.Status {
	case cvmessage.StatusDone:
		result = "completed"
	case cvmessage.StatusFailed:
		result = "failed"
	}

	detail := map[string]string{
		"chat_platform":     string(cv.Type), // message|line|whatsapp (F13)
		"last_activity_by":  lastActivityBy,
		"turns_self":        strconv.Itoa(turnsSelf),
		"turns_peer":        strconv.Itoa(turnsPeer),
		"unanswered":        unanswered,
		"delivery_failures": strconv.Itoa(failures),
		"thread_span_sec":   strconv.Itoa(threadSpanSec(first.TMCreate, last.TMCreate)),
		"truncated":         boolStr(len(msgs) >= 1000),
	}

	return &verdict.SessionOutcome{
		Result: result,
		Reason: string(last.Status),
		Detail: detailOrNil(detail),
	}
}

// enrichAIcall resolves an ai reference best-effort (P1): header + an end-status
// outcome from the AIcall Get. A second hop to the underlying call leg for
// participants is deferred to P2 (design Q3). Customer-scoped.
func (h *analysisHandler) enrichAIcall(ctx context.Context, customerID, refID uuid.UUID, sc *verdict.SessionContext) *verdict.SessionOutcome {
	log := logrus.WithFields(logrus.Fields{"func": "enrichAIcall", "customer_id": customerID, "reference_id": refID})

	ac, err := h.reqHandler.AIV1AIcallGet(ctx, refID)
	if err != nil || ac == nil {
		log.Warnf("could not get aicall. err: %v", err)
		return nil
	}
	if ac.CustomerID != customerID {
		log.Warnf("aicall customer mismatch; refusing.")
		return nil
	}

	sc.StartedAt = rfc3339OrEmpty(ac.TMCreate)

	result := "in_progress"
	if ac.Status == amaicall.StatusTerminated {
		result = "completed"
	}
	return &verdict.SessionOutcome{
		Result: result,
		Reason: string(ac.Status),
	}
}

// scopedFlowName returns the flow's name, customer-scoped (F17 — a foreign flow
// name must not leak). Empty flowID or any error/mismatch yields "".
func (h *analysisHandler) scopedFlowName(ctx context.Context, customerID, flowID uuid.UUID) string {
	if flowID == uuid.Nil {
		return ""
	}
	f, err := h.reqHandler.FlowV1FlowGet(ctx, flowID)
	if err != nil || f == nil {
		return ""
	}
	if f.CustomerID != customerID {
		return ""
	}
	return f.Name
}

// --- pure helpers (no RPC) ---

// channelOf is TOTAL over the 8 activeflow reference_types PLUS confbridge (a
// transcribe/recording origin type, F5). transcribe/recording never map to
// their own channel (the chased card's channel is set from the origin).
func channelOf(refType fmactiveflow.ReferenceType) string {
	switch refType {
	case fmactiveflow.ReferenceTypeCall:
		return "voice"
	case fmactiveflow.ReferenceTypeConversation:
		return "chat"
	case fmactiveflow.ReferenceTypeAI:
		return "ai"
	case fmactiveflow.ReferenceTypeAPI:
		return "api"
	case fmactiveflow.ReferenceTypeCampaign:
		return "voice" // a campaign dials calls (design Q1)
	case fmactiveflow.ReferenceType("confbridge"):
		return "voice"
	default:
		// none, transcribe, recording, unknown -> no own channel.
		return ""
	}
}

// normalizeDir maps the call source enum to the normalized inbound/outbound.
func normalizeDir(raw string) string {
	switch raw {
	case string(cmcall.DirectionIncoming):
		return "inbound"
	case string(cmcall.DirectionOutgoing):
		return "outbound"
	default:
		return ""
	}
}

// mapCallResult maps (HangupReason, Status) to the normalized Result enum. There
// is NO "abandoned": a customer hanging up early is Result=completed +
// EndedBy=remote + short duration_sec; the UI/LLM interprets that, the data does
// not invent a reason the source lacks (F8).
func mapCallResult(reason cmcall.HangupReason, status cmcall.Status) string {
	switch reason {
	case cmcall.HangupReasonNormal:
		return "completed"
	case cmcall.HangupReasonNoanswer, cmcall.HangupReasonDialout:
		return "no_answer"
	case cmcall.HangupReasonBusy:
		return "busy"
	case cmcall.HangupReasonFailed, cmcall.HangupReasonCanceled, cmcall.HangupReasonTimeout, cmcall.HangupReasonAMD:
		return "failed"
	case cmcall.HangupReasonNone:
		if status == cmcall.StatusHangup {
			return "completed"
		}
		return "in_progress"
	default:
		return "unknown"
	}
}

// headerOnly builds a header-only SessionContext for a transcribe/recording
// whose origin is unresolvable, ALWAYS stamping the OriginKind marker so the
// "transcription/recording of" signal survives a chase miss (R5-L1).
func headerOnly(refType fmactiveflow.ReferenceType) *verdict.SessionContext {
	return &verdict.SessionContext{
		ReferenceType: string(refType),
		Channel:       channelOf(refType), // "" for transcribe/recording
		OriginKind:    markerFor(refType),
	}
}

// markerFor returns the origin_kind marker for a transcribe/recording reference.
func markerFor(refType fmactiveflow.ReferenceType) string {
	switch refType {
	case fmactiveflow.ReferenceTypeTranscribe:
		return "transcription"
	case fmactiveflow.ReferenceTypeRecording:
		return "recording"
	default:
		return ""
	}
}

// aiHandled returns true if a pipecat/ai session was present in the stream.
func aiHandled(events []*canonicalEvent) bool {
	for _, ce := range events {
		switch ce.Publisher {
		case string(commonoutline.ServiceNameAIManager), string(commonoutline.ServiceNamePipecatManager):
			return true
		}
	}
	return false
}

// agentLegConnected returns true if an agent-manager event landed in THIS
// activeflow-scoped stream, i.e. an agent leg participated. Queue entry is
// queue-manager (not agent-manager), so a queue-then-abandon flow does NOT emit
// agent-manager events here; this signal is therefore stronger than mere queue
// entry (design §6d). If agent-manager never publishes into the activeflow, this
// stays false (conservative).
func agentLegConnected(events []*canonicalEvent) bool {
	for _, ce := range events {
		if ce.Publisher == string(commonoutline.ServiceNameAgentManager) {
			return true
		}
	}
	return false
}

// aggregateMetrics computes the voice/AI interaction-quality aggregate over the
// FULL pre-reduction stream. Returns nil when there are no transcription turns
// (no misleading zero-value block). Latency fields are nil when their inputs are
// absent (no misleading 0).
func aggregateMetrics(events []*canonicalEvent) *verdict.SessionMetrics {
	const (
		evUser = "message_user_transcription"
		evBot  = "message_bot_transcription"
		evInit = "pipecatcall_initialized"
	)

	type tsType struct {
		ts time.Time
		t  string // "user" | "bot"
	}

	var turns []tsType
	userTurns, botTurns := 0, 0
	var answerTime time.Time
	haveAnswer := false

	for _, ce := range events {
		// exact event-type match (substring would catch *_llm_intermediate).
		switch ce.EventType {
		case evUser:
			userTurns++
			if ts, ok := parseCanonicalTS(ce.Timestamp); ok {
				turns = append(turns, tsType{ts: ts, t: "user"})
			}
		case evBot:
			botTurns++
			if ts, ok := parseCanonicalTS(ce.Timestamp); ok {
				turns = append(turns, tsType{ts: ts, t: "bot"})
			}
		case evInit:
			if ts, ok := parseCanonicalTS(ce.Timestamp); ok {
				if !haveAnswer || ts.Before(answerTime) {
					answerTime = ts
					haveAnswer = true
				}
			}
		}
	}

	if userTurns == 0 && botTurns == 0 {
		return nil
	}

	m := &verdict.SessionMetrics{TurnsUser: userTurns, TurnsBot: botTurns}

	// sort interaction turns chronologically for latency/gap math.
	sort.SliceStable(turns, func(i, j int) bool { return turns[i].ts.Before(turns[j].ts) })

	// first bot turn.
	var firstBot time.Time
	haveFirstBot := false
	for _, tt := range turns {
		if tt.t == "bot" {
			firstBot = tt.ts
			haveFirstBot = true
			break
		}
	}
	if haveAnswer && haveFirstBot {
		if ms := clamp0MS(firstBot.Sub(answerTime)); ms >= 0 {
			m.FirstResponseMS = &ms
		}
	}

	// response latencies over (userTurn -> next botTurn) pairs.
	var sum, maxResp int
	pairs := 0
	for i := 0; i < len(turns); i++ {
		if turns[i].t != "user" {
			continue
		}
		for j := i + 1; j < len(turns); j++ {
			if turns[j].t == "bot" {
				ms := clamp0MS(turns[j].ts.Sub(turns[i].ts))
				sum += ms
				if ms > maxResp {
					maxResp = ms
				}
				pairs++
				break
			}
		}
	}
	if pairs > 0 {
		avg := sum / pairs
		m.AvgResponseMS = &avg
		m.MaxResponseMS = &maxResp
	}

	// max gap between adjacent interaction events (NOT silence).
	if len(turns) >= 2 {
		maxGap := 0
		for i := 1; i < len(turns); i++ {
			ms := clamp0MS(turns[i].ts.Sub(turns[i-1].ts))
			if ms > maxGap {
				maxGap = ms
			}
		}
		m.MaxGapMS = &maxGap
	}

	return m
}

func parseCanonicalTS(s string) (time.Time, bool) {
	t, err := time.Parse(canonicalTimestampLayout, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// clamp0MS converts a duration to milliseconds, clamped to >= 0 (cross-service
// clock skew can produce a small negative; clamp rather than emit a misleading
// negative latency).
func clamp0MS(d time.Duration) int {
	ms := int(d.Milliseconds())
	if ms < 0 {
		return 0
	}
	return ms
}

func callDurationSec(c *cmcall.Call) int {
	if c.TMProgressing == nil || c.TMHangup == nil {
		return -1 // unknown (not answered, or still up)
	}
	d := c.TMHangup.Sub(*c.TMProgressing)
	if d < 0 {
		return 0
	}
	return int(d.Seconds())
}

func threadSpanSec(first, last *time.Time) int {
	if first == nil || last == nil {
		return 0
	}
	d := last.Sub(*first)
	if d < 0 {
		return 0
	}
	return int(d.Seconds())
}

func tmBefore(a, b *time.Time) bool {
	switch {
	case a == nil:
		return false
	case b == nil:
		return true
	default:
		return a.Before(*b)
	}
}

func firstNonNil(a, b *time.Time) *time.Time {
	if a != nil {
		return a
	}
	return b
}

func rfc3339OrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func detailOrNil(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	return m
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
