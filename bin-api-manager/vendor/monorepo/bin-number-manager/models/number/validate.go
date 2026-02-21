package number

import (
	"fmt"
	"strings"
)

const (
	// VirtualNumberPrefix is the required prefix for virtual numbers
	VirtualNumberPrefix = "+899"

	// VirtualNumberLength is the required length for virtual numbers (+ followed by 12 digits)
	VirtualNumberLength = 13

	// VirtualNumberReservedPrefix is the prefix for reserved virtual numbers (+899000XXXXXX)
	VirtualNumberReservedPrefix = "+899000"

	// VirtualNumberCountryCode is the country code used for virtual numbers in available number responses
	VirtualNumberCountryCode = "899"
)

// ValidateVirtualNumber validates a virtual number string.
// If allowReserved is false, numbers in the reserved range +899000XXXXXX are rejected.
func ValidateVirtualNumber(num string, allowReserved bool) error {
	if !strings.HasPrefix(num, VirtualNumberPrefix) {
		return fmt.Errorf("virtual number must start with %s", VirtualNumberPrefix)
	}

	if len(num) != VirtualNumberLength {
		return fmt.Errorf("virtual number must be exactly %d characters", VirtualNumberLength)
	}

	for _, c := range num[1:] {
		if c < '0' || c > '9' {
			return fmt.Errorf("virtual number must contain only digits after +")
		}
	}

	if !allowReserved && strings.HasPrefix(num, VirtualNumberReservedPrefix) {
		return fmt.Errorf("virtual number range %sXXXXXX is reserved", VirtualNumberReservedPrefix)
	}

	return nil
}
