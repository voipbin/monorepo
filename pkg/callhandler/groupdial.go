package callhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
)

// createGroupdial creates a new group dial.
func (h *callHandler) createGroupdial(
	ctx context.Context,
	customerID uuid.UUID,
	flowID uuid.UUID,
	masterCallID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
	earlyExecution bool,
	executeNextMasterOnHangup bool,
) (*groupdial.Groupdial, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "createGroupdial",
		"master_call_id": masterCallID,
	})

	mapDialDestination := map[commonaddress.Type]func(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]*commonaddress.Address, error){
		commonaddress.TypeEndpoint: h.getDestinationsAddressTypeEndpoint,
		commonaddress.TypeAgent:    h.getDestinationsAddressTypeAgent,
	}

	f, ok := mapDialDestination[destination.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported destination type")
	}

	dialDestinations, err := f(ctx, customerID, destination)
	if err != nil {
		log.Errorf("Could not get dial uris. err: %v", err)
		return nil, errors.Wrap(err, "Could not get dial uris.")
	}

	if len(dialDestinations) == 0 {
		log.Debugf("No dial destination found. len: %d", len(dialDestinations))
		return nil, fmt.Errorf("no dial destination found")
	}

	// currently, we support only ringall and hangup others.
	ringMethod := groupdial.RingMethodRingAll
	answerMethod := groupdial.AnswerMethodHangupOthers
	log.WithField("dial_destinations", dialDestinations).Debugf("Found dial destinations for group dial. destination_type: %s, ring_method: %s, answer_method: %s", destination.Type, ringMethod, answerMethod)

	callIDs := []uuid.UUID{}
	switch ringMethod {
	case groupdial.RingMethodRingAll:
		for _, dialDestination := range dialDestinations {
			tmp, err := h.CreateCallOutgoing(ctx, uuid.Nil, customerID, flowID, uuid.Nil, masterCallID, *source, *dialDestination, earlyExecution, executeNextMasterOnHangup)
			if err != nil {
				log.Errorf("Could not create an outgoing call. err: %v", err)
				continue
			}

			callIDs = append(callIDs, tmp.ID)
		}

	case groupdial.RingMethodLinear:
	default:
		log.Errorf("Unsupported ring method type. ring_method: %s", ringMethod)
		return nil, fmt.Errorf("unsupported ring method type")
	}

	// create group dial
	tmp := &groupdial.Groupdial{
		ID:         h.utilHandler.CreateUUID(),
		CustomerID: customerID,

		Destination:  destination,
		CallIDs:      callIDs,
		RingMethod:   ringMethod,
		AnswerMethod: answerMethod,
	}

	if errCreate := h.db.GroupdialCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create the group dial. err: %v", errCreate)
		return nil, errors.Wrap(errCreate, "Could not create the group dial.")
	}

	res, err := h.db.GroupdialGet(ctx, tmp.ID)
	if err != nil {
		log.Errorf("Could not get created group dial info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get created group dial info.")
	}
	h.notifyHandler.PublishEvent(ctx, groupdial.EventTypeGroupdialCreated, res)

	return res, nil
}

// getGroupdial returns a groupdial of the given id.
func (h *callHandler) getGroupdial(ctx context.Context, id uuid.UUID) (*groupdial.Groupdial, error) {
	return h.db.GroupdialGet(ctx, id)
}

// updateGroupdialAnswerCallID updates the answer call id.
func (h *callHandler) updateGroupdialAnswerCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) (*groupdial.Groupdial, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "updateGroupdialAnswerCallID",
		"groupdial_id": id,
		"call_id":      callID,
	})

	gd, err := h.getGroupdial(ctx, id)
	if err != nil {
		log.Errorf("Could not get group dial info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get group dial info.")
	}

	gd.AnswerCallID = callID
	if errUpdate := h.db.GroupdialUpdate(ctx, gd); errUpdate != nil {
		log.Errorf("Could not update the group dial info. err: %v", errUpdate)
		return nil, errors.Wrap(errUpdate, "Could not update the group dial info.")
	}

	res, err := h.db.GroupdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated group dial info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get updated group dial info.")
	}

	return res, nil
}

