package callhandler

import (
	"context"
	"fmt"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/jart/gosip/sip"
	"github.com/sirupsen/logrus"
)

type recoveryDetail struct {
	RequestURI string
	// Routes     []string
	Routes string
	CallID string

	// From        string
	FromDisplay string
	FromURI     string
	FromTag     string

	// To        string
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
	defaultFromTimestampMs = time.Now().Add(-24 * time.Hour).UnixMilli() // 24 hours ago
	defaultToTimestampMs   = time.Now().UnixMilli()                      // current time
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
	log.WithField("res", res).Debug("Recovery details extracted successfully")

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

	firstInvite := messages[0]
	role, err := h.determineRole(firstInvite)
	if err != nil {
		return nil, err
	} else if role == asteriskRoleUnknown {
		return nil, fmt.Errorf("first message is not an INVITE request, method: %s, isResponse: %t", firstInvite.Method, firstInvite.IsResponse())
	}
	log.Debugf("Determined role. role: %s", role)

	res := &recoveryDetail{}
	requestURI, routes := h.extractContactAndRoutes(messages, firstInvite, role)
	if requestURI == "" {
		return nil, errors.New("the request URI is missing")
	}
	res.RequestURI = requestURI
	res.Routes = routes
	log.Debugf("Extracted request URI and routes. RequestURI: %s, Routes: %s", res.RequestURI, res.Routes)

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

	// res.From = h.formatSIPAddress(lastMsg.From)
	// res.To = h.formatSIPAddress(lastMsg.To)
	res.CSeq = lastMsg.CSeq + 1
	// log.Debugf("Extracted last message details. callID: %s, from: %s, to: %s, cseq: %d", res.CallID, res.From, res.To, res.CSeq)

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

func (h *recoveryHandler) extractContactAndRoutes(messages []*sip.Msg, firstInvite *sip.Msg, role asteriskRole) (string, string) {
	remoteContact := ""
	routes := ""

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
		}
	default:
		// Handle default case in the calling function. This should never happen, but leaving here as documentation
		return remoteContact, routes
	}

	return remoteContact, routes
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

// func (h *recoveryHandler) formatSIPAddress(addr *sip.Addr) (string, string, string) {
// 	if addr.Display != "" && addr.Uri != nil {
// 		return fmt.Sprintf("\"%s\" <%s>;tag=%s", addr.Display, addr.Uri, addr.Param.Get("tag").Value)
// 	} else if addr != nil {
// 		return addr.String()
// 	}

// 	return ""
// }
