package request

// TelnyxV2DataNumberOrdersPost struct
// See detail: https://developers.telnyx.com/docs/api/v2/numbers/Number-Orders#createNumberOrder
type TelnyxV2DataNumberOrdersPost struct {
	PhoneNumbers       []TelnyxPhoneNumber `json:"phone_numbers"`
	ConnectionID       string              `json:"connection_id"`
	MessagingProfileID string              `json:"messaging_profile_id"`
}

// TelnyxPhoneNumber struct
type TelnyxPhoneNumber struct {
	PhoneNumber string `json:"phone_number"`
}
