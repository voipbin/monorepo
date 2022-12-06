package requestexternal

import (
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requestexternal/models/telnyx"
)

const (
	testToken     = "KEY017B6ED1E90D8FC5DB6ED95F1ACFE4F5_WzTaTxsXJCdwOviG4t1xMM"
	testProfileID = "40017f8e-49bd-4f16-9e3d-ef103f916228"
)

func TestTelnyxAvailableNumberGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := requestExternal{}

	tests := []struct {
		name               string
		country            string
		locality           string
		administrativeArea string
		limit              int
	}{
		{
			"normal us",
			"us",
			"",
			"",
			1,
		},
		{
			"multiple numbers us",
			"us",
			"",
			"",
			3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := h.TelnyxAvailableNumberGets(testToken, tt.country, tt.locality, tt.administrativeArea, uint(tt.limit))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.limit {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.limit, len(res))
			}

			time.Sleep(time.Second * 1)
		})
	}
}

func Test_TelnyxPhoneNumbersIDGet(t *testing.T) {

	tests := []struct {
		name      string
		id        string
		expectRes *telnyx.PhoneNumber
	}{
		{
			"normal us number",
			"1748688147379652251",
			&telnyx.PhoneNumber{
				ID:                    "1748688147379652251",
				RecordType:            "phone_number",
				PhoneNumber:           "+14703298699",
				Status:                "active",
				Tags:                  []string{},
				ConnectionID:          "1943381594949551263",
				CustomerReference:     "",
				ExternalPin:           "",
				T38FaxGatewayEnabled:  true,
				PurchasedAt:           "2021-10-16T17:31:11Z",
				BillingGroupID:        "",
				EmergencyEnabled:      false,
				EmergencyAddressID:    "",
				CallForwardingEnabled: true,
				CNAMListingEnabled:    false,
				CallRecordingEnabled:  false,
				MessagingProfileID:    testProfileID,
				MessagingProfileName:  "",
				NumberBlockID:         "",
				CreatedAt:             "2021-10-16T17:31:11.737Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := requestExternal{}

			res, err := h.TelnyxPhoneNumbersIDGet(testToken, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.UpdatedAt = res.UpdatedAt
			tt.expectRes.MessagingProfileName = res.MessagingProfileName
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
