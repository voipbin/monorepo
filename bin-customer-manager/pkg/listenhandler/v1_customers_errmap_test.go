package listenhandler

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	pkgerrors "github.com/pkg/errors"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/pkg/customerhandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

// Test that processV1CustomersPost maps dbhandler.ErrNotFound → 404.
// List/Create are collection endpoints that would not naturally 404 from DB, but the
// handler must still relay whatever error the business layer returns via errorResponse.
// We use a typed cerrors.NotFound (which business handlers wrap around ErrNotFound)
// to validate the end-to-end mapping through processRequest.
func Test_processV1CustomersPost_errmap_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomer := customerhandler.NewMockCustomerHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		reqHandler:      mockReq,
		customerHandler: mockCustomer,
	}

	req := &sock.Request{
		URI:      "/v1/customers",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"name":"test","email":"test@example.com"}`),
	}

	// Business handler returns a typed cerrors.NotFound (wrapping dbhandler.ErrNotFound),
	// which should map to 404 via errorResponse rather than the legacy 500.
	notFoundErr := cerrors.NotFound(
		commonoutline.ServiceNameCustomerManager,
		"CUSTOMER_NOT_FOUND",
		"customer not found",
	).Wrap(pkgerrors.Wrap(dbhandler.ErrNotFound, "db lookup"))

	mockCustomer.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, notFoundErr)

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
	}
}

// Test that processV1CustomersGet maps dbhandler.ErrNotFound → 404.
func Test_processV1CustomersGet_errmap_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomer := customerhandler.NewMockCustomerHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		reqHandler:      mockReq,
		customerHandler: mockCustomer,
	}

	req := &sock.Request{
		URI:    "/v1/customers?page_size=10&page_token=2021-11-23T17:55:39.712000Z",
		Method: sock.RequestMethodGet,
		Data:   []byte(`{}`),
	}

	mockCustomer.EXPECT().
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
