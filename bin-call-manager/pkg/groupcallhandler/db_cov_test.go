package groupcallhandler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

var (
	errCov = errors.New("cov error")

	covID       = uuid.FromStringOrNil("c0a00000-0000-4000-8000-000000000001")
	covCustomer = uuid.FromStringOrNil("c0a00000-0000-4000-8000-000000000002")
	covOther    = uuid.FromStringOrNil("c0a00000-0000-4000-8000-000000000003")
	covMasterGC = uuid.FromStringOrNil("c0a00000-0000-4000-8000-000000000004")
	covMasterCL = uuid.FromStringOrNil("c0a00000-0000-4000-8000-000000000005")
	covNewID    = uuid.FromStringOrNil("c0a00000-0000-4000-8000-000000000006")
)

type covMocks struct {
	util   *utilhandler.MockUtilHandler
	req    *requesthandler.MockRequestHandler
	db     *dbhandler.MockDBHandler
	notify *notifyhandler.MockNotifyHandler
}

func newCovHandler(t *testing.T) (*groupcallHandler, *covMocks) {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &covMocks{
		util:   utilhandler.NewMockUtilHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
	}
	h := &groupcallHandler{
		utilHandler:   m.util,
		reqHandler:    m.req,
		db:            m.db,
		notifyHandler: m.notify,
	}
	return h, m
}

// waitCov waits for the wait group with a timeout.
func waitCov(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for the goroutines")
	}
}

func Test_db_twoStep_cov(t *testing.T) {
	ctx := context.Background()
	gc := &groupcall.Groupcall{Identity: commonidentity.Identity{ID: covID, CustomerID: covCustomer}}

	tests := []struct {
		name string
		// first expects the first db call and returns the given error
		first func(m *dbhandler.MockDBHandler, err error)
		call  func(h *groupcallHandler) error
		// notify is true if the success path publishes a webhook event
		notify bool
	}{
		{
			name: "Create",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().GroupcallCreate(ctx, gomock.Any()).Return(err)
			},
			call: func(h *groupcallHandler) error {
				_, err := h.Create(ctx, covID, covCustomer, commonidentity.OwnerTypeNone, uuid.Nil, uuid.Nil, nil, nil, nil, nil, uuid.Nil, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "")
				return err
			},
			notify: true,
		},
		{
			name: "UpdateAnswerCallID",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().GroupcallSetAnswerCallID(ctx, covID, covOther).Return(err)
			},
			call: func(h *groupcallHandler) error {
				_, err := h.UpdateAnswerCallID(ctx, covID, covOther)
				return err
			},
			notify: true,
		},
		{
			name: "UpdateAnswerGroupcallID",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().GroupcallSetAnswerGroupcallID(ctx, covID, covOther).Return(err)
			},
			call: func(h *groupcallHandler) error {
				_, err := h.UpdateAnswerGroupcallID(ctx, covID, covOther)
				return err
			},
			notify: true,
		},
		{
			name: "dbDelete",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().GroupcallDelete(ctx, covID).Return(err)
			},
			call: func(h *groupcallHandler) error {
				_, err := h.dbDelete(ctx, covID)
				return err
			},
			notify: true,
		},
		{
			name: "DecreaseCallCount",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().GroupcallDecreaseCallCount(ctx, covID).Return(err)
			},
			call: func(h *groupcallHandler) error {
				_, err := h.DecreaseCallCount(ctx, covID)
				return err
			},
		},
		{
			name: "DecreaseGroupcallCount",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().GroupcallDecreaseGroupcallCount(ctx, covID).Return(err)
			},
			call: func(h *groupcallHandler) error {
				_, err := h.DecreaseGroupcallCount(ctx, covID)
				return err
			},
		},
		{
			name: "UpdateCallIDsAndCallCountAndDialIndex",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().GroupcallSetCallIDsAndCallCountAndDialIndex(ctx, covID, gomock.Any(), 1, 1).Return(err)
			},
			call: func(h *groupcallHandler) error {
				_, err := h.UpdateCallIDsAndCallCountAndDialIndex(ctx, covID, []uuid.UUID{covOther}, 1, 1)
				return err
			},
		},
		{
			name: "UpdateGroupcallIDsAndGroupcallCountAndDialIndex",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex(ctx, covID, gomock.Any(), 1, 1).Return(err)
			},
			call: func(h *groupcallHandler) error {
				_, err := h.UpdateGroupcallIDsAndGroupcallCountAndDialIndex(ctx, covID, []uuid.UUID{covOther}, 1, 1)
				return err
			},
		},
		{
			name: "UpdateStatus",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().GroupcallSetStatus(ctx, covID, groupcall.StatusHangup).Return(err)
			},
			call: func(h *groupcallHandler) error {
				_, err := h.UpdateStatus(ctx, covID, groupcall.StatusHangup)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_first_fail_cov", func(t *testing.T) {
			h, m := newCovHandler(t)
			tt.first(m.db, errCov)

			if err := tt.call(h); err == nil {
				t.Errorf("expect error, got nil")
			}
		})

		t.Run(tt.name+"_get_fail_cov", func(t *testing.T) {
			h, m := newCovHandler(t)
			tt.first(m.db, nil)
			m.db.EXPECT().GroupcallGet(ctx, gomock.Any()).Return(nil, errCov)

			if err := tt.call(h); err == nil {
				t.Errorf("expect error, got nil")
			}
		})

		t.Run(tt.name+"_ok_cov", func(t *testing.T) {
			h, m := newCovHandler(t)
			tt.first(m.db, nil)
			m.db.EXPECT().GroupcallGet(ctx, gomock.Any()).Return(gc, nil)
			if tt.notify {
				m.notify.EXPECT().PublishWebhookEvent(ctx, covCustomer, gomock.Any(), gc)
			}

			if err := tt.call(h); err != nil {
				t.Errorf("expect ok, got %v", err)
			}
		})
	}
}

