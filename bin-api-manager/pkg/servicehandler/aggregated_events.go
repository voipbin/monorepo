package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	tmevent "monorepo/bin-timeline-manager/models/event"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// AggregatedEventList retrieves aggregated timeline events for an activeflow.
// It resolves the activeflow_id from either activeflow_id or call_id,
// validates ownership/permissions, and queries timeline-manager.
func (h *serviceHandler) AggregatedEventList(
	ctx context.Context,
	a *amagent.Agent,
	activeflowID uuid.UUID,
	callID uuid.UUID,
	pageSize int,
	pageToken string,
) ([]*TimelineEvent, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "AggregatedEventList",
		"customer_id":   a.CustomerID,
		"activeflow_id": activeflowID,
		"call_id":       callID,
	})

	// Validate: exactly one of activeflow_id or call_id must be provided
	if activeflowID == uuid.Nil && callID == uuid.Nil {
		log.Info("Neither activeflow_id nor call_id provided")
		return nil, "", fmt.Errorf("either activeflow_id or call_id is required")
	}
	if activeflowID != uuid.Nil && callID != uuid.Nil {
		log.Info("Both activeflow_id and call_id provided")
		return nil, "", fmt.Errorf("only one of activeflow_id or call_id is allowed")
	}

	// Resolve to activeflow_id
	var resolvedActiveflowID uuid.UUID
	if activeflowID != uuid.Nil {
		// Query by activeflow_id: validate ownership
		af, err := h.activeflowGet(ctx, activeflowID)
		if err != nil {
			log.Infof("Could not get activeflow: %v", err)
			return nil, "", fmt.Errorf("not found")
		}
		log.WithField("activeflow", af).Debugf("Retrieved activeflow info. activeflow_id: %s", af.ID)

		if !h.hasPermission(ctx, a, af.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			log.Info("Agent has no permission")
			return nil, "", fmt.Errorf("user has no permission")
		}
		resolvedActiveflowID = af.ID
	} else {
		// Query by call_id: get call, extract activeflow_id
		c, err := h.callGet(ctx, callID)
		if err != nil {
			log.Infof("Could not get call: %v", err)
			return nil, "", fmt.Errorf("not found")
		}
		log.WithField("call", c).Debugf("Retrieved call info. call_id: %s", c.ID)

		if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
			log.Info("Agent has no permission")
			return nil, "", fmt.Errorf("user has no permission")
		}

		if c.ActiveflowID == uuid.Nil {
			log.Info("Call has no activeflow")
			return nil, "", fmt.Errorf("not found")
		}
		resolvedActiveflowID = c.ActiveflowID
	}

	// Query timeline-manager
	req := &tmevent.AggregatedEventListRequest{
		ActiveflowID: resolvedActiveflowID,
		PageSize:     pageSize,
		PageToken:    pageToken,
	}

	resp, err := h.reqHandler.TimelineV1AggregatedEventList(ctx, req)
	if err != nil {
		log.Errorf("Failed to query aggregated events: %v", err)
		return nil, "", fmt.Errorf("internal error")
	}

	// Return raw events as-is (data is already WebhookMessage JSON from ClickHouse)
	result := make([]*TimelineEvent, 0, len(resp.Result))
	for _, ev := range resp.Result {
		result = append(result, &TimelineEvent{
			Timestamp: ev.Timestamp.Format("2006-01-02T15:04:05.000Z"),
			EventType: ev.EventType,
			Data:      ev.Data,
		})
	}

	return result, resp.NextPageToken, nil
}
