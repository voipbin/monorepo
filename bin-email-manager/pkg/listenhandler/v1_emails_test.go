package listenhandler

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/emailhandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_v1EmailsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageToken string
		pageSize  uint64

		responseFilters map[string]string
		expectFilters   map[email.Field]any
		responseEmails  []*email.Email

		expectRes *sock.Response
	}{
		{
			name: "1 item",
			request: &sock.Request{
				URI:    "/v1/emails?page_token=2020-10-10T03:30:17.000000Z&page_size=10&filter_customer_id=16d3fcf0-7f4c-11ec-a4c3-7bf43125108d&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			pageToken: "2020-10-10T03:30:17.000000Z",
			pageSize:  10,

			responseFilters: map[string]string{
				"customer_id": "16d3fcf0-7f4c-11ec-a4c3-7bf43125108d",
				"deleted":     "false",
			},
			expectFilters: map[email.Field]any{
				email.FieldCustomerID: "16d3fcf0-7f4c-11ec-a4c3-7bf43125108d",
				email.FieldDeleted:    "false",
			},
			responseEmails: []*email.Email{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("2a046c74-00c7-11f0-b07e-a385bcd60724"),
						CustomerID: uuid.FromStringOrNil("16d3fcf0-7f4c-11ec-a4c3-7bf43125108d"),
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2a046c74-00c7-11f0-b07e-a385bcd60724","customer_id":"16d3fcf0-7f4c-11ec-a4c3-7bf43125108d","activeflow_id":"00000000-0000-0000-0000-000000000000","provider_type":"","provider_reference_id":"","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
		{
			name: "2 items",
			request: &sock.Request{
				URI:    "/v1/emails?page_token=2020-10-10T03:30:17.000000Z&page_size=10&filter_customer_id=2457d824-7f4c-11ec-9489-b3552a7c9d63&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			pageToken: "2020-10-10T03:30:17.000000Z",
			pageSize:  10,

			responseFilters: map[string]string{
				"customer_id": "2457d824-7f4c-11ec-9489-b3552a7c9d63",
				"deleted":     "false",
			},
			expectFilters: map[email.Field]any{
				email.FieldCustomerID: "2457d824-7f4c-11ec-9489-b3552a7c9d63",
				email.FieldDeleted:    "false",
			},
			responseEmails: []*email.Email{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("2a242316-00c7-11f0-96db-93d500b33431"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("2c2a494c-00c7-11f0-be49-8f6777e928d8"),
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2a242316-00c7-11f0-96db-93d500b33431","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","provider_type":"","provider_reference_id":"","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":null,"tm_update":null,"tm_delete":null},{"id":"2c2a494c-00c7-11f0-be49-8f6777e928d8","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","provider_type":"","provider_reference_id":"","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
		{
			name: "empty",
			request: &sock.Request{
				URI:      "/v1/emails?page_token=2020-10-10T03:30:17.000000Z&page_size=10&filter_customer_id=3ee14bee-7f4c-11ec-a1d8-a3a488ed5885&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			pageToken: "2020-10-10T03:30:17.000000Z",
			pageSize:  10,

			responseFilters: map[string]string{
				"customer_id": "3ee14bee-7f4c-11ec-a1d8-a3a488ed5885",
				"deleted":     "false",
			},
			expectFilters: map[email.Field]any{
				email.FieldCustomerID: "3ee14bee-7f4c-11ec-a1d8-a3a488ed5885",
				email.FieldDeleted:    "false",
			},
			responseEmails: []*email.Email{},
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

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEmail := emailhandler.NewMockEmailHandler(mc)

			h := &listenHandler{
				utilHandler: mockUtil,
				sockHandler: mockSock,

				emailHandler: mockEmail,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockEmail.EXPECT().List(gomock.Any(), tt.pageToken, tt.pageSize, gomock.Any()).Return(tt.responseEmails, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1EmailsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseEmail *email.Email

		expectCustomerID   uuid.UUID
		expectActiveflowID uuid.UUID
		expectDestinations []commonaddress.Address
		expectSubject      string
		expectContent      string
		expectAttachments  []email.Attachment
		expectRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/emails",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"ad4682ec-00c8-11f0-8781-af3ef9b0ddfe","activeflow_id": "ad6feb50-00c8-11f0-90ee-4bc83c1d684b", "destinations": [{"type":"email","target":"test@voipbin.net","target_name":"test name"}], "subject": "test subject", "content": "test content", "attachments": [{"reference_type":"recording","reference_id":"ad9dee1a-00c8-11f0-b90b-1be758235eec"}]}`),
			},

			responseEmail: &email.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("adcb77d6-00c8-11f0-aadc-1fe067d7823a"),
				},
			},

			expectCustomerID:   uuid.FromStringOrNil("ad4682ec-00c8-11f0-8781-af3ef9b0ddfe"),
			expectActiveflowID: uuid.FromStringOrNil("ad6feb50-00c8-11f0-90ee-4bc83c1d684b"),
			expectDestinations: []commonaddress.Address{
				{
					Type:       commonaddress.TypeEmail,
					Target:     "test@voipbin.net",
					TargetName: "test name",
				},
			},
			expectSubject: "test subject",
			expectContent: "test content",
			expectAttachments: []email.Attachment{
				{
					ReferenceType: email.AttachmentReferenceTypeRecording,
					ReferenceID:   uuid.FromStringOrNil("ad9dee1a-00c8-11f0-b90b-1be758235eec"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"adcb77d6-00c8-11f0-aadc-1fe067d7823a","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","provider_type":"","provider_reference_id":"","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEmail := emailhandler.NewMockEmailHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				emailHandler: mockEmail,
			}

			mockEmail.EXPECT().Create(gomock.Any(), tt.expectCustomerID, tt.expectActiveflowID, tt.expectDestinations, tt.expectSubject, tt.expectContent, tt.expectAttachments).Return(tt.responseEmail, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1EmailsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseEmail *email.Email

		expectEmailID uuid.UUID
		expectRes     *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/emails/ee70fd3c-00c9-11f0-8369-b773a83214a0",
				Method: sock.RequestMethodGet,
			},

			responseEmail: &email.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("ee70fd3c-00c9-11f0-8369-b773a83214a0"),
				},
			},

			expectEmailID: uuid.FromStringOrNil("ee70fd3c-00c9-11f0-8369-b773a83214a0"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"ee70fd3c-00c9-11f0-8369-b773a83214a0","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","provider_type":"","provider_reference_id":"","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEmail := emailhandler.NewMockEmailHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				emailHandler: mockEmail,
			}

			mockEmail.EXPECT().Get(gomock.Any(), tt.expectEmailID).Return(tt.responseEmail, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1EmailsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseEmail *email.Email

		expectEmailID uuid.UUID
		expectRes     *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/emails/eeac7ce0-00c9-11f0-99a7-2742f8cc6c8b",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			responseEmail: &email.Email{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("eeac7ce0-00c9-11f0-99a7-2742f8cc6c8b"),
				},
			},

			expectEmailID: uuid.FromStringOrNil("eeac7ce0-00c9-11f0-99a7-2742f8cc6c8b"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"eeac7ce0-00c9-11f0-99a7-2742f8cc6c8b","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","provider_type":"","provider_reference_id":"","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEmail := emailhandler.NewMockEmailHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				emailHandler: mockEmail,
			}

			mockEmail.EXPECT().Delete(gomock.Any(), tt.expectEmailID).Return(tt.responseEmail, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
