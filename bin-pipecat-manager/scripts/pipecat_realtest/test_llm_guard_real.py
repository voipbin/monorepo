"""Real-library tests for the runner silent failure guards (VOIP-1633).

Real pipecat 1.12, openai 2.15 and a local OpenAI compatible stub server (no external call, no cost).
Lives outside scripts/pipecat/ because that conftest replaces pipecat with MagicMocks. Not run by CI.
Run manually in a venv with the pinned versions (scripts/pipecat/requirements.txt or `uv sync --frozen`
in scripts/pipecat) plus `pytest pytest-asyncio`:

    pytest --noconftest bin-pipecat-manager/scripts/pipecat_realtest/test_llm_guard_real.py

Each test runs a real PipelineTask: user aggregator -> LLM (or RoutingLLMService for a team) -> optional
stub TTS -> assistant aggregator, with the guards wired the way run.py wires them. "RTVI errors" are the
`error` messages PipelineTask's own RTVI processor sends to the transport (what pipecat-manager receives).
"""
import asyncio
import json
import os
import sys
import warnings

import pytest

HERE = os.path.dirname(__file__)
sys.path.insert(0, os.path.abspath(os.path.join(HERE, "..", "pipecat")))
for k in ("DEEPGRAM_API_KEY", "CARTESIA_API_KEY", "ELEVENLABS_API_KEY", "GOOGLE_API_KEY", "OPENAI_API_KEY"):
    os.environ.setdefault(k, "x")
warnings.simplefilter("ignore")

from aiohttp import web  # noqa: E402
from loguru import logger  # noqa: E402
from pipecat.frames.frames import (  # noqa: E402
    EndFrame,
    LLMRunFrame,
    OutputTransportMessageFrame,
    OutputTransportMessageUrgentFrame,
    TTSAudioRawFrame,
)
from pipecat.observers.base_observer import BaseObserver  # noqa: E402
from pipecat.pipeline.pipeline import Pipeline  # noqa: E402
from pipecat.pipeline.runner import PipelineRunner  # noqa: E402
from pipecat.pipeline.task import PipelineParams, PipelineTask  # noqa: E402
from pipecat.processors.aggregators.llm_context import LLMContext  # noqa: E402
from pipecat.processors.aggregators.llm_response_universal import LLMContextAggregatorPair  # noqa: E402
from pipecat.services.openai.llm import OpenAILLMService  # noqa: E402
from pipecat.services.tts_service import TTSService  # noqa: E402

import llm_guard  # noqa: E402
import tools  # noqa: E402
from message_filters import filter_valid_messages  # noqa: E402
from routing_llm import RoutingLLMService  # noqa: E402

pytestmark = pytest.mark.asyncio

EMPTY_TEXT = "LLM returned no usable output"
UNADVERTISED_TEXT = "LLM called an unadvertised tool"


def _chunk(delta, finish=None):
    return {"id": "c1", "object": "chat.completion.chunk", "created": 1, "model": "m",
            "choices": [{"index": 0, "delta": delta, "finish_reason": finish}]}


def _tool_chunks(name, call_id, arguments="{}"):
    call = {"index": 0, "type": "function", "function": {"arguments": arguments}}
    if name is not None:
        call["function"]["name"] = name
    if call_id is not None:
        call["id"] = call_id
    return [_chunk({"role": "assistant", "tool_calls": [call]}), _chunk({}, "tool_calls")]


RESPONSES = {
    "empty": [_chunk({"role": "assistant", "content": ""}), _chunk({}, "stop")],
    "text": [_chunk({"role": "assistant", "content": "Hello there"}), _chunk({}, "stop")],
    "valid_tool": _tool_chunks("get_x", "call_1"),
    "no_id_tool": _tool_chunks("get_x", None),
    "unknown_tool": _tool_chunks("nope", "call_9"),
    "bad_args": _tool_chunks("get_x", "call_1", '{"a": '),
    "no_name": _tool_chunks(None, "call_1"),
}


