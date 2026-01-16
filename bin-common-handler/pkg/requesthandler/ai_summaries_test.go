package requesthandler

import (
	"context"
	amsummary "monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AIV1SummaryList(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[amsummary.Field]any

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []amsummary.Summary
	}{
		{
			name: "normal",

			pageToken: "2020-09-20 03:23:20.995000",
			pageSize:  10,
			filters: map[amsummary.Field]any{
				amsummary.FieldDeleted:    false,
				amsummary.FieldCustomerID: uuid.FromStringOrNil("8e6595e6-0bb0-11f0-b462-43d51b0d2d1f"),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"8ea18d12-0bb0-11f0-9081-cb954d06e7e9"},{"id":"8ec63202-0bb0-11f0-b9bd-8b0b90e07b50"}]`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/summaries?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"8e6595e6-0bb0-11f0-b462-43d51b0d2d1f","deleted":false}`),
			},
			expectRes: []amsummary.Summary{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("8ea18d12-0bb0-11f0-9081-cb954d06e7e9"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("8ec63202-0bb0-11f0-b9bd-8b0b90e07b50"),
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

			res, err := h.AIV1SummaryList(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1SummaryCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		activeflowID  uuid.UUID
		onEndFlowID   uuid.UUID
		referenceType amsummary.ReferenceType
		referenceID   uuid.UUID
		language      string
		timeout       int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amsummary.Summary
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("e596f3c2-0bb1-11f0-87f7-6bb6766cfb44"),
			activeflowID:  uuid.FromStringOrNil("e5c48710-0bb1-11f0-9d7a-87b8ef0b2bc1"),
			onEndFlowID:   uuid.FromStringOrNil("a81faebc-0cbf-11f0-ac14-db723a486023"),
			referenceType: amsummary.ReferenceTypeRecording,
			referenceID:   uuid.FromStringOrNil("e5e8fce4-0bb1-11f0-8fef-db941787c5a4"),
			language:      "en-US",
			timeout:       30000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e612990a-0bb1-11f0-bb65-b38f5eb28363"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/summaries",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"e596f3c2-0bb1-11f0-87f7-6bb6766cfb44","activeflow_id":"e5c48710-0bb1-11f0-9d7a-87b8ef0b2bc1","on_end_flow_id":"a81faebc-0cbf-11f0-ac14-db723a486023","reference_type":"recording","reference_id":"e5e8fce4-0bb1-11f0-8fef-db941787c5a4","language":"en-US"}`),
			},
			expectRes: &amsummary.Summary{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e612990a-0bb1-11f0-bb65-b38f5eb28363"),
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

			cf, err := reqHandler.AIV1SummaryCreate(ctx, tt.customerID, tt.activeflowID, tt.onEndFlowID, tt.referenceType, tt.referenceID, tt.language, tt.timeout)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_AIV1SummaryGet(t *testing.T) {

	type test struct {
		name      string
		summaryID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *amsummary.Summary
	}

	tests := []test{
		{
			name:      "normal",
			summaryID: uuid.FromStringOrNil("42d92c58-0bb2-11f0-805d-07c84b69c9be"),

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/summaries/42d92c58-0bb2-11f0-805d-07c84b69c9be",
				Method: sock.RequestMethodGet,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"42d92c58-0bb2-11f0-805d-07c84b69c9be"}`),
			},
			expectRes: &amsummary.Summary{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("42d92c58-0bb2-11f0-805d-07c84b69c9be"),
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

			res, err := reqHandler.AIV1SummaryGet(ctx, tt.summaryID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1SummaryDelete(t *testing.T) {

	tests := []struct {
		name string

		aiID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amsummary.Summary
	}{
		{
			name: "normal",

			aiID: uuid.FromStringOrNil("432b67fc-0bb2-11f0-8ffb-f3f90e210b16"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"432b67fc-0bb2-11f0-8ffb-f3f90e210b16"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/summaries/432b67fc-0bb2-11f0-8ffb-f3f90e210b16",
				Method: sock.RequestMethodDelete,
			},
			expectRes: &amsummary.Summary{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("432b67fc-0bb2-11f0-8ffb-f3f90e210b16"),
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

			res, err := reqHandler.AIV1SummaryDelete(ctx, tt.aiID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
