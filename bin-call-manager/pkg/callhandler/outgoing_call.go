package callhandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	rmroute "monorepo/bin-route-manager/models/route"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/ttacon/libphonenumber"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/models/groupcall"
)

const (
	constTransportUDP = "udp"
	constTransportTCP = "tcp" //nolint:deadcode,varcheck
	constTransportTLS = "tls" //nolint:deadcode,varcheck
	constTransportWS  = "ws"  //nolint:deadcode,varcheck
	constTransportWSS = "wss" //nolint:deadcode,varcheck
)

// CreateCallsOutgoing creates multiple outgoing calls.
func (h *callHandler) CreateCallsOutgoing(
	ctx context.Context,
	customerID uuid.UUID,
	flowID uuid.UUID,
	masterCallID uuid.UUID,
	source commonaddress.Address,
	destinations []commonaddress.Address,
	earlyExecution bool,
	connect bool,
) ([]*call.Call, []*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "CreateCallsOutgoing",
		"customer_id":     customerID,
		"flow_id":         flowID,
		"master_call_id":  masterCallID,
		"source":          source,
		"destinations":    destinations,
		"early_execution": earlyExecution,
		"connect":         connect,
	})

	resCalls := []*call.Call{}
	resGroupcalls := []*groupcall.Groupcall{}
	for _, destination := range destinations {
		switch {
		case destination.Type == commonaddress.TypeSIP || destination.Type == commonaddress.TypeTel:
			c, err := h.CreateCallOutgoing(ctx, uuid.Nil, customerID, flowID, uuid.Nil, masterCallID, uuid.Nil, source, destination, earlyExecution, connect)
			if err != nil {
				log.WithField("destination", destination).Errorf("Could not create an outgoing call. destination_type: %s, err: %v", destination.Type, err)
				continue
			}
			log.WithField("call", c).Debugf("Created outgoing call. call_id: %s, destination_type: %s, destination_target: %s", c.ID, destination.Type, destination.Target)

			resCalls = append(resCalls, c)

		case h.groupcallHandler.IsGroupcallTypeAddress(&destination):
			gc, err := h.createCallsOutgoingGroupcall(ctx, customerID, flowID, masterCallID, &source, &destination)
			if err != nil {
				log.Errorf("Could not create outgoing groupcall. err: %v", err)
				continue
			}
			log.WithField("groupcall", gc).Debugf("Created outgoing groupcall. groupcall_id: %s, destination_type: %s, destination_target: %s", gc.ID, destination.Type, destination.Target)

			resGroupcalls = append(resGroupcalls, gc)

		default:
			log.WithField("destination", destination).Errorf("Unsupported destination type. destination_type: %s", destination.Type)
		}
	}

	return resCalls, resGroupcalls, nil
}

