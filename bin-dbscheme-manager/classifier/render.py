"""Offline render of one revision step with Alembic (`upgrade <down>:<rev> --sql`)."""

import io
import logging
import os

from alembic import command
from alembic.config import Config

# Offline render and online execution use the same SQLAlchemy dialect, so the
# rendered quoting equals the executed quoting.
RENDER_URL = "mariadb+pymysql://render@localhost/%s"


class NotRenderable(Exception):
    pass


def make_config(root, script_dir, schema, url=None):
    """Config for one stream. `root` holds alembic.ini.sample, `script_dir` the env.py and versions."""
    cfg = Config(os.path.join(root, "alembic.ini.sample"))
    cfg.set_main_option("script_location", os.path.join(root, script_dir))
    cfg.set_main_option("sqlalchemy.url", (url or RENDER_URL % schema).replace("%", "%%"))
    return cfg


def _snapshot_logging():
    root = logging.getLogger()
    return (
        list(root.handlers),
        root.level,
        {name: (lg.disabled, lg.level, list(lg.handlers), lg.propagate) for name, lg in logging.root.manager.loggerDict.items() if isinstance(lg, logging.Logger)},
    )


def _restore_logging(snap):
    handlers, level, loggers = snap
    root = logging.getLogger()
    root.handlers[:] = handlers
    root.setLevel(level)
    for name, (disabled, lvl, hs, prop) in loggers.items():
        lg = logging.getLogger(name)
        lg.disabled = disabled
        lg.setLevel(lvl)
        lg.handlers[:] = hs
        lg.propagate = prop


def render_step(root, script_dir, schema, down, rev):
    """Return the offline SQL text of one step. Raises NotRenderable on any failure.

    The caller runs the source gate first: rendering imports the revision modules.
    """
    cfg = make_config(root, script_dir, schema)
    cfg.output_buffer = io.StringIO()
    snap = _snapshot_logging()
    try:
        command.upgrade(cfg, "%s:%s" % (down or "base", rev), sql=True)
    except Exception as exc:  # any failure to render is "not renderable"
        raise NotRenderable(type(exc).__name__) from None
    finally:
        _restore_logging(snap)
    return cfg.output_buffer.getvalue()
