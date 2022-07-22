package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	tmcommon "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	tmtranscript "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestTranscribeCreate(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer

		referenceID uuid.UUID
		language    string

		responseRecording *cmrecording.Recording
		response          *tmtranscribe.Transcribe
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("4a1b66dc-a3f3-11eb-bdef-bb62ebd98cdd"),
			"en-US",

			&cmrecording.Recording{
				ID:         uuid.FromStringOrNil("4a1b66dc-a3f3-11eb-bdef-bb62ebd98cdd"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&tmtranscribe.Transcribe{
				ID:          uuid.FromStringOrNil("77bd4574-a3f3-11eb-a790-c7e151065eb9"),
				Type:        tmtranscribe.TypeRecording,
				ReferenceID: uuid.FromStringOrNil("4a1b66dc-a3f3-11eb-bdef-bb62ebd98cdd"),
				Language:    "en-US",
				Transcripts: []tmtranscript.Transcript{
					{
						Direction: tmcommon.DirectionIn,
						Message:   "Hello, this is voipbin.",
					},
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

			mockReq.EXPECT().CMV1RecordingGet(gomock.Any(), tt.referenceID).Return(tt.responseRecording, nil)
			mockReq.EXPECT().TSV1RecordingCreate(gomock.Any(), tt.customer.ID, tt.referenceID, tt.language).Return(tt.response, nil)

			res, err := h.TranscribeCreate(tt.customer, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.response.ConvertWebhookMessage()) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response.ConvertWebhookMessage(), res)
			}
		})
	}
}
