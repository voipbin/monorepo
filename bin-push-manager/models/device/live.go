package device

import "time"

// Expiry is the time after the last refresh when a device is not used any more. The app refreshes its
// registration on every start.
const Expiry = 30 * 24 * time.Hour

// IsLive returns true if the device was refreshed within Expiry before now.
func (h *Device) IsLive(now time.Time) bool {
	if h.TMUpdate == nil {
		return false
	}

	return now.Sub(*h.TMUpdate) <= Expiry
}
