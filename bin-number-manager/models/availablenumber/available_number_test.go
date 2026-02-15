package availablenumber

import (
	"encoding/json"
	"testing"
	"time"

	"monorepo/bin-number-manager/models/number"
)

func TestConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    *AvailableNumber
		expected *WebhookMessage
	}{
		{
			name: "basic conversion",
			input: &AvailableNumber{
				Number:       "+15551234567",
				ProviderName: number.ProviderNameTelnyx,
				Country:      "US",
				Region:       "CA",
				PostalCode:   "94105",
				Features:     []Feature{FeatureVoice, FeatureSMS},
			},
			expected: &WebhookMessage{
				Number:     "+15551234567",
				Country:    "US",
				Region:     "CA",
				PostalCode: "94105",
				Features:   []Feature{FeatureVoice, FeatureSMS},
			},
		},
		{
			name: "with all features",
			input: &AvailableNumber{
				Number:       "+15551234567",
				ProviderName: number.ProviderNameTelnyx,
				Country:      "US",
				Region:       "NY",
				PostalCode:   "10001",
				Features:     []Feature{FeatureVoice, FeatureSMS, FeatureMMS, FeatureFax, FeatureEmergency},
			},
			expected: &WebhookMessage{
				Number:     "+15551234567",
				Country:    "US",
				Region:     "NY",
				PostalCode: "10001",
				Features:   []Feature{FeatureVoice, FeatureSMS, FeatureMMS, FeatureFax, FeatureEmergency},
			},
		},
		{
			name: "empty features",
			input: &AvailableNumber{
				Number:       "+15551234567",
				ProviderName: number.ProviderNameTelnyx,
				Country:      "US",
				Region:       "TX",
				PostalCode:   "75001",
				Features:     []Feature{},
			},
			expected: &WebhookMessage{
				Number:     "+15551234567",
				Country:    "US",
				Region:     "TX",
				PostalCode: "75001",
				Features:   []Feature{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.ConvertWebhookMessage()
			if result.Number != tt.expected.Number {
				t.Errorf("Expected Number %v, got %v", tt.expected.Number, result.Number)
			}
			if result.Country != tt.expected.Country {
				t.Errorf("Expected Country %v, got %v", tt.expected.Country, result.Country)
			}
			if result.Region != tt.expected.Region {
				t.Errorf("Expected Region %v, got %v", tt.expected.Region, result.Region)
			}
			if result.PostalCode != tt.expected.PostalCode {
				t.Errorf("Expected PostalCode %v, got %v", tt.expected.PostalCode, result.PostalCode)
			}
			if len(result.Features) != len(tt.expected.Features) {
				t.Errorf("Expected %d features, got %d", len(tt.expected.Features), len(result.Features))
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     *AvailableNumber
		shouldErr bool
	}{
		{
			name: "valid available number",
			input: &AvailableNumber{
				Number:       "+15551234567",
				ProviderName: number.ProviderNameTelnyx,
				Country:      "US",
				Region:       "CA",
				PostalCode:   "94105",
				Features:     []Feature{FeatureVoice, FeatureSMS},
			},
			shouldErr: false,
		},
		{
			name: "with timestamps",
			input: &AvailableNumber{
				Number:       "+15551234567",
				ProviderName: number.ProviderNameTelnyx,
				Country:      "US",
				Region:       "NY",
				PostalCode:   "10001",
				Features:     []Feature{FeatureVoice},
				TMCreate:     timePtr(time.Now()),
				TMUpdate:     timePtr(time.Now()),
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.input.CreateWebhookEvent()
			if tt.shouldErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error but got %v", err)
			}
			if !tt.shouldErr {
				// Verify it's valid JSON
				var msg WebhookMessage
				if err := json.Unmarshal(data, &msg); err != nil {
					t.Errorf("Failed to unmarshal webhook event: %v", err)
				}
				if msg.Number != tt.input.Number {
					t.Errorf("Expected Number %v, got %v", tt.input.Number, msg.Number)
				}
			}
		})
	}
}

func TestFeatureConstants(t *testing.T) {
	tests := []struct {
		feature  Feature
		expected string
	}{
		{FeatureEmergency, "emergency"},
		{FeatureFax, "fax"},
		{FeatureMMS, "mms"},
		{FeatureSMS, "sms"},
		{FeatureVoice, "voice"},
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			if string(tt.feature) != tt.expected {
				t.Errorf("Expected feature %v, got %v", tt.expected, string(tt.feature))
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
