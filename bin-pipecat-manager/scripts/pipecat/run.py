import asyncio
import importlib
import os
import sys
import json
import common
import time

from loguru import logger
from functools import partial

# tts
from pipecat.services.cartesia.tts import CartesiaTTSService
from pipecat.services.elevenlabs.tts import ElevenLabsTTSService
from pipecat.services.google.tts import GoogleTTSService

# stt
from pipecat.services.deepgram.stt import DeepgramSTTService, LiveOptions
from pipecat.services.google.stt import GoogleSTTService
from pipecat.transcriptions.language import Language

# llm
from pipecat.services.openai.llm import OpenAILLMService
from pipecat.services.openrouter.llm import OpenRouterLLMService
from pipecat.services.google.llm import GoogleLLMService

# aggregators / context
from pipecat.processors.aggregators.llm_context import LLMContext, NOT_GIVEN
from pipecat.processors.aggregators.llm_response_universal import (
    LLMContextAggregatorPair,
    LLMUserAggregatorParams,
)
from pipecat.adapters.schemas.function_schema import FunctionSchema
from pipecat.adapters.schemas.tools_schema import ToolsSchema

# pipeline
from pipecat.audio.vad.silero import SileroVADAnalyzer
from pipecat.audio.vad.vad_analyzer import VADParams
from pipecat.frames.frames import InterruptionFrame, LLMRunFrame, TextFrame
from pipecat.pipeline.pipeline import Pipeline
from pipecat.pipeline.runner import PipelineRunner
from pipecat.pipeline.task import PipelineParams, PipelineTask
from pipecat.serializers.protobuf import ProtobufFrameSerializer
from pipecat.utils.network import QuickFailureTracker
from deepgram.core import ApiError
from pipecat.transports.websocket.client import (
    WebsocketClientInputTransport,
    WebsocketClientOutputTransport,
    WebsocketClientParams,
    WebsocketClientTransport,
)

from message_filters import filter_valid_messages
from gemini_tool_filter import drop_gemini_invalid_tools
from tools import tool_register, tool_unregister, convert_to_openai_format, get_tool_names, register_missing_tool_logging
from task import task_manager
from routing_llm import RoutingLLMService
from routing_tts import RoutingTTSService
from routing_stt import RoutingSTTService
from team_flow import build_team_flow
from pipecat.flows import FlowManager


def _make_aggregator(ctx):
    """Build every LLMContextAggregatorPair the same way (design 2.3).

    pipecat 1.12 turns on empty-user-turn recovery by default; pin it off so
    the upgrade is audibly identical to 1.4.x.
    """
    return LLMContextAggregatorPair(
        ctx, user_params=LLMUserAggregatorParams(empty_user_turn=None)
    )


def _keep_usable(svc):
    """Ignore set_usable(False) so a transient provider failure cannot leave the
    call permanently mute or deaf (design 2.5c). set_usable(True) passes through."""
    original = svc.set_usable

    async def set_usable(is_usable):
        if not is_usable:
            logger.warning(f"{svc}: ignoring set_usable(False) to keep the service usable (design 2.5c)")
            return
        await original(is_usable)

    svc.set_usable = set_usable
    return svc


_DEEPGRAM_SETUP_WAIT_SECS = 2.0
_DEEPGRAM_RETRYABLE_HANDSHAKE_STATUS = (408, 429)


class _RetryableHandshake:
    """Async context manager proxy: 408/429 ApiError at the handshake -> ConnectionError."""

    def __init__(self, cm):
        self._cm = cm

    async def __aenter__(self):
        try:
            return await self._cm.__aenter__()
        except ApiError as e:
            if getattr(e, "status_code", None) in _DEEPGRAM_RETRYABLE_HANDSHAKE_STATUS:
                raise ConnectionError(f"Deepgram transient rejection (status {e.status_code}): {e}") from e
            raise

    async def __aexit__(self, *exc):
        return await self._cm.__aexit__(*exc)


def _harden_deepgram(svc):
    """Restore 1.4.0 retry behavior on a Deepgram STT instance (design 2.5c)."""
    # (1) never give up on quick failures
    svc._quick_failure_tracker = QuickFailureTracker(max_consecutive_failures=sys.maxsize)

    # (2) 408/429 at the handshake retry instead of giving up
    listen_v1 = svc._client.listen.v1
    original_connect = listen_v1.connect

    def connect(*args, **kwargs):
        return _RetryableHandshake(original_connect(*args, **kwargs))

    listen_v1.connect = connect

    # (3) bounded setup wait
    async def setup(setup_arg):
        await super(DeepgramSTTService, svc).setup(setup_arg)
        # Owned by the service's task manager so it is cancelled with the
        # pipeline. Not cancelled on timeout: it keeps retrying in the background.
        task = svc.create_task(svc._connect(), "initial_connect")
        await asyncio.wait({task}, timeout=_DEEPGRAM_SETUP_WAIT_SECS)

    svc.setup = setup
    return svc


