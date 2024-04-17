package streaminghandler

import (
	"context"
	"fmt"
	"net"

	"monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// Start starts the live streaming transcribe of the given transcribe
func (h *streamingHandler) Start(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction) (*streaming.Streaming, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":           "Start",
			"transcribe_id":  transcribeID,
			"reference_type": referenceType,
			"reference_id":   referenceID,
			"direction":      direction,
		},
	)

	// start rtp listen
	conn, err := h.serveListen()
	if err != nil {
		log.Errorf("Could not listen for the rtp. err: %v", err)
		return nil, err
	}
	log.Debugf("Listening RTP streaming. local_addr: %s", conn.LocalAddr())

	// start the external media
	// send request to the call-manager
	hostAddr := conn.LocalAddr().String()
	em, err := h.reqHandler.CallV1ExternalMediaStart(
		ctx,
		externalmedia.ReferenceType(referenceType),
		referenceID,
		true,
		hostAddr,
		constEncapsulation,
		constTransport,
		constConnectionType,
		constFormat,
		string(direction),
	)
	if err != nil {
		log.Errorf("Could not create external media. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", em).Debugf("Started external media. external_media_id: %s, host_addr: %s, media_ip: %s, media_port: %d", em.ID, hostAddr, em.LocalIP, em.LocalPort)

	// create streaming database record
	res, err := h.Create(ctx, customerID, transcribeID, em.ID, language, direction)
	if err != nil {
		log.Errorf("Could not create streaming. err: %v", err)
		return nil, err
	}
	log.WithField("streaming", res).Debugf("Created a new streaming. streaming_id: %s", res.ID)

	// run the stt process.
	// currently, we have only one stt service provider handler
	go func() {
		if errStart := h.gcpStart(ctx, res, conn); errStart != nil {
			log.Errorf("Could not finish the gcp stt correctly. err: %v", errStart)
		}
	}()

	return res, nil
}

// serveListen starts the UDP listen.
func (h *streamingHandler) serveListen() (*net.UDPConn, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "serveListen",
		},
	)

	// find available port for 10 times
	for i := 0; i < 10; i++ {
		// get listen port
		port := getRandomPort()
		conn, err := net.ListenUDP("udp", &net.UDPAddr{
			IP:   net.ParseIP(defaultListenIP),
			Port: port,
		})
		if err != nil {
			log.Errorf("Could not listen the address. ip: %s, port: %d, err: %v", defaultListenIP, port, err)
			continue
		}

		return conn, nil
	}

	return nil, fmt.Errorf("no available port")
}
