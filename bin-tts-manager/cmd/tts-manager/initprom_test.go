package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// initProm must leave the default mux untouched when the listen address is
// missing; registering twice would panic, so only the disabled path is tested.
func Test_initProm_withoutListenAddressRegistersNothing(t *testing.T) {
	const endpoint = "/metrics-test-disabled"

	initProm(endpoint, "")

	_, pattern := http.DefaultServeMux.Handler(httptest.NewRequest(http.MethodGet, endpoint, nil))
	if pattern != "" {
		t.Errorf("no handler may be registered for %s, got pattern %q", endpoint, pattern)
	}
}
