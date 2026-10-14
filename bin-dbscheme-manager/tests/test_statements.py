import pytest

from classifier.sqltext import SqlError, split_statements
from classifier.statements import ADDITIVE, BOOKKEEPING, DESTRUCTIVE, UNPARSABLE, Context, classify_statement

CTX = Context(schema="bin_manager", from_rev="aaa111", to_rev="bbb222")
EMPTY_DB = Context(schema="bin_manager", from_rev=None, to_rev="bbb222")


def kinds(text, ctx=CTX):
    return [classify_statement(s, ctx).kind for s in split_statements(text)]


def kind(text, ctx=CTX):
    ks = kinds(text, ctx)
    assert len(ks) == 1, ks
    return ks[0]


ADDITIVE_STATEMENTS = [
    "ALTER TABLE t ADD COLUMN a INT",
    "alter table t add column a int null",
    "ALTER TABLE t ADD a VARCHAR(255) NOT NULL DEFAULT ''",
    "ALTER TABLE t ADD COLUMN IF NOT EXISTS a BIGINT UNSIGNED NULL AFTER b",
    "ALTER TABLE `t` ADD COLUMN `position` INT NULL",
    "ALTER TABLE bin_manager.t ADD COLUMN a INT",
    "ALTER TABLE t ADD COLUMN a INTEGER NOT NULL COMMENT 'x y' DEFAULT '0'",
    "ALTER TABLE t ADD COLUMN a BINARY(16) NOT NULL DEFAULT 0x00000000000000000000000000000000",
    "ALTER TABLE t ADD COLUMN a DATETIME(6) NULL DEFAULT CURRENT_TIMESTAMP(6)",
    "ALTER TABLE t ADD COLUMN a DATETIME NULL DEFAULT now()",
    "ALTER TABLE t ADD COLUMN a JSON NOT NULL DEFAULT (JSON_OBJECT())",
    "ALTER TABLE t ADD COLUMN a JSON NOT NULL DEFAULT (JSON_ARRAY())",
    "ALTER TABLE t ADD COLUMN a BINARY(16) NOT NULL DEFAULT (UNHEX('00ff'))",
    "ALTER TABLE t ADD COLUMN a ENUM('x','y') NOT NULL DEFAULT 'x'",
    "ALTER TABLE t ADD COLUMN a DECIMAL(10, 2) NOT NULL DEFAULT -1.5",
    "ALTER TABLE t ADD COLUMN a BOOL NOT NULL DEFAULT true",
    "ALTER TABLE t ADD COLUMN a VARCHAR(5) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL",
    "ALTER TABLE t ADD COLUMN a INT NULL, ADD COLUMN b INT NULL, ADD INDEX idx_ab (a, b(10))",
    "ALTER TABLE t ADD KEY idx_a (a)",
    "ALTER TABLE t ADD COLUMN a VARCHAR(5) NULL DEFAULT '50%'",
    "ALTER TABLE t ADD COLUMN a INT NOT NULL DEFAULT 0",
    "ALTER TABLE t ADD COLUMN a DECIMAL(5,2) NOT NULL DEFAULT 1.5",
    "CREATE TABLE t (a INT NOT NULL) ROW_FORMAT=COMPACT",
    "create table t (a int)",
    "ALTER TABLE t ADD COLUMN a VARCHAR(5) NULL DEFAULT '9999-01-01 00:00:00.000000'",
    "ALTER TABLE t ADD COLUMN a VARCHAR(5) NULL COMMENT 'ratio 1:2' DEFAULT '12:30:00'",
    "ALTER TABLE t ADD COLUMN a VARCHAR(5) NULL COMMENT 'users like this'",
    "CREATE INDEX idx_t_a ON t(a)",
    "CREATE INDEX idx_t_a ON t (a, b(20))",
    "CREATE TABLE t (id BINARY(16) NOT NULL, a VARCHAR(5), PRIMARY KEY (id))",
    "CREATE TABLE IF NOT EXISTS t (id INT NOT NULL AUTO_INCREMENT, a VARCHAR(5) NOT NULL, PRIMARY KEY (id), KEY idx_a (a), UNIQUE KEY uq_a (a)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin",
    "CREATE TABLE t (id INT NOT NULL, CONSTRAINT uq UNIQUE (id), CONSTRAINT pk PRIMARY KEY (id)) ROW_FORMAT=DYNAMIC COMMENT='x'",
    "CREATE TABLE t (id INT NOT NULL, a INT, -- note\n PRIMARY KEY (id))",
]


