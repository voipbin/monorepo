package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"sync"
)

// fakeBroker is a loopback server that speaks the small part of AMQP 0-9-1 the broker client
// needs to connect, declare exchanges and queues, bind and consume. It records queue
// declarations, bindings and consumers, and checks no credentials.
type fakeBroker struct {
	// address is an amqp URI without credentials.
	address string

	mu        sync.Mutex
	queues    []string
	binds     []declaredBind
	consumers []string
}

// declaredBind is a queue binding received by the fake broker.
type declaredBind struct {
	queue    string
	exchange string
	key      string
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

// snapshot returns a copy of the queues, bindings and consumers received so far.
func (b *fakeBroker) snapshot() (queues []string, binds []declaredBind, consumers []string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return append([]string(nil), b.queues...), append([]declaredBind(nil), b.binds...), append([]string(nil), b.consumers...)
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
		case class == 50 && method == 10: // queue declare -> declare-ok (queue name, 0 messages, 0 consumers)
			name := shortStringAt(payload[4:], 0)
			b.mu.Lock()
			b.queues = append(b.queues, name)
			b.mu.Unlock()

			args := append([]byte{byte(len(name))}, name...)
			args = binary.BigEndian.AppendUint32(args, 0)
			args = binary.BigEndian.AppendUint32(args, 0)
			reply = amqpMethod(channel, 50, 11, args)
		case class == 50 && method == 20: // queue bind -> bind-ok
			b.mu.Lock()
			b.binds = append(b.binds, declaredBind{
				queue:    shortStringAt(payload[4:], 0),
				exchange: shortStringAt(payload[4:], 1),
				key:      shortStringAt(payload[4:], 2),
			})
			b.mu.Unlock()
			reply = amqpMethod(channel, 50, 21, nil)
		case class == 60 && method == 10: // basic qos -> qos-ok
			reply = amqpMethod(channel, 60, 11, nil)
		case class == 60 && method == 20: // basic consume -> consume-ok with the requested tag
			b.mu.Lock()
			b.consumers = append(b.consumers, shortStringAt(payload[4:], 0))
			b.mu.Unlock()

			tag := shortStringAt(payload[4:], 1)
			reply = amqpMethod(channel, 60, 21, append([]byte{byte(len(tag))}, tag...))
		case class == 40 && method == 10: // exchange declare -> declare-ok
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

// shortStringAt reads the index-th short string of the arguments of a queue or basic method,
// which start with two reserved bytes.
func shortStringAt(args []byte, index int) string {
	if len(args) < 2 {
		return ""
	}
	args = args[2:]
	for i := 0; ; i++ {
		if len(args) < 1 || len(args) < 1+int(args[0]) {
			return ""
		}
		if i == index {
			return string(args[1 : 1+int(args[0])])
		}
		args = args[1+int(args[0]):]
	}
}
