package action

// meta.go holds the Flow AI Builder's metadata registry (VOIP-1573). The
// builder (bin-ai-manager) never branches on an action Type name; it reads
// only this file's Meta/MetaByType and the ref struct tags in option.go.
//
// SYNC NOTE (read before editing TypeListAll):
//   - Adding a new action Type requires adding it to MetaByType, or
//     TestMetaCoversAllTypes (in this package) will fail.
//   - Meta.Exposure and Meta.FlowKind are confirmed by reading the action's
//     real execution path in bin-flow-manager/pkg/activeflowhandler before
//     merge; see docs/plans/2026-10-07-flow-ai-builder-design.md Appendix B
//     for the per-type reasoning recorded during design review.

// Exposure says whether, and under what disclosure, the Flow AI Builder may
// place this action type into a drafted graph.
type Exposure string

const (
	// ExposureCore: no added cost, no external effect. Always offered.
	ExposureCore Exposure = "core"

	// ExposureSensitive: incurs cost or sends data externally (outbound
	// call, message, email, webhook, fetch, LLM calls). Offered, but the
	// builder marks every node of this kind in ChatResponse.SensitiveNodes
	// and the system prompt instructs the model to include it only when
	// the user actually asked for that behavior.
	ExposureSensitive Exposure = "sensitive"

	// ExposureInternal: never offered to the builder. Either a pure
	// platform-internal action, a type the editor cannot round-trip
	// faithfully (goto), a type whose option carries a nested []Action or
	// []Attachment the builder's ref-tag scheme does not reach (call,
	// email_send are excluded structurally, not by this value, but are
	// still internal from the builder's point of view), or a type whose
	// only reference field is a value the customer cannot supply (e.g. a
	// confbridge_id minted at runtime by connect).
	ExposureInternal Exposure = "internal"
)

// FlowKind says how control flow continues after this action executes, for
// the subset of behavior the builder's graph validator needs. It is a
// coarser, builder-facing view of the real execution semantics in
// pkg/activeflowhandler/pkg/stackmaphandler; see the design doc Appendix B
// for the per-type verification notes.
type FlowKind string

const (
	// FlowKindContinue: next_id if set, otherwise the next array element,
	// otherwise finish (pkg/stackmaphandler.GetNextAction). Most types,
	// including the true branch of every condition_* type.
	FlowKindContinue FlowKind = "continue"

	// FlowKindTerminate: flow does not continue after this action, on the
	// media type(s) the action actually applies to (see
	// MapRequiredMediasByType); next_id is ignored. stop, hangup.
	FlowKindTerminate FlowKind = "terminate"

	// FlowKindJump: the next action is decided only by the action's own
	// ref:"action" option field(s); next_id/array-adjacency is never
	// consulted, and an empty/invalid target aborts the activeflow at
	// runtime. branch, and the false path of every condition_* type.
	FlowKindJump FlowKind = "jump"
)

// Meta is the single per-type metadata record the Flow AI Builder reads.
type Meta struct {
	Exposure Exposure
	Flow     FlowKind
}

