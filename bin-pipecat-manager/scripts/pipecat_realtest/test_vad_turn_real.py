"""Real-library guards for VOIP-1547 (vad_config / smart_turn_enabled reach the user aggregator).

In pipecat 1.x the transport params have no vad/turn fields (pydantic drops them
without an error); VAD and the turn strategies live in the user aggregator. These
tests pin that premise and the runner's wiring against the real library, so a
future bump or refactor fails here instead of silently dropping the customer's
settings again.

Lives outside scripts/pipecat/ for the same reason as the other files in this
directory (that conftest replaces pipecat with MagicMocks). Not run by CI. Run
manually in a venv with the pinned versions:

    pytest --noconftest bin-pipecat-manager/scripts/pipecat_realtest/test_vad_turn_real.py

The behavior test at the end needs a 16 kHz mono wav with one spoken utterance
that ends in a clear pause (not committed, a binary). Without it the test skips:

    VOIP1547_SPEECH_WAV=/path/to/speech16k.wav pytest --noconftest ...
"""
import asyncio
import os
import sys
import time
import wave
import warnings

import pytest

pytest.importorskip("pipecat.flows")

HERE = os.path.dirname(__file__)
sys.path.insert(0, os.path.abspath(os.path.join(HERE, "..", "pipecat")))
warnings.simplefilter("ignore")

import run  # noqa: E402
from pipecat.processors.aggregators.llm_context import LLMContext  # noqa: E402
from pipecat.transports.base_transport import TransportParams  # noqa: E402
from pipecat.transports.websocket.client import WebsocketClientParams  # noqa: E402
from pipecat.turns.user_start import TranscriptionUserTurnStartStrategy, VADUserTurnStartStrategy  # noqa: E402
from pipecat.turns.user_stop import SpeechTimeoutUserTurnStopStrategy, TurnAnalyzerUserTurnStopStrategy  # noqa: E402


def _user_aggregator(vad_config=None, smart_turn_enabled=False):
    vad, turn = run.build_user_turn_analyzers(vad_config, smart_turn_enabled)
    agg = run._make_aggregator(LLMContext(messages=[]), vad, turn).user()
    return agg, vad, turn


def test_premise_transport_params_have_no_vad_or_turn_fields():
    # If a future pipecat adds these fields, this fails: revisit run.py (the aggregator
    # wiring may then be redundant or double) instead of assuming it still works.
    for name in ("vad_analyzer", "turn_analyzer"):
        assert name not in TransportParams.model_fields
        assert name not in WebsocketClientParams.model_fields
    params = WebsocketClientParams(vad_analyzer=object(), turn_analyzer=object())
    assert not hasattr(params, "vad_analyzer") and not hasattr(params, "turn_analyzer")


def test_create_websocket_transport_passes_no_vad_or_turn():
    t = run.create_websocket_transport("input", "id")
    assert not hasattr(t._params, "vad_analyzer")


def test_aggregator_with_smart_turn_off_uses_vad_and_speech_timeout():
    agg, vad, turn = _user_aggregator({"confidence": 0.8, "start_secs": 0.3, "stop_secs": 0.5}, False)
    p = agg._params
    assert turn is None
    assert p.vad_analyzer is vad
    assert p.empty_user_turn is None  # 1.4.x parity pin (upgrade design 2.3)
    assert [type(s) for s in p.user_turn_strategies.start] == [VADUserTurnStartStrategy, TranscriptionUserTurnStartStrategy]
    assert [type(s) for s in p.user_turn_strategies.stop] == [SpeechTimeoutUserTurnStopStrategy]
    assert agg._user_turn_controller.user_turn_strategies is p.user_turn_strategies
    assert agg._vad_controller is not None  # private attribute, pinned to 1.12 on purpose
    # vad_config reaches the real analyzer
    assert (vad.params.confidence, vad.params.start_secs, vad.params.stop_secs) == (0.8, 0.3, 0.5)


def test_aggregator_with_smart_turn_on_uses_turn_analyzer_and_forces_stop_secs():
    agg, vad, turn = _user_aggregator({"stop_secs": 0.5, "min_volume": 0.4}, True)
    p = agg._params
    assert turn is not None
    assert [type(s) for s in p.user_turn_strategies.stop] == [TurnAnalyzerUserTurnStopStrategy]
    assert p.user_turn_strategies.stop[0]._turn_analyzer is turn  # private attribute, pinned to 1.12
    assert vad.params.stop_secs == 0.2  # smart turn requires 0.2, whatever the customer set
    assert vad.params.min_volume == 0.4


def test_sessions_do_not_share_analyzers():
    a, b = run.build_user_turn_analyzers(None, True), run.build_user_turn_analyzers(None, True)
    assert a[0] is not b[0] and a[1] is not b[1]


