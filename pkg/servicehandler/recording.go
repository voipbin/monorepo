package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// RecordingGet returns downloadable url for recording
func (h *serviceHandler) RecordingGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cmrecording.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"customer_id": u.ID,
			"recording":   id,
		},
	)

	// get recording info from call-manager
	rec, err := h.reqHandler.CallV1RecordingGet(ctx, id)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	// check the recording ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != rec.CustomerID {
		log.Error("The user has no permission for this recording.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := rec.ConvertWebhookMessage()

	return res, nil
}

// RecordingGets sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) RecordingGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cmrecording.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = getCurTime()
	}

	tmp, err := h.reqHandler.CallV1RecordingGets(ctx, u.ID, size, token)
	if err != nil {
		log.Errorf("Could not get recordings from the call manager. err: %v", err)
		return nil, err
	}

	res := []*cmrecording.WebhookMessage{}
	for _, tmpRecord := range tmp {
		record := tmpRecord.ConvertWebhookMessage()
		res = append(res, record)
	}

	return res, nil
}
