.. _ai-overview: ai-overview

Overview
========
VoIPBin's  AI is a built-in AI agent that enables automated, intelligent voice interactions during live calls. Designed for seamless integration within VoIPBin's flow, the AI utilizes ChatGPT as its AI engine to process and respond to user inputs in real time. This allows developers to create dynamic and interactive voice experiences without requiring manual intervention.

How it works
============

Action component
----------------

The AI is integrated as one of the configurable components within a VoIPBin flow. When a call reaches a AI action, the system triggers the AI to generate a response based on the provided prompt. The response is then processed and played back to the caller using text-to-speech (TTS). If the response is in a structured JSON format, VoIPBin executes the defined actions accordingly.

.. image:: _static/images/ai_overview_overview.png
    :alt: AI component in action builder
    :align: center

TTS/STT + AI Engine
-------------------

VoIPBin's AI is built using TTS/STT + AI Engine, where speech-to-text (STT) converts spoken words into text, and text-to-speech (TTS) converts responses back into audio. The system processes these in real time, enabling seamless conversations.

.. image:: _static/images/ai_overview_stt_tts.png
    :alt: AI implementation using TTS/STT + AI Engine
    :align: center

Voice Detection and Play Interruption:
--------------------------------------

In addition to basic TTS and STT functionalities, VoIPBin incorporates voice detection to create a more natural conversational flow. While the AI is speaking (i.e., playing TTS media), if the system detects the caller's voice, it immediately stops the TTS playback and routes the caller's speech (via STT) to the AI engine. This play interruption feature ensures that if the user starts talking, their input is prioritized, enabling a dynamic interaction that more closely resembles a real conversation.

External AI Agent Integration
-----------------------------

For users who prefer to use external AI services, such as VAPI or other AI agent service providers, VoIPBin offers media stream access. This allows third-party AI engines to process voice data directly, enabling deeper customization and advanced AI capabilities.

Multiple AI Actions in a Flow
----------------------------------

VoIPBin allows multiple AI actions within a single flow. Developers can configure different AI interactions at various points, enabling flexible and context-aware automation.

Handling Responses
------------------

* Text String Response: The AI's response is played as speech using TTS.
* JSON Response: The AI returns a structured JSON array of action objects, which VoIPBin executes accordingly.
* Error Handling: If the AI generates an invalid JSON response, VoIPBin treats it as a normal text response and plays it via TTS.

Using the AI
=================

Initial Prompt
--------------

The initial prompt serves as the foundation for the AI's behavior. A well-crafted prompt ensures accurate and relevant responses. There is no limit to prompt length, but this should remain confidential for future considerations.

Example Prompt:
+++++++++++++++

.. code::

    Pretend you are an expert customer service agent.

    Please respond kindly.

    But, if you receive a request to connect to the agent, respond with the next message in JSON format.
    Do not include any explanations in the response.
    Only provide an RFC8259-compliant JSON response following this format without deviation.

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

Action Object Structure
-----------------------

See detail :ref:`here <flow-struct-action-action>`.

VoIPBin supports a wide range of actions. Developers should refer to VoIPBin's documentation for a complete list of available actions.

Technical Considerations
========================

Escalation to Live Agents
-------------------------

VoIPBin does not provide an automatic escalation mechanism for transferring calls to human agents. Instead, developers must configure AI responses accordingly by ensuring that AI logic returns a JSON action when escalation is required.

Logging & Debugging
-------------------

Developers can debug AI interactions through VoIPBin's transcription logs, which capture AI responses and interactions.

Current Limitations & Future Enhancements
-----------------------------------------

* TTS Customization: Currently, voice, language, and speed customization are not available but will be added in future updates.
* Multilingual Support: The AI currently supports only English, but additional language support is planned.
* Context Retention: Each AI request is processed independently, meaning there is no built-in conversation memory.

VoIPBin's AI feature offers a flexible and intelligent way to automate voice interactions within flows. By leveraging AI-powered responses and structured action execution, developers can enhance call experiences with minimal effort. As VoIPBin continues to evolve, future updates will introduce greater customization options and multilingual capabilities.
