package aihandler

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

var (
	g4AIID     = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000001")
	g4Customer = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000002")
	g4DirectID = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000003")
	g4Server1  = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000004")
	g4Server2  = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000005")
)

type g4Mocks struct {
	h      *aiHandler
	db     *dbhandler.MockDBHandler
	req    *requesthandler.MockRequestHandler
	notify *notifyhandler.MockNotifyHandler
	util   *utilhandler.MockUtilHandler
}

func g4New(mc *gomock.Controller) *g4Mocks {
	m := &g4Mocks{
		db:     dbhandler.NewMockDBHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
	}
	m.h = &aiHandler{
		utilHandler:   m.util,
		reqHandler:    m.req,
		notifyHandler: m.notify,
		db:            m.db,
	}
	return m
}

func g4AI() *ai.AI {
	return &ai.AI{Identity: identity.Identity{ID: g4AIID, CustomerID: g4Customer}}
}

func g4AsVoipbinError(t *testing.T, err error) *cerrors.VoipbinError {
	t.Helper()
	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected *cerrors.VoipbinError, got %T: %v", err, err)
	}
	return ve
}

// ---- db.go ----

func Test_Get_notFoundIsTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbhandler.ErrNotFound)

	res, err := m.h.Get(context.Background(), g4AIID)
	if res != nil {
		t.Errorf("expected nil result, got %v", res)
	}
	ve := g4AsVoipbinError(t, err)
	if ve.Reason != "AI_NOT_FOUND" || ve.Status != cerrors.StatusNotFound {
		t.Errorf("unexpected error: %+v", ve)
	}
	if !stderrors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("cause must stay reachable: %v", err)
	}
}

func Test_Get_otherErrorIsWrappedNotTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	dbErr := stderrors.New("db down")
	m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbErr)

	res, err := m.h.Get(context.Background(), g4AIID)
	var ve *cerrors.VoipbinError
	if res != nil || !stderrors.Is(err, dbErr) || stderrors.As(err, &ve) || !strings.Contains(err.Error(), "could not get ai") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_List_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	dbErr := stderrors.New("db down")
	m.db.EXPECT().AIList(gomock.Any(), uint64(3), "tok", gomock.Any()).Return(nil, dbErr)

	res, err := m.h.List(context.Background(), 3, "tok", nil)
	if res != nil || !stderrors.Is(err, dbErr) {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Delete_errors(t *testing.T) {
	dbErr := stderrors.New("db down")

	t.Run("lookup fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbErr)

		res, err := m.h.Delete(context.Background(), g4AIID)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "for delete") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("direct deletion failure does not block the delete", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		a := g4AI()
		a.DirectID = g4DirectID
		deleted := g4AI()
		gomock.InOrder(
			m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(a, nil),
			m.req.EXPECT().DirectV1DirectDelete(gomock.Any(), g4DirectID).Return(nil, dbErr),
			m.db.EXPECT().AIDelete(gomock.Any(), g4AIID).Return(nil),
			m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(deleted, nil),
			m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), g4Customer, ai.EventTypeDeleted, deleted),
		)

		res, err := m.h.Delete(context.Background(), g4AIID)
		if err != nil || res != deleted {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("no direct id skips the direct deletion", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		// no DirectV1DirectDelete expectation: it must not be called.
		m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(g4AI(), nil)
		m.db.EXPECT().AIDelete(gomock.Any(), g4AIID).Return(dbErr)

		res, err := m.h.Delete(context.Background(), g4AIID)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not delete ai") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("reload after delete fails and nothing is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		gomock.InOrder(
			m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(g4AI(), nil),
			m.db.EXPECT().AIDelete(gomock.Any(), g4AIID).Return(nil),
			m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbErr),
		)

		res, err := m.h.Delete(context.Background(), g4AIID)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not get deleted ai") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})
}

func g4Create(h *aiHandler) (*ai.AI, error) {
	return h.dbCreate(context.Background(), g4Customer, "name", "detail", ai.TypeNormal, ai.EngineModelOpenaiGPT5Dot1,
		nil, "key", "", nil, uuid.Nil, "prompt", ai.TTSTypeCartesia, "voice", ai.STTTypeDeepgram, "en-US",
		nil, nil, false, false, uuid.Nil)
}

func Test_dbCreate_errors(t *testing.T) {
	dbErr := stderrors.New("db down")
	direct := &dmdirect.Direct{Identity: identity.Identity{ID: g4DirectID}, Hash: "hash-value"}

	t.Run("direct creation fails and no row is written", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.util.EXPECT().UUIDCreate().Return(g4AIID)
		m.req.EXPECT().DirectV1DirectCreate(gomock.Any(), g4Customer, dmdirect.ResourceTypeAI, g4AIID).Return(nil, dbErr)

		res, err := g4Create(m.h)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not create direct hash") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("insert fails and the orphaned direct is cleaned up", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.util.EXPECT().UUIDCreate().Return(g4AIID)
		m.req.EXPECT().DirectV1DirectCreate(gomock.Any(), g4Customer, dmdirect.ResourceTypeAI, g4AIID).Return(direct, nil)
		m.db.EXPECT().AICreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, a *ai.AI) error {
			if a.DirectID != g4DirectID || a.DirectHash != "hash-value" || a.ID != g4AIID {
				t.Errorf("direct info missing from the row: %+v", a)
			}
			return dbErr
		})
		m.req.EXPECT().DirectV1DirectDelete(gomock.Any(), g4DirectID).Return(nil, nil)

		res, err := g4Create(m.h)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not create ai") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("cleanup failure does not hide the insert error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.util.EXPECT().UUIDCreate().Return(g4AIID)
		m.req.EXPECT().DirectV1DirectCreate(gomock.Any(), g4Customer, dmdirect.ResourceTypeAI, g4AIID).Return(direct, nil)
		m.db.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(dbErr)
		m.req.EXPECT().DirectV1DirectDelete(gomock.Any(), g4DirectID).Return(nil, stderrors.New("cleanup failed"))

		res, err := g4Create(m.h)
		if res != nil || !stderrors.Is(err, dbErr) {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("reload fails and nothing is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.util.EXPECT().UUIDCreate().Return(g4AIID)
		m.req.EXPECT().DirectV1DirectCreate(gomock.Any(), g4Customer, dmdirect.ResourceTypeAI, g4AIID).Return(direct, nil)
		m.db.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(nil)
		m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbErr)

		res, err := g4Create(m.h)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not get created ai") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})
}

func Test_dbUpdate_errors(t *testing.T) {
	dbErr := stderrors.New("db down")
	call := func(h *aiHandler) (*ai.AI, error) {
		return h.dbUpdate(context.Background(), g4AIID, "name", "detail", ai.TypeNormal, ai.EngineModelOpenaiGPT5Dot1,
			nil, nil, uuid.Nil, "prompt", ai.TTSTypeCartesia, "voice", ai.STTTypeDeepgram, "en-US", nil, nil, false, false, endpointWrite{})
	}

	t.Run("update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().AIUpdate(gomock.Any(), g4AIID, gomock.Any()).Return(dbErr)

		res, err := call(m.h)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not update ai") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("reload fails and nothing is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().AIUpdate(gomock.Any(), g4AIID, gomock.Any()).Return(nil)
		m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbErr)

		res, err := call(m.h)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not get updated ai") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})
}

func Test_ActivateInsight_otherErrorIsWrapped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	dbErr := stderrors.New("db down")
	m.db.EXPECT().AIActivateInsight(gomock.Any(), g4AIID).Return(nil, nil, dbErr)

	res, err := m.h.ActivateInsight(context.Background(), g4AIID)
	var ve *cerrors.VoipbinError
	if res != nil || !stderrors.Is(err, dbErr) || stderrors.As(err, &ve) || !strings.Contains(err.Error(), "could not activate insight ai") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

// ---- direct_hash.go ----

func Test_DirectHashRegenerate_errors(t *testing.T) {
	dbErr := stderrors.New("backend failed")
	newDirect := &dmdirect.Direct{Identity: identity.Identity{ID: g4DirectID}, Hash: "new-hash"}

	t.Run("ai lookup fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbErr)

		res, err := m.h.DirectHashRegenerate(context.Background(), g4AIID)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not get ai") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("regenerate fails for an AI that has a direct", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		a := g4AI()
		a.DirectID = g4DirectID
		m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(a, nil)
		m.req.EXPECT().DirectV1DirectRegenerate(gomock.Any(), g4DirectID).Return(nil, dbErr)

		res, err := m.h.DirectHashRegenerate(context.Background(), g4AIID)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not regenerate direct hash") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("create fails for an AI without a direct", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(g4AI(), nil)
		m.req.EXPECT().DirectV1DirectCreate(gomock.Any(), g4Customer, dmdirect.ResourceTypeAI, g4AIID).Return(nil, dbErr)

		res, err := m.h.DirectHashRegenerate(context.Background(), g4AIID)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not create direct hash") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(g4AI(), nil)
		m.req.EXPECT().DirectV1DirectCreate(gomock.Any(), g4Customer, dmdirect.ResourceTypeAI, g4AIID).Return(newDirect, nil)
		m.db.EXPECT().AIUpdate(gomock.Any(), g4AIID, map[ai.Field]any{
			ai.FieldDirectID:   g4DirectID,
			ai.FieldDirectHash: "new-hash",
		}).Return(dbErr)

		res, err := m.h.DirectHashRegenerate(context.Background(), g4AIID)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not update ai") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("reload fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		gomock.InOrder(
			m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(g4AI(), nil),
			m.req.EXPECT().DirectV1DirectCreate(gomock.Any(), g4Customer, dmdirect.ResourceTypeAI, g4AIID).Return(newDirect, nil),
			m.db.EXPECT().AIUpdate(gomock.Any(), g4AIID, gomock.Any()).Return(nil),
			m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbErr),
		)

		res, err := m.h.DirectHashRegenerate(context.Background(), g4AIID)
		if res != nil || !stderrors.Is(err, dbErr) {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})
}

