package campaign

// ReconcileResult is the result of a single orphaned-flow reconciliation
// pass (VOIP-1444: pkg/campaignhandler.ReconcileOrphanedFlows). It is a
// domain type that also serves as the wire shape for
// POST /v1/campaigns/flows/reconcile's response — style (A) per the root
// CLAUDE.md's transport-DTO layering rule: json tags on a domain type are
// declarative metadata, not a response.* DTO, and listenhandler marshals
// this type directly with no separate response.* copy.
type ReconcileResult struct {
	// Cleaned is the number of genuinely orphaned flows (owning campaign
	// deleted, flow still live) that were successfully deleted this pass.
	// Each increment is mirrored in campaign_flow_reconcile_cleaned_total.
	Cleaned int `json:"cleaned"`

	// Skipped is the number of candidates that needed no action this pass:
	// the flow was already deleted (TMDelete set by an earlier pass or a
	// successful Delete() call), FlowV1FlowGet returned a typed not-found
	// (the flow row does not exist at all — an equally legitimate clean
	// end state), or the candidate campaign's own TMDelete was
	// unexpectedly nil (the defensive, second-layer guard against a future
	// query regression).
	Skipped int `json:"skipped"`

	// Failed is the number of candidates that could not be resolved this
	// pass: a non-not-found FlowV1FlowGet error, or a FlowV1FlowDelete
	// error on a genuine orphan. Both branches increment the single,
	// shared campaign_flow_reconcile_failed_total counter — there is no
	// "reason" label distinguishing which RPC failed.
	Failed int `json:"failed"`

	// Saturated is true when this pass's batch reached scanLimit rows.
	// Informational only: the window holds more history than a single
	// pass returns, which is expected at scale and implies no action by
	// itself. See docs/operations.md for the full runbook.
	Saturated bool `json:"saturated"`

	// RecentSaturated is true when the count of candidates deleted within
	// the caller-supplied recentIntervalSec already fills scanLimit on its
	// own — the actionable rate-risk signal, distinct from Saturated. It
	// means the actual safety condition (deletions-per-interval <
	// scanLimit) is being violated. See docs/operations.md for the
	// documented remedy (raise scanLimit and/or shorten the schedule's
	// cron interval).
	RecentSaturated bool `json:"recent_saturated"`

	// Partial is true when the pass's own self-imposed
	// context.WithTimeout cut the RPC loop short before every candidate
	// was examined. The counts above reflect only what was processed
	// before the bail-out; a Partial pass is never silently recorded as a
	// full success with no trace it was incomplete.
	Partial bool `json:"partial"`
}
