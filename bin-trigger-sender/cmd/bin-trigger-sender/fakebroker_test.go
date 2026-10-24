package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// This file holds a minimal in-process AMQP 0-9-1 broker bound to a loopback
// listener. It implements only what the amqp091-go client used by run() needs:
// handshake, one channel, queue declare, consume, publish and deliver.

const (
	frameMethod    = 1
	frameHeader    = 2
	frameBody      = 3
	frameHeartbeat = 8
	frameEnd       = 0xCE

	// content header property flags
	propContentType   = 0x8000
	propCorrelationID = 0x0400
	propReplyTo       = 0x0200

	fakeGeneratedQueue = "amq.gen-fake"
	fakeReplyCorrID    = "bin-trigger-sender-cronjob"
)

type brokerConfig struct {
	rejectOpen    bool // answer Channel.Open with Connection.Close
	rejectDeclare bool // answer Queue.Declare with Channel.Close 406
	rejectConsume bool // answer Basic.Consume with Channel.Close 406
	noReply       bool // accept the publish but never answer
	dropOnClose   bool // drop the TCP connection when the client sends Channel.Close

	status  int    // status_code in the response body (default 200)
	corrID  string // correlation id of the reply (default fakeReplyCorrID)
	rawBody []byte // if set, used as the reply body verbatim
}

type declareRecord struct {
	queue string
	bits  byte
}

type consumeRecord struct {
	queue string
	tag   string
	bits  byte
}

type publishRecord struct {
	exchange    string
	routingKey  string
	flags       uint16
	contentType string
	corrID      string
	replyTo     string
	body        []byte
}

type fakeBroker struct {
	t    *testing.T
	cfg  brokerConfig
	ln   net.Listener
	wg   sync.WaitGroup
	conn sync.Map

	mu       sync.Mutex
	declares []declareRecord
	consumes []consumeRecord
	pubs     []publishRecord
}

func startFakeBroker(t *testing.T, cfg brokerConfig) *fakeBroker {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	b := &fakeBroker{t: t, cfg: cfg, ln: ln}
	b.wg.Add(1)
	go b.acceptLoop()
	t.Cleanup(func() {
		_ = ln.Close()
		b.conn.Range(func(k, _ any) bool {
			_ = k.(net.Conn).Close()
			return true
		})
		b.wg.Wait()
	})
	return b
}

func (b *fakeBroker) addr() string {
	return "amqp://" + b.ln.Addr().String()
}

func (b *fakeBroker) acceptLoop() {
	defer b.wg.Done()
	for {
		c, err := b.ln.Accept()
		if err != nil {
			return
		}
		b.conn.Store(c, struct{}{})
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			defer func() { _ = c.Close() }()
			b.serve(c)
		}()
	}
}

func (b *fakeBroker) published() []publishRecord {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]publishRecord(nil), b.pubs...)
}

// waitPublished waits until at least n publishes were recorded.
func (b *fakeBroker) waitPublished(n int) []publishRecord {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if p := b.published(); len(p) >= n {
			return p
		}
		time.Sleep(5 * time.Millisecond)
	}
	b.t.Fatalf("broker did not receive %d publish(es)", n)
	return nil
}

// ---- wire helpers ----

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

func method(class, meth uint16, args ...[]byte) []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, class)
	_ = binary.Write(&buf, binary.BigEndian, meth)
	for _, a := range args {
		buf.Write(a)
	}
	return buf.Bytes()
}

func u8(v byte) []byte { return []byte{v} }

func u16(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}

