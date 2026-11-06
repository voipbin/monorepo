package messagehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	nmnumber "monorepo/bin-number-manager/models/number"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/models/telnyx"
	"monorepo/bin-message-manager/pkg/segmenthandler"
)

// list of the telnyx event types
const (
	telnyxEventReceived  = "message.received"
	telnyxEventSent      = "message.sent"
	telnyxEventFinalized = "message.finalized"
)

// receiptWindow is how far back a receipt looks for the message.
const receiptWindow = 7 * 24 * time.Hour

// ErrReceiptRetry marks a receipt that could not be processed because of a database error or a write conflict that
// did not resolve. The listen handler answers 500 for it, so the hook-manager reports the failure and the provider can
// deliver the receipt again; processing a receipt twice is safe because a target only moves forward.
var ErrReceiptRetry = errors.New("the receipt could not be processed, retry")

// Hook handles hook message
func (h *messageHandler) Hook(ctx context.Context, uri string, data []byte) error {
	var m *message.Message
	var num *nmnumber.Number
	var err error
	switch {
	case strings.HasSuffix(uri, hookTelnyx):
		m, num, err = h.hookTelnyx(ctx, data)
		if err != nil {
			return errors.Wrapf(err, "Could not handle the hook message. uri: %s", uri)
		}

	default:
		return fmt.Errorf("unknown hook uri. uri: %s", uri)
	}

	if m == nil || num == nil {
		// nothing to do.
		return nil
	}

	return nil
}

// hookTelnyx telnyx type hook message.
func (h *messageHandler) hookTelnyx(ctx context.Context, data []byte) (*message.Message, *nmnumber.Number, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "hookTelnyx",
	})

	hm := telnyx.MessageEvent{}
	if errUnmarshal := json.Unmarshal(data, &hm); errUnmarshal != nil {
		// the body is not part of the error: it carries the numbers, the text and the media urls.
		return nil, nil, errors.Wrapf(errUnmarshal, "Could not unmarshal the data. data_size: %d", len(data))
	}
	log = log.WithFields(logrus.Fields{
		"event_type": hm.Data.EventType,
		"event_id":   hm.Data.ID,
	})
	log.Debugf("Unmarshalled hook message. event_type: %s", hm.Data.EventType)

	// the event type decides first: only message.received creates a message.
	switch hm.Data.EventType {
	case telnyxEventReceived:
		// handled below

	case telnyxEventSent, telnyxEventFinalized:
		return nil, nil, h.hookTelnyxReceipt(ctx, &hm)

	default:
		log.Debugf("Ignoring the event. event_type: %s", hm.Data.EventType)
		return nil, nil, nil
	}

	if len(hm.Data.Payload.To) == 0 {
		return nil, nil, fmt.Errorf("destination address is empty")
	}

	destinationNumber := hm.Data.Payload.To[0].PhoneNumber
	log.Debugf("Parsed destination number. destination_number: %s", destinationNumber)

	// get number info
	filters := map[nmnumber.Field]any{
		nmnumber.FieldNumber:  destinationNumber,
		nmnumber.FieldDeleted: false,
	}
	numbs, err := h.reqHandler.NumberV1NumberList(ctx, "", 1, filters)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Could not get number info. number: %s", destinationNumber)
	}

	if len(numbs) == 0 {
		return nil, nil, fmt.Errorf("no number info found. len: %d", len(numbs))
	}

	num := numbs[0]
	log.WithField("number", num).Infof("Found number info. number_id: %s", num.ID)

	// get information
	source := hm.GetSource()
	targets := hm.GetTargets()
	text := hm.GetText()

	// Canonicalize inbound source/targets through the shared address
	// normalization authority. NormalizeTarget is loss-proof, so the error is
	// discarded. Nil-guard the source pointer; normalize targets by index.
	if source != nil {
		source.Target, _ = commonaddress.NormalizeTarget(source.Type, source.Target)
	}
	for i := range targets {
		targets[i].Destination.Target, _ = commonaddress.NormalizeTarget(targets[i].Destination.Type, targets[i].Destination.Target)
	}

	// the media is not downloaded here: the hook has to answer within the rpc timeout and the provider delivers the
	// event again when it does not. The message is created at once and the media is stored in the background.
	medias, skipped := hm.GetMedias()
	if skipped > 0 {
		promInboundMediaTotal.WithLabelValues(string(FetchResultParse)).Add(float64(skipped))
	}

	msgType := message.TypeSMS
	if hm.IsMMS() {
		msgType = message.TypeMMS
	}

	// the platform computes the segment count of an inbound sms; the count of the provider is kept for comparison.
	for i := range targets {
		targets[i].ProviderParts = targets[i].Parts
		if msgType == message.TypeSMS {
			targets[i].Parts = segmenthandler.Count(text).Segments
		} else {
			targets[i].Parts = 1
		}
	}

	res, err := h.create(ctx, uuid.Nil, num.CustomerID, source, targets, message.ProviderNameTelnyx, text, message.DirectionInbound, msgType)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Could not create a message. number_id: %s, number: %s", num.ID, num.Number)
	}
	log.Debugf("Created message. message_id: %s, type: %s", res.ID, res.Type)

	if len(medias) > 0 {
		h.startInboundMediaIngest(ctx, res, medias)
	}

	return res, &num, nil
}

