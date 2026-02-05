package listenhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/contacthandler"
)

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestProcessV1ContactsGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		pageSize  uint64
		pageToken string

		contacts  []*contact.Contact
		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&page_size=10&page_token=2021-11-23T17:55:39.712000Z",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			pageSize:  10,
			pageToken: "2021-11-23T17:55:39.712000Z",

			contacts: []*contact.Contact{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
						CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
					},
					FirstName:   "John",
					LastName:    "Doe",
					DisplayName: "John Doe",
					Source:      "manual",
					TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
					TMUpdate:    nil,
					TMDelete:    nil,
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().List(gomock.Any(), tt.pageSize, tt.pageToken, gomock.Any()).Return(tt.contacts, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"Jane","last_name":"Smith","display_name":"Jane Smith","source":"manual"}`),
			},

			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "Jane",
				LastName:    "Smith",
				DisplayName: "Jane Smith",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"Jane","last_name":"Smith","display_name":"Jane Smith","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().Create(gomock.Any(), gomock.Any()).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		contactID       uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			contactID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().Get(gomock.Any(), tt.contactID).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsIDPut(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		contactID       uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"first_name":"Updated","display_name":"Updated Name"}`),
			},

			contactID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "Updated",
				LastName:    "Doe",
				DisplayName: "Updated Name",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    timePtr(time.Date(2021, 11, 24, 10, 0, 0, 0, time.UTC)),
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"Updated","last_name":"Doe","display_name":"Updated Name","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":"2021-11-24T10:00:00Z","tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().Update(gomock.Any(), tt.contactID, gomock.Any()).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		contactID       uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			contactID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    timePtr(time.Date(2021, 11, 24, 10, 0, 0, 0, time.UTC)),
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":"2021-11-24T10:00:00Z"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().Delete(gomock.Any(), tt.contactID).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsLookupGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		customerID      uuid.UUID
		phoneE164       string
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "lookup by phone",
			request: &sock.Request{
				URI:      "/v1/contacts/lookup?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&phone_e164=%2B15551234567",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			customerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
			phoneE164:  "+15551234567",
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().LookupByPhone(gomock.Any(), tt.customerID, tt.phoneE164).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsPhoneNumbersPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		contactID       uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/phone-numbers",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"number":"+1-555-123-4567","number_e164":"+15551234567","type":"mobile","is_primary":true}`),
			},

			contactID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().AddPhoneNumber(gomock.Any(), tt.contactID, gomock.Any()).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsTagsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		contactID       uuid.UUID
		tagID           uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/tags",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"tag_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}`),
			},

			contactID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			tagID:     uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().AddTag(gomock.Any(), tt.contactID, tt.tagID).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsPhoneNumbersIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		contactID       uuid.UUID
		phoneID         uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/phone-numbers/d1111111-1111-1111-1111-111111111111",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			contactID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			phoneID:   uuid.FromStringOrNil("d1111111-1111-1111-1111-111111111111"),
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().RemovePhoneNumber(gomock.Any(), tt.contactID, tt.phoneID).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsEmailsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		contactID       uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/emails",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"address":"john@example.com","type":"work","is_primary":true}`),
			},

			contactID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().AddEmail(gomock.Any(), tt.contactID, gomock.Any()).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsEmailsIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		contactID       uuid.UUID
		emailID         uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/emails/e2222222-2222-2222-2222-222222222222",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			contactID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			emailID:   uuid.FromStringOrNil("e2222222-2222-2222-2222-222222222222"),
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().RemoveEmail(gomock.Any(), tt.contactID, tt.emailID).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsTagsIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		contactID       uuid.UUID
		tagID           uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/tags/a3333333-3333-3333-3333-333333333333",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			contactID: uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			tagID:     uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333"),
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().RemoveTag(gomock.Any(), tt.contactID, tt.tagID).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessV1ContactsLookupGetByEmail(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		customerID      uuid.UUID
		email           string
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "lookup by email",
			request: &sock.Request{
				URI:      "/v1/contacts/lookup?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&email=john%40example.com",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			customerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
			email:      "john@example.com",
			responseContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
					CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Source:      "manual",
				TMCreate:    timePtr(time.Date(2021, 11, 23, 17, 55, 39, 712000000, time.UTC)),
				TMUpdate:    nil,
				TMDelete:    nil,
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":"2021-11-23T17:55:39.712Z","tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().LookupByEmail(gomock.Any(), tt.customerID, tt.email).Return(tt.responseContact, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", string(tt.expectRes.Data), string(res.Data))
			}
		})
	}
}

func TestProcessRequestNotFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		expectRes *sock.Response
	}{
		{
			name: "not found",
			request: &sock.Request{
				URI:      "/v1/unknown",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			expectRes: &sock.Response{
				StatusCode: 404,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				contactHandler: mockContact,
			}

			ctx := context.Background()
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			_ = ctx

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// Error case tests

func TestProcessV1ContactsGet_ListError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&page_size=10",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
	}

	mockContact.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("database error"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %v, want 500", res.StatusCode)
	}
}

func TestProcessV1ContactsPost_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`invalid json`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsPost_CreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"Test"}`),
	}

	mockContact.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("database error"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %v, want 500", res.StatusCode)
	}
}

func TestProcessV1ContactsIDPut_ShortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Test with a short URI that doesn't have enough segments
	request := &sock.Request{
		URI:      "/v1/c",
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`{"first_name":"Test"}`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	// Should return 404 for unrecognized path
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsIDGet_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
	}

	mockContact.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %v, want 500", res.StatusCode)
	}
}

func TestProcessV1ContactsIDPut_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527",
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`invalid json`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsIDPut_UpdateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527",
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`{"first_name":"Updated"}`),
	}

	mockContact.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("database error"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %v, want 500", res.StatusCode)
	}
}

func TestProcessV1ContactsIDDelete_DeleteError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527",
		Method:   sock.RequestMethodDelete,
		DataType: "application/json",
	}

	mockContact.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsLookupGet_InvalidCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/lookup?customer_id=invalid&phone_e164=%2B15551234567",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsLookupGet_MissingPhoneAndEmail(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/lookup?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsLookupGet_LookupError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/lookup?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&phone_e164=%2B15551234567",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
	}

	mockContact.EXPECT().LookupByPhone(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsPhoneNumbersPost_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/phone-numbers",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`invalid json`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsPhoneNumbersPost_AddError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/phone-numbers",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"number":"+1-555-123-4567","number_e164":"+15551234567"}`),
	}

	mockContact.EXPECT().AddPhoneNumber(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("database error"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %v, want 500", res.StatusCode)
	}
}

func TestProcessV1ContactsPhoneNumbersIDDelete_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/phone-numbers/d1111111-1111-1111-1111-111111111111",
		Method:   sock.RequestMethodDelete,
		DataType: "application/json",
	}

	mockContact.EXPECT().RemovePhoneNumber(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsEmailsPost_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/emails",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`invalid json`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsEmailsPost_AddError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/emails",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"address":"john@example.com"}`),
	}

	mockContact.EXPECT().AddEmail(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("database error"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %v, want 500", res.StatusCode)
	}
}

func TestProcessV1ContactsEmailsIDDelete_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/emails/e2222222-2222-2222-2222-222222222222",
		Method:   sock.RequestMethodDelete,
		DataType: "application/json",
	}

	mockContact.EXPECT().RemoveEmail(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsTagsPost_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/tags",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`invalid json`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsTagsPost_AddError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/tags",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"tag_id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}`),
	}

	mockContact.EXPECT().AddTag(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("database error"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %v, want 500", res.StatusCode)
	}
}

func TestProcessV1ContactsTagsIDDelete_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/tags/a3333333-3333-3333-3333-333333333333",
		Method:   sock.RequestMethodDelete,
		DataType: "application/json",
	}

	mockContact.EXPECT().RemoveTag(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsIDPut_AllFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527",
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`{"first_name":"Updated","last_name":"User","display_name":"Updated User","company":"Acme","job_title":"Engineer","external_id":"ext-123","notes":"Some notes"}`),
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
		},
		FirstName:   "Updated",
		LastName:    "User",
		DisplayName: "Updated User",
		Company:     "Acme",
		JobTitle:    "Engineer",
		ExternalID:  "ext-123",
		Notes:       "Some notes",
		Source:      "manual",
	}

	mockContact.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(responseContact, nil)

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("processRequest() StatusCode = %v, want 200", res.StatusCode)
	}
}

