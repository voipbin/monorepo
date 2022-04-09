package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {

	// outplan
	OutplanCreate(ctx context.Context, t *outplan.Outplan) error
	OutplanDelete(ctx context.Context, id uuid.UUID) error
	OutplanGet(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error)
	OutplanGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*outplan.Outplan, error)
	OutplanUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	OutplanUpdateActionInfo(ctx context.Context, id uuid.UUID, actions []fmaction.Action, source *cmaddress.Address, endHandle outplan.EndHandle) error
	OutplanUpdateDialInfo(ctx context.Context, id uuid.UUID, dialTimeout, tryInterval, maxTryCount0, maxTryCount1, maxTryCount2, maxTryCount3, maxTryCount4 int) error

	// campaign
	CampaignCreate(ctx context.Context, t *campaign.Campaign) error
	CampaignDelete(ctx context.Context, id uuid.UUID) error
	CampaignGet(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error)
	CampaignGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaign.Campaign, error)
	CampaignUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	CampaignUpdateResourceInfo(ctx context.Context, id, outplanID, outdialID, queueID uuid.UUID) error
	CampaignUpdateNextCampaignID(ctx context.Context, id, nextCampaignID uuid.UUID) error
	CampaignUpdateStatus(ctx context.Context, id uuid.UUID, status campaign.Status) error
	CampaignUpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) error

	// campaigncall
	CampaigncallCreate(ctx context.Context, t *campaigncall.Campaigncall) error
	CampaigncallGet(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error)
	CampaigncallGetsByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	CampaigncallGetsByCampaignIDAndStatus(ctx context.Context, campaignID uuid.UUID, status campaigncall.Status, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	CampaigncallUpdateStatus(ctx context.Context, id uuid.UUID, status campaigncall.Status) error
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
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
