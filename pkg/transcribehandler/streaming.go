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
	customerID uuid.UUID,
	referenceID uuid.UUID,
	transType transcribe.Type,
	language string,
	webhookURI string,
	webhookMethod string,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "transcribeRawStream",
			"reference_id": referenceID,
			"type":         transType,
		},
	)

	id := uuid.Must(uuid.NewV4())
	log.Debugf("Created transcribe id. id: %v", id)

	tr := &transcribe.Transcribe{
		ID:            id,
		CustomerID:    customerID,
		Type:          transType,
		ReferenceID:   referenceID,
		HostID:        h.hostID,
		Language:      language,
		WebhookURI:    webhookURI,
		WebhookMethod: webhookMethod,
		Transcripts:   []transcribe.Transcript{},
	}
	if err := h.TranscribeCreate(ctx, tr); err != nil {
		log.Errorf("Could not create transcribe. err: %v", err)
		return nil, err
	}
	log.Debugf("Created transcribe. transcribe: %v", tr)

	h.streamingTranscribeStartDirection(ctx, tr, transcribe.TranscriptDirectionIn)
	h.streamingTranscribeStartDirection(ctx, tr, transcribe.TranscriptDirectionOut)

	return tr, nil
}

func (h *transcribeHandler) StreamingTranscribeStop(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "TranscribeStreamStop",
		},
	)

	for _, i := range []transcribe.TranscriptDirection{transcribe.TranscriptDirectionBoth, transcribe.TranscriptDirectionIn, transcribe.TranscriptDirectionOut} {

		k := fmt.Sprintf("%s:%s", id, i)

		s := getServiceStreaming(k)
		if s == nil {
			return fmt.Errorf("no streaming found")
		}
		log.WithFields(
			logrus.Fields{
				"streaming": s,
			},
		).Debugf("Stopping streaming.")

		// stop rtp listen
		if errCloseConn := s.conn.Close(); errCloseConn != nil {
			log.Errorf("Could not close the rtp listen socket correctly. err: %v", errCloseConn)
		}

		// stop stream
		if errCloseStream := s.stream.CloseSend(); errCloseStream != nil {
			log.Errorf("Could not close the stream correctly. err: %v", errCloseStream)
		}

		// delete service streaming
		deleteServiceStreaming(k)
	}

	return nil
}

func (h *transcribeHandler) streamingTranscribeStartDirection(
	ctx context.Context,
	tr *transcribe.Transcribe,
	direction transcribe.TranscriptDirection,
) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "streamingTranscribeStartDirection",
		},
	)

	// currently, support calltype only
	if tr.Type != transcribe.TypeCall {
		return fmt.Errorf("no support transcribe type. type: %s", tr.Type)
	}

	// start rtp listen
	conn, err := h.serveListen()
	if err != nil {
		log.Errorf("Could not listen for the rtp. err: %v", err)
		return err
	}

	// create client
	stream, err := h.clientSpeech.StreamingRecognize(ctx)
	if err != nil {
		log.Errorf("Could not create a client for speech. err: %v", err)
		return err
	}

	// send external media request to the call-manager
	hostAddr := conn.LocalAddr().String()
	ip, port, err := h.reqHandler.CMCallExternalMedia(
		tr.ReferenceID,
		hostAddr,
		externalMediaOptEncapsulation,
		externalMediaOptTransport,
		externalMediaOptConnectionType,
		externalMediaOptFormat,
		string(direction),
	)
	if err != nil {
		log.Errorf("Could not create external media. err: %v", err)
		return err
	}
	log.Debugf("Created external media. ip: %s, port: %d", ip, port)

	// start rtp handle and transcribe
	go h.streamTranscribeHandle(ctx, tr, conn, stream, direction)

	// add current streaming
	k := fmt.Sprintf("%s:%s", tr.ID.String(), direction)
	addServiceStreaming(k, tr.ID, direction, conn, stream)

	return nil
}

func (h *transcribeHandler) streamTranscribeHandle(
	ctx context.Context,
	tr *transcribe.Transcribe,
	conn *net.UDPConn,
	stream speechpb.Speech_StreamingRecognizeClient,
	direction transcribe.TranscriptDirection,
) (string, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "transcribeStreamHandle",
			"transcribe_id": tr.ID,
		},
	)
	defer conn.Close()

	// start rtp receive channel
	chanRTP := make(chan []byte)
	rtpHandler := rtphandler.NewRTPHandler(conn, chanRTP)
	go rtpHandler.Serve()

	// init stream
	if errInit := h.streamingTranscribeInit(ctx, stream, tr.Language); errInit != nil {
		log.Errorf("Could not initate the transcribeStream. err: %v", errInit)
		return "", errInit
	}

	// start result handle
	go h.streamingTranscribeResultHandle(ctx, tr, stream, direction)

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

// streamingTranscribeInit initiates streaming transcribe.
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

// streamingTranscribeResultHandle handles transcript result from the google stt
func (h *transcribeHandler) streamingTranscribeResultHandle(
	ctx context.Context,
	tr *transcribe.Transcribe,
	stream speechpb.Speech_StreamingRecognizeClient,
	direction transcribe.TranscriptDirection,
) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "transcribeStreamResultHandle",
		},
	)

	t1 := time.Now()
	for {
		res, err := stream.Recv()
		if err != nil {
			log.Errorf("Could not received the result. err: %v", err)
			return err
		} else if len(res.Results) == 0 {
			// end
			return nil
		}

		if res.Results[0].IsFinal != true {
			time.Sleep(time.Millisecond * 400)
			continue
		}

		// get transcript message and create transcript
		message := res.Results[0].Alternatives[0].Transcript
		log.Debugf("Received transcript message. direction: %s, message: %s", direction, message)

		t2 := time.Now()
		t3 := t2.Sub(t1)
		tmGap := time.Time{}.Add(t3)
		tmp := &transcribe.Transcript{
			Direction: direction,
			Message:   message,
			TMCreate:  tmGap.Format("15:04:05.000"),
		}

		// send transcript
		log.Debugf("Transcript. tr: %v", tr)
		h.db.TranscribeAddTranscript(ctx, tr.ID, tmp)

		h.sendWebhookTranscript(tr, tmp)
	}
}
