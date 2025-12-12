package requesthandler

import (
	"context"
	"reflect"
	"testing"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	aisummary "monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/service"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_AIV1ServiceTypeChabotcallStart(t *testing.T) {

	tests := []struct {
		name string

		aiID           uuid.UUID
		activeflowID   uuid.UUID
		referenceType  amaicall.ReferenceType
		referenceID    uuid.UUID
		resume         bool
		gender         amaicall.Gender
		language       string
		requestTimeout int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *service.Service
	}{
		{
			name: "normal",

			aiID:           uuid.FromStringOrNil("9469e101-d269-4895-9679-fe49531f7c12"),
			activeflowID:   uuid.FromStringOrNil("db21d8b6-fbab-11ed-8d21-332400f26ee4"),
			referenceType:  amaicall.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("865089bd-dc1b-45d5-89af-4a09c1d90cea"),
			resume:         true,
			gender:         "female",
			language:       "en-US",
			requestTimeout: 5000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"134c25c9-c9f9-4800-83bb-b5eaa84bb4ab"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/services/type/aicall",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"ai_id":"9469e101-d269-4895-9679-fe49531f7c12","activeflow_id":"db21d8b6-fbab-11ed-8d21-332400f26ee4","reference_type":"call","reference_id":"865089bd-dc1b-45d5-89af-4a09c1d90cea","resume":true,"gender":"female","language":"en-US"}`),
			},
			expectRes: &service.Service{
				ID: uuid.FromStringOrNil("134c25c9-c9f9-4800-83bb-b5eaa84bb4ab"),
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

			cf, err := reqHandler.AIV1ServiceTypeAIcallStart(ctx, tt.aiID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.resume, tt.gender, tt.language, tt.requestTimeout)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_AIV1ServiceTypeSummaryStart(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		activeflowID   uuid.UUID
		onEndFlowID    uuid.UUID
		referenceType  aisummary.ReferenceType
		referenceID    uuid.UUID
		language       string
		requestTimeout int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *service.Service
	}{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("80a34c96-0cb4-11f0-9765-a7c52a032ca2"),
			activeflowID:   uuid.FromStringOrNil("80da784c-0cb4-11f0-8aa8-5f4096804e3b"),
			onEndFlowID:    uuid.FromStringOrNil("811137a6-0cb4-11f0-8f10-b34b9c144dfc"),
			referenceType:  aisummary.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("813a7a1c-0cb4-11f0-9f22-ebb78dd1d30c"),
			language:       "en-US",
			requestTimeout: 5000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"816332c2-0cb4-11f0-a74f-f3e41dabbe79"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/services/type/summary",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"80a34c96-0cb4-11f0-9765-a7c52a032ca2","activeflow_id":"80da784c-0cb4-11f0-8aa8-5f4096804e3b","on_end_flow_id":"811137a6-0cb4-11f0-8f10-b34b9c144dfc","reference_type":"call","reference_id":"813a7a1c-0cb4-11f0-9f22-ebb78dd1d30c","language":"en-US"}`),
			},
			expectRes: &service.Service{
				ID: uuid.FromStringOrNil("816332c2-0cb4-11f0-a74f-f3e41dabbe79"),
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

			cf, err := reqHandler.AIV1ServiceTypeSummaryStart(
				ctx,
				tt.customerID,
				tt.activeflowID,
				tt.onEndFlowID,
				tt.referenceType,
				tt.referenceID,
				tt.language,
				tt.requestTimeout,
			)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_AIV1ServiceTypeTaskStart(t *testing.T) {

	tests := []struct {
		name string

		aiID         uuid.UUID
		activeflowID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *service.Service
	}{
		{
			name: "normal",

			aiID:         uuid.FromStringOrNil("17e0c1ca-d7a2-11f0-b895-272756e82e9c"),
			activeflowID: uuid.FromStringOrNil("18093aba-d7a2-11f0-8461-7f7066a41d60"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1832a922-d7a2-11f0-ab7c-af445f822391"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/services/type/task",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"ai_id":"17e0c1ca-d7a2-11f0-b895-272756e82e9c","activeflow_id":"18093aba-d7a2-11f0-8461-7f7066a41d60"}`),
			},
			expectRes: &service.Service{
				ID: uuid.FromStringOrNil("1832a922-d7a2-11f0-ab7c-af445f822391"),
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

			cf, err := reqHandler.AIV1ServiceTypeTaskStart(ctx, tt.aiID, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}
