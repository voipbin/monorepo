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

from tools import tool_register, tool_unregister, tools
from task import task_manager


# async def run_pipeline(id: str, llm_type: str, llm_key: str, tts: str, stt: str, voice_id: str = None, messages: list = None):
#     logger.info(f"Starting Pipecat client pipeline. id: {id}, llm_type: {llm_type}, tts: {tts}, stt: {stt}, voice_id: {voice_id}")

#     if messages is None:
#         messages = []

#     init_tasks = {}

#     if stt:
#         init_tasks["stt_service"] = asyncio.create_task(async_create_stt(stt))
#         init_tasks["vad"] = asyncio.create_task(async_create_vad())
#         init_tasks["transport_input"] = None

#     if tts:
#         init_tasks["tts_service"] = asyncio.create_task(async_create_tts(tts, voice_id))

#     init_tasks["llm"] = asyncio.create_task(async_create_llm(llm_type, llm_key, messages))
#     init_tasks["rtvi"] = asyncio.create_task(async_create_rtvi())
#     init_tasks["transport_output"] = None

#     results = {}
#     for key, task in init_tasks.items():
#         if task is not None:
#             results[key] = await task
#         else:
#             results[key] = None

#     stt_service = results.get("stt_service")
#     vad_analyzer = results.get("vad")
#     tts_service = results.get("tts_service")
#     (llm_service, llm_context_aggregator) = results.get("llm")
#     rtvi = results.get("rtvi")

#     transport_input = create_websocket_transport("input", id, vad_analyzer) if stt else None
#     transport_output = create_websocket_transport("output", id, None)

#     pipeline_stages = []

#     if stt:
#         pipeline_stages.append(transport_input.input())
#         pipeline_stages.append(stt_service)

#     pipeline_stages.append(rtvi)
#     pipeline_stages.append(llm_context_aggregator.user())
#     pipeline_stages.append(llm_service)

#     if tts:
#         pipeline_stages.append(tts_service)

#     pipeline_stages.append(llm_context_aggregator.assistant())
#     pipeline_stages.append(transport_output.output())

#     pipeline = Pipeline(pipeline_stages)

#     tool_register(llm_service, id)

#     logger.info(f"Starting Pipecat client pipeline task. id: {id}")
#     task = PipelineTask(
#         pipeline,
#         params=PipelineParams(enable_metrics=True, enable_usage_metrics=True),
#         observers=[RTVIObserver(rtvi)],
#     )
#     await task_manager.add(id, task)

#     async def handle_disconnect_or_error(name, transport, error=None):
#         logger.error(f"{name} WebSocket disconnected or errored: {error}")
#         await task.cancel()

#     if stt:
#         transport_input.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Input", transport_input))
#         transport_input.event_handler("on_error")(partial(handle_disconnect_or_error, "Input", transport_input))

#     transport_output.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Output", transport_output))
#     transport_output.event_handler("on_error")(partial(handle_disconnect_or_error, "Output", transport_output))

#     runner = PipelineRunner()
#     await task.queue_frames([LLMRunFrame()])

#     try:
#         logger.info(f"Running Pipecat client pipeline. id: {id}")
#         await runner.run(task)
#     except asyncio.CancelledError:
#         logger.info("Pipecat client pipeline cancelled.")
#     except Exception as e:
#         logger.error(f"Pipecat client pipeline error: {e}")
#     finally:
#         logger.info(f"Cleaning up Pipecat client pipeline. id: {id}")
#         if task:
#             await task.cancel()
#         if transport_input:
#             await transport_input.cleanup()
#         if transport_output:
#             await transport_output.cleanup()
#         if llm_service:
#             tool_unregister(llm_service)
#         await task_manager.remove(id)
#         logger.info(f"Pipeline cleaned up (id={id})")



