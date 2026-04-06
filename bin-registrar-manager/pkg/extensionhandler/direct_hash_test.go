package extensionhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func Test_DirectHashRegenerate_ExistingDirect(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseExtension        *extension.Extension
		responseDirect           *dmdirect.Direct
		responseUpdatedExtension *extension.Extension

		expectUpdateFields map[extension.Field]any
	}

	tests := []test{
		{
			name: "extension with existing direct_id regenerates",

			id: uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),

			responseExtension: &extension.Extension{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				DirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				DirectHash: "oldhash123",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				ResourceType: "extension",
				ResourceID:   uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
				Hash:         "newhash456",
			},
			responseUpdatedExtension: &extension.Extension{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				DirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				DirectHash: "newhash456",
			},

			expectUpdateFields: map[extension.Field]any{
				extension.FieldDirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				extension.FieldDirectHash: "newhash456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDBBin := dbhandler.NewMockDBHandler(mc)
			h := &extensionHandler{
				reqHandler: mockReq,
				dbBin:      mockDBBin,
			}

			ctx := context.Background()

			mockDBBin.EXPECT().ExtensionGet(ctx, tt.id).Return(tt.responseExtension, nil)
			mockReq.EXPECT().DirectV1DirectRegenerate(ctx, tt.responseExtension.DirectID).Return(tt.responseDirect, nil)
			mockDBBin.EXPECT().ExtensionUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDBBin.EXPECT().ExtensionGet(ctx, tt.id).Return(tt.responseUpdatedExtension, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedExtension) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedExtension, res)
			}
		})
	}
}

func Test_DirectHashRegenerate_NoDirect(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseExtension        *extension.Extension
		responseDirect           *dmdirect.Direct
		responseUpdatedExtension *extension.Extension

		expectUpdateFields map[extension.Field]any
	}

	tests := []test{
		{
			name: "extension with no direct_id creates new direct",

			id: uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),

			responseExtension: &extension.Extension{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				DirectID:   uuid.Nil,
				DirectHash: "",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				ResourceType: "extension",
				ResourceID:   uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
				Hash:         "createdhash789",
			},
			responseUpdatedExtension: &extension.Extension{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				DirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				DirectHash: "createdhash789",
			},

			expectUpdateFields: map[extension.Field]any{
				extension.FieldDirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				extension.FieldDirectHash: "createdhash789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDBBin := dbhandler.NewMockDBHandler(mc)
			h := &extensionHandler{
				reqHandler: mockReq,
				dbBin:      mockDBBin,
			}

			ctx := context.Background()

			mockDBBin.EXPECT().ExtensionGet(ctx, tt.id).Return(tt.responseExtension, nil)
			mockReq.EXPECT().DirectV1DirectCreate(ctx, tt.responseExtension.CustomerID, dmdirect.ResourceTypeExtension, tt.id).Return(tt.responseDirect, nil)
			mockDBBin.EXPECT().ExtensionUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDBBin.EXPECT().ExtensionGet(ctx, tt.id).Return(tt.responseUpdatedExtension, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedExtension) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedExtension, res)
			}
		})
	}
}
