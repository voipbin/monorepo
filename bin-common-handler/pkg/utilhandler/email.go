package utilhandler

import (
	"regexp"
	"strings"
)

// EmailIsValid returns true if the email is valid
func (h *utilHandler) EmailIsValid(e string) bool {
	return EmailIsValid(e)
}

func EmailIsValid(e string) bool {
	// Check for consecutive dots
	if regexp.MustCompile(`\.\.`).MatchString(e) {
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
	if local[len(local)-1] == '.' {
		return false
	}
	if domain[len(domain)-1] == '.' {
		return false
	}

	emailRegex := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return emailRegex.MatchString(e)
}