@pytest.mark.parametrize("sql", ADDITIVE_STATEMENTS)
def test_additive(sql):
    assert kind(sql) == ADDITIVE


DESTRUCTIVE_STATEMENTS = [
    "ALTER TABLE t ADD COLUMN a INT NULL COMMENT 'a''', DROP COLUMN b -- '\n",
    "ALTER TABLE t ADD COLUMN a INT NULL COMMENT 'a''b''c', DROP COLUMN b",
    "ALTER TABLE t ADD COLUMN a INT NULL, DROP COLUMN b",
    "ALTER TABLE t DROP COLUMN b",
    "ALTER TABLE t ADD COLUMN a INT NULL, MODIFY b INT",
    "ALTER TABLE t ADD COLUMN a INT NULL, CHANGE b c INT",
    "ALTER TABLE t ADD COLUMN a INT NULL, RENAME COLUMN b TO c",
    "ALTER TABLE t ADD COLUMN a INT NULL, ENGINE=MyISAM",
    "ALTER TABLE t DROP INDEX i",
    "ALTER TABLE t ALTER COLUMN a SET DEFAULT 1",
    "DROP TABLE t",
    "DROP INDEX i ON t",
    "TRUNCATE TABLE t",
    "DELETE FROM t",
    "UPDATE t SET a = 1",
    "INSERT INTO t (a) VALUES (1)",
    "RENAME TABLE a TO b",
]


@pytest.mark.parametrize("sql", DESTRUCTIVE_STATEMENTS)
def test_destructive(sql):
    assert kind(sql) == DESTRUCTIVE


