package callhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"

	commonaddress	"gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

const (
	constTransportUDP = "udp"
	constTransportTCP = "tcp" //nolint:deadcode,varcheck
	constTransportTLS = "tls" //nolint:deadcode,varcheck
	constTransportWS  = "ws"  //nolint:deadcode,varcheck
	constTransportWSS = "wss" //nolint:deadcode,varcheck
)

// CreateCallsOutgoing creates multiple outgoing calls.
func (h *callHandler) CreateCallsOutgoing(ctx context.Context, customerID, flowID, masterCallID uuid.UUID, source commonaddress.Address, destinations []commonaddress.Address) ([]*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CreateCallsOutgoing",
		"customer_id": customerID,
		"flow_id":     flowID,
		"source":      source,
	})

	res := []*call.Call{}
	for _, destination := range destinations {
		callID := uuid.Must(uuid.NewV4())
		log.WithField("destination", destination).Debugf("Creating an outgoing call. call_id: %s, destination_type: %s, destination_target: %s", callID, destination.Type, destination.Target)

		switch destination.Type {
		case commonaddress.TypeSIP, commonaddress.TypeTel:
			c, err := h.CreateCallOutgoing(ctx, callID, customerID, flowID, uuid.Nil, masterCallID, source, destination)
			if err != nil {
				log.Errorf("Could not create an outgoing call. err: %v", err)
				continue
			}
			log.WithField("call", c).Debugf("Created outgoing call. call_id: %s, destination_type: %s, destination_target: %s", callID, destination.Type, destination.Target)

			res = append(res, c)

		case commonaddress.TypeAgent:
			calls, err := h.createCallOutgoingAgent(ctx, customerID, flowID, masterCallID, source, destination)
			if err != nil {
				log.Errorf("Could not create an outgoing call to the agent. err: %v", err)
				continue
			}
			log.WithField("calls", calls).Debugf("Created outgoing call to the agent. destination_type: %s, destination_target: %s", destination.Type, destination.Target)

			res = append(res, calls...)
		}
	}

	return res, nil
}

// CreateCallOutgoing creates a call for outgoing
func (h *callHandler) CreateCallOutgoing(ctx context.Context, id, customerID, flowID, activeflowID, masterCallID uuid.UUID, source commonaddress.Address, destination commonaddress.Address) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"funcs":          "CreateCallOutgoing",
		"id":             id,
		"customer_id":    customerID,
		"flow":           flowID,
		"activeflow_id":  activeflowID,
		"master_call_id": masterCallID,
		"source":         source,
		"destination":    destination,
	})
	log.Debug("Creating a call for outgoing.")

	// check destination type
	if destination.Type != commonaddress.TypeSIP && destination.Type != commonaddress.TypeTel {
		return nil, fmt.Errorf("the destination type must be sip or tel")
	}

	// create active-flow
	af, err := h.reqHandler.FMV1ActiveflowCreate(ctx, activeflowID, flowID, fmactiveflow.ReferenceTypeCall, id)
	if err != nil {
		af = &fmactiveflow.Activeflow{}
		log.Errorf("Could not get an active flow for outgoing call. Created dummy active flow. This call will be hungup. call: %s, flow: %s, err: %v", id, flowID, err)
	}
	log.Debugf("Created active-flow. active-flow: %v", af)

	channelID := uuid.Must(uuid.NewV4()).String()
	cTmp := &call.Call{
		ID:           id,
		CustomerID:   customerID,
		ChannelID:    channelID,
		FlowID:       flowID,
		ActiveFlowID: af.ID,
		Type:         call.TypeFlow,
		Status:       call.StatusDialing,
		Direction:    call.DirectionOutgoing,
		Source:       source,
		Destination:  destination,
		Action:       af.CurrentAction,

		TMCreate: dbhandler.GetCurTime(),
	}

	// create a call
	c, err := h.create(ctx, cTmp)
	if err != nil {
		log.Errorf("Could not create a call for outgoing call. err: %v", err)
		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, dbhandler.GetCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}

		return nil, err
	}

	// set variables
	if errVariables := h.setVariablesCall(ctx, c); errVariables != nil {
		log.Errorf("Could not set variables. err: %v", errVariables)
		return nil, errVariables
	}

	if masterCallID != uuid.Nil {
		tmp, errChained := h.ChainedCallIDAdd(ctx, masterCallID, c.ID)
		if errChained != nil {
			// could not add the chained call id. but this is minor issue compare to the creating a call.
			// so just keep moving.
			log.Errorf("Could not add the chained call id. But keep moving on. master_call_id: %s, call_id: %s err: %v", masterCallID, c.ID, errChained)
		}
		log.WithField("call", tmp).Debugf("Added chained call id. master_call_id: %s, call_id: %s", masterCallID, c.ID)
	}

	// get a endpoint destination
	dialURI, err := h.getDialURI(ctx, destination)
	if err != nil {
		log.Errorf("Could not create a destination endpoint. err: %v", err)

		// hangup
		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, dbhandler.GetCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}
		return nil, err
	}

	// get sdp-transport
	sdpTransport := h.getEndpointSDPTransport(dialURI)
	log.Debugf("Endpoint detail. endpoint_destination: %s, sdp_transport: %s", dialURI, sdpTransport)

	// create a source endpoint
	var endpointSrc string
	if source.Type == commonaddress.TypeTel {
		endpointSrc = source.Target
	} else {
		endpointSrc = fmt.Sprintf("\"%s\" <sip:%s>", source.TargetName, source.Target)
	}

	// set app args
	appArgs := fmt.Sprintf("context=%s,call_id=%s", ContextOutgoingCall, c.ID)

	// set variables
	variables := map[string]string{
		"CALLERID(all)":                         endpointSrc,
		"PJSIP_HEADER(add,VBOUT-SDP_Transport)": sdpTransport,
	}

	// create a channel
	if err := h.reqHandler.AstChannelCreate(ctx, requesthandler.AsteriskIDCall, channelID, appArgs, dialURI, "", "", "", variables); err != nil {
		log.Errorf("Could not create a channel for outgoing call. err: %v", err)

		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, dbhandler.GetCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}
		return nil, err
	}

	return c, nil
}

