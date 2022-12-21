package transcribehandler

// // CallRecording transcribe the call's recordings
// func (h *transcribeHandler) CallRecording(ctx context.Context, customerID, callID uuid.UUID, language string) ([]*transcribe.Transcribe, error) {
// 	log := logrus.New().WithFields(
// 		logrus.Fields{
// 			"func":        "CallRecording",
// 			"customer_id": customerID,
// 			"call_id":     callID,
// 		},
// 	)

// 	lang := getBCP47LanguageCode(language)
// 	log.Debugf("Parsed BCP47 language code. lang: %s", lang)

// 	// get call info
// 	c, err := h.reqHandler.CallV1CallGet(ctx, callID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	res := []*transcribe.Transcribe{}
// 	for _, recordingID := range c.RecordingIDs {

// 		// create transcribe
// 		tr, err := h.Create(ctx, customerID, recordingID, transcribe.TypeRecording, lang, common.DirectionBoth)
// 		if err != nil {
// 			log.Errorf("Could not create the transcribe. err: %v", err)
// 			continue
// 		}
// 		log.WithField("transcribe", tr).Debugf("Created the transcribe. transcribe_id: %s", tr.ID)

// 		// do transcribe recording
// 		tmp, err := h.transcriptHandler.Recording(ctx, customerID, tr.ID, recordingID, lang)
// 		if err != nil {
// 			log.Errorf("Could not transcribe the recording. err: %v", err)
// 			continue
// 		}
// 		log.WithField("transcript", tmp).Debugf("Transcripted. transcribe_id: %s, transcript_id: %s", tr.ID, tmp.ID)

// 		res = append(res, tr)
// 	}

// 	return res, nil
// }
