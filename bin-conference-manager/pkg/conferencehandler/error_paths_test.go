package conferencehandler

import (
	"context"
	"errors"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	dmdirect "monorepo/bin-direct-manager/models/direct"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

var errTestFailure = errors.New("simulated failure")

type testMocks struct {
	req    *requesthandler.MockRequestHandler
	db     *dbhandler.MockDBHandler
	notify *notifyhandler.MockNotifyHandler
	util   *utilhandler.MockUtilHandler
}

func newTestHandler(mc *gomock.Controller) (*conferenceHandler, *testMocks) {
	m := &testMocks{
		req:    requesthandler.NewMockRequestHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
	}

	return &conferenceHandler{
		reqHandler:    m.req,
		notifyHandler: m.notify,
		db:            m.db,
		utilHandler:   m.util,
	}, m
}

type errorCase struct {
	name  string
	setup func(ctx context.Context, m *testMocks)

	// expectCause is the error that must be reachable through errors.Is.
	// nil means that any non-nil error is accepted (the error is created
	// from a message and carries no cause).
	expectCause error
}

// runErrorCases runs the call and verifies that it fails with no result.
// The mocks have no expectation for the calls that must not happen.
func runErrorCases(t *testing.T, tests []errorCase, call func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error)) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()
			tt.setup(ctx, m)

			res, err := call(h, ctx)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
			if tt.expectCause != nil && !errors.Is(err, tt.expectCause) {
				t.Errorf("Wrong cause.\nexpect: %v\ngot: %v", tt.expectCause, err)
			}
		})
	}
}

func progressing(id uuid.UUID) *conference.Conference {
	return &conference.Conference{
		Identity: commonidentity.Identity{ID: id},
		Status:   conference.StatusProgressing,
	}
}

func Test_Create_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0001-11f1-8b2f-7a3d4c5e6f01")
	customerID := uuid.FromStringOrNil("6c4b2e55-0002-11f1-8b2f-7a3d4c5e6f02")
	confbridgeID := uuid.FromStringOrNil("6c4b2e55-0003-11f1-8b2f-7a3d4c5e6f03")
	directID := uuid.FromStringOrNil("6c4b2e55-0004-11f1-8b2f-7a3d4c5e6f04")

	validLimit := func(ctx context.Context, m *testMocks) {
		m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeConference).Return(true, nil)
	}
	confbridgeOK := func(ctx context.Context, m *testMocks) {
		m.req.EXPECT().CallV1ConfbridgeCreate(ctx, customerID, uuid.Nil, cmconfbridge.ReferenceTypeConference, id, cmconfbridge.TypeConference).Return(&cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: confbridgeID}}, nil)
	}
	directOK := func(ctx context.Context, m *testMocks) {
		m.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeConference, id).Return(&dmdirect.Direct{Identity: commonidentity.Identity{ID: directID}, Hash: "hash"}, nil)
	}

	tests := []errorCase{
		{
			name: "resource limit check fails: nothing is created",
			setup: func(ctx context.Context, m *testMocks) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeConference).Return(false, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "resource limit exceeded: nothing is created",
			setup: func(ctx context.Context, m *testMocks) {
				m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeConference).Return(false, nil)
			},
		},
		{
			name: "confbridge creation fails: no direct hash is created",
			setup: func(ctx context.Context, m *testMocks) {
				validLimit(ctx, m)
				m.req.EXPECT().CallV1ConfbridgeCreate(ctx, customerID, uuid.Nil, cmconfbridge.ReferenceTypeConference, id, cmconfbridge.TypeConference).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "direct hash creation fails: nothing is stored",
			setup: func(ctx context.Context, m *testMocks) {
				validLimit(ctx, m)
				confbridgeOK(ctx, m)
				m.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeConference, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "db create fails: the created direct hash is deleted",
			setup: func(ctx context.Context, m *testMocks) {
				validLimit(ctx, m)
				confbridgeOK(ctx, m)
				directOK(ctx, m)
				m.db.EXPECT().ConferenceCreate(ctx, gomock.Any()).Return(errTestFailure)
				m.req.EXPECT().DirectV1DirectDelete(ctx, directID).Return(&dmdirect.Direct{}, nil)
			},
			expectCause: errTestFailure,
		},
		{
			name: "db create fails and the direct hash cleanup fails too: the db error is returned",
			setup: func(ctx context.Context, m *testMocks) {
				validLimit(ctx, m)
				confbridgeOK(ctx, m)
				directOK(ctx, m)
				m.db.EXPECT().ConferenceCreate(ctx, gomock.Any()).Return(errTestFailure)
				m.req.EXPECT().DirectV1DirectDelete(ctx, directID).Return(nil, errors.New("cleanup failure"))
			},
			expectCause: errTestFailure,
		},
		{
			name: "reading back the created conference fails: no event is published",
			setup: func(ctx context.Context, m *testMocks) {
				validLimit(ctx, m)
				confbridgeOK(ctx, m)
				directOK(ctx, m)
				m.db.EXPECT().ConferenceCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}

	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.Create(ctx, id, customerID, conference.TypeConference, "name", "detail", nil, 0, uuid.Nil, uuid.Nil)
	})
}