// CreateCallOutgoing creates a call for outgoing
func (h *callHandler) CreateCallOutgoing(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	flowID uuid.UUID,
	activeflowID uuid.UUID,
	masterCallID uuid.UUID,
	groupcallID uuid.UUID,
	source commonaddress.Address,
	destination commonaddress.Address,
	earlyExecution bool,
	executeNextMasterOnHangup bool,
) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"funcs":                         "CreateCallOutgoing",
		"id":                            id,
		"customer_id":                   customerID,
		"flow":                          flowID,
		"activeflow_id":                 activeflowID,
		"master_call_id":                masterCallID,
		"groupcall_id":                  groupcallID,
		"source":                        source,
		"destination":                   destination,
		"early_execution":               earlyExecution,
		"execute_next_master_on_hangup": executeNextMasterOnHangup,
	})
	log.Debug("Creating a call for outgoing.")

	if id == uuid.Nil {
		id = h.utilHandler.UUIDCreate()
		log = log.WithField("id", id)
		log.Debugf("The given call id is empty. Create new call id. call_id: %s", id)
	}

	// check destination type
	if destination.Type != commonaddress.TypeSIP && destination.Type != commonaddress.TypeTel {
		log.Errorf("Wrong destination type to call. destination_type: %s", destination.Type)
		return nil, fmt.Errorf("the destination type must be sip or tel")
	}

	// validate customer's account balance
	if validBalance := h.ValidateCustomerBalance(ctx, id, customerID, call.DirectionOutgoing, source, destination); !validBalance {
		log.Debugf("Could not pass the balance validation. customer_id: %s", customerID)
		return nil, fmt.Errorf("could not pass the balance validation")
	}

	// validate destination
	if validDestination := h.ValidateDestination(ctx, customerID, destination); !validDestination {
		log.Debugf("Could not pass the destination validation. customer_id: %s", customerID)
		return nil, fmt.Errorf("could not pass the destination validation")
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
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, activeflowID, customerID, flowID, fmactiveflow.ReferenceTypeCall, id)
	if err != nil {
		af = &fmactiveflow.Activeflow{}
		log.Errorf("Could not get an active flow for outgoing call. Created dummy active flow. This call will be hungup. call: %s, flow: %s, err: %v", id, flowID, err)
	}
	log.Debugf("Created active-flow. active-flow: %v", af)

	// create channel id
	channelID := h.utilHandler.UUIDCreate().String()

	// get source address for outgoing
	s := getSourceForOutgoingCall(&source, &destination)

	// create data
	data := map[call.DataType]string{
		call.DataTypeEarlyExecution:            strconv.FormatBool(earlyExecution),
		call.DataTypeExecuteNextMasterOnHangup: strconv.FormatBool(executeNextMasterOnHangup),
	}

	// get address owner info
	ownerType, ownerID, err := h.getAddressOwner(ctx, customerID, &destination)
	if err != nil {
		// we could not find owner info, but just write the log here.
		log.Errorf("Could not get address owner info. err: %v", err)
	}

	// create a call
	res, err := h.Create(
		ctx,

		id,
		customerID,
		ownerType,
		ownerID,

		channelID,
		"",

		flowID,
		af.ID,
		uuid.Nil,
		call.TypeFlow,
		groupcallID,

		s,
		&destination,
		call.StatusDialing,
		data,

		af.CurrentAction,
		call.DirectionOutgoing,

		dialrouteID,
		dialroutes,
	)
	if err != nil {
		log.Errorf("Could not create a call for outgoing call. err: %v", err)
		return nil, err
	}

	// set variables
	if errVariables := h.setVariablesCall(ctx, res); errVariables != nil {
		log.Errorf("Could not set variables. err: %v", errVariables)
		return nil, errVariables
	}

	if masterCallID != uuid.Nil {
		tmp, errChained := h.ChainedCallIDAdd(ctx, masterCallID, res.ID)
		if errChained != nil {
			// could not add the chained call id. but this is minor issue compare to the creating a call.
			// so just keep moving.
			log.Errorf("Could not add the chained call id. But keep moving on. master_call_id: %s, call_id: %s err: %v", masterCallID, res.ID, errChained)
		}
		log.WithField("call", tmp).Debugf("Added chained call id. master_call_id: %s, call_id: %s", masterCallID, res.ID)
	}

	// create a channel for the call
	if err := h.createChannel(ctx, res); err != nil {
		log.Errorf("Could not create channel. err: %v", err)
		return nil, err
	}

	return res, nil
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

// getDialURISIP returns dial uri of the given sip type destination.
func (h *callHandler) getDialURISIPDirect(ctx context.Context, c *call.Call) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "getDialURISIPDirect",
		"destination_target": c.Destination.Target,
	})

	endpointTarget := c.Destination.Target
	if !strings.HasPrefix(c.Destination.Target, "sip:") && !strings.HasPrefix(c.Destination.Target, "sips:") {
		endpointTarget = "sip:" + endpointTarget
	}

	tmpTargets := strings.Split(endpointTarget, ";")
	if len(tmpTargets) < 1 {
		return "", fmt.Errorf("wrong destination uri")
	}

	// get target host/port
	porxyHost := ""
	for _, tmp := range tmpTargets {
		if strings.HasPrefix(tmp, "outbound_proxy=") {
			porxyHost, _ = strings.CutPrefix(tmp, "outbound_proxy=")
		}
	}
	log.Debugf("Found outbound proxy host info. outbound_proxy: %s", porxyHost)

	res := fmt.Sprintf("pjsip/%s%s/%s", pjsipEndpointOutgoingDirect, porxyHost, endpointTarget)
	return res, nil
}

// getDialURI returns the given destination address's dial URI for Asterisk's dialing
func (h *callHandler) getDialURI(ctx context.Context, c *call.Call) (string, error) {

	switch c.Destination.Type {
	case commonaddress.TypeTel:
		return h.getDialURITel(ctx, c)

	case commonaddress.TypeSIP:
		if strings.Contains(c.Destination.Target, "transport=ws") {
			// websocket transport(WebRTC)
			return h.getDialURISIPDirect(ctx, c)
		}
		return h.getDialURISIP(ctx, c)

	default:
		// for address type endpoint, conference, ... are possible to return the multiple address.
		// so we can not handle those address types are here.
		return "", fmt.Errorf("unsupported address type for get dial uri")
	}
}

// getGroupcallRingMethod returns groupcall ring method of the given destination
func (h *callHandler) getGroupcallRingMethod(ctx context.Context, destination commonaddress.Address) (groupcall.RingMethod, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getGroupcallRingMethod",
		"destination": destination,
	})

	switch destination.Type {
	case commonaddress.TypeAgent:
		// the destination type is agent. we need to check the agent's ring method.
		// get agent
		ag, err := h.reqHandler.AgentV1AgentGet(ctx, uuid.FromStringOrNil(destination.Target))
		if err != nil {
			log.Errorf("Could not get agent info. err: %v", err)
			return groupcall.RingMethodNone, errors.Wrap(err, "could not get agent info")
		}
		log.WithField("agent", ag).Debugf("Found agent info. ring_method: %s", ag.RingMethod)

		// check the agent's ring method
		if ag.RingMethod == amagent.RingMethodLinear {
			return groupcall.RingMethodLinear, nil
		}

		return groupcall.RingMethodRingAll, nil

	default:
		log.Debugf("Selecting default groupcall ringmethod. ring_method: %s", groupcall.RingMethodRingAll)
		return groupcall.RingMethodRingAll, nil
	}
}

