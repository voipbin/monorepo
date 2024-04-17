package websockhandler

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/pion/rtp"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// mediaStreamRun starts the media stream forwarding
func (h *websockHandler) mediaStreamRun(ctx context.Context, w http.ResponseWriter, r *http.Request, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, encapsulation string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "mediaStreamRun",
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"encapsulation":  encapsulation,
	})

	newCtx, newCancel := context.WithCancel(ctx)
	defer newCancel()

	// create a websock
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Could not create websocket. err: %v", err)
		return err
	}
	defer ws.Close()
	log.Debugf("Created a new websocket correctly.")

	localEndpoint := endpointLocalGet(referenceID)
	log.Debugf("Created local endpoint. local_endpoint: %s", localEndpoint)
	defer endpointLocalRelease(referenceID)

	switch encapsulation {
	case "audiosocket":
		err = h.mediaStreamRunAudioSocket(newCtx, newCancel, ws, referenceType, referenceID, localEndpoint)

	case "rtp":
		err = h.mediaStreamRunRTP(newCtx, newCancel, ws, referenceType, referenceID, localEndpoint)

	case "sln":
		err = h.mediaStreamRunSLN(newCtx, newCancel, ws, referenceType, referenceID, localEndpoint)

	default:
		log.Errorf("Unsupported encapsulation type. type: %s", encapsulation)
		return fmt.Errorf("unsupported encapsulation type")
	}

	if err != nil {
		log.Errorf("Could not run the media stream handler correctly. err: %v", err)
		return err
	}

	return nil
}

// mediaStreamRunAudioSocket starts the media stream forwarding with audiosocket encapsulation
func (h *websockHandler) mediaStreamRunAudioSocket(ctx context.Context, cancel context.CancelFunc, ws *websocket.Conn, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, localEndpoint string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "mediaStreamRunAudioSocket",
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"local_endpoint": localEndpoint,
	})
	log.Debugf("Starting media stream handler encapsulationtype audiosocket.")

	// start the listener
	listener, err := h.mediaStreamListenerTCP(localEndpoint)
	if err != nil {
		log.Errorf("Could not start media stream listener. err: %v", err)
		return err
	}
	defer listener.Close()

	// start external media
	em, err := h.reqHandler.CallV1ExternalMediaStart(
		ctx,
		referenceType,
		referenceID,
		false,
		localEndpoint,
		defaultEncapsulationForAudioSocket,
		defaultTransportForAudioSocket,
		defaultConnectionType,
		defaultFormat,
		defualtDirection,
	)
	if err != nil {
		log.Errorf("Could not start the external media service. err: %v", err)
		return err
	}
	log.WithField("external_media", em).Debugf("Started external media. external_media_id: %s", em.ID)

	if errHandle := h.mediaStreamHandleDataAudioSocket(ctx, cancel, listener, ws); errHandle != nil {
		log.Errorf("Could not start the media stream handler. err: %v", errHandle)
		return errHandle
	}

	<-ctx.Done()
	log.Debugf("Websocket connection has been closed. reference_id: %s", referenceID)

	return nil
}

// mediaStreamListenRTP starts the TCP listener
func (h *websockHandler) mediaStreamListenerTCP(endpoint string) (*net.TCPListener, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "runRTPListen",
		"endpoint": endpoint,
	})

	addr, err := net.ResolveTCPAddr("tcp", endpoint)
	if err != nil {
		log.Errorf("Could not resovle the address. err: %v", err)
		return nil, err
	}

	res, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Errorf("Could not listen the address. err: %v", err)
		return nil, err
	}

	return res, nil
}

