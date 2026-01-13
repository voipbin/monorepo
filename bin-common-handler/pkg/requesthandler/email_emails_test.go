package requesthandler

import (
	"context"
	"monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	ememail "monorepo/bin-email-manager/models/email"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_EmailV1EmailGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[ememail.Field]any

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []ememail.Email
	}{
		{
			name: "normal",

			pageToken: "2021-03-02 03:23:20.995000",
			pageSize:  10,
			filters: map[ememail.Field]any{
				ememail.FieldDeleted: false,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"67770202-0076-11f0-8091-bb84394f5e3b"},{"id":"679b8e06-0076-11f0-b24b-47c12990318f"}]`),
			},

			expectTarget: "bin-manager.email-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/emails?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"deleted":false}`),
			},
			expectRes: []ememail.Email{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("67770202-0076-11f0-8091-bb84394f5e3b"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("679b8e06-0076-11f0-b24b-47c12990318f"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.EmailV1EmailGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_EmailV1EmailSend(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		activeflowID uuid.UUID
		destinations []address.Address
		subject      string
		content      string
		attachments  []ememail.Attachment

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *ememail.Email
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("b9750256-0077-11f0-acf2-13e93f5406e0"),
			activeflowID: uuid.FromStringOrNil("b9a787c6-0077-11f0-a27f-53ed03e505eb"),
			destinations: []address.Address{
				{
					Type:       address.TypeEmail,
					Target:     "test@voipbin.net",
					TargetName: "test name",
				},
			},
			subject: "test subject",
			content: "test content",
			attachments: []ememail.Attachment{
				{
					ReferenceType: ememail.AttachmentReferenceTypeRecording,
					ReferenceID:   uuid.FromStringOrNil("b9de7150-0077-11f0-a65a-dfc71909ce11"),
				},
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"ba07d31a-0077-11f0-abe8-ebb162d6c0fa"}`),
			},

			expectTarget: "bin-manager.email-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/emails",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"b9750256-0077-11f0-acf2-13e93f5406e0","activeflow_id":"b9a787c6-0077-11f0-a27f-53ed03e505eb","destinations":[{"type":"email","target":"test@voipbin.net","target_name":"test name"}],"subject":"test subject","content":"test content","attachments":[{"reference_type":"recording","reference_id":"b9de7150-0077-11f0-a65a-dfc71909ce11"}]}`),
			},
			expectRes: &ememail.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("ba07d31a-0077-11f0-abe8-ebb162d6c0fa"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.EmailV1EmailSend(
				ctx,
				tt.customerID,
				tt.activeflowID,
				tt.destinations,
				tt.subject,
				tt.content,
				tt.attachments,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_EmailV1EmailGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *ememail.Email
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("0ae8879c-0079-11f0-aaa3-93cf00bef66b"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0ae8879c-0079-11f0-aaa3-93cf00bef66b"}`),
			},

			expectTarget: "bin-manager.email-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/emails/0ae8879c-0079-11f0-aaa3-93cf00bef66b",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			expectRes: &ememail.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("0ae8879c-0079-11f0-aaa3-93cf00bef66b"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.EmailV1EmailGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_EmailV1EmailDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *ememail.Email
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("52e8c7f0-0079-11f0-b725-9fb8f6fb69c3"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"52e8c7f0-0079-11f0-b725-9fb8f6fb69c3"}`),
			},

			expectTarget: "bin-manager.email-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/emails/52e8c7f0-0079-11f0-b725-9fb8f6fb69c3",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			expectRes: &ememail.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("52e8c7f0-0079-11f0-b725-9fb8f6fb69c3"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.EmailV1EmailDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
