package callhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
)

// createGroupDial creates a new group dial.
func (h *callHandler) createGroupDial(
	ctx context.Context,
	customerID uuid.UUID,
	flowID uuid.UUID,
	masterCallID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
	earlyExecution bool,
	executeNextMasterOnHangup bool,
) (*groupdial.GroupDial, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "createGroupDial",
		"master_call_id": masterCallID,
	})

	// get dial destination.
	var err error
	var dialDestinations []*commonaddress.Address
	var ringMethod groupdial.RingMethod
	var answerMethod groupdial.AnswerMethod
	switch destination.Type {
	case commonaddress.TypeEndpoint:
		// get detailed destinations
		// create a call
		dialDestinations, err = h.getDestinationsAddressTypeEndpoint(ctx, customerID, destination)
		ringMethod = groupdial.RingMethodRingAll
		answerMethod = groupdial.AnswerMethodHangupOthers

	default:
		// nothing to do
		return nil, fmt.Errorf("unsupported destination type")
	}
	if err != nil {
		log.Errorf("Could not get dial uris. err: %v", err)
		return nil, errors.Wrap(err, "Could not get dial uris.")
	}
	log.WithField("dial_destinations", dialDestinations).Debugf("Found dial destinations for group dial. destination_type: %s", destination.Type)

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
	res := &groupdial.GroupDial{
		ID:         h.utilHandler.CreateUUID(),
		CustomerID: customerID,

		Destination:  destination,
		CallIDs:      callIDs,
		RingMethod:   ringMethod,
		AnswerMethod: answerMethod,
	}

	if errCreate := h.db.GroupDialCreate(ctx, res); errCreate != nil {
		log.Errorf("Could not create the group dial. err: %v", errCreate)
		return nil, errors.Wrap(errCreate, "Could not create the group dial.")
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
