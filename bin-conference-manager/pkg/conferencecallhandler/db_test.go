package conferencecallhandler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		conferenceID  uuid.UUID
		referenceType conferencecall.ReferenceType
		referenceID   uuid.UUID

		responseUUID           uuid.UUID
		responseConferencecall *conferencecall.Conferencecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("6f6934f0-1350-11ed-8084-2f82e5efd9c2"),
			uuid.FromStringOrNil("6fabe796-1350-11ed-a9be-63d034c16c8d"),
			conferencecall.ReferenceTypeCall,
			uuid.FromStringOrNil("6fdaccaa-1350-11ed-8a93-cb0e3c8d6bf8"),

			uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().ConferencecallCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, gomock.Any()).Return(tt.responseConferencecall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, conferencecall.EventTypeConferencecallJoining, tt.responseConferencecall)
			mockReq.EXPECT().ConferenceV1ConferencecallHealthCheck(ctx, tt.responseConferencecall.ID, 0, defaultHealthCheckDelay)
			res, err := h.Create(ctx, tt.customerID, tt.conferenceID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[string]string

		responseConferencecall []*conferencecall.Conferencecall
	}{
		{
			"normal",

			10,
			"2023-01-03 21:35:02.809",
			map[string]string{
				"deleted": "false",
			},

			[]*conferencecall.Conferencecall{
				{
					ID: uuid.FromStringOrNil("ae267468-50c2-11ee-9ddb-0f6ca6c40243"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallGets(ctx, tt.size, tt.token, tt.filters).Return(tt.responseConferencecall, nil)
			res, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
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

func Test_updateStatusJoined(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConferencecall *conferencecall.Conferencecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("6c149caa-94d2-11ed-9638-c3b106edcdbd"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("6c149caa-94d2-11ed-9638-c3b106edcdbd"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallUpdateStatus(ctx, tt.responseConferencecall.ID, conferencecall.StatusJoined).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, tt.responseConferencecall.ID).Return(tt.responseConferencecall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, conferencecall.EventTypeConferencecallJoined, tt.responseConferencecall)

			res, err := h.updateStatusJoined(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_updateStatusLeaving(t *testing.T) {

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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallUpdateStatus(ctx, tt.responseConferencecall.ID, conferencecall.StatusLeaving).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, tt.responseConferencecall.ID).Return(tt.responseConferencecall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, conferencecall.EventTypeConferencecallLeaving, tt.responseConferencecall)

			res, err := h.updateStatusLeaving(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_updateStatusLeaved(t *testing.T) {

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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallUpdateStatus(ctx, tt.responseConferencecall.ID, conferencecall.StatusLeaved).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, tt.responseConferencecall.ID).Return(tt.responseConferencecall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, conferencecall.EventTypeConferencecallLeaved, tt.responseConferencecall)

			res, err := h.updateStatusLeaved(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}
