package messagehandler

import (
	"context"
	"fmt"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

// Send sends the message.
func (h *messageHandler) Send(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Send",
		"customer_id":  customerID,
		"source":       source,
		"destinations": destinations,
	})
	log.Debugf("Sending the message. message_len: %d", len(text))

	// gate: customer identity verification (fail-closed for unverified customers).
	// The helper logs the rejection reason, so we do not log again here.
	if !h.validateCustomerIdentityVerified(ctx, customerID) {
		return nil, fmt.Errorf("customer identity verification required to send message")
	}

	// Canonicalize source/destinations through the shared address normalization
	// authority so BOTH the persisted message and the provider wire value are in
	// canonical form. NormalizeTarget is loss-proof (an alphanumeric/sentinel
	// sender with no digit is preserved verbatim), so the error is discarded.
	// Nil-guard the source pointer; normalize destinations by index.
	if source != nil {
		source.Target, _ = commonaddress.NormalizeTarget(source.Type, source.Target)
	}
	for i := range destinations {
		destinations[i].Target, _ = commonaddress.NormalizeTarget(destinations[i].Type, destinations[i].Target)
	}

	// create targets
	targets := []target.Target{}
	for _, destination := range destinations {
		t := target.Target{
			Destination: destination,
			Status:      target.StatusQueued,
		}

		targets = append(targets, t)
	}

	// check the balance
	count := len(targets)
	valid, err := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeSMS, "", count)
	if err != nil {
		log.Errorf("Could not validate the customer's balance. err: %v", err)
		return nil, errors.Wrap(err, "could not validate the customer's balance")
	}
	if !valid {
		log.Errorf("Customer has insufficient balance. customer_id: %s", customerID)
		return nil, fmt.Errorf("insufficient balance for message")
	}

	// gate: outbound SMS rate limit (per-minute and per-hour, fail-closed). VOIP-1259.
	if !h.validateCustomerMessageRate(ctx, customerID) {
		return nil, cerrors.ResourceExhausted(commonoutline.ServiceNameMessageManager, "RATE_LIMIT_EXCEEDED", "outbound SMS rate limit exceeded")
	}

	// select provider
	// currently, we use the messagebird only
	provider := message.ProviderNameMessagebird

	res, err := h.Create(ctx, id, customerID, source, targets, provider, text, message.DirectionOutbound)
	if err != nil {
		log.Errorf("Could not create a new message. err: %v", err)
		return nil, err
	}

	// send the message
	go func() {

		handlers := map[message.ProviderName]func(context.Context, uuid.UUID, *commonaddress.Address, []target.Target, string) ([]target.Target, error){
			message.ProviderNameTelnyx:      h.messageHandlerTelnyx.SendMessage,
			message.ProviderNameMessagebird: h.messageHandlerMessagebird.SendMessage,
		}

		for providerName, handler := range handlers {
			tmp, err := handler(ctx, res.ID, source, targets, text)
			if err != nil {
				log.Errorf("Could not send the message correctly. handler: %s, err: %v", providerName, err)
				continue
			}

			updatedTmp, err := h.dbUpdateTargets(ctx, res.ID, providerName, tmp)
			if err != nil {
				log.Errorf("Could not update the message targets. handler: %s, err: %v", providerName, err)
				return
			}

			log.Debugf("Sent the message correctly. provider_name: %s, message_id: %s", providerName, updatedTmp.ID)
			return
		}
	}()

	return res, nil
}