// mediaStreamListenRTP starts the RTP listen of call server
func (h *websockHandler) mediaStreamHandleDataAudioSocket(ctx context.Context, cancel context.CancelFunc, listener *net.TCPListener, ws *websocket.Conn) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runRTPListen",
	})

	// accept only 1 connection
	conn, err := listener.Accept()
	if err != nil {
		log.Errorf("Could not accept connection, err: %v", err)
		return err
	}
	defer conn.Close()
	log.Debugf("Connected new client.")

	go func(connection net.Conn) {
		log.Debugf("Start the media stream data handler for asterisk -> listener -> websocket")
		defer cancel()
		for {

			// Read the header
			header := make([]byte, defaultAudioSocketHeaderSize)
			_, err := io.ReadFull(conn, header)
			if err != nil {
				log.Errorf("Could not read the header. err: %v", err)
				return
			}

			// Extract type and payload length from the header
			messageType := header[0]
			payloadLength := binary.BigEndian.Uint16(header[1:])

			// Read the payload
			payload := make([]byte, payloadLength)
			_, err = io.ReadFull(conn, payload)
			if err != nil {
				log.Errorf("Could not read the payload. err: %v", err)
				break
			}

			// make message
			message := append(header, payload...)

			// 0x00 - Terminate the connection (socket closure is also sufficient)
			// 0x01 - Payload will contain the UUID (16-byte binary representation) for the audio stream
			// 0x10 - Payload is signed linear, 16-bit, 8kHz, mono PCM (little-endian)
			// 0xff - An error has occurred; payload is the (optional) application-specific error code. Asterisk-generated error codes are listed below.
			log.Debugf("Recevied message. message_type: %x, total_len: %d, payload_len: %d", header[0], len(message), payloadLength)
			switch messageType {
			case 0x01:
				// this contains none usable info
				id := uuid.FromBytesOrNil(payload)
				log.Debugf("audio socket info. id: %s", id)
				continue

			case 0xff:
				// this contains none usable info
				log.Debugf("Recevied error code. err: %s", string(payload))
				continue
			}

			// bypass the data to the websocket connection
			if errWrite := ws.WriteMessage(websocket.BinaryMessage, message); errWrite != nil {
				log.Debugf("The websocket connect is over. err: %v", errWrite)
				return
			}
		}
	}(conn)

	go func(connection net.Conn) {
		log.Debugf("Start the media stream data handler for websocket -> listener -> asterisk")
		defer cancel()

		for {
			if ctx.Err() != nil {
				log.Debugf("The context is over. Exiting the process.")
				return
			}

			m, err := h.receiveBinaryFromWebsock(ctx, ws)
			if err != nil {
				log.Errorf("Could not receive the RTP from the websock. err: %v", err)
				return
			}
			log.Debugf("Received the data from the websocket connection. len: %d", len(m))

			// bypass the received packet
			_, err = connection.Write(m)
			if err != nil {
				log.Errorf("Could not send the rtp payload. err: %v", err)
				return
			}
		}
	}(conn)

	<-ctx.Done()

	return nil
}

// mediaStreamRunRTP starts the media stream forwarding for RTP encapsulationt type
func (h *websockHandler) mediaStreamRunRTP(ctx context.Context, cancel context.CancelFunc, ws *websocket.Conn, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, localEndpoint string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "mediaStreamRunRTP",
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"local_endpoint": localEndpoint,
	})
	log.Debugf("Starting media stream handler encapsulationtype rtp.")

	// start rtp listen server
	conn, err := h.mediaStreamListenerUDP(localEndpoint)
	if err != nil {
		log.Errorf("Could not start the rtp listen server. err: %v", err)
		return err
	}
	defer conn.Close()

	// run the
	go func() {
		h.mediaStreamHandleRTPFromAsterisk(ctx, conn, ws)
		log.Debugf("The media stream handler has finished.")
		cancel()
	}()

	// start external media
	em, err := h.reqHandler.CallV1ExternalMediaStart(
		ctx,
		referenceType,
		referenceID,
		false,
		localEndpoint,
		defaultEncapsulationForRTP,
		defaultTransportForRTP,
		defaultConnectionType,
		defaultFormat,
		defualtDirection,
	)
	if err != nil {
		log.Errorf("Could not start the external media service. err: %v", err)
		return err
	}
	log.WithField("external_media", em).Debugf("Started external media. external_media_id: %s", em.ID)

	remoteEndpoint := fmt.Sprintf("%s:%d", em.LocalIP, em.LocalPort)
	go func() {
		h.mediaStreamHandleRTPFromWebsock(ctx, ws, remoteEndpoint)
		cancel()
	}()

	go func() {
		h.mediaStreamHandleReferenceWatcher(ctx, referenceType, referenceID)
		cancel()
	}()

	<-ctx.Done()
	log.Debugf("Websocket connection has been closed. reference_id: %s", referenceID)

	return nil
}

// mediaStreamListenerUDP starts the RTP listen of call server
func (h *websockHandler) mediaStreamListenerUDP(endpoint string) (*net.UDPConn, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "runRTPListen",
		"endpoint": endpoint,
	})

	addr, err := net.ResolveUDPAddr("udp", endpoint)
	if err != nil {
		log.Errorf("Could not resovle the address. err: %v", err)
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Errorf("Could not listen the address. err: %v", err)
		return nil, err
	}

	return conn, nil
}

