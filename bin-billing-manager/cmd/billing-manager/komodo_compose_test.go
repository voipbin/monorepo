package main

import (
	"os"
	"strings"
	"testing"

	"monorepo/bin-billing-manager/internal/config"
)

// The deployment definition must pass the master key list to the container under the name that the
// service reads (the stored Paddle credentials can not be written without it).
func Test_komodoComposePassesTheMasterKeyList(t *testing.T) {
	raw, err := os.ReadFile("../../komodo/docker-compose.yml")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	want := config.EnvConfigEncryptionKeys + "=[[BIN_MANAGER__BILLING_MANAGER_CONFIG_ENCRYPTION_KEYS]]"
	if !strings.Contains(string(raw), "- "+want) {
		t.Errorf("Wrong match. expect the line: %s", want)
	}
}

// The Kubernetes definition passes the master key list as an optional secret reference: a cluster
// whose secret has no such key still starts the pod (the stored credentials are not usable then,
// and the Paddle webhooks are rejected until the credentials are stored).
func Test_k8sDeploymentPassesTheMasterKeyList(t *testing.T) {
	raw, err := os.ReadFile("../../k8s/deployment.yml")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	want := "- name: " + config.EnvConfigEncryptionKeys + "\n" +
		"              valueFrom:\n" +
		"                secretKeyRef:\n" +
		"                  name: voipbin\n" +
		"                  key: " + config.EnvConfigEncryptionKeys + "\n" +
		"                  optional: true\n"
	if !strings.Contains(string(raw), want) {
		t.Errorf("Wrong match. expect the optional secret reference of %s", config.EnvConfigEncryptionKeys)
	}
}
