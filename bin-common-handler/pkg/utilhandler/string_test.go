package utilhandler

import "testing"

func TestStringGenerateRandom(t *testing.T) {
	tests := []struct {
		name      string
		size      int
		wantErr   bool
		checkFunc func(t *testing.T, result string)
	}{
		{
			name:    "valid size 16",
			size:    16,
			wantErr: false,
			checkFunc: func(t *testing.T, result string) {
				if len(result) == 0 {
					t.Error("StringGenerateRandom() returned empty string")
				}
				if len(result) > 16 {
					t.Errorf("StringGenerateRandom() length = %d, want <= 16", len(result))
				}
			},
		},
		{
			name:    "valid size 32",
			size:    32,
			wantErr: false,
			checkFunc: func(t *testing.T, result string) {
				if len(result) == 0 {
					t.Error("StringGenerateRandom() returned empty string")
				}
				if len(result) > 32 {
					t.Errorf("StringGenerateRandom() length = %d, want <= 32", len(result))
				}
			},
		},
		{
			name:    "zero size",
			size:    0,
			wantErr: true,
		},
		{
			name:    "negative size",
			size:    -5,
			wantErr: true,
		},
		{
			name:    "small size",
			size:    1,
			wantErr: false,
			checkFunc: func(t *testing.T, result string) {
				if len(result) > 1 {
					t.Errorf("StringGenerateRandom() length = %d, want <= 1", len(result))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringGenerateRandom(tt.size)

			if (err != nil) != tt.wantErr {
				t.Errorf("StringGenerateRandom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, got)
			}
		})
	}
}

func TestStringGenerateRandom_Uniqueness(t *testing.T) {
	size := 16
	iterations := 100
	generated := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		result, err := StringGenerateRandom(size)
		if err != nil {
			t.Fatalf("StringGenerateRandom() error = %v", err)
		}

		if generated[result] {
			t.Errorf("StringGenerateRandom() generated duplicate: %s", result)
		}

		generated[result] = true
	}

	if len(generated) != iterations {
		t.Errorf("StringGenerateRandom() uniqueness: got %d unique values, want %d", len(generated), iterations)
	}
}