_IMMEDIATE_FIRST_RETRY_MARK = "_voipbin_immediate_first_retry"


def _install_deepgram_immediate_first_retry():
    """(1b) Reconnect immediately after a stable Deepgram connection drops.

    Rebinds only pipecat.services.deepgram.stt's module-level
    exponential_backoff_time: attempt 0 -> 0, otherwise delegate. Idempotent.
    """
    mod = importlib.import_module("pipecat.services.deepgram.stt")
    current = mod.exponential_backoff_time
    if getattr(current, _IMMEDIATE_FIRST_RETRY_MARK, None) is True:
        return

    def exponential_backoff_time(attempt, *args, **kwargs):
        if attempt == 0:
            return 0
        return current(attempt, *args, **kwargs)

    setattr(exponential_backoff_time, _IMMEDIATE_FIRST_RETRY_MARK, True)
    exponential_backoff_time.__wrapped__ = current
    mod.exponential_backoff_time = exponential_backoff_time


_install_deepgram_immediate_first_retry()


def build_vad_params(vad_config: dict | None, smart_turn_enabled: bool = False) -> VADParams:
    """Build VADParams from config dict. None/empty = Pipecat defaults.
    When smart_turn_enabled is True, forces stop_secs=0.2 for optimal turn detection.
    """
    if not vad_config:
        vad_config = {}

    kwargs = {}
    if vad_config.get("confidence") is not None:
        kwargs["confidence"] = vad_config["confidence"]
    if vad_config.get("start_secs") is not None:
        kwargs["start_secs"] = vad_config["start_secs"]

    # Smart turn requires stop_secs=0.2 (matches training data)
    if smart_turn_enabled:
        if vad_config.get("stop_secs") is not None and vad_config["stop_secs"] != 0.2:
            logger.warning(f"smart_turn_enabled: overriding vad_config stop_secs={vad_config['stop_secs']} to 0.2")
        kwargs["stop_secs"] = 0.2
    elif vad_config.get("stop_secs") is not None:
        kwargs["stop_secs"] = vad_config["stop_secs"]

    if vad_config.get("min_volume") is not None:
        kwargs["min_volume"] = vad_config["min_volume"]

    return VADParams(**kwargs)


async def init_pipeline(
    id: str,
    llm_type: str,
    llm_key: str,
    llm_messages: list = None,
    stt_type: str = None,
    stt_language: str = None,
    tts_type: str = None,
    tts_language: str = None,
    tts_voice_id: str = None,
    tools_data: list = None,
    resolved_team: dict = None,
    vad_config: dict = None,
    smart_turn_enabled: bool = False,
) -> dict:
    """Initialize the pipeline. Returns context dict. Raises on failure."""
    if resolved_team:
        ctx = await init_team_pipeline(
            id, resolved_team,
            stt_language=stt_language,
            tts_language=tts_language,
            stt_type=stt_type,
            tts_type=tts_type,
            llm_messages=llm_messages,
            vad_config=vad_config,
            smart_turn_enabled=smart_turn_enabled,
        )
        ctx["type"] = "team"
        return ctx
    else:
        ctx = await init_single_ai_pipeline(
            id, llm_type, llm_key, llm_messages,
            stt_type, stt_language, tts_type, tts_language,
            tts_voice_id, tools_data,
            vad_config=vad_config,
            smart_turn_enabled=smart_turn_enabled,
        )
        ctx["type"] = "single"
        return ctx


async def execute_pipeline(id: str, ctx: dict):
    """Execute the pipeline loop. Runs as background task."""
    if ctx["type"] == "team":
        await execute_team_pipeline(id, ctx)
    else:
        await execute_single_ai_pipeline(id, ctx)


