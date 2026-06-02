package listenhandler

import (
	"net/http"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	pkgerrors "github.com/pkg/errors"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

// Test that processV1AccesskeysGet maps dbhandler.ErrNotFound → 404 via errorResponse.
func Test_processV1AccesskeysGet_errmap_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &listenHandler{
		sockHandler:      mockSock,
		reqHandler:       mockReq,
		accesskeyHandler: mockAccesskey,
	}

	req := &sock.Request{
		URI:    "/v1/accesskeys?page_size=10&page_token=2021-11-23T17:55:39.712000Z",
		Method: sock.RequestMethodGet,
		Data:   []byte(`{}`),
	}

	mockAccesskey.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, pkgerrors.Wrap(dbhandler.ErrNotFound, "db lookup"))

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
	}
}

// Test that processV1AccesskeysPost maps dbhandler.ErrNotFound → 404 via errorResponse.
func Test_processV1AccesskeysPost_errmap_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &listenHandler{
		sockHandler:      mockSock,
		reqHandler:       mockReq,
		accesskeyHandler: mockAccesskey,
	}

	req := &sock.Request{
		URI:      "/v1/accesskeys",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"customer_id":"0324e804-ab36-11ef-8a73-971d5f0ad5ee","name":"n","detail":"d","expire":3600}`),
	}

	mockAccesskey.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, pkgerrors.Wrap(dbhandler.ErrNotFound, "db lookup"))

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
	}
}
