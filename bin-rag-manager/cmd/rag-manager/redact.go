package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
)

// reURIUserinfo matches the userinfo part of a URI in a message: scheme://userinfo@
// It runs to the last '@' before the next unescaped double quote (the url.Error text quotes the URI
// with %q, so a quote inside it appears as \"), so a userinfo that the URL parser rejected (spaces,
// quotes, control characters) is masked as well.
var reURIUserinfo = regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.\-]*://)(?:\\.|[^"\\])*@`)

// redactDSNError returns an error whose text has the userinfo of every URI masked.
// The PostgreSQL DSN carries the password, and the parse error of a malformed DSN echoes it back.
func redactDSNError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(reURIUserinfo.ReplaceAllString(err.Error(), "${1}***@"))
}

// openPostgres opens the PostgreSQL connection and pings it. The returned errors never carry the DSN userinfo.
func openPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("could not connect to PostgreSQL: %w", redactDSNError(err))
	}

	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("could not ping PostgreSQL: %w", redactDSNError(err))
	}

	return db, nil
}
