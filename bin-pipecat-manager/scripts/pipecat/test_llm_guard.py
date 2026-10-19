"""Tests for llm_guard (VOIP-1633): EmptyOutputObserver and normalize_tool_call_ids.

conftest.py replaces pipecat with MagicMocks and registers a mock ``llm_guard``, so the real module is
loaded here with the genuine installed pipecat, under its own name, the way test_missing_tool_logging.py
loads tools.py. The mock entries are put back afterwards so the other test modules keep their mocks.
Local only: CI does not run these.
"""

import asyncio
import importlib.util
import os
import sys
from unittest.mock import MagicMock

import pytest


def _load_real_llm_guard():
    saved = {n: m for n, m in sys.modules.items() if n == "pipecat" or n.startswith("pipecat.")}
    for name in saved:
        del sys.modules[name]
    try:
        path = os.path.join(os.path.dirname(__file__), "llm_guard.py")
        spec = importlib.util.spec_from_file_location("_real_llm_guard_voip1633", path)
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        # the frame classes the module was bound to, for building test frames
        import pipecat.frames.frames as real_frames
        import pipecat.observers.base_observer as real_observer
        import pipecat.processors.aggregators.llm_context as real_context
        import pipecat.services.llm_service as real_llm_service

        return mod, real_frames, real_observer, real_context, real_llm_service
    finally:
        for name in list(sys.modules):
            if name == "pipecat" or name.startswith("pipecat."):
                del sys.modules[name]
        sys.modules.update(saved)


GUARD, F, OBS, CTX, LLMS = _load_real_llm_guard()


class _Source:
    """Stands in for the processor that pushed the End frame."""

    def __init__(self):
        self.errors = []

    async def push_error(self, error_msg, **kwargs):
        self.errors.append((error_msg, kwargs))


def _llm_processor():
    return MagicMock(spec=LLMS.LLMService)


def _stt_processor():
    return MagicMock()  # not an LLMService


def _push(observer, source, frame):
    data = OBS.FramePushed(
        source=source, destination=source, frame=frame, direction=MagicMock(), timestamp=0
    )
    return asyncio.run(observer.on_push_frame(data))


def _run(frames, observer=None, source=None):
    observer = observer or GUARD.EmptyOutputObserver("pcc-1")
    source = source or _Source()
    for frame in frames:
        _push(observer, source, frame)
    return observer, source


def _start():
    return F.LLMFullResponseStartFrame()


def _end():
    return F.LLMFullResponseEndFrame()


def _ctx_frame(last_role):
    ctx = CTX.LLMContext([{"role": "user", "content": "hi"}, {"role": last_role, "content": "x"}])
    return F.LLMContextFrame(context=ctx)


class TestEmptyOutputObserver:
    def test_empty_response_pushes_one_fixed_error(self):
        _, src = _run([_start(), _end()])
        assert src.errors == [("LLM returned no usable output", {})]

    def test_blank_text_is_empty(self):
        _, src = _run([_start(), F.LLMTextFrame("  \n"), _end()])
        assert len(src.errors) == 1

    def test_text_answer_is_not_reported(self):
        _, src = _run([_start(), F.LLMTextFrame("Hello"), _end()])
        assert src.errors == []

    def test_tool_call_is_not_reported(self):
        _, src = _run([_start(), F.FunctionCallsStartedFrame(function_calls=[]), _end()])
        assert src.errors == []

    def test_llm_error_in_the_response_is_not_reported_twice(self):
        err = F.ErrorFrame(error="Error during completion")
        err.processor = _llm_processor()
        _, src = _run([_start(), err, _end()])
        assert src.errors == []

    def test_stt_error_does_not_hide_an_empty_response(self):
        err = F.ErrorFrame(error="stt reconnect")
        err.processor = _stt_processor()
        _, src = _run([_start(), err, _end()])
        assert len(src.errors) == 1

    def test_error_without_processor_does_not_hide_an_empty_response(self):
        _, src = _run([_start(), F.ErrorFrame(error="x"), _end()])
        assert len(src.errors) == 1

    @pytest.mark.parametrize(
        "marker",
        [F.InterruptionFrame, F.CancelFrame, F.EndFrame, F.EagerEndOfTurnCancelFrame],
    )
    def test_interruption_and_session_end_are_not_reported(self, marker):
        _, src = _run([_start(), marker(), _end()])
        assert src.errors == []

    def test_transcription_text_does_not_count_as_an_answer(self):
        _, src = _run([_start(), F.TranscriptionFrame(text="hello", user_id="u", timestamp="t"), _end()])
        assert len(src.errors) == 1

    def test_empty_after_a_tool_result_is_log_only(self, monkeypatch):
        logged = []
        monkeypatch.setattr(GUARD.logger, "warning", lambda msg: logged.append(msg))
        _, src = _run([_ctx_frame("tool"), _start(), _end()])
        assert src.errors == []
        assert any("after_tool=true" in m and "pcc-1" in m for m in logged)

    def test_after_tool_flag_applies_to_one_response_only(self):
        obs, src = _run([_ctx_frame("tool"), _start(), _end(), _ctx_frame("user"), _start(), _end()])
        assert len(src.errors) == 1

    def test_unreadable_context_is_treated_as_not_after_tool(self):
        bad = F.LLMContextFrame(context=MagicMock(get_messages=MagicMock(side_effect=RuntimeError("x"))))
        _, src = _run([bad, _start(), _end()])
        assert len(src.errors) == 1

    def test_two_responses_are_judged_separately(self):
        obs, src = _run([_start(), F.LLMTextFrame("ok"), _end(), _start(), _end()])
        assert len(src.errors) == 1

    def test_end_without_start_is_ignored(self):
        _, src = _run([_end()])
        assert src.errors == []

    def test_output_before_start_does_not_count(self):
        _, src = _run([F.LLMTextFrame("late"), _start(), _end()])
        assert len(src.errors) == 1

    def test_error_pushed_by_the_observer_opens_nothing(self):
        obs, src = _run([_start(), _end()])
        own = F.ErrorFrame(error="LLM returned no usable output")
        own.processor = _llm_processor()
        _push(obs, src, own)
        _push(obs, src, _end())
        assert len(src.errors) == 1
        _, src2 = _run([_start(), F.LLMTextFrame("ok"), _end()], observer=obs, source=src)
        assert len(src.errors) == 1

    def test_error_log_has_only_the_pipecatcall_id(self, monkeypatch):
        logged = []
        monkeypatch.setattr(GUARD.logger, "error", lambda msg: logged.append(msg))
        _run([_start(), _end()])
        assert logged == ["[empty_completion] pipecatcall_id=pcc-1"]

    def test_an_exception_does_not_escape_and_later_detection_still_works(self, monkeypatch):
        logged = []
        monkeypatch.setattr(GUARD.logger, "error", lambda msg: logged.append(msg))

        class _Broken:
            async def push_error(self, **kwargs):
                raise RuntimeError("secret detail must not be logged")

        obs = GUARD.EmptyOutputObserver("pcc-1")
        _push(obs, _Broken(), _start())
        _push(obs, _Broken(), _end())
        assert any("observer failed" in m and "RuntimeError" in m for m in logged)
        assert all("secret detail" not in m for m in logged)
        _, src = _run([_start(), _end()], observer=obs)
        assert len(src.errors) == 1

    def test_error_is_never_fatal_or_permanent(self):
        _, src = _run([_start(), _end()])
        (msg, kwargs), = src.errors
        assert "fatal" not in kwargs and "force_treat_as_permanent" not in kwargs

    def test_observer_sees_each_frame_once(self):
        assert GUARD.EmptyOutputObserver("x").observe_every_push is False


