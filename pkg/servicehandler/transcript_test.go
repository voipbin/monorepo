package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	tmtranscript "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_TranscriptGets(t *testing.T) {

	type test struct {
		name string

		agent        *amagent.Agent
		transcribeID uuid.UUID

		responseTranscribe  *tmtranscribe.Transcribe
		responseTranscripts []tmtranscript.Transcript

		expectRes []*tmtranscript.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},

			transcribeID: uuid.FromStringOrNil("9eafc870-8284-11ed-92de-d74d9e2342cb"),

			responseTranscribe: &tmtranscribe.Transcribe{
				ID:         uuid.FromStringOrNil("9eafc870-8284-11ed-92de-d74d9e2342cb"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			responseTranscripts: []tmtranscript.Transcript{
				{
					ID: uuid.FromStringOrNil("9ede9632-8284-11ed-bf13-43420adb75f6"),
				},
				{
					ID: uuid.FromStringOrNil("9f06037a-8284-11ed-8b1a-1f5800b90993"),
				},
			},

			expectRes: []*tmtranscript.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("9ede9632-8284-11ed-bf13-43420adb75f6"),
				},
				{
					ID: uuid.FromStringOrNil("9f06037a-8284-11ed-8b1a-1f5800b90993"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, tt.transcribeID).Return(tt.responseTranscribe, nil)
			mockReq.EXPECT().TranscribeV1TranscriptGets(ctx, tt.transcribeID).Return(tt.responseTranscripts, nil)

			res, err := h.TranscriptGets(ctx, tt.agent, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