// statusRank orders the target statuses of an outbound message: a receipt only moves a target to a higher rank, so a
// repeated, late or reordered receipt is harmless and a final state is never replaced.
func statusRank(s target.Status) int {
	switch s {
	case target.StatusQueued:
		return 0
	case target.StatusSent:
		return 1
	case target.StatusDelivered, target.StatusFailed, target.StatusDLRTimeout, target.StatusGWTimeout:
		return 2
	default:
		return -1
	}
}

// mapTelnyxStatus converts the status of a telnyx recipient to the target status. An empty result keeps the current
// status.
func mapTelnyxStatus(s string) target.Status {
	switch s {
	case "sent":
		return target.StatusSent
	case "delivered":
		return target.StatusDelivered
	case "delivery_failed", "sending_failed":
		return target.StatusFailed
	case "delivery_unconfirmed":
		return target.StatusDLRTimeout
	default:
		return ""
	}
}

// hookTelnyxReceipt applies a message.sent or message.finalized event of an outbound message to its target.
//
// The message is found by the telnyx message id that the dispatch stored in the target. A receipt that matches nothing
// is not an error (messages sent before the id was stored, other senders of the account): it is answered with success
// and counted. Only a database error, or a conflict that is not resolved, returns ErrReceiptRetry.
func (h *messageHandler) hookTelnyxReceipt(ctx context.Context, hm *telnyx.MessageEvent) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "hookTelnyxReceipt",
		"event_type": hm.Data.EventType,
		"event_id":   hm.Data.ID,
	})

	p := hm.Data.Payload
	if p.Direction != "outbound" {
		log.Debugf("Ignoring the receipt of a message that is not outbound. direction: %s", p.Direction)
		return nil
	}
	if _, err := uuid.FromString(p.ID); err != nil {
		log.Debugf("Ignoring the receipt: the message id is not a uuid.")
		return nil
	}
	if len(p.To) == 0 {
		log.Debugf("Ignoring the receipt: it has no recipient.")
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, receiptBudget)
	defer cancel()

	since := h.utilHandler.TimeNow().Add(-receiptWindow)
	msgs, err := h.db.MessageListByProviderReference(ctx, p.ID, since)
	if err != nil {
		return fmt.Errorf("%w: could not look up the message: %w", ErrReceiptRetry, err)
	}

	var m *message.Message
	var t target.Target
	for _, c := range msgs {
		for _, ct := range c.Targets {
			if ct.ProviderReferenceID == p.ID {
				m, t = c, ct
				break
			}
		}
		if m != nil {
			break
		}
	}
	if m == nil {
		promReceiptUnmatchedTotal.Inc()
		log.Debugf("No message matches the receipt.")
		return nil
	}

	// the recipient entry of this target: a receipt can list several recipients.
	destination, _ := commonaddress.NormalizeTarget(t.Destination.Type, t.Destination.Target)
	status := ""
	found := false
	for _, to := range p.To {
		normalized, _ := commonaddress.NormalizeTarget(commonaddress.TypeTel, to.PhoneNumber)
		if digitsOf(normalized) == digitsOf(destination) {
			status = to.Status
			found = true
			break
		}
	}
	if !found {
		log.Warnf("No recipient of the receipt equals the destination of the target. message_id: %s", m.ID)
		return nil
	}

	newStatus := mapTelnyxStatus(status)
	if newStatus == "" {
		log.Debugf("The status of the receipt does not change the target. status: %s", status)
		return nil
	}

	changed, err := h.updateTargets(ctx, m.ID, updateOpts{maxTries: callbackWriteTries, budget: receiptBudget}, func(targets []target.Target) error {
		for j := range targets {
			if targets[j].ProviderReferenceID != p.ID {
				continue
			}

			if statusRank(newStatus) <= statusRank(targets[j].Status) {
				return errNoChange
			}

			targets[j].Status = newStatus
			targets[j].TMUpdate = h.utilHandler.TimeNow()
			if p.Parts > 0 {
				targets[j].ProviderParts = p.Parts
			}
			return nil
		}

		return errNoChange
	})
	if err != nil {
		return fmt.Errorf("%w: could not update the target: %w", ErrReceiptRetry, err)
	}
	if !changed {
		log.Debugf("The receipt changes nothing. message_id: %s", m.ID)
		return nil
	}

	h.publishUpdated(ctx, m.ID)

	return nil
}
