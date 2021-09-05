package transcribehandler

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/rtphandler"
)

const (
	defaultEncoding          = speechpb.RecognitionConfig_MULAW
	defaultSampleRate        = 8000
	defaultAudioChannelCount = 1
)

const (
	externalMediaOptEncapsulation  = "rtp"
	externalMediaOptTransport      = "udp"
	externalMediaOptConnectionType = "client"
	externalMediaOptFormat         = "ulaw"
	externalMediaOptDirection      = "both"
)

func (h *transcribeHandler) StreamingTranscribeStart(
	ctx context.Context,
	referenceID uuid.UUID,
	transType transcribe.Type,
	language string,
	webhookURI string,
	webhookMethod string,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "transcribeRawStream",
		},
	)
	id := uuid.Must(uuid.NewV4())

	// currently, support calltype only
	if transType != transcribe.TypeCall {
		return nil, fmt.Errorf("no support transcribe type. type: %s", transType)
	}

	// start rtp listen
	conn, err := h.serveListen()
	if err != nil {
		log.Errorf("Could not listen for the rtp. err: %v", err)
		return nil, err
	}

	// create client
	stream, err := h.clientSpeech.StreamingRecognize(ctx)
	if err != nil {
		log.Errorf("Could not create a client for speech. err: %v", err)
		return nil, err
	}

	// send external media request to the call-manager
	// todo:
	hostAddr := conn.LocalAddr().String()
	ip, port, err := h.reqHandler.CMCallExternalMedia(
		referenceID,
		hostAddr,
		externalMediaOptEncapsulation,
		externalMediaOptTransport,
		externalMediaOptConnectionType,
		externalMediaOptFormat,
		externalMediaOptDirection,
		"",
	)
	if err != nil {
		log.Errorf("Could not create external media. err: %v", err)
		return nil, err
	}
	log.Debugf("Created external media. ip: %s, port: %d", ip, port)

	// start rtp handle and transcribe
	go h.streamTranscribeHandle(ctx, conn, stream, language)

	res := &transcribe.Transcribe{
		ID:            id,
		Type:          transType,
		ReferenceID:   referenceID,
		HostID:        h.hostID,
		Language:      language,
		WebhookURI:    webhookURI,
		WebhookMethod: webhookMethod,
		Transcription: "",
	}

	// add transcribe to cache
	if errSet := h.cache.TranscribeSet(ctx, res); errSet != nil {
		log.Errorf("Could not set transcribe. err: %v", errSet)
		return nil, errSet
	}

	// add current streaming
	addServiceStreaming(id, conn, stream)

	return res, nil

}

func (h *transcribeHandler) StreamingTranscribeStop(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "TranscribeStreamStop",
		},
	)

	s := getServiceStreaming(id)
	if s == nil {
		return fmt.Errorf("no streaming found")
	}

	// stop rtp listen
	if errCloseConn := s.conn.Close(); errCloseConn != nil {
		log.Errorf("Could not close the rtp listen socket correctly. err: %v", errCloseConn)
	}

	// stop stream
	if errCloseStream := s.stream.CloseSend(); errCloseStream != nil {
		log.Errorf("Could not close the stream correctly. err: %v", errCloseStream)
	}

	// delete service streaming
	deleteServiceStreaming(id)

	return nil
}

func (h *transcribeHandler) streamTranscribeHandle(ctx context.Context, conn *net.UDPConn, stream speechpb.Speech_StreamingRecognizeClient, language string) (string, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "transcribeStreamHandle",
		},
	)
	defer conn.Close()

	// start rtp receive channel
	chanRTP := make(chan []byte)
	rtpHandler := rtphandler.NewRTPHandler(conn, chanRTP)
	go rtpHandler.Serve()

	// init stream
	if errInit := h.streamingTranscribeInit(ctx, stream, language); errInit != nil {
		log.Errorf("Could not initate the transcribeStream. err: %v", errInit)
		return "", errInit
	}

	// start result handle
	go h.streamingTranscribeResultHandle(ctx, stream)

	// start rtp forward
	for {
		select {
		case data := <-chanRTP:

			if err := stream.Send(&speechpb.StreamingRecognizeRequest{
				StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
					AudioContent: data,
				},
			}); err != nil {
				log.Printf("Could not send audio data correctly: %v", err)
			}
		}
	}
}

func (h *transcribeHandler) streamingTranscribeInit(ctx context.Context, stream speechpb.Speech_StreamingRecognizeClient, language string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "transcribeStreamInit",
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

func (h *transcribeHandler) streamingTranscribeResultHandle(ctx context.Context, stream speechpb.Speech_StreamingRecognizeClient) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "transcribeStreamResultHandle",
		},
	)

	resTranscript := ""
	for {
		res, err := stream.Recv()
		if err != nil {
			log.Errorf("Could not received the result. err: %v", err)
			return err
		}
		log.Debugf("Recevied result. res: %v", res.Results)
		if len(res.Results) == 0 {
			return nil
		}

		if res.Results[0].IsFinal == false && len(res.Results) == 1 {
			continue
		}

		tmpTranscript := res.Results[0].Alternatives[0].Transcript
		newTranscript := ""
		if len(tmpTranscript) > len(resTranscript) {
			newTranscript = tmpTranscript[len(resTranscript):]
		}

		resTranscript = tmpTranscript
		if newTranscript != "" {
			log.Debugf("Received new transcript. transcript: %s", newTranscript)
		}

		time.Sleep(time.Millisecond * 400)
	}
}
