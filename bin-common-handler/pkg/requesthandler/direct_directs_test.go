package requesthandler

import (
	"context"
	"testing"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
)

const directQueue = "bin-manager.direct-manager.request"

func Test_DirectV1DirectWrappers(t *testing.T) {
	directID := uuid.FromStringOrNil("2f4a6c8e-0b1d-4f35-9a47-1c3e5a7c9e51")
	customerID := uuid.FromStringOrNil("6b8d0f2a-4c3e-4a79-8b61-5e7a9c1e3f73")
	resourceID := uuid.FromStringOrNil("a0c2e4f6-8d5b-4c97-b083-9a1c3e5b7d95")

	directWant := &dmdirect.Direct{
		Identity:     identity.Identity{ID: directID, CustomerID: customerID},
		ResourceType: "extension",
		ResourceID:   resourceID,
		Hash:         "hash-1",
	}
	const directResponse = `{"id":"2f4a6c8e-0b1d-4f35-9a47-1c3e5a7c9e51","customer_id":"6b8d0f2a-4c3e-4a79-8b61-5e7a9c1e3f73","resource_type":"extension","resource_id":"a0c2e4f6-8d5b-4c97-b083-9a1c3e5b7d95","hash":"hash-1"}`

	cases := []rpcCase{
		{
			name:           "create",
			queue:          directQueue,
			expectURI:      "/v1/directs",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"6b8d0f2a-4c3e-4a79-8b61-5e7a9c1e3f73","resource_type":"extension","resource_id":"a0c2e4f6-8d5b-4c97-b083-9a1c3e5b7d95"}`,
			responseData:   directResponse,
			call: func(h *requestHandler) (any, error) {
				return h.DirectV1DirectCreate(context.Background(), customerID, "extension", resourceID)
			},
			expectRes: directWant,
		},
		{
			name:           "get",
			queue:          directQueue,
			expectURI:      "/v1/directs/2f4a6c8e-0b1d-4f35-9a47-1c3e5a7c9e51",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   directResponse,
			call: func(h *requestHandler) (any, error) {
				return h.DirectV1DirectGet(context.Background(), directID)
			},
			expectRes: directWant,
		},
		{
			name:           "get_by_hash_escapes_the_path_segment",
			queue:          directQueue,
			expectURI:      "/v1/directs/by-hash/a%2Fb%20c",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   directResponse,
			call: func(h *requestHandler) (any, error) {
				return h.DirectV1DirectGetByHash(context.Background(), "a/b c")
			},
			expectRes: directWant,
		},
		{
			name:           "gets",
			queue:          directQueue,
			expectURI:      "/v1/directs?page_token=2020-09-20T03%3A23%3A20.995000Z&page_size=10",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"6b8d0f2a-4c3e-4a79-8b61-5e7a9c1e3f73","resource_type":"extension"}`,
			responseData:   `[` + directResponse + `]`,
			call: func(h *requestHandler) (any, error) {
				return h.DirectV1DirectGets(context.Background(), "2020-09-20T03:23:20.995000Z", 10, map[dmdirect.Field]any{
					dmdirect.FieldCustomerID:   customerID,
					dmdirect.FieldResourceType: "extension",
				})
			},
			expectRes: []*dmdirect.Direct{directWant},
		},
		{
			name:           "delete",
			queue:          directQueue,
			expectURI:      "/v1/directs/2f4a6c8e-0b1d-4f35-9a47-1c3e5a7c9e51",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   directResponse,
			call: func(h *requestHandler) (any, error) {
				return h.DirectV1DirectDelete(context.Background(), directID)
			},
			expectRes: directWant,
		},
		{
			name:           "regenerate",
			queue:          directQueue,
			expectURI:      "/v1/directs/2f4a6c8e-0b1d-4f35-9a47-1c3e5a7c9e51/regenerate",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeNone,
			responseData:   directResponse,
			call: func(h *requestHandler) (any, error) {
				return h.DirectV1DirectRegenerate(context.Background(), directID)
			},
			expectRes: directWant,
		},
	}

	runRPCCases(t, cases)

	runMarshalErrorCase(t, "gets_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.DirectV1DirectGets(ctx, "", 10, map[dmdirect.Field]any{dmdirect.FieldHash: make(chan int)})
	})
}
