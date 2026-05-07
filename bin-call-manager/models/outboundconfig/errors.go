package outboundconfig

import "errors"

// ErrDestinationNotWhitelisted is returned when a PSTN destination's country
// is not in the customer's OutboundConfig.DestinationWhitelist.
// bin-api-manager maps this sentinel to 400 Bad Request.
var ErrDestinationNotWhitelisted = errors.New("outbound destination country not whitelisted")
