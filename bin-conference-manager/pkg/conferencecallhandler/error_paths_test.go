package conferencecallhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

var errTestFailure = errors.New("simulated failure")

type testMocks struct {
	req    *requesthandler.MockRequestHandler
	db     *dbhandler.MockDBHandler
	notify *notifyhandler.MockNotifyHandler
	util   *utilhandler.MockUtilHandler
	conf   *conferencehandler.MockConferenceHandler
}

func newTestHandler(mc *gomock.Controller) (*conferencecallHandler, *testMocks) {
	m := &testMocks{
		req:    requesthandler.NewMockRequestHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
		conf:   conferencehandler.NewMockConferenceHandler(mc),
	}

	return &conferencecallHandler{
		utilHandler:       m.util,
		reqHandler:        m.req,
		notifyHandler:     m.notify,
		db:                m.db,
		conferenceHandler: m.conf,
	}, m
}

// waitDone waits until the channel is closed or fails the test.
func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for the asynchronous call")
	}
}

func Test_NewConferencecallHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	res, ok := NewConferencecallHandler(mockReq, mockNotify, mockDB, mockConf).(*conferencecallHandler)
	if !ok {
		t.Fatalf("Wrong type")
	}
	if res.reqHandler != mockReq || res.notifyHandler != mockNotify || res.db != mockDB || res.conferenceHandler != mockConf {
		t.Errorf("Wrong handler wiring")
	}
	if res.utilHandler == nil {
		t.Errorf("utilHandler must be set")
	}
}

func Test_ServiceStart_actions(t *testing.T) {
	ccID := uuid.FromStringOrNil("5b3a1d44-0001-11f1-9a1e-6f2c3b4d5e01")
	confID := uuid.FromStringOrNil("5b3a1d44-0002-11f1-9a1e-6f2c3b4d5e02")
	customerID := uuid.FromStringOrNil("5b3a1d44-0003-11f1-9a1e-6f2c3b4d5e03")
	activeflowID := uuid.FromStringOrNil("5b3a1d44-0004-11f1-9a1e-6f2c3b4d5e04")
	referenceID := uuid.FromStringOrNil("5b3a1d44-0005-11f1-9a1e-6f2c3b4d5e05")
	confbridgeID := uuid.FromStringOrNil("5b3a1d44-0006-11f1-9a1e-6f2c3b4d5e06")
	preFlowID := uuid.FromStringOrNil("5b3a1d44-0007-11f1-9a1e-6f2c3b4d5e07")
	actionID1 := uuid.FromStringOrNil("5b3a1d44-0008-11f1-9a1e-6f2c3b4d5e08")
	actionID2 := uuid.FromStringOrNil("5b3a1d44-0009-11f1-9a1e-6f2c3b4d5e09")

	tests := []struct {
		name string

		preFlowID    uuid.UUID
		actionIDs    []uuid.UUID
		expectAction []fmaction.Action
	}{
		{
			name:      "without pre flow only the confbridge join action is pushed",
			preFlowID: uuid.Nil,
			actionIDs: []uuid.UUID{actionID1},
			expectAction: []fmaction.Action{
				{
					ID:     actionID1,
					Type:   fmaction.TypeConfbridgeJoin,
					Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{ConfbridgeID: confbridgeID}),
				},
			},
		},
		{
			name:      "with pre flow the fetch flow action precedes the confbridge join action",
			preFlowID: preFlowID,
			actionIDs: []uuid.UUID{actionID1, actionID2},
			expectAction: []fmaction.Action{
				{
					ID:     actionID1,
					Type:   fmaction.TypeFetchFlow,
					Option: fmaction.ConvertOption(fmaction.OptionFetchFlow{FlowID: preFlowID}),
				},
				{
					ID:     actionID2,
					Type:   fmaction.TypeConfbridgeJoin,
					Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{ConfbridgeID: confbridgeID}),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()

			cf := &conference.Conference{
				Identity:     commonidentity.Identity{ID: confID, CustomerID: customerID},
				ConfbridgeID: confbridgeID,
				PreFlowID:    tt.preFlowID,
			}
			created := &conferencecall.Conferencecall{
				Identity:      commonidentity.Identity{ID: ccID, CustomerID: customerID},
				ConferenceID:  confID,
				ReferenceType: conferencecall.ReferenceTypeCall,
				ReferenceID:   referenceID,
				Status:        conferencecall.StatusJoining,
			}

			done := make(chan struct{})
			m.conf.EXPECT().Get(ctx, confID).Return(cf, nil)
			// the first uuid is for the conferencecall, the rest are for the actions in order.
			m.util.EXPECT().UUIDCreate().Return(ccID)
			m.db.EXPECT().ConferencecallCreate(ctx, gomock.Any()).Return(nil)
			m.db.EXPECT().ConferencecallGet(ctx, ccID).Return(created, nil)
			m.notify.EXPECT().PublishEvent(ctx, conferencecall.EventTypeConferencecallJoining, created)
			m.req.EXPECT().ConferenceV1ConferencecallHealthCheck(ctx, ccID, 0, defaultHealthCheckDelay).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, _ int, _ int) error {
					defer close(done)
					return nil
				})
			for _, id := range tt.actionIDs {
				m.util.EXPECT().UUIDCreate().Return(id)
			}

			res, err := h.ServiceStart(ctx, activeflowID, confID, conferencecall.ReferenceTypeCall, referenceID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			waitDone(t, done)

			if res.ID != ccID || res.Type != commonservice.TypeConferencecall {
				t.Errorf("Wrong service. id: %s, type: %s", res.ID, res.Type)
			}
			if len(res.PushActions) != len(tt.expectAction) {
				t.Fatalf("Wrong action count. expect: %d, got: %d", len(tt.expectAction), len(res.PushActions))
			}
			for i, expect := range tt.expectAction {
				got := res.PushActions[i]
				if got.ID != expect.ID || got.Type != expect.Type || !reflect.DeepEqual(got.Option, expect.Option) {
					t.Errorf("Wrong action at %d.\nexpect: %v\ngot: %v", i, expect, got)
				}
			}
		})
	}
}

