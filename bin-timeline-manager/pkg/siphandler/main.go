package siphandler

//go:generate mockgen -package siphandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-timeline-manager/models/sipmessage"
	"monorepo/bin-timeline-manager/pkg/homerhandler"
)

// SIPHandler interface for SIP message and PCAP operations.
type SIPHandler interface {
	GetSIPMessages(ctx context.Context, sipCallID string, fromTime, toTime time.Time) (*sipmessage.SIPMessagesResponse, error)
	GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error)
}

type sipHandler struct {
	homerHandler homerhandler.HomerHandler
}

// NewSIPHandler creates a new SIPHandler.
func NewSIPHandler(homerHandler homerhandler.HomerHandler) SIPHandler {
	return &sipHandler{
		homerHandler: homerHandler,
	}
}

// GetSIPMessages retrieves SIP messages for a given SIP call ID and time range,
// and builds a SIPMessagesResponse.
func (h *sipHandler) GetSIPMessages(ctx context.Context, sipCallID string, fromTime, toTime time.Time) (*sipmessage.SIPMessagesResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetSIPMessages",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"from_time": fromTime,
		"to_time":   toTime,
	}).Info("SIPHandler called - fetching SIP messages")

	messages, err := h.homerHandler.GetSIPMessages(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get SIP messages from Homer. err: %v", err)
		return nil, err
	}

	res := &sipmessage.SIPMessagesResponse{
		SIPCallID: sipCallID,
		Messages:  messages,
	}

	log.WithField("message_count", len(messages)).Debug("Successfully retrieved SIP messages.")

	return res, nil
}

// GetPcap retrieves PCAP data for a given SIP call ID and time range.
func (h *sipHandler) GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetPcap",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"from_time": fromTime,
		"to_time":   toTime,
	}).Info("SIPHandler called - fetching PCAP data")

	pcapData, err := h.homerHandler.GetPcap(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get PCAP data from Homer. err: %v", err)
		return nil, err
	}

	log.WithField("pcap_size", len(pcapData)).Debug("Successfully retrieved PCAP data.")

	return pcapData, nil
}
