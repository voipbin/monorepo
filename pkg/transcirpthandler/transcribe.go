package transcirpthandler

import (
	"context"
	"time"

	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/rtphandler"
)

// transcribeStart starts the transcribe the RTP stream.
func (h *transcriptHandler) transcribeStart(ctx context.Context, st *streaming.Streaming) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "transcribeStart",
			"transcribe_id": st.TranscribeID,
			"streaming_id":  st.ID,
		},
	)
	log.Debugf("Starting streaming transcribe.")

	defer func() {
		log.Debugf("Stopping transcribe.")
		st.Conn.Close()
		h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingStopped, st)
	}()

	// start rtp receive channel
	chanRTP := make(chan []byte)
	rtpHandler := rtphandler.NewRTPHandler(st.Conn, chanRTP)
	go rtpHandler.Serve()

	// init transcribe
	if errInit := h.transcribeInit(ctx, st.Stream, st.Language); errInit != nil {
		log.Errorf("Could not initate the transcribeStream. err: %v", errInit)
		return
	}

	// start result handle
	go h.transcribeResultHandler(ctx, st)

	// start rtp forward
	for {
		data, ok := <-chanRTP
		if !ok {
			log.Debugf("Streaming has finished.")
			return
		}

		if err := st.Stream.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
				AudioContent: data,
			},
		}); err != nil {
			log.Printf("Could not send audio data correctly: %v", err)
		}

	}
}

// transcribeInit initiates streaming transcribe.
func (h *transcriptHandler) transcribeInit(ctx context.Context, stream speechpb.Speech_StreamingRecognizeClient, language string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "transcribeInit",
			"language": language,
		},
	)

	streamingConfig := speechpb.StreamingRecognitionConfig{
		Config: &speechpb.RecognitionConfig{
			Encoding:                   defaultEncoding,
			SampleRateHertz:            int32(defaultSampleRate),
			AudioChannelCount:          int32(defaultAudioChannelCount),
			LanguageCode:               language,
			EnableAutomaticPunctuation: true,
		},
		InterimResults: true,
	}

	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &streamingConfig,
		},
	}); err != nil {
		log.Errorf("Could not send the stream recognition request. err: %v", err)
		return err
	}

	return nil
}

// transcribeResultHandler handles transcript result from the google stt
func (h *transcriptHandler) transcribeResultHandler(
	ctx context.Context,
	st *streaming.Streaming,
) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "transcribeResultHandler",
			"transcribe_id": st.TranscribeID,
			"streaming_id":  st.ID,
		},
	)
	log.Debugf("Starting transcribeResultHandler.")

	t1 := time.Now()
	for {
		res, err := st.Stream.Recv()
		if err != nil {
			log.Errorf("Could not received the result. err: %v", err)
			return
		} else if len(res.Results) == 0 {
			// end
			log.Debug("Result end.")
			return
		}

		if !res.Results[0].IsFinal {
			time.Sleep(time.Millisecond * 400)
			continue
		}

		// get transcript message and create transcript
		message := res.Results[0].Alternatives[0].Transcript
		log.Debugf("Received transcript message. transcribe_id: %s, direction: %s, message: %s", st.TranscribeID, st.Direction, message)

		t2 := time.Now()
		t3 := t2.Sub(t1)
		tmGap := time.Time{}.Add(t3)

		// create
		tmp, err := h.Create(ctx, st.CustomerID, st.TranscribeID, st.Direction, message, tmGap.Format("2006-01-02 15:04:05.00000"))
		if err != nil {
			log.Errorf("Could not create transript. err: %v", err)
			continue
		}
		log.WithField("transcript", tmp).Debugf("Created transcript. transcribe_id: %s, direction: %s", tmp.TranscribeID, tmp.Direction)
	}
}

// transcribeFromBucket transcribes from the bucket file
func (h *transcriptHandler) transcribeFromBucket(ctx context.Context, mediaLink string, language string) (*transcript.Transcript, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "transcribeFromBucket",
			"media_link": mediaLink,
		},
	)

	// Send the contents of the audio file with the encoding and
	// and sample rate information to be transcripted.
	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 8000,
			LanguageCode:    language,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{
				Uri: mediaLink,
			},
		},
	}

	op, err := h.clientSpeech.LongRunningRecognize(ctx, req)
	if err != nil {
		log.Errorf("Could not init google stt. err: %v", err)
		return nil, err
	}

	// wait for result
	resp, err := op.Wait(ctx)
	if err != nil {
		log.Errorf("Could not get google stt result. err: %v", err)
		return nil, err
	}

	// Print the results.
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			log.Debugf("\"%v\" (confidence=%3f)\n", alt.Transcript, alt.Confidence)
		}
	}

	// create transcript
	message := resp.Results[0].Alternatives[0].Transcript
	ts := "0000-00-00 00:00:00.00000"
	res := &transcript.Transcript{
		Direction:    transcript.DirectionBoth,
		Message:      message,
		TMTranscript: ts,
	}

	return res, nil
}
