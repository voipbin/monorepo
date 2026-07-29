package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/aihandler"
)

func Test_processV1AIsIDActivateInsightPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		expectAIID uuid.UUID
		responseAI *ai.AI
		expectRes  *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/ais/d1d1d1d1-0000-0000-0000-000000000001/activate_insight",
				Method: sock.RequestMethodPost,
			},

			expectAIID: uuid.FromStringOrNil("d1d1d1d1-0000-0000-0000-000000000001"),
			responseAI: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d1d1d1d1-0000-0000-0000-000000000001"),
				},
				Type:            ai.TypeInsight,
				IsInsightActive: true,
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d1d1d1d1-0000-0000-0000-000000000001","customer_id":"00000000-0000-0000-0000-000000000000","type":"insight","is_insight_active":true,"rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","direct_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				aiHandler:   mockAI,
			}

			mockAI.EXPECT().ActivateInsight(gomock.Any(), tt.expectAIID).Return(tt.responseAI, nil).Times(1)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes.Data, res.Data)
			}
		})
	}
}

// Filter-hop regression test (design round-3 finding).
//
// processV1AIsGet re-parses incoming filters through
// utilhandler.ConvertFilters[ai.FieldStruct, ai.Field], which treats
// ai.FieldStruct as an ALLOWLIST and silently drops any key not declared there
// — no error, no warning. If ai.FieldStruct ever loses its IsInsightActive
// entry, the Case panel's "which Insight AI is active?" query would silently
// degrade to an unfiltered most-recently-created list.
//
// A unit test at the servicehandler layer would not catch that; this test
// asserts the filter actually survives the listenhandler hop and reaches
// aiHandler.List.
func Test_processV1AIsGet_IsInsightActiveFilterSurvivesConversion(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)

	h := &listenHandler{
		sockHandler: mockSock,
		aiHandler:   mockAI,
	}

	customerID := uuid.FromStringOrNil("d2d2d2d2-0000-0000-0000-000000000001")

	request := &sock.Request{
		URI:      "/v1/ais?page_size=1",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
		Data: []byte(`{"customer_id":"d2d2d2d2-0000-0000-0000-000000000001","type":"insight","is_insight_active":true,"deleted":false}`),
	}

	expectFilters := map[ai.Field]any{
		ai.FieldCustomerID: customerID,
		// ConvertFilters leaves string-kinded values as plain strings.
		ai.FieldType:            string(ai.TypeInsight),
		ai.FieldIsInsightActive: true,
		ai.FieldDeleted:         false,
	}

	mockAI.EXPECT().
		List(gomock.Any(), uint64(1), "", expectFilters).
		Return([]*ai.AI{}, nil).
		Times(1)

	if _, err := h.processRequest(request); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}
