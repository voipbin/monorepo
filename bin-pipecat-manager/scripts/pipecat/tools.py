from functools import partial
from pipecat.frames.frames import MessageFrame
import asyncio
import json

def tool_register(llm_service, transport):
    llm_service.register_function("connect", partial(tool_connect, transport))
    llm_service.register_function("message_send", partial(tool_message_send, transport))

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

    await transport.queue_frame(MessageFrame(role="tool", content=json.dumps({
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
    msg = f"SMS from {src['target']} to {[d['target'] for d in dsts]}: {text}"
    print(msg)
    
    await transport.queue_frame(MessageFrame(role="tool", content=json.dumps({
        "type": "connect",
        "options": {
            "source": src,
            "destinations": dsts
        },
        # "request_id": request_id
    })))

    return {"status": "sent", "text": text}
