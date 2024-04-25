package utilhandler

import (
	"net/url"
	"strings"
)

// URLParseFilters parses filter and returns parsed filters
func (h *utilHandler) URLParseFilters(u *url.URL) map[string]string {
	return URLParseFilters(u)
}

// URLParseFilters parses filter and returns parsed filters
func URLParseFilters(u *url.URL) map[string]string {
	res := map[string]string{}

	tmpQueris := u.Query()
	for k, v := range tmpQueris {
		if !strings.HasPrefix(k, "filter_") {
			continue
		}

		filter := strings.TrimPrefix(k, "filter_")
		res[filter] = v[0]
	}

	return res
}
