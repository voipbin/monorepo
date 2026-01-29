package utilhandler

import (
	"regexp"
	"strings"
)

// Pre-compiled regex patterns for email validation (performance optimization)
var (
	consecutiveDotsRegex = regexp.MustCompile(`\.\.`)
	emailFormatRegex     = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// EmailIsValid returns true if the email is valid
func (h *utilHandler) EmailIsValid(e string) bool {
	return EmailIsValid(e)
}

func EmailIsValid(e string) bool {
	// Empty string is not a valid email
	if len(e) == 0 {
		return false
	}

	// Check for consecutive dots
	if consecutiveDotsRegex.MatchString(e) {
		return false
	}

	// Check for trailing dot
	if e[len(e)-1] == '.' {
		return false
	}

	// Check for trailing dot in domain part
	parts := strings.Split(e, "@")
	if len(parts) != 2 {
		return false
	}

	// Check for trailing dot in local part
	local := parts[0]
	domain := parts[1]

	// Validate local and domain are not empty
	if len(local) == 0 || len(domain) == 0 {
		return false
	}

	if local[len(local)-1] == '.' {
		return false
	}
	if domain[len(domain)-1] == '.' {
		return false
	}

	return emailFormatRegex.MatchString(e)
}
