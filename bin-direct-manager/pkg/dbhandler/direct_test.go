package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/models/direct"
)

func Test_DirectCreate(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name   string
		direct *direct.Direct

		responseCurTime *time.Time
		expectRes       *direct.Direct
	}{
		{
			name: "normal",
			direct: &direct.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
				},
				ResourceType: "extension",
				ResourceID:   uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
				Hash:         "direct.abcdef123456",
			},

			responseCurTime: curTime,
			expectRes: &direct.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
				},
				ResourceType: "extension",
				ResourceID:   uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
				Hash:         "direct.abcdef123456",
				TMCreate:     curTime,
				TMUpdate:     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.DirectCreate(ctx, tt.direct); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.DirectGet(ctx, tt.direct.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			// cleanup
			_ = h.DirectDelete(ctx, tt.direct.ID)
		})
	}
}

func Test_DirectGetByHash(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name   string
		direct *direct.Direct
		hash   string

		responseCurTime *time.Time
		expectRes       *direct.Direct
	}{
		{
			name: "normal",
			direct: &direct.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-50d7-11ec-a6b1-8f9671a9e70e"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
				},
				ResourceType: "conference",
				ResourceID:   uuid.FromStringOrNil("d4e5f6a7-4e69-11ec-afe3-77ba49fae527"),
				Hash:         "direct.hash123test",
			},
			hash: "direct.hash123test",

			responseCurTime: curTime,
			expectRes: &direct.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-50d7-11ec-a6b1-8f9671a9e70e"),
					CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
				},
				ResourceType: "conference",
				ResourceID:   uuid.FromStringOrNil("d4e5f6a7-4e69-11ec-afe3-77ba49fae527"),
				Hash:         "direct.hash123test",
				TMCreate:     curTime,
				TMUpdate:     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.DirectCreate(ctx, tt.direct); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.DirectGetByHash(ctx, tt.hash)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			// cleanup
			_ = h.DirectDelete(ctx, tt.direct.ID)
		})
	}
}

func Test_DirectGets(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name    string
		directs []*direct.Direct

		size    uint64
		filters map[direct.Field]any

		responseCurTime *time.Time
		expectRes       []*direct.Direct
	}{
		{
			name: "normal",
			directs: []*direct.Direct{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
						CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					},
					ResourceType: "extension",
					ResourceID:   uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					Hash:         "direct.list00000001",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
						CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					},
					ResourceType: "conference",
					ResourceID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
					Hash:         "direct.list00000002",
				},
			},

			size: 2,
			filters: map[direct.Field]any{
				direct.FieldCustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
			},

			responseCurTime: curTime,
			expectRes: []*direct.Direct{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
						CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					},
					ResourceType: "extension",
					ResourceID:   uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					Hash:         "direct.list00000001",
					TMCreate:     curTime,
					TMUpdate:     nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
						CustomerID: uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54"),
					},
					ResourceType: "conference",
					ResourceID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
					Hash:         "direct.list00000002",
					TMCreate:     curTime,
					TMUpdate:     nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
			}
			ctx := context.Background()

			for _, d := range tt.directs {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				if err := h.DirectCreate(ctx, d); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.DirectGets(ctx, tt.size, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			// cleanup
			for _, d := range tt.directs {
				_ = h.DirectDelete(ctx, d.ID)
			}
		})
	}
}

func Test_DirectGets_Empty(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
	}
	ctx := context.Background()

	filters := map[direct.Field]any{
		direct.FieldCustomerID: uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff"),
	}

	res, err := h.DirectGets(ctx, 10, utilhandler.TimeGetCurTime(), filters)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res == nil {
		t.Errorf("Expected non-nil empty slice, got nil")
	}

	if len(res) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(res))
	}
}

func Test_DirectDelete(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name   string
		direct *direct.Direct

		responseCurTime *time.Time
	}{
		{
			name: "normal",
			direct: &direct.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3963dbc6-50d7-11ec-916c-1b7d3056c90a"),
					CustomerID: uuid.FromStringOrNil("dd805a3e-7fe1-11ec-b37d-134362dec03c"),
				},
				ResourceType: "agent",
				ResourceID:   uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				Hash:         "direct.deletetest01",
			},

			responseCurTime: curTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.DirectCreate(ctx, tt.direct); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			err := h.DirectDelete(ctx, tt.direct.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// verify it's actually deleted
			_, err = h.DirectGet(ctx, tt.direct.ID)
			if err == nil {
				t.Errorf("Expected error after deletion, got nil")
			}
		})
	}
}

func Test_DirectUpdate(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name   string
		direct *direct.Direct

		updateFields map[direct.Field]any

		responseCurTime *time.Time
		expectHash      string
	}{
		{
			name: "normal",
			direct: &direct.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ae1e0150-4c6b-11ec-922d-27336e407864"),
					CustomerID: uuid.FromStringOrNil("b7442490-7fe1-11ec-a66b-b7a03a06132f"),
				},
				ResourceType: "ai",
				ResourceID:   uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
				Hash:         "direct.update000001",
			},

			updateFields: map[direct.Field]any{
				direct.FieldHash: "direct.updated00001",
			},

			responseCurTime: curTime,
			expectHash:      "direct.updated00001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.DirectCreate(ctx, tt.direct); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			err := h.DirectUpdate(ctx, tt.direct.ID, tt.updateFields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.DirectGet(ctx, tt.direct.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.Hash != tt.expectHash {
				t.Errorf("Wrong match. expect hash: %s, got: %s", tt.expectHash, res.Hash)
			}

			// cleanup
			_ = h.DirectDelete(ctx, tt.direct.ID)
		})
	}
}
