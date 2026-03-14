package servicehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	amagent "monorepo/bin-agent-manager/models/agent"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amsummary "monorepo/bin-ai-manager/models/summary"
	cmcall "monorepo/bin-call-manager/models/call"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cpcampaigncall "monorepo/bin-campaign-manager/models/campaigncall"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	tmevent "monorepo/bin-timeline-manager/models/event"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

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

	// Convert events to WebhookMessage format to strip internal fields
	result := make([]*TimelineEvent, 0, len(resp.Result))
	for _, ev := range resp.Result {
		converted, err := convertAggregatedEventData(ev)
		if err != nil {
			log.Warnf("Failed to convert event. event_type: %s, err: %v", ev.EventType, err)
			continue // Skip failed conversions
		}
		result = append(result, converted)
	}

	return result, resp.NextPageToken, nil
}

// convertAggregatedEventData converts a timeline event's raw data to WebhookMessage format.
// Events are matched by event_type prefix to determine the correct internal struct and conversion.
// Events with unknown prefixes are skipped (returned as error) to prevent leaking internal fields.
func convertAggregatedEventData(event *tmevent.Event) (*TimelineEvent, error) {
	var data any

	switch {
	case strings.HasPrefix(event.EventType, "call_"):
		var v cmcall.Call
		if err := json.Unmarshal(event.Data, &v); err != nil {
			return nil, err
		}
		data = v.ConvertWebhookMessage()

	case strings.HasPrefix(event.EventType, "recording_"):
		var v cmrecording.Recording
		if err := json.Unmarshal(event.Data, &v); err != nil {
			return nil, err
		}
		data = v.ConvertWebhookMessage()

	case strings.HasPrefix(event.EventType, "activeflow_"):
		var v fmactiveflow.Activeflow
		if err := json.Unmarshal(event.Data, &v); err != nil {
			return nil, err
		}
		data = v.ConvertWebhookMessage()

	case strings.HasPrefix(event.EventType, "aicall_"):
		var v amaicall.AIcall
		if err := json.Unmarshal(event.Data, &v); err != nil {
			return nil, err
		}
		data = v.ConvertWebhookMessage()

	case strings.HasPrefix(event.EventType, "summary_"):
		var v amsummary.Summary
		if err := json.Unmarshal(event.Data, &v); err != nil {
			return nil, err
		}
		data = v.ConvertWebhookMessage()

	case strings.HasPrefix(event.EventType, "transcribe_"):
		var v tmtranscribe.Transcribe
		if err := json.Unmarshal(event.Data, &v); err != nil {
			return nil, err
		}
		data = v.ConvertWebhookMessage()

	case strings.HasPrefix(event.EventType, "conferencecall_"):
		var v cfconferencecall.Conferencecall
		if err := json.Unmarshal(event.Data, &v); err != nil {
			return nil, err
		}
		data = v.ConvertWebhookMessage()

	case strings.HasPrefix(event.EventType, "campaigncall_"):
		var v cpcampaigncall.Campaigncall
		if err := json.Unmarshal(event.Data, &v); err != nil {
			return nil, err
		}
		data = v.ConvertWebhookMessage()

	default:
		return nil, fmt.Errorf("unsupported event type: %s", event.EventType)
	}

	return &TimelineEvent{
		Timestamp: event.Timestamp.Format("2006-01-02T15:04:05.000Z"),
		EventType: event.EventType,
		Data:      data,
	}, nil
}
