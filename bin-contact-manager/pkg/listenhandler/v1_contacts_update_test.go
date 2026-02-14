package listenhandler

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/contacthandler"
)

func TestProcessV1ContactsPhoneNumbersIDPut(t *testing.T) {
	tests := []struct {
		name            string
		request         *sock.Request
		contactID       uuid.UUID
		phoneID         uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "update phone number",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/phone-numbers/d1111111-1111-1111-1111-111111111111",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"number":"+1-555-999-8888","type":"home","is_primary":true}`),
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
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":null,"tm_update":null,"tm_delete":null}`),
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

			mockContact.EXPECT().UpdatePhoneNumber(gomock.Any(), tt.contactID, tt.phoneID, gomock.Any()).Return(tt.responseContact, nil)

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

func TestProcessV1ContactsPhoneNumbersIDPut_ShortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Too short URI - missing phone ID
	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/phone-numbers",
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`{"number":"+1-555-999-8888"}`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsPhoneNumbersIDPut_InvalidJSON(t *testing.T) {
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

func TestProcessV1ContactsPhoneNumbersIDPut_UpdateError(t *testing.T) {
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
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`{"number":"+1-555-999-8888"}`),
	}

	mockContact.EXPECT().UpdatePhoneNumber(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("database error"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %v, want 500", res.StatusCode)
	}
}

func TestProcessV1ContactsEmailsIDPut(t *testing.T) {
	tests := []struct {
		name            string
		request         *sock.Request
		contactID       uuid.UUID
		emailID         uuid.UUID
		responseContact *contact.Contact
		expectRes       *sock.Response
	}{
		{
			name: "update email",
			request: &sock.Request{
				URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/emails/e2222222-2222-2222-2222-222222222222",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"address":"newemail@example.com","type":"personal","is_primary":true}`),
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
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c31676f0-4e69-11ec-afe3-77ba49fae527","customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","first_name":"John","last_name":"Doe","display_name":"John Doe","company":"","job_title":"","source":"manual","external_id":"","notes":"","tm_create":null,"tm_update":null,"tm_delete":null}`),
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

			mockContact.EXPECT().UpdateEmail(gomock.Any(), tt.contactID, tt.emailID, gomock.Any()).Return(tt.responseContact, nil)

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

func TestProcessV1ContactsEmailsIDPut_ShortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		contactHandler: mockContact,
	}

	// Too short URI - missing email ID
	request := &sock.Request{
		URI:      "/v1/contacts/c31676f0-4e69-11ec-afe3-77ba49fae527/emails",
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`{"address":"test@example.com"}`),
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", res.StatusCode)
	}
}

func TestProcessV1ContactsEmailsIDPut_InvalidJSON(t *testing.T) {
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

func TestProcessV1ContactsEmailsIDPut_UpdateError(t *testing.T) {
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
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`{"address":"test@example.com"}`),
	}

	mockContact.EXPECT().UpdateEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("database error"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest() error = %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %v, want 500", res.StatusCode)
	}
}
