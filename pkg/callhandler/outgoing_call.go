package callhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

const (
	constVoIPBINDomainSuffix = ".sip.voipbin.net"

	constTransportUDP = "udp"
	constTransportTCP = "tcp"
	constTransportTLS = "tls"
	constTransportWS  = "ws"
	constTransportWSS = "wss"
)

// CreateCallOutgoing creates a call for outgoing
func (h *callHandler) CreateCallOutgoing(id uuid.UUID, userID uint64, flowID uuid.UUID, source address.Address, destination address.Address) (*call.Call, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"id":          id,
		"user":        userID,
		"flow":        flowID,
		"source":      source,
		"destination": destination,
	})
	log.Debug("Creating a call for outgoing.")

	// create active-flow
	af, err := h.reqHandler.FlowActvieFlowPost(id, flowID)
	if err != nil {
		af = &activeflow.ActiveFlow{}
		log.Errorf("Could not get an active flow for outgoing call. Created dummy active flow. This call will be hungup. call: %s, flow: %s, err: %v", id, flowID, err)
	}
	log.Debugf("Created active-flow. active-flow: %v", af)

	channelID := uuid.Must(uuid.NewV4()).String()
	cTmp := &call.Call{
		ID:          id,
		UserID:      userID,
		ChannelID:   channelID,
		FlowID:      flowID,
		Type:        call.TypeFlow,
		Status:      call.StatusDialing,
		Direction:   call.DirectionOutgoing,
		WebhookURI:  af.WebhookURI,
		Source:      source,
		Destination: destination,
		Action:      af.CurrentAction,
		TMCreate:    getCurTime(),
	}

	// create a call
	c, err := h.createCall(ctx, cTmp)
	if err != nil {
		log.Errorf("Could not create a call for outgoing call. err: %v", err)
		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, getCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}

		return nil, err
	}

	// get a endpoint destination
	endpointDst, err := h.getEndpointDestination(destination)
	if err != nil {
		log.Errorf("Could not create a destination endpoint. err: %v", err)

		// hangup
		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, getCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}
		return nil, err
	}

	// get sdp-transport
	sdpTransport := h.getEndpointSDPTransport(endpointDst)
	log.Debugf("Endpoint detail. endpoint_destination: %s, sdp_transport: %s", endpointDst, sdpTransport)

	// create a source endpoint
	var endpointSrc string
	if source.Type == address.TypeTel {
		endpointSrc = source.Target
	} else {
		endpointSrc = fmt.Sprintf("\"%s\" <sip:%s>", source.Name, source.Target)
	}

	// set app args
	appArgs := fmt.Sprintf("context=%s,call_id=%s", contextOutgoingCall, c.ID)

	// set variables
	variables := map[string]string{
		"CALLERID(all)":                         endpointSrc,
		"PJSIP_HEADER(add,VBOUT-SDP_Transport)": sdpTransport,
	}

	// create a channel
	if err := h.reqHandler.AstChannelCreate(requesthandler.AsteriskIDCall, channelID, appArgs, endpointDst, "", "", "", variables); err != nil {
		log.Errorf("Could not create a channel for outgoing call. err: %v", err)

		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, getCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}
		return nil, err
	}

	return c, nil
}

// getEndpointDestination returns corresponded endpoint's destination address for Asterisk's dialing
func (h *callHandler) getEndpointDestination(destination address.Address) (string, error) {

	var res string

	// create a destination endpoint
	// if the type is tel type, uses default gw
	if destination.Type == address.TypeTel {
		res = fmt.Sprintf("pjsip/%s/sip:%s@%s;transport=%s", pjsipEndpointOutgoing, destination.Target, trunkTelnyx, constTransportUDP)
		return res, nil
	}

	// destination is normal sip address.
	tmp := strings.Split(destination.Target, ";")
	if strings.HasSuffix(tmp[0], constVoIPBINDomainSuffix) == false {
		endpoint := destination.Target
		if strings.HasPrefix(endpoint, "sip") == false && strings.HasPrefix(endpoint, "sips:") == false {
			endpoint = "sip:" + endpoint
		}

		res = fmt.Sprintf("pjsip/%s/%s", pjsipEndpointOutgoing, endpoint)
		return res, nil
	}

	// get endpoint
	// trim the sip: or sips:
	endpoint := destination.Target
	endpoint = strings.TrimPrefix(endpoint, "sip:")
	endpoint = strings.TrimPrefix(endpoint, "sips:")

	// get contacts
	contacts, err := h.reqHandler.RMV1ContactsGet(endpoint)
	if err != nil {
		return "", fmt.Errorf("could not get contacts info. target: err: %v", err)
	}

	if len(contacts) == 0 {
		return "", fmt.Errorf("no available contact")
	}

	// we need only one
	contact := strings.ReplaceAll(contacts[0].URI, "^3B", ";")
	res = fmt.Sprintf("pjsip/%s/%s", pjsipEndpointOutgoing, contact)

	return res, nil
}

// getEndpointSDPTransport returns corresponded sdp-transport
func (h *callHandler) getEndpointSDPTransport(endpointDestination string) string {

	// webrtc allows only UDP/TLS/RTP/SAVPF
	if strings.Contains(endpointDestination, "transport=ws") || strings.Contains(endpointDestination, "transport=wss") {
		return "UDP/TLS/RTP/SAVPF"
	}

	return "RTP/AVP"
}
