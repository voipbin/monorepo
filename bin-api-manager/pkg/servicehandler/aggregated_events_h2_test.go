package servicehandler

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

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
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"
	rtprovider "monorepo/bin-route-manager/models/provider"
	rtroute "monorepo/bin-route-manager/models/route"
	smaccount "monorepo/bin-storage-manager/models/account"
	smfile "monorepo/bin-storage-manager/models/file"
	tgtag "monorepo/bin-tag-manager/models/tag"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkparticipant "monorepo/bin-talk-manager/models/participant"
	tmstreaming "monorepo/bin-transcribe-manager/models/streaming"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
	tftransfer "monorepo/bin-transfer-manager/models/transfer"
	ttspeaking "monorepo/bin-tts-manager/models/speaking"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	tmevent "monorepo/bin-timeline-manager/models/event"
)

func Test_convertAggregatedEventData_h2_allPrefixes(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 678000000, time.UTC)
	tests := []struct {
		prefix     string
		expectType reflect.Type
	}{
		{prefix: "Account_", expectType: reflect.TypeOf((&smaccount.Account{}).ConvertWebhookMessage())},
		{prefix: "accesskey_", expectType: reflect.TypeOf((&cuaccesskey.Accesskey{}).ConvertWebhookMessage())},
		{prefix: "account_", expectType: reflect.TypeOf((&bmaccount.Account{}).ConvertWebhookMessage())},
		{prefix: "activeflow_", expectType: reflect.TypeOf((&fmactiveflow.Activeflow{}).ConvertWebhookMessage())},
		{prefix: "agent_", expectType: reflect.TypeOf((&amagent.Agent{}).ConvertWebhookMessage())},
		{prefix: "ai_", expectType: reflect.TypeOf((&amai.AI{}).ConvertWebhookMessage())},
		{prefix: "aicall_", expectType: reflect.TypeOf((&amaicall.AIcall{}).ConvertWebhookMessage())},
		{prefix: "aimessage_", expectType: reflect.TypeOf((&ammessage.Message{}).ConvertWebhookMessage())},
		{prefix: "availablenumber_", expectType: reflect.TypeOf((&nmavailablenumber.AvailableNumber{}).ConvertWebhookMessage())},
		{prefix: "billing_", expectType: reflect.TypeOf((&bmbilling.Billing{}).ConvertWebhookMessage())},
		{prefix: "call_", expectType: reflect.TypeOf((&cmcall.Call{}).ConvertWebhookMessage())},
		{prefix: "campaign_", expectType: reflect.TypeOf((&cpcampaign.Campaign{}).ConvertWebhookMessage())},
		{prefix: "campaigncall_", expectType: reflect.TypeOf((&cpcampaigncall.Campaigncall{}).ConvertWebhookMessage())},
		{prefix: "chat_", expectType: reflect.TypeOf((&tkchat.Chat{}).ConvertWebhookMessage())},
		{prefix: "conference_", expectType: reflect.TypeOf((&cfconference.Conference{}).ConvertWebhookMessage())},
		{prefix: "conferencecall_", expectType: reflect.TypeOf((&cfconferencecall.Conferencecall{}).ConvertWebhookMessage())},
		{prefix: "contact_", expectType: reflect.TypeOf((&ctcontact.Contact{}).ConvertWebhookMessage())},
		{prefix: "conversation_", expectType: reflect.TypeOf((&cvconversation.Conversation{}).ConvertWebhookMessage())},
		{prefix: "conversation_message_", expectType: reflect.TypeOf((&cvmessage.Message{}).ConvertWebhookMessage())},
		{prefix: "customer_", expectType: reflect.TypeOf((&cucustomer.Customer{}).ConvertWebhookMessage())},
		{prefix: "email_", expectType: reflect.TypeOf((&ememail.Email{}).ConvertWebhookMessage())},
		{prefix: "extension_", expectType: reflect.TypeOf((&rmextension.Extension{}).ConvertWebhookMessage())},
		{prefix: "file_", expectType: reflect.TypeOf((&smfile.File{}).ConvertWebhookMessage())},
		{prefix: "flow_", expectType: reflect.TypeOf((&fmflow.Flow{}).ConvertWebhookMessage())},
		{prefix: "groupcall_", expectType: reflect.TypeOf((&cmgroupcall.Groupcall{}).ConvertWebhookMessage())},
		{prefix: "message_", expectType: reflect.TypeOf((&mmmessage.Message{}).ConvertWebhookMessage())},
		{prefix: "number_", expectType: reflect.TypeOf((&nmnumber.Number{}).ConvertWebhookMessage())},
		{prefix: "outdial_", expectType: reflect.TypeOf((&omodial.Outdial{}).ConvertWebhookMessage())},
		{prefix: "outdialtarget_", expectType: reflect.TypeOf((&omtarget.OutdialTarget{}).ConvertWebhookMessage())},
		{prefix: "outplan_", expectType: reflect.TypeOf((&cpoutplan.Outplan{}).ConvertWebhookMessage())},
		{prefix: "participant_", expectType: reflect.TypeOf((&tkparticipant.Participant{}).ConvertWebhookMessage())},
		{prefix: "provider_", expectType: reflect.TypeOf((&rtprovider.Provider{}).ConvertWebhookMessage())},
		{prefix: "queue_", expectType: reflect.TypeOf((&qmqueue.Queue{}).ConvertWebhookMessage())},
		{prefix: "queuecall_", expectType: reflect.TypeOf((&qmqueuecall.Queuecall{}).ConvertWebhookMessage())},
		{prefix: "recording_", expectType: reflect.TypeOf((&cmrecording.Recording{}).ConvertWebhookMessage())},
		{prefix: "route_", expectType: reflect.TypeOf((&rtroute.Route{}).ConvertWebhookMessage())},
		{prefix: "speaking_", expectType: reflect.TypeOf((&ttspeaking.Speaking{}).ConvertWebhookMessage())},
		{prefix: "summary_", expectType: reflect.TypeOf((&amsummary.Summary{}).ConvertWebhookMessage())},
		{prefix: "tag_", expectType: reflect.TypeOf((&tgtag.Tag{}).ConvertWebhookMessage())},
		{prefix: "team_", expectType: reflect.TypeOf((&amteam.Team{}).ConvertWebhookMessage())},
		{prefix: "transcript_", expectType: reflect.TypeOf((&tmtranscript.Transcript{}).ConvertWebhookMessage())},
		{prefix: "transcribe_", expectType: reflect.TypeOf((&tmtranscribe.Transcribe{}).ConvertWebhookMessage())},
		{prefix: "transcribe_speech_", expectType: reflect.TypeOf((&tmstreaming.Speech{}).ConvertWebhookMessage())},
		{prefix: "transfer_", expectType: reflect.TypeOf((&tftransfer.Transfer{}).ConvertWebhookMessage())},
		{prefix: "trunk_", expectType: reflect.TypeOf((&rmtrunk.Trunk{}).ConvertWebhookMessage())},
	}

	if len(tests) != len(eventConverters) {
		t.Fatalf("Wrong match. the table covers %d prefixes but eventConverters has %d", len(tests), len(eventConverters))
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			eventType := tt.prefix + "created"
			res, err := convertAggregatedEventData(&tmevent.Event{
				EventType: eventType,
				Timestamp: ts,
				Data:      json.RawMessage(`{}`),
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.EventType != eventType {
				t.Errorf("Wrong event type. expect: %s, got: %s", eventType, res.EventType)
			}
			if res.Timestamp != "2026-01-02T03:04:05.678Z" {
				t.Errorf("Wrong timestamp. expect: 2026-01-02T03:04:05.678Z, got: %s", res.Timestamp)
			}
			if reflect.TypeOf(res.Data) != tt.expectType {
				t.Errorf("Wrong data type. expect: %v, got: %v", tt.expectType, reflect.TypeOf(res.Data))
			}
		})
	}
}

func Test_convertAggregatedEventData_h2_errors(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		data      string
		expectErr error
	}{
		{name: "unsupported event type", eventType: "nosuchresource_created", data: `{}`, expectErr: serviceerrors.ErrInvalidArgument},
		{name: "empty event type", eventType: "", data: `{}`, expectErr: serviceerrors.ErrInvalidArgument},
		{name: "malformed json", eventType: "call_created", data: `{not json`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := convertAggregatedEventData(&tmevent.Event{EventType: tt.eventType, Data: json.RawMessage(tt.data)})
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil (%v)", res)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if tt.expectErr != nil && !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_convertAggregatedEventData_h2_longestPrefix(t *testing.T) {
	// conversation_message_* must not be handled by the shorter conversation_ converter.
	tests := []struct {
		eventType string
		expect    string
	}{
		{"conversation_message_created", "monorepo/bin-conversation-manager/models/message"},
		{"transcribe_speech_started", "monorepo/bin-transcribe-manager/models/streaming"},
	}
	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			res, err := convertAggregatedEventData(&tmevent.Event{EventType: tt.eventType, Data: json.RawMessage(`{}`)})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if got := reflect.TypeOf(res.Data).Elem().PkgPath(); got != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, got)
			}
		})
	}
}
