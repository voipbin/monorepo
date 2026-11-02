package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	talkmessage "monorepo/bin-talk-manager/models/message"
)

func Test_TalkV1MentionList(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		agentID    uuid.UUID
		pageToken  string
		pageSize   uint64

		expectRequest *sock.Request

		response  *sock.Response
		expectRes []*talkmessage.Message
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			agentID:    uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
			pageToken:  "2020-09-20T03:23:20.995000Z",
			pageSize:   10,

			expectRequest: &sock.Request{
				URI:      "/v1/mentions?page_token=2020-09-20T03%3A23%3A20.995000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"550e8400-e29b-41d4-a716-446655440000","agent_id":"660e8400-e29b-41d4-a716-446655440000"}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`[{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","chat_id":"770e8400-e29b-41d4-a716-446655440000","type":"normal","text":"Hello","medias":[],"metadata":{"reactions":[]}}]`),
			},
			expectRes: []*talkmessage.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					ChatID:   uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
					Type:     talkmessage.TypeNormal,
					Text:     "Hello",
					Medias:   []talkmessage.Media{},
					Metadata: talkmessage.Metadata{Reactions: []talkmessage.Reaction{}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{sock: mockSock}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.talk-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TalkV1MentionList(ctx, tt.customerID, tt.agentID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TalkV1MentionList_error(t *testing.T) {
	tests := []struct {
		name     string
		response *sock.Response
	}{
		{"status not ok", &sock.Response{StatusCode: 500, DataType: ContentTypeJSON, Data: []byte(`{}`)}},
		{"invalid body", &sock.Response{StatusCode: 200, DataType: ContentTypeJSON, Data: []byte(`{`)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{sock: mockSock}

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.talk-manager.request", gomock.Any()).Return(tt.response, nil)

			res, err := reqHandler.TalkV1MentionList(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "", 10)
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
			}
		})
	}
}
