package groupcallhandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/groupcall"
)

// Start the groupcall process
func (h *groupcallHandler) Start(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	ownerType groupcall.OwnerType,
	ownerID uuid.UUID,
	flowID uuid.UUID,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	masterCallID uuid.UUID,
	masterGroupcallID uuid.UUID,
	ringMethod groupcall.RingMethod,
	answerMethod groupcall.AnswerMethod,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "Start",
		"id":                  id,
		"customer_id":         customerID,
		"owner_type":          ownerType,
		"owner_id":            ownerID,
		"flow_id":             flowID,
		"source":              source,
		"destinations":        destinations,
		"master_call_id":      masterCallID,
		"master_groupcall_id": masterGroupcallID,
		"ring_method":         ringMethod,
		"answer_method":       answerMethod,
	})
	log.Debugf("Starting the groupcall service.")

	if len(destinations) == 0 {
		log.Errorf("Groupcall request has no valid destination. No destination. destination_len: %d", len(destinations))
		return nil, fmt.Errorf("have no destination")
	}

	if id == uuid.Nil {
		id = h.utilHandler.UUIDCreate()
		log.Debugf("The given groupcall start request has no id. Created a new id. id: %s", id)
	}

	if ringMethod == groupcall.RingMethodNone {
		ringMethod = groupcall.RingMethodRingAll
		log.Infof("The ring method is empty. Setting a default. ring_method: %s", ringMethod)
	}

	switch ringMethod {
	case groupcall.RingMethodRingAll:
		return h.startRingall(ctx, id, customerID, ownerType, ownerID, source, destinations, flowID, masterCallID, masterGroupcallID, answerMethod)

	case groupcall.RingMethodLinear:
		return h.startLinear(ctx, id, customerID, ownerType, ownerID, flowID, source, destinations, masterCallID, masterGroupcallID, answerMethod)

	default:
		log.Errorf("Unsupported ring method. ring_method: %s", ringMethod)
		return nil, fmt.Errorf("unsupported ring_method ")
	}
}

// startRingall starts the groupcall process for ringall ringmethod
func (h *groupcallHandler) startRingall(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	ownerType groupcall.OwnerType,
	ownerID uuid.UUID,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	flowID uuid.UUID,
	masterCallID uuid.UUID,
	masterGroupcallID uuid.UUID,
	answerMethod groupcall.AnswerMethod,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "startRingall",
		"id":                  id,
		"customer_id":         customerID,
		"owner_type":          ownerType,
		"owner_id":            ownerID,
		"flow_id":             flowID,
		"source":              source,
		"destinations":        destinations,
		"master_call_id":      masterCallID,
		"master_groupcall_id": masterGroupcallID,
		"answer_method":       answerMethod,
	})
	log.Debugf("Starting the groupcall service.")

	// get dial destinations
	groupcallIDs := []uuid.UUID{}
	groupcallDialDestinations := []*commonaddress.Address{}
	callIDs := []uuid.UUID{}
	callDialDestinations := []*commonaddress.Address{}
	for _, destination := range destinations {
		tmpID := h.utilHandler.UUIDCreate()

		if h.IsGroupcallTypeAddress(&destination) {
			groupcallIDs = append(groupcallIDs, tmpID)
			groupcallDialDestinations = append(groupcallDialDestinations, &destination)
		} else {
			callIDs = append(callIDs, tmpID)
			callDialDestinations = append(callDialDestinations, &destination)
		}
	}

	// create groupcall
	// we need to create groupcall earlier than the call. because if we create a call first, it is possible to hangup/answer the call before the create a groupcall
	// if that is happen, we will loose the groupcall control.
	res, err := h.Create(ctx, id, customerID, ownerType, ownerID, flowID, source, destinations, callIDs, groupcallIDs, masterCallID, masterGroupcallID, groupcall.RingMethodRingAll, answerMethod)
	if err != nil {
		log.Errorf("Could not create groupcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create groupcall.")
	}

	// create chained groupcalls
	for i, destination := range groupcallDialDestinations {

		go func(i int, destination *commonaddress.Address) {
			tmp, err := h.dialChainedGroupcall(ctx, groupcallIDs[i], customerID, flowID, source, destination, masterCallID, id)
			if err != nil {
				log.WithField("dial_destination", destination).Errorf("Could not create the chained groupcall info. err: %v", err)
				_, _ = h.HangupGroupcall(ctx, id)
				return
			}
			log.WithField("chained_groupcall", tmp).Debugf("Created chained groupcall info. chained_groupcall_id: %s", tmp.ID)
		}(i, destination)
	}

	// create chained calls
	for i, destination := range callDialDestinations {

		go func(i int, destination *commonaddress.Address) {
			// we don't allow to add the connect option for groupcall
			tmp, err := h.reqHandler.CallV1CallCreateWithID(ctx, callIDs[i], customerID, flowID, uuid.Nil, masterCallID, source, destination, id, false, false)
			if err != nil {
				log.WithField("dial_destination", destination).Errorf("Could not create a chained call. err: %v", err)
				_, _ = h.HangupCall(ctx, id)
				return
			}
			log.WithField("chained_call", tmp).Debugf("Created chained call info. chanined_call_id: %s", tmp.ID)
		}(i, destination)
	}

	return res, nil
}

