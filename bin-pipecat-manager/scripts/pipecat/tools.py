import common
import json
from loguru import logger
import aiohttp
import asyncio
from enum import Enum

from pipecat.services.llm_service import FunctionCallParams
from pipecat.frames.frames import FunctionCallResultProperties

class ToolName(str, Enum):
    # NOTICE: The following tool names must match those defined in the ai-manager.
    CONNECT_CALL = "connect_call"                   # Connects caller to endpoints
    GET_VARIABLES = "get_variables"                 # Gets flow variables
    GET_AICALL_MESSAGES = "get_aicall_messages"     # Gets AI call messages
    SEND_EMAIL = "send_email"                       # Sends emails
    SEND_MESSAGE = "send_message"                   # Sends SMS messages
    SET_VARIABLES = "set_variables"                 # Sets flow variables
    STOP_FLOW = "stop_flow"                         # Stops current flow execution
    STOP_MEDIA = "stop_media"                       # Stops current media playback
    STOP_SERVICE = "stop_service"                   # Stops current AI talk and proceeds to next Action

TOOLNAMES = [tool.value for tool in ToolName]

tools = [
    {
        "type": "function",
        "function": {
            "name": ToolName.CONNECT_CALL.value,
            "description": """
Establishes a call from a source endpoint to one or more destination endpoints. 
Use this when you need to connect a caller to specific endpoints like agents, conferences, or lines. 
The source and destination types can be agent, conference, extension, sip, or tel. 
Each endpoint must include a type and target, and optionally a target_name for display purposes.
""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set to false if the AI should remain silent after the transfer initiates. Set to true if the AI needs to confirm the transfer verbally.",
                        "default": False
                    },
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
            "name": ToolName.SEND_EMAIL.value,
            "description": "Sends an email with subject, content, and optional attachments to one or more destination email addresses.",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set to false if the AI should remain silent after the transfer initiates. Set to true if the AI needs to confirm the transfer verbally.",
                        "default": False
                    },
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
    {
        "type": "function",
        "function": {
            "name": ToolName.STOP_MEDIA.value,
            "description": """
Stops the media currently playing on the active call. Use this to immediately halt any ongoing audio playback.
""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set to false if the AI should remain silent after the transfer initiates. Set to true if the AI needs to confirm the transfer verbally.",
                        "default": False
                    },
                },
                "required": []
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": ToolName.SEND_MESSAGE.value,
            "description": """
Sends an SMS text message from a source telephone number to one or more destination telephone numbers.
Use this when you need to deliver SMS messages between phone numbers.
The source and destination types must be "tel".
""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set to false if the AI should remain silent after the transfer initiates. Set to true if the AI needs to confirm the transfer verbally.",
                        "default": False
                    },
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
            "name": ToolName.STOP_SERVICE.value,
            "description": """
Stops the currently ongoing talk conversation and immediately proceeds to the next Action.
Use this when you want to terminate the current talk without any additional input.
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
            "name": ToolName.STOP_FLOW.value,
            "description": """
Immediately stops the currently ongoing flow execution.
Use this to completely terminate the current process without executing subsequent actions.
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
            "name": ToolName.SET_VARIABLES.value,
            "description": """
Saves one or more variables to the flow context.
Pass a dictionary object to the 'variables' argument where keys are variable names and values are the content to save.
""",
            "parameters": {
                "type": "object",
                "description": "Parameters for setting variables in the flow context.",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set to false if the AI should remain silent after the transfer initiates. Set to true if the AI needs to confirm the transfer verbally.",
                        "default": False
                    },
                    "variables": {
                        "type": "object",
                        "description": "A dictionary containing key-value pairs to be saved. Example: {'ai_summary': 'text...', 'status': 'done'}",
                        "additionalProperties": {
                            "type": "string"
                        }
                    }
                },
                "required": ["variables"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": ToolName.GET_VARIABLES.value,
            "description": """
Retrieves all currently set key-value variables for the current flow execution.
""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set to false if the AI should remain silent after the transfer initiates. Set to true if the AI needs to confirm the transfer verbally.",
                        "default": False
                    },
                },
                "required": []
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": ToolName.GET_AICALL_MESSAGES.value,
            "description": """
Retrieves all messages associated with the specified AI call.
Use this to fetch the complete message history for a given aicall_id.
""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set to false if the AI should remain silent after the transfer initiates. Set to true if the AI needs to confirm the transfer verbally.",
                        "default": False
                    },
                    "aicall_id": {
                        "type": "string",
                        "description": "The ID of the AI call whose messages should be retrieved."
                    }
                },
                "required": ["aicall_id"]
            }
        }
    },
]

def tool_register(llm_service, pipecatcall_id):
    def create_wrapper(tool_name, pipecatcall_id):
        async def wrapper(params: FunctionCallParams):
            return await tool_execute(tool_name, params, pipecatcall_id)
        return wrapper

    for tool_name in TOOLNAMES:
        wrapper = create_wrapper(tool_name, pipecatcall_id)
        llm_service.register_function(tool_name, wrapper)


def tool_unregister(llm_service):
    """Unregisters tools from the LLM service."""
    for tool_name in TOOLNAMES:
        llm_service.unregister_function(tool_name)


async def tool_execute(tool_name: str, params: FunctionCallParams, pipecatcall_id: str):
    """Generic executor for tool calls (connect, message_send, etc)."""
    
    args = params.arguments if isinstance(params.arguments, dict) else {}
    logger.info(f"[{tool_name}] Executing. Args: {json.dumps(args, ensure_ascii=False)}")

    should_run_llm = args.pop("run_llm", False)

    http_url = f"{common.PIPECATCALL_HTTP_URL}/{pipecatcall_id}/tools"
    http_body = {
        "id": params.tool_call_id,
        "type": "function",
        "function": {
            "name": tool_name,
            "arguments": json.dumps(args, ensure_ascii=False),
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
                    run_llm=should_run_llm,
                )

                await params.result_callback(
                    {
                        "status": "ok",
                        "data": data,
                    },
                    properties=properties,
                )

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
