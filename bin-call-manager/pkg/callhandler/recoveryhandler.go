package callhandler

//go:generate mockgen -package callhandler -destination ./mock_recoveryhandler.go -source recoveryhandler.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/jart/gosip/sip"
	"github.com/sirupsen/logrus"
)

type recoveryDetail struct {
	RequestURI   string
	Routes       string
	RecordRoutes string
	CallID       string

	FromDisplay string
	FromURI     string
	FromTag     string

	ToDisplay string
	ToURI     string
	ToTag     string

	CSeq int
}

type asteriskRole string

const (
	asteriskRoleUnknown asteriskRole = ""
	asteriskRoleUAC     asteriskRole = "uac" // User Agent Client
	asteriskRoleUAS     asteriskRole = "uas" // User Agent Server
)

type RecoveryHandler interface {
	GetRecoveryDetail(ctx context.Context, callID string) (*recoveryDetail, error)
}

type recoveryHandler struct {
	requestHandler requesthandler.RequestHandler

	httpClient      *http.Client
	homerAPIAddress string
	homerAuthToken  string
	loadBalancerIPs []string
}

var (
	defaultHomerSearchTimeRange = -24 * time.Hour // from 24 hours ago
)

// NewRecoveryHandler creates a new RecoveryHandler instance
func NewRecoveryHandler(
	requestHandler requesthandler.RequestHandler,

	homerAPIAddress string,
	homerAuthToken string,
	loadBalancerIPs []string,
) RecoveryHandler {
	return &recoveryHandler{
		requestHandler: requestHandler,

		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},

		homerAPIAddress: homerAPIAddress,
		homerAuthToken:  homerAuthToken,

		loadBalancerIPs: loadBalancerIPs,
	}
}

func (h *recoveryHandler) GetRecoveryDetail(ctx context.Context, callID string) (*recoveryDetail, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "GetRecoveryDetail",
		"callID": callID,
	})

	if h.homerAPIAddress == "" || h.homerAuthToken == "" {
		return nil, fmt.Errorf("missing Homer API address or auth token")
	}

	sipMessages, err := h.getSIPMessages(ctx, callID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get SIP messages for call ID. call_id: %s", callID)
	}

	res, err := h.getRecoveryDetail(ctx, sipMessages)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get recovery details for call ID. call_id: %s", callID)
	}
	log.WithField("res", res).Debug("Found recovery details successfully")

	return res, nil
}

func (h *recoveryHandler) getRecoveryDetail(ctx context.Context, messages []*sip.Msg) (*recoveryDetail, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "getRecoveryDetails",
		"count": len(messages),
	})

	if len(messages) == 0 {
		return nil, errors.New("no SIP messages provided")
	}

	var firstInvite *sip.Msg
	var role asteriskRole

	// Find the first INVITE message
	for _, msg := range messages {
		if msg.Method == sip.MethodInvite {
			firstInvite = msg
			var err error
			role, err = h.determineRole(msg)
			if err != nil {
				log.Warnf("Error determining role for INVITE message: %v", err)
				continue
			} else if role != asteriskRoleUnknown {
				log.Debugf("Found INVITE message with role: %s", role)
				break
			} else {
				log.Debug("Found an INVITE message but could not determine its role.")
			}
		} else {
			log.Tracef("Skipping non-INVITE message: %s", msg.Method)
		}
	}

	if firstInvite == nil {
		return nil, errors.New("no INVITE message found in the SIP message list")
	}

	if role == asteriskRoleUnknown {
		return nil, errors.New("no INVITE message with a known role found")
	}

	res := &recoveryDetail{}
	requestURI, routes, recordRoutes := h.extractContactAndRoutes(messages, firstInvite, role)
	if requestURI == "" {
		return nil, errors.New("the request URI is missing")
	}
	res.RequestURI = requestURI

	listRoutes := strings.Split(routes, ",")
	if len(listRoutes) > 1 {
		res.Routes = strings.Join(listRoutes, ",")
		res.Routes = strings.TrimSpace(res.Routes)
	}

	listRecordRoutes := strings.Split(recordRoutes, ",")
	if len(listRecordRoutes) > 1 {
		res.RecordRoutes = strings.Join(listRecordRoutes, ",")
		res.RecordRoutes = strings.TrimSpace(res.RecordRoutes)
	}
	log.Debugf("Extracted request URI and routes. RequestURI: %s, Routes: %s, RecordRoutes: %s", res.RequestURI, res.Routes, res.RecordRoutes)

	lastMsg := messages[len(messages)-1]
	if errValidate := h.validateLastMessage(lastMsg); errValidate != nil {
		return nil, errValidate
	}

	res.CallID = lastMsg.CallID

	res.FromDisplay = lastMsg.From.Display
	res.FromURI = lastMsg.From.Uri.String()
	res.FromTag = lastMsg.From.Param.Get("tag").Value

	res.ToDisplay = lastMsg.To.Display
	res.ToURI = lastMsg.To.Uri.String()
	res.ToTag = lastMsg.To.Param.Get("tag").Value

	res.CSeq = lastMsg.CSeq + 1

	return res, nil
}

