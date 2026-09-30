
"""New mocked tests from design section 3 (round 12 rehearsal #2)."""
import asyncio
import importlib
import sys
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

import run


class TestPreservedSettings:
    @patch("run.ElevenLabsTTSService")
    def test_elevenlabs_model_pinned(self, svc):
        run.create_tts_service("elevenlabs", voice_id="v")
        assert svc.call_args.kwargs["model"] == "eleven_turbo_v2_5"
        assert svc.call_args.kwargs["max_consecutive_zero_audio_contexts"] == 0

    @patch("run.GoogleTTSService")
    def test_google_tts_zero_audio_counter_off(self, svc):
        run.create_tts_service("google", voice_id="en-US-Chirp3-HD-Charon")
        assert svc.call_args.kwargs["max_consecutive_zero_audio_contexts"] == 0

    @patch("run._harden_deepgram", side_effect=lambda s: s)
    @patch("run.DeepgramSTTService")
    @patch("run.LiveOptions")
    def test_deepgram_profanity_pinned_and_hardened(self, lo, svc, harden):
        run.create_stt_service("deepgram", language="en")
        assert lo.call_args.kwargs["profanity_filter"] is True
        harden.assert_called_once_with(svc.return_value)

    @patch("run.LLMUserAggregatorParams")
    @patch("run.LLMContextAggregatorPair")
    def test_make_aggregator_disables_empty_user_turn(self, pair, params):
        run._make_aggregator("ctx")
        params.assert_called_once_with(empty_user_turn=None)
        pair.assert_called_once_with("ctx", user_params=params.return_value)

    @pytest.mark.parametrize("llm_type", ["openai.gpt-4o", "grok.grok-3", "gemini.gemini-2.5-flash"])
    def test_every_single_ai_path_uses_make_aggregator(self, llm_type):
        with patch("run.OpenAILLMService"), patch("run.GoogleLLMService"), patch("run.LLMContext"), \
             patch("run._make_aggregator") as mk:
            _, agg = run.create_llm_service(llm_type, "k", [], [])
        assert agg is mk.return_value

    @patch("run.LLMContextAggregatorPair")
    @patch("run.LLMContext")
    @patch("run.GoogleLLMService")
    def test_gemini_idle_timeout_disabled(self, svc, ctx, pair):
        run.create_llm_service("gemini.gemini-2.5-flash", "k", [], [])
        assert svc.call_args.kwargs["stream_idle_timeout_secs"] is None


class TestKeepUsable:
    def test_ignores_false_passes_true(self):
        svc = MagicMock()
        inner = AsyncMock()
        svc.set_usable = inner
        run._keep_usable(svc)
        asyncio.run(svc.set_usable(False))
        inner.assert_not_awaited()
        asyncio.run(svc.set_usable(True))
        inner.assert_awaited_once_with(True)

    @pytest.mark.parametrize("name,cls", [("cartesia", "CartesiaTTSService"), ("elevenlabs", "ElevenLabsTTSService"), ("google", "GoogleTTSService")])
    def test_every_tts_wrapped(self, name, cls):
        with patch(f"run.{cls}") as svc, patch("run._keep_usable", side_effect=lambda s: s) as ku:
            run.create_tts_service(name, voice_id="en-US-x")
            ku.assert_called_once_with(svc.return_value)

    @pytest.mark.parametrize("name,cls", [("deepgram", "DeepgramSTTService"), ("google", "GoogleSTTService")])
    def test_every_stt_wrapped(self, name, cls):
        with patch(f"run.{cls}") as svc, patch("run._harden_deepgram", side_effect=lambda s: s), \
             patch("run._keep_usable", side_effect=lambda s: s) as ku:
            run.create_stt_service(name)
            ku.assert_called_once_with(svc.return_value)


class _FakeCM:
    def __init__(self, exc=None):
        self.exc = exc

    async def __aenter__(self):
        if self.exc:
            raise self.exc
        return "conn"

    async def __aexit__(self, *a):
        return False


class _FakeDeepgram:
    """Small fake Deepgram class (doc section 3 technique)."""

    def __init__(self, connect_cm=None, connect_coro=None):
        inner = MagicMock(return_value=connect_cm or _FakeCM())
        self._client = SimpleNamespace(listen=SimpleNamespace(v1=SimpleNamespace(connect=inner)))
        self._quick_failure_tracker = object()
        self._connect_coro = connect_coro

    async def _connect(self):
        if self._connect_coro:
            await self._connect_coro()

    def create_task(self, coroutine, name=None):
        return asyncio.ensure_future(coroutine)


