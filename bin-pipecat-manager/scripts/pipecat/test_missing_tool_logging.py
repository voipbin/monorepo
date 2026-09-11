"""Tests for register_missing_tool_logging (VOIP-1512).

conftest.py mocks the whole ``tools`` module (so run.py can be imported without
the real pipecat package), and also pre-populates ``sys.modules['pipecat']``
and several submodules with MagicMocks. ``tools.py`` itself imports
``pipecat.services.llm_service`` directly, which conftest does NOT mock, so
loading the real tools.py under those mocks raises ModuleNotFoundError
(``pipecat.services`` is a MagicMock, not a real package).

The pipecat package IS actually installed in this venv (see .venv), so to
exercise the real ``register_missing_tool_logging`` we drop conftest's
``pipecat*`` mock entries from ``sys.modules`` right before importing tools.py,
forcing Python to import the genuine installed package instead. The helper
itself only touches the passed-in llm_service's ``event_handler`` decorator and
``service._functions`` membership, so a plain fake service is enough for the
rest of the test.
"""

import importlib.util
import os
import sys

import pytest


def _load_real_tools():
    """Import the real tools.py, forcing genuine (non-mocked) pipecat imports.

    Snapshots and restores conftest's ``pipecat*`` sys.modules entries so this
    does not leak the real pipecat package into other test modules that run in
    the same pytest process and rely on conftest's mocks (e.g. test_run.py's
    ``Language`` enum identity checks).
    """
    saved = {
        name: mod
        for name, mod in sys.modules.items()
        if name == "pipecat" or name.startswith("pipecat.")
    }
    for mod_name in saved:
        del sys.modules[mod_name]
    try:
        path = os.path.join(os.path.dirname(__file__), "tools.py")
        spec = importlib.util.spec_from_file_location("_real_tools_voip1512", path)
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        return mod
    finally:
        for mod_name in list(sys.modules):
            if mod_name == "pipecat" or mod_name.startswith("pipecat."):
                del sys.modules[mod_name]
        sys.modules.update(saved)


class _FakeCall:
    def __init__(self, function_name):
        self.function_name = function_name


class _FakeService:
    """Minimal stand-in for a pipecat LLMService.

    Captures the handler registered via the ``event_handler`` decorator and
    exposes a ``_functions`` dict just like the real service.
    """

    def __init__(self, registered_names):
        self._functions = {name: object() for name in registered_names}
        self._handlers = {}

    def event_handler(self, event_name):
        def _decorator(fn):
            self._handlers[event_name] = fn
            return fn
        return _decorator


@pytest.mark.asyncio
async def test_logs_error_for_unadvertised_tool(caplog):
    tools = _load_real_tools()
    svc = _FakeService(registered_names=["get_contact_profile"])

    tools.register_missing_tool_logging(svc, "pcc-123")
    handler = svc._handlers["on_function_calls_started"]

    import logging
    with caplog.at_level(logging.ERROR):
        # loguru is used in tools.py; route it into a plain assertion instead.
        # We assert on the handler behavior directly by capturing logger calls.
        logged = []
        tools.logger.error = lambda msg: logged.append(msg)
        await handler(svc, [_FakeCall("notify_agent")])

    assert len(logged) == 1
    assert "[missing_tool]" in logged[0]
    assert "pipecatcall_id=pcc-123" in logged[0]
    assert "tool_name=notify_agent" in logged[0]


@pytest.mark.asyncio
async def test_no_error_for_registered_tool():
    tools = _load_real_tools()
    svc = _FakeService(registered_names=["get_contact_profile", "notify_agent"])

    tools.register_missing_tool_logging(svc, "pcc-456")
    handler = svc._handlers["on_function_calls_started"]

    logged = []
    tools.logger.error = lambda msg: logged.append(msg)
    await handler(svc, [_FakeCall("notify_agent"), _FakeCall("get_contact_profile")])

    assert logged == []


@pytest.mark.asyncio
async def test_mixed_calls_only_logs_missing():
    tools = _load_real_tools()
    svc = _FakeService(registered_names=["get_contact_profile"])

    tools.register_missing_tool_logging(svc, "pcc-789")
    handler = svc._handlers["on_function_calls_started"]

    logged = []
    tools.logger.error = lambda msg: logged.append(msg)
    await handler(
        svc,
        [_FakeCall("get_contact_profile"), _FakeCall("notify_agent"), _FakeCall("emit_info_card")],
    )

    # only the two unregistered tools are logged
    assert len(logged) == 2
    assert any("tool_name=notify_agent" in m for m in logged)
    assert any("tool_name=emit_info_card" in m for m in logged)
    assert all("tool_name=get_contact_profile" not in m for m in logged)


def test_handler_registered_on_correct_event():
    tools = _load_real_tools()
    svc = _FakeService(registered_names=["x"])
    tools.register_missing_tool_logging(svc, "pcc-1")
    assert "on_function_calls_started" in svc._handlers
