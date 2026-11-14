package dbhandler

import (
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// mysqlDuplicateEntry is the MySQL and MariaDB error number of a duplicate key.
const mysqlDuplicateEntry = 1062

// isDuplicateKey tells whether err is a duplicate key error: MySQL and MariaDB error 1062, or the
// unique constraint error of the sqlite driver used by the tests. The sqlite driver is not
// imported here, so its error is recognized by its text.
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}

	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == mysqlDuplicateEntry
	}

	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// IsDuplicateKey tells whether err is a duplicate key error, matched by the database error number
// (MySQL and MariaDB 1062, found through errors.As) and, for the sqlite test driver, by its
// unique constraint text. It differs from the older text based IsErrDuplicate, which matches the
// message of the error only.
func IsDuplicateKey(err error) bool {
	return isDuplicateKey(err)
}