class _Stub:
    """Local OpenAI compatible server: request n answers kinds[n] (the last kind repeats)."""

    def __init__(self, kinds):
        self.kinds = kinds
        self.bodies = []

    async def handler(self, request):
        body = await request.json()
        self.bodies.append(body)
        kind = self.kinds[min(len(self.bodies) - 1, len(self.kinds) - 1)]
        if kind == "http500":
            return web.json_response({"error": {"message": "boom"}}, status=500)
        resp = web.StreamResponse(headers={"Content-Type": "text/event-stream"})
        await resp.prepare(request)
        for c in RESPONSES[kind]:
            await resp.write(f"data: {json.dumps(c)}\n\n".encode())
        await resp.write(b"data: [DONE]\n\n")
        return resp

    def second_request_tool_ids(self):
        out = []
        for m in (self.bodies[1]["messages"] if len(self.bodies) > 1 else []):
            if m.get("tool_calls"):
                out.append(("call", [tc.get("id") for tc in m["tool_calls"]]))
            if m.get("role") == "tool":
                out.append(("result", m.get("tool_call_id")))
        return out


class _StubTTS(TTSService):
    async def run_tts(self, text, context_id):
        yield TTSAudioRawFrame(audio=b"\x00\x00" * 160, sample_rate=16000, num_channels=1)


class _RTVITap(BaseObserver):
    """Collects the RTVI `error` messages that PipelineTask sends towards the transport."""

    def __init__(self):
        super().__init__(observe_every_push=False)
        self.errors = []

    async def on_push_frame(self, data):
        frame = data.frame
        if isinstance(frame, (OutputTransportMessageFrame, OutputTransportMessageUrgentFrame)):
            msg = frame.message
            if isinstance(msg, dict):
                if msg.get("type") == "error":
                    self.errors.append(msg["data"]["error"])
            elif getattr(msg, "type", None) == "error":
                self.errors.append(msg.data.error)


async def _run_session(kinds, *, team=False, tts=False, register_tool=True, wire_guard=True):
    stub = _Stub(kinds)
    app = web.Application()
    app.router.add_post("/v1/chat/completions", stub.handler)
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "127.0.0.1", 0)
    await site.start()
    port = site._server.sockets[0].getsockname()[1]

    llm = OpenAILLMService(api_key="x", base_url=f"http://127.0.0.1:{port}/v1",
                           settings=OpenAILLMService.Settings(model="m"))
    handler_calls = []

    async def get_x(params):
        handler_calls.append(params.tool_call_id)
        await params.result_callback({"ok": 1})

    if register_tool:
        llm.register_function("get_x", get_x)
    if wire_guard:
        tools.register_missing_tool_logging(llm, "pcc-real")
        llm_guard.normalize_tool_call_ids(llm, "pcc-real")

    stage = llm
    if team:
        stage = RoutingLLMService({"m1": llm})
        stage.set_active_member("m1")

    ctx = LLMContext([{"role": "user", "content": "hi"}])
    agg = LLMContextAggregatorPair(ctx)
    stages = [agg.user(), stage]
    if tts:
        stages.append(_StubTTS())
    stages.append(agg.assistant())

    tap = _RTVITap()
    observers = [tap]
    if wire_guard:
        observers.append(llm_guard.EmptyOutputObserver("pcc-real"))
    task = PipelineTask(Pipeline(stages), params=PipelineParams(), observers=observers,
                        cancel_on_idle_timeout=False)
    await task.queue_frames([LLMRunFrame()])

    async def stopper():
        await asyncio.sleep(1.5)
        await task.queue_frame(EndFrame())

    asyncio.create_task(stopper())
    await PipelineRunner(handle_sigint=False).run(task)
    await runner.cleanup()
    return stub, tap.errors, handler_calls


