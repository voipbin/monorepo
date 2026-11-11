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

// SendOutcome is the result of one target at one provider.
type SendOutcome int

// list of SendOutcome
const (
	// SendOutcomeFailed means the provider did not confirm the target: an error without a proof of
	// refusal, or the target was not attempted. The message may exist at the provider.
	SendOutcomeFailed SendOutcome = iota
	// SendOutcomeAccepted means the provider accepted the target.
	SendOutcomeAccepted
	// SendOutcomeRejected means the provider refused the target and nothing was queued.
	SendOutcomeRejected
)

// SendResult is the answer of a provider for one target.
type SendResult struct {
	Outcome SendOutcome

	// ProviderReferenceID is the message id of the provider (set when accepted).
	ProviderReferenceID string
	// ProviderParts is the segment count that the provider reported (0 when it did not report one).
	ProviderParts int
	// Status is the delivery status that the provider reported at once. Empty keeps the target queued.
	Status target.Status
}

// ResultCallback is called by a provider inside the goroutine of target i right after its request returned, with the
// index in the slice that the provider was given. It may be nil.
type ResultCallback func(i int, r SendResult)

// MMSProvider sends an mms through one provider. It never returns a Go error: the result of every
// target is reported by itself, in the order of the given targets, so that a partial failure cannot
// hide an accepted target.
type MMSProvider interface {
	SendMMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, mediaURLs []string, onResult ResultCallback) []SendResult
}

// sendTargets runs send for every target with at most mmsProviderConcurrency requests in flight
// and returns the results by index. A target whose turn does not come before the context ends is
// not attempted and stays SendOutcomeFailed. onResult (if not nil) is called for each attempted target.
func sendTargets(ctx context.Context, count int, send func(ctx context.Context, i int) SendResult, onResult ResultCallback) []SendResult {
	res := make([]SendResult, count)

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
			if onResult != nil {
				onResult(i, res[i])
			}
		}(i)
	}
	wg.Wait()

	return res
}

// classifyError converts the error of a request to the failed or rejected result.
func classifyError(err error) SendOutcome {
	if requestexternal.IsDefinitiveRejection(err) {
		return SendOutcomeRejected
	}

	return SendOutcomeFailed
}

type mmsTelnyx struct {
	requestExternal requestexternal.RequestExternal
}

// NewMMSTelnyx returns the telnyx mms provider.
func NewMMSTelnyx(reqExternal requestexternal.RequestExternal) MMSProvider {
	return &mmsTelnyx{requestExternal: reqExternal}
}

// SendMMS sends the mms to every target, one request per target.
func (h *mmsTelnyx) SendMMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, mediaURLs []string, onResult ResultCallback) []SendResult {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SendMMS",
		"provider":   "telnyx",
		"message_id": messageID,
	})

	return sendTargets(ctx, len(targets), func(ctx context.Context, i int) SendResult {
		m, err := h.requestExternal.TelnyxSendMMS(ctx, source.Target, targets[i].Destination.Target, text, mediaURLs)
		if err == nil {
			promTelnyxSendTotal.WithLabelValues("mms").Inc()
			res := SendResult{Outcome: SendOutcomeAccepted}
			if m != nil {
				res.ProviderReferenceID = m.Data.ID
				res.ProviderParts = m.Data.Parts
			}
			return res
		}

		outcome := classifyError(err)
		switch {
		case outcome == SendOutcomeRejected:
			log.Errorf("The provider refused the mms. target_index: %d, err: %v", i, err)
		case requestexternal.IsTokenUnavailable(err):
			log.Errorf("The mms was not sent: the provider token could not be read. target_index: %d, err: %v", i, err)
		default:
			log.Errorf("Could not send the mms. The result is not known. target_index: %d, err: %v", i, err)
		}
		return SendResult{Outcome: outcome}
	}, onResult)
}

type mmsMessagebird struct {
	requestExternal requestexternal.RequestExternal
}

// NewMMSMessagebird returns the messagebird mms provider.
func NewMMSMessagebird(reqExternal requestexternal.RequestExternal) MMSProvider {
	return &mmsMessagebird{requestExternal: reqExternal}
}

// SendMMS sends the mms to every target, one request per target.
func (h *mmsMessagebird) SendMMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, mediaURLs []string, onResult ResultCallback) []SendResult {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SendMMS",
		"provider":   "messagebird",
		"message_id": messageID,
	})

	return sendTargets(ctx, len(targets), func(ctx context.Context, i int) SendResult {
		m, err := h.requestExternal.MessagebirdSendMMS(ctx, source.Target, targets[i].Destination.Target, text, mediaURLs)
		if err == nil {
			promMessagebirdSendTotal.WithLabelValues("mms").Inc()
			res := SendResult{Outcome: SendOutcomeAccepted}
			if m != nil {
				res.ProviderReferenceID = m.ID
			}
			return res
		}

		outcome := classifyError(err)
		switch {
		case outcome == SendOutcomeRejected:
			log.Errorf("The provider refused the mms. target_index: %d, err: %v", i, err)
		case requestexternal.IsTokenUnavailable(err):
			log.Errorf("The mms was not sent: the provider token could not be read. target_index: %d, err: %v", i, err)
		default:
			log.Errorf("Could not send the mms. The result is not known. target_index: %d, err: %v", i, err)
		}
		return SendResult{Outcome: outcome}
	}, onResult)
}
