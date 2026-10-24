package listenhandler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/pkg/messagehandler"
)

var errHandlerFailure = errors.New("injected handler failure")

func Test_processRequest_handlerPaths_error(t *testing.T) {
	chatID := uuid.FromStringOrNil(testChatID)
	customerID := uuid.FromStringOrNil(testCustomerID)
	ownerID := uuid.FromStringOrNil(testOwnerID)
	messageID := uuid.FromStringOrNil(testMessageID)
	participantID := uuid.FromStringOrNil(testParticipantID)
	parentID := uuid.FromStringOrNil("11111111-2222-11ed-8e95-a3bc92af9795")

	const (
		ownerBody    = `{"owner_type":"agent","owner_id":"` + testOwnerID + `","emoji":"smile"}`
		messageBody  = `{"chat_id":"` + testChatID + `","owner_type":"agent","owner_id":"` + testOwnerID + `","type":"normal","text":"hi"}`
		participants = "/v1/chats/" + testChatID + "/participants"
		reactions    = "/v1/messages/" + testMessageID + "/reactions"
	)

	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
		data   string

		// setup registers the only collaborator calls that must happen. Anything else fails the test,
		// so the 400 cases also prove that the business handler was not reached.
		setup func(m *lhMocks)

		wantStatus int
		wantErr    error
	}{
		// chats
		{
			name: "chats post: participant owner id is not a uuid", method: sock.RequestMethodPost, uri: "/v1/chats",
			data:       `{"customer_id":"` + testCustomerID + `","type":"group","participants":[{"owner_type":"agent","owner_id":"bad"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "chats post: participants are passed through", method: sock.RequestMethodPost, uri: "/v1/chats",
			data: `{"customer_id":"` + testCustomerID + `","type":"group","creator_type":"agent","creator_id":"` + testOwnerID + `","participants":[{"owner_type":"agent","owner_id":"` + testOwnerID + `"}]}`,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatCreate(gomock.Any(), customerID, chat.TypeGroup, "", "", "agent", ownerID,
					[]participant.ParticipantInput{{OwnerType: "agent", OwnerID: ownerID}}).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "chats get: body is not json", method: sock.RequestMethodGet, uri: "/v1/chats", data: `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "chats get: filter value has the wrong type", method: sock.RequestMethodGet, uri: "/v1/chats", data: `{"customer_id":"not-a-uuid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "chats get: list failure", method: sock.RequestMethodGet, uri: "/v1/chats", data: `{}`,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatList(gomock.Any(), map[chat.Field]any{}, "", uint64(50)).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "chats id put: body is not json", method: sock.RequestMethodPut, uri: "/v1/chats/" + testChatID, data: `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "chats id put: update failure", method: sock.RequestMethodPut, uri: "/v1/chats/" + testChatID, data: `{}`,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatUpdate(gomock.Any(), chatID, gomock.Nil(), gomock.Nil()).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "chats id delete: delete failure", method: sock.RequestMethodDelete, uri: "/v1/chats/" + testChatID,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatDelete(gomock.Any(), chatID).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},

		// chat participants
		{
			// the body is valid, so only the chat id can be the reason for the 400
			name: "participants post: chat id is not a uuid", method: sock.RequestMethodPost, uri: "/v1/chats/bad/participants", data: ownerBody,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "participants post: body is not json", method: sock.RequestMethodPost, uri: participants, data: `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "participants post: owner id is missing", method: sock.RequestMethodPost, uri: participants, data: `{"owner_type":"agent"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "participants post: add failure", method: sock.RequestMethodPost, uri: participants, data: ownerBody,
			setup: func(m *lhMocks) {
				m.part.EXPECT().ParticipantAdd(gomock.Any(), chatID, ownerID, "agent").Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "participants get: chat id is not a uuid", method: sock.RequestMethodGet, uri: "/v1/chats/bad/participants",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "participants get: chat lookup failure is returned to the dispatcher", method: sock.RequestMethodGet, uri: participants,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatGet(gomock.Any(), chatID).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError, wantErr: errHandlerFailure,
		},
		{
			name: "participants get: list failure", method: sock.RequestMethodGet, uri: participants,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatGet(gomock.Any(), chatID).Return(chatWithCustomer(), nil)
				m.part.EXPECT().ParticipantList(gomock.Any(), customerID, chatID).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "participants id delete: participant id is not a uuid", method: sock.RequestMethodDelete, uri: participants + "/bad",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "participants id delete: chat id is not a uuid", method: sock.RequestMethodDelete, uri: "/v1/chats/bad/participants/" + testParticipantID,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "participants id delete: chat lookup failure is returned to the dispatcher", method: sock.RequestMethodDelete, uri: participants + "/" + testParticipantID,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatGet(gomock.Any(), chatID).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError, wantErr: errHandlerFailure,
		},
		{
			name: "participants id delete: remove failure", method: sock.RequestMethodDelete, uri: participants + "/" + testParticipantID,
			setup: func(m *lhMocks) {
				m.chat.EXPECT().ChatGet(gomock.Any(), chatID).Return(chatWithCustomer(), nil)
				m.part.EXPECT().ParticipantRemove(gomock.Any(), customerID, participantID).Return(errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},

		// messages
		{
			name: "messages post: body is not json", method: sock.RequestMethodPost, uri: "/v1/messages", data: `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "messages post: owner id is missing", method: sock.RequestMethodPost, uri: "/v1/messages", data: `{"chat_id":"` + testChatID + `"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "messages post: chat id is missing", method: sock.RequestMethodPost, uri: "/v1/messages", data: `{"owner_id":"` + testOwnerID + `"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "messages post: parent id is passed through", method: sock.RequestMethodPost, uri: "/v1/messages",
			data: `{"chat_id":"` + testChatID + `","parent_id":"11111111-2222-11ed-8e95-a3bc92af9795","owner_type":"agent","owner_id":"` + testOwnerID + `","type":"normal","text":"hi"}`,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MessageCreate(gomock.Any(), messagehandler.MessageCreateRequest{
					ChatID: chatID, ParentID: &parentID, OwnerType: "agent", OwnerID: ownerID, Type: "normal", Text: "hi",
				}).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "messages post: create failure", method: sock.RequestMethodPost, uri: "/v1/messages", data: messageBody,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "messages get: body is not json", method: sock.RequestMethodGet, uri: "/v1/messages", data: `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "messages get: filter value has the wrong type", method: sock.RequestMethodGet, uri: "/v1/messages", data: `{"chat_id":"not-a-uuid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "messages get: list failure", method: sock.RequestMethodGet, uri: "/v1/messages", data: `{}`,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MessageList(gomock.Any(), map[message.Field]any{}, "", uint64(50)).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "messages id get: failure is returned to the dispatcher", method: sock.RequestMethodGet, uri: "/v1/messages/" + testMessageID,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MessageGet(gomock.Any(), messageID).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError, wantErr: errHandlerFailure,
		},
		{
			name: "messages id delete: delete failure", method: sock.RequestMethodDelete, uri: "/v1/messages/" + testMessageID,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MessageDelete(gomock.Any(), messageID).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},

		// participants collection
		{
			name: "participants collection get: body is not json", method: sock.RequestMethodGet, uri: "/v1/participants", data: `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "participants collection get: filter value has the wrong type", method: sock.RequestMethodGet, uri: "/v1/participants", data: `{"chat_id":"not-a-uuid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "participants collection get: list failure", method: sock.RequestMethodGet, uri: "/v1/participants", data: `{}`,
			setup: func(m *lhMocks) {
				m.part.EXPECT().ParticipantListWithFilters(gomock.Any(), map[participant.Field]any{}, "", uint64(50)).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},

		// reactions
		{
			name: "reactions post: body is not json", method: sock.RequestMethodPost, uri: reactions, data: `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "reactions post: owner id is missing", method: sock.RequestMethodPost, uri: reactions, data: `{"emoji":"smile"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "reactions post: add failure", method: sock.RequestMethodPost, uri: reactions, data: ownerBody,
			setup: func(m *lhMocks) {
				m.react.EXPECT().ReactionAdd(gomock.Any(), messageID, "smile", "agent", ownerID).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "reactions delete: body is not json", method: sock.RequestMethodDelete, uri: reactions, data: `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "reactions delete: owner id is missing", method: sock.RequestMethodDelete, uri: reactions, data: `{"emoji":"smile"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "reactions delete: remove failure", method: sock.RequestMethodDelete, uri: reactions, data: ownerBody,
			setup: func(m *lhMocks) {
				m.react.EXPECT().ReactionRemove(gomock.Any(), messageID, "smile", "agent", ownerID).Return(nil, errHandlerFailure)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newRoutingHandler(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, DataType: "application/json", Data: []byte(tt.data)})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if res.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", res.StatusCode, tt.wantStatus)
			}
		})
	}
}
