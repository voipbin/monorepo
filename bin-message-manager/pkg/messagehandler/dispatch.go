package messagehandler

import (
	"context"
	"strconv"
	"sync"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

// namedSMSProvider is one entry of the ordered sms provider chain.
type namedSMSProvider struct {
	name     message.ProviderName
	provider SMSProvider
}

// reasons of a promoted target that is sent as sms segments.
const (
	fallbackReasonIneligible = "ineligible" // the source or the destination is not a +1 number, so the mms stage is skipped
	fallbackReasonRejected   = "rejected"   // the mms stage refused the target
)

// dispatchSession holds the state of the dispatch of one message. Every index is the position of the target in the
// stored Targets, never the position in the partial slice that a stage hands to a provider.
type dispatchSession struct {
	h         *messageHandler
	messageID uuid.UUID
	dests     []string // the destination of each target when the dispatch started, to verify the stored targets
	parts     []int    // the platform segment count of each target

	mu         sync.Mutex
	results    map[int]SendResult // the accepted results by index
	failed     map[int]bool
	accepted   map[message.ProviderName]int
	cbFailures int
}

func (h *messageHandler) newDispatchSession(messageID uuid.UUID, targets []target.Target) *dispatchSession {
	s := &dispatchSession{
		h:         h,
		messageID: messageID,
		dests:     make([]string, len(targets)),
		parts:     make([]int, len(targets)),
		results:   map[int]SendResult{},
		failed:    map[int]bool{},
		accepted:  map[message.ProviderName]int{},
	}
	for i, t := range targets {
		s.dests[i] = t.Destination.Target
		s.parts[i] = t.Parts
	}

	return s
}

// check refuses a write on targets that are not the ones of this dispatch.
func (s *dispatchSession) check(targets []target.Target) error {
	if len(targets) != len(s.dests) {
		return errTargetsMismatch
	}
	for i := range targets {
		if targets[i].Destination.Target != s.dests[i] {
			return errTargetsMismatch
		}
	}

	return nil
}

func (s *dispatchSession) accept(name message.ProviderName, i int, r SendResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.results[i] = r
	s.accepted[name]++
}

func (s *dispatchSession) fail(i int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failed[i] = true
}

// providerName returns the provider that accepted the most targets. A tie goes to the first provider of the chain.
func (s *dispatchSession) providerName() message.ProviderName {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := message.ProviderNameTelnyx
	best := s.accepted[res]
	for _, name := range []message.ProviderName{message.ProviderNameMessagebird} {
		if s.accepted[name] > best {
			res = name
			best = s.accepted[name]
		}
	}

	return res
}

// applyResult writes an accepted result into the target. The id and the provider parts are filled only when empty, and the
// status only when the target is still queued, so a receipt that arrived earlier is never lowered. It returns true if
// the target changed.
func (h *messageHandler) applyResult(t *target.Target, r SendResult) bool {
	changed := false

	if t.ProviderReferenceID == "" && r.ProviderReferenceID != "" {
		t.ProviderReferenceID = r.ProviderReferenceID
		changed = true
	}
	if t.ProviderParts == 0 && r.ProviderParts > 0 {
		t.ProviderParts = r.ProviderParts
		changed = true
	}
	if r.Status != "" && r.Status != target.StatusQueued && t.Status == target.StatusQueued {
		t.Status = r.Status
		t.TMUpdate = h.utilHandler.TimeNow()
		changed = true
	}

	return changed
}

// callback returns the per target callback of a stage. idxs maps the position in the stage slice to the stored
// index. It writes the id of an accepted target at once, best effort: a failed write is left to the final write, and
// after callbackMaxFailures consecutive failures the callbacks of the dispatch are skipped.
func (s *dispatchSession) callback(ctx context.Context, idxs []int) ResultCallback {
	idxs = append([]int(nil), idxs...)

	return func(i int, r SendResult) {
		if r.Outcome != SendOutcomeAccepted || i < 0 || i >= len(idxs) {
			return
		}

		s.mu.Lock()
		skip := s.cbFailures >= callbackMaxFailures
		s.mu.Unlock()
		if skip {
			return
		}

		orig := idxs[i]
		_, err := s.h.updateTargets(context.WithoutCancel(ctx), s.messageID, updateOpts{maxTries: callbackWriteTries, budget: callbackWriteBudget}, func(targets []target.Target) error {
			if errCheck := s.check(targets); errCheck != nil {
				return errCheck
			}
			if !s.h.applyResult(&targets[orig], r) {
				return errNoChange
			}
			return nil
		})

		s.mu.Lock()
		defer s.mu.Unlock()
		if err != nil {
			s.cbFailures++
			logrus.WithField("message_id", s.messageID).Warnf("Could not store the result of a target. The final write will do it. err: %v", err)
			return
		}
		s.cbFailures = 0
	}
}

// finalFn is the change of the final write: it marks the failed targets (only the ones still queued) and fills the
// results of the accepted ones.
func (s *dispatchSession) finalFn() func(targets []target.Target) error {
	return func(targets []target.Target) error {
		if err := s.check(targets); err != nil {
			return err
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		var tmFailed *time.Time
		for i := range targets {
			if s.failed[i] {
				if targets[i].Status != target.StatusQueued {
					continue
				}
				if tmFailed == nil {
					tmFailed = s.h.utilHandler.TimeNow()
				}
				targets[i].Status = target.StatusFailed
				targets[i].TMUpdate = tmFailed
				continue
			}

			if r, ok := s.results[i]; ok {
				s.h.applyResult(&targets[i], r)
			}
		}

		return nil
	}
}

// finalWrite writes the result of the dispatch and publishes one message_updated event. It must not be lost, so it
// retries the compare and set for finalWriteCASBudget and then merges once without the comparison in a fresh
// context. The contexts are detached from ctx, so a dispatch that ran out of time still records its result.
func (h *messageHandler) finalWrite(ctx context.Context, s *dispatchSession) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "finalWrite",
		"message_id": s.messageID,
	})

	provider := s.providerName()
	detached := context.WithoutCancel(ctx)

	_, err := h.updateTargets(detached, s.messageID, updateOpts{budget: finalWriteCASBudget, provider: &provider}, s.finalFn())
	if err != nil {
		log.Warnf("The compare and set final write failed. Merging. err: %v", err)

		mergeCtx, cancel := context.WithTimeout(detached, finalWriteMergeBudget)
		defer cancel()

		if errMerge := h.mergeTargets(mergeCtx, s.messageID, provider, s.finalFn()); errMerge != nil {
			promFinalWriteFailedTotal.Inc()
			log.Errorf("Could not write the result of the dispatch. The targets stay queued. err: %v", errMerge)
			return
		}
	}

	h.publishUpdated(detached, s.messageID)

	s.mu.Lock()
	log.Debugf("Dispatched the message. accepted: %v, failed: %d, total: %d", s.accepted, len(s.failed), len(s.dests))
	s.mu.Unlock()
}

