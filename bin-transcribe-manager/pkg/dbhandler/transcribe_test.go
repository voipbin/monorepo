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

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
)

func Test_TranscribeCreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name string

		transcribe *transcribe.Transcribe

		responseCurTime *time.Time
		expectRes       *transcribe.Transcribe
	}

	tests := []test{
		{
			name: "all items",
			transcribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("63b17070-0edb-11ec-8563-33766d40e3fa"),
					CustomerID: uuid.FromStringOrNil("e3c0d790-7ffd-11ec-9bb3-6bd5fb4a12e4"),
				},
				ActiveflowID: uuid.FromStringOrNil("9eecf228-0922-11f0-b03b-af4f7da49ec7"),
				OnEndFlowID:  uuid.FromStringOrNil("9f3955fa-0922-11f0-b449-1bd4aaff1d50"),

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

			responseCurTime: curTime,
			expectRes: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("63b17070-0edb-11ec-8563-33766d40e3fa"),
					CustomerID: uuid.FromStringOrNil("e3c0d790-7ffd-11ec-9bb3-6bd5fb4a12e4"),
				},
				ActiveflowID: uuid.FromStringOrNil("9eecf228-0922-11f0-b03b-af4f7da49ec7"),
				OnEndFlowID:  uuid.FromStringOrNil("9f3955fa-0922-11f0-b449-1bd4aaff1d50"),

				ReferenceType: transcribe.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c2220b88-0edb-11ec-8cf6-1fcc5a2e6786"),
				HostID:        uuid.FromStringOrNil("cd612952-0edb-11ec-a725-cf67d5b3d232"),
				Language:      "en-US",
				Direction:     transcribe.DirectionBoth,
				StreamingIDs: []uuid.UUID{
					uuid.FromStringOrNil("41eb75da-9878-11ed-9f62-97d0d501930c"),
					uuid.FromStringOrNil("421e8bc8-9878-11ed-84be-1f7747406f78"),
				},
				TMCreate: curTime,
				TMUpdate: nil,
				TMDelete: nil,
			},
		},
		{
			name: "empty",
			transcribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("81ce2448-0edd-11ec-861d-c7b56c3e942a"),
				},
			},

			responseCurTime: curTime,
			expectRes: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("81ce2448-0edd-11ec-861d-c7b56c3e942a"),
				},
				StreamingIDs: []uuid.UUID{},
				TMCreate:     curTime,
				TMUpdate:     nil,
				TMDelete:     nil,
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

