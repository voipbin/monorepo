package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	wcwidget "monorepo/bin-webchat-manager/models/widget"
	"testing"
)

func Test_convertWebchatThemeConfig(t *testing.T) {
	t.Run("nil pointer returns nil", func(t *testing.T) {
		got, err := convertWebchatThemeConfig(nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("empty struct returns zero-value ThemeConfig, not nil", func(t *testing.T) {
		got, err := convertWebchatThemeConfig(&openapi_server.WebchatManagerWidgetThemeConfig{})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
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

		got, err := convertWebchatThemeConfig(req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
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

		got, err := convertWebchatThemeConfig(req)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
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

	t.Run("invalid primary_color hex format is rejected", func(t *testing.T) {
		bad := "blue"
		req := &openapi_server.WebchatManagerWidgetThemeConfig{PrimaryColor: &bad}

		got, err := convertWebchatThemeConfig(req)
		if err == nil {
			t.Fatalf("expected an error for invalid hex color, got nil")
		}
		if got != nil {
			t.Errorf("expected nil ThemeConfig on validation failure, got %+v", got)
		}
	})

	t.Run("invalid secondary_color hex format (missing #) is rejected", func(t *testing.T) {
		bad := "1a73e8"
		req := &openapi_server.WebchatManagerWidgetThemeConfig{SecondaryColor: &bad}

		if _, err := convertWebchatThemeConfig(req); err == nil {
			t.Fatalf("expected an error for invalid hex color, got nil")
		}
	})

	t.Run("invalid header_background_color hex format (3-digit shorthand) is rejected", func(t *testing.T) {
		bad := "#fff"
		req := &openapi_server.WebchatManagerWidgetThemeConfig{HeaderBackgroundColor: &bad}

		if _, err := convertWebchatThemeConfig(req); err == nil {
			t.Fatalf("expected an error for 3-digit shorthand hex color, got nil")
		}
	})

	t.Run("invalid header_text_color hex format is rejected", func(t *testing.T) {
		bad := "#gggggg"
		req := &openapi_server.WebchatManagerWidgetThemeConfig{HeaderTextColor: &bad}

		if _, err := convertWebchatThemeConfig(req); err == nil {
			t.Fatalf("expected an error for invalid hex color, got nil")
		}
	})

	t.Run("invalid theme_mode enum value is rejected", func(t *testing.T) {
		bad := openapi_server.WebchatManagerWidgetThemeMode("rainbow")
		req := &openapi_server.WebchatManagerWidgetThemeConfig{ThemeMode: &bad}

		got, err := convertWebchatThemeConfig(req)
		if err == nil {
			t.Fatalf("expected an error for invalid theme_mode, got nil")
		}
		if got != nil {
			t.Errorf("expected nil ThemeConfig on validation failure, got %+v", got)
		}
	})

	t.Run("header_title over 100 characters is rejected", func(t *testing.T) {
		tooLong := make([]byte, 101)
		for i := range tooLong {
			tooLong[i] = 'a'
		}
		bad := string(tooLong)
		req := &openapi_server.WebchatManagerWidgetThemeConfig{HeaderTitle: &bad}

		if _, err := convertWebchatThemeConfig(req); err == nil {
			t.Fatalf("expected an error for header_title exceeding 100 chars, got nil")
		}
	})

	t.Run("header_title at exactly 100 characters is accepted", func(t *testing.T) {
		exact := make([]byte, 100)
		for i := range exact {
			exact[i] = 'a'
		}
		val := string(exact)
		req := &openapi_server.WebchatManagerWidgetThemeConfig{HeaderTitle: &val}

		got, err := convertWebchatThemeConfig(req)
		if err != nil {
			t.Errorf("expected no error at exactly the max length, got %v", err)
		}
		if got == nil || got.HeaderTitle != val {
			t.Errorf("expected HeaderTitle to be set to the 100-char value")
		}
	})

	t.Run("header_subtitle over 200 characters is rejected", func(t *testing.T) {
		tooLong := make([]byte, 201)
		for i := range tooLong {
			tooLong[i] = 'a'
		}
		bad := string(tooLong)
		req := &openapi_server.WebchatManagerWidgetThemeConfig{HeaderSubtitle: &bad}

		if _, err := convertWebchatThemeConfig(req); err == nil {
			t.Fatalf("expected an error for header_subtitle exceeding 200 chars, got nil")
		}
	})
}