async def run_pipeline(id: str, llm_type: str, llm_key: str, tts: str, stt: str, voice_id: str = None, messages: list = None):
    total_start = time.monotonic()
    logger.info(f"[INIT] Starting Pipecat client pipeline id={id}")

    transport_input = None
    transport_output = None
    task = None

    if messages is None:
        messages = []

    # ----------------------------------------------------------------------
    # 1) 병렬 초기화를 위한 async tasks 생성
    # ----------------------------------------------------------------------
    init_tasks = {}

    if stt:
        async def init_stt_and_input_ws():
            start = time.monotonic()
            stt_service = create_stt_service(stt)

            vad = SileroVADAnalyzer()  # must create per-session
            transport = create_websocket_transport("input", id, vad)

            logger.info(f"[INIT][stt+ws_input] done in {time.monotonic() - start:.3f} sec")
            return stt_service, transport, vad

        init_tasks["stt_input"] = asyncio.create_task(init_stt_and_input_ws())

    if tts:
        async def init_tts():
            start = time.monotonic()
            tts_service = create_tts_service(tts, voice_id=voice_id)
            logger.info(f"[INIT][tts] done in {time.monotonic() - start:.3f} sec")
            return tts_service

        init_tasks["tts"] = asyncio.create_task(init_tts())

    async def init_llm():
        start = time.monotonic()
        llm_service, aggregator = create_llm_service(llm_type, llm_key, messages)
        logger.info(f"[INIT][llm] done in {time.monotonic() - start:.3f} sec")
        return llm_service, aggregator

    init_tasks["llm"] = asyncio.create_task(init_llm())

    async def init_output_ws():
        start = time.monotonic()
        transport = create_websocket_transport("output", id, vad_analyzer=None)
        logger.info(f"[INIT][ws_output] done in {time.monotonic() - start:.3f} sec")
        return transport

    init_tasks["ws_output"] = asyncio.create_task(init_output_ws())

    async def init_rtvi():
        start = time.monotonic()
        rtvi = RTVIProcessor(config=RTVIConfig(config=[]))
        logger.info(f"[INIT][rtvi] done in {time.monotonic() - start:.3f} sec")
        return rtvi

    init_tasks["rtvi"] = asyncio.create_task(init_rtvi())

    # ----------------------------------------------------------------------
    # 2) await all initialization tasks
    # ----------------------------------------------------------------------
    results = await asyncio.gather(*init_tasks.values())

    # Unpack results in same order
    idx = 0
    if stt:
        stt_service, transport_input, vad_analyzer = results[idx]; idx += 1
    if tts:
        tts_service = results[idx]; idx += 1

    llm_service, llm_context_aggregator = results[idx]; idx += 1
    transport_output = results[idx]; idx += 1
    rtvi = results[idx]; idx += 1

    # ----------------------------------------------------------------------
    # 3) Pipeline 조립
    # ----------------------------------------------------------------------
    pipeline_stages = []

    if stt:
        pipeline_stages.append(transport_input.input())
        pipeline_stages.append(stt_service)

    pipeline_stages.append(rtvi)
    pipeline_stages.append(llm_context_aggregator.user())
    pipeline_stages.append(llm_service)

    if tts:
        pipeline_stages.append(tts_service)

    pipeline_stages.append(llm_context_aggregator.assistant())
    pipeline_stages.append(transport_output.output())

    pipeline = Pipeline(pipeline_stages)

    # Register tool
    tool_register(llm_service, id)

    # ----------------------------------------------------------------------
    # 4) Pipeline Task 생성
    # ----------------------------------------------------------------------
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
    logger.info(f"[INIT][task_create] done in {time.monotonic() - task_start:.3f} sec")

    # Configure WS error handlers
    async def handle_disconnect_or_error(name, transport, error=None):
        logger.error(f"{name} WebSocket disconnected or errored: {error}")
        await task.cancel()

    if stt:
        transport_input.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Input", transport_input))
        transport_input.event_handler("on_error")(partial(handle_disconnect_or_error, "Input", transport_input))

    transport_output.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Output", transport_output))
    transport_output.event_handler("on_error")(partial(handle_disconnect_or_error, "Output", transport_output))

    # Warmup frame
    await task.queue_frames([LLMRunFrame()])

    # Runner
    runner = PipelineRunner()

    init_total = time.monotonic() - total_start
    logger.info(f"[INIT][total] All initialization completed in {init_total:.3f} sec")

    # ----------------------------------------------------------------------
    # 5) Run the pipeline
    # ----------------------------------------------------------------------
    try:
        logger.info(f"[RUN] Starting pipeline id={id}")
        await runner.run(task)
    except asyncio.CancelledError:
        logger.info("[RUN] Pipeline cancelled")
    except Exception as e:
        logger.error(f"[RUN] Pipeline error: {e}")
    finally:
        logger.info("[CLEANUP] Cleaning up pipeline")

        if task:
            await task.cancel()

        if stt and transport_input:
            await transport_input.cleanup()

        if transport_output:
            await transport_output.cleanup()

        if llm_service:
            tool_unregister(llm_service)

        await task_manager.remove(id)

        logger.info(f"[CLEANUP] Pipeline cleaned. id={id}")


# ----------------------------
# 병렬 실행용 async 초기화 함수
# ----------------------------

async def async_create_vad():
    return SileroVADAnalyzer()   # 내부 비동기 아님 → 바로 반환

async def async_create_stt(name: str):
    return create_stt_service(name)

async def async_create_tts(name: str, voice_id: str):
    return create_tts_service(name, voice_id=voice_id)

async def async_create_llm(llm_type: str, llm_key: str, messages: list):
    # ⚠️ 반드시 매번 새로 생성해야 함 (당신 요구사항 반영)
    return create_llm_service(llm_type, llm_key, messages)

async def async_create_rtvi():
    return RTVIProcessor(config=RTVIConfig(config=[]))


# ----------------------------

def create_tts_service(name: str, **options):
    name = name.lower()
    voice_id = options.get("voice_id") or "default_voice_id"

    if name == "cartesia":
        return CartesiaTTSService(api_key=os.getenv("CARTESIA_API_KEY"), voice_id=voice_id)
    elif name == "elevenlabs":
        return ElevenLabsTTSService(api_key=os.getenv("ELEVENLABS_API_KEY"), voice_id=voice_id)
    else:
        raise ValueError(f"Unsupported TTS service: {name}")


def create_stt_service(name: str, **options):
    name = name.lower()
    if name == "deepgram":
        return DeepgramSTTService(api_key=os.getenv("DEEPGRAM_API_KEY"))
    else:
        raise ValueError(f"Unsupported STT service: {name}")


def create_llm_service(type: str, key: str, messages: list[dict], **options):
    if "." in type:
        service_name, model_name = type.split(".", 1)
    elif ":" in type:
        service_name, model_name = type.split(":", 1)
    else:
        raise ValueError(f"Wrong LLM format: {type}")

    service_name = service_name.lower()

    valid_messages = [m for m in messages if m.get("role") and m.get("content")]

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
