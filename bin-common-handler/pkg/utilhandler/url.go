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

	tmpParams, err := url.QueryUnescape(u.Query().Encode())
	if err != nil {
		return res
	}

	params := strings.Split(tmpParams, "&")
	for _, param := range params {
		if !strings.HasPrefix(param, "filter_") {
			continue
		}

		filter := strings.TrimPrefix(param, "filter_")
		tmps := strings.Split(filter, "=")
		if len(tmps) == 1 {
			res[tmps[0]] = ""
		} else {
			res[tmps[0]] = tmps[1]
		}
	}

	return res
}
