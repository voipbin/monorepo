package confbridgehandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

var (
	errCov       = errors.New("cov error")
	covID        = uuid.FromStringOrNil("c0b00000-0000-4000-8000-000000000001")
	covRecording = uuid.FromStringOrNil("c0b00000-0000-4000-8000-000000000002")
)

type covMocks struct {
	req       *requesthandler.MockRequestHandler
	db        *dbhandler.MockDBHandler
	notify    *notifyhandler.MockNotifyHandler
	recording *recordinghandler.MockRecordingHandler
}

func newCovHandler(t *testing.T) (*confbridgeHandler, *covMocks) {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &covMocks{
		req:       requesthandler.NewMockRequestHandler(mc),
		db:        dbhandler.NewMockDBHandler(mc),
		notify:    notifyhandler.NewMockNotifyHandler(mc),
		recording: recordinghandler.NewMockRecordingHandler(mc),
	}
	h := &confbridgeHandler{
		reqHandler:       m.req,
		db:               m.db,
		notifyHandler:    m.notify,
		recordingHandler: m.recording,
	}
	return h, m
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
				m.db.EXPECT().ConfbridgeGet(ctx, covID).Return(nil, tt.dbErr)
			} else {
				m.db.EXPECT().ConfbridgeGet(ctx, covID).Return(&confbridge.Confbridge{}, nil)
			}

			res, err := h.Get(ctx, covID)
			if (err != nil) != tt.expectErr {
				t.Fatalf("unexpected error state: %v", err)
			}
			if tt.expectErr && res != nil {
				t.Errorf("expect nil result")
			}
			if tt.dbErr != nil && !errors.Is(err, tt.dbErr) {
				t.Errorf("expect wrapped %v, got %v", tt.dbErr, err)
			}
			if tt.expectErr {
				var ve *cerrors.VoipbinError
				if got := errors.As(err, &ve); got != tt.expectNF {
					t.Errorf("Wrong typed error. expect: %v, got: %v", tt.expectNF, got)
				}
			}
		})
	}
}

func Test_GetByBridgeID_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.db.EXPECT().ConfbridgeGetByBridgeID(ctx, "bridge-1").Return(nil, errCov)
		if res, err := h.GetByBridgeID(ctx, "bridge-1"); err == nil || res != nil {
			t.Errorf("expect error, got %v, %v", res, err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		h, m := newCovHandler(t)
		cb := &confbridge.Confbridge{}
		m.db.EXPECT().ConfbridgeGetByBridgeID(ctx, "bridge-1").Return(cb, nil)
		if res, err := h.GetByBridgeID(ctx, "bridge-1"); err != nil || res != cb {
			t.Errorf("unexpected: %v, %v", res, err)
		}
	})
}

func Test_db_twoStep_cov(t *testing.T) {
	ctx := context.Background()
	other := uuid.FromStringOrNil("c0b00000-0000-4000-8000-000000000003")

	tests := []struct {
		name  string
		first func(m *dbhandler.MockDBHandler, err error)
		call  func(h *confbridgeHandler) error
	}{
		{
			name: "AddExternalMediaID",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ConfbridgeAddExternalMediaID(ctx, covID, other).Return(err)
			},
			call: func(h *confbridgeHandler) error {
				_, err := h.AddExternalMediaID(ctx, covID, other)
				return err
			},
		},
		{
			name: "RemoveExternalMediaID",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ConfbridgeRemoveExternalMediaID(ctx, covID, other).Return(err)
			},
			call: func(h *confbridgeHandler) error {
				_, err := h.RemoveExternalMediaID(ctx, covID, other)
				return err
			},
		},
		{
			name: "UpdateBridgeID",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ConfbridgeSetBridgeID(ctx, covID, "b").Return(err)
			},
			call: func(h *confbridgeHandler) error {
				_, err := h.UpdateBridgeID(ctx, covID, "b")
				return err
			},
		},
		{
			name: "RemoveChannelCallID",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ConfbridgeRemoveChannelCallID(ctx, covID, "ch").Return(err)
			},
			call: func(h *confbridgeHandler) error {
				_, err := h.RemoveChannelCallID(ctx, covID, "ch", other)
				return err
			},
		},
		{
			name: "AddChannelCallID",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ConfbridgeAddChannelCallID(ctx, covID, "ch", other).Return(err)
			},
			call: func(h *confbridgeHandler) error {
				_, err := h.AddChannelCallID(ctx, covID, "ch", other)
				return err
			},
		},
		{
			name:  "dbDelete",
			first: func(m *dbhandler.MockDBHandler, err error) { m.EXPECT().ConfbridgeDelete(ctx, covID).Return(err) },
			call: func(h *confbridgeHandler) error {
				_, err := h.dbDelete(ctx, covID)
				return err
			},
		},
		{
			name: "UpdateFlags",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ConfbridgeSetFlags(ctx, covID, gomock.Any()).Return(err)
			},
			call: func(h *confbridgeHandler) error {
				_, err := h.UpdateFlags(ctx, covID, []confbridge.Flag{})
				return err
			},
		},
		{
			name: "UpdateStatus",
			first: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ConfbridgeSetStatus(ctx, covID, confbridge.StatusTerminating).Return(err)
			},
			call: func(h *confbridgeHandler) error {
				_, err := h.UpdateStatus(ctx, covID, confbridge.StatusTerminating)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_first_fail_cov", func(t *testing.T) {
			h, m := newCovHandler(t)
			tt.first(m.db, errCov)
			if err := tt.call(h); err == nil {
				t.Errorf("expect error")
			}
		})

		t.Run(tt.name+"_get_fail_cov", func(t *testing.T) {
			h, m := newCovHandler(t)
			tt.first(m.db, nil)
			m.db.EXPECT().ConfbridgeGet(ctx, covID).Return(nil, errCov)
			if err := tt.call(h); err == nil {
				t.Errorf("expect error")
			}
		})
	}
}

func Test_UpdateRecordingID_errors_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("set error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.db.EXPECT().ConfbridgeSetRecordingID(ctx, covID, covRecording).Return(errCov)
		if _, err := h.UpdateRecordingID(ctx, covID, covRecording); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("add error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.db.EXPECT().ConfbridgeSetRecordingID(ctx, covID, covRecording).Return(nil)
		m.db.EXPECT().ConfbridgeAddRecordingIDs(ctx, covID, covRecording).Return(errCov)
		if _, err := h.UpdateRecordingID(ctx, covID, covRecording); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("get error with nil recording id", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.db.EXPECT().ConfbridgeSetRecordingID(ctx, covID, uuid.Nil).Return(nil)
		m.db.EXPECT().ConfbridgeGet(ctx, covID).Return(nil, errCov)
		if _, err := h.UpdateRecordingID(ctx, covID, uuid.Nil); err == nil {
			t.Errorf("expect error")
		}
	})
}

func Test_dbDelete_publishesEvent_cov(t *testing.T) {
	ctx := context.Background()
	h, m := newCovHandler(t)

	cb := &confbridge.Confbridge{}
	cb.ID = covID

	m.db.EXPECT().ConfbridgeDelete(ctx, covID).Return(nil)
	m.db.EXPECT().ConfbridgeGet(ctx, covID).Return(cb, nil)
	m.notify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeDeleted, cb).Times(1)

	res, err := h.dbDelete(ctx, covID)
	if err != nil || res != cb {
		t.Errorf("Wrong match. expect: (%v, nil), got: (%v, %v)", cb, res, err)
	}
}