func Test_ServiceStart_errors(t *testing.T) {
	confID := uuid.FromStringOrNil("5b3a1d44-0102-11f1-9a1e-6f2c3b4d5e02")
	ccID := uuid.FromStringOrNil("5b3a1d44-0101-11f1-9a1e-6f2c3b4d5e01")

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *testMocks)
	}{
		{
			name: "conference lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.conf.EXPECT().Get(ctx, confID).Return(nil, errTestFailure)
			},
		},
		{
			name: "conferencecall creation fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.conf.EXPECT().Get(ctx, confID).Return(&conference.Conference{}, nil)
				m.util.EXPECT().UUIDCreate().Return(ccID)
				m.db.EXPECT().ConferencecallCreate(ctx, gomock.Any()).Return(errTestFailure)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()
			tt.setup(ctx, m)

			res, err := h.ServiceStart(ctx, uuid.Nil, confID, conferencecall.ReferenceTypeCall, uuid.Nil)
			if !errors.Is(err, errTestFailure) {
				t.Errorf("Wrong error. expect: wrapped simulated failure, got: %v", err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_Create_errors(t *testing.T) {
	ccID := uuid.FromStringOrNil("5b3a1d44-0201-11f1-9a1e-6f2c3b4d5e01")

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *testMocks)
	}{
		{
			name: "db create fails: nothing is published and no health check is scheduled",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallCreate(ctx, gomock.Any()).Return(errTestFailure)
			},
		},
		{
			name: "db get of the created conferencecall fails: nothing is published",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().ConferencecallGet(ctx, ccID).Return(nil, errTestFailure)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()
			m.util.EXPECT().UUIDCreate().Return(ccID)
			tt.setup(ctx, m)

			res, err := h.Create(ctx, uuid.Nil, uuid.Nil, uuid.Nil, conferencecall.ReferenceTypeCall, uuid.Nil)
			if !errors.Is(err, errTestFailure) {
				t.Errorf("Wrong error. expect: simulated failure, got: %v", err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_List_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newTestHandler(mc)
	ctx := context.Background()
	m.db.EXPECT().ConferencecallList(ctx, uint64(5), "tok", gomock.Any()).Return(nil, errTestFailure)

	res, err := h.List(ctx, 5, "tok", nil)
	if !errors.Is(err, errTestFailure) || res != nil {
		t.Errorf("Wrong match. expect: nil and simulated failure, got: %v, %v", res, err)
	}
}

func Test_Get_errors(t *testing.T) {
	id := uuid.FromStringOrNil("5b3a1d44-0301-11f1-9a1e-6f2c3b4d5e01")

	t.Run("not found is returned as a typed not found error that keeps the cause", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()
		m.db.EXPECT().ConferencecallGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

		_, err := h.Get(ctx, id)
		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) {
			t.Fatalf("Wrong error. expect: typed error, got: %v", err)
		}
		if ve.Status != cerrors.StatusNotFound {
			t.Errorf("Wrong status. expect: %v, got: %v", cerrors.StatusNotFound, ve.Status)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("The cause must stay reachable through errors.Is")
		}
	})

	t.Run("other errors are returned unchanged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()
		m.db.EXPECT().ConferencecallGet(ctx, id).Return(nil, errTestFailure)

		_, err := h.Get(ctx, id)
		var ve *cerrors.VoipbinError
		if errors.As(err, &ve) {
			t.Errorf("Wrong error. a generic failure must not become a typed error: %v", err)
		}
		if !errors.Is(err, errTestFailure) {
			t.Errorf("Wrong error. expect: simulated failure, got: %v", err)
		}
	})
}

func Test_GetByReferenceID_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newTestHandler(mc)
	ctx := context.Background()
	refID := uuid.FromStringOrNil("5b3a1d44-0401-11f1-9a1e-6f2c3b4d5e01")
	m.db.EXPECT().ConferencecallGetByReferenceID(ctx, refID).Return(nil, errTestFailure)

	res, err := h.GetByReferenceID(ctx, refID)
	if !errors.Is(err, errTestFailure) || res != nil {
		t.Errorf("Wrong match. expect: nil and wrapped simulated failure, got: %v, %v", res, err)
	}
}

