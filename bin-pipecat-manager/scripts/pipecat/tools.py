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
    CONNECT_CALL = "connect_call"                   # Connects to endpoints
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
            "description": """Connects to another endpoint (person, department, or phone number).

WHEN TO USE:
- User asks to be transferred: "transfer me to...", "connect me to...", "put me through to..."
- User wants to speak to a person: "let me talk to a human", "I need an agent", "get me a representative"
- User requests a specific department: "sales", "support", "billing", "customer service"
- User provides a phone number to call: "call +1234567890", "dial my wife"

WHEN NOT TO USE:
- User mentions a person/department without requesting transfer (just discussing)
- User asks ABOUT a department but doesn't want to be connected
- User is asking for information you can provide directly

EXAMPLES:
- "Transfer me to sales" -> type="extension", target="sales"
- "Can you call my wife at 555-1234?" -> type="tel", target="+15551234"
- "I need to speak to a human" -> type="agent", target=appropriate agent
- "Put me through to billing" -> type="extension", target="billing"

run_llm: Set true to confirm verbally ("Connecting you now..."), false for silent transfer.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set true to speak after connecting (e.g., 'Connecting you now'). Set false for silent transfer.",
                        "default": False
                    },
                    "source": {
                        "type": "object",
                        "properties": {
                            "type": {
                                "type":        "string",
                                "description": "Source endpoint type: agent, conference, extension, sip, or tel",
                            },
                            "target": {
                                "type":        "string",
                                "description": "Source address/identifier (e.g., extension name, +E.164 phone number)",
                            },
                            "target_name": {
                                "type":        "string",
                                "description": "Display name for the source (optional)",
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
                                    "description": "Destination type: agent (human agent), conference (conference room), extension (department/extension), line, sip (SIP address), tel (phone number)",
                                },
                                "target": {
                                    "type":        "string",
                                    "description": "Destination address (e.g., 'sales', 'support', '+15551234567', 'sip:user@domain.com')",
                                },
                                "target_name": {
                                    "type":        "string",
                                    "description": "Display name for the destination (optional)",
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
            "description": """Sends an email to one or more email addresses.

WHEN TO USE:
- User explicitly requests email: "email me", "send me an email", "send that to my email"
- User asks for documents/information to be emailed
- User provides an email address for receiving information

WHEN NOT TO USE:
- User says "send me" or "message me" without specifying email (ask first: email or text?)
- User wants a text/SMS (use send_message instead)

EXAMPLES:
- "Email me the transcript" -> send email with transcript content
- "Send the receipt to john@example.com" -> send to that address
- "Can you send me that info?" -> ASK: "Would you like that by email or text message?"

