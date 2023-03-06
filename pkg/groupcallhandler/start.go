package groupcallhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// Start the groupcall process
func (h *groupcallHandler) Start(
	ctx context.Context,
	customerID uuid.UUID,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	flowID uuid.UUID,
	masterCallID uuid.UUID,
	ringMethod groupcall.RingMethod,
	answerMethod groupcall.AnswerMethod,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"customer_id":    customerID,
		"source":         source,
		"destinations":   destinations,
		"flow_id":        flowID,
		"master_call_id": masterCallID,
		"ring_method":    ringMethod,
		"answer_method":  answerMethod,
	})
	log.Debugf("Starting the groupcall service.")

	// get dial destinations
	dialDestinations, err := h.getDialDestinations(ctx, customerID, destinations)
	if err != nil {
		log.Errorf("Could not get dial destinations. err: %v", err)
		return nil, errors.Wrap(err, "Could not get dial destinations.")
	}

	// create call id
	// generate call ids
	callIDs := []uuid.UUID{}
	for range dialDestinations {
		callID := h.utilHandler.CreateUUID()
		callIDs = append(callIDs, callID)
	}

	// create groupcall
	// we need to create groupcall earlier than the call. because if we create a call first, it is possible to hangup/answer the call before the create a groupcall
	// if that is happen, we will loose the groupcall control.
	res, err := h.Create(ctx, customerID, source, destinations, callIDs, ringMethod, answerMethod)
	if err != nil {
		log.Errorf("Could not create groupcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create groupcall.")
	}

	// create calls
	for i, dialDestination := range dialDestinations {
		// about the connect option. and because the groupcall is making the multiple outgoing calls, it is not possible to add the connect option.
		_, err := h.reqHandler.CallV1CallCreateWithID(ctx, callIDs[i], customerID, flowID, uuid.Nil, masterCallID, source, dialDestination, res.ID, false, false)
		if err != nil {
			// could not create a call, but we don't stop the call creating.
			log.WithField("dial_destination", dialDestination).Errorf("Could not create a call. err: %v", err)
		}
	}

	return res, nil
}

// getDialDestinations returns given destination's dial destinations.
func (h *groupcallHandler) getDialDestinations(ctx context.Context, customerID uuid.UUID, destinations []commonaddress.Address) ([]*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "getDialDestinations",
		"customer_id":  customerID,
		"destinations": destinations,
	})

	res := []*commonaddress.Address{}
	for _, destination := range destinations {
		var tmp []*commonaddress.Address
		var err error

		switch destination.Type {
		case commonaddress.TypeEndpoint:
			tmp, err = h.getDialDestinationsAddressTypeEndpoint(ctx, customerID, &destination)

		case commonaddress.TypeAgent:
			tmp, err = h.getDialDestinationsAddressTypeAgent(ctx, customerID, &destination)

		case commonaddress.TypeSIP, commonaddress.TypeTel:
			tmp, err = h.getDialDestinationsAddressBypass(ctx, &destination)

		default:
			log.Errorf("Unsupported destination type. destination_type: %s", destination.Type)
			return nil, fmt.Errorf("unsupported destinatio type. destination_type: %s", destination.Type)
		}
		if err != nil {
			log.WithField("destination", destination).Errorf("Could not get dial destination. err: %v", err)
			return nil, errors.Wrap(err, "Could not get dial destination.")
		}

		res = append(res, tmp...)
	}

	if len(res) == 0 {
		log.Errorf("No dial destination found. len: %d", len(res))
		return nil, fmt.Errorf("no dial destination found")
	}

	return res, nil
}

// getDialDestinationsAddressBypass returns destinations for address type endpoint.
func (h *groupcallHandler) getDialDestinationsAddressBypass(ctx context.Context, destination *commonaddress.Address) ([]*commonaddress.Address, error) {
	res := []*commonaddress.Address{
		destination,
	}

	return res, nil
}

// getDialDestinationsAddressTypeEndpoint returns destinations for address type endpoint.
func (h *groupcallHandler) getDialDestinationsAddressTypeEndpoint(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDialDestinationsAddressTypeEndpoint",
		"customer_id": customerID,
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
	log.WithField("contacts", contacts).Debugf("Found contacts. len: %d", len(contacts))

	res := []*commonaddress.Address{}
	for _, contact := range contacts {
		uri := strings.ReplaceAll(contact.URI, "^3B", ";")
		tmp := &commonaddress.Address{
			Type:       commonaddress.TypeSIP,
			TargetName: destination.TargetName, // update the target name to the destination's target name
			Target:     uri,
		}

		res = append(res, tmp)
	}

	return res, nil
}

// getDialDestinationsAddressTypeAgent returns destinations for address type agent.
func (h *groupcallHandler) getDialDestinationsAddressTypeAgent(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDialDestinationsAddressTypeAgent",
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

	// check the customer id
	if customerID != ag.CustomerID {
		log.Debugf("The customer id is different. customer_id: %s, agent_customer_id: %s", customerID, ag.CustomerID)
		return nil, fmt.Errorf("the customer id is different")
	}

	res := []*commonaddress.Address{}
	for _, address := range ag.Addresses {
		// update address target name
		address.TargetName = destination.TargetName

		switch address.Type {
		case commonaddress.TypeTel, commonaddress.TypeSIP:
			res = append(res, &address)

		case commonaddress.TypeEndpoint:
			tmp, err := h.getDialDestinationsAddressTypeEndpoint(ctx, ag.CustomerID, &address)
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
