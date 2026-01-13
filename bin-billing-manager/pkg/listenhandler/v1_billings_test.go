package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/billinghandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_processV1BillingsGet(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseBillings []*billing.Billing

		expectFilters map[billing.Field]any
		expectSize    uint64
		expectToken   string
		expectRes     *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/billings?page_size=10&page_token=2023-06-08%2003:22:17.995000&customer_id=6a93f71e-f542-11ee-9a48-7f8011d36229",
				Method: sock.RequestMethodGet,
			},

			responseBillings: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("69cacd9e-f542-11ee-ab6d-afb3c2c93e56"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6a1d387c-f542-11ee-b2cb-a36ed20fc369"),
					},
				},
			},

			expectFilters: map[billing.Field]any{
				billing.FieldCustomerID: uuid.FromStringOrNil("6a93f71e-f542-11ee-9a48-7f8011d36229"),
			},
			expectSize:  10,
			expectToken: "2023-06-08 03:22:17.995000",
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"69cacd9e-f542-11ee-ab6d-afb3c2c93e56","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_per_unit":0,"cost_total":0,"billing_unit_count":0,"tm_billing_start":"","tm_billing_end":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"6a1d387c-f542-11ee-b2cb-a36ed20fc369","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_per_unit":0,"cost_total":0,"billing_unit_count":0,"tm_billing_start":"","tm_billing_end":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				billingHandler: mockBilling,
			}

			mockBilling.EXPECT().Gets(gomock.Any(), tt.expectSize, tt.expectToken, tt.expectFilters).Return(tt.responseBillings, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