// ---- mcpserver_validation.go ----

func Test_UpdateMcpServerIDs_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	ids := []uuid.UUID{g4Server1, g4Server2}
	updated := g4AI()
	updated.McpServerIDs = ids
	gomock.InOrder(
		m.db.EXPECT().AIUpdate(gomock.Any(), g4AIID, map[ai.Field]any{ai.FieldMcpServerIDs: ids}).Return(nil),
		m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(updated, nil),
		m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), g4Customer, ai.EventTypeUpdated, updated),
	)

	res, err := m.h.UpdateMcpServerIDs(context.Background(), g4AIID, ids)
	if err != nil || res != updated {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_UpdateMcpServerIDs_errors(t *testing.T) {
	dbErr := stderrors.New("db down")

	t.Run("update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().AIUpdate(gomock.Any(), g4AIID, gomock.Any()).Return(dbErr)

		res, err := m.h.UpdateMcpServerIDs(context.Background(), g4AIID, nil)
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not update ai mcp_server_ids") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("reload fails and nothing is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().AIUpdate(gomock.Any(), g4AIID, gomock.Any()).Return(nil)
		m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbErr)

		res, err := m.h.UpdateMcpServerIDs(context.Background(), g4AIID, []uuid.UUID{g4Server1})
		if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not get updated ai") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})
}

// ---- chatbot.go validation helpers ----

func Test_endpointValidationError_mapsSentinels(t *testing.T) {
	tests := []struct {
		in         error
		wantReason string
	}{
		{ai.ErrInvalidEngineBaseURL, "INVALID_ENGINE_BASE_URL"},
		{ai.ErrEngineBaseURLRejected, "ENGINE_BASE_URL_REJECTED"},
		{ai.ErrEngineBaseURLRequired, "ENGINE_BASE_URL_REQUIRED"},
		{ai.ErrEngineBaseURLNotAllowed, "ENGINE_BASE_URL_NOT_ALLOWED"},
		{ai.ErrInvalidEngineBaseKey, "INVALID_ENGINE_BASE_KEY"},
		{ai.ErrEngineBaseKeyNotAllowed, "ENGINE_BASE_KEY_NOT_ALLOWED"},
	}
	for _, tt := range tests {
		t.Run(tt.wantReason, func(t *testing.T) {
			ve := g4AsVoipbinError(t, endpointValidationError(tt.in))
			if ve.Reason != tt.wantReason || ve.Status != cerrors.StatusInvalidArgument {
				t.Errorf("unexpected error: %+v", ve)
			}
		})
	}

	other := stderrors.New("something else")
	if got := endpointValidationError(other); got != other {
		t.Errorf("unknown errors must pass through unchanged, got %v", got)
	}
}

func Test_engineValidationError_passThroughAndKeyRequired(t *testing.T) {
	other := stderrors.New("something else")
	if got := engineValidationError(ai.EngineModelOpenaiGPT5Dot1, other); got != other {
		t.Errorf("unknown errors must pass through unchanged, got %v", got)
	}

	ve := g4AsVoipbinError(t, engineValidationError(ai.EngineModelOpenaiGPT5Dot1, ai.ErrEngineKeyRequired))
	if ve.Status != cerrors.StatusInvalidArgument {
		t.Errorf("unexpected error: %+v", ve)
	}
}

func Test_Create_endpointInputIsRejectedBeforeAnyBackendCall(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		baseKey    *string
		wantReason string
	}{
		{"url on a non-custom model", "https://api.example.com/v1", nil, "ENGINE_BASE_URL_NOT_ALLOWED"},
		{"key on a non-custom model", "", g4Ptr("synthetic-key"), "ENGINE_BASE_KEY_NOT_ALLOWED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc) // strict mocks: any backend call fails the test

			res, err := m.h.Create(context.Background(), g4Customer, "n", "d", ai.TypeNormal, ai.EngineModelOpenaiGPT5Dot1,
				nil, "key", tt.baseURL, tt.baseKey, uuid.Nil, "", ai.TTSTypeCartesia, "v", ai.STTTypeDeepgram, "en-US", nil, nil, false, false)
			if res != nil {
				t.Errorf("expected nil result, got %v", res)
			}
			if ve := g4AsVoipbinError(t, err); ve.Reason != tt.wantReason {
				t.Errorf("unexpected error: %+v", ve)
			}
		})
	}
}

func Test_Create_invalidTypeIsRejected(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	res, err := m.h.Create(context.Background(), g4Customer, "n", "d", ai.Type("bogus"), ai.EngineModelOpenaiGPT5Dot1,
		nil, "key", "", nil, uuid.Nil, "", ai.TTSTypeCartesia, "v", ai.STTTypeDeepgram, "en-US", nil, nil, false, false)
	if res != nil || err == nil || !strings.Contains(err.Error(), "invalid type: bogus") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Update_currentAILookupFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	dbErr := stderrors.New("db down")
	m.db.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbErr)

	res, err := m.h.Update(context.Background(), g4AIID, "n", "d", ai.TypeNormal, ai.EngineModelOpenaiGPT5Dot1,
		nil, nil, "", nil, uuid.Nil, "", ai.TTSTypeCartesia, "v", ai.STTTypeDeepgram, "en-US", nil, nil, false, false)
	if res != nil || !stderrors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not get current ai for update") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func g4Ptr(s string) *string { return &s }
