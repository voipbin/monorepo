package listenhandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/pkg/chathandler"
	"monorepo/bin-talk-manager/pkg/dbhandler"
	"monorepo/bin-talk-manager/pkg/messagehandler"
	"monorepo/bin-talk-manager/pkg/participanthandler"
	"monorepo/bin-talk-manager/pkg/reactionhandler"
)

const (
	testChatID        = "6ebc6880-31da-11ed-8e95-a3bc92af9795"
	testCustomerID    = "5e4a0680-804e-11ec-8477-2fea5968d85b"
	testOwnerID       = "7fcd7990-42eb-11ed-9fa6-b4cd93af9796"
	testMessageID     = "9a1c3f10-42eb-11ed-9fa6-b4cd93af9797"
	testParticipantID = "bbef9d30-75fe-11ed-c3ea-f8e017af9700"
)

type lhMocks struct {
	sock  *sockhandler.MockSockHandler
	chat  *chathandler.MockChatHandler
	msg   *messagehandler.MockMessageHandler
	part  *participanthandler.MockParticipantHandler
	react *reactionhandler.MockReactionHandler
}

// newRoutingHandler builds a listenHandler through New() with mocks for every collaborator
// and the real util handler (filter parsing is a pure function).
func newRoutingHandler(t *testing.T) (*listenHandler, *lhMocks) {
	t.Helper()

	mc := gomock.NewController(t)
	m := &lhMocks{
		sock:  sockhandler.NewMockSockHandler(mc),
		chat:  chathandler.NewMockChatHandler(mc),
		msg:   messagehandler.NewMockMessageHandler(mc),
		part:  participanthandler.NewMockParticipantHandler(mc),
		react: reactionhandler.NewMockReactionHandler(mc),
	}

	return New(m.sock, m.chat, m.msg, m.part, m.react, utilhandler.NewUtilHandler()), m
}

func chatWithCustomer() *chat.Chat {
	c := &chat.Chat{Type: chat.TypeGroup}
	c.ID = uuid.FromStringOrNil(testChatID)
	c.CustomerID = uuid.FromStringOrNil(testCustomerID)
	return c
}

