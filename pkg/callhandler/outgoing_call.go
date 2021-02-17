package callhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

const (
	constVoIPBINDomainSuffix = ".sip.voipbin.net"

	constTransportUDP = "UDP"
	constTransportTCP = "TCP"
	constTransportTLS = "TLS"
)

// CreateCallOutgoing creates a call for outgoing
func (h *callHandler) CreateCallOutgoing(id uuid.UUID, userID uint64, flowID uuid.UUID, source call.Address, destination call.Address) (*call.Call, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"id":          id,
		"user":        userID,
		"flow":        flowID,
		"source":      source,
		"destination": destination,
	})
	log.Debug("Creating a call for outgoing.")

	channelID := uuid.Must(uuid.NewV4()).String()
	cTmp := &call.Call{
		ID:          id,
		UserID:      userID,
		ChannelID:   channelID,
		FlowID:      flowID,
		Type:        call.TypeFlow,
		Status:      call.StatusDialing,
		Direction:   call.DirectionOutgoing,
		Source:      source,
		Destination: destination,
		Action: action.Action{
			ID: action.IDBegin,
		},
		TMCreate: getCurTime(),
	}

	// create a call
	if err := h.createCall(ctx, cTmp); err != nil {
		log.Errorf("Could not create a call for outgoing call. err: %v", err)
		return nil, err
	}

	// get created call
	c, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get a created call for outgoing call. err: %v", err)
		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, getCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}
		return nil, err
	}

	// create active-flow
	af, err := h.reqHandler.FlowActvieFlowPost(c.ID, flowID)
	if err != nil {
		log.Errorf("Could not create an active-flow for outgoing call. err: %v", err)

		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, getCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}
		return nil, err
	}
	log.Debugf("Created active-flow. active-flow: %v", af)

	// create a destination endpoint
	endpointDst, endpointTransport, err := h.getEndpointDestination(destination)
	if err != nil {
		log.Errorf("Could not create a destination endpoint. err: %v", err)

		// hangup
		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, getCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}
		return nil, err
	}

	// create a source endpoint
	var endpointSrc string
	if source.Type == call.AddressTypeTel {
		endpointSrc = source.Target
	} else {
		endpointSrc = fmt.Sprintf("\"%s\" <sip:%s>", source.Name, source.Target)
	}

	// set app args
	appArgs := fmt.Sprintf("context=%s,call_id=%s", contextOutgoingCall, c.ID)

	// set variables
	variables := map[string]string{
		"CALLERID(all)": endpointSrc,
		"SIPADDHEADER0": "VBOUT-Transport: " + endpointTransport,
		"SIPADDHEADER1": "VBOUT-SDP_Transport: " + "RTP/AVP",
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
func (h *callHandler) getEndpointDestination(destination call.Address) (string, string, error) {

	var res string

	// create a destination endpoint
	if destination.Type == call.AddressTypeTel {
		res = fmt.Sprintf("pjsip/%s/sip:%s@%s", pjsipEndpointOutgoing, destination.Target, trunkTelnyx)
		return res, constTransportUDP, nil
	}

	// check voipbin suffix
	if strings.HasSuffix(destination.Target, constVoIPBINDomainSuffix) == false {
		res = fmt.Sprintf("pjsip/%s/sip:%s", pjsipEndpointOutgoing, destination.Target)
		return res, constTransportUDP, nil
	}

	// get contacts
	contacts, err := h.reqHandler.RMV1ContactsGet(destination.Target)
	if err != nil {
		return "", "", fmt.Errorf("could not get contacts info. target: err: %v", err)
	}

	// has no contacts
	if len(contacts) <= 0 {
		return "", "", fmt.Errorf("not found registered contact address")
	}

	// we need only one
	contact := contacts[0]

	// sip:test11@211.178.226.108:35551^3Btransport=UDP^3Brinstance=8a1f981a77f30a22
	var transport string
	slices := strings.Split(contact.URI, "^3B")
	for _, s := range slices[1:] {

		tmp := strings.ToUpper(s)
		kv := strings.Split(tmp, "=")

		if len(kv) < 2 || kv[0] != "TRANSPORT" {
			continue
		}

		switch kv[1] {
		case constTransportTCP:
			transport = constTransportTCP

		case constTransportTLS:
			transport = constTransportTLS

		case constTransportUDP:
			fallthrough
		default:
			transport = constTransportUDP
		}
	}

	addr := strings.Split(contact.URI, "^3B")[0]
	res = fmt.Sprintf("pjsip/%s/%s", pjsipEndpointOutgoing, addr)

	return res, transport, nil
}
