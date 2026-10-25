package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
	fmaction "monorepo/bin-flow-manager/models/action"
	rmprovidercall "monorepo/bin-route-manager/models/providercall"
)

const routeQueue = "bin-manager.route-manager.request"

func Test_RouteV1ProviderCallWrappers(t *testing.T) {
	providerCallID := uuid.FromStringOrNil("b8c9d0e1-f2a3-4b4c-9d6e-7f8091a2b3c4")
	customerID := uuid.FromStringOrNil("c9d0e1f2-a3b4-4c5d-8e7f-8091a2b3c4d5")
	providerID := uuid.FromStringOrNil("d0e1f2a3-b4c5-4d6e-9f80-91a2b3c4d5e6")
	flowID := uuid.FromStringOrNil("e1f2a3b4-c5d6-4e7f-8091-a2b3c4d5e6f7")

	providerCallWant := &rmprovidercall.ProviderCall{
		ID:         providerCallID,
		CustomerID: customerID,
		ProviderID: providerID,
		Anonymous:  "auto",
	}
	const providerCallResponse = `{"id":"b8c9d0e1-f2a3-4b4c-9d6e-7f8091a2b3c4","customer_id":"c9d0e1f2-a3b4-4c5d-8e7f-8091a2b3c4d5","provider_id":"d0e1f2a3-b4c5-4d6e-9f80-91a2b3c4d5e6","anonymous":"auto"}`

	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100002"}}
	actions := []fmaction.Action{{Type: fmaction.TypeHangup}}

	cases := []rpcCase{
		{
			name:           "create_with_actions",
			queue:          routeQueue,
			expectURI:      "/v1/providercalls",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"c9d0e1f2-a3b4-4c5d-8e7f-8091a2b3c4d5","provider_id":"d0e1f2a3-b4c5-4d6e-9f80-91a2b3c4d5e6","flow_id":"00000000-0000-0000-0000-000000000000","actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"hangup","tm_execute":null}],"source":{"type":"tel","target":"+15550100001"},"destinations":[{"type":"tel","target":"+15550100002"}],"anonymous":"auto"}`,
			responseData:   providerCallResponse,
			call: func(h *requestHandler) (any, error) {
				return h.RouteV1ProviderCallCreate(context.Background(), customerID, providerID, uuid.Nil, actions, source, destinations, "auto")
			},
			expectRes: providerCallWant,
		},
		{
			name:           "create_with_flow_and_no_addresses",
			queue:          routeQueue,
			expectURI:      "/v1/providercalls",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"c9d0e1f2-a3b4-4c5d-8e7f-8091a2b3c4d5","provider_id":"d0e1f2a3-b4c5-4d6e-9f80-91a2b3c4d5e6","flow_id":"e1f2a3b4-c5d6-4e7f-8091-a2b3c4d5e6f7","actions":null,"source":null,"destinations":null,"anonymous":""}`,
			responseData:   providerCallResponse,
			call: func(h *requestHandler) (any, error) {
				return h.RouteV1ProviderCallCreate(context.Background(), customerID, providerID, flowID, nil, nil, nil, "")
			},
			expectRes: providerCallWant,
		},
		{
			name:           "get",
			queue:          routeQueue,
			expectURI:      "/v1/providercalls/b8c9d0e1-f2a3-4b4c-9d6e-7f8091a2b3c4",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   providerCallResponse,
			call: func(h *requestHandler) (any, error) {
				return h.RouteV1ProviderCallGet(context.Background(), providerCallID)
			},
			expectRes: providerCallWant,
		},
		{
			// the query keys are sorted by url.Values.Encode.
			name:           "gets_with_two_filters",
			queue:          routeQueue,
			expectURI:      "/v1/providercalls?customer_id=c9d0e1f2-a3b4-4c5d-8e7f-8091a2b3c4d5&page_size=20&page_token=token-1&provider_id=d0e1f2a3-b4c5-4d6e-9f80-91a2b3c4d5e6",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `[` + providerCallResponse + `]`,
			call: func(h *requestHandler) (any, error) {
				return h.RouteV1ProviderCallGets(context.Background(), "token-1", 20, map[rmprovidercall.Field]any{
					rmprovidercall.FieldProviderID: providerID,
					rmprovidercall.FieldCustomerID: customerID,
				})
			},
			expectRes: []rmprovidercall.ProviderCall{*providerCallWant},
		},
		{
			name:           "gets_without_filters",
			queue:          routeQueue,
			expectURI:      "/v1/providercalls?page_size=0&page_token=",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   `[]`,
			call: func(h *requestHandler) (any, error) {
				return h.RouteV1ProviderCallGets(context.Background(), "", 0, nil)
			},
			expectRes: []rmprovidercall.ProviderCall{},
		},
		{
			name:           "delete",
			queue:          routeQueue,
			expectURI:      "/v1/providercalls/b8c9d0e1-f2a3-4b4c-9d6e-7f8091a2b3c4",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   providerCallResponse,
			call: func(h *requestHandler) (any, error) {
				return h.RouteV1ProviderCallDelete(context.Background(), providerCallID)
			},
			expectRes: providerCallWant,
		},
	}

	runRPCCases(t, cases)

	runMarshalErrorCase(t, "create_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		badActions := []fmaction.Action{{Type: fmaction.TypeHangup, Option: map[string]any{"reason": make(chan int)}}}
		return h.RouteV1ProviderCallCreate(ctx, customerID, providerID, uuid.Nil, badActions, source, destinations, "auto")
	})
}
