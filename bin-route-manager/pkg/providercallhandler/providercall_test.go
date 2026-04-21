package providercallhandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

func Test_Get(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseProviderCall *providercall.ProviderCall
	}{
		{
			"normal",
			uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			&providercall.ProviderCall{
				ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerCallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ProviderCallGet(ctx, tt.id).Return(tt.responseProviderCall, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.responseProviderCall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseProviderCall, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {
	tests := []struct {
		name string

		customerID   uuid.UUID
		providerID   uuid.UUID
		flowID       uuid.UUID
		source       *commonaddress.Address
		destinations []commonaddress.Address
		anonymous    string
		callIDs      []uuid.UUID
		groupcallIDs []uuid.UUID

		responseProviderCall *providercall.ProviderCall
	}{
		{
			"normal",

			uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
			uuid.Nil,
			&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+14155551234"},
			[]commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+821012345678"}},
			"auto",
			[]uuid.UUID{uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")},
			[]uuid.UUID{},

			&providercall.ProviderCall{
				ID:         uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
				ProviderID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerCallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			// Accept any ProviderCall struct (the handler mints its own UUID).
			mockDB.EXPECT().ProviderCallCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ProviderCallGet(ctx, gomock.Any()).Return(tt.responseProviderCall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, providercall.EventTypeProviderCallCreated, tt.responseProviderCall).Return()

			res, err := h.Create(ctx, tt.customerID, tt.providerID, tt.flowID, tt.source, tt.destinations, tt.anonymous, tt.callIDs, tt.groupcallIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.responseProviderCall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseProviderCall, res)
			}
		})
	}
}

func Test_List(t *testing.T) {
	tests := []struct {
		name string

		token     string
		limit     uint64
		filters   map[providercall.Field]any
		responses []*providercall.ProviderCall
	}{
		{
			"normal",
			"",
			10,
			map[providercall.Field]any{providercall.FieldCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001")},
			[]*providercall.ProviderCall{{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")}},
		},
		{
			"empty result",
			"",
			10,
			map[providercall.Field]any{},
			[]*providercall.ProviderCall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerCallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ProviderCallList(ctx, tt.token, tt.limit, tt.filters).Return(tt.responses, nil)

			res, err := h.List(ctx, tt.token, tt.limit, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.responses) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responses, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseDeleted *providercall.ProviderCall
	}{
		{
			"normal",
			uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			&providercall.ProviderCall{
				ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerCallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ProviderCallDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ProviderCallGet(ctx, tt.id).Return(tt.responseDeleted, nil)
			mockNotify.EXPECT().PublishEvent(ctx, providercall.EventTypeProviderCallDeleted, tt.responseDeleted).Return()

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.responseDeleted) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseDeleted, res)
			}
		})
	}
}
