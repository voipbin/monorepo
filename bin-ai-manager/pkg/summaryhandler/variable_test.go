package summaryhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/summary"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_variableSet(t *testing.T) {
	tests := []struct {
		name string

		activeflowID uuid.UUID
		summary      *summary.Summary

		expectedVariables map[string]string
	}{
		{
			name: "normal",

			activeflowID: uuid.FromStringOrNil("2f01f4b8-0bf7-11f0-8014-7b3d54df3f9b"),
			summary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2f2f2eb0-0bf7-11f0-98db-2f607a4fecc7"),
				},

				ReferenceType: summary.ReferenceTypeTranscribe,
				ReferenceID:   uuid.FromStringOrNil("2f5cb2e0-0bf7-11f0-8e07-1b4c4dbb76eb"),

				Language: "en-US",
				Content:  "Hello world",
			},

			expectedVariables: map[string]string{
				variableSummaryID:            "2f2f2eb0-0bf7-11f0-98db-2f607a4fecc7",
				variableSummaryReferenceType: string(summary.ReferenceTypeTranscribe),
				variableSummaryReferenceID:   "2f5cb2e0-0bf7-11f0-8e07-1b4c4dbb76eb",
				variableSummaryLanguage:      "en-US",
				variableSummaryContent:       "Hello world",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := summaryHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectedVariables.Return(nil)

			if errSet := h.variableSet(ctx, tt.activeflowID, tt.summary); errSet != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", errSet)
			}

		})
	}
}