func Test_Get_List_GetByConfbridgeID_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0101-11f1-8b2f-7a3d4c5e6f01")

	t.Run("get: not found becomes a typed not found error that keeps the cause", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()
		m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

		_, err := h.Get(ctx, id)
		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) || ve.Status != cerrors.StatusNotFound {
			t.Fatalf("Wrong error. expect: typed not found, got: %v", err)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("The cause must stay reachable through errors.Is")
		}
	})

	t.Run("get: other errors stay untyped and keep the cause", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()
		m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)

		_, err := h.Get(ctx, id)
		var ve *cerrors.VoipbinError
		if errors.As(err, &ve) {
			t.Errorf("Wrong error. a generic failure must not become a typed error: %v", err)
		}
		if !errors.Is(err, errTestFailure) {
			t.Errorf("Wrong cause. got: %v", err)
		}
	})

	t.Run("list", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()
		m.db.EXPECT().ConferenceList(ctx, uint64(3), "tok", gomock.Any()).Return(nil, errTestFailure)

		res, err := h.List(ctx, 3, "tok", nil)
		if !errors.Is(err, errTestFailure) || res != nil {
			t.Errorf("Wrong match. expect: nil and simulated failure, got: %v, %v", res, err)
		}
	})

	t.Run("get by confbridge id", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		ctx := context.Background()
		m.db.EXPECT().ConferenceGetByConfbridgeID(ctx, id).Return(nil, errTestFailure)

		res, err := h.GetByConfbridgeID(ctx, id)
		if !errors.Is(err, errTestFailure) || res != nil {
			t.Errorf("Wrong match. expect: nil and simulated failure, got: %v, %v", res, err)
		}
	})
}

func Test_Delete_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0201-11f1-8b2f-7a3d4c5e6f01")
	terminated := &conference.Conference{
		Identity: commonidentity.Identity{ID: id},
		Status:   conference.StatusTerminated,
	}

	tests := []errorCase{
		{
			name: "conference lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "terminating lookup fails: nothing is deleted",
			setup: func(ctx context.Context, m *testMocks) {
				gomock.InOrder(
					m.db.EXPECT().ConferenceGet(ctx, id).Return(terminated, nil),
					m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure),
				)
			},
			expectCause: errTestFailure,
		},
		{
			name: "db delete fails: no event is published",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(terminated, nil).Times(2)
				m.db.EXPECT().ConferenceDelete(ctx, id).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "reading back the deleted conference fails: no event is published",
			setup: func(ctx context.Context, m *testMocks) {
				gomock.InOrder(
					m.db.EXPECT().ConferenceGet(ctx, id).Return(terminated, nil).Times(2),
					m.db.EXPECT().ConferenceDelete(ctx, id).Return(nil),
					m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure),
				)
			},
			expectCause: errTestFailure,
		},
	}

	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.Delete(ctx, id)
	})
}

