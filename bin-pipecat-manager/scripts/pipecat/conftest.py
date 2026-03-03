"""Mock pipecat and other unavailable modules so routing tests can run without pipecat installed."""

import sys
import enum
from unittest.mock import MagicMock


# --- Real stub classes needed for isinstance() checks in routing services -----

class _Frame:
    pass

class _StartFrame(_Frame):
    pass

class _EndFrame(_Frame):
    pass

class _FrameDirection(enum.Enum):
    DOWNSTREAM = "downstream"
    UPSTREAM = "upstream"

class _FrameProcessor:
    """Minimal stub replicating the interface used by routing services."""
    def __init__(self, **kwargs):
        self.push_frame = self._default_push_frame

    def _check_started(self, frame):
        """Default: not started (matches pipecat before StartFrame is received)."""
        return False

    async def _default_push_frame(self, frame, direction=_FrameDirection.DOWNSTREAM):
        pass

    async def setup(self, setup):
        pass

    async def process_frame(self, frame, direction):
        pass


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


# Build mock module tree before anything imports pipecat.
# frames and frame_processor use real classes so isinstance() works in routing code.
_frames_module = _make_mock_module(
    "LLMRunFrame",
    Frame=_Frame,
    StartFrame=_StartFrame,
    EndFrame=_EndFrame,
)

_processor_module = _make_mock_module(
    FrameDirection=_FrameDirection,
    FrameProcessor=_FrameProcessor,
)

_mocks = {
    "pipecat": MagicMock(),
    "pipecat.services": MagicMock(),
    "pipecat.services.llm_service": _make_mock_module("FunctionCallParams"),
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
    "pipecat.processors.frame_processor": _processor_module,
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
    "pipecat.frames.frames": _frames_module,
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
    # aiohttp (used by team_flow.py)
    "aiohttp": _make_mock_module("ClientSession", "ClientTimeout"),
    # pipecat-flows
    "pipecat_flows": _make_mock_module("FlowManager", "FlowArgs", "FlowsFunctionSchema", "NodeConfig"),
    # local modules that may not exist in test env
    "common": MagicMock(PIPECATCALL_WS_URL="ws://localhost", PIPECATCALL_HTTP_URL="http://localhost", PIPELINE_SESSION_TIMEOUT=300),
    "tools": _make_mock_module("tool_register", "tool_unregister", "convert_to_openai_format", "convert_to_gemini_format", "get_tool_names", "tool_execute"),
    "task": _make_mock_module("task_manager"),
}

for mod_name, mock_obj in _mocks.items():
    if mod_name not in sys.modules:
        sys.modules[mod_name] = mock_obj
