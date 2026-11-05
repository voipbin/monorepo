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
	mmsDispatchTimeout   = 5 * time.Minute  // all the attempts of one message
	mmsFinalWriteTimeout = 30 * time.Second // the final write of the result, which has its own context
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
	go h.dispatchMMS(ctx, res.ID, &sourceCopy, targetsCopy, text, urls)

	return res, nil
}

// createMMS creates the message record of an mms. The provider is the first one that is tried,
// and dispatchMMS replaces it with the provider that accepted the targets.
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

// dispatchMMS sends the targets through telnyx and sends the targets that telnyx refused through
// messagebird, then writes the final state once. Every target is sent to each provider at most
// once, and only a definitive refusal leads to the fallback, so a target is never sent twice. A
// target that no provider accepted becomes failed: an mms is never sent as text only.
func (h *messageHandler) dispatchMMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, urls []string) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "dispatchMMS",
		"message_id": messageID,
	})

	sendCtx, cancel := context.WithTimeout(ctx, mmsDispatchTimeout)
	defer cancel()

	final := append([]target.Target(nil), targets...)
	failed := make([]bool, len(final))
	telnyxAccepted := 0
	messagebirdAccepted := 0

	// stage 1: telnyx
	outcomes := normalizeOutcomes(h.mmsTelnyx.SendMMS(sendCtx, messageID, source, targets, text, urls), len(targets))

	fallbackIndexes := []int{}
	for i, outcome := range outcomes {
		switch outcome {
		case MMSOutcomeAccepted:
			telnyxAccepted++

		case MMSOutcomeRejected:
			if mmsFallbackEligible(source, targets[i]) {
				fallbackIndexes = append(fallbackIndexes, i)
				continue
			}
			failed[i] = true
			promMMSTargetFailedTotal.WithLabelValues("telnyx", "ineligible").Inc()

		default:
			failed[i] = true
			promMMSTargetFailedTotal.WithLabelValues("telnyx", "error").Inc()
		}
	}

	// stage 2: messagebird, only for the targets that telnyx refused
	if len(fallbackIndexes) > 0 {
		fallbackTargets := make([]target.Target, 0, len(fallbackIndexes))
		for _, i := range fallbackIndexes {
			fallbackTargets = append(fallbackTargets, targets[i])
		}

		fallbackOutcomes := normalizeOutcomes(h.mmsMessagebird.SendMMS(sendCtx, messageID, source, fallbackTargets, text, urls), len(fallbackTargets))
		for j, i := range fallbackIndexes {
			switch fallbackOutcomes[j] {
			case MMSOutcomeAccepted:
				messagebirdAccepted++

			case MMSOutcomeRejected:
				failed[i] = true
				promMMSTargetFailedTotal.WithLabelValues("messagebird", "rejected").Inc()

			default:
				failed[i] = true
				promMMSTargetFailedTotal.WithLabelValues("messagebird", "error").Inc()
			}
		}
	}

	// the final state
	var tmFailed *time.Time
	for i := range final {
		if !failed[i] {
			continue
		}

		if tmFailed == nil {
			tmFailed = h.utilHandler.TimeNow()
		}
		final[i].Status = target.StatusFailed
		final[i].TMUpdate = tmFailed
	}

	providerName := message.ProviderNameTelnyx
	if telnyxAccepted == 0 && messagebirdAccepted > 0 {
		providerName = message.ProviderNameMessagebird
	}

	// the write has its own context, so a dispatch that ran out of time still records the result.
	writeCtx, writeCancel := context.WithTimeout(context.WithoutCancel(ctx), mmsFinalWriteTimeout)
	defer writeCancel()

	if _, err := h.dbUpdateTargets(writeCtx, messageID, providerName, final); err != nil {
		log.Errorf("Could not update the message targets. err: %v", err)
		return
	}
	log.Debugf("Dispatched the mms. telnyx_accepted: %d, messagebird_accepted: %d, total: %d", telnyxAccepted, messagebirdAccepted, len(final))
}

// normalizeOutcomes returns exactly count outcomes. A provider that returns a different number breaks
// its contract, so every target is treated as not confirmed (failed) instead of guessing which
// outcome belongs to which target.
func normalizeOutcomes(outcomes []MMSOutcome, count int) []MMSOutcome {
	if len(outcomes) == count {
		return outcomes
	}

	return make([]MMSOutcome, count)
}
