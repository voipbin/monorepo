package pipecatcallhandler

import (
	"net/http"
	"testing"
	"time"
)

func Test_httpClientConfiguration(t *testing.T) {
	// Verify HTTP client is configured for connection pooling

	if httpClient == nil {
		t.Fatal("httpClient should not be nil")
	}

	// Verify exact timeout value
	expectedTimeout := 30 * time.Second
	if httpClient.Timeout != expectedTimeout {
		t.Errorf("httpClient.Timeout = %v, want %v", httpClient.Timeout, expectedTimeout)
	}

	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("httpClient.Transport should be *http.Transport")
	}

	// Verify exact MaxIdleConns value
	if transport.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", transport.MaxIdleConns)
	}

	// Verify exact MaxIdleConnsPerHost value
	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 10", transport.MaxIdleConnsPerHost)
	}

	// Verify exact IdleConnTimeout value
	expectedIdleTimeout := 90 * time.Second
	if transport.IdleConnTimeout != expectedIdleTimeout {
		t.Errorf("IdleConnTimeout = %v, want %v", transport.IdleConnTimeout, expectedIdleTimeout)
	}
}