async def init_single_ai_pipeline(
    id: str,
    llm_type: str,
    llm_key: str,
    llm_messages: list = None,
    stt_type: str = None,
    stt_language: str = None,
    tts_type: str = None,
    tts_language: str = None,
    tts_voice_id: str = None,
    tools_data: list = None,
    vad_config: dict = None,
    smart_turn_enabled: bool = False,
) -> dict:
    """Initialize single AI pipeline. Returns context dict. Raises on failure."""
    total_start = time.monotonic()
    logger.info(f"[INIT] Starting Pipecat client pipeline id={id}")

    if llm_messages is None:
        llm_messages = []

    if tools_data is None:
        tools_data = []
    openai_tools = convert_to_openai_format(tools_data)
    tool_names = get_tool_names(tools_data)
    logger.info(f"[INIT] Received {len(tool_names)} tools: {tool_names}")

    init_tasks = {}

    if stt_type:
        async def init_stt_and_input_ws():
            start = time.monotonic()
            stt_service = create_stt_service(stt_type, language=stt_language)
            vad_analyzer = SileroVADAnalyzer(params=build_vad_params(vad_config, smart_turn_enabled=smart_turn_enabled))
            turn_analyzer = None
            if smart_turn_enabled:
                from pipecat.audio.turn.smart_turn.local_smart_turn_v3 import LocalSmartTurnAnalyzerV3
                turn_analyzer = LocalSmartTurnAnalyzerV3()
            transport = create_websocket_transport("input", id, vad_analyzer=vad_analyzer, turn_analyzer=turn_analyzer)
            logger.info(f"[INIT][stt+ws_input] done in {time.monotonic() - start:.3f} sec. pipeline id={id}")
            return {
                "stt_service": stt_service,
                "transport_input": transport,
                "vad_analyzer": vad_analyzer,
            }
        init_tasks["stt_input"] = asyncio.create_task(init_stt_and_input_ws())

    if tts_type:
        async def init_tts():
            start = time.monotonic()
            tts_service = create_tts_service(tts_type, voice_id=tts_voice_id, language=tts_language)
            logger.info(f"[INIT][tts] done in {time.monotonic() - start:.3f} sec. pipeline id={id}")
            return {
                "tts_service": tts_service,
            }
        init_tasks["tts"] = asyncio.create_task(init_tts())

    async def init_llm():
        start = time.monotonic()
        llm_service, aggregator = create_llm_service(llm_type, llm_key, llm_messages, openai_tools, pipeline_id=id)
        logger.info(f"[INIT][llm] done in {time.monotonic() - start:.3f} sec. pipeline id={id}")
        return {
            "llm_service": llm_service,
            "llm_context_aggregator": aggregator,
        }
    init_tasks["llm"] = asyncio.create_task(init_llm())

    async def init_output_ws():
        start = time.monotonic()
        transport = create_websocket_transport("output", id, vad_analyzer=None)
        logger.info(f"[INIT][ws_output] done in {time.monotonic() - start:.3f} sec. pipeline id={id}")
        return {
            "transport_output": transport,
        }
    init_tasks["ws_output"] = asyncio.create_task(init_output_ws())

    # Await all init tasks
    try:
        results_list = await asyncio.gather(*init_tasks.values())
    except Exception as e:
        logger.error(f"[INIT] Pipeline initialization failed: {e}")
        for t in init_tasks.values():
            if not t.done():
                t.cancel()
        raise
    logger.info(f"[INIT] All components initialized in {time.monotonic() - total_start:.3f} sec. pipeline id={id}")

    results = {}
    for part in results_list:
        results.update(part)

    stt_service = results.get("stt_service")
    transport_input = results.get("transport_input")
    tts_service = results.get("tts_service")
    llm_service = results["llm_service"]
    llm_context_aggregator = results["llm_context_aggregator"]
    transport_output = results["transport_output"]

    # Assemble pipeline stages
    pipeline_stages = []
    if transport_input:
        pipeline_stages.append(transport_input.input())
        pipeline_stages.append(stt_service)
    pipeline_stages.append(llm_context_aggregator.user())
    pipeline_stages.append(llm_service)
    if tts_service:
        pipeline_stages.append(tts_service)
    pipeline_stages.append(llm_context_aggregator.assistant())
    pipeline_stages.append(transport_output.output())

    pipeline = Pipeline(pipeline_stages)

    # Create Pipeline Task
    task_start = time.monotonic()
    task = PipelineTask(
        pipeline,
        params=PipelineParams(
            audio_out_sample_rate=16000,
            enable_metrics=True,
            enable_usage_metrics=True,
        ),
    )

    await task_manager.add(id, task)
    logger.info(f"[INIT][task_create] done in {time.monotonic() - task_start:.3f} sec. pipeline id={id}")

    try:
        # Register tools (after task_manager.add so cleanup-on-failure can unregister)
        tool_register(llm_service, id, tool_names, tools_data=tools_data)
        # VOIP-1512: surface missing/unadvertised tool calls as a structured ERROR.
        register_missing_tool_logging(llm_service, id)

        async def handle_disconnect_or_error(name, transport, error=None):
            logger.error(f"{name} WebSocket disconnected or errored: {error}. pipeline id={id}")
            await task.cancel()

        if transport_input:
            transport_input.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Input"))
            transport_input.event_handler("on_error")(partial(handle_disconnect_or_error, "Input"))
        transport_output.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Output"))
        transport_output.event_handler("on_error")(partial(handle_disconnect_or_error, "Output"))

        # Warmup frame
        await task.queue_frames([LLMRunFrame()])

    except Exception:
        # Cleanup resources created after task_manager.add on init failure
        await task.cancel()
        if transport_input:
            await transport_input.cleanup()
        await transport_output.cleanup()
        if stt_service:
            await stt_service.cleanup()
        if tts_service:
            await tts_service.cleanup()
        if llm_service:
            tool_unregister(llm_service, tool_names)
            await llm_service.cleanup()
        await task_manager.remove(id)
        raise

    init_total = time.monotonic() - total_start
    logger.info(f"[INIT][total] All initialization completed in {init_total:.3f} sec. pipeline id={id}")

    return {
        "task": task,
        "transport_input": transport_input,
        "transport_output": transport_output,
        "llm_service": llm_service,
        "tts_service": tts_service,
        "stt_service": stt_service,
        "tool_names": tool_names,
    }


