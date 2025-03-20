package recordinghandler

import (
	"context"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_setVariables(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID
		recording    *recording.Recording

		expectVariables map[string]string
	}{
		{
			name: "normal",

			activeflowID: uuid.FromStringOrNil("ef882140-054d-11f0-b88e-bf4eafef620a"),
			recording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3246b8aa-054b-11f0-9b6d-0376f87148e7"),
				},

				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("32958e26-054b-11f0-936f-a3f6dcc0bb9c"),
				Status:        recording.StatusRecording,
				Format:        recording.FormatWAV,
				OnEndFlowID:   uuid.FromStringOrNil("32b975a2-054b-11f0-80ac-1bd5b2777b9e"),
				RecordingName: "call_32958e26-054b-11f0-936f-a3f6dcc0bb9c_2020-04-18T03:22:17.995000",
				Filenames: []string{
					"call_32958e26-054b-11f0-936f-a3f6dcc0bb9c_2020-04-18T03:22:17.995000_in.wav",
					"call_32958e26-054b-11f0-936f-a3f6dcc0bb9c_2020-04-18T03:22:17.995000_out.wav",
				},
			},

			expectVariables: map[string]string{
				variableRecordingID: "3246b8aa-054b-11f0-9b6d-0376f87148e7",

				variableRecordingReferenceType: string(recording.ReferenceTypeCall),
				variableRecordingReferenceID:   "32958e26-054b-11f0-936f-a3f6dcc0bb9c",
				variableRecordingFormat:        string(recording.FormatWAV),

				variableRecordingRecordingName: "call_32958e26-054b-11f0-936f-a3f6dcc0bb9c_2020-04-18T03:22:17.995000",
				variableRecordingFilenames:     "call_32958e26-054b-11f0-936f-a3f6dcc0bb9c_2020-04-18T03:22:17.995000_in.wav,call_32958e26-054b-11f0-936f-a3f6dcc0bb9c_2020-04-18T03:22:17.995000_out.wav",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &recordingHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectVariables).Return(nil)

			if err := h.setVariablesCall(ctx, tt.activeflowID, tt.recording); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