// startLinear the groupcall process linear type
func (h *groupcallHandler) startLinear(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	ownerType groupcall.OwnerType,
	ownerID uuid.UUID,
	flowID uuid.UUID,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	masterCallID uuid.UUID,
	masterGroupcallID uuid.UUID,
	answerMethod groupcall.AnswerMethod,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "startLinear",
		"id":                  id,
		"customer_id":         customerID,
		"owner_type":          ownerType,
		"owner_id":            ownerID,
		"flow_id":             flowID,
		"source":              source,
		"destinations":        destinations,
		"master_call_id":      masterCallID,
		"master_groupcall_id": masterGroupcallID,
		"answer_method":       answerMethod,
	})
	log.Debugf("Starting the linear ring method groupcall service.")

	// get dialdestination
	dialDestination := destinations[0]

	// set groupcallIDs or callIDs
	tmpID := h.utilHandler.UUIDCreate()
	groupcallIDs := []uuid.UUID{}
	callIDs := []uuid.UUID{}

	if h.IsGroupcallTypeAddress(&dialDestination) {
		groupcallIDs = append(groupcallIDs, tmpID)
	} else {
		callIDs = append(callIDs, tmpID)
	}

	// create groupcall
	// we need to create groupcall earlier than the call. because if we create a call first, it is possible to hangup/answer the call before the create a groupcall
	// if that is happen, we will loose the groupcall control.
	res, err := h.Create(ctx, id, customerID, ownerType, ownerID, flowID, source, destinations, callIDs, groupcallIDs, masterCallID, masterGroupcallID, groupcall.RingMethodLinear, answerMethod)
	if err != nil {
		log.Errorf("Could not create groupcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create groupcall.")
	}

	go func() {
		// create chained groupcall/call
		if h.IsGroupcallTypeAddress(&dialDestination) {
			tmp, err := h.dialChainedGroupcall(ctx, tmpID, customerID, flowID, source, &dialDestination, masterCallID, id)
			if err != nil {
				log.Errorf("Could not create the chained groupcall info. err: %v", err)
				_, _ = h.HangupGroupcall(ctx, id)
				return
			}
			log.WithField("chained_groupcall", tmp).Debugf("Created chained groupcall info. chained_groupcall_id: %s", tmp.ID)
		} else {
			// we don't allow to add the connect option for groupcall
			tmp, err := h.reqHandler.CallV1CallCreateWithID(ctx, tmpID, customerID, flowID, uuid.Nil, masterCallID, source, &dialDestination, res.ID, false, false)
			if err != nil {
				log.Errorf("Could not create the chained call info. err: %v", err)
				_, _ = h.HangupCall(ctx, id)
				return
			}
			log.WithField("chained_call", tmp).Debugf("Created chained call info. chained_call_id: %s", tmp.ID)
		}
	}()

	return res, nil
}

// IsGroupcallTypeAddress returns true if the given destination is groupcall type address.
func (h *groupcallHandler) IsGroupcallTypeAddress(destination *commonaddress.Address) bool {
	switch destination.Type {
	case commonaddress.TypeAgent, commonaddress.TypeExtension:
		return true

	default:
		return false
	}
}

// getDialDestinationsAddressTypeAgent returns destinations for address type agent.
func (h *groupcallHandler) dialChainedGroupcall(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	flowID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
	masterCallID uuid.UUID,
	masterGroupcallID uuid.UUID,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "dialChaingedGroupcalls",
		"customer_id":         customerID,
		"flow_id":             flowID,
		"source":              source,
		"destinations":        destination,
		"master_call_id":      masterCallID,
		"master_groupcall_id": masterGroupcallID,
	})

	// get dial destinations and ring method
	var dialDestinations []commonaddress.Address
	var ringMethod groupcall.RingMethod
	var err error
	switch destination.Type {
	case commonaddress.TypeAgent:
		// get destinations
		dialDestinations, ringMethod, err = h.getDialDestinationsAddressAndRingMethodTypeAgent(ctx, customerID, destination)
		if err != nil {
			log.Errorf("Could not get destination addresss. err: %v", err)
			return nil, errors.Wrap(err, "could not get destination addresses")
		}

	case commonaddress.TypeExtension:
		// get destinations
		dialDestinations, err = h.getDialDestinationsAddressTypeExtension(ctx, customerID, destination)
		ringMethod = groupcall.RingMethodRingAll
		if err != nil {
			log.Errorf("Could not get destination address. err: %v", err)
			return nil, errors.Wrap(err, "could not get destination addresses")
		}

	default:
		log.Errorf("Unsupported address type. address_type: %s", destination.Type)
		return nil, fmt.Errorf("unsupported address type")
	}

	// create chained groupcall
	res, err := h.reqHandler.CallV1GroupcallCreate(ctx, id, customerID, flowID, *source, dialDestinations, masterCallID, masterGroupcallID, ringMethod, groupcall.AnswerMethodHangupOthers)
	if err != nil {
		log.Errorf("Could not create chained groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "could not create chained groupcall info")
	}
	log.WithField("chained_groupcall", res).Debugf("Created chained groupcall info. chained_groupcall_id: %s", res.ID)

	return res, nil
}