async def execute_single_ai_pipeline(id: str, ctx: dict):
    """Run the single AI pipeline loop and cleanup. Runs as background task."""
    task = ctx["task"]
    transport_input = ctx["transport_input"]
    transport_output = ctx["transport_output"]
    llm_service = ctx["llm_service"]
    tts_service = ctx["tts_service"]
    stt_service = ctx["stt_service"]
    tool_names = ctx["tool_names"]

    try:
        runner = PipelineRunner()
        logger.info(f"[RUN] Starting pipeline id={id}")
        await runner.run(task)
    except asyncio.CancelledError:
        logger.info(f"[RUN] Pipeline cancelled. pipeline id={id}")
    except Exception as e:
        logger.error(f"[RUN] Pipeline error: {e}. pipeline id={id}")
    finally:
        logger.info(f"[CLEANUP] Cleaning up pipeline. pipeline id={id}")
        if task:
            await task.cancel()
        if transport_input:
            await transport_input.cleanup()
        if transport_output:
            await transport_output.cleanup()
        if stt_service:
            await stt_service.cleanup()
        if tts_service:
            await tts_service.cleanup()
        if llm_service:
            tool_unregister(llm_service, tool_names)
            await llm_service.cleanup()
        await task_manager.remove(id)
        logger.info(f"[CLEANUP] Pipeline cleaned. pipeline id={id}")


def _parse_language(language_str: str) -> Language:
    """Convert language string (e.g., 'en-US') to pipecat Language enum."""
    try:
        return Language[language_str.replace("-", "_").upper()]
    except (KeyError, AttributeError):
        return Language.EN_US


def create_tts_service(name: str, **options):
    name = name.lower()
    voice_id = options.get("voice_id") or "default_voice_id"
    language = options.get("language")

    if name == "cartesia":
        return _keep_usable(CartesiaTTSService(
            api_key=os.getenv("CARTESIA_API_KEY"),
            voice_id=voice_id,
            model="sonic-3.5",
            language=language,
            max_consecutive_zero_audio_contexts=0,
        ))
    elif name == "elevenlabs":
        return _keep_usable(ElevenLabsTTSService(
            api_key=os.getenv("ELEVENLABS_API_KEY"),
            voice_id=voice_id,
            model="eleven_turbo_v2_5",
            language=language,
            max_consecutive_zero_audio_contexts=0,
        ))
    elif name == "google":
        # Default to Chirp3 HD voice based on language when no voice specified.
        if not options.get("voice_id"):
            lang_code = language if language else "en-US"
            voice_id = f"{lang_code}-Chirp3-HD-Charon"
        # Extract language from voice name (e.g., "en-US-Chirp3-HD-Fenrir" -> "en-US")
        # Google TTS API requires language_code to match the voice's language.
        parts = voice_id.split("-")
        if len(parts) >= 2:
            lang = _parse_language(f"{parts[0]}-{parts[1]}")
        else:
            lang = _parse_language(language) if language else Language.EN_US
        return _keep_usable(GoogleTTSService(
            voice_id=voice_id,
            params=GoogleTTSService.InputParams(language=lang),
            max_consecutive_zero_audio_contexts=0,
        ))
    else:
        raise ValueError(f"Unsupported TTS service: {name}")


def create_stt_service(name: str, **options):
    name = name.lower()
    language = options.get("language") or None
    if name == "deepgram":
        live_options = LiveOptions(
            model="nova-2",
            language=language,
            interim_results=True,
            profanity_filter=True,
        )
        return _keep_usable(_harden_deepgram(DeepgramSTTService(
            api_key=os.getenv("DEEPGRAM_API_KEY"),
            live_options=live_options,
        )))
    elif name == "google":
        lang = _parse_language(language) if language else Language.EN_US
        return _keep_usable(GoogleSTTService(
            params=GoogleSTTService.InputParams(
                languages=[lang],
                model="latest_long",
                enable_automatic_punctuation=True,
                enable_interim_results=True,
            ),
        ))
    else:
        raise ValueError(f"Unsupported STT service: {name}")


