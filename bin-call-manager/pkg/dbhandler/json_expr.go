package dbhandler

import (
	"github.com/Masterminds/squirrel"
)

// Shared MySQL JSON-function expressions for the VOIP-1078 squirrel migration.
//
// WHY squirrel.Expr (applies to every helper in this file, and to every call
// site that uses one): squirrel has no builder form for MySQL's JSON functions.
// json_array_append / json_remove / json_search / json_insert / json_set must be
// emitted as raw SQL expressions, so they go through squirrel.Expr with their
// operands still bound as query arguments (never interpolated).
//
// Sanctioned by docs/conventions/database.md §7.1, whose exception list is
// extended by this change to name the MySQL JSON functions explicitly.
// Precedents already in the monorepo:
//   - bin-agent-manager/pkg/dbhandler/agent.go:709 — MySQL JSON function + ?
//     argument + WHERE + PlaceholderFormat(squirrel.Question), the exact
//     combination used here.
//   - bin-rag-manager/pkg/dbhandler/document.go:355 — sq.Expr inside SetMap,
//     the Set()-position precedent.
//   - bin-conversation-manager/pkg/dbhandler/conversation.go:160-163,
//     bin-billing-manager/pkg/dbhandler/billing.go:154,
//     bin-schedule-manager/pkg/dbhandler/execution.go:360-369.
//
// These 15 sites have no executable coverage (SQLite, the unit-test database,
// implements none of these functions), so each one is pinned by a golden
// ToSql() string assertion in json_expr_golden_test.go. Factoring the three
// recurring shapes here means there are three SQL strings to review rather than
// twelve near-identical ones that could drift apart.

// exprJSONArrayAppend builds `json_array_append(<column>, '$', ?)`, appending a
// value to the end of a JSON array column.
func exprJSONArrayAppend(column string, value any) squirrel.Sqlizer {
	return squirrel.Expr("json_array_append("+column+", '$', ?)", value)
}

// exprJSONArrayRemoveByValue builds
// `json_remove(<column>, replace(json_search(<column>, 'one', ?), '"', ''))`,
// deleting the first array element equal to the given value.
//
// Note this form is not null-safe: if the value is absent, json_search returns
// NULL and json_remove yields NULL, blanking the column. That is the
// pre-migration behavior at these sites and is carried over unchanged — see
// exprJSONArrayRemoveByValueIfPresent for the guarded variant, which other
// sites already used.
func exprJSONArrayRemoveByValue(column string, value any) squirrel.Sqlizer {
	return squirrel.Expr(
		"json_remove("+column+", replace(json_search("+column+", 'one', ?), '\"', ''))",
		value,
	)
}

// exprJSONArrayRemoveByValueIfPresent builds the guarded delete-by-value form:
//
//	IF(json_search(<column>, 'one', ?) IS NOT NULL,
//	   json_remove(<column>, replace(json_search(<column>, 'one', ?), '"', '')),
//	   <column>)
//
// which leaves the column untouched when the value is absent. Takes the value
// twice because the expression references it in both the guard and the removal.
func exprJSONArrayRemoveByValueIfPresent(column string, value any) squirrel.Sqlizer {
	return squirrel.Expr(
		"IF(json_search("+column+", 'one', ?) IS NOT NULL, json_remove("+column+
			", replace(json_search("+column+", 'one', ?), '\"', '')), "+column+")",
		value, value,
	)
}
