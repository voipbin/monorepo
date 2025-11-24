import asyncio
import os
import json
import common

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


async def run_pipeline(id: str, llm_type: str, llm_key: str, tts: str, stt: str, voice_id: str = None, messages: list = None):
    logger.info(f"Starting Pipecat client pipeline. id: {id}, llm_type: {llm_type}, tts: {tts}, stt: {stt}, voice_id: {voice_id}")

    transport_input = None
    transport_output = None
    task = None

    if messages is None:
        messages = []
    pipeline_stages = []

    if stt:
        logger.info(f"Creating WebSocket transport for input. id: {id}")
        stt_service = create_stt_service(stt)
        vad_analyzer = SileroVADAnalyzer()
        transport_input = create_websocket_transport("input", id, vad_analyzer=vad_analyzer)

    if tts:
        tts_service = create_tts_service(tts, voice_id=voice_id)

    
    llm_service, llm_context_aggregator = create_llm_service(llm_type, llm_key, messages)
    transport_output = create_websocket_transport("output", id, vad_analyzer=None)
    rtvi = RTVIProcessor(config=RTVIConfig(config=[]))
    
    # Assemble pipeline stages
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

    # Build the pipeline
    pipeline = Pipeline(pipeline_stages)

    # Register tool functions
    tool_register(llm_service, id)

    # Create RTVI processor and observer
    logger.info(f"Starting Pipecat client pipeline task. id: {id}")
    task = PipelineTask(
        pipeline,
        params=PipelineParams(
            enable_metrics=True,
            enable_usage_metrics=True,
        ),
        observers=[RTVIObserver(rtvi)],
    )
    await task_manager.add(id, task)

    async def handle_disconnect_or_error(name, transport, error=None):
        logger.error(f"{name} WebSocket disconnected or errored: {error}")
        await task.cancel()

    if stt:
        transport_input.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Input", transport_input))
        transport_input.event_handler("on_error")(partial(handle_disconnect_or_error, "Input", transport_input))

    transport_output.event_handler("on_disconnected")(partial(handle_disconnect_or_error, "Output", transport_output))
    transport_output.event_handler("on_error")(partial(handle_disconnect_or_error, "Output", transport_output))
    runner = PipelineRunner()
    await task.queue_frames([LLMRunFrame()])

    try:
        logger.info(f"Running Pipecat client pipeline. id: {id}")
        await runner.run(task)
    except asyncio.CancelledError:
        logger.info("Pipecat client pipeline cancelled.")
    except Exception as e:
        logger.error(f"Pipecat client pipeline error: {e}")
    finally:
        logger.info(f"Cleaning up Pipecat client pipeline. id: {id}")
        if task:
            logger.info(f"Cancelling pipeline task (id={id})")
            await task.cancel()
        if transport_input:
            logger.info(f"Cleaning up input transport (id={id})")
            await transport_input.cleanup()
        if transport_output:
            logger.info(f"Cleaning up output transport (id={id})")
            await transport_output.cleanup()
        if llm_service:
            logger.info(f"Unregistering tool functions (id={id})")
            tool_unregister(llm_service)
        await task_manager.remove(id)
        logger.info(f"Pipeline cleaned up (id={id})")


def create_tts_service(name: str, **options):
    """
    Factory function to create a Pipecat TTS service instance
    based on the service name and optional parameters.
    """
    name = name.lower()
    
    voice_id = options.get("voice_id")
    if not voice_id:
        logger.warning(f"No voice_id specified for {name}, using default system voice.")
        voice_id = "default_voice_id"   # currently, we don't have a default voice id, so just a placeholder

    if name == "cartesia":
        return CartesiaTTSService(
            api_key=options.get("api_key", os.getenv("CARTESIA_API_KEY")),
            voice_id=voice_id,
        )
    elif name == "elevenlabs":
        return ElevenLabsTTSService(
            api_key=options.get("api_key", os.getenv("ELEVENLABS_API_KEY")),
            voice_id=voice_id,
        )
    else:
        raise ValueError(f"Unsupported TTS service: {name}")


def create_stt_service(name: str, **options):
    """
    Factory function to create a Pipecat STT service instance
    based on the service name and optional parameters.
    """
    name = name.lower()

    if name == "deepgram":
        return DeepgramSTTService(
            api_key=options.get("api_key", os.getenv("DEEPGRAM_API_KEY"))
        )
    else:
        raise ValueError(f"Unsupported STT service: {name}")


def create_llm_service(type: str, key: str, messages: list[dict], **options):
    
    # validate name
    if "." in type:
        service_name, model_name = type.split(".", 1)
    elif ":" in type:
        service_name, model_name = type.split(":", 1)
    else:
        raise ValueError(f"Wrong LLM: {type}. LLM argument must be in 'service.model' or 'service:model' format, e.g., 'openai.gpt-4o-mini' or 'openai:gpt-4o-mini'")
    service_name = service_name.lower()

    # validate messages
    valid_messages = []
    for msg in messages:
        if "role" not in msg or "content" not in msg or msg["role"] is None or msg["content"] is None:
            logger.warning(f"Skipping invalid message format: {msg}")
            continue
        valid_messages.append(msg)
    logger.debug(f"Valid Messages Count: {len(valid_messages)}")

    res_llm = None
    res_context_aggregator = None
    if service_name == "openai":
        logger.info(f"Creating OpenAI LLM Service with model: {model_name}")
        
        api_key = key if key else os.getenv("OPENAI_API_KEY")
        res_llm = OpenAILLMService(
            api_key=api_key,
            model=model_name
        )
        
        context = OpenAILLMContext(
            messages = valid_messages,
            tools = tools,
        )
        res_context_aggregator = res_llm.create_context_aggregator(context)
                
    else:
        raise ValueError(f"Unsupported LLM service: {service_name}")

    return res_llm, res_context_aggregator

def create_websocket_transport(direction: str, id: str, vad_analyzer: SileroVADAnalyzer = None):
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
