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
