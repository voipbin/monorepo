"""Runner log policy (VOIP-1633).

Default level is INFO; LOG_LEVEL (process environment) changes it. pipecat library records are
limited to max(level, INFO) so its DEBUG conversation lines never print, and library WARNING and
above records are withheld (length only) unless they come from an allowlisted module, because
several of those lines print frame text or model output.
"""

import os
import sys

from loguru import logger

_LEVELS = frozenset({"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"})
_LIBRARY = "pipecat"
_LIBRARY_ALLOWED = frozenset({"pipecat.pipeline.worker", "pipecat.transports.websocket.client"})


def resolve_level(raw):
    """LOG_LEVEL value to a level name; unset or invalid means INFO."""
    name = (raw or "").strip().upper()
    return name if name in _LEVELS else "INFO"


def _is_library(name):
    return isinstance(name, str) and (name == _LIBRARY or name.startswith(_LIBRARY + "."))


def _make_filter(level_name):
    base = logger.level(level_name).no
    info = logger.level("INFO").no
    warning = logger.level("WARNING").no

    def _filter(record):
        try:
            no = record["level"].no
            if no < base:
                return False
            if _is_library(record["name"]):
                if no < info:
                    return False
                if no >= warning and record["name"] not in _LIBRARY_ALLOWED:
                    record["message"] = f"library message withheld (len={len(record['message'])})"
                    record["exception"] = None
            return True
        except Exception:
            record["message"] = "log filter error"
            record["exception"] = None
            return True

    return _filter


def setup(sink=sys.stdout):
    """Replace every loguru handler with one JSON sink that follows the policy."""
    raw = os.environ.get("LOG_LEVEL")
    level_name = resolve_level(raw)
    logger.remove()
    logger.add(
        sink,
        level="TRACE",
        filter=_make_filter(level_name),
        serialize=True,
        diagnose=False,  # tracebacks must not print local variables (keys, endpoint URLs)
    )
    if raw and raw.strip() and raw.strip().upper() not in _LEVELS:
        logger.warning("LOG_LEVEL is not one of DEBUG, INFO, WARNING, ERROR, CRITICAL; using INFO")
