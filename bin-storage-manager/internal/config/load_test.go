package config

import (
	"reflect"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

func Test_LoadGlobalConfig_mapsEveryField(t *testing.T) {
	oldConfig := globalConfig
	t.Cleanup(func() {
		globalConfig = oldConfig
		// leave the once spent, as it is after any other test has called LoadGlobalConfig
		once = sync.Once{}
		once.Do(func() {})
		viper.Reset()
	})

	viper.Reset()
	once = sync.Once{}

	values := map[string]any{
		"database_dsn":              "dsn-value",
		"prometheus_endpoint":       "/prom",
		"prometheus_listen_address": ":9999",
		"rabbitmq_address":          "mq-value",
		"redis_address":             "redis-value",
		"redis_database":            7,
		"gcp_project_id":            "project-value",
		"gcp_bucket_name_media":     "media-value",
		"gcp_bucket_name_tmp":       "tmp-value",
	}
	for k, v := range values {
		viper.Set(k, v)
	}
	fakeSecret := "x"
	viper.Set("redis_password", fakeSecret)

	LoadGlobalConfig()

	expect := Config{
		DatabaseDSN:             "dsn-value",
		PrometheusEndpoint:      "/prom",
		PrometheusListenAddress: ":9999",
		RabbitMQAddress:         "mq-value",
		RedisAddress:            "redis-value",
		RedisDatabase:           7,
		RedisPassword:           fakeSecret,
		GCPProjectID:            "project-value",
		GCPBucketNameMedia:      "media-value",
		GCPBucketNameTmp:        "tmp-value",
	}
	if !reflect.DeepEqual(*Get(), expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, *Get())
	}

	// the second call is ignored while the once is spent
	viper.Set("database_dsn", "changed")
	LoadGlobalConfig()
	if Get().DatabaseDSN != "dsn-value" {
		t.Errorf("Wrong match. expect: dsn-value, got: %s", Get().DatabaseDSN)
	}
}
