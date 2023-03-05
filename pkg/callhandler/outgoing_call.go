package callhandler

import (
	"context"
	"fmt"
	"strconv"
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
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
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
) ([]*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "CreateCallsOutgoing",
		"customer_id":  customerID,
		"flow_id":      flowID,
		"source":       source,
		"destinations": destinations,
	})

	res := []*call.Call{}
	for _, destination := range destinations {
		switch destination.Type {
		case commonaddress.TypeSIP, commonaddress.TypeTel:
			c, err := h.CreateCallOutgoing(ctx, uuid.Nil, customerID, flowID, uuid.Nil, masterCallID, uuid.Nil, source, destination, earlyExecution, connect)
			if err != nil {
				log.WithField("destination", destination).Errorf("Could not create an outgoing call. destination_type: %s, err: %v", destination.Type, err)
				continue
			}
			log.WithField("call", c).Debugf("Created outgoing call. call_id: %s, destination_type: %s, destination_target: %s", c.ID, destination.Type, destination.Target)

			res = append(res, c)

		case commonaddress.TypeEndpoint, commonaddress.TypeAgent:
			calls, err := h.createCallsOutgoingGroupcall(ctx, customerID, flowID, masterCallID, source, destination)
			if err != nil {
				log.WithField("destination", destination).Errorf("Could not create outgoing calls. destination_type: %s, err: %v", destination.Type, err)
				continue
			}
			log.WithField("calls", calls).Debugf("Created outgoing calls. destination_type: %s, destination_target: %s", destination.Type, destination.Target)

			res = append(res, calls...)

		default:
			log.WithField("destination", destination).Errorf("Unsupported destination type. destination_type: %s", destination.Type)
		}
	}

	return res, nil
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
		id = h.utilHandler.CreateUUID()
		log = log.WithField("id", id)
		log.Debugf("The given call id is empty. Create new call id. call_id: %s", id)
	}

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

	// get source address for outgoing
	s := getSourceForOutgoingCall(&source, &destination)

	// create data
	data := map[call.DataType]string{
		call.DataTypeEarlyExecution:            strconv.FormatBool(earlyExecution),
		call.DataTypeExecuteNextMasterOnHangup: strconv.FormatBool(executeNextMasterOnHangup),
	}

	// create a call
	res, err := h.Create(
		ctx,

		id,
		customerID,

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

// getDialURI returns the given destination address's dial URI for Asterisk's dialing
func (h *callHandler) getDialURI(ctx context.Context, c *call.Call) (string, error) {

	switch c.Destination.Type {
	case commonaddress.TypeTel:
		return h.getDialURITel(ctx, c)

	case commonaddress.TypeSIP:
		return h.getDialURISIP(ctx, c)

	default:
		// for address type endpoint, conference, ... are possible to return the multiple address.
		// so we can not handle those address types are here.
		return "", fmt.Errorf("unsupported address type for get dial uri")
	}
}

// createCallsOutgoingGroupcall creates an outgoing call to the endpoint type destination
func (h *callHandler) createCallsOutgoingGroupcall(
	ctx context.Context,
	customerID uuid.UUID,
	flowID uuid.UUID,
	masterCallID uuid.UUID,
	source commonaddress.Address,
	destination commonaddress.Address,
) ([]*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "createCallsOutgoingGroupcall",
		"customer_id":    customerID,
		"flow_id":        flowID,
		"master_call_id": masterCallID,
		"source":         source,
		"destination":    destination,
	})

	// get dial destinations
	dialDestinations, err := h.getDialDestinations(ctx, customerID, &destination)
	if err != nil {
		log.Errorf("Could not dial destination. err: %v", err)
		return nil, errors.Wrap(err, "Could not get dial destinations.")
	}

	if len(dialDestinations) == 0 {
		log.Debugf("No dial destination found. len: %d", len(dialDestinations))
		return nil, fmt.Errorf("no dial destination found")
	}
	log.WithField("dial_destinations", dialDestinations).Debugf("Found dial destinations for group dial. destination_type: %s", destination.Type)

	// generate call ids
	callIDs := []uuid.UUID{}
	for range dialDestinations {
		callID := h.utilHandler.CreateUUID()
		callIDs = append(callIDs, callID)
	}

	// create groupcall
	gd, err := h.createGroupcall(ctx, customerID, &destination, callIDs, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers)
	if err != nil {
		log.Errorf("Could not create groupcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not create groupcall.")
	}
	log.WithField("groupcall", gd).Debugf("Created groupcall. groupcall_id: %s", gd.ID)

	// create outgoing
	res := []*call.Call{}
	switch gd.RingMethod {
	case groupcall.RingMethodRingAll:
		for i, dialDestination := range dialDestinations {
			log.WithField("dial_destination", dialDestination).Debugf("Creating a new outgoing call. call_id: %s, target: %s", gd.CallIDs[i], dialDestination.Target)

			// we don't allow to earlyExecution(earlymedia) for groupcall.
			// this is very obvious. because if we allow the early media for groupcall, it will mess the media handle.
			// and we can not set the execute next master on hangup flag in the same reason.
			tmp, err := h.CreateCallOutgoing(ctx, gd.CallIDs[i], customerID, flowID, uuid.Nil, masterCallID, gd.ID, source, *dialDestination, false, false)
			if err != nil {
				log.WithField("dial_destination", dialDestination).Errorf("Could not create an outgoing call. destination_target: %s, err: %v", dialDestination.Target, err)
				continue
			}
			res = append(res, tmp)
		}

	case groupcall.RingMethodLinear:
		log.Errorf("Not imeplemented yet.")
		return nil, fmt.Errorf("not implemented yet")

	default:
		log.Errorf("Unsupported ring method type. ring_method: %s", gd.RingMethod)
		return nil, fmt.Errorf("unsupported ring method type")
	}

	return res, nil
}