// getDestinationsAddressTypeAgent returns destinations for address type agent.
func (h *callHandler) getDestinationsAddressTypeAgent(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDestinationsAddressTypeAgent",
		"destination": destination,
	})

	// get agnet info
	agID := uuid.FromStringOrNil(destination.Target)
	if agID == uuid.Nil {
		log.Errorf("Could not parse the agent id. agent_id: %s", destination.Target)
		return nil, fmt.Errorf("could not parse the agent id")
	}

	ag, err := h.reqHandler.AgentV1AgentGet(ctx, agID)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get agnet info.")
	}

	res := []*commonaddress.Address{}
	for _, address := range ag.Addresses {
		// update address target name
		address.TargetName = destination.TargetName

		switch address.Type {
		case commonaddress.TypeTel, commonaddress.TypeSIP:
			res = append(res, &address)

		case commonaddress.TypeEndpoint:
			tmp, err := h.getDestinationsAddressTypeEndpoint(ctx, ag.CustomerID, &address)
			if err != nil {
				log.Errorf("Could not get destination address. err: %v", err)
				continue
			}
			res = append(res, tmp...)

		default:
			log.WithField("address", address).Errorf("Unsupported address type for agent outgoing. address_type: %s", address.Type)
		}
	}

	return res, nil
}

// getDestinationsAddressTypeEndpoint returns destinations for address type endpoint.
func (h *callHandler) getDestinationsAddressTypeEndpoint(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDestinationsAddressTypeEndpoint",
		"destination": destination,
	})

	e, err := h.reqHandler.RegistrarV1ExtensionGetByEndpoint(ctx, destination.Target)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get extension info.")
	}

	// check the customer id
	if customerID != e.CustomerID {
		log.Debugf("The customer id is different. customer_id: %s, extension_customer_id: %s", customerID, e.CustomerID)
		return nil, fmt.Errorf("the customer id is different")
	}

	// get contacts
	contacts, err := h.reqHandler.RegistrarV1ContactGets(ctx, destination.Target)
	if err != nil {
		log.Errorf("Could not get contacts info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get contacts info.")
	}

	res := []*commonaddress.Address{}
	for _, contact := range contacts {
		uri := strings.ReplaceAll(contact.URI, "^3B", ";")
		tmp := &commonaddress.Address{
			Type:       commonaddress.TypeSIP,
			TargetName: destination.TargetName,
			Target:     uri,
		}

		res = append(res, tmp)
	}

	return res, nil
}

// answerGroupdial handles the answered group dial.
func (h *callHandler) answerGroupdial(ctx context.Context, groupdialID uuid.UUID, answercallID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "answerGroupDial",
		"groupdial_id":   groupdialID,
		"answer_call_id": answercallID,
	})

	// get groupdial
	gd, err := h.getGroupdial(ctx, groupdialID)
	if err != nil {
		log.Errorf("Could not get group dial info. err: %v", err)
		return errors.Wrap(err, "Could not get group dial info.")
	}

	if gd.AnswerMethod != groupdial.AnswerMethodHangupOthers {
		log.Debugf("Unsupported answer method. answer_method: %s", gd.AnswerMethod)
		return fmt.Errorf("unsupported answer method")
	}

	// update answer call id
	tmp, err := h.updateGroupdialAnswerCallID(ctx, gd.ID, answercallID)
	if err != nil {
		log.Errorf("Could not update the answer call id. err: %v", err)
		return errors.Wrap(err, "Could not update the answer call id.")
	}

	for _, callID := range tmp.CallIDs {
		if callID == answercallID {
			continue
		}

		log.Debugf("Hanging up group dial calls. call_id: %s", callID)
		go func(id uuid.UUID) {
			_, _ = h.HangingUp(ctx, id, call.HangupReasonNormal)
		}(callID)
	}

	return nil
}
