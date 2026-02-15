package emailhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownload(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		expectError    bool
	}{
		{
			name:           "successfully_downloads_file",
			serverResponse: "test file content",
			serverStatus:   http.StatusOK,
			expectError:    false,
		},
		{
			name:           "fails_on_non_200_status",
			serverResponse: "error",
			serverStatus:   http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "downloads_empty_file",
			serverResponse: "",
			serverStatus:   http.StatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			ctx := context.Background()
			result, err := download(ctx, server.URL)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if string(result) != tt.serverResponse {
					t.Errorf("Wrong content. expect: %s, got: %s", tt.serverResponse, string(result))
				}
			}
		})
	}
}

func TestDownloadToBase64(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		expectError    bool
	}{
		{
			name:           "successfully_converts_to_base64",
			serverResponse: "test content",
			serverStatus:   http.StatusOK,
			expectError:    false,
		},
		{
			name:           "fails_on_download_error",
			serverResponse: "error",
			serverStatus:   http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			ctx := context.Background()
			result, err := downloadToBase64(ctx, server.URL)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result) == 0 {
					t.Errorf("Expected non-empty base64 string")
				}
			}
		})
	}
}

func TestDownload_FileSizeLimit(t *testing.T) {
	tests := []struct {
		name        string
		contentSize int
		expectError bool
	}{
		{
			name:        "handles_large_file",
			contentSize: 1024 * 1024, // 1 MB
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server with large response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				content := strings.Repeat("a", tt.contentSize)
				_, _ = w.Write([]byte(content))
			}))
			defer server.Close()

			ctx := context.Background()
			result, err := download(ctx, server.URL)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for large file")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result) != tt.contentSize {
					t.Errorf("Wrong content size. expect: %d, got: %d", tt.contentSize, len(result))
				}
			}
		})
	}
}
