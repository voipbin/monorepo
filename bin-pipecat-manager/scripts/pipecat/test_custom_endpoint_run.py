"""run.py and main.py behavior for customer specified endpoints (VOIP-1599 Layer 2, design 3.8 and 8a).

custom_endpoint.py needs the real pipecat (conftest.py mocks it), so it is replaced here by a fake
module; the real subclass and factory are covered by scripts/pipecat_realtest/test_custom_endpoint_real.py.
"""

import asyncio
import json
import sys
import types
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from loguru import logger

import custom_endpoint_net as net
import run
from test_init_pipeline import _run_team_init, _two_google_members

CUSTOMER = "11111111-2222-3333-4444-555555555555"
URL = "https://secret-llm.example.test/v1"
KEY = "sk-SECRET-endpoint-key"
LEGACY = "legacy-engine-key-DO-NOT-FORWARD"


class FakeLLM:
    def __init__(self, kwargs):
        self.kwargs = kwargs
        self.slot_released = 0

    def release_slot(self):
        self.slot_released += 1


@pytest.fixture
def fake_endpoint(monkeypatch):
    """Fake custom_endpoint module; fresh slot table and registry."""
    built = []
    state = types.SimpleNamespace(built=built, error=None)

    def build(model, **kwargs):
        if state.error:
            raise state.error
        svc = FakeLLM(dict(model=model, **kwargs))
        built.append(svc)
        return svc

    monkeypatch.setitem(sys.modules, "custom_endpoint", types.SimpleNamespace(build_custom_endpoint_llm=build))
    monkeypatch.setattr(net, "SLOTS", net.SlotTable())
    monkeypatch.setattr(net, "_PIPELINES", {})
    monkeypatch.setattr(net, "_RESERVED", set())
    return state


@pytest.fixture
def stub_context():
    with patch("run.LLMContextAggregatorPair"), patch("run.LLMContext"):
        yield


# --- create_llm_service ---------------------------------------------------------------------------


def test_customendpoint_branch_passes_the_dedicated_values_and_ignores_the_legacy_key(fake_endpoint, stub_context):
    llm, aggregator = run.create_llm_service(
        "customendpoint.my-model", LEGACY, [], [], pipeline_id="p1",
        customer_id=CUSTOMER, llm_base_url=URL, llm_base_key=KEY,
    )
    assert llm is fake_endpoint.built[0]
    assert fake_endpoint.built[0].kwargs == dict(
        model="my-model", base_url=URL, api_key=KEY, customer_id=CUSTOMER, pipeline_id="p1"
    )
    assert LEGACY not in json.dumps(fake_endpoint.built[0].kwargs)


@pytest.mark.parametrize("type_", ["CustomEndpoint.m", "customendpoint:m"])
def test_service_name_is_case_insensitive_and_accepts_colon(fake_endpoint, stub_context, type_):
    run.create_llm_service(type_, "", [], [], pipeline_id="p1", customer_id=CUSTOMER, llm_base_url=URL, llm_base_key="")
    assert fake_endpoint.built[0].kwargs["model"] == "m"


@pytest.mark.parametrize("error", [RuntimeError("boom " + URL + KEY), ValueError("custom endpoint is not available"), OSError(KEY)])
def test_any_build_failure_is_the_fixed_text_without_cause(fake_endpoint, stub_context, error):
    fake_endpoint.error = error
    with pytest.raises(ValueError) as ei:
        run.create_llm_service("customendpoint.m", LEGACY, [], [], pipeline_id="p1", customer_id=CUSTOMER, llm_base_url=URL, llm_base_key=KEY)
    assert str(ei.value) == "custom endpoint is not available"
    assert ei.value.__cause__ is None and ei.value.__suppress_context__ is True


def test_missing_options_reach_the_factory_unchanged_so_it_can_refuse(fake_endpoint, stub_context):
    run.create_llm_service("customendpoint.m", LEGACY, [], [], pipeline_id="p1")
    kw = fake_endpoint.built[0].kwargs
    assert kw["customer_id"] is None and kw["base_url"] is None and kw["api_key"] is None


def test_context_failure_after_the_build_gives_the_slot_back(fake_endpoint):
    with patch("run.LLMContext", side_effect=RuntimeError(KEY)), patch("run.LLMContextAggregatorPair"):
        with pytest.raises(ValueError) as ei:
            run.create_llm_service("customendpoint.m", "", [], [], pipeline_id="p1", customer_id=CUSTOMER, llm_base_url=URL, llm_base_key=KEY)
    assert str(ei.value) == "custom endpoint is not available" and ei.value.__cause__ is None
    assert fake_endpoint.built[0].slot_released == 1


# --- _member_llm_type ---------------------------------------------------------------------------------


@pytest.mark.parametrize("model", ["customendpoint.m", "CustomEndpoint.m", "customendpoint:m"])
def test_member_type_never_honors_customendpoint_from_customer_data(model):
    with pytest.raises(ValueError, match="engine model is not available"):
        run._member_llm_type({"engine_model": model})  # llm_type absent: older Go


