package dbhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

// Golden ToSql() tests for the MySQL JSON-function squirrel.Expr call sites
// (VOIP-1078, plan §5 R1).
//
// WHY these exist: the 15 JSON-function sites migrated in this change have zero
// executable test coverage, because the unit-test database is SQLite, which does
// not implement json_array_append / json_insert / json_set / json_search /
// json_remove (see the pre-existing exclusion comments at call_test.go and
// confbridge_test.go). These are therefore pure string-assertion tests: they
// assert the SQL text and the argument order that ToSql() produces, and they do
// NOT execute against any database. They are the regression signal that the
// generated SQL still matches the pre-migration raw SQL (modulo whitespace) and
// that arguments are still bound in the same order.
//
// Each `want` string below was diffed 1:1 against the raw SQL that the function
// issued before this migration.

// goldenCase describes one expected ToSql() output.
type goldenCase struct {
	name     string
	sql      string
	args     []any
	wantSQL  string
	wantArgs []any
}

func runGoldenCases(t *testing.T, cases []goldenCase) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.sql != tt.wantSQL {
				t.Errorf("unexpected SQL.\nwant: %s\ngot:  %s", tt.wantSQL, tt.sql)
			}
			if !reflect.DeepEqual(tt.args, tt.wantArgs) {
				t.Errorf("unexpected args.\nwant: %#v\ngot:  %#v", tt.wantArgs, tt.args)
			}
		})
	}
}

// mustToSQL runs ToSql() and fails the test on error.
func mustToSQL(t *testing.T, sq interface{ ToSql() (string, []any, error) }) (string, []any) {
	t.Helper()
	q, args, err := sq.ToSql()
	if err != nil {
		t.Fatalf("ToSql() failed. err: %v", err)
	}
	return q, args
}

func Test_goldenBridgeJSONExpr(t *testing.T) {
	tmUpdate := "2026-08-04 01:02:03.000000"
	channelID := "1b2f5f9e-0000-11f0-9e2b-0242ac110002"
	bridgeID := "8a0c2c1a-0000-11f0-9e2b-0242ac110002"

	addSQL, addArgs := mustToSQL(t, buildBridgeAddChannelID(bridgeID, channelID, tmUpdate))
	removeSQL, removeArgs := mustToSQL(t, buildBridgeRemoveChannelID(bridgeID, channelID, tmUpdate))

	runGoldenCases(t, []goldenCase{
		{
			name: "BridgeRemoveChannelID",
			sql:  removeSQL,
			args: removeArgs,
			// pre-migration raw SQL:
			//   update call_bridges set
			//     channel_ids = json_remove(
			//       channel_ids, replace(json_search(channel_ids, 'one', ?), '"', '')
			//     ),
			//     tm_update = ?
			//   where id = ?
			wantSQL:  `UPDATE call_bridges SET channel_ids = json_remove(channel_ids, replace(json_search(channel_ids, 'one', ?), '"', '')), tm_update = ? WHERE id = ?`,
			wantArgs: []any{channelID, tmUpdate, bridgeID},
		},
		{
			name: "BridgeAddChannelID",
			sql:  addSQL,
			args: addArgs,
			// pre-migration raw SQL:
			//   update call_bridges set
			//     channel_ids = json_array_append(channel_ids, '$', ?),
			//     tm_update = ?
			//   where id = ?
			wantSQL:  "UPDATE call_bridges SET channel_ids = json_array_append(channel_ids, '$', ?), tm_update = ? WHERE id = ?",
			wantArgs: []any{channelID, tmUpdate, bridgeID},
		},
	})
}

func Test_goldenGroupcallDecreaseCountExpr(t *testing.T) {
	tmUpdate := "2026-08-04 01:02:03.000000"
	id := uuid.FromStringOrNil("2a3b4c5d-0000-11f0-9e2b-0242ac110002")

	callSQL, callArgs := mustToSQL(t, buildGroupcallDecreaseCount("call_count", id, tmUpdate))
	groupSQL, groupArgs := mustToSQL(t, buildGroupcallDecreaseCount("groupcall_count", id, tmUpdate))

	runGoldenCases(t, []goldenCase{
		{
			name: "GroupcallDecreaseCallCount",
			sql:  callSQL,
			args: callArgs,
			// pre-migration raw SQL:
			//   update call_groupcalls set
			//     call_count = call_count - 1,
			//     tm_update = ?
			//   where id = ?
			wantSQL:  "UPDATE call_groupcalls SET call_count = call_count - 1, tm_update = ? WHERE id = ?",
			wantArgs: []any{tmUpdate, id.Bytes()},
		},
		{
			name: "GroupcallDecreaseGroupcallCount",
			sql:  groupSQL,
			args: groupArgs,
			// pre-migration raw SQL:
			//   update call_groupcalls set
			//     groupcall_count = groupcall_count - 1,
			//     tm_update = ?
			//   where id = ?
			wantSQL:  "UPDATE call_groupcalls SET groupcall_count = groupcall_count - 1, tm_update = ? WHERE id = ?",
			wantArgs: []any{tmUpdate, id.Bytes()},
		},
	})
}

