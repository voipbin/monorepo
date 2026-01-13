package transcribehandler

import (
	"context"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	commonidentity "monorepo/bin-common-handler/models/identity"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_EventCUCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer            *cucustomer.Customer
		responseTranscribes []*transcribe.Transcribe

		expectFilters map[transcribe.Field]any
	}{
		{
			name: "normal",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("cac89366-f2e4-11ee-a393-f7b712d3f1a4"),
			},
			responseTranscribes: []*transcribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("caeca0bc-f2e4-11ee-a152-e7331c726b3b"),
					},
					Status:   transcribe.StatusDone,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("cb17285a-f2e4-11ee-aaee-c31bb727756e"),
					},
					Status:   transcribe.StatusDone,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},

			expectFilters: map[transcribe.Field]any{
				transcribe.FieldCustomerID: uuid.FromStringOrNil("cac89366-f2e4-11ee-a393-f7b712d3f1a4"),
				transcribe.FieldDeleted:    false,
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

			mockDB.EXPECT().TranscribeGets(ctx, uint64(1000), "", tt.expectFilters.Return(tt.responseTranscribes, nil)

			// delete each
			for _, tr := range tt.responseTranscribes {
				mockDB.EXPECT().TranscribeGet(ctx, tr.ID.Return(tr, nil)

				tmpFilters := map[transcript.Field]any{
					transcript.FieldTranscribeID: tr.ID,
					transcript.FieldDeleted:      false,
				}
				mockTranscript.EXPECT().Gets(ctx, uint64(1000), "", gomock.Any(.Return([]*transcript.Transcript{}, nil)

				mockDB.EXPECT().TranscribeDelete(ctx, tr.ID.Return(nil)
				mockDB.EXPECT().TranscribeGet(ctx, tr.ID.Return(tr, nil)
				mockNotify.EXPECT().PublishEvent(ctx, transcribe.EventTypeTranscribeDeleted, tr)
			}

			if err := h.EventCUCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCMCallHangup(t *testing.T) {

	tests := []struct {
		name string

		call                *cmcall.Call
		responseTranscribes []*transcribe.Transcribe

		expectFilters map[transcribe.Field]any
	}{
		{
			name: "normal",

			call: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("60865546-f2ef-11ee-93b7-13e9bdae3b1c"),
				},
			},
			responseTranscribes: []*transcribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("60c49f2c-f2ef-11ee-b34e-0ff8d9efb06c"),
					},
					Status:   transcribe.StatusDone,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("61321d36-f2ef-11ee-91a2-1b4be4f8b801"),
					},
					Status:   transcribe.StatusDone,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},

			expectFilters: map[transcribe.Field]any{
				transcribe.FieldReferenceID: uuid.FromStringOrNil("60865546-f2ef-11ee-93b7-13e9bdae3b1c"),
				transcribe.FieldDeleted:     false,
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

			mockDB.EXPECT().TranscribeGets(ctx, uint64(1000), "", tt.expectFilters.Return(tt.responseTranscribes, nil)

			// delete each
			for _, tr := range tt.responseTranscribes {
				mockDB.EXPECT().TranscribeGet(ctx, tr.ID.Return(tr, nil)
			}

			if err := h.EventCMCallHangup(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCMConfbridgeTerminated(t *testing.T) {

	tests := []struct {
		name string

		confbridge          *cmconfbridge.Confbridge
		responseTranscribes []*transcribe.Transcribe

		expectFilters map[transcribe.Field]any
	}{
		{
			name: "normal",

			confbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a722a706-f2f0-11ee-9467-abd69c44e65f"),
				},
			},
			responseTranscribes: []*transcribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a75970f6-f2f0-11ee-8f5f-c3086817f421"),
					},
					Status:   transcribe.StatusDone,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a789fbcc-f2f0-11ee-9902-4b978847b531"),
					},
					Status:   transcribe.StatusDone,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},

			expectFilters: map[transcribe.Field]any{
				transcribe.FieldReferenceID: uuid.FromStringOrNil("a722a706-f2f0-11ee-9467-abd69c44e65f"),
				transcribe.FieldDeleted:     false,
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

			mockDB.EXPECT().TranscribeGets(ctx, uint64(1000), "", tt.expectFilters.Return(tt.responseTranscribes, nil)

			// delete each
			for _, tr := range tt.responseTranscribes {
				mockDB.EXPECT().TranscribeGet(ctx, tr.ID.Return(tr, nil)
			}

			if err := h.EventCMConfbridgeTerminated(ctx, tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
