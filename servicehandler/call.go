package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/cmcall"
)

// CallCreate sends a request to call-manager
// to creating a call.
// it returns created call info if it succeed.
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
	res, err := h.reqHandler.CMCallCreate(u.ID, flowID, addrSrc, addrDest)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}

	// create call.Call
	c := res.ConvertCall()

	return c, err
}

// CallGet sends a request to call-manager
// to getting a call.
// it returns call if it succeed.
func (h *serviceHandler) CallGet(u *user.User, callID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"call_id":  callID,
	})

	// send request
	c, err := h.reqHandler.CMCallGet(callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	if u.Permission != user.PermissionAdmin && u.ID != c.UserID {
		log.Info("The user has no permission for this call.")
		return nil, fmt.Errorf("user has no permission")
	}

	// convert
	res := c.ConvertCall()

	return res, nil
}

// CallGets sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) CallGets(u *user.User, size uint64, token string) ([]*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    token,
	})

	if token == "" {
		token = getCurTime()
	}

	// get calls
	tmps, err := h.reqHandler.CMCallGets(u.ID, token, size)
	if err != nil {
		log.Infof("Could not get calls info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*call.Call{}
	for _, tmp := range tmps {
		c := tmp.ConvertCall()
		res = append(res, c)
	}

	return res, nil
}

// CallDelete sends a request to call-manager
// to hangup the call.
// it returns call if it succeed.
func (h *serviceHandler) CallDelete(u *user.User, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"call_id":  callID,
	})

	// todo need to check the call's ownership
	c, err := h.reqHandler.CMCallGet(callID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return err
	}

	// check call's ownership
	if u.Permission != user.PermissionAdmin && u.ID != c.UserID {
		log.Info("The user has no permission for this call.")
		return fmt.Errorf("user has no permission")
	}

	// send request
	if err := h.reqHandler.CMCallDelete(callID); err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	return nil
}
