package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// conferencecallGet vaildates the customer's ownership and returns the conferencecall info.
func (h *serviceHandler) conferencecallGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cfconferencecall.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "conferencecallGet",
			"customer_id":       u.ID,
			"conferencecall_id": id,
		},
	)

	// send request
	tmp, err := h.reqHandler.ConferenceV1ConferencecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the conferencecall. err: %v", err)
		return nil, err
	}
	log.WithField("conferencecall", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The customer has no permission for this conference.")
		return nil, fmt.Errorf("customer has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferencecallGet vaildates the customer's ownership and returns the conferencecall info.
func (h *serviceHandler) ConferencecallGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cfconferencecall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id":       u.ID,
		"username":          u.Username,
		"conferencecall_id": id,
	})
	log.Debugf("Get conferencecall. conferencecall_id: %s", id)

	// get conference
	res, err := h.conferencecallGet(ctx, u, id)
	if err != nil {
		log.Infof("Could not get conference info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ConferencecallCreate is a service handler for conferencecall creating.
func (h *serviceHandler) ConferencecallCreate(ctx context.Context, u *cscustomer.Customer, conferenceID uuid.UUID, referenceType cfconferencecall.ReferenceType, referenceID uuid.UUID) (*cfconferencecall.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"customer_id":    u.ID,
			"username":       u.Username,
			"conference_id":  conferenceID,
			"reference_type": referenceType,
			"reference_id":   referenceID,
		},
	)
	log.Debugf("Creating a new conferencecall.")

	// get conference for ownership check
	_, err := h.conferenceGet(ctx, u, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.ConferenceV1ConferencecallCreate(ctx, conferenceID, referenceType, referenceID)
	if err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ConferencecallKick is a service handler for kick the conferencecall from the conference.
func (h *serviceHandler) ConferencecallKick(ctx context.Context, u *cscustomer.Customer, conferencecallID uuid.UUID) (*cfconferencecall.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"customer_id":       u.ID,
			"username":          u.Username,
			"conferencecall_id": conferencecallID,
		},
	)

	// get conference for ownership check
	_, err := h.conferencecallGet(ctx, u, conferencecallID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	// kick the conferencecall from the conference
	tmp, err := h.reqHandler.ConferenceV1ConferencecallKick(ctx, conferencecallID)
	if err != nil {
		log.Errorf("Could not kick the call from the conference. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
