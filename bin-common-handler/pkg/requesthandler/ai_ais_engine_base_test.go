package requesthandler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	amai "monorepo/bin-ai-manager/models/ai"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func ptrStr(s string) *string { return &s }

// Test_AIV1AI_EngineBaseKeyWire checks that the key pointer keeps its three
// states across the wire: nil is omitted, a pointer to "" is serialized, a
// value is serialized. The captured bytes are unmarshaled into the ai-manager
// DTO (the type the listenhandler unmarshals). JSON null has the same wire
// result as omitted because the pointer is nil in both cases.
func Test_AIV1AI_EngineBaseKeyWire(t *testing.T) {
	tests := []struct {
		name string
		key  *string

		expectInWire bool
	}{
		{"nil (omitted or null)", nil, false},
		{"empty string", ptrStr(""), true},
		{"value", ptrStr("k-1"), true},
	}

	for _, tt := range tests {
		for _, op := range []string{"create", "update"} {
			t.Run(op+"/"+tt.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				mockSock := sockhandler.NewMockSockHandler(mc)
				h := requestHandler{sock: mockSock}

				var wire []byte
				mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, _ string, req *sock.Request) (*sock.Response, error) {
						wire = req.Data
						return &sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(`{}`)}, nil
					})

				var err error
				if op == "create" {
					_, err = h.AIV1AICreate(context.Background(), uuid.Nil, "n", "d", amai.TypeNone, "customendpoint.m", nil, "", "https://a.example.com/v1", tt.key,
						uuid.Nil, "", "", "", "", "", nil, nil, false)
				} else {
					_, err = h.AIV1AIUpdate(context.Background(), uuid.Nil, "n", "d", amai.TypeNone, "customendpoint.m", nil, "", "https://a.example.com/v1", tt.key,
						uuid.Nil, "", "", "", "", "", nil, nil, false)
				}
				if err != nil {
					t.Fatalf("request failed: %v", err)
				}

				if got := strings.Contains(string(wire), `"engine_base_key"`); got != tt.expectInWire {
					t.Errorf("Wrong wire presence. expect: %v, got: %v, wire: %s", tt.expectInWire, got, wire)
				}
				if !strings.Contains(string(wire), `"engine_base_url":"https://a.example.com/v1"`) {
					t.Errorf("url missing in wire: %s", wire)
				}

				var gotURL string
				var gotKey *string
				if op == "create" {
					var d amrequest.V1DataAIsPost
					if err := json.Unmarshal(wire, &d); err != nil {
						t.Fatalf("unmarshal: %v", err)
					}
					gotURL, gotKey = d.EngineBaseURL, d.EngineBaseKey
				} else {
					var d amrequest.V1DataAIsIDPut
					if err := json.Unmarshal(wire, &d); err != nil {
						t.Fatalf("unmarshal: %v", err)
					}
					gotURL, gotKey = d.EngineBaseURL, d.EngineBaseKey
				}
				if gotURL != "https://a.example.com/v1" {
					t.Errorf("Wrong url: %q", gotURL)
				}
				if (gotKey == nil) != (tt.key == nil) || (gotKey != nil && *gotKey != *tt.key) {
					t.Errorf("Wrong key. expect: %v, got: %v", tt.key, gotKey)
				}
			})
		}
	}
}

// Test_V1DataAIs_EngineBaseKeyNullIsOmitted pins the decode side: explicit
// JSON null and a missing key both give a nil pointer, "" gives a pointer to "".
func Test_V1DataAIs_EngineBaseKeyNullIsOmitted(t *testing.T) {
	tests := []struct {
		body      string
		expectNil bool
		expect    string
	}{
		{`{}`, true, ""},
		{`{"engine_base_key":null}`, true, ""},
		{`{"engine_base_key":""}`, false, ""},
		{`{"engine_base_key":"v"}`, false, "v"},
	}
	for _, tt := range tests {
		t.Run(tt.body, func(t *testing.T) {
			var p amrequest.V1DataAIsPost
			var u amrequest.V1DataAIsIDPut
			if err := json.Unmarshal([]byte(tt.body), &p); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tt.body), &u); err != nil {
				t.Fatal(err)
			}
			for _, k := range []*string{p.EngineBaseKey, u.EngineBaseKey} {
				if (k == nil) != tt.expectNil || (k != nil && *k != tt.expect) {
					t.Errorf("Wrong key state: %v", k)
				}
			}
		})
	}
}