func Test_Delete_directCleanupFailureIsIgnored(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newTestHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("6c4b2e55-0301-11f1-8b2f-7a3d4c5e6f01")
	directID := uuid.FromStringOrNil("6c4b2e55-0302-11f1-8b2f-7a3d4c5e6f02")
	cf := &conference.Conference{
		Identity: commonidentity.Identity{ID: id},
		Status:   conference.StatusTerminated,
		DirectID: directID,
	}

	m.db.EXPECT().ConferenceGet(ctx, id).Return(cf, nil).Times(3)
	m.req.EXPECT().DirectV1DirectDelete(ctx, directID).Return(nil, errTestFailure)
	m.db.EXPECT().ConferenceDelete(ctx, id).Return(nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, cf.CustomerID, conference.EventTypeConferenceDeleted, cf)

	res, err := h.Delete(ctx, id)
	if err != nil || res != cf {
		t.Errorf("Wrong match. expect: the conference and no error, got: %v, %v", res, err)
	}
}

func Test_Update_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0401-11f1-8b2f-7a3d4c5e6f01")
	name := "new name"

	tests := []errorCase{
		{
			name: "no field is given and the lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}
	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.Update(ctx, id, nil, nil, nil, nil, nil, nil)
	})

	tests = []errorCase{
		{
			name: "db update fails: the conference is not read back",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldName: name}).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "read back fails: no event is published",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldName: name}).Return(nil)
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}
	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.Update(ctx, id, &name, nil, nil, nil, nil, nil)
	})
}

func Test_Update_timeoutBelowMinimumIsRaised(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newTestHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("6c4b2e55-0501-11f1-8b2f-7a3d4c5e6f01")
	short := 30 // below the 60 seconds minimum
	updated := &conference.Conference{Identity: commonidentity.Identity{ID: id}, Timeout: defaultConferenceTimeout}

	m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldTimeout: defaultConferenceTimeout}).Return(nil)
	m.db.EXPECT().ConferenceGet(ctx, id).Return(updated, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, updated.CustomerID, conference.EventTypeConferenceUpdated, updated)
	m.req.EXPECT().ConferenceV1ConferenceDeleteDelay(ctx, id, defaultConferenceTimeout*1000).Return(errTestFailure)

	res, err := h.Update(ctx, id, nil, nil, nil, &short, nil, nil)
	if err != nil || res != updated {
		t.Errorf("Wrong match. expect: the updated conference (a delete delay failure is ignored), got: %v, %v", res, err)
	}
}

func Test_Update_timeoutBoundaries(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0511-11f1-8b2f-7a3d4c5e6f01")

	tests := []struct {
		name         string
		timeout      int
		expectStored int
	}{
		{"zero keeps the zero value (no timeout)", 0, 0},
		{"one second is raised to the default", 1, defaultConferenceTimeout},
		{"59 seconds is raised to the default", 59, defaultConferenceTimeout},
		{"60 seconds is kept", 60, 60},
		{"above the minimum is kept", 3600, 3600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			ctx := context.Background()
			// the stored conference has no timeout so that no delayed delete is scheduled.
			updated := &conference.Conference{Identity: commonidentity.Identity{ID: id}}

			m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldTimeout: tt.expectStored}).Return(nil)
			m.db.EXPECT().ConferenceGet(ctx, id).Return(updated, nil)
			m.notify.EXPECT().PublishWebhookEvent(ctx, updated.CustomerID, conference.EventTypeConferenceUpdated, updated)

			timeout := tt.timeout
			if _, err := h.Update(ctx, id, nil, nil, nil, &timeout, nil, nil); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateRecordingID_UpdateTranscribeID_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0601-11f1-8b2f-7a3d4c5e6f01")
	otherID := uuid.FromStringOrNil("6c4b2e55-0602-11f1-8b2f-7a3d4c5e6f02")

	recording := []errorCase{
		{
			name: "recording: db update fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldRecordingID: otherID}).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "recording: adding the id to the history fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldRecordingID: otherID}).Return(nil)
				m.db.EXPECT().ConferenceAddRecordingIDs(ctx, id, otherID).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "recording: read back fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldRecordingID: otherID}).Return(nil)
				m.db.EXPECT().ConferenceAddRecordingIDs(ctx, id, otherID).Return(nil)
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}
	runErrorCases(t, recording, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.UpdateRecordingID(ctx, id, otherID)
	})

	transcribe := []errorCase{
		{
			name: "transcribe: db update fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldTranscribeID: otherID}).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "transcribe: adding the id to the history fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldTranscribeID: otherID}).Return(nil)
				m.db.EXPECT().ConferenceAddTranscribeIDs(ctx, id, otherID).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "transcribe: read back fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldTranscribeID: otherID}).Return(nil)
				m.db.EXPECT().ConferenceAddTranscribeIDs(ctx, id, otherID).Return(nil)
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}
	runErrorCases(t, transcribe, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.UpdateTranscribeID(ctx, id, otherID)
	})
}

