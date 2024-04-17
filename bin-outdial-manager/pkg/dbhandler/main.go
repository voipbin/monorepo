package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/models/outdialtargetcall"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	// outdial
	OutdialCreate(ctx context.Context, f *outdial.Outdial) error
	OutdialDelete(ctx context.Context, id uuid.UUID) error
	OutdialGet(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error)
	OutdialGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*outdial.Outdial, error)
	OutdialUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	OutdialUpdateCampaignID(ctx context.Context, id, campaignID uuid.UUID) error
	OutdialUpdateData(ctx context.Context, id uuid.UUID, data string) error

	// outdialtarget
	OutdialTargetCreate(ctx context.Context, t *outdialtarget.OutdialTarget) error
	OutdialTargetDelete(ctx context.Context, id uuid.UUID) error
	OutdialTargetGet(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error)
	OutdialTargetGetsByOutdialID(ctx context.Context, outdialID uuid.UUID, token string, limit uint64) ([]*outdialtarget.OutdialTarget, error)
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
	OutdialTargetUpdateDestinations(
		ctx context.Context,
		id uuid.UUID,
		destination0 *commonaddress.Address,
		destination1 *commonaddress.Address,
		destination2 *commonaddress.Address,
		destination3 *commonaddress.Address,
		destination4 *commonaddress.Address,
	) error
	OutdialTargetUpdateStatus(ctx context.Context, id uuid.UUID, status outdialtarget.Status) error
	OutdialTargetUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	OutdialTargetUpdateData(ctx context.Context, id uuid.UUID, data string) error
	OutdialTargetUpdateProgressing(ctx context.Context, id uuid.UUID, destinationIndex int) error

	// outdialtargetcall
	OutdialTargetCallCreate(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error
	OutdialTargetCallGetsByOutdialIDAndStatus(ctx context.Context, outdialID uuid.UUID, status outdialtargetcall.Status) ([]*outdialtargetcall.OutdialTargetCall, error)
	OutdialTargetCallGetsByCampaignIDAndStatus(ctx context.Context, outdialID uuid.UUID, status outdialtargetcall.Status) ([]*outdialtargetcall.OutdialTargetCall, error)
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("Record not found")
)

// list of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:000"
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		db:    db,
		cache: cache,
	}
	return h
}

// GetCurTime return current utc time string
func GetCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// GetCurTimeAdd return current utc time string
func GetCurTimeAdd(d time.Duration) string {
	now := time.Now().Add(d).UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