def test_member_type_accepts_the_resolved_customendpoint_type_from_go():
    assert run._member_llm_type({"llm_type": "customendpoint.m", "engine_model": "customendpoint.m"}) == "customendpoint.m"
    with pytest.raises(ValueError):
        run._member_llm_type({"llm_type": "", "engine_model": "customendpoint.m"})


# --- init paths pass the arguments ---------------------------------------------------------------------


@pytest.mark.asyncio
async def test_init_pipeline_flows_customer_and_endpoint_into_single_pipeline():
    with patch("run.init_single_ai_pipeline", new=AsyncMock(return_value={})) as single:
        await run.init_pipeline(
            "p1", "customendpoint.m", "", [], customer_id=CUSTOMER, llm_base_url=URL, llm_base_key=KEY
        )
    kw = single.call_args.kwargs
    assert (kw["customer_id"], kw["llm_base_url"], kw["llm_base_key"]) == (CUSTOMER, URL, KEY)


@pytest.mark.asyncio
async def test_init_single_creates_the_llm_with_the_options():
    with patch("run.create_llm_service", side_effect=ValueError("custom endpoint is not available")) as create, \
         patch("run.create_websocket_transport"):
        with pytest.raises(ValueError):
            await run.init_single_ai_pipeline(
                "p1", "customendpoint.m", LEGACY, [], customer_id=CUSTOMER, llm_base_url=URL, llm_base_key=KEY
            )
    assert create.call_args.args[:2] == ("customendpoint.m", LEGACY)
    kw = create.call_args.kwargs
    assert (kw["pipeline_id"], kw["customer_id"], kw["llm_base_url"], kw["llm_base_key"]) == ("p1", CUSTOMER, URL, KEY)


@pytest.mark.asyncio
async def test_failed_init_log_has_no_exception_text_for_a_custom_session():
    lines = []
    sink = logger.add(lines.append, level="DEBUG")
    try:
        with patch("run.create_llm_service", side_effect=ValueError("custom endpoint is not available " + KEY)), \
             patch("run.create_websocket_transport"):
            with pytest.raises(ValueError):
                await run.init_single_ai_pipeline("p1", "customendpoint.m", "", [], customer_id=CUSTOMER, llm_base_url=URL, llm_base_key=KEY)
        with patch("run.create_llm_service", side_effect=ValueError("plain failure text")), patch("run.create_websocket_transport"):
            with pytest.raises(ValueError):
                await run.init_single_ai_pipeline("p2", "openai.gpt-4o", "k", [])
    finally:
        logger.remove(sink)
    text = "".join(str(m) for m in lines)
    assert KEY not in text and URL not in text
    assert "Pipeline initialization failed: custom endpoint session error (details withheld)" in text
    assert "Pipeline initialization failed: plain failure text" in text


@pytest.mark.asyncio
async def test_team_members_get_their_own_url_key_and_the_session_customer():
    team = _two_google_members()
    team[0]["ai"].update(llm_type="customendpoint.a", llm_base_url=URL + "/a", llm_base_key="key-a")
    team[1]["ai"].update(llm_type="customendpoint.b", llm_base_url=URL + "/b", llm_base_key="key-b", engine_key=LEGACY)
    ctx, mocks = await _run_team_init(
        {"id": "t", "start_member_id": "member-1", "members": team}, pipeline_id="p9", customer_id=CUSTOMER
    )
    calls = mocks.create_llm_service.call_args_list
    assert [c.args[0] for c in calls] == ["customendpoint.a", "customendpoint.b"]
    for c, (url, key) in zip(calls, [(URL + "/a", "key-a"), (URL + "/b", "key-b")]):
        assert c.kwargs["pipeline_id"] == "p9" and c.kwargs["customer_id"] == CUSTOMER
        assert (c.kwargs["llm_base_url"], c.kwargs["llm_base_key"]) == (url, key)
    assert ctx["custom_endpoint"] is True


@pytest.mark.asyncio
async def test_team_without_custom_member_is_not_marked():
    team = _two_google_members()
    ctx, mocks = await _run_team_init({"id": "t", "start_member_id": "member-1", "members": team}, pipeline_id="p9")
    assert ctx["custom_endpoint"] is False
    assert all(c.kwargs["llm_base_url"] is None for c in mocks.create_llm_service.call_args_list)


@pytest.mark.asyncio
async def test_team_member_that_fails_raises_value_error():
    team = _two_google_members()
    team[1]["ai"]["llm_type"] = ""
    with pytest.raises(ValueError, match="engine model is not available"):
        await _run_team_init({"id": "t", "start_member_id": "member-1", "members": team}, pipeline_id="p9")


# --- release_pipeline safety net (outermost finally) --------------------------------------------------------


