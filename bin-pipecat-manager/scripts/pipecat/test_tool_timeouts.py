"""Tests for the tool call timeouts (VOIP-1632).

The chain is: ai-manager bounds one MCP call at 10 s, the Go RPC from
pipecat-manager to ai-manager at 15 s, and the two HTTP calls this runner makes
to pipecat-manager's ``/tools`` endpoint (``tools.tool_execute`` and
``team_flow._call_go_tool_endpoint``) at 20 s. Each outer limit must outlast the
inner one, or the caller reports a failure while the inner call is still
running and its result is stored after the answer.

conftest.py replaces ``aiohttp`` with a mock module, so only the arguments given
to ``ClientTimeout`` are asserted here, not real aiohttp behaviour.
"""

import asyncio
import importlib.util
import os
import sys
from unittest.mock import AsyncMock, MagicMock, patch

HERE = os.path.dirname(__file__)


def _load_real_tools():
    """Import the real tools.py with genuine pipecat imports (see test_missing_tool_logging)."""
    saved = {
        name: mod
        for name, mod in sys.modules.items()
        if name == "pipecat" or name.startswith("pipecat.")
    }
    for mod_name in saved:
        del sys.modules[mod_name]
    try:
        spec = importlib.util.spec_from_file_location("_real_tools_voip1632", os.path.join(HERE, "tools.py"))
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        return mod
    finally:
        for mod_name in list(sys.modules):
            if mod_name == "pipecat" or mod_name.startswith("pipecat."):
                del sys.modules[mod_name]
        sys.modules.update(saved)


def _fake_session_cm(status=200):
    """A ClientSession mock whose post() answers with a JSON 200 response."""
    response = MagicMock()
    response.status = status
    response.headers = {"Content-Type": "application/json"}
    response.text = AsyncMock(return_value="{}")

    post_cm = MagicMock()
    post_cm.__aenter__ = AsyncMock(return_value=response)
    post_cm.__aexit__ = AsyncMock(return_value=False)

    session = MagicMock()
    session.post = MagicMock(return_value=post_cm)

    session_cm = MagicMock()
    session_cm.__aenter__ = AsyncMock(return_value=session)
    session_cm.__aexit__ = AsyncMock(return_value=False)
    return session_cm


def test_tool_execute_waits_20_seconds():
    tools = _load_real_tools()

    params = MagicMock()
    params.arguments = {}
    params.tool_call_id = "tc-1"
    params.result_callback = AsyncMock()

    with patch.object(tools.aiohttp, "ClientSession", return_value=_fake_session_cm()), patch.object(
        tools.aiohttp, "ClientTimeout"
    ) as client_timeout:
        asyncio.run(tools.tool_execute("mcp_12345678_x", params, "pc-1"))

    client_timeout.assert_called_once_with(total=20)


def test_team_flow_session_waits_20_seconds():
    # conftest.py puts a mock team_flow in sys.modules for run.py's tests; import the real one
    # for this test only and put back whatever was there.
    previous = sys.modules.pop("team_flow", None)
    try:
        import team_flow

        team_flow._http_session = None
        with patch.object(team_flow.aiohttp, "ClientSession") as client_session, patch.object(
            team_flow.aiohttp, "ClientTimeout"
        ) as client_timeout:
            client_session.return_value.closed = False
            asyncio.run(team_flow._get_http_session())

        client_timeout.assert_called_once_with(total=20)
    finally:
        sys.modules.pop("team_flow", None)
        if previous is not None:
            sys.modules["team_flow"] = previous


def test_pipecat_function_call_timeout_is_left_at_its_default():
    """Pins the assumption behind the 20 s order (not a regression test).

    pipecat-ai 1.12 has no function call timeout unless ``function_call_timeout_secs``
    or a per tool ``timeout_secs`` is set. If either is ever set below 15 s, a slow
    MCP result would again be cut before it reaches the LLM.
    """
    tools = _load_real_tools()

    llm_service = MagicMock()
    tools.tool_register(llm_service, "pc-1", ["mcp_12345678_x"], [{"name": "mcp_12345678_x", "run_llm": True}])

    llm_service.register_function.assert_called_once()
    args, kwargs = llm_service.register_function.call_args
    assert len(args) == 2
    assert kwargs == {}

    with open(os.path.join(HERE, "run.py"), encoding="utf-8") as f:
        assert "function_call_timeout_secs" not in f.read()
