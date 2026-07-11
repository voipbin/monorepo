"""Unit tests for bin-dbscheme-manager/ci/migrate.py's pure DSN-handling
functions. Does not touch any real database -- only exercises parse_dsn()
and sqlalchemy_url(), which are the parts covered by the VOIP-1246 design
review's adversarial findings (regex correctness, query-param allowlisting,
backslash-safety of the alembic.ini write path).
"""
import re
import sys
import os
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from migrate import parse_dsn, sqlalchemy_url, run_streamed  # noqa: E402


class TestParseDSN(unittest.TestCase):
    def test_no_query_params(self):
        fields = parse_dsn("root:secret@tcp(10.0.0.1:3306)/bin_manager")
        self.assertEqual(fields["user"], "root")
        self.assertEqual(fields["pass"], "secret")
        self.assertEqual(fields["host"], "10.0.0.1")
        self.assertEqual(fields["port"], "3306")
        self.assertEqual(fields["db"], "bin_manager")
        self.assertEqual(fields["params"], {})

    def test_with_forwarded_and_dropped_params(self):
        fields = parse_dsn(
            "root:secret@tcp(10.0.0.1:3306)/bin_manager?parseTime=true&tls=custom"
        )
        self.assertEqual(fields["db"], "bin_manager")
        # tls is on the allowlist and forwarded; parseTime is Go-specific
        # and dropped (logged to stderr, not silently lost per design doc).
        self.assertEqual(fields["params"], {"tls": "custom"})

    def test_empty_password(self):
        fields = parse_dsn("root:@tcp(10.0.0.1:3306)/bin_manager")
        self.assertEqual(fields["user"], "root")
        self.assertEqual(fields["pass"], "")

    def test_malformed_dsn_exits(self):
        with self.assertRaises(SystemExit):
            parse_dsn("this-is-not-a-valid-dsn")

    def test_password_containing_at_sign_fails_loud_not_silent(self):
        # Design doc's round-3 finding: a password with a literal '@'
        # makes the regex fail to find the mandatory '@tcp(' literal, so
        # parse_dsn must fail LOUD (SystemExit), never silently mis-parse.
        with self.assertRaises(SystemExit):
            parse_dsn("root:sec@ret@tcp(10.0.0.1:3306)/bin_manager")


class TestSQLAlchemyURL(unittest.TestCase):
    def test_no_params(self):
        fields = parse_dsn("root:secret@tcp(10.0.0.1:3306)/bin_manager")
        url = sqlalchemy_url(fields)
        self.assertEqual(url, "mysql+pymysql://root:secret@10.0.0.1:3306/bin_manager")

    def test_with_tls_param(self):
        fields = parse_dsn("root:secret@tcp(10.0.0.1:3306)/bin_manager?tls=custom")
        url = sqlalchemy_url(fields)
        self.assertEqual(
            url,
            "mysql+pymysql://root:secret@10.0.0.1:3306/bin_manager?tls=custom",
        )

    def test_backslash_password_survives_ini_write_via_lambda_replacement(self):
        """Design doc's round-3 finding: re.sub() with a plain string
        replacement interprets backslash escape sequences that look like
        backreferences (\\1..\\99, \\g<name>), which would crash on a
        password containing e.g. a literal '\\1'. migrate.py's actual
        re.sub call uses a lambda to avoid this. This test exercises the
        exact same re.sub pattern migrate.py uses, with such a password,
        to confirm the lambda approach does not raise re.error and
        produces the literal expected text."""
        fields = parse_dsn(r"root:sec\1ret@tcp(10.0.0.1:3306)/bin_manager")
        url = sqlalchemy_url(fields)
        ini_content = "sqlalchemy.url = mysql://root@localhost/bin_manager\n[alembic]\n"

        # Same call shape as migrate.py's main().
        result = re.sub(r"^sqlalchemy\.url.*$",
                         lambda _m: f"sqlalchemy.url = {url}",
                         ini_content, flags=re.MULTILINE)
        self.assertIn(url, result)
        self.assertIn(r"sec\1ret", result)

        # Sanity check: confirm the OLD buggy approach (plain string
        # replacement) really would have raised re.error for this input,
        # proving the lambda fix is not redundant.
        with self.assertRaises(re.error):
            re.sub(r"^sqlalchemy\.url.*$", f"sqlalchemy.url = {url}",
                    ini_content, flags=re.MULTILINE)


class TestRunStreamed(unittest.TestCase):
    """Round-2 PR review finding: the original implementation used
    subprocess.run(capture_output=True), which buffers all output until
    the process exits -- risking a false-positive no_output_timeout on a
    slow-but-healthy migration, and leaving zero diagnostic log trail on
    a real hang. run_streamed() forwards output line-by-line instead."""

    def test_streams_output_and_returns_exit_code(self):
        code = [
            "import sys, time",
            "print('line1'); sys.stdout.flush()",
            "print('line2'); sys.stdout.flush()",
        ]
        rc = run_streamed([sys.executable, "-c", "\n".join(code)], cwd=".", redact="")
        self.assertEqual(rc, 0)

    def test_nonzero_exit_code_propagates(self):
        rc = run_streamed([sys.executable, "-c", "import sys; sys.exit(3)"],
                           cwd=".", redact="")
        self.assertEqual(rc, 3)

    def test_redaction_applied_per_line(self):
        import io
        import contextlib

        code = "print('password is supersecret123 in this line')"
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            rc = run_streamed([sys.executable, "-c", code], cwd=".",
                               redact="supersecret123")
        self.assertEqual(rc, 0)
        self.assertNotIn("supersecret123", buf.getvalue())
        self.assertIn("***", buf.getvalue())


if __name__ == "__main__":
    unittest.main()
