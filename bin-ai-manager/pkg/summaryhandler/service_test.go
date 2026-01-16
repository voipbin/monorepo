package summaryhandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmaction "monorepo/bin-flow-manager/models/action"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceStart_referencetype_call(t *testing.T) {
	tests := []struct {
		name string

		customerID    uuid.UUID
		activeflowID  uuid.UUID
		onEndFlowID   uuid.UUID
		referenceType summary.ReferenceType
		referenceID   uuid.UUID
		language      string

		responseCall *cmcall.Call
		responseUUID uuid.UUID

		expectedSummary   *summary.Summary
		expectedVariables map[string]string
		expectedRes       *commonservice.Service
	}{
		{
			name: "normal - english female",

			customerID:    uuid.FromStringOrNil("d636df18-0cb6-11f0-a92f-2f5d1db21b1c"),
			activeflowID:  uuid.FromStringOrNil("d66d357c-0cb6-11f0-8f8f-57b1732573d0"),
			onEndFlowID:   uuid.FromStringOrNil("d6968850-0cb6-11f0-97ab-6bd052826646"),
			referenceType: summary.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("d6bfb5ea-0cb6-11f0-8db4-a32464ec8a1c"),
			language:      "en-US",

			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d6bfb5ea-0cb6-11f0-8db4-a32464ec8a1c"),
				},
			},
			responseUUID: uuid.FromStringOrNil("d6edbfda-0cb6-11f0-bc8f-ffb8645d4a00"),

			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d6edbfda-0cb6-11f0-bc8f-ffb8645d4a00"),
					CustomerID: uuid.FromStringOrNil("d636df18-0cb6-11f0-a92f-2f5d1db21b1c"),
				},
				ActiveflowID:  uuid.FromStringOrNil("d66d357c-0cb6-11f0-8f8f-57b1732573d0"),
				OnEndFlowID:   uuid.FromStringOrNil("d6968850-0cb6-11f0-97ab-6bd052826646"),
				ReferenceType: summary.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("d6bfb5ea-0cb6-11f0-8db4-a32464ec8a1c"),
				Status:        summary.StatusProgressing,
				Language:      "en-US",
			},
			expectedVariables: map[string]string{
				variableSummaryID:            "d6edbfda-0cb6-11f0-bc8f-ffb8645d4a00",
				variableSummaryReferenceType: string(summary.ReferenceTypeCall),
				variableSummaryReferenceID:   "d6bfb5ea-0cb6-11f0-8db4-a32464ec8a1c",
				variableSummaryLanguage:      "en-US",
				variableSummaryContent:       "",
			},
			expectedRes: &commonservice.Service{
				ID:          uuid.FromStringOrNil("d6edbfda-0cb6-11f0-bc8f-ffb8645d4a00"),
				Type:        commonservice.TypeAISummary,
				PushActions: []fmaction.Action{},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &summaryHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)
			mockDB.EXPECT().SummaryList(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockReq.EXPECT().TranscribeV1TranscribeStart(
				ctx,
				cmcustomer.IDAIManager,
				tt.activeflowID,
				uuid.Nil,
				tmtranscribe.ReferenceTypeCall,
				tt.referenceID,
				tt.language,
				tmtranscribe.DirectionBoth,
				gomock.Any(),
			).Return(&tmtranscribe.Transcribe{}, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().SummaryCreate(ctx, tt.expectedSummary).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseUUID).Return(tt.expectedSummary, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectedVariables).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.customerID, summary.EventTypeCreated, tt.expectedSummary)

			res, err := h.ServiceStart(
				ctx,
				tt.customerID,
				tt.activeflowID,
				tt.onEndFlowID,
				tt.referenceType,
				tt.referenceID,
				tt.language,
			)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Expected result %v, got %v", tt.expectedRes, res)
			}

			time.Sleep(100 * time.Millisecond)
		})
	}
}
