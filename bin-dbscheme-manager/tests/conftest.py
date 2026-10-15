import os
import shutil
import sys

import pytest

SERVICE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if SERVICE_DIR not in sys.path:
    sys.path.insert(0, SERVICE_DIR)

from classifier.classify import Stream  # noqa: E402

TEMPLATE_STREAM = os.path.join(SERVICE_DIR, "bin-manager")


def make_revision(rev, down, body="pass", extra="", branch_labels="None", depends_on="None", imports="from alembic import op\nimport sqlalchemy as sa\n"):
    """Source text of a revision file in the repository template shape."""
    return (
        '"""t\n\nRevision ID: %s\n"""\n%s\n%s\nrevision = %r\ndown_revision = %r\nbranch_labels = %s\ndepends_on = %s\n\n\n'
        "def upgrade():\n%s\n\n\ndef downgrade():\n    pass\n"
        % (rev, imports, extra, rev, down, branch_labels, depends_on, _indent(body))
    )


def _indent(body):
    return "\n".join("    " + line if line.strip() else line for line in body.splitlines()) or "    pass"


@pytest.fixture
def make_stream(tmp_path):
    """Build a throwaway stream (own env.py and ini sample) from {file name: source}."""

    count = []

    def build(files, schema="bin_manager"):
        count.append(1)
        root = tmp_path / ("stream%d" % len(count))
        script = root / "main"
        versions = script / "versions"
        versions.mkdir(parents=True)
        shutil.copy(os.path.join(TEMPLATE_STREAM, "alembic.ini.sample"), root / "alembic.ini.sample")
        shutil.copy(os.path.join(TEMPLATE_STREAM, "main", "env.py"), script / "env.py")
        shutil.copy(os.path.join(TEMPLATE_STREAM, "main", "script.py.mako"), script / "script.py.mako")
        for name, src in files.items():
            data = src if isinstance(src, bytes) else src.encode("utf-8")
            (versions / name).write_bytes(data)
        return Stream(schema, str(root), "main")

    return build


# --- scratch database server --------------------------------------------------


def _server_url():
    from sqlalchemy.engine import make_url

    raw = os.environ.get("DBSCHEME_TEST_DB_URL")
    if not raw:
        return None
    return make_url(raw)


@pytest.fixture(scope="session")
def db_server():
    """Admin URL of a scratch MariaDB server. CI sets DBSCHEME_TEST_REQUIRE_DB so a missing server fails."""
    import time

    import pymysql

    url = _server_url()
    if url is None:
        if os.environ.get("DBSCHEME_TEST_REQUIRE_DB"):
            pytest.fail("DBSCHEME_TEST_DB_URL is required but not set")
        pytest.skip("DBSCHEME_TEST_DB_URL not set")
    deadline = time.time() + 60
    while True:
        try:
            pymysql.connect(host=url.host, port=url.port or 3306, user=url.username, password=url.password or "").close()
            break
        except pymysql.err.OperationalError:
            if time.time() > deadline:
                raise
            time.sleep(1)
    return url


@pytest.fixture
def fresh_db(db_server):
    """Create an empty database; yields (name, url). Dropped afterwards."""
    import uuid

    import pymysql

    created = []

    def connect():
        return pymysql.connect(
            host=db_server.host, port=db_server.port or 3306, user=db_server.username, password=db_server.password or "", autocommit=True
        )

    def make():
        name = "dbscheme_t_" + uuid.uuid4().hex[:12]
        c = connect()
        c.cursor().execute("CREATE DATABASE `%s`" % name)
        c.close()
        created.append(name)
        url = db_server.set(drivername="mariadb+pymysql", database=name, query={"charset": "utf8mb4"})
        return name, url.render_as_string(hide_password=False)

    yield make
    c = connect()
    for name in created:
        c.cursor().execute("DROP DATABASE IF EXISTS `%s`" % name)
    c.close()