// getDialURITel returns dial uri of the given tel type destination.
func (h *callHandler) getDialURITel(ctx context.Context, destination commonaddress.Address) (string, error) {
	res := fmt.Sprintf("pjsip/%s/sip:%s@%s;transport=%s", pjsipEndpointOutgoing, destination.Target, trunkTelnyx, constTransportUDP)
	return res, nil
}

// getDialURISIP returns dial uri of the given sip type destination.
func (h *callHandler) getDialURISIP(ctx context.Context, destination commonaddress.Address) (string, error) {
	endpoint := destination.Target
	if !strings.HasPrefix(destination.Target, "sip:") && !strings.HasPrefix(destination.Target, "sips:") {
		endpoint = "sip:" + endpoint
	}

	res := fmt.Sprintf("pjsip/%s/%s", pjsipEndpointOutgoing, endpoint)
	return res, nil
}

// getDialURIEndpoint returns dial uri of the given extension type destination.
func (h *callHandler) getDialURIEndpoint(ctx context.Context, destination commonaddress.Address) (string, error) {

	// get contacts
	contacts, err := h.reqHandler.RMV1ContactGets(ctx, destination.Target)
	if err != nil {
		return "", fmt.Errorf("could not get contacts info. target: err: %v", err)
	}

	if len(contacts) == 0 {
		return "", fmt.Errorf("no available contact")
	}

	ct := contacts[0]
	tmp := strings.ReplaceAll(ct.URI, "^3B", ";")
	res := fmt.Sprintf("pjsip/%s/%s", pjsipEndpointOutgoing, tmp)

	return res, nil
}

// getDialURI returns the given destination address's dial URI for Asterisk's dialing
func (h *callHandler) getDialURI(ctx context.Context, destination commonaddress.Address) (string, error) {

	switch destination.Type {
	case commonaddress.TypeTel:
		return h.getDialURITel(ctx, destination)

	case commonaddress.TypeEndpoint:
		return h.getDialURIEndpoint(ctx, destination)

	case commonaddress.TypeSIP:
		return h.getDialURISIP(ctx, destination)

	default:
		return "", fmt.Errorf("unsupported address type")
	}
}

// getEndpointSDPTransport returns corresponded sdp-transport
func (h *callHandler) getEndpointSDPTransport(endpointDestination string) string {

	// webrtc allows only UDP/TLS/RTP/SAVPF
	if strings.Contains(endpointDestination, "transport=ws") || strings.Contains(endpointDestination, "transport=wss") {
		return "UDP/TLS/RTP/SAVPF"
	}

	return "RTP/AVP"
}

// CreateCallOutgoingAgent creates an outgoing call to the agent
func (h *callHandler) createCallOutgoingAgent(ctx context.Context, customerID, flowID, masterCallID uuid.UUID, source commonaddress.Address, destination commonaddress.Address) ([]*call.Call, error) {

	log := logrus.WithFields(logrus.Fields{
		"func":        "CreateCallOutgoingAgent",
		"customer_id": customerID,
	})

	// get agent id
	agentID := uuid.FromStringOrNil(destination.Target)
	agentDial, err := h.reqHandler.AMV1AgentDial(ctx, agentID, &source, flowID, masterCallID)
	if err != nil {
		log.Errorf("Could not create an outgoing call to agent. err: %v", err)
		return nil, err
	}
	log.WithField("agent_dial", agentDial).Debugf("Created an agent dial. agent_dial_id: %s", agentDial.ID)

	res := []*call.Call{}
	for _, callID := range agentDial.AgentCallIDs {
		c, err := h.Get(ctx, callID)
		if err != nil {
			log.Errorf("Could not get call info. err: %v", err)
			continue
		}

		res = append(res, c)
	}

	return res, nil
}
