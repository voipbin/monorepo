package utilhandler

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/exp/maps"
)

// URLParseFilters parses filter and returns parsed filters
func (h *utilHandler) URLParseFilters(u *url.URL) map[string]string {
	return URLParseFilters(u)
}

// URLMergeFilters merges the given urii with the given filters
// the filters items will be have the "filter_" prefix
func (h *utilHandler) URLMergeFilters(uri string, filters map[string]string) string {
	return URLMergeFilters(uri, filters)
}

// URLParseFilters parses filter and returns parsed filters
func URLParseFilters(u *url.URL) map[string]string {
	res := map[string]string{}

	tmpQueris := u.Query()
	for k, v := range tmpQueris {
		if !strings.HasPrefix(k, "filter_") {
			continue
		}

		// Safety check: ensure slice is not empty before accessing
		if len(v) == 0 {
			continue
		}

		filter := strings.TrimPrefix(k, "filter_")
		res[filter] = v[0]
	}

	return res
}

// URLMergeFilters merges the given uri with the given filters.
// The filters items will have the "filter_" prefix.
// Note: This function always uses "&" as separator because it's designed to append
// to URIs that already have query parameters from the caller.
func URLMergeFilters(uri string, filters map[string]string) string {
	if len(filters) == 0 {
		return uri
	}

	res := uri

	keys := maps.Keys(filters)
	sort.Strings(keys)

	for _, k := range keys {
		res = fmt.Sprintf("%s&filter_%s=%s", res, url.QueryEscape(k), url.QueryEscape(filters[k]))
	}

	return res
}
