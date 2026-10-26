package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
)

// This file holds a minimal in-process AMQP 0-9-1 broker bound to a loopback
// listener. It answers only what the amqp091 client used by the message
// broker handler needs to declare queues and start consumers: the connection
// handshake, channel open and close, exchange declare, queue declare and bind,
// basic qos and basic consume. Nothing is ever delivered.

const (
	frameMethod = 1
	frameBeat   = 8
	frameEnd    = 0xCE

	// queue.declare and exchange.declare flag bits
	queueFlagDurable    = 0x02
	queueFlagAutoDelete = 0x08
	queueFlagNoWait     = 0x10

	exchangeFlagDurable = 0x02
	exchangeFlagNoWait  = 0x10
)

type declaredQueue struct {
	name  string
	flags byte
	args  map[string]any
}

type declaredExchange struct {
	name  string
	kind  string
	flags byte
	args  map[string]any
}

type boundQueue struct {
	queue    string
	exchange string
	key      string
}

type startedConsumer struct {
	queue    string
	tag      string
	prefetch uint16
}

type fakeBroker struct {
	ln net.Listener

	mu        sync.Mutex
	dead      bool
	conns     []net.Conn
	queues    []declaredQueue
	exchanges []declaredExchange
	binds     []boundQueue
	consumers []startedConsumer
	problems  []string
}

// startFakeBroker listens on an ephemeral loopback port. On cleanup the broker
// refuses and drops every connection but keeps the listener open until the
// process exits: the client under test never closes its connection handler, so
// its reconnect loop outlives the test and must not reach a later broker that
// would be handed the same port.
func startFakeBroker(t *testing.T) *fakeBroker {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Could not listen: %v", err)
	}

	b := &fakeBroker{ln: ln}
	go b.acceptLoop()

	t.Cleanup(func() {
		b.mu.Lock()
		b.dead = true
		conns := b.conns
		b.conns = nil
		b.mu.Unlock()

		for _, c := range conns {
			_ = c.Close()
		}
	})

	return b
}

// uri returns the broker address with throwaway credentials.
func (b *fakeBroker) uri() string {
	return "amqp://test:test@" + b.ln.Addr().String()
}

func (b *fakeBroker) acceptLoop() {
	for {
		c, err := b.ln.Accept()
		if err != nil {
			return
		}

		b.mu.Lock()
		if b.dead {
			b.mu.Unlock()
			_ = c.Close()
			continue
		}
		b.conns = append(b.conns, c)
		b.mu.Unlock()

		go b.serve(c)
	}
}

func (b *fakeBroker) problem(format string, args ...any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.problems = append(b.problems, fmt.Sprintf(format, args...))
}

// snapshot copies the recorded state under the lock.
func (b *fakeBroker) snapshot() (queues []declaredQueue, exchanges []declaredExchange, binds []boundQueue, consumers []startedConsumer, problems []string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return append([]declaredQueue(nil), b.queues...),
		append([]declaredExchange(nil), b.exchanges...),
		append([]boundQueue(nil), b.binds...),
		append([]startedConsumer(nil), b.consumers...),
		append([]string(nil), b.problems...)
}

func writeFrame(w io.Writer, typ byte, ch uint16, payload []byte) error {
	var buf bytes.Buffer
	buf.WriteByte(typ)
	_ = binary.Write(&buf, binary.BigEndian, ch)
	_ = binary.Write(&buf, binary.BigEndian, uint32(len(payload)))
	buf.Write(payload)
	buf.WriteByte(frameEnd)
	_, err := w.Write(buf.Bytes())
	return err
}

func readFrame(r io.Reader) (typ byte, ch uint16, payload []byte, err error) {
	var hdr [7]byte
	if _, err = io.ReadFull(r, hdr[:]); err != nil {
		return
	}
	typ = hdr[0]
	ch = binary.BigEndian.Uint16(hdr[1:3])
	size := binary.BigEndian.Uint32(hdr[3:7])
	payload = make([]byte, size+1)
	if _, err = io.ReadFull(r, payload); err != nil {
		return
	}
	if payload[size] != frameEnd {
		err = fmt.Errorf("bad frame end 0x%x", payload[size])
		return
	}
	payload = payload[:size]
	return
}

func methodFrame(class, meth uint16, args ...[]byte) []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, class)
	_ = binary.Write(&buf, binary.BigEndian, meth)
	for _, a := range args {
		buf.Write(a)
	}
	return buf.Bytes()
}

func wireU16(v uint16) []byte { return binary.BigEndian.AppendUint16(nil, v) }

func wireU32(v uint32) []byte { return binary.BigEndian.AppendUint32(nil, v) }

func wireShortStr(s string) []byte { return append([]byte{byte(len(s))}, s...) }

func wireLongStr(s string) []byte { return append(wireU32(uint32(len(s))), s...) }

// wireReader decodes method arguments. A short read panics, which serve turns
// into a recorded problem.
type wireReader struct {
	b []byte
	p int
}

func (r *wireReader) u8() byte {
	v := r.b[r.p]
	r.p++
	return v
}

func (r *wireReader) u16() uint16 {
	v := binary.BigEndian.Uint16(r.b[r.p:])
	r.p += 2
	return v
}

