package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
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
	OutplanUpdateDialInfo(
		ctx context.Context,
		id uuid.UUID,
		source *commonaddress.Address,
		dialTimeout int,
		tryInterval int,
		maxTryCount0 int,
		maxTryCount1 int,
		maxTryCount2 int,
		maxTryCount3 int,
		maxTryCount4 int,
	) error

	// campaign
	CampaignCreate(ctx context.Context, t *campaign.Campaign) error
	CampaignDelete(ctx context.Context, id uuid.UUID) error
	CampaignGet(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error)
	CampaignGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaign.Campaign, error)
	CampaignUpdateBasicInfo(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		campaignType campaign.Type,
		serviceLevel int,
		endHandle campaign.EndHandle,
	) error
	CampaignUpdateResourceInfo(ctx context.Context, id, outplanID, outdialID, queueID, nextCampaignID uuid.UUID) error
	CampaignUpdateNextCampaignID(ctx context.Context, id, nextCampaignID uuid.UUID) error
	CampaignUpdateStatus(ctx context.Context, id uuid.UUID, status campaign.Status) error
	CampaignUpdateStatusAndExecute(ctx context.Context, id uuid.UUID, status campaign.Status, execute campaign.Execute) error
	CampaignUpdateExecute(ctx context.Context, id uuid.UUID, execute campaign.Execute) error
	CampaignUpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) error
	CampaignUpdateEndHandle(ctx context.Context, id uuid.UUID, endHandle campaign.EndHandle) error
	CampaignUpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) error
	CampaignUpdateType(ctx context.Context, id uuid.UUID, campaignType campaign.Type) error

	// campaigncall
	CampaigncallCreate(ctx context.Context, t *campaigncall.Campaigncall) error
	CampaigncallDelete(ctx context.Context, id uuid.UUID) error
	CampaigncallGet(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error)
	CampaigncallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*campaigncall.Campaigncall, error)
	CampaigncallGetByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*campaigncall.Campaigncall, error)
	CampaigncallGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	CampaigncallGetsByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	CampaigncallGetsByCampaignIDAndStatus(ctx context.Context, campaignID uuid.UUID, status campaigncall.Status, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	CampaigncallGetsOngoingByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error)
	CampaigncallUpdateStatus(ctx context.Context, id uuid.UUID, status campaigncall.Status) error
	CampaigncallUpdateStatusAndResult(ctx context.Context, id uuid.UUID, status campaigncall.Status, result campaigncall.Result) error
}

// handler database handler
type handler struct {
	util  utilhandler.UtilHandler
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
		util:  utilhandler.NewUtilHandler(),
		db:    db,
		cache: cache,
	}
	return h
}

// // GetCurTime return current utc time string
// func TimeGetCurTime() string {
// 	now := time.Now().UTC().String()
// 	res := strings.TrimSuffix(now, " +0000 UTC")

// 	return res
// }

// GetCurTimeAdd return current utc time string
func GetCurTimeAdd(d time.Duration) string {
	now := time.Now().Add(d).UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
