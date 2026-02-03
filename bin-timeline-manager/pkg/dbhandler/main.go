package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-timeline-manager/models/event"
)

const clickhouseRetryInterval = 30 * time.Second

// DBHandler interface for database operations.
type DBHandler interface {
	EventList(ctx context.Context, publisher string, resourceID uuid.UUID, events []string, pageToken string, pageSize int) ([]*event.Event, error)
	WaitForConnection(ctx context.Context) error
}

type dbHandler struct {
	address  string
	database string
	conn     clickhouse.Conn
}

// NewHandler creates a new DBHandler.
func NewHandler(address, database string) DBHandler {
	h := &dbHandler{
		address:  address,
		database: database,
	}

	go h.connectionLoop()

	return h
}

// WaitForConnection waits for the ClickHouse connection to be established.
func (h *dbHandler) WaitForConnection(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if h.conn != nil {
				return nil
			}
		}
	}
}

func (h *dbHandler) connectionLoop() {
	log := logrus.WithFields(logrus.Fields{
		"func":    "connectionLoop",
		"address": h.address,
	})

	for {
		if h.conn != nil {
			time.Sleep(clickhouseRetryInterval)
			continue
		}

		conn, err := clickhouse.Open(&clickhouse.Options{
			Addr: []string{h.address},
			Auth: clickhouse.Auth{
				Database: h.database,
			},
			Settings: clickhouse.Settings{
				"max_execution_time": 60,
			},
			DialTimeout: 5 * time.Second,
		})
		if err != nil {
			log.Errorf("Could not connect to ClickHouse, retrying in %v. err: %v", clickhouseRetryInterval, err)
			time.Sleep(clickhouseRetryInterval)
			continue
		}

		if err := conn.Ping(context.Background()); err != nil {
			log.Errorf("Could not ping ClickHouse, retrying in %v. err: %v", clickhouseRetryInterval, err)
			time.Sleep(clickhouseRetryInterval)
			continue
		}

		log.Info("Successfully connected to ClickHouse")
		h.conn = conn
		time.Sleep(clickhouseRetryInterval)
	}
}
