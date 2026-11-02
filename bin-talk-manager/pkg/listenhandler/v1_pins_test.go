package listenhandler

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/models/pin"
)

const (
	testPinID    = "c1d2e3f4-42eb-11ed-9fa6-b4cd93af9798"
	testTargetID = "d1e2f3a4-42eb-11ed-9fa6-b4cd93af9799"
)

func testPinResponse() *pin.Pin {
	tm := time.Date(2026, 11, 2, 9, 15, 30, 0, time.UTC)
	return &pin.Pin{
		Identity:   commonidentity.Identity{ID: uuid.FromStringOrNil(testPinID), CustomerID: uuid.FromStringOrNil(testCustomerID)},
		Owner:      commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: uuid.FromStringOrNil(testOwnerID)},
		ChatID:     uuid.FromStringOrNil(testChatID),
		TargetType: pin.TargetTypeMessage,
		TargetID:   uuid.FromStringOrNil(testTargetID),
		TMCreate:   &tm,
	}
}

const expectPinJSON = `{"id":"` + testPinID + `","customer_id":"` + testCustomerID + `","owner_type":"agent","owner_id":"` + testOwnerID + `","chat_id":"` + testChatID + `","target_type":"message","target_id":"` + testTargetID + `","tm_create":"2026-11-02T09:15:30Z"}`

const ownerBody = `"customer_id":"` + testCustomerID + `","owner_type":"agent","owner_id":"` + testOwnerID + `"`

func Test_processRequest_pinsPost(t *testing.T) {
	customerID := uuid.FromStringOrNil(testCustomerID)
	ownerID := uuid.FromStringOrNil(testOwnerID)
	targetID := uuid.FromStringOrNil(testTargetID)

	h, m := newRoutingHandler(t)
	m.pin.EXPECT().PinCreate(gomock.Any(), customerID, "agent", ownerID, pin.TargetTypeMessage, targetID).Return(testPinResponse(), nil)

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/pins",
		Method: sock.RequestMethodPost,
		Data:   []byte(`{` + ownerBody + `,"target_type":"message","target_id":"` + testTargetID + `"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 200 || string(res.Data) != expectPinJSON {
		t.Errorf("Wrong match.\nexpect: 200 %s\ngot: %d %s", expectPinJSON, res.StatusCode, res.Data)
	}
}

func Test_processRequest_pinsPost_badRequest(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"not json", `{`},
		{"bad customer id", `{"customer_id":"x","owner_type":"agent","owner_id":"` + testOwnerID + `","target_type":"chat","target_id":"` + testTargetID + `"}`},
		{"bad owner id", `{"customer_id":"` + testCustomerID + `","owner_type":"agent","owner_id":"x","target_type":"chat","target_id":"` + testTargetID + `"}`},
		{"bad target id", `{` + ownerBody + `,"target_type":"chat","target_id":"x"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newRoutingHandler(t)
			res, err := h.processRequest(&sock.Request{URI: "/v1/pins", Method: sock.RequestMethodPost, Data: []byte(tt.data)})
			if err != nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %v %v", res, err)
			}
		})
	}
}

func Test_processRequest_pins_handlerError(t *testing.T) {
	notFound := cerrors.NotFound(commonoutline.ServiceNameTalkManager, "PIN_NOT_FOUND", "The pin was not found.")

	t.Run("post", func(t *testing.T) {
		h, m := newRoutingHandler(t)
		m.pin.EXPECT().PinCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, notFound)
		res, err := h.processRequest(&sock.Request{URI: "/v1/pins", Method: sock.RequestMethodPost, Data: []byte(`{` + ownerBody + `,"target_type":"chat","target_id":"` + testTargetID + `"}`)})
		if err != nil || res.StatusCode != http.StatusNotFound {
			t.Errorf("Wrong match. expect: 404, got: %v %v", res, err)
		}
	})

	t.Run("get", func(t *testing.T) {
		h, m := newRoutingHandler(t)
		m.pin.EXPECT().PinList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, notFound)
		res, err := h.processRequest(&sock.Request{URI: "/v1/pins", Method: sock.RequestMethodGet, Data: []byte(`{` + ownerBody + `}`)})
		if err != nil || res.StatusCode != http.StatusNotFound {
			t.Errorf("Wrong match. expect: 404, got: %v %v", res, err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		h, m := newRoutingHandler(t)
		m.pin.EXPECT().PinDelete(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, notFound)
		res, err := h.processRequest(&sock.Request{URI: "/v1/pins/" + testPinID, Method: sock.RequestMethodDelete, Data: []byte(`{` + ownerBody + `}`)})
		if err != nil || res.StatusCode != http.StatusNotFound {
			t.Errorf("Wrong match. expect: 404, got: %v %v", res, err)
		}
	})

	t.Run("unclassified error is a 500", func(t *testing.T) {
		h, m := newRoutingHandler(t)
		m.pin.EXPECT().PinList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("boom"))
		res, err := h.processRequest(&sock.Request{URI: "/v1/pins", Method: sock.RequestMethodGet, Data: []byte(`{` + ownerBody + `}`)})
		if err != nil || res.StatusCode != http.StatusInternalServerError {
			t.Errorf("Wrong match. expect: 500, got: %v %v", res, err)
		}
	})
}

