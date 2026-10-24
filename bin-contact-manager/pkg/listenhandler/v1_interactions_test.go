package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"

	"monorepo/bin-contact-manager/pkg/addresshandler"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
)

func newTestListenHandlerAll(mc *gomock.Controller) (*listenHandler, *contacthandler.MockContactHandler, *addresshandler.MockAddressHandler, *casehandler.MockCaseHandler) {
	mockContact := contacthandler.NewMockContactHandler(mc)
	mockAddr := addresshandler.NewMockAddressHandler(mc)
	mockCase := casehandler.NewMockCaseHandler(mc)

	h := &listenHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		contactHandler: mockContact,
		addressHandler: mockAddr,
		caseHandler:    mockCase,
	}
	return h, mockContact, mockAddr, mockCase
}

func Test_processV1InteractionsGet_Validation(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-0051-0051-0051-000000000001")
	contactID := uuid.FromStringOrNil("11111111-0051-0051-0051-000000000002")
	addressID := uuid.FromStringOrNil("11111111-0051-0051-0051-000000000003")

	validBody := []byte(fmt.Sprintf(`{"customer_id":"%s"}`, customerID))

	tests := []struct {
		name string
		uri  string
		data []byte
	}{
		{"unparsable uri", "/v1/interactions?%zz", validBody},
		{"body absent", "/v1/interactions?contact_id=" + contactID.String(), nil},
		{"body malformed json", "/v1/interactions?contact_id=" + contactID.String(), []byte(`{not json`)},
		{"customer_id not a string", "/v1/interactions?contact_id=" + contactID.String(), []byte(`{"customer_id":123}`)},
		{"customer_id not a uuid", "/v1/interactions?contact_id=" + contactID.String(), []byte(`{"customer_id":"abc"}`)},
		{"filter count 0", "/v1/interactions?page_size=10", validBody},
		{"filter count 0 with malformed contact_id", "/v1/interactions?contact_id=not-a-uuid", validBody},
		{"filter count 2 contact and address", "/v1/interactions?contact_id=" + contactID.String() + "&address_id=" + addressID.String(), validBody},
		{"filter count 2 peer and contact", "/v1/interactions?peer_type=tel&peer_target=%2B15550000001&contact_id=" + contactID.String(), validBody},
		{"filter count 3", "/v1/interactions?peer_type=tel&peer_target=%2B15550000001&contact_id=" + contactID.String() + "&address_id=" + addressID.String(), validBody},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no downstream expectation: strict gomock fails if the handler is reached.
			h, _, _, _ := newTestListenHandlerAll(mc)

			res, err := h.processV1InteractionsGet(context.Background(), &sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: tt.data})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != 400 {
				t.Errorf("StatusCode = %d, want 400", res.StatusCode)
			}
		})
	}
}

func Test_processV1InteractionsGet_Success(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-0052-0052-0052-000000000001")
	contactID := uuid.FromStringOrNil("11111111-0052-0052-0052-000000000002")
	addressID := uuid.FromStringOrNil("11111111-0052-0052-0052-000000000003")
	referenceID := uuid.FromStringOrNil("11111111-0052-0052-0052-000000000004")
	body := []byte(fmt.Sprintf(`{"customer_id":"%s"}`, customerID))

	tests := []struct {
		name string
		uri  string

		expectSize       uint64
		expectToken      string
		expectPeerType   string
		expectPeerTarget string
		expectContactID  uuid.UUID
		expectAddressID  uuid.UUID
	}{
		{
			name:             "peer filter with paging",
			uri:              "/v1/interactions?peer_type=tel&peer_target=%2B15550000001&page_size=25&page_token=tok1",
			expectSize:       25,
			expectToken:      "tok1",
			expectPeerType:   "tel",
			expectPeerTarget: "+15550000001",
		},
		{
			name:            "contact filter",
			uri:             "/v1/interactions?contact_id=" + contactID.String(),
			expectContactID: contactID,
		},
		{
			name:            "address filter with non numeric page_size",
			uri:             "/v1/interactions?address_id=" + addressID.String() + "&page_size=abc",
			expectAddressID: addressID,
		},
		{
			name:           "peer_type alone is accepted as a peer filter",
			uri:            "/v1/interactions?peer_type=email",
			expectPeerType: "email",
		},
		{
			name:             "malformed contact_id with a peer filter is forwarded as nil uuid",
			uri:              "/v1/interactions?peer_type=tel&peer_target=%2B15550000002&contact_id=not-a-uuid",
			expectPeerType:   "tel",
			expectPeerTarget: "+15550000002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockContact, _, _ := newTestListenHandlerAll(mc)

			events := []*tmpeerevent.PeerEvent{
				{CustomerID: customerID, Publisher: "call-manager", EventType: "call_created", ReferenceID: referenceID, Direction: "incoming", Data: json.RawMessage(`{}`)},
			}
			mockContact.EXPECT().InteractionList(gomock.Any(), customerID, tt.expectSize, tt.expectToken, tt.expectPeerType, tt.expectPeerTarget, tt.expectContactID, tt.expectAddressID).Return(events, "next", nil)

			res, err := h.processV1InteractionsGet(context.Background(), &sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: body})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != 200 || res.DataType != "application/json" {
				t.Fatalf("response = %d %q, want 200 application/json", res.StatusCode, res.DataType)
			}

			var got []*tmpeerevent.PeerEvent
			if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil {
				t.Fatalf("response body is not a peer event list: %v", errUnmarshal)
			}
			if len(got) != 1 || got[0].ReferenceID != referenceID || got[0].EventType != "call_created" {
				t.Errorf("unexpected body: %s", res.Data)
			}
		})
	}
}

func Test_processV1InteractionsGet_DownstreamError(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-0053-0053-0053-000000000001")
	contactID := uuid.FromStringOrNil("11111111-0053-0053-0053-000000000002")
	body := []byte(fmt.Sprintf(`{"customer_id":"%s"}`, customerID))

	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockContact, _, _ := newTestListenHandlerAll(mc)

	mockContact.EXPECT().InteractionList(gomock.Any(), customerID, uint64(0), "", "", "", contactID, uuid.Nil).Return(nil, "", fmt.Errorf("backend unavailable"))

	res, err := h.processV1InteractionsGet(context.Background(), &sock.Request{URI: "/v1/interactions?contact_id=" + contactID.String(), Method: sock.RequestMethodGet, Data: body})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// an unclassified error maps through errorResponse
	if expect := errorResponse(fmt.Errorf("backend unavailable")); !reflect.DeepEqual(res, expect) {
		t.Errorf("response = %+v, want %+v", res, expect)
	}
}