func Test_updateStatus_errors(t *testing.T) {
	id := uuid.FromStringOrNil("5b3a1d44-0501-11f1-9a1e-6f2c3b4d5e01")

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *testMocks)
	}{
		{
			name: "db update fails: the conferencecall is not read back",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallUpdate(ctx, id, gomock.Any()).Return(errTestFailure)
			},
		},
		{
			name: "read back fails after a successful update",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallUpdate(ctx, id, gomock.Any()).Return(nil)
				m.db.EXPECT().ConferencecallGet(ctx, id).Return(nil, errTestFailure)
			},
		},
	}

	targets := []struct {
		name string
		call func(h *conferencecallHandler, ctx context.Context) (*conferencecall.Conferencecall, error)
	}{
		{"updateStatus", func(h *conferencecallHandler, ctx context.Context) (*conferencecall.Conferencecall, error) {
			return h.updateStatus(ctx, id, conferencecall.StatusJoined)
		}},
		{"updateStatusJoined", func(h *conferencecallHandler, ctx context.Context) (*conferencecall.Conferencecall, error) {
			return h.updateStatusJoined(ctx, id)
		}},
		{"updateStatusLeaving", func(h *conferencecallHandler, ctx context.Context) (*conferencecall.Conferencecall, error) {
			return h.updateStatusLeaving(ctx, id)
		}},
		{"updateStatusLeaved", func(h *conferencecallHandler, ctx context.Context) (*conferencecall.Conferencecall, error) {
			return h.updateStatusLeaved(ctx, id)
		}},
	}

	for _, target := range targets {
		for _, tt := range tests {
			t.Run(target.name+": "+tt.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				h, m := newTestHandler(mc)
				ctx := context.Background()
				tt.setup(ctx, m)
				// no event must be published when the status update failed.

				res, err := target.call(h, ctx)
				if !errors.Is(err, errTestFailure) || res != nil {
					t.Errorf("Wrong match. expect: nil and simulated failure, got: %v, %v", res, err)
				}
			})
		}
	}
}