func Test_processRequest_pinsGet(t *testing.T) {
	customerID := uuid.FromStringOrNil(testCustomerID)
	ownerID := uuid.FromStringOrNil(testOwnerID)
	chatID := uuid.FromStringOrNil(testChatID)

	tests := []struct {
		name string
		uri  string
		data string

		expectTargetType pin.TargetType
		expectChatID     uuid.UUID
		expectToken      string
		expectSize       uint64
	}{
		{"no filters", "/v1/pins", `{` + ownerBody + `}`, "", uuid.Nil, "", 0},
		{"paging", "/v1/pins?page_size=20&page_token=2026-11-02T09:15:30.000000Z", `{` + ownerBody + `}`, "", uuid.Nil, "2026-11-02T09:15:30.000000Z", 20},
		{"target type and chat id", "/v1/pins?page_size=5", `{` + ownerBody + `,"target_type":"message","chat_id":"` + testChatID + `"}`, pin.TargetTypeMessage, chatID, "", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newRoutingHandler(t)
			m.pin.EXPECT().PinList(gomock.Any(), customerID, "agent", ownerID, tt.expectTargetType, tt.expectChatID, tt.expectToken, tt.expectSize).Return([]*pin.Pin{testPinResponse()}, nil)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: []byte(tt.data)})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			expect := `[` + expectPinJSON + `]`
			if res.StatusCode != 200 || string(res.Data) != expect {
				t.Errorf("Wrong match.\nexpect: 200 %s\ngot: %d %s", expect, res.StatusCode, res.Data)
			}
		})
	}
}

func Test_processRequest_pinsGet_badRequest(t *testing.T) {
	for name, data := range map[string]string{
		"not json":        `{`,
		"bad customer id": `{"customer_id":"x","owner_type":"agent","owner_id":"` + testOwnerID + `"}`,
		"bad owner id":    `{"customer_id":"` + testCustomerID + `","owner_type":"agent","owner_id":"x"}`,
		"bad chat id":     `{` + ownerBody + `,"chat_id":"x"}`,
	} {
		t.Run(name, func(t *testing.T) {
			h, _ := newRoutingHandler(t)
			res, err := h.processRequest(&sock.Request{URI: "/v1/pins", Method: sock.RequestMethodGet, Data: []byte(data)})
			if err != nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %v %v", res, err)
			}
		})
	}
}

func Test_processRequest_pinsIDDelete(t *testing.T) {
	h, m := newRoutingHandler(t)
	m.pin.EXPECT().PinDelete(gomock.Any(), uuid.FromStringOrNil(testPinID), uuid.FromStringOrNil(testCustomerID), "agent", uuid.FromStringOrNil(testOwnerID)).Return(testPinResponse(), nil)

	res, err := h.processRequest(&sock.Request{URI: "/v1/pins/" + testPinID, Method: sock.RequestMethodDelete, Data: []byte(`{` + ownerBody + `}`)})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 200 || string(res.Data) != expectPinJSON {
		t.Errorf("Wrong match.\nexpect: 200 %s\ngot: %d %s", expectPinJSON, res.StatusCode, res.Data)
	}
}

func Test_processRequest_pinsIDDelete_badRequest(t *testing.T) {
	for name, tt := range map[string]struct{ uri, data string }{
		"not json":        {"/v1/pins/" + testPinID, `{`},
		"bad pin id":      {"/v1/pins/x", `{` + ownerBody + `}`},
		"bad customer id": {"/v1/pins/" + testPinID, `{"customer_id":"x","owner_type":"agent","owner_id":"` + testOwnerID + `"}`},
		"bad owner id":    {"/v1/pins/" + testPinID, `{"customer_id":"` + testCustomerID + `","owner_type":"agent","owner_id":"x"}`},
	} {
		t.Run(name, func(t *testing.T) {
			h, _ := newRoutingHandler(t)
			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: sock.RequestMethodDelete, Data: []byte(tt.data)})
			if err != nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %v %v", res, err)
			}
		})
	}
}
