package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdialtargetcall"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
)

func Test_OutdialTargetCallCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:          dbTest,
		cache:       mockCache,
		utilHandler: utilhandler.NewUtilHandler(),
	}

	curTime := func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC); return &t }()

	tests := []struct {
		name              string
		outdialTargetCall *outdialtargetcall.OutdialTargetCall
		expectRes         *outdialtargetcall.OutdialTargetCall
	}{
		{
			"normal",
			&outdialtargetcall.OutdialTargetCall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("26f03072-b01b-11ec-8b95-b7c2633990d7"),
					CustomerID: uuid.FromStringOrNil("27219f68-b01b-11ec-8ee4-fb5ebac774e1"),
				},
				CampaignID: uuid.FromStringOrNil("c09ed544-b1d7-11ec-8f0c-f7faf20556e5"),

				ActiveflowID:  uuid.FromStringOrNil("c0d3ec02-b1d7-11ec-9929-d38b66ac57aa"),
				ReferenceType: outdialtargetcall.ReferenceTypeNone,

				Status: outdialtargetcall.StatusProgressing,

				DestinationIndex: 0,
				TryCount:         1,

				TMCreate: curTime,
			},
			&outdialtargetcall.OutdialTargetCall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("26f03072-b01b-11ec-8b95-b7c2633990d7"),
					CustomerID: uuid.FromStringOrNil("27219f68-b01b-11ec-8ee4-fb5ebac774e1"),
				},
				CampaignID: uuid.FromStringOrNil("c09ed544-b1d7-11ec-8f0c-f7faf20556e5"),

				ActiveflowID:  uuid.FromStringOrNil("c0d3ec02-b1d7-11ec-9929-d38b66ac57aa"),
				ReferenceType: outdialtargetcall.ReferenceTypeNone,

				Status: outdialtargetcall.StatusProgressing,

				DestinationIndex: 0,
				TryCount:         1,

				TMCreate: curTime,
			},
		},
		{
			"reference type call",
			&outdialtargetcall.OutdialTargetCall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2ee34b0-b1bb-11ec-91c0-1b8448615480"),
					CustomerID: uuid.FromStringOrNil("b545c91c-b1bb-11ec-9c75-fbb8e010e73f"),
				},
				CampaignID: uuid.FromStringOrNil("b5713cfa-b1bb-11ec-9c93-470838a6ca99"),

				ActiveflowID:  uuid.FromStringOrNil("b599b0c2-b1bb-11ec-9f9f-3b7044794773"),
				ReferenceType: outdialtargetcall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("fc9051c0-b1bb-11ec-9168-b743d28a2dc9"),

				Status: outdialtargetcall.StatusProgressing,

				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DestinationIndex: 0,
				TryCount:         1,

				TMCreate: curTime,
			},
			&outdialtargetcall.OutdialTargetCall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2ee34b0-b1bb-11ec-91c0-1b8448615480"),
					CustomerID: uuid.FromStringOrNil("b545c91c-b1bb-11ec-9c75-fbb8e010e73f"),
				},
				CampaignID: uuid.FromStringOrNil("b5713cfa-b1bb-11ec-9c93-470838a6ca99"),

				ActiveflowID:  uuid.FromStringOrNil("b599b0c2-b1bb-11ec-9f9f-3b7044794773"),
				ReferenceType: outdialtargetcall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("fc9051c0-b1bb-11ec-9168-b743d28a2dc9"),

				Status: outdialtargetcall.StatusProgressing,

				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DestinationIndex: 0,
				TryCount:         1,

				TMCreate: curTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().OutdialTargetCallSet(ctx, tt.outdialTargetCall).Return(nil)
			if tt.outdialTargetCall.ActiveflowID != uuid.Nil {
				mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, tt.outdialTargetCall).Return(nil)
			}
			if tt.outdialTargetCall.ReferenceID != uuid.Nil {
				mockCache.EXPECT().OutdialTargetCallSetByReferenceID(ctx, tt.outdialTargetCall).Return(nil)
			}
			if err := h.OutdialTargetCallCreate(context.Background(), tt.outdialTargetCall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetCallGet(gomock.Any(), tt.outdialTargetCall.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialTargetCallSet(gomock.Any(), gomock.Any())
			if tt.outdialTargetCall.ActiveflowID != uuid.Nil {
				mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, tt.outdialTargetCall).Return(nil)
			}
			if tt.outdialTargetCall.ReferenceID != uuid.Nil {
				mockCache.EXPECT().OutdialTargetCallSetByReferenceID(ctx, tt.outdialTargetCall).Return(nil)
			}

			res, err := h.OutdialTargetCallGet(ctx, tt.outdialTargetCall.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created outdial. outdial: %v", res)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

			// get by ActiveflowID
			if tt.outdialTargetCall.ActiveflowID != uuid.Nil {

				mockCache.EXPECT().OutdialTargetCallGetByActiveflowID(ctx, tt.outdialTargetCall.ActiveflowID).Return(nil, fmt.Errorf(""))
				mockCache.EXPECT().OutdialTargetCallSet(ctx, tt.outdialTargetCall).Return(nil)
				mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, tt.outdialTargetCall).Return(nil)
				if tt.outdialTargetCall.ReferenceID != uuid.Nil {
					mockCache.EXPECT().OutdialTargetCallSetByReferenceID(ctx, tt.outdialTargetCall).Return(nil)
				}
				tmp, err := h.OutdialTargetCallGetByActiveflowID(ctx, tt.outdialTargetCall.ActiveflowID)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				if reflect.DeepEqual(tt.expectRes, tmp) == false {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
				}

			}

			// get by ReferenceID
			if tt.outdialTargetCall.ReferenceID != uuid.Nil {

				mockCache.EXPECT().OutdialTargetCallGetByReferenceID(ctx, tt.outdialTargetCall.ReferenceID).Return(nil, fmt.Errorf(""))
				mockCache.EXPECT().OutdialTargetCallSet(ctx, tt.outdialTargetCall).Return(nil)
				mockCache.EXPECT().OutdialTargetCallSetByReferenceID(ctx, tt.outdialTargetCall).Return(nil)
				if tt.outdialTargetCall.ActiveflowID != uuid.Nil {
					mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, tt.outdialTargetCall).Return(nil)
				}
				tmp, err := h.OutdialTargetCallGetByReferenceID(ctx, tt.outdialTargetCall.ReferenceID)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				if reflect.DeepEqual(tt.expectRes, tmp) == false {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
				}

			}

		})
	}
}