func Test_UpdateRecordingID_nilIDSkipsHistory(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newTestHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("6c4b2e55-0701-11f1-8b2f-7a3d4c5e6f01")
	cf := progressing(id)

	// clearing the id must not add the nil uuid to the recording history.
	m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldRecordingID: uuid.Nil}).Return(nil)
	m.db.EXPECT().ConferenceGet(ctx, id).Return(cf, nil)

	res, err := h.UpdateRecordingID(ctx, id, uuid.Nil)
	if err != nil || res != cf {
		t.Errorf("Wrong match. expect: the conference, got: %v, %v", res, err)
	}
}

func Test_Terminating_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0801-11f1-8b2f-7a3d4c5e6f01")

	tests := []errorCase{
		{
			name: "conference lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "status update fails: no conferencecall is kicked",
			setup: func(ctx context.Context, m *testMocks) {
				cf := progressing(id)
				cf.ConferencecallIDs = []uuid.UUID{uuid.FromStringOrNil("6c4b2e55-0802-11f1-8b2f-7a3d4c5e6f02")}
				m.db.EXPECT().ConferenceGet(ctx, id).Return(cf, nil)
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldStatus: conference.StatusTerminating}).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}

	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.Terminating(ctx, id)
	})
}

func Test_Destroy_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0901-11f1-8b2f-7a3d4c5e6f01")
	confbridgeID := uuid.FromStringOrNil("6c4b2e55-0902-11f1-8b2f-7a3d4c5e6f02")
	cf := &conference.Conference{Identity: commonidentity.Identity{ID: id}, ConfbridgeID: confbridgeID, Type: conference.TypeConference}

	tests := []errorCase{
		{
			name: "marking the conference ended fails: the confbridge delete failure is not the reason",
			setup: func(ctx context.Context, m *testMocks) {
				m.req.EXPECT().CallV1ConfbridgeDelete(ctx, confbridgeID).Return(nil, errors.New("confbridge failure"))
				m.db.EXPECT().ConferenceEnd(ctx, id).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "reading back the ended conference fails: no event is published",
			setup: func(ctx context.Context, m *testMocks) {
				m.req.EXPECT().CallV1ConfbridgeDelete(ctx, confbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
				m.db.EXPECT().ConferenceEnd(ctx, id).Return(nil)
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}

	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.Destroy(ctx, cf)
	})
}

func Test_RecordingStart_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0a01-11f1-8b2f-7a3d4c5e6f01")
	confbridgeID := uuid.FromStringOrNil("6c4b2e55-0a02-11f1-8b2f-7a3d4c5e6f02")
	recordingID := uuid.FromStringOrNil("6c4b2e55-0a03-11f1-8b2f-7a3d4c5e6f03")
	cf := &conference.Conference{Identity: commonidentity.Identity{ID: id}, ConfbridgeID: confbridgeID, Status: conference.StatusProgressing}

	tests := []errorCase{
		{
			name: "conference lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "recording request fails: the recording id is not stored",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(cf, nil)
				m.req.EXPECT().CallV1RecordingStart(ctx, uuid.Nil, cmrecording.ReferenceTypeConfbridge, confbridgeID, cmrecording.FormatWAV, 0, "", 10, uuid.Nil).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "storing the recording id fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(cf, nil)
				m.req.EXPECT().CallV1RecordingStart(ctx, uuid.Nil, cmrecording.ReferenceTypeConfbridge, confbridgeID, cmrecording.FormatWAV, 0, "", 10, uuid.Nil).Return(&cmrecording.Recording{Identity: commonidentity.Identity{ID: recordingID}}, nil)
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldRecordingID: recordingID}).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}

	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.RecordingStart(ctx, id, uuid.Nil, cmrecording.FormatWAV, 10, uuid.Nil)
	})
}

