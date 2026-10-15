// Package migrationcheck validates the structure of the embedded migration files of a service
// that owns its schema. It reads files only and never connects to a database.
package migrationcheck

import (
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The table options a baseline may carry. A service declares which of its tables uses which
// one in Options.TableOptions; the list itself is fixed here so that a service cannot extend it.
const (
	OptionUTF8MB4          = "DEFAULT CHARSET=utf8mb4"
	OptionUTF8MB4UnicodeCI = "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci"
	OptionUTF8MB3          = "DEFAULT CHARSET=utf8mb3"
)

// Options holds the service specific input of Check.
type Options struct {
	// TableOptions maps a table name to the table option string its baseline statement must
	// end with (one of the Option constants). A table that is not listed must have no table option.
	// The migration directory is the root of the fs.FS passed to Check (use fs.Sub for a subdirectory).
	TableOptions map[string]string
}

// Check validates the structure of the migration files at the root of fsys and returns every
// violation, sorted. An empty result means the files are valid.
func Check(fsys fs.FS, opt Options) []string {
	return checkFiles(fsys, opt)
}

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
func checkFiles(fsys fs.FS, opt Options) []string {
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
		problems = append(problems, checkContent(fsys, name, number == 1, opt)...)
	}
	for number, name := range down {
		problems = append(problems, checkContent(fsys, name, false, opt)...)
		problems = append(problems, checkDownLossy(fsys, name)...)
		if number == 1 {
			problems = append(problems, checkBaselineDown(fsys, name)...)
		}
	}

	sort.Strings(problems)
	return problems
}

// checkContent enforces the one statement rule. Only the baseline may hold several statements,
// and only CREATE TABLE IF NOT EXISTS ones, which are safe to run again after a partial failure.
func checkContent(fsys fs.FS, name string, baselineUp bool, opt Options) []string {
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
		if len(res) > 0 {
			return res
		}
		return checkBaselineTables(name, stmts, opt)
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

var createTableHeadRegexp = regexp.MustCompile(`(?i)^CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+([A-Za-z0-9_]+)\s*\(`)

var wordRegexp = regexp.MustCompile(`[A-Za-z0-9_]+`)

// forbiddenWords must not appear in a baseline table definition (outside quotes). Table options
// are allowed only through the exact tail comparison of Options.TableOptions. This is a deny
// list: the final guard is the normalized dump comparison of scripts/verify-service-migrations.sh.
var forbiddenWords = map[string]bool{
	"ENGINE": true, "ENGINE_ATTRIBUTE": true, "SECONDARY_ENGINE_ATTRIBUTE": true, "ROW_FORMAT": true,
	"CHARSET": true, "CHARACTER": true, "COLLATE": true, "ASCII": true, "UNICODE": true,
	"NATIONAL": true, "NCHAR": true, "NVARCHAR": true,
}

func normalizeSpace(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), " "))
}

// matchingParen returns the index of the ')' that closes the '(' at open, ignoring quoted text.
func matchingParen(s string, open int) int {
	depth := 0
	var quote byte
	for i := open; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			switch {
			case c == '\\' && quote != '`' && i+1 < len(s):
				i++
			case c == quote:
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// stripQuoted replaces quoted text with a space.
func stripQuoted(s string) string {
	var b strings.Builder
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			switch {
			case c == '\\' && quote != '`' && i+1 < len(s):
				i++
			case c == quote:
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			quote = c
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// checkBaselineTables checks the table definitions of the baseline against Options.
func checkBaselineTables(name string, stmts []string, opt Options) []string {
	var res []string
	seen := map[string]bool{}

	for _, stmt := range stmts {
		m := createTableHeadRegexp.FindStringSubmatchIndex(stmt)
		if m == nil {
			res = append(res, fmt.Sprintf("%s: a baseline statement must start with CREATE TABLE IF NOT EXISTS <name> (: %.60s", name, stmt))
			continue
		}
		table := stmt[m[2]:m[3]]
		seen[table] = true

		end := matchingParen(stmt, m[1]-1)
		if end < 0 {
			res = append(res, fmt.Sprintf("%s: %s: unbalanced parentheses", name, table))
			continue
		}

		// the definition between the parentheses
		words := wordRegexp.FindAllString(stripQuoted(stmt[m[1]:end]), -1)
		for i, w := range words {
			upper := strings.ToUpper(w)
			if forbiddenWords[upper] || (upper == "CHAR" && i+1 < len(words) && strings.EqualFold(words[i+1], "SET")) {
				res = append(res, fmt.Sprintf("%s: %s: %s is not allowed inside the table definition", name, table, upper))
			}
		}

		// the table options after the closing parenthesis must be exactly the declared one
		expect := normalizeSpace(opt.TableOptions[table])
		if got := normalizeSpace(stmt[end+1:]); got != expect {
			res = append(res, fmt.Sprintf("%s: %s: table options are %q, expected %q", name, table, got, expect))
		}
	}

	allowed := map[string]bool{OptionUTF8MB4: true, OptionUTF8MB4UnicodeCI: true, OptionUTF8MB3: true}
	tables := make([]string, 0, len(opt.TableOptions))
	for table := range opt.TableOptions {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		if !allowed[opt.TableOptions[table]] {
			res = append(res, fmt.Sprintf("%s: %s: the table option %q is not one of the allowed options", name, table, opt.TableOptions[table]))
		}
		if !seen[table] {
			res = append(res, fmt.Sprintf("%s: %s: a table option is declared for a table the baseline does not create", name, table))
		}
	}

	return res
}

// checkBaselineDown requires the baseline down to be the no-op SELECT 1: dropping the owned
// tables would destroy data.
func checkBaselineDown(fsys fs.FS, name string) []string {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil
	}
	stmts, err := splitStatements(string(data))
	if err != nil || len(stmts) != 1 || stmts[0] != "SELECT 1" {
		return []string{fmt.Sprintf("%s: the baseline down must be the no-op SELECT 1", name)}
	}
	return nil
}
