package projectconfig

import (
	"os"
	"sync"
)

// Default values
const (
	defaultBaseDomain = "voipbin.net"
	defaultBucketName = "voipbin-voip-media-bucket-europe-west4"
)

var (
	config *ProjectConfig
	once   sync.Once
)

// ProjectConfig holds project-wide configuration values
type ProjectConfig struct {
	// ProjectBaseDomain is the base domain for the project (e.g., "voipbin.net", "example.com")
	ProjectBaseDomain string

	// SIP Domains (derived from ProjectBaseDomain)
	DomainConference      string // conference.{base}
	DomainPSTN            string // pstn.{base}
	DomainTrunkSuffix     string // .trunk.{base}
	DomainRegistrarSuffix string // .registrar.{base}

	// Storage
	ProjectBucketName string
}

// Get returns the singleton ProjectConfig instance.
// Configuration is loaded from environment variables on first call.
func Get() *ProjectConfig {
	once.Do(func() {
		config = load()
	})
	return config
}

// load reads environment variables and returns a new ProjectConfig
func load() *ProjectConfig {
	baseDomain := getEnv("PROJECT_BASE_DOMAIN", defaultBaseDomain)
	bucketName := getEnv("PROJECT_BUCKET_NAME", defaultBucketName)

	return &ProjectConfig{
		ProjectBaseDomain: baseDomain,

		// SIP domains derived from base domain
		DomainConference:      "conference." + baseDomain,
		DomainPSTN:            "pstn." + baseDomain,
		DomainTrunkSuffix:     ".trunk." + baseDomain,
		DomainRegistrarSuffix: ".registrar." + baseDomain,

		// Storage
		ProjectBucketName: bucketName,
	}
}

// getEnv returns the value of an environment variable or a default value if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
