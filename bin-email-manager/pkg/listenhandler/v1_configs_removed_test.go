package listenhandler

import (
	"net/http"
	"testing"

	"monorepo/bin-common-handler/models/sock"
)

// the stored platform settings are gone, so the config routes are not served
func Test_processRequest_configsNotServed(t *testing.T) {
	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
	}{
		{"list", sock.RequestMethodGet, "/v1/configs"},
		{"put secret", sock.RequestMethodPut, "/v1/configs/example.key/secret"},
		{"delete", sock.RequestMethodDelete, "/v1/configs/example.key"},
		{"histories", sock.RequestMethodGet, "/v1/configs/example.key/histories"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &listenHandler{}

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusNotFound {
				t.Errorf("Wrong match. expect: %d, got: %v", http.StatusNotFound, res)
			}
		})
	}
}