func Test_processRequest_routes(t *testing.T) {
	chatID := uuid.FromStringOrNil(testChatID)
	customerID := uuid.FromStringOrNil(testCustomerID)
	ownerID := uuid.FromStringOrNil(testOwnerID)
	messageID := uuid.FromStringOrNil(testMessageID)
	participantID := uuid.FromStringOrNil(testParticipantID)

	respChat := chatWithCustomer()
	respMsg := &message.Message{}
	respMsg.ID = messageID
	respPart := &participant.Participant{}
	respPart.ID = participantID

	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
		data   string

		setup      func(m *lhMocks)
		wantStatus int
		wantBody   any
	}{
		{
			name: "chats post", method: sock.RequestMethodPost, uri: "/v1/chats",
			data: `{"customer_id":"` + testCustomerID + `","type":"group","name":"n"}`,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatCreate(gomock.Any(), customerID, chat.TypeGroup, "n", "", "", uuid.Nil, gomock.Nil()).Return(respChat, nil)
			},
			wantStatus: 201, wantBody: respChat,
		},
		{
			name: "chats get with query", method: sock.RequestMethodGet, uri: "/v1/chats?page_size=7&page_token=tok",
			data: `{"customer_id":"` + testCustomerID + `"}`,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatList(gomock.Any(), map[chat.Field]any{chat.FieldCustomerID: customerID}, "tok", uint64(7)).Return([]*chat.Chat{respChat}, nil)
			},
			wantStatus: 200, wantBody: []*chat.Chat{respChat},
		},
		{
			name: "chats id get", method: sock.RequestMethodGet, uri: "/v1/chats/" + testChatID,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatGet(gomock.Any(), chatID).Return(respChat, nil)
			},
			wantStatus: 200, wantBody: respChat,
		},
		{
			name: "chats id put", method: sock.RequestMethodPut, uri: "/v1/chats/" + testChatID,
			data: `{"name":"renamed"}`,
			setup: func(m *lhMocks) {
				name := "renamed"
				m.chat.EXPECT().ChatUpdate(gomock.Any(), chatID, &name, gomock.Nil()).Return(respChat, nil)
			},
			wantStatus: 200, wantBody: respChat,
		},
		{
			name: "chats id delete", method: sock.RequestMethodDelete, uri: "/v1/chats/" + testChatID,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatDelete(gomock.Any(), chatID).Return(respChat, nil)
			},
			wantStatus: 200, wantBody: respChat,
		},
		{
			name: "chats id participants post", method: sock.RequestMethodPost, uri: "/v1/chats/" + testChatID + "/participants",
			data: `{"owner_type":"agent","owner_id":"` + testOwnerID + `"}`,
			setup: func(m *lhMocks) {
				m.part.EXPECT().ParticipantAdd(gomock.Any(), chatID, ownerID, "agent").Return(respPart, nil)
			},
			wantStatus: 201, wantBody: respPart,
		},
		{
			name: "chats id participants get", method: sock.RequestMethodGet, uri: "/v1/chats/" + testChatID + "/participants",
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatGet(gomock.Any(), chatID).Return(respChat, nil)
				m.part.EXPECT().ParticipantList(gomock.Any(), customerID, chatID).Return([]*participant.Participant{respPart}, nil)
			},
			wantStatus: 200, wantBody: []*participant.Participant{respPart},
		},
		{
			name: "chats id participants id delete", method: sock.RequestMethodDelete, uri: "/v1/chats/" + testChatID + "/participants/" + testParticipantID,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatGet(gomock.Any(), chatID).Return(respChat, nil)
				m.part.EXPECT().ParticipantRemove(gomock.Any(), customerID, participantID).Return(nil)
			},
			wantStatus: 204,
		},
		{
			name: "messages post", method: sock.RequestMethodPost, uri: "/v1/messages",
			data: `{"chat_id":"` + testChatID + `","owner_type":"agent","owner_id":"` + testOwnerID + `","type":"normal","text":"hi"}`,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MessageCreate(gomock.Any(), messagehandler.MessageCreateRequest{
					ChatID: chatID, OwnerType: "agent", OwnerID: ownerID, Type: "normal", Text: "hi",
				}).Return(respMsg, nil)
			},
			wantStatus: 201, wantBody: respMsg,
		},
		{
			name: "messages get", method: sock.RequestMethodGet, uri: "/v1/messages?page_size=3",
			data: `{"chat_id":"` + testChatID + `"}`,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MessageList(gomock.Any(), map[message.Field]any{message.FieldChatID: chatID}, "", uint64(3)).Return([]*message.Message{respMsg}, nil)
			},
			wantStatus: 200, wantBody: []*message.Message{respMsg},
		},
		{
			name: "messages id get", method: sock.RequestMethodGet, uri: "/v1/messages/" + testMessageID,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MessageGet(gomock.Any(), messageID).Return(respMsg, nil)
			},
			wantStatus: 200, wantBody: respMsg,
		},
		{
			name: "messages id delete", method: sock.RequestMethodDelete, uri: "/v1/messages/" + testMessageID,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MessageDelete(gomock.Any(), messageID).Return(respMsg, nil)
			},
			wantStatus: 200, wantBody: respMsg,
		},
		{
			name: "participants get", method: sock.RequestMethodGet, uri: "/v1/participants?page_size=2",
			data: `{"chat_id":"` + testChatID + `"}`,
			setup: func(m *lhMocks) {
				m.part.EXPECT().ParticipantListWithFilters(gomock.Any(), map[participant.Field]any{participant.FieldChatID: chatID}, "", uint64(2)).Return([]*participant.Participant{respPart}, nil)
			},
			wantStatus: 200, wantBody: []*participant.Participant{respPart},
		},
		{
			name: "messages id reactions post", method: sock.RequestMethodPost, uri: "/v1/messages/" + testMessageID + "/reactions",
			data: `{"owner_type":"agent","owner_id":"` + testOwnerID + `","emoji":"smile"}`,
			setup: func(m *lhMocks) {
				m.react.EXPECT().ReactionAdd(gomock.Any(), messageID, "smile", "agent", ownerID).Return(respMsg, nil)
			},
			wantStatus: 200, wantBody: respMsg,
		},
		{
			name: "messages id reactions delete", method: sock.RequestMethodDelete, uri: "/v1/messages/" + testMessageID + "/reactions",
			data: `{"owner_type":"agent","owner_id":"` + testOwnerID + `","emoji":"smile"}`,
			setup: func(m *lhMocks) {
				m.react.EXPECT().ReactionRemove(gomock.Any(), messageID, "smile", "agent", ownerID).Return(respMsg, nil)
			},
			wantStatus: 200, wantBody: respMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newRoutingHandler(t)
			tt.setup(m)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, DataType: "application/json", Data: []byte(tt.data)})
			if err != nil {
				t.Fatalf("processRequest() error = %v", err)
			}
			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", res.StatusCode, tt.wantStatus)
			}

			if tt.wantBody != nil {
				want, _ := json.Marshal(tt.wantBody)
				if !reflect.DeepEqual([]byte(res.Data), want) {
					t.Errorf("body = %s, want %s", res.Data, want)
				}
			}
		})
	}
}

