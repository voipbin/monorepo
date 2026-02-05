package telnyx

import (
	"time"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/providernumber"
)

// PhoneNumber struct struct
type PhoneNumber struct {
	ID                    string            `json:"id"`
	RecordType            string            `json:"record_type"`
	PhoneNumber           string            `json:"phone_number"`
	Status                PhoneNumberStatus `json:"status"`
	Tags                  []string          `json:"tags"`
	ConnectionID          string            `json:"connection_id"`
	ConnectionName        string            `json:"connection_name"`
	CustomerReference     string            `json:"customer_reference"`
	ExternalPin           string            `json:"external_pin"`
	T38FaxGatewayEnabled  bool              `json:"t38_fax_gateway_enabled"`
	PurchasedAt           string            `json:"purchased_at"`
	BillingGroupID        string            `json:"billing_group_id"`
	EmergencyEnabled      bool              `json:"emergency_enabled"`
	EmergencyAddressID    string            `json:"emergency_address_id"`
	EmergencyStatus       string            `json:"emergency_status"`
	CallForwardingEnabled bool              `json:"call_forwarding_enabled"`
	CNAMListingEnabled    bool              `json:"cnam_listing_enabled"`
	CallRecordingEnabled  bool              `json:"call_recording_enabled"`
	PhoneNumberType       string            `json:"phone_number_type"`
	BundleID              string            `json:"bundle_id"`
	MessagingProfileID    string            `json:"messaging_profile_id"`
	MessagingProfileName  string            `json:"messaging_profile_name"`
	NumberBlockID         string            `json:"number_block_id"`
	CreatedAt             string            `json:"created_at"`
	UpdatedAt             string            `json:"updated_at"`
	NumberLevelRouting    string            `json:"number_level_routing"`
	HDVoiceEnabled        bool              `json:"hd_voice_enabled"`

	// 2024.04.09
	//   {
	// 	"id": "1748688147379652251",
	// 	"record_type": "phone_number",
	// 	"phone_number": "+14703298699",
	// 	"status": "active",
	// 	"tags": [
	// 	  "tag1",
	// 	  "tag2"
	// 	],
	// 	"connection_id": "2054833017033065613",
	// 	"connection_name": "voipbin prod",
	// 	"customer_reference": null,
	// 	"external_pin": null,
	// 	"t38_fax_gateway_enabled": true,
	// 	"purchased_at": "2021-10-16T17:31:11Z",
	// 	"billing_group_id": null,
	// 	"emergency_enabled": false,
	// 	"emergency_address_id": "",
	// 	"emergency_status": "disabled",
	// 	"call_forwarding_enabled": true,
	// 	"cnam_listing_enabled": false,
	// 	"call_recording_enabled": false,
	// 	"phone_number_type": "local",
	// 	"bundle_id": null,
	// 	"messaging_profile_id": "40017f8e-49bd-4f16-9e3d-ef103f916228",
	// 	"messaging_profile_name": "voipbin production",
	// 	"number_block_id": null,
	// 	"created_at": "2021-10-16T17:31:11.737Z",
	// 	"updated_at": "2024-04-09T14:39:02.674Z",
	// 	"number_level_routing": "disabled",
	// 	"hd_voice_enabled": false
	//   }
}

// PhoneNumberMetaData struct
type PhoneNumberMetaData struct {
	PageNumber   int `json:"page_number"`
	PageSize     int `json:"page_size"`
	TotalPages   int `json:"total_pages"`
	TatalResults int `json:"total_results"`
}

// PhoneNumberStatus type
type PhoneNumberStatus string

// list of PhoneNumberStatus types
const (
	PhoneNumberStatusPurchasePending PhoneNumberStatus = "purchase_pending"
	PhoneNumberStatusPurchaseFailed  PhoneNumberStatus = "purchase_failed"
	PhoneNumberStatusPortPending     PhoneNumberStatus = "port_pending"
	PhoneNumberStatusActive          PhoneNumberStatus = "active"
	PhoneNumberStatusDeleted         PhoneNumberStatus = "deleted"
	PhoneNumberStatusPortFailed      PhoneNumberStatus = "port_failed"
	PhoneNumberStatusEmergencyOnly   PhoneNumberStatus = "emergency_only"
	PhoneNumberStatusPortedOut       PhoneNumberStatus = "ported_out"
	PhoneNumberStatusPortOutPending  PhoneNumberStatus = "port_out_pending"
)

