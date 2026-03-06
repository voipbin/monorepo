package ngclient

//go:generate mockgen -source=main.go -destination=mock_main.go -package=ngclient

import "time"

// NGClient sends commands to RTPEngine via the NG protocol (bencode/UDP).
type NGClient interface {
	Send(cmd map[string]interface{}) (map[string]interface{}, error)
	Close()
}

// New creates a connected NG client.
func New(addr string, timeout time.Duration) (NGClient, error) {
	return newNGClient(addr, timeout)
}
