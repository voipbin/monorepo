package widget

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
)

func TestWidgetStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	directID := uuid.Must(uuid.NewV4())
	sessionFlowID := uuid.Must(uuid.NewV4())
	messageFlowID := uuid.Must(uuid.NewV4())

	w := Widget{
		Name:               "test widget",
		Status:             StatusActive,
		DirectID:           directID,
		SessionFlowID:      sessionFlowID,
		MessageFlowID:      messageFlowID,
		SessionIdleTimeout: 1800,
		ThemeConfig: &ThemeConfig{
			PrimaryColor:          "#112233",
			SecondaryColor:        "#445566",
			HeaderBackgroundColor: "#778899",
			HeaderTextColor:       "#aabbcc",
			LogoURL:               "https://example.com/logo.png",
			Position:              WidgetPositionBottomLeft,
			ThemeMode:             ThemeModeDark,
			HeaderTitle:           "Support",
			HeaderSubtitle:        "We usually reply in a few minutes",
		},
	}
	w.ID = id
	w.CustomerID = customerID

	if w.ID != id {
		t.Errorf("Widget.ID = %v, expected %v", w.ID, id)
	}
	if w.CustomerID != customerID {
		t.Errorf("Widget.CustomerID = %v, expected %v", w.CustomerID, customerID)
	}
	if w.Name != "test widget" {
		t.Errorf("Widget.Name = %v, expected %v", w.Name, "test widget")
	}
	if w.Status != StatusActive {
		t.Errorf("Widget.Status = %v, expected %v", w.Status, StatusActive)
	}
	if w.DirectID != directID {
		t.Errorf("Widget.DirectID = %v, expected %v", w.DirectID, directID)
	}
	if w.SessionFlowID != sessionFlowID {
		t.Errorf("Widget.SessionFlowID = %v, expected %v", w.SessionFlowID, sessionFlowID)
	}
	if w.MessageFlowID != messageFlowID {
		t.Errorf("Widget.MessageFlowID = %v, expected %v", w.MessageFlowID, messageFlowID)
	}
	if w.SessionIdleTimeout != 1800 {
		t.Errorf("Widget.SessionIdleTimeout = %v, expected %v", w.SessionIdleTimeout, 1800)
	}
	if w.ThemeConfig.PrimaryColor != "#112233" {
		t.Errorf("Widget.ThemeConfig.PrimaryColor = %v, expected %v", w.ThemeConfig.PrimaryColor, "#112233")
	}
	if w.ThemeConfig.SecondaryColor != "#445566" {
		t.Errorf("Widget.ThemeConfig.SecondaryColor = %v, expected %v", w.ThemeConfig.SecondaryColor, "#445566")
	}
	if w.ThemeConfig.HeaderBackgroundColor != "#778899" {
		t.Errorf("Widget.ThemeConfig.HeaderBackgroundColor = %v, expected %v", w.ThemeConfig.HeaderBackgroundColor, "#778899")
	}
	if w.ThemeConfig.HeaderTextColor != "#aabbcc" {
		t.Errorf("Widget.ThemeConfig.HeaderTextColor = %v, expected %v", w.ThemeConfig.HeaderTextColor, "#aabbcc")
	}
	if w.ThemeConfig.Position != WidgetPositionBottomLeft {
		t.Errorf("Widget.ThemeConfig.Position = %v, expected %v", w.ThemeConfig.Position, WidgetPositionBottomLeft)
	}
	if w.ThemeConfig.ThemeMode != ThemeModeDark {
		t.Errorf("Widget.ThemeConfig.ThemeMode = %v, expected %v", w.ThemeConfig.ThemeMode, ThemeModeDark)
	}
	if w.ThemeConfig.HeaderTitle != "Support" {
		t.Errorf("Widget.ThemeConfig.HeaderTitle = %v, expected %v", w.ThemeConfig.HeaderTitle, "Support")
	}
	if w.ThemeConfig.HeaderSubtitle != "We usually reply in a few minutes" {
		t.Errorf("Widget.ThemeConfig.HeaderSubtitle = %v, expected %v", w.ThemeConfig.HeaderSubtitle, "We usually reply in a few minutes")
	}
}

