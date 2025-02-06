.. _chatbot-overview: chatbot-overview

Overview
========
A chatbot is a computer program designed to simulate conversation with human users, typically over messaging platforms or voice calls. Chatbots use natural language processing (NLP) and artificial intelligence (AI) technologies to understand and interpret user requests and respond with appropriate actions.

In VoIPBin, the chatbot is integrated directly with the VoIP services. This enables the chatbot to interact with voice calls, allowing for seamless voice interactions during live calls. Currently, the AI engine uses ChatGPT to process and respond to user inputs.

You can set up the chatbot as one of the flow components in VoIPBin. When a call's active flow reaches the chatbot action, the VoIPBin system will trigger the chatbot to engage in the conversation, executing the chatbot's responses via text-to-speech (TTS).

.. image:: _static/images/chatbot_overview_overview.png
    :alt: Chatbot component in action builder

Init prompt
===========

The initial prompt is the first message given to a chatbot system to initiate a conversation or request information. It sets the context for the chatbot to generate a response.

When interacting with the VoIPBin chatbot, the initial prompt plays a critical role in determining the quality and relevance of the chatbot's response. The prompt can be tailored to a specific domain or task, and it can be structured or open-ended based on the desired outcome.

There is currently no length limit for the prompt, but this detail should remain confidential for future considerations.

Exmaple
-------

Here is a simple example of an initial prompt that requests a chatbot to generate a JSON response for connecting a call:

.. code::

    Pretend you are an expert customer service agent.

    Please respond kindly.

    But, if you receive a request to connect to the agent,
    respond with the next message in JSON format.
    Do not include any explanations in the response. Only provide a RFC8259 compliant JSON response following this format without deviation.

    [
        {
            "action": "connect",
            "option": {
                "source": {
                    "type": "tel",
                    "target": "+821100000001"
                },
                "destinations": [
                    {
                        "type": "tel",
                        "target": "+821100000002"
                    }
                ]
            }
        }
    ]

Guidelines for Effective Prompts
--------------------------------

Keep prompts clear and specific to ensure the chatbot generates accurate responses.

The format of the prompt can vary depending on the action required, but the primary goal is to provide context for the chatbot's response.

Standard prompt practices can be followed, but no strict limitations exist at this time.

Response Message
================

When the chatbot generates a response, it can take the form of a normal text string or a JSON formatted list of action objects.

* Text String Response: This is the usual form of a response, used for regular conversation. It can be played as audio using TTS technology.
* JSON Response: This is used when the chatbot generates a response containing instructions for executing specific actions. The response will be a structured JSON list of action objects.

The JSON format allows for structured communication between the chatbot and external systems. The list of action objects within the JSON response represents the actions to be performed.

Action Object Structure
-----------------------
Each action object should have the following basic fields:

* action: A string that defines the action to be executed (e.g., "connect", "transfer").
* option: An object containing additional parameters necessary for executing the action (e.g., source and destination details).

See detail :ref:`here <flow-struct-action-action>`.
