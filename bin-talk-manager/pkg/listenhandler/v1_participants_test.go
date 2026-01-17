package listenhandler

import (
	"context"
	"encoding/json"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/pkg/participanthandler"
)

func Test_processV1TalksIDParticipantsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		chatID      uuid.UUID
		customerID  uuid.UUID
		ownerType   string
		ownerID     uuid.UUID
		responseParticipant *participant.Participant
		expectRes   *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/talks/6ebc6880-31da-11ed-8e95-a3bc92af9795/participants",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796"}`),
			},

			chatID:     uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			ownerType:  "agent",
			ownerID:    uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
			responseParticipant: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bbef9d30-75fe-11ed-c3ea-f8e017af9700"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
				},
				ChatID:   uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
				TMJoined: "2021-11-23 17:55:39.712000",
			},
			expectRes: &sock.Response{
				StatusCode: 201,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbef9d30-75fe-11ed-c3ea-f8e017af9700","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","tm_joined":"2021-11-23 17:55:39.712000"}`),
			},
		},
		{
			name: "customer owner type",
			request: &sock.Request{
				URI:      "/v1/talks/6ebc6880-31da-11ed-8e95-a3bc92af9795/participants",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"customer","owner_id":"8ede8b40-86ef-11ed-d4fb-e9e028af9801"}`),
			},

			chatID:     uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			ownerType:  "customer",
			ownerID:    uuid.FromStringOrNil("8ede8b40-86ef-11ed-d4fb-e9e028af9801"),
			responseParticipant: &participant.Participant{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cce09e40-86ef-11ed-e5ec-e0e039af9902"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "customer",
					OwnerID:   uuid.FromStringOrNil("8ede8b40-86ef-11ed-d4fb-e9e028af9801"),
				},
				ChatID:   uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
				TMJoined: "2021-11-23 18:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 201,
				DataType:   "application/json",
				Data:       []byte(`{"id":"cce09e40-86ef-11ed-e5ec-e0e039af9902","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"customer","owner_id":"8ede8b40-86ef-11ed-d4fb-e9e028af9801","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","tm_joined":"2021-11-23 18:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockParticipant := participanthandler.NewMockParticipantHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				participantHandler: mockParticipant,
			}

			ctx := context.Background()
			mockParticipant.EXPECT().ParticipantAdd(ctx, tt.customerID, tt.chatID, tt.ownerID, tt.ownerType).Return(tt.responseParticipant, nil)

			res, err := h.v1TalkChatsIDParticipantsPost(ctx, *tt.request, tt.chatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TalksIDParticipantsPost_error(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		chatID    uuid.UUID
		expectRes *sock.Response
	}{
		{
			name: "invalid json",
			request: &sock.Request{
				URI:      "/v1/talks/6ebc6880-31da-11ed-8e95-a3bc92af9795/participants",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{invalid json`),
			},
			chatID: uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
			expectRes: &sock.Response{
				StatusCode: 400,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
		{
			name: "nil customer id",
			request: &sock.Request{
				URI:      "/v1/talks/6ebc6880-31da-11ed-8e95-a3bc92af9795/participants",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796"}`),
			},
			chatID: uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
			expectRes: &sock.Response{
				StatusCode: 400,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
		{
			name: "nil owner id",
			request: &sock.Request{
				URI:      "/v1/talks/6ebc6880-31da-11ed-8e95-a3bc92af9795/participants",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":""}`),
			},
			chatID: uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
			expectRes: &sock.Response{
				StatusCode: 400,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockParticipant := participanthandler.NewMockParticipantHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				participantHandler: mockParticipant,
			}

			ctx := context.Background()
			res, err := h.v1TalkChatsIDParticipantsPost(ctx, *tt.request, tt.chatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TalksIDParticipantsGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		chatID               uuid.UUID
		customerID           uuid.UUID
		responseParticipants []*participant.Participant
		expectRes            *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/talks/6ebc6880-31da-11ed-8e95-a3bc92af9795/participants",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b"}`),
			},

			chatID:     uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			responseParticipants: []*participant.Participant{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bbef9d30-75fe-11ed-c3ea-f8e017af9700"),
						CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
					},
					ChatID:   uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
					TMJoined: "2021-11-23 17:55:39.712000",
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bbef9d30-75fe-11ed-c3ea-f8e017af9700","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","tm_joined":"2021-11-23 17:55:39.712000"}]`),
			},
		},
		{
			name: "empty list",
			request: &sock.Request{
				URI:      "/v1/talks/6ebc6880-31da-11ed-8e95-a3bc92af9795/participants",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b"}`),
			},

			chatID:               uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
			customerID:           uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			responseParticipants: []*participant.Participant{},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockParticipant := participanthandler.NewMockParticipantHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				participantHandler: mockParticipant,
			}

			ctx := context.Background()
			mockParticipant.EXPECT().ParticipantList(ctx, tt.customerID, tt.chatID).Return(tt.responseParticipants, nil)

			res, err := h.v1TalkChatsIDParticipantsGet(ctx, *tt.request, tt.chatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TalksIDParticipantsIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		participantID uuid.UUID
		customerID    uuid.UUID
		expectRes     *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/talks/6ebc6880-31da-11ed-8e95-a3bc92af9795/participants/bbef9d30-75fe-11ed-c3ea-f8e017af9700",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b"}`),
			},

			participantID: uuid.FromStringOrNil("bbef9d30-75fe-11ed-c3ea-f8e017af9700"),
			customerID:    uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			expectRes: &sock.Response{
				StatusCode: 204,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockParticipant := participanthandler.NewMockParticipantHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				participantHandler: mockParticipant,
			}

			ctx := context.Background()
			mockParticipant.EXPECT().ParticipantRemove(ctx, tt.customerID, tt.participantID).Return(nil)

			res, err := h.processV1TalkChatsIDParticipantsID(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TalksIDParticipantsID_unsupported_method(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			name: "GET method",
			request: &sock.Request{
				URI:      "/v1/talks/6ebc6880-31da-11ed-8e95-a3bc92af9795/participants/bbef9d30-75fe-11ed-c3ea-f8e017af9700",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			expectRes: &sock.Response{
				StatusCode: 405,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockParticipant := participanthandler.NewMockParticipantHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				participantHandler: mockParticipant,
			}

			ctx := context.Background()
			res, err := h.processV1TalkChatsIDParticipantsID(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
