"""main.py: fixed texts, masked logs, duplicate pipeline ids and release on failure (VOIP-1599 Layer 2)."""

import asyncio
import sys
import types
from unittest.mock import AsyncMock, patch

import pytest
from fastapi.testclient import TestClient
from loguru import logger

import custom_endpoint_net as net
import main

CUSTOMER = "11111111-2222-3333-4444-555555555555"
URL = "https://secret-llm.example.test/v1"
KEY = "sk-SECRET-endpoint-key"
FIXED = "custom endpoint is not available"


@pytest.fixture(autouse=True)
def isolated(monkeypatch):
    monkeypatch.setattr(net, "SLOTS", net.SlotTable())
    monkeypatch.setattr(net, "_PIPELINES", {})
    monkeypatch.setattr(net, "_RESERVED", set())


@pytest.fixture
def client():
    return TestClient(main.app, raise_server_exceptions=True)


@pytest.fixture
def logs():
    lines = []
    sink = logger.add(lines.append, level="DEBUG")
    yield lines
    logger.remove(sink)


def body(**over):
    b = {
        "id": "pipe-1", "llm_type": "customendpoint.my-model", "llm_key": "LEGACY-KEY", "customer_id": CUSTOMER,
        "llm_base_url": URL, "llm_base_key": KEY, "llm_messages": [{"role": "user", "content": "hi"}],
    }
    b.update(over)
    return b


def team_body(member_type="customendpoint.m"):
    def member(mid, t):
        return {"id": mid, "name": mid, "ai": {"engine_model": "x.y", "engine_key": "LEGACY", "llm_type": t,
                                                "llm_base_url": URL, "llm_base_key": KEY}}
    return {"id": "pipe-t", "customer_id": CUSTOMER, "resolved_team": {
        "id": "t", "start_member_id": "m1", "members": [member("m1", "openai.gpt-4o"), member("m2", member_type)]}}


def text_of(logs):
    return "".join(str(m) for m in logs)


# --- RequestValidationError ----------------------------------------------------------------------


def test_validation_error_body_is_fixed_and_does_not_echo_the_request(client):
    r = client.post("/run", json=body(id=5, llm_messages="not-a-list"))
    assert r.status_code == 422
    assert r.json() == {"detail": "invalid request"}
    assert KEY not in r.text and URL not in r.text and "LEGACY" not in r.text


def test_validation_error_logs_field_and_type_only(client, logs):
    r = client.post("/run", json=body(id=5, llm_messages="not-a-list"))
    assert r.status_code == 422
    text = text_of(logs)
    assert "Request validation failed" in text and "llm_messages" in text
    assert KEY not in text and URL not in text and "LEGACY" not in text and "not-a-list" not in text


def test_validation_error_for_a_missing_body_is_fixed_too(client):
    r = client.post("/run", content=b"not json", headers={"content-type": "application/json"})
    assert r.status_code == 422 and r.json() == {"detail": "invalid request"}


# --- masked request log and repr -----------------------------------------------------------------------


def test_run_request_log_masks_the_llm_type(client, logs):
    with patch.object(main, "init_pipeline", AsyncMock(return_value={"type": "single"})), patch.object(main, "execute_pipeline", AsyncMock()):
        assert client.post("/run", json=body()).status_code == 200
    text = text_of(logs)
    assert "customendpoint.<redacted>" in text
    assert "my-model" not in text and KEY not in text and URL not in text and "LEGACY-KEY" not in text


def test_run_request_log_keeps_other_llm_types(client, logs):
    with patch.object(main, "init_pipeline", AsyncMock(return_value={"type": "single"})), patch.object(main, "execute_pipeline", AsyncMock()):
        assert client.post("/run", json={"id": "p2", "llm_type": "openai.gpt-4o", "llm_key": "k"}).status_code == 200
    assert "openai.gpt-4o" in text_of(logs)


def test_models_do_not_repr_the_secrets():
    req = main.PipelineRequest(**body())
    assert KEY not in repr(req) and URL not in repr(req) and "LEGACY-KEY" not in repr(req)
    ai = main.ResolvedAI(engine_model="a.b", engine_key="LEGACY-KEY", llm_base_url=URL, llm_base_key=KEY)
    assert KEY not in repr(ai) and URL not in repr(ai) and "LEGACY-KEY" not in repr(ai)