func Test_Terminate_errors(t *testing.T) {
	ccID := uuid.FromStringOrNil("5b3a1d44-0601-11f1-9a1e-6f2c3b4d5e01")
	confID := uuid.FromStringOrNil("5b3a1d44-0602-11f1-9a1e-6f2c3b4d5e02")
	confbridgeID := uuid.FromStringOrNil("5b3a1d44-0603-11f1-9a1e-6f2c3b4d5e03")
	refID := uuid.FromStringOrNil("5b3a1d44-0604-11f1-9a1e-6f2c3b4d5e04")

	joined := &conferencecall.Conferencecall{
		Identity:     commonidentity.Identity{ID: ccID},
		ConferenceID: confID,
		ReferenceID:  refID,
		Status:       conferencecall.StatusJoined,
	}
	cf := &conference.Conference{
		Identity:     commonidentity.Identity{ID: confID},
		ConfbridgeID: confbridgeID,
	}

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *testMocks)
	}{
		{
			name: "conferencecall lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, ccID).Return(nil, errTestFailure)
			},
		},
		{
			name: "conference lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, ccID).Return(joined, nil)
				m.conf.EXPECT().Get(ctx, confID).Return(nil, errTestFailure)
			},
		},
		{
			name: "status update to leaving fails: the call is not kicked",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, ccID).Return(joined, nil)
				m.conf.EXPECT().Get(ctx, confID).Return(cf, nil)
				m.db.EXPECT().ConferencecallUpdate(ctx, ccID, gomock.Any()).Return(errTestFailure)
			},
		},
		{
			name: "kick request fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, ccID).Return(joined, nil).Times(2)
				m.conf.EXPECT().Get(ctx, confID).Return(cf, nil)
				m.db.EXPECT().ConferencecallUpdate(ctx, ccID, gomock.Any()).Return(nil)
				m.notify.EXPECT().PublishEvent(ctx, conferencecall.EventTypeConferencecallLeaving, gomock.Any())
				m.req.EXPECT().CallV1ConfbridgeCallKick(ctx, confbridgeID, refID).Return(errTestFailure)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()
			tt.setup(ctx, m)

			res, err := h.Terminate(ctx, ccID)
			if !errors.Is(err, errTestFailure) || res != nil {
				t.Errorf("Wrong match. expect: nil and simulated failure, got: %v, %v", res, err)
			}
		})
	}
}

func Test_Terminated_errors(t *testing.T) {
	ccID := uuid.FromStringOrNil("5b3a1d44-0701-11f1-9a1e-6f2c3b4d5e01")
	confID := uuid.FromStringOrNil("5b3a1d44-0702-11f1-9a1e-6f2c3b4d5e02")
	cc := &conferencecall.Conferencecall{
		Identity:     commonidentity.Identity{ID: ccID},
		ConferenceID: confID,
		Status:       conferencecall.StatusLeaving,
	}

	t.Run("status update to leaved fails: the conferencecall id stays on the conference", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()
		m.db.EXPECT().ConferencecallUpdate(ctx, ccID, gomock.Any()).Return(errTestFailure)

		res, err := h.Terminated(ctx, cc)
		if !errors.Is(err, errTestFailure) || res != nil {
			t.Errorf("Wrong match. expect: nil and simulated failure, got: %v, %v", res, err)
		}
	})

	t.Run("removing the conferencecall id from the conference fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()
		leaved := &conferencecall.Conferencecall{Identity: commonidentity.Identity{ID: ccID}, ConferenceID: confID, Status: conferencecall.StatusLeaved}
		m.db.EXPECT().ConferencecallUpdate(ctx, ccID, gomock.Any()).Return(nil)
		m.db.EXPECT().ConferencecallGet(ctx, ccID).Return(leaved, nil)
		m.notify.EXPECT().PublishEvent(ctx, conferencecall.EventTypeConferencecallLeaved, leaved)
		m.conf.EXPECT().RemoveConferencecallID(ctx, confID, ccID).Return(nil, errTestFailure)

		res, err := h.Terminated(ctx, cc)
		if !errors.Is(err, errTestFailure) || res != nil {
			t.Errorf("Wrong match. expect: nil and simulated failure, got: %v, %v", res, err)
		}
	})
}

