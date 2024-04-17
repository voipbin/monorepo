package response

import (
	"encoding/json"
	reflect "reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requestexternal/models/telnyx"
)

func TestTelnyxV2ResponsePhoneNumbersIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	type test struct {
		name string
		msg  []byte

		expectRes TelnyxV2ResponsePhoneNumbersIDGet
	}

	tests := []test{
		{
			"normal",
			[]byte(`{"data":{"id":"1579827332531618841","record_type":"phone_number","phone_number":"+15078888932","status":"active","tags":[],"connection_id":"","customer_reference":null,"external_pin":null,"t38_fax_gateway_enabled":true,"purchased_at":"2021-02-25T17:54:53Z","billing_group_id":null,"emergency_enabled":false,"emergency_address_id":"","call_forwarding_enabled":true,"cnam_listing_enabled":false,"call_recording_enabled":false,"messaging_profile_id":"","messaging_profile_name":"","number_block_id":null,"created_at":"2021-02-25T17:54:53.965Z","updated_at":"2021-02-25T17:54:55.001Z"}}`),
			TelnyxV2ResponsePhoneNumbersIDGet{
				Data: telnyx.PhoneNumber{
					ID:                    "1579827332531618841",
					RecordType:            "phone_number",
					PhoneNumber:           "+15078888932",
					Status:                "active",
					Tags:                  []string{},
					T38FaxGatewayEnabled:  true,
					PurchasedAt:           "2021-02-25T17:54:53Z",
					CallForwardingEnabled: true,
					CreatedAt:             "2021-02-25T17:54:53.965Z",
					UpdatedAt:             "2021-02-25T17:54:55.001Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := TelnyxV2ResponsePhoneNumbersIDGet{}
			if err := json.Unmarshal(tt.msg, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func TestTelnyxV2ResponseAvailableNumbersGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	type test struct {
		name string
		msg  []byte

		expectRes TelnyxV2ResponseAvailableNumbersGet
	}

	tests := []test{
		{
			"normal",
			[]byte(`{"data": [{"phone_number": "+16188850188", "reservable": true, "quickship": true, "vanity_format": null, "record_type": "available_phone_number", "cost_information": {"monthly_cost": "1.00000", "upfront_cost": "1.00000", "currency": "USD"}, "best_effort": false, "features": [{"name": "emergency"}, {"name": "fax"}, {"name": "voice"}, {"name": "sms"}], "region_information": [{"region_name": "IL", "region_type": "state"}, {"region_name": "US", "region_type": "country_code"}, {"region_name": "DOW", "region_type": "rate_center"}]}], "metadata": {"total_results": 1, "best_effort_results": 0}}`),
			TelnyxV2ResponseAvailableNumbersGet{
				Data: []telnyx.AvailableNumber{
					{
						PhoneNumber:  "+16188850188",
						Reservable:   true,
						QuickShip:    true,
						VanityFormat: "",
						RecordType:   "available_phone_number",
						CostInformation: telnyx.AvailableCostInformation{
							MonthlyCost: "1.00000",
							UpfrontCost: "1.00000",
							Currency:    "USD",
						},
						Features: []telnyx.AvailableFeature{
							{
								Name: "emergency",
							},
							{
								Name: "fax",
							},
							{
								Name: "voice",
							},
							{
								Name: "sms",
							},
						},
						RegionInformation: []telnyx.AvailableRegionInformation{
							{
								RegionName: "IL",
								RegionType: "state",
							},
							{
								RegionName: "US",
								RegionType: "country_code",
							},
							{
								RegionName: "DOW",
								RegionType: "rate_center",
							},
						},
					},
				},
				MetaData: telnyx.AvailableMetaData{
					TotalResults:      1,
					BestEffortResults: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := TelnyxV2ResponseAvailableNumbersGet{}
			if err := json.Unmarshal(tt.msg, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func TestTelnyxV2ResponseNumberOrdersPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	type test struct {
		name string
		msg  []byte

		expectRes TelnyxV2ResponseNumberOrdersPost
	}

	tests := []test{
		{
			"normal",
			[]byte(`{"data": {"id": "1d068d77-a651-49ef-b65b-85a888a8bd0b", "requirements_met": true, "created_at": "2021-02-25T17:54:53.632293+00:00", "status": "pending", "connection_id": null, "phone_numbers": [{"id": "ff252886-6145-4d90-ab02-5d56a1952af9", "requirements_met": true, "phone_number": "+15078888932", "regulatory_requirements": [], "status": "pending", "record_type": "number_order_phone_number"}], "messaging_profile_id": null, "customer_reference": null, "updated_at": "2021-02-25T17:54:53.632293+00:00", "billing_group_id": null, "phone_numbers_count": 1, "record_type": "number_order"}}`),
			TelnyxV2ResponseNumberOrdersPost{
				Data: telnyx.OrderNumber{
					ID:              "1d068d77-a651-49ef-b65b-85a888a8bd0b",
					RequirementsMet: true,
					Status:          "pending",
					PhoneNumbers: []telnyx.OrderNumberPhoneNumber{
						{
							ID:                     "ff252886-6145-4d90-ab02-5d56a1952af9",
							RequirementsMet:        true,
							PhoneNumber:            "+15078888932",
							Status:                 "pending",
							RecordType:             "number_order_phone_number",
							RegulatoryRequirements: []string{},
						},
					},
					CreatedAt:         "2021-02-25T17:54:53.632293+00:00",
					UpdatedAt:         "2021-02-25T17:54:53.632293+00:00",
					PhoneNumbersCount: 1,
					RecordType:        "number_order",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := TelnyxV2ResponseNumberOrdersPost{}
			if err := json.Unmarshal(tt.msg, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func TestTelnyxV2ResponsePhoneNumbersGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	type test struct {
		name string
		msg  []byte

		expectRes *TelnyxV2ResponsePhoneNumbersGet
	}

	tests := []test{
		{
			"normal",
			[]byte(`{"meta":{"total_pages":1,"total_results":2,"page_number":1,"page_size":250},"data":[{"id":"1580568175064384684","record_type":"phone_number","phone_number":"+12704940136","status":"active","tags":[],"connection_id":"","customer_reference":null,"external_pin":null,"t38_fax_gateway_enabled":true,"purchased_at":"2021-02-26T18:26:49Z","billing_group_id":null,"emergency_enabled":false,"emergency_address_id":"","call_forwarding_enabled":true,"cnam_listing_enabled":false,"call_recording_enabled":false,"messaging_profile_id":"","messaging_profile_name":"","number_block_id":null,"created_at":"2021-02-26T18:26:49.277Z","updated_at":"2021-02-26T18:26:50.340Z"}]}`),
			&TelnyxV2ResponsePhoneNumbersGet{
				Data: []telnyx.PhoneNumber{
					{
						ID:                    "1580568175064384684",
						RecordType:            "phone_number",
						PhoneNumber:           "+12704940136",
						Status:                telnyx.PhoneNumberStatusActive,
						Tags:                  []string{},
						ConnectionID:          "",
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
						UpdatedAt:             "2021-02-26T18:26:50.340Z",
					},
				},
				Meta: telnyx.PhoneNumberMetaData{
					PageNumber:   1,
					PageSize:     250,
					TotalPages:   1,
					TatalResults: 2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := TelnyxV2ResponsePhoneNumbersGet{}
			if err := json.Unmarshal(tt.msg, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(tt.expectRes, &res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}
