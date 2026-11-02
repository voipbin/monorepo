package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/contactstar"
)

var (
	starTestCustomerID = uuid.FromStringOrNil("aaaaaaaa-1673-0002-0001-000000000001")
	starTestAgentID    = uuid.FromStringOrNil("aaaaaaaa-1673-0002-0001-000000000002")
	starTestContactID  = uuid.FromStringOrNil("aaaaaaaa-1673-0002-0001-000000000003")
)

func starBody(t *testing.T, fields map[string]any) []byte {
	t.Helper()

	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("could not marshal the body. err: %v", err)
	}

	return data
}

func Test_ProcessV1ContactsIDStarsPost(t *testing.T) {
	uri := "/v1/contacts/" + starTestContactID.String() + "/stars/" + starTestAgentID.String()
	created := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	notFound := cerrors.NotFound(commonoutline.ServiceNameContactManager, "CONTACT_NOT_FOUND", "The contact was not found.")

	tests := []struct {
		name string
		data []byte

		expectCall bool
		callErr    error
		expectCode int
	}{
		{name: "ok", data: starBody(t, map[string]any{"customer_id": starTestCustomerID}), expectCall: true, expectCode: http.StatusOK},
		{name: "missing customer_id", data: starBody(t, map[string]any{}), expectCode: http.StatusBadRequest},
		{name: "no body", data: nil, expectCode: http.StatusBadRequest},
		{name: "contact not found", data: starBody(t, map[string]any{"customer_id": starTestCustomerID}), expectCall: true, callErr: notFound, expectCode: http.StatusNotFound},
		{name: "other error", data: starBody(t, map[string]any{"customer_id": starTestCustomerID}), expectCall: true, callErr: fmt.Errorf("database error"), expectCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockContact, _, _ := newTestListenHandlerAll(mc)
			ctx := context.Background()

			if tt.expectCall {
				var ret *contactstar.ContactStar
				if tt.callErr == nil {
					ret = &contactstar.ContactStar{
						Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("aaaaaaaa-1673-0002-0001-0000000000f1"), CustomerID: starTestCustomerID},
						AgentID:   starTestAgentID,
						ContactID: starTestContactID,
						TMCreate:  &created,
					}
				}
				mockContact.EXPECT().StarSet(ctx, starTestCustomerID, starTestAgentID, starTestContactID).Return(ret, tt.callErr)
			}

			res, err := h.processV1ContactsIDStarsPost(ctx, &sock.Request{URI: uri, Method: sock.RequestMethodPost, Data: tt.data})
			if err != nil || res.StatusCode != tt.expectCode {
				t.Fatalf("Wrong match. expect: %d, got: %v, %v", tt.expectCode, res, err)
			}
			if tt.expectCode == http.StatusOK {
				var got contactstar.ContactStar
				if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil || got.ContactID != starTestContactID || got.TMCreate == nil {
					t.Errorf("Wrong match. body: %s, err: %v", res.Data, errUnmarshal)
				}
			}
		})
	}
}

func Test_ProcessV1ContactsIDStarsPost_BadURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _, _, _ := newTestListenHandlerAll(mc)

	res, err := h.processV1ContactsIDStarsPost(context.Background(), &sock.Request{
		URI: "/v1/contacts/not-a-uuid/stars/also-not", Method: sock.RequestMethodPost,
		Data: starBody(t, map[string]any{"customer_id": starTestCustomerID}),
	})
	if err != nil || res.StatusCode != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: 400, got: %v, %v", res, err)
	}
}

func Test_ProcessV1ContactsIDStarsDelete(t *testing.T) {
	uri := "/v1/contacts/" + starTestContactID.String() + "/stars/" + starTestAgentID.String()
	notFound := cerrors.NotFound(commonoutline.ServiceNameContactManager, "CONTACT_NOT_FOUND", "The contact was not found.")

	tests := []struct {
		name string
		data []byte

		expectCall bool
		callErr    error
		expectCode int
	}{
		{name: "ok", data: starBody(t, map[string]any{"customer_id": starTestCustomerID}), expectCall: true, expectCode: http.StatusOK},
		{name: "missing customer_id", data: starBody(t, map[string]any{}), expectCode: http.StatusBadRequest},
		{name: "contact not found", data: starBody(t, map[string]any{"customer_id": starTestCustomerID}), expectCall: true, callErr: notFound, expectCode: http.StatusNotFound},
		{name: "other error", data: starBody(t, map[string]any{"customer_id": starTestCustomerID}), expectCall: true, callErr: fmt.Errorf("database error"), expectCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockContact, _, _ := newTestListenHandlerAll(mc)
			ctx := context.Background()

			if tt.expectCall {
				mockContact.EXPECT().StarDelete(ctx, starTestCustomerID, starTestAgentID, starTestContactID).Return(tt.callErr)
			}

			res, err := h.processV1ContactsIDStarsDelete(ctx, &sock.Request{URI: uri, Method: sock.RequestMethodDelete, Data: tt.data})
			if err != nil || res.StatusCode != tt.expectCode {
				t.Fatalf("Wrong match. expect: %d, got: %v, %v", tt.expectCode, res, err)
			}
		})
	}
}