func Test_processRequest_unrouted(t *testing.T) {
	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
	}{
		{"put on messages id", sock.RequestMethodPut, "/v1/messages/" + testMessageID},
		{"get on participants id", sock.RequestMethodGet, "/v1/chats/" + testChatID + "/participants/" + testParticipantID},
		{"delete on chats collection", sock.RequestMethodDelete, "/v1/chats"},
		{"put on chats collection", sock.RequestMethodPut, "/v1/chats"},
		{"post on participants collection", sock.RequestMethodPost, "/v1/participants"},
		{"suffix after chats", sock.RequestMethodGet, "/v1/chatsX"},
		{"prefix before chats", sock.RequestMethodGet, "/x/v1/chats"},
		{"prefix before chats id", sock.RequestMethodGet, "/x/v1/chats/" + testChatID},
		{"prefix before chat participants", sock.RequestMethodGet, "/x/v1/chats/" + testChatID + "/participants"},
		{"prefix before chat participants id", sock.RequestMethodDelete, "/x/v1/chats/" + testChatID + "/participants/" + testParticipantID},
		{"prefix before messages", sock.RequestMethodGet, "/x/v1/messages"},
		{"prefix before messages id", sock.RequestMethodGet, "/x/v1/messages/" + testMessageID},
		{"prefix before reactions", sock.RequestMethodPost, "/x/v1/messages/" + testMessageID + "/reactions"},
		{"prefix before participants", sock.RequestMethodGet, "/x/v1/participants"},
		{"extra segment after chat id", sock.RequestMethodGet, "/v1/chats/" + testChatID + "/unknown"},
		{"get on reactions", sock.RequestMethodGet, "/v1/messages/" + testMessageID + "/reactions"},
		{"unknown version", sock.RequestMethodGet, "/v2/chats"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newRoutingHandler(t)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method})
			if err != nil {
				t.Fatalf("processRequest() error = %v", err)
			}
			if res.StatusCode != http.StatusNotFound {
				t.Errorf("status = %d, want 404", res.StatusCode)
			}
		})
	}
}

func Test_processRequest_error(t *testing.T) {
	tests := []struct {
		name string
		err  error

		wantStatus   int
		wantDataType string
		wantErr      bool
	}{
		{
			name:         "typed error is converted to its response",
			err:          cerrors.NotFound(commonoutline.ServiceNameTalkManager, "CHAT_NOT_FOUND", "The chat was not found."),
			wantStatus:   http.StatusNotFound,
			wantDataType: cerrors.DataTypeVoipbinError,
		},
		{
			name:         "wrapped not found is a 404",
			err:          errors.Join(errors.New("outer"), dbhandler.ErrNotFound),
			wantStatus:   http.StatusNotFound,
			wantDataType: "application/json",
		},
		{
			name:         "other errors are a 500 and are returned",
			err:          errors.New("boom"),
			wantStatus:   http.StatusInternalServerError,
			wantDataType: "application/json",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newRoutingHandler(t)
			m.chat.EXPECT().ChatGet(gomock.Any(), uuid.FromStringOrNil(testChatID)).Return(nil, tt.err)

			res, err := h.processRequest(&sock.Request{URI: "/v1/chats/" + testChatID, Method: sock.RequestMethodGet})
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, tt.err) {
				t.Errorf("returned error = %v, want %v", err, tt.err)
			}
			if res.StatusCode != tt.wantStatus || res.DataType != tt.wantDataType {
				t.Errorf("response = %d %s, want %d %s", res.StatusCode, res.DataType, tt.wantStatus, tt.wantDataType)
			}
		})
	}
}

