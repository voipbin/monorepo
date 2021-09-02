package rtphandler

import (
	"github.com/pion/rtp"
	"github.com/sirupsen/logrus"
)

func (h *rtpHandler) Serve() error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Serve",
		},
	)
	log.Debug("Starting RTP serve")

	// receive rtp
	if err := h.recvRTP(); err != nil {
		log.Errorf("Could not receive the RTP. err: %v", err)
		return err
	}

	return nil
}

// recvRTP receives the RTP from the UDP connect and forward it to the target channel.
func (h *rtpHandler) recvRTP() error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "recvRTP",
		},
	)

	b := make([]byte, 2000)
	rtpPacket := &rtp.Packet{}
	for {
		n, remote, err := h.conn.ReadFromUDP(b)
		if err != nil {
			log.Infof("Connection has closed. err: %v", err)
			break
		}

		// Unmarshal the packet and update the PayloadType
		if errUnmarshal := rtpPacket.Unmarshal(b[:n]); err != nil {
			log.Errorf("Could not unmarshal the received data. len: %d, remote: %s, err: %v", n, remote, errUnmarshal)
			break
		}

		// check the payload type
		if rtpPacket.PayloadType > 63 && rtpPacket.PayloadType < 96 {
			// rtcp packet.
			continue
		}

		// send it to the channel
		h.chanRTP <- rtpPacket.Payload
	}

	return nil
}
