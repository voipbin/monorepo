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
from pipecat.processors.filters.stt_mute_filter import STTMuteConfig, STTMuteFilter, STTMuteStrategy

# llm
from pipecat.services.openai.llm import OpenAILLMService

# aggregators / context
from pipecat.processors.aggregators.openai_llm_context import OpenAILLMContext

# pipeline
from pipecat.audio.vad.silero import SileroVADAnalyzer
from pipecat.frames.frames import LLMRunFrame
from pipecat.pipeline.pipeline import Pipeline
from pipecat.pipeline.runner import PipelineRunner
from pipecat.pipeline.task import PipelineParams, PipelineTask
from pipecat.processors.frameworks.rtvi import RTVIConfig, RTVIObserver, RTVIProcessor
from pipecat.serializers.protobuf import ProtobufFrameSerializer
from pipecat.transports.websocket.client import (
    WebsocketClientParams,
    WebsocketClientTransport,
)

from tools import tool_register, tool_unregister, convert_to_openai_format, get_tool_names
from task import task_manager


async def run_pipeline(
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
):
    total_start = time.monotonic()
    logger.info(f"[INIT] Starting Pipecat client pipeline id={id}")

    transport_input = None
    transport_output = None
    task = None

    if llm_messages is None:
        llm_messages = []

    # Convert tools from ai-manager format to OpenAI format
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

            vad_analyzer = SileroVADAnalyzer()
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

    async def init_rtvi():
        start = time.monotonic()
        rtvi = RTVIProcessor(config=RTVIConfig(config=[]))
        logger.info(f"[INIT][rtvi] done in {time.monotonic() - start:.3f} sec. pipeline id={id}")
        return {
            "rtvi": rtvi,
        }
    init_tasks["rtvi"] = asyncio.create_task(init_rtvi())

    # Await all init tasks
    try:
        results_list = await asyncio.gather(*init_tasks.values())
    except Exception as e:
        logger.error(f"[INIT] Pipeline initialization failed: {e}")
        for task in init_tasks.values():
            if not task.done():
                task.cancel()
        raise
    logger.info(f"[INIT] All components initialized in {time.monotonic() - total_start:.3f} sec. pipeline id={id}")

    results = {}
    for part in results_list:
        results.update(part)

    # Access initialized services by key
    stt_service = results.get("stt_service")
    transport_input = results.get("transport_input")

    tts_service = results.get("tts_service")

    llm_service = results["llm_service"]
    llm_context_aggregator = results["llm_context_aggregator"]

    transport_output = results["transport_output"]
    rtvi = results["rtvi"]

    # Assemble pipeline stages
    pipeline_stages = []

    if transport_input:
        pipeline_stages.append(transport_input.input())
        pipeline_stages.append(stt_service)

    pipeline_stages.append(rtvi)
    pipeline_stages.append(llm_context_aggregator.user())
    pipeline_stages.append(llm_service)

    if tts_service:
        pipeline_stages.append(tts_service)

    pipeline_stages.append(llm_context_aggregator.assistant())
    pipeline_stages.append(transport_output.output())

    pipeline = Pipeline(pipeline_stages)

    # Register tools
    tool_register(llm_service, id, tool_names)

    # Create Pipeline Task
    task_start = time.monotonic()
    task = PipelineTask(
        pipeline,
        params=PipelineParams(
            enable_metrics=True,
            enable_usage_metrics=True,
        ),
        observers=[RTVIObserver(rtvi)],
    )

    await task_manager.add(id, task)
    logger.info(f"[INIT][task_create] done in {time.monotonic() - task_start:.3f} sec. pipeline id={id}")

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

    # Runner
    runner = PipelineRunner()

    init_total = time.monotonic() - total_start
    logger.info(f"[INIT][total] All initialization completed in {init_total:.3f} sec. pipeline id={id}")

    # Run the pipeline
    try:
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
    else:
        raise ValueError(f"Unsupported STT service: {name}")


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
    else:
        raise ValueError(f"Unsupported LLM service: {service_name}")


def create_websocket_transport(direction: str, id: str, vad_analyzer=None):
    uri = f"{common.PIPECATCALL_WS_URL}/{id}/ws?direction={direction}"
    logger.info(f"Establishing WebSocket connection to URI: {uri}")

    return WebsocketClientTransport(
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
