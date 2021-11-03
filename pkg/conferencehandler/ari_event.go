package conferencehandler

// // ARIStasisStart handles StasisStart ARI event for conference types.
// func (h *conferenceHandler) ARIStasisStart(cn *channel.Channel, data map[string]interface{}) error {

// 	log := logrus.WithFields(logrus.Fields{
// 		"channel_id":  cn.ID,
// 		"asterisk_id": cn.AsteriskID,
// 		"data":        cn.Data,
// 	})

// 	confContext := data["context"]
// 	switch confContext {
// 	case contextConferenceIncoming:
// 		return h.ariStasisStartContextIncoming(cn, data)

// 	case contextConferenceOutgoing:
// 		log.Errorf("Currently, we don't support conference outgoing context. Something was wrong. context: %s", confContext)
// 		_ = h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
// 		return fmt.Errorf("unsupported conference context type. context: %s", confContext)

// 	default:
// 		log.Errorf("Unsuppurted context type. context: %s", confContext)
// 		_ = h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
// 		return fmt.Errorf("unsupported conference context type. context: %s", confContext)
// 	}
// }

// // ariStasisStartContextIncoming handles the call which has CONTEXT=conf-in in the StasisStart argument.
// func (h *conferenceHandler) ariStasisStartContextIncoming(cn *channel.Channel, data map[string]interface{}) error {

// 	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeConf)); err != nil {
// 		_ = h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
// 		return fmt.Errorf("could not set channel var. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
// 	}

// 	// answer the call. it is safe to call this for answered call.
// 	if err := h.reqHandler.AstChannelAnswer(cn.AsteriskID, cn.ID); err != nil {
// 		logrus.Errorf("Could not answer the call. err: %v", err)
// 		_ = h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
// 		return err
// 	}

// 	if err := h.reqHandler.AstBridgeAddChannel(cn.AsteriskID, cn.DestinationNumber, cn.ID, "", false, false); err != nil {
// 		_ = h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
// 		return fmt.Errorf("could not put the channel to the bridge. id: %s, asterisk: %s, bridge: %s, err: %v", cn.ID, cn.AsteriskID, cn.DestinationNumber, err)
// 	}

// 	return nil
// }

// // ARIChannelLeftBridge handles ChannelLeftBridge ARI event for conference types.
// func (h *conferenceHandler) ARIChannelLeftBridge(cn *channel.Channel, br *bridge.Bridge) error {
// 	if cn.Type != channel.TypeConf {
// 		return nil
// 	}

// 	return h.leaved(cn, br)
// }
