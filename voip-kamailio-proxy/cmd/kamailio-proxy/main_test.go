package main

import (
	"net"
	"testing"
)

func Test_getKamailioID(t *testing.T) {
	type test struct {
		name      string
		ifaceName string
		wantErr   bool
	}

	// Find real interfaces for happy and no-MAC test cases.
	validIface := ""
	noMACIface := ""
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if len(iface.HardwareAddr) > 0 && validIface == "" {
			validIface = iface.Name
		}
		if len(iface.HardwareAddr) == 0 && noMACIface == "" {
			noMACIface = iface.Name
		}
	}

	tests := []test{
		{
			name:      "interface not found",
			ifaceName: "nonexistent999",
			wantErr:   true,
		},
	}

	if validIface != "" {
		tests = append(tests, test{
			name:      "valid interface with MAC",
			ifaceName: validIface,
			wantErr:   false,
		})
	}

	if noMACIface != "" {
		tests = append(tests, test{
			name:      "interface found but has no MAC address",
			ifaceName: noMACIface,
			wantErr:   true,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := getKamailioID(tt.ifaceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKamailioID(%q) error = %v, wantErr %v", tt.ifaceName, err, tt.wantErr)
			}
			if !tt.wantErr && res == "" {
				t.Errorf("getKamailioID(%q) returned empty string, expected MAC address", tt.ifaceName)
			}
		})
	}
}

// Test_defaults pins the three configuration defaults that the Komodo stack
// definition (komodo/docker-compose.yml, VOIP-1486) relies on rather than
// setting explicitly.
//
//   - defaultRabbitMQQueueListen must stay voip.kamailio.request, the only
//     queue bin-route-manager publishes health-check RPCs to. The compose file
//     omits RABBITMQ_QUEUE_LISTEN entirely, so a change here would silently
//     move the service off the queue it serves.
//   - defaultInterfaceName must stay eth0, the interface a container on the
//     Docker production bridge actually has. The compose file omits
//     INTERFACE_NAME; getKamailioID fails on a wrong name and main() then
//     returns with exit code 0, leaving a container that is up but deaf.
//   - defaultPrometheusListenAddress must stay :2112, which the Prometheus
//     dns_sd job scrapes by port number. The earlier Ansible deployment used
//     :9102; that value would never be scraped.
func Test_defaults(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"rabbitmq queue listen", defaultRabbitMQQueueListen, "voip.kamailio.request"},
		{"interface name", defaultInterfaceName, "eth0"},
		{"prometheus listen address", defaultPrometheusListenAddress, ":2112"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("default = %q, want %q", tt.got, tt.want)
			}
		})
	}
}
