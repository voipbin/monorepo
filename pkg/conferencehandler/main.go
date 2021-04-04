package conferencehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package conferencehandler -destination ./mock_conferencehandler_conferencehandler.go -source main.go -build_flags=-mod=mod

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

// ConferenceHandler is interface for conference handle
type ConferenceHandler interface {
	// ari event handlers
	ARIStasisStart(cn *channel.Channel, data map[string]interface{}) error
	ARIChannelEnteredBridge(cn *channel.Channel, bridge *bridge.Bridge) error
	ARIChannelLeftBridge(cn *channel.Channel, br *bridge.Bridge) error

	Destroy(id uuid.UUID) error
	Start(reqConf *conference.Conference) (*conference.Conference, error)
	Join(conferenceID, callID uuid.UUID) error
	Leave(conferenceID, callID uuid.UUID) error
	Terminate(conferenceID uuid.UUID) error
}

// conferenceHandler structure for service handle
type conferenceHandler struct {
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
	cache         cachehandler.CacheHandler
}

// Contexts of conference types
const (
	contextConferenceJoin     string = "conf-join"
	contextConferenceIncoming string = "conf-in"
)

// List of default values
const (
	defaultDialTimeout = 60                           // default outgoing dial timeout
	defaultTimeStamp   = "9999-01-01 00:00:00.000000" // default timestamp
)

// conference termination type
type termType string

const (
	termTypeNone         termType = "none"
	termTypeTerminatable termType = "terminatable" // the conference is terminatable
	termTypeDestroyable  termType = "destroyable"  // the conference is destroyable
)

var (
	metricsNamespace = "call_manager"

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

	promConferenceLeaveTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "conference_leave_total",
			Help:      "Total number of leaved calls from the conference with type.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promConferenceCreateTotal,
		promConferenceCloseTotal,
		promConferenceJoinTotal,
		promConferenceLeaveTotal,
	)
}

// NewConferHandler returns new service handler
func NewConferHandler(req requesthandler.RequestHandler, notify notifyhandler.NotifyHandler, db dbhandler.DBHandler, cache cachehandler.CacheHandler) ConferenceHandler {

	h := &conferenceHandler{
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,
		cache:         cache,
	}

	return h
}

func (h *conferenceHandler) leaveTypeEcho(c *call.Call) error {
	return nil
}

// generateBridgeName generates the bridge name for conference
// all of conference created bridge must use this function for bridge's name.
// join: true if the bridge is for joining to the other conference
func generateBridgeName(conferenceType conference.Type, conferenceID uuid.UUID, join bool) string {
	res := fmt.Sprintf("conference_type=%s,conference_id=%s,join=%t", conferenceType, conferenceID.String(), join)

	return res
}

// isContextConf returns true if
func isContextConf(contextType string) bool {
	tmp := strings.Split(contextType, "-")[0]
	if tmp == "conf" {
		return true
	}

	return false
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
