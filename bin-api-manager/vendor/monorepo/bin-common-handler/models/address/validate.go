package address

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/gofrs/uuid"
)

// telRegex validates E.164 format: + followed by 7-15 digits
var telRegex = regexp.MustCompile(`^\+[0-9]{7,15}$`)

// Validate validates the Address Target field based on Type.
// Returns nil if valid, error with details if invalid.
func (a *Address) Validate() error {
	return ValidateTarget(a.Type, a.Target)
}

// ValidateTarget validates a target string for a specific type.
// Useful when validating before constructing an Address.
func ValidateTarget(addressType Type, target string) error {
	switch addressType {
	case TypeNone:
		return nil
	case TypeTel:
		return validateTel(target)
	case TypeEmail:
		return validateEmail(target)
	case TypeSIP:
		return validateSIP(target)
	case TypeAgent, TypeConference, TypeLine, TypeExtension:
		return validateUUID(target)
	default:
		return fmt.Errorf("unknown address type: %s", addressType)
	}
}

// validateTel validates E.164 format: + followed by 7-15 digits
func validateTel(target string) error {
	if !telRegex.MatchString(target) {
		return fmt.Errorf("invalid tel format: must be + followed by 7-15 digits")
	}
	return nil
}

// validateEmail validates RFC 5322 email format
func validateEmail(target string) error {
	_, err := mail.ParseAddress(target)
	if err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}
	return nil
}

// validateSIP validates user@domain format
func validateSIP(target string) error {
	parts := strings.SplitN(target, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid sip format: must be user@domain")
	}
	return nil
}

// validateUUID validates UUID format
func validateUUID(target string) error {
	if uuid.FromStringOrNil(target) == uuid.Nil {
		return fmt.Errorf("invalid uuid format")
	}
	return nil
}
