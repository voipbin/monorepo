"""Tool and team flow log lines carry lengths and identifiers, never tool arguments or response bodies (VOIP-1633)."""

import importlib.util
import json
import os
import sys
from types import SimpleNamespace
from unittest.mock import MagicMock

import pytest
from loguru import logger

SENTENCE = "ZXQ secret sentence from the caller"


class _Resp:
    def __init__(self, status, body, content_type="application/json"):
        self.status = status
        self.headers = {"Content-Type": content_type}
        self._body = body

    async def text(self):
        return self._body

    async def __aenter__(self):
        return self

    async def __aexit__(self, *exc):
        return False


class _Session:
    closed = False

    def __init__(self, resp):
        self._resp = resp
        self.posted = []

    def post(self, url, **kwargs):
        self.posted.append((url, kwargs))
        return self._resp

    async def __aenter__(self):
        return self

    async def __aexit__(self, *exc):
        return False


def _fake_aiohttp(resp):
    session = _Session(resp)
    ns = SimpleNamespace(
        ClientSession=lambda *a, **k: session,
        ClientTimeout=lambda **k: None,
        ClientError=type("ClientError", (Exception,), {}),
    )
    return ns, session


def _load(name, file_name, monkeypatch):
    monkeypatch.setitem(sys.modules, "pipecat.services.llm_service", MagicMock())
    path = os.path.join(os.path.dirname(__file__), file_name)
    spec = importlib.util.spec_from_file_location(name, path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


@pytest.fixture
def messages():
    out = []
    sink = logger.add(lambda m: out.append(m.record["message"]), level="TRACE")
    yield out
    logger.remove(sink)


class _Params:
    def __init__(self, arguments):
        self.tool_call_id = "call-123"
        self.arguments = arguments
        self.results = []

    async def result_callback(self, result, properties=None):
        self.results.append(result)


@pytest.mark.asyncio
async def test_tool_execute_logs_keys_and_lengths_not_arguments(monkeypatch, messages):
    tools = _load("_tools_voip1633", "tools.py", monkeypatch)
    fake, session = _fake_aiohttp(_Resp(200, json.dumps({"ok": True})))
    monkeypatch.setattr(tools, "aiohttp", fake)
    params = _Params({"body": SENTENCE, "run_llm": False})

    await tools.tool_execute("send_message", params, "pcc-1")

    joined = "\n".join(messages)
    assert SENTENCE not in joined and "ZXQ" not in joined
    executing = [m for m in messages if "Executing" in m]
    assert len(executing) == 1
    assert "[send_message]" in executing[0] and "tool_call_id=call-123" in executing[0]
    assert "arg_keys=['body']" in executing[0]  # run_llm is control data and is popped first
    assert f"args_len={len(json.dumps({'body': SENTENCE}, ensure_ascii=False))}" in executing[0]
    posting = [m for m in messages if "POST" in m]
    assert len(posting) == 1 and "args_len=" in posting[0] and "pcc-1" in posting[0]
    assert session.posted  # the request still carries the real arguments
    assert SENTENCE in session.posted[0][1]["json"]["function"]["arguments"]


@pytest.mark.asyncio
async def test_tool_execute_http_error_logs_status_and_length(monkeypatch, messages):
    tools = _load("_tools_voip1633_err", "tools.py", monkeypatch)
    body = json.dumps({"error": SENTENCE})
    fake, _ = _fake_aiohttp(_Resp(400, body))
    monkeypatch.setattr(tools, "aiohttp", fake)
    params = _Params({})

    await tools.tool_execute("send_message", params, "pcc-1")

    joined = "\n".join(messages)
    assert "ZXQ" not in joined
    warn = [m for m in messages if "HTTP 400" in m]
    assert len(warn) == 1 and f"body_len={len(body)}" in warn[0] and "[send_message]" in warn[0]
    assert params.results[0]["status"] == "error"  # the LLM still gets the error text


@pytest.mark.asyncio
async def test_team_tool_endpoint_error_logs_status_and_length(monkeypatch, messages):
    team_flow = _load("_team_flow_voip1633", "team_flow.py", monkeypatch)
    body = SENTENCE + " detail"
    fake, _ = _fake_aiohttp(_Resp(500, body))
    monkeypatch.setattr(team_flow, "aiohttp", fake)
    monkeypatch.setattr(team_flow, "_http_session", None)

    result = await team_flow._call_go_tool_endpoint("lookup", {"q": SENTENCE}, "pcc-9")

    joined = "\n".join(messages)
    assert "ZXQ" not in joined
    warn = [m for m in messages if "HTTP 500" in m]
    assert len(warn) == 1 and f"body_len={len(body)}" in warn[0] and "[team_flow][lookup]" in warn[0]
    assert result["status"] == "error"


@pytest.mark.asyncio
async def test_member_switched_error_logs_status_and_length(monkeypatch, messages):
    team_flow = _load("_team_flow_voip1633_ms", "team_flow.py", monkeypatch)
    body = SENTENCE
    fake, _ = _fake_aiohttp(_Resp(502, body))
    monkeypatch.setattr(team_flow, "aiohttp", fake)
    monkeypatch.setattr(team_flow, "_http_session", None)
    team = {"members": [{"id": "a", "ai": {}}, {"id": "b", "ai": {}}]}

    await team_flow._notify_member_switched("pcc-9", "a", "b", "go_b", team)

    joined = "\n".join(messages)
    assert "ZXQ" not in joined
    warn = [m for m in messages if "HTTP 502" in m]
    assert len(warn) == 1 and f"body_len={len(body)}" in warn[0]
