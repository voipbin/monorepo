package backuphandler

import (
	"testing"
)

func TestNewBackupHandler(t *testing.T) {
	h := NewBackupHandler("testuser:testpass@tcp(127.0.0.1:3306)/voipbin", "/var/backups/voipbin", 7)

	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}
