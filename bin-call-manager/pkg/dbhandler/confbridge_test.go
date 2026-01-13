package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/cachehandler"
)

func Test_ConfbridgeCreateAndGet(t *testing.T) {

	tests := []struct {
		name string

		confbridge *confbridge.Confbridge

		responseCurTime string
		expectRes       *confbridge.Confbridge
	}{
		{
			name: "empty",
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("32318203-58bf-4105-adf4-e3b9866ee9a9"),
				},
			},

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("32318203-58bf-4105-adf4-e3b9866ee9a9"),
				},
				Flags:          []confbridge.Flag{},
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				TMCreate:       "2023-01-18 03:22:18.995000",
				TMUpdate:       DefaultTimeStamp,
				TMDelete:       DefaultTimeStamp,
			},
		},
		{
			name: "have all",
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fc07eed6-3301-11ec-8218-f37dfb357914"),
				},
				ActiveflowID:  uuid.FromStringOrNil("e45c06be-06ac-11f0-824f-f7ccba9579aa"),
				ReferenceType: confbridge.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("f3f3b7f4-972c-11ed-8b3b-0b7b0c5441ac"),
				Type:          confbridge.TypeConnect,
				Status:        confbridge.StatusProgressing,
				BridgeID:      "f4959208-972c-11ed-be90-6b3eb4bef16d",
				Flags: []confbridge.Flag{
					confbridge.FlagNoAutoLeave,
				},
				ChannelCallIDs: map[string]uuid.UUID{
					"e655ea4e-972c-11ed-9de8-bbd3892344ca": uuid.FromStringOrNil("f4173688-972c-11ed-88f7-7b8c8c882ba4"),
				},
				RecordingID: uuid.FromStringOrNil("f4bb44a8-972c-11ed-b242-d37f337f0809"),
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("f469a080-972c-11ed-823d-5777dc5a0e95"),
					uuid.FromStringOrNil("f4bb44a8-972c-11ed-b242-d37f337f0809"),
				},
				ExternalMediaID: uuid.FromStringOrNil("f4deecf0-972c-11ed-8ad1-1b7b0c5441ac"),
			},

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fc07eed6-3301-11ec-8218-f37dfb357914"),
				},
				ActiveflowID:  uuid.FromStringOrNil("e45c06be-06ac-11f0-824f-f7ccba9579aa"),
				ReferenceType: confbridge.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("f3f3b7f4-972c-11ed-8b3b-0b7b0c5441ac"),
				Type:          confbridge.TypeConnect,
				Status:        confbridge.StatusProgressing,
				BridgeID:      "f4959208-972c-11ed-be90-6b3eb4bef16d",
				Flags: []confbridge.Flag{
					confbridge.FlagNoAutoLeave,
				},
				ChannelCallIDs: map[string]uuid.UUID{
					"e655ea4e-972c-11ed-9de8-bbd3892344ca": uuid.FromStringOrNil("f4173688-972c-11ed-88f7-7b8c8c882ba4"),
				},
				RecordingID: uuid.FromStringOrNil("f4bb44a8-972c-11ed-b242-d37f337f0809"),
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("f469a080-972c-11ed-823d-5777dc5a0e95"),
					uuid.FromStringOrNil("f4bb44a8-972c-11ed-b242-d37f337f0809"),
				},
				ExternalMediaID: uuid.FromStringOrNil("f4deecf0-972c-11ed-8ad1-1b7b0c5441ac"),
				TMCreate:        "2023-01-18 03:22:18.995000",
				TMUpdate:        DefaultTimeStamp,
				TMDelete:        DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any().Return(nil)
			if err := h.ConfbridgeCreate(ctx, tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			res, err := h.ConfbridgeGet(ctx, tt.confbridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConfbridgeGetByBridgeID(t *testing.T) {

	type test struct {
		name string

		confbridge *confbridge.Confbridge

		responseCurTime string
		expectRes       *confbridge.Confbridge
	}

	tests := []test{
		{
			name: "type conference",
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bf738558-34ef-11ec-a927-6ba7cd3ff490"),
				},
				BridgeID:       "bfc5a1e4-34ef-11ec-ad12-870a5704955c",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bf738558-34ef-11ec-a927-6ba7cd3ff490"),
				},
				BridgeID:       "bfc5a1e4-34ef-11ec-ad12-870a5704955c",
				Flags:          []confbridge.Flag{},
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				TMCreate:       "2023-01-18 03:22:18.995000",
				TMUpdate:       DefaultTimeStamp,
				TMDelete:       DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any().Return(nil)
			if err := h.ConfbridgeCreate(context.Background(), tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ConfbridgeGetByBridgeID(ctx, tt.confbridge.BridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConfbridgeGets(t *testing.T) {

	type test struct {
		name        string
		confbridges []*confbridge.Confbridge

		filters map[confbridge.Field]any

		responseCurTime string

		expectRes []*confbridge.Confbridge
	}

	tests := []test{
		{
			name: "normal",
			confbridges: []*confbridge.Confbridge{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e0b62376-f0ca-11ee-b6e5-ab3d7196a40e"),
						CustomerID: uuid.FromStringOrNil("e14965b4-f0ca-11ee-9715-3bb35c382030"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e11f5a26-f0ca-11ee-a8ac-eba0c97a0aa7"),
						CustomerID: uuid.FromStringOrNil("e14965b4-f0ca-11ee-9715-3bb35c382030"),
					},
				},
			},

			filters: map[confbridge.Field]any{
				confbridge.FieldCustomerID: uuid.FromStringOrNil("e14965b4-f0ca-11ee-9715-3bb35c382030"),
				confbridge.FieldDeleted:    false,
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectRes: []*confbridge.Confbridge{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e0b62376-f0ca-11ee-b6e5-ab3d7196a40e"),
						CustomerID: uuid.FromStringOrNil("e14965b4-f0ca-11ee-9715-3bb35c382030"),
					},

					Flags:          []confbridge.Flag{},
					ChannelCallIDs: map[string]uuid.UUID{},
					RecordingIDs:   []uuid.UUID{},

					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e11f5a26-f0ca-11ee-a8ac-eba0c97a0aa7"),
						CustomerID: uuid.FromStringOrNil("e14965b4-f0ca-11ee-9715-3bb35c382030"),
					},

					Flags:          []confbridge.Flag{},
					ChannelCallIDs: map[string]uuid.UUID{},
					RecordingIDs:   []uuid.UUID{},

					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*confbridge.Confbridge{},

			map[confbridge.Field]any{
				confbridge.FieldCustomerID: uuid.FromStringOrNil("e173bd1e-f0ca-11ee-8744-976dc414fbc2"),
				confbridge.FieldDeleted:    false,
			},

			"2020-04-18 03:22:17.995000",
			[]*confbridge.Confbridge{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates calls for test
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, c := range tt.confbridges {
				mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
				mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
				_ = h.ConfbridgeCreate(ctx, c)
			}

			res, err := h.ConfbridgeGets(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConfbridgeSetRecordingID(t *testing.T) {

	type test struct {
		name       string
		confbridge *confbridge.Confbridge
		recordID   uuid.UUID

		responseCurTime string
		expectRes       *confbridge.Confbridge
	}

	tests := []test{
		{
			name: "normal",
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("75b1275e-3305-11ec-8dba-8bf525336b2b"),
				},
			},
			recordID: uuid.FromStringOrNil("760b193a-3305-11ec-a9af-0fbbe717a04f"),

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("75b1275e-3305-11ec-8dba-8bf525336b2b"),
				},
				Flags:          []confbridge.Flag{},
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingID:    uuid.FromStringOrNil("760b193a-3305-11ec-a9af-0fbbe717a04f"),
				RecordingIDs:   []uuid.UUID{},
				TMCreate:       "2023-01-18 03:22:18.995000",
				TMUpdate:       "2023-01-18 03:22:18.995000",
				TMDelete:       DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			if err := h.ConfbridgeCreate(ctx, tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			if err := h.ConfbridgeSetRecordingID(ctx, tt.confbridge.ID, tt.recordID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			res, err := h.ConfbridgeGet(ctx, tt.confbridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConfbridgeSetExternalMediaID(t *testing.T) {

	type test struct {
		name            string
		confbridge      *confbridge.Confbridge
		externalMediaID uuid.UUID

		responseCurTime string
		expectRes       *confbridge.Confbridge
	}

	tests := []test{
		{
			name: "normal",
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a587afdc-972e-11ed-9c8a-c71ab8ef38bd"),
				},
			},
			externalMediaID: uuid.FromStringOrNil("a5b2cc80-972e-11ed-86cc-a31ac34ae6bc"),

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a587afdc-972e-11ed-9c8a-c71ab8ef38bd"),
				},
				Flags:           []confbridge.Flag{},
				ChannelCallIDs:  map[string]uuid.UUID{},
				RecordingIDs:    []uuid.UUID{},
				ExternalMediaID: uuid.FromStringOrNil("a5b2cc80-972e-11ed-86cc-a31ac34ae6bc"),
				TMCreate:        "2023-01-18 03:22:18.995000",
				TMUpdate:        "2023-01-18 03:22:18.995000",
				TMDelete:        DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			if err := h.ConfbridgeCreate(ctx, tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			if err := h.ConfbridgeSetExternalMediaID(ctx, tt.confbridge.ID, tt.externalMediaID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			res, err := h.ConfbridgeGet(ctx, tt.confbridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConfbridgeSetFlags(t *testing.T) {

	type test struct {
		name       string
		confbridge *confbridge.Confbridge
		flags      []confbridge.Flag

		responseCurTime string
		expectRes       *confbridge.Confbridge
	}

	tests := []test{
		{
			name: "normal",
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("64a9668e-d709-11ed-a3f4-d75381c89660"),
				},
			},
			flags: []confbridge.Flag{
				confbridge.FlagNoAutoLeave,
			},

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("64a9668e-d709-11ed-a3f4-d75381c89660"),
				},
				Flags: []confbridge.Flag{
					confbridge.FlagNoAutoLeave,
				},
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				TMCreate:       "2023-01-18 03:22:18.995000",
				TMUpdate:       "2023-01-18 03:22:18.995000",
				TMDelete:       DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			if err := h.ConfbridgeCreate(ctx, tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			if err := h.ConfbridgeSetFlags(ctx, tt.confbridge.ID, tt.flags); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			res, err := h.ConfbridgeGet(ctx, tt.confbridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConfbridgeSetStatus(t *testing.T) {

	type test struct {
		name       string
		confbridge *confbridge.Confbridge
		status     confbridge.Status

		responseCurTime string
		expectRes       *confbridge.Confbridge
	}

	tests := []test{
		{
			name: "normal",
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("623042b0-6193-42dc-9b80-299f12b3df24"),
				},
				Status: confbridge.StatusProgressing,
			},
			status: confbridge.StatusTerminating,

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("623042b0-6193-42dc-9b80-299f12b3df24"),
				},
				Status:         confbridge.StatusTerminating,
				Flags:          []confbridge.Flag{},
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				TMCreate:       "2023-01-18 03:22:18.995000",
				TMUpdate:       "2023-01-18 03:22:18.995000",
				TMDelete:       DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			if err := h.ConfbridgeCreate(ctx, tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			if err := h.ConfbridgeSetStatus(ctx, tt.confbridge.ID, tt.status); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
			res, err := h.ConfbridgeGet(ctx, tt.confbridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