def _openai_tools_to_standard(openai_tools: list[dict]) -> list[FunctionSchema]:
    """Convert OpenAI-format tools to pipecat FunctionSchema objects.

    OpenAI format: [{"type": "function", "function": {"name": ..., "description": ..., "parameters": {...}}}]
    FunctionSchema: FunctionSchema(name=..., description=..., properties=..., required=...)

    NOTE: FunctionSchema carries only name/description/properties/required.
    Anything inside `properties` (nested `additionalProperties`, `enum`, `pattern`,
    nested types) is preserved, but top-level `parameters.additionalProperties`
    and `function.strict` are NOT carried. The current ai-manager tool catalog
    uses only nested `additionalProperties` and no top-level `strict`, so this is
    lossless today. If a tool ever needs top-level strict/additionalProperties,
    extend this conversion rather than relying on passthrough.
    """
    if not openai_tools:
        return []

    schemas = []
    for tool in openai_tools:
        func = tool.get("function", {})
        params = func.get("parameters", {})
        # Warn if a tool carries top-level fields FunctionSchema cannot represent,
        # so a future ai-manager catalog change that relies on top-level
        # strict / additionalProperties fails loud here instead of silently
        # losing them. Nested (per-property) additionalProperties is preserved.
        if func.get("strict") is not None or params.get("additionalProperties") is not None:
            logger.warning(
                f"Tool '{func.get('name', '')}' sets top-level strict/additionalProperties "
                "which FunctionSchema does not carry; these are dropped on the universal "
                "LLMContext path. Extend _openai_tools_to_standard if they are required."
            )
        schemas.append(FunctionSchema(
            name=func.get("name", ""),
            description=func.get("description", ""),
            properties=params.get("properties", {}),
            required=params.get("required", []),
        ))
    return schemas


def _member_llm_type(ai: dict) -> str:
    """Return the LLM type the runner must build for a resolved team member.

    llm_type is computed by the Go resolver. None means an older Go that does
    not send the field (legacy behavior, but routed/internal services are never
    honored from the customer-facing engine_model); "" means Go rejected the
    member, which never falls back to engine_model for any provider.
    """
    resolved = ai.get("llm_type")
    if resolved is None:
        fallback = ai.get("engine_model") or ""
        svc = fallback.replace(":", ".", 1).split(".", 1)[0].lower() if fallback else ""
        if svc in ("platform_openrouter", "openrouter"):
            raise ValueError("engine model is not available")
        return fallback
    if resolved == "":
        raise ValueError("engine model is not available")
    return resolved


def create_llm_service(type: str, key: str, messages: list[dict], tools: list[dict], pipeline_id: str = "", **options):
    valid_messages = filter_valid_messages(messages)

    if "." in type:
        service_name, model_name = type.split(".", 1)
    elif ":" in type:
        service_name, model_name = type.split(":", 1)
    else:
        raise ValueError(f"Wrong LLM format: {type}. Expected format: 'service.model' or 'service:model' (e.g., 'openai.gpt-4o-mini')")

    service_name = service_name.lower()
    if service_name == "openai":
        api_key = key or os.getenv("OPENAI_API_KEY")
        llm = OpenAILLMService(api_key=api_key, model=model_name)

        # Universal LLMContext + LLMContextAggregatorPair (the legacy
        # OpenAILLMContext + create_context_aggregator API was removed in
        # pipecat 1.x). Tools are converted to FunctionSchema/ToolsSchema so the
        # provider adapter serializes them correctly, matching the Gemini path.
        standard_tools = _openai_tools_to_standard(tools)
        tools_schema = ToolsSchema(standard_tools=standard_tools) if standard_tools else NOT_GIVEN
        ctx = LLMContext(messages=valid_messages, tools=tools_schema)
        aggregator = _make_aggregator(ctx)

        return llm, aggregator

    elif service_name == "grok":
        api_key = key or os.getenv("XAI_API_KEY")
        llm = OpenAILLMService(
            api_key=api_key,
            model=model_name,
            base_url="https://api.x.ai/v1"
        )

        standard_tools = _openai_tools_to_standard(tools)
        tools_schema = ToolsSchema(standard_tools=standard_tools) if standard_tools else NOT_GIVEN
        ctx = LLMContext(messages=valid_messages, tools=tools_schema)
        aggregator = _make_aggregator(ctx)

        return llm, aggregator

    elif service_name == "gemini":
        api_key = key or os.getenv("GOOGLE_API_KEY")
        llm = GoogleLLMService(api_key=api_key, model=model_name, stream_idle_timeout_secs=None)

        # Use universal LLMContext so GeminiLLMAdapter properly converts
        # OpenAI-format tools to Google's function_declarations format.
        # OpenAILLMContext passes tools as-is to GenerateContentConfig,
        # which rejects the OpenAI {"type":"function","function":{...}} format.
        standard_tools = _openai_tools_to_standard(tools)
        # Drop only the tools the installed Gemini schema validator rejects,
        # so one bad tool cannot fail every turn of the session (design 15.5).
        standard_tools = drop_gemini_invalid_tools(standard_tools, pipeline_id)
        if standard_tools:
            tools_schema = ToolsSchema(standard_tools=standard_tools)
            logger.debug(f"Converted {len(standard_tools)} tools to FunctionSchema for Gemini")
        else:
            tools_schema = NOT_GIVEN
        # Design 2.5e: 1.4.0 converted a lone initial system message to user in
        # place (len(messages) == 1). Reproduce it on a copy (input untouched).
        if len(valid_messages) == 1 and valid_messages[0]["role"] == "system":
            valid_messages = [{**valid_messages[0], "role": "user"}]
        ctx = LLMContext(messages=valid_messages, tools=tools_schema)
        aggregator = _make_aggregator(ctx)

        return llm, aggregator

    elif service_name == "platform_openrouter":
        # Internal service name emitted only by the Go resolver
        # (platform_openrouter.<slug>). The raw "openrouter" service is
        # intentionally NOT supported. The key argument is intentionally
        # ignored: customers never supply it.
        api_key = os.getenv("OPENROUTER_API_KEY", "")
        if not api_key:
            raise ValueError("OpenRouter is not configured")
        # The OpenAI client merges settings.extra into the top-level kwargs of
        # chat.completions.create(), and the SDK rejects an unknown top-level
        # `provider` kwarg; it must travel inside `extra_body`.
        llm = OpenRouterLLMService(
            api_key=api_key,
            settings=OpenRouterLLMService.Settings(
                model=model_name,
                extra={"extra_body": {"provider": {
                    "zdr": True,
                    "data_collection": "deny",
                    "require_parameters": True,
                }}},
            ),
        )

        standard_tools = _openai_tools_to_standard(tools)
        tools_schema = ToolsSchema(standard_tools=standard_tools) if standard_tools else NOT_GIVEN
        ctx = LLMContext(messages=valid_messages, tools=tools_schema)
        aggregator = _make_aggregator(ctx)

        return llm, aggregator

    else:
        raise ValueError(f"Unsupported LLM service: {service_name}")


