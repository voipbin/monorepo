package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestQueuecallGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name      string
		user      *user.User
		pageToken string
		pageSize  uint64

		response  []qmqueuecall.Queuecall
		expectRes []*qmqueuecall.Event
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			"2021-03-01 01:00:00.995000",
			10,

			[]qmqueuecall.Queuecall{
				{
					ID: uuid.FromStringOrNil("cccf3e1a-6413-11ec-9874-afa5340c4843"),
				},
			},
			[]*qmqueuecall.Event{
				{
					ID: uuid.FromStringOrNil("cccf3e1a-6413-11ec-9874-afa5340c4843"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().QMV1QueuecallGets(gomock.Any(), tt.user.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.QueuecallGets(tt.user, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, num := range res {
				num.TMCreate = ""
				num.TMUpdate = ""
				num.TMDelete = ""
			}

			if !reflect.DeepEqual(res[0], tt.expectRes[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func TestQueuecallGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string
		user *user.User
		id   uuid.UUID

		response  *qmqueuecall.Queuecall
		expectRes *qmqueuecall.Event
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("cd268152-6413-11ec-8e49-4bc7bcc6d465"),

			&qmqueuecall.Queuecall{
				ID:     uuid.FromStringOrNil("cd268152-6413-11ec-8e49-4bc7bcc6d465"),
				UserID: 1,
			},
			&qmqueuecall.Event{
				ID: uuid.FromStringOrNil("cd268152-6413-11ec-8e49-4bc7bcc6d465"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().QMV1QueuecallGet(gomock.Any(), tt.id).Return(tt.response, nil)

			res, err := h.QueuecallGet(tt.user, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestQueuecallDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name string
		user *user.User
		id   uuid.UUID

		response *qmqueuecall.Queuecall
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("00043d94-6414-11ec-9c13-eb81c8c76e8d"),

			&qmqueuecall.Queuecall{
				ID:     uuid.FromStringOrNil("00043d94-6414-11ec-9c13-eb81c8c76e8d"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().QMV1QueuecallGet(gomock.Any(), tt.id).Return(tt.response, nil)
			mockReq.EXPECT().QMV1QueuecallDelete(gomock.Any(), tt.id).Return(nil)

			if err := h.QueuecallDelete(tt.user, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestQueuecallDeleteByReferenceID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name        string
		user        *user.User
		referenceID uuid.UUID

		response *qmqueuecall.Queuecall
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("9b16e8ae-6414-11ec-a2b0-1f3fc925581e"),

			&qmqueuecall.Queuecall{
				ID:     uuid.FromStringOrNil("00043d94-6414-11ec-9c13-eb81c8c76e8d"),
				UserID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().QMV1QueuecallGet(gomock.Any(), tt.referenceID).Return(tt.response, nil)
			mockReq.EXPECT().QMV1QueuecallDelete(gomock.Any(), tt.referenceID).Return(nil)

			if err := h.QueuecallDelete(tt.user, tt.referenceID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