func Test_goldenCallJSONExpr(t *testing.T) {
	tmUpdate := "2026-08-04 01:02:03.000000"
	id := uuid.FromStringOrNil("3a3b4c5d-0000-11f0-9e2b-0242ac110002")
	other := uuid.FromStringOrNil("4a3b4c5d-0000-11f0-9e2b-0242ac110002")

	addChainSQL, addChainArgs := mustToSQL(t, buildCallAddChainedCallID(id, other, tmUpdate))
	rmChainSQL, rmChainArgs := mustToSQL(t, buildCallRemoveChainedCallID(id, other, tmUpdate))
	addEmSQL, addEmArgs := mustToSQL(t, buildCallAddExternalMediaID(id, other, tmUpdate))
	rmEmSQL, rmEmArgs := mustToSQL(t, buildCallRemoveExternalMediaID(id, other, tmUpdate))
	addRecSQL, addRecArgs := mustToSQL(t, buildCallAddRecordingIDs(id, other, tmUpdate))

	runGoldenCases(t, []goldenCase{
		{
			// also covers CallTXAddChainedCallID, which shares this builder
			name:     "CallAddChainedCallID",
			sql:      addChainSQL,
			args:     addChainArgs,
			wantSQL:  "UPDATE call_calls SET chained_call_ids = json_array_append(chained_call_ids, '$', ?), tm_update = ? WHERE id = ?",
			wantArgs: []any{other.String(), tmUpdate, id.Bytes()},
		},
		{
			// also covers CallTXRemoveChainedCallID, which shares this builder
			name:     "CallRemoveChainedCallID",
			sql:      rmChainSQL,
			args:     rmChainArgs,
			wantSQL:  `UPDATE call_calls SET chained_call_ids = json_remove(chained_call_ids, replace(json_search(chained_call_ids, 'one', ?), '"', '')), tm_update = ? WHERE id = ?`,
			wantArgs: []any{other.String(), tmUpdate, id.Bytes()},
		},
		{
			name:     "CallAddExternalMediaID",
			sql:      addEmSQL,
			args:     addEmArgs,
			wantSQL:  "UPDATE call_calls SET external_media_ids = json_array_append(external_media_ids, '$', ?), tm_update = ? WHERE id = ?",
			wantArgs: []any{other.String(), tmUpdate, id.Bytes()},
		},
		{
			name:     "CallRemoveExternalMediaID",
			sql:      rmEmSQL,
			args:     rmEmArgs,
			wantSQL:  `UPDATE call_calls SET external_media_ids = IF(json_search(external_media_ids, 'one', ?) IS NOT NULL, json_remove(external_media_ids, replace(json_search(external_media_ids, 'one', ?), '"', '')), external_media_ids), tm_update = ? WHERE id = ?`,
			wantArgs: []any{other.String(), other.String(), tmUpdate, id.Bytes()},
		},
		{
			name:     "CallAddRecordingIDs",
			sql:      addRecSQL,
			args:     addRecArgs,
			wantSQL:  "UPDATE call_calls SET recording_ids = json_array_append(recording_ids, '$', ?), tm_update = ? WHERE id = ?",
			wantArgs: []any{other.String(), tmUpdate, id.Bytes()},
		},
	})
}

