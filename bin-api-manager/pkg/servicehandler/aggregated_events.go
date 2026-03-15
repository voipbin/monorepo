package servicehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	ammessage "monorepo/bin-ai-manager/models/message"
	amsummary "monorepo/bin-ai-manager/models/summary"
	amteam "monorepo/bin-ai-manager/models/team"
	bmaccount "monorepo/bin-billing-manager/models/account"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cpcampaign "monorepo/bin-campaign-manager/models/campaign"
	cpcampaigncall "monorepo/bin-campaign-manager/models/campaigncall"
	cpoutplan "monorepo/bin-campaign-manager/models/outplan"
	cfconference "monorepo/bin-conference-manager/models/conference"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"
	ctcontact "monorepo/bin-contact-manager/models/contact"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	cuaccesskey "monorepo/bin-customer-manager/models/accesskey"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	ememail "monorepo/bin-email-manager/models/email"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmflow "monorepo/bin-flow-manager/models/flow"
	mmmessage "monorepo/bin-message-manager/models/message"
	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"
	nmnumber "monorepo/bin-number-manager/models/number"
	omodial "monorepo/bin-outdial-manager/models/outdial"
	omtarget "monorepo/bin-outdial-manager/models/outdialtarget"
	qmqueue "monorepo/bin-queue-manager/models/queue"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"
	rmextension "monorepo/bin-registrar-manager/models/extension"
	rmextensiondirect "monorepo/bin-registrar-manager/models/extensiondirect"
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"
	rtprovider "monorepo/bin-route-manager/models/provider"
	rtroute "monorepo/bin-route-manager/models/route"
	smaccount "monorepo/bin-storage-manager/models/account"
	smfile "monorepo/bin-storage-manager/models/file"
	tgtag "monorepo/bin-tag-manager/models/tag"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkparticipant "monorepo/bin-talk-manager/models/participant"
	tmevent "monorepo/bin-timeline-manager/models/event"
	tmstreaming "monorepo/bin-transcribe-manager/models/streaming"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
	tftransfer "monorepo/bin-transfer-manager/models/transfer"
	ttspeaking "monorepo/bin-tts-manager/models/speaking"

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

// eventConverter unmarshals event data and returns its WebhookMessage representation.
type eventConverter func(data json.RawMessage) (any, error)

// newEventConverter creates a converter that unmarshals JSON into type T
// and applies the given convert function to produce the WebhookMessage.
func newEventConverter[T any](convert func(*T) any) eventConverter {
	return func(data json.RawMessage) (any, error) {
		var v T
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return convert(&v), nil
	}
}

