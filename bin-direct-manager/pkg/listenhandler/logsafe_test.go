package listenhandler

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/directhandler"
)

func Test_maskURI(t *testing.T) {
	tests := []struct {
		name   string
		uri    string
		expect string
	}{
		{"no hash uri", "/v1/directs?page_size=10", "/v1/directs?page_size=10"},
		{"id uri", "/v1/directs/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a", "/v1/directs/bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a"},
		{"by-hash uri", "/v1/directs/by-hash/" + secretLogHash, "/v1/directs/by-hash/direct.01234..."},
		{"by-hash uri with query string", "/v1/directs/by-hash/" + secretLogHash + "?a=b", "/v1/directs/by-hash/direct.01234...?a=b"},
		{"by-hash uri with trailing path", "/v1/directs/by-hash/" + secretLogHash + "/x", "/v1/directs/by-hash/direct.01234.../x"},
		{"short hash unchanged", "/v1/directs/by-hash/direct.abc", "/v1/directs/by-hash/direct.abc"},
		{"empty hash", "/v1/directs/by-hash/", "/v1/directs/by-hash/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskURI(tt.uri)
			if got != tt.expect {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expect, got)
			}
			if strings.Contains(got, secretLogHashFragment) {
				t.Errorf("the masked uri still carries the hash: %s", got)
			}
		})
	}
}

func Test_requestLogFields(t *testing.T) {
	tests := []struct {
		name   string
		req    *sock.Request
		expect logrus.Fields
	}{
		{"nil", nil, logrus.Fields{}},
		{
			"by-hash request with body",
			&sock.Request{
				URI:      "/v1/directs/by-hash/" + secretLogHash,
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     json.RawMessage(`{"hash":"` + secretLogHash + `"}`),
			},
			logrus.Fields{
				"method":    sock.RequestMethodGet,
				"uri":       "/v1/directs/by-hash/direct.01234...",
				"data_type": "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requestLogFields(tt.req)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, got)
			}
		})
	}
}

func Test_responseLogFields(t *testing.T) {
	tests := []struct {
		name   string
		res    *sock.Response
		expect logrus.Fields
	}{
		{"nil", nil, logrus.Fields{}},
		{
			"response with direct body",
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       json.RawMessage(`{"hash":"` + secretLogHash + `"}`),
			},
			logrus.Fields{
				"status_code": 200,
				"data_type":   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := responseLogFields(tt.res)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, got)
			}
		})
	}
}

func Test_processRequest_noHashInLogs(t *testing.T) {
	directID := uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a")
	sentinelDirect := &direct.Direct{
		Identity: commonidentity.Identity{
			ID:         directID,
			CustomerID: uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"),
		},
		ResourceType: "extension",
		ResourceID:   uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
		Hash:         secretLogHash,
	}

	tests := []struct {
		name    string
		request *sock.Request

		setup     func(mockDirect *directhandler.MockDirectHandler)
		expectMsg string
		expectSC  int
		expectBod bool // response body carries the sentinel hash
	}{
		{
			name: "by-hash get success, response body carries the direct json",
			request: &sock.Request{
				URI:      "/v1/directs/by-hash/" + secretLogHash,
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			setup: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().GetByHash(gomock.Any(), secretLogHash).Return(sentinelDirect, nil)
			},
			expectMsg: "Sending response.",
			expectSC:  200,
			expectBod: true,
		},
		{
			name: "by-hash get failure",
			request: &sock.Request{
				URI:      "/v1/directs/by-hash/" + secretLogHash,
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			setup: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().GetByHash(gomock.Any(), secretLogHash).Return(nil, errors.New("backend failure"))
			},
			expectMsg: "Could not handle the message correctly.",
			expectSC:  400,
		},
		{
			name: "by-hash uri with post method goes to the default branch",
			request: &sock.Request{
				URI:      "/v1/directs/by-hash/" + secretLogHash,
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     json.RawMessage(`{"hash":"` + secretLogHash + `"}`),
			},
			setup:     func(m *directhandler.MockDirectHandler) {},
			expectMsg: "Could not find corresponded message handler.",
			expectSC:  404,
		},
		{
			name: "by-hash uri with query string",
			request: &sock.Request{
				URI:      "/v1/directs/by-hash/" + secretLogHash + "?x=y",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},
			setup:     func(m *directhandler.MockDirectHandler) {},
			expectMsg: "Received request.",
			expectSC:  404,
		},
		{
			name: "id get success, response body carries the direct json",
			request: &sock.Request{
				URI:      "/v1/directs/" + directID.String(),
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			setup: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Get(gomock.Any(), directID).Return(sentinelDirect, nil)
			},
			expectMsg: "Sending response.",
			expectSC:  200,
			expectBod: true,
		},
		{
			name: "regenerate success, response body carries the new hash",
			request: &sock.Request{
				URI:      "/v1/directs/" + directID.String() + "/regenerate",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},
			setup: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Regenerate(gomock.Any(), directID).Return(sentinelDirect, nil)
			},
			expectMsg: "Sending response.",
			expectSC:  200,
			expectBod: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := newSecretLogHook(t)

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockDirect := directhandler.NewMockDirectHandler(mc)
			h := &listenHandler{
				sockHandler:   mockSock,
				directHandler: mockDirect,
			}
			tt.setup(mockDirect)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectSC {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectSC, res.StatusCode)
			}
			if tt.expectBod && !strings.Contains(string(res.Data), secretLogHash) {
				t.Errorf("the response body should carry the sentinel hash so the test is meaningful: %s", res.Data)
			}

			// the leading 12 chars of the hash may remain in the masked uri.
			assertNoSecretInLogs(t, hook, tt.expectMsg, secretLogHash, secretLogHashFragment)
		})
	}
}
