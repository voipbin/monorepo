package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"sync"
)

// declaredExchange is an exchange declaration received by the fake broker.
type declaredExchange struct {
	name string
	kind string
}

// fakeBroker is a loopback server that speaks the small part of AMQP 0-9-1 the broker client
// needs to connect and to declare exchanges. It stores nothing and checks no credentials.
type fakeBroker struct {
	// address is an amqp URI without credentials.
	address string

	mu        sync.Mutex
	exchanges []declaredExchange
}

// amqpFrame builds a frame of the given type for a channel.
func amqpFrame(frameType byte, channel uint16, payload []byte) []byte {
	b := []byte{frameType, byte(channel >> 8), byte(channel)}
	b = binary.BigEndian.AppendUint32(b, uint32(len(payload)))
	b = append(b, payload...)
	return append(b, 0xCE)
}

// amqpMethod builds a method frame.
func amqpMethod(channel uint16, class, method uint16, args []byte) []byte {
	p := binary.BigEndian.AppendUint16(nil, class)
	p = binary.BigEndian.AppendUint16(p, method)
	return amqpFrame(1, channel, append(p, args...))
}

// newFakeBroker starts the broker on a free loopback port. It is never closed: the broker
// client keeps reconnect goroutines for the life of the process, so closing the listener would
// only turn them into retry noise.
func newFakeBroker() (*fakeBroker, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	res := &fakeBroker{address: "amqp://" + ln.Addr().String()}
	go func() {
		for {
			conn, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}
			go res.serve(conn)
		}
	}()

	return res, nil
}

// declared returns a copy of the exchanges declared so far.
func (b *fakeBroker) declared() []declaredExchange {
	b.mu.Lock()
	defer b.mu.Unlock()

	return append([]declaredExchange(nil), b.exchanges...)
}

// serve answers one connection until the client closes it.
func (b *fakeBroker) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	if _, err := io.ReadFull(reader, make([]byte, 8)); err != nil { // protocol header
		return
	}

	// connection.start: version 0.9, empty server properties, PLAIN, en_US
	start := []byte{0, 9}
	start = binary.BigEndian.AppendUint32(start, 0)
	start = binary.BigEndian.AppendUint32(start, 5)
	start = append(start, "PLAIN"...)
	start = binary.BigEndian.AppendUint32(start, 5)
	start = append(start, "en_US"...)
	if _, err := conn.Write(amqpMethod(0, 10, 10, start)); err != nil {
		return
	}

	for {
		head := make([]byte, 7)
		if _, err := io.ReadFull(reader, head); err != nil {
			return
		}
		channel := binary.BigEndian.Uint16(head[1:3])
		size := binary.BigEndian.Uint32(head[3:7])

		payload := make([]byte, size+1) // payload and the frame end byte
		if _, err := io.ReadFull(reader, payload); err != nil {
			return
		}
		payload = payload[:size]
		if head[0] != 1 || len(payload) < 4 {
			continue
		}

		class := binary.BigEndian.Uint16(payload[0:2])
		method := binary.BigEndian.Uint16(payload[2:4])
		var reply []byte
		switch {
		case class == 10 && method == 11: // start-ok -> tune with no heartbeat
			tune := binary.BigEndian.AppendUint16(nil, 0)
			tune = binary.BigEndian.AppendUint32(tune, 131072)
			tune = binary.BigEndian.AppendUint16(tune, 0)
			reply = amqpMethod(0, 10, 30, tune)
		case class == 10 && method == 40: // open -> open-ok
			reply = amqpMethod(0, 10, 41, []byte{0})
		case class == 10 && method == 50: // close -> close-ok
			_, _ = conn.Write(amqpMethod(0, 10, 51, nil))
			return
		case class == 20 && method == 10: // channel open -> open-ok
			reply = amqpMethod(channel, 20, 11, binary.BigEndian.AppendUint32(nil, 0))
		case class == 20 && method == 40: // channel close -> close-ok
			reply = amqpMethod(channel, 20, 41, nil)
		case class == 40 && method == 10: // exchange declare -> declare-ok
			if name, kind, ok := parseExchangeDeclare(payload[4:]); ok {
				b.mu.Lock()
				b.exchanges = append(b.exchanges, declaredExchange{name: name, kind: kind})
				b.mu.Unlock()
			}
			reply = amqpMethod(channel, 40, 11, nil)
		}

		if reply == nil {
			continue
		}
		if _, err := conn.Write(reply); err != nil {
			return
		}
	}
}

// parseExchangeDeclare reads the exchange name and kind from the arguments of exchange.declare.
func parseExchangeDeclare(args []byte) (string, string, bool) {
	if len(args) < 3 { // reserved short, name length
		return "", "", false
	}
	args = args[2:]

	nameLen := int(args[0])
	if len(args) < 2+nameLen {
		return "", "", false
	}
	name := string(args[1 : 1+nameLen])
	args = args[1+nameLen:]

	kindLen := int(args[0])
	if len(args) < 1+kindLen {
		return "", "", false
	}

	return name, string(args[1 : 1+kindLen]), true
}
