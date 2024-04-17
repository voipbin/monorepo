package telnyx

import (
	reflect "reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

func TestPhoneNumberConvertNumber(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	type test struct {
		name   string
		number *PhoneNumber

		expectRes *number.Number
	}

	tests := []test{
		{
			"normal",
			&PhoneNumber{
				ID:                    "1579827332531618841",
				RecordType:            "phone_number",
				PhoneNumber:           "+15078888932",
				Status:                "active",
				Tags:                  []string{},
				ConnectionID:          "",
				CustomerReference:     "",
				ExternalPin:           "",
				T38FaxGatewayEnabled:  true,
				PurchasedAt:           "2021-02-25T17:54:53Z",
				BillingGroupID:        "",
				EmergencyEnabled:      false,
				EmergencyAddressID:    "",
				CallForwardingEnabled: true,
				CNAMListingEnabled:    false,
				CallRecordingEnabled:  false,
				MessagingProfileID:    "",
				MessagingProfileName:  "",
				NumberBlockID:         "",
				CreatedAt:             "2021-02-25T17:54:53.965Z",
				UpdatedAt:             "2021-02-25T17:54:55.001Z",
			},
			&number.Number{
				Number:              "+15078888932",
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "1579827332531618841",
				Status:              "active",
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-25 17:54:53.000",
				TMCreate:            "",
				TMUpdate:            "",
				TMDelete:            "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.number.ConvertNumber()
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}
