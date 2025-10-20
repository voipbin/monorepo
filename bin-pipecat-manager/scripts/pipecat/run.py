import asyncio
import os
import json

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

async def run_pipeline(ws_server_url: str, llm: str, tts: str, stt: str, voice_id: str = None, messages: list = []):
    logger.info(f"Connecting Pipecat client to Go WebSocket server at: {ws_server_url}")

    ws_transport = WebsocketClientTransport(
        uri=ws_server_url,
        params=WebsocketClientParams(
            serializer=ProtobufFrameSerializer(),
            audio_in_enabled=True,
            audio_out_enabled=True,
            add_wav_header=False,
            vad_analyzer=SileroVADAnalyzer(),
            session_timeout=60 * 3,
        )
    )

    stt_service = create_stt_service(stt)

    tts_service = create_tts_service(
        tts,
        voice_id=voice_id if voice_id else None,
    )

    llm_service = create_llm_server(llm)
    
    context_aggregator = create_context_aggregator(llm, messages)

    rtvi = RTVIProcessor(config=RTVIConfig(config=[]))

    pipeline = Pipeline(
        [
            ws_transport.input(),
            rtvi,
            stt_service,
            context_aggregator.user(),
            llm_service,
            tts_service,
            context_aggregator.assistant(),
            ws_transport.output(),
        ]
    )

    task = PipelineTask(
        pipeline,
        params=PipelineParams(
            enable_metrics=True,
            enable_usage_metrics=True,
        ),
        observers=[RTVIObserver(rtvi)],
    )

    @ws_transport.event_handler("on_disconnected")
    async def on_client_disconnected(transport, error):
        logger.info(f"Pipecat Client disconnected from Go server. Error: {error}")
        await task.cancel()

    @ws_transport.event_handler("on_error")
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


def create_llm_server(name: str, **options):
    """
    Factory function to create a Pipecat LLM service instance
    based on the argument in 'service.model' format.
    """
    if "." not in name:
        raise ValueError(f"Wrong LLM: {name}. LLM argument must be in 'service.model' format, e.g., 'openai.gpt-4o-mini'")

    service_name, model_name = name.split(".", 1)
    service_name = service_name.lower()

    if service_name == "openai":
        llm = OpenAILLMService(
            api_key=options.get("api_key", os.getenv("OPENAI_API_KEY")),
            model=model_name
        )
        llm.context_class = OpenAILLMContext
    else:
        raise ValueError(f"Unsupported LLM service: {service_name}")
    
    return llm


def create_context_aggregator(llm, messages):
    # logger.info(f"Executing create_context_aggregator. filepath: {filepath}")

    # messages = []
    # try:
    #     with open(filepath, "r", encoding="utf-8") as f:
    #         messages = json.load(f)
    #     logger.info(f"Loaded {len(messages)} initial messages from {filepath}")
    # except FileNotFoundError:
    #     logger.warning(f"Could not find messages_file. {filepath}. Starting with empty messages.")
    # except json.JSONDecodeError:
    #     logger.error(f"Could not decode JSON from {filepath}. Starting with empty messages.")

    for msg in messages:
        if "role" not in msg or "content" not in msg:
            raise ValueError("Each message must contain 'role' and 'content'")
    logger.info(f"Initial Messages (first 2): {messages[:2]}")

    context = llm.context_class(messages)
    context_aggregator = llm.create_context_aggregator(context)
    
    return context_aggregator