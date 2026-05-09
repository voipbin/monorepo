package callhandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	nmnumber "monorepo/bin-number-manager/models/number"

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
	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/outboundconfighandler"
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
	anonymous string,
	metadata map[string]interface{},
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
		"anonymous":       anonymous,
		"metadata":        metadata,
	})

	resCalls := []*call.Call{}
	resGroupcalls := []*groupcall.Groupcall{}
	for _, destination := range destinations {
		switch {
		case destination.Type == commonaddress.TypeSIP || destination.Type == commonaddress.TypeTel:
			c, err := h.CreateCallOutgoing(ctx, uuid.Nil, customerID, flowID, uuid.Nil, masterCallID, uuid.Nil, source, destination, earlyExecution, connect, anonymous, metadata)
			if err != nil {
				log.WithField("destination", destination).Errorf("Could not create an outgoing call. destination_type: %s, err: %v", destination.Type, err)
				continue
			}
			log.WithField("call", c).Debugf("Created outgoing call. call_id: %s, destination_type: %s, destination_target: %s", c.ID, destination.Type, destination.Target)

			resCalls = append(resCalls, c)

		case h.groupcallHandler.IsGroupcallTypeAddress(&destination):
			gc, err := h.createCallsOutgoingGroupcall(ctx, customerID, flowID, masterCallID, &source, &destination, anonymous)
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

	if len(resCalls) == 0 && len(resGroupcalls) == 0 {
		return nil, nil, fmt.Errorf("all destinations failed to create outgoing calls")
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
	anonymous string,
	metadata map[string]interface{},
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
		"anonymous":                     anonymous,
		"metadata":                      metadata,
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

	// fetch customer info
	cu, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get customer info")
	}
	log.WithField("customer", cu).Debugf("Retrieved customer info. customer_id: %s", cu.ID)

	// embed rtp_debug in call metadata at creation time so status.go doesn't need to re-fetch the customer.
	// guard: if rtp_debug is already set (e.g. forced by providercallhandler), preserve it even when
	// the customer has RTPDebug=false — do not overwrite/clear an already-set flag.
	if _, alreadySet := metadata[call.MetadataKeyRTPDebug]; !alreadySet {
		if cu.Metadata.RTPDebug {
			if metadata == nil {
				metadata = map[string]any{}
			}
			metadata[call.MetadataKeyRTPDebug] = true
		}
	}

	// validate outgoing call permission (customer status + identity verification)
	if err := h.validateOutgoingCallPermission(ctx, cu, destination); err != nil {
		return nil, err
	}

	// validate customer's account balance
	if validBalance := h.ValidateCustomerBalance(ctx, id, customerID, call.DirectionOutgoing, source, destination); !validBalance {
		log.Debugf("Could not pass the balance validation. customer_id: %s", customerID)
		return nil, fmt.Errorf("could not pass the balance validation")
	}

	// Fetch outbound config once for all non-internal customers.
	// Used for: codec embedding (SIP only), whitelist + source validation (PSTN).
	// Internal system IDs (IDCallManager, IDAIManager, etc.) skip this block entirely —
	// they have no OutboundConfig row and must not be gated by whitelist or codec injection.
	var outboundCfg *outboundconfig.OutboundConfig
	if !cucustomer.IsInternalSystemID(customerID) {
		var cfgErr error
		outboundCfg, cfgErr = h.outboundConfigHandler.GetByCustomerID(ctx, customerID)
		if cfgErr != nil {
			log.Errorf("Could not get outbound config; rejecting call (fail-closed). err: %v", cfgErr)
			outboundconfighandler.IncFetchError("db_error")
			return nil, fmt.Errorf("could not get outbound config: %w", cfgErr)
		}
		// Codec embedding is destination-type-specific.
		// PSTN trunks negotiate codecs directly with the carrier via SDP; injecting a
		// codec header into PSTN calls overrides that negotiation, which is incorrect.
		switch destination.Type {
		case commonaddress.TypeSIP:
			metadata = embedCodecs(metadata, outboundCfg)
		case commonaddress.TypeTel:
			// no codec embedding for PSTN
		}
	}

	// PSTN-only: whitelist + source number validation.
	// outboundCfg is nil for internal system IDs — ValidateDestination returns true
	// for internal callers regardless of config (bypass path in validate.go).
	if destination.Type == commonaddress.TypeTel {
		if !h.ValidateDestination(ctx, customerID, outboundCfg, destination) {
			log.Infof("Outbound destination not in whitelist. customer_id: %s", customerID)
			country := h.getCountry(ctx, destination.Target)
			h.notifyHandler.PublishEvent(ctx, call.EventTypeCallOutboundWhitelistRejected, map[string]interface{}{
				"customer_id":         customerID,
				"call_id":             id,
				"destination_country": country,
			})
			return nil, outboundconfig.ErrDestinationNotWhitelisted
		}
	}

	// get dialroutes
	dialroutes := []rmroute.Route{}
	dialrouteID := uuid.Nil
	if destination.Type == commonaddress.TypeTel {
		var err error
		dialroutes, err = h.getDialroutes(ctx, customerID, &destination, metadata)
		if err != nil || len(dialroutes) == 0 {
			log.Errorf("Could not get the dialroute. err: %v", err)
			return nil, errors.Wrap(err, "could not get the dialroutes")
		}
		dialrouteID = dialroutes[0].ID
	}

	// create activeflow
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, activeflowID, customerID, flowID, fmactiveflow.ReferenceTypeCall, id, uuid.Nil)
	if err != nil {
		af = &fmactiveflow.Activeflow{}
		log.Errorf("Could not get an active flow for outgoing call. Created dummy active flow. This call will be hungup. call: %s, flow: %s, err: %v", id, flowID, err)
	}
	log.Debugf("Created active-flow. active-flow: %v", af)

	// create channel id
	channelID := h.utilHandler.UUIDCreate().String()

	// validate and resolve the source address for outgoing call
	s := h.getValidatedSourceForOutgoingCall(ctx, source, destination, cu, outboundCfg, metadata)
	if s == nil {
		log.Errorf("No valid source number available for outgoing call.")
		if af.ID != uuid.Nil {
			if _, errStop := h.reqHandler.FlowV1ActiveflowStop(ctx, af.ID); errStop != nil {
				log.Errorf("Could not stop orphaned activeflow. activeflow_id: %s, err: %v", af.ID, errStop)
			}
		}
		return nil, fmt.Errorf("no valid source number available for outgoing call")
	}

	// normalize anonymous flag: only "yes" and "no" are valid; everything else defaults to "auto"
	anonymousOption := call.AnonymousOption(anonymous)
	switch anonymousOption {
	case call.AnonymousOptionYes, call.AnonymousOptionNo:
		// valid, use as-is
	default:
		if anonymous != "" && anonymous != string(call.AnonymousOptionAuto) {
			log.Infof("Invalid anonymous option provided, defaulting to auto. anonymous: %s", anonymous)
		}
		anonymousOption = call.AnonymousOptionAuto
	}

	// resolve anonymous flag
	// TODO: when anonymousOption == AnonymousOptionAuto, inherit from incoming channel's SIP Privacy header
	// (check channel.StasisDataTypeSIPPrivacy). Currently "auto" defaults to not anonymous.
	// IMPORTANT: when implementing auto-inherit, "no" must explicitly opt OUT of anonymous (never anonymize),
	// while "auto" inherits from the incoming call. Today both resolve the same way (not anonymous),
	// but they must diverge once auto-inherit is implemented.
	resolvedAnonymous := anonymousOption == call.AnonymousOptionYes
	log.Debugf("Resolved anonymous flag. input: %s, normalized: %s, resolved: %v", anonymous, anonymousOption, resolvedAnonymous)

	// create data
	data := map[call.DataType]string{
		call.DataTypeEarlyExecution:            strconv.FormatBool(earlyExecution),
		call.DataTypeExecuteNextMasterOnHangup: strconv.FormatBool(executeNextMasterOnHangup),
		call.DataTypeAnonymous:                 strconv.FormatBool(resolvedAnonymous),
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

		metadata,
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
	if err := h.createChannelOutgoing(ctx, res); err != nil {
		log.Errorf("Could not create channel. err: %v", err)
		return nil, err
	}

	return res, nil
}

// getDialURITel returns a dialTarget for the given tel type destination.
// Prefix/postfix from the Provider wrap the user part of the URI;
// tech_headers, codecs, and providerID are populated from the matched provider.
func (h *callHandler) getDialURITel(ctx context.Context, c *call.Call) (*dialTarget, error) {
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
		return nil, fmt.Errorf("no available dialroute left")
	}

	pr, err := h.reqHandler.RouteV1ProviderGet(ctx, providerID)
	if err != nil {
		log.Errorf("Could not get provider info. err: %v", err)
		return nil, err
	}

	userPart := pr.TechPrefix + c.Destination.Target + pr.TechPostfix
	res := fmt.Sprintf("pjsip/%s/sip:%s@%s;transport=%s", pjsipEndpointOutgoing, userPart, pr.Hostname, constTransportUDP)

	return &dialTarget{
		URI:         res,
		TechHeaders: pr.TechHeaders,
		Codecs:      pr.Codecs,
		ProviderID:  pr.ID,
	}, nil
}