func Test_Get_cov(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		dbErr     error
		expectErr bool
		expectNF  bool
	}{
		{"not found", dbhandler.ErrNotFound, true, true},
		{"other error", errCov, true, false},
		{"ok", nil, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			if tt.dbErr != nil {
				m.db.EXPECT().GroupcallGet(ctx, covID).Return(nil, tt.dbErr)
			} else {
				m.db.EXPECT().GroupcallGet(ctx, covID).Return(&groupcall.Groupcall{}, nil)
			}

			res, err := h.Get(ctx, covID)
			if (err != nil) != tt.expectErr {
				t.Errorf("unexpected error state: %v", err)
			}
			if tt.expectErr && res != nil {
				t.Errorf("expect nil result")
			}
			if tt.dbErr != nil && !errors.Is(err, tt.dbErr) {
				t.Errorf("expect wrapped %v, got %v", tt.dbErr, err)
			}
			if tt.expectErr {
				var ve *cerrors.VoipbinError
				got := errors.As(err, &ve)
				if got != tt.expectNF {
					t.Fatalf("Wrong typed error. expect: %v, got: %v", tt.expectNF, got)
				}
				if got && (ve.Status != cerrors.StatusNotFound || ve.Reason != "GROUPCALL_NOT_FOUND") {
					t.Errorf("Wrong typed error. expect: %v/GROUPCALL_NOT_FOUND, got: %v/%v", cerrors.StatusNotFound, ve.Status, ve.Reason)
				}
			}
		})
	}
}

func Test_List_cov(t *testing.T) {
	ctx := context.Background()
	filters := map[groupcall.Field]any{}

	t.Run("error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.db.EXPECT().GroupcallList(ctx, uint64(10), "t", filters).Return(nil, errCov)

		if _, err := h.List(ctx, 10, "t", filters); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("ok", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.db.EXPECT().GroupcallList(ctx, uint64(10), "t", filters).Return([]*groupcall.Groupcall{{}}, nil)

		res, err := h.List(ctx, 10, "t", filters)
		if err != nil || len(res) != 1 {
			t.Errorf("unexpected result: %v, %v", res, err)
		}
	})
}
