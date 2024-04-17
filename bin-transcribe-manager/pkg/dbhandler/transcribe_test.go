package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
)

func Test_TranscribeCreate(t *testing.T) {

	type test struct {
		name string

		transcribe *transcribe.Transcribe

		responseCurTime string
		expectRes       *transcribe.Transcribe
	}

	tests := []test{
		{
			"all items",
			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("63b17070-0edb-11ec-8563-33766d40e3fa"),
				CustomerID:    uuid.FromStringOrNil("e3c0d790-7ffd-11ec-9bb3-6bd5fb4a12e4"),
				ReferenceType: transcribe.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c2220b88-0edb-11ec-8cf6-1fcc5a2e6786"),
				HostID:        uuid.FromStringOrNil("cd612952-0edb-11ec-a725-cf67d5b3d232"),
				Language:      "en-US",
				Direction:     transcribe.DirectionBoth,
				StreamingIDs: []uuid.UUID{
					uuid.FromStringOrNil("41eb75da-9878-11ed-9f62-97d0d501930c"),
					uuid.FromStringOrNil("421e8bc8-9878-11ed-84be-1f7747406f78"),
				},
			},

			"2021-01-01 00:00:00.000",
			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("63b17070-0edb-11ec-8563-33766d40e3fa"),
				CustomerID:    uuid.FromStringOrNil("e3c0d790-7ffd-11ec-9bb3-6bd5fb4a12e4"),
				ReferenceType: transcribe.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c2220b88-0edb-11ec-8cf6-1fcc5a2e6786"),
				HostID:        uuid.FromStringOrNil("cd612952-0edb-11ec-a725-cf67d5b3d232"),
				Language:      "en-US",
				Direction:     transcribe.DirectionBoth,
				StreamingIDs: []uuid.UUID{
					uuid.FromStringOrNil("41eb75da-9878-11ed-9f62-97d0d501930c"),
					uuid.FromStringOrNil("421e8bc8-9878-11ed-84be-1f7747406f78"),
				},
				TMCreate: "2021-01-01 00:00:00.000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"empty",
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("81ce2448-0edd-11ec-861d-c7b56c3e942a"),
			},

			"2021-01-01 00:00:00.000",
			&transcribe.Transcribe{
				ID:           uuid.FromStringOrNil("81ce2448-0edd-11ec-861d-c7b56c3e942a"),
				StreamingIDs: []uuid.UUID{},
				TMCreate:     "2021-01-01 00:00:00.000",
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
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
			mockCache.EXPECT().TranscribeSet(gomock.Any(), gomock.Any())
			if err := h.TranscribeCreate(context.Background(), tt.transcribe); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TranscribeGet(gomock.Any(), tt.transcribe.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TranscribeSet(gomock.Any(), gomock.Any())
			res, err := h.TranscribeGet(context.Background(), tt.transcribe.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscribeGets(t *testing.T) {

	tests := []struct {
		name        string
		transcribes []*transcribe.Transcribe

		// customerID uuid.UUID
		filters map[string]string

		responseCurTime string
		expectRes       []*transcribe.Transcribe
	}{
		{
			"normal",
			[]*transcribe.Transcribe{
				{
					ID:         uuid.FromStringOrNil("68ad867c-ed95-11ee-b44e-a707b64e6732"),
					CustomerID: uuid.FromStringOrNil("68fdd924-ed95-11ee-a7ea-57b90b872fde"),
				},
			},

			// uuid.FromStringOrNil("5a89c4de-7e2e-11ed-97c8-a30faed31cf2"),
			map[string]string{
				"customer_id": "68fdd924-ed95-11ee-a7ea-57b90b872fde",
				"deleted":     "false",
			},

			"2021-01-01 00:00:00.000",
			[]*transcribe.Transcribe{
				{
					ID:           uuid.FromStringOrNil("68ad867c-ed95-11ee-b44e-a707b64e6732"),
					CustomerID:   uuid.FromStringOrNil("68fdd924-ed95-11ee-a7ea-57b90b872fde"),
					StreamingIDs: []uuid.UUID{},
					TMCreate:     "2021-01-01 00:00:00.000",
					TMUpdate:     DefaultTimeStamp,
					TMDelete:     DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*transcribe.Transcribe{},

			map[string]string{
				"customer_id": "b231e14e-ed95-11ee-a29f-7be740276529",
			},

			"",
			[]*transcribe.Transcribe{},
		},
		{
			"2 items",
			[]*transcribe.Transcribe{
				{
					ID:         uuid.FromStringOrNil("e710ddd4-ed95-11ee-8c7b-6327cccb7082"),
					CustomerID: uuid.FromStringOrNil("c1644a94-ed95-11ee-b2c8-8bf8e129a2f7"),
				},
				{
					ID:         uuid.FromStringOrNil("e73f074a-ed95-11ee-9634-176a962a17b8"),
					CustomerID: uuid.FromStringOrNil("c1644a94-ed95-11ee-b2c8-8bf8e129a2f7"),
				},
			},

			map[string]string{
				"customer_id": "c1644a94-ed95-11ee-b2c8-8bf8e129a2f7",
				"deleted":     "false",
			},

			"2021-01-01 00:00:00.000",
			[]*transcribe.Transcribe{
				{
					ID:           uuid.FromStringOrNil("e710ddd4-ed95-11ee-8c7b-6327cccb7082"),
					CustomerID:   uuid.FromStringOrNil("c1644a94-ed95-11ee-b2c8-8bf8e129a2f7"),
					StreamingIDs: []uuid.UUID{},
					TMCreate:     "2021-01-01 00:00:00.000",
					TMUpdate:     DefaultTimeStamp,
					TMDelete:     DefaultTimeStamp,
				},
				{
					ID:           uuid.FromStringOrNil("e73f074a-ed95-11ee-9634-176a962a17b8"),
					CustomerID:   uuid.FromStringOrNil("c1644a94-ed95-11ee-b2c8-8bf8e129a2f7"),
					StreamingIDs: []uuid.UUID{},
					TMCreate:     "2021-01-01 00:00:00.000",
					TMUpdate:     DefaultTimeStamp,
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
			for i := 0; i < len(tt.transcribes); i++ {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())

				if err := h.TranscribeCreate(ctx, tt.transcribes[i]); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.TranscribeGets(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_TranscribeSetStatus(t *testing.T) {
	type test struct {
		name string

		transcribe *transcribe.Transcribe

		id              uuid.UUID
		status          transcribe.Status
		responseCurTime string

		expectRes *transcribe.Transcribe
	}

	tests := []test{
		{
			"normal",

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("dc3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
			},

			uuid.FromStringOrNil("dc3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
			transcribe.StatusProgressing,
			"2020-04-18T03:22:17.995000",

			&transcribe.Transcribe{
				ID:           uuid.FromStringOrNil("dc3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
				Status:       transcribe.StatusProgressing,
				StreamingIDs: []uuid.UUID{},

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())
			if err := h.TranscribeCreate(ctx, tt.transcribe); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())
			if err := h.TranscribeSetStatus(ctx, tt.id, tt.status); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TranscribeGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())
			res, err := h.TranscribeGet(ctx, tt.transcribe.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscribeDelete(t *testing.T) {

	type test struct {
		name       string
		transcribe *transcribe.Transcribe

		responseCurTime string
		expectRes       *transcribe.Transcribe
	}

	tests := []test{
		{
			"normal",
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("6a51f5b2-f196-11ee-b98b-cfc1a7583b20"),
			},

			"2021-01-01 00:00:00.000",
			&transcribe.Transcribe{
				ID:           uuid.FromStringOrNil("6a51f5b2-f196-11ee-b98b-cfc1a7583b20"),
				StreamingIDs: []uuid.UUID{},
				TMCreate:     "2021-01-01 00:00:00.000",
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     "2021-01-01 00:00:00.000",
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
			mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())
			if err := h.TranscribeCreate(ctx, tt.transcribe); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())
			if errDelete := h.TranscribeDelete(ctx, tt.transcribe.ID); errDelete != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDelete)
			}

			mockCache.EXPECT().TranscribeGet(gomock.Any(), tt.transcribe.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().TranscribeSet(gomock.Any(), gomock.Any())
			res, err := h.TranscribeGet(context.Background(), tt.transcribe.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
