package requesthandler

import (
	"context"
	"testing"

	cmcontact "monorepo/bin-contact-manager/models/contact"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
)

const contactQueue = "bin-manager.contact-manager.request"

func Test_ContactV1ContactAddressWrappers(t *testing.T) {
	customerID := uuid.FromStringOrNil("2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13")
	contactID := uuid.FromStringOrNil("4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35")
	addressID := uuid.FromStringOrNil("6f8a0c24-5e7a-4c9b-a025-4a6c8e0a2d57")

	addressWant := &cmcontact.Address{
		AddressInput: cmcontact.AddressInput{
			Address:   commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"},
			IsPrimary: true,
		},
		ID:         addressID,
		CustomerID: customerID,
		ContactID:  contactID,
	}
	const addressResponse = `{"id":"6f8a0c24-5e7a-4c9b-a025-4a6c8e0a2d57","customer_id":"2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13","contact_id":"4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35","type":"tel","target":"+15550100","is_primary":true}`

	newTarget := "+15550199"
	primary := true

	cases := []rpcCase{
		{
			name:           "list_with_all_filters",
			queue:          contactQueue,
			expectURI:      "/v1/contact_addresses?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13&contact_id=4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35&type=tel&target=+15550100&page_token=token-1&page_size=20",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `{"result":[` + addressResponse + `]}`,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactAddressList(context.Background(), customerID, map[string]any{
					"contact_id": contactID,
					"type":       "tel",
					"target":     "+15550100",
				}, "token-1", 20)
			},
			expectRes: []cmcontact.Address{*addressWant},
		},
		{
			name:           "list_ignores_empty_and_mistyped_filters",
			queue:          contactQueue,
			expectURI:      "/v1/contact_addresses?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `{"result":null}`,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactAddressList(context.Background(), customerID, map[string]any{
					"contact_id": uuid.Nil,
					"type":       "",
					"target":     123,
				}, "", 0)
			},
			expectRes: []cmcontact.Address(nil),
		},
		{
			name:           "list_page_size_one",
			queue:          contactQueue,
			expectURI:      "/v1/contact_addresses?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13&page_size=1",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `{"result":[]}`,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactAddressList(context.Background(), customerID, nil, "", 1)
			},
			expectRes: []cmcontact.Address{},
		},
		{
			name:           "list_ignores_mistyped_contact_id",
			queue:          contactQueue,
			expectURI:      "/v1/contact_addresses?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13&type=email",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `{"result":[]}`,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactAddressList(context.Background(), customerID, map[string]any{
					"contact_id": contactID.String(),
					"type":       "email",
				}, "", 0)
			},
			expectRes: []cmcontact.Address{},
		},
		{
			name:           "create",
			queue:          contactQueue,
			expectURI:      "/v1/contact_addresses",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13","contact_id":"4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35","type":"tel","target":"+15550100","name":"mobile","detail":"note","is_primary":true}`,
			responseData:   addressResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactAddressCreate(context.Background(), customerID, contactID, "tel", "+15550100", true, "mobile", "note")
			},
			expectRes: addressWant,
		},
		{
			name:           "claim",
			queue:          contactQueue,
			expectURI:      "/v1/contact_addresses/6f8a0c24-5e7a-4c9b-a025-4a6c8e0a2d57/claim?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"contact_id":"4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35","force":true}`,
			responseData:   addressResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactAddressClaim(context.Background(), customerID, addressID, contactID, true)
			},
			expectRes: addressWant,
		},
		{
			name:           "get",
			queue:          contactQueue,
			expectURI:      "/v1/contact_addresses/6f8a0c24-5e7a-4c9b-a025-4a6c8e0a2d57?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   addressResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactAddressGet(context.Background(), customerID, addressID)
			},
			expectRes: addressWant,
		},
		{
			name:           "update_known_fields",
			queue:          contactQueue,
			expectURI:      "/v1/contact_addresses/6f8a0c24-5e7a-4c9b-a025-4a6c8e0a2d57?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13&contact_id=4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35",
			expectMethod:   sock.RequestMethodPut,
			expectDataType: ContentTypeJSON,
			expectData:     `{"target":"+15550199","is_primary":true}`,
			responseData:   addressResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactAddressUpdate(context.Background(), customerID, contactID, addressID, map[string]any{
					"target":     newTarget,
					"is_primary": primary,
				})
			},
			expectRes: addressWant,
		},
		{
			name:           "update_drops_unknown_and_mistyped_fields",
			queue:          contactQueue,
			expectURI:      "/v1/contact_addresses/6f8a0c24-5e7a-4c9b-a025-4a6c8e0a2d57?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13&contact_id=4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35",
			expectMethod:   sock.RequestMethodPut,
			expectDataType: ContentTypeJSON,
			expectData:     `{}`,
			responseData:   addressResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactAddressUpdate(context.Background(), customerID, contactID, addressID, map[string]any{
					"target":     42,
					"is_primary": "yes",
					"type":       "email",
				})
			},
			expectRes: addressWant,
		},
		{
			name:           "delete",
			queue:          contactQueue,
			expectURI:      "/v1/contact_addresses/6f8a0c24-5e7a-4c9b-a025-4a6c8e0a2d57?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13&contact_id=4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   addressResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactAddressDelete(context.Background(), customerID, contactID, addressID)
			},
			expectRes: addressWant,
		},
	}

	runRPCCases(t, cases)
}
