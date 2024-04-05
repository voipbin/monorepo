package transcribehandler

import (
	"context"
	"testing"

	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcripthandler"
)

func Test_EventCUCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer            *cucustomer.Customer
		responseTranscribes []*transcribe.Transcribe

		expectFilters           map[string]string
		expectFilterTranscripts map[string]string
	}{
		{
			name: "normal",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("cac89366-f2e4-11ee-a393-f7b712d3f1a4"),
			},
			responseTranscribes: []*transcribe.Transcribe{
				{
					ID:       uuid.FromStringOrNil("caeca0bc-f2e4-11ee-a152-e7331c726b3b"),
					Status:   transcribe.StatusDone,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("cb17285a-f2e4-11ee-aaee-c31bb727756e"),
					Status:   transcribe.StatusDone,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},

			expectFilters: map[string]string{
				"customer_id": "cac89366-f2e4-11ee-a393-f7b712d3f1a4",
				"deleted":     "false",
			},
			expectFilterTranscripts: map[string]string{
				"customer_id": "8c0daf80-f0c3-11ee-9ed5-6b65132a6fc3",
				"deleted":     "false",
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				utilHandler:       mockUtil,
				transcriptHandler: mockTranscript,
			}
			ctx := context.Background()

			mockDB.EXPECT().TranscribeGets(ctx, uint64(1000), "", tt.expectFilters).Return(tt.responseTranscribes, nil)

			// delete each
			for _, tr := range tt.responseTranscribes {
				mockDB.EXPECT().TranscribeGet(ctx, tr.ID).Return(tr, nil)

				tmpFilters := map[string]string{
					"transcribe_id": tr.ID.String(),
					"deleted":       "false",
				}
				mockTranscript.EXPECT().Gets(ctx, uint64(1000), "", tmpFilters).Return([]*transcript.Transcript{}, nil)

				mockDB.EXPECT().TranscribeDelete(ctx, tr.ID).Return(nil)
				mockDB.EXPECT().TranscribeGet(ctx, tr.ID).Return(tr, nil)
				mockNotify.EXPECT().PublishEvent(ctx, transcribe.EventTypeTranscribeDeleted, tr)
			}

			if err := h.EventCUCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
