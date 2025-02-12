package streaminghandler

import (
	"context"
	"net"
	"time"

	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/pion/rtp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/streaming"
)

// gcpStart starts the stt process using the gcp
func (h *streamingHandler) gcpStart(st *streaming.Streaming, conn *net.UDPConn) error {
	log := logrus.WithFields(logrus.Fields{
		"func":              "gcpStart",
		"streaming_id":      st.ID,
		"transcribe_id":     st.TranscribeID,
		"external_media_id": st.ExternalMediaID,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streamClient, err := h.gcpInit(ctx, st)
	if err != nil {
		log.Errorf("Could not create streaming client: %v", err)
		return err
	}

	go h.gcpProcessResult(ctx, cancel, st, streamClient)
	go h.gcpProcessRTP(ctx, cancel, st, conn, streamClient)

	<-ctx.Done()
	log.Debugf("Finished the gcp process. transcribe_id: %s, streaming_id: %s", st.TranscribeID, st.ID)

	return nil
}

// gcpInit inits the gcp process
func (h *streamingHandler) gcpInit(ctx context.Context, st *streaming.Streaming) (speechpb.Speech_StreamingRecognizeClient, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "gcpInit",
		"streaming_id":      st.ID,
		"transcribe_id":     st.TranscribeID,
		"external_media_id": st.ExternalMediaID,
	})

	// create stt client
	res, err := h.clientSpeech.StreamingRecognize(ctx)
	if err != nil {
		log.Errorf("Could not create a client for speech. err: %v", err)
		return nil, err
	}

	streamingConfig := speechpb.StreamingRecognitionConfig{
		Config: &speechpb.RecognitionConfig{
			Encoding:                   defaultEncoding,
			SampleRateHertz:            int32(defaultSampleRate),
			AudioChannelCount:          int32(defaultAudioChannelCount),
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
		"func":              "gcpProcessResult",
		"streaming_id":      st.ID,
		"transcribe_id":     st.TranscribeID,
		"external_media_id": st.ExternalMediaID,
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
			log.Errorf("Could not received the result. err: %v", err)
			return
		} else if len(tmp.Results) == 0 {
			// result end
			return
		}

		if !tmp.Results[0].IsFinal {
			time.Sleep(time.Millisecond * 400)
			continue
		}

		// get transcript message and create transcript
		message := tmp.Results[0].Alternatives[0].Transcript
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

// gcpProcessRTP receives the RTP from the given the asterisk(conn) then send it to the google stt
func (h *streamingHandler) gcpProcessRTP(ctx context.Context, cancel context.CancelFunc, st *streaming.Streaming, conn *net.UDPConn, streamClient speechpb.Speech_StreamingRecognizeClient) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "gcpProcessRTP",
		"streaming_id":      st.ID,
		"transcribe_id":     st.TranscribeID,
		"external_media_id": st.ExternalMediaID,
	})
	log.Debugf("Starting gcpProcessRTP. transcribe_id: %s", st.TranscribeID)
	defer func() {
		log.Debugf("Finished gcpProcessRTP. transcribe_id: %s", st.TranscribeID)
		cancel()
	}()

	// we are define the some variables which is used in the below go routine to boost up the process spped.
	data := make([]byte, 2000)
	for {
		if ctx.Err() != nil {
			log.Debugf("Context has finsished. transcribe_id: %s, streaming_id: %s", st.TranscribeID, st.ID)
			return
		}

		n, remote, err := conn.ReadFromUDP(data)
		if err != nil {
			log.Infof("Connection has closed. err: %v", err)
			return
		}

		// Unmarshal the packet and update the PayloadType
		rtpPacket := &rtp.Packet{}
		if errUnmarshal := rtpPacket.Unmarshal(data[:n]); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the received data. len: %d, remote: %s, err: %v", n, remote, errUnmarshal)
			break
		}

		if rtpPacket.PayloadType > 63 && rtpPacket.PayloadType < 96 {
			// this is a rtcp packet.
			// we don't send it
			continue
		}

		if errSend := streamClient.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
				AudioContent: rtpPacket.Payload,
			},
		}); errSend != nil {
			log.Debugf("Could not send audio data correctly: %v", errSend)
		}
	}
}
