"""Mock pipecat and other unavailable modules so run.py can be imported in tests."""

import sys
import enum
from unittest.mock import MagicMock


class _Language(enum.Enum):
    """Minimal Language enum mirroring pipecat.transcriptions.language.Language."""
    EN_US = "en-US"
    FR_FR = "fr-FR"
    DE_DE = "de-DE"
    ES_ES = "es-ES"
    JA_JP = "ja-JP"
    KO_KR = "ko-KR"
    ZH_CN = "zh-CN"


def _make_mock_module(*attrs, **kw_attrs):
    m = MagicMock()
    for a in attrs:
        setattr(m, a, MagicMock())
    for k, v in kw_attrs.items():
        setattr(m, k, v)
    return m


# Build mock module tree before anything imports pipecat
_mocks = {
    "pipecat": MagicMock(),
    "pipecat.services": MagicMock(),
    "pipecat.services.cartesia": MagicMock(),
    "pipecat.services.cartesia.tts": _make_mock_module("CartesiaTTSService"),
    "pipecat.services.elevenlabs": MagicMock(),
    "pipecat.services.elevenlabs.tts": _make_mock_module("ElevenLabsTTSService"),
    "pipecat.services.google": MagicMock(),
    "pipecat.services.google.tts": _make_mock_module("GoogleTTSService"),
    "pipecat.services.google.stt": _make_mock_module("GoogleSTTService"),
    "pipecat.services.google.llm": _make_mock_module("GoogleLLMService"),
    "pipecat.services.deepgram": MagicMock(),
    "pipecat.services.deepgram.stt": _make_mock_module("DeepgramSTTService"),
    "pipecat.services.whisper": MagicMock(),
    "pipecat.services.whisper.stt": _make_mock_module("Model", "WhisperSTTService"),
    "pipecat.services.openai": MagicMock(),
    "pipecat.services.openai.llm": _make_mock_module("OpenAILLMService"),
    "pipecat.transcriptions": MagicMock(),
    "pipecat.transcriptions.language": _make_mock_module(Language=_Language),
    "pipecat.processors": MagicMock(),
    "pipecat.processors.frame_processor": _make_mock_module("FrameDirection", "FrameProcessor"),
    "pipecat.processors.filters": MagicMock(),
    "pipecat.processors.filters.stt_mute_filter": _make_mock_module(
        "STTMuteConfig", "STTMuteFilter", "STTMuteStrategy"
    ),
    "pipecat.processors.aggregators": MagicMock(),
    "pipecat.processors.aggregators.openai_llm_context": _make_mock_module("OpenAILLMContext"),
    "pipecat.audio": MagicMock(),
    "pipecat.audio.vad": MagicMock(),
    "pipecat.audio.vad.silero": _make_mock_module("SileroVADAnalyzer"),
    "pipecat.audio.vad.vad_analyzer": _make_mock_module("VADParams"),
    "pipecat.frames": MagicMock(),
    "pipecat.frames.frames": _make_mock_module("LLMRunFrame"),
    "pipecat.pipeline": MagicMock(),
    "pipecat.pipeline.pipeline": _make_mock_module("Pipeline"),
    "pipecat.pipeline.runner": _make_mock_module("PipelineRunner"),
    "pipecat.pipeline.task": _make_mock_module("PipelineParams", "PipelineTask"),
    "pipecat.serializers": MagicMock(),
    "pipecat.serializers.protobuf": _make_mock_module("ProtobufFrameSerializer"),
    "pipecat.transports": MagicMock(),
    "pipecat.transports.websocket": MagicMock(),
    "pipecat.transports.websocket.client": _make_mock_module(
        "WebsocketClientParams", "WebsocketClientTransport"
    ),
    # deepgram SDK
    "deepgram": _make_mock_module("LiveOptions"),
    # pipecat-flows
    "pipecat_flows": _make_mock_module("FlowManager", "FlowArgs", "FlowsFunctionSchema", "NodeConfig"),
    # local modules that may not exist in test env
    "common": MagicMock(PIPECATCALL_WS_URL="ws://localhost", PIPECATCALL_HTTP_URL="http://localhost", PIPELINE_SESSION_TIMEOUT=300),
    "tools": _make_mock_module("tool_register", "tool_unregister", "convert_to_openai_format", "get_tool_names"),
    "task": _make_mock_module("task_manager"),
}

for mod_name, mock_obj in _mocks.items():
    if mod_name not in sys.modules:
        sys.modules[mod_name] = mock_obj
