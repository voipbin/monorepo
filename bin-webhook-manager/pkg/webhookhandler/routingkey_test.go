package webhookhandler

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func TestParseWebhookOwnerData(t *testing.T) {
	data := json.RawMessage(`{"customer_id":"a1b2c3d4-0000-0000-0000-000000000001","owner_id":"98765432-0000-0000-0000-000000000002","owner_type":"agent","id":"12345678-0000-0000-0000-000000000003"}`)

	d, err := parseWebhookOwnerData(data)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if d.CustomerID.String() != "a1b2c3d4-0000-0000-0000-000000000001" {
		t.Errorf("Expected CustomerID 'a1b2c3d4-0000-0000-0000-000000000001', got '%s'", d.CustomerID.String())
	}
	if d.OwnerID.String() != "98765432-0000-0000-0000-000000000002" {
		t.Errorf("Expected OwnerID '98765432-0000-0000-0000-000000000002', got '%s'", d.OwnerID.String())
	}
}

// TestParseWebhookOwnerData_ChatFields verifies AIcallID/ChatID are extracted correctly when
// present -- these fields were untested in the initial commit (code-quality review gap).
func TestParseWebhookOwnerData_ChatFields(t *testing.T) {
	data := json.RawMessage(`{"customer_id":"a1b2c3d4-0000-0000-0000-000000000001","aicall_id":"a1ca11ff-0000-0000-0000-000000000004","chat_id":"c4a70000-0000-0000-0000-000000000005"}`)

	d, err := parseWebhookOwnerData(data)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if d.AIcallID.String() != "a1ca11ff-0000-0000-0000-000000000004" {
		t.Errorf("Expected AIcallID 'a1ca11ff-0000-0000-0000-000000000004', got '%s'", d.AIcallID.String())
	}
	if d.ChatID.String() != "c4a70000-0000-0000-0000-000000000005" {
		t.Errorf("Expected ChatID 'c4a70000-0000-0000-0000-000000000005', got '%s'", d.ChatID.String())
	}
}

// TestParseWebhookOwnerData_AbsentOptionalFields verifies the "best-effort tolerance for
// absent fields" claim in parseWebhookOwnerData's doc comment: a payload missing owner_id/
// aicall_id/chat_id entirely must NOT error, and those fields must stay at their zero value
// (uuid.Nil) rather than causing an unmarshal failure.
func TestParseWebhookOwnerData_AbsentOptionalFields(t *testing.T) {
	data := json.RawMessage(`{"customer_id":"a1b2c3d4-0000-0000-0000-000000000001"}`)

	d, err := parseWebhookOwnerData(data)
	if err != nil {
		t.Fatalf("Expected no error for a payload with absent optional fields, got %v", err)
	}

	if d.CustomerID.String() != "a1b2c3d4-0000-0000-0000-000000000001" {
		t.Errorf("Expected CustomerID to still be extracted, got '%s'", d.CustomerID.String())
	}
	if d.OwnerID != uuid.Nil {
		t.Errorf("Expected OwnerID to be zero-value (uuid.Nil) when absent, got '%s'", d.OwnerID.String())
	}
	if d.AIcallID != uuid.Nil {
		t.Errorf("Expected AIcallID to be zero-value (uuid.Nil) when absent, got '%s'", d.AIcallID.String())
	}
	if d.ChatID != uuid.Nil {
		t.Errorf("Expected ChatID to be zero-value (uuid.Nil) when absent, got '%s'", d.ChatID.String())
	}
}

// TestParseWebhookOwnerData_MalformedJSON verifies the error path: malformed JSON must return
// a non-nil error (the primary behavior parseWebhookOwnerData's signature promises), rather
// than silently succeeding with a zero-value struct.
func TestParseWebhookOwnerData_MalformedJSON(t *testing.T) {
	data := json.RawMessage(`{"customer_id": this is not valid json`)

	d, err := parseWebhookOwnerData(data)
	if err == nil {
		t.Fatal("Expected an error for malformed JSON, got nil")
	}
	if d != nil {
		t.Errorf("Expected nil result on error, got %+v", d)
	}
}

