package groupcallhandler

import (
	"context"
	"fmt"
	"strings"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/groupcall"
)

// dialNextDestination dials the next destination of the given groupcall.
func (h *groupcallHandler) dialNextDestination(ctx context.Context, gc *groupcall.Groupcall) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "dialNextDestination",
		"groupcall": gc,
	})

	// get next destination
	destination := gc.Destinations[gc.DialIndex+1]

	var res *groupcall.Groupcall
	var err error
	if h.IsGroupcallTypeAddress(&destination) {
		res, err = h.dialNextDestinationGroupcall(ctx, gc, &destination)
	} else {
		res, err = h.dialNextDestinationCall(ctx, gc, &destination)
	}
	if err != nil {
		log.Errorf("Could not dial the next destination. destination: %v", destination)
		return nil, errors.Wrap(err, "could not dial the next destination")
	}

	return res, nil
}

// dialNextDestinationGroupcall dials to next next destination for groupcall
func (h *groupcallHandler) dialNextDestinationGroupcall(ctx context.Context, gc *groupcall.Groupcall, destination *commonaddress.Address) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dialNextDestinationGroupcall",
		"groupcall":   gc,
		"destination": destination,
	})

	dialDestinations := []commonaddress.Address{}
	ringMethod := groupcall.RingMethodRingAll
	var err error
	switch destination.Type {
	case commonaddress.TypeAgent:
		dialDestinations, ringMethod, err = h.getDialDestinationsAddressAndRingMethodTypeAgent(ctx, gc.CustomerID, destination)
		if err != nil {
			log.Errorf("Could not get dial destination. err: %v", err)
			return nil, errors.Wrap(err, "could not get dial destination")
		}

	case commonaddress.TypeExtension:
		ringMethod = groupcall.RingMethodRingAll
		dialDestinations, err = h.getDialDestinationsAddressTypeExtension(ctx, gc.CustomerID, destination)
		if err != nil {
			log.Errorf("Could not get dial destinations. err: %v", err)
			return nil, errors.Wrap(err, "could not get dial destination")
		}

	default:
		log.Errorf("Unsupported address type. address_type: %s", destination.Type)
		return nil, fmt.Errorf("unsupported address type")
	}

	// update chained groupcall info
	id := h.utilHandler.UUIDCreate()
	groupcallIDs := append(gc.GroupcallIDs, id)
	res, err := h.UpdateGroupcallIDsAndGroupcallCountAndDialIndex(ctx, gc.ID, groupcallIDs, gc.GroupcallCount+1, gc.DialIndex+1)
	if err != nil {
		log.Errorf("Could not update the groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "could not update the groupcall info")
	}

	// create a chained groupcall
	go func() {
		tmp, err := h.reqHandler.CallV1GroupcallCreate(ctx, id, res.CustomerID, res.FlowID, *res.Source, dialDestinations, res.MasterCallID, res.ID, ringMethod, res.AnswerMethod)
		if err != nil {
			log.Errorf("Could not create a chained groupcall info. err: %v", err)
			_, _ = h.HangupGroupcall(ctx, gc.ID)
			return
		}
		log.WithField("chained_groupcall", tmp).Debugf("Created chained groupcall info. chained_groupcall_id: %s", tmp.ID)
	}()

	return res, nil
}

// dialNextDestinationCall dials the next destination for call.
func (h *groupcallHandler) dialNextDestinationCall(ctx context.Context, gc *groupcall.Groupcall, destination *commonaddress.Address) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dialNextDestinationGroupcall",
		"groupcall":   gc,
		"destination": destination,
	})

	// update chained groupcall info
	id := h.utilHandler.UUIDCreate()
	callIDs := append(gc.CallIDs, id)
	res, err := h.UpdateCallIDsAndCallCountAndDialIndex(ctx, gc.ID, callIDs, gc.GroupcallCount+1, gc.DialIndex+1)
	if err != nil {
		log.Errorf("Could not update the groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "could not update the groupcall info")
	}

	// create chained call
	go func() {
		// about the connect option. and because the groupcall is making the multiple outgoing calls, it is not possible to add the connect option.
		tmp, err := h.reqHandler.CallV1CallCreateWithID(ctx, id, gc.CustomerID, gc.FlowID, uuid.Nil, gc.MasterCallID, gc.Source, destination, res.ID, false, false)
		if err != nil {
			// could not create a call, but we don't stop the call creating.
			log.Errorf("Could not create a chained call. err: %v", err)
			_, _ = h.HangupCall(ctx, gc.ID)
			return
		}
		log.WithField("chained_call", tmp).Debugf("Created a chained call. chained_call_id: %s", tmp.ID)

	}()

	return res, nil
}

