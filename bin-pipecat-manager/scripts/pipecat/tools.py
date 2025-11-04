from functools import partial
from pipecat.frames.frames import LLMMessagesFrame
import asyncio
import json

def tool_register(llm_service, transport):
    llm_service.register_function(
        "connect", 
        partial(tool_connect, transport),
        description="""
Establishes a call from a source endpoint to one or more destination endpoints. 
Use this when you need to connect a caller to specific endpoints like agents, conferences, or lines. 
The source and destination types can be agent, conference, extension, sip, or tel. 
Each endpoint must include a type and target, and optionally a target_name for display purposes.
""",
        parameters={
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
                                "description": "one of agent/conference/email/extension/line/sip/tel",
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
        },
    )
    
    llm_service.register_function(
        "message_send", 
        partial(tool_message_send, transport),
        description="""
Sends an SMS text message from a source telephone number to one or more destination telephone numbers.
Use this when you need to deliver SMS messages between phone numbers.
The source and destination types must be "tel".
""",
        parameters={
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
                                "description": "E.164 formatted phone number",
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
        }
    )

# email send
# params: destinations
# params: subject
# params: content

# connect to someone
async def tool_connect(params, transport):
    """
    Establishes a call from a source endpoint to one or more destination endpoints.
    """
    if isinstance(params, str):
        params = json.loads(params)
    print(f"Checking params: {params}")

    src = params.get("source")
    dsts = params.get("destinations", [])
    msg = f"Connecting {src} -> {', '.join([d['target'] for d in dsts])}"
    print(msg)

    await transport.queue_frame(LLMMessagesFrame(role="tool", content=json.dumps({
        "type": "connect",
        "options": {
            "source": src,
            "destinations": dsts
        },
        # "request_id": request_id
    })))

    return {"status": "connected", "destinations": dsts}


async def tool_message_send(params, transport):
    """
    Sends an SMS text message from a source telephone number to one or more destination numbers.
    """
    if isinstance(params, str):
        params = json.loads(params)
    print(f"Checking params: {params}")

    src = params.get("source")
    dsts = params.get("destinations", [])
    text = params.get("text")
    msg = f"SMS from {src} to {[d for d in dsts]}: {text}"
    print(msg)
    
    await transport.queue_frame(LLMMessagesFrame(role="tool", content=json.dumps({
        "type": "message_send",
        "options": {
            "source": src,
            "destinations": dsts
        },
        # "request_id": request_id
    })))

    return {"status": "sent", "text": text}
