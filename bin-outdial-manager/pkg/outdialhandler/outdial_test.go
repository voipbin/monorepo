package outdialhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		campaignID  uuid.UUID
		outdialName string
		detail      string
		data        string
	}{
		{
			"normal",

			uuid.FromStringOrNil("617b1948-b631-11ec-83b8-eb85cbdfa671"),
			uuid.FromStringOrNil("6550027c-b631-11ec-a081-f37bd020eee2"),
			"test name",
			"test detail",
			"test data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &outdialHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialCreate(ctx, gomock.Any().Return(nil)
			mockDB.EXPECT().OutdialGet(ctx, gomock.Any().Return(&outdial.Outdial{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), outdial.EventTypeOutdialCreated, gomock.Any())

			_, err := h.Create(ctx, tt.customerID, tt.campaignID, tt.outdialName, tt.detail, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("094dcfbc-b632-11ec-9d5c-bbf5bde28472"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &outdialHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialDelete(ctx, tt.id.Return(nil)
			mockDB.EXPECT().OutdialGet(ctx, tt.id.Return(&outdial.Outdial{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), outdial.EventTypeOutdialDeleted, gomock.Any())

			_, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("661f8b5e-b632-11ec-bbe6-6b59b41d8015"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &outdialHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialGet(ctx, tt.id.Return(&outdial.Outdial{}, nil)

			_, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		token      string
		limit      uint64
	}{
		{
			"normal",

			uuid.FromStringOrNil("8777908a-b632-11ec-95d3-07799e382868"),
			"2020-10-10 03:30:17.000000",
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &outdialHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialGets(ctx, tt.token, tt.limit, gomock.Any().Return([]*outdial.Outdial{}, nil)

			_, err := h.GetsByCustomerID(ctx, tt.customerID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		updateName string
		detail     string
	}{
		{
			"normal",

			uuid.FromStringOrNil("fd1e7e66-b632-11ec-9fd4-aba605cc2755"),
			"update name",
			"update detail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &outdialHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialUpdate(ctx, tt.id, gomock.Any().Return(nil)
			mockDB.EXPECT().OutdialGet(ctx, tt.id.Return(&outdial.Outdial{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), outdial.EventTypeOutdialUpdated, gomock.Any())

			_, err := h.UpdateBasicInfo(ctx, tt.id, tt.updateName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateCampaignID(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		campaignID uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("20a1227a-b634-11ec-be5c-c35101224e82"),
			uuid.FromStringOrNil("20ca3854-b634-11ec-ade7-3bd0d6a87630"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &outdialHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialUpdate(ctx, tt.id, gomock.Any().Return(nil)
			mockDB.EXPECT().OutdialGet(ctx, tt.id.Return(&outdial.Outdial{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), outdial.EventTypeOutdialUpdated, gomock.Any())

			_, err := h.UpdateCampaignID(ctx, tt.id, tt.campaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateData(t *testing.T) {

	tests := []struct {
		name string

		id   uuid.UUID
		data string
	}{
		{
			"normal",

			uuid.FromStringOrNil("536af80c-b634-11ec-9303-fb88aa8d178a"),
			"update data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &outdialHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialUpdate(ctx, tt.id, gomock.Any().Return(nil)
			mockDB.EXPECT().OutdialGet(ctx, tt.id.Return(&outdial.Outdial{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), outdial.EventTypeOutdialUpdated, gomock.Any())

			_, err := h.UpdateData(ctx, tt.id, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
