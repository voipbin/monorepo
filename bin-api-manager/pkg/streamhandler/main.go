package streamhandler

import (
	"context"
	"monorepo/bin-api-manager/models/stream"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	defaultAudioSocketHeaderSize = 3 // audosocket's default header size. https://docs.asterisk.org/Configuration/Channel-Drivers/AudioSocket/
)

type StreamHandler interface {
}

type streamHandler struct {
	listenAddress string

	listenSockUDP *net.UDPConn
	listenSockTCP *net.TCPListener

	streamLock sync.Mutex
	streamData map[string]*stream.Stream
}

func NewStreamHandler(listenAddress string) StreamHandler {
	return &streamHandler{
		listenAddress: listenAddress,
	}
}

func (h *streamHandler) runListenUDP() error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "runListenUDP",
		"endpoint": h.listenAddress,
	})

	addr, err := net.ResolveUDPAddr("udp", h.listenAddress)
	if err != nil {
		log.Errorf("Could not resovle the address. err: %v", err)
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Errorf("Could not listen the address. err: %v", err)
		return err
	}

	h.listenSockUDP = conn
	return nil
}

func (h *streamHandler) runListenTCP(ctx context.Context) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "runListenTCP",
		"endpoint": h.listenAddress,
	})

	addr, err := net.ResolveTCPAddr("tcp", h.listenAddress)
	if err != nil {
		log.Errorf("Could not resovle the address. err: %v", err)
		return err
	}

	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Errorf("Could not listen the address. err: %v", err)
		return err
	}
	defer listen.Close()

	h.listenSockTCP = listen

	for ctx.Err() == nil {
		conn, err := listen.Accept()
		if err != nil {
			log.Errorf("Could not accept the connection. err: %v", err)
			continue
		}

		go h.ProcessStreamsock(conn)
	}

	return nil
}
