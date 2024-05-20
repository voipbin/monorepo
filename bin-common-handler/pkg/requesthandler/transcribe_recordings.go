package requesthandler

// // TranscribeV1RecordingCreate sends a request to transcribe-manager
// // to transcode the exist recording.
// // it returns transcoded text if it succeed.
// func (r *requestHandler) TranscribeV1RecordingCreate(ctx context.Context, customerID, recordingID uuid.UUID, language string) (*tstranscribe.Transcribe, error) {
// 	uri := "/v1/recordings"

// 	req := &tsrequest.V1DataRecordingsPost{
// 		CustomerID:  customerID,
// 		ReferenceID: recordingID,
// 		Language:    language,
// 	}

// 	m, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	res, err := r.sendRequestTranscribe(ctx, uri, rabbitmqhandler.RequestMethodPost, "storage/recording", 60000, 0, ContentTypeJSON, m)
// 	switch {
// 	case err != nil:
// 		return nil, err
// 	case res == nil:
// 		// not found
// 		return nil, fmt.Errorf("response code: %d", 404)
// 	case res.StatusCode > 299:
// 		return nil, fmt.Errorf("response code: %d", res.StatusCode)
// 	}

// 	var data tstranscribe.Transcribe
// 	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
// 		return nil, err
// 	}

// 	return &data, nil
// }
