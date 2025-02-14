package streaminghandler

import (
	"context"
	"io"
	"net"
	"time"

	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/streaming"
)

// gcpRun runs the stt process using the gcp
func (h *streamingHandler) gcpRun(st *streaming.Streaming, conn net.Conn) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "gcpRun",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streamClient, err := h.gcpInit(ctx, st)
	if err != nil {
		log.Errorf("Could not create streaming client: %v", err)
		return err
	}

	go h.gcpProcessResult(ctx, cancel, st, streamClient)
	go h.gcpProcessMedia(ctx, cancel, st, conn, streamClient)

	<-ctx.Done()
	log.Debugf("Finished the gcp process. transcribe_id: %s, streaming_id: %s", st.TranscribeID, st.ID)

	return nil
}

// gcpInit inits the gcp process
func (h *streamingHandler) gcpInit(ctx context.Context, st *streaming.Streaming) (speechpb.Speech_StreamingRecognizeClient, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "gcpInit",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})

	// create stt client
	res, err := h.gcpClient.StreamingRecognize(ctx)
	if err != nil {
		log.Errorf("Could not create a client for speech. err: %v", err)
		return nil, err
	}

	streamingConfig := speechpb.StreamingRecognitionConfig{
		Config: &speechpb.RecognitionConfig{
			Encoding:                   defaultGCPEncoding,
			SampleRateHertz:            int32(defaultGCPSampleRate),
			AudioChannelCount:          int32(defaultGCPAudioChannelCount),
			LanguageCode:               st.Language,
			EnableAutomaticPunctuation: true,
		},
		InterimResults: true,
	}

	// init config
	if err := res.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &streamingConfig,
		},
	}); err != nil {
		log.Errorf("Could not send the stream recognition request. err: %v", err)
		return nil, err
	}

	return res, nil
}

// gcpProcessResult handles transcript result from the google stt
func (h *streamingHandler) gcpProcessResult(ctx context.Context, cancel context.CancelFunc, st *streaming.Streaming, streamClient speechpb.Speech_StreamingRecognizeClient) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "gcpProcessResult",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})
	log.Debugf("Starting gcpProcessResult. transcribe_id: %s", st.TranscribeID)
	defer func() {
		log.Debugf("Finished gcpProcessResult. transcribe_id: %s", st.TranscribeID)
		cancel()
	}()

	t1 := time.Now()
	for {
		if ctx.Err() != nil {
			log.Debugf("Context has finsished. transcribe_id: %s, streaming_id: %s", st.TranscribeID, st.ID)
			return
		}

		tmp, err := streamClient.Recv()
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Errorf("Could not received the result. err: %v", err)
			return
		} else if len(tmp.Results) == 0 {
			// result end
			return
		}

		if !tmp.Results[0].IsFinal {
			time.Sleep(time.Millisecond * 100)
			continue
		}
		log.WithField("transceipt_event", tmp).Debugf("Reeived transcript event. transcribe_id: %s, direction: %s", st.TranscribeID, st.Direction)

		// get transcript message and create transcript
		message := tmp.Results[0].Alternatives[0].Transcript
		if len(message) == 0 {
			continue
		}
		log.Debugf("Received transcript message. transcribe_id: %s, direction: %s, message: %s", st.TranscribeID, st.Direction, message)

		t2 := time.Now()
		t3 := t2.Sub(t1)
		tmGap := time.Time{}.Add(t3)

		// create transcript
		ts, err := h.transcriptHandler.Create(ctx, st.CustomerID, st.TranscribeID, st.Direction, message, tmGap.Format("2006-01-02 15:04:05.00000"))
		if err != nil {
			log.Errorf("Could not create transript. err: %v", err)
			break
		}
		log.WithField("transcript", ts).Debugf("Created transcript. transcribe_id: %s, direction: %s", ts.TranscribeID, ts.Direction)
	}
}

// gcpProcessMedia receives the media from the given the asterisk(conn) then send it to the google stt
func (h *streamingHandler) gcpProcessMedia(ctx context.Context, cancel context.CancelFunc, st *streaming.Streaming, conn net.Conn, streamClient speechpb.Speech_StreamingRecognizeClient) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "gcpProcessMedia",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})
	log.Debugf("Starting gcpProcessAudiosocket. transcribe_id: %s", st.TranscribeID)
	defer func() {
		log.Debugf("Finished gcpProcessAudiosocket. transcribe_id: %s", st.TranscribeID)
		cancel()
	}()

	for {
		if ctx.Err() != nil {
			log.Debugf("Context has finsished. transcribe_id: %s, streaming_id: %s", st.TranscribeID, st.ID)
			return
		}

		m, err := h.audiosocketGetNextMedia(conn)
		if err != nil {
			log.Infof("Connection has closed. err: %v", err)
			return
		}

		if errSend := streamClient.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
				AudioContent: m.Payload(),
			},
		}); errSend != nil {
			if errSend != io.EOF {
				log.Errorf("Could not send audio data correctly. err: %v", errSend)
			}
			return
		}
	}
}
