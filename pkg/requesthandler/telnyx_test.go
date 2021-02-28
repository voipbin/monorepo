package requesthandler

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler/models/telnyx"
)

func TestTelnyxAvailableNumberGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay")

	type test struct {
		name               string
		country            string
		locality           string
		administrativeArea string
		limit              int
	}

	tests := []test{
		{
			"normal us",
			"us",
			"",
			"",
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := reqHandler.TelnyxAvailableNumberGets(tt.country, tt.locality, tt.administrativeArea, uint(tt.limit))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.limit {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.limit, len(res))
			}
		})
	}
}

func TestTelnyxPhoneNumbersIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay")

	type test struct {
		name      string
		id        string
		expectRes *telnyx.PhoneNumber
	}

	tests := []test{
		{
			"normal us number",
			"1580568175064384684",
			&telnyx.PhoneNumber{
				ID:                    "1580568175064384684",
				RecordType:            "phone_number",
				PhoneNumber:           "+12704940136",
				Status:                "active",
				Tags:                  []string{"test"},
				ConnectionID:          "1526401767787464160",
				CustomerReference:     "",
				ExternalPin:           "",
				T38FaxGatewayEnabled:  true,
				PurchasedAt:           "2021-02-26T18:26:49Z",
				BillingGroupID:        "",
				EmergencyEnabled:      false,
				EmergencyAddressID:    "",
				CallForwardingEnabled: true,
				CNAMListingEnabled:    false,
				CallRecordingEnabled:  false,
				MessagingProfileID:    "",
				MessagingProfileName:  "",
				NumberBlockID:         "",
				CreatedAt:             "2021-02-26T18:26:49.277Z",
				UpdatedAt:             "2021-02-27T17:44:09.724Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := reqHandler.TelnyxPhoneNumbersIDGet(tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// Number creation/delete test
// Careful!!! this test does actually create the number and delete
// it takes a actual money!
// func TestTelnyxPhoneNumbersCreateDelete(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSock := rabbitmqhandler.NewMockRabbit(mc)
// 	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay")

// 	type test struct {
// 		name string
// 	}

// 	tests := []test{
// 		{
// 			"normal us number",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			// get available numbers
// 			availNumbers, err := reqHandler.TelnyxAvailableNumberGets("US", "", "", 1)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			// // order numbers
// 			// orderNumber, err := reqHandler.TelnyxNumberOrdersPost([]string{availNumbers[0].PhoneNumber})
// 			// if err != nil {
// 			// 	t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			// }

// 			tmp, err := reqHandler.TelnyxPhoneNumbersGet(1, "", availNumbers[0].PhoneNumber)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			// delete numbers
// 			if _, err := reqHandler.TelnyxPhoneNumbersIDDelete(tmp[0].ID); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }
