package dbhandler

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	mysql_driver "github.com/go-sql-driver/mysql"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-contact-manager/models/interaction"
)

const (
	interactionTable = "contact_interactions"
)

// InteractionCreate inserts an Interaction row into contact_interactions.
// It is idempotent: a duplicate-key error (MySQL errno 1062) is silently
// ignored so the caller does not need to guard against at-least-once delivery.
func (h *handler) InteractionCreate(ctx context.Context, i *interaction.Interaction) error {
	fields, err := commondatabasehandler.PrepareFields(i)
	if err != nil {
		return fmt.Errorf("could not prepare fields. InteractionCreate. err: %v", err)
	}

	query, args, err := sq.Insert(interactionTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. InteractionCreate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		// Idempotent: ignore duplicate-key errors so at-least-once event
		// delivery (e.g. RabbitMQ requeue on crash) does not surface errors.
		// MySQL errno 1062 in production; SQLite UNIQUE constraint in tests.
		if me, ok := err.(*mysql_driver.MySQLError); ok && me.Number == 1062 {
			return nil
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil
		}
		return fmt.Errorf("could not create interaction. InteractionCreate. err: %v", err)
	}

	return nil
}