NOT_ADDITIVE = [
    ("ALTER TABLE t ADD COLUMN a INT NOT NULL, ADD COLUMN b INT DEFAULT 0", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN a INT NOT NULL COMMENT 'default'", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN a INT NOT NULL DEFAULT NULL", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN a INT NULL NOT NULL DEFAULT 1", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN a INT NOT NULL NULL", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN a INT NULL DEFAULT 1 DEFAULT 2", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN a INT NULL FIRST", UNPARSABLE),
    ("CREATE TABLE t AS SELECT 1", UNPARSABLE),
    ("CREATE TABLE t (a int) SELECT 1", UNPARSABLE),
    ("CREATE TABLE t LIKE u", UNPARSABLE),
    ("CREATE OR REPLACE TABLE t (a int)", UNPARSABLE),
    ("CREATE TEMPORARY TABLE t (a int)", UNPARSABLE),
    ("CREATE TABLE t (a int, FOREIGN KEY (a) REFERENCES u (id))", UNPARSABLE),
    ("CREATE TABLE t (a int, CHECK (a > 0))", UNPARSABLE),
    ("CREATE TABLE t (a int, b int GENERATED ALWAYS AS (a))", UNPARSABLE),
    ("CREATE TABLE t (a int) ENGINE=MyISAM", UNPARSABLE),
    ("CREATE TABLE t (a int) ENGINE=CONNECT", UNPARSABLE),
    ("CREATE TABLE t (a int) ROW_FORMAT=COMPRESSED", UNPARSABLE),
    ("CREATE TABLE t (a int) ROW_FORMAT=REDUNDANT", UNPARSABLE),
    ("CREATE TABLE t (a int, b int AFTER a)", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN a BINARY(16) NOT NULL DEFAULT (UNHEX('zz'))", UNPARSABLE),
    ("CREATE TABLE t (a int) PARTITION BY HASH (a)", UNPARSABLE),
    ("CREATE TABLE t (a int, FULLTEXT KEY (a))", UNPARSABLE),
    ("CREATE TABLE t (a int PRIMARY KEY)", UNPARSABLE),
    ("CREATE UNIQUE INDEX i ON t (a)", UNPARSABLE),
    ("CREATE INDEX i ON t (a) USING HASH", UNPARSABLE),
    ("CREATE INDEX i ON t (a) ALGORITHM=COPY", UNPARSABLE),
    ("ALTER TABLE t ADD FULLTEXT INDEX i (a)", UNPARSABLE),
    ("ALTER TABLE t ADD UNIQUE bool (b)", UNPARSABLE),
    ("ALTER TABLE t ADD FULLTEXT text (c)", UNPARSABLE),
    ("ALTER TABLE t ADD CONSTRAINT year UNIQUE (a)", UNPARSABLE),
    ("ALTER TABLE t ADD UNIQUE INDEX i (a)", UNPARSABLE),
    ("ALTER TABLE t ADD PRIMARY KEY (a)", UNPARSABLE),
    ("ALTER TABLE t ADD CONSTRAINT fk FOREIGN KEY (a) REFERENCES u (id)", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN s serial", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN a INT PRIMARY KEY", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN a INT NULL FOO bar", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int key", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int unique", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int NULL AUTO_INCREMENT", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int NULL INVISIBLE", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k datetime NULL ON UPDATE CURRENT_TIMESTAMP", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int NULL REFERENCES u (id)", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int NULL CHECK (k > 0)", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int NULL, ALGORITHM=INPLACE", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int NULL, LOCK=NONE", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int NULL DEFAULT (UUID())", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int NULL DEFAULT (1+1)", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int NULL WITH SYSTEM VERSIONING", UNPARSABLE),
    ("ALTER TABLE t ADD INDEX (a)", UNPARSABLE),
    ("ALTER TABLE t ADD INDEX i (a) USING BTREE", UNPARSABLE),
    ("ALTER TABLE t", UNPARSABLE),
    ("ALTER TABLE t ADD COLUMN k int NULL,", UNPARSABLE),
    ("ALTER TABLE mysql.user ADD COLUMN a INT NULL", UNPARSABLE),
    ("ALTER TABLE `mysql`.`user` ADD COLUMN a INT NULL", UNPARSABLE),
    ("CREATE TABLE other_db.x (a int)", UNPARSABLE),
    ("ALTER TABLE bm . ALEMBIC_VERSION ADD COLUMN a INT NULL", UNPARSABLE),
    ("ALTER TABLE alembic_version ADD COLUMN a INT NULL", UNPARSABLE),
    ("ALTER TABLE bin_manager . ALEMBIC_VERSION ADD COLUMN a INT NULL", UNPARSABLE),
    ("ALTER TABLE `a-b` ADD COLUMN a INT NULL", UNPARSABLE),
    ("CREATE EVENT e ON SCHEDULE EVERY 1 DAY DO SELECT 1", UNPARSABLE),
    ("CREATE TRIGGER x BEFORE INSERT ON t FOR EACH ROW SET NEW.a = 1", UNPARSABLE),
    ("SET GLOBAL x = 1", UNPARSABLE),
    ("GRANT ALL ON *.* TO x", UNPARSABLE),
    ("SELECT 1", UNPARSABLE),
    ("CALL p()", UNPARSABLE),
]


@pytest.mark.parametrize("sql,expected", NOT_ADDITIVE)
def test_not_additive(sql, expected):
    assert kind(sql) == expected


def test_two_statements_in_one_string_are_both_seen():
    assert kinds("ALTER TABLE t ADD COLUMN a INT NULL; DROP TABLE x") == [ADDITIVE, DESTRUCTIVE]


def test_unterminated_and_exotic_text_is_unparsable():
    for text in ["ALTER TABLE t ADD COLUMN a INT NULL COMMENT 'x", "ALTER TABLE t ADD COLUMN a INT NULL /*M! , drop column b */"]:
        with pytest.raises(SqlError):
            kinds(text)


def test_multi_add_statement_is_additive():
    sql = "ALTER TABLE t ADD COLUMN a VARCHAR(5) NOT NULL COMMENT 'c' DEFAULT 'x', ADD COLUMN b BINARY(16) NOT NULL DEFAULT 0x00000000000000000000000000000000, ADD INDEX idx_a (a)"
    assert kind(sql) == ADDITIVE


# bookkeeping -----------------------------------------------------------------


def test_expected_bookkeeping_pair_is_accepted():
    assert kind("UPDATE alembic_version SET version_num='bbb222' WHERE alembic_version.version_num = 'aaa111'") == BOOKKEEPING


def test_bookkeeping_with_wrong_pair_is_not_accepted():
    for sql in [
        "UPDATE alembic_version SET version_num='zzz' WHERE alembic_version.version_num = 'aaa111'",
        "UPDATE alembic_version SET version_num='bbb222' WHERE alembic_version.version_num = 'zzz'",
        "UPDATE alembic_version SET version_num='bbb222' WHERE alembic_version.version_num = 'aaa111' OR 1=1",
        "UPDATE alembic_version SET version_num='bbb222', x=1 WHERE alembic_version.version_num = 'aaa111'",
    ]:
        assert kind(sql) in (UNPARSABLE, DESTRUCTIVE), sql


NOT_ACCEPTED_AS_BOOKKEEPING_OR_ADDITIVE = [
    # (sql, context)
    ("UPDATE alembic_version SET version_num='bbb222' WHERE alembic_version.version_num = 'aaa111' AND 1=1", CTX),
    ("UPDATE alembic_version SET version_num='bbb222' WHERE alembic_version.version_num = 'aaa111' x", CTX),
    ("UPDATE alembic_version SET other='bbb222' WHERE alembic_version.version_num = 'aaa111'", CTX),
    ("INSERT INTO alembic_version (version_num) VALUES ('bbb222'), ('x')", EMPTY_DB),
    ("INSERT INTO alembic_version (version_num) VALUES ('bbb222') x", EMPTY_DB),
    ("INSERT INTO alembic_version (other) VALUES ('bbb222')", EMPTY_DB),
    ("CREATE TABLE alembic_version (version_num VARCHAR(32) NOT NULL, CONSTRAINT alembic_version_pkc PRIMARY KEY (version_num)) ENGINE=MyISAM", EMPTY_DB),
    ("INSERT INTO other_table (version_num) VALUES ('bbb222')", EMPTY_DB),
    ("CREATE INDEX i ON alembic_version (a)", CTX),
    ("COMMIT", CTX),
    ("SHUTDOWN", CTX),
    ("`x` y", CTX),
    ("(DELETE FROM t)", CTX),
    ("CREATE TABLE t (a int) CONNECTION='x'", CTX),
    ("CREATE TABLE t (a int, PRIMARY KEY (a) USING HASH)", CTX),
    ("CREATE TABLE t (a int, PRIMARY KEY (a b c))", CTX),
    ("CREATE TABLE t (a int, CONSTRAINT c PRIMARY KEY (a b c))", CTX),
    ("CREATE TABLE t (a int, KEY i ())", CTX),
    ("CREATE TABLE t (a int) COMMENT=5", CTX),
    ("CREATE TABLE t (a int) ENGINE=InnoDB ENGINE=InnoDB", CTX),
    ("ALTER TABLE t ADD COLUMN a BINARY(16) NOT NULL DEFAULT (UUID(x))", CTX),
    ("ALTER TABLE t ADD COLUMN a DATETIME NULL DEFAULT now", CTX),
    ("ALTER TABLE t ADD COLUMN a INT NULL COMMENT 5", CTX),
    ("ALTER TABLE t ADD COLUMN lock INT NULL", CTX),
    ("ALTER TABLE t ADD COLUMN a INT NULL, ADD COLUMN lock INT NULL", CTX),
    ("ALTER TABLE t ADD INDEX i (a(x))", CTX),
    ("ALTER TABLE t ADD COLUMN a VARCHAR(x) NULL", CTX),
    ("ALTER TABLE t DROP COLUMN b, ADD COLUMN s serial", CTX),
]


@pytest.mark.parametrize("sql,ctx", NOT_ACCEPTED_AS_BOOKKEEPING_OR_ADDITIVE)
def test_trailing_and_malformed_input_is_never_accepted(sql, ctx):
    assert kind(sql, ctx) not in (ADDITIVE, BOOKKEEPING), sql


def test_insert_bookkeeping_only_on_empty_database():
    sql = "INSERT INTO alembic_version (version_num) VALUES ('bbb222')"
    assert kind(sql, EMPTY_DB) == BOOKKEEPING
    assert kind(sql, CTX) == DESTRUCTIVE
    assert kind("INSERT INTO alembic_version (version_num) VALUES ('other')", EMPTY_DB) == UNPARSABLE


def test_update_bookkeeping_not_accepted_on_empty_database():
    assert kind("UPDATE alembic_version SET version_num='bbb222' WHERE alembic_version.version_num = 'aaa111'", EMPTY_DB) == DESTRUCTIVE


def test_alembic_version_create_only_on_empty_database():
    sql = "CREATE TABLE alembic_version (\n\tversion_num VARCHAR(32) NOT NULL, \n\tCONSTRAINT alembic_version_pkc PRIMARY KEY (version_num)\n)"
    assert kind(sql, EMPTY_DB) == BOOKKEEPING
    assert kind(sql, CTX) == UNPARSABLE
    assert kind(sql.replace("32", "64"), EMPTY_DB) == UNPARSABLE
