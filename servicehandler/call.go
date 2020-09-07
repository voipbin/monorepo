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
func (h *servicHandler) CallCreate(u *user.User, flowID uuid.UUID, source, destination string) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":        u.ID,
		"username":    u.Username,
		"flow_id":     flowID,
		"source":      source,
		"destination": destination,
	})

	// parse source/destination
	addrSrc := cmcall.Address{
		Type:   cmcall.AddressTypeSIP,
		Target: source,
	}
	addrDest := cmcall.Address{
		Type:   cmcall.AddressTypeSIP,
		Target: destination,
	}
	log = log.WithFields(logrus.Fields{
		"source_address":      addrSrc,
		"destination_address": addrDest,
	})

	// send request
	log.Debug("Creating a new call.")
	res, err := h.reqHandler.CallCallCreate(u.ID, flowID, addrSrc, addrDest)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}

	c := &call.Call{
		ID:     res.ID,
		UserID: res.UserID,
		FlowID: res.FlowID,
		ConfID: res.ConfID,
		Type:   call.Type(res.Type),

		Source: call.Address{
			Type:   call.AddressType(res.Source.Type),
			Name:   res.Source.Name,
			Target: res.Source.Target,
		},
		Destination: call.Address{
			Type:   call.AddressType(res.Destination.Type),
			Name:   res.Destination.Name,
			Target: res.Destination.Target,
		},

		Status: call.Status(res.Status),

		Direction:    call.Direction(res.Direction),
		HangupBy:     call.HangupBy(res.HangupBy),
		HangupReason: call.HangupReason(res.HangupReason),

		TMCreate: res.TMCreate,
		TMUpdate: res.TMUpdate,

		TMProgressing: res.TMProgressing,
		TMRinging:     res.TMRinging,
		TMHangup:      res.TMHangup,
	}

	return c, err
}
