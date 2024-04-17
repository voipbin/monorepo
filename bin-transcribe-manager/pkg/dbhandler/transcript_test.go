package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
)

func Test_TranscriptCreate(t *testing.T) {

	type test struct {
		name string

		transcript *transcript.Transcript

		responseCurTime string
		expectRes       *transcript.Transcript
	}

	tests := []test{
		{
			"all items",
			&transcript.Transcript{
				ID:           uuid.FromStringOrNil("6029f7a0-7e2f-11ed-b9d0-5f9428aa8958"),
				CustomerID:   uuid.FromStringOrNil("605f0d64-7e2f-11ed-bfc9-3b8a21f4c79c"),
				TranscribeID: uuid.FromStringOrNil("60942350-7e2f-11ed-be15-9fea382cfa70"),
				Direction:    transcript.DirectionIn,
				Message:      "Hello, this is test message",
				TMTranscript: "0000-00-00 00:00:01.00000",
			},

			"2021-01-01 00:00:00.000",
			&transcript.Transcript{
				ID:           uuid.FromStringOrNil("6029f7a0-7e2f-11ed-b9d0-5f9428aa8958"),
				CustomerID:   uuid.FromStringOrNil("605f0d64-7e2f-11ed-bfc9-3b8a21f4c79c"),
				TranscribeID: uuid.FromStringOrNil("60942350-7e2f-11ed-be15-9fea382cfa70"),
				Direction:    transcript.DirectionIn,
				Message:      "Hello, this is test message",
				TMTranscript: "0000-00-00 00:00:01.00000",
				TMCreate:     "2021-01-01 00:00:00.000",
				TMDelete:     DefaultTimeStamp,
			},
		},
		{
			"empty",
			&transcript.Transcript{
				ID: uuid.FromStringOrNil("f2757f08-7e2f-11ed-a6c5-af8cdad93769"),
			},

			"2021-01-01 00:00:00.000",
			&transcript.Transcript{
				ID:       uuid.FromStringOrNil("f2757f08-7e2f-11ed-a6c5-af8cdad93769"),
				TMCreate: "2021-01-01 00:00:00.000",
				TMDelete: DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

func Test_TranscriptGets(t *testing.T) {

	tests := []struct {
		name        string
		transcripts []*transcript.Transcript

		filters map[string]string

		responseCurTime string
		expectRes       []*transcript.Transcript
	}{
		{
			"normal",
			[]*transcript.Transcript{
				{
					ID:           uuid.FromStringOrNil("4db68b46-7e30-11ed-8c50-0fe723a2648a"),
					TranscribeID: uuid.FromStringOrNil("4dda38ac-7e30-11ed-876b-3bf1eb420c31"),
				},
			},

			map[string]string{
				"transcribe_id": "4dda38ac-7e30-11ed-876b-3bf1eb420c31",
				"deleted":       "false",
			},

			"2021-01-01 00:00:00.000",
			[]*transcript.Transcript{
				{
					ID:           uuid.FromStringOrNil("4db68b46-7e30-11ed-8c50-0fe723a2648a"),
					TranscribeID: uuid.FromStringOrNil("4dda38ac-7e30-11ed-876b-3bf1eb420c31"),
					TMCreate:     "2021-01-01 00:00:00.000",
					TMDelete:     DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*transcript.Transcript{},

			map[string]string{
				"customer_id": "8077cdec-7e30-11ed-93c6-efc2ead67105",
			},

			"",
			[]*transcript.Transcript{},
		},
		{
			"2 items",
			[]*transcript.Transcript{
				{
					ID:           uuid.FromStringOrNil("892afd74-7e30-11ed-808f-578ac2e92b40"),
					TranscribeID: uuid.FromStringOrNil("89560596-7e30-11ed-b714-e7b0632f17e9"),
				},
				{
					ID:           uuid.FromStringOrNil("8937c114-7e2e-11ed-8566-ffc2c99e510d"),
					TranscribeID: uuid.FromStringOrNil("89560596-7e30-11ed-b714-e7b0632f17e9"),
				},
			},

			map[string]string{
				"transcribe_id": "89560596-7e30-11ed-b714-e7b0632f17e9",
				"deleted":       "false",
			},

			"2021-01-01 00:00:00.000",
			[]*transcript.Transcript{
				{
					ID:           uuid.FromStringOrNil("892afd74-7e30-11ed-808f-578ac2e92b40"),
					TranscribeID: uuid.FromStringOrNil("89560596-7e30-11ed-b714-e7b0632f17e9"),
					TMCreate:     "2021-01-01 00:00:00.000",
					TMDelete:     DefaultTimeStamp,
				},
				{
					ID:           uuid.FromStringOrNil("8937c114-7e2e-11ed-8566-ffc2c99e510d"),
					TranscribeID: uuid.FromStringOrNil("89560596-7e30-11ed-b714-e7b0632f17e9"),
					TMCreate:     "2021-01-01 00:00:00.000",
					TMDelete:     DefaultTimeStamp,
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
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().TranscriptSet(ctx, gomock.Any())

				if err := h.TranscriptCreate(ctx, tt.transcripts[i]); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.TranscriptGets(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
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

	type test struct {
		name       string
		transcript *transcript.Transcript

		responseCurTime string
		expectRes       *transcript.Transcript
	}

	tests := []test{
		{
			"normal",
			&transcript.Transcript{
				ID: uuid.FromStringOrNil("15f2b1f4-f197-11ee-b786-7b9a797be96c"),
			},

			"2021-01-01 00:00:00.000",
			&transcript.Transcript{
				ID:       uuid.FromStringOrNil("15f2b1f4-f197-11ee-b786-7b9a797be96c"),
				TMCreate: "2021-01-01 00:00:00.000",
				TMDelete: "2021-01-01 00:00:00.000",
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()
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