class _FakeParent:
    setup_calls = []

    async def setup(self, s):
        _FakeParent.setup_calls.append(s)


class _FakeDeepgramCls(_FakeParent):
    pass


class TestHardenDeepgram:
    def _svc(self, **kw):
        svc = _FakeDeepgram(**kw)
        return svc

    @pytest.mark.parametrize("status", [408, 429])
    def test_retryable_status_becomes_connection_error(self, status):
        ApiError = sys.modules["deepgram.core"].ApiError
        svc = run._harden_deepgram(self._svc(connect_cm=_FakeCM(ApiError(status_code=status))))

        async def go():
            async with svc._client.listen.v1.connect(model="nova-2"):
                pass
        with pytest.raises(ConnectionError):
            asyncio.run(go())

    def test_401_passes_through(self):
        ApiError = sys.modules["deepgram.core"].ApiError
        svc = run._harden_deepgram(self._svc(connect_cm=_FakeCM(ApiError(status_code=401))))

        async def go():
            async with svc._client.listen.v1.connect():
                pass
        with pytest.raises(ApiError):
            asyncio.run(go())

    def test_connect_forwards_kwargs(self):
        svc = self._svc()
        inner = svc._client.listen.v1.connect
        run._harden_deepgram(svc)

        async def go():
            async with svc._client.listen.v1.connect(model="nova-2") as c:
                return c
        assert asyncio.run(go()) == "conn"
        inner.assert_called_once_with(model="nova-2")

    def test_tracker_replaced(self):
        with patch("run.QuickFailureTracker") as qft:
            svc = run._harden_deepgram(self._svc())
        assert qft.call_args.kwargs["max_consecutive_failures"] == sys.maxsize
        assert svc._quick_failure_tracker is qft.return_value

    def _harden_with_fake_class(self, svc):
        # DeepgramSTTService is a MagicMock under conftest; patch a fake class
        # into run so super(DeepgramSTTService, svc) resolves to _FakeParent.
        svc.__class__ = type("_Svc", (_FakeDeepgramCls,), {
            "_connect": _FakeDeepgram._connect,
            "create_task": _FakeDeepgram.create_task,
        })
        return svc

    def test_setup_returns_when_connect_completes(self):
        done = {}

        async def c():
            done["ok"] = True
        svc = self._harden_with_fake_class(self._svc(connect_coro=c))
        _FakeParent.setup_calls.clear()
        with patch("run.DeepgramSTTService", _FakeDeepgramCls):
            run._harden_deepgram(svc)

            async def go():
                t0 = asyncio.get_running_loop().time()
                await svc.setup("s")
                return asyncio.get_running_loop().time() - t0
            elapsed = asyncio.run(go())
        assert done["ok"]
        assert elapsed < 0.5
        assert _FakeParent.setup_calls == ["s"]

    def test_setup_bounded_and_does_not_cancel(self):
        state = {"cancelled": False, "finished": False}

        async def c():
            try:
                await asyncio.sleep(0.2)
                state["finished"] = True
            except asyncio.CancelledError:
                state["cancelled"] = True
                raise
        svc = self._harden_with_fake_class(self._svc(connect_coro=c))
        with patch("run.DeepgramSTTService", _FakeDeepgramCls), patch("run._DEEPGRAM_SETUP_WAIT_SECS", 0.05):
            run._harden_deepgram(svc)

            async def go():
                t0 = asyncio.get_running_loop().time()
                await svc.setup("s")
                el = asyncio.get_running_loop().time() - t0
                assert not state["finished"]
                await asyncio.sleep(0.3)
                return el
            el = asyncio.run(go())
        assert el < 0.15
        assert not state["cancelled"] and state["finished"]


