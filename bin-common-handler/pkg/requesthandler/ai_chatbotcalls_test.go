package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	amaicall "monorepo/bin-ai-manager/models/aicall"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_AIV1AIcallStart(t *testing.T) {

	tests := []struct {
		name string

		aiID          uuid.UUID
		referenceType amaicall.ReferenceType
		referenceID   uuid.UUID
		gender        amaicall.Gender
		language      string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaicall.AIcall
	}{
		{
			name: "normal",

			aiID:          uuid.FromStringOrNil("e8604e8a-ef52-11ef-88be-43d681e412f7"),
			referenceType: amaicall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e8c3a34a-ef52-11ef-b4d1-93c7d17c08e9"),
			gender:        amaicall.GenderFemale,
			language:      "en-US",

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e8ec8062-ef52-11ef-8fe9-27921b0be03c"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/aicalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"ai_id":"e8604e8a-ef52-11ef-88be-43d681e412f7","reference_type":"call","reference_id":"e8c3a34a-ef52-11ef-b4d1-93c7d17c08e9","gender":"female","language":"en-US"}`),
			},
			expectRes: &amaicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e8ec8062-ef52-11ef-8fe9-27921b0be03c"),
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

			cf, err := reqHandler.AIV1AIcallStart(ctx, tt.aiID, tt.referenceType, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_AIV1AIcallGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64
		filters    map[string]string

		response *sock.Response

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		expectResult  []amaicall.AIcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("ccf7720e-4838-4f97-bb61-3021e14c185a"),
			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c3ac26c7-567c-4230-aaf8-d19b6fde4d6c"},{"id":"eb36875a-0d7a-4a8f-92a9-7551f4f29fd6"}]`),
			},

			"/v1/aicalls?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&customer_id=ccf7720e-4838-4f97-bb61-3021e14c185a",
			string(outline.QueueNameAIRequest),
			&sock.Request{
				URI:    fmt.Sprintf("/v1/aicalls?page_token=%s&page_size=10&customer_id=ccf7720e-4838-4f97-bb61-3021e14c185a&filter_deleted=false", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method: sock.RequestMethodGet,
			},
			[]amaicall.AIcall{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("c3ac26c7-567c-4230-aaf8-d19b6fde4d6c"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("eb36875a-0d7a-4a8f-92a9-7551f4f29fd6"),
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
			h := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: h,
			}
			ctx := context.Background()

			h.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AIV1AIcallGetsByCustomerID(ctx, tt.customerID, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_AIV1AIcallGet(t *testing.T) {

	type test struct {
		name string

		aicallID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *amaicall.AIcall
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("d3937170-ee3b-40d0-8b81-4261e5bb5ba4"),

			string(outline.QueueNameAIRequest),
			&sock.Request{
				URI:    "/v1/aicalls/d3937170-ee3b-40d0-8b81-4261e5bb5ba4",
				Method: sock.RequestMethodGet,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"d3937170-ee3b-40d0-8b81-4261e5bb5ba4"}`),
			},
			&amaicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d3937170-ee3b-40d0-8b81-4261e5bb5ba4"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AIV1AIcallGet(ctx, tt.aicallID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIcallDelete(t *testing.T) {

	tests := []struct {
		name string

		aicallID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaicall.AIcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("6078c492-25e6-4f31-baa0-2fef98379db7"),

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"6078c492-25e6-4f31-baa0-2fef98379db7"}`),
			},

			string(outline.QueueNameAIRequest),
			&sock.Request{
				URI:    "/v1/aicalls/6078c492-25e6-4f31-baa0-2fef98379db7",
				Method: sock.RequestMethodDelete,
			},
			&amaicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("6078c492-25e6-4f31-baa0-2fef98379db7"),
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

			res, err := reqHandler.AIV1AIcallDelete(ctx, tt.aicallID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