// getDialURISIP returns a dialTarget for the given sip type destination.
func (h *callHandler) getDialURISIP(ctx context.Context, c *call.Call) (*dialTarget, error) {
	endpoint := c.Destination.Target
	if !strings.HasPrefix(c.Destination.Target, "sip:") && !strings.HasPrefix(c.Destination.Target, "sips:") {
		endpoint = "sip:" + endpoint
	}

	res := fmt.Sprintf("pjsip/%s/%s", pjsipEndpointOutgoing, endpoint)
	return &dialTarget{URI: res}, nil
}

// getDialURISIPDirect returns a dialTarget for the given sip type destination via the direct endpoint.
func (h *callHandler) getDialURISIPDirect(ctx context.Context, c *call.Call) (*dialTarget, error) {
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
		return nil, fmt.Errorf("wrong destination uri")
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
	return &dialTarget{URI: res}, nil
}

// getDialURI returns a dialTarget for Asterisk's dialing.
// For provider (tel) paths, TechHeaders, Codecs, and ProviderID are populated.
// For SIP paths, only URI is set (TechHeaders nil, Codecs empty, ProviderID uuid.Nil).
func (h *callHandler) getDialURI(ctx context.Context, c *call.Call) (*dialTarget, error) {

	switch c.Destination.Type {
	case commonaddress.TypeTel:
		return h.getDialURITel(ctx, c)

	case commonaddress.TypeSIP:
		if strings.Contains(c.Destination.Target, "transport=ws") {
			return h.getDialURISIPDirect(ctx, c)
		}
		return h.getDialURISIP(ctx, c)

	default:
		return nil, fmt.Errorf("unsupported address type for get dial uri")
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
	anonymous string,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "createCallsOutgoingGroupcall",
		"customer_id":    customerID,
		"flow_id":        flowID,
		"master_call_id": masterCallID,
		"source":         source,
		"destination":    destination,
		"anonymous":      anonymous,
	})

	// start groupcall
	res, err := h.groupcallHandler.Start(ctx, uuid.Nil, customerID, flowID, source, []commonaddress.Address{*destination}, masterCallID, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, anonymous)
	if err != nil {
		log.Errorf("Could not start the groupcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not start the groupcall.")
	}
	log.WithField("groulcall", res).Debugf("Created groupcall. groupcall_id: %s", res.ID)

	return res, nil
}

