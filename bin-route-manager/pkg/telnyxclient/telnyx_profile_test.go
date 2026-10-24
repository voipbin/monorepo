package telnyxclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func Test_parseRejectedCountries(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		expect []string
	}{
		{"no marker", "something else went wrong", nil},
		{"codes after the marker", "You must have your account approved to use the following countries: US, CA, GB", []string{"US", "CA", "GB"}},
		{"only 2 letter codes are kept", "x following countries: US, Canada, CAN, JP, ", []string{"US", "JP"}},
		{"surrounding spaces are trimmed", "following countries:  US ,  KR", []string{"US", "KR"}},
		{"marker without any code", "following countries: ", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := parseRejectedCountries(tt.detail)
			if !reflect.DeepEqual(tt.expect, res) {
				t.Errorf("Wrong match.\nexpect: %#v\ngot: %#v", tt.expect, res)
			}
		})
	}
}

// profileServer answers POST /outbound_voice_profiles with the given status/body sequence and
// records the whitelisted_destinations of every request it receives.
func profileServer(t *testing.T, responses []struct {
	status int
	body   string
}, seen *[][]string) *httptest.Server {
	t.Helper()

	call := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			WhitelistedDestinations []string `json:"whitelisted_destinations"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		*seen = append(*seen, req.WhitelistedDestinations)

		idx := call
		if idx >= len(responses) {
			idx = len(responses) - 1
		}
		call++
		w.WriteHeader(responses[idx].status)
		_, _ = w.Write([]byte(responses[idx].body))
	}))
}

func Test_CreateOutboundVoiceProfile_RetriesWithoutRejectedCountries(t *testing.T) {
	first := allTelnyxDestinations[0]
	second := allTelnyxDestinations[1]

	var seen [][]string
	srv := profileServer(t, []struct {
		status int
		body   string
	}{
		{http.StatusUnprocessableEntity, `{"errors":[{"code":"x","detail":"not approved for the following countries: ` + first + `, ` + second + `"}]}`},
		{http.StatusCreated, `{"data":{"id":"profile-retry-1"}}`},
	}, &seen)
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	id, err := c.CreateOutboundVoiceProfile(context.Background(), "retry")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if id != "profile-retry-1" {
		t.Errorf("Wrong match. expect: profile-retry-1, got: %s", id)
	}

	if len(seen) != 2 {
		t.Fatalf("Wrong match. expect: 2 requests, got: %d", len(seen))
	}
	if len(seen[0]) != len(allTelnyxDestinations) {
		t.Errorf("Wrong match. the first request must carry every destination. expect: %d, got: %d", len(allTelnyxDestinations), len(seen[0]))
	}
	if len(seen[1]) != len(allTelnyxDestinations)-2 {
		t.Errorf("Wrong match. the retry must drop exactly the 2 rejected codes. expect: %d, got: %d", len(allTelnyxDestinations)-2, len(seen[1]))
	}
	for _, d := range seen[1] {
		if d == first || d == second {
			t.Errorf("Wrong match. the retry must not contain the rejected code %s", d)
		}
	}
}

func Test_CreateOutboundVoiceProfile_Errors(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		expectErr error // nil means any error
		expectReq int
	}{
		{"unauthorized", http.StatusUnauthorized, ``, ErrInvalidKey, 1},
		{"forbidden", http.StatusForbidden, ``, ErrInvalidKey, 1},
		{"unprocessable without identifiable countries", http.StatusUnprocessableEntity, `{"errors":[{"code":"x","detail":"invalid name"}]}`, nil, 1},
		{"unprocessable with an unparsable body", http.StatusUnprocessableEntity, `not json`, nil, 1},
		{"unexpected status", http.StatusInternalServerError, `oops`, nil, 1},
		{"success with an unparsable body", http.StatusCreated, `not json`, nil, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen [][]string
			srv := profileServer(t, []struct {
				status int
				body   string
			}{{tt.status, tt.body}}, &seen)
			defer srv.Close()

			c := newTelnyxClientWithBase("testkey", srv.URL)
			id, err := c.CreateOutboundVoiceProfile(context.Background(), "p")
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil (id %q)", id)
			}
			if id != "" {
				t.Errorf("Wrong match. expect: empty id, got: %q", id)
			}
			if tt.expectErr != nil && !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if len(seen) != tt.expectReq {
				t.Errorf("Wrong match. expect requests: %d, got: %d", tt.expectReq, len(seen))
			}
		})
	}
}