def test_request_models_carry_the_new_fields():
    req = main.PipelineRequest(**body())
    assert (req.customer_id, req.llm_base_url, req.llm_base_key) == (CUSTOMER, URL, KEY)
    ai = main.ResolvedAI(engine_model="a.b", engine_key="", llm_base_url=URL, llm_base_key=KEY)
    assert (ai.llm_base_url, ai.llm_base_key) == (URL, KEY)
    assert main.ResolvedAI(engine_model="a.b", engine_key="").llm_base_url is None


# --- init arguments -----------------------------------------------------------------------------------------


def test_run_passes_customer_and_endpoint_to_init_pipeline(client):
    init = AsyncMock(return_value={"type": "single"})
    with patch.object(main, "init_pipeline", init), patch.object(main, "execute_pipeline", AsyncMock()):
        assert client.post("/run", json=body()).status_code == 200
    kw = init.call_args.kwargs
    assert (kw["customer_id"], kw["llm_base_url"], kw["llm_base_key"]) == (CUSTOMER, URL, KEY)


def test_team_member_url_and_key_reach_the_runner_data(client):
    init = AsyncMock(return_value={"type": "team"})
    with patch.object(main, "init_pipeline", init), patch.object(main, "execute_pipeline", AsyncMock()):
        assert client.post("/run", json=team_body()).status_code == 200
    members = init.call_args.kwargs["resolved_team"]["members"]
    assert [(m["ai"]["llm_base_url"], m["ai"]["llm_base_key"]) for m in members] == [(URL, KEY)] * 2


# --- fixed details ---------------------------------------------------------------------------------------------


def test_value_error_of_a_custom_request_is_a_fixed_400(client, logs):
    with patch.object(main, "init_pipeline", AsyncMock(side_effect=ValueError("bad " + URL + KEY))):
        r = client.post("/run", json=body())
    assert r.status_code == 400 and r.json() == {"detail": FIXED}
    assert KEY not in text_of(logs) and URL not in text_of(logs)


def test_other_error_of_a_custom_request_is_a_fixed_500(client, logs):
    with patch.object(main, "init_pipeline", AsyncMock(side_effect=RuntimeError("boom " + URL + KEY))):
        r = client.post("/run", json=body())
    assert r.status_code == 500 and r.json() == {"detail": "pipeline initialization failed"}
    assert KEY not in r.text and KEY not in text_of(logs) and URL not in text_of(logs)
    assert "Traceback" not in text_of(logs)


def test_team_with_a_custom_member_gets_the_fixed_details_too(client):
    with patch.object(main, "init_pipeline", AsyncMock(side_effect=ValueError("engine model is not available"))):
        r = client.post("/run", json=team_body())
    assert r.status_code == 400 and r.json() == {"detail": FIXED}


def test_errors_of_other_requests_keep_their_text(client):
    plain = {"id": "p3", "llm_type": "openai.gpt-4o", "llm_key": "k"}
    with patch.object(main, "init_pipeline", AsyncMock(side_effect=ValueError("Unsupported thing"))):
        r = client.post("/run", json=plain)
    assert r.status_code == 400 and r.json() == {"detail": "Unsupported thing"}
    with patch.object(main, "init_pipeline", AsyncMock(side_effect=RuntimeError("some failure"))):
        r = client.post("/run", json=plain)
    assert r.status_code == 500 and r.json() == {"detail": "some failure"}


@pytest.mark.asyncio
async def test_execution_failure_log_has_no_raw_error_for_a_custom_session(logs):
    with patch.object(main, "execute_pipeline", AsyncMock(side_effect=RuntimeError(KEY + URL))):
        await main.execute_pipeline_wrapper("p1", {"custom_endpoint": True})
        await main.execute_pipeline_wrapper("p2", {"custom_endpoint": False})
    text = text_of(logs)
    assert KEY not in text.split("p2")[0] and "custom endpoint session error (details withheld)" in text
    assert text.count(KEY) >= 1  # the other (non custom) session still logs its error as before


# --- duplicate ids and release on failure ---------------------------------------------------------------------------


def test_duplicate_pipeline_id_is_refused_and_keeps_the_running_pipeline(client):
    class Held:
        released = 0

        def release_slot(self):
            Held.released += 1

        async def release(self):
            Held.released += 1

    assert net.SLOTS.acquire(CUSTOMER)
    net.reserve_pipeline("pipe-1")
    net.register_service("pipe-1", Held())
    init = AsyncMock(return_value={"type": "single"})
    release = AsyncMock()
    with patch.object(main, "init_pipeline", init), patch.object(net, "release_pipeline", release):
        r = client.post("/run", json=body())
    assert r.status_code == 409 and r.json() == {"detail": FIXED}
    init.assert_not_called()
    release.assert_not_called()  # a refused request releases nothing
    assert Held.released == 0 and net.pipeline_registered("pipe-1") and net.SLOTS.held() == 1


