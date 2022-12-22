package transcripthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// Start starts the streaming transcribe with the given direction.
func (h *transcriptHandler) Start(ctx context.Context, tr *transcribe.Transcribe, direction transcript.Direction) (*streaming.Streaming, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Start",
			"transcribe_id": tr.ID,
			"reference_id":  tr.ReferenceID,
			"direction":     direction,
		},
	)

	// currently, support calltype only
	if tr.ReferenceType != transcribe.ReferenceTypeCall {
		return nil, fmt.Errorf("no support transcribe type. type: %s", tr.ReferenceType)
	}

	// start rtp listen
	conn, err := h.serveListen()
	if err != nil {
		log.Errorf("Could not listen for the rtp. err: %v", err)
		return nil, err
	}
	log.Debugf("Listening RTP streaming. local_addr: %s", conn.LocalAddr())

	// create streamClient client
	streamClient, err := h.clientSpeech.StreamingRecognize(ctx)
	if err != nil {
		log.Errorf("Could not create a client for speech. err: %v", err)
		return nil, err
	}

	// create the external media
	// send request to the call-manager
	hostAddr := conn.LocalAddr().String()
	tmp, err := h.reqHandler.CallV1CallAddExternalMedia(
		ctx,
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
		return nil, err
	}
	log.Debugf("Created external media. host_addr: %s, media_ip: %s, media_port: %d", hostAddr, tmp.MediaAddrIP, tmp.MediaAddrPort)

	// create streaming
	res := &streaming.Streaming{
		ID:           uuid.Must(uuid.NewV4()),
		TranscribeID: tr.ID,
		CustomerID:   tr.CustomerID,
		Language:     tr.Language,
		Direction:    direction,
		Conn:         conn,
		Stream:       streamClient,
	}
	h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingStarted, res)

	// start transcribe
	go h.processStart(ctx, res)

	return res, nil
}

// Stop stops streaming transcribe.
func (h *transcriptHandler) Stop(ctx context.Context, st *streaming.Streaming) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Stop",
			"transcribe_id": st.TranscribeID,
			"streaming_id":  st.ID,
		},
	)

	// stop rtp listen
	if errCloseConn := st.Conn.Close(); errCloseConn != nil {
		log.Errorf("Could not close the rtp listen socket correctly. err: %v", errCloseConn)
	}

	// stop stream
	if errCloseStream := st.Stream.CloseSend(); errCloseStream != nil {
		log.Errorf("Could not close the stream correctly. err: %v", errCloseStream)
	}

	return nil
}
