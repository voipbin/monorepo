package request

// V1DataCampaignsFlowsReconcilePost is
// v1 data type request struct for
// /v1/campaigns/flows/reconcile POST
type V1DataCampaignsFlowsReconcilePost struct {
	// RecentIntervalSec is the caller's (bin-schedule-manager's) cron
	// interval, in seconds, for the schedule dispatching this pass. Used
	// by ReconcileOrphanedFlows to compute the RecentSaturated rate-risk
	// signal. A missing or non-positive value falls back to a
	// conservative default inside the handler.
	RecentIntervalSec int64 `json:"recent_interval_sec"`
}