class UnpacedWebsocketClientOutputTransport(WebsocketClientOutputTransport):
    """Output transport that delivers audio faster than real-time.

    No-ops _write_audio_sleep() so TTS frames are forwarded immediately to
    Asterisk's chan_websocket, which handles re-timing internally.

    On InterruptionFrame (barge-in), sends FLUSH_MEDIA text command so
    Asterisk discards queued audio instantly.
    """

    async def _write_audio_sleep(self):
        pass

    async def process_frame(self, frame, direction):
        if isinstance(frame, InterruptionFrame):
            await self._write_frame(TextFrame(text="FLUSH_MEDIA"))
        await super().process_frame(frame, direction)


class UnpacedWebsocketClientTransport(WebsocketClientTransport):
    """WebSocket transport using unpaced output for audio delivery."""

    def output(self) -> UnpacedWebsocketClientOutputTransport:
        if not self._output:
            self._output = UnpacedWebsocketClientOutputTransport(
                self, self._session, self._params
            )
        return self._output


class EarlyAudioBufferingInputTransport(WebsocketClientInputTransport):
    """Hold audio that arrives before StartFrame and replay it in order (design 2.5b)."""

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._early_audio = []

    async def push_audio_frame(self, frame):
        if self._audio_task is None:
            if self._params.audio_in_enabled and not self._paused:
                self._early_audio.append(frame)
            return
        await super().push_audio_frame(frame)

    async def set_transport_ready(self, frame):
        await super().set_transport_ready(frame)
        held, self._early_audio = self._early_audio, []
        for f in held:
            await super().push_audio_frame(f)


class BufferingInputWebsocketClientTransport(WebsocketClientTransport):
    """WebSocket transport whose input buffers early audio."""

    def input(self) -> EarlyAudioBufferingInputTransport:
        if not self._input:
            self._input = EarlyAudioBufferingInputTransport(
                self, self._session, self._params
            )
        return self._input


def create_websocket_transport(direction: str, id: str, vad_analyzer=None, turn_analyzer=None):
    uri = f"{common.PIPECATCALL_WS_URL}/{id}/ws?direction={direction}"
    logger.info(f"Establishing WebSocket connection to URI: {uri}")

    params = WebsocketClientParams(
        serializer=ProtobufFrameSerializer(),
        audio_in_enabled=True,
        audio_out_enabled=True,
        add_wav_header=False,
        vad_analyzer=vad_analyzer,
        turn_analyzer=turn_analyzer,
        session_timeout=common.PIPELINE_SESSION_TIMEOUT,
    )

    if direction == "output":
        transport = UnpacedWebsocketClientTransport(uri=uri, params=params)
    else:
        transport = BufferingInputWebsocketClientTransport(uri=uri, params=params)

    return transport