// MetaByType declares Exposure and FlowKind for every action Type in
// TypeListAll. TestMetaCoversAllTypes enforces completeness.
//
// goto is deliberately absent from the FlowKind model (it is a conditional
// jump: loop_count<=0 continues, otherwise it jumps) and from the editor's
// round-trip (actiongraph/store.js never restores its next_id), so it is
// ExposureInternal with no exported FlowKind meaning in v1; the builder must
// not place it. The Flow field is still set to FlowKindContinue as a safe,
// unused placeholder (the type is filtered out by Exposure before FlowKind
// is ever consulted).
var MetaByType = map[Type]Meta{
	TypeAMD:                 {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeAnswer:              {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeAISummary:           {Exposure: ExposureInternal, Flow: FlowKindContinue}, // reference_id is a runtime-minted id, not customer-fillable
	TypeAITalk:              {Exposure: ExposureSensitive, Flow: FlowKindContinue},
	TypeAITask:              {Exposure: ExposureSensitive, Flow: FlowKindContinue},
	TypeBeep:                {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeBlock:               {Exposure: ExposureInternal, Flow: FlowKindContinue}, // catalog summary: internal grouping/block action
	TypeBranch:              {Exposure: ExposureCore, Flow: FlowKindJump},
	TypeCall:                {Exposure: ExposureSensitive, Flow: FlowKindContinue}, // structurally excluded from the builder catalog (nested []Action); Meta value still declared
	TypeCaseCreate:          {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeConditionCallDigits: {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeConditionCallStatus: {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeConditionDatetime:   {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeConditionVariable:   {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeConfbridgeJoin:      {Exposure: ExposureInternal, Flow: FlowKindContinue}, // confbridge_id is minted at runtime by connect; customer cannot supply one
	TypeConferenceJoin:      {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeConnect:             {Exposure: ExposureSensitive, Flow: FlowKindContinue}, // originates an outbound call
	TypeConversationSend:    {Exposure: ExposureSensitive, Flow: FlowKindContinue},
	TypeDigitsReceive:       {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeDigitsSend:          {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeEcho:                {Exposure: ExposureInternal, Flow: FlowKindContinue},  // internal/test use
	TypeEmailSend:           {Exposure: ExposureSensitive, Flow: FlowKindContinue}, // structurally excluded (nested []Attachment); Meta value still declared
	TypeExternalMediaStart:  {Exposure: ExposureInternal, Flow: FlowKindContinue},
	TypeExternalMediaStop:   {Exposure: ExposureInternal, Flow: FlowKindContinue},
	TypeFetch:               {Exposure: ExposureInternal, Flow: FlowKindContinue}, // fetches actions from an arbitrary URL with a plain http.Client (pkg/actionhandler ActionFetchGet): no SSRF guard, so the builder does not place it (VOIP-1573 OQ8). Revisit when the executor validates the URL
	TypeFetchFlow:           {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeGoto:                {Exposure: ExposureInternal, Flow: FlowKindContinue}, // outside the FlowKind model, see doc comment above
	TypeHangup:              {Exposure: ExposureCore, Flow: FlowKindTerminate},
	TypeMessageSend:         {Exposure: ExposureSensitive, Flow: FlowKindContinue},
	TypeMute:                {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypePlay:                {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeQueueJoin:           {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeRecordingStart:      {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeRecordingStop:       {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeSleep:               {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeStop:                {Exposure: ExposureCore, Flow: FlowKindTerminate},
	TypeStreamEcho:          {Exposure: ExposureInternal, Flow: FlowKindContinue}, // internal/test use
	TypeTalk:                {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeTranscribeStart:     {Exposure: ExposureSensitive, Flow: FlowKindContinue}, // STT cost
	TypeTranscribeStop:      {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeTranscribeRecording: {Exposure: ExposureSensitive, Flow: FlowKindContinue}, // STT cost
	TypeVariableSet:         {Exposure: ExposureCore, Flow: FlowKindContinue},
	TypeWebhookSend:         {Exposure: ExposureSensitive, Flow: FlowKindContinue},
}

// BuilderExcludedTypes lists action types the Flow AI Builder never offers
// regardless of Exposure, because their option carries a nested []Action or
// []Attachment the ref-tag scheme in option.go does not reach
// (OptionCall.Actions, OptionEmailSend.Attachments[].ReferenceID).
// TestBuilderExcludedTypes pins this list so a future change is visible.
var BuilderExcludedTypes = map[Type]bool{
	TypeCall:      true,
	TypeEmailSend: true,
}

// IsBuilderExposable reports whether t may ever appear in a Flow AI Builder
// draft: it has a core or sensitive Meta.Exposure and is not structurally
// excluded. internal types and goto are never exposable.
func IsBuilderExposable(t Type) bool {
	if BuilderExcludedTypes[t] {
		return false
	}
	m, ok := MetaByType[t]
	if !ok {
		return false
	}
	return m.Exposure == ExposureCore || m.Exposure == ExposureSensitive
}