func Test_OutdialTargetCallList(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC); return &t }()

	tests := []struct {
		name               string
		outdialTargetCalls []*outdialtargetcall.OutdialTargetCall

		filters map[outdialtargetcall.Field]any

		expectRes []*outdialtargetcall.OutdialTargetCall
	}{
		{
			"filter by outdial_id and status - 1 item",
			[]*outdialtargetcall.OutdialTargetCall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a3713122-b1de-11ec-9466-7fd2fa979b69"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("214a0e6e-b1dc-11ec-906f-f3e4c2dc53bc"),
					OutdialID:  uuid.FromStringOrNil("a39ef300-b1de-11ec-be84-23ef6b7b9999"),

					ActiveflowID:  uuid.FromStringOrNil("2175bea6-b1dc-11ec-91d7-1793f6803550"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
			},

			map[outdialtargetcall.Field]any{
				outdialtargetcall.FieldOutdialID: uuid.FromStringOrNil("a39ef300-b1de-11ec-be84-23ef6b7b9999"),
				outdialtargetcall.FieldStatus:    outdialtargetcall.StatusProgressing,
			},

			[]*outdialtargetcall.OutdialTargetCall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a3713122-b1de-11ec-9466-7fd2fa979b69"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("214a0e6e-b1dc-11ec-906f-f3e4c2dc53bc"),
					OutdialID:  uuid.FromStringOrNil("a39ef300-b1de-11ec-be84-23ef6b7b9999"),

					ActiveflowID:  uuid.FromStringOrNil("2175bea6-b1dc-11ec-91d7-1793f6803550"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
			},
		},
		{
			"filter by outdial_id and status - 2 items",
			[]*outdialtargetcall.OutdialTargetCall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("fb1c2652-b1de-11ec-840c-8773feae8a2b"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("214a0e6e-b1dc-11ec-906f-f3e4c2dc53bc"),
					OutdialID:  uuid.FromStringOrNil("fb752900-b1de-11ec-a188-632f7f0a767c"),

					ActiveflowID:  uuid.FromStringOrNil("2175bea6-b1dc-11ec-91d7-1793f6803550"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("fb4bba52-b1de-11ec-8a01-a37216cfc989"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("214a0e6e-b1dc-11ec-906f-f3e4c2dc53bc"),
					OutdialID:  uuid.FromStringOrNil("fb752900-b1de-11ec-a188-632f7f0a767c"),

					ActiveflowID:  uuid.FromStringOrNil("2175bea6-b1dc-11ec-91d7-1793f6803550"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
			},

			map[outdialtargetcall.Field]any{
				outdialtargetcall.FieldOutdialID: uuid.FromStringOrNil("fb752900-b1de-11ec-a188-632f7f0a767c"),
				outdialtargetcall.FieldStatus:    outdialtargetcall.StatusProgressing,
			},

			[]*outdialtargetcall.OutdialTargetCall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("fb1c2652-b1de-11ec-840c-8773feae8a2b"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("214a0e6e-b1dc-11ec-906f-f3e4c2dc53bc"),
					OutdialID:  uuid.FromStringOrNil("fb752900-b1de-11ec-a188-632f7f0a767c"),

					ActiveflowID:  uuid.FromStringOrNil("2175bea6-b1dc-11ec-91d7-1793f6803550"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("fb4bba52-b1de-11ec-8a01-a37216cfc989"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("214a0e6e-b1dc-11ec-906f-f3e4c2dc53bc"),
					OutdialID:  uuid.FromStringOrNil("fb752900-b1de-11ec-a188-632f7f0a767c"),

					ActiveflowID:  uuid.FromStringOrNil("2175bea6-b1dc-11ec-91d7-1793f6803550"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
			},
		},
		{
			"filter by campaign_id and status - 1 item",
			[]*outdialtargetcall.OutdialTargetCall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("77f03e0e-b2a0-11ec-8aa0-d36c2d6ec7cd"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("785e03f8-b2a0-11ec-8fb4-9f19d7e07207"),
					OutdialID:  uuid.FromStringOrNil("a39ef300-b1de-11ec-be84-23ef6b7b9999"),

					ActiveflowID:  uuid.FromStringOrNil("789100c8-b2a0-11ec-b6b9-578b62d31d2b"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
			},

			map[outdialtargetcall.Field]any{
				outdialtargetcall.FieldCampaignID: uuid.FromStringOrNil("785e03f8-b2a0-11ec-8fb4-9f19d7e07207"),
				outdialtargetcall.FieldStatus:     outdialtargetcall.StatusProgressing,
			},

			[]*outdialtargetcall.OutdialTargetCall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("77f03e0e-b2a0-11ec-8aa0-d36c2d6ec7cd"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("785e03f8-b2a0-11ec-8fb4-9f19d7e07207"),
					OutdialID:  uuid.FromStringOrNil("a39ef300-b1de-11ec-be84-23ef6b7b9999"),

					ActiveflowID:  uuid.FromStringOrNil("789100c8-b2a0-11ec-b6b9-578b62d31d2b"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
			},
		},
		{
			"filter by campaign_id and status - 2 items",
			[]*outdialtargetcall.OutdialTargetCall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b06d8538-b2a1-11ec-b907-7bbeee8bac96"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("b1080ce8-b2a1-11ec-962f-3322bc600589"),
					OutdialID:  uuid.FromStringOrNil("fb752900-b1de-11ec-a188-632f7f0a767c"),

					ActiveflowID:  uuid.FromStringOrNil("2175bea6-b1dc-11ec-91d7-1793f6803550"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b0a22f5e-b2a1-11ec-9304-4b68c392be3e"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("b1080ce8-b2a1-11ec-962f-3322bc600589"),
					OutdialID:  uuid.FromStringOrNil("fb752900-b1de-11ec-a188-632f7f0a767c"),

					ActiveflowID:  uuid.FromStringOrNil("2175bea6-b1dc-11ec-91d7-1793f6803550"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
			},

			map[outdialtargetcall.Field]any{
				outdialtargetcall.FieldCampaignID: uuid.FromStringOrNil("b1080ce8-b2a1-11ec-962f-3322bc600589"),
				outdialtargetcall.FieldStatus:     outdialtargetcall.StatusProgressing,
			},

			[]*outdialtargetcall.OutdialTargetCall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b06d8538-b2a1-11ec-b907-7bbeee8bac96"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("b1080ce8-b2a1-11ec-962f-3322bc600589"),
					OutdialID:  uuid.FromStringOrNil("fb752900-b1de-11ec-a188-632f7f0a767c"),

					ActiveflowID:  uuid.FromStringOrNil("2175bea6-b1dc-11ec-91d7-1793f6803550"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b0a22f5e-b2a1-11ec-9304-4b68c392be3e"),
						CustomerID: uuid.FromStringOrNil("2121b158-b1dc-11ec-ab32-6f32cc331ec2"),
					},
					CampaignID: uuid.FromStringOrNil("b1080ce8-b2a1-11ec-962f-3322bc600589"),
					OutdialID:  uuid.FromStringOrNil("fb752900-b1de-11ec-a188-632f7f0a767c"),

					ActiveflowID:  uuid.FromStringOrNil("2175bea6-b1dc-11ec-91d7-1793f6803550"),
					ReferenceType: outdialtargetcall.ReferenceTypeNone,

					Status: outdialtargetcall.StatusProgressing,

					DestinationIndex: 0,
					TryCount:         1,

					TMCreate: curTime,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:          dbTest,
				cache:       mockCache,
				utilHandler: utilhandler.NewUtilHandler(),
			}

			ctx := context.Background()

			for _, targetCall := range tt.outdialTargetCalls {

				mockCache.EXPECT().OutdialTargetCallSet(ctx, targetCall).Return(nil)
				if targetCall.ActiveflowID != uuid.Nil {
					mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, targetCall).Return(nil)
				}
				if targetCall.ReferenceID != uuid.Nil {
					mockCache.EXPECT().OutdialTargetCallSetByReferenceID(ctx, targetCall).Return(nil)
				}
				if err := h.OutdialTargetCallCreate(ctx, targetCall); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.OutdialTargetCallList(ctx, GetCurTime(), 100, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialTargetCallUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	curTime := func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC); return &t }()

	tests := []struct {
		name              string
		outdialTargetCall *outdialtargetcall.OutdialTargetCall
		fields            map[outdialtargetcall.Field]any
		expectRes         *outdialtargetcall.OutdialTargetCall
	}{
		{
			"update status",
			&outdialtargetcall.OutdialTargetCall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1d2e3f4-b1bb-11ec-91c0-1b8448615480"),
					CustomerID: uuid.FromStringOrNil("c2d3e4f5-b1bb-11ec-9c75-fbb8e010e73f"),
				},
				Status:   outdialtargetcall.StatusProgressing,
				TMCreate: curTime,
			},
			map[outdialtargetcall.Field]any{
				outdialtargetcall.FieldStatus: outdialtargetcall.StatusDone,
			},
			&outdialtargetcall.OutdialTargetCall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1d2e3f4-b1bb-11ec-91c0-1b8448615480"),
					CustomerID: uuid.FromStringOrNil("c2d3e4f5-b1bb-11ec-9c75-fbb8e010e73f"),
				},
				Status: outdialtargetcall.StatusDone,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			mockCache.EXPECT().OutdialTargetCallSet(ctx, tt.outdialTargetCall).Return(nil)
			if tt.outdialTargetCall.ActiveflowID != uuid.Nil {
				mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, tt.outdialTargetCall).Return(nil)
			}
			if tt.outdialTargetCall.ReferenceID != uuid.Nil {
				mockCache.EXPECT().OutdialTargetCallSetByReferenceID(ctx, tt.outdialTargetCall).Return(nil)
			}
			if err := h.OutdialTargetCallCreate(ctx, tt.outdialTargetCall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetCallSet(gomock.Any(), gomock.Any())
			if tt.outdialTargetCall.ActiveflowID != uuid.Nil {
				mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, gomock.Any()).Return(nil)
			}
			if tt.outdialTargetCall.ReferenceID != uuid.Nil {
				mockCache.EXPECT().OutdialTargetCallSetByReferenceID(ctx, gomock.Any()).Return(nil)
			}
			if err := h.OutdialTargetCallUpdate(ctx, tt.outdialTargetCall.ID, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().OutdialTargetCallGet(gomock.Any(), tt.outdialTargetCall.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().OutdialTargetCallSet(gomock.Any(), gomock.Any())
			if tt.outdialTargetCall.ActiveflowID != uuid.Nil {
				mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, gomock.Any()).Return(nil)
			}
			if tt.outdialTargetCall.ReferenceID != uuid.Nil {
				mockCache.EXPECT().OutdialTargetCallSetByReferenceID(ctx, gomock.Any()).Return(nil)
			}
			res, err := h.OutdialTargetCallGet(ctx, tt.outdialTargetCall.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			tt.expectRes.TMCreate = res.TMCreate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
