package transcribehandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_variableSet(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID
		transcribe   *transcribe.Transcribe

		expectedVariables map[string]string
	}{
		{
			name: "normal",

			activeflowID: uuid.FromStringOrNil("67fc7f4a-0936-11f0-afd4-eb9900f06e41"),
			transcribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6853c98a-0936-11f0-b357-c3182f3cb158"),
				},
				Language:  "en-US",
				Direction: transcribe.DirectionBoth,
			},

			expectedVariables: map[string]string{
				variableTranscribeID:        "6853c98a-0936-11f0-b357-c3182f3cb158",
				variableTranscribeLanguage:  "en-US",
				variableTranscribeDirection: "both",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockGoogle := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockGoogle,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectedVariables).Return(nil)

			if errSet := h.variableSet(ctx, tt.activeflowID, tt.transcribe); errSet != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", errSet)
			}
		})
	}
}
