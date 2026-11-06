package requestexternal

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	providerErrorBodyLimit   = 16 * 1024 // bytes read from an error answer
	providerErrorDetailLimit = 200       // characters kept of the detail
)

// digitRun matches a run of 6 or more digits: a phone number or an id must not reach the log.
var digitRun = regexp.MustCompile(`[0-9]{6,}`)

// providerErrorDetail reads the error answer of an SMS provider and returns the error codes and descriptions
// only, in a form that is safe to log: it never returns the raw body, long digit runs are replaced by <num> and
// the result is cut to providerErrorDetailLimit characters. A body that is not the expected JSON gives "".
//
// Telnyx: {"errors":[{"code":"40301","title":"...","detail":"..."}]}
// Messagebird: {"errors":[{"code":2,"description":"...","parameter":"..."}]}
func providerErrorDetail(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, providerErrorBodyLimit))
	if err != nil {
		return ""
	}

	var parsed struct {
		Errors []struct {
			Code        any    `json:"code"`
			Title       string `json:"title"`
			Detail      string `json:"detail"`
			Description string `json:"description"`
		} `json:"errors"`
	}
	if errUnmarshal := json.Unmarshal(raw, &parsed); errUnmarshal != nil {
		return ""
	}

	parts := []string{}
	for _, e := range parsed.Errors {
		text := e.Description
		if text == "" {
			text = e.Title
		}
		if text == "" {
			text = e.Detail
		}

		code := ""
		if e.Code != nil {
			code = fmt.Sprintf("%v", e.Code)
		}

		if code != "" {
			text = strings.TrimSpace("code " + code + " " + text)
		}
		if text != "" {
			parts = append(parts, text)
		}
	}

	res := digitRun.ReplaceAllString(strings.Join(parts, "; "), "<num>")
	runes := []rune(res)
	if len(runes) > providerErrorDetailLimit {
		res = string(runes[:providerErrorDetailLimit])
	}

	return res
}