class _Call:
    def __init__(self, name, call_id):
        self.function_name = name
        self.tool_call_id = call_id


class _Svc:
    def __init__(self):
        self.received = []

    async def run_function_calls(self, function_calls):
        self.received.append(function_calls)
        return "done"


def _wrapped(pcc="pcc-1"):
    svc = _Svc()
    GUARD.normalize_tool_call_ids(svc, pcc)
    return svc


class TestNormalizeToolCallIds:
    @pytest.mark.parametrize("empty", [None, ""])
    def test_empty_id_gets_a_generated_one(self, empty):
        svc = _wrapped()
        call = _Call("get_x", empty)
        assert asyncio.run(svc.run_function_calls([call])) == "done"
        assert call.tool_call_id.startswith("call_") and len(call.tool_call_id) == 37
        assert svc.received[0] == [call]

    def test_two_id_less_calls_get_different_ids(self):
        svc = _wrapped()
        a, b = _Call("a", None), _Call("b", "")
        asyncio.run(svc.run_function_calls([a, b]))
        assert a.tool_call_id != b.tool_call_id

    def test_normal_calls_pass_through_unchanged(self):
        svc = _wrapped()
        calls = [_Call("a", "call_1"), _Call("b", "call_2")]
        asyncio.run(svc.run_function_calls(calls))
        assert svc.received == [calls]
        assert svc.received[0] is calls
        assert [c.tool_call_id for c in calls] == ["call_1", "call_2"]

    def test_call_without_name_is_dropped_and_only_the_count_is_logged(self, monkeypatch):
        logged = []
        monkeypatch.setattr(GUARD.logger, "warning", lambda msg: logged.append(msg))
        svc = _wrapped()
        good, nameless = _Call("get_x", "call_1"), _Call("", "call_2")
        asyncio.run(svc.run_function_calls([nameless, good]))
        assert svc.received == [[good]]
        assert logged == ["[tool_call] dropped calls without a name. pipecatcall_id=pcc-1 count=1"]

    def test_all_dropped_still_calls_through_with_an_empty_list(self):
        svc = _wrapped()
        asyncio.run(svc.run_function_calls([_Call("", None)]))
        assert svc.received == [[]]

    def test_applying_twice_is_harmless(self):
        svc = _Svc()
        GUARD.normalize_tool_call_ids(svc)
        GUARD.normalize_tool_call_ids(svc)
        call = _Call("get_x", None)
        asyncio.run(svc.run_function_calls([call]))
        assert svc.received == [[call]]
        assert call.tool_call_id.startswith("call_")

    def test_unregistered_name_with_empty_id_and_nameless_call_never_reach_a_handler(self):
        """Nothing new becomes executable: the wrapper only fills an id or removes a call."""
        handler_calls = []

        class _Real:
            _functions = {"registered": lambda: handler_calls.append("registered")}

            async def run_function_calls(self, function_calls):
                for fc in function_calls:
                    if fc.function_name in self._functions:
                        self._functions[fc.function_name]()

        svc = _Real()
        GUARD.normalize_tool_call_ids(svc)
        asyncio.run(svc.run_function_calls([_Call("unregistered", None), _Call("", None)]))
        assert handler_calls == []
        asyncio.run(svc.run_function_calls([_Call("registered", None)]))
        assert handler_calls == ["registered"]
