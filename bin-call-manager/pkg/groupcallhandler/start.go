package groupcallhandler

import (
	"context"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

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
		return h.startRingall(ctx, id, customerID, source, destinations, flowID, masterCallID, masterGroupcallID, answerMethod)

	case groupcall.RingMethodLinear:
		return h.startLinear(ctx, id, customerID, flowID, source, destinations, masterCallID, masterGroupcallID, answerMethod)

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
		"flow_id":             flowID,
		"source":              source,
		"destinations":        destinations,
		"master_call_id":      masterCallID,
		"master_groupcall_id": masterGroupcallID,
		"answer_method":       answerMethod,
	})
	log.Debugf("Starting the groupcall service.")

	if len(destinations) == 1 {
		// single destination
		res, err := h.startWithDestination(ctx, id, customerID, flowID, source, &destinations[0], masterCallID, masterGroupcallID, groupcall.RingMethodRingAll, answerMethod)
		if err != nil {
			log.Errorf("Could not start the groupcall with destination. err: %v", err)
			return nil, errors.Wrap(err, "could not start the groupcall with destination")
		}
		return res, nil
	}

	// generate call/groupcall ids and mapping the each destination
	mapGroupcalls := map[uuid.UUID]*commonaddress.Address{}
	mapCalls := map[uuid.UUID]*commonaddress.Address{}
	for _, destination := range destinations {
		tmpID := h.utilHandler.UUIDCreate()
		if h.IsGroupcallTypeAddress(&destination) {
			mapGroupcalls[tmpID] = &destination
		} else {
			mapCalls[tmpID] = &destination
		}
	}

	// create groupcall
	// we need to create groupcall earlier than the call. because if we create a call first, it is possible to hangup/answer the call before the create a groupcall
	// if that is happen, we will loose the groupcall control.
	callIDs := getKeys(mapCalls)
	groupcallIDs := getKeys(mapGroupcalls)
	res, err := h.Create(ctx, id, customerID, commonidentity.OwnerTypeNone, uuid.Nil, flowID, source, destinations, callIDs, groupcallIDs, masterCallID, masterGroupcallID, groupcall.RingMethodRingAll, answerMethod)
	if err != nil {
		log.Errorf("Could not create groupcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create groupcall.")
	}

	// create chained groupcalls
	for chainedGroupcallID, destination := range mapGroupcalls {
		go func(groupcallID uuid.UUID, destination *commonaddress.Address) {
			// create subgroupcalls
			tmp, err := h.startWithDestination(ctx, groupcallID, customerID, flowID, source, destination, masterCallID, id, groupcall.RingMethodRingAll, answerMethod)
			if err != nil {
				log.WithField("dial_destination", destination).Errorf("Could not create the chained groupcall info. err: %v", err)
				_, _ = h.HangupGroupcall(ctx, id)
				return
			}
			log.WithField("chained_groupcall", tmp).Debugf("Created chained groupcall info. chained_groupcall_id: %s", tmp.ID)
		}(chainedGroupcallID, destination)
	}

	// create chained calls
	for chainedCallID, destination := range mapCalls {
		go func(callID uuid.UUID, destination *commonaddress.Address) {
			// we don't allow to add the connect option for groupcall
			tmp, err := h.reqHandler.CallV1CallCreateWithID(ctx, callID, customerID, flowID, uuid.Nil, masterCallID, source, destination, id, false, false)
			if err != nil {
				log.WithField("dial_destination", destination).Errorf("Could not create a chained call. err: %v", err)
				_, _ = h.HangupGroupcall(ctx, id)
				return
			}
			log.WithField("chained_call", tmp).Debugf("Created chained call info. chanined_call_id: %s", tmp.ID)
		}(chainedCallID, destination)
	}

	return res, nil
}

