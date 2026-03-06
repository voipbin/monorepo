import asyncio
import os
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
from pipecat.services.deepgram.stt import DeepgramSTTService
from deepgram import LiveOptions
from pipecat.services.whisper.stt import Model, WhisperSTTService
from pipecat.services.google.stt import GoogleSTTService
from pipecat.transcriptions.language import Language
from pipecat.processors.filters.stt_mute_filter import STTMuteConfig, STTMuteFilter, STTMuteStrategy

# llm
from pipecat.services.openai.llm import OpenAILLMService
from pipecat.services.google.llm import GoogleLLMService

# aggregators / context
from pipecat.processors.aggregators.openai_llm_context import OpenAILLMContext
from pipecat.processors.aggregators.llm_context import LLMContext, NOT_GIVEN
from pipecat.processors.aggregators.llm_response_universal import LLMContextAggregatorPair
from pipecat.adapters.schemas.function_schema import FunctionSchema
from pipecat.adapters.schemas.tools_schema import ToolsSchema

# pipeline
from pipecat.audio.vad.silero import SileroVADAnalyzer
from pipecat.audio.vad.vad_analyzer import VADParams
from pipecat.frames.frames import LLMRunFrame
from pipecat.pipeline.pipeline import Pipeline
from pipecat.pipeline.runner import PipelineRunner
from pipecat.pipeline.task import PipelineParams, PipelineTask
from pipecat.serializers.protobuf import ProtobufFrameSerializer
from pipecat.transports.websocket.client import (
    WebsocketClientParams,
    WebsocketClientTransport,
    WebsocketClientOutputTransport,
)

from tools import tool_register, tool_unregister, convert_to_openai_format, get_tool_names
from task import task_manager
from routing_llm import RoutingLLMService
from routing_tts import RoutingTTSService
from routing_stt import RoutingSTTService
from team_flow import build_team_flow
from pipecat_flows import FlowManager


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
    vad_stop_secs: float = 0.5,
) -> dict:
    """Initialize the pipeline. Returns context dict. Raises on failure."""
    if resolved_team:
        ctx = await init_team_pipeline(
            id, resolved_team,
            stt_language=stt_language,
            tts_language=tts_language,
            llm_messages=llm_messages,
            vad_stop_secs=vad_stop_secs,
        )
        ctx["type"] = "team"
        return ctx
    else:
        ctx = await init_single_ai_pipeline(
            id, llm_type, llm_key, llm_messages,
            stt_type, stt_language, tts_type, tts_language,
            tts_voice_id, tools_data,
            vad_stop_secs=vad_stop_secs,
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
    vad_stop_secs: float = 0.5,
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
            vad_analyzer = SileroVADAnalyzer(params=VADParams(stop_secs=max(vad_stop_secs, 0.3)))
            transport = create_websocket_transport("input", id, vad_analyzer=vad_analyzer)
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
        llm_service, aggregator = create_llm_service(llm_type, llm_key, llm_messages, openai_tools)
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
        tool_register(llm_service, id, tool_names)

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
        tool_unregister(llm_service, tool_names)
        await task_manager.remove(id)
        raise

    init_total = time.monotonic() - total_start
    logger.info(f"[INIT][total] All initialization completed in {init_total:.3f} sec. pipeline id={id}")

    return {
        "task": task,
        "transport_input": transport_input,
        "transport_output": transport_output,
        "llm_service": llm_service,
        "tool_names": tool_names,
    }


async def execute_single_ai_pipeline(id: str, ctx: dict):
    """Run the single AI pipeline loop and cleanup. Runs as background task."""
    task = ctx["task"]
    transport_input = ctx["transport_input"]
    transport_output = ctx["transport_output"]
    llm_service = ctx["llm_service"]
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
        if llm_service:
            tool_unregister(llm_service, tool_names)
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
        return CartesiaTTSService(
            api_key=os.getenv("CARTESIA_API_KEY"),
            voice_id=voice_id,
            language=language,
        )
    elif name == "elevenlabs":
        return ElevenLabsTTSService(
            api_key=os.getenv("ELEVENLABS_API_KEY"),
            voice_id=voice_id,
            language=language,
        )
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
        return GoogleTTSService(
            voice_id=voice_id,
            params=GoogleTTSService.InputParams(language=lang),
        )
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
        )
        return DeepgramSTTService(
            api_key=os.getenv("DEEPGRAM_API_KEY"),
            live_options=live_options,
        )
    elif name == "google":
        lang = _parse_language(language) if language else Language.EN_US
        return GoogleSTTService(
            params=GoogleSTTService.InputParams(
                languages=[lang],
                model="latest_long",
                enable_automatic_punctuation=True,
                enable_interim_results=True,
            ),
        )
    else:
        raise ValueError(f"Unsupported STT service: {name}")


