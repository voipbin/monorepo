package embedder

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"google.golang.org/genai"
)

const testEmbedModel = "test-embed-model"

// clearGenaiEnv removes every environment variable the genai client reads, so the tests
// do not depend on the host configuration.
func clearGenaiEnv(t *testing.T) {
	t.Helper()

	for _, k := range []string{
		"GOOGLE_API_KEY",
		"GEMINI_API_KEY",
		"GOOGLE_GENAI_USE_VERTEXAI",
		"GOOGLE_CLOUD_PROJECT",
		"GOOGLE_CLOUD_LOCATION",
		"GOOGLE_CLOUD_REGION",
	} {
		t.Setenv(k, "")
	}
}

type embedRequestBody struct {
	Requests []struct {
		TaskType string `json:"taskType"`
		Content  struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"requests"`
}

// embedServer is a fake embedding endpoint. The vector it returns depends on the request text,
// so a caller that mixes up the order of the texts is detected.
type embedServer struct {
	srv *httptest.Server

	mu       sync.Mutex
	paths    []string
	taskType []string
	texts    []string
	calls    atomic.Int32
}

func newEmbedServer(t *testing.T, status int, respond func(text string) string) *embedServer {
	t.Helper()

	es := &embedServer{}
	es.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		es.calls.Add(1)

		raw, _ := io.ReadAll(r.Body)
		var body embedRequestBody
		_ = json.Unmarshal(raw, &body)

		text := ""
		taskType := ""
		if len(body.Requests) > 0 {
			taskType = body.Requests[0].TaskType
			if len(body.Requests[0].Content.Parts) > 0 {
				text = body.Requests[0].Content.Parts[0].Text
			}
		}

		es.mu.Lock()
		es.paths = append(es.paths, r.URL.Path)
		es.taskType = append(es.taskType, taskType)
		es.texts = append(es.texts, text)
		es.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respond(text)))
	}))
	t.Cleanup(es.srv.Close)

	return es
}

func newTestEmbedder(t *testing.T, es *embedServer) *googleEmbedder {
	t.Helper()
	clearGenaiEnv(t)

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:      "test-placeholder",
		Backend:     genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{BaseURL: es.srv.URL},
	})
	if err != nil {
		t.Fatalf("could not create the genai client. err: %v", err)
	}

	return &googleEmbedder{client: client, model: testEmbedModel}
}

func vectorByText(text string) string {
	switch text {
	case "alpha":
		return `{"embeddings":[{"values":[0.1,0.2]}]}`
	case "beta":
		return `{"embeddings":[{"values":[0.3,0.4]}]}`
	}
	return `{"embeddings":[{"values":[9.9]}]}`
}

func Test_EmbedTexts(t *testing.T) {
	tests := []struct {
		name string

		texts  []string
		status int
		resp   func(text string) string

		expectRes   [][]float32
		expectTexts []string
		expectErr   string
	}{
		{
			name: "empty input makes no request",

			texts:  []string{},
			status: http.StatusOK,
			resp:   vectorByText,

			expectRes:   nil,
			expectTexts: nil,
		},
		{
			name: "vectors follow the order of the texts",

			texts:  []string{"beta", "alpha"},
			status: http.StatusOK,
			resp:   vectorByText,

			expectRes:   [][]float32{{0.3, 0.4}, {0.1, 0.2}},
			expectTexts: []string{"beta", "alpha"},
		},
		{
			name: "server error",

			texts:  []string{"alpha"},
			status: http.StatusInternalServerError,
			resp:   func(string) string { return `{"error":{"code":500,"message":"boom","status":"INTERNAL"}}` },

			expectTexts: []string{"alpha"},
			expectErr:   "could not create embedding",
		},
		{
			name: "response without embeddings",

			texts:  []string{"alpha"},
			status: http.StatusOK,
			resp:   func(string) string { return `{"embeddings":[]}` },

			expectTexts: []string{"alpha"},
			expectErr:   "no embedding returned for text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := newEmbedServer(t, tt.status, tt.resp)
			e := newTestEmbedder(t, es)

			res, err := e.EmbedTexts(context.Background(), tt.texts)

			es.mu.Lock()
			gotTexts := append([]string(nil), es.texts...)
			gotTypes := append([]string(nil), es.taskType...)
			gotPaths := append([]string(nil), es.paths...)
			es.mu.Unlock()

			if !reflect.DeepEqual(gotTexts, tt.expectTexts) {
				t.Errorf("Wrong match. expect requested texts: %v, got: %v", tt.expectTexts, gotTexts)
			}
			for i := range gotTypes {
				if gotTypes[i] != "RETRIEVAL_DOCUMENT" {
					t.Errorf("Wrong match. expect task type: RETRIEVAL_DOCUMENT, got: %v", gotTypes[i])
				}
				if !strings.Contains(gotPaths[i], testEmbedModel) {
					t.Errorf("Wrong match. expect model %v in path, got: %v", testEmbedModel, gotPaths[i])
				}
			}

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect nil result on error, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_EmbedText(t *testing.T) {
	tests := []struct {
		name string

		text   string
		status int
		resp   func(text string) string

		expectRes []float32
		expectErr string
	}{
		{
			name: "returns the first embedding",

			text:   "beta",
			status: http.StatusOK,
			resp:   vectorByText,

			expectRes: []float32{0.3, 0.4},
		},
		{
			name: "server error",

			text:   "alpha",
			status: http.StatusTooManyRequests,
			resp: func(string) string {
				return `{"error":{"code":429,"message":"slow down","status":"RESOURCE_EXHAUSTED"}}`
			},

			expectErr: "could not create embedding",
		},
		{
			name: "response without embeddings",

			text:   "alpha",
			status: http.StatusOK,
			resp:   func(string) string { return `{"embeddings":[]}` },

			expectErr: "no embedding returned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := newEmbedServer(t, tt.status, tt.resp)
			e := newTestEmbedder(t, es)

			res, err := e.EmbedText(context.Background(), tt.text)

			es.mu.Lock()
			gotTexts := append([]string(nil), es.texts...)
			gotTypes := append([]string(nil), es.taskType...)
			es.mu.Unlock()

			if !reflect.DeepEqual(gotTexts, []string{tt.text}) {
				t.Errorf("Wrong match. expect requested texts: %v, got: %v", []string{tt.text}, gotTexts)
			}
			if !reflect.DeepEqual(gotTypes, []string{"RETRIEVAL_QUERY"}) {
				t.Errorf("Wrong match. expect task types: [RETRIEVAL_QUERY], got: %v", gotTypes)
			}

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect nil result on error, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_NewGoogleEmbedder_missingProject(t *testing.T) {
	clearGenaiEnv(t)

	res, err := NewGoogleEmbedder(context.Background(), "", "", testEmbedModel)
	if err == nil || !strings.HasPrefix(err.Error(), "could not create genai client") {
		t.Fatalf("Wrong match. expect error prefix: could not create genai client, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect nil embedder, got: %v", res)
	}
}
