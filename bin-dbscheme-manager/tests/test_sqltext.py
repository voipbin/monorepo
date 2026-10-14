import pytest

from classifier.sqltext import SqlError, normalize, split_statements, tokenize


def test_tabs_are_four_spaces_even_inside_quotes():
    assert normalize("ALTER TABLE t ADD COLUMN a VARCHAR(5) COMMENT 'a\tb'") == normalize(
        "ALTER TABLE t ADD COLUMN a VARCHAR(5) COMMENT 'a    b'"
    )


def test_comment_forms_are_equal_after_tab_replacement():
    assert normalize("SELECT 1 --\tx\n") == normalize("SELECT 1 --    x\n")


def test_line_comments_are_stripped_and_whitespace_collapsed():
    rendered = "CREATE TABLE t (\n\ta INTEGER NOT NULL, -- c\n\tb INT\n)\n\n"
    online = "CREATE TABLE t (a INTEGER NOT NULL, b INT)"
    assert normalize(rendered) == normalize(online, online=True)


def test_backticks_removed_only_for_simple_identifiers():
    assert normalize("ALTER TABLE `t` ADD COLUMN `position` INT") == "ALTER TABLE t ADD COLUMN position INT"
    assert normalize("SELECT `a-b`") == "SELECT `a-b`"


def test_spaces_next_to_parentheses_and_commas_are_removed():
    assert normalize("enum( 'a' , 'b' )") == "enum('a','b')"


def test_trailing_semicolons_dropped():
    assert normalize("DROP TABLE x ; ;") == "DROP TABLE x"
    assert normalize("DROP TABLE x;;") == "DROP TABLE x"


def test_percent_unescape_is_online_only():
    assert normalize("DEFAULT '50%'") == normalize("DEFAULT '50%%'", online=True)
    assert normalize("DEFAULT '50%%'") == normalize("DEFAULT '50%%%%'", online=True)
    assert normalize("DEFAULT '50%%'") != normalize("DEFAULT '50%'")


def test_whitespace_inside_quotes_is_kept():
    assert normalize("COMMENT 'a   b'") == "COMMENT 'a   b'"


@pytest.mark.parametrize("ch", ["\x0b", "\x0c", "\u00a0", "\u3000", "\x85", "\x00", "\x01", "\x7f"])
def test_exotic_whitespace_outside_quotes_is_rejected(ch):
    with pytest.raises(SqlError):
        normalize("SELECT" + ch + "1")
    with pytest.raises(SqlError):
        normalize("SELECT 1" + ch)
    with pytest.raises(SqlError):
        normalize(ch + "SELECT 1", online=True)


@pytest.mark.parametrize("text", ["SELECT 1 /* x */", "SELECT 1 /*M! , drop column b */", "SELECT 1 /*!50000 , drop column a */"])
def test_executable_comments_are_rejected_on_both_sides(text):
    with pytest.raises(SqlError):
        normalize(text)
    with pytest.raises(SqlError):
        normalize(text, online=True)


@pytest.mark.parametrize("text", ["SELECT 1 # c\n", "SELECT 1 --x\n", "SELECT 1 --\n", "SELECT 1 ---  x\n"])
def test_other_comment_forms_are_rejected(text):
    with pytest.raises(SqlError):
        tokenize(text)


def test_comment_must_be_closed_by_line_feed():
    with pytest.raises(SqlError):
        tokenize("ALTER TABLE t ADD COLUMN a INT -- c")
    with pytest.raises(SqlError):
        split_statements("ALTER TABLE t ADD COLUMN a INT -- c;")


@pytest.mark.parametrize("ch", ["\r", "\x0b", "\x0c", "\x85", "\u2028", "\U000e0041", "\u202e", "\u200b", "\x00", "\u00a0"])
def test_invisible_characters_in_a_comment_are_rejected(ch):
    with pytest.raises(SqlError):
        tokenize("SELECT 1 -- a" + ch + "b\n")


def test_visible_non_ascii_in_a_comment_is_accepted():
    tokenize("SELECT 1 -- see \u00a7 and \u2014 and \ud55c\uae00\n")


def test_comment_text_with_quotes_and_semicolons_does_not_split():
    stmts = split_statements("SELECT 1 -- it's; \"x\" /* y\nFROM t;\n")
    assert len(stmts) == 1
    assert [t.val for t in stmts[0].tokens] == ["SELECT", "1", "FROM", "t"]


def test_empty_statements_are_ignored():
    assert len(split_statements("ALTER TABLE a ADD COLUMN b INT;;\n\n")) == 1


def test_backslash_in_string_is_rejected():
    with pytest.raises(SqlError):
        tokenize("SELECT 'a\\'b'")


@pytest.mark.parametrize("text", ["SELECT 'a\\\\b'", "SELECT 'a\\\\\\\\'", "SELECT 'x\\\\', DROP"])
def test_backslash_in_a_closed_string_is_rejected(text):
    with pytest.raises(SqlError, match="backslash"):
        tokenize(text)


@pytest.mark.parametrize("text", ["'a'''", "''''", "'a''b''c'"])
def test_doubled_quotes_are_consumed_as_a_pair(text):
    toks = tokenize("SELECT " + text + " , x")
    assert [t.val for t in toks] == ["SELECT", text, ",", "x"]


def test_nul_inside_a_string_is_rejected():
    with pytest.raises(SqlError):
        tokenize("SELECT 'a\x00b'")


def test_doubled_quotes_stay_inside_the_string():
    toks = tokenize("SELECT 'it''s'")
    assert toks[-1].val == "'it''s'"