func TestProcessV1ContactsPost_WithPhoneAndEmail(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"Jane","phone_numbers":[{"number":"+1-555-123-4567","number_e164":"+15551234567","type":"mobile","is_primary":true}],"emails":[{"address":"jane@example.com","type":"work","is_primary":true}],"tag_ids":["aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"]}`),
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
		},
		FirstName: "Jane",
	}

	mockContact.EXPECT().Create(gomock.Any(), gomock.Any()).Return(responseContact, nil)

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("processRequest() StatusCode = %v, want 200", res.StatusCode)
	}
}

func TestNewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := NewListenHandler(mockSock, mockContact)
	if h == nil {
		t.Error("NewListenHandler() returned nil")
	}
}

func TestRun_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", "contact-manager", false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	err := h.Run("test-queue", "delay-exchange")
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}

func TestRun_QueueCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(fmt.Errorf("queue create error"))

	err := h.Run("test-queue", "delay-exchange")
	if err == nil {
		t.Error("Run() expected error for queue create failure")
	}
}

func TestProcessV1ContactsGet_WithValidFilters(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	customerID := uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9")
	contacts := []*contact.Contact{
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"),
				CustomerID: customerID,
			},
			FirstName:   "John",
			LastName:    "Doe",
			DisplayName: "John Doe",
		},
	}

	// Test with valid filter field name (customer_id)
	request := &sock.Request{
		URI:      "/v1/contacts?page_size=10",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
		Data:     []byte(`{"filters":[{"field":"customer_id","operator":"equal","value":"92883d56-7fe3-11ec-8931-37d08180a2b9"}]}`),
	}

	mockContact.EXPECT().List(gomock.Any(), uint64(10), "", gomock.Any()).Return(contacts, nil)

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("processRequest() StatusCode = %v, want 200", res.StatusCode)
	}
}

func TestProcessV1ContactsGet_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Test with invalid JSON in body
	request := &sock.Request{
		URI:      "/v1/contacts?page_size=10",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
		Data:     []byte(`{invalid json`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %v, want 400", res.StatusCode)
	}
}

func TestProcessV1ContactsIDGet_InvalidURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Test with invalid UUID format - should still parse but return nil UUID
	request := &sock.Request{
		URI:      "/v1/contacts/invalid-uuid",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
	}

	// UUID parsing will return nil UUID, which causes 404
	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	// Should return 404 since the regex won't match invalid UUID format
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsIDPut_EmptyFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527",
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`{}`), // Empty update, no fields
	}

	responseContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
			CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
		},
	}

	mockContact.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(responseContact, nil)

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("processRequest() StatusCode = %v, want 200", res.StatusCode)
	}
}

func TestProcessV1ContactsPhoneNumbersPost_ShortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Too short URI
	request := &sock.Request{
		URI:      "/v1/contacts/phone-numbers",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{}`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	// Should return 404 since the URI doesn't match the expected pattern
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsPhoneNumbersIDDelete_ShortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Missing phone ID in URI
	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/phone-numbers",
		Method:   sock.RequestMethodDelete,
		DataType: "application/json",
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	// Should return 404 since the URI doesn't match DELETE pattern (needs phone ID)
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsEmailsPost_ShortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Missing contact ID
	request := &sock.Request{
		URI:      "/v1/contacts/emails",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{}`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsEmailsIDDelete_ShortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Missing email ID
	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/emails",
		Method:   sock.RequestMethodDelete,
		DataType: "application/json",
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsTagsPost_ShortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Missing contact ID
	request := &sock.Request{
		URI:      "/v1/contacts/tags",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{}`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsTagsIDDelete_ShortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Missing tag ID
	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/tags",
		Method:   sock.RequestMethodDelete,
		DataType: "application/json",
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsLookupGet_ByEmail_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts/lookup?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&email=test%40example.com",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
	}

	mockContact.EXPECT().LookupByEmail(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsGet_EmptyResult(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	request := &sock.Request{
		URI:      "/v1/contacts?page_size=10",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
		Data:     []byte(`{}`),
	}

	// Return empty list
	mockContact.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*contact.Contact{}, nil)

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("processRequest() StatusCode = %v, want 200", res.StatusCode)
	}
}

func TestProcessV1ContactsIDDelete_ShortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Too short URI - missing ID
	request := &sock.Request{
		URI:      "/v1/contacts",
		Method:   sock.RequestMethodDelete,
		DataType: "application/json",
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	// Should return 404 since the URI doesn't match the expected pattern
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}
