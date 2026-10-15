package migrations

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

var fileNameRegexp = regexp.MustCompile(`^(\d{6})_([a-z0-9_]+)\.(up|down)\.sql$`)

var createIfNotExistsRegexp = regexp.MustCompile(`(?i)^CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s`)

var downLossyRegexp = regexp.MustCompile(`^-- DOWN-LOSSY: \S.*$`)

// splitStatements removes comments and returns the non-empty statements of an SQL file.
//
// Comments are "-- " line comments and block comments. Semicolons inside '...', "..." and
// `...` are ignored; backslash escapes and doubled quote characters are honored. Constructs
// whose meaning this scanner cannot judge are rejected instead of guessed: conditional
// comments (/*! and /*M!), # comments and unterminated strings or comments.
func splitStatements(src string) ([]string, error) {
	var (
		res   []string
		cur   strings.Builder
		quote byte
	)

	flush := func() {
		if stmt := strings.TrimSpace(cur.String()); stmt != "" {
			res = append(res, stmt)
		}
		cur.Reset()
	}

	for i := 0; i < len(src); {
		c := src[i]

		if quote != 0 {
			cur.WriteByte(c)
			switch {
			case c == '\\' && quote != '`' && i+1 < len(src):
				cur.WriteByte(src[i+1])
				i += 2
				continue
			case c == quote && i+1 < len(src) && src[i+1] == quote:
				cur.WriteByte(src[i+1])
				i += 2
				continue
			case c == quote:
				quote = 0
			}
			i++
			continue
		}

		switch {
		case c == '\'' || c == '"' || c == '`':
			quote = c
			cur.WriteByte(c)
			i++
		case c == '#':
			return nil, fmt.Errorf("# comments are not supported")
		case strings.HasPrefix(src[i:], "--") && (i+2 == len(src) || src[i+2] == ' ' || src[i+2] == '\t' || src[i+2] == '\n' || src[i+2] == '\r'):
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case strings.HasPrefix(src[i:], "/*"):
			if i+2 < len(src) && (src[i+2] == '!' || strings.HasPrefix(src[i+2:], "M!")) {
				return nil, fmt.Errorf("conditional comments are not supported")
			}
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("unterminated block comment")
			}
			cur.WriteByte(' ')
			i += 2 + end + 2
		case c == ';':
			flush()
			i++
		default:
			cur.WriteByte(c)
			i++
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("unterminated string")
	}
	flush()

	return res, nil
}

// checkFiles validates the structure of a set of migration files and returns every violation.
func checkFiles(fsys fs.FS) []string {
	var problems []string

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return []string{fmt.Sprintf("could not read the directory: %v", err)}
	}

	up := map[int]string{}
	down := map[int]string{}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") {
			continue
		}

		m := fileNameRegexp.FindStringSubmatch(name)
		if m == nil {
			problems = append(problems, fmt.Sprintf("%s: file name must be NNNNNN_<description>.(up|down).sql", name))
			continue
		}
		number, _ := strconv.Atoi(m[1])

		target := up
		if m[3] == "down" {
			target = down
		}
		if prev, ok := target[number]; ok {
			problems = append(problems, fmt.Sprintf("%s: duplicate %s number %06d (also %s)", name, m[3], number, prev))
			continue
		}
		target[number] = name
	}

	// pairs
	for number, name := range up {
		if _, ok := down[number]; !ok {
			problems = append(problems, fmt.Sprintf("%s: missing the down file", name))
		}
	}
	for number, name := range down {
		if _, ok := up[number]; !ok {
			problems = append(problems, fmt.Sprintf("%s: missing the up file", name))
		}
	}

	// contiguous from 1
	numbers := make([]int, 0, len(up))
	for number := range up {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)
	for i, number := range numbers {
		if number != i+1 {
			problems = append(problems, fmt.Sprintf("version numbers must start at 000001 without gaps, expected %06d, got %06d", i+1, number))
			break
		}
	}
	if len(numbers) == 0 {
		problems = append(problems, "no migration files")
	}

	if name, ok := up[1]; ok && !strings.HasPrefix(name, "000001_baseline.") {
		problems = append(problems, fmt.Sprintf("%s: the first migration must be named 000001_baseline", name))
	}

	// contents
	for number, name := range up {
		problems = append(problems, checkContent(fsys, name, number == 1)...)
	}
	for _, name := range down {
		problems = append(problems, checkContent(fsys, name, false)...)
		problems = append(problems, checkDownLossy(fsys, name)...)
	}

	sort.Strings(problems)
	return problems
}