// subset returns the targets at the given stored indexes.
func subset(targets []target.Target, idxs []int) []target.Target {
	res := make([]target.Target, 0, len(idxs))
	for _, i := range idxs {
		res = append(res, targets[i])
	}

	return res
}

// normalizeResults returns exactly count results. A provider that returns a different number breaks its contract,
// so every target is treated as not confirmed (failed) instead of guessing which result belongs to which target.
func normalizeResults(results []SendResult, count int) []SendResult {
	if len(results) == count {
		return results
	}

	return make([]SendResult, count)
}

// runSMSChain sends the targets at idxs through the ordered sms providers. An accepted target is recorded; a rejected
// one goes to the next provider; a failed one (result unknown) is not sent again, because it could be a duplicate;
// what the last provider rejected is failed.
func (h *messageHandler) runSMSChain(ctx context.Context, s *dispatchSession, source *commonaddress.Address, targets []target.Target, text string, idxs []int) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "runSMSChain",
		"message_id": s.messageID,
	})

	remaining := append([]int(nil), idxs...)
	for _, np := range h.smsProviders {
		if len(remaining) == 0 {
			break
		}

		stage := remaining
		results := normalizeResults(np.provider.SendSMS(ctx, s.messageID, source, subset(targets, stage), text, s.callback(ctx, stage)), len(stage))

		next := []int{}
		for j, r := range results {
			i := stage[j]
			switch r.Outcome {
			case SendOutcomeAccepted:
				s.accept(np.name, i, r)
				if r.ProviderParts > 0 && s.parts[i] > 0 && r.ProviderParts != s.parts[i] {
					promSegmentMismatchTotal.WithLabelValues(string(np.name)).Inc()
					log.Infof("The provider counted other segments than the platform. provider: %s, platform: %d, provider_count: %d", np.name, s.parts[i], r.ProviderParts)
				}

			case SendOutcomeRejected:
				next = append(next, i)

			default:
				s.fail(i)
			}
		}
		remaining = next
	}

	for _, i := range remaining {
		s.fail(i)
	}
}

