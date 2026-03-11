package callhandler

import (
	"testing"
)

func Test_extractLocalPorts(t *testing.T) {
	tests := []struct {
		name       string
		input      map[string]interface{}
		excludeTag string
		wantPorts  []int
		wantErr    bool
	}{
		{
			name: "two tags with two streams each, no exclusion",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"from-tag-1": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{
								"streams": []interface{}{
									map[string]interface{}{"local port": float64(30000)},
									map[string]interface{}{"local port": float64(30001)},
								},
							},
						},
					},
					"from-tag-2": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{
								"streams": []interface{}{
									map[string]interface{}{"local port": float64(30010)},
									map[string]interface{}{"local port": float64(30011)},
								},
							},
						},
					},
				},
			},
			excludeTag: "",
			wantPorts:  nil, // order is non-deterministic; checked separately
			wantErr:    false,
		},
		{
			name: "exclude internal tag, return only external ports",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"internal-tag": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{
								"streams": []interface{}{
									map[string]interface{}{"local port": float64(30000)},
									map[string]interface{}{"local port": float64(30001)},
								},
							},
						},
					},
					"external-tag": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{
								"streams": []interface{}{
									map[string]interface{}{"local port": float64(30010)},
									map[string]interface{}{"local port": float64(30011)},
								},
							},
						},
					},
				},
			},
			excludeTag: "internal-tag",
			wantPorts:  []int{30010, 30011},
			wantErr:    false,
		},
		{
			name:    "no tags key",
			input:   map[string]interface{}{"result": "ok"},
			wantErr: true,
		},
		{
			name: "empty tags",
			input: map[string]interface{}{
				"tags": map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "all tags excluded results in error",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"only-tag": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{
								"streams": []interface{}{
									map[string]interface{}{"local port": float64(30000)},
								},
							},
						},
					},
				},
			},
			excludeTag: "only-tag",
			wantErr:    true,
		},
		{
			name: "no streams in media",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"tag1": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{"type": "audio"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "single port",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"tag1": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{
								"streams": []interface{}{
									map[string]interface{}{"local port": float64(30000)},
								},
							},
						},
					},
				},
			},
			wantPorts: []int{30000},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractLocalPorts(tt.input, tt.excludeTag)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractLocalPorts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// For the two-tag no-exclusion case, just check we got 4 ports
			if tt.name == "two tags with two streams each, no exclusion" {
				if len(got) != 4 {
					t.Errorf("expected 4 ports, got %d: %v", len(got), got)
				}
				return
			}

			if len(got) != len(tt.wantPorts) {
				t.Errorf("extractLocalPorts() got %v, want %v", got, tt.wantPorts)
				return
			}
			for i := range got {
				if got[i] != tt.wantPorts[i] {
					t.Errorf("port[%d] = %d, want %d", i, got[i], tt.wantPorts[i])
				}
			}
		})
	}
}

func Test_buildBPFFilter(t *testing.T) {
	tests := []struct {
		name  string
		ports []int
		want  string
	}{
		{
			name:  "single port",
			ports: []int{30000},
			want:  "udp port 30000",
		},
		{
			name:  "two ports",
			ports: []int{30000, 30002},
			want:  "udp port 30000 or udp port 30002",
		},
		{
			name:  "four ports",
			ports: []int{30000, 30001, 30010, 30011},
			want:  "udp port 30000 or udp port 30001 or udp port 30010 or udp port 30011",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBPFFilter(tt.ports)
			if got != tt.want {
				t.Errorf("buildBPFFilter() = %q, want %q", got, tt.want)
			}
		})
	}
}