def test_text_only_aggregator_is_unchanged():
    agg = run._make_aggregator(LLMContext(messages=[])).user()
    assert agg._params.vad_analyzer is None
    assert agg._params.empty_user_turn is None
    assert agg._vad_controller is None


# --- behavior with the real Silero and smart-turn models ----------------------------

SPEECH_WAV = os.environ.get("VOIP1547_SPEECH_WAV")


async def _run_turns(smart_turn_enabled):
    """Feed the clip in real time; STT is simulated as Deepgram (first interim 1.2 s after
    the start, a final segment every 2.5 s, last final 0.3 s after the end of speech).
    Returns (events, speech_end_time); events are (seconds from start, frame name)."""
    from pipecat.frames.frames import (
        BotStartedSpeakingFrame, InputAudioRawFrame, InterimTranscriptionFrame,
        STTMetadataFrame, TranscriptionFrame,
    )
    from pipecat.pipeline.pipeline import Pipeline
    from pipecat.pipeline.runner import PipelineRunner
    from pipecat.pipeline.task import PipelineParams, PipelineTask
    from pipecat.processors.frame_processor import FrameProcessor

    vad, turn = run.build_user_turn_analyzers(None, smart_turn_enabled)
    pair = run._make_aggregator(LLMContext(messages=[{"role": "system", "content": "x"}]), vad, turn)

    names = {"UserStartedSpeakingFrame", "UserStoppedSpeakingFrame", "InterruptionFrame", "LLMContextFrame"}
    events = []

    class Sink(FrameProcessor):
        async def process_frame(self, frame, direction):
            await super().process_frame(frame, direction)
            if type(frame).__name__ in names:
                events.append((time.monotonic(), type(frame).__name__))
            await self.push_frame(frame, direction)

    task = PipelineTask(
        Pipeline([pair.user(), Sink()]),
        params=PipelineParams(audio_in_sample_rate=16000, audio_out_sample_rate=16000),
    )
    w = wave.open(SPEECH_WAV)
    assert (w.getframerate(), w.getnchannels(), w.getsampwidth()) == (16000, 1, 2)
    pcm = w.readframes(w.getnframes())
    dur = len(pcm) / 32000
    marks = {}

    async def stt():
        await asyncio.sleep(1.2)
        await task.queue_frame(InterimTranscriptionFrame("hi", "u", "now"))
        now = 1.2
        for t in (2.5, 5.0):
            if t < dur - 1.0:
                await asyncio.sleep(t - now)
                now = t
                await task.queue_frame(TranscriptionFrame("seg", "u", "now", finalized=True))
        await asyncio.sleep(dur + 0.3 - now)
        await task.queue_frame(TranscriptionFrame("end", "u", "now", finalized=True))

    async def feed():
        await asyncio.sleep(0.3)
        await task.queue_frame(STTMetadataFrame(service_name="deepgram", ttfs_p99_latency=0.35))
        await task.queue_frame(BotStartedSpeakingFrame())
        await asyncio.sleep(0.2)
        marks["start"] = time.monotonic()
        stt_task = asyncio.create_task(stt())
        for i in range(0, len(pcm), 640):
            await task.queue_frame(InputAudioRawFrame(pcm[i:i + 640], 16000, 1))
            await asyncio.sleep(0.02)
        marks["end"] = time.monotonic()
        for _ in range(int(7 / 0.02)):  # trailing silence
            await task.queue_frame(InputAudioRawFrame(b"\x00" * 640, 16000, 1))
            await asyncio.sleep(0.02)
        await stt_task
        await task.cancel()

    feeder = asyncio.create_task(feed())
    await PipelineRunner(handle_sigint=False).run(task)
    feeder.cancel()
    return [(t - marks["start"], n) for t, n in events], marks["end"] - marks["start"]


@pytest.mark.skipif(not SPEECH_WAV, reason="set VOIP1547_SPEECH_WAV to a 16 kHz mono speech clip")
@pytest.mark.parametrize("smart_turn_enabled", [False, True])
def test_one_utterance_is_one_turn_and_barge_in_comes_from_vad(smart_turn_enabled):
    events, speech_end = asyncio.run(_run_turns(smart_turn_enabled))
    print(smart_turn_enabled, [(round(t, 2), n) for t, n in events], round(speech_end, 2))
    started = [t for t, n in events if n == "UserStartedSpeakingFrame"]
    interrupts = [t for t, n in events if n == "InterruptionFrame"]
    runs = [t for t, n in events if n == "LLMContextFrame"]
    # barge-in at VAD speech start, before the first simulated transcript (1.2 s)
    assert started and started[0] < 1.1
    assert interrupts and interrupts[0] < 1.1
    # the utterance is one turn (without the VAD the STT finals cut it into several)
    assert len(started) == 1 and len(runs) == 1
    # the LLM runs after the end of speech, not in the middle of it
    assert runs[0] > speech_end