// dispatchSMS sends an sms through the provider chain and writes the result once.
func (h *messageHandler) dispatchSMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string) {
	sendCtx, cancel := context.WithTimeout(ctx, mmsDispatchTimeout)
	defer cancel()

	s := h.newDispatchSession(messageID, targets)

	idxs := make([]int, len(targets))
	for i := range idxs {
		idxs[i] = i
	}

	h.runSMSChain(sendCtx, s, source, targets, text, idxs)
	h.finalWrite(ctx, s)
}

// mmsOutcomeLabel is the label value of the outcome of a promoted mms request.
func mmsOutcomeLabel(o SendOutcome) string {
	switch o {
	case SendOutcomeAccepted:
		return "accepted"
	case SendOutcomeRejected:
		return "rejected"
	default:
		return "failed"
	}
}

// dispatchMMS sends the targets through telnyx and sends the targets that telnyx refused through messagebird, then
// writes the final state once. Every target is sent to each provider at most once, and only a definitive refusal leads
// to the next one, so a target is never sent twice.
//
// An mms with attached media (promoted == false) never goes out as text only: a target that no provider accepted
// becomes failed. A promoted message (a text longer than one segment, no media) is different: a target that is not
// +1 to +1 skips the mms stages, and a target that the mms stages refused is sent as sms segments through the sms
// provider chain. The price was fixed when the message was created and does not change.
func (h *messageHandler) dispatchMMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, urls []string, promoted bool) {
	sendCtx, cancel := context.WithTimeout(ctx, mmsDispatchTimeout)
	defer cancel()

	s := h.newDispatchSession(messageID, targets)

	mmsIdxs := []int{}
	smsIdxs := []int{}
	for i := range targets {
		if promoted && !mmsFallbackEligible(source, targets[i]) {
			smsIdxs = append(smsIdxs, i)
			promPromotedFallbackTotal.WithLabelValues(fallbackReasonIneligible, strconv.Itoa(s.parts[i])).Inc()
			continue
		}
		mmsIdxs = append(mmsIdxs, i)
	}

	// stage 1: telnyx
	fallbackIdxs := []int{}
	if len(mmsIdxs) > 0 {
		results := normalizeResults(h.mmsTelnyx.SendMMS(sendCtx, messageID, source, subset(targets, mmsIdxs), text, urls, s.callback(sendCtx, mmsIdxs)), len(mmsIdxs))
		for j, r := range results {
			i := mmsIdxs[j]
			if promoted {
				promPromotedMMSTotal.WithLabelValues("telnyx", mmsOutcomeLabel(r.Outcome)).Inc()
			}

			switch r.Outcome {
			case SendOutcomeAccepted:
				s.accept(message.ProviderNameTelnyx, i, r)

			case SendOutcomeRejected:
				// for a promoted target the split already left only the +1 to +1 targets here.
				if mmsFallbackEligible(source, targets[i]) {
					fallbackIdxs = append(fallbackIdxs, i)
					continue
				}
				s.fail(i)
				promMMSTargetFailedTotal.WithLabelValues("telnyx", "ineligible").Inc()

			default:
				s.fail(i)
				promMMSTargetFailedTotal.WithLabelValues("telnyx", "error").Inc()
			}
		}
	}

	// stage 2: messagebird, only for the targets that telnyx refused
	if len(fallbackIdxs) > 0 {
		results := normalizeResults(h.mmsMessagebird.SendMMS(sendCtx, messageID, source, subset(targets, fallbackIdxs), text, urls, s.callback(sendCtx, fallbackIdxs)), len(fallbackIdxs))
		for j, r := range results {
			i := fallbackIdxs[j]
			if promoted {
				promPromotedMMSTotal.WithLabelValues("messagebird", mmsOutcomeLabel(r.Outcome)).Inc()
			}

			switch r.Outcome {
			case SendOutcomeAccepted:
				s.accept(message.ProviderNameMessagebird, i, r)

			case SendOutcomeRejected:
				if promoted {
					smsIdxs = append(smsIdxs, i)
					promPromotedFallbackTotal.WithLabelValues(fallbackReasonRejected, strconv.Itoa(s.parts[i])).Inc()
					continue
				}
				s.fail(i)
				promMMSTargetFailedTotal.WithLabelValues("messagebird", "rejected").Inc()

			default:
				s.fail(i)
				promMMSTargetFailedTotal.WithLabelValues("messagebird", "error").Inc()
			}
		}
	}

	// the promoted targets that were not sent as mms go out as sms segments
	if len(smsIdxs) > 0 {
		h.runSMSChain(sendCtx, s, source, targets, text, smsIdxs)
	}

	h.finalWrite(ctx, s)
}