func u32(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

func u64(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func shortStr(s string) []byte { return append([]byte{byte(len(s))}, s...) }

func longStr(s string) []byte { return append(u32(uint32(len(s))), s...) }

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

func (r *wireReader) u64() uint64 {
	v := binary.BigEndian.Uint64(r.b[r.p:])
	r.p += 8
	return v
}

func (r *wireReader) shortStr() string {
	n := int(r.u8())
	v := string(r.b[r.p : r.p+n])
	r.p += n
	return v
}

// ---- protocol loop ----

func (b *fakeBroker) serve(c net.Conn) {
	var proto [8]byte
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := io.ReadFull(c, proto[:]); err != nil {
		return
	}
	// Connection.Start: version 0.9, empty server properties, PLAIN, en_US
	start := method(10, 10, u8(0), u8(9), u32(0), longStr("PLAIN"), longStr("en_US"))
	if writeFrame(c, frameMethod, 0, start) != nil {
		return
	}

	for {
		_ = c.SetDeadline(time.Now().Add(10 * time.Second))
		typ, ch, payload, err := readFrame(c)
		if err != nil {
			return
		}
		if typ == frameHeartbeat {
			continue
		}
		if typ != frameMethod {
			b.t.Errorf("fake broker: unexpected frame type %d", typ)
			return
		}
		r := &wireReader{b: payload}
		class, meth := r.u16(), r.u16()

		var out [][]byte // method frames to send on channel ch
		switch {
		case class == 10 && meth == 11: // StartOk -> Tune
			out = append(out, method(10, 30, u16(2047), u32(131072), u16(0)))
		case class == 10 && meth == 31: // TuneOk
		case class == 10 && meth == 40: // Open -> OpenOk
			out = append(out, method(10, 41, shortStr("")))
		case class == 10 && meth == 50: // Connection.Close -> CloseOk
			_ = writeFrame(c, frameMethod, 0, method(10, 51))
			return
		case class == 10 && meth == 51: // CloseOk to our own Close
			return
		case class == 20 && meth == 10: // Channel.Open
			if b.cfg.rejectOpen {
				_ = writeFrame(c, frameMethod, 0, method(10, 50, u16(320), shortStr("rejected"), u16(20), u16(10)))
				continue
			}
			out = append(out, method(20, 11, longStr("")))
		case class == 20 && meth == 40: // Channel.Close
			if b.cfg.dropOnClose {
				return
			}
			out = append(out, method(20, 41))
		case class == 20 && meth == 41: // Channel.CloseOk
		case class == 50 && meth == 10: // Queue.Declare
			r.u16()
			queue := r.shortStr()
			bits := r.u8()
			b.mu.Lock()
			b.declares = append(b.declares, declareRecord{queue: queue, bits: bits})
			b.mu.Unlock()
			if b.cfg.rejectDeclare {
				out = append(out, method(20, 40, u16(406), shortStr("precondition failed"), u16(50), u16(10)))
			} else {
				out = append(out, method(50, 11, shortStr(fakeGeneratedQueue), u32(0), u32(0)))
			}
		case class == 60 && meth == 20: // Basic.Consume
			r.u16()
			queue := r.shortStr()
			tag := r.shortStr()
			bits := r.u8()
			b.mu.Lock()
			b.consumes = append(b.consumes, consumeRecord{queue: queue, tag: tag, bits: bits})
			b.mu.Unlock()
			if b.cfg.rejectConsume {
				out = append(out, method(20, 40, u16(406), shortStr("access refused"), u16(60), u16(20)))
			} else {
				out = append(out, method(60, 21, shortStr(tag)))
			}
		case class == 60 && meth == 40: // Basic.Publish
			r.u16()
			rec := publishRecord{exchange: r.shortStr(), routingKey: r.shortStr()}
			if !b.readContent(c, &rec) {
				return
			}
			b.mu.Lock()
			b.pubs = append(b.pubs, rec)
			tag := ""
			if len(b.consumes) > 0 {
				tag = b.consumes[len(b.consumes)-1].tag
			}
			b.mu.Unlock()
			if tag == "" {
				b.t.Errorf("fake broker: publish received before consume")
				return
			}
			if b.cfg.noReply {
				continue
			}
			if !b.reply(c, ch, tag, rec.replyTo) {
				return
			}
		default:
			b.t.Errorf("fake broker: unexpected method %d.%d", class, meth)
			return
		}

		for _, m := range out {
			if writeFrame(c, frameMethod, ch, m) != nil {
				return
			}
		}
	}
}

// readContent reads the content header and body frames that follow Basic.Publish.
func (b *fakeBroker) readContent(c net.Conn, rec *publishRecord) bool {
	typ, _, payload, err := readFrame(c)
	if err != nil || typ != frameHeader {
		b.t.Errorf("fake broker: expected content header, got type %d err %v", typ, err)
		return false
	}
	r := &wireReader{b: payload}
	r.u16() // class
	r.u16() // weight
	size := r.u64()
	rec.flags = r.u16()
	if rec.flags&propContentType != 0 {
		rec.contentType = r.shortStr()
	}
	if rec.flags&propCorrelationID != 0 {
		rec.corrID = r.shortStr()
	}
	if rec.flags&propReplyTo != 0 {
		rec.replyTo = r.shortStr()
	}
	for uint64(len(rec.body)) < size {
		typ, _, payload, err = readFrame(c)
		if err != nil || typ != frameBody {
			b.t.Errorf("fake broker: expected body frame, got type %d err %v", typ, err)
			return false
		}
		rec.body = append(rec.body, payload...)
	}
	return true
}

// reply sends Basic.Deliver with a content header and one body frame.
func (b *fakeBroker) reply(c net.Conn, ch uint16, tag, routingKey string) bool {
	body := b.cfg.rawBody
	if body == nil {
		status := b.cfg.status
		if status == 0 {
			status = 200
		}
		body = []byte(fmt.Sprintf(`{"status_code":%d,"data_type":"application/json","data":{"ok":true}}`, status))
	}
	corrID := b.cfg.corrID
	if corrID == "" {
		corrID = fakeReplyCorrID
	}

	deliver := method(60, 60, shortStr(tag), u64(1), u8(0), shortStr(""), shortStr(routingKey))
	header := append(u16(60), u16(0)...)
	header = append(header, u64(uint64(len(body)))...)
	header = append(header, u16(propContentType|propCorrelationID)...)
	header = append(header, shortStr("application/json")...)
	header = append(header, shortStr(corrID)...)

	return writeFrame(c, frameMethod, ch, deliver) == nil &&
		writeFrame(c, frameHeader, ch, header) == nil &&
		writeFrame(c, frameBody, ch, body) == nil
}
