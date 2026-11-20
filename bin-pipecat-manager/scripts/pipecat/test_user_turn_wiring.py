"""VOIP-1547: vad_config / smart_turn_enabled reach the user aggregator.

Unit level wiring checks (pipecat is mocked by conftest.py). The real behavior of the
analyzers and strategies is pinned by scripts/pipecat_realtest/test_vad_turn_real.py.
"""
import ast
import os
import sys
from unittest.mock import MagicMock, patch

import pytest

import run
from test_init_pipeline import _run_team_init, _two_google_members


class _StopAfterLLM(Exception):
    """Raised by the patched create_llm_service to end init right after the call."""


class TestMakeAggregator:
    @patch("run.LLMUserAggregatorParams")
    @patch("run.LLMContextAggregatorPair")
    def test_no_analyzer_keeps_todays_params(self, pair, params):
        run._make_aggregator("ctx")
        params.assert_called_once_with(empty_user_turn=None)
        pair.assert_called_once_with("ctx", user_params=params.return_value)

    @patch("run.default_user_turn_start_strategies")
    @patch("run.UserTurnStrategies")
    @patch("run.SpeechTimeoutUserTurnStopStrategy")
    @patch("run.TurnAnalyzerUserTurnStopStrategy")
    @patch("run.LLMUserAggregatorParams")
    @patch("run.LLMContextAggregatorPair")
    def test_vad_without_turn_analyzer_uses_speech_timeout(
        self, pair, params, turn_stop, speech_stop, strategies, start
    ):
        vad = object()
        run._make_aggregator("ctx", vad, None)

        speech_stop.assert_called_once_with()
        turn_stop.assert_not_called()
        strategies.assert_called_once_with(start=start.return_value, stop=[speech_stop.return_value])
        params.assert_called_once_with(
            empty_user_turn=None, vad_analyzer=vad, user_turn_strategies=strategies.return_value
        )
        pair.assert_called_once_with("ctx", user_params=params.return_value)

    @patch("run.default_user_turn_start_strategies")
    @patch("run.UserTurnStrategies")
    @patch("run.SpeechTimeoutUserTurnStopStrategy")
    @patch("run.TurnAnalyzerUserTurnStopStrategy")
    @patch("run.LLMUserAggregatorParams")
    @patch("run.LLMContextAggregatorPair")
    def test_turn_analyzer_uses_turn_stop_strategy(
        self, pair, params, turn_stop, speech_stop, strategies, start
    ):
        vad, turn = object(), object()
        run._make_aggregator("ctx", vad, turn)

        turn_stop.assert_called_once_with(turn_analyzer=turn)
        speech_stop.assert_not_called()
        strategies.assert_called_once_with(start=start.return_value, stop=[turn_stop.return_value])
        assert params.call_args.kwargs["vad_analyzer"] is vad
        assert params.call_args.kwargs["empty_user_turn"] is None


class TestBuildUserTurnAnalyzers:
    @patch("run.build_vad_params")
    @patch("run.SileroVADAnalyzer")
    def test_smart_turn_off_has_no_turn_analyzer(self, silero, build):
        vad, turn = run.build_user_turn_analyzers({"stop_secs": 0.5}, False)

        build.assert_called_once_with({"stop_secs": 0.5}, smart_turn_enabled=False)
        silero.assert_called_once_with(params=build.return_value)
        assert vad is silero.return_value
        assert turn is None

    @patch("run.build_vad_params")
    @patch("run.SileroVADAnalyzer")
    def test_smart_turn_on_builds_a_fresh_turn_analyzer_each_time(self, silero, build):
        module = sys.modules["pipecat.audio.turn.smart_turn.local_smart_turn_v3"]
        with patch.object(module, "LocalSmartTurnAnalyzerV3", side_effect=lambda: object()) as cls:
            _, first = run.build_user_turn_analyzers(None, True)
            _, second = run.build_user_turn_analyzers(None, True)

        build.assert_called_with(None, smart_turn_enabled=True)
        assert cls.call_count == 2
        assert first is not second


