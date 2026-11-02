package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_ProcessV1ContactsIDCaseNotesGet covers GET /v1/contacts/{id}/case_notes
// (VOIP-1673 A1): query parsing, the unwrapped domain call, the wire result
// and the error mapping.
func Test_ProcessV1ContactsIDCaseNotesGet(t *testing.T) {
	customerID := uuid.FromStringOrNil("aaaaaaaa-1673-0001-0001-000000000001")
	contactID := uuid.FromStringOrNil("aaaaaaaa-1673-0001-0001-000000000002")
	caseID := uuid.FromStringOrNil("aaaaaaaa-1673-0001-0001-000000000003")
	noteID := uuid.FromStringOrNil("aaaaaaaa-1673-0001-0001-000000000004")
	body, _ := json.Marshal(map[string]any{"customer_id": customerID.String()})

	t.Run("ok with paging", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCase := newTestListenHandlerWithCase(mc)
		ctx := context.Background()

		mockCase.EXPECT().CaseNoteListByContact(ctx, customerID, contactID, uint64(5), "2026-06-01T09:30:01.000000Z").
			Return([]*casenote.CaseNote{{ID: noteID, CaseID: caseID}}, "", nil)

		res, err := h.processV1ContactsIDCaseNotesGet(ctx, &sock.Request{
			URI:    "/v1/contacts/" + contactID.String() + "/case_notes?page_size=5&page_token=2026-06-01T09:30:01.000000Z",
			Method: sock.RequestMethodGet, Data: body,
		})
		if err != nil || res.StatusCode != 200 {
			t.Fatalf("res=%v err=%v", res, err)
		}
		var got []*casenote.CaseNote
		if err := json.Unmarshal(res.Data, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(got) != 1 || got[0].ID != noteID || got[0].CaseID != caseID {
			t.Errorf("unexpected body: %s", res.Data)
		}
	})

	t.Run("ok without query", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCase := newTestListenHandlerWithCase(mc)
		ctx := context.Background()

		mockCase.EXPECT().CaseNoteListByContact(ctx, customerID, contactID, uint64(0), "").
			Return([]*casenote.CaseNote{}, "", nil)
		res, err := h.processV1ContactsIDCaseNotesGet(ctx, &sock.Request{
			URI: "/v1/contacts/" + contactID.String() + "/case_notes", Method: sock.RequestMethodGet, Data: body,
		})
		if err != nil || res.StatusCode != 200 || string(res.Data) != "[]" {
			t.Fatalf("res=%v err=%v", res, err)
		}
	})

	t.Run("missing customer_id", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _ := newTestListenHandlerWithCase(mc)
		res, err := h.processV1ContactsIDCaseNotesGet(context.Background(), &sock.Request{
			URI: "/v1/contacts/" + contactID.String() + "/case_notes", Method: sock.RequestMethodGet, Data: []byte(`{}`),
		})
		if err != nil || res.StatusCode != 400 {
			t.Fatalf("expected 400, res=%v err=%v", res, err)
		}
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCase := newTestListenHandlerWithCase(mc)
		ctx := context.Background()
		mockCase.EXPECT().CaseNoteListByContact(ctx, customerID, contactID, uint64(0), "").
			Return(nil, "", dbhandler.ErrNotFound)
		res, err := h.processV1ContactsIDCaseNotesGet(ctx, &sock.Request{
			URI: "/v1/contacts/" + contactID.String() + "/case_notes", Method: sock.RequestMethodGet, Data: body,
		})
		if err != nil || res.StatusCode != 404 {
			t.Fatalf("expected 404, res=%v err=%v", res, err)
		}
	})

	t.Run("internal error maps to 500", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCase := newTestListenHandlerWithCase(mc)
		ctx := context.Background()
		mockCase.EXPECT().CaseNoteListByContact(ctx, customerID, contactID, uint64(0), "").
			Return(nil, "", errors.New("boom"))
		res, err := h.processV1ContactsIDCaseNotesGet(ctx, &sock.Request{
			URI: "/v1/contacts/" + contactID.String() + "/case_notes", Method: sock.RequestMethodGet, Data: body,
		})
		if err != nil || res.StatusCode != 500 {
			t.Fatalf("expected 500, res=%v err=%v", res, err)
		}
	})
}
