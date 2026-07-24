package address

import "testing"

func Test_DeriveEndpoints(t *testing.T) {
	src := Address{Type: TypeTel, Target: "src"}
	dst := Address{Type: TypeTel, Target: "dst"}

	tests := []struct {
		name      string
		direction string
		wantPeer  Address
		wantLocal Address
	}{
		{
			name:      "incoming: remote is source",
			direction: "incoming",
			wantPeer:  src,
			wantLocal: dst,
		},
		{
			name:      "outgoing: remote is destination",
			direction: "outgoing",
			wantPeer:  dst,
			wantLocal: src,
		},
		{
			name:      "empty direction: zero values, no guess",
			direction: "",
			wantPeer:  Address{},
			wantLocal: Address{},
		},
		{
			name:      "unrecognized direction: zero values, no guess",
			direction: "sideways",
			wantPeer:  Address{},
			wantLocal: Address{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer, local := DeriveEndpoints(tt.direction, src, dst)
			if peer != tt.wantPeer {
				t.Errorf("DeriveEndpoints(%q) peer = %+v, want %+v", tt.direction, peer, tt.wantPeer)
			}
			if local != tt.wantLocal {
				t.Errorf("DeriveEndpoints(%q) local = %+v, want %+v", tt.direction, local, tt.wantLocal)
			}
		})
	}
}
