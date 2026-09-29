"""Real-library guards for the pipecat 1.12 upgrade (design 2026-09-30, section 3).

Pins the upstream behaviors and private APIs the runner relies on, so the next
pipecat bump fails here instead of silently regressing a live call.

Lives outside scripts/pipecat/ for the same reason as the other files in this
directory (that conftest replaces pipecat with MagicMocks). Not run by CI. Run
manually in a venv with the pinned versions:

    pytest --noconftest bin-pipecat-manager/scripts/pipecat_realtest
"""
import asyncio
import http
import inspect
import os
import sys
import time
import warnings

import pytest

pytest.importorskip("pipecat.flows")

HERE = os.path.dirname(__file__)
sys.path.insert(0, os.path.abspath(os.path.join(HERE, "..", "pipecat")))
for k in ("DEEPGRAM_API_KEY", "CARTESIA_API_KEY", "ELEVENLABS_API_KEY", "GOOGLE_API_KEY", "OPENAI_API_KEY"):
    os.environ.setdefault(k, "x")
warnings.simplefilter("ignore")

import run  # noqa: E402
import team_flow, routing_llm, routing_stt, routing_tts, main, tools, task, message_filters, gemini_tool_filter  # noqa
import pipecat.services.deepgram.stt as dgmod  # noqa: E402
from pipecat.services.deepgram.stt import DeepgramSTTService, LiveOptions  # noqa: E402
from pipecat.utils.network import QuickFailureTracker  # noqa: E402


def test_all_modules_import_and_flows_repoint():
    import pipecat.flows as pf
    assert team_flow.FlowManager is pf.FlowManager
    assert "pipecat_flows" not in sys.modules


def test_settings_pins():
    dg = run.create_stt_service("deepgram", language="en")
    ca = run.create_tts_service("cartesia", voice_id="v")
    el = run.create_tts_service("elevenlabs", voice_id="v")
    go = run.create_tts_service("google", voice_id="en-US-Chirp3-HD-Charon")
    kw = dg._build_connect_kwargs()
    assert str(kw.get("profanity_filter")).lower() == "true", kw
    assert ca._settings.model == "sonic-3.5"
    assert el._settings.model == "eleven_turbo_v2_5"
    for s in (ca, el, go):
        assert s._max_consecutive_zero_audio_contexts == 0
    for llm_type in ("openai.gpt-4o", "grok.grok-3", "gemini.gemini-2.5-flash"):
        _, agg = run.create_llm_service(llm_type, "k", [], [])
        assert agg.user()._params.empty_user_turn is None
    g, _ = run.create_llm_service("gemini.gemini-2.5-flash", "k", [], [])
    assert g._stream_idle_timeout_secs is None


def test_deepgram_private_surface_guards():
    s = DeepgramSTTService(api_key="x")
    assert isinstance(s._quick_failure_tracker, QuickFailureTracker)
    assert hasattr(s._client.listen.v1, "connect") and hasattr(s, "_connect")
    src = inspect.getsource(DeepgramSTTService._connection_handler)
    assert "exponential_backoff_time(self._quick_failure_tracker.count)" in src
    setup_src = inspect.getsource(DeepgramSTTService.setup)
    body = [l.strip() for l in setup_src.splitlines() if l.strip() and not l.strip().startswith(('"""', "Args", "setup:", "Set up"))]
    assert body[-2:] == ["await super().setup(setup)", "await self._connect()"], body
    assert hasattr(type(s), "set_usable")
    # 1b rebind in place, delegating
    w = dgmod.exponential_backoff_time
    assert getattr(w, run._IMMEDIATE_FIRST_RETRY_MARK, None) is True
    from pipecat.utils.network import exponential_backoff_time as up
    assert w(0) == 0 and w(1) == up(1) and w(5) == up(5)
    run._install_deepgram_immediate_first_retry()
    assert dgmod.exponential_backoff_time is w
    hd = run._harden_deepgram(run._keep_usable(DeepgramSTTService(api_key="x")))
    assert hd._quick_failure_tracker.max_consecutive_failures == sys.maxsize


def test_async_tools_private_api():
    from pipecat.services.llm_service import LLMService
    assert hasattr(LLMService, "_has_async_tools")


