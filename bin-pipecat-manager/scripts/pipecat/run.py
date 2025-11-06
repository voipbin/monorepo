import asyncio
import os
import json
import common

from loguru import logger

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

# tool
from tools import tool_register, tools

async def run_pipeline(id: str, llm: str, tts: str, stt: str, voice_id: str = None, messages: list = None):
    logger.info(f"Connecting Pipecat client to Go WebSocket server. id: {id}")

    if messages is None:
        messages = []
    pipeline_stages = []
    uri = common.PIPECATCALL_WS_URL + f"/{id}/ws"
    logger.info(f"Establishing WebSocket connection to URI: {uri}")
    
    # transport
    transport = WebsocketClientTransport(
        uri=uri,
        params=WebsocketClientParams(
            serializer=ProtobufFrameSerializer(),
            audio_in_enabled=True,
            audio_out_enabled=True,
            add_wav_header=False,
            vad_analyzer=SileroVADAnalyzer(),
            session_timeout=60 * 3,
        )
    )    
    pipeline_stages.append(transport.input())
    
    # rtvi
    rtvi = RTVIProcessor(config=RTVIConfig(config=[]))
    pipeline_stages.append(rtvi)
    
    # Create STT service
    if stt:
        stt_service = create_stt_service(stt)
        pipeline_stages.append(stt_service)

    # Create LLM service
    llm_service, llm_context_aggregator = create_llm_service(llm, messages)
    pipeline_stages.append(llm_context_aggregator.user())
    pipeline_stages.append(llm_service)

    # Create TTS service
    if tts:
        tts_service = create_tts_service(tts, voice_id=voice_id)
        pipeline_stages.append(tts_service)

    # Add context aggregator assistant stage
    pipeline_stages.append(llm_context_aggregator.assistant())
    pipeline_stages.append(transport.output())
 
    # Build the pipeline
    pipeline = Pipeline(pipeline_stages)

    # Create RTVI processor and observer
    task = PipelineTask(
        pipeline,
        params=PipelineParams(
            enable_metrics=True,
            enable_usage_metrics=True,
        ),
        observers=[RTVIObserver(rtvi)],
    )
    
    # Register tool functions
    tool_register(llm_service, task, id)

    @transport.event_handler("on_disconnected")
    async def on_client_disconnected(transport, error):
        logger.info(f"Pipecat Client disconnected from Go server. Error: {error}")
        await task.cancel()

    @transport.event_handler("on_error")
    async def on_error(transport, error):
        logger.error(f"Pipecat Client WebSocket error: {error}")
        await task.cancel()

    runner = PipelineRunner()

    await task.queue_frames([LLMRunFrame()])

    try:
        await runner.run(task)
    except asyncio.CancelledError:
        logger.info("Pipecat client pipeline cancelled.")
    except Exception as e:
        logger.error(f"Pipecat client pipeline error: {e}")


def create_tts_service(name: str, **options):
    """
    Factory function to create a Pipecat TTS service instance
    based on the service name and optional parameters.
    """
    name = name.lower()

    if name == "cartesia":
        return CartesiaTTSService(
            api_key=options.get("api_key", os.getenv("CARTESIA_API_KEY")),
            voice_id=options.get("voice_id", "default_voice_id")
        )
    elif name == "elevenlabs":
        return ElevenLabsTTSService(
            api_key=options.get("api_key", os.getenv("ELEVENLABS_API_KEY")),
            voice_id=options.get("voice_id", "default_voice_id")
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


def create_llm_service(name: str, messages, **options):
    
    # validate name
    if "." not in name:
        raise ValueError(f"Wrong LLM: {name}. LLM argument must be in 'service.model' format, e.g., 'openai.gpt-4o-mini'")
    service_name, model_name = name.split(".", 1)
    service_name = service_name.lower()

    # validate messages
    valid_messages = []
    for msg in messages:
        if "role" not in msg or "content" not in msg or msg["role"] is None or msg["content"] is None:
            logger.warning(f"Skipping invalid message format: {msg}")
            continue
        valid_messages.append(msg)
    logger.info(f"Valid Messages Count: {len(valid_messages)}")

    res_llm = None
    res_context_aggregator = None
    if service_name == "openai":
        logger.info(f"Creating OpenAI LLM Service with model: {model_name}")
        res_llm = OpenAILLMService(
            api_key=options.get("api_key", os.getenv("OPENAI_API_KEY")),
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