async def init_team_pipeline(
    id: str,
    resolved_team: dict,
    stt_language: str = None,
    tts_language: str = None,
    stt_type: str = None,
    tts_type: str = None,
    llm_messages: list = None,
    vad_config: dict = None,
    smart_turn_enabled: bool = False,
) -> dict:
    """Initialize team pipeline. Returns context dict. Raises on failure.

    `stt_type`/`tts_type` are the request-level types from bin-ai-manager. Their
    presence selects the session's audio mode: both `None` (the default) means a
    text-only session (conversation, task, contact_case listen turn) and no
    per-member TTS/STT or input transport is built; non-empty means a voice call
    and each member's TTS/STT is built from that member's own config. The two
    fields gate independently (`tts_type` gates member TTS, `stt_type` gates
    member STT), mirroring init_single_ai_pipeline. The values themselves are
    never used for members.
    """
    total_start = time.monotonic()
    logger.info(f"[TEAM][INIT] Starting team pipeline. pipeline id={id}")

    if llm_messages is None:
        llm_messages = []

    members = resolved_team.get("members", [])
    start_member_id = resolved_team["start_member_id"]

    # --- Step 1: Create per-member service instances ---
    llm_services = {}
    tts_services = {}
    stt_services = {}

    # Create per-member services directly (no asyncio.to_thread).
    # Some service constructors (e.g. GoogleTTSService) internally create gRPC
    # async channels that require a running event loop, which thread pool threads
    # lack. Running them on the event loop is safe — they are fast object creation.
    for member in members:
        mid = member["id"]
        ai = member["ai"]
        start = time.monotonic()

        llm_svc, _ = create_llm_service(_member_llm_type(ai), ai["engine_key"], [], [], pipeline_id=id)
        # Design 2.5: flows tools default to cancel_on_interruption=False, which
        # makes 1.12 compose an ASYNC TOOLS system instruction; keep the member
        # init_prompt in the system slot (1.4 parity). Private API, guarded by
        # the real-library test.
        llm_svc._has_async_tools = lambda: False
        llm_services[mid] = llm_svc

        if tts_type and ai.get("tts_type"):
            tts_services[mid] = create_tts_service(
                ai["tts_type"],
                voice_id=ai.get("tts_voice_id"), language=tts_language,
            )

        if stt_type and ai.get("stt_type"):
            stt_services[mid] = create_stt_service(
                ai["stt_type"], language=stt_language,
            )

        logger.info(f"[TEAM][INIT] Member {mid} services created in {time.monotonic() - start:.3f}s")

    logger.info(f"[TEAM][INIT] Created {len(llm_services)} LLM, {len(tts_services)} TTS, {len(stt_services)} STT services. pipeline id={id}")
    if not stt_type and not tts_type:
        logger.info(f"[TEAM][INIT] Text-only session; skipping per-member TTS/STT. pipeline id={id}")

    # VOIP-1512: surface missing/unadvertised tool calls as a structured ERROR.
    # Anchored on each member's real LLMService (the RoutingLLMService wrapper
    # does not fire on_function_calls_started itself). tool_register is never
    # called on the team path, so this is the only place team sessions get
    # missing-tool visibility.
    for _svc in llm_services.values():
        register_missing_tool_logging(_svc, id)

    # --- Step 2: Create routing services ---
    routing_llm = RoutingLLMService(llm_services)
    routing_llm.set_active_member(start_member_id)

    routing_tts = None
    if tts_services:
        routing_tts = RoutingTTSService(tts_services)
        routing_tts.set_active_member(start_member_id)

    routing_stt = None
    if stt_services:
        routing_stt = RoutingSTTService(stt_services)
        routing_stt.set_active_member(start_member_id)

    # --- Step 3: Create context aggregator ---
    start_member = next((m for m in members if m["id"] == start_member_id), None)
    if start_member is None:
        raise ValueError(f"start_member_id {start_member_id} not found in members list")
    start_messages = []
    if start_member["ai"].get("init_prompt"):
        start_messages.append({"role": "system", "content": start_member["ai"]["init_prompt"]})
    start_messages.extend(filter_valid_messages(llm_messages))

    # Universal LLMContext + LLMContextAggregatorPair. Built-in pipecat.flows
    # always uses the universal adapter, so tools go through
    # ToolsSchema/FunctionSchema for every provider.
    context = LLMContext(messages=start_messages, tools=NOT_GIVEN)
    context_aggregator = _make_aggregator(context)

    # --- Step 4: Create transports ---
    transport_input = None
    if routing_stt:
        vad_analyzer = SileroVADAnalyzer(params=build_vad_params(vad_config, smart_turn_enabled=smart_turn_enabled))
        turn_analyzer = None
        if smart_turn_enabled:
            from pipecat.audio.turn.smart_turn.local_smart_turn_v3 import LocalSmartTurnAnalyzerV3
            turn_analyzer = LocalSmartTurnAnalyzerV3()
        transport_input = create_websocket_transport("input", id, vad_analyzer=vad_analyzer, turn_analyzer=turn_analyzer)
    transport_output = create_websocket_transport("output", id, vad_analyzer=None)

    # --- Step 5: Build pipeline ---
    pipeline_stages = []
    if routing_stt:
        pipeline_stages.append(transport_input.input())
        pipeline_stages.append(routing_stt)
    pipeline_stages.append(context_aggregator.user())
    pipeline_stages.append(routing_llm)
    if routing_tts:
        pipeline_stages.append(routing_tts)
    pipeline_stages.append(context_aggregator.assistant())
    pipeline_stages.append(transport_output.output())

    pipeline = Pipeline(pipeline_stages)

    # --- Step 6: Create pipeline task ---
    task = PipelineTask(
        pipeline,
        params=PipelineParams(
            audio_out_sample_rate=16000,
            enable_metrics=True,
            enable_usage_metrics=True,
        ),
    )

    await task_manager.add(id, task)

    try:
        # --- Step 7: Set up FlowManager ---
        member_nodes, start_node = build_team_flow(
            resolved_team, id,
            routing_llm, routing_tts, routing_stt,
            llm_messages=llm_messages,
        )

        # pipecat.flows advertises transition handlers via FunctionSchema
        # (LLMSetToolsFrame); each member LLM auto-registers them on its own
        # LLMContextFrame, so FlowManager no longer calls register_function.
        # The _llm swap below is kept only as a harmless reference (flows reads
        # _llm only in generate_summary, unused here).
        active_llm = routing_llm.active_service
        if active_llm is None:
            raise ValueError(f"No active LLM service for start_member_id={start_member_id}")

        flow_manager = FlowManager(
            task=task,
            llm=active_llm,
            context_aggregator=context_aggregator,
        )
        # CAUTION: relies on FlowManager storing the LLM as _llm. Verified with
        # pipecat 1.12.0 built-in flows. The guard fails loudly if a future flows
        # version renames the attribute, so the private-API drift is noticed on
        # the next bump instead of the swap silently becoming a no-op.
        if not hasattr(flow_manager, "_llm"):
            raise RuntimeError(
                "FlowManager no longer exposes _llm; team LLM reference swap broken. "
                "Re-verify pipecat.flows internals after the upgrade."
            )
        flow_manager._llm = routing_llm

        # --- Step 8: Event handlers ---
        async def handle_disconnect_or_error(name, transport, error=None):
            logger.error(f"[TEAM] {name} WebSocket disconnected or errored: {error}. pipeline id={id}")
            await task.cancel()

        if transport_input:
            transport_input.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Input"))
            transport_input.event_handler("on_error")(partial(handle_disconnect_or_error, "Input"))
        transport_output.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Output"))
        transport_output.event_handler("on_error")(partial(handle_disconnect_or_error, "Output"))

        # --- Step 9: Initialize FlowManager ---
        await flow_manager.initialize(start_node)

    except Exception:
        # Cleanup resources created after task_manager.add on init failure
        await task.cancel()
        if transport_input:
            await transport_input.cleanup()
        await transport_output.cleanup()
        if routing_stt:
            await routing_stt.cleanup()
        if routing_tts:
            await routing_tts.cleanup()
        if routing_llm:
            await routing_llm.cleanup()
        await task_manager.remove(id)
        raise

    init_total = time.monotonic() - total_start
    logger.info(f"[TEAM][INIT][total] All initialization completed in {init_total:.3f} sec. pipeline id={id}")

    return {
        "task": task,
        "transport_input": transport_input,
        "transport_output": transport_output,
        "routing_llm": routing_llm,
        "routing_tts": routing_tts,
        "routing_stt": routing_stt,
    }


