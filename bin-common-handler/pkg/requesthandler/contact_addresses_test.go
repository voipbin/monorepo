package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	cmcontact "monorepo/bin-contact-manager/models/contact"
)

func Test_ContactV1AddressWrappers(t *testing.T) {
	contactID := uuid.FromStringOrNil("3d4e5f6a-7b8c-4d9e-8f0a-2b3c4d5e6f70")
	addressID := uuid.FromStringOrNil("4e5f6a7b-8c9d-4e0f-9a1b-3c4d5e6f7081")
	customerID := uuid.FromStringOrNil("5f6a7b8c-9d0e-4f1a-8b2c-4d5e6f708192")

	addressWant := cmcontact.Address{
		AddressInput: cmcontact.AddressInput{
			Address:   commonaddress.Address{Type: commonaddress.TypeEmail, Target: "contact-a@example.com"},
			IsPrimary: true,
		},
		ID:         addressID,
		CustomerID: customerID,
		ContactID:  contactID,
	}
	const addressJSON = `{"id":"4e5f6a7b-8c9d-4e0f-9a1b-3c4d5e6f7081","customer_id":"5f6a7b8c-9d0e-4f1a-8b2c-4d5e6f708192","contact_id":"3d4e5f6a-7b8c-4d9e-8f0a-2b3c4d5e6f70","type":"email","target":"contact-a@example.com","is_primary":true}`

	contactWant := &cmcontact.Contact{Identity: identity.Identity{ID: contactID, CustomerID: customerID}, DisplayName: "Contact A"}
	const contactResponse = `{"id":"3d4e5f6a-7b8c-4d9e-8f0a-2b3c4d5e6f70","customer_id":"5f6a7b8c-9d0e-4f1a-8b2c-4d5e6f708192","display_name":"Contact A"}`

	const addressesURI = "/v1/contacts/3d4e5f6a-7b8c-4d9e-8f0a-2b3c4d5e6f70/addresses"
	const addressURI = addressesURI + "/4e5f6a7b-8c9d-4e0f-9a1b-3c4d5e6f7081"

	updateCase := func(name string, fields map[string]any, expectBody string) rpcCase {
		return rpcCase{
			name:           name,
			queue:          contactQueue,
			expectURI:      addressURI,
			expectMethod:   sock.RequestMethodPut,
			expectDataType: ContentTypeJSON,
			expectData:     expectBody,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1AddressUpdate(context.Background(), contactID, addressID, fields)
			},
			expectRes: contactWant,
		}
	}

	cases := []rpcCase{
		{
			name:           "create_primary",
			queue:          contactQueue,
			expectURI:      addressesURI,
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"type":"email","target":"contact-a@example.com","name":"work","detail":"office mailbox","is_primary":true}`,
			responseData:   addressJSON,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1AddressCreate(context.Background(), contactID, "email", "contact-a@example.com", true, "work", "office mailbox")
			},
			expectRes: &addressWant,
		},
		{
			name:           "create_not_primary",
			queue:          contactQueue,
			expectURI:      addressesURI,
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"type":"tel","target":"+15550100001","name":"","detail":"","is_primary":false}`,
			responseData:   addressJSON,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1AddressCreate(context.Background(), contactID, "tel", "+15550100001", false, "", "")
			},
			expectRes: &addressWant,
		},
		{
			name:           "get_with_items",
			queue:          contactQueue,
			expectURI:      addressesURI,
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `{"result":[` + addressJSON + `]}`,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1AddressGet(context.Background(), contactID)
			},
			expectRes: []cmcontact.Address{addressWant},
		},
		{
			name:           "get_empty_result",
			queue:          contactQueue,
			expectURI:      addressesURI,
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `{"result":[]}`,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1AddressGet(context.Background(), contactID)
			},
			expectRes: []cmcontact.Address{},
		},
		updateCase("update_target_only", map[string]any{"target": "contact-b@example.com"}, `{"target":"contact-b@example.com"}`),
		updateCase("update_is_primary_false_only", map[string]any{"is_primary": false}, `{"is_primary":false}`),
		updateCase("update_both", map[string]any{"target": "contact-b@example.com", "is_primary": true}, `{"target":"contact-b@example.com","is_primary":true}`),
		updateCase("update_empty_target_is_kept", map[string]any{"target": ""}, `{"target":""}`),
		updateCase("update_nil_fields", nil, `{}`),
		updateCase("update_empty_fields", map[string]any{}, `{}`),
		updateCase("update_wrong_types_are_dropped", map[string]any{"target": 42, "is_primary": "yes"}, `{}`),
		{
			name:           "delete",
			queue:          contactQueue,
			expectURI:      addressURI,
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1AddressDelete(context.Background(), contactID, addressID)
			},
			expectRes: contactWant,
		},
	}

	runRPCCases(t, cases)
}
