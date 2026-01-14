package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/models/outdialtargetcall"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
)

// DBHandler interface for outdial_manager database handle
type DBHandler interface {
	// outdial
	OutdialCreate(ctx context.Context, f *outdial.Outdial) error
	OutdialDelete(ctx context.Context, id uuid.UUID) error
	OutdialGet(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error)
	OutdialGets(ctx context.Context, token string, size uint64, filters map[outdial.Field]any) ([]*outdial.Outdial, error)
	OutdialUpdate(ctx context.Context, id uuid.UUID, fields map[outdial.Field]any) error

	// outdialtarget
	OutdialTargetCreate(ctx context.Context, t *outdialtarget.OutdialTarget) error
	OutdialTargetDelete(ctx context.Context, id uuid.UUID) error
	OutdialTargetGet(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error)
	OutdialTargetGets(ctx context.Context, token string, size uint64, filters map[outdialtarget.Field]any) ([]*outdialtarget.OutdialTarget, error)
	OutdialTargetUpdate(ctx context.Context, id uuid.UUID, fields map[outdialtarget.Field]any) error
	OutdialTargetGetAvailable(
		ctx context.Context,
		outdialID uuid.UUID,
		tryCount0 int,
		tryCount1 int,
		tryCount2 int,
		tryCount3 int,
		tryCount4 int,
		limit uint64,
	) ([]*outdialtarget.OutdialTarget, error)
	OutdialTargetUpdateProgressing(ctx context.Context, id uuid.UUID, destinationIndex int) error

	// outdialtargetcall
	OutdialTargetCallCreate(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error
	OutdialTargetCallGet(ctx context.Context, id uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error)
	OutdialTargetCallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error)
	OutdialTargetCallGetByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error)
	OutdialTargetCallGets(ctx context.Context, token string, size uint64, filters map[outdialtargetcall.Field]any) ([]*outdialtargetcall.OutdialTargetCall, error)
	OutdialTargetCallUpdate(ctx context.Context, id uuid.UUID, fields map[outdialtargetcall.Field]any) error
}

// handler database handler
type handler struct {
	db          *sql.DB
	cache       cachehandler.CacheHandler
	utilHandler utilhandler.UtilHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
)

// list of default values
var (
	DefaultTimeStamp = commondatabasehandler.DefaultTimeStamp
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		db:          db,
		cache:       cache,
		utilHandler: utilhandler.NewUtilHandler(),
	}
	return h
}

// parseDestination parses a JSON string into an Address pointer
func parseDestination(jsonStr string, dest **commonaddress.Address) error {
	if jsonStr == "" {
		return nil
	}
	*dest = &commonaddress.Address{}
	return json.Unmarshal([]byte(jsonStr), *dest)
}

// GetCurTime returns the current time in the database format.
func GetCurTime() string {
	return utilhandler.NewUtilHandler().TimeGetCurTime()
}