async def execute_team_pipeline(id: str, ctx: dict):
    """Run the team pipeline loop and cleanup. Runs as background task."""
    task = ctx["task"]
    transport_input = ctx["transport_input"]
    transport_output = ctx["transport_output"]
    routing_llm = ctx["routing_llm"]
    routing_tts = ctx["routing_tts"]
    routing_stt = ctx["routing_stt"]

    try:
        runner = PipelineRunner()
        logger.info(f"[TEAM][RUN] Starting team pipeline. pipeline id={id}")
        await runner.run(task)
    except asyncio.CancelledError:
        logger.info(f"[TEAM][RUN] Pipeline cancelled. pipeline id={id}")
    except Exception as e:
        logger.error(f"[TEAM][RUN] Pipeline error: {e}. pipeline id={id}")
    finally:
        logger.info(f"[TEAM][CLEANUP] Cleaning up team pipeline. pipeline id={id}")
        if task:
            await task.cancel()
        if transport_input:
            await transport_input.cleanup()
        if transport_output:
            await transport_output.cleanup()
        if routing_stt:
            await routing_stt.cleanup()
        if routing_tts:
            await routing_tts.cleanup()
        if routing_llm:
            await routing_llm.cleanup()
        await task_manager.remove(id)
        logger.info(f"[TEAM][CLEANUP] Team pipeline cleaned. pipeline id={id}")
