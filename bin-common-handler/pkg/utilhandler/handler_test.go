package utilhandler

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

// Test utilHandler methods to increase coverage

func TestUtilHandler_EmailIsValid(t *testing.T) {
	h := NewUtilHandler()

	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{
			name:  "valid email",
			email: "test@example.com",
			want:  true,
		},
		{
			name:  "invalid email",
			email: "invalid",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.EmailIsValid(tt.email)
			if got != tt.want {
				t.Errorf("EmailIsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUtilHandler_HashGenerate(t *testing.T) {
	h := NewUtilHandler()

	tests := []struct {
		name    string
		org     string
		cost    int
		wantErr bool
	}{
		{
			name:    "valid generation",
			org:     "password",
			cost:    10,
			wantErr: false,
		},
		{
			name:    "invalid cost",
			org:     "password",
			cost:    2,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.HashGenerate(tt.org, tt.cost)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashGenerate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUtilHandler_HashCheckPassword(t *testing.T) {
	h := NewUtilHandler()

	// Generate a hash first
	hash, err := h.HashGenerate("password123", 10)
	if err != nil {
		t.Fatalf("Failed to generate hash: %v", err)
	}

	tests := []struct {
		name       string
		password   string
		hashString string
		want       bool
	}{
		{
			name:       "correct password",
			password:   "password123",
			hashString: hash,
			want:       true,
		},
		{
			name:       "incorrect password",
			password:   "wrongpassword",
			hashString: hash,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.HashCheckPassword(tt.password, tt.hashString)
			if got != tt.want {
				t.Errorf("HashCheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUtilHandler_UUIDCreate(t *testing.T) {
	h := NewUtilHandler()

	uuid1 := h.UUIDCreate()
	uuid2 := h.UUIDCreate()

	if uuid1 == uuid.Nil {
		t.Error("UUIDCreate() returned Nil UUID")
	}

	if uuid1 == uuid2 {
		t.Error("UUIDCreate() returned same UUID twice")
	}
}

func TestUtilHandler_NewV5UUID(t *testing.T) {
	h := NewUtilHandler()

	namespace := uuid.Must(uuid.NewV4())
	data := "test-data"

	uuid1 := h.NewV5UUID(namespace, data)
	uuid2 := h.NewV5UUID(namespace, data)

	if uuid1 == uuid.Nil {
		t.Error("NewV5UUID() returned Nil UUID")
	}

	if uuid1 != uuid2 {
		t.Error("NewV5UUID() should return same UUID for same input")
	}

	uuid3 := h.NewV5UUID(namespace, "different-data")
	if uuid1 == uuid3 {
		t.Error("NewV5UUID() should return different UUID for different data")
	}
}

func TestUtilHandler_StringGenerateRandom(t *testing.T) {
	h := NewUtilHandler()

	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			name:    "valid size",
			size:    16,
			wantErr: false,
		},
		{
			name:    "small size",
			size:    4,
			wantErr: false,
		},
		{
			name:    "large size",
			size:    100,
			wantErr: false,
		},
		{
			name:    "zero size",
			size:    0,
			wantErr: true,
		},
		{
			name:    "negative size",
			size:    -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := h.StringGenerateRandom(tt.size)

			if (err != nil) != tt.wantErr {
				t.Errorf("StringGenerateRandom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got) > tt.size {
					t.Errorf("StringGenerateRandom() length = %d, want <= %d", len(got), tt.size)
				}
				if len(got) == 0 {
					t.Error("StringGenerateRandom() returned empty string")
				}
			}
		})
	}
}

func TestUtilHandler_ParseFiltersFromRequestBody(t *testing.T) {
	h := NewUtilHandler()

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid JSON",
			data:    []byte(`{"key": "value"}`),
			wantErr: false,
		},
		{
			name:    "empty JSON",
			data:    []byte(`{}`),
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			data:    []byte(`{invalid}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.ParseFiltersFromRequestBody(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFiltersFromRequestBody() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUtilHandler_TimeGetCurTime(t *testing.T) {
	h := NewUtilHandler()
	result := h.TimeGetCurTime()

	if len(result) == 0 {
		t.Error("TimeGetCurTime() returned empty string")
	}

	// Should contain ISO 8601 format markers
	if !strings.Contains(result, "T") || !strings.Contains(result, "Z") {
		t.Errorf("TimeGetCurTime() = %s, doesn't look like ISO 8601", result)
	}
}

func TestUtilHandler_TimeGetCurTimeAdd(t *testing.T) {
	h := NewUtilHandler()
	duration := 1 * time.Hour
	result := h.TimeGetCurTimeAdd(duration)

	if len(result) == 0 {
		t.Error("TimeGetCurTimeAdd() returned empty string")
	}

	// Should contain ISO 8601 format markers
	if !strings.Contains(result, "T") || !strings.Contains(result, "Z") {
		t.Errorf("TimeGetCurTimeAdd() = %s, doesn't look like ISO 8601", result)
	}
}

func TestUtilHandler_TimeGetCurTimeRFC3339(t *testing.T) {
	h := NewUtilHandler()
	result := h.TimeGetCurTimeRFC3339()

	if len(result) == 0 {
		t.Error("TimeGetCurTimeRFC3339() returned empty string")
	}

	// Try parsing as RFC3339
	_, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("TimeGetCurTimeRFC3339() = %s, not valid RFC3339: %v", result, err)
	}
}

func TestUtilHandler_TimeParse(t *testing.T) {
	h := NewUtilHandler()

	tests := []struct {
		name       string
		timeString string
		wantZero   bool
	}{
		{
			name:       "valid ISO 8601",
			timeString: "2024-01-15T10:30:45.123456Z",
			wantZero:   false,
		},
		{
			name:       "invalid time",
			timeString: "not-a-time",
			wantZero:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.TimeParse(tt.timeString)

			if tt.wantZero {
				if !result.IsZero() {
					t.Errorf("TimeParse() = %v, want zero time", result)
				}
			} else {
				if result.IsZero() {
					t.Error("TimeParse() returned zero time for valid input")
				}
			}
		})
	}
}

func TestUtilHandler_TimeParseWithError(t *testing.T) {
	h := NewUtilHandler()

	tests := []struct {
		name       string
		timeString string
		wantErr    bool
	}{
		{
			name:       "valid ISO 8601",
			timeString: "2024-01-15T10:30:45.123456Z",
			wantErr:    false,
		},
		{
			name:       "invalid time",
			timeString: "not-a-time",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.TimeParseWithError(tt.timeString)

			if (err != nil) != tt.wantErr {
				t.Errorf("TimeParseWithError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUtilHandler_TimeNow(t *testing.T) {
	h := NewUtilHandler()
	result := h.TimeNow()

	if result == nil {
		t.Error("TimeNow() returned nil")
	}

	if result.IsZero() {
		t.Error("TimeNow() returned zero time")
	}
}

func TestUtilHandler_TimeNowAdd(t *testing.T) {
	h := NewUtilHandler()
	duration := 1 * time.Hour
	result := h.TimeNowAdd(duration)

	if result == nil {
		t.Error("TimeNowAdd() returned nil")
	}

	if result.IsZero() {
		t.Error("TimeNowAdd() returned zero time")
	}
}

func TestUtilHandler_IsDeleted(t *testing.T) {
	h := NewUtilHandler()

	now := time.Now()

	tests := []struct {
		name string
		t    *time.Time
		want bool
	}{
		{
			name: "nil time is not deleted",
			t:    nil,
			want: false,
		},
		{
			name: "non-nil time is deleted",
			t:    &now,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.IsDeleted(tt.t)
			if got != tt.want {
				t.Errorf("IsDeleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUtilHandler_URLParseFilters(t *testing.T) {
	h := NewUtilHandler()

	tests := []struct {
		name    string
		urlStr  string
		wantLen int
	}{
		{
			name:    "URL with filter",
			urlStr:  "http://example.com?filter_name=value",
			wantLen: 1,
		},
		{
			name:    "URL without filter",
			urlStr:  "http://example.com?name=value",
			wantLen: 0,
		},
		{
			name:    "URL with multiple filters",
			urlStr:  "http://example.com?filter_a=1&filter_b=2",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.urlStr)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			result := h.URLParseFilters(u)

			if len(result) != tt.wantLen {
				t.Errorf("URLParseFilters() returned %d filters, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestUtilHandler_URLMergeFilters(t *testing.T) {
	h := NewUtilHandler()

	tests := []struct {
		name    string
		uri     string
		filters map[string]string
		wantLen int // Minimal check: length should be >= original URI length
	}{
		{
			name: "merge single filter",
			uri:  "http://example.com?existing=param",
			filters: map[string]string{
				"name": "value",
			},
			wantLen: 1,
		},
		{
			name:    "merge empty filters",
			uri:     "http://example.com",
			filters: map[string]string{},
			wantLen: 0,
		},
		{
			name: "merge multiple filters",
			uri:  "http://example.com",
			filters: map[string]string{
				"a": "1",
				"b": "2",
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.URLMergeFilters(tt.uri, tt.filters)

			if len(result) < len(tt.uri) {
				t.Errorf("URLMergeFilters() result shorter than input")
			}

			if tt.wantLen == 0 && result != tt.uri {
				t.Errorf("URLMergeFilters() with empty filters should return original URI")
			}

			// Check that each filter is present
			for k, v := range tt.filters {
				if !strings.Contains(result, "filter_"+url.QueryEscape(k)) {
					t.Errorf("URLMergeFilters() missing filter %s", k)
				}
				if !strings.Contains(result, url.QueryEscape(v)) {
					t.Errorf("URLMergeFilters() missing value %s", v)
				}
			}
		})
	}
}
