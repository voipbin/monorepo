package listenhandler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/models/message"
)

func Test_processRequest_mentionsGet(t *testing.T) {
	customerID := uuid.FromStringOrNil(testCustomerID)
	agentID := uuid.FromStringOrNil(testOwnerID)
	resp := &message.Message{}
	resp.ID = uuid.FromStringOrNil(testMessageID)

	tests := []struct {
		name string
		uri  string
		data string

		setup      func(m *lhMocks)
		wantStatus int
		wantBody   any
	}{
		{
			name: "with query", uri: "/v1/mentions?page_size=7&page_token=tok",
			data: `{"customer_id":"` + testCustomerID + `","agent_id":"` + testOwnerID + `"}`,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MentionList(gomock.Any(), customerID, agentID, "tok", uint64(7)).Return([]*message.Message{resp}, nil)
			},
			wantStatus: 200, wantBody: []*message.Message{resp},
		},
		{
			name: "default page size", uri: "/v1/mentions",
			data: `{"customer_id":"` + testCustomerID + `","agent_id":"` + testOwnerID + `"}`,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MentionList(gomock.Any(), customerID, agentID, "", uint64(50)).Return([]*message.Message{}, nil)
			},
			wantStatus: 200, wantBody: []*message.Message{},
		},
		{
			name: "empty result is an empty array", uri: "/v1/mentions",
			data: `{"customer_id":"` + testCustomerID + `","agent_id":"` + testOwnerID + `"}`,
			setup: func(m *lhMocks) {
				m.msg.EXPECT().MentionList(gomock.Any(), customerID, agentID, "", uint64(50)).Return(nil, nil)
			},
			wantStatus: 200, wantBody: []*message.Message{},
		},
		{
			name: "invalid body", uri: "/v1/mentions", data: `{`,
			setup: func(m *lhMocks) {}, wantStatus: 400,
		},
		{
			name: "missing customer id", uri: "/v1/mentions", data: `{"agent_id":"` + testOwnerID + `"}`,
			setup: func(m *lhMocks) {}, wantStatus: 400,
		},
		{
			name: "missing agent id", uri: "/v1/mentions", data: `{"customer_id":"` + testCustomerID + `"}`,
			setup: func(m *lhMocks) {}, wantStatus: 400,
		},
		{
			name: "invalid agent id", uri: "/v1/mentions", data: `{"customer_id":"` + testCustomerID + `","agent_id":"x"}`,
			setup: func(m *lhMocks) {}, wantStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newRoutingHandler(t)
			tt.setup(m)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, DataType: "application/json", Data: []byte(tt.data)})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.wantStatus {
				t.Fatalf("Wrong status. expect: %d, got: %d", tt.wantStatus, res.StatusCode)
			}
			if tt.wantBody != nil {
				expect, _ := json.Marshal(tt.wantBody)
				if !reflect.DeepEqual([]byte(res.Data), expect) {
					t.Errorf("Wrong body.\nexpect: %s\ngot: %s", expect, res.Data)
				}
			}
		})
	}
}

func Test_processRequest_mentionsGet_error(t *testing.T) {
	h, m := newRoutingHandler(t)
	m.msg.EXPECT().MentionList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("boom"))

	res, err := h.processRequest(&sock.Request{
		URI: "/v1/mentions", Method: sock.RequestMethodGet,
		Data: []byte(`{"customer_id":"` + testCustomerID + `","agent_id":"` + testOwnerID + `"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("Wrong status. expect: 500, got: %d", res.StatusCode)
	}
}

func Test_processRequest_mentionsOtherMethods(t *testing.T) {
	h, _ := newRoutingHandler(t)

	for _, method := range []sock.RequestMethod{sock.RequestMethodPost, sock.RequestMethodPut, sock.RequestMethodDelete} {
		res, err := h.processRequest(&sock.Request{URI: "/v1/mentions", Method: method})
		if err != nil || res.StatusCode != 404 {
			t.Errorf("Wrong match for %s. expect: 404, got: %v, %v", method, res, err)
		}
	}
}