func Test_TranscribeList(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name        string
		transcribes []*transcribe.Transcribe

		filters map[transcribe.Field]any

		responseCurTime *time.Time
		expectRes       []*transcribe.Transcribe
	}{
		{
			name: "normal",
			transcribes: []*transcribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("68ad867c-ed95-11ee-b44e-a707b64e6732"),
						CustomerID: uuid.FromStringOrNil("68fdd924-ed95-11ee-a7ea-57b90b872fde"),
					},
				},
			},

			filters: map[transcribe.Field]any{
				transcribe.FieldCustomerID: uuid.FromStringOrNil("68fdd924-ed95-11ee-a7ea-57b90b872fde"),
				transcribe.FieldDeleted:    false,
			},

			responseCurTime: curTime,
			expectRes: []*transcribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("68ad867c-ed95-11ee-b44e-a707b64e6732"),
						CustomerID: uuid.FromStringOrNil("68fdd924-ed95-11ee-a7ea-57b90b872fde"),
					},
					StreamingIDs: []uuid.UUID{},
					TMCreate:     curTime,
					TMUpdate:     nil,
					TMDelete:     nil,
				},
			},
		},
		{
			name:        "empty",
			transcribes: []*transcribe.Transcribe{},

			filters: map[transcribe.Field]any{
				transcribe.FieldCustomerID: uuid.FromStringOrNil("b231e14e-ed95-11ee-a29f-7be740276529"),
			},

			responseCurTime: nil,
			expectRes:       []*transcribe.Transcribe{},
		},
		{
			name: "2 items",
			transcribes: []*transcribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e710ddd4-ed95-11ee-8c7b-6327cccb7082"),
						CustomerID: uuid.FromStringOrNil("c1644a94-ed95-11ee-b2c8-8bf8e129a2f7"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e73f074a-ed95-11ee-9634-176a962a17b8"),
						CustomerID: uuid.FromStringOrNil("c1644a94-ed95-11ee-b2c8-8bf8e129a2f7"),
					},
				},
			},

			filters: map[transcribe.Field]any{
				transcribe.FieldCustomerID: uuid.FromStringOrNil("c1644a94-ed95-11ee-b2c8-8bf8e129a2f7"),
				transcribe.FieldDeleted:    false,
			},

			responseCurTime: curTime,
			expectRes: []*transcribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e710ddd4-ed95-11ee-8c7b-6327cccb7082"),
						CustomerID: uuid.FromStringOrNil("c1644a94-ed95-11ee-b2c8-8bf8e129a2f7"),
					},
					StreamingIDs: []uuid.UUID{},
					TMCreate:     curTime,
					TMUpdate:     nil,
					TMDelete:     nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e73f074a-ed95-11ee-9634-176a962a17b8"),
						CustomerID: uuid.FromStringOrNil("c1644a94-ed95-11ee-b2c8-8bf8e129a2f7"),
					},
					StreamingIDs: []uuid.UUID{},
					TMCreate:     curTime,
					TMUpdate:     nil,
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
			for i := 0; i < len(tt.transcribes); i++ {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())

				if err := h.TranscribeCreate(ctx, tt.transcribes[i]); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.TranscribeList(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_TranscribeUpdate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name string

		transcribe *transcribe.Transcribe

		id              uuid.UUID
		fields          map[transcribe.Field]any
		responseCurTime *time.Time

		expectRes *transcribe.Transcribe
	}

	tests := []test{
		{
			name: "normal",

			transcribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dc3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
				},
			},

			id: uuid.FromStringOrNil("dc3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
			fields: map[transcribe.Field]any{
				transcribe.FieldStatus: transcribe.StatusProgressing,
			},
			responseCurTime: curTime,

			expectRes: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dc3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
				},
				Status:       transcribe.StatusProgressing,
				StreamingIDs: []uuid.UUID{},

				TMCreate: curTime,
				TMUpdate: curTime,
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
			mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())
			if err := h.TranscribeCreate(ctx, tt.transcribe); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())
			if err := h.TranscribeUpdate(ctx, tt.id, tt.fields); err != nil {
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

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name       string
		transcribe *transcribe.Transcribe

		responseCurTime *time.Time
		expectRes       *transcribe.Transcribe
	}

	tests := []test{
		{
			name: "normal",
			transcribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6a51f5b2-f196-11ee-b98b-cfc1a7583b20"),
				},
			},

			responseCurTime: curTime,
			expectRes: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6a51f5b2-f196-11ee-b98b-cfc1a7583b20"),
				},
				StreamingIDs: []uuid.UUID{},
				TMCreate:     curTime,
				TMUpdate:     curTime,
				TMDelete:     curTime,
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

func Test_TranscribeGetByReferenceIDAndLanguage(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name       string
		transcribe *transcribe.Transcribe

		referenceID uuid.UUID
		language    string

		responseCurTime *time.Time
		expectRes       *transcribe.Transcribe
	}

	tests := []test{
		{
			name: "normal",
			transcribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aa3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
					CustomerID: uuid.FromStringOrNil("bb3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
				},
				ReferenceID: uuid.FromStringOrNil("cc3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
				Language:    "en-US",
			},

			referenceID: uuid.FromStringOrNil("cc3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
			language:    "en-US",

			responseCurTime: curTime,

			expectRes: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aa3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
					CustomerID: uuid.FromStringOrNil("bb3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
				},
				ReferenceID:  uuid.FromStringOrNil("cc3b0b60-7f54-11ed-aed1-8363cc29dfe3"),
				Language:     "en-US",
				StreamingIDs: []uuid.UUID{},

				TMCreate: curTime,
				TMUpdate: nil,
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
			mockCache.EXPECT().TranscribeSet(ctx, gomock.Any())
			if err := h.TranscribeCreate(ctx, tt.transcribe); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.TranscribeGetByReferenceIDAndLanguage(ctx, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