// checkContent enforces the one statement rule. Only the baseline may hold several statements,
// and only CREATE TABLE IF NOT EXISTS ones, which are safe to run again after a partial failure.
func checkContent(fsys fs.FS, name string, baselineUp bool) []string {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", name, err)}
	}

	stmts, err := splitStatements(string(data))
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", name, err)}
	}

	if baselineUp {
		if len(stmts) == 0 {
			return []string{fmt.Sprintf("%s: no statement", name)}
		}
		var res []string
		for _, stmt := range stmts {
			if !createIfNotExistsRegexp.MatchString(stmt) {
				res = append(res, fmt.Sprintf("%s: the baseline may only contain CREATE TABLE IF NOT EXISTS statements", name))
				break
			}
		}
		return res
	}

	if len(stmts) != 1 {
		return []string{fmt.Sprintf("%s: exactly one statement is allowed per file, got %d", name, len(stmts))}
	}
	return nil
}

// checkDownLossy checks only the format of the marker: "-- DOWN-LOSSY: <reason>".
func checkDownLossy(fsys fs.FS, name string) []string {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.Contains(line, "DOWN-LOSSY") && !downLossyRegexp.MatchString(line) {
			return []string{fmt.Sprintf("%s: the marker must be written as \"-- DOWN-LOSSY: <reason>\"", name)}
		}
	}
	return nil
}

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
	if problems := checkFiles(FS); len(problems) > 0 {
		t.Errorf("Wrong migration files:\n%s", strings.Join(problems, "\n"))
	}
}

func Test_baseline(t *testing.T) {
	data, err := fs.ReadFile(FS, "000001_baseline.up.sql")
	if err != nil {
		t.Fatalf("Wrong match. expect: baseline up file, got: %v", err)
	}
	stmts, err := splitStatements(string(data))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	for _, stmt := range stmts {
		upper := strings.ToUpper(stmt)
		for _, forbidden := range []string{"ENGINE", "CHARSET", "CHARACTER SET", "COLLATE"} {
			if strings.Contains(upper, forbidden) {
				t.Errorf("Wrong match. the baseline must not set table options, found %s in: %s", forbidden, stmt)
			}
		}
	}

	down, err := fs.ReadFile(FS, "000001_baseline.down.sql")
	if err != nil {
		t.Fatalf("Wrong match. expect: baseline down file, got: %v", err)
	}
	downStmts, err := splitStatements(string(down))
	if err != nil || len(downStmts) != 1 || downStmts[0] != "SELECT 1" {
		t.Errorf("Wrong match. the baseline down must be the no-op SELECT 1, got: %v %v", downStmts, err)
	}
}

func Test_splitStatements(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		expect    []string
		expectErr bool
	}{
		{"single", "SELECT 1;", []string{"SELECT 1"}, false},
		{"no trailing semicolon", "SELECT 1", []string{"SELECT 1"}, false},
		{"two", "SELECT 1; SELECT 2;", []string{"SELECT 1", "SELECT 2"}, false},
		{"line comment hides a semicolon", "-- a; b\nSELECT 1;", []string{"SELECT 1"}, false},
		{"block comment hides a semicolon", "/* a; b */ SELECT 1;", []string{"SELECT 1"}, false},
		{"semicolon in single quotes", "SELECT 'a;b';", []string{"SELECT 'a;b'"}, false},
		{"semicolon in double quotes", `SELECT "a;b";`, []string{`SELECT "a;b"`}, false},
		{"semicolon in backticks", "SELECT `a;b`;", []string{"SELECT `a;b`"}, false},
		{"doubled quote escape", "SELECT 'it''s; fine';", []string{"SELECT 'it''s; fine'"}, false},
		{"backslash escape", `SELECT 'it\'s; fine';`, []string{`SELECT 'it\'s; fine'`}, false},
		{"two backslashes then a quote ends the string", `SELECT 'a\\'; SELECT 2;`, []string{`SELECT 'a\\'`, "SELECT 2"}, false},
		{"semicolon in backticks with doubled backtick", "SELECT `a``;b`;", []string{"SELECT `a``;b`"}, false},
		{"quote chars in comments are ignored", "-- it's\nSELECT 1;", []string{"SELECT 1"}, false},
		{"comment marker inside a string", "SELECT '-- not a comment; really';", []string{"SELECT '-- not a comment; really'"}, false},
		{"comments only", "-- nothing\n/* nothing */", nil, false},
		{"double dash without space is not a comment", "SELECT 1--1;", []string{"SELECT 1--1"}, false},
		{"conditional comment is rejected", "/*! SELECT 1 */;", nil, true},
		{"maria conditional comment is rejected", "/*M! SELECT 1 */;", nil, true},
		{"hash comment is rejected", "# a\nSELECT 1;", nil, true},
		{"unterminated string", "SELECT 'a;", nil, true},
		{"unterminated block comment", "/* a", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := splitStatements(tt.src)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if strings.Join(res, "|") != strings.Join(tt.expect, "|") {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, res)
			}
		})
	}
}

