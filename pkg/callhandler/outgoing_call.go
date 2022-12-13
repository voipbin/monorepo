package callhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/ttacon/libphonenumber"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
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
		log.Errorf("Wrong destination type to call. destination_type: %s", destination.Type)
		return nil, fmt.Errorf("the destination type must be sip or tel")
	}

	// get dialroutes
	dialroutes := []rmroute.Route{}
	dialrouteID := uuid.Nil
	if destination.Type == commonaddress.TypeTel {
		var err error
		dialroutes, err = h.getDialroutes(ctx, customerID, &destination)
		if err != nil || len(dialroutes) == 0 {
			log.Errorf("Could not get the dialroute. err: %v", err)
			return nil, errors.Wrap(err, "could not get the dialroutes")
		}
		dialrouteID = dialroutes[0].ID
	}

	// create activeflow
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, activeflowID, flowID, fmactiveflow.ReferenceTypeCall, id)
	if err != nil {
		af = &fmactiveflow.Activeflow{}
		log.Errorf("Could not get an active flow for outgoing call. Created dummy active flow. This call will be hungup. call: %s, flow: %s, err: %v", id, flowID, err)
	}
	log.Debugf("Created active-flow. active-flow: %v", af)

	// create channel id
	channelID := h.utilHandler.CreateUUID().String()

	// create a call
	c, err := h.Create(
		ctx,

		id,
		customerID,

		"",
		channelID,
		"",

		flowID,
		af.ID,
		uuid.Nil,
		call.TypeFlow,

		uuid.Nil,
		[]uuid.UUID{},

		uuid.Nil,
		[]uuid.UUID{},

		&source,
		&destination,
		call.StatusDialing,
		map[string]string{},

		af.CurrentAction,
		call.DirectionOutgoing,

		dialrouteID,
		dialroutes,
	)
	if err != nil {
		log.Errorf("Could not create a call for outgoing call. err: %v", err)
		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, h.utilHandler.GetCurTime()); err != nil {
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

	// create a channel for the call
	if err := h.createChannel(ctx, c); err != nil {
		log.Errorf("Could not create channel. err: %v", err)
		return nil, err
	}

	return c, nil
}

// getDialURITel returns dial uri of the given tel type destination.
func (h *callHandler) getDialURITel(ctx context.Context, c *call.Call) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "getDialURITel",
		"call_id": c.ID,
	})

	providerID := uuid.Nil
	for _, dialroute := range c.Dialroutes {
		if dialroute.ID == c.DialrouteID {
			providerID = dialroute.ProviderID
			break
		}
	}

	if providerID == uuid.Nil {
		log.Debugf("No available dialroute left.")
		return "", fmt.Errorf("no available dialroute left")
	}

	// get provider info
	pr, err := h.reqHandler.RouteV1ProviderGet(ctx, providerID)
	if err != nil {
		log.Errorf("Could not get provider info. err: %v", err)
		return "", err
	}

	res := fmt.Sprintf("pjsip/%s/sip:%s@%s;transport=%s", pjsipEndpointOutgoing, c.Destination.Target, pr.Hostname, constTransportUDP)

	return res, nil
}

// getDialURISIP returns dial uri of the given sip type destination.
func (h *callHandler) getDialURISIP(ctx context.Context, c *call.Call) (string, error) {
	endpoint := c.Destination.Target
	if !strings.HasPrefix(c.Destination.Target, "sip:") && !strings.HasPrefix(c.Destination.Target, "sips:") {
		endpoint = "sip:" + endpoint
	}

	res := fmt.Sprintf("pjsip/%s/%s", pjsipEndpointOutgoing, endpoint)
	return res, nil
}

