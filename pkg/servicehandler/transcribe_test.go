package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

func TestTranscribeCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string
		user *user.User

		referenceID uuid.UUID
		language    string

		responseRecording *cmrecording.Recording
		response          *tmtranscribe.Transcribe
		expectRes         *transcribe.Transcribe
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},

			uuid.FromStringOrNil("4a1b66dc-a3f3-11eb-bdef-bb62ebd98cdd"),
			"en-US",

			&cmrecording.Recording{
				ID:     uuid.FromStringOrNil("4a1b66dc-a3f3-11eb-bdef-bb62ebd98cdd"),
				UserID: 1,
			},
			&tmtranscribe.Transcribe{
				ID:            uuid.FromStringOrNil("77bd4574-a3f3-11eb-a790-c7e151065eb9"),
				Type:          tmtranscribe.TypeRecording,
				ReferenceID:   uuid.FromStringOrNil("4a1b66dc-a3f3-11eb-bdef-bb62ebd98cdd"),
				Language:      "en-US",
				WebhookURI:    "",
				WebhookMethod: "",
				Transcripts: []tmtranscribe.Transcript{
					{
						Direction: tmtranscribe.TranscriptDirectionIn,
						Message:   "Hello, this is voipbin.",
					},
				},
			},

			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("77bd4574-a3f3-11eb-a790-c7e151065eb9"),
				Type:          transcribe.TypeRecording,
				ReferenceID:   uuid.FromStringOrNil("4a1b66dc-a3f3-11eb-bdef-bb62ebd98cdd"),
				Language:      "en-US",
				WebhookURI:    "",
				WebhookMethod: "",
				Transcripts: []transcribe.Transcript{
					{
						Direction: transcribe.TranscriptDirectionIn,
						Message:   "Hello, this is voipbin.",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().CMV1RecordingGet(gomock.Any(), tt.referenceID).Return(tt.responseRecording, nil)
			mockReq.EXPECT().TSV1RecordingCreate(gomock.Any(), tt.referenceID, tt.language).Return(tt.response, nil)

			res, err := h.TranscribeCreate(tt.user, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
