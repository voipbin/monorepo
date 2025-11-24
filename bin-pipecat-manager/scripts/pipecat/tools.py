import common
import requests
import json
from loguru import logger
import aiohttp
import asyncio
from pipecat.services.llm_service import FunctionCallParams
from pipecat.frames.frames import FunctionCallResultProperties

tools = [
    {
        "type": "function",
        "function": {
            "name": "tool_finalize",
            "description": """
A general-purpose tool that triggers a follow-up LLM response at the appropriate point after 
tool execution (e.g., SMS, Email, or database updates). 
This tool should be called only once when a final response from the LLM is needed.
""",
            "parameters": {
                "type": "object",
                "properties": {},
                "required": []
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "connect",
            "description": """
Establishes a call from a source endpoint to one or more destination endpoints. 
Use this when you need to connect a caller to specific endpoints like agents, conferences, or lines. 
The source and destination types can be agent, conference, extension, sip, or tel. 
Each endpoint must include a type and target, and optionally a target_name for display purposes.
""",
            "parameters": {
                "type": "object",
                "properties": {
                    "source": {
                        "type": "object",
                        "properties": {
                            "type": {
                                "type":        "string",
                                "description": "one of agent/conference/extension/sip/tel",
                            },
                            "target": {
                                "type":        "string",
                                "description": "address endpoint",
                            },
                            "target_name": {
                                "type":        "string",
                                "description": "address's name",
                            },
                        },
                        "required": ["type", "target"],
                    },
                    "destinations": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "type": {
                                    "type":        "string",
                                    "description": "one of agent/conference/extension/line/sip/tel",
                                },
                                "target": {
                                    "type":        "string",
                                    "description": "address endpoint",
                                },
                                "target_name": {
                                    "type":        "string",
                                    "description": "address's name",
                                },
                            },
                            "required": ["type", "target"],
                        },
                    },
                },
                "required": ["destinations"],
            }
        },
    },
    {
        "type": "function",
        "function": {
            "name": "message_send",
            "description": """
Sends an SMS text message from a source telephone number to one or more destination telephone numbers.
Use this when you need to deliver SMS messages between phone numbers.
The source and destination types must be "tel".
""",
            "parameters": {
                "type": "object",
                "properties": {
                    "source": {
                        "type": "object",
                        "properties": {
                            "type": {
                                "type":        "string",
                                "enum":        ["tel"],
                                "description": "must be tel",
                            },
                            "target": {
                                "type":        "string",
                                "description": "+E.164 formatted phone number",
                            },
                            "target_name": {
                                "type":        "string",
                                "description": "optional display name for the number",
                            },
                        },
                        "required": ["type", "target"],
                    },
                    "destinations": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "type": {
                                    "type":        "string",
                                    "enum":        ["tel"],
                                    "description": "must be tel",
                                },
                                "target": {
                                    "type":        "string",
                                    "description": "+E.164 formatted phone number",
                                },
                                "target_name": {
                                    "type":        "string",
                                    "description": "optional display name for the number",
                                },
                            },
                            "required": ["type", "target"],
                        },
                    },
                    "text": {
                        "type":        "string",
                        "description": "SMS message content",
                    },
                },
                "required": ["destinations", "text"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "email_send",
            "description": "Sends an email with subject, content, and optional attachments to one or more destination email addresses.",
            "parameters": {
                "type": "object",
                "properties": {
                    "destinations": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "type": {
                                    "type":        "string",
                                    "enum":        ["email"],
                                    "description": "must be email",
                                },
                                "target": {
                                    "type":        "string",
                                    "description": "Email address",
                                },
                                "target_name": {
                                    "type":        "string",
                                    "description": "Optional display name",
                                },
                            },
                            "required": ["type", "target"],
                        },
                    },
                    "subject": {
                        "type": "string",
                        "description": "Email subject"
                    },
                    "content": {
                        "type": "string",
                        "description": "Email body content (HTML or plain text)"
                    },
                    "attachments": {
                        "type": "array",
                        "description": "Optional list of attachments",
                        "items": {
                            "type": "object",
                            "properties": {
                                "reference_type": {
                                    "type": "string",
                                    "enum":        ["recording"],
                                    "description": "Attachment reference type"
                                },
                                "reference_id": {
                                    "type": "string",
                                    "description": "UUID of referenced object"
                                }
                            },
                            "required": ["reference_type", "reference_id"]
                        }
                    }
                },
                "required": [
                    "destinations",
                    "subject",
                    "content"
                ]
            }
        }
    },
]

