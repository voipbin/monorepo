package ngclient

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zeebo/bencode"
)

type ngClient struct {
	conn    *net.UDPConn
	timeout time.Duration
	mu      sync.Mutex
	pending map[string]chan map[string]interface{}
}

func newNGClient(addr string, timeout time.Duration) (*ngClient, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("resolve NG address: %w", err)
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("dial RTPEngine NG: %w", err)
	}
	c := &ngClient{
		conn:    conn,
		timeout: timeout,
		pending: map[string]chan map[string]interface{}{},
	}
	go c.readLoop()
	return c, nil
}

func (c *ngClient) Send(cmd map[string]interface{}) (map[string]interface{}, error) {
	cookie := newCookie()

	// Encode without cookie — do not mutate caller's map.
	// NG protocol wire format: "<cookie> <bencode>"
	encoded, err := bencode.EncodeString(cmd)
	if err != nil {
		return nil, fmt.Errorf("bencode encode: %w", err)
	}
	payload := cookie + " " + encoded

	ch := make(chan map[string]interface{}, 1)
	c.mu.Lock()
	c.pending[cookie] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, cookie)
		c.mu.Unlock()
	}()

	if _, err := c.conn.Write([]byte(payload)); err != nil {
		return nil, fmt.Errorf("send NG command: %w", err)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(c.timeout):
		return nil, fmt.Errorf("RTPEngine NG timeout after %s", c.timeout)
	}
}

func (c *ngClient) Close() {
	if err := c.conn.Close(); err != nil {
		logrus.WithError(err).Warn("could not close NG client connection")
	}
}

func (c *ngClient) readLoop() {
	buf := make([]byte, 65535)
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			logrus.WithError(err).Debug("NG client read loop terminated")
			return
		}

		// Parse NG protocol: "<cookie> <bencode>"
		data := buf[:n]
		idx := bytes.IndexByte(data, ' ')
		if idx < 0 {
			logrus.Warn("NG response missing space separator")
			continue
		}
		cookie := string(data[:idx])

		var resp map[string]interface{}
		if err := bencode.DecodeString(string(data[idx+1:]), &resp); err != nil {
			logrus.WithError(err).Warn("Failed to decode NG response")
			continue
		}

		c.mu.Lock()
		ch, found := c.pending[cookie]
		c.mu.Unlock()

		if found {
			ch <- resp
		}
	}
}

func newCookie() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read should never fail on supported platforms
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
	return hex.EncodeToString(b)
}
