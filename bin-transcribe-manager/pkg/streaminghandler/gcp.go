package streaminghandler

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/streaming"
)

// gcpRun runs the stt process using the gcp
func (h *streamingHandler) gcpRun(st *streaming.Streaming) error {
	if h.gcpClient == nil {
		return fmt.Errorf("GCP provider not initialized")
	}

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
	go h.gcpProcessMedia(ctx, cancel, st, streamClient)

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
			UseEnhanced:                true,
			// Model:                      "phone_call", // note: we can not use the phone_call model because it supports only limited languages
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

	rp := newResultProcessor(st, h.notifyHandler, h.transcriptHandler)
	for {
		if ctx.Err() != nil {
			log.Debugf("Context has finished. transcribe_id: %s, streaming_id: %s", st.TranscribeID, st.ID)
			return
		}

		resp, err := streamClient.Recv()
		if err != nil {
			log.Debugf("Could not received the result. Consider this hangup. err: %v", err)
			return
		}

		for _, result := range resp.Results {
			message := ""
			if len(result.Alternatives) > 0 {
				message = result.Alternatives[0].Transcript
			}
			rp.process(ctx, sttResult{isFinal: result.IsFinal, message: message})
		}
	}
}

// gcpProcessMedia receives the media from Asterisk via WebSocket then sends it to the google stt
func (h *streamingHandler) gcpProcessMedia(ctx context.Context, cancel context.CancelFunc, st *streaming.Streaming, streamClient speechpb.Speech_StreamingRecognizeClient) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "gcpProcessMedia",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})
	log.Debugf("Starting gcpProcessMedia. transcribe_id: %s", st.TranscribeID)
	defer func() {
		log.Debugf("Finished gcpProcessMedia. transcribe_id: %s", st.TranscribeID)
		cancel()
	}()

	for {
		if ctx.Err() != nil {
			log.Debugf("Context has finsished. transcribe_id: %s, streaming_id: %s", st.TranscribeID, st.ID)
			return
		}

		msgType, data, err := st.ConnAst.ReadMessage()
		if err != nil {
			log.Infof("WebSocket connection has closed. err: %v", err)
			return
		}

		if msgType != websocket.BinaryMessage {
			continue
		}

		if len(data) == 0 {
			continue
		}

		if errSend := streamClient.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
				AudioContent: data,
			},
		}); errSend != nil {
			if errSend != io.EOF {
				log.Errorf("Could not send audio data correctly. err: %v", errSend)
			}
			return
		}
	}
}