// mediaStreamHandleRTPFromAsterisk receives the RTP from the given the asterisk(conn) and put the received rtp stream to the given websocket.
func (h *websockHandler) mediaStreamHandleRTPFromAsterisk(ctx context.Context, conn *net.UDPConn, ws *websocket.Conn) {
	log := logrus.WithFields(logrus.Fields{
		"func": "mediaStreamHandleRTPFromAsterisk",
	})

	data := make([]byte, 2000)
	for {
		if ctx.Err() != nil {
			log.Debugf("The context is over. Exiting the process.")
			return
		}

		n, _, err := conn.ReadFromUDP(data)
		if err != nil {
			log.Infof("Connection has closed. err: %v", err)
			return
		}

		// write the rtp
		if errWrite := ws.WriteMessage(websocket.BinaryMessage, data[:n]); errWrite != nil {
			log.Debugf("The websocket connect is over. err: %v", errWrite)
			return
		}
	}
}

// mediaStreamHandleRTPFromWebsock receives the RTP from the given the websocket and forward the received rtp stream to the given remote endpoint.
func (h *websockHandler) mediaStreamHandleRTPFromWebsock(ctx context.Context, ws *websocket.Conn, remoteEndpoint string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "mediaStreamHandleRTPFromWebsock",
		"remote_endpoint": remoteEndpoint,
	})

	remoteAddr, err := net.ResolveUDPAddr("udp", remoteEndpoint)
	if err != nil {
		log.Errorf("Could not resolve the remote address. err: %v", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Errorf("Could not initialize the connection. err: %v", err)
		return
	}
	defer conn.Close()

	for {
		if ctx.Err() != nil {
			log.Debugf("The context is over. Exiting the process.")
			return
		}

		m, err := h.receiveBinaryFromWebsock(ctx, ws)
		if err != nil {
			log.Errorf("Could not receive the RTP from the websock. err: %v", err)
			return
		}

		// bypass the received packet
		_, err = conn.Write(m)
		if err != nil {
			log.Errorf("Could not send the rtp payload. err: %v", err)
			return
		}
	}
}

// mediaStreamHandleReferenceWatcher watches the given reference's status and return it if the reference is over
// this is neccessary because the udp server can not recognize the reference has ended or not
func (h *websockHandler) mediaStreamHandleReferenceWatcher(ctx context.Context, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "mediaStreamHandleReferenceWatcher",
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	switch referenceType {
	case cmexternalmedia.ReferenceTypeCall:
		for {
			c, err := h.reqHandler.CallV1CallGet(ctx, referenceID)
			if err != nil {
				log.Errorf("Could not get the call info. err: %v", err)
				return
			}

			if c.Status == cmcall.StatusHangup {
				log.Debugf("The call has ended. call_id: %s", referenceID)
				return
			}

			time.Sleep(defaultReferenceWatcherDelay)
		}

	case cmexternalmedia.ReferenceTypeConfbridge:
		for {
			c, err := h.reqHandler.CallV1ConfbridgeGet(ctx, referenceID)
			if err != nil {
				log.Errorf("Could not get the confbridge info. err: %v", err)
				return
			}

			if c.Status == cmconfbridge.StatusTerminated {
				log.Debugf("The confbridge has ended. confbridge_id: %s", referenceID)
				return
			}

			time.Sleep(defaultReferenceWatcherDelay)
		}

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", referenceType)
	}
}

