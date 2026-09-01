package summaryhandler

import (
	"context"

	"monorepo/bin-ai-manager/models/summary"
	cmcall "monorepo/bin-call-manager/models/call"
	cfconference "monorepo/bin-conference-manager/models/conference"
)

// EventCMCallHangup handles the call-manager's call_hangup event
func (h *summaryHandler) EventCMCallHangup(ctx context.Context, c *cmcall.Call) {
	sm, err := h.GetByReferenceID(ctx, c.ID)
	if err != nil {
		// summary not found. nothing todo
		return
	}

	h.ContentProcess(ctx, sm)
}

// EventCMConferenceUpdated handles the conference-manager's conference_deleted event
// (VOIP-1422: this is dispatched from EventTypeConferenceDeleted, not
// EventTypeConferenceUpdated -- conference_updated is never published with
// Status == StatusTerminated in bin-conference-manager, so a conference_updated
// binding would make this permanently unreachable).
//
// Deliberately has NO check on c.Status (the conference's own lifecycle field),
// matching EventCMCallHangup's pattern immediately above: the event TYPE
// (conference_deleted) is the terminal signal, not any field on the payload. This
// matters because bin-conference-manager has two conference_deleted publish sites and
// they disagree on c.Status at publish time -- Destroy() (the participants-all-left
// auto-cleanup path) calls ConferenceEnd first, which sets Status = StatusTerminated
// before publishing; but Delete() (the customer-facing DELETE /conferences/{id} API
// path) calls Terminating() (Status = StatusTerminating) then ConferenceDelete()
// (which only touches tm_update/tm_delete, never status) before publishing -- so
// Delete()'s conference_deleted payload carries c.Status == StatusTerminating, not
// Terminated. An earlier version of this fix checked c.Status == StatusTerminated,
// which silently dropped every explicit-API-deletion summary finalization while
// appearing to work for the auto-cleanup path.
//
// DOES check sm.Status (the summary's own processing state), for a different reason:
// bin-conference-manager's Delete() kicks every conferencecall asynchronously
// (ConferenceV1ConferencecallKick, fire-and-forget) and publishes conference_deleted
// #1 immediately afterward with c.Status == StatusTerminating, WITHOUT waiting for
// those kicks to land. When the kicked participants actually leave later,
// RemoveConferencecallID sees the conference is still StatusTerminating (Delete()'s
// own DB call never advanced it) with zero conferencecalls left, and calls Destroy()
// -- which publishes conference_deleted #2 with c.Status == StatusTerminated. Both
// events reach this handler for the same conference (ConferenceGet has no tm_delete
// filter, so the second lookup still succeeds), and neither ContentProcess nor its
// downstream (OpenAI call, summary_updated webhook, on-end-flow execution) is
// idempotent on its own.
//
// This sm.Status == StatusDone check here is a cheap, non-authoritative fast path,
// NOT the correctness guarantee: it is a read-then-branch, so a narrow window remains
// where both deliveries could pass it before either write lands (every event runs on
// its own unserialized goroutine two hops deep -- subscribehandler's processEventRun
// spawns one per message, and processEventCMConferenceUpdated spawns a second to call
// this handler -- and the OpenAI round-trip inside ContentProcess can run for
// conference_deleted publishes; an earlier version of this fix asserted that gap was
// "much wider" without evidence, which code review correctly rejected). The actual
// correctness guarantee is downstream, in UpdateStatusDone's conditional
// SummaryUpdateStatusDoneIfNotDone (`WHERE ... AND status != 'done'`, checked via
// RowsAffected) -- whichever goroutine's write lands first wins the DB row
// unconditionally, and the loser gets ErrSummaryAlreadyDone and skips startOnEndFlow
// and the webhook. This early check exists purely to skip the expensive
// conference/transcript/OpenAI round-trip in the common case where the second
// delivery arrives well after the first has already finished -- the DB check would
// catch it either way, just after paying for work that was already going to be
// thrown away.
func (h *summaryHandler) EventCMConferenceUpdated(ctx context.Context, c *cfconference.Conference) {
	sm, err := h.GetByReferenceID(ctx, c.ID)
	if err != nil {
		// summary not found. nothing todo
		return
	}

	if sm.Status == summary.StatusDone {
		// already finalized by an earlier conference_deleted delivery for the same
		// conference (see doc comment above) -- do not reprocess.
		return
	}

	h.ContentProcess(ctx, sm)
}
