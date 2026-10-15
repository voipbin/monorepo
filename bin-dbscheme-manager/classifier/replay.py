"""Runtime replay check: only the classified statements may reach the server.

The guard holds the normalized, classified, rendered statements of the current
step. Every statement SQLAlchemy is about to execute must equal the next one of
that list, or be one of the two read-only statements Alembic itself issues.
"""

from sqlalchemy import event
from sqlalchemy.engine import Engine

from .sqltext import SqlError, normalize

STATEMENT_NOT_ALLOWED = "statement_not_allowed"

_TABLE_CHECK = normalize(
    "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = %(table_schema)s AND table_name = %(table_name)s"
)
_VERSION_READ = normalize("SELECT alembic_version.version_num FROM alembic_version")


class StatementNotAllowed(Exception):
    """Raised before execution; the statement is never sent to the server."""

    reason = STATEMENT_NOT_ALLOWED


class ReplayGuard:
    def __init__(self, expected, schema):
        self.expected = list(expected)
        self.schema = schema
        self.position = 0

    @property
    def remaining(self):
        return len(self.expected) - self.position

    def check(self, statement, parameters=None):
        try:
            flat = normalize(statement, online=True)
        except SqlError:
            raise StatementNotAllowed() from None
        if flat == _TABLE_CHECK:
            if dict(parameters or {}) == {"table_schema": self.schema, "table_name": "alembic_version"}:
                return
            raise StatementNotAllowed()
        if flat == _VERSION_READ:
            if not parameters:
                return
            raise StatementNotAllowed()
        if self.position < len(self.expected) and flat == self.expected[self.position]:
            self.position += 1
            return
        raise StatementNotAllowed()

    # SQLAlchemy before_cursor_execute signature
    def __call__(self, conn, cursor, statement, parameters, context, executemany):
        if executemany:
            raise StatementNotAllowed()
        self.check(statement, parameters)

    def install(self):
        event.listen(Engine, "before_cursor_execute", self)

    def remove(self):
        event.remove(Engine, "before_cursor_execute", self)
