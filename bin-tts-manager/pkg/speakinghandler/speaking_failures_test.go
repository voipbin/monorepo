package speakinghandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/dbhandler"
	"monorepo/bin-tts-manager/pkg/streaminghandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var errTestDB = errors.New("test db failure")

type testMocks struct {
	db        *dbhandler.MockDBHandler
	streaming *streaminghandler.MockStreamingHandler
	notify    *notifyhandler.MockNotifyHandler
	handler   *speakingHandler
}

func newTestMocks(mc *gomock.Controller) *testMocks {
	m := &testMocks{
		db:        dbhandler.NewMockDBHandler(mc),
		streaming: streaminghandler.NewMockStreamingHandler(mc),
		notify:    notifyhandler.NewMockNotifyHandler(mc),
	}
	m.handler = &speakingHandler{
		db:               m.db,
		streamingHandler: m.streaming,
		notifyHandler:    m.notify,
		podID:            "test-pod",
	}
	return m
}

func Test_Create_lookupFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newTestMocks(mc)

	// No SpeakingCreate expectation: a failed lookup must stop before creating a record.
	m.db.EXPECT().SpeakingGets(gomock.Any(), "", uint64(100), gomock.Any()).Return(nil, errTestDB)

	res, err := m.handler.Create(context.Background(), uuid.Must(uuid.NewV4()), streaming.ReferenceTypeCall, uuid.Must(uuid.NewV4()), "en-US", "", "", streaming.DirectionIncoming)
	if res != nil || !errors.Is(err, errTestDB) {
		t.Errorf("Wrong result. res: %v, err: %v", res, err)
	}
	if err != nil && !strings.Contains(err.Error(), "could not check existing sessions") {
		t.Errorf("Wrong error message: %v", err)
	}
}

func Test_Create_concurrentRecheck(t *testing.T) {
	// The new session id is random (v4), so the competing ids are chosen to be
	// always lower (all zero) or always higher (all f) than it.
	lowerID := uuid.Nil
	higherID := uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff")

	tests := []struct {
		name string

		recheck    []*speaking.Speaking
		recheckErr error

		expectBackOff bool
	}{
		{
			name:          "active session with a lower id wins",
			recheck:       []*speaking.Speaking{{Identity: commonidentity.Identity{ID: lowerID}, Status: speaking.StatusActive}},
			expectBackOff: true,
		},
		{
			name:          "initiating session with a lower id wins",
			recheck:       []*speaking.Speaking{{Identity: commonidentity.Identity{ID: lowerID}, Status: speaking.StatusInitiating}},
			expectBackOff: true,
		},
		{
			name:    "active session with a higher id loses",
			recheck: []*speaking.Speaking{{Identity: commonidentity.Identity{ID: higherID}, Status: speaking.StatusActive}},
		},
		{
			name:    "stopped session with a lower id is ignored",
			recheck: []*speaking.Speaking{{Identity: commonidentity.Identity{ID: lowerID}, Status: speaking.StatusStopped}},
		},
		{
			name:       "recheck failure does not block the creation",
			recheckErr: errTestDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newTestMocks(mc)

			ctx := context.Background()
			customerID := uuid.Must(uuid.NewV4())
			referenceID := uuid.Must(uuid.NewV4())

			gomock.InOrder(
				m.db.EXPECT().SpeakingGets(gomock.Any(), "", uint64(100), gomock.Any()).Return([]*speaking.Speaking{}, nil),
				m.db.EXPECT().SpeakingCreate(gomock.Any(), gomock.Any()).Return(nil),
				m.db.EXPECT().SpeakingGets(gomock.Any(), "", uint64(100), gomock.Any()).Return(tt.recheck, tt.recheckErr),
			)

			if tt.expectBackOff {
				m.db.EXPECT().SpeakingUpdate(gomock.Any(), gomock.Any(), map[speaking.Field]any{speaking.FieldStatus: speaking.StatusStopped}).Return(nil)

				res, err := m.handler.Create(ctx, customerID, streaming.ReferenceTypeCall, referenceID, "en-US", "", "", streaming.DirectionIncoming)
				if res != nil || err == nil || !strings.Contains(err.Error(), "concurrent session creation detected") {
					t.Errorf("Wrong result. res: %v, err: %v", res, err)
				}
				return
			}

			stored := &speaking.Speaking{Status: speaking.StatusActive}
			m.streaming.EXPECT().StartWithID(gomock.Any(), gomock.Any(), customerID, streaming.ReferenceTypeCall, referenceID, "en-US", string(streaming.VendorNameElevenlabs), "", streaming.DirectionIncoming).Return(&streaming.Streaming{}, nil)
			m.db.EXPECT().SpeakingUpdate(gomock.Any(), gomock.Any(), map[speaking.Field]any{speaking.FieldStatus: speaking.StatusActive}).Return(nil)
			m.db.EXPECT().SpeakingGet(gomock.Any(), gomock.Any()).Return(stored, nil)
			m.notify.EXPECT().PublishEvent(gomock.Any(), speaking.EventTypeSpeakingStarted, stored)

			res, err := m.handler.Create(ctx, customerID, streaming.ReferenceTypeCall, referenceID, "en-US", "", "", streaming.DirectionIncoming)
			if err != nil || res != stored {
				t.Errorf("Wrong result. res: %v, err: %v", res, err)
			}
		})
	}
}

