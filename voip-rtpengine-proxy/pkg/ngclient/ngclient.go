package ngclient

import (
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
	cmd["cookie"] = cookie

	encoded, err := bencode.EncodeString(cmd)
	if err != nil {
		return nil, fmt.Errorf("bencode encode: %w", err)
	}

	ch := make(chan map[string]interface{}, 1)
	c.mu.Lock()
	c.pending[cookie] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, cookie)
		c.mu.Unlock()
	}()

	if _, err := c.conn.Write([]byte(encoded)); err != nil {
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
	c.conn.Close()
}

func (c *ngClient) readLoop() {
	buf := make([]byte, 65535)
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			logrus.WithError(err).Debug("NG client read loop terminated")
			return
		}
		var resp map[string]interface{}
		if err := bencode.DecodeString(string(buf[:n]), &resp); err != nil {
			logrus.WithError(err).Warn("Failed to decode NG response")
			continue
		}
		cookie, ok := resp["cookie"].(string)
		if !ok {
			logrus.Warn("NG response missing cookie")
			continue
		}
		delete(resp, "cookie")

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
	rand.Read(b)
	return hex.EncodeToString(b)
}
