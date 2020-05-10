package conferencehandler

//go:generate mockgen -destination ./mock_conferencehandler_conferencehandler.go -package conferencehandler gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler ConferenceHandler

import (
	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

// ConferenceHandler is interface for conference handle
type ConferenceHandler interface {
	// ari event handlers
	ARIStasisStart(cn *channel.Channel) error

	Start(cType conference.Type, c *call.Call) (*conference.Conference, error)

	Join(id, callID uuid.UUID) error
	Joined(id, callID uuid.UUID) error

	Leave(id, callID uuid.UUID) error
	Leaved(id, callID uuid.UUID) error
	Terminate(id uuid.UUID) error
}

// conferenceHandler structure for service handle
type conferenceHandler struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler
}

// Contexts of conference types
const (
	ContextConferenceEcho string = "conf-echo"
)

// NewConferHandler returns new service handler
func NewConferHandler(r requesthandler.RequestHandler, d dbhandler.DBHandler) ConferenceHandler {

	h := &conferenceHandler{
		reqHandler: r,
		db:         d,
	}

	return h
}

func (h *conferenceHandler) Join(id, callID uuid.UUID) error {
	return nil
}

func (h *conferenceHandler) Joined(id, callID uuid.UUID) error {
	return nil
}

func (h *conferenceHandler) leaveTypeEcho(c *call.Call) error {
	// cf := h.db.

	return nil
}