@pytest.fixture(autouse=True)
def _quiet_logs():
    logger.remove()
    yield
    # A later synchronous test in the same run (test_upgrade_112_real.py) builds services that call
    # asyncio.get_event_loop(); give it a current loop back after the async tests here.
    try:
        asyncio.get_event_loop()
    except RuntimeError:
        asyncio.set_event_loop(asyncio.new_event_loop())


VARIANTS = [
    pytest.param(dict(), id="single"),
    pytest.param(dict(tts=True), id="single-tts"),
    pytest.param(dict(team=True), id="team"),
]


@pytest.mark.parametrize("variant", VARIANTS)
async def test_empty_completion_becomes_one_rtvi_error(variant):
    stub, errors, _ = await _run_session(["empty"], **variant)
    assert errors == [EMPTY_TEXT]


@pytest.mark.parametrize("variant", VARIANTS)
@pytest.mark.parametrize("kind", ["bad_args", "no_name"])
async def test_malformed_tool_call_becomes_one_rtvi_error(variant, kind):
    stub, errors, handler_calls = await _run_session([kind], **variant)
    assert errors == [EMPTY_TEXT]
    assert handler_calls == []


@pytest.mark.parametrize("variant", VARIANTS)
async def test_tool_without_handler_becomes_one_rtvi_error(variant):
    stub, errors, _ = await _run_session(["unknown_tool", "text"], **variant)
    assert errors == [UNADVERTISED_TEXT]


@pytest.mark.parametrize("variant", VARIANTS)
async def test_normal_text_answer_raises_no_error(variant):
    stub, errors, _ = await _run_session(["text"], **variant)
    assert errors == []


@pytest.mark.parametrize("variant", VARIANTS)
async def test_valid_tool_call_runs_unchanged_and_raises_no_error(variant):
    stub, errors, handler_calls = await _run_session(["valid_tool", "text"], **variant)
    assert errors == []
    assert handler_calls == ["call_1"]
    assert stub.second_request_tool_ids() == [("call", ["call_1"]), ("result", "call_1")]


@pytest.mark.parametrize("variant", VARIANTS)
async def test_empty_answer_after_a_tool_result_is_not_reported(variant):
    stub, errors, handler_calls = await _run_session(["valid_tool", "empty"], **variant)
    assert errors == []
    assert handler_calls == ["call_1"]


@pytest.mark.parametrize("variant", VARIANTS)
async def test_http_500_gives_one_error_not_two(variant):
    stub, errors, _ = await _run_session(["http500"], **variant)
    assert len(errors) == 1 and errors[0].startswith("Error during completion")


@pytest.mark.parametrize("variant", VARIANTS)
async def test_tool_call_without_id_gets_one_id_everywhere(variant):
    stub, errors, handler_calls = await _run_session(["no_id_tool", "text"], **variant)
    assert errors == []
    (generated,) = handler_calls
    assert generated.startswith("call_") and len(generated) == 37
    assert stub.second_request_tool_ids() == [("call", [generated]), ("result", generated)]


async def test_before_the_fix_the_same_failures_were_silent():
    """Without the guards (pre-change behaviour) nothing reaches the transport and the id stays None."""
    _, errors, _ = await _run_session(["empty"], wire_guard=False)
    assert errors == []
    _, errors, _ = await _run_session(["bad_args"], wire_guard=False)
    assert errors == []
    stub, errors, handler_calls = await _run_session(["no_id_tool", "text"], wire_guard=False)
    assert handler_calls == [None]
    assert stub.second_request_tool_ids() == [("call", [None]), ("result", None)]


async def test_legacy_idless_history_is_repaired_before_it_reaches_a_provider():
    messages = [
        {"role": "user", "content": "hi"},
        {"role": "assistant", "tool_calls": [{"type": "function", "function": {"name": "get_x", "arguments": "{}"}}]},
        {"role": "tool", "content": "{\"ok\": 1}"},
        {"role": "user", "content": "again"},
    ]
    result = filter_valid_messages(messages)
    assert result[1]["tool_calls"][0]["id"] == result[2]["tool_call_id"] == "legacy_call_1"
