package messagehandler

import (
	"context"
	"fmt"
	"strings"
	"time"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

const (
	mmsDispatchTimeout = 5 * time.Minute // all the attempts of one message, over all the stages
)

// SendMMS validates the medias, checks the balance and sends the message with the medias.
//
// Everything that can reject the request runs before the message is created, so a rejected request
// has no side effect and can be retried. The send itself runs in a goroutine, like Send.
func (h *messageHandler) SendMMS(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string, medias []uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "SendMMS",
		"customer_id":  customerID,
		"source":       source,
		"destinations": destinations,
		"medias":       len(medias),
	})
	log.Debugf("Sending the mms. message_len: %d", len(text))

	// gate: customer identity verification, the same gate as Send.
	if !h.validateCustomerIdentityVerified(ctx, customerID) {
		return nil, fmt.Errorf("customer identity verification required to send message")
	}

	if source == nil {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameMessageManager, "SOURCE_REQUIRED", "The source is required.")
	}
	if len(destinations) == 0 {
		return nil, cerrors.InvalidArgument(commonoutline.ServiceNameMessageManager, "DESTINATION_REQUIRED", "At least one destination is required.")
	}
	normalizeSendAddresses(source, destinations)

	if err := validateMMSText(text); err != nil {
		return nil, err
	}

	// one deadline for the media validation, the balance check in between and the url refresh. A slow
	// balance check can therefore end the url refresh with MEDIA_CHECK_TIMEOUT, which is safe because
	// the message is not created yet.
	tmStart := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, mmsMediaCheckTimeout)
	defer cancel()

	if err := h.validateMedias(checkCtx, customerID, medias); err != nil {
		return nil, err
	}

	targets := []target.Target{}
	for _, destination := range destinations {
		targets = append(targets, target.Target{
			Destination: destination,
			Status:      target.StatusQueued,
			Parts:       1,
		})
	}

	valid, err := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeMMS, "", len(targets))
	if err != nil {
		log.Errorf("Could not validate the customer's balance. err: %v", err)
		return nil, errors.Wrap(err, "could not validate the customer's balance")
	}
	if !valid {
		log.Errorf("Customer has insufficient balance. customer_id: %s", customerID)
		return nil, cerrors.PaymentRequired(commonoutline.ServiceNameMessageManager, "INSUFFICIENT_BALANCE", "The balance is not enough to send the message.")
	}

	urls, err := h.refreshMediaURLs(checkCtx, medias)
	if err != nil {
		return nil, err
	}
	promMMSMediaCheckSeconds.Observe(time.Since(tmStart).Seconds())

	res, err := h.createMMS(ctx, id, customerID, source, targets, text, medias)
	if err != nil {
		log.Errorf("Could not create a new message. err: %v", err)
		return nil, err
	}

	// the goroutine gets its own copies, so it shares nothing with the stored message.
	sourceCopy := *source
	targetsCopy := append([]target.Target(nil), targets...)
	go h.dispatchMMS(ctx, res.ID, &sourceCopy, targetsCopy, text, urls, false)

	return res, nil
}

// createMMS creates the message record of an mms. The provider is the first one that is tried,
// and the final write of the dispatch replaces it with the provider that accepted the targets.
func (h *messageHandler) createMMS(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, medias []uuid.UUID) (*message.Message, error) {
	if id == uuid.Nil {
		id = h.utilHandler.UUIDCreate()
	}

	mediaIDs := make([]string, 0, len(medias))
	for _, m := range medias {
		mediaIDs = append(mediaIDs, m.String())
	}

	m := &message.Message{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Type: message.TypeMMS,

		Source:  source,
		Targets: targets,

		ProviderName: message.ProviderNameTelnyx,

		Text:      text,
		Medias:    mediaIDs,
		Direction: message.DirectionOutbound,
	}

	return h.dbCreate(ctx, m)
}

// mmsFallbackEligible returns true if a target that the first provider refused can be sent through
// the fallback provider. The fallback provider sends mms only inside the US and Canada, so the
// sender and the destination must both be +1 numbers.
func mmsFallbackEligible(source *commonaddress.Address, t target.Target) bool {
	return strings.HasPrefix(source.Target, "+1") && strings.HasPrefix(t.Destination.Target, "+1")
}
