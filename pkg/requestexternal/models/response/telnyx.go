package response

import "gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requestexternal/models/telnyx"

// TelnyxV2ResponsePhoneNumbersGet struct
type TelnyxV2ResponsePhoneNumbersGet struct {
	Data []telnyx.PhoneNumber       `json:"data"`
	Meta telnyx.PhoneNumberMetaData `json:"meta"`
}

// TelnyxV2ResponsePhoneNumbersIDGet struct
type TelnyxV2ResponsePhoneNumbersIDGet struct {
	Data telnyx.PhoneNumber `json:"data"`
}

// TelnyxV2ResponsePhoneNumbers struct
type TelnyxV2ResponsePhoneNumbers struct {
	Data []telnyx.PhoneNumber `json:"data"`
}

// TelnyxV2ResponsePhoneNumbersIDDelete struct
type TelnyxV2ResponsePhoneNumbersIDDelete struct {
	Data telnyx.PhoneNumber `json:"data"`
}

// TelnyxV2ResponseNumbersIDUpdateConnectionID struct
type TelnyxV2ResponseNumbersIDUpdateConnectionID struct {
	Data telnyx.PhoneNumber `json:"data"`
}

// TelnyxV2ResponseAvailableNumbersGet struct
type TelnyxV2ResponseAvailableNumbersGet struct {
	Data     []telnyx.AvailableNumber `json:"data"`
	MetaData telnyx.AvailableMetaData `json:"metadata"`
}

// TelnyxV2ResponseNumberOrdersPost struct
type TelnyxV2ResponseNumberOrdersPost struct {
	Data telnyx.OrderNumber `json:"data"`
}

// TelnyxV2ResponsePhoneNumbersIDPPatch struct
type TelnyxV2ResponsePhoneNumbersIDPPatch struct {
	Data telnyx.PhoneNumber `json:"data"`
}
