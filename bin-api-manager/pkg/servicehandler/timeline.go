package servicehandler

import (
	"context"
	"encoding/json"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcall "monorepo/bin-call-manager/models/call"
	commonoutline "monorepo/bin-common-handler/models/outline"
	cfconference "monorepo/bin-conference-manager/models/conference"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmflow "monorepo/bin-flow-manager/models/flow"
	tmevent "monorepo/bin-timeline-manager/models/event"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// TimelineEvent represents a timeline event with converted WebhookMessage data.
type TimelineEvent struct {
	Timestamp string      `json:"timestamp"`
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}

// resourceTypeConfig maps resource types to their configuration.
type resourceTypeConfig struct {
	ServiceName  commonoutline.ServiceName
	EventPattern string
}

// resourceTypeConfigs defines the mapping from resource_type to ServiceName and event pattern.
var resourceTypeConfigs = map[string]resourceTypeConfig{
	"calls":       {ServiceName: commonoutline.ServiceNameCallManager, EventPattern: "call_*"},
	"conferences": {ServiceName: commonoutline.ServiceNameConferenceManager, EventPattern: "conference_*"},
	"flows":       {ServiceName: commonoutline.ServiceNameFlowManager, EventPattern: "flow_*"},
	"activeflows": {ServiceName: commonoutline.ServiceNameFlowManager, EventPattern: "activeflow_*"},
}

// TimelineEventList retrieves timeline events for a resource.
func (h *serviceHandler) TimelineEventList(
	ctx context.Context,
	a *amagent.Agent,
	resourceType string,
	resourceID uuid.UUID,
	pageSize int,
	pageToken string,
) ([]*TimelineEvent, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TimelineEventList",
		"customer_id":   a.CustomerID,
		"username":      a.Username,
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})

	// Validate resource type
	config, ok := resourceTypeConfigs[resourceType]
	if !ok {
		log.Info("Invalid resource type")
		return nil, "", fmt.Errorf("invalid resource type")
	}

	// Validate resource ownership
	customerID, err := h.validateResourceOwnership(ctx, resourceType, resourceID)
	if err != nil {
		log.Infof("Resource validation failed: %v", err)
		return nil, "", err
	}

	// Check permission
	if !h.hasPermission(ctx, a, customerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("Agent has no permission")
		return nil, "", fmt.Errorf("user has no permission")
	}

	// Query timeline events
	req := &tmevent.EventListRequest{
		Publisher:  config.ServiceName,
		ResourceID: resourceID,
		Events:     []string{config.EventPattern},
		PageSize:   pageSize,
		PageToken:  pageToken,
	}

	resp, err := h.reqHandler.TimelineV1EventList(ctx, req)
	if err != nil {
		log.Errorf("Failed to query timeline: %v", err)
		return nil, "", fmt.Errorf("internal error")
	}

	// Convert events to WebhookMessage format
	result := make([]*TimelineEvent, 0, len(resp.Result))
	for _, event := range resp.Result {
		converted, err := h.convertEventToWebhookMessage(resourceType, event)
		if err != nil {
			log.Warnf("Failed to convert event: %v", err)
			continue // Skip failed conversions
		}
		result = append(result, converted)
	}

	return result, resp.NextPageToken, nil
}

// validateResourceOwnership validates the resource exists and returns its customer ID.
func (h *serviceHandler) validateResourceOwnership(ctx context.Context, resourceType string, resourceID uuid.UUID) (uuid.UUID, error) {
	switch resourceType {
	case "calls":
		c, err := h.callGet(ctx, resourceID)
		if err != nil {
			return uuid.Nil, err
		}
		return c.CustomerID, nil

	case "conferences":
		c, err := h.conferenceGet(ctx, resourceID)
		if err != nil {
			return uuid.Nil, err
		}
		return c.CustomerID, nil

	case "flows":
		f, err := h.flowGet(ctx, resourceID)
		if err != nil {
			return uuid.Nil, err
		}
		return f.CustomerID, nil

	case "activeflows":
		af, err := h.activeflowGet(ctx, resourceID)
		if err != nil {
			return uuid.Nil, err
		}
		return af.CustomerID, nil

	default:
		return uuid.Nil, fmt.Errorf("unsupported resource type")
	}
}

// convertEventToWebhookMessage converts a timeline event to WebhookMessage format.
func (h *serviceHandler) convertEventToWebhookMessage(resourceType string, event *tmevent.Event) (*TimelineEvent, error) {
	var data interface{}

	switch resourceType {
	case "calls":
		var call cmcall.Call
		if err := json.Unmarshal(event.Data, &call); err != nil {
			return nil, err
		}
		data = call.ConvertWebhookMessage()

	case "conferences":
		var conf cfconference.Conference
		if err := json.Unmarshal(event.Data, &conf); err != nil {
			return nil, err
		}
		data = conf.ConvertWebhookMessage()

	case "flows":
		var flow fmflow.Flow
		if err := json.Unmarshal(event.Data, &flow); err != nil {
			return nil, err
		}
		data = flow.ConvertWebhookMessage()

	case "activeflows":
		var af fmactiveflow.Activeflow
		if err := json.Unmarshal(event.Data, &af); err != nil {
			return nil, err
		}
		data = af.ConvertWebhookMessage()

	default:
		return nil, fmt.Errorf("unsupported resource type for conversion")
	}

	return &TimelineEvent{
		Timestamp: event.Timestamp.Format("2006-01-02T15:04:05.000Z"),
		EventType: event.EventType,
		Data:      data,
	}, nil
}
