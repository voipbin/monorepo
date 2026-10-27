package dbhandler

import (
	"context"
	"database/sql/driver"
	"testing"
)

func Test_BridgeJSONFunctions_cov(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	id := "d1f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e01"
	channelID := "d1f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e02"
	sel := `SELECT .* FROM call_bridges WHERE id = \?`

	runCovSQLCases(t, []covSQLCase{
		{
			name: "BridgeAddChannelID",
			do:   func(h *handler) error { return h.BridgeAddChannelID(ctx, id, channelID) },
			wantSQL: "UPDATE call_bridges SET channel_ids = json_array_append(channel_ids, '$', ?), " +
				"tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{channelID, nil, id},
			cacheSelect: sel,
		},
		{
			name: "BridgeRemoveChannelID",
			do:   func(h *handler) error { return h.BridgeRemoveChannelID(ctx, id, channelID) },
			wantSQL: "UPDATE call_bridges SET channel_ids = json_remove(channel_ids, " +
				"replace(json_search(channel_ids, 'one', ?), '\"', '')), tm_update = ? WHERE id = ?",
			wantArgs:    []driver.Value{channelID, nil, id},
			cacheSelect: sel,
		},
	})
}
