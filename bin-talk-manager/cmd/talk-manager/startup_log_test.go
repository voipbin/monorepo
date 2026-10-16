package main

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"

	"monorepo/bin-talk-manager/internal/config"
)

// Test_logStartup checks the real joonix output of the startup log: no credential value of the
// config may appear in it, and every other setting appears under its flag name.
func Test_logStartup(t *testing.T) {
	// secrets are the Config fields that must never be logged.
	secrets := []string{
		"RabbitMQAddress",
		"DatabaseDSN",
		"RedisPassword",
	}

	// logged maps every other Config field to the key it is logged under.
	logged := map[string]string{
		"PrometheusEndpoint":      "prometheus_endpoint",
		"PrometheusListenAddress": "prometheus_listen_address",
		"RedisAddress":            "redis_address",
		"RedisDatabase":           "redis_database",
	}

	var cfg config.Config
	v := reflect.ValueOf(&cfg).Elem()
	typ := v.Type()

	if len(secrets)+len(logged) != typ.NumField() {
		t.Fatalf("secrets and logged must cover all %d Config fields, got %d and %d", typ.NumField(), len(secrets), len(logged))
	}

	isSecret := map[string]bool{}
	for _, name := range secrets {
		isSecret[name] = true
	}

	expected := map[string]any{}
	secretValues := []string{}
	nextInt := 1000
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		key, isLogged := logged[name]
		if isSecret[name] == isLogged {
			t.Fatalf("Config field %s must be in exactly one of secrets and logged", name)
		}

		var value any
		switch v.Field(i).Kind() {
		case reflect.String:
			s := "SENTINEL_" + name + "_END"
			v.Field(i).SetString(s)
			value = s
			if isSecret[name] {
				secretValues = append(secretValues, s)
			}
		case reflect.Int:
			nextInt++
			v.Field(i).SetInt(int64(nextInt))
			value = float64(nextInt)
		default:
			t.Fatalf("Config field %s has the unsupported kind %s", name, v.Field(i).Kind())
		}

		if isLogged {
			expected[key] = value
		}
	}

	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(joonix.NewFormatter())

	logStartup(logger.WithField("func", "runDaemon"), &cfg)
	out := buf.String()

	for _, s := range secretValues {
		if strings.Contains(out, s) {
			t.Errorf("the startup log contains a secret value: %s", s)
		}
	}

	got := map[string]any{}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("the startup log is not a json line. err: %v", err)
	}

	for key, want := range expected {
		if got[key] != want {
			t.Errorf("the startup log key %s: expected %v, got %v", key, want, got[key])
		}
	}

	if msg, _ := got["message"].(string); !strings.HasPrefix(msg, "Starting talk-manager") {
		t.Errorf("unexpected message: %v", got["message"])
	}

	// The call in runDaemon must use logStartup and must not log the config as one field again.
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("could not read main.go. err: %v", err)
	}
	if !regexp.MustCompile(`(?m)^\s*logStartup\(`).Match(src) {
		t.Errorf("main.go does not call logStartup")
	}
	if regexp.MustCompile(`WithField\(\s*"config"`).Match(src) {
		t.Errorf("main.go logs the config as one field")
	}
}