func Test_HealthCheck_branches(t *testing.T) {
	id := uuid.FromStringOrNil("5b3a1d44-0801-11f1-9a1e-6f2c3b4d5e01")
	confID := uuid.FromStringOrNil("5b3a1d44-0802-11f1-9a1e-6f2c3b4d5e02")
	refID := uuid.FromStringOrNil("5b3a1d44-0803-11f1-9a1e-6f2c3b4d5e03")

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	expired := now.Add(-maxConferencecallDuration - time.Minute)
	fresh := now.Add(-time.Minute)
	timeout := now.Add(-maxConferencecallDuration)

	newCC := func(status conferencecall.Status, created time.Time) *conferencecall.Conferencecall {
		return &conferencecall.Conferencecall{
			Identity:     commonidentity.Identity{ID: id},
			ConferenceID: confID,
			ReferenceID:  refID,
			Status:       status,
			TMCreate:     &created,
		}
	}

	const noRetry = -1

	tests := []struct {
		name       string
		retryCount int
		setup      func(ctx context.Context, m *testMocks)

		// expectRetry is the retry count passed to the next health check request.
		// noRetry means that no next request must be scheduled.
		expectRetry int
	}{
		{
			name:       "retry count exceeded and the terminate fails: no next health check",
			retryCount: defaultHealthCheckRetryMax + 1,
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, id).Return(nil, errTestFailure)
			},
			expectRetry: noRetry,
		},
		{
			name:       "conferencecall lookup fails: retry with an increased count",
			retryCount: 1,
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, id).Return(nil, errTestFailure)
			},
			expectRetry: 2,
		},
		{
			name:       "conferencecall exceeded the max duration: retry with an increased count",
			retryCount: 0,
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, id).Return(newCC(conferencecall.StatusJoined, expired), nil)
				m.util.EXPECT().TimeNowAdd(-maxConferencecallDuration).Return(&timeout)
			},
			expectRetry: 1,
		},
		{
			name:       "conferencecall already leaved: no next health check",
			retryCount: 0,
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, id).Return(newCC(conferencecall.StatusLeaved, fresh), nil)
				m.util.EXPECT().TimeNowAdd(-maxConferencecallDuration).Return(&timeout)
			},
			expectRetry: noRetry,
		},
		{
			name:       "call lookup fails: retry with an increased count",
			retryCount: 0,
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, id).Return(newCC(conferencecall.StatusJoined, fresh), nil)
				m.util.EXPECT().TimeNowAdd(-maxConferencecallDuration).Return(&timeout)
				m.req.EXPECT().CallV1CallGet(ctx, refID).Return(nil, errTestFailure)
			},
			expectRetry: 1,
		},
		{
			name:       "call is not progressing: retry with an increased count",
			retryCount: 1,
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, id).Return(newCC(conferencecall.StatusJoined, fresh), nil)
				m.util.EXPECT().TimeNowAdd(-maxConferencecallDuration).Return(&timeout)
				m.req.EXPECT().CallV1CallGet(ctx, refID).Return(&cmcall.Call{Status: cmcall.StatusHangup}, nil)
			},
			expectRetry: 2,
		},
		{
			name:       "conference lookup fails: retry with an increased count",
			retryCount: 0,
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, id).Return(newCC(conferencecall.StatusJoined, fresh), nil)
				m.util.EXPECT().TimeNowAdd(-maxConferencecallDuration).Return(&timeout)
				m.req.EXPECT().CallV1CallGet(ctx, refID).Return(&cmcall.Call{Status: cmcall.StatusProgressing}, nil)
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, confID).Return(nil, errTestFailure)
			},
			expectRetry: 1,
		},
		{
			name:       "conference is not progressing: retry with an increased count",
			retryCount: 0,
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, id).Return(newCC(conferencecall.StatusJoined, fresh), nil)
				m.util.EXPECT().TimeNowAdd(-maxConferencecallDuration).Return(&timeout)
				m.req.EXPECT().CallV1CallGet(ctx, refID).Return(&cmcall.Call{Status: cmcall.StatusProgressing}, nil)
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, confID).Return(&conference.Conference{Status: conference.StatusTerminated}, nil)
			},
			expectRetry: 1,
		},
		{
			name:       "everything is healthy: the retry count is reset",
			retryCount: 2,
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferencecallGet(ctx, id).Return(newCC(conferencecall.StatusJoined, fresh), nil)
				m.util.EXPECT().TimeNowAdd(-maxConferencecallDuration).Return(&timeout)
				m.req.EXPECT().CallV1CallGet(ctx, refID).Return(&cmcall.Call{Status: cmcall.StatusProgressing}, nil)
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, confID).Return(&conference.Conference{Status: conference.StatusProgressing}, nil)
			},
			expectRetry: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()
			tt.setup(ctx, m)

			done := make(chan struct{})
			if tt.expectRetry != noRetry {
				m.req.EXPECT().ConferenceV1ConferencecallHealthCheck(ctx, id, tt.expectRetry, defaultHealthCheckDelay).DoAndReturn(
					func(_ context.Context, _ uuid.UUID, _ int, _ int) error {
						defer close(done)
						return nil
					})
			} else {
				close(done)
			}

			h.HealthCheck(ctx, id, tt.retryCount)
			waitDone(t, done)
		})
	}
}
