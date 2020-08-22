package conferencehandler

//go:generate mockgen -destination ./mock_conferencehandler_conferencehandler.go -package conferencehandler gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler ConferenceHandler

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

// ConferenceHandler is interface for conference handle
type ConferenceHandler interface {
	// ari event handlers
	ARIStasisStart(cn *channel.Channel) error
	ARIChannelEnteredBridge(cn *channel.Channel, bridge *bridge.Bridge) error
	ARIChannelLeftBridge(cn *channel.Channel, br *bridge.Bridge) error

	Start(reqConf *conference.Conference, c *call.Call) (*conference.Conference, error)
	Join(conferenceID, callID uuid.UUID) error
	Leave(conferenceID, callID uuid.UUID) error
	Terminate(conferenceID uuid.UUID, reason string) error
}

// conferenceHandler structure for service handle
type conferenceHandler struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler
	cache      cachehandler.CacheHandler
}

// Contexts of conference types
const (
	contextConferenceEcho     string = "conf-echo"
	contextConferenceJoin     string = "conf-join"
	contextConferenceIncoming string = "conf-in"
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
func NewConferHandler(req requesthandler.RequestHandler, db dbhandler.DBHandler, cache cachehandler.CacheHandler) ConferenceHandler {

	h := &conferenceHandler{
		reqHandler: req,
		db:         db,
		cache:      cache,
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
