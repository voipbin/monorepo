package requesthandler

import (
	"context"
	"testing"

	cmcontact "monorepo/bin-contact-manager/models/contact"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
)

func Test_ContactV1ContactWrappers(t *testing.T) {
	customerID := uuid.FromStringOrNil("2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13")
	contactID := uuid.FromStringOrNil("4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35")
	tagID := uuid.FromStringOrNil("8a0c2e46-7a9c-4eab-b247-6c8e0a2c4f79")

	contactWant := &cmcontact.Contact{
		Identity:    identity.Identity{ID: contactID, CustomerID: customerID},
		FirstName:   "Alex",
		DisplayName: "Alex Sample",
	}
	const contactResponse = `{"id":"4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35","customer_id":"2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13","first_name":"Alex","display_name":"Alex Sample"}`

	first := "Alex"
	blank := ""

	cases := []rpcCase{
		{
			name:           "create_with_addresses_and_tags",
			queue:          contactQueue,
			expectURI:      "/v1/contacts",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13","first_name":"Alex","last_name":"Sample","display_name":"Alex Sample","company":"Example Co","job_title":"Engineer","source":"manual","external_id":"ext-1","notes":"n","addresses":[{"type":"tel","target":"+15550100","name":"mobile","detail":"d","is_primary":true},{"type":"email","target":"alex@example.invalid","name":"","detail":"","is_primary":false}],"tag_ids":["8a0c2e46-7a9c-4eab-b247-6c8e0a2c4f79"]}`,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactCreate(context.Background(), customerID, "Alex", "Sample", "Alex Sample", "Example Co", "Engineer", "manual", "ext-1", "n",
					[]cmcontact.AddressInput{
						{Address: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100", Name: "mobile", Detail: "d"}, IsPrimary: true},
						{Address: commonaddress.Address{Type: commonaddress.TypeEmail, Target: "alex@example.invalid"}},
					},
					[]uuid.UUID{tagID})
			},
			expectRes: contactWant,
		},
		{
			name:           "create_without_addresses",
			queue:          contactQueue,
			expectURI:      "/v1/contacts",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13","first_name":"","last_name":"","display_name":"","company":"","job_title":"","source":"","external_id":"","notes":""}`,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactCreate(context.Background(), customerID, "", "", "", "", "", "", "", "", nil, nil)
			},
			expectRes: contactWant,
		},
		{
			name:           "get",
			queue:          contactQueue,
			expectURI:      "/v1/contacts/4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactGet(context.Background(), contactID)
			},
			expectRes: contactWant,
		},
		{
			name:           "list",
			queue:          contactQueue,
			expectURI:      "/v1/contacts?page_token=2020-09-20T03%3A23%3A20.995000Z&page_size=7",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13","first_name":"Alex"}`,
			responseData:   `[` + contactResponse + `]`,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactList(context.Background(), "2020-09-20T03:23:20.995000Z", 7, map[cmcontact.Field]any{
					cmcontact.FieldCustomerID: customerID,
					cmcontact.FieldFirstName:  "Alex",
				})
			},
			expectRes: []cmcontact.Contact{*contactWant},
		},
		{
			name:           "update_only_set_fields",
			queue:          contactQueue,
			expectURI:      "/v1/contacts/4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35",
			expectMethod:   sock.RequestMethodPut,
			expectDataType: ContentTypeJSON,
			expectData:     `{"first_name":"Alex","company":""}`,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactUpdate(context.Background(), contactID, &first, nil, nil, &blank, nil, nil, nil)
			},
			expectRes: contactWant,
		},
		{
			name:           "delete",
			queue:          contactQueue,
			expectURI:      "/v1/contacts/4d6f8a02-3c5e-4a79-8d03-2f4a6c8e0b35",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactDelete(context.Background(), contactID)
			},
			expectRes: contactWant,
		},
		{
			name:           "lookup_by_phone_and_email_escapes_values",
			queue:          contactQueue,
			expectURI:      "/v1/contacts/lookup?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13&phone_e164=%2B15550100&email=a%2Bb%40example.invalid",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactLookup(context.Background(), customerID, "+15550100", "a+b@example.invalid")
			},
			expectRes: contactWant,
		},
		{
			name:           "lookup_by_email_only",
			queue:          contactQueue,
			expectURI:      "/v1/contacts/lookup?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13&email=alex%40example.invalid",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactLookup(context.Background(), customerID, "", "alex@example.invalid")
			},
			expectRes: contactWant,
		},
		{
			name:           "lookup_without_keys",
			queue:          contactQueue,
			expectURI:      "/v1/contacts/lookup?customer_id=2b4d6f80-1a3c-4e57-9b81-0d2f4a6c8e13",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1ContactLookup(context.Background(), customerID, "", "")
			},
			expectRes: contactWant,
		},
	}

	runRPCCases(t, cases)

	runMarshalErrorCase(t, "list_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.ContactV1ContactList(ctx, "", 7, map[cmcontact.Field]any{cmcontact.FieldFirstName: make(chan int)})
	})
}