func Test_Create_afterStart(t *testing.T) {
	tests := []struct {
		name string

		updateErr error
		getErr    error

		expectPublish bool
	}{
		{
			name:          "status update failure is tolerated",
			updateErr:     errTestDB,
			expectPublish: true,
		},
		{
			name:   "final lookup failure returns the created record without an event",
			getErr: errTestDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newTestMocks(mc)

			stored := &speaking.Speaking{Status: speaking.StatusActive}
			m.db.EXPECT().SpeakingGets(gomock.Any(), "", uint64(100), gomock.Any()).Return([]*speaking.Speaking{}, nil).Times(2)
			m.db.EXPECT().SpeakingCreate(gomock.Any(), gomock.Any()).Return(nil)
			m.streaming.EXPECT().StartWithID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&streaming.Streaming{}, nil)
			m.db.EXPECT().SpeakingUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.updateErr)
			if tt.getErr != nil {
				m.db.EXPECT().SpeakingGet(gomock.Any(), gomock.Any()).Return(nil, tt.getErr)
			} else {
				m.db.EXPECT().SpeakingGet(gomock.Any(), gomock.Any()).Return(stored, nil)
			}
			if tt.expectPublish {
				m.notify.EXPECT().PublishEvent(gomock.Any(), speaking.EventTypeSpeakingStarted, stored)
			}

			res, err := m.handler.Create(context.Background(), uuid.Must(uuid.NewV4()), streaming.ReferenceTypeCall, uuid.Must(uuid.NewV4()), "en-US", "", "voice", streaming.DirectionIncoming)
			if err != nil || res == nil {
				t.Fatalf("Wrong result. res: %v, err: %v", res, err)
			}
			if tt.getErr != nil && (res.Status != speaking.StatusInitiating || res.VoiceID != "voice") {
				t.Errorf("Expected the locally built record, got: %v", res)
			}
			if tt.getErr == nil && res != stored {
				t.Errorf("Expected the stored record, got: %v", res)
			}
		})
	}
}

func Test_Get_errors(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	t.Run("not found becomes a typed error that keeps the cause", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := newTestMocks(mc)

		m.db.EXPECT().SpeakingGet(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)

		res, err := m.handler.Get(context.Background(), id)
		if res != nil {
			t.Errorf("expected no result, got: %v", res)
		}
		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) {
			t.Fatalf("expected a typed error, got: %v", err)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("expected the cause to be preserved, got: %v", err)
		}
	})

	t.Run("other errors are returned unchanged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := newTestMocks(mc)

		m.db.EXPECT().SpeakingGet(gomock.Any(), id).Return(nil, errTestDB)

		res, err := m.handler.Get(context.Background(), id)
		var ve *cerrors.VoipbinError
		if res != nil || err != errTestDB || errors.As(err, &ve) {
			t.Errorf("Wrong result. res: %v, err: %v", res, err)
		}
	})
}

func Test_Gets_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newTestMocks(mc)

	filters := map[speaking.Field]any{speaking.FieldDeleted: false}
	m.db.EXPECT().SpeakingGets(gomock.Any(), "token", uint64(10), filters).Return(nil, errTestDB)

	res, err := m.handler.Gets(context.Background(), "token", 10, filters)
	if res != nil || err != errTestDB {
		t.Errorf("Wrong result. res: %v, err: %v", res, err)
	}
}

func Test_Stop_failures(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	active := &speaking.Speaking{Identity: commonidentity.Identity{ID: id}, Status: speaking.StatusActive}

	t.Run("status update failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := newTestMocks(mc)

		m.db.EXPECT().SpeakingGet(gomock.Any(), id).Return(active, nil)
		m.streaming.EXPECT().Stop(gomock.Any(), id).Return(nil, nil)
		m.db.EXPECT().SpeakingUpdate(gomock.Any(), id, map[speaking.Field]any{speaking.FieldStatus: speaking.StatusStopped}).Return(errTestDB)

		res, err := m.handler.Stop(context.Background(), id)
		if res != nil || !errors.Is(err, errTestDB) {
			t.Errorf("Wrong result. res: %v, err: %v", res, err)
		}
	})

	t.Run("lookup failure after the update returns the earlier record without an event", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := newTestMocks(mc)

		gomock.InOrder(
			m.db.EXPECT().SpeakingGet(gomock.Any(), id).Return(active, nil),
			m.streaming.EXPECT().Stop(gomock.Any(), id).Return(nil, errTestDB),
			m.db.EXPECT().SpeakingUpdate(gomock.Any(), id, gomock.Any()).Return(nil),
			m.db.EXPECT().SpeakingGet(gomock.Any(), id).Return(nil, errTestDB),
		)

		res, err := m.handler.Stop(context.Background(), id)
		if err != nil || res != active {
			t.Errorf("Wrong result. res: %v, err: %v", res, err)
		}
	})
}

func Test_Delete_failures(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	found := &speaking.Speaking{Identity: commonidentity.Identity{ID: id}, Status: speaking.StatusStopped}

	t.Run("delete failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := newTestMocks(mc)

		m.db.EXPECT().SpeakingGet(gomock.Any(), id).Return(found, nil)
		m.db.EXPECT().SpeakingDelete(gomock.Any(), id).Return(errTestDB)

		res, err := m.handler.Delete(context.Background(), id)
		if res != nil || !errors.Is(err, errTestDB) {
			t.Errorf("Wrong result. res: %v, err: %v", res, err)
		}
	})

	t.Run("lookup failure after the delete returns the earlier record", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := newTestMocks(mc)

		gomock.InOrder(
			m.db.EXPECT().SpeakingGet(gomock.Any(), id).Return(found, nil),
			m.db.EXPECT().SpeakingDelete(gomock.Any(), id).Return(nil),
			m.db.EXPECT().SpeakingGet(gomock.Any(), id).Return(nil, errTestDB),
		)

		res, err := m.handler.Delete(context.Background(), id)
		if err != nil || res != found {
			t.Errorf("Wrong result. res: %v, err: %v", res, err)
		}
	})
}
