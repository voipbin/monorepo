package request

// TelnyxV2DataNumberOrdersPost struct
type TelnyxV2DataNumberOrdersPost struct {
	PhoneNumbers []TelnyxPhoneNumber `json:"phone_numbers"`
}

// TelnyxPhoneNumber struct
type TelnyxPhoneNumber struct {
	PhoneNumber string `json:"phone_number"`
}
