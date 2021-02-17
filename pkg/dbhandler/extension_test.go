package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	_ "github.com/mattn/go-sqlite3"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
)

func TestExtensionCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name      string
		ext       *models.Extension
		expectExt *models.Extension
	}

	tests := []test{
		{
			"test normal",
			&models.Extension{
				ID:     uuid.FromStringOrNil("3fecf3d6-6ebc-11eb-a0e7-23ecc297d9a5"),
				UserID: 1,

				DomainID: uuid.FromStringOrNil("4dc1e430-6ebc-11eb-b355-b35fc1cfc5a1"),

				EndpointID: "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AORID:      "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AuthID:     "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",

				Extension: "608cbfae-6ebc-11eb-a74b-671d17dda173",
				Password:  "7818abce-6ebc-11eb-b4fe-e748480c228a",
			},
			&models.Extension{
				ID:     uuid.FromStringOrNil("3fecf3d6-6ebc-11eb-a0e7-23ecc297d9a5"),
				UserID: 1,

				DomainID: uuid.FromStringOrNil("4dc1e430-6ebc-11eb-b355-b35fc1cfc5a1"),

				EndpointID: "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AORID:      "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",
				AuthID:     "608cbfae-6ebc-11eb-a74b-671d17dda173@test.sip.voipbin.net",

				Extension: "608cbfae-6ebc-11eb-a74b-671d17dda173",
				Password:  "7818abce-6ebc-11eb-b4fe-e748480c228a",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			if err := h.ExtensionCreate(context.Background(), tt.ext); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ExtensionGet(gomock.Any(), tt.ext.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			res, err := h.ExtensionGet(context.Background(), tt.ext.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectExt, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectExt, res)
			}
		})
	}
}

func TestExtensionDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string
		ext  *models.Extension
	}

	tests := []test{
		{
			"test normal",
			&models.Extension{
				ID:     uuid.FromStringOrNil("def11a70-6ebc-11eb-ae2b-d31ef2c6d22d"),
				UserID: 1,

				DomainID: uuid.FromStringOrNil("e22acb78-6ebc-11eb-848e-bfb26fcad363"),

				EndpointID: "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",
				AORID:      "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",
				AuthID:     "e56c33b2-6ebc-11eb-bada-4f15e459e32f@test.sip.voipbin.net",

				Extension: "e56c33b2-6ebc-11eb-bada-4f15e459e32f",
				Password:  "eb605618-6ebc-11eb-a421-4bbf5d9a2fac",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			if err := h.ExtensionCreate(context.Background(), tt.ext); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			if err := h.ExtensionDelete(context.Background(), tt.ext.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ExtensionGet(gomock.Any(), tt.ext.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			res, err := h.ExtensionGet(context.Background(), tt.ext.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}
		})
	}
}

func TestExtensionGetsByDomainID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name             string
		domainID         uuid.UUID
		limit            uint64
		extensions       []models.Extension
		expectExtensions []*models.Extension
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("3802a548-6f49-11eb-9362-3b77d3873657"),
			10,
			[]models.Extension{
				{
					ID:       uuid.FromStringOrNil("1d2cb402-6f49-11eb-a22c-5f2f23cba3a2"),
					UserID:   1,
					Name:     "test1",
					DomainID: uuid.FromStringOrNil("3802a548-6f49-11eb-9362-3b77d3873657"),
				},
				{
					ID:       uuid.FromStringOrNil("1d792bb6-6f49-11eb-be2e-0ff2f1c87d93"),
					UserID:   1,
					Name:     "test2",
					DomainID: uuid.FromStringOrNil("3802a548-6f49-11eb-9362-3b77d3873657"),
				},
			},
			[]*models.Extension{
				{
					ID:       uuid.FromStringOrNil("1d792bb6-6f49-11eb-be2e-0ff2f1c87d93"),
					UserID:   1,
					Name:     "test2",
					DomainID: uuid.FromStringOrNil("3802a548-6f49-11eb-9362-3b77d3873657"),
				},
				{
					ID:       uuid.FromStringOrNil("1d2cb402-6f49-11eb-a22c-5f2f23cba3a2"),
					UserID:   1,
					Name:     "test1",
					DomainID: uuid.FromStringOrNil("3802a548-6f49-11eb-9362-3b77d3873657"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			for _, d := range tt.extensions {
				mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
				if err := h.ExtensionCreate(ctx, &d); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			exts, err := h.ExtensionGetsByDomainID(ctx, tt.domainID, getCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, d := range exts {
				d.TMCreate = ""
			}

			if reflect.DeepEqual(exts, tt.expectExtensions) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectExtensions, exts)
			}
		})
	}
}

func TestExtensionUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name            string
		extension       *models.Extension
		updateExtension *models.Extension
		expectExtension *models.Extension
	}

	tests := []test{
		{
			"test normal",
			&models.Extension{
				ID:     uuid.FromStringOrNil("e3ebc6fe-711b-11eb-8385-ef7ccec2e41a"),
				UserID: 1,

				Name:     "test",
				Detail:   "detail",
				DomainID: uuid.FromStringOrNil("b2277afa-711b-11eb-a695-f71fad093e64"),

				Extension: "test",
				Password:  "password",
			},
			&models.Extension{
				ID:     uuid.FromStringOrNil("e3ebc6fe-711b-11eb-8385-ef7ccec2e41a"),
				UserID: 1,
				Name:   "update name",
				Detail: "update detail",

				Password: "update password",
			},
			&models.Extension{
				ID:     uuid.FromStringOrNil("e3ebc6fe-711b-11eb-8385-ef7ccec2e41a"),
				UserID: 1,

				Name:   "update name",
				Detail: "update detail",

				DomainID: uuid.FromStringOrNil("b2277afa-711b-11eb-a695-f71fad093e64"),

				Extension: "test",
				Password:  "update password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			if err := h.ExtensionCreate(ctx, tt.extension); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			if err := h.ExtensionUpdate(ctx, tt.updateExtension); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ExtensionGet(gomock.Any(), tt.extension.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ExtensionSet(gomock.Any(), gomock.Any())
			res, err := h.ExtensionGet(context.Background(), tt.extension.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectExtension, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectExtension, res)
			}
		})
	}
}
