package requesthandler

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	initPrometheus("test")

	os.Exit(m.Run())
}
