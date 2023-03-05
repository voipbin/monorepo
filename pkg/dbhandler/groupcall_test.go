package dbhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
)

func Test_GroupcallCreate(t *testing.T) {

	tests := []struct {
		name string

		data *groupcall.Groupcall
	}{
		{
			name: "normal",

			data: &groupcall.Groupcall{
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

			mockUtil.EXPECT().GetCurTime().Return(utilhandler.GetCurTime()).AnyTimes()
			mockCache.EXPECT().GroupcallSet(ctx, tt.data).Return(nil)

			if errCreate := h.GroupcallCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}
		})
	}
}

func Test_GroupcallGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",
			id:   uuid.FromStringOrNil("3ca215ce-5fee-4854-9edb-2b75bbeef022"),
			responseGroupcall: &groupcall.Groupcall{
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

			mockCache.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			res, err := h.GroupcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseGroupcall, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_GroupcallUpdate(t *testing.T) {

	tests := []struct {
		name string

		data *groupcall.Groupcall
	}{
		{
			name: "normal",

			data: &groupcall.Groupcall{
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

			mockUtil.EXPECT().GetCurTime().Return(utilhandler.GetCurTime())
			mockCache.EXPECT().GroupcallSet(ctx, tt.data).Return(nil)

			if errCreate := h.GroupcallUpdate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}
		})
	}
}
