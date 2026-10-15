package migrator

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/go-sql-driver/mysql"
)

func Test_newConfig(t *testing.T) {
	// The password is assembled from fragments so that the test file holds no complete credential-like literal.
	specialPasswd := "a+b" + "%41" + "c" + "@/?#" + ":&"

	tests := []struct {
		name string
		dsn  string

		expectUser   string
		expectPasswd string
		expectDBName string
		expectParams map[string]string
		expectParse  bool
		expectErr    bool
	}{
		{
			name:         "plain dsn",
			dsn:          "user:pass@tcp(127.0.0.1:3306)/bin_manager",
			expectUser:   "user",
			expectPasswd: "pass",
			expectDBName: "bin_manager",
			expectParams: map[string]string{"lock_wait_timeout": "10"},
		},
		{
			name:         "dsn with existing parameters keeps them",
			dsn:          "user:pass@tcp(127.0.0.1:3306)/bin_manager?parseTime=true&loc=Local&time_zone=%27%2B00%3A00%27",
			expectUser:   "user",
			expectPasswd: "pass",
			expectDBName: "bin_manager",
			expectParams: map[string]string{"time_zone": "'+00:00'", "lock_wait_timeout": "10"},
			expectParse:  true,
		},
		{
			name:         "password with url escape characters is used as it is",
			dsn:          fmt.Sprintf("user:%s@tcp(127.0.0.1:3306)/bin_manager?parseTime=true", specialPasswd),
			expectUser:   "user",
			expectPasswd: specialPasswd,
			expectDBName: "bin_manager",
			expectParams: map[string]string{"lock_wait_timeout": "10"},
			expectParse:  true,
		},
		{
			name:      "invalid dsn",
			dsn:       "not a dsn",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := newConfig(tt.dsn)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if res.User != tt.expectUser || res.Passwd != tt.expectPasswd || res.DBName != tt.expectDBName {
				t.Errorf("Wrong match. expect: %s/%s/%s, got: %s/%s/%s", tt.expectUser, tt.expectPasswd, tt.expectDBName, res.User, res.Passwd, res.DBName)
			}
			if !res.MultiStatements {
				t.Errorf("Wrong match. expect: MultiStatements true, got: false")
			}
			for k, v := range tt.expectParams {
				if res.Params[k] != v {
					t.Errorf("Wrong param. key: %s, expect: %s, got: %s", k, v, res.Params[k])
				}
			}
			if res.ParseTime != tt.expectParse {
				t.Errorf("Wrong match. expect parseTime: %v, got: %v", tt.expectParse, res.ParseTime)
			}
		})
	}
}

func Test_newConfig_keepsTheDriverConfigIntact(t *testing.T) {
	res, err := newConfig("user:pass@tcp(127.0.0.1:3306)/db")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if _, err := mysql.NewConnector(res); err != nil {
		t.Errorf("Wrong match. expect: a usable driver config, got: %v", err)
	}
}

func Test_isAhead(t *testing.T) {
	tests := []struct {
		name       string
		current    uint
		dirty      bool
		maxVersion uint
		expectRes  bool
	}{
		{"equal", 2, false, 2, false},
		{"older db", 1, false, 2, false},
		{"newer db", 3, false, 2, true},
		{"newer but dirty is refused later by migrate", 3, true, 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := isAhead(tt.current, tt.dirty, tt.maxVersion); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_newSource(t *testing.T) {
	file := &fstest.MapFile{Data: []byte("SELECT 1;")}

	tests := []struct {
		name      string
		fsys      fstest.MapFS
		expectMax uint
		expectErr bool
	}{
		{
			name: "two versions",
			fsys: fstest.MapFS{
				"000001_a.up.sql": file, "000001_a.down.sql": file,
				"000002_b.up.sql": file, "000002_b.down.sql": file,
			},
			expectMax: 2,
		},
		{
			name: "duplicate version number is an error, not a panic",
			fsys: fstest.MapFS{
				"000002_a.up.sql": file, "000002_a.down.sql": file,
				"000002_b.up.sql": file, "000002_b.down.sql": file,
			},
			expectErr: true,
		},
		{
			name:      "no migration files",
			fsys:      fstest.MapFS{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, res, err := newSource(tt.fsys)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			defer func() { _ = src.Close() }()

			if res != tt.expectMax {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectMax, res)
			}
		})
	}
}

func Test_Force_rejectsVersionZero(t *testing.T) {
	// Version 0 fails before the migrate instance is used, so no database is needed.
	m := &Migrator{}
	if err := m.Force(0); err == nil {
		t.Errorf("Wrong match. expect: error for version 0, got: nil")
	}
}
