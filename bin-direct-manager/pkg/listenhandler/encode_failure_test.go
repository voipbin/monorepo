package listenhandler

import (
	"net/http"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/directhandler"
)

// Test_processRequest_resultNotEncodable checks the answer of every handler that encodes a
// result: a result the JSON encoder rejects is answered with 500 and carries no data, while
// the same request with a valid result is answered with 200 and the encoded data.
func Test_processRequest_resultNotEncodable(t *testing.T) {
	directID := uuid.FromStringOrNil("bbb3bed0-4d89-11ec-9cf7-4351c0fdbd4a")
	customerID := uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9")
	resourceID := uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527")

	// the encoder accepts years 0 to 9999 only
	badTime := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	goodTime := time.Date(2021, 11, 23, 17, 55, 39, 0, time.UTC)

	newDirect := func(tm time.Time) *direct.Direct {
		return &direct.Direct{
			Identity: commonidentity.Identity{
				ID:         directID,
				CustomerID: customerID,
			},
			ResourceType: "extension",
			ResourceID:   resourceID,
			Hash:         "direct.abcdef123456",
			TMCreate:     &tm,
		}
	}

	tests := []struct {
		name string

		request *sock.Request
		expect  func(m *directhandler.MockDirectHandler, res *direct.Direct)
	}{
		{
			name: "list",
			request: &sock.Request{
				URI:      "/v1/directs?page_size=10&page_token=2021-11-23T17:55:39.712000Z",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},
			expect: func(m *directhandler.MockDirectHandler, res *direct.Direct) {
				m.EXPECT().Gets(gomock.Any(), uint64(10), "2021-11-23T17:55:39.712000Z", gomock.Any()).Return([]*direct.Direct{res}, nil)
			},
		},
		{
			name: "get",
			request: &sock.Request{
				URI:    "/v1/directs/" + directID.String(),
				Method: sock.RequestMethodGet,
			},
			expect: func(m *directhandler.MockDirectHandler, res *direct.Direct) {
				m.EXPECT().Get(gomock.Any(), directID).Return(res, nil)
			},
		},
		{
			name: "get by hash",
			request: &sock.Request{
				URI:    "/v1/directs/by-hash/direct.abcdef123456",
				Method: sock.RequestMethodGet,
			},
			expect: func(m *directhandler.MockDirectHandler, res *direct.Direct) {
				m.EXPECT().GetByHash(gomock.Any(), "direct.abcdef123456").Return(res, nil)
			},
		},
		{
			name: "create",
			request: &sock.Request{
				URI:      "/v1/directs",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","resource_type":"extension","resource_id":"c31676f0-4e69-11ec-afe3-77ba49fae527"}`),
			},
			expect: func(m *directhandler.MockDirectHandler, res *direct.Direct) {
				m.EXPECT().Create(gomock.Any(), customerID, "extension", resourceID).Return(res, nil)
			},
		},
		{
			name: "regenerate",
			request: &sock.Request{
				URI:    "/v1/directs/" + directID.String() + "/regenerate",
				Method: sock.RequestMethodPost,
			},
			expect: func(m *directhandler.MockDirectHandler, res *direct.Direct) {
				m.EXPECT().Regenerate(gomock.Any(), directID).Return(res, nil)
			},
		},
		{
			name: "delete",
			request: &sock.Request{
				URI:    "/v1/directs/" + directID.String(),
				Method: sock.RequestMethodDelete,
			},
			expect: func(m *directhandler.MockDirectHandler, res *direct.Direct) {
				m.EXPECT().Delete(gomock.Any(), directID).Return(res, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" with an encodable result", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDirect := directhandler.NewMockDirectHandler(mc)
			tt.expect(mockDirect, newDirect(goodTime))
			h := NewListenHandler(sockhandler.NewMockSockHandler(mc), mockDirect).(*listenHandler)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("processRequest() error = %v", err)
			}
			if res.StatusCode != http.StatusOK || !strings.Contains(string(res.Data), "direct.abcdef123456") {
				t.Errorf("processRequest() = %d with data %q, want 200 with the encoded direct", res.StatusCode, res.Data)
			}
		})

		t.Run(tt.name+" with a result the encoder rejects", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDirect := directhandler.NewMockDirectHandler(mc)
			tt.expect(mockDirect, newDirect(badTime))
			h := NewListenHandler(sockhandler.NewMockSockHandler(mc), mockDirect).(*listenHandler)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("processRequest() error = %v, want the failure to be answered, not returned", err)
			}
			if res.StatusCode != http.StatusInternalServerError {
				t.Errorf("processRequest() status = %d, want 500", res.StatusCode)
			}
			if len(res.Data) != 0 {
				t.Errorf("processRequest() data = %q, want none", res.Data)
			}
		})
	}
}