// ConvertNumber returns converted number
func (t *PhoneNumber) ConvertNumber() *number.Number {

	// Parse purchasedAt from Telnyx API format
	var tmPurchase *time.Time
	if t.PurchasedAt != "" {
		parsed, err := time.Parse(time.RFC3339, t.PurchasedAt)
		if err == nil {
			tmPurchase = &parsed
		}
	}

	res := &number.Number{
		Number: t.PhoneNumber,

		ProviderName:        number.ProviderNameTelnyx,
		ProviderReferenceID: t.ID,

		Status: number.Status(t.Status),

		T38Enabled:       t.T38FaxGatewayEnabled,
		EmergencyEnabled: t.EmergencyEnabled,

		TMPurchase: tmPurchase,
		TMCreate:   nil,
		TMUpdate:   nil,
		TMDelete:   nil,
	}

	return res
}

// ConvertProviderNumber returns converted ProviderNumber
func (t *PhoneNumber) ConvertProviderNumber() *providernumber.ProviderNumber {

	res := &providernumber.ProviderNumber{
		ID:               t.ID,
		Status:           number.Status(t.Status),
		T38Enabled:       t.T38FaxGatewayEnabled,
		EmergencyEnabled: t.EmergencyEnabled,
	}

	return res
}

// note. for future use
//
// {
// 	"data": {
// 	  "id": "1579827332531618841",
// 	  "record_type": "phone_number",
// 	  "phone_number": "+15078888932",
// 	  "status": "deleted",
// 	  "tags": [],
// 	  "connection_id": "",
// 	  "customer_reference": null,
// 	  "external_pin": null,
// 	  "t38_fax_gateway_enabled": true,
// 	  "purchased_at": "2021-02-25T17:54:53Z",
// 	  "billing_group_id": null,
// 	  "emergency_enabled": false,
// 	  "emergency_address_id": "",
// 	  "call_forwarding_enabled": false,
// 	  "cnam_listing_enabled": false,
// 	  "call_recording_enabled": false,
// 	  "messaging_profile_id": null,
// 	  "messaging_profile_name": null,
// 	  "number_block_id": null,
// 	  "created_at": "2021-02-25T17:54:53.965Z",
// 	  "updated_at": "2021-02-26T16:17:50.908Z",
// 	  "voice": {
// 		"id": "1579827332531618841",
// 		"record_type": "voice_settings",
// 		"phone_number": "+15078888932",
// 		"connection_id": "",
// 		"customer_reference": null,
// 		"origination_verification_status": null,
// 		"origination_verification_status_updated_at": null,
// 		"caller_id_name_enabled": false,
// 		"tech_prefix_enabled": false,
// 		"translated_number": "",
// 		"usage_payment_method": "pay-per-minute",
// 		"call_forwarding": {
// 		  "call_forwarding_enabled": false,
// 		  "forwards_to": null,
// 		  "forwarding_type": null
// 		},
// 		"call_recording": {
// 		  "inbound_call_recording_enabled": false,
// 		  "inbound_call_recording_channels": "single",
// 		  "inbound_call_recording_format": "wav"
// 		},
// 		"cnam_listing": {
// 		  "cnam_listing_enabled": false,
// 		  "cnam_listing_details": null
// 		},
// 		"emergency": {
// 		  "emergency_enabled": false,
// 		  "emergency_address_id": "",
// 		  "emergency_status": "disabled"
// 		},
// 		"media_features": {
// 		  "media_handling_mode": "default",
// 		  "rtp_auto_adjust_enabled": true,
// 		  "accept_any_rtp_packets_enabled": false,
// 		  "t38_fax_gateway_enabled": true
// 		}
// 	  }
// 	}
// }
