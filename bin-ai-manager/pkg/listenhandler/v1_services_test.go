package listenhandler

import (
	reflect "reflect"
	"testing"

	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/aicallhandler"
	"monorepo/bin-ai-manager/pkg/summaryhandler"
)

func Test_processV1ServicesTypeAIcallPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		responseService *commonservice.Service

		expectedAIID          uuid.UUID
		expectedActiveflowID  uuid.UUID
		expectedReferenceType aicall.ReferenceType
		expectedReferenceID   uuid.UUID
		expectedGender        aicall.Gender
		expectedLanguage      string
		expectedResume        bool

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/services/type/aicall",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"71db8f9c-abde-475e-a060-dc95e63281c3","ai_id":"e7f085d0-c7d9-4da4-9992-eda14282cb86","activeflow_id":"80a5199e-fba5-11ed-90aa-6b9821d2ad5b","reference_type":"call","reference_id":"10662882-5ff8-4788-a605-55614dc8d330","gender":"female","language":"en-US","resume":true}`),
			},

			responseService: &commonservice.Service{
				ID: uuid.FromStringOrNil("9d5b7e72-2cc9-4868-bfab-c8e758cd5045"),
			},

			expectedAIID:          uuid.FromStringOrNil("e7f085d0-c7d9-4da4-9992-eda14282cb86"),
			expectedActiveflowID:  uuid.FromStringOrNil("80a5199e-fba5-11ed-90aa-6b9821d2ad5b"),
			expectedReferenceType: aicall.ReferenceTypeCall,
			expectedReferenceID:   uuid.FromStringOrNil("10662882-5ff8-4788-a605-55614dc8d330"),
			expectedGender:        aicall.GenderFemale,
			expectedLanguage:      "en-US",
			expectedResume:        true,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9d5b7e72-2cc9-4868-bfab-c8e758cd5045","type":"","push_actions":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAIcall := aicallhandler.NewMockAIcallHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				aicallHandler: mockAIcall,
			}

			mockAIcall.EXPECT().ServiceStart(gomock.Any(), tt.expectedAIID, tt.expectedActiveflowID, tt.expectedReferenceType, tt.expectedReferenceID, tt.expectedGender, tt.expectedLanguage, tt.expectedResume).Return(tt.responseService, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1ServicesTypeSummaryPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		responseService *commonservice.Service

		expectedCustomerID    uuid.UUID
		expectedActiveflowID  uuid.UUID
		expectedOnEndFlowID   uuid.UUID
		expectedReferenceType summary.ReferenceType
		expectedReferenceID   uuid.UUID
		expectedLanguage      string

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/services/type/summary",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"dfc75222-0cae-11f0-b912-c31db4ee4c89","activeflow_id":"dfa33068-0cae-11f0-8a14-cf95de241a6c","on_end_flow_id":"df797bba-0cae-11f0-ad2a-c336f03a042c","reference_type":"call","reference_id":"df4b9d30-0cae-11f0-8a19-9b60da0bf67a","language":"en-US"}`),
			},

			responseService: &commonservice.Service{
				ID: uuid.FromStringOrNil("dfec07fc-0cae-11f0-b0e8-b308c2bf4c33"),
			},

			expectedCustomerID:    uuid.FromStringOrNil("dfc75222-0cae-11f0-b912-c31db4ee4c89"),
			expectedActiveflowID:  uuid.FromStringOrNil("dfa33068-0cae-11f0-8a14-cf95de241a6c"),
			expectedOnEndFlowID:   uuid.FromStringOrNil("df797bba-0cae-11f0-ad2a-c336f03a042c"),
			expectedReferenceType: summary.ReferenceTypeCall,
			expectedReferenceID:   uuid.FromStringOrNil("df4b9d30-0cae-11f0-8a19-9b60da0bf67a"),
			expectedLanguage:      "en-US",

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dfec07fc-0cae-11f0-b0e8-b308c2bf4c33","type":"","push_actions":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSummary := summaryhandler.NewMockSummaryHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				summaryHandler: mockSummary,
			}

			mockSummary.EXPECT().ServiceStart(
				gomock.Any(),
				tt.expectedCustomerID,
				tt.expectedActiveflowID,
				tt.expectedOnEndFlowID,
				tt.expectedReferenceType,
				tt.expectedReferenceID,
				tt.expectedLanguage,
			).Return(tt.responseService, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