def test_async_tools_override_keeps_system_instruction_clean():
    """2.5: a flows-style tool (cancel_on_interruption=False) must not add the
    ASYNC TOOLS block once the team override is applied."""
    from pipecat.services.google.llm import GoogleLLMService

    async def handler(params):
        pass

    def compose(override):
        svc = GoogleLLMService(api_key="x", model="gemini-2.5-flash", system_instruction="PROMPT_A")
        if override:
            svc._has_async_tools = lambda: False
        svc.register_function("to_b", handler, cancel_on_interruption=False)
        svc._compose_system_instruction()
        return svc._settings.system_instruction or ""

    assert "ASYNC TOOLS" in compose(False)  # upstream still composes it
    assert "ASYNC TOOLS" not in compose(True)
    assert compose(True).startswith("PROMPT_A")


def test_placeholder_regex_and_roundtrip():
    from pipecat.flows import manager as fm
    from types import SimpleNamespace
    assert team_flow._FLOW_PLACEHOLDER_RE.pattern == fm._PLACEHOLDER.pattern
    hist = [
        {"role": "user", "content": "hi {{ name }} and \\{{ x }} and \\\\{{ y }} {{ a.b }} {not} {{ 1x }}"},
        {"role": "user", "content": [{"type": "text", "text": "{{ z }}"}]},
    ]
    orig = [dict(m) for m in hist]
    esc = team_flow._escape_flow_placeholders(hist)
    assert hist == orig
    out = fm.FlowManager._render_node(SimpleNamespace(state={}), "n", {"task_messages": esc})
    assert out["task_messages"] == orig


def test_real_input_transport_class():
    tr = run.create_websocket_transport("input", "id")
    inp = tr.input()
    assert isinstance(inp, run.EarlyAudioBufferingInputTransport)
    assert inp._audio_task is None and inp._paused is False


async def _dg_reconnect(port, hold):
    import websockets
    from pipecat.frames.frames import InputAudioRawFrame, EndFrame
    from pipecat.pipeline.pipeline import Pipeline
    from pipecat.pipeline.task import PipelineTask, PipelineParams
    from pipecat.pipeline.runner import PipelineRunner
    hs = []
    t0 = time.monotonic()

    async def pr(conn, req):
        hs.append(time.monotonic() - t0)

    async def handler(ws):
        async def closer():
            await asyncio.sleep(hold)
            await ws.close(1011, "drop")
        if len(hs) == 1:
            asyncio.create_task(closer())
        try:
            async for _ in ws:
                pass
        except Exception:
            pass

    srv = await websockets.serve(handler, "127.0.0.1", port, process_request=pr)
    real_ctor = run.DeepgramSTTService
    run.DeepgramSTTService = lambda **kw: real_ctor(base_url=f"ws://127.0.0.1:{port}", **kw)
    try:
        s = run.create_stt_service("deepgram", language="en-US")
    finally:
        run.DeepgramSTTService = real_ctor
    task = PipelineTask(Pipeline([s]), params=PipelineParams(audio_out_sample_rate=16000), idle_timeout_secs=None)
    rt = asyncio.create_task(PipelineRunner(handle_sigint=False).run(task))
    while time.monotonic() - t0 < hold + 3:
        await task.queue_frame(InputAudioRawFrame(audio=b"\x00\x00" * 320, sample_rate=16000, num_channels=1))
        await asyncio.sleep(0.02)
    await task.queue_frame(EndFrame())
    await asyncio.wait_for(rt, 20)
    srv.close()
    return hs


def test_deepgram_immediate_reconnect_after_stable_drop():
    from loguru import logger
    logger.remove()
    hs = asyncio.run(_dg_reconnect(18931, 6.0))
    assert len(hs) >= 2, hs
    gap = hs[1] - 6.0 - hs[0]
    assert gap < 0.5, (hs, gap)


