package outdialtargethandler

import (
	"context"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		outdialID    uuid.UUID
		outdialName  string
		detail       string
		data         string
		destination0 *commonaddress.Address
		destination1 *commonaddress.Address
		destination2 *commonaddress.Address
		destination3 *commonaddress.Address
		destination4 *commonaddress.Address
	}{
		{
			"normal",

			uuid.FromStringOrNil("4b290cbe-b2c0-11ec-ae19-9773a0bdaf28"),
			"test name",
			"test detail",
			"test data",
			&commonaddress.Address{},
			nil,
			nil,
			nil,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().OutdialTargetGet(ctx, gomock.Any()).Return(&outdialtarget.OutdialTarget{}, nil)

			_, err := h.Create(
				ctx,
				tt.outdialID,
				tt.outdialName,
				tt.detail,
				tt.data,
				tt.destination0,
				tt.destination1,
				tt.destination2,
				tt.destination3,
				tt.destination4,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		outdialID uuid.UUID
	}{
		{
			"normal",
			uuid.FromStringOrNil("81597cf8-b561-11ec-bd35-fbc2e9bf73b8"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetDelete(ctx, tt.outdialID).Return(nil)
			mockDB.EXPECT().OutdialTargetGet(ctx, tt.outdialID).Return(&outdialtarget.OutdialTarget{}, nil)

			_, err := h.Delete(ctx, tt.outdialID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ListByOutdialID(t *testing.T) {

	tests := []struct {
		name string

		outdialID uuid.UUID
		token     string
		limit     uint64
	}{
		{
			"normal",

			uuid.FromStringOrNil("05b5c738-b2c2-11ec-acd3-27fc70dc1b15"),
			"2020-10-10T03:30:17.000000Z",
			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetList(ctx, tt.token, tt.limit, gomock.Any()).Return([]*outdialtarget.OutdialTarget{}, nil)
			_, err := h.GetsByOutdialID(ctx, tt.outdialID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetAvailable(t *testing.T) {

	tests := []struct {
		name string

		outdialID uuid.UUID
		tryCount0 int
		tryCount1 int
		tryCount2 int
		tryCount3 int
		tryCount4 int
		limit     uint64
	}{
		{
			"normal",

			uuid.FromStringOrNil("e16c55f8-b2c2-11ec-8d2b-bf830eef4a7d"),
			1,
			2,
			3,
			4,
			5,
			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetGetAvailable(ctx, tt.outdialID, tt.tryCount0, tt.tryCount1, tt.tryCount2, tt.tryCount3, tt.tryCount4, tt.limit).Return([]*outdialtarget.OutdialTarget{}, nil)
			_, err := h.GetAvailable(ctx, tt.outdialID, tt.tryCount0, tt.tryCount1, tt.tryCount2, tt.tryCount3, tt.tryCount4, tt.limit)
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
			uuid.FromStringOrNil("a1b2c3d4-b561-11ec-bd35-fbc2e9bf73b8"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetGet(ctx, tt.id).Return(&outdialtarget.OutdialTarget{}, nil)

			_, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		status outdialtarget.Status
	}{
		{
			"update to progressing",
			uuid.FromStringOrNil("b2c3d4e5-b561-11ec-bd35-fbc2e9bf73b8"),
			outdialtarget.StatusProgressing,
		},
		{
			"update to done",
			uuid.FromStringOrNil("c3d4e5f6-b561-11ec-bd35-fbc2e9bf73b8"),
			outdialtarget.StatusDone,
		},
		{
			"update to idle",
			uuid.FromStringOrNil("d4e5f6g7-b561-11ec-bd35-fbc2e9bf73b8"),
			outdialtarget.StatusIdle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetUpdate(ctx, tt.id, gomock.Any()).Return(nil)
			mockDB.EXPECT().OutdialTargetGet(ctx, tt.id).Return(&outdialtarget.OutdialTarget{Status: tt.status}, nil)

			result, err := h.UpdateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if result.Status != tt.status {
				t.Errorf("Expected status %v, got %v", tt.status, result.Status)
			}
		})
	}
}

func Test_UpdateProgressing(t *testing.T) {

	tests := []struct {
		name string

		id               uuid.UUID
		destinationIndex int
	}{
		{
			"update destination index 0",
			uuid.FromStringOrNil("e5f6g7h8-b561-11ec-bd35-fbc2e9bf73b8"),
			0,
		},
		{
			"update destination index 1",
			uuid.FromStringOrNil("f6g7h8i9-b561-11ec-bd35-fbc2e9bf73b8"),
			1,
		},
		{
			"update destination index 4",
			uuid.FromStringOrNil("g7h8i9j0-b561-11ec-bd35-fbc2e9bf73b8"),
			4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetUpdateProgressing(ctx, tt.id, tt.destinationIndex).Return(nil)
			mockDB.EXPECT().OutdialTargetGet(ctx, tt.id).Return(&outdialtarget.OutdialTarget{}, nil)

			_, err := h.UpdateProgressing(ctx, tt.id, tt.destinationIndex)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