TOOLS = ["tool_finalize", "connect", "message_send", "email_send"]

def tool_register(llm_service, pipecatcall_id):

    for tool_name in TOOLS:

        if tool_name == "tool_finalize":
            async def wrapper(params: FunctionCallParams, _pipecatcall_id=pipecatcall_id):
                return await tool_finalize(params, _pipecatcall_id)
        else:
            async def wrapper(params: FunctionCallParams, _tool_name=tool_name, _pipecatcall_id=pipecatcall_id):
                return await tool_execute(_tool_name, params, _pipecatcall_id)

        llm_service.register_function(tool_name, wrapper)



def tool_unregister(llm_service):
    """Unregisters tools from the LLM service."""
    for func_name in TOOLS:
        llm_service.unregister_function(func_name)



async def tool_finalize(params: FunctionCallParams, pipecatcall_id: str):
    """Finalizes the tool execution and triggers a follow-up LLM response."""
    logger.info(f"[tool_finalize] Finalizing tool execution. pipecatcall_id: {pipecatcall_id}")

    properties = FunctionCallResultProperties(
        run_llm=True,  # Trigger LLM response after finalization
    )

    await params.result_callback(
        {
            "status": "ok",
            "data": {"message": "Tool execution finalized."},
        },
        properties=properties,
    )


async def tool_execute(tool_name: str, params: FunctionCallParams, pipecatcall_id: str):
    """Generic executor for tool calls (connect, message_send, etc)."""
    logger.info(f"[{tool_name}] Executing with params: {json.dumps(params.arguments, ensure_ascii=False)}")
    
    http_url = f"{common.PIPECATCALL_HTTP_URL}/{pipecatcall_id}/tools"
    http_body = {
        "id": params.tool_call_id,
        "type": "function",
        "function": {
            "name": tool_name,
            "arguments": json.dumps(params.arguments, ensure_ascii=False),
        },
    }
    logger.debug(f"[{tool_name}] POST {http_url} with body: {json.dumps(http_body, ensure_ascii=False)}")

    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(http_url, json=http_body, timeout=aiohttp.ClientTimeout(total=10)) as response:
                status = response.status
                content_type = response.headers.get("Content-Type", "")
                text = await response.text()

                if content_type.startswith("application/json"):
                    try:
                        data = json.loads(text)
                    except json.JSONDecodeError:
                        data = {"raw": text}
                else:
                    data = {"raw": text}

                if status >= 400:
                    logger.warning(f"[{tool_name}] HTTP {status} Error: {text[:500]}")
                    await params.result_callback({
                        "status": "error",
                        "error": f"HTTP {status}: {text}",
                    })
                    return

                logger.info(f"[{tool_name}] Success: {status}")
                properties = FunctionCallResultProperties(
                    run_llm=False,  # Do not run LLM again after tool execution
                )

                await params.result_callback(
                    {
                        "status": "ok",
                        "data": data,
                    },
                    properties=properties,
                )

    except requests.exceptions.RequestException as e:
        logger.error(f"[{tool_name}] Request failed: {e}")
        await params.result_callback({
            "status": "error",
            "error": str(e),
        })

    except asyncio.TimeoutError:
        logger.error(f"[{tool_name}] Request timed out after 10s")
        await params.result_callback({
            "status": "error",
            "error": "Request timed out",
        })

    except aiohttp.ClientError as e:
        logger.exception(f"[{tool_name}] Client error: {e}")
        await params.result_callback({
            "status": "error",
            "error": str(e),
        })

    except Exception as e:
        logger.exception(f"[{tool_name}] Unexpected error: {e}")
        await params.result_callback({
            "status": "error",
            "error": f"Unexpected error: {e}",
        })
