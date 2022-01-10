package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// CallCreate sends a request to call-manager
// to creating a call.
// it returns created call info if it succeed.
func (h *serviceHandler) CallCreate(u *user.User, flowID uuid.UUID, source, destination *address.Address) (*cmcall.Event, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"user":        u.ID,
		"username":    u.Username,
		"flow_id":     flowID,
		"source":      source,
		"destination": destination,
	})

	// parse source/destination
	addrSrc := &cmaddress.Address{
		Type:   cmaddress.Type(source.Type),
		Target: source.Target,
		Name:   source.Name,
	}
	addrDest := &cmaddress.Address{
		Type:   cmaddress.Type(destination.Type),
		Target: destination.Target,
		Name:   destination.Name,
	}

	// send request
	log.Debug("Creating a new call.")
	tmp, err := h.reqHandler.CMV1CallCreate(ctx, u.ID, flowID, addrSrc, addrDest)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertEvent()

	return res, err
}

// CallGet sends a request to call-manager
// to getting a call.
// it returns call if it succeed.
func (h *serviceHandler) CallGet(u *user.User, callID uuid.UUID) (*cmcall.Event, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"call_id":  callID,
	})

	// send request
	c, err := h.reqHandler.CMV1CallGet(ctx, callID)
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
	res := c.ConvertEvent()

	return res, nil
}

// CallGets sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) CallGets(u *user.User, size uint64, token string) ([]*cmcall.Event, error) {
	ctx := context.Background()
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
	tmps, err := h.reqHandler.CMV1CallGets(ctx, u.ID, token, size)
	if err != nil {
		log.Infof("Could not get calls info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*cmcall.Event{}
	for _, tmp := range tmps {
		c := tmp.ConvertEvent()
		res = append(res, c)
	}

	return res, nil
}

// CallDelete sends a request to call-manager
// to hangup the call.
// it returns call if it succeed.
func (h *serviceHandler) CallDelete(u *user.User, callID uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"call_id":  callID,
	})

	// todo need to check the call's ownership
	c, err := h.reqHandler.CMV1CallGet(ctx, callID)
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
	if _, err := h.reqHandler.CMV1CallHangup(ctx, callID); err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	return nil
}
