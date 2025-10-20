import asyncio
import os
from contextlib import asynccontextmanager
from typing import Any, Dict

import argparse
import json

from dotenv import load_dotenv
import sys
from loguru import logger

load_dotenv(override=True)

from run import run_pipeline

async def python_client_main():
    logger.info("--- Received raw arguments ---")
    logger.info(f"sys.argv: {sys.argv}")
    logger.info("----------------------------")

    parser = argparse.ArgumentParser()
    parser.add_argument("--ws_server_url", type=str, required=True, help="WebSocket URL of the Go server (e.g., ws://localhost:8080/ws)")
    parser.add_argument("--llm", type=str, required=True)
    parser.add_argument("--tts", type=str, required=True)
    parser.add_argument("--stt", type=str, required=True)
    parser.add_argument("--voice_id", type=str, required=False)
    parser.add_argument("--messages_file", type=str, required=True, help="Path to the JSON file containing initial messages for the LLM context.")
    args = parser.parse_args()

    logger.info(f"Go WebSocket Server URL: {args.ws_server_url}")
    logger.info(f"LLM: {args.llm}")
    logger.info(f"TTS: {args.tts}")
    logger.info(f"STT: {args.stt}")
    logger.info(f"Voice ID: {args.voice_id}")
    logger.info(f"Messages File: {args.messages_file}")

    # run the pipeline with the loaded messages
    await run_pipeline(args)


if __name__ == "__main__":
    try:
        asyncio.run(python_client_main())
    except asyncio.CancelledError:
        logger.info("Python client tasks cancelled.")
    except Exception as e:
        logger.error(f"Python client encountered an error: {e}")
