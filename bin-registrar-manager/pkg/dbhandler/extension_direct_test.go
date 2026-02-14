package dbhandler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/extensiondirect"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

func Test_ExtensionDirectCreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name string
		ed   *extensiondirect.ExtensionDirect

		responseCurTime *time.Time
		expectRes       *extensiondirect.ExtensionDirect
	}

	tests := []test{
		{
			"normal",
			&extensiondirect.ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-2222-2222-2222-000000000001"),
				},
				ExtensionID: uuid.FromStringOrNil("a1b2c3d4-3333-3333-3333-000000000001"),
				Hash:        "abc123def456",
			},

			curTime,
			&extensiondirect.ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-2222-2222-2222-000000000001"),
				},
				ExtensionID: uuid.FromStringOrNil("a1b2c3d4-3333-3333-3333-000000000001"),
				Hash:        "abc123def456",

				TMCreate: curTime,
				TMUpdate: nil,
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
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.ExtensionDirectCreate(ctx, tt.ed); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ExtensionDirectGet(ctx, tt.ed.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionDirectGetByExtensionID(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name string
		ed   *extensiondirect.ExtensionDirect

		extensionID uuid.UUID

		responseCurTime *time.Time
		expectRes       *extensiondirect.ExtensionDirect
	}

	tests := []test{
		{
			"normal",
			&extensiondirect.ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000001"),
				},
				ExtensionID: uuid.FromStringOrNil("b1b2c3d4-3333-3333-3333-000000000001"),
				Hash:        "bbb123def456",
			},

			uuid.FromStringOrNil("b1b2c3d4-3333-3333-3333-000000000001"),

			curTime,
			&extensiondirect.ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("b1b2c3d4-2222-2222-2222-000000000001"),
				},
				ExtensionID: uuid.FromStringOrNil("b1b2c3d4-3333-3333-3333-000000000001"),
				Hash:        "bbb123def456",

				TMCreate: curTime,
				TMUpdate: nil,
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
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.ExtensionDirectCreate(ctx, tt.ed); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ExtensionDirectGetByExtensionID(ctx, tt.extensionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionDirectGetByHash(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name string
		ed   *extensiondirect.ExtensionDirect

		hash string

		responseCurTime *time.Time
		expectRes       *extensiondirect.ExtensionDirect
	}

	tests := []test{
		{
			"normal",
			&extensiondirect.ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-2222-2222-2222-000000000001"),
				},
				ExtensionID: uuid.FromStringOrNil("c1b2c3d4-3333-3333-3333-000000000001"),
				Hash:        "ccc123def456",
			},

			"ccc123def456",

			curTime,
			&extensiondirect.ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("c1b2c3d4-2222-2222-2222-000000000001"),
				},
				ExtensionID: uuid.FromStringOrNil("c1b2c3d4-3333-3333-3333-000000000001"),
				Hash:        "ccc123def456",

				TMCreate: curTime,
				TMUpdate: nil,
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
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.ExtensionDirectCreate(ctx, tt.ed); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ExtensionDirectGetByHash(ctx, tt.hash)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionDirectDelete(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name string
		ed   *extensiondirect.ExtensionDirect

		responseCurTime *time.Time
		expectRes       *extensiondirect.ExtensionDirect
	}

	tests := []test{
		{
			"normal",
			&extensiondirect.ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-2222-2222-2222-000000000001"),
				},
				ExtensionID: uuid.FromStringOrNil("d1b2c3d4-3333-3333-3333-000000000001"),
				Hash:        "ddd123def456",
			},

			curTime,
			&extensiondirect.ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("d1b2c3d4-2222-2222-2222-000000000001"),
				},
				ExtensionID: uuid.FromStringOrNil("d1b2c3d4-3333-3333-3333-000000000001"),
				Hash:        "ddd123def456",

				TMCreate: curTime,
				TMUpdate: curTime,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.ExtensionDirectCreate(ctx, tt.ed); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.ExtensionDirectDelete(ctx, tt.ed.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ExtensionDirectGet(ctx, tt.ed.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionDirectUpdate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name     string
		edCreate *extensiondirect.ExtensionDirect

		id     uuid.UUID
		fields map[extensiondirect.Field]any

		responseCurTime *time.Time
		expectRes       *extensiondirect.ExtensionDirect
	}

	tests := []test{
		{
			"normal",
			&extensiondirect.ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("e1b2c3d4-2222-2222-2222-000000000001"),
				},
				ExtensionID: uuid.FromStringOrNil("e1b2c3d4-3333-3333-3333-000000000001"),
				Hash:        "eee123def456",
			},

			uuid.FromStringOrNil("e1b2c3d4-1111-1111-1111-000000000001"),
			map[extensiondirect.Field]any{
				extensiondirect.FieldHash: "updated12hash",
			},

			curTime,
			&extensiondirect.ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e1b2c3d4-1111-1111-1111-000000000001"),
					CustomerID: uuid.FromStringOrNil("e1b2c3d4-2222-2222-2222-000000000001"),
				},
				ExtensionID: uuid.FromStringOrNil("e1b2c3d4-3333-3333-3333-000000000001"),
				Hash:        "updated12hash",

				TMCreate: curTime,
				TMUpdate: curTime,
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
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.ExtensionDirectCreate(ctx, tt.edCreate); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.ExtensionDirectUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ExtensionDirectGet(ctx, tt.edCreate.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionDirectGetByExtensionIDs(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	type test struct {
		name string
		eds  []*extensiondirect.ExtensionDirect

		extensionIDs []uuid.UUID

		responseCurTime *time.Time
		expectCount     int
	}

	tests := []test{
		{
			"empty_list",
			[]*extensiondirect.ExtensionDirect{},
			[]uuid.UUID{},
			curTime,
			0,
		},
		{
			"multiple_extension_directs",
			[]*extensiondirect.ExtensionDirect{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("f1b2c3d4-1111-1111-1111-000000000001"),
						CustomerID: uuid.FromStringOrNil("f1b2c3d4-2222-2222-2222-000000000001"),
					},
					ExtensionID: uuid.FromStringOrNil("f1b2c3d4-3333-3333-3333-000000000001"),
					Hash:        "fff123def456",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("f1b2c3d4-4444-4444-4444-000000000001"),
						CustomerID: uuid.FromStringOrNil("f1b2c3d4-2222-2222-2222-000000000001"),
					},
					ExtensionID: uuid.FromStringOrNil("f1b2c3d4-5555-5555-5555-000000000001"),
					Hash:        "fff456ghi789",
				},
			},
			[]uuid.UUID{
				uuid.FromStringOrNil("f1b2c3d4-3333-3333-3333-000000000001"),
				uuid.FromStringOrNil("f1b2c3d4-5555-5555-5555-000000000001"),
			},
			curTime,
			2,
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

			for _, ed := range tt.eds {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				if err := h.ExtensionDirectCreate(ctx, ed); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.ExtensionDirectGetByExtensionIDs(ctx, tt.extensionIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.expectCount {
				t.Errorf("Wrong count. expect: %d, got: %d", tt.expectCount, len(res))
			}
		})
	}
}