// createCallsOutgoingGroupcallOld creates an outgoing call to the endpoint type destination
func (h *callHandler) createCallsOutgoingGroupcall(
	ctx context.Context,
	customerID uuid.UUID,
	flowID uuid.UUID,
	masterCallID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "createCallsOutgoingGroupcall",
		"customer_id":    customerID,
		"flow_id":        flowID,
		"master_call_id": masterCallID,
		"source":         source,
		"destination":    destination,
	})

	// start groupcall
	res, err := h.groupcallHandler.Start(ctx, uuid.Nil, customerID, flowID, source, []commonaddress.Address{*destination}, masterCallID, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers)
	if err != nil {
		log.Errorf("Could not start the groupcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not start the groupcall.")
	}
	log.WithField("groulcall", res).Debugf("Created groupcall. groupcall_id: %s", res.ID)

	return res, nil
}

// getDialroutes generates dialroutes for outgoing call
func (h *callHandler) getDialroutes(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]rmroute.Route, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDialroutes",
		"customer_id": customerID,
		"destination": destination,
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
		return err
	}

	// set channel variables
	channelVariables := map[string]string{}
	transport := getDestinationTransport(dialURI)
	setChannelVariableTransport(channelVariables, transport)
	setChannelVariablesCallerID(channelVariables, c)
	log.Debugf("Endpoint detail. endpoint_destination: %s, variables: %v", dialURI, channelVariables)

	// set app args
	appArgs := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s",
		channel.StasisDataTypeContextType, channel.TypeCall,
		channel.StasisDataTypeContext, channel.ContextCallOutgoing,
		channel.StasisDataTypeCallID, c.ID,
		channel.StasisDataTypeTransport, transport,
	)

	// create a channel
	tmp, err := h.channelHandler.StartChannel(ctx, requesthandler.AsteriskIDCall, c.ChannelID, appArgs, dialURI, "", "", "", channelVariables)
	if err != nil {
		log.Errorf("Could not create a channel for outgoing call. err: %v", err)
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
	channelID := h.utilHandler.UUIDCreate().String()

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

// getDestinationTransport returns given destination's transport
func getDestinationTransport(endpointDestination string) channel.SIPTransport {

	if strings.Contains(endpointDestination, "transport=wss") {
		return channel.SIPTransportWSS
	} else if strings.Contains(endpointDestination, "transport=ws") {
		return channel.SIPTransportWS
	} else if strings.Contains(endpointDestination, "transport=tcp") {
		return channel.SIPTransportTCP
	} else if strings.Contains(endpointDestination, "transport=tls") {
		return channel.SIPTransportTLS
	} else {
		return channel.SIPTransportUDP
	}
}

// setChannelVariableTransport sets the outgoit call's media transport type
func setChannelVariableTransport(variables map[string]string, transport channel.SIPTransport) {

	switch transport {
	case channel.SIPTransportWS, channel.SIPTransportWSS:
		variables["PJSIP_HEADER(add,"+common.SIPHeaderSDPTransport+")"] = "UDP/TLS/RTP/SAVPF"
		return

	default:
		variables["PJSIP_HEADER(add,"+common.SIPHeaderSDPTransport+")"] = "RTP/AVP"
		return
	}
}

// setChannelVariablesCallerID sets the outgoit call's caller
func setChannelVariablesCallerID(variables map[string]string, c *call.Call) {

	if c.Destination.Type == commonaddress.TypeTel && c.Source.Target == "anonymous" {
		// we can't verify the caller's id. setting the default anonymous caller id
		variables["CALLERID(pres)"] = "prohib"
		variables["PJSIP_HEADER(add,P-Asserted-Identity)"] = "\"Anonymous\" <sip:+821100000001@pstn.voipbin.net>"
		variables["PJSIP_HEADER(add,Privacy)"] = "id"

		return
	}

	variables["CALLERID(name)"] = c.Source.TargetName
	variables["CALLERID(num)"] = c.Source.Target
}

// getSourceForOutgoingCall returns a source address for outgoing call
func getSourceForOutgoingCall(source *commonaddress.Address, destination *commonaddress.Address) *commonaddress.Address {

	if destination.Type != commonaddress.TypeTel {
		// the only tel type destination need a source address chage
		return source
	}

	// validate source number
	if strings.HasPrefix(source.Target, "+") {
		return source
	}

	// invalid source address for the tel type destination. we need to set the caller id to the anonymous
	return &commonaddress.Address{
		Type:       source.Type,
		TargetName: "Anonymous",
		Target:     "anonymous",
	}
}
