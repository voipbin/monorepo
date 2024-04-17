package conferencecallhandler

import (
	"context"
	"encoding/json"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/models/service"
)

// ServiceStart is starting a new service conferencecall.
// it increases corresponded counter
func (h *conferencecallHandler) ServiceStart(
	ctx context.Context,
	conferenceID uuid.UUID,
	referenceType conferencecall.ReferenceType,
	referenceID uuid.UUID,
) (*service.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceStart",
		"conference_id":  conferenceID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	// get conference info
	cf, err := h.conferenceHandler.Get(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get conference info.")
	}

	// create conference call
	cc, err := h.Create(ctx, cf.CustomerID, conferenceID, referenceType, referenceID)
	if err != nil {
		log.Errorf("Could not create conferencecall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create conferencecall.")
	}
	log.WithField("conferencecall", cc).Debugf("Created conferencecall. conferencecall_id: %s", cc.ID)

	// create push actions for service start
	optJoin := fmaction.OptionFetchFlow{
		FlowID: cf.FlowID,
	}
	optString, err := json.Marshal(optJoin)
	if err != nil {
		log.Errorf("Could not marshal the conference join option. err: %v", err)
		return nil, errors.Wrap(err, "Could not marshal the conference join option.")
	}
	actions := []fmaction.Action{
		{
			ID:     h.utilHandler.UUIDCreate(),
			Type:   fmaction.TypeFetchFlow,
			Option: optString,
		},
	}

	res := &service.Service{
		ID:          cc.ID,
		Type:        service.TypeConferencecall,
		PushActions: actions,
	}

	return res, nil
}
