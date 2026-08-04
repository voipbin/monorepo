package agent

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
)

// maxTagIDsFilter bounds how many tag ids a single tag_ids filter value may
// contain. applyTagIDsFilter builds one OR-ed JSON_CONTAINS clause per id, so
// an unbounded count would let an authenticated caller build an arbitrarily
// large query against the shared agents table. 100 comfortably covers any
// realistic skill-tag combination.
const maxTagIDsFilter = 100

// FormatTagIDsFilter joins tag ids into the comma-separated wire format used
// by the tag_ids filter (see FieldStruct.TagIDs and ParseTagIDsFilter). Nil
// UUIDs are dropped. An empty/nil input returns "".
func FormatTagIDsFilter(ids []uuid.UUID) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != uuid.Nil {
			parts = append(parts, id.String())
		}
	}
	return strings.Join(parts, ",")
}

// ParseTagIDsFilter parses a comma-joined tag_ids filter value. An empty
// input means "no filter" (nil, nil) -- callers that must distinguish "no
// tag_ids key at all" from "tag_ids key present but empty" have to check
// that before calling this function (see bin-api-manager's
// convertAgentFilters). A non-empty input that parses to zero valid ids
// (e.g. ",", " ", or the nil UUID) is treated as a caller error, not as "no
// constraint" -- silently matching everyone when the caller explicitly
// supplied garbage tag_ids would be surprising and mask bugs upstream.
// uuid.Nil is rejected as an invalid tag id, symmetric with
// FormatTagIDsFilter dropping it.
func ParseTagIDsFilter(s string) ([]uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	if len(parts) > maxTagIDsFilter {
		return nil, fmt.Errorf("tag_ids filter contains too many ids (max %d)", maxTagIDsFilter)
	}

	res := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		id, err := uuid.FromString(p)
		if err != nil {
			return nil, fmt.Errorf("invalid tag id %q: %w", p, err)
		}
		if id == uuid.Nil {
			return nil, fmt.Errorf("invalid tag id %q: nil UUID is not a valid tag id", p)
		}

		res = append(res, id)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("tag_ids filter %q contains no valid ids", s)
	}

	return res, nil
}
