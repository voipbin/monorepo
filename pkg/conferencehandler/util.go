package conferencehandler

// func (h *conferHandler) kickOutCall(id, callID uuid.UUID) error {
// 	ctx := context.Background()

// 	log := log.WithFields(
// 		log.Fields{
// 			"conference": id.String(),
// 			"call":       callID.String(),
// 		})
// 	log.Debugf("kicking out the call from the conference.")

// 	call, err := h.db.CallGet(ctx, callID)
// 	if err != nil {
// 		log.Errorf("Could not get call. err: %v", err)
// 		return err
// 	}

// 	// get channel
// 	channel, err := h.db.ChannelGet(ctx, call.AsteriskID, call.ChannelID)
// 	if err != nil {
// 		log.Errorf("Could not get channel. err: %v", err)
// 		return err
// 	}

// 	// kick out the call's from the bridge
// 	if err := h.reqHandler.AstBridgeRemoveChannel(channel.AsteriskID, channel.BridgeID, channel.ID); err != nil {
// 		log.WithFields(
// 			logrus.Fields{
// 				"bridge":  channel.BridgeID,
// 				"channel": channel.ID,
// 			}).Errorf("Could not kick out the call from the conference. err: %v", err)
// 		return err
// 	}

// 	return nil
// }
