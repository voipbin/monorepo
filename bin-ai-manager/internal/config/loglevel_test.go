package config

import (
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// restoreLogger puts the global logrus level, output and hooks back when the test ends.
func restoreLogger(t *testing.T) {
	t.Helper()

	prev := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetLevel(prev)
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})
}

func Test_applyLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		value string

		expectLevel logrus.Level
		expectWarn  bool
	}{
		{"debug", "debug", logrus.DebugLevel, false},
		{"debug upper case and spaces", " DEBUG ", logrus.DebugLevel, false},
		{"info", "info", logrus.InfoLevel, false},
		{"empty keeps info", "", logrus.InfoLevel, false},
		{"warn", "warn", logrus.WarnLevel, false},
		{"error", "error", logrus.ErrorLevel, false},
		{"unknown keeps info and warns", "verbose", logrus.InfoLevel, true},
		{"trace is not offered", "trace", logrus.InfoLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreLogger(t)
			hook := logrustest.NewLocal(logrus.StandardLogger())
			logrus.SetLevel(logrus.PanicLevel)

			applyLogLevel(tt.value)

			if got := logrus.GetLevel(); got != tt.expectLevel {
				t.Errorf("level: expect %v, got %v", tt.expectLevel, got)
			}

			warned := false
			for _, e := range hook.AllEntries() {
				if e.Level == logrus.WarnLevel {
					warned = true
				}
			}
			if warned != tt.expectWarn {
				t.Errorf("warn: expect %v, got %v", tt.expectWarn, warned)
			}
		})
	}
}

// loadWithEnv runs the real start path: Bootstrap binds the settings, LoadGlobalConfig applies the level.
func loadWithEnv(t *testing.T, level string) {
	t.Helper()

	restoreLogger(t)
	t.Setenv("LOG_LEVEL", level)

	viper.Reset()
	prevConfig := globalConfig
	t.Cleanup(func() {
		viper.Reset()
		globalConfig = prevConfig
		once = sync.Once{}
	})
	once = sync.Once{}

	if err := Bootstrap(&cobra.Command{}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	LoadGlobalConfig()
}

// Without any setting the service starts at info, so a Debug line is not written.
func Test_LoadGlobalConfig_defaultIsInfo(t *testing.T) {
	loadWithEnv(t, "")
	hook := logrustest.NewLocal(logrus.StandardLogger())

	if got := logrus.GetLevel(); got != logrus.InfoLevel {
		t.Errorf("level: expect info, got %v", got)
	}

	logrus.Debug("debug line")
	logrus.Info("info line")
	entries := hook.AllEntries()
	if len(entries) != 1 || entries[0].Message != "info line" {
		t.Errorf("only the info line must be written, got %d entries", len(entries))
	}
}

// LOG_LEVEL=debug turns the Debug lines on.
func Test_LoadGlobalConfig_debugFromEnv(t *testing.T) {
	loadWithEnv(t, "debug")
	hook := logrustest.NewLocal(logrus.StandardLogger())

	logrus.Debug("debug line")
	if len(hook.AllEntries()) != 1 {
		t.Errorf("the debug line must be written, got %d entries", len(hook.AllEntries()))
	}
}

// Bootstrap alone leaves the level at info even if a test or an earlier run set debug.
func Test_Bootstrap_setsInfo(t *testing.T) {
	restoreLogger(t)
	logrus.SetLevel(logrus.DebugLevel)
	viper.Reset()
	t.Cleanup(viper.Reset)

	if err := Bootstrap(&cobra.Command{}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if got := logrus.GetLevel(); got != logrus.InfoLevel {
		t.Errorf("level: expect info, got %v", got)
	}
}

// The flag --log_level turns the Debug lines on, like the environment variable.
func Test_LoadGlobalConfig_debugFromFlag(t *testing.T) {
	restoreLogger(t)
	t.Setenv("LOG_LEVEL", "")
	viper.Reset()
	prevConfig := globalConfig
	t.Cleanup(func() {
		viper.Reset()
		globalConfig = prevConfig
		once = sync.Once{}
	})
	once = sync.Once{}

	rootCmd := &cobra.Command{}
	if err := Bootstrap(rootCmd); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := rootCmd.PersistentFlags().Set("log_level", "debug"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	LoadGlobalConfig()

	if got := logrus.GetLevel(); got != logrus.DebugLevel {
		t.Errorf("level: expect debug, got %v", got)
	}
}