func Test_RecordingStart_durationIsCapped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newTestHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("6c4b2e55-0b01-11f1-8b2f-7a3d4c5e6f01")
	confbridgeID := uuid.FromStringOrNil("6c4b2e55-0b02-11f1-8b2f-7a3d4c5e6f02")
	recordingID := uuid.FromStringOrNil("6c4b2e55-0b03-11f1-8b2f-7a3d4c5e6f03")
	cf := &conference.Conference{Identity: commonidentity.Identity{ID: id}, ConfbridgeID: confbridgeID, Status: conference.StatusProgressing}

	m.db.EXPECT().ConferenceGet(ctx, id).Return(cf, nil).Times(2)
	// a requested duration above the maximum is reduced to the maximum.
	m.req.EXPECT().CallV1RecordingStart(ctx, uuid.Nil, cmrecording.ReferenceTypeConfbridge, confbridgeID, cmrecording.FormatWAV, 0, "", defaultRecordingTimeout, uuid.Nil).Return(&cmrecording.Recording{Identity: commonidentity.Identity{ID: recordingID}}, nil)
	m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldRecordingID: recordingID}).Return(nil)
	m.db.EXPECT().ConferenceAddRecordingIDs(ctx, id, recordingID).Return(nil)

	if _, err := h.RecordingStart(ctx, id, uuid.Nil, cmrecording.FormatWAV, defaultRecordingTimeout+1000, uuid.Nil); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_RecordingStop_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0c01-11f1-8b2f-7a3d4c5e6f01")
	recordingID := uuid.FromStringOrNil("6c4b2e55-0c02-11f1-8b2f-7a3d4c5e6f02")
	withRecording := func(status conference.Status) *conference.Conference {
		return &conference.Conference{Identity: commonidentity.Identity{ID: id}, RecordingID: recordingID, Status: status}
	}

	tests := []errorCase{
		{
			name: "conference lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "no recording in progress: no stop request is sent",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(progressing(id), nil)
			},
		},
		{
			name: "conference is not progressing: no stop request is sent",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withRecording(conference.StatusTerminating), nil)
			},
		},
		{
			name: "stop request fails: the recording id is kept",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withRecording(conference.StatusProgressing), nil)
				m.req.EXPECT().CallV1RecordingStop(ctx, recordingID).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "clearing the recording id fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withRecording(conference.StatusProgressing), nil)
				m.req.EXPECT().CallV1RecordingStop(ctx, recordingID).Return(&cmrecording.Recording{}, nil)
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldRecordingID: uuid.Nil}).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}

	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.RecordingStop(ctx, id)
	})
}

func Test_TranscribeStart_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0d01-11f1-8b2f-7a3d4c5e6f01")
	confbridgeID := uuid.FromStringOrNil("6c4b2e55-0d02-11f1-8b2f-7a3d4c5e6f02")
	customerID := uuid.FromStringOrNil("6c4b2e55-0d03-11f1-8b2f-7a3d4c5e6f03")
	transcribeID := uuid.FromStringOrNil("6c4b2e55-0d04-11f1-8b2f-7a3d4c5e6f04")
	cf := &conference.Conference{
		Identity:     commonidentity.Identity{ID: id, CustomerID: customerID},
		ConfbridgeID: confbridgeID,
		Status:       conference.StatusProgressing,
	}
	expectStart := func(ctx context.Context, m *testMocks) *gomock.Call {
		return m.req.EXPECT().TranscribeV1TranscribeStart(ctx, uuid.Nil, customerID, uuid.Nil, uuid.Nil, tmtranscribe.ReferenceTypeConfbridge, confbridgeID, "en-US", tmtranscribe.DirectionIn, tmtranscribe.ProviderEmpty, 30000)
	}

	tests := []errorCase{
		{
			name: "conference lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "transcribe request fails: the transcribe id is not stored",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(cf, nil)
				expectStart(ctx, m).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "storing the transcribe id fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(cf, nil)
				expectStart(ctx, m).Return(&tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: transcribeID}}, nil)
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldTranscribeID: transcribeID}).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}

	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.TranscribeStart(ctx, id, "en-US")
	})
}

