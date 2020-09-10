package servicehandler

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager/models/call"
	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/cmcall"
)

// CallCreate sends a request to call-manager
// to creating a conference.
// it returns created conference if it succeed.
func (h *serviceHandler) CallCreate(u *user.User, flowID uuid.UUID, source, destination call.Address) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":        u.ID,
		"username":    u.Username,
		"flow_id":     flowID,
		"source":      source,
		"destination": destination,
	})

	// parse source/destination
	addrSrc := cmcall.Address{
		Type:   cmcall.AddressType(source.Type),
		Target: source.Target,
		Name:   source.Name,
	}
	addrDest := cmcall.Address{
		Type:   cmcall.AddressType(destination.Type),
		Target: destination.Target,
		Name:   destination.Name,
	}

	// send request
	log.Debug("Creating a new call.")
	res, err := h.reqHandler.CallCallCreate(u.ID, flowID, addrSrc, addrDest)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}

	// create call.Call
	c := res.ConvertCall()

	return c, err
}