def _single_ctx(**over):
    ctx = {
        "task": MagicMock(cancel=AsyncMock()),
        "transport_input": None,
        "transport_output": MagicMock(cleanup=AsyncMock()),
        "llm_service": MagicMock(cleanup=AsyncMock()),
        "tts_service": None,
        "stt_service": None,
        "tool_names": [],
        "custom_endpoint": True,
    }
    ctx.update(over)
    return ctx


def _team_ctx(**over):
    ctx = {
        "task": MagicMock(cancel=AsyncMock()),
        "transport_input": None,
        "transport_output": MagicMock(cleanup=AsyncMock()),
        "routing_llm": MagicMock(cleanup=AsyncMock()),
        "routing_tts": None,
        "routing_stt": None,
        "custom_endpoint": True,
    }
    ctx.update(over)
    return ctx


class Held:
    """A registered service holding one slot."""

    def __init__(self, pipeline="p1"):
        self.pipeline = pipeline
        self.released = 0
        assert net.SLOTS.acquire(CUSTOMER)
        self.slot = True
        net.register_service(pipeline, self)

    def release_slot(self):
        if self.slot:
            self.slot = False
            net.SLOTS.release(CUSTOMER)
            net.unregister_service(self.pipeline, self)

    async def release(self):
        self.released += 1
        self.release_slot()


def _runner(run_error=None):
    runner = MagicMock()
    runner.run = AsyncMock(side_effect=run_error)
    return MagicMock(return_value=runner)


@pytest.mark.asyncio
@pytest.mark.parametrize("kind", ["single", "team"])
@pytest.mark.parametrize("failing", ["task.cancel", "transport_output.cleanup", "llm.cleanup"])
async def test_release_pipeline_runs_when_the_cleanup_chain_fails(fake_endpoint, kind, failing):
    held = Held("p1")
    ctx = _single_ctx() if kind == "single" else _team_ctx()
    if failing == "task.cancel":
        ctx["task"].cancel = AsyncMock(side_effect=RuntimeError("cleanup failed"))
    elif failing == "transport_output.cleanup":
        ctx["transport_output"].cleanup = AsyncMock(side_effect=RuntimeError("cleanup failed"))
    else:
        (ctx["llm_service"] if kind == "single" else ctx["routing_llm"]).cleanup = AsyncMock(side_effect=RuntimeError("cleanup failed"))
    fn = run.execute_single_ai_pipeline if kind == "single" else run.execute_team_pipeline
    with patch("run.PipelineRunner", _runner()), patch("run.task_manager", MagicMock(remove=AsyncMock())), patch("run.tool_unregister"):
        with pytest.raises(RuntimeError, match="cleanup failed"):
            await fn("p1", ctx)
    assert net.SLOTS.held() == 0 and held.released == 1
    assert not net.pipeline_registered("p1")


@pytest.mark.asyncio
@pytest.mark.parametrize("kind", ["single", "team"])
async def test_release_pipeline_runs_on_cancellation_during_cleanup(fake_endpoint, kind):
    held = Held("p1")
    ctx = _single_ctx() if kind == "single" else _team_ctx()
    ctx["task"].cancel = AsyncMock(side_effect=asyncio.CancelledError())
    fn = run.execute_single_ai_pipeline if kind == "single" else run.execute_team_pipeline
    with patch("run.PipelineRunner", _runner()), patch("run.task_manager", MagicMock(remove=AsyncMock())), patch("run.tool_unregister"):
        with pytest.raises(asyncio.CancelledError):
            await fn("p1", ctx)
    assert net.SLOTS.held() == 0 and held.released == 1


@pytest.mark.asyncio
@pytest.mark.parametrize("kind", ["single", "team"])
async def test_release_pipeline_runs_on_normal_end_and_logs_no_raw_error(fake_endpoint, kind):
    held = Held("p1")
    ctx = _single_ctx() if kind == "single" else _team_ctx()
    fn = run.execute_single_ai_pipeline if kind == "single" else run.execute_team_pipeline
    lines = []
    sink = logger.add(lines.append, level="DEBUG")
    try:
        with patch("run.PipelineRunner", _runner(RuntimeError("body " + KEY + URL))), \
             patch("run.task_manager", MagicMock(remove=AsyncMock())), patch("run.tool_unregister"):
            await fn("p1", ctx)
    finally:
        logger.remove(sink)
    text = "".join(str(m) for m in lines)
    assert KEY not in text and URL not in text and "custom endpoint session error (details withheld)" in text
    assert net.SLOTS.held() == 0 and held.released == 1


@pytest.mark.asyncio
async def test_non_custom_session_keeps_the_raw_error_text(fake_endpoint):
    lines = []
    sink = logger.add(lines.append, level="DEBUG")
    try:
        with patch("run.PipelineRunner", _runner(RuntimeError("plain pipeline failure"))), \
             patch("run.task_manager", MagicMock(remove=AsyncMock())), patch("run.tool_unregister"):
            await run.execute_single_ai_pipeline("p2", _single_ctx(custom_endpoint=False))
    finally:
        logger.remove(sink)
    assert "plain pipeline failure" in "".join(str(m) for m in lines)