// getDialroutes generates dialroutes for outgoing call
func (h *callHandler) getDialroutes(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]rmroute.Route, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDialroutes",
		"customer_id": customerID,
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
	setChannelVariableTransport(channelVariables, dialURI)
	setChannelVariablesCallerID(channelVariables, c)
	log.Debugf("Endpoint detail. endpoint_destination: %s, variables: %v", dialURI, channelVariables)

	// set app args
	appArgs := fmt.Sprintf("context=%s,call_id=%s", common.ContextOutgoingCall, c.ID)

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

// setChannelVariableTransport sets the outgoit call's media transport type
func setChannelVariableTransport(variables map[string]string, endpointDestination string) {

	// webrtc allows only UDP/TLS/RTP/SAVPF
	if strings.Contains(endpointDestination, "transport=ws") || strings.Contains(endpointDestination, "transport=wss") {
		variables["PJSIP_HEADER(add,VBOUT-SDP_Transport)"] = "UDP/TLS/RTP/SAVPF"
		return
	}

	variables["PJSIP_HEADER(add,VBOUT-SDP_Transport)"] = "RTP/AVP"
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

// getDialDestinations returns given destination's dial destinations.
func (h *callHandler) getDialDestinations(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDialDestinations",
		"customer_id": customerID,
		"destination": destination,
	})

	// get dial destinations
	mapDialDestination := map[commonaddress.Type]func(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]*commonaddress.Address, error){
		commonaddress.TypeEndpoint: h.getDialDestinationsAddressTypeEndpoint,
		commonaddress.TypeAgent:    h.getDialDestinationsAddressTypeAgent,
	}

	f, ok := mapDialDestination[destination.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported destination type")
	}

	res, err := f(ctx, customerID, destination)
	if err != nil {
		log.Errorf("Could not get dial uris. err: %v", err)
		return nil, errors.Wrap(err, "Could not get dial uris.")
	}

	return res, nil
}

// getDialDestinationsAddressTypeEndpoint returns destinations for address type endpoint.
func (h *callHandler) getDialDestinationsAddressTypeEndpoint(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDialDestinationsAddressTypeEndpoint",
		"customer_id": customerID,
		"destination": destination,
	})

	e, err := h.reqHandler.RegistrarV1ExtensionGetByEndpoint(ctx, destination.Target)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get extension info.")
	}

	// check the customer id
	if customerID != e.CustomerID {
		log.Debugf("The customer id is different. customer_id: %s, extension_customer_id: %s", customerID, e.CustomerID)
		return nil, fmt.Errorf("the customer id is different")
	}

	// get contacts
	contacts, err := h.reqHandler.RegistrarV1ContactGets(ctx, destination.Target)
	if err != nil {
		log.Errorf("Could not get contacts info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get contacts info.")
	}
	log.WithField("contacts", contacts).Debugf("Found contacts. len: %d", len(contacts))

	res := []*commonaddress.Address{}
	for _, contact := range contacts {
		uri := strings.ReplaceAll(contact.URI, "^3B", ";")
		tmp := &commonaddress.Address{
			Type:       commonaddress.TypeSIP,
			TargetName: destination.TargetName, // update the target name to the destination's target name
			Target:     uri,
		}

		res = append(res, tmp)
	}

	return res, nil
}

// getDialDestinationsAddressTypeAgent returns destinations for address type agent.
func (h *callHandler) getDialDestinationsAddressTypeAgent(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address) ([]*commonaddress.Address, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getDialDestinationsAddressTypeAgent",
		"destination": destination,
	})

	// get agnet info
	agID := uuid.FromStringOrNil(destination.Target)
	if agID == uuid.Nil {
		log.Errorf("Could not parse the agent id. agent_id: %s", destination.Target)
		return nil, fmt.Errorf("could not parse the agent id")
	}

	ag, err := h.reqHandler.AgentV1AgentGet(ctx, agID)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get agnet info.")
	}

	// check the customer id
	if customerID != ag.CustomerID {
		log.Debugf("The customer id is different. customer_id: %s, agent_customer_id: %s", customerID, ag.CustomerID)
		return nil, fmt.Errorf("the customer id is different")
	}

	res := []*commonaddress.Address{}
	for _, address := range ag.Addresses {
		// update address target name
		address.TargetName = destination.TargetName

		switch address.Type {
		case commonaddress.TypeTel, commonaddress.TypeSIP:
			res = append(res, &address)

		case commonaddress.TypeEndpoint:
			tmp, err := h.getDialDestinationsAddressTypeEndpoint(ctx, ag.CustomerID, &address)
			if err != nil {
				log.Errorf("Could not get destination address. err: %v", err)
				continue
			}
			res = append(res, tmp...)

		default:
			log.WithField("address", address).Errorf("Unsupported address type for agent outgoing. address_type: %s", address.Type)
		}
	}

	return res, nil
}