def test_duplicate_check_is_made_once_per_request_and_registers_the_id(client):
    init = AsyncMock(return_value={"type": "single"})
    with patch.object(main, "init_pipeline", init), patch.object(main, "execute_pipeline", AsyncMock()):
        assert client.post("/run", json=body()).status_code == 200
        assert net.pipeline_registered("pipe-1")
        assert client.post("/run", json=body()).status_code == 409
    assert init.await_count == 1


def test_non_custom_requests_do_not_touch_the_registry(client):
    plain = {"id": "pipe-1", "llm_type": "openai.gpt-4o", "llm_key": "k"}
    init = AsyncMock(return_value={"type": "single"})
    with patch.object(main, "init_pipeline", init), patch.object(main, "execute_pipeline", AsyncMock()):
        assert client.post("/run", json=plain).status_code == 200
        assert client.post("/run", json=plain).status_code == 200  # same id again: no custom endpoint, no check
    assert not net.pipeline_registered("pipe-1")


def test_custom_request_without_an_id_is_refused(client):
    r = client.post("/run", json=body(id=None))
    assert r.status_code == 409 and r.json() == {"detail": FIXED}


@pytest.mark.parametrize("error", [ValueError("x"), RuntimeError("x")])
def test_failed_init_releases_what_the_request_registered(client, error):
    released = []

    async def release(pid):
        released.append(pid)
        net._PIPELINES.pop(pid, None)
        net._RESERVED.discard(pid)

    with patch.object(main, "init_pipeline", AsyncMock(side_effect=error)), patch.object(net, "release_pipeline", release):
        r = client.post("/run", json=body())
    assert r.status_code in (400, 500) and released == ["pipe-1"]
    assert not net.pipeline_registered("pipe-1")  # the id can be used again


def test_failed_init_with_a_registered_service_gives_its_slot_back(client):
    """A built LLM service whose STT or TTS sibling failed: the /run failure path releases it."""

    class Svc:
        def __init__(self):
            assert net.SLOTS.acquire(CUSTOMER)
            net.register_service("pipe-1", self)
            self.closed = 0

        def release_slot(self):
            net.SLOTS.release(CUSTOMER)

        async def release(self):
            self.closed += 1

    holder = []

    async def init(*a, **k):
        holder.append(Svc())
        raise RuntimeError("stt failed")

    with patch.object(main, "init_pipeline", init):
        r = client.post("/run", json=body())
    assert r.status_code == 500
    assert net.SLOTS.held() == 0 and holder[0].closed == 1


def test_base_exception_during_init_releases_and_propagates():
    class Boom(BaseException):
        pass

    released = []

    async def release(pid):
        released.append(pid)

    async def go():
        with patch.object(main, "init_pipeline", AsyncMock(side_effect=Boom())), patch.object(net, "release_pipeline", release):
            await main.run_pipeline_endpoint(main.PipelineRequest(**body()))

    with pytest.raises(Boom):
        asyncio.run(go())
    assert released == ["pipe-1"]


def test_cancellation_during_init_releases_and_propagates():
    released = []

    async def release(pid):
        released.append(pid)

    async def go():
        with patch.object(main, "init_pipeline", AsyncMock(side_effect=asyncio.CancelledError())), patch.object(net, "release_pipeline", release):
            await main.run_pipeline_endpoint(main.PipelineRequest(**body()))

    with pytest.raises(asyncio.CancelledError):
        asyncio.run(go())
    assert released == ["pipe-1"]


def test_request_has_custom_endpoint_detection():
    assert main.request_has_custom_endpoint(main.PipelineRequest(**body()))
    assert main.request_has_custom_endpoint(main.PipelineRequest(**team_body()))
    assert not main.request_has_custom_endpoint(main.PipelineRequest(**team_body("openai.gpt-4o")))
    assert not main.request_has_custom_endpoint(main.PipelineRequest(id="x", llm_type="openai.gpt-4o"))
    # a member whose Go resolution was rejected still counts when its engine model is customendpoint
    t = team_body("openai.gpt-4o")
    t["resolved_team"]["members"][1]["ai"].update(engine_model="customendpoint.m", llm_type="")
    assert main.request_has_custom_endpoint(main.PipelineRequest(**t))