def _openai_tools_to_standard(openai_tools: list[dict]) -> list[FunctionSchema]:
    """Convert OpenAI-format tools to pipecat FunctionSchema objects.

    OpenAI format: [{"type": "function", "function": {"name": ..., "description": ..., "parameters": {...}}}]
    FunctionSchema: FunctionSchema(name=..., description=..., properties=..., required=...)
    """
    if not openai_tools:
        return []

    schemas = []
    for tool in openai_tools:
        func = tool.get("function", {})
        params = func.get("parameters", {})
        schemas.append(FunctionSchema(
            name=func.get("name", ""),
            description=func.get("description", ""),
            properties=params.get("properties", {}),
            required=params.get("required", []),
        ))
    return schemas


def create_llm_service(type: str, key: str, messages: list[dict], tools: list[dict], **options):
    valid_messages = [m for m in messages if m.get("role") and m.get("content")]

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

        ctx = OpenAILLMContext(messages=valid_messages, tools=tools)
        aggregator = llm.create_context_aggregator(ctx)

        return llm, aggregator

    elif service_name == "grok":
        api_key = key or os.getenv("XAI_API_KEY")
        llm = OpenAILLMService(
            api_key=api_key,
            model=model_name,
            base_url="https://api.x.ai/v1"
        )

        ctx = OpenAILLMContext(messages=valid_messages, tools=tools)
        aggregator = llm.create_context_aggregator(ctx)

        return llm, aggregator

    elif service_name == "gemini":
        api_key = key or os.getenv("GOOGLE_API_KEY")
        llm = GoogleLLMService(api_key=api_key, model=model_name)

        # Use universal LLMContext so GeminiLLMAdapter properly converts
        # OpenAI-format tools to Google's function_declarations format.
        # OpenAILLMContext passes tools as-is to GenerateContentConfig,
        # which rejects the OpenAI {"type":"function","function":{...}} format.
        standard_tools = _openai_tools_to_standard(tools)
        if standard_tools:
            tools_schema = ToolsSchema(standard_tools=standard_tools)
            logger.debug(f"Converted {len(standard_tools)} tools to FunctionSchema for Gemini")
        else:
            tools_schema = NOT_GIVEN
        ctx = LLMContext(messages=valid_messages, tools=tools_schema)
        aggregator = LLMContextAggregatorPair(ctx)

        return llm, aggregator

    else:
        raise ValueError(f"Unsupported LLM service: {service_name}")


class UnpacedWebsocketClientOutputTransport(WebsocketClientOutputTransport):
    """Output transport that skips real-time audio pacing.

    Pipecat's default WebsocketClientOutputTransport sleeps between audio
    frames to simulate real-time playback (via _write_audio_sleep). This is
    correct for browser/speaker endpoints but counterproductive when forwarding
    to Asterisk via Go: it prevents faster-than-real-time delivery, so any
    asyncio contention causes audio gaps. By no-oping the sleep, TTS audio is
    forwarded as fast as the TTS generates it — matching the proven pattern
    from bin-tts-manager where ElevenLabs delivers audio faster than real-time.
    Asterisk's chan_websocket buffers incoming audio internally.
    """

    async def _write_audio_sleep(self):
        pass


class UnpacedWebsocketClientTransport(WebsocketClientTransport):
    """WebSocket transport that uses UnpacedWebsocketClientOutputTransport.

    CAUTION: Relies on WebsocketClientTransport internals (_output, _session,
    _params). Verified with pipecat-ai>=0.0.101. Re-verify after upgrades.
    """

    def output(self) -> UnpacedWebsocketClientOutputTransport:
        if not self._output:
            self._output = UnpacedWebsocketClientOutputTransport(
                self, self._session, self._params
            )
        return self._output


