package migrationcheck

import (
	"strings"
	"testing"
	"testing/fstest"
)

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
			problems := strings.Join(checkFiles(tt.fsys, Options{}), "\n")
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

func Test_Check_baselineTableOptions(t *testing.T) {
	file := func(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }
	baseline := func(up string) fstest.MapFS {
		return fstest.MapFS{"000001_baseline.up.sql": file(up), "000001_baseline.down.sql": file("SELECT 1;")}
	}

	tests := []struct {
		name         string
		up           string
		opt          Options
		expectSubstr string
	}{
		{"no option, none declared", "CREATE TABLE IF NOT EXISTS a (id binary(16) NOT NULL, PRIMARY KEY (id));", Options{}, ""},
		{"space before the parenthesis is optional", "CREATE TABLE IF NOT EXISTS a(id int);", Options{}, ""},
		{"utf8mb4 declared", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET=utf8mb4;", Options{TableOptions: map[string]string{"a": OptionUTF8MB4}}, ""},
		{"unicode_ci declared", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;", Options{TableOptions: map[string]string{"a": OptionUTF8MB4UnicodeCI}}, ""},
		{"unicode_ci with row format declared", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;", Options{TableOptions: map[string]string{"a": OptionUTF8MB4UnicodeCIDynamic}}, ""},
		{"row format in the tail but not declared", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;", Options{TableOptions: map[string]string{"a": OptionUTF8MB4UnicodeCI}}, "table options are"},
		{"row format alone in the tail", "CREATE TABLE IF NOT EXISTS a (id int) ROW_FORMAT=DYNAMIC;", Options{}, "table options are"},
		{"row format declared but not used", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;", Options{TableOptions: map[string]string{"a": OptionUTF8MB4UnicodeCIDynamic}}, "table options are"},
		{"utf8mb3 declared", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET=utf8mb3;", Options{TableOptions: map[string]string{"a": OptionUTF8MB3}}, ""},
		{"lower case and extra white space in the tail", "CREATE TABLE IF NOT EXISTS a (id int)\n  default  charset=UTF8MB4;", Options{TableOptions: map[string]string{"a": OptionUTF8MB4}}, ""},
		{"option used but not declared", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET=utf8mb4;", Options{}, "table options are"},
		{"declared but not used", "CREATE TABLE IF NOT EXISTS a (id int);", Options{TableOptions: map[string]string{"a": OptionUTF8MB4}}, "table options are"},
		{"declared for another string", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET=utf8mb3;", Options{TableOptions: map[string]string{"a": OptionUTF8MB4}}, "table options are"},
		{"space around equals is rejected", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET = utf8mb4;", Options{TableOptions: map[string]string{"a": OptionUTF8MB4}}, "table options are"},
		{"declared string outside the allowed list", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET=latin1;", Options{TableOptions: map[string]string{"a": "DEFAULT CHARSET=latin1"}}, "not one of the allowed options"},
		{"declared for a missing table", "CREATE TABLE IF NOT EXISTS a (id int);", Options{TableOptions: map[string]string{"b": OptionUTF8MB4}}, "does not create"},
		{"table name keys are case sensitive", "CREATE TABLE IF NOT EXISTS a (id int) DEFAULT CHARSET=utf8mb4;", Options{TableOptions: map[string]string{"A": OptionUTF8MB4}}, "does not create"},
		{"engine select tail", "CREATE TABLE IF NOT EXISTS a (id int) ENGINE=MyISAM SELECT concat('a');", Options{}, "table options are"},
		{"hidden engine and union before a declared tail", "CREATE TABLE IF NOT EXISTS a (id int) ENGINE=MERGE UNION=(b) DEFAULT CHARSET=utf8mb4;", Options{TableOptions: map[string]string{"a": OptionUTF8MB4}}, "table options are"},
		{"like form", "CREATE TABLE IF NOT EXISTS a LIKE b;", Options{}, "must start with CREATE TABLE IF NOT EXISTS"},
		{"quoted table name", "CREATE TABLE IF NOT EXISTS `a` (id int);", Options{}, "must start with CREATE TABLE IF NOT EXISTS"},
		{"database prefix", "CREATE TABLE IF NOT EXISTS db.a (id int);", Options{}, "must start with CREATE TABLE IF NOT EXISTS"},
		{"unbalanced parentheses", "CREATE TABLE IF NOT EXISTS a (id int;", Options{}, "unbalanced"},
		{"parenthesis in a comment string", "CREATE TABLE IF NOT EXISTS a (id int COMMENT 'x)', b int);", Options{}, ""},
		{"parenthesis in a column comment", "CREATE TABLE IF NOT EXISTS a (id int COMMENT 'a)b');", Options{}, ""},
		{"check constraint and generated column", "CREATE TABLE IF NOT EXISTS a (j json, g int AS (length(j)) STORED, CHECK (json_valid(j)));", Options{}, ""},
		{"named foreign key", "CREATE TABLE IF NOT EXISTS a (id int, b int, CONSTRAINT fk FOREIGN KEY (b) REFERENCES c (id));", Options{}, ""},
		{"column named engine_model", "CREATE TABLE IF NOT EXISTS a (engine_model varchar(255), ai_engine_type varchar(32));", Options{}, ""},
		{"charset word in a comment string", "CREATE TABLE IF NOT EXISTS a (id int COMMENT 'charset');", Options{}, ""},
		{"backtick quoted column named collate", "CREATE TABLE IF NOT EXISTS a (`collate` int);", Options{}, ""},
		{"lower case create table", "create table if not exists a (id int);", Options{}, ""},
		{"engine in the definition", "CREATE TABLE IF NOT EXISTS a (id int) ENGINE=InnoDB;", Options{}, "table options are"},
		{"engine word as a column option", "CREATE TABLE IF NOT EXISTS a (c int ENGINE=InnoDB);", Options{}, "ENGINE is not allowed"},
		{"charset word as a column option", "CREATE TABLE IF NOT EXISTS a (c text CHARSET utf8mb4);", Options{}, "CHARSET is not allowed"},
		{"lower case char set", "CREATE TABLE IF NOT EXISTS a (c text char set utf8mb4);", Options{}, "CHAR is not allowed"},
		{"secondary engine attribute", "CREATE TABLE IF NOT EXISTS a (c int SECONDARY_ENGINE_ATTRIBUTE='{}');", Options{}, "SECONDARY_ENGINE_ATTRIBUTE is not allowed"},
		{"unicode attribute", "CREATE TABLE IF NOT EXISTS a (c varchar(10) UNICODE);", Options{}, "UNICODE is not allowed"},
		{"national type", "CREATE TABLE IF NOT EXISTS a (c NATIONAL VARCHAR(10));", Options{}, "NATIONAL is not allowed"},
		{"nvarchar type", "CREATE TABLE IF NOT EXISTS a (c NVARCHAR(10));", Options{}, "NVARCHAR is not allowed"},
		{"column character set", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4);", Options{}, "CHARACTER is not allowed"},
		{"column char set synonym", "CREATE TABLE IF NOT EXISTS a (c text CHAR SET utf8mb4);", Options{}, "CHAR is not allowed"},
		{"column collate on a new line", "CREATE TABLE IF NOT EXISTS a (c varchar(10)\nCOLLATE utf8mb4_bin);", Options{}, "COLLATE is not allowed"},
		{"column ascii attribute", "CREATE TABLE IF NOT EXISTS a (c varchar(10) ASCII);", Options{}, "ASCII is not allowed"},
		{"column nchar type", "CREATE TABLE IF NOT EXISTS a (c NCHAR(10));", Options{}, "NCHAR is not allowed"},
		{"engine attribute in the definition", "CREATE TABLE IF NOT EXISTS a (c int ENGINE_ATTRIBUTE='{}');", Options{}, "ENGINE_ATTRIBUTE is not allowed"},
		{"row format in the definition", "CREATE TABLE IF NOT EXISTS a (c int ROW_FORMAT=DYNAMIC);", Options{}, "ROW_FORMAT is not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := strings.Join(Check(baseline(tt.up), tt.opt), "\n")
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

func Test_Check_noMigrationFiles(t *testing.T) {
	if problems := strings.Join(Check(fstest.MapFS{}, Options{}), "\n"); !strings.Contains(problems, "no migration files") {
		t.Errorf("Wrong match. expect: no migration files, got: %q", problems)
	}
}

func Test_Check_baselineDownMustBeNoop(t *testing.T) {
	file := func(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }
	fsys := fstest.MapFS{
		"000001_baseline.up.sql":   file("CREATE TABLE IF NOT EXISTS a (id int);"),
		"000001_baseline.down.sql": file("DROP TABLE a;"),
	}
	if problems := strings.Join(Check(fsys, Options{}), "\n"); !strings.Contains(problems, "no-op SELECT 1") {
		t.Errorf("Wrong match. expect: no-op SELECT 1 violation, got: %q", problems)
	}
}

func Test_Check_columnCharsets(t *testing.T) {
	file := func(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }
	baseline := func(up string) fstest.MapFS {
		return fstest.MapFS{"000001_baseline.up.sql": file(up), "000001_baseline.down.sql": file("SELECT 1;")}
	}
	decl := func(keys ...string) Options {
		res := Options{ColumnCharsets: map[string]string{}}
		for _, key := range keys {
			res.ColumnCharsets[key] = ColumnCharsetUTF8MB4GeneralCI
		}
		return res
	}

	tests := []struct {
		name         string
		up           string
		opt          Options
		expectSubstr string
	}{
		{"declared column", "CREATE TABLE IF NOT EXISTS a (id int, c longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL);", decl("a.c"), ""},
		{"declared column with backticks", "CREATE TABLE IF NOT EXISTS a (`id` int, `c` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", decl("a.c"), ""},
		{"declared column, lower case and extra white space", "CREATE TABLE IF NOT EXISTS a (c text character  set\nUTF8MB4 collate utf8mb4_general_ci);", decl("a.c"), ""},
		{"declared column followed by other columns", "CREATE TABLE IF NOT EXISTS a (c varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci, d int, e text);", decl("a.c"), ""},
		{"declared column with a comma in the type", "CREATE TABLE IF NOT EXISTS a (n decimal(10,2), c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", decl("a.c"), ""},
		{"declared column named key", "CREATE TABLE IF NOT EXISTS a (`key` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", decl("a.key"), ""},
		{"two declared columns in two tables", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci); CREATE TABLE IF NOT EXISTS b (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", decl("a.c", "b.c"), ""},
		{"no declaration, no column charset", "CREATE TABLE IF NOT EXISTS a (c text);", Options{ColumnCharsets: map[string]string{}}, ""},

		{"column charset not declared", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", Options{}, "CHARACTER is not allowed"},
		{"another column is declared, this one is not", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci, d text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", decl("a.c"), "CHARACTER is not allowed"},
		{"same column name in another table is not declared", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci); CREATE TABLE IF NOT EXISTS b (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", decl("a.c"), "CHARACTER is not allowed"},
		{"other collation on a declared column", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_bin);", decl("a.c"), "does not contain"},
		{"other charset on a declared column", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET latin1 COLLATE latin1_swedish_ci);", decl("a.c"), "does not contain"},
		{"declared column without any charset", "CREATE TABLE IF NOT EXISTS a (c text);", decl("a.c"), "does not contain"},
		{"declared string only in a comment", "CREATE TABLE IF NOT EXISTS a (c text COMMENT 'CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci');", decl("a.c"), "does not contain"},
		{"declared string only in a default", "CREATE TABLE IF NOT EXISTS a (c text DEFAULT 'CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci');", decl("a.c"), "does not contain"},
		{"declared string only inside parentheses", "CREATE TABLE IF NOT EXISTS a (c text GENERATED ALWAYS AS (cast(d as char CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci)) STORED);", decl("a.c"), "does not contain"},
		{"declared string twice in one column", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", decl("a.c"), "CHARACTER is not allowed"},
		{"extra collate after the declared string", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COLLATE utf8mb4_bin);", decl("a.c"), "COLLATE is not allowed"},
		{"partial word is no match", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci_x);", decl("a.c"), "does not contain"},
		{"collate before character set is no match", "CREATE TABLE IF NOT EXISTS a (c text COLLATE utf8mb4_general_ci CHARACTER SET utf8mb4);", decl("a.c"), "does not contain"},
		{"declared string in a key item", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci, KEY k (c) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", decl("a.c"), "CHARACTER is not allowed"},
		{"declared string in the next column", "CREATE TABLE IF NOT EXISTS a (c text, d text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", decl("a.c"), "does not contain"},
		{"table created twice, second without the string", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci); CREATE TABLE IF NOT EXISTS a (c text);", decl("a.c"), "does not contain"},

		{"declared column does not exist", "CREATE TABLE IF NOT EXISTS a (c text);", decl("a.d"), "column the table does not have"},
		{"declared column that is a key item", "CREATE TABLE IF NOT EXISTS a (c int, KEY k (c));", decl("a.k"), "column the table does not have"},
		{"declared column names are case sensitive", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci);", decl("a.C"), "column the table does not have"},
		{"declared table does not exist", "CREATE TABLE IF NOT EXISTS a (c text);", decl("b.c"), "table the baseline does not create"},
		{"key without a dot", "CREATE TABLE IF NOT EXISTS a (c text);", decl("c"), "must be written as table.column"},
		{"key with an empty column", "CREATE TABLE IF NOT EXISTS a (c text);", decl("a."), "must be written as table.column"},
		{"key with an empty table", "CREATE TABLE IF NOT EXISTS a (c text);", decl(".c"), "must be written as table.column"},
		{"value outside the allowed list", "CREATE TABLE IF NOT EXISTS a (c text CHARACTER SET utf8mb4 COLLATE utf8mb4_bin);", Options{ColumnCharsets: map[string]string{"a.c": "CHARACTER SET utf8mb4 COLLATE utf8mb4_bin"}}, "not one of the allowed column charsets"},
		{"empty value", "CREATE TABLE IF NOT EXISTS a (c text);", Options{ColumnCharsets: map[string]string{"a.c": ""}}, "not one of the allowed column charsets"},
		{"unbalanced table reports no missing column", "CREATE TABLE IF NOT EXISTS a (c text;", decl("a.c"), "unbalanced"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := strings.Join(Check(baseline(tt.up), tt.opt), "\n")
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

func Test_Check_unbalancedTableReportsOnlyUnbalanced(t *testing.T) {
	fsys := fstest.MapFS{
		"000001_baseline.up.sql":   &fstest.MapFile{Data: []byte("CREATE TABLE IF NOT EXISTS a (c text;")},
		"000001_baseline.down.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}
	problems := Check(fsys, Options{ColumnCharsets: map[string]string{"a.c": ColumnCharsetUTF8MB4GeneralCI}})
	if len(problems) != 1 || !strings.Contains(problems[0], "unbalanced") {
		t.Errorf("Wrong match. expect: only the unbalanced problem, got: %q", problems)
	}
}

func Test_splitTopLevel(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		expect []string
	}{
		{"plain", "a int, b int", []string{"a int", " b int"}},
		{"decimal", "a decimal(10,2), b int", []string{"a decimal(10,2)", " b int"}},
		{"enum string with a comma", "a enum('x,y','z'), b int", []string{"a enum('x,y','z')", " b int"}},
		{"comment with a comma", "a int COMMENT 'p,q', b int", []string{"a int COMMENT 'p,q'", " b int"}},
		{"key columns", "KEY k (a,b), c int", []string{"KEY k (a,b)", " c int"}},
		{"check with in list", "CHECK (a IN (1,2)), c int", []string{"CHECK (a IN (1,2))", " c int"}},
		{"backtick name with a comma", "`a,b` int, c int", []string{"`a,b` int", " c int"}},
		{"doubled quote and backslash escape", `a int COMMENT 'it''s, ok', b int COMMENT 'x\', y', c int`, []string{`a int COMMENT 'it''s, ok'`, ` b int COMMENT 'x\', y'`, " c int"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := splitTopLevel(tt.src); strings.Join(res, "|") != strings.Join(tt.expect, "|") {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, res)
			}
		})
	}
}

func Test_columnName(t *testing.T) {
	tests := []struct {
		name   string
		item   string
		expect string
	}{
		{"plain", " id int", "id"},
		{"backticks", " `id` int", "id"},
		{"backtick key", "`key` int", "key"},
		{"primary key", "PRIMARY KEY (id)", ""},
		{"lower case key", "key k (id)", ""},
		{"unique key", "UNIQUE KEY k (id)", ""},
		{"constraint", "CONSTRAINT fk FOREIGN KEY (b) REFERENCES c (id)", ""},
		{"check", "CHECK (a > 0)", ""},
		{"empty", "  ", ""},
		{"unterminated backtick", "`id int", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := columnName(tt.item); res != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, res)
			}
		})
	}
}
