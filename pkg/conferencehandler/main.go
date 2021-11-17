package conferencehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package conferencehandler -destination ./mock_conferencehandler_conferencehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	conference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"
)

// ConferenceHandler is interface for conference handle
type ConferenceHandler interface {
	Create(
		ctx context.Context,
		conferenceType conference.Type,
		userID uint64,
		name string,
		detail string,
		timeout int,
		webhookURI string,
		preActions []action.Action,
		postActions []action.Action,
	) (*conference.Conference, error)
	Get(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	Gets(ctx context.Context, userID uint64, confType conference.Type, size uint64, token string) ([]*conference.Conference, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		timeout int,
		webhookURI string,
		preActions []action.Action,
		postActions []action.Action,
	) (*conference.Conference, error)

	Join(ctx context.Context, conferenceID, callID uuid.UUID) error
	Joined(ctx context.Context, conferenceID, callID uuid.UUID) error
	Leave(ctx context.Context, id, callID uuid.UUID) error
	Leaved(ctx context.Context, id uuid.UUID, callID uuid.UUID) error
	Terminate(ctx context.Context, id uuid.UUID) error
}

// conferenceHandler structure for service handle
type conferenceHandler struct {
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
	cache         cachehandler.CacheHandler
}

// List of default values
const (
	defaultDialTimeout = 60                           //nolint:deadcode,varcheck // default outgoing dial timeout
	defaultTimeStamp   = "9999-01-01 00:00:00.000000" // default timestamp
)

var (
	metricsNamespace = "conference_manager"

	promConferenceCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "conference_create_total",
			Help:      "Total number of created conference with type.",
		},
		[]string{"type"},
	)

	promConferenceCloseTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "conference_close_total",
			Help:      "Total number of closed conference type.",
		},
		[]string{"type"},
	)

	promConferenceJoinTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "conference_join_total",
			Help:      "Total number of joined calls to the conference with type.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promConferenceCreateTotal,
		promConferenceCloseTotal,
		promConferenceJoinTotal,
	)
}

// NewConferenceHandler returns new service handler
func NewConferenceHandler(req requesthandler.RequestHandler, notify notifyhandler.NotifyHandler, db dbhandler.DBHandler, cache cachehandler.CacheHandler) ConferenceHandler {

	h := &conferenceHandler{
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,
		cache:         cache,
	}

	return h
}

// // generateBridgeName generates the bridge name for conference
// // all of conference created bridge must use this function for bridge's name.
// func generateBridgeName(referenceType bridge.ReferenceType, conferenceID uuid.UUID) string {
// 	res := fmt.Sprintf("reference_type=%s,reference_id=%s", referenceType, conferenceID.String())

// 	return res
// }

// isContextConf returns true if
//nolint:unused,deadcode // this is ok
func isContextConf(contextType string) bool {
	tmp := strings.Split(contextType, "-")[0]

	return tmp == "conf"
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
