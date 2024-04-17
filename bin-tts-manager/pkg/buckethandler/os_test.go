package buckethandler

import (
	"context"
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
)

func Test_OSGetFilepath(t *testing.T) {

	type test struct {
		name              string
		osBucketDirectory string

		target    string
		expectRes string
	}

	tests := []test{
		{
			name:              "normal",
			osBucketDirectory: "/shared-data",

			target:    "766e587168455d862b8ef2a931341e7adaa106e1.wav",
			expectRes: "/shared-data/766e587168455d862b8ef2a931341e7adaa106e1.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &bucketHandler{
				osBucketDirectory: tt.osBucketDirectory,
			}
			ctx := context.Background()

			res := h.OSGetFilepath(ctx, tt.target)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_OSGetMediaFilepath(t *testing.T) {

	type test struct {
		name           string
		osLocalAddress string

		target    string
		expectRes string
	}

	tests := []test{
		{
			name:           "normal",
			osLocalAddress: "10-96-0-112.bin-manager.pod.cluster.local",

			target:    "766e587168455d862b8ef2a931341e7adaa106e1.wav",
			expectRes: "http://10-96-0-112.bin-manager.pod.cluster.local/766e587168455d862b8ef2a931341e7adaa106e1.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &bucketHandler{
				osLocalAddress: tt.osLocalAddress,
			}
			ctx := context.Background()

			res := h.OSGetMediaFilepath(ctx, tt.target)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}