func Test_New(t *testing.T) {
	// every collaborator is routed to by exactly one kind of request, so a field that New()
	// leaves unset or fills with the wrong collaborator is detected by the mock expectations.
	mc := gomock.NewController(t)
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockChat := chathandler.NewMockChatHandler(mc)
	mockMsg := messagehandler.NewMockMessageHandler(mc)
	mockPart := participanthandler.NewMockParticipantHandler(mc)
	mockReact := reactionhandler.NewMockReactionHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := New(mockSock, mockChat, mockMsg, mockPart, mockReact, mockUtil)

	chatID := uuid.FromStringOrNil(testChatID)
	messageID := uuid.FromStringOrNil(testMessageID)
	ownerID := uuid.FromStringOrNil(testOwnerID)

	mockChat.EXPECT().ChatGet(gomock.Any(), chatID).Return(chatWithCustomer(), nil)
	mockMsg.EXPECT().MessageGet(gomock.Any(), messageID).Return(&message.Message{}, nil)
	mockUtil.EXPECT().ParseFiltersFromRequestBody(gomock.Any()).Return(map[string]any{}, nil)
	mockPart.EXPECT().ParticipantListWithFilters(gomock.Any(), gomock.Any(), "", uint64(50)).Return([]*participant.Participant{}, nil)
	mockReact.EXPECT().ReactionAdd(gomock.Any(), messageID, "smile", "agent", ownerID).Return(&message.Message{}, nil)

	for _, req := range []*sock.Request{
		{URI: "/v1/chats/" + testChatID, Method: sock.RequestMethodGet},
		{URI: "/v1/messages/" + testMessageID, Method: sock.RequestMethodGet},
		{URI: "/v1/participants", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
		{URI: "/v1/messages/" + testMessageID + "/reactions", Method: sock.RequestMethodPost, Data: []byte(`{"owner_type":"agent","owner_id":"` + testOwnerID + `","emoji":"smile"}`)},
	} {
		res, err := h.processRequest(req)
		if err != nil || res.StatusCode >= 300 {
			t.Errorf("%s %s: response = %v, error = %v", req.Method, req.URI, res, err)
		}
	}
}

func Test_Listen(t *testing.T) {
	t.Run("queue create failure stops before consuming", func(t *testing.T) {
		h, m := newRoutingHandler(t)
		errQueue := errors.New("queue failure")

		m.sock.EXPECT().QueueCreate("bin-manager.talk-manager.request", "normal").Return(errQueue)

		if err := h.Listen(t.Context()); !errors.Is(err, errQueue) {
			t.Errorf("Listen() error = %v, want %v", err, errQueue)
		}
	})

	t.Run("consume failure is returned", func(t *testing.T) {
		h, m := newRoutingHandler(t)
		errConsume := errors.New("consume failure")

		m.sock.EXPECT().QueueCreate("bin-manager.talk-manager.request", "normal").Return(nil)
		m.sock.EXPECT().ConsumeRPC(gomock.Any(), "bin-manager.talk-manager.request", "talk-manager", false, false, false, 10, gomock.Any()).Return(errConsume)

		if err := h.Listen(t.Context()); !errors.Is(err, errConsume) {
			t.Errorf("Listen() error = %v, want %v", err, errConsume)
		}
	})

	t.Run("the consume callback is the request router", func(t *testing.T) {
		h, m := newRoutingHandler(t)

		var callback sock.CbMsgRPC
		m.sock.EXPECT().QueueCreate("bin-manager.talk-manager.request", "normal").Return(nil)
		m.sock.EXPECT().ConsumeRPC(gomock.Any(), "bin-manager.talk-manager.request", "talk-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ any, _, _ string, _, _, _ bool, _ int, cb sock.CbMsgRPC) error {
				callback = cb
				return nil
			})

		if err := h.Listen(t.Context()); err != nil {
			t.Fatalf("Listen() error = %v", err)
		}
		if callback == nil {
			t.Fatalf("the callback was not registered")
		}

		m.chat.EXPECT().ChatGet(gomock.Any(), uuid.FromStringOrNil(testChatID)).Return(chatWithCustomer(), nil)
		res, err := callback(&sock.Request{URI: "/v1/chats/" + testChatID, Method: sock.RequestMethodGet})
		if err != nil || res.StatusCode != 200 {
			t.Errorf("routed request: response = %v, error = %v", res, err)
		}

		res, err = callback(&sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet})
		if err != nil || res.StatusCode != http.StatusNotFound {
			t.Errorf("unknown request: response = %v, error = %v", res, err)
		}
	})
}
