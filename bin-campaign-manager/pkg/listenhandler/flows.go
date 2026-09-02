package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-campaign-manager/pkg/listenhandler/models/request"
)

// v1CampaignsFlowsReconcilePost handles /v1/campaigns/flows/reconcile POST
// request (VOIP-1444). Internal-only route, dispatched by the
// bin-schedule-manager "campaign-flow-reconcile" schedule (or a manual
// POST /v1/schedules/{id}/execute) — not exposed through bin-api-manager.
//
// The handler always returns HTTP 200 for a pass that ran, even if
// Failed > 0 or Partial == true (individual row failures and a self-timeout
// cutoff are data, not a pass-level error). A non-nil err from the handler
// (the initial query itself failing) is the only case that produces a
// non-2xx response, via the default error handling in processRequest.
func (h *listenHandler) v1CampaignsFlowsReconcilePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaignsFlowsReconcilePost",
		"request": m,
	})
	log.Debug("Received request.")

	var req request.V1DataCampaignsFlowsReconcilePost
	if len(m.Data) > 0 {
		if err := json.Unmarshal(m.Data, &req); err != nil {
			log.Errorf("Could not marshal the data. err: %v", err)
			return nil, err
		}
	}

	result, err := h.campaignHandler.ReconcileOrphanedFlows(ctx, req.RecentIntervalSec)
	if err != nil {
		log.Errorf("Could not reconcile orphaned flows. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(result)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