class TestDeepgramImmediateFirstRetry:
    """Design 2.5c (1b)."""

    def _mod(self):
        return importlib.import_module("pipecat.services.deepgram.stt")

    def test_wrapper_semantics_and_idempotent(self):
        mod = self._mod()
        saved = mod.exponential_backoff_time
        calls = []

        def fake(attempt, *a, **kw):
            calls.append(attempt)
            return 4 + attempt
        try:
            mod.exponential_backoff_time = fake
            run._install_deepgram_immediate_first_retry()
            w = mod.exponential_backoff_time
            assert w is not fake
            assert w(0) == 0 and calls == []
            assert w(1) == fake(1) and w(3) == fake(3)
            run._install_deepgram_immediate_first_retry()
            assert mod.exponential_backoff_time is w
        finally:
            mod.exponential_backoff_time = saved

    def test_installed_at_import_and_reimport_does_not_double_wrap(self):
        mod = self._mod()
        w = mod.exponential_backoff_time
        assert getattr(w, run._IMMEDIATE_FIRST_RETRY_MARK, None) is True
        importlib.reload(run)
        assert mod.exponential_backoff_time is w


class TestGeminiLoneSystem:
    def _ctx_messages(self, messages):
        with patch("run.GoogleLLMService"), patch("run.LLMContext") as ctx, \
             patch("run._make_aggregator"), patch("run.filter_valid_messages", side_effect=lambda m: list(m)):
            run.create_llm_service("gemini.gemini-2.5-flash", "k", messages, [])
        return ctx.call_args.kwargs["messages"]

    def test_lone_system_becomes_user(self):
        msgs = [{"role": "system", "content": "p"}]
        out = self._ctx_messages(msgs)
        assert out == [{"role": "user", "content": "p"}]
        assert msgs == [{"role": "system", "content": "p"}]

    @pytest.mark.parametrize("msgs", [
        [{"role": "system", "content": "p"}, {"role": "user", "content": "u"}],
        [{"role": "system", "content": "p"}, {"role": "assistant", "content": "a"}],
        [{"role": "system", "content": "p"}, {"role": "system", "content": "q"}],
    ])
    def test_others_unchanged(self, msgs):
        import copy
        snap = copy.deepcopy(msgs)
        assert self._ctx_messages(msgs) == snap
        assert msgs == snap

    def test_openai_lone_system_untouched(self):
        with patch("run.OpenAILLMService"), patch("run.LLMContext") as ctx, patch("run._make_aggregator"):
            run.create_llm_service("openai.gpt-4o", "k", [{"role": "system", "content": "p"}], [])
        assert ctx.call_args.kwargs["messages"][0]["role"] == "system"


class TestTransportClasses:
    def test_input_uses_buffering_transport(self):
        with patch("run.BufferingInputWebsocketClientTransport") as b:
            t = run.create_websocket_transport("input", "id")
        assert t is b.return_value

    def test_buffering_input_returns_early_audio_transport(self):
        import ast, os
        tree = ast.parse(open(os.path.join(os.path.dirname(__file__), "run.py")).read())
        for node in ast.walk(tree):
            if isinstance(node, ast.ClassDef) and node.name == "BufferingInputWebsocketClientTransport":
                assert [b.id for b in node.bases] == ["WebsocketClientTransport"]
                fn = [i for i in node.body if getattr(i, "name", "") == "input"]
                assert fn and "EarlyAudioBufferingInputTransport" in ast.dump(fn[0])
                return
        pytest.fail("class missing")


class TestEarlyAudioBuffering:
    def _inp(self, enabled=True):
        return run.EarlyAudioBufferingInputTransport(None, None, SimpleNamespace(audio_in_enabled=enabled))

    def test_replay_in_order_none_lost(self):
        inp = self._inp()

        async def go():
            for i in range(5):
                await inp.push_audio_frame(i)
            assert inp.pushed == []
            await inp.set_transport_ready("start")
            await inp.push_audio_frame(5)
        asyncio.run(go())
        assert inp.pushed == [0, 1, 2, 3, 4, 5]
        assert inp._early_audio == []

    def test_disabled_or_paused_not_held(self):
        inp = self._inp(enabled=False)
        asyncio.run(inp.push_audio_frame(1))
        assert inp._early_audio == []
        inp2 = self._inp()
        inp2._paused = True
        asyncio.run(inp2.push_audio_frame(1))
        assert inp2._early_audio == []

    def test_frames_during_replay_stay_behind(self):
        """Tripwire: frames pushed while set_transport_ready runs land after the replay."""
        inp = self._inp()

        async def go():
            for i in range(3):
                await inp.push_audio_frame(i)
            t = asyncio.ensure_future(inp.set_transport_ready("s"))
            await asyncio.sleep(0)
            await inp.push_audio_frame(99)
            await t
        asyncio.run(go())
        assert inp.pushed == [0, 1, 2, 99], inp.pushed