// mediaStreamRunSLN starts the media stream forwarding for SLN(Signed-Linear Mono) encapsulationt type
func (h *websockHandler) mediaStreamRunSLN(ctx context.Context, cancel context.CancelFunc, ws *websocket.Conn, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, localEndpoint string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "mediaStreamRunSLN",
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"local_endpoint": localEndpoint,
	})
	log.Debugf("Starting media stream handler encapsulationtype sln.")

	// start rtp listen server
	conn, err := h.mediaStreamListenerUDP(localEndpoint)
	if err != nil {
		log.Errorf("Could not start the rtp listen server. err: %v", err)
		return err
	}
	defer conn.Close()

	// run the
	go func() {
		h.mediaStreamHandleSLNFromAsterisk(ctx, conn, ws)
		log.Debugf("The media stream handler has finished.")
		cancel()
	}()

	// start external media
	em, err := h.reqHandler.CallV1ExternalMediaStart(
		ctx,
		referenceType,
		referenceID,
		false,
		localEndpoint,
		defaultEncapsulationForRTP,
		defaultTransportForRTP,
		defaultConnectionType,
		defaultFormat,
		defualtDirection,
	)
	if err != nil {
		log.Errorf("Could not start the external media service. err: %v", err)
		return err
	}
	log.WithField("external_media", em).Debugf("Started external media. external_media_id: %s", em.ID)

	remoteEndpoint := fmt.Sprintf("%s:%d", em.LocalIP, em.LocalPort)
	go func() {
		h.mediaStreamHandleSLNFromWebsock(ctx, ws, remoteEndpoint)
		cancel()
	}()

	go func() {
		h.mediaStreamHandleReferenceWatcher(ctx, referenceType, referenceID)
		cancel()
	}()

	<-ctx.Done()
	log.Debugf("Websocket connection has been closed. call_id: %s", referenceID)

	return nil
}

// mediaStreamHandleSLNFromAsterisk receives the RTP from the given the asterisk(conn) and put the received rtp stream to the given websocket.
func (h *websockHandler) mediaStreamHandleSLNFromAsterisk(ctx context.Context, conn *net.UDPConn, ws *websocket.Conn) {
	log := logrus.WithFields(logrus.Fields{
		"func": "mediaStreamHandleSLNFromAsterisk",
	})

	p := &rtp.Packet{}
	data := make([]byte, 2000)
	for {
		if ctx.Err() != nil {
			log.Debugf("The context is over. Exiting the process.")
			return
		}

		n, _, err := conn.ReadFromUDP(data)
		if err != nil {
			log.Infof("Connection has closed. err: %v", err)
			return
		}

		// Unmarshal the packet and update the PayloadType
		if errUnmarshal := p.Unmarshal(data[:n]); errUnmarshal != nil {
			log.Errorf("Could not unmarshal the received data. len: %d", len(data))
			continue
		}

		// check the payload type
		if p.PayloadType > 63 && p.PayloadType < 96 {
			// rtcp packet.
			// we don't deal with the rtcp
			continue
		}
		log.WithField("rtp", p).Debugf("payload size: %d", len(p.Payload))

		// write the rtp payload
		if errWrite := ws.WriteMessage(websocket.BinaryMessage, p.Payload); errWrite != nil {
			log.Debugf("The websocket connect is over. err: %v", errWrite)
			return
		}
	}
}

// mediaStreamHandleSLNFromWebsock receives the RTP from the given the websocket and forward the received rtp stream to the given remote endpoint.
// currently, only signed linear, 16-bit, 8kHz, mono ulaw allowed
func (h *websockHandler) mediaStreamHandleSLNFromWebsock(ctx context.Context, ws *websocket.Conn, remoteEndpoint string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "mediaStreamHandleSLNFromWebsock",
		"remote_endpoint": remoteEndpoint,
	})

	remoteAddr, err := net.ResolveUDPAddr("udp", remoteEndpoint)
	if err != nil {
		log.Errorf("Could not resolve the remote address. err: %v", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Errorf("Could not initialize the connection. err: %v", err)
		return
	}
	defer conn.Close()

	sequenceNumber := uint16(0)
	timestamp := uint32(0)
	for {
		if ctx.Err() != nil {
			log.Debugf("The context is over. Exiting the process.")
			return
		}

		m, err := h.receiveBinaryFromWebsock(ctx, ws)
		if err != nil {
			log.Errorf("Could not receive the RTP from the websock. err: %v", err)
			return
		}
		sequenceNumber++
		timestamp += uint32(len(m))

		p := rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				Marker:         false,
				PayloadType:    0,
				SequenceNumber: sequenceNumber,
				Timestamp:      timestamp,
				SSRC:           0,
				CSRC:           []uint32{},
			},
			Payload: m,
		}

		// marshal the packet
		data, err := p.Marshal()
		if err != nil {
			log.Errorf("Could not marshal the RTP data. err: %v", err)
			continue
		}

		// send the marshaled rtp packet
		_, err = conn.Write(data)
		if err != nil {
			log.Errorf("Could not send the RTP payload. err: %v", err)
			return
		}
	}
}