// getDialAddressesAndRingMethod returns the dial address and ring method of the given destination.
func (h *groupcallHandler) getDialAddressesAndRingMethod(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]commonaddress.Address, groupcall.RingMethod, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDialDestinationsAddressAndRingMethod",
		"customer_id": customerID,
		"destination": destination,
	})

	var resDialDestinations []commonaddress.Address
	resRingMethod := groupcall.RingMethodRingAll
	var err error
	switch destination.Type {
	case commonaddress.TypeAgent:
		resDialDestinations, resRingMethod, err = h.getDialDestinationsAddressAndRingMethodTypeAgent(ctx, customerID, destination)
		if err != nil {
			log.Errorf("Could not get dial destination. err: %v", err)
			return nil, groupcall.RingMethodNone, errors.Wrap(err, "could not get dial destination")
		}

	case commonaddress.TypeExtension:
		resRingMethod = groupcall.RingMethodRingAll
		resDialDestinations, err = h.getDialDestinationsAddressTypeExtension(ctx, customerID, destination)
		if err != nil {
			log.Errorf("Could not get dial destinations. err: %v", err)
			return nil, groupcall.RingMethodNone, errors.Wrap(err, "could not get dial destination")
		}

	case commonaddress.TypeTel, commonaddress.TypeSIP:
		resDialDestinations = []commonaddress.Address{*destination}
		resRingMethod = groupcall.RingMethodRingAll

	default:
		log.Errorf("Unsupported address type. address_type: %s", destination.Type)
		return nil, groupcall.RingMethodNone, fmt.Errorf("unsupported address type")
	}

	return resDialDestinations, resRingMethod, nil
}

// getDialDestinationsAddressTypeExtension returns destinations for address type extension.
func (h *groupcallHandler) getDialDestinationsAddressTypeExtension(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDialDestinationsAddressTypeExtension",
		"customer_id": customerID,
		"destination": destination,
	})

	contacts, err := h.reqHandler.RegistrarV1ContactGets(ctx, customerID, destination.TargetName)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get extension info.")
	}
	log.WithField("contacts", contacts).Debugf("Found contacts info. len: %d", len(contacts))

	res := []commonaddress.Address{}
	for _, contact := range contacts {
		uri := strings.ReplaceAll(contact.URI, "^3B", ";")
		tmp := commonaddress.Address{
			Type:       commonaddress.TypeSIP,
			TargetName: destination.TargetName, // update the target name to the destination's target name
			Target:     uri,
		}

		res = append(res, tmp)
	}

	return res, nil
}

// getDialDestinationsAddressTypeAgent returns destinations for address type agent.
func (h *groupcallHandler) getDialDestinationsAddressAndRingMethodTypeAgent(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]commonaddress.Address, groupcall.RingMethod, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDialDestinationsAddressAndRingMethodTypeAgent",
		"customer_id": customerID,
		"destination": destination,
	})

	// get agnet info
	agID := uuid.FromStringOrNil(destination.Target)
	if agID == uuid.Nil {
		log.Errorf("Could not parse the agent id. agent_id: %s", destination.Target)
		return nil, groupcall.RingMethodNone, fmt.Errorf("could not parse the agent id")
	}

	ag, err := h.reqHandler.AgentV1AgentGet(ctx, agID)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, groupcall.RingMethodNone, errors.Wrap(err, "Could not get agnet info.")
	}
	log.WithField("agent", ag).Debugf("Found agent info. agent_id: %s", ag.ID)

	// check the customer id
	if customerID != ag.CustomerID {
		log.Debugf("The customer id is different. customer_id: %s, agent_customer_id: %s", customerID, ag.CustomerID)
		return nil, groupcall.RingMethodNone, fmt.Errorf("the customer id is different")
	}

	ringMethod := groupcall.RingMethodRingAll
	if ag.RingMethod == agent.RingMethodLinear {
		ringMethod = groupcall.RingMethodLinear
	}

	return ag.Addresses, ringMethod, nil
}

// getAddressOwner returns owner's type and id.
func (h *groupcallHandler) getAddressOwner(ctx context.Context, customerID uuid.UUID, addr *commonaddress.Address) (groupcall.OwnerType, uuid.UUID, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getAddressOwner",
		"customer_id": customerID,
		"address":     addr,
	})

	var tmp *agent.Agent
	var err error

	if addr.Type == commonaddress.TypeAgent {
		id := uuid.FromStringOrNil(addr.Target)
		tmp, err = h.reqHandler.AgentV1AgentGet(ctx, id)
		if err != nil {
			log.Errorf("Could not get owner info. err: %v", err)
			return groupcall.OwnerTypeNone, uuid.Nil, err
		}
	} else {
		tmp, err = h.reqHandler.AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, customerID, *addr)
		if err != nil {
			log.Errorf("Could not get agent info. err: %v", err)
			return groupcall.OwnerTypeNone, uuid.Nil, nil
		}
	}

	if tmp == nil {
		return groupcall.OwnerTypeNone, uuid.Nil, nil
	}

	if tmp.CustomerID != customerID {
		log.Errorf("The customer id is not valid.")
		return groupcall.OwnerTypeNone, uuid.Nil, err
	}

	return groupcall.OwnerTypeAgent, tmp.ID, nil
}