// getDialURIEndpoint returns dial uri of the given extension type destination.
func (h *callHandler) getDialURIEndpoint(ctx context.Context, c *call.Call) (string, error) {

	// get contacts
	contacts, err := h.reqHandler.RegistrarV1ContactGets(ctx, c.Destination.Target)
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
func (h *callHandler) getDialURI(ctx context.Context, c *call.Call) (string, error) {

	switch c.Destination.Type {
	case commonaddress.TypeTel:
		return h.getDialURITel(ctx, c)

	case commonaddress.TypeEndpoint:
		return h.getDialURIEndpoint(ctx, c)

	case commonaddress.TypeSIP:
		return h.getDialURISIP(ctx, c)

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
	agentDial, err := h.reqHandler.AgentV1AgentDial(ctx, agentID, &source, flowID, masterCallID)
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

// getDialroutes generates dialroutes for outgoing call
func (h *callHandler) getDialroutes(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]rmroute.Route, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "generateDialroutes",
	})

	if destination.Type != commonaddress.TypeTel {
		return []rmroute.Route{}, nil
	}

	// parse number
	n, err := libphonenumber.Parse(destination.Target, "US") // default country code is US.
	if err != nil {
		log.Errorf("Could not parse the libphonenumber. err: %v", err)
		return nil, err
	}
	target := fmt.Sprintf("+%d", *n.CountryCode)

	// send request
	res, err := h.reqHandler.RouteV1DialrouteGets(ctx, customerID, target)
	if err != nil {
		log.Errorf("Could not get dialroutes. err: %v", err)
		return nil, err
	}

	return res, nil
}

// createChannel creates a new channel for outgoing call
func (h *callHandler) createChannel(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "createChannel",
		"call_id": c.ID,
	})

	// get a endpoint destination
	dialURI, err := h.getDialURI(ctx, c)
	if err != nil {
		log.Errorf("Could not create a destination endpoint. err: %v", err)

		// hangup
		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, h.utilHandler.GetCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}
		return err
	}

	// get sdp-transport
	sdpTransport := h.getEndpointSDPTransport(dialURI)
	log.Debugf("Endpoint detail. endpoint_destination: %s, sdp_transport: %s", dialURI, sdpTransport)

	// create a source endpoint
	var endpointSrc string
	if c.Source.Type == commonaddress.TypeTel {
		endpointSrc = c.Source.Target
	} else {
		endpointSrc = fmt.Sprintf("\"%s\" <sip:%s>", c.Source.TargetName, c.Source.Target)
	}

	// set app args
	appArgs := fmt.Sprintf("context=%s,call_id=%s", ContextOutgoingCall, c.ID)

	// set variables
	variables := map[string]string{
		"CALLERID(all)":                         endpointSrc,
		"PJSIP_HEADER(add,VBOUT-SDP_Transport)": sdpTransport,
	}

	// create a channel
	tmp, err := h.reqHandler.AstChannelCreate(ctx, requesthandler.AsteriskIDCall, c.ChannelID, appArgs, dialURI, "", "", "", variables)
	if err != nil {
		log.Errorf("Could not create a channel for outgoing call. err: %v", err)

		if err := h.HangupWithReason(ctx, c, call.HangupReasonFailed, call.HangupByLocal, h.utilHandler.GetCurTime()); err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
		}
		return err
	}
	log.WithField("channel", tmp).Debugf("Created a new channel. channel_id: %s", tmp.ID)

	return nil
}

// createFailoverChannel creates a new channel for outgoing call(failover)
func (h *callHandler) createFailoverChannel(ctx context.Context, c *call.Call) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "createFailoverChannel",
		"call_id": c.ID,
	})

	// get next dialroute
	dialroute, err := h.getNextDialroute(ctx, c)
	if err != nil {
		log.Errorf("Could not get next dialroute. err: %v", err)
		return nil, err
	}
	dialrouteID := dialroute.ID

	// create a new channel id
	channelID := h.utilHandler.CreateUUID().String()

	// update call
	cc, err := h.updateForRouteFailover(ctx, c.ID, channelID, dialrouteID)
	if err != nil {
		log.Errorf("Could not update the call for route failover. err: %v", err)
		return nil, err
	}
	log.WithField("call", cc).Debugf("Updated call for route failover. call_id: %s", cc.ID)

	if errCreate := h.createChannel(ctx, cc); errCreate != nil {
		log.Errorf("Could not create a channel for routefailover. err: %v", err)
		return nil, errCreate
	}

	return cc, nil
}

// getNextDialroute returns the next available dialroute.
func (h *callHandler) getNextDialroute(ctx context.Context, c *call.Call) (*rmroute.Route, error) {
	// get next dialroute
	idx := 0
	for _, dialroute := range c.Dialroutes {
		if dialroute.ID == c.DialrouteID {
			break
		}
		idx++
	}
	if idx >= (len(c.Dialroutes) - 1) {
		// no more dialroute left
		return nil, fmt.Errorf("no more dialroute left to dial")
	}

	return &c.Dialroutes[idx+1], nil
}
