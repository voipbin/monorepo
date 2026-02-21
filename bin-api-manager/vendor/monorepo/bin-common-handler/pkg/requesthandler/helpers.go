package requesthandler

import (
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	"slices"

	"github.com/pkg/errors"
)

func GetFilteredItems(m *sock.Request, filters []string) (map[string]any, error) {
	var req map[string]any
	if err := json.Unmarshal(m.Data, &req); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal request data")
	}

	res := make(map[string]any)
	for key, val := range req {
		if slices.Contains(filters, key) {
			res[key] = val
		}
	}

	return res, nil
}