// getDialroutes generates dialroutes for outgoing call.
//
// If metadata contains MetadataKeyRouteProviderIDs (a []interface{} of string UUIDs after JSON
// round-trip), the parsed provider IDs are forwarded to route-manager as targetProviderIDs.
// Route-manager then returns synthetic routes targeted at those providers, bypassing the
// normal customer/default route merging. See Task 5 / Task 8 in the NOJIRA-call-metadata-
// route-provider-ids design doc.
//
// Malformed UUID strings are logged and skipped — the admin may still have valid provider IDs
// in the array, so we don't fail the whole call over one typo.
func (h *callHandler) getDialroutes(ctx context.Context, customerID uuid.UUID, destination *commonaddress.Address, metadata map[string]interface{}) ([]rmroute.Route, error) {
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

	// extract optional route_provider_ids from metadata.
	// expected shape after JSON round-trip through Call.Metadata: []interface{} of string UUIDs.
	//
	// Fail fast if the key is present but yields zero valid UUIDs — silently
	// falling through to normal routing would mis-route an admin test call.
	var targetProviderIDs []uuid.UUID
	if raw, ok := metadata[call.MetadataKeyRouteProviderIDs]; ok {
		arr, ok := raw.([]interface{})
		if !ok {
			log.Errorf("route_provider_ids metadata is not a []interface{}: %T", raw)
			return nil, fmt.Errorf("route_provider_ids metadata has invalid shape: %T", raw)
		}
		for _, v := range arr {
			s, ok := v.(string)
			if !ok {
				log.Warnf("Skipping non-string entry in route_provider_ids metadata: %v", v)
				continue
			}
			id, errParse := uuid.FromString(s)
			if errParse != nil {
				log.Warnf("Skipping invalid provider ID in route_provider_ids metadata: %s, err: %v", s, errParse)
				continue
			}
			targetProviderIDs = append(targetProviderIDs, id)
		}
		if len(arr) > 0 && len(targetProviderIDs) == 0 {
			log.Errorf("route_provider_ids metadata contained no valid UUIDs, refusing to fall through to normal routing")
			return nil, fmt.Errorf("route_provider_ids contained no valid UUIDs")
		}
	}

	// send request
	filters := map[rmroute.Field]any{
		rmroute.FieldCustomerID: customerID,
		rmroute.FieldTarget:     target,
	}
	res, err := h.reqHandler.RouteV1DialrouteList(ctx, filters, targetProviderIDs)
	if err != nil {
		log.Errorf("Could not get dialroutes. err: %v", err)
		return nil, err
	}

	return res, nil
}

