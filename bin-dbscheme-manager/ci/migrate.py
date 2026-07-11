#!/usr/bin/env python3
"""Convert a go-sql-driver/mysql DSN to a SQLAlchemy URL, write alembic.ini,
acquire a MySQL named lock, run alembic upgrade head (dry-run then real),
and release the lock.

Used only by CI against production; never by AI agents (see
bin-dbscheme-manager/CLAUDE.md's alembic-upgrade prohibition -- this script
is the one sanctioned exception, gated behind CircleCI human approval +
main-branch-only filters, not something an agent invokes directly).

See docs/plans/2026-07-11-dbscheme-manager-production-migration-job-design.md
(VOIP-1246) for the full design history and adversarial review findings
this implementation encodes fixes for.
"""
import argparse
import os
import re
import subprocess
import sys

import pymysql

# go-sql-driver/mysql DSN grammar: user:pass@tcp(host:port)/dbname?k=v&...
DSN_RE = re.compile(
    r'^(?P<user>[^:]+):(?P<pass>[^@]*)@tcp\((?P<host>[^:]+):(?P<port>\d+)\)'
    r'/(?P<db>[^?]+)(?:\?(?P<query>.*))?$'
)

# Only forward query params that materially affect the connection's
# security/behavior. Silently dropping the whole query string risks losing
# e.g. tls=... and connecting unencrypted (round-2 design review finding).
FORWARDED_PARAMS = {"tls"}


def parse_dsn(dsn: str) -> dict:
    """Parse a go-sql-driver/mysql DSN into a field dict.

    Called exactly once per invocation -- both the alembic.ini URL and the
    pymysql.connect() kwargs for the lock are derived from this same dict,
    never from a second independent parse (round-2 design review finding:
    a second regex re-parsing the converted URL string had a fatal bug).
    """
    m = DSN_RE.match(dsn)
    if not m:
        print("ERROR: could not parse DSN format (expected "
              "user:pass@tcp(host:port)/db[?params])", file=sys.stderr)
        sys.exit(1)
    fields = m.groupdict()
    query = fields.pop("query") or ""
    params = dict(p.split("=", 1) for p in query.split("&") if "=" in p)
    fields["params"] = {k: v for k, v in params.items() if k in FORWARDED_PARAMS}
    dropped = set(params) - FORWARDED_PARAMS
    if dropped:
        print(f"NOTE: DSN query params not forwarded (not in allowlist): "
              f"{sorted(dropped)}", file=sys.stderr)
    return fields


def sqlalchemy_url(fields: dict) -> str:
    qs = "&".join(f"{k}={v}" for k, v in fields["params"].items())
    base = (f"mysql+pymysql://{fields['user']}:{fields['pass']}@"
            f"{fields['host']}:{fields['port']}/{fields['db']}")
    return f"{base}?{qs}" if qs else base


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--stream", required=True, choices=["bin-manager", "asterisk_config"])
    ap.add_argument("--dsn-env", required=True)
    ap.add_argument("--lock-name", required=True)
    args = ap.parse_args()

    raw_dsn = os.environ.get(args.dsn_env)
    if not raw_dsn:
        print(f"ERROR: {args.dsn_env} not set", file=sys.stderr)
        sys.exit(1)

    fields = parse_dsn(raw_dsn)  # single parse, single source of truth
    url = sqlalchemy_url(fields)  # for alembic.ini

    stream_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", args.stream)
    ini_path = os.path.join(stream_dir, "alembic.ini")
    with open(os.path.join(stream_dir, "alembic.ini.sample")) as f:
        ini_content = f.read()
    with open(ini_path, "w") as f:
        # NOTE: `url` (built from the production password) is passed via a
        # replacement FUNCTION, not a replacement STRING. re.sub() treats a
        # string replacement's backslashes specially (\1, \g<name>, a bare
        # trailing \ raises re.error). Since `url` embeds untrusted secret
        # data, a password containing a literal backslash would crash this
        # script mid-write if the replacement were a plain f-string
        # (round-3 design review finding). A lambda forces the replacement
        # to be inserted literally, with no backslash reinterpretation.
        f.write(re.sub(r"^sqlalchemy\.url.*$",
                        lambda _m: f"sqlalchemy.url = {url}",
                        ini_content, flags=re.MULTILINE))

    # Discrete fields (not a second URL parse) drive the DB connection used
    # only for the named lock -- avoids the round-2 regex-mismatch bug
    # where re-parsing the already-converted URL string could disagree
    # with the first parse.
    conn = pymysql.connect(
        host=fields["host"], port=int(fields["port"]),
        user=fields["user"], password=fields["pass"], database=fields["db"],
    )
    cur = conn.cursor()
    cur.execute("SELECT GET_LOCK(%s, 30)", (args.lock_name,))
    row = cur.fetchone()
    if row is None or row[0] != 1:
        print(f"ERROR: could not acquire lock '{args.lock_name}' "
              f"(another run in progress?)", file=sys.stderr)
        sys.exit(1)
    try:
        for cmd in (
            ["alembic", "-c", ini_path, "current", "--verbose"],
            ["alembic", "-c", ini_path, "upgrade", "head", "--sql"],
            ["alembic", "-c", ini_path, "upgrade", "head"],
            ["alembic", "-c", ini_path, "current", "--verbose"],
        ):
            result = subprocess.run(cmd, cwd=stream_dir, env=os.environ.copy(),
                                     capture_output=True, text=True)
            # Redaction: defense in depth, on top of the primary claim
            # (verified in tests / verification plan) that alembic
            # --verbose output does not print sqlalchemy.url itself.
            out = result.stdout.replace(fields["pass"], "***") if fields["pass"] else result.stdout
            err = result.stderr.replace(fields["pass"], "***") if fields["pass"] else result.stderr
            print(out)
            print(err, file=sys.stderr)
            if result.returncode != 0:
                sys.exit(result.returncode)
    finally:
        cur.execute("SELECT RELEASE_LOCK(%s)", (args.lock_name,))
        conn.close()


if __name__ == "__main__":
    main()
