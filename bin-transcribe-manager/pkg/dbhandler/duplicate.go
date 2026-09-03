package dbhandler

import (
	"errors"
	"strings"

	gomysql "github.com/go-sql-driver/mysql"
)

// mysqlErrDupEntry is MySQL's ER_DUP_ENTRY errno, raised on a UNIQUE/PK
// constraint violation.
const mysqlErrDupEntry = 1062

// isDuplicateKeyErr reports whether the given error is a duplicate-key/unique
// constraint violation. It covers MySQL (errno 1062, matched via the typed
// *mysql.MySQLError so an unrelated message containing "1062" cannot be
// misclassified) and sqlite (used by the test harness).
//
// Mirrors bin-agent-manager/pkg/dbhandler/agent_address.go's
// isDuplicateKeyErr.
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}

	var myErr *gomysql.MySQLError
	if errors.As(err, &myErr) && myErr.Number == mysqlErrDupEntry {
		return true
	}

	// sqlite (test harness) surfaces a textual error, not a typed code.
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return true
	}

	return false
}
