package streaminghandler

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming/types"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/streaming"
)

// awsEventStream abstracts the AWS Transcribe event stream for testing.
type awsEventStream interface {
	Events() <-chan types.TranscriptResultStream
	Close() error
}

func awsNewClient(accessKey string, secretKey string) (*transcribestreaming.Client, error) {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(defaultAWSRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS SDK v2 config: %v", err)
	}

	res := transcribestreaming.NewFromConfig(cfg)
	return res, nil
}

// awsRun runs the STT process using AWS Transcribe Streaming
func (h *streamingHandler) awsRun(st *streaming.Streaming, conn net.Conn) error {
	if h.awsClient == nil {
		return fmt.Errorf("AWS provider not initialized")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "awsRun",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streamClient, err := h.awsInit(ctx, st)
	if err != nil {
		log.Errorf("Could not create streaming client: %v", err)
		return err
	}

	go h.awsProcessResult(ctx, cancel, st, streamClient)
	go h.awsProcessMedia(ctx, cancel, st, conn, streamClient)

	<-ctx.Done()
	log.Debugf("Finished the AWS process. transcribe_id: %s, streaming_id: %s", st.TranscribeID, st.ID)

	return nil
}

// awsInit initializes the AWS Transcribe Streaming client
func (h *streamingHandler) awsInit(ctx context.Context, st *streaming.Streaming) (*transcribestreaming.StartStreamTranscriptionOutput, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "awsInit",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})

	input := &transcribestreaming.StartStreamTranscriptionInput{
		LanguageCode:         types.LanguageCode(st.Language),
		MediaEncoding:        types.MediaEncoding(defaultAWSEncoding),
		MediaSampleRateHertz: aws.Int32(defaultAWSSampleRate),
	}

	res, err := h.awsClient.StartStreamTranscription(ctx, input)
	if err != nil {
		log.Errorf("Could not create a client for speech. err: %v", err)
		return nil, err
	}

	return res, nil
}

// awsProcessResult handles transcript results from AWS Transcribe
func (h *streamingHandler) awsProcessResult(ctx context.Context, cancel context.CancelFunc, st *streaming.Streaming, streamClient *transcribestreaming.StartStreamTranscriptionOutput) {
	stream := streamClient.GetStream()
	h.awsProcessEvents(ctx, cancel, st, stream)
}

// awsProcessEvents processes events from an AWS event stream.
func (h *streamingHandler) awsProcessEvents(ctx context.Context, cancel context.CancelFunc, st *streaming.Streaming, stream awsEventStream) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "awsProcessEvents",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})
	log.Debugf("Starting awsProcessEvents. transcribe_id: %s", st.TranscribeID)

	defer func() {
		log.Debugf("Finished awsProcessEvents. transcribe_id: %s", st.TranscribeID)
		_ = stream.Close()
		cancel()
	}()

	speaking := false
	t1 := time.Now()
	for {
		select {
		case <-ctx.Done():
			log.Debug("Context canceled, stopping awsProcessEvents.")
			return

		case event, ok := <-stream.Events():
			if !ok {
				log.Debug("TranscriptResultStream closed.")
				return
			}

			transcriptEvent, ok := event.(*types.TranscriptResultStreamMemberTranscriptEvent)
			if !ok {
				continue
			}

			for _, result := range transcriptEvent.Value.Transcript.Results {
				if len(result.Alternatives) == 0 {
					continue
				}

				if result.IsPartial {
					// partial result — publish VAD events
					message := ""
					if result.Alternatives[0].Transcript != nil {
						message = *result.Alternatives[0].Transcript
					}

					if !speaking {
						speaking = true
						now := time.Now()
						webhookMsg := st.ConvertWebhookMessage("", &now)
						h.notifyHandler.PublishWebhookEvent(ctx, st.CustomerID, streaming.EventTypeSpeechStarted, webhookMsg)
						log.Debugf("Published speech_started. transcribe_id: %s, direction: %s", st.TranscribeID, st.Direction)
					}

					now := time.Now()
					webhookMsg := st.ConvertWebhookMessage(message, &now)
					h.notifyHandler.PublishWebhookEvent(ctx, st.CustomerID, streaming.EventTypeSpeechInterim, webhookMsg)
					log.Debugf("Published speech_interim. transcribe_id: %s, direction: %s, message: %s", st.TranscribeID, st.Direction, message)
					continue
				}

				// final result — publish speech_ended if was speaking
				if speaking {
					speaking = false
					now := time.Now()
					webhookMsg := st.ConvertWebhookMessage("", &now)
					h.notifyHandler.PublishWebhookEvent(ctx, st.CustomerID, streaming.EventTypeSpeechEnded, webhookMsg)
					log.Debugf("Published speech_ended. transcribe_id: %s, direction: %s", st.TranscribeID, st.Direction)
				}

				if result.Alternatives[0].Transcript == nil {
					continue
				}
				message := *result.Alternatives[0].Transcript
				if len(message) == 0 {
					continue
				}
				log.Debugf("Received transcript message. transcribe_id: %s, direction: %s, message: %s", st.TranscribeID, st.Direction, message)

				t2 := time.Now()
				t3 := t2.Sub(t1)
				tmGap := time.Time{}.Add(t3)

				// create transcript
				ts, err := h.transcriptHandler.Create(ctx, st.CustomerID, st.TranscribeID, st.Direction, message, &tmGap)
				if err != nil {
					log.Errorf("Could not create transript. err: %v", err)
					break
				}
				log.WithField("transcript", ts).Debugf("Created transcript. transcribe_id: %s, direction: %s", ts.TranscribeID, ts.Direction)
			}
		}
	}
}

// awsProcessMedia receives media from Asterisk and sends it to AWS Transcribe
func (h *streamingHandler) awsProcessMedia(ctx context.Context, cancel context.CancelFunc, st *streaming.Streaming, conn net.Conn, streamClient *transcribestreaming.StartStreamTranscriptionOutput) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "awsProcessMedia",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})
	log.Debugf("Starting awsProcessMedia. transcribe_id: %s", st.TranscribeID)
	defer func() {
		log.Debugf("Finished awsProcessMedia. transcribe_id: %s", st.TranscribeID)
		cancel()
	}()

	stream := streamClient.GetStream()
	defer func() {
		_ = stream.Close()
	}()

	for {
		if ctx.Err() != nil {
			log.Debugf("Context has finished. transcribe_id: %s, streaming_id: %s", st.TranscribeID, st.ID)
			return
		}

		m, err := h.audiosocketGetNextMedia(conn)
		if err != nil {
			log.Infof("Connection has closed. err: %v", err)
			return
		}

		if m == nil {
			continue
		}

		if errSend := stream.Send(ctx, &types.AudioStreamMemberAudioEvent{
			Value: types.AudioEvent{
				AudioChunk: m.Payload(),
			},
		}); errSend != nil {
			if errSend != io.EOF {
				log.Errorf("Could not send audio data correctly. err: %v", errSend)
			}
			return

		}
	}
}
