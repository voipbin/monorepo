package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/cachehandler"
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
			},

			"2021-01-01 00:00:00.000",
			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("63b17070-0edb-11ec-8563-33766d40e3fa"),
				CustomerID:    uuid.FromStringOrNil("e3c0d790-7ffd-11ec-9bb3-6bd5fb4a12e4"),
				ReferenceType: transcribe.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c2220b88-0edb-11ec-8cf6-1fcc5a2e6786"),
				HostID:        uuid.FromStringOrNil("cd612952-0edb-11ec-a725-cf67d5b3d232"),
				Language:      "en-US",
				TMCreate:      "2021-01-01 00:00:00.000",
				TMUpdate:      DefaultTimeStamp,
				TMDelete:      DefaultTimeStamp,
			},
		},
		{
			"empty",
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("81ce2448-0edd-11ec-861d-c7b56c3e942a"),
			},

			"2021-01-01 00:00:00.000",
			&transcribe.Transcribe{
				ID:       uuid.FromStringOrNil("81ce2448-0edd-11ec-861d-c7b56c3e942a"),
				TMCreate: "2021-01-01 00:00:00.000",
				TMUpdate: DefaultTimeStamp,
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

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
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

func Test_TranscribeGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name        string
		transcribes []*transcribe.Transcribe

		customerID uuid.UUID

		responseCurTime string
		expectRes       []*transcribe.Transcribe
	}{
		{
			"normal",
			[]*transcribe.Transcribe{
				{
					ID:         uuid.FromStringOrNil("5a543dc8-7e2e-11ed-98b6-1f655462baf9"),
					CustomerID: uuid.FromStringOrNil("5a89c4de-7e2e-11ed-97c8-a30faed31cf2"),
				},
			},

			uuid.FromStringOrNil("5a89c4de-7e2e-11ed-97c8-a30faed31cf2"),

			"2021-01-01 00:00:00.000",
			[]*transcribe.Transcribe{
				{
					ID:         uuid.FromStringOrNil("5a543dc8-7e2e-11ed-98b6-1f655462baf9"),
					CustomerID: uuid.FromStringOrNil("5a89c4de-7e2e-11ed-97c8-a30faed31cf2"),
					TMCreate:   "2021-01-01 00:00:00.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*transcribe.Transcribe{},

			uuid.FromStringOrNil("a8053398-a296-11ec-a7c7-33a89a071234"),

			"",
			[]*transcribe.Transcribe{},
		},
		{
			"2 items",
			[]*transcribe.Transcribe{
				{
					ID:         uuid.FromStringOrNil("89076d02-7e2e-11ed-9d33-cb5ddc245e50"),
					CustomerID: uuid.FromStringOrNil("89673048-7e2e-11ed-96ed-d70dc363ece5"),
				},
				{
					ID:         uuid.FromStringOrNil("8937c114-7e2e-11ed-8566-ffc2c99e510d"),
					CustomerID: uuid.FromStringOrNil("89673048-7e2e-11ed-96ed-d70dc363ece5"),
				},
			},

			uuid.FromStringOrNil("89673048-7e2e-11ed-96ed-d70dc363ece5"),

			"2021-01-01 00:00:00.000",
			[]*transcribe.Transcribe{
				{
					ID:         uuid.FromStringOrNil("89076d02-7e2e-11ed-9d33-cb5ddc245e50"),
					CustomerID: uuid.FromStringOrNil("89673048-7e2e-11ed-96ed-d70dc363ece5"),
					TMCreate:   "2021-01-01 00:00:00.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("8937c114-7e2e-11ed-8566-ffc2c99e510d"),
					CustomerID: uuid.FromStringOrNil("89673048-7e2e-11ed-96ed-d70dc363ece5"),
					TMCreate:   "2021-01-01 00:00:00.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
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
				mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())

				if err := h.TranscribeCreate(ctx, tt.transcribes[i]); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.TranscribeGetsByCustomerID(ctx, tt.customerID, 10, utilhandler.GetCurTime())
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
				ID:     uuid.FromStringOrNil("dc3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
				Status: transcribe.StatusProgressing,

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

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())
			if err := h.TranscribeCreate(ctx, tt.transcribe); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
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