func TestThemeConfigIndicatorAndShapeFields(t *testing.T) {
	connectingEnabled := true
	typingEnabled := false

	tc := ThemeConfig{
		ConnectingIndicatorEnabled: &connectingEnabled,
		ConnectingIndicatorText:    "Please wait…",
		TypingIndicatorEnabled:     &typingEnabled,
		BorderRadius:               BorderRadiusPill,
		FontSize:                   FontSizeLarge,
	}

	if tc.ConnectingIndicatorEnabled == nil || *tc.ConnectingIndicatorEnabled != true {
		t.Errorf("ThemeConfig.ConnectingIndicatorEnabled = %v, expected pointer to true", tc.ConnectingIndicatorEnabled)
	}
	if tc.ConnectingIndicatorText != "Please wait…" {
		t.Errorf("ThemeConfig.ConnectingIndicatorText = %v, expected %v", tc.ConnectingIndicatorText, "Please wait…")
	}
	if tc.TypingIndicatorEnabled == nil || *tc.TypingIndicatorEnabled != false {
		t.Errorf("ThemeConfig.TypingIndicatorEnabled = %v, expected pointer to false", tc.TypingIndicatorEnabled)
	}
	if tc.BorderRadius != BorderRadiusPill {
		t.Errorf("ThemeConfig.BorderRadius = %v, expected %v", tc.BorderRadius, BorderRadiusPill)
	}
	if tc.FontSize != FontSizeLarge {
		t.Errorf("ThemeConfig.FontSize = %v, expected %v", tc.FontSize, FontSizeLarge)
	}
}

// TestThemeConfigBoolOmittedVsFalseRoundTrip verifies the specific
// *bool/omitempty behavior the design doc calls out as the highest-risk
// regression: a nil (never-set) *bool field must marshal as an ABSENT
// JSON key (falls back to the platform default of "enabled" at render
// time), while an explicit false must marshal as a present "false" key
// -- these two states must remain distinguishable through a full
// marshal/unmarshal round-trip, or an unrelated field edit could
// silently flip an existing widget's default-on indicator off.
func TestThemeConfigBoolOmittedVsFalseRoundTrip(t *testing.T) {
	explicitFalse := false

	tests := []struct {
		name           string
		tc             ThemeConfig
		wantKeyPresent bool
	}{
		{
			name:           "nil (never set) omits the key entirely",
			tc:             ThemeConfig{ConnectingIndicatorEnabled: nil},
			wantKeyPresent: false,
		},
		{
			name:           "explicit false keeps the key present",
			tc:             ThemeConfig{ConnectingIndicatorEnabled: &explicitFalse},
			wantKeyPresent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.tc)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var raw map[string]any
			if errUnmarshal := json.Unmarshal(data, &raw); errUnmarshal != nil {
				t.Fatalf("json.Unmarshal() error = %v", errUnmarshal)
			}

			_, keyPresent := raw["connecting_indicator_enabled"]
			if keyPresent != tt.wantKeyPresent {
				t.Errorf("connecting_indicator_enabled key present = %v, expected %v (marshaled: %s)", keyPresent, tt.wantKeyPresent, string(data))
			}

			// Round-trip back into a fresh struct and confirm the
			// nil-vs-false distinction survives unmarshal too.
			var roundTripped ThemeConfig
			if errUnmarshal := json.Unmarshal(data, &roundTripped); errUnmarshal != nil {
				t.Fatalf("json.Unmarshal() into ThemeConfig error = %v", errUnmarshal)
			}
			if tt.tc.ConnectingIndicatorEnabled == nil {
				if roundTripped.ConnectingIndicatorEnabled != nil {
					t.Errorf("round-tripped ConnectingIndicatorEnabled = %v, expected nil", roundTripped.ConnectingIndicatorEnabled)
				}
			} else {
				if roundTripped.ConnectingIndicatorEnabled == nil || *roundTripped.ConnectingIndicatorEnabled != *tt.tc.ConnectingIndicatorEnabled {
					t.Errorf("round-tripped ConnectingIndicatorEnabled = %v, expected pointer to %v", roundTripped.ConnectingIndicatorEnabled, *tt.tc.ConnectingIndicatorEnabled)
				}
			}
		})
	}
}

func TestBorderRadiusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant BorderRadius
		expected string
	}{
		{"border_radius_sharp", BorderRadiusSharp, "sharp"},
		{"border_radius_rounded", BorderRadiusRounded, "rounded"},
		{"border_radius_pill", BorderRadiusPill, "pill"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestFontSizeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant FontSize
		expected string
	}{
		{"font_size_compact", FontSizeCompact, "compact"},
		{"font_size_default", FontSizeDefault, "default"},
		{"font_size_large", FontSizeLarge, "large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestThemeModeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ThemeMode
		expected string
	}{
		{"theme_mode_light", ThemeModeLight, "light"},
		{"theme_mode_dark", ThemeModeDark, "dark"},
		{"theme_mode_auto", ThemeModeAuto, "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_active", StatusActive, "active"},
		{"status_inactive", StatusInactive, "inactive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestWidgetPositionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant WidgetPosition
		expected string
	}{
		{"position_bottom_right", WidgetPositionBottomRight, "bottom_right"},
		{"position_bottom_left", WidgetPositionBottomLeft, "bottom_left"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDefaultSessionIdleTimeout(t *testing.T) {
	if DefaultSessionIdleTimeout != 1800 {
		t.Errorf("DefaultSessionIdleTimeout = %v, expected %v", DefaultSessionIdleTimeout, 1800)
	}
}
