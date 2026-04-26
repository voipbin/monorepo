package pipecatcall

import "time"

// PingResult is returned by the per-pod GET /v1/ping route.
// HostID is the responding pod's POD_IP, used by the caller to verify the
// queue is consumed by the expected pod (best-effort; does not detect Calico
// POD_IP recycle).
type PingResult struct {
	HostID    string    `json:"host_id"`
	Timestamp time.Time `json:"timestamp"`
}