func Test_ProcessV1ContactsStarredGet(t *testing.T) {
	body := starBody(t, map[string]any{"customer_id": starTestCustomerID, "agent_id": starTestAgentID})
	invalidToken := cerrors.InvalidArgument(commonoutline.ServiceNameContactManager, "INVALID_PAGE_TOKEN", "The page token is not valid.")

	tests := []struct {
		name string
		uri  string
		data []byte

		expectCall bool
		expectSize uint64
		expectTok  string
		callErr    error
		expectCode int
	}{
		{name: "ok with paging", uri: "/v1/contacts/starred?page_size=5&page_token=2026-06-01T09:30:01.000000Z", data: body, expectCall: true, expectSize: 5, expectTok: "2026-06-01T09:30:01.000000Z", expectCode: 200},
		{name: "ok without query", uri: "/v1/contacts/starred", data: body, expectCall: true, expectCode: 200},
		{name: "missing customer_id", uri: "/v1/contacts/starred?page_size=5", data: starBody(t, map[string]any{"agent_id": starTestAgentID}), expectCode: 400},
		{name: "missing agent_id", uri: "/v1/contacts/starred?page_size=5", data: starBody(t, map[string]any{"customer_id": starTestCustomerID}), expectCode: 400},
		{name: "invalid token", uri: "/v1/contacts/starred?page_token=bad", data: body, expectCall: true, expectTok: "bad", callErr: invalidToken, expectCode: 400},
		{name: "other error", uri: "/v1/contacts/starred", data: body, expectCall: true, callErr: fmt.Errorf("database error"), expectCode: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockContact, _, _ := newTestListenHandlerAll(mc)
			ctx := context.Background()

			if tt.expectCall {
				ret := []*contact.Contact{}
				if tt.callErr == nil {
					ret = []*contact.Contact{{Identity: commonidentity.Identity{ID: starTestContactID, CustomerID: starTestCustomerID}, FirstName: "Pat"}}
				}
				var retArg []*contact.Contact
				if tt.callErr == nil {
					retArg = ret
				}
				mockContact.EXPECT().ListStarred(ctx, starTestCustomerID, starTestAgentID, tt.expectSize, tt.expectTok).Return(retArg, tt.callErr)
			}

			res, err := h.processV1ContactsStarredGet(ctx, &sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: tt.data})
			if err != nil || res.StatusCode != tt.expectCode {
				t.Fatalf("Wrong match. expect: %d, got: %v, %v", tt.expectCode, res, err)
			}
			if tt.expectCode == 200 && tt.name == "ok with paging" {
				var got []*contact.Contact
				if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil || len(got) != 1 || got[0].ID != starTestContactID {
					t.Errorf("Wrong match. body: %s, err: %v", res.Data, errUnmarshal)
				}
			}
		})
	}
}

func Test_ProcessV1ContactStarsGet(t *testing.T) {
	body := starBody(t, map[string]any{"customer_id": starTestCustomerID, "agent_id": starTestAgentID})
	created := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		uri  string
		data []byte

		expectCall bool
		expectSize uint64
		expectTok  string
		callErr    error
		expectCode int
	}{
		{name: "ok with paging", uri: "/v1/contact_stars?page_size=3&page_token=2026-06-02T10:00:00.000000Z", data: body, expectCall: true, expectSize: 3, expectTok: "2026-06-02T10:00:00.000000Z", expectCode: 200},
		{name: "ok without query", uri: "/v1/contact_stars", data: body, expectCall: true, expectCode: 200},
		{name: "missing agent_id", uri: "/v1/contact_stars", data: starBody(t, map[string]any{"customer_id": starTestCustomerID}), expectCode: 400},
		{name: "missing customer_id", uri: "/v1/contact_stars", data: starBody(t, map[string]any{"agent_id": starTestAgentID}), expectCode: 400},
		{name: "other error", uri: "/v1/contact_stars", data: body, expectCall: true, callErr: fmt.Errorf("database error"), expectCode: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockContact, _, _ := newTestListenHandlerAll(mc)
			ctx := context.Background()

			if tt.expectCall {
				var ret []*contactstar.ContactStar
				if tt.callErr == nil {
					ret = []*contactstar.ContactStar{{ContactID: starTestContactID, AgentID: starTestAgentID, TMCreate: &created}}
				}
				mockContact.EXPECT().StarList(ctx, starTestCustomerID, starTestAgentID, tt.expectSize, tt.expectTok).Return(ret, tt.callErr)
			}

			res, err := h.processV1ContactStarsGet(ctx, &sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: tt.data})
			if err != nil || res.StatusCode != tt.expectCode {
				t.Fatalf("Wrong match. expect: %d, got: %v, %v", tt.expectCode, res, err)
			}
			if tt.expectCode == 200 {
				var got []*contactstar.ContactStar
				if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil || len(got) != 1 || got[0].ContactID != starTestContactID {
					t.Errorf("Wrong match. body: %s, err: %v", res.Data, errUnmarshal)
				}
			}
		})
	}
}