func Test_goldenConfbridgeJSONExpr(t *testing.T) {
	tmUpdate := "2026-08-04 01:02:03.000000"
	id := uuid.FromStringOrNil("5a3b4c5d-0000-11f0-9e2b-0242ac110002")
	other := uuid.FromStringOrNil("6a3b4c5d-0000-11f0-9e2b-0242ac110002")
	channelID := "channel-abc"

	addRecSQL, addRecArgs := mustToSQL(t, buildConfbridgeAddRecordingIDs(id, other, tmUpdate))
	addEmSQL, addEmArgs := mustToSQL(t, buildConfbridgeAddExternalMediaID(id, other, tmUpdate))
	rmEmSQL, rmEmArgs := mustToSQL(t, buildConfbridgeRemoveExternalMediaID(id, other, tmUpdate))
	addCcSQL, addCcArgs := mustToSQL(t, buildConfbridgeAddChannelCallID(id, channelID, other, tmUpdate))
	rmCcSQL, rmCcArgs := mustToSQL(t, buildConfbridgeRemoveChannelCallID(id, channelID, tmUpdate))

	runGoldenCases(t, []goldenCase{
		{
			name:     "ConfbridgeAddRecordingIDs",
			sql:      addRecSQL,
			args:     addRecArgs,
			wantSQL:  "UPDATE call_confbridges SET recording_ids = json_array_append(recording_ids, '$', ?), tm_update = ? WHERE id = ?",
			wantArgs: []any{other.String(), tmUpdate, id.Bytes()},
		},
		{
			name:     "ConfbridgeAddExternalMediaID",
			sql:      addEmSQL,
			args:     addEmArgs,
			wantSQL:  "UPDATE call_confbridges SET external_media_ids = json_array_append(external_media_ids, '$', ?), tm_update = ? WHERE id = ?",
			wantArgs: []any{other.String(), tmUpdate, id.Bytes()},
		},
		{
			name:     "ConfbridgeRemoveExternalMediaID",
			sql:      rmEmSQL,
			args:     rmEmArgs,
			wantSQL:  `UPDATE call_confbridges SET external_media_ids = IF(json_search(external_media_ids, 'one', ?) IS NOT NULL, json_remove(external_media_ids, replace(json_search(external_media_ids, 'one', ?), '"', '')), external_media_ids), tm_update = ? WHERE id = ?`,
			wantArgs: []any{other.String(), other.String(), tmUpdate, id.Bytes()},
		},
		{
			name:     "ConfbridgeAddChannelCallID",
			sql:      addCcSQL,
			args:     addCcArgs,
			wantSQL:  "UPDATE call_confbridges SET channel_call_ids = json_insert(channel_call_ids, ?, ?), tm_update = ? WHERE id = ?",
			wantArgs: []any{`$."channel-abc"`, other.String(), tmUpdate, id.Bytes()},
		},
		{
			name:     "ConfbridgeRemoveChannelCallID",
			sql:      rmCcSQL,
			args:     rmCcArgs,
			wantSQL:  "UPDATE call_confbridges SET channel_call_ids = json_remove(channel_call_ids, ?), tm_update = ? WHERE id = ?",
			wantArgs: []any{`$."channel-abc"`, tmUpdate, id.Bytes()},
		},
	})
}

func Test_goldenChannelSetDataItemExpr(t *testing.T) {
	tmUpdate := "2026-08-04 01:02:03.000000"
	id := "channel-xyz"

	setSQL, setArgs := mustToSQL(t, buildChannelSetDataItem(id, "sip_pai", "alice", tmUpdate))

	// The pre-migration SQL interpolated the key: "json_set(data, '$.<key>', ?)".
	// It is now a bound argument, which is both the injection fix (plan §6
	// item 1) and the reason the arg list gained a leading path element.
	injectSQL, injectArgs := mustToSQL(t, buildChannelSetDataItem(id, `a'); drop table call_channels; --`, "x", tmUpdate))

	runGoldenCases(t, []goldenCase{
		{
			name:     "ChannelSetDataItem",
			sql:      setSQL,
			args:     setArgs,
			wantSQL:  "UPDATE call_channels SET data = json_set(data, ?, ?), tm_update = ? WHERE id = ?",
			wantArgs: []any{`$."sip_pai"`, "alice", tmUpdate, id},
		},
		{
			// A hostile key must not alter the SQL text at all; it stays a value.
			name:     "ChannelSetDataItem_injectionAttemptStaysAnArgument",
			sql:      injectSQL,
			args:     injectArgs,
			wantSQL:  "UPDATE call_channels SET data = json_set(data, ?, ?), tm_update = ? WHERE id = ?",
			wantArgs: []any{`$."a'); drop table call_channels; --"`, "x", tmUpdate, id},
		},
	})
}