func (r *wireReader) u32() uint32 {
	v := binary.BigEndian.Uint32(r.b[r.p:])
	r.p += 4
	return v
}

func (r *wireReader) shortStr() string {
	n := int(r.u8())
	v := string(r.b[r.p : r.p+n])
	r.p += n
	return v
}

// table decodes a field table holding only the value types the client sends
// for the arguments under test: S (string), I (int32) and t (bool).
func (r *wireReader) table() (map[string]any, error) {
	size := int(r.u32())
	end := r.p + size
	out := map[string]any{}

	for r.p < end {
		name := r.shortStr()
		switch typ := r.u8(); typ {
		case 'S':
			n := int(r.u32())
			out[name] = string(r.b[r.p : r.p+n])
			r.p += n
		case 'I':
			out[name] = int32(r.u32())
		case 't':
			out[name] = r.u8() != 0
		default:
			return nil, fmt.Errorf("unsupported field table type %q for %q", typ, name)
		}
	}

	return out, nil
}

func (b *fakeBroker) serve(c net.Conn) {
	defer func() {
		if rec := recover(); rec != nil {
			b.problem("malformed frame: %v", rec)
		}
		_ = c.Close()
	}()

	var proto [8]byte
	if _, err := io.ReadFull(c, proto[:]); err != nil {
		return
	}

	// Connection.Start: version 0.9, no server properties, PLAIN, en_US
	start := methodFrame(10, 10, []byte{0, 9}, wireU32(0), wireLongStr("PLAIN"), wireLongStr("en_US"))
	if writeFrame(c, frameMethod, 0, start) != nil {
		return
	}

	// prefetch is per channel and ids are reused after a channel closes.
	prefetch := map[uint16]uint16{}

	for {
		typ, ch, payload, err := readFrame(c)
		if err != nil {
			return
		}
		if typ == frameBeat {
			continue
		}
		if typ != frameMethod {
			b.problem("unexpected frame type %d", typ)
			return
		}

		r := &wireReader{b: payload}
		class, meth := r.u16(), r.u16()

		var reply []byte
		switch {
		case class == 10 && meth == 11: // StartOk -> Tune with heartbeat 0
			reply = methodFrame(10, 30, wireU16(2047), wireU32(131072), wireU16(0))
		case class == 10 && meth == 31: // TuneOk
		case class == 10 && meth == 40: // Open -> OpenOk
			reply = methodFrame(10, 41, wireShortStr(""))
		case class == 10 && meth == 50: // Close -> CloseOk
			_ = writeFrame(c, frameMethod, 0, methodFrame(10, 51))
			return
		case class == 10 && meth == 51: // CloseOk
			return
		case class == 20 && meth == 10: // Channel.Open
			delete(prefetch, ch)
			reply = methodFrame(20, 11, wireLongStr(""))
		case class == 20 && meth == 40: // Channel.Close
			delete(prefetch, ch)
			reply = methodFrame(20, 41)
		case class == 20 && meth == 41: // Channel.CloseOk
		case class == 40 && meth == 10: // Exchange.Declare
			r.u16()
			e := declaredExchange{name: r.shortStr(), kind: r.shortStr(), flags: r.u8()}
			if e.args, err = r.table(); err != nil {
				b.problem("exchange.declare: %v", err)
				return
			}
			b.mu.Lock()
			b.exchanges = append(b.exchanges, e)
			b.mu.Unlock()
			if e.flags&exchangeFlagNoWait == 0 {
				reply = methodFrame(40, 11)
			}
		case class == 50 && meth == 10: // Queue.Declare
			r.u16()
			q := declaredQueue{name: r.shortStr(), flags: r.u8()}
			if q.args, err = r.table(); err != nil {
				b.problem("queue.declare: %v", err)
				return
			}
			b.mu.Lock()
			b.queues = append(b.queues, q)
			b.mu.Unlock()
			if q.flags&queueFlagNoWait == 0 {
				reply = methodFrame(50, 11, wireShortStr(q.name), wireU32(0), wireU32(0))
			}
		case class == 50 && meth == 20: // Queue.Bind
			r.u16()
			bq := boundQueue{queue: r.shortStr(), exchange: r.shortStr(), key: r.shortStr()}
			noWait := r.u8()&0x01 != 0
			b.mu.Lock()
			b.binds = append(b.binds, bq)
			b.mu.Unlock()
			if !noWait {
				reply = methodFrame(50, 21)
			}
		case class == 60 && meth == 10: // Basic.Qos
			r.u32() // prefetch size
			prefetch[ch] = r.u16()
			reply = methodFrame(60, 11)
		case class == 60 && meth == 20: // Basic.Consume
			r.u16()
			sc := startedConsumer{queue: r.shortStr(), tag: r.shortStr(), prefetch: prefetch[ch]}
			flags := r.u8()
			b.mu.Lock()
			b.consumers = append(b.consumers, sc)
			b.mu.Unlock()
			if flags&0x08 == 0 { // no-wait
				reply = methodFrame(60, 21, wireShortStr(sc.tag))
			}
		default:
			b.problem("unexpected method %d.%d", class, meth)
			return
		}

		if reply != nil && writeFrame(c, frameMethod, ch, reply) != nil {
			return
		}
	}
}
