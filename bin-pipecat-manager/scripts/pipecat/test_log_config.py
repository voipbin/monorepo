"""Runner log policy (VOIP-1633): default INFO, LOG_LEVEL, library cap, library WARNING+ withheld."""

import json
import os
import subprocess
import sys

import pytest
from loguru import logger

import log_config

SENTENCE = "ZXQ secret sentence ] with bracket\nnext line, language: ko) tail"


@pytest.fixture
def capture(monkeypatch):
    """setup() with a list sink; every handler is removed afterwards."""
    lines = []

    def start(level=None):
        if level is None:
            monkeypatch.delenv("LOG_LEVEL", raising=False)
        else:
            monkeypatch.setenv("LOG_LEVEL", level)
        log_config.setup(sink=lines.append)
        return lines

    yield start
    logger.remove()


def library(name):
    return logger.patch(lambda r: r.update(name=name))


def records(lines):
    return [json.loads(line)["record"] for line in lines]


@pytest.mark.parametrize("raw,want", [
    (None, "INFO"), ("", "INFO"), ("debug", "DEBUG"), (" Warning ", "WARNING"), ("ERROR", "ERROR"),
    ("CRITICAL", "CRITICAL"), ("TRACE", "INFO"), ("SUCCESS", "INFO"), ("verbose", "INFO"),
])
def test_resolve_level(raw, want):
    assert log_config.resolve_level(raw) == want


def test_default_is_info_no_debug(capture):
    lines = capture()
    logger.debug("a debug line")
    logger.info("an info line")
    assert [r["message"] for r in records(lines)] == ["an info line"]


def test_invalid_level_falls_back_to_info_with_warning(capture):
    lines = capture("verbose")
    logger.debug("a debug line")
    msgs = [r["message"] for r in records(lines)]
    assert "a debug line" not in msgs
    assert any("LOG_LEVEL" in m and "verbose" not in m for m in msgs)


def test_debug_enabled_by_env(capture):
    lines = capture("DEBUG")
    logger.debug("runner debug")
    assert [r["message"] for r in records(lines)] == ["runner debug"]


def test_library_debug_is_blocked_even_with_debug(capture):
    lines = capture("DEBUG")
    library("pipecat.services.openai.base_llm").debug("Generating chat from context [" + SENTENCE + "]")
    library("pipecat").debug("exact name debug")
    assert lines == []


def test_library_info_passes_but_is_capped_by_runner_level(capture):
    lines = capture()
    library("pipecat.services.stt_service").info("lib info")
    assert [r["message"] for r in records(lines)] == ["lib info"]
    logger.remove()
    lines.clear()
    lines = capture("WARNING")
    library("pipecat.services.stt_service").info("lib info")
    assert lines == []


@pytest.mark.parametrize("name", [
    "pipecat.services.tts_service", "pipecat.frames.frames", "pipecat.processors.frame_processor", "pipecat",
])
def test_library_warning_and_error_are_withheld(capture, name):
    lines = capture("DEBUG")
    library(name).warning("not speaking [" + SENTENCE + "]")
    try:
        raise ValueError(SENTENCE)
    except ValueError:
        library(name).opt(exception=True).error("TextFrame(text: [" + SENTENCE + "], language: ko)")
    assert len(lines) == 2
    for line in lines:
        assert "ZXQ" not in line
        assert "secret" not in line
    recs = records(lines)
    assert all(r["message"].startswith("library message withheld (len=") for r in recs)
    assert all(r["exception"] is None for r in recs)
    assert recs[0]["level"]["name"] == "WARNING" and recs[1]["level"]["name"] == "ERROR"
    assert recs[1]["name"] == name


@pytest.mark.parametrize("name", ["pipecat.pipeline.worker", "pipecat.transports.websocket.client"])
def test_allowlisted_library_modules_keep_their_message(capture, name):
    lines = capture()
    library(name).error("A fatal error occurred: ErrorFrame(error: boom)")
    assert [r["message"] for r in records(lines)] == ["A fatal error occurred: ErrorFrame(error: boom)"]


@pytest.mark.parametrize("name", ["pipecat_x", "pipecatx.sub", "__main__", "main", "run", None])
def test_non_library_names_are_not_withheld(capture, name):
    lines = capture()
    library(name).warning("runner warning")
    assert [r["message"] for r in records(lines)] == ["runner warning"]


def test_filter_error_withholds_message(capture, monkeypatch):
    lines = capture()

    class Boom(str):
        def startswith(self, *a):
            raise RuntimeError("x")

    library(Boom("pipecat.x")).warning(SENTENCE)
    assert len(lines) == 1
    assert "ZXQ" not in lines[0]
    assert records(lines)[0]["message"] == "log filter error"


_CHILD = """
import asyncio, sys
from unittest.mock import AsyncMock, MagicMock
import log_config
log_config.setup(sys.stdout)
from pipecat.services.openai.llm import OpenAILLMService
from pipecat.processors.aggregators.llm_context import LLMContext
svc = OpenAILLMService(api_key="k", model="gpt-4o")
svc._client = MagicMock()
svc._client.chat.completions.create = AsyncMock(return_value=MagicMock())
ctx = LLMContext(messages=[{"role": "user", "content": "ZXQ real library sentence"}])
asyncio.run(svc.get_chat_completions(ctx))
"""


@pytest.mark.parametrize("level", [None, "DEBUG"])
def test_real_pipecat_llm_service_does_not_print_conversation(level):
    """Runs the real pipecat OpenAILLMService in a child process (conftest mocks pipecat here)."""
    probe = subprocess.run([sys.executable, "-c", "import pipecat.services.openai.llm"],
                           capture_output=True, text=True, cwd=os.path.dirname(__file__))
    if probe.returncode != 0:
        pytest.skip("pipecat-ai is not installed in this environment")
    env = dict(os.environ)
    env.pop("LOG_LEVEL", None)
    if level:
        env["LOG_LEVEL"] = level
    out = subprocess.run([sys.executable, "-c", _CHILD], capture_output=True, text=True,
                         cwd=os.path.dirname(__file__), env=env)
    assert out.returncode == 0, out.stderr[-500:]
    assert "ZXQ" not in out.stdout + out.stderr
    assert "Pipecat" in out.stdout  # the import time banner follows the policy and is emitted


def test_main_sets_up_logging_before_importing_run():
    """main.py must call log_config.setup() before `from run import`: run, pipecat and the other
    modules log at import time, and without the call loguru's default DEBUG sink stays active."""
    import ast

    tree = ast.parse(open(os.path.join(os.path.dirname(__file__), "main.py")).read())
    setup_line = None
    run_import_line = None
    for node in tree.body:
        if (isinstance(node, ast.Expr) and isinstance(node.value, ast.Call)
                and isinstance(node.value.func, ast.Attribute) and node.value.func.attr == "setup"
                and isinstance(node.value.func.value, ast.Name) and node.value.func.value.id == "log_config"):
            setup_line = node.lineno
        if isinstance(node, ast.ImportFrom) and node.module == "run":
            run_import_line = node.lineno
    assert setup_line is not None, "main.py must call log_config.setup() at module level"
    assert run_import_line is not None
    assert setup_line < run_import_line
