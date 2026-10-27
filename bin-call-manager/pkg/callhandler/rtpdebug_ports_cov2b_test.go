package callhandler

import (
	"reflect"
	"sort"
	"testing"
)

func Test_extractLocalPorts_cov2b_skippedShapes(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		wantPorts []int
		wantErr   bool
	}{
		{
			name:    "tags is not a map",
			input:   map[string]interface{}{"tags": "not-a-map"},
			wantErr: true,
		},
		{
			name: "non-map tag data, missing medias, non-list medias are skipped",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"bad-tag":      "text",
					"no-medias":    map[string]interface{}{},
					"medias-wrong": map[string]interface{}{"medias": "text"},
					"good": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{"streams": []interface{}{map[string]interface{}{"local port": float64(30000)}}},
						},
					},
				},
			},
			wantPorts: []int{30000},
		},
		{
			name: "non-map media, missing streams, non-list streams are skipped",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"t": map[string]interface{}{
						"medias": []interface{}{
							"text",
							map[string]interface{}{},
							map[string]interface{}{"streams": "text"},
							map[string]interface{}{"streams": []interface{}{map[string]interface{}{"local port": float64(30001)}}},
						},
					},
				},
			},
			wantPorts: []int{30001},
		},
		{
			name: "non-map stream, missing port, unsupported port type, out of range are skipped",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"t": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{"streams": []interface{}{
								"text",
								map[string]interface{}{},
								map[string]interface{}{"local port": "30002"},
								map[string]interface{}{"local port": float64(0)},
								map[string]interface{}{"local port": float64(70000)},
								map[string]interface{}{"local port": float64(30003)},
							}},
						},
					},
				},
			},
			wantPorts: []int{30003},
		},
		{
			name: "int and int64 ports are accepted",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"t": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{"streams": []interface{}{
								map[string]interface{}{"local port": int64(30004)},
								map[string]interface{}{"local port": int(30005)},
							}},
						},
					},
				},
			},
			wantPorts: []int{30004, 30005},
		},
		{
			name: "only unusable entries yields an error",
			input: map[string]interface{}{
				"tags": map[string]interface{}{
					"t": map[string]interface{}{
						"medias": []interface{}{
							map[string]interface{}{"streams": []interface{}{map[string]interface{}{"local port": "x"}}},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := extractLocalPorts(tt.input, "")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Wrong error. wantErr: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}
			sort.Ints(res)
			if !reflect.DeepEqual(res, tt.wantPorts) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.wantPorts, res)
			}
		})
	}
}