async def _dg_reject(port, status, dur):
    import websockets
    from pipecat.frames.frames import EndFrame, StartFrame
    from pipecat.pipeline.pipeline import Pipeline
    from pipecat.pipeline.task import PipelineTask, PipelineParams
    from pipecat.pipeline.runner import PipelineRunner
    hs = []
    t0 = time.monotonic()

    async def pr(conn, req):
        hs.append(time.monotonic() - t0)
        return conn.respond(http.HTTPStatus(status), "no\n")

    async def handler(ws):
        pass

    srv = await websockets.serve(handler, "127.0.0.1", port, process_request=pr)
    real_ctor = run.DeepgramSTTService
    run.DeepgramSTTService = lambda **kw: real_ctor(base_url=f"ws://127.0.0.1:{port}", **kw)
    try:
        s = run.create_stt_service("deepgram", language="en-US")
    finally:
        run.DeepgramSTTService = real_ctor
    task = PipelineTask(Pipeline([s]), params=PipelineParams(audio_out_sample_rate=16000), idle_timeout_secs=None)
    rt = asyncio.create_task(PipelineRunner(handle_sigint=False).run(task))
    await asyncio.sleep(dur)
    await task.queue_frame(EndFrame())
    await asyncio.wait_for(rt, 20)
    srv.close()
    return hs, s


@pytest.mark.parametrize("status,expect_retry", [(429, True), (401, False)])
def test_deepgram_handshake_status_policy(status, expect_retry):
    from loguru import logger
    logger.remove()
    hs, s = asyncio.run(_dg_reject(18940 + status % 100, status, 7.0))
    if expect_retry:
        assert len(hs) >= 2, hs
    else:
        assert len(hs) == 1, hs
    assert s._is_usable is True


def test_broadcast_service_metadata_upstream_shape():
    """2.5: the routers call broadcast_service_metadata() because 1.12 emits the
    metadata from AIService.push_frame, which their per-instance routing_push
    bypasses. A rename upstream would make the router's hasattr guard a silent
    no-op, so pin both halves here."""
    from pipecat.services.ai_service import AIService

    assert hasattr(AIService, "broadcast_service_metadata")
    assert "broadcast_service_metadata" in inspect.getsource(AIService.push_frame)


def test_team_routers_real_pipeline_start_and_metadata():
    """2.5: through real routers, StartFrame reaches downstream exactly once and
    every STT member's STTMetadataFrame arrives (0 without the router fix)."""
    import pipecat.frames.frames as F
    from pipecat.frames.frames import EndFrame, StartFrame
    from pipecat.pipeline.pipeline import Pipeline
    from pipecat.pipeline.runner import PipelineRunner
    from pipecat.pipeline.task import PipelineParams, PipelineTask
    from pipecat.processors.frame_processor import FrameProcessor
    from pipecat.services.openai.llm import OpenAILLMService
    from pipecat.services.stt_service import STTService

    class DummySTT(STTService):
        def __init__(self, t):
            super().__init__(ttfs_p99_latency=t)

        async def run_stt(self, audio):
            yield None

    class Tap(FrameProcessor):
        def __init__(self):
            super().__init__()
            self.starts = 0
            self.stt_meta = []

        async def process_frame(self, frame, direction):
            await super().process_frame(frame, direction)
            if isinstance(frame, StartFrame):
                self.starts += 1
            if isinstance(frame, F.STTMetadataFrame):
                self.stt_meta.append(frame.ttfs_p99_latency)
            await self.push_frame(frame, direction)

    async def go():
        stt = routing_stt.RoutingSTTService({"A": DummySTT(0.35), "B": DummySTT(1.57)})
        stt.set_active_member("A")
        llm = routing_llm.RoutingLLMService({
            "A": OpenAILLMService(api_key="x", model="gpt-4o"),
            "B": OpenAILLMService(api_key="x", model="gpt-4o-mini"),
        })
        llm.set_active_member("A")
        tap = Tap()
        task = PipelineTask(Pipeline([stt, llm, tap]), params=PipelineParams(audio_out_sample_rate=16000))

        async def stop():
            await asyncio.sleep(0.5)
            await task.queue_frame(EndFrame())

        asyncio.get_running_loop().create_task(stop())
        await PipelineRunner(handle_sigint=False).run(task)
        return tap

    from loguru import logger
    logger.remove()
    tap = asyncio.run(go())
    assert tap.starts == 1
    assert sorted(tap.stt_meta) == [0.35, 1.57]
