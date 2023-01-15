package conferencehandler

// // Leave outs the call from the conference
// func (h *conferenceHandler) Leave(ctx context.Context, conferencecallID uuid.UUID) (*conferencecall.Conferencecall, error) {
// 	log := logrus.WithFields(
// 		logrus.Fields{
// 			"func":              "Leave",
// 			"conferencecall_id": conferencecallID,
// 		})
// 	log.Debugf("Leaving the call from the conference.")

// 	// update conferencecall status to leaving
// 	cc, err := h.conferencecallHandler.UpdateStatusLeaving(ctx, conferencecallID)
// 	if err != nil {
// 		log.Errorf("Could not update the conferencecall status. err: %v", err)
// 		return nil, err
// 	}
// 	log.WithField("conferencecall", cc).Debugf("Updated conferencecall info. conferencecall_id: %s", cc.ID)

// 	// get conference
// 	cf, err := h.Get(ctx, cc.ConferenceID)
// 	if err != nil {
// 		log.Errorf("Could not get conference info. conference_id: %s, err: %v", cf.ID, err)
// 		return nil, err
// 	}

// 	switch cc.ReferenceType {

// 	default:
// 		// send the kick request
// 		if err := h.reqHandler.CallV1ConfbridgeCallKick(ctx, cf.ConfbridgeID, cc.ReferenceID); err != nil {
// 			log.Errorf("Could not kick the call from the conference. err: %v", err)
// 			return nil, err
// 		}
// 	}

// 	return cc, nil
// }

// func (h *conferenceHandler) LeaveByReferenceID(ctx context.Context, conferenceID, referenceID uuid.UUID) (*conferencecall.Conferencecall, error) {

// 	log := logrus.WithFields(
// 		logrus.Fields{
// 			"func":          "LeaveByReferenceID",
// 			"conference_id": conferenceID,
// 			"reference_id":  referenceID,
// 		})
// 	log.Debugf("Leaving the call from the conference.")

// 	cc, err := h.conferencecallHandler.GetByReferenceID(ctx, referenceID)
// 	if err != nil {
// 		log.Errorf("Could not get conferencecall info. err: %v", err)
// 		return nil, err
// 	}

// 	return h.Leave(ctx, cc.ID)
// }
