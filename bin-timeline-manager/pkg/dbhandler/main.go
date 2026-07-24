package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"errors"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-timeline-manager/models/correlation"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/models/peerevent"
	commonaddress "monorepo/bin-common-handler/models/address"
)

const clickhouseRetryInterval = 30 * time.Second

// ErrNotFound is returned when a requested record does not exist.
// Currently unused — timeline-manager has no single-row Get methods —
// but exported to keep the listenhandler errorResponse pattern consistent
// with the rest of the monorepo so future Get methods can wire ErrNotFound
// translation and have it route to 404 automatically.
var ErrNotFound = errors.New("record not found")

// EventRow represents a single event row for batch insert.
type EventRow struct {
	Timestamp time.Time
	EventType string
	Publisher string
	DataType  string
	Data      string
}

// DBHandler interface for database operations.
type DBHandler interface {
	EventBatchInsert(ctx context.Context, rows []EventRow) error
	EventList(ctx context.Context, publisher string, resourceID uuid.UUID, events []string, pageToken string, pageSize int) ([]*event.Event, error)
	AggregatedEventList(ctx context.Context, activeflowID string, pageToken string, pageSize int) ([]*event.Event, error)
	ResourceActiveflowIDGet(ctx context.Context, resourceID string) (string, error)
	ResourceExists(ctx context.Context, resourceID string) (bool, error)
	CorrelatedResourceList(ctx context.Context, activeflowID string, limit int) ([]*correlation.CorrelatedRow, error)
	PeerEventBatchInsert(ctx context.Context, rows []PeerEventRow) error
	PeerEventList(ctx context.Context, customerID uuid.UUID, addrs []commonaddress.Address, pageToken string, pageSize int) ([]*peerevent.PeerEvent, error)
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
