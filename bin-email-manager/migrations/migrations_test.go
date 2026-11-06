package migrations

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/migrationhandler/migrationcheck"
)

var createTableRegexp = regexp.MustCompile(`(?m)^CREATE TABLE `)

// Test_noStrayFiles fails on files that exist on disk but are not embedded (for example a
// misspelled extension), because such a file would be neither checked nor applied.
func Test_noStrayFiles(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") {
			continue
		}
		if _, err := fs.Stat(FS, name); err != nil {
			t.Errorf("Wrong match. %s is not embedded, migration files must end with .sql", name)
		}
	}
}

func Test_embeddedMigrations(t *testing.T) {
	// the email table uses the database default table options, so no table option is declared.
	if problems := migrationcheck.Check(FS, migrationcheck.Options{}); len(problems) > 0 {
		t.Errorf("Wrong migration files:\n%s", strings.Join(problems, "\n"))
	}
}

func Test_baselineCreatesOwnedTables(t *testing.T) {
	up, err := fs.ReadFile(FS, "000001_baseline.up.sql")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	tests := []string{"email_emails"}
	for _, table := range tests {
		if !strings.Contains(string(up), "CREATE TABLE IF NOT EXISTS "+table+" (") {
			t.Errorf("Wrong match. the baseline does not create %s", table)
		}
	}
	// statements start at the beginning of a line; the header comment does not count
	if got := len(createTableRegexp.FindAllString(string(up), -1)); got != len(tests) {
		t.Errorf("Wrong match. expect: %d, got: %d", len(tests), got)
	}
}

// Test_configTables checks that the two configs tables are created by their own files, one
// statement per file, and that their down files carry the lossy marker.
func Test_configTables(t *testing.T) {
	tests := []struct {
		file  string
		table string
	}{
		{file: "000002_email_configs_create_table_all", table: "email_configs"},
		{file: "000003_email_config_histories_create_table_all", table: "email_config_histories"},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			up, err := fs.ReadFile(FS, tt.file+".up.sql")
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !strings.HasPrefix(string(up), "CREATE TABLE IF NOT EXISTS "+tt.table+" (") {
				t.Errorf("Wrong match. the up file does not create %s", tt.table)
			}
			if got := len(createTableRegexp.FindAllString(string(up), -1)); got != 1 {
				t.Errorf("Wrong match. expect: 1 statement, got: %d", got)
			}
			if strings.Contains(string(up), "customer") {
				t.Errorf("Wrong match. the configs tables are platform wide and have no customer column")
			}

			down, err := fs.ReadFile(FS, tt.file+".down.sql")
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !strings.HasPrefix(string(down), "-- DOWN-LOSSY: ") || !strings.Contains(string(down), "DROP TABLE IF EXISTS "+tt.table+";") {
				t.Errorf("Wrong match. the down file is wrong: %s", down)
			}
		})
	}
}

// stringFuncRegexp matches the string returning functions whose result carries the collation of
// the system or connection character set. Comparing that result with a literal fails with
// Error 1267 on a database whose default collation differs (MariaDB 12.x uca1400 default).
var stringFuncRegexp = regexp.MustCompile(`(?i)(CAST\(\s*)?\b(JSON_TYPE|JSON_VALUE|JSON_UNQUOTE|JSON_EXTRACT|LOWER|UPPER|CONCAT|SUBSTRING|TRIM)\(`)

// collationDependentFuncs returns the string function calls of sql that are not the direct
// argument of CAST( ... ). A binary cast has no collation, so the comparison is the same on
// every database default.
func collationDependentFuncs(sql string) []string {
	var res []string
	for _, m := range stringFuncRegexp.FindAllStringSubmatch(sql, -1) {
		if m[1] == "" {
			res = append(res, m[2])
		}
	}
	return res
}

func Test_collationDependentFuncs(t *testing.T) {
	tests := []struct {
		name   string
		sql    string
		expect int
	}{
		{"bare json_type", "CHECK (JSON_TYPE(v) <> 'NULL')", 1},
		{"lower case function", "CHECK (json_type(v) = 'x')", 1},
		{"cast to binary", "CHECK (CAST(JSON_TYPE(v) AS BINARY) <> CAST('NULL' AS BINARY))", 0},
		{"cast with space", "CHECK (CAST( JSON_TYPE(v) AS BINARY) <> CAST('NULL' AS BINARY))", 0},
		{"no function", "CHECK (value_type IN ('string','int'))", 0},
		{"two bare", "CHECK (LOWER(a) = 'x' AND UPPER(b) = 'y')", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(collationDependentFuncs(tt.sql)); got != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expect, got)
			}
		})
	}
}

// Test_noCollationDependentCompare fails when an up file calls a string function without the
// binary cast (VOIP-1686). The dynamic guard is the default-collation database in
// scripts/verify-service-migrations.sh.
func Test_noCollationDependentCompare(t *testing.T) {
	files, err := fs.Glob(FS, "*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("Wrong match. expect: files, got: %v, %v", files, err)
	}
	for _, f := range files {
		up, err := fs.ReadFile(FS, f)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if bad := collationDependentFuncs(string(up)); len(bad) > 0 {
			t.Errorf("Wrong match. %s calls %v without CAST(... AS BINARY); the result depends on the database collation", f, bad)
		}
	}
}
