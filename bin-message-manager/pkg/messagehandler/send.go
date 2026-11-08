package messagehandler

import (
	"context"
	"fmt"
	"strconv"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/segmenthandler"
)

// limits of the text of a message without media, counted in segments (3GPP TS 23.038 and TS 23.040).
const (
	// smsPromoteSegments is the number of segments of an sms. A text that needs more is promoted to an mms (a text
	// only mms, LMS) and charged the flat mms rate. The decision never looks at the destination or the provider.
	smsPromoteSegments = 1
	// smsMaxSegments is the most segments of a text without media (the documented limit of one of the providers).
	// A longer text is refused.
	smsMaxSegments = 10
)

// Send sends the message.
//
// A text that fits one segment is an sms. A longer text is promoted to a text only mms: the type is mms, the price is the
// mms rate for each recipient (whatever the segment count and whatever the provider finally sends) and the targets
// carry the segment count of the text in parts. The type and the parts are decided here, before the message is created,
// because the message_created event, which the billing reads, is published when the message is created.
func (h *messageHandler) Send(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Send",
		"customer_id": customerID,
		"recipients":  len(destinations),
	})
	log.Debugf("Sending the message. message_len: %d", len(text))

	// gate: customer identity verification (fail-closed for unverified customers).
	// The helper logs the rejection reason, so we do not log again here.
	if !h.validateCustomerIdentityVerified(ctx, customerID) {
		return nil, fmt.Errorf("customer identity verification required to send message")
	}

	if len(destinations) == 0 {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameMessageManager, "DESTINATION_REQUIRED", "At least one destination is required.")
	}

	// the format of the source is checked on the raw input, before anything is normalized, and without any call to
	// another service. A missing source is nil here and is resolved to the default of the customer below.
	preparedSource, err := prepareSource(source)
	if err != nil {
		return nil, err
	}

	// Canonicalize the destinations through the shared address normalization authority so BOTH the persisted message
	// and the provider wire value are in canonical form. NormalizeTarget is loss-proof, so the error is discarded.
	normalizeDestinations(destinations)

	count := segmenthandler.Count(text)
	if count.Segments > smsMaxSegments {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameMessageManager, "TEXT_TOO_LONG", fmt.Sprintf("The text of a message without media can have at most %d segments.", smsMaxSegments))
	}

	// the sender must be a number of the customer; this is the first check that calls another service, so a rejected
	// request has no side effect and costs nothing.
	source, err = h.resolveSource(ctx, customerID, preparedSource)
	if err != nil {
		return nil, err
	}

	promoted := count.Segments > smsPromoteSegments
	msgType := message.TypeSMS
	refType := bmbilling.ReferenceTypeSMS
	if promoted {
		msgType = message.TypeMMS
		refType = bmbilling.ReferenceTypeMMS
	}

	// create targets
	targets := []target.Target{}
	for _, destination := range destinations {
		targets = append(targets, target.Target{
			Destination: destination,
			Status:      target.StatusQueued,
			Parts:       count.Segments,
		})
	}

	// check the balance with the type that is stored
	valid, err := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, refType, "", len(targets))
	if err != nil {
		log.Errorf("Could not validate the customer's balance. err: %v", err)
		return nil, errors.Wrap(err, "could not validate the customer's balance")
	}
	if !valid {
		log.Errorf("Customer has insufficient balance. customer_id: %s", customerID)
		return nil, fmt.Errorf("insufficient balance for message")
	}

	// the first provider of the chain is recorded; the final write of the dispatch replaces it with the provider that
	// accepted the most targets.
	res, err := h.create(ctx, id, customerID, source, targets, message.ProviderNameTelnyx, text, message.DirectionOutbound, msgType)
	if err != nil {
		log.Errorf("Could not create a new message. err: %v", err)
		return nil, err
	}

	if promoted {
		promPromotedMessagesTotal.WithLabelValues(strconv.Itoa(count.Segments)).Inc()
	}

	// the goroutine gets its own copies, so it shares nothing with the stored message.
	sourceCopy := *source
	targetsCopy := append([]target.Target(nil), targets...)
	if promoted {
		go h.dispatchMMS(ctx, res.ID, &sourceCopy, targetsCopy, text, nil, true)
	} else {
		go h.dispatchSMS(ctx, res.ID, &sourceCopy, targetsCopy, text)
	}

	return res, nil
}

// normalizeDestinations canonicalizes the destinations in place, by index.
func normalizeDestinations(destinations []commonaddress.Address) {
	for i := range destinations {
		destinations[i].Target, _ = commonaddress.NormalizeTarget(destinations[i].Type, destinations[i].Target)
	}
}