// createChannelOutgoing creates a new channel for outgoing call
func (h *callHandler) createChannelOutgoing(ctx context.Context, c *call.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "createChannelOutgoing",
		"call_id": c.ID,
	})

	// get dial target (URI + provider tech headers)
	target, err := h.getDialURI(ctx, c)
	if err != nil {
		log.Errorf("Could not create a destination endpoint. err: %v", err)
		return err
	}

	// set channel variables — tech_headers first so system-set headers
	// (transport, CALLERID, PAI when anonymous) overwrite on collision.
	// mergeTechHeaders additionally enforces the reserved-key denylist.
	channelVariables := map[string]string{}
	techApplied, techSkipped := mergeTechHeaders(channelVariables, target.TechHeaders, log)

	transport := getDestinationTransport(target.URI)
	setChannelVariableTransport(channelVariables, transport)
	anonymous := c.Data[call.DataTypeAnonymous] == "true"
	if err := setChannelVariablesCallerID(channelVariables, c, anonymous); err != nil {
		log.Errorf("Could not set caller ID variables. err: %v", err)
		return err
	}
	setChannelVariableCodecs(channelVariables, c.Metadata)
	setProviderCodecs(channelVariables, target.Codecs)
	if target.Codecs != "" {
		log.Debugf("Provider codec applied for dial attempt. provider_id: %s, codecs: %s",
			target.ProviderID, target.Codecs)
	}

	if techApplied > 0 || techSkipped > 0 {
		log.Infof("Applied provider tech config. headers_applied=%d headers_skipped=%d",
			techApplied, techSkipped)
	}

	log.Debugf("Endpoint detail. endpoint_destination: %s, variables: %v, anonymous: %v", target.URI, channelVariables, anonymous)

	// set app args
	appArgs := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s,%s=%s",
		channel.StasisDataTypeContextType, channel.ContextTypeCall,
		channel.StasisDataTypeContext, channel.ContextCallOutgoing,
		channel.StasisDataTypeCallID, c.ID,
		channel.StasisDataTypeTransport, transport,
		channel.StasisDataTypeDirection, channel.DirectionOutgoing,
	)

	// create a channel
	tmp, err := h.channelHandler.StartChannel(ctx, requesthandler.AsteriskIDCall, c.ChannelID, appArgs, target.URI, "", "", "", channelVariables)
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

	if errCreate := h.createChannelOutgoing(ctx, cc); errCreate != nil {
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

// setChannelVariablesCallerID sets the outgoing call's caller ID variables.
// When anonymous is true, the caller ID is set to anonymous via CALLERID(pres)=prohib (RFC 3323),
// and the P-Asserted-Identity carries the real source number for carrier routing (RFC 3325).
// Returns an error if anonymous is requested but the source is invalid for the PAI header,
// because silently falling back to normal caller ID would be a privacy violation.
// Note: Do NOT add PJSIP_HEADER(add,From) — Asterisk generates the From header from CALLERID fields,
// and adding a second From header violates RFC 3261 §8.1.1.3 (causes Kamailio sanity check to drop the INVITE).
func setChannelVariablesCallerID(variables map[string]string, c *call.Call, anonymous bool) error {

	if anonymous && c.Destination.Type == commonaddress.TypeTel {
		// Source.Target must be a valid E.164 number for PAI header.
		// getValidatedSourceForOutgoingCall ensures a valid source, but guard against
		// unexpected call paths (e.g., route failover).
		if c.Source.Target == "" || !strings.HasPrefix(c.Source.Target, "+") {
			return fmt.Errorf("anonymous caller ID requested but source target is invalid for PAI header. source_target: %s", c.Source.Target)
		}

		// RFC 3323: anonymous From header
		variables["CALLERID(name)"] = "Anonymous"
		variables["CALLERID(num)"] = "anonymous"
		variables["CALLERID(pres)"] = "prohib"

		// RFC 3325: PAI carries the real source number so the PSTN carrier can route/bill correctly.
		variables["PJSIP_HEADER(add,P-Asserted-Identity)"] = fmt.Sprintf("<tel:%s>", c.Source.Target)
		variables["PJSIP_HEADER(add,Privacy)"] = "id"
		return nil
	}

	variables["CALLERID(name)"] = c.Source.TargetName
	variables["CALLERID(num)"] = c.Source.Target
	return nil
}

// getValidatedSourceForOutgoingCall validates and resolves the source address for an outgoing call.
// For tel-type destinations, the source number must:
// 1. Have a valid E.164 format (starts with "+")
// 2. Belong to the customer as a normal (non-virtual) number
// If either condition fails, it falls back to OutboundConfig.DefaultOutgoingSourceNumberID
// (re-validated against number-manager because stored values may be stale).
// If no valid source can be determined, it returns nil.
// For non-tel destinations, the source is returned as-is.
//
// If metadata contains MetadataKeySkipSourceValidation == true, all of the above is
// bypassed and the caller-supplied source is returned verbatim. This is used by
// internal admin-test flows (e.g., the provider-call endpoint) where the admin needs
// to preserve a source the provider's carrier has pre-authorized.
func (h *callHandler) getValidatedSourceForOutgoingCall(
	ctx context.Context,
	source commonaddress.Address,
	destination commonaddress.Address,
	cu *cucustomer.Customer,
	outboundCfg *outboundconfig.OutboundConfig,
	metadata map[string]interface{},
) *commonaddress.Address {
	log := logrus.WithFields(logrus.Fields{
		"func":        "getValidatedSourceForOutgoingCall",
		"source":      source,
		"destination": destination,
	})

	// non-tel destinations don't need source validation
	if destination.Type != commonaddress.TypeTel {
		return &source
	}

	// metadata opts out of customer-ownership validation entirely
	// (used by internal admin-test flows — see MetadataKeySkipSourceValidation).
	// Logged at Debug to avoid surfacing the source phone number (PII) in
	// production Info logs. The skip flag is set server-side by trusted code,
	// so its presence itself is not a signal operators need to alert on.
	if skip, ok := metadata[call.MetadataKeySkipSourceValidation].(bool); ok && skip {
		log.Debug("Source validation skipped per metadata.")
		return &source
	}

	// Defensive guard: cu is never nil at this point because CreateCallOutgoing
	// returns an error if the customer-manager fetch fails. This guard exists in
	// case a future call path is added that bypasses that check; if so, this
	// branch fails open and skips source validation, which would be a security
	// regression. Adding such a path requires explicitly setting MetadataKeySkipSourceValidation
	// or supplying a valid customer-owned source.
	if cu == nil {
		log.Infof("Customer info not available. Skipping source number validation.")
		return &source
	}
	log = log.WithField("customer_id", cu.ID)

	// validate source: must be E.164 format and belong to the customer as a normal number
	if strings.HasPrefix(source.Target, "+") {
		filters := map[nmnumber.Field]any{
			nmnumber.FieldCustomerID: cu.ID,
			nmnumber.FieldNumber:     source.Target,
			nmnumber.FieldType:       nmnumber.TypeNormal,
			nmnumber.FieldStatus:     nmnumber.StatusActive,
			nmnumber.FieldDeleted:    false,
		}
		nums, err := h.reqHandler.NumberV1NumberList(ctx, "", 1, filters)
		if err != nil {
			log.Errorf("Could not validate source number ownership. source: %s, err: %v", source.Target, err)
		} else if len(nums) > 0 {
			log.Debugf("Source number validated. source: %s", source.Target)
			return &source
		} else {
			log.Infof("Source number is not a valid normal number owned by the customer. source: %s", source.Target)
		}
	} else {
		log.Infof("Source number is not in E.164 format. source: %s", source.Target)
	}

	// source is not valid; fall back to OutboundConfig.DefaultOutgoingSourceNumberID.
	// outboundCfg may legitimately be nil (non-tel destination, internal-system caller,
	// or transient fetch failure handled at the call site).
	if outboundCfg == nil || outboundCfg.DefaultOutgoingSourceNumberID == uuid.Nil {
		log.Infof("No valid source number available. Rejecting call.")
		return nil
	}

	// Re-validate against the same filters as the caller-supplied path above —
	// OutboundConfig values may be stale (number released, ownership changed,
	// soft-deleted, or status flipped after the operator set the default).
	filters := map[nmnumber.Field]any{
		nmnumber.FieldCustomerID: cu.ID,
		nmnumber.FieldID:         outboundCfg.DefaultOutgoingSourceNumberID,
		nmnumber.FieldType:       nmnumber.TypeNormal,
		nmnumber.FieldStatus:     nmnumber.StatusActive,
		nmnumber.FieldDeleted:    false,
	}
	nums, err := h.reqHandler.NumberV1NumberList(ctx, "", 1, filters)
	if err != nil || len(nums) == 0 {
		log.WithField("number_id", outboundCfg.DefaultOutgoingSourceNumberID).
			Errorf("Default outgoing source number is not valid (released, wrong customer, inactive, or virtual). err: %v", err)
		return nil
	}
	defaultNum := nums[0]
	log.WithField("number", defaultNum).Debugf("Applying outbound_config default outgoing source number. number_id: %s, number: %s", defaultNum.ID, defaultNum.Number)
	return &commonaddress.Address{
		Type:       commonaddress.TypeTel,
		Target:     defaultNum.Number,
		TargetName: defaultNum.Number,
	}
}