// eventConverters maps event_type prefixes to their conversion functions.
// To support a new event type, add one line here.
// NOTE: When prefixes overlap (e.g. "conversation_" vs "conversation_message_"),
// convertAggregatedEventData uses longest-prefix-first matching to select the correct converter.
var eventConverters = map[string]eventConverter{
	"Account_":              newEventConverter(func(v *smaccount.Account) any { return v.ConvertWebhookMessage() }), // capital A: storage-manager uses "Account_created" etc.
	"accesskey_":            newEventConverter(func(v *cuaccesskey.Accesskey) any { return v.ConvertWebhookMessage() }),
	"account_":              newEventConverter(func(v *bmaccount.Account) any { return v.ConvertWebhookMessage() }),
	"activeflow_":           newEventConverter(func(v *fmactiveflow.Activeflow) any { return v.ConvertWebhookMessage() }),
	"agent_":                newEventConverter(func(v *amagent.Agent) any { return v.ConvertWebhookMessage() }),
	"ai_":                   newEventConverter(func(v *amai.AI) any { return v.ConvertWebhookMessage() }),
	"aicall_":               newEventConverter(func(v *amaicall.AIcall) any { return v.ConvertWebhookMessage() }),
	"aimessage_":            newEventConverter(func(v *ammessage.Message) any { return v.ConvertWebhookMessage() }),
	"availablenumber_":      newEventConverter(func(v *nmavailablenumber.AvailableNumber) any { return v.ConvertWebhookMessage() }),
	"billing_":              newEventConverter(func(v *bmbilling.Billing) any { return v.ConvertWebhookMessage() }),
	"call_":                 newEventConverter(func(v *cmcall.Call) any { return v.ConvertWebhookMessage() }),
	"campaign_":             newEventConverter(func(v *cpcampaign.Campaign) any { return v.ConvertWebhookMessage() }),
	"campaigncall_":         newEventConverter(func(v *cpcampaigncall.Campaigncall) any { return v.ConvertWebhookMessage() }),
	"chat_":                 newEventConverter(func(v *tkchat.Chat) any { return v.ConvertWebhookMessage() }),
	"conference_":           newEventConverter(func(v *cfconference.Conference) any { return v.ConvertWebhookMessage() }),
	"conferencecall_":       newEventConverter(func(v *cfconferencecall.Conferencecall) any { return v.ConvertWebhookMessage() }),
	"contact_":              newEventConverter(func(v *ctcontact.Contact) any { return v.ConvertWebhookMessage() }),
	"conversation_":         newEventConverter(func(v *cvconversation.Conversation) any { return v.ConvertWebhookMessage() }),
	"conversation_message_": newEventConverter(func(v *cvmessage.Message) any { return v.ConvertWebhookMessage() }),
	"customer_":             newEventConverter(func(v *cucustomer.Customer) any { return v.ConvertWebhookMessage() }),
	"email_":                newEventConverter(func(v *ememail.Email) any { return v.ConvertWebhookMessage() }),
	"extension_":            newEventConverter(func(v *rmextension.Extension) any { return v.ConvertWebhookMessage() }),
	"extension_direct_":     newEventConverter(func(v *rmextensiondirect.ExtensionDirect) any { return v.ConvertWebhookMessage() }),
	"file_":                 newEventConverter(func(v *smfile.File) any { return v.ConvertWebhookMessage() }),
	"flow_":                 newEventConverter(func(v *fmflow.Flow) any { return v.ConvertWebhookMessage() }),
	"groupcall_":            newEventConverter(func(v *cmgroupcall.Groupcall) any { return v.ConvertWebhookMessage() }),
	"message_":              newEventConverter(func(v *mmmessage.Message) any { return v.ConvertWebhookMessage() }),
	"number_":               newEventConverter(func(v *nmnumber.Number) any { return v.ConvertWebhookMessage() }),
	"outdial_":              newEventConverter(func(v *omodial.Outdial) any { return v.ConvertWebhookMessage() }),
	"outdialtarget_":        newEventConverter(func(v *omtarget.OutdialTarget) any { return v.ConvertWebhookMessage() }),
	"outplan_":              newEventConverter(func(v *cpoutplan.Outplan) any { return v.ConvertWebhookMessage() }),
	"participant_":          newEventConverter(func(v *tkparticipant.Participant) any { return v.ConvertWebhookMessage() }),
	"provider_":             newEventConverter(func(v *rtprovider.Provider) any { return v.ConvertWebhookMessage() }),
	"queue_":                newEventConverter(func(v *qmqueue.Queue) any { return v.ConvertWebhookMessage() }),
	"queuecall_":            newEventConverter(func(v *qmqueuecall.Queuecall) any { return v.ConvertWebhookMessage() }),
	"recording_":            newEventConverter(func(v *cmrecording.Recording) any { return v.ConvertWebhookMessage() }),
	"route_":                newEventConverter(func(v *rtroute.Route) any { return v.ConvertWebhookMessage() }),
	"speaking_":             newEventConverter(func(v *ttspeaking.Speaking) any { return v.ConvertWebhookMessage() }),
	// NOTE: "streaming_" events (streaming_started/stopped) carry a Streaming struct which lacks
	// ConvertWebhookMessage, so they cannot be converted. Only "transcribe_speech_" events use Speech.
	"summary_":              newEventConverter(func(v *amsummary.Summary) any { return v.ConvertWebhookMessage() }),
	"tag_":                  newEventConverter(func(v *tgtag.Tag) any { return v.ConvertWebhookMessage() }),
	"team_":                 newEventConverter(func(v *amteam.Team) any { return v.ConvertWebhookMessage() }),
	"transcript_":           newEventConverter(func(v *tmtranscript.Transcript) any { return v.ConvertWebhookMessage() }),
	"transcribe_":           newEventConverter(func(v *tmtranscribe.Transcribe) any { return v.ConvertWebhookMessage() }),
	"transcribe_speech_":    newEventConverter(func(v *tmstreaming.Speech) any { return v.ConvertWebhookMessage() }),
	"transfer_":             newEventConverter(func(v *tftransfer.Transfer) any { return v.ConvertWebhookMessage() }),
	"trunk_":                newEventConverter(func(v *rmtrunk.Trunk) any { return v.ConvertWebhookMessage() }),
}

// convertAggregatedEventData converts a timeline event's raw data to WebhookMessage format.
// Events are matched by the longest matching event_type prefix using the eventConverters registry.
// Longest-prefix-first matching ensures overlapping prefixes (e.g. "conversation_" vs
// "conversation_message_") resolve to the most specific converter.
// Events with unknown prefixes are skipped (returned as error) to prevent leaking internal fields.
func convertAggregatedEventData(event *tmevent.Event) (*TimelineEvent, error) {
	var bestPrefix string
	var bestConverter eventConverter
	for prefix, converter := range eventConverters {
		if strings.HasPrefix(event.EventType, prefix) && len(prefix) > len(bestPrefix) {
			bestPrefix = prefix
			bestConverter = converter
		}
	}
	if bestConverter == nil {
		return nil, fmt.Errorf("unsupported event type: %s", event.EventType)
	}

	data, err := bestConverter(event.Data)
	if err != nil {
		return nil, err
	}
	return &TimelineEvent{
		Timestamp: event.Timestamp.Format("2006-01-02T15:04:05.000Z"),
		EventType: event.EventType,
		Data:      data,
	}, nil
}
