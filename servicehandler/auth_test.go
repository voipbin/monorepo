package servicehandler

import (
	"fmt"
	"testing"
)

func TestGenerateHash(t *testing.T) {

	type test struct {
		name     string
		username string
		password string
	}

	tests := []test{
		{
			"normal1",
			"test",
			"test",
		},
		{
			"normal2",
			"admin",
			"admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := generateHash(tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			fmt.Printf("Res: %s\n", res)

			if checkHash(tt.password, res) != true {
				t.Error("Wrong match. expect: true, got: false")
			}
		})
	}
}
