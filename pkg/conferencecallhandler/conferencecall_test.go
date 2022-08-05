package conferencecallhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		conferenceID  uuid.UUID
		referenceType conferencecall.ReferenceType
		referenceID   uuid.UUID

		responseConferencecall *conferencecall.Conferencecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("6f6934f0-1350-11ed-8084-2f82e5efd9c2"),
			uuid.FromStringOrNil("6fabe796-1350-11ed-a9be-63d034c16c8d"),
			conferencecall.ReferenceTypeCall,
			uuid.FromStringOrNil("6fdaccaa-1350-11ed-8a93-cb0e3c8d6bf8"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, gomock.Any()).Return(tt.responseConferencecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseConferencecall.CustomerID, conferencecall.EventTypeConferencecallJoining, tt.responseConferencecall)
			res, err := h.Create(ctx, tt.customerID, tt.conferenceID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_GetByReferenceID(t *testing.T) {

	tests := []struct {
		name string

		referenceID uuid.UUID

		responseConferencecall *conferencecall.Conferencecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("e1d0969a-1351-11ed-9ea6-9b31710d7e97"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("e85478a6-1351-11ed-9dfa-f744cd7d0d42"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallGetByReferenceID(ctx, tt.referenceID).Return(tt.responseConferencecall, nil)
			res, err := h.GetByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_updateStatus(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		status conferencecall.Status

		responseConferencecall *conferencecall.Conferencecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("385e5d94-1352-11ed-af65-03b479fb4c5b"),
			conferencecall.StatusJoining,

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("385e5d94-1352-11ed-af65-03b479fb4c5b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallUpdateStatus(ctx, tt.id, tt.status).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, tt.id).Return(tt.responseConferencecall, nil)
			res, err := h.updateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_updateStatusByReferenceID(t *testing.T) {

	tests := []struct {
		name string

		referenceID uuid.UUID
		status      conferencecall.Status

		responseConferencecall *conferencecall.Conferencecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("7e17659c-1352-11ed-bfbe-774659fa11e9"),
			conferencecall.StatusJoining,

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("7e47b5e4-1352-11ed-b286-ef1d6830ab8a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallGetByReferenceID(ctx, tt.referenceID).Return(tt.responseConferencecall, nil)
			mockDB.EXPECT().ConferencecallUpdateStatus(ctx, tt.responseConferencecall.ID, tt.status).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, tt.responseConferencecall.ID).Return(tt.responseConferencecall, nil)
			res, err := h.updateStatusByReferenceID(ctx, tt.referenceID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_UpdateStatusLeaving(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConferencecall *conferencecall.Conferencecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("b85d4172-13bb-11ed-b867-7b2a6353ea11"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("b85d4172-13bb-11ed-b867-7b2a6353ea11"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallUpdateStatus(ctx, tt.responseConferencecall.ID, conferencecall.StatusLeaving).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, tt.responseConferencecall.ID).Return(tt.responseConferencecall, nil)

			res, err := h.UpdateStatusLeaving(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_UpdateStatusLeaved(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConferencecall *conferencecall.Conferencecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("54b337fc-13bc-11ed-86cb-5b5a22ffd1e3"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("54b337fc-13bc-11ed-86cb-5b5a22ffd1e3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallUpdateStatus(ctx, tt.responseConferencecall.ID, conferencecall.StatusLeaved).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, tt.responseConferencecall.ID).Return(tt.responseConferencecall, nil)

			res, err := h.UpdateStatusLeaved(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}
