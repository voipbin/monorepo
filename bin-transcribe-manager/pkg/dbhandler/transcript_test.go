package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
)

func Test_TranscriptCreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()
	tmTranscript := func() *time.Time { t := time.Date(0, 1, 1, 0, 0, 1, 0, time.UTC); return &t }()

	type test struct {
		name string

		transcript *transcript.Transcript

		responseCurTime *time.Time
		expectRes       *transcript.Transcript
	}

	tests := []test{
		{
			name: "all items",
			transcript: &transcript.Transcript{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6029f7a0-7e2f-11ed-b9d0-5f9428aa8958"),
					CustomerID: uuid.FromStringOrNil("605f0d64-7e2f-11ed-bfc9-3b8a21f4c79c"),
				},
				TranscribeID: uuid.FromStringOrNil("60942350-7e2f-11ed-be15-9fea382cfa70"),
				Direction:    transcript.DirectionIn,
				Message:      "Hello, this is test message",
				TMTranscript: tmTranscript,
			},

			responseCurTime: curTime,
			expectRes: &transcript.Transcript{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6029f7a0-7e2f-11ed-b9d0-5f9428aa8958"),
					CustomerID: uuid.FromStringOrNil("605f0d64-7e2f-11ed-bfc9-3b8a21f4c79c"),
				},
				TranscribeID: uuid.FromStringOrNil("60942350-7e2f-11ed-be15-9fea382cfa70"),
				Direction:    transcript.DirectionIn,
				Message:      "Hello, this is test message",
				TMTranscript: tmTranscript,
				TMCreate:     curTime,
				TMDelete:     nil,
			},
		},
		{
			name: "empty",
			transcript: &transcript.Transcript{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f2757f08-7e2f-11ed-a6c5-af8cdad93769"),
				},
			},

			responseCurTime: curTime,
			expectRes: &transcript.Transcript{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f2757f08-7e2f-11ed-a6c5-af8cdad93769"),
				},
				TMCreate: curTime,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().TranscriptSet(gomock.Any(), gomock.Any())
			if err := h.TranscriptCreate(context.Background(), tt.transcript); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TranscriptGet(gomock.Any(), tt.transcript.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TranscriptSet(gomock.Any(), gomock.Any())
			res, err := h.TranscriptGet(context.Background(), tt.transcript.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscriptList(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name        string
		transcripts []*transcript.Transcript

		filters map[transcript.Field]any

		responseCurTime *time.Time
		expectRes       []*transcript.Transcript
	}{
		{
			name: "normal",
			transcripts: []*transcript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4db68b46-7e30-11ed-8c50-0fe723a2648a"),
					},
					TranscribeID: uuid.FromStringOrNil("4dda38ac-7e30-11ed-876b-3bf1eb420c31"),
				},
			},

			filters: map[transcript.Field]any{
				transcript.FieldTranscribeID: uuid.FromStringOrNil("4dda38ac-7e30-11ed-876b-3bf1eb420c31"),
				transcript.FieldDeleted:      false,
			},

			responseCurTime: curTime,
			expectRes: []*transcript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4db68b46-7e30-11ed-8c50-0fe723a2648a"),
					},
					TranscribeID: uuid.FromStringOrNil("4dda38ac-7e30-11ed-876b-3bf1eb420c31"),
					TMCreate:     curTime,
					TMDelete:     nil,
				},
			},
		},
		{
			name:        "empty",
			transcripts: []*transcript.Transcript{},

			filters: map[transcript.Field]any{
				transcript.FieldCustomerID: uuid.FromStringOrNil("8077cdec-7e30-11ed-93c6-efc2ead67105"),
			},

			responseCurTime: nil,
			expectRes:       []*transcript.Transcript{},
		},
		{
			name: "2 items",
			transcripts: []*transcript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("892afd74-7e30-11ed-808f-578ac2e92b40"),
					},
					TranscribeID: uuid.FromStringOrNil("89560596-7e30-11ed-b714-e7b0632f17e9"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8937c114-7e2e-11ed-8566-ffc2c99e510d"),
					},
					TranscribeID: uuid.FromStringOrNil("89560596-7e30-11ed-b714-e7b0632f17e9"),
				},
			},

			filters: map[transcript.Field]any{
				transcript.FieldTranscribeID: uuid.FromStringOrNil("89560596-7e30-11ed-b714-e7b0632f17e9"),
				transcript.FieldDeleted:      false,
			},

			responseCurTime: curTime,
			expectRes: []*transcript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("892afd74-7e30-11ed-808f-578ac2e92b40"),
					},
					TranscribeID: uuid.FromStringOrNil("89560596-7e30-11ed-b714-e7b0632f17e9"),
					TMCreate:     curTime,
					TMDelete:     nil,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8937c114-7e2e-11ed-8566-ffc2c99e510d"),
					},
					TranscribeID: uuid.FromStringOrNil("89560596-7e30-11ed-b714-e7b0632f17e9"),
					TMCreate:     curTime,
					TMDelete:     nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			// creates messages for test
			for i := 0; i < len(tt.transcripts); i++ {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().TranscriptSet(ctx, gomock.Any())

				if err := h.TranscriptCreate(ctx, tt.transcripts[i]); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.TranscriptList(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_TranscriptDelete(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name       string
		transcript *transcript.Transcript

		responseCurTime *time.Time
		expectRes       *transcript.Transcript
	}

	tests := []test{
		{
			name: "normal",
			transcript: &transcript.Transcript{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("15f2b1f4-f197-11ee-b786-7b9a797be96c"),
				},
			},

			responseCurTime: curTime,
			expectRes: &transcript.Transcript{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("15f2b1f4-f197-11ee-b786-7b9a797be96c"),
				},
				TMCreate: curTime,
				TMDelete: curTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime).AnyTimes()
			mockCache.EXPECT().TranscriptSet(ctx, gomock.Any())
			if err := h.TranscriptCreate(ctx, tt.transcript); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TranscriptSet(ctx, gomock.Any())
			if errDelete := h.TranscriptDelete(ctx, tt.transcript.ID); errDelete != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDelete)
			}

			mockCache.EXPECT().TranscriptGet(gomock.Any(), tt.transcript.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TranscriptSet(gomock.Any(), gomock.Any())
			res, err := h.TranscriptGet(context.Background(), tt.transcript.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscriptUpdate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name string

		transcript *transcript.Transcript

		id     uuid.UUID
		fields map[transcript.Field]any

		responseCurTime *time.Time

		expectRes *transcript.Transcript
	}

	tests := []test{
		{
			name: "normal",

			transcript: &transcript.Transcript{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ee3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
				},
			},

			id: uuid.FromStringOrNil("ee3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
			fields: map[transcript.Field]any{
				transcript.FieldMessage: "updated message",
			},
			responseCurTime: curTime,

			expectRes: &transcript.Transcript{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ee3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
				},
				Message:  "updated message",
				TMCreate: curTime,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().TranscriptSet(ctx, gomock.Any())
			if err := h.TranscriptCreate(ctx, tt.transcript); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TranscriptSet(ctx, gomock.Any())
			if err := h.TranscriptUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TranscriptGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TranscriptSet(ctx, gomock.Any())
			res, err := h.TranscriptGet(ctx, tt.transcript.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