func (h *recoveryHandler) determineRole(firstMessage *sip.Msg) (asteriskRole, error) {
	if firstMessage == nil {
		return asteriskRoleUnknown, errors.New("first message is nil")
	}

	if !firstMessage.IsResponse() && firstMessage.Method == sip.MethodInvite {
		vboutHeader := firstMessage.XHeader.Get("VBOUT-SDP_Transport")
		if vboutHeader != nil {
			return asteriskRoleUAC, nil
		} else {
			return asteriskRoleUAS, nil
		}
	}

	return "", fmt.Errorf("first message is not an INVITE request, method: %s, isResponse: %t",
		firstMessage.Method, firstMessage.IsResponse())
}

func (h *recoveryHandler) extractContactAndRoutes(messages []*sip.Msg, firstInvite *sip.Msg, role asteriskRole) (string, string, string) {
	remoteContact := ""
	routes := ""
	recordRoutes := ""

	switch role {
	case asteriskRoleUAC:
		firstSuccessfulResponse := h.findFirstSuccessfulResponse(messages, firstInvite.CSeq)
		if firstSuccessfulResponse != nil {
			if contact := firstSuccessfulResponse.Contact; contact != nil {
				remoteContact = contact.Uri.String()
			} else if firstInvite.To != nil {
				remoteContact = firstInvite.To.Uri.String()
			}

			if firstSuccessfulResponse.RecordRoute != nil {
				routes = firstSuccessfulResponse.RecordRoute.Reversed().String()
				recordRoutes = firstSuccessfulResponse.RecordRoute.String()
			}
		} else if firstInvite.To != nil {
			remoteContact = firstInvite.To.Uri.String()
		}

	case asteriskRoleUAS:
		if contact := firstInvite.Contact; contact != nil {
			remoteContact = contact.Uri.String()
		} else if firstInvite.From != nil {
			remoteContact = firstInvite.From.Uri.String()
		}

		if firstInvite.RecordRoute != nil {
			routes = firstInvite.RecordRoute.String()
			recordRoutes = firstInvite.RecordRoute.String()
		}
	default:
		// Handle default case in the calling function. This should never happen, but leaving here as documentation
		return remoteContact, routes, recordRoutes
	}

	return remoteContact, routes, recordRoutes
}

func (h *recoveryHandler) findFirstSuccessfulResponse(messages []*sip.Msg, inviteCSeq int) *sip.Msg {
	for _, msg := range messages {
		if msg != nil && msg.IsResponse() && msg.Status >= 200 && msg.Status < 300 &&
			msg.CSeqMethod == sip.MethodInvite && msg.CSeq == inviteCSeq {
			return msg
		}
	}
	return nil
}

func (h *recoveryHandler) validateLastMessage(lastMsg *sip.Msg) error {
	if lastMsg == nil {
		return errors.New("last message is nil")
	}

	if lastMsg.CallID == "" {
		return errors.New("the Call-ID is missing")
	}

	if lastMsg.From == nil {
		return errors.New("the From header is missing")
	}

	if lastMsg.To == nil {
		return errors.New("the To header is missing")
	}

	return nil
}
