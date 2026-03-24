package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"fmt"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-direct-manager/models/direct"
)

// DBHandler interface
type DBHandler interface {
	DirectCreate(ctx context.Context, d *direct.Direct) error
	DirectGet(ctx context.Context, id uuid.UUID) (*direct.Direct, error)
	DirectGetByHash(ctx context.Context, hash string) (*direct.Direct, error)
	DirectGets(ctx context.Context, size uint64, token string, filters map[direct.Field]any) ([]*direct.Direct, error)
	DirectDelete(ctx context.Context, id uuid.UUID) error
	DirectUpdate(ctx context.Context, id uuid.UUID, fields map[direct.Field]any) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
}

// handler errors
var (
	ErrNotFound = fmt.Errorf("record not found")
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
	}
	return h
}