func Test_checkFiles(t *testing.T) {
	file := func(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }
	baseline := fstest.MapFS{
		"000001_baseline.up.sql":   file("CREATE TABLE IF NOT EXISTS a (id int);\nCREATE TABLE IF NOT EXISTS b (id int);"),
		"000001_baseline.down.sql": file("SELECT 1;"),
	}
	with := func(extra fstest.MapFS) fstest.MapFS {
		res := fstest.MapFS{}
		for k, v := range baseline {
			res[k] = v
		}
		for k, v := range extra {
			res[k] = v
		}
		return res
	}

	tests := []struct {
		name         string
		fsys         fstest.MapFS
		expectSubstr string
	}{
		{"valid baseline with several create statements", baseline, ""},
		{
			"valid second migration", with(fstest.MapFS{
				"000002_a_add_column_c.up.sql":   file("ALTER TABLE a ADD COLUMN c int NULL;"),
				"000002_a_add_column_c.down.sql": file("-- DOWN-LOSSY: the column data is lost\nALTER TABLE a DROP COLUMN c;"),
			}), "",
		},
		{
			"missing down", fstest.MapFS{"000001_baseline.up.sql": file("CREATE TABLE IF NOT EXISTS a (id int);")},
			"missing the down file",
		},
		{
			"missing up", fstest.MapFS{"000001_baseline.down.sql": file("SELECT 1;")},
			"missing the up file",
		},
		{
			"gap in numbers", with(fstest.MapFS{
				"000003_x.up.sql": file("SELECT 1;"), "000003_x.down.sql": file("SELECT 1;"),
			}), "without gaps",
		},
		{
			"duplicate number", with(fstest.MapFS{
				"000002_x.up.sql": file("SELECT 1;"), "000002_x.down.sql": file("SELECT 1;"),
				"000002_y.up.sql": file("SELECT 1;"),
			}), "duplicate up number",
		},
		{
			"two statements in a later migration", with(fstest.MapFS{
				"000002_x.up.sql": file("ALTER TABLE a ADD COLUMN c int NULL; ALTER TABLE a ADD COLUMN d int NULL;"), "000002_x.down.sql": file("SELECT 1;"),
			}), "exactly one statement",
		},
		{
			"semicolon in a string is one statement", with(fstest.MapFS{
				"000002_x.up.sql": file("UPDATE a SET c = 'x;y';"), "000002_x.down.sql": file("SELECT 1;"),
			}), "",
		},
		{
			"baseline with a non create statement", fstest.MapFS{
				"000001_baseline.up.sql":   file("CREATE TABLE IF NOT EXISTS a (id int); DROP TABLE a;"),
				"000001_baseline.down.sql": file("SELECT 1;"),
			}, "only contain CREATE TABLE IF NOT EXISTS",
		},
		{
			"baseline create without if not exists", fstest.MapFS{
				"000001_baseline.up.sql":   file("CREATE TABLE a (id int);"),
				"000001_baseline.down.sql": file("SELECT 1;"),
			}, "only contain CREATE TABLE IF NOT EXISTS",
		},
		{
			"first migration must be the baseline", fstest.MapFS{
				"000001_create_a.up.sql":   file("CREATE TABLE IF NOT EXISTS a (id int);"),
				"000001_create_a.down.sql": file("SELECT 1;"),
			}, "must be named 000001_baseline",
		},
		{
			"bad file name", with(fstest.MapFS{"2_x.up.sql": file("SELECT 1;")}), "file name must be",
		},
		{
			"conditional comment is rejected", with(fstest.MapFS{
				"000002_x.up.sql": file("/*!40101 ALTER TABLE a ADD COLUMN c int */;"), "000002_x.down.sql": file("SELECT 1;"),
			}), "conditional comments",
		},
		{
			"malformed down lossy marker", with(fstest.MapFS{
				"000002_x.up.sql": file("SELECT 1;"), "000002_x.down.sql": file("-- DOWN-LOSSY\nSELECT 1;"),
			}), "DOWN-LOSSY: <reason>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := strings.Join(checkFiles(tt.fsys), "\n")
			if tt.expectSubstr == "" {
				if problems != "" {
					t.Errorf("Wrong match. expect: no problem, got: %s", problems)
				}
				return
			}
			if !strings.Contains(problems, tt.expectSubstr) {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectSubstr, problems)
			}
		})
	}
}
