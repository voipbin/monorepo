package aicallhandler

import (
	"context"
	"errors"
	"time"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cerrors "monorepo/bin-common-handler/models/errors"
	kmkase "monorepo/bin-contact-manager/models/kase"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// defaultListenTranscribeStartTimeout is the transcribe-start RPC timeout, in
// milliseconds. Matches summaryhandler's own transcribe start.
const defaultListenTranscribeStartTimeout = 5000

// listenResolverTTL bounds how long a transcribe -> AIcall resolver entry can
// outlive a lost cleanup. Twelve hours comfortably exceeds any real call while
// still guaranteeing the key cannot leak forever.
//
// Deliberately a constant and not a config flag (design §5.2.4): unlike the
// timing flags in internal/config, this bounds a worst-case safety margin --
// how long a genuinely orphaned resolver entry can outlive its transcribe --
// rather than a value anyone is expected to tune.
const listenResolverTTL = 12 * time.Hour

// transcribeReasonAlreadyProgressing is transcribe-manager's rejection reason
// when a session for this (customer_id, reference_id, language) is already
// live. If bin-transcribe-manager ever exports this as a constant, use that
// instead of the literal.
const transcribeReasonAlreadyProgressing = "TRANSCRIBE_ALREADY_PROGRESSING"

// ProcessListen is the sole exported entry point for the listen trigger,
// called once by processV1AIcallsIDListenPost -- the same one-call shape as
// ProcessTerminate (design §5.1, rev 16).
//
// It resolves the AIcall, runs design §5.1.1 steps 1-6 INLINE (synchronously;
// nothing slower than the three RPCs the caller's own longer timeout already
// budgets for), and -- only if every step passes -- spawns a detached
// goroutine for steps 7-8, closing over the already-resolved a/c/kase/callID/
// call values directly. No value crosses a function boundary by itself, so
// there is nothing to re-fetch and nothing to silently lose (review round 13
// finding HIGH-1).
//
// It returns the AIcall UNCHANGED by steps 1-6 themselves; steps 7-8 write
// asynchronously. The response deliberately carries no listening-status field
// (design §5.1, §11 item 14) -- the caller cannot tell "started" from "reused"
// from "not eligible" from "still waiting on the confbridge", and MUST NOT
// block waiting to find out.
func (h *aicallHandler) ProcessListen(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	c, err := h.Get(ctx, id) // cache-first, same as every other single-AIcall route
	if err != nil {
		return nil, err
	}

	a, kase, callID, call, proceed, err := h.checkListenEligible(ctx, c) // §5.1.1 steps 1-6, inline
	if err != nil {
		return nil, err
	}
	if proceed {
		go func() {
			timeout := time.Duration(config.Get().AIcallListenEnsureGoroutineTimeoutSeconds) * time.Second
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// runListenStartHook is the injected seam design §7 item 2 calls
			// for; it is nil everywhere except in the tests that need to
			// observe this detached stage deterministically.
			if h.runListenStartHook != nil {
				h.runListenStartHook(ctx, a, c, kase, callID, call)
				return
			}

			h.runListenStart(ctx, a, c, kase, callID, call) // §5.1.1 steps 7-8
		}()
	}

	return c, nil
}

// checkListenEligible runs design §5.1.1 steps 1-6 and reports whether the
// detached steps 7-8 may proceed.
//
// THE TENANT BOUNDARY FOR THE WHOLE FEATURE IS HERE. The transcribe session
// runs under a platform system customer id, so its transcript events carry
// that system id, never a tenant id -- an event-time "does this transcript
// belong to this AIcall's customer?" check would ALWAYS fail and is
// impossible. Instead the tenant is checked once, here (customer-scoped
// CaseGet plus a CustomerID recheck on both the Case and the call), and the
// event path verifies PROVENANCE instead: is this transcribe id one we
// ourselves started and recorded? That is a stronger property -- the id is one
// ai-manager generated and persisted, not anything an attacker can influence.
//
// IT RETURNS EVERY VALUE STEPS 7-8 NEED, not a bare bool. a/kase/callID/call
// are all resolved here and handed straight to runListenStart; a bool-only
// boundary would force the goroutine to re-fetch the Case and the call, which
// is exactly the defect review round 13's HIGH-1 found in the first draft of
// this split.
//
// A LOOKUP FAILURE IS NOT AN ERROR RETURN. Design §6's first row is explicit
// that a Case/call/transcribe lookup failure is "logged, metered, listening
// simply does not start" and must never fail the triggering call -- which,
// since rev 15, is the POST itself. So those paths return proceed=false with a
// NIL error and a metered outcome; the error return exists for genuinely
// unexpected conditions only.
//
// CONSEQUENCE, STATED SO NOBODY "SIMPLIFIES" IT AWAY: no branch in the body
// below ever returns a non-nil error. That is intentional, not an oversight.
// The six-value signature matches design §5.1's own snippet shape, and the
// error return is the seam a genuinely unexpected condition would use if one
// is ever added. Dropping it to five values would make this function's
// contract diverge from the design for no gain, and would have to be undone
// the first time a real error case appears.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.1, §5.1.1.
func (h *aicallHandler) checkListenEligible(ctx context.Context, c *aicall.AIcall) (*ai.AI, *kmkase.Case, uuid.UUID, *cmcall.Call, bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "checkListenEligible",
		"aicall_id": c.ID,
	})

	// Step 1: feature gate.
	//
	// NOT METERED, and that is a gap this plan records rather than papers
	// over (review round 1 finding LOW-8). Design §5.13 enumerates
	// aicall_listen_start_total's `result` values as started / reused /
	// skipped_not_listenable / skipped_confbridge_not_ready /
	// skipped_confbridge_error / failed, and never says which one covers "the
	// feature flag is off." Folding it into skipped_not_listenable is the
	// plausible reading, but the design does not state it and inventing a
	// seventh value here would be exactly the kind of unilateral decision this
	// task flags elsewhere. Leaving it unmetered is also defensible on its own:
	// during a flag-off rollout stage EVERY call takes this branch, so the
	// counter would say nothing the flag's own value does not already say. If
	// a reviewer wants it metered, decide the label in the design first.
	if !config.Get().AIcallListenEnabled {
		return nil, nil, uuid.Nil, nil, false, nil
	}

	// Step 2: AIcall gate -- type AND liveness, combined.
	//
	// The liveness half is NEW with the public endpoint (design rev 16, review
	// round 13 finding MEDIUM-2). Start's old hook only ever ran against an
	// AIcall it had just created or reused as active; an arbitrarily-callable
	// POST removes that guarantee. Without this, a terminated AIcall could
	// pass steps 3-6, spawn the 45s goroutine, and start a BILLED STT session
	// that only RunListenTurn's own unrelated precondition would eventually
	// reap, on the first transcript segment -- later and less directly than
	// catching it here.
	//
	// Deny by default on the type: contact_case AIcalls are Insight in
	// practice, but this does not rely on that.
	//
	// DEVIATION FROM THE PLAN'S SNIPPET, recorded rather than silently taken:
	// the plan writes h.aiHandler.Get(ctx, c.AIID), but aicall.AIcall has no
	// AIID field -- the AI is addressed through AssistanceType/AssistanceID
	// (and, for a team, through the current member). resolveActiveAIIDFromAIcall
	// is this service's own existing resolution of exactly that, and is what
	// every other "which AI is active on this AIcall" site already uses.
	a, err := h.aiHandler.Get(ctx, h.resolveActiveAIIDFromAIcall(ctx, c))
	if err != nil {
		log.Errorf("Could not get the ai. err: %v", err)
		promListenStartTotal.WithLabelValues("failed").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
	if a.Type != ai.TypeInsight || c.Status != aicall.StatusProgressing || c.TMDelete != nil {
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}

	// Step 3: idempotency. This is what makes repeated panel opens free -- and
	// the panel re-open path is the common one.
	//
	// THE REFERENCE-ID COMPARISON IS AN APPROXIMATION, DELIBERATELY, AND IS
	// FLAGGED RATHER THAN SILENTLY RESOLVED. Design §5.1.1 step 3 words the
	// predicate as comparing the existing transcribe's ReferenceID against
	// "the call we are about to resolve" -- but that call id does not exist
	// yet at step 3: steps 4-5 (Case lookup, reference typing) are what resolve
	// it, and the design's own step numbering puts them AFTER this check. The
	// requirement is therefore a forward reference to a value that, at this
	// point in the design's own ordering, is structurally unavailable.
	//
	// This compares against c.ListenCallID instead -- the call id a PRIOR
	// successful listen-start already persisted on this AIcall row. For the
	// overwhelmingly common case (repeated panel opens where the Case's call
	// linkage has not changed) the two are the same value, so the check is
	// exact, not merely close. It diverges in exactly one scenario: if a Case's
	// associated call somehow changed between two ProcessListen calls on the
	// same AIcall, this would read a stale prior session as "still listening,
	// skip," where §5.1.1 step 3's literal wording appears to want it treated
	// as a mismatch and a fresh start.
	//
	// DO NOT "FIX" THIS BY MOVING THE IDEMPOTENCY CHECK AFTER STEPS 4-5. That
	// reorders the design's explicit step numbering, which is a larger
	// divergence than the approximation itself, and it would put an RPC pair
	// ahead of the cheap short-circuit the common path exists for. Whoever
	// confirms whether a Case's reference call is fixed for the Case's lifetime
	// should settle this: if it is, the divergent scenario is provably
	// impossible and this predicate is exact.
	if existingID := listenTranscribeIDFromMetadata(c); existingID != uuid.Nil {
		if tr, errGet := h.reqHandler.TranscribeV1TranscribeGet(ctx, existingID); errGet == nil &&
			tr.Status == tmtranscribe.StatusProgressing && tr.TMDelete == nil && tr.ReferenceID == c.ListenCallID {
			// Metered as "reused": this AIcall is listening, via a session it
			// or another AIcall already started. Without this increment the
			// design's own stated common path (repeated panel opens) would
			// never appear in aicall_listen_start_total at all (review round 1
			// finding LOW-8).
			promListenStartTotal.WithLabelValues("reused").Inc()
			return nil, nil, uuid.Nil, nil, false, nil
		}
	}

	// Step 4: Case lookup, and the tenant boundary.
	//
	// THE REFERENCE-TYPE GATE IS EXPLICIT (review round 1 security finding
	// LOW-1). c.ReferenceID is only a Case id when ReferenceType is
	// contact_case; for any other reference type it addresses a call or a
	// conversation, and handing that to ContactV1CaseGet is a lookup for a
	// resource that is not a Case. It fails closed today (the Case simply is
	// not found, or its CustomerID recheck below rejects it), but every sibling
	// call site in this package -- RunListenTurn and all six Insight tools in
	// tool_insight.go -- states this precondition explicitly rather than
	// relying on that, and this one now matches them.
	if c.ReferenceType != aicall.ReferenceTypeContactCase {
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}

	kase, err := h.reqHandler.ContactV1CaseGet(ctx, c.CustomerID, c.ReferenceID)
	if err != nil {
		log.Errorf("Could not get the case. err: %v", err)
		promListenStartTotal.WithLabelValues("failed").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
	if kase.CustomerID != c.CustomerID || kase.CustomerID == uuid.Nil {
		// Defensive: the tenant is already embedded in the RPC, but fail closed
		// on any mismatch rather than trust a foreign response shape.
		log.Warnf("Cross-customer case access blocked. case_customer_id: %s", kase.CustomerID)
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}

	// Step 5: reference typing.
	if kase.ReferenceType != kmkase.ReferenceTypeCall {
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
	callID := uuid.FromStringOrNil(kase.ReferenceID)
	if callID == uuid.Nil {
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}

	// Step 6: call liveness + ownership.
	call, err := h.reqHandler.CallV1CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get the call. err: %v", err)
		promListenStartTotal.WithLabelValues("failed").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
	if call.CustomerID != c.CustomerID {
		log.Warnf("Cross-customer call access blocked. call_customer_id: %s", call.CustomerID)
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}
	if call.TMDelete != nil || !isListenableCallStatus(call.Status) {
		// The call is over. The agent can still read its finished transcript
		// with get_call_transcript, unchanged.
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return nil, nil, uuid.Nil, nil, false, nil
	}

	return a, kase, callID, call, true, nil
}

// runListenStart runs design §5.1.1 steps 7-8 in its own detached goroutine.
//
// Every argument is already resolved by checkListenEligible and passed
// directly -- NOTHING here re-fetches the Case or the call (review round 13
// finding HIGH-1). `a` and `kase` are unused by the steps below today; they are
// still passed because the design's own signature passes them, and because
// re-deriving either later would silently re-open that same defect.
//
// Fire-and-forget by design: no listening failure may ever fail the POST that
// triggered it, and the POST has already returned by the time this runs.
//
// THE TIMEOUT ITS CALLER GIVES IT IS PURPOSE-BUILT FOR THIS FEATURE, NOT
// INHERITED (design §5.1.1 intro, corrected in review round 11 finding LOW-3).
// It must stay strictly greater than AIcallListenConfbridgeReadyMaxWaitSeconds,
// since waitForConfbridgeReady's bounded retry runs inside this same goroutine
// and needs headroom for the RPC calls each poll makes -- and strictly less
// than AIcallListenStartLockTTLSeconds, so the lock below can never expire
// under a goroutine still working inside its own budget.
func (h *aicallHandler) runListenStart(ctx context.Context, a *ai.AI, c *aicall.AIcall, kase *kmkase.Case, callID uuid.UUID, call *cmcall.Call) {
	_, _ = a, kase // resolved by checkListenEligible; passed so nothing is ever re-fetched here

	log := logrus.WithFields(logrus.Fields{
		"func":      "runListenStart",
		"aicall_id": c.ID,
		"call_id":   callID,
	})

	// Step 7: the bounded confbridge-readiness retry.
	//
	// lastPartyCount is the LAST OBSERVED len(cb.ChannelCallIDs), or -1 if no
	// confbridge was ever observed. Design §6 requires it in the LOG LINE of
	// the NOT-READY branch specifically (§6's `skipped_confbridge_not_ready`
	// row), explicitly NOT as a metric label -- the count is unbounded-ish and
	// would be cardinality-bearing, but without it in the log there is no way
	// to tell a stuck-at-1 (slow ring) timeout from a stuck-at-3 (genuinely
	// wrong topology) one, since §5.1.1 step 7 deliberately gives both the same
	// label. This plan also logs it on the ERROR branch, for the same
	// diagnostic value, though §6's `skipped_confbridge_error` row does not
	// mandate it there. The third give-up path -- confbridgeCallEnded -- has no
	// log line at all, matching §6's own silence on that outcome.
	//
	// Named readyResult, not result: `result` is taken below by step 8's
	// metric label string, which is a different type.
	readyResult, lastPartyCount := h.waitForConfbridgeReady(ctx, callID)
	switch readyResult {
	case confbridgeReady:
		// proceed to step 8
	case confbridgeCallEnded:
		promListenStartTotal.WithLabelValues("skipped_not_listenable").Inc()
		return
	case confbridgeNotReady:
		log.Warnf("The confbridge did not become ready within the wait budget. Listening does not start. last_party_count: %d", lastPartyCount)
		promListenStartTotal.WithLabelValues("skipped_confbridge_not_ready").Inc()
		return
	case confbridgeError:
		log.Warnf("Could not check the confbridge readiness. Listening does not start. last_party_count: %d", lastPartyCount)
		promListenStartTotal.WithLabelValues("skipped_confbridge_error").Inc()
		return
	}

	// Step 8: the locked create-or-reuse sequence.
	result, err := h.startListenTranscribe(ctx, c, call, callID)
	promListenStartTotal.WithLabelValues(result).Inc()
	if err != nil {
		log.Errorf("Could not start listening. result: %s, err: %v", result, err)
		return
	}

	log.Debugf("Listen start finished. result: %s", result)
}

// startListenTranscribe is design §5.2.2's create-or-reuse sequence for one
// AIcall, wrapped in that section's per-AIcall lock.
//
// The returned string is this attempt's aicall_listen_start_total `result`
// label; runListenStart emits it. Every branch below is the design's own
// snippet, kept structurally line-for-line, because each branch is a
// regression fix from a specific review round and reordering or collapsing
// them reopens the corresponding race.
//
// WHY THE LOCK EXISTS (design §5.2.2, review round 14 finding HIGH-2). Design
// §5.1.1 step 7's retry means the SAME AIcall can have several concurrent
// runListenStart goroutines in flight from repeated panel re-opens during one
// long ring -- step 3's idempotency check cannot short-circuit them, because
// listen_transcribe_id is not set while step 7 is still polling. Since the
// event-ordering fix makes each goroutine mint its OWN speculative transcribe
// id and pre-write against it, two of them can both pass the List check below
// before either finishes writing, and then either (a) have the second SREM the
// first's ALREADY-LIVE session out of the resolver set, or (b) have a later
// rollback delete DB/Redis state belonging to the first's live, billed
// session. Neither is fixable by write-ordering; both need mutual exclusion.
//
// SCOPE OF THE LOCK, stated so nobody widens it by accident: it serializes ONE
// AIcall's own create-or-reuse attempts. It does not serialize a different
// AIcall reusing a session this one created (that session was already running
// and emitting before this AIcall ever looked -- a narrower, effectively
// unclosable race shared by every revision of this design), and teardown paths
// (clearListenState, stopListenByCallID) do NOT take this lock and can still
// interleave with it.
func (h *aicallHandler) startListenTranscribe(ctx context.Context, c *aicall.AIcall, call *cmcall.Call, callID uuid.UUID) (string, error) {
	// dupFilters -- bound once, referenced by name from BOTH
	// TranscribeV1TranscribeList calls below. Keyed by the typed
	// tmtranscribe.Field, not a bare string: TranscribeV1TranscribeList's
	// actual parameter is `filters map[tmtranscribe.Field]any`, a distinct
	// named type Go does not implicitly convert to (review round 18 finding
	// MEDIUM-2 -- an earlier draft used map[string]any and would not compile).
	dupFilters := map[tmtranscribe.Field]any{
		tmtranscribe.FieldCustomerID:  cmcustomer.IDAIManagerListen,
		tmtranscribe.FieldReferenceID: callID,
		tmtranscribe.FieldStatus:      tmtranscribe.StatusProgressing,
		tmtranscribe.FieldDeleted:     false,
	}

	// this goroutine's own identity for the lock -- independent of
	// newTranscribeID, minted below
	lockToken := h.utilHandler.UUIDCreate()
	lockTTL := time.Duration(config.Get().AIcallListenStartLockTTLSeconds) * time.Second
	releaseTimeout := time.Duration(config.Get().AIcallListenStartLockReleaseTimeoutSeconds) * time.Second

	acquired, err := h.cache.ListenStartLockAcquire(ctx, c.ID, lockToken.String(), lockTTL)
	if err != nil {
		// Ambiguous outcome (review round 17 finding B-7): the SET NX may have
		// landed server-side even though the client saw an error (timeout,
		// connection reset mid-response -- a Redis client cannot always tell
		// "definitely not set" from "set, but the response was lost"). Attempt
		// a best-effort release with our own token so an ambiguous acquire
		// error doesn't strand the lock for the full TTL the same way a
		// genuine crash does. If this second call also fails, the outcome
		// collapses into that same crash case -- accepted, not specially
		// handled further.
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
		_ = h.cache.ListenStartLockRelease(releaseCtx, c.ID, lockToken.String())
		cancel()

		// fail closed, same as every other §5.2 RPC failure -- no transcribe
		// list/start call has been made yet
		return "failed", err
	}
	if !acquired {
		// Another goroutine for this exact AIcall is already inside this
		// sequence (§5.1.1 step 7's own retry, or a second panel-open during
		// the same ring). Let it finish -- this goroutine's job is now
		// redundant, and racing it is exactly the race above.
		return "skipped_start_locked", nil
	}
	defer func() {
		// Detached from ctx's own cancellation/deadline (review round 16
		// finding MEDIUM-2) so a goroutine that reaches its own outer timeout
		// still releases promptly instead of stranding the lock for the full
		// TTL -- combined with the best-effort release on the acquire-error
		// path above, stranding for the full TTL is now reserved for an actual
		// crash (pod loss, process kill -- anywhere this defer itself never
		// runs), not merely an error return from either call.
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
		defer cancel()
		_ = h.cache.ListenStartLockRelease(releaseCtx, c.ID, lockToken.String()) // compare-and-delete, best-effort
	}()

	// RE-VALIDATE THE AICALL, FRESH, UNDER THE LOCK (review round 1 finding
	// HIGH-1). checkListenEligible's step 2 already checked Status/TMDelete --
	// but that ran BEFORE waitForConfbridgeReady, which can block for the whole
	// AIcallListenConfbridgeReadyMaxWaitSeconds budget. Two concrete failures
	// live in that window, and this single re-read is what closes them:
	//
	//   (a) The AIcall is terminated while the goroutine is still polling for
	//       confbridge readiness. ProcessTerminate's own teardown no-ops on
	//       this AIcall, because ListenCallID is still uuid.Nil until the
	//       pre-write below lands -- so without this check the goroutine would
	//       go on to pre-write and start a BILLED STT session on an AIcall that
	//       is already dead. Fully closed here.
	//
	//   (b) Terminate interleaves between the pre-write and
	//       TranscribeV1TranscribeStart. Teardown deliberately does NOT take
	//       this start lock (see the scope note above), so it can clear the
	//       resolver membership and listen_call_id and still leave the
	//       transcribe to be created a moment later -- a live, billed session
	//       with no resolver entry and no listen_call_id, unreapable by either
	//       the transcript-intake path or the hangup path. This check NARROWS
	//       that window to the sub-RPC gap between here and the start call; it
	//       does not eliminate it. That residual is accepted and recorded in
	//       docs/operations.md rather than closed by widening the lock over
	//       teardown.
	//
	// Metered exactly as step 2's own check meters this outcome, and no
	// transcribe RPC is made on this path. The read is the ordinary cache-first
	// AIcallGet for the same reason UpdateListenState's own read is (every
	// ai_aicalls writer refreshes the cache after its write, so a terminate
	// that has landed is visible here).
	fresh, errFresh := h.db.AIcallGet(ctx, c.ID)
	if errFresh != nil {
		// Unknown liveness. Fail closed rather than start a billed session on
		// an AIcall whose state we could not confirm.
		return "failed", errFresh
	}
	if fresh.Status != aicall.StatusProgressing || fresh.TMDelete != nil {
		return "skipped_not_listenable", nil
	}

	// REUSE IS LANGUAGE-TOLERANT ON PURPOSE. Any progressing
	// IDAIManagerListen session on this call is reused regardless of its
	// language string -- starting a second session only because a language
	// string differs would double the STT cost on one call to gain nothing,
	// since the LLM reads whatever language comes out.
	//
	// A session ai-manager does NOT own -- one the customer started under
	// their own customer_id, or an AI summary's under IDAIManager -- is never
	// reused and never touched. The owner scoping makes that structural, not a
	// convention: this list simply cannot see them.
	existing, errList := h.reqHandler.TranscribeV1TranscribeList(ctx, "", 10, dupFilters)
	if errList != nil {
		// fail closed -- an unhandled error here previously read as "no
		// existing session found" (review round 15 finding LOW-4) and could
		// have started a duplicate session
		return "failed", errList
	}
	if reuseID, ok := pickReusableListenTranscribe(existing, callID); ok {
		if _, errUpdate := h.UpdateListenState(ctx, c.ID, callID, reuseID, false); errUpdate != nil {
			return "failed", errUpdate
		}
		return "reused", nil
	}

	language := c.STTLanguage
	if language == "" {
		language = config.Get().AIcallListenDefaultLanguage
	}

	// THE STATE WRITE COMES FIRST, SPECULATIVELY, AGAINST AN ID WE GENERATE
	// (design §5.2.2/§5.2.4, review round 13 finding HIGH-3). Both the DB
	// write and the Redis SADD land before the transcribe exists, so the
	// session cannot emit a single event for a listener nobody has registered.
	// Do not "simplify" this back to writing after creation.
	newTranscribeID := h.utilHandler.UUIDCreate()
	if _, errPre := h.UpdateListenState(ctx, c.ID, callID, newTranscribeID, true); errPre != nil {
		// Fail closed -- no transcribe was created, so nothing is billed. But
		// "nothing to roll back" is NOT true (review round 1 finding
		// MEDIUM-4): UpdateListenState writes the ai_aicalls row BEFORE the
		// Redis SADD and returns the SADD's error, so a SADD failure leaves the
		// row pointing at a transcribe id that will never exist. That state is
		// self-healing eventually (the next listen start overwrites it; the
		// hangup path clears it), but rolling back here is cheap and makes the
		// invariant actually hold instead of merely converging.
		_ = h.rollbackListenState(ctx, c.ID, newTranscribeID)
		return "failed", errPre
	}

	_, err = h.reqHandler.TranscribeV1TranscribeStart(
		ctx,
		newTranscribeID,              // id -- caller-specified, not uuid.Nil; this
		                              //   ordering fix is the one and only reason
		                              //   this design uses that capability
		cmcustomer.IDAIManagerListen, // customerID: the platform sentinel, never the tenant
		call.ActiveflowID,            // the CALL's activeflow, not the AIcall's -- a
		                              //   panel-started contact_case AIcall has
		                              //   ActiveflowID == uuid.Nil
		uuid.Nil,                     // onEndFlowID: no on-end flow for listening
		tmtranscribe.ReferenceTypeCall,
		callID,
		language,
		tmtranscribe.DirectionBoth, // both legs; the speaker tag comes from each segment's own direction
		tmtranscribe.ProviderEmpty, // default provider order
		defaultListenTranscribeStartTimeout,
	)
	switch {
	case err == nil:
		// The created transcribe's id equals newTranscribeID (caller-specified,
		// above), and the DB/Redis state written above already matches it, so
		// there is nothing further to write on this path -- and nothing to
		// capture from the return value.
		return "started", nil

	case isAlreadyProgressing(err):
		// The read-then-create race design §6 already documents: this AIcall's
		// own List above ran just before another writer (a DIFFERENT AIcall on
		// the same call -- the lock only serializes writers sharing this same
		// AIcall) won the create. Re-run the list once and, if a winner is
		// found, rewrite our state to point at it instead of giving up. A
		// blanket rollback-and-fail here would silently drop the
		// reuse-on-conflict behaviour §6 promises (review round 13 finding
		// MEDIUM-3).
		existingRetry, errListRetry := h.reqHandler.TranscribeV1TranscribeList(ctx, "", 10, dupFilters)
		if errListRetry != nil {
			_ = h.rollbackListenState(ctx, c.ID, newTranscribeID) // no winner found either; give up
			return "failed", err
		}
		// A row that does not survive pickReusableListenTranscribe is treated
		// EXACTLY like an empty list: no winner found, roll back, give up.
		// Adopting an unverified row here would be worse than finding none.
		winnerID, okWinner := pickReusableListenTranscribe(existingRetry, callID)
		if !okWinner {
			_ = h.rollbackListenState(ctx, c.ID, newTranscribeID)
			return "failed", err
		}
		if _, errUpdate := h.UpdateListenState(ctx, c.ID, callID, winnerID, false); errUpdate != nil {
			_ = h.rollbackListenState(ctx, c.ID, newTranscribeID)
			return "failed", errUpdate
		}
		// Our own speculative id never got created -- remove only THAT
		// membership, never the winner's (UpdateListenState above already
		// registered us against the winner correctly).
		//
		// This is design §5.2.2's ListenTranscribeAIcallRemove; see the
		// cachehandler listen primitives for why this plan calls it
		// ListenAIcallIDRemove.
		_ = h.cache.ListenAIcallIDRemove(ctx, newTranscribeID, c.ID)
		return "reused", nil

	default:
		// Any other TranscribeV1TranscribeStart failure: give up, undo the
		// speculative write.
		_ = h.rollbackListenState(ctx, c.ID, newTranscribeID)
		return "failed", err
	}
}

// pickReusableListenTranscribe returns the id of the first row in rows that is
// genuinely one of OUR listen sessions on callID, and whether such a row exists.
//
// WHY THIS EXISTS AT ALL (review round 1 security finding MEDIUM-1). This
// package already states the invariant explicitly, in paginateUntilExact's own
// doc comment in tool_insight.go: the RPC's filter map is CALLER-SUPPLIED and
// NOT server-enforced, so every call site MUST independently re-verify the
// ownership fields (CustomerID, ReferenceID, TMDelete) of the rows it gets
// back. tool_insight.go's two `keep` closures do exactly that; the two reuse
// branches in this file were the one place in this feature that trusted the
// filter it sent. Adopting an unverified row means writing a foreign
// transcribe's id onto this AIcall -- and, since the resolver membership is
// keyed by transcribe id, subscribing this tenant's AIcall to another session's
// transcript segments.
//
// The three checked fields are the same three that closure checks, and they are
// checked against constants this process controls (the platform sentinel
// customer id) or values it resolved itself (callID), never against anything
// the listed row supplied about itself.
//
// A mismatch is not an error -- callers treat it identically to "the list came
// back empty," which is the correct, fail-closed reading of "there is no
// session here that we may reuse."
func pickReusableListenTranscribe(rows []tmtranscribe.Transcribe, callID uuid.UUID) (uuid.UUID, bool) {
	for _, row := range rows {
		if row.CustomerID != cmcustomer.IDAIManagerListen {
			logrus.Warnf("Skipping listen transcribe with a foreign customer id. call_id: %s, transcribe_id: %s, transcribe_customer_id: %s", callID, row.ID, row.CustomerID)
			continue
		}
		if row.ReferenceID != callID {
			logrus.Warnf("Skipping listen transcribe with a mismatched reference. call_id: %s, transcribe_id: %s, transcribe_reference_id: %s", callID, row.ID, row.ReferenceID)
			continue
		}
		if row.TMDelete != nil {
			logrus.Warnf("Skipping deleted listen transcribe. call_id: %s, transcribe_id: %s", callID, row.ID)
			continue
		}
		if row.ID == uuid.Nil {
			// A nil id cannot address a session; adopting it would write
			// uuid.Nil into listen_transcribe_id and collide with every other
			// non-listening row on the shared nil key.
			continue
		}

		return row.ID, true
	}

	return uuid.Nil, false
}

// isAlreadyProgressing reports whether err is transcribe-manager's
// "a session for this reference is already live" rejection.
//
// This is the established pattern in this monorepo -- errors.As against
// *cerrors.VoipbinError plus a Reason comparison
// (bin-common-handler/models/errors/voipbin_error.go, used at
// bin-transcribe-manager/pkg/transcribehandler/stop.go and, in exactly this
// one-line-wrapper shape, at bin-storage-manager/pkg/filehandler/signing.go).
// There is no cerrors.IsReason helper in this codebase; an earlier design
// draft invented one (review round 14 finding MEDIUM-1). Do not add one to
// bin-common-handler for this -- it is a local wrapper, named for readability
// in the switch above.
func isAlreadyProgressing(err error) bool {
	var ve *cerrors.VoipbinError
	return errors.As(err, &ve) && ve.Reason == transcribeReasonAlreadyProgressing
}

// rollbackListenState undoes the speculative pre-write for a transcribe that
// was never actually created (design §5.2.2).
//
// A small, dedicated helper -- NOT a reuse of clearListenState, whose contract
// reads listen_transcribe_id off an AIcall struct "already in hand," an
// assumption that does not hold here since UpdateListenState writes through the
// DB rather than mutating the caller's in-memory c. This one takes the known
// transcribeID directly, so it can only ever remove the membership it was told
// about.
func (h *aicallHandler) rollbackListenState(ctx context.Context, aicallID uuid.UUID, transcribeID uuid.UUID) error {
	if errRem := h.cache.ListenAIcallIDRemove(ctx, transcribeID, aicallID); errRem != nil {
		logrus.Warnf("Could not remove the speculative listen resolver membership. aicall_id: %s, transcribe_id: %s, err: %v", aicallID, transcribeID, errRem)
	}

	// Read fresh, then drop only the two listen keys. FieldMetadata is a
	// whole-column write, so building the map from scratch here would silently
	// destroy every other metadata key on the row (prompt snapshots, the
	// auto-audit flag). Same reasoning as UpdateListenState's own copy loop
	// below -- including its choice of the cache-first AIcallGet over
	// AIcallGetSkipCache, for the reason spelled out in that function's doc
	// comment.
	cur, err := h.db.AIcallGet(ctx, aicallID)
	if err != nil {
		return err
	}

	metadata := map[string]any{}
	for k, v := range cur.Metadata {
		metadata[k] = v
	}
	delete(metadata, aicall.MetaKeyListenTranscribeID)
	delete(metadata, aicall.MetaKeyListenOwnsTranscribe)

	// Same targeted, tm_update-bypassing write UpdateListenState uses -- a
	// rollback must not bump tm_update either, or it feeds Send()'s cooldown
	// for a session that never even started.
	return h.db.AIcallUpdateNoTouchTMUpdate(ctx, aicallID, map[aicall.Field]any{
		aicall.FieldListenCallID: uuid.Nil,
		aicall.FieldMetadata:     metadata,
	})
}

// confbridgeReadyResult is why waitForConfbridgeReady stopped polling.
type confbridgeReadyResult int

const (
	confbridgeReady confbridgeReadyResult = iota
	// confbridgeNotReady: the wait budget elapsed without the call ever
	// settling into a live, exactly-2-party confbridge. Covers a stuck
	// 1-party count (still ringing) and a stuck 3+-party count (a genuinely
	// non-standard topology) identically -- see the function's own doc
	// comment for why this design does not try to tell them apart.
	confbridgeNotReady
	// confbridgeCallEnded: the call itself stopped being listenable during
	// the wait (step 6's own check, re-run every poll).
	confbridgeCallEnded
	// confbridgeError: a CallV1CallGet or CallV1ConfbridgeGet RPC failed.
	confbridgeError
)

// waitForConfbridgeReady polls, bounded, until the given call is live inside a
// confbridge with exactly 2 parties -- the shape speakerTag's in=CUSTOMER/
// out=AGENT mapping assumes.
//
// WHY THIS POLLS INSTEAD OF CHECKING ONCE (design §5.1.1 step 7; review round
// 9 finding BLOCKING-1). Call.ConfbridgeID and the confbridge's own
// ChannelCallIDs are only updated once THIS leg's own join channel enters the
// bridge. For the A-leg that happens at queue-forward time, well before the
// agent (the B-leg) answers -- so the confbridge reads as exactly 1 party for
// the whole queue-wait-plus-ring window. ProcessListen can run as early as
// panel-open (a screen-pop UI opening the Case panel at ring time is entirely
// plausible), so a one-shot check would silently never start listening on a
// perfectly ordinary call, with nothing recorded to explain why.
//
// WHY THIS NEVER FAST-FAILS ON A NON-2 PARTY COUNT (review round 10 finding
// HIGH-A). An earlier version of this function gave up immediately once
// call.Status was progressing and the party count was >= 3, reasoning that a
// progressing call was past the point where an extra party could still be
// transient pre-answer noise. That reasoning is unsound: the LISTENED leg
// (the call this function is checking) is progressing for this entire wait --
// queue-wait through agent-ring -- so the fast-fail condition was true the
// instant ANY 3rd party appeared, not just once one had lingered. A
// documented, legitimate flow hits exactly this: a connect action with
// early_media=true and multiple destinations bridges every ringing
// destination before the losing ones hang up
// (bin-call-manager/pkg/confbridgehandler/joined.go explicitly iterates
// looking for a ringing/dialing member -- i.e. this state is anticipated, not
// a bug). The earlier fast-fail would have permanently killed listening on
// such a call, on possibly the only ProcessListen invocation it ever gets. So:
// any non-2 count, at any call status, just keeps polling until the wait budget
// runs out. This means a stably-wrong topology and a merely-slow ring share one
// outcome (confbridgeNotReady/skipped_confbridge_not_ready) and one budget --
// accepted, and documented in design §11 item 13 as a reason to err toward a
// longer AIcallListenConfbridgeReadyMaxWaitSeconds default rather than a
// shorter one.
//
// NOTE ON CONCURRENT CALLERS: repeated panel re-opens during one long ring
// spawn multiple concurrent, independently-bounded calls to this function for
// the SAME AIcall (design §5.1.1 step 7's closing note). Step 3's idempotency
// check cannot short-circuit them, because listen_transcribe_id is not set
// while this function is still polling. That is expected, not a bug, and it is
// bounded -- but it is NOT harmless on its own: what makes it safe is
// startListenTranscribe's per-AIcall lock, which serializes the create-or-reuse
// sequence those goroutines then race into (design §5.2.2). Do not weaken that
// lock on the theory that transcribe-manager's own cross-AIcall dedup guard
// already covers this; review round 14 (HIGH-2) found that reasoning covers
// only cross-AIcall duplication, not this same-AIcall race.
//
// One consequence worth knowing when reading the metric:
// skipped_confbridge_not_ready's raw rate can be inflated by repeated re-opens
// of the same still-ringing call, not just by distinct calls.
//
// IT TAKES ONLY callID, NOT THE AICALL'S CUSTOMER ID, AND THAT IS DELIBERATE.
// Design §5.1.1 step 6 pairs a liveness check with an ownership check
// (call.CustomerID == c.CustomerID), but only the LIVENESS half needs
// re-checking on each poll: a live call's CustomerID is immutable, so
// checkListenEligible's single step-6 ownership check still holds for the whole
// wait. Re-checking it every poll would cost nothing and prove nothing.
//
// IT RETURNS THE LAST OBSERVED PARTY COUNT alongside the outcome (design §6).
// -1 means no confbridge was ever observed (ConfbridgeID stayed uuid.Nil, or
// the very first CallV1CallGet failed), which is a materially different
// diagnosis from "observed, but stuck at 1" and must not be collapsed into 0.
// The caller logs it; it is deliberately NOT a metric label.
func (h *aicallHandler) waitForConfbridgeReady(ctx context.Context, callID uuid.UUID) (confbridgeReadyResult, int) {
	interval := time.Duration(config.Get().AIcallListenConfbridgeReadyPollIntervalSeconds) * time.Second
	deadline := time.Now().Add(time.Duration(config.Get().AIcallListenConfbridgeReadyMaxWaitSeconds) * time.Second)

	lastPartyCount := -1 // -1: no confbridge observed yet, distinct from an observed 0

	for {
		call, err := h.reqHandler.CallV1CallGet(ctx, callID)
		if err != nil {
			return confbridgeError, lastPartyCount
		}
		if call.TMDelete != nil || !isListenableCallStatus(call.Status) {
			return confbridgeCallEnded, lastPartyCount
		}

		if call.ConfbridgeID != uuid.Nil {
			cb, errCb := h.reqHandler.CallV1ConfbridgeGet(ctx, call.ConfbridgeID)
			if errCb != nil {
				return confbridgeError, lastPartyCount
			}
			lastPartyCount = len(cb.ChannelCallIDs)
			if cb.TMDelete == nil && cb.Status == cmconfbridge.StatusProgressing && lastPartyCount == 2 {
				return confbridgeReady, lastPartyCount
			}
			// Not yet a live 2-party bridge: ConfbridgeID unset (not yet
			// joined), a live 1-party bridge (still ringing the other leg),
			// or a transient 3+-party state that has not yet settled (see the
			// HIGH-A note above). All three fall through to the same "keep
			// polling" behaviour below.
		}

		if time.Now().After(deadline) {
			return confbridgeNotReady, lastPartyCount
		}

		select {
		case <-ctx.Done():
			// CATEGORY MISMATCH, CURRENTLY UNREACHABLE -- REVISIT IF THE
			// DEFAULTS CHANGE. This branch fires when the goroutine's OWN outer
			// timeout (AIcallListenEnsureGoroutineTimeoutSeconds) expires, which
			// is not an RPC failure -- and design §5.13 defines
			// confbridgeError/skipped_confbridge_error as the RPC-failure
			// outcome specifically. At the shipped defaults it cannot happen:
			// the poll budget (AIcallListenConfbridgeReadyMaxWaitSeconds, 30s)
			// is strictly less than the goroutine timeout (45s), so the deadline
			// check above always wins and returns confbridgeNotReady first.
			// Task 10 pins that ordering as a standing invariant. If either
			// default ever moves such that the poll budget can outlast the
			// goroutine, this becomes reachable and must be given its own
			// outcome rather than mislabelled as an RPC error.
			return confbridgeError, lastPartyCount
		case <-time.After(interval):
		}
	}
}

// UpdateListenState persists that this AIcall is now listening: one AIcall row
// write plus one Redis set membership.
//
// It takes the AIcall ID, NOT the caller's *aicall.AIcall (design §5.2.4,
// review round 15 finding LOW-7). Both merge rules below turn on "the row's
// CURRENT value," and the calling goroutine's own in-hand copy can be stale --
// it is never mutated by this write, and a concurrent goroutine for the same
// AIcall may have written since. So the current values come from a fresh read
// here, immediately before the merge decision.
//
// THE FRESH READ IS THE ORDINARY CACHE-FIRST h.db.AIcallGet, NOT
// AIcallGetSkipCache. Design §5.2.4's contrast is between this function's own
// read and the CALLER's stale in-hand struct -- not between the cache and the
// database. Nothing in this feature writes this AIcall's cache entry from a
// concurrent path that could race within one request: the only writers are
// UpdateListenState and rollbackListenState themselves, both of which go
// through AIcallUpdateNoTouchTMUpdate and both of which are serialized per
// AIcall by startListenTranscribe's own lock. AIcallGetSkipCache does exist,
// and it is deliberately not used here -- its one justified caller is
// messagehandler's stale-reply guard, where a stale PipecatcallID would cause a
// wrong, irreversible decision.
//
// This is the ONLY ai_aicalls write the feature makes during a listening session
// (one at start, one at stop) -- never per turn. And it goes through
// AIcallUpdateNoTouchTMUpdate specifically, so listen's own bookkeeping never
// bumps tm_update and therefore never feeds Send()'s cooldown.
//
// TWO CALLING CONVENTIONS (design §5.2.4, rev 16), and they are not
// interchangeable. On the REUSE path it is called AFTER the List call found an
// existing session -- there is nothing to pre-write ahead of when reusing
// someone else's already-running session. On the CREATE path it is called
// BEFORE TranscribeV1TranscribeStart, speculatively, against the id that call
// then generates for itself.
func (h *aicallHandler) UpdateListenState(ctx context.Context, aicallID uuid.UUID, callID uuid.UUID, transcribeID uuid.UUID, owns bool) (*aicall.AIcall, error) {
	cur, err := h.db.AIcallGet(ctx, aicallID)
	if err != nil {
		return nil, err
	}

	oldID := listenTranscribeIDFromMetadata(cur)

	// Drop a stale membership first, when step 3's idempotency check found the
	// old session invalid and started a fresh one, or when the create path fell
	// back to reusing a winner. Without this the stale entry survives until its
	// 12h TTL -- harmless functionally, since the old transcribe's events have
	// stopped, but a dangling entry nobody can explain.
	if oldID != uuid.Nil && oldID != transcribeID {
		if errRem := h.cache.ListenAIcallIDRemove(ctx, oldID, aicallID); errRem != nil {
			logrus.Warnf("Could not remove the stale listen resolver membership. aicall_id: %s, err: %v", aicallID, errRem)
		}
	}

	// THE OWNS-MERGE IS SCOPED TO SAME-ID WRITES ONLY (design §5.2.4, review
	// round 12 finding MEDIUM-2, scoped in review round 14 finding HIGH-1).
	//
	// Same id: step 7's bounded retry means the SAME AIcall can have two
	// concurrent runListenStart goroutines racing to write this field on the
	// same row, in an unspecified order. A blind overwrite could persist
	// owns=false for the goroutine that actually started the session, so OR a
	// true in and never let a true already on this row be overwritten.
	//
	// DIFFERENT id: set owns directly, with NO carry-over. This is the branch
	// review round 14 added, and getting it wrong is worse than the race it
	// guards. The create-then-fall-back-to-reuse branch legitimately writes
	// this row against two different transcribe ids in sequence; an
	// unconditional OR would carry a stale owns=true from the abandoned
	// speculative id onto a row now describing a DIFFERENT session this AIcall
	// does not own. Design §5.7.2's stop path then reads `!owns` as false,
	// SKIPS its "never touch it" branch, and tears the session down -- in the
	// two-Cases-one-call scenario, out from under the other Case that is still
	// listening to it.
	if oldID == transcribeID {
		owns = owns || listenOwnsTranscribeFromMetadata(cur)
	}

	// FieldMetadata is a whole-column write, so copy the existing map rather
	// than building a fresh one -- otherwise every other metadata key on the
	// row (prompt snapshots, the auto-audit flag) is silently destroyed.
	metadata := map[string]any{}
	for k, v := range cur.Metadata {
		metadata[k] = v
	}
	metadata[aicall.MetaKeyListenTranscribeID] = transcribeID.String()
	metadata[aicall.MetaKeyListenOwnsTranscribe] = owns

	if errUpdate := h.db.AIcallUpdateNoTouchTMUpdate(ctx, aicallID, map[aicall.Field]any{
		aicall.FieldListenCallID: callID,
		aicall.FieldMetadata:     metadata,
	}); errUpdate != nil {
		return nil, errUpdate
	}

	if errAdd := h.cache.ListenAIcallIDAdd(ctx, transcribeID, aicallID, listenResolverTTL); errAdd != nil {
		return nil, errAdd
	}

	res := *cur
	res.ListenCallID = callID
	res.Metadata = metadata

	return &res, nil
}
