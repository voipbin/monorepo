package widget

import (
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
		WelcomeMessage:     "welcome!",
		SessionFlowID:      sessionFlowID,
		MessageFlowID:      messageFlowID,
		SessionIdleTimeout: 1800,
		ThemeConfig: &ThemeConfig{
			PrimaryColor: "#112233",
			LogoURL:      "https://example.com/logo.png",
			Position:     WidgetPositionBottomLeft,
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
	if w.WelcomeMessage != "welcome!" {
		t.Errorf("Widget.WelcomeMessage = %v, expected %v", w.WelcomeMessage, "welcome!")
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
	if w.ThemeConfig.Position != WidgetPositionBottomLeft {
		t.Errorf("Widget.ThemeConfig.Position = %v, expected %v", w.ThemeConfig.Position, WidgetPositionBottomLeft)
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
