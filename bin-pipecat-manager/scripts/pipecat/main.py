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

    # load messages from messages_file
    messages = []
    try:
        with open(args.messages_file, "r", encoding="utf-8") as f:
            messages = json.load(f)
        logger.info(f"✅ Loaded {len(messages)} initial messages from {args.messages_file}")
    except FileNotFoundError:
        logger.info(f"⚠️ Warning: messages_file not found at {args.messages_file}. Starting with empty messages.")
    except json.JSONDecodeError:
        logger.info(f"❌ Error decoding JSON from {args.messages_file}. Starting with empty messages.")


    logger.info(f"Go WebSocket Server URL: {args.ws_server_url}")
    logger.info(f"LLM: {args.llm}")
    logger.info(f"TTS: {args.tts}")
    logger.info(f"STT: {args.stt}")

    # for debugging, print first 2 messages if available
    if messages:
        logger.info(f"Initial Messages (first 2): {messages[:2]}")
    else:
        logger.info("No initial messages loaded.")

    # run the pipeline with the loaded messages
    await run_pipeline(args, messages)


if __name__ == "__main__":
    try:
        asyncio.run(python_client_main())
    except asyncio.CancelledError:
        logger.info("Python client tasks cancelled.")
    except Exception as e:
        logger.info(f"Python client encountered an error: {e}")