run_llm: Set true to confirm ("I've sent that to your email"), false for silent send.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set true to confirm verbally after sending. Set false to send silently.",
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
                                    "description": "Must be 'email'",
                                },
                                "target": {
                                    "type":        "string",
                                    "description": "Email address (e.g., user@example.com)",
                                },
                                "target_name": {
                                    "type":        "string",
                                    "description": "Recipient display name (optional)",
                                },
                            },
                            "required": ["type", "target"],
                        },
                    },
                    "subject": {
                        "type": "string",
                        "description": "Email subject line"
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
                                    "description": "Type of attachment reference"
                                },
                                "reference_id": {
                                    "type": "string",
                                    "description": "UUID of the referenced object to attach"
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
            "description": """Stops media from a previous action that is currently playing on the call (internal tool).

WHEN TO USE:
- When AI/pipecat has finished loading and needs to stop hold music or greeting that was playing
- When a previous flow action's media playback should be stopped before AI starts speaking
- When transitioning from pre-recorded media to live AI conversation

WHEN NOT TO USE:
- To stop the AI's own speech (this is handled by the framework)
- User wants to end the conversation (use stop_service instead)
- User wants to hang up the call (use stop_flow instead)

DIFFERS FROM OTHER STOP TOOLS:
- stop_media = Stop previous action's media playback, AI conversation continues
- stop_service = End AI conversation, flow continues to next action
- stop_flow = Terminate everything, call ends

EXAMPLES:
- AI loaded and ready to speak -> stop_media to stop hold music, then greet user
- Previous action played announcement -> stop_media before AI takes over

run_llm: Set true to speak immediately after stopping media, false to stop silently.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set true to speak after stopping media. Set false to stop silently.",
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
            "description": """Sends an SMS text message to a phone number.

WHEN TO USE:
- User explicitly requests a text/SMS: "text me", "send me a text", "SMS me", "message my phone"
- User asks for information sent to their phone number
- User provides a phone number and asks for a message there

WHEN NOT TO USE:
- User says "message me" generically without specifying SMS (ask: email or text?)
- User wants an email (use send_email instead)
- User is discussing messaging but not requesting action

EXAMPLES:
- "Text me the confirmation number" -> send SMS with confirmation
- "Send an SMS to +1555123456 saying I'll be late" -> send that content
- "Can you message me the details?" -> ASK FIRST: "Would you like that as a text message or email?"

run_llm: Set true to confirm ("I've texted you the details"), false for silent send.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set true to confirm verbally after sending. Set false to send silently.",
                        "default": False
                    },
                    "source": {
                        "type": "object",
                        "properties": {
                            "type": {
                                "type":        "string",
                                "enum":        ["tel"],
                                "description": "Must be 'tel' for phone number",
                            },
                            "target": {
                                "type":        "string",
                                "description": "Source phone number in +E.164 format (e.g., +15551234567)",
                            },
                            "target_name": {
                                "type":        "string",
                                "description": "Display name for the source number (optional)",
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
                                    "description": "Must be 'tel' for phone number",
                                },
                                "target": {
                                    "type":        "string",
                                    "description": "Destination phone number in +E.164 format (e.g., +15551234567)",
                                },
                                "target_name": {
                                    "type":        "string",
                                    "description": "Display name for the recipient (optional)",
                                },
                            },
                            "required": ["type", "target"],
                        },
                    },
                    "text": {
                        "type":        "string",
                        "description": "The SMS message content to send",
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
            "description": """Ends the AI conversation and proceeds to the next action in the flow.

WHEN TO USE:
- User says goodbye and conversation is complete: "bye", "goodbye", "thanks, that's all"
- User indicates they're done: "I'm all set", "that's everything", "nothing else"
- AI has successfully completed its purpose (appointment booked, issue resolved)
- Natural conversation conclusion

WHEN NOT TO USE:
- User is frustrated but still needs help (de-escalate instead)
- Conversation has unresolved issues
- User wants to END THE ENTIRE CALL (use stop_flow instead)

DIFFERS FROM stop_flow:
- stop_service = SOFT STOP - End AI portion, flow continues to next action
- stop_flow = HARD STOP - Terminate everything, no further actions run

EXAMPLES:
- "Thanks, bye!" -> stop_service (natural end)
- "I'm done here" -> stop_service (user signals completion)
- After successfully booking appointment -> stop_service (task complete)
- "Great, that's all I needed" -> stop_service""",
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
            "description": """Immediately terminates the entire flow/call. Nothing else executes after this.

WHEN TO USE:
- User explicitly wants to end everything: "hang up", "end the call", "terminate this", "disconnect"
- Critical error requiring full termination
- Emergency stop needed

WHEN NOT TO USE:
- User just wants to end AI conversation (use stop_service instead)
- User says casual goodbye like "bye" or "thanks" (use stop_service instead)
- There are more flow actions that should execute after AI

DIFFERS FROM stop_service:
- stop_flow = HARD STOP - Terminates everything, no further actions run
- stop_service = SOFT STOP - Ends AI, flow continues normally to next action

EXAMPLES:
- "Hang up now" -> stop_flow
- "End this call immediately" -> stop_flow
- "Just disconnect" -> stop_flow
- "Terminate the call" -> stop_flow""",
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
            "description": """Saves key-value data to the flow context for later use (internal tool).

WHEN TO USE:
- Save information for downstream flow actions
- User provides data needed later: name, account number, preferences, choices
- Conversation reaches conclusions to record: appointment time, issue category, resolution
- Any data that subsequent flow actions will need

WHEN NOT TO USE:
- Information only needed for current response (no need to persist)
- Data already stored elsewhere

EXAMPLES:
- User says "My name is John Smith" -> set_variables({"customer_name": "John Smith"})
- User confirms "3pm works" -> set_variables({"appointment_time": "15:00"})
- AI categorizes issue -> set_variables({"issue_category": "billing"})
- User provides account number -> set_variables({"account_number": "12345"})

run_llm: Set true to continue conversation after saving, false to save silently.""",
            "parameters": {
                "type": "object",
                "description": "Parameters for setting variables in the flow context.",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set true to continue conversation after saving. Set false to save silently.",
                        "default": False
                    },
                    "variables": {
                        "type": "object",
                        "description": "Key-value pairs to save. Example: {'customer_name': 'John', 'issue_type': 'billing'}",
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
            "description": """Retrieves previously saved variables from the flow context (internal tool).

WHEN TO USE:
- Need context set earlier in the flow
- Need information from previous actions (e.g., confirmation number, customer info)
- User asks about something that should be in saved context
- Before performing an action that requires previously collected data

WHEN NOT TO USE:
- Information is already in current conversation history
- You're guessing if data exists (just try to retrieve it and handle if empty)

EXAMPLES:
- Need customer name collected earlier -> get_variables
- Previous action saved confirmation number -> get_variables to retrieve it
- User asks "what was my confirmation?" -> get_variables
- Need user's phone number for SMS -> get_variables

run_llm: Set true to respond using retrieved data, false for silent retrieval before another action.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set true to respond based on retrieved data. Set false for silent retrieval.",
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
            "description": """Retrieves message history from a specific AI call session (internal tool).

WHEN TO USE:
- Need message history from a different AI call (not current conversation)
- Building summaries of past conversations
- User asks about previous interactions: "what did we discuss last time?"
- Referencing a specific past call by ID

WHEN NOT TO USE:
- Current conversation history is sufficient (already in your context)
- Need saved variables, not messages (use get_variables instead)
- No specific aicall_id to query

EXAMPLES:
- User: "What did we discuss in my last call?" -> get_aicall_messages (if you have the ID)
- Generating summary of a previous call -> get_aicall_messages
- Need to reference specific past conversation -> get_aicall_messages

run_llm: Set true to respond based on retrieved messages, false for silent retrieval.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": {
                        "type": "boolean",
                        "description": "Set true to respond based on messages. Set false for silent retrieval.",
                        "default": False
                    },
                    "aicall_id": {
                        "type": "string",
                        "description": "UUID of the AI call whose message history should be retrieved"
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
        try:
            llm_service.unregister_function(tool_name)
        except KeyError:
            logger.debug(f"Tool '{tool_name}' was not registered or already removed.")
        except Exception as e:
            logger.warning(f"Error while unregistering tool '{tool_name}': {e}")


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
