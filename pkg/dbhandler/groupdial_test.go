package dbhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
)

func Test_GroupdialCreate(t *testing.T) {

	tests := []struct {
		name string

		data *groupdial.Groupdial
	}{
		{
			name: "normal",

			data: &groupdial.Groupdial{
				ID: uuid.FromStringOrNil("39ee40d7-9f83-45bb-ba29-7bb9de62c93e"),
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

			mockCache.EXPECT().GroupdialSet(ctx, tt.data).Return(nil)

			if errCreate := h.GroupdialCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}
		})
	}
}

func Test_GroupdialGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupdial *groupdial.Groupdial
	}{
		{
			name: "normal",
			id:   uuid.FromStringOrNil("3ca215ce-5fee-4854-9edb-2b75bbeef022"),
			responseGroupdial: &groupdial.Groupdial{
				ID: uuid.FromStringOrNil("3ca215ce-5fee-4854-9edb-2b75bbeef022"),
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

			mockCache.EXPECT().GroupdialGet(ctx, tt.id).Return(tt.responseGroupdial, nil)

			res, err := h.GroupdialGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseGroupdial, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupdial, res)
			}
		})
	}
}

func Test_GroupdialUpdate(t *testing.T) {

	tests := []struct {
		name string

		data *groupdial.Groupdial
	}{
		{
			name: "normal",

			data: &groupdial.Groupdial{
				ID: uuid.FromStringOrNil("6355ea6b-e87e-4895-944c-c986744dc1d5"),
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

			mockCache.EXPECT().GroupdialSet(ctx, tt.data).Return(nil)

			if errCreate := h.GroupdialUpdate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}
		})
	}
}