func TestCreateRoutingKeys_CustomerOnly(t *testing.T) {
	d := &webhookOwnerData{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
			ID:         uuid.FromStringOrNil("12345678-0000-0000-0000-000000000003"),
		},
	}

	keys := createRoutingKeys(d, "call", "call_updated")

	expected := []string{"customer_id.a1b2c3d4-0000-0000-0000-000000000001.call.call_updated.12345678-0000-0000-0000-000000000003"}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Expected %v, got %v", expected, keys)
	}
}

func TestCreateRoutingKeys_CustomerAndOwner(t *testing.T) {
	d := &webhookOwnerData{
		Identity: commonidentity.Identity{
			CustomerID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
			ID:         uuid.FromStringOrNil("12345678-0000-0000-0000-000000000003"),
		},
		Owner: commonidentity.Owner{
			OwnerID: uuid.FromStringOrNil("98765432-0000-0000-0000-000000000002"),
		},
	}

	keys := createRoutingKeys(d, "queue", "queue_updated")

	expected := []string{
		"customer_id.a1b2c3d4-0000-0000-0000-000000000001.queue.queue_updated.12345678-0000-0000-0000-000000000003",
		"agent_id.98765432-0000-0000-0000-000000000002.queue.queue_updated.12345678-0000-0000-0000-000000000003",
	}
	sort.Strings(keys)
	sort.Strings(expected)
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Expected %v, got %v", expected, keys)
	}
}

// TestCreateRoutingKeys_NoCustomerNoOwner verifies that when neither CustomerID nor OwnerID
// is present (uuid.Nil), createRoutingKeys returns an empty slice rather than a key with a
// nil UUID string.
func TestCreateRoutingKeys_NoCustomerNoOwner(t *testing.T) {
	d := &webhookOwnerData{}

	keys := createRoutingKeys(d, "call", "call_updated")

	if len(keys) != 0 {
		t.Errorf("Expected empty keys, got %v", keys)
	}
}