def create_websocket_transport(direction: str, id: str, vad_analyzer=None):
    uri = f"{common.PIPECATCALL_WS_URL}/{id}/ws?direction={direction}"
    logger.info(f"Establishing WebSocket connection to URI: {uri}")

    # For output direction, use unpaced transport so TTS audio is forwarded
    # to Go as fast as the TTS generates it. Asterisk's chan_websocket buffers
    # incoming audio internally. Without this, pipecat paces audio at real-time
    # rate (via _write_audio_sleep), leaving zero buffer margin and causing
    # choppy audio when asyncio contention delays frame delivery.
    transport_cls = UnpacedWebsocketClientTransport if direction == "output" else WebsocketClientTransport

    transport = transport_cls(
        uri=uri,
        params=WebsocketClientParams(
            serializer=ProtobufFrameSerializer(),
            audio_in_enabled=True,
            audio_out_enabled=True,
            add_wav_header=False,
            vad_analyzer=vad_analyzer,
            session_timeout=common.PIPELINE_SESSION_TIMEOUT,
        )
    )

    return transport


async def init_team_pipeline(
    id: str,
    resolved_team: dict,
    stt_language: str = None,
    tts_language: str = None,
    llm_messages: list = None,
    vad_stop_secs: float = 0.5,
) -> dict:
    """Initialize team pipeline. Returns context dict. Raises on failure."""
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

        llm_svc, _ = create_llm_service(ai["engine_model"], ai["engine_key"], [], [])
        llm_services[mid] = llm_svc

        if ai.get("tts_type"):
            tts_services[mid] = create_tts_service(
                ai["tts_type"],
                voice_id=ai.get("tts_voice_id"), language=tts_language,
            )

        if ai.get("stt_type"):
            stt_services[mid] = create_stt_service(
                ai["stt_type"], language=stt_language,
            )

        logger.info(f"[TEAM][INIT] Member {mid} services created in {time.monotonic() - start:.3f}s")

    logger.info(f"[TEAM][INIT] Created {len(llm_services)} LLM, {len(tts_services)} TTS, {len(stt_services)} STT services. pipeline id={id}")

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
    start_messages.extend([m for m in llm_messages if m.get("role") and m.get("content")])

    # Use universal LLMContext + LLMContextAggregatorPair so FlowManager's
    # create_adapter() returns UniversalLLMAdapter. This ensures tools are
    # converted through ToolsSchema/FunctionSchema, which all providers
    # (OpenAI, Gemini, Anthropic) handle correctly. The legacy path
    # (OpenAILLMContext + provider-specific aggregator) causes tool format
    # mismatches — e.g. Gemini rejects OpenAI-format tools passed as-is.
    context = LLMContext(messages=start_messages, tools=NOT_GIVEN)
    context_aggregator = LLMContextAggregatorPair(context)

    # --- Step 4: Create transports ---
    transport_input = None
    if routing_stt:
        vad_analyzer = SileroVADAnalyzer(params=VADParams(stop_secs=max(vad_stop_secs, 0.3)))
        transport_input = create_websocket_transport("input", id, vad_analyzer=vad_analyzer)
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
        )

        # FlowManager's create_adapter checks the LLM class type to determine
        # the provider adapter (OpenAI, Google, Anthropic, etc.). Pass the start
        # member's actual LLM service so the adapter is created correctly, then
        # swap the internal reference to routing_llm so register_function and
        # unregister_function delegate to ALL member services.
        active_llm = routing_llm.active_service
        if active_llm is None:
            raise ValueError(f"No active LLM service for start_member_id={start_member_id}")

        flow_manager = FlowManager(
            task=task,
            llm=active_llm,
            context_aggregator=context_aggregator,
        )
        # CAUTION: Relies on FlowManager storing the LLM as _llm.
        # Verified with pipecat-ai-flows>=0.0.10. Re-verify after upgrades.
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
        await task_manager.remove(id)
        raise

    init_total = time.monotonic() - total_start
    logger.info(f"[TEAM][INIT][total] All initialization completed in {init_total:.3f} sec. pipeline id={id}")

    return {
        "task": task,
        "transport_input": transport_input,
        "transport_output": transport_output,
    }


async def execute_team_pipeline(id: str, ctx: dict):
    """Run the team pipeline loop and cleanup. Runs as background task."""
    task = ctx["task"]
    transport_input = ctx["transport_input"]
    transport_output = ctx["transport_output"]

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
        await task_manager.remove(id)
        logger.info(f"[TEAM][CLEANUP] Team pipeline cleaned. pipeline id={id}")
