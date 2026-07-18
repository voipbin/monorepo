package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	wcwidget "monorepo/bin-webchat-manager/models/widget"
	"testing"
)

func Test_convertWebchatThemeConfig(t *testing.T) {
	t.Run("nil pointer returns nil", func(t *testing.T) {
		if got := convertWebchatThemeConfig(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("empty struct returns zero-value ThemeConfig, not nil", func(t *testing.T) {
		got := convertWebchatThemeConfig(&openapi_server.WebchatManagerWidgetThemeConfig{})
		if got == nil {
			t.Fatalf("expected non-nil ThemeConfig, got nil")
		}
		if *got != (wcwidget.ThemeConfig{}) {
			t.Errorf("expected zero-value ThemeConfig, got %+v", got)
		}
	})

	t.Run("all 9 fields map through correctly", func(t *testing.T) {
		primaryColor := "#1a73e8"
		secondaryColor := "#f5f5f5"
		headerBackgroundColor := "#111827"
		headerTextColor := "#f9fafb"
		logoURL := "https://cdn.example.com/logo.png"
		position := openapi_server.WebchatManagerWidgetPosition("bottom_left")
		themeMode := openapi_server.WebchatManagerWidgetThemeMode("dark")
		headerTitle := "Support"
		headerSubtitle := "We usually reply in a few minutes"

		req := &openapi_server.WebchatManagerWidgetThemeConfig{
			PrimaryColor:          &primaryColor,
			SecondaryColor:        &secondaryColor,
			HeaderBackgroundColor: &headerBackgroundColor,
			HeaderTextColor:       &headerTextColor,
			LogoUrl:               &logoURL,
			Position:              &position,
			ThemeMode:             &themeMode,
			HeaderTitle:           &headerTitle,
			HeaderSubtitle:        &headerSubtitle,
		}

		got := convertWebchatThemeConfig(req)
		if got == nil {
			t.Fatalf("expected non-nil ThemeConfig, got nil")
		}

		expect := &wcwidget.ThemeConfig{
			PrimaryColor:          primaryColor,
			SecondaryColor:        secondaryColor,
			HeaderBackgroundColor: headerBackgroundColor,
			HeaderTextColor:       headerTextColor,
			LogoURL:               logoURL,
			Position:              wcwidget.WidgetPositionBottomLeft,
			ThemeMode:             wcwidget.ThemeModeDark,
			HeaderTitle:           headerTitle,
			HeaderSubtitle:        headerSubtitle,
		}
		if *got != *expect {
			t.Errorf("wrong conversion.\nexpect: %+v\ngot: %+v", expect, got)
		}
	})

	t.Run("only primary_color set, other 8 fields stay zero-value", func(t *testing.T) {
		primaryColor := "#1a73e8"
		req := &openapi_server.WebchatManagerWidgetThemeConfig{
			PrimaryColor: &primaryColor,
		}

		got := convertWebchatThemeConfig(req)
		if got == nil {
			t.Fatalf("expected non-nil ThemeConfig, got nil")
		}
		if got.PrimaryColor != primaryColor {
			t.Errorf("PrimaryColor: expect %q, got %q", primaryColor, got.PrimaryColor)
		}

		expectRest := wcwidget.ThemeConfig{PrimaryColor: primaryColor}
		if *got != expectRest {
			t.Errorf("expected all other fields to stay zero-value.\nexpect: %+v\ngot: %+v", expectRest, got)
		}
	})
}