// containsString is a small helper to avoid pulling testify's require into this file, matching
// the existing plain if/t.Errorf convention used throughout this package.
func containsString(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

// TestCreateRoutingKeysForChat verifies the fan-out path.
func TestCreateRoutingKeysForChat(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &webhookHandler{reqHandler: mockReq}

	// NOTE: the plan's original literal "chat0000-0000-0000-0000-000000000001" contains
	// non-hex characters ('h', 't') and silently parses to uuid.Nil via FromStringOrNil,
	// which would make this test vacuously check the "both Nil" empty-slice path instead of
	// the fan-out path it's meant to exercise. Using a valid hex UUID here instead.
	chatID := uuid.FromStringOrNil("ca740000-0000-0000-0000-000000000001")
	d := &webhookOwnerData{
		Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001")},
		ChatID:   chatID,
	}

	// NOTE: the plan's original participant OwnerID literals ("p1000000-...", "p2000000-...")
	// contain the non-hex character 'p' and silently parse to uuid.Nil via FromStringOrNil.
	// Since d.OwnerID is also uuid.Nil here (not set on webhookOwnerData), that would make the
	// participant-skip check "p.OwnerID == d.OwnerID" incorrectly true for every participant,
	// silently dropping all participant keys. Using valid hex UUIDs instead.
	mockReq.EXPECT().TalkV1ParticipantList(gomock.Any(), chatID).Return([]*tkparticipant.Participant{
		{Owner: commonidentity.Owner{OwnerID: uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")}},
		{Owner: commonidentity.Owner{OwnerID: uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000002")}},
	}, nil)

	keys := h.createRoutingKeysForChat(context.Background(), d, "chatmessage_created")

	expectedKeys := []string{
		"customer_id.a1b2c3d4-0000-0000-0000-000000000001.talk.chatmessage_created." + chatID.String(),
		"agent_id.a1000000-0000-0000-0000-000000000001.talk.chatmessage_created." + chatID.String(),
		"agent_id.a2000000-0000-0000-0000-000000000002.talk.chatmessage_created." + chatID.String(),
	}
	for _, expected := range expectedKeys {
		if !containsString(keys, expected) {
			t.Errorf("Expected keys to contain %q, got %v", expected, keys)
		}
	}
}

func Test_createRoutingKeysForChat_cases(t *testing.T) {
	customerID := uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-0000000000c1")
	ownerID := uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-0000000000a1")
	otherID := uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-0000000000a2")
	chatID := uuid.FromStringOrNil("ca740000-0000-0000-0000-000000000011")
	identityID := uuid.FromStringOrNil("ca740000-0000-0000-0000-000000000022")
	if customerID == uuid.Nil || ownerID == uuid.Nil || otherID == uuid.Nil || chatID == uuid.Nil || identityID == uuid.Nil {
		t.Fatalf("invalid test uuid")
	}

	tests := []struct {
		name string

		data *webhookOwnerData

		// expectListChatID is the chat id the participant list is requested for; Nil means no RPC.
		expectListChatID uuid.UUID
		responseList     []*tkparticipant.Participant
		responseErr      error

		expect []string
	}{
		{
			name:   "no chat id and no identity id returns an empty non-nil slice without RPC",
			data:   &webhookOwnerData{Identity: commonidentity.Identity{CustomerID: customerID}},
			expect: []string{},
		},
		{
			name: "missing chat id falls back to the identity id",
			data: &webhookOwnerData{
				Identity: commonidentity.Identity{ID: identityID, CustomerID: customerID},
			},
			expectListChatID: identityID,
			responseList:     []*tkparticipant.Participant{},
			expect: []string{
				"customer_id." + customerID.String() + ".talk.chat_created." + identityID.String(),
			},
		},
		{
			name: "participant list failure keeps the customer and owner keys",
			data: &webhookOwnerData{
				Identity: commonidentity.Identity{CustomerID: customerID},
				Owner:    commonidentity.Owner{OwnerID: ownerID},
				ChatID:   chatID,
			},
			expectListChatID: chatID,
			responseErr:      errors.New("rpc failed"),
			expect: []string{
				"customer_id." + customerID.String() + ".talk.chat_created." + chatID.String(),
				"agent_id." + ownerID.String() + ".talk.chat_created." + chatID.String(),
			},
		},
		{
			name: "owner participant is not duplicated",
			data: &webhookOwnerData{
				Identity: commonidentity.Identity{CustomerID: customerID},
				Owner:    commonidentity.Owner{OwnerID: ownerID},
				ChatID:   chatID,
			},
			expectListChatID: chatID,
			responseList: []*tkparticipant.Participant{
				{Owner: commonidentity.Owner{OwnerID: ownerID}},
				{Owner: commonidentity.Owner{OwnerID: otherID}},
			},
			expect: []string{
				"customer_id." + customerID.String() + ".talk.chat_created." + chatID.String(),
				"agent_id." + ownerID.String() + ".talk.chat_created." + chatID.String(),
				"agent_id." + otherID.String() + ".talk.chat_created." + chatID.String(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &webhookHandler{reqHandler: mockReq}

			ctx := context.Background()
			if tt.expectListChatID != uuid.Nil {
				mockReq.EXPECT().TalkV1ParticipantList(ctx, tt.expectListChatID).Return(tt.responseList, tt.responseErr).Times(1)
			}

			got := h.createRoutingKeysForChat(ctx, tt.data, "chat_created")
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, got)
			}
		})
	}
}

func Test_publishRoutingKeyedEvent_cases(t *testing.T) {
	customerID := "a1b2c3d4-0000-0000-0000-0000000000c1"
	ownerID := "a1b2c3d4-0000-0000-0000-0000000000a1"
	otherID := "a1b2c3d4-0000-0000-0000-0000000000a2"
	chatID := "ca740000-0000-0000-0000-000000000011"

	t.Run("event type without an underscore publishes nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// neither mock has an expectation, so any call fails the test.
		h := &webhookHandler{
			reqHandler:         requesthandler.NewMockRequestHandler(mc),
			topicNotifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		}

		data := json.RawMessage(`{"type":"call","data":{"id":"` + chatID + `","customer_id":"` + customerID + `"}}`)
		h.publishRoutingKeyedEvent(context.Background(), "", data)
	})

	t.Run("chat event publishes one message per routing key in order", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &webhookHandler{reqHandler: mockReq, topicNotifyHandler: mockNotify}

		ctx := context.Background()
		inner := `{"id":"` + chatID + `","customer_id":"` + customerID + `","owner_id":"` + ownerID + `","chat_id":"` + chatID + `"}`
		data := json.RawMessage(`{"type":"chatmessage_created","data":` + inner + `}`)

		mockReq.EXPECT().TalkV1ParticipantList(ctx, uuid.FromStringOrNil(chatID)).Return([]*tkparticipant.Participant{
			{Owner: commonidentity.Owner{OwnerID: uuid.FromStringOrNil(ownerID)}},
			{Owner: commonidentity.Owner{OwnerID: uuid.FromStringOrNil(otherID)}},
		}, nil).Times(1)

		payload := json.RawMessage(inner)
		gomock.InOrder(
			mockNotify.EXPECT().PublishEventWithRoutingKey(ctx, "chatmessage_created", "customer_id."+customerID+".talk.chatmessage_created."+chatID, payload).Times(1),
			mockNotify.EXPECT().PublishEventWithRoutingKey(ctx, "chatmessage_created", "agent_id."+ownerID+".talk.chatmessage_created."+chatID, payload).Times(1),
			mockNotify.EXPECT().PublishEventWithRoutingKey(ctx, "chatmessage_created", "agent_id."+otherID+".talk.chatmessage_created."+chatID, payload).Times(1),
		)

		h.publishRoutingKeyedEvent(ctx, "", data)
	})
}

func Test_publishRoutingKeyedEvent_conversationMessage(t *testing.T) {
	customerID := "a1b2c3d4-0000-0000-0000-0000000000c1"
	otherCustomerID := "a1b2c3d4-0000-0000-0000-0000000000c2"
	ownerID := "a1b2c3d4-0000-0000-0000-0000000000a1"
	messageID := "ca740000-0000-0000-0000-000000000021"
	conversationID := "ca740000-0000-0000-0000-000000000022"

	inner := `{"id":"` + messageID + `","customer_id":"` + customerID + `","conversation_id":"` + conversationID + `","text":"a<b"}`
	injected := `{"conversation_id":"` + conversationID + `","customer_id":"` + customerID + `","id":"` + messageID + `","owner_id":"` + ownerID + `","owner_type":"agent","text":"a<b"}`

	customerKey := "customer_id." + customerID + ".conversation.conversation_message_created." + messageID
	agentKey := "agent_id." + ownerID + ".conversation.conversation_message_created." + messageID

	tests := []struct {
		name string

		eventType    string
		conversation *cvconversation.Conversation
		lookupErr    error
		inner        string

		expectLookup  bool
		expectPayload string
		expectKeys    []string
	}{
		{
			name:      "agent owned conversation publishes the agent key and the owner in the payload",
			eventType: "conversation_message_created",
			conversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil(customerID)},
				Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: uuid.FromStringOrNil(ownerID)},
			},
			inner:         inner,
			expectLookup:  true,
			expectPayload: injected,
			expectKeys:    []string{customerKey, agentKey},
		},
		{
			name:      "updated event is handled the same way",
			eventType: "conversation_message_updated",
			conversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil(customerID)},
				Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: uuid.FromStringOrNil(ownerID)},
			},
			inner:         inner,
			expectLookup:  true,
			expectPayload: injected,
			expectKeys: []string{
				"customer_id." + customerID + ".conversation.conversation_message_updated." + messageID,
				"agent_id." + ownerID + ".conversation.conversation_message_updated." + messageID,
			},
		},
		{
			name:      "conversation without an owner keeps the customer key and the original payload",
			eventType: "conversation_message_created",
			conversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil(customerID)},
			},
			inner:         inner,
			expectLookup:  true,
			expectPayload: inner,
			expectKeys:    []string{customerKey},
		},
		{
			name:      "owner type other than agent is ignored",
			eventType: "conversation_message_created",
			conversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil(customerID)},
				Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeNone, OwnerID: uuid.FromStringOrNil(ownerID)},
			},
			inner:         inner,
			expectLookup:  true,
			expectPayload: inner,
			expectKeys:    []string{customerKey},
		},
		{
			name:      "conversation of another customer is ignored",
			eventType: "conversation_message_created",
			conversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil(otherCustomerID)},
				Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: uuid.FromStringOrNil(ownerID)},
			},
			inner:         inner,
			expectLookup:  true,
			expectPayload: inner,
			expectKeys:    []string{customerKey},
		},
		{
			name:          "lookup failure degrades to the customer key",
			eventType:     "conversation_message_created",
			lookupErr:     errors.New("rpc failed"),
			inner:         inner,
			expectLookup:  true,
			expectPayload: inner,
			expectKeys:    []string{customerKey},
		},
		{
			name:          "conversation not found degrades to the customer key",
			eventType:     "conversation_message_deleted",
			lookupErr:     requesthandler.ErrNotFound,
			inner:         inner,
			expectLookup:  true,
			expectPayload: inner,
			expectKeys:    []string{"customer_id." + customerID + ".conversation.conversation_message_deleted." + messageID},
		},
		{
			name:          "message without conversation_id does not call the lookup",
			eventType:     "conversation_message_created",
			inner:         `{"id":"` + messageID + `","customer_id":"` + customerID + `"}`,
			expectLookup:  false,
			expectPayload: `{"id":"` + messageID + `","customer_id":"` + customerID + `"}`,
			expectKeys:    []string{customerKey},
		},
		{
			name:      "agent owner without an owner id keeps the customer key",
			eventType: "conversation_message_created",
			conversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{CustomerID: uuid.FromStringOrNil(customerID)},
				Owner:    commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent},
			},
			inner:         inner,
			expectLookup:  true,
			expectPayload: inner,
			expectKeys:    []string{customerKey},
		},
		{
			// the payload carries a conversation_id on purpose: an over-wide prefix match must
			// call the lookup, which the mock rejects as an unexpected call.
			name:      "conversation_created is not looked up and keeps the payload",
			eventType: "conversation_created",
			inner:     `{"id":"` + conversationID + `","customer_id":"` + customerID + `","conversation_id":"` + conversationID + `","owner_type":"agent","owner_id":"` + ownerID + `"}`,

			expectLookup:  false,
			expectPayload: `{"id":"` + conversationID + `","customer_id":"` + customerID + `","conversation_id":"` + conversationID + `","owner_type":"agent","owner_id":"` + ownerID + `"}`,
			expectKeys: []string{
				"customer_id." + customerID + ".conversation.conversation_created." + conversationID,
				"agent_id." + ownerID + ".conversation.conversation_created." + conversationID,
			},
		},
		{
			// the payload carries a conversation_id on purpose: an over-wide prefix match must
			// call the lookup, which the mock rejects as an unexpected call.
			name:      "conversation_updated is not looked up and keeps the payload",
			eventType: "conversation_updated",
			inner:     `{"id":"` + conversationID + `","customer_id":"` + customerID + `","conversation_id":"` + conversationID + `","owner_type":"agent","owner_id":"` + ownerID + `"}`,

			expectLookup:  false,
			expectPayload: `{"id":"` + conversationID + `","customer_id":"` + customerID + `","conversation_id":"` + conversationID + `","owner_type":"agent","owner_id":"` + ownerID + `"}`,
			expectKeys: []string{
				"customer_id." + customerID + ".conversation.conversation_updated." + conversationID,
				"agent_id." + ownerID + ".conversation.conversation_updated." + conversationID,
			},
		},
		{
			// the payload carries a conversation_id on purpose: an over-wide prefix match must
			// call the lookup, which the mock rejects as an unexpected call.
			name:      "conversation_deleted is not looked up and keeps the payload",
			eventType: "conversation_deleted",
			inner:     `{"id":"` + conversationID + `","customer_id":"` + customerID + `","conversation_id":"` + conversationID + `","owner_type":"agent","owner_id":"` + ownerID + `"}`,

			expectLookup:  false,
			expectPayload: `{"id":"` + conversationID + `","customer_id":"` + customerID + `","conversation_id":"` + conversationID + `","owner_type":"agent","owner_id":"` + ownerID + `"}`,
			expectKeys: []string{
				"customer_id." + customerID + ".conversation.conversation_deleted." + conversationID,
				"agent_id." + ownerID + ".conversation.conversation_deleted." + conversationID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &webhookHandler{reqHandler: mockReq, topicNotifyHandler: mockNotify}

			ctx := context.Background()
			data := json.RawMessage(`{"type":"` + tt.eventType + `","data":` + tt.inner + `}`)
			original := string(data)

			if tt.expectLookup {
				mockReq.EXPECT().ConversationV1ConversationGet(gomock.Any(), uuid.FromStringOrNil(conversationID)).Return(tt.conversation, tt.lookupErr).Times(1)
			}

			var gotKeys []string
			var gotPayloads []string
			mockNotify.EXPECT().PublishEventWithRoutingKey(ctx, tt.eventType, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, key string, payload interface{}) {
					gotKeys = append(gotKeys, key)
					raw, ok := payload.(json.RawMessage)
					if !ok {
						t.Errorf("Wrong payload type: %T", payload)
					}
					gotPayloads = append(gotPayloads, string(raw))
				}).Times(len(tt.expectKeys))

			h.publishRoutingKeyedEvent(ctx, "", data)

			if !reflect.DeepEqual(gotKeys, tt.expectKeys) {
				t.Errorf("Wrong keys.\nexpect: %v\ngot: %v", tt.expectKeys, gotKeys)
			}
			for _, p := range gotPayloads {
				if p != tt.expectPayload {
					t.Errorf("Wrong payload.\nexpect: %s\ngot: %s", tt.expectPayload, p)
				}
			}

			// the data handed to the customer HTTP delivery must stay untouched.
			if string(data) != original {
				t.Errorf("The original data was modified.\nexpect: %s\ngot: %s", original, string(data))
			}
		})
	}
}

func Test_injectOwner(t *testing.T) {
	owner := commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-0000000000a1")}

	tests := []struct {
		name   string
		data   string
		expect string
	}{
		{"object", `{"id":"x","n":12345678901234567890}`, `{"id":"x","n":12345678901234567890,"owner_id":"a1b2c3d4-0000-0000-0000-0000000000a1","owner_type":"agent"}`},
		{"keeps html and non-ascii characters", `{"text":"<a>&é"}`, `{"owner_id":"a1b2c3d4-0000-0000-0000-0000000000a1","owner_type":"agent","text":"<a>&é"}`},
		{"null is unchanged", `null`, `null`},
		{"array is unchanged", `[1]`, `[1]`},
		{"scalar is unchanged", `1`, `1`},
		{"empty input is unchanged", ``, ``},
		{"invalid json is unchanged", `{"id":`, `{"id":`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(injectOwner(json.RawMessage(tt.data), owner))
			if got != tt.expect {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expect, got)
			}
		})
	}
}