func Test_TranscribeStop_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0e01-11f1-8b2f-7a3d4c5e6f01")
	transcribeID := uuid.FromStringOrNil("6c4b2e55-0e02-11f1-8b2f-7a3d4c5e6f02")
	hostID := uuid.FromStringOrNil("6c4b2e55-0e03-11f1-8b2f-7a3d4c5e6f03")
	withTranscribe := func(status conference.Status, tid uuid.UUID) *conference.Conference {
		return &conference.Conference{Identity: commonidentity.Identity{ID: id}, TranscribeID: tid, Status: status}
	}
	tr := &tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: transcribeID}, HostID: hostID}

	tests := []errorCase{
		{
			name: "conference lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "conference is not progressing: no request is sent",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withTranscribe(conference.StatusTerminating, transcribeID), nil)
			},
		},
		{
			name: "no transcribe in progress: no request is sent",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withTranscribe(conference.StatusProgressing, uuid.Nil), nil)
			},
		},
		{
			name: "transcribe lookup fails: the stop request is not sent",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withTranscribe(conference.StatusProgressing, transcribeID), nil)
				m.req.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "stop request fails: the transcribe id is kept",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withTranscribe(conference.StatusProgressing, transcribeID), nil)
				m.req.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(tr, nil)
				m.req.EXPECT().TranscribeV1TranscribeStop(ctx, hostID, transcribeID).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "clearing the transcribe id fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withTranscribe(conference.StatusProgressing, transcribeID), nil)
				m.req.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(tr, nil)
				m.req.EXPECT().TranscribeV1TranscribeStop(ctx, hostID, transcribeID).Return(tr, nil)
				m.db.EXPECT().ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldTranscribeID: uuid.Nil}).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
	}

	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.TranscribeStop(ctx, id)
	})
}

func Test_DirectHashRegenerate_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c4b2e55-0f01-11f1-8b2f-7a3d4c5e6f01")
	customerID := uuid.FromStringOrNil("6c4b2e55-0f02-11f1-8b2f-7a3d4c5e6f02")
	oldDirectID := uuid.FromStringOrNil("6c4b2e55-0f03-11f1-8b2f-7a3d4c5e6f03")
	newDirectID := uuid.FromStringOrNil("6c4b2e55-0f04-11f1-8b2f-7a3d4c5e6f04")

	withDirect := &conference.Conference{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, DirectID: oldDirectID}
	withoutDirect := &conference.Conference{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}}
	newDirect := &dmdirect.Direct{Identity: commonidentity.Identity{ID: newDirectID}, Hash: "newhash"}
	fields := map[conference.Field]any{conference.FieldDirectID: newDirectID, conference.FieldDirectHash: "newhash"}

	tests := []errorCase{
		{
			name: "conference lookup fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "regenerating the existing direct hash fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withDirect, nil)
				m.req.EXPECT().DirectV1DirectRegenerate(ctx, oldDirectID).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "creating the missing direct hash fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withoutDirect, nil)
				m.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeConference, id).Return(nil, errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "storing the new direct hash fails",
			setup: func(ctx context.Context, m *testMocks) {
				m.db.EXPECT().ConferenceGet(ctx, id).Return(withDirect, nil)
				m.req.EXPECT().DirectV1DirectRegenerate(ctx, oldDirectID).Return(newDirect, nil)
				m.db.EXPECT().ConferenceUpdate(ctx, id, fields).Return(errTestFailure)
			},
			expectCause: errTestFailure,
		},
		{
			name: "reading back the updated conference fails",
			setup: func(ctx context.Context, m *testMocks) {
				gomock.InOrder(
					m.db.EXPECT().ConferenceGet(ctx, id).Return(withoutDirect, nil),
					m.req.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeConference, id).Return(newDirect, nil),
					m.db.EXPECT().ConferenceUpdate(ctx, id, fields).Return(nil),
					m.db.EXPECT().ConferenceGet(ctx, id).Return(nil, errTestFailure),
				)
			},
			expectCause: errTestFailure,
		},
	}

	runErrorCases(t, tests, func(h *conferenceHandler, ctx context.Context) (*conference.Conference, error) {
		return h.DirectHashRegenerate(ctx, id)
	})
}
