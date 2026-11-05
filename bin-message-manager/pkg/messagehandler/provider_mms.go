package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_provider_mms.go -source provider_mms.go -build_flags=-mod=mod

import (
	"context"
	"sync"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

const (
	mmsProviderConcurrency = 10
	mmsRequestTimeout      = 30 * time.Second
)

// MMSOutcome is the result of one mms target at one provider.
type MMSOutcome int

// list of MMSOutcome
const (
	// MMSOutcomeFailed means the provider did not confirm the target: an error without a proof of
	// refusal, or the target was not attempted. The message may exist at the provider.
	MMSOutcomeFailed MMSOutcome = iota
	// MMSOutcomeAccepted means the provider accepted the target.
	MMSOutcomeAccepted
	// MMSOutcomeRejected means the provider refused the target and nothing was queued.
	MMSOutcomeRejected
)

// MMSProvider sends an mms through one provider. It never returns a Go error: the outcome of every
// target is reported by itself, in the order of the given targets, so that a partial failure cannot
// hide an accepted target.
type MMSProvider interface {
	SendMMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, mediaURLs []string) []MMSOutcome
}

// sendMMSTargets runs send for every target with at most mmsProviderConcurrency requests in flight
// and returns the outcomes by index. A target whose turn does not come before the context ends is
// not attempted and stays MMSOutcomeFailed.
func sendMMSTargets(ctx context.Context, count int, send func(ctx context.Context, i int) MMSOutcome) []MMSOutcome {
	res := make([]MMSOutcome, count)

	var wg sync.WaitGroup
	sem := make(chan struct{}, mmsProviderConcurrency)
	for i := 0; i < count; i++ {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()

			reqCtx, cancel := context.WithTimeout(ctx, mmsRequestTimeout)
			defer cancel()

			res[i] = send(reqCtx, i)
		}(i)
	}
	wg.Wait()

	return res
}

type mmsTelnyx struct {
	requestExternal requestexternal.RequestExternal
}

// NewMMSTelnyx returns the telnyx mms provider.
func NewMMSTelnyx(reqExternal requestexternal.RequestExternal) MMSProvider {
	return &mmsTelnyx{requestExternal: reqExternal}
}

// SendMMS sends the mms to every target, one request per target.
func (h *mmsTelnyx) SendMMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, mediaURLs []string) []MMSOutcome {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SendMMS",
		"provider":   "telnyx",
		"message_id": messageID,
	})

	return sendMMSTargets(ctx, len(targets), func(ctx context.Context, i int) MMSOutcome {
		_, err := h.requestExternal.TelnyxSendMMS(ctx, source.Target, targets[i].Destination.Target, text, mediaURLs)
		if err == nil {
			promTelnyxSendTotal.WithLabelValues("mms").Inc()
			return MMSOutcomeAccepted
		}

		if requestexternal.IsDefinitiveRejection(err) {
			log.Errorf("The provider refused the mms. target_index: %d, err: %v", i, err)
			return MMSOutcomeRejected
		}

		log.Errorf("Could not send the mms. The result is not known. target_index: %d, err: %v", i, err)
		return MMSOutcomeFailed
	})
}

type mmsMessagebird struct {
	requestExternal requestexternal.RequestExternal
}

// NewMMSMessagebird returns the messagebird mms provider.
func NewMMSMessagebird(reqExternal requestexternal.RequestExternal) MMSProvider {
	return &mmsMessagebird{requestExternal: reqExternal}
}

// SendMMS sends the mms to every target, one request per target.
func (h *mmsMessagebird) SendMMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, mediaURLs []string) []MMSOutcome {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SendMMS",
		"provider":   "messagebird",
		"message_id": messageID,
	})

	return sendMMSTargets(ctx, len(targets), func(ctx context.Context, i int) MMSOutcome {
		_, err := h.requestExternal.MessagebirdSendMMS(ctx, source.Target, targets[i].Destination.Target, text, mediaURLs)
		if err == nil {
			promMessagebirdSendTotal.WithLabelValues("mms").Inc()
			return MMSOutcomeAccepted
		}

		if requestexternal.IsDefinitiveRejection(err) {
			log.Errorf("The provider refused the mms. target_index: %d, err: %v", i, err)
			return MMSOutcomeRejected
		}

		log.Errorf("Could not send the mms. The result is not known. target_index: %d, err: %v", i, err)
		return MMSOutcomeFailed
	})
}