// startLinear the groupcall process linear type
func (h *groupcallHandler) startLinear(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
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
		"flow_id":             flowID,
		"source":              source,
		"destinations":        destinations,
		"master_call_id":      masterCallID,
		"master_groupcall_id": masterGroupcallID,
		"answer_method":       answerMethod,
	})
	log.Debugf("Starting the linear ring method groupcall service.")

	// get dialdestination
	destination := destinations[0]

	// set groupcallIDs or callIDs
	tmpID := h.utilHandler.UUIDCreate()
	groupcallIDs := []uuid.UUID{}
	callIDs := []uuid.UUID{}

	if h.IsGroupcallTypeAddress(&destination) {
		groupcallIDs = append(groupcallIDs, tmpID)
	} else {
		callIDs = append(callIDs, tmpID)
	}

	// create groupcall
	// we need to create groupcall earlier than the call. because if we create a call first, it is possible to hangup/answer the call before the create a groupcall
	// if that is happen, we will loose the groupcall control.
	res, err := h.Create(ctx, id, customerID, commonidentity.OwnerTypeNone, uuid.Nil, flowID, source, destinations, callIDs, groupcallIDs, masterCallID, masterGroupcallID, groupcall.RingMethodLinear, answerMethod)
	if err != nil {
		log.Errorf("Could not create groupcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create groupcall.")
	}

	go func() {
		// create chained groupcall/call
		if h.IsGroupcallTypeAddress(&destination) {
			tmp, err := h.startWithDestination(ctx, tmpID, customerID, flowID, source, &destination, masterCallID, masterGroupcallID, groupcall.RingMethodLinear, groupcall.AnswerMethodHangupOthers)

			// tmp, err := h.dialChainedGroupcall(ctx, tmpID, customerID, flowID, source, &dialDestination, masterCallID, id)
			if err != nil {
				log.Errorf("Could not create the chained groupcall info. err: %v", err)
				_, _ = h.HangupGroupcall(ctx, id)
				return
			}
			log.WithField("chained_groupcall", tmp).Debugf("Created chained groupcall info. chained_groupcall_id: %s", tmp.ID)
		} else {
			// we don't allow to add the connect option for groupcall
			tmp, err := h.reqHandler.CallV1CallCreateWithID(ctx, tmpID, customerID, flowID, uuid.Nil, masterCallID, source, &destination, res.ID, false, false)
			if err != nil {
				log.Errorf("Could not create the chained call info. err: %v", err)
				_, _ = h.HangupGroupcall(ctx, id)
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

func (h *groupcallHandler) startWithDestination(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	flowID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
	masterCallID uuid.UUID,
	masterGroupcallID uuid.UUID,
	ringMethod groupcall.RingMethod,
	answerMethod groupcall.AnswerMethod,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "startWithDestination",
		"id":                  id,
		"customer_id":         customerID,
		"flow_id":             flowID,
		"source":              source,
		"destinations":        destination,
		"master_call_id":      masterCallID,
		"master_groupcall_id": masterGroupcallID,
		"ring_method":         ringMethod,
		"answer_method":       answerMethod,
	})

	if id == uuid.Nil {
		// the id is nil. Create a new id
		id = h.utilHandler.UUIDCreate()
		log = log.WithField("id", id)
		log.Debugf("The no id has given. Generate a new id. id: %s", id)
	}

	// get dial addresses and ring method
	dialAddresses, dialRingMethod, err := h.getDialAddressesAndRingMethod(ctx, customerID, destination)
	if err != nil {
		log.Errorf("Could not get dial addresses. err: %v", err)
		return nil, errors.Wrap(err, "could not get dial addresses")
	}

	mapGroupcalls := map[uuid.UUID]*commonaddress.Address{}
	mapCalls := map[uuid.UUID]*commonaddress.Address{}
	for _, dialDestination := range dialAddresses {
		tmpID := h.utilHandler.UUIDCreate()
		if h.IsGroupcallTypeAddress(&dialDestination) {
			mapGroupcalls[tmpID] = &dialDestination
		} else {
			mapCalls[tmpID] = &dialDestination
		}
	}

	// get address owner
	ownerType, ownerID, err := h.getAddressOwner(ctx, customerID, destination)
	if err != nil {
		// could not get owner info, but just write the log only
		log.Errorf("Could not get address owner. err: %v", err)
	}

	// create groupcall
	callIDs := getKeys(mapCalls)
	groupcallIDs := getKeys(mapGroupcalls)
	res, err := h.Create(ctx, id, customerID, ownerType, ownerID, flowID, source, []commonaddress.Address{*destination}, callIDs, groupcallIDs, masterCallID, masterGroupcallID, ringMethod, answerMethod)
	if err != nil {
		log.Errorf("Could not create groupcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create groupcall.")
	}

	// create chained groupcall
	for chainedGroupcallID, dest := range mapGroupcalls {
		go func(targetGroupcallID uuid.UUID, targetDestination *commonaddress.Address) {
			log = log.WithFields(logrus.Fields{
				"chained_groupcall_id":          targetGroupcallID,
				"chained_groupcall_destination": targetDestination,
			})
			log.Debugf("Creating chained groupcall. chained_groupcall_id: %v", targetGroupcallID)

			dests := []commonaddress.Address{
				*targetDestination,
			}

			tmp, err := h.reqHandler.CallV1GroupcallCreate(ctx, targetGroupcallID, customerID, flowID, *source, dests, masterCallID, id, dialRingMethod, groupcall.AnswerMethodHangupOthers)
			if err != nil {
				log.Errorf("Could not create chained groupcall info. err: %v", err)
				return
			}
			log.WithField("chained_groupcall", tmp).Debugf("Created chained groupcall info. chained_groupcall_id: %s", tmp.ID)
		}(chainedGroupcallID, dest)
	}

	// create chained calls
	for chainedCallID, destination := range mapCalls {

		go func(targetCallID uuid.UUID, targetDestination *commonaddress.Address) {
			log = log.WithFields(logrus.Fields{
				"chained_call_id":          targetCallID,
				"chained_call_destination": targetDestination,
			})
			log.Debugf("Creating chained call. chained_groupcall_id: %v", targetCallID)

			// we don't allow to add the connect option for groupcall
			tmp, err := h.reqHandler.CallV1CallCreateWithID(ctx, targetCallID, customerID, flowID, uuid.Nil, masterCallID, source, targetDestination, id, false, false)
			if err != nil {
				log.WithField("dial_destination", targetDestination).Errorf("Could not create a chained call. err: %v", err)
				_, _ = h.HangupCall(ctx, id)
				return
			}
			log.WithField("chained_call", tmp).Debugf("Created chained call info. chanined_call_id: %s", tmp.ID)
		}(chainedCallID, destination)
	}

	return res, nil
}
