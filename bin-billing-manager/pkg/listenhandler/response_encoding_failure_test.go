package listenhandler

import (
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
)

// Test_processRequest_responseCannotBeEncoded verifies that a result the JSON encoder rejects is
// answered with a plain 500 and no body. A timestamp outside the year range 0 to 9999 is rejected
// by the encoder.
func Test_processRequest_responseCannotBeEncoded(t *testing.T) {
	accountID := uuid.FromStringOrNil("6c000001-0000-0000-0000-000000000001")
	billingID := uuid.FromStringOrNil("6c000002-0000-0000-0000-000000000001")
	outOfRange := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	badAccount := &account.Account{Identity: commonidentity.Identity{ID: accountID}, TMCreate: &outOfRange}
	badBilling := &billing.Billing{Identity: commonidentity.Identity{ID: billingID}, TMCreate: &outOfRange}

	tests := []struct {
		name    string
		request *sock.Request
		prepare func(mockAccount *accounthandler.MockAccountHandler, mockBilling *billinghandler.MockBillingHandler)
	}{
		{
			"account list",
			&sock.Request{URI: "/v1/accounts?page_size=5", Method: sock.RequestMethodGet},
			func(a *accounthandler.MockAccountHandler, _ *billinghandler.MockBillingHandler) {
				a.EXPECT().List(gomock.Any(), uint64(5), gomock.Any(), gomock.Any()).Return([]*account.Account{badAccount}, nil)
			},
		},
		{
			"account get",
			&sock.Request{URI: "/v1/accounts/" + accountID.String(), Method: sock.RequestMethodGet},
			func(a *accounthandler.MockAccountHandler, _ *billinghandler.MockBillingHandler) {
				a.EXPECT().Get(gomock.Any(), accountID).Return(badAccount, nil)
			},
		},
		{
			"account basic info update",
			&sock.Request{URI: "/v1/accounts/" + accountID.String(), Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{"name":"n","detail":"d"}`)},
			func(a *accounthandler.MockAccountHandler, _ *billinghandler.MockBillingHandler) {
				a.EXPECT().UpdateBasicInfo(gomock.Any(), accountID, "n", "d").Return(badAccount, nil)
			},
		},
		{
			"balance add",
			&sock.Request{URI: "/v1/accounts/" + accountID.String() + "/balance_add_force", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{"balance":20}`)},
			func(a *accounthandler.MockAccountHandler, _ *billinghandler.MockBillingHandler) {
				a.EXPECT().AddBalance(gomock.Any(), accountID, int64(20)).Return(badAccount, nil)
			},
		},
		{
			"balance subtract",
			&sock.Request{URI: "/v1/accounts/" + accountID.String() + "/balance_subtract_force", Method: sock.RequestMethodPost, DataType: "application/json", Data: []byte(`{"balance":20}`)},
			func(a *accounthandler.MockAccountHandler, _ *billinghandler.MockBillingHandler) {
				a.EXPECT().SubtractBalance(gomock.Any(), accountID, int64(20)).Return(badAccount, nil)
			},
		},
		{
			"payment info update",
			&sock.Request{URI: "/v1/accounts/" + accountID.String() + "/payment_info", Method: sock.RequestMethodPut, DataType: "application/json", Data: []byte(`{"payment_type":"prepaid","payment_method":""}`)},
			func(a *accounthandler.MockAccountHandler, _ *billinghandler.MockBillingHandler) {
				a.EXPECT().UpdatePaymentInfo(gomock.Any(), accountID, account.PaymentTypePrepaid, account.PaymentMethodNone).Return(badAccount, nil)
			},
		},
		{
			"billing get",
			&sock.Request{URI: "/v1/billings/" + billingID.String(), Method: sock.RequestMethodGet},
			func(_ *accounthandler.MockAccountHandler, b *billinghandler.MockBillingHandler) {
				b.EXPECT().Get(gomock.Any(), billingID).Return(badBilling, nil)
			},
		},
		{
			"billing list",
			&sock.Request{URI: "/v1/billings?page_size=5", Method: sock.RequestMethodGet},
			func(_ *accounthandler.MockAccountHandler, b *billinghandler.MockBillingHandler) {
				b.EXPECT().List(gomock.Any(), uint64(5), gomock.Any(), gomock.Any()).Return([]*billing.Billing{badBilling}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
				billingHandler: mockBilling,
			}
			tt.prepare(mockAccount, mockBilling)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != 500 || len(res.Data) != 0 {
				t.Errorf("Wrong match. expect an empty 500 response, got: %+v", res)
			}
		})
	}
}