class TestSingleAiPipeline:
    @pytest.mark.asyncio
    async def test_audio_call_builds_once_and_forwards_to_llm_only(self):
        vad, turn = object(), object()
        with patch("run.build_user_turn_analyzers", return_value=(vad, turn)) as build, \
             patch("run.create_stt_service"), \
             patch("run.create_websocket_transport") as transport, \
             patch("run.create_llm_service", side_effect=_StopAfterLLM()) as llm:
            with pytest.raises(_StopAfterLLM):
                await run.init_single_ai_pipeline(
                    id="pl", llm_type="openai.gpt-4o", llm_key="k", stt_type="deepgram",
                    vad_config={"confidence": 0.8}, smart_turn_enabled=True,
                )

        build.assert_called_once_with({"confidence": 0.8}, True)
        assert llm.call_args.kwargs["vad_analyzer"] is vad
        assert llm.call_args.kwargs["turn_analyzer"] is turn
        # the transport params have no such fields: nothing is passed to it
        assert [c.args for c in transport.call_args_list] == [("input", "pl"), ("output", "pl")]
        assert all(not c.kwargs for c in transport.call_args_list)

    @pytest.mark.asyncio
    async def test_text_only_call_builds_no_analyzers(self):
        with patch("run.build_user_turn_analyzers") as build, \
             patch("run.create_llm_service", side_effect=_StopAfterLLM()) as llm:
            with pytest.raises(_StopAfterLLM):
                await run.init_single_ai_pipeline(id="pl", llm_type="openai.gpt-4o", llm_key="k")

        build.assert_not_called()
        assert llm.call_args.kwargs["vad_analyzer"] is None
        assert llm.call_args.kwargs["turn_analyzer"] is None

    @pytest.mark.asyncio
    async def test_analyzer_build_failure_logs_init_error_and_raises(self):
        with patch("run.build_user_turn_analyzers", side_effect=RuntimeError("model missing")), \
             patch("run.create_stt_service") as stt, \
             patch("run.create_llm_service") as llm, \
             patch("run.logger") as log:
            with pytest.raises(RuntimeError, match="model missing"):
                await run.init_single_ai_pipeline(
                    id="pl", llm_type="openai.gpt-4o", llm_key="k", stt_type="deepgram"
                )

        stt.assert_not_called()
        llm.assert_not_called()
        assert any(
            "[INIT] Pipeline initialization failed" in str(c.args[0]) for c in log.error.call_args_list
        )


class TestTeamPipeline:
    @pytest.mark.asyncio
    async def test_audio_team_builds_analyzers_for_the_single_aggregator_only(self):
        team = {"id": "t", "start_member_id": "member-1", "members": _two_google_members()}
        vad, turn = object(), object()
        with patch("run.build_user_turn_analyzers", return_value=(vad, turn)) as build:
            _, mocks = await _run_team_init(
                team, stt_type="deepgram", tts_type=None, stt_language="en-US", tts_language="en-US",
                vad_config={"start_secs": 0.3}, smart_turn_enabled=True,
            )

        build.assert_called_once_with({"start_secs": 0.3}, True)
        mocks.make_aggregator.assert_called_once_with(mocks.LLMContext.return_value, vad, turn)
        for c in mocks.create_llm_service.call_args_list:
            assert "vad_analyzer" not in c.kwargs and "turn_analyzer" not in c.kwargs
        assert [c.args for c in mocks.create_websocket_transport.call_args_list] == [("input", "test-gate"), ("output", "test-gate")]

    @pytest.mark.asyncio
    async def test_team_without_audio_input_builds_no_analyzers(self):
        team = {"id": "t", "start_member_id": "member-1", "members": _two_google_members()}
        with patch("run.build_user_turn_analyzers") as build:
            _, mocks = await _run_team_init(team, stt_type=None, tts_type="elevenlabs")

        build.assert_not_called()
        mocks.make_aggregator.assert_called_once_with(mocks.LLMContext.return_value, None, None)


class TestStaticWiring:
    def _run_tree(self):
        return ast.parse(open(os.path.join(os.path.dirname(run.__file__), "run.py")).read())

    def test_every_aggregator_in_create_llm_service_gets_the_analyzers(self):
        fn = next(n for n in ast.walk(self._run_tree())
                  if isinstance(n, ast.FunctionDef) and n.name == "create_llm_service")
        calls = [n for n in ast.walk(fn)
                 if isinstance(n, ast.Call) and getattr(n.func, "id", "") == "_make_aggregator"]
        assert len(calls) == 7
        for c in calls:
            assert [ast.unparse(a) for a in c.args[1:]] == [
                "options.get('vad_analyzer')", "options.get('turn_analyzer')"
            ]

    def test_transport_params_no_longer_carry_ignored_fields(self):
        fn = next(n for n in ast.walk(self._run_tree())
                  if isinstance(n, ast.FunctionDef) and n.name == "create_websocket_transport")
        assert [a.arg for a in fn.args.args] == ["direction", "id"]
        params = next(n for n in ast.walk(fn)
                      if isinstance(n, ast.Call) and getattr(n.func, "id", "") == "WebsocketClientParams")
        names = {k.arg for k in params.keywords}
        assert "vad_analyzer" not in names and "turn_analyzer" not in names
        # VOIP-1548 is a separate ticket: session_timeout is left as it was
        assert "session_timeout" in names
