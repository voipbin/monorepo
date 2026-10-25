package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	cmcontact "monorepo/bin-contact-manager/models/contact"
)

func Test_ContactV1TagWrappers(t *testing.T) {
	contactID := uuid.FromStringOrNil("0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d")
	tagID := uuid.FromStringOrNil("1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e")
	customerID := uuid.FromStringOrNil("2c3d4e5f-6a7b-4c8d-8e9f-1a2b3c4d5e6f")

	contactWant := &cmcontact.Contact{Identity: identity.Identity{ID: contactID, CustomerID: customerID}, DisplayName: "Contact A"}
	const contactResponse = `{"id":"0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d","customer_id":"2c3d4e5f-6a7b-4c8d-8e9f-1a2b3c4d5e6f","display_name":"Contact A"}`

	cases := []rpcCase{
		{
			name:           "add",
			queue:          contactQueue,
			expectURI:      "/v1/contacts/0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d/tags",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"tag_id":"1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e"}`,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1TagAdd(context.Background(), contactID, tagID)
			},
			expectRes: contactWant,
		},
		{
			name:           "remove",
			queue:          contactQueue,
			expectURI:      "/v1/contacts/0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d/tags/1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   contactResponse,
			call: func(h *requestHandler) (any, error) {
				return h.ContactV1TagRemove(context.Background(), contactID, tagID)
			},
			expectRes: contactWant,
		},
	}

	runRPCCases(t, cases)
}
