import common
import requests
from loguru import logger
from pipecat.services.llm_service import FunctionCallParams

tools = [
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
]


def tool_register(llm_service, pipecatcall_id):
    async def connect_wrapper(params: FunctionCallParams):
        return await tool_connect(params, pipecatcall_id)
    llm_service.register_function("connect", connect_wrapper)

    async def message_send_wrapper(params: FunctionCallParams):
        return await tool_message_send(params, pipecatcall_id)
    llm_service.register_function("message_send", message_send_wrapper)


# connect to someone
async def tool_connect(params: FunctionCallParams, pipecatcall_id):
    """
    Establishes a call from a source endpoint to one or more destination endpoints.
    """
    logger.info(f"Checking params: {params}")

    src = params.arguments.get("source")
    dsts = params.arguments.get("destinations", [])
    
    # send request
    http_url = common.PIPECATCALL_HTTP_URL + f"/{pipecatcall_id}/tools"
    http_params = {
        "type": "function",
        "function": {
            "name": "connect",
            "arguments": {
                "source": src,
                "destinations": dsts
            }, 
        },
    }
    logger.debug(f"HTTP Request URL: {http_url}, Params: {http_params}")

    try:
        response = requests.post(http_url, json=http_params)
        if response.status_code != 200:
            logger.error(f"HTTP request failed with status code {response.status_code}: {response.text}")
            await params.result_callback({
                "status": "error",
                "error": f"HTTP request failed with status code {response.status_code}"
            })
            return
        await params.result_callback({
            "status": "ok",
            "data": response.text,
        })

    except requests.exceptions.RequestException as e:
        logger.error(f"HTTP request exception: {e}")
        await params.result_callback({
            "status": "error",
            "error": str(e)
        })
        return


async def tool_message_send(params: FunctionCallParams, pipecatcall_id):
    """
    Sends an SMS text message from a source telephone number to one or more destination numbers.
    """
    logger.info(f"Checking params: {params}")

    src = params.arguments.get("source")
    dsts = params.arguments.get("destinations", [])
    text = params.arguments.get("text")

    # send request
    http_url = common.PIPECATCALL_HTTP_URL + f"/{pipecatcall_id}/tools"
    http_params = {
        "type": "function",
        "function": {
            "name": "message_send",
            "arguments": {
                "source": src,
                "destinations": dsts,
                "text": text,
            }, 
        },
    }
    logger.debug(f"HTTP Request URL: {http_url}, Params: {http_params}")

    try:
        response = requests.post(http_url, json=http_params)
        if response.status_code != 200:
            logger.error(f"HTTP request failed with status code {response.status_code}: {response.text}")
            await params.result_callback({
                "status": "error",
                "error": f"HTTP request failed with status code {response.status_code}"
            })
            return
        await params.result_callback({
            "status": "ok",
            "data": response.text,
        })

    except requests.exceptions.RequestException as e:
        logger.error(f"HTTP request exception: {e}")
        await params.result_callback({
            "status": "error",
            "error": str(e)
        })
        return
