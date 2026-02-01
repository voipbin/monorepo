.. _ai-struct-ai:

AI
========

.. _ai-struct-ai-ai:

AI
--------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "engine_type": "<string>",
        "engine_model": "<string>",
        "engine_data": "<object>",
        "engine_key": "<string>",
        "init_prompt": "<string>",
        "tts_type": "<string>",
        "tts_voice_id": "<string>",
        "stt_type": "<string>",
        "tool_names": ["<string>"],
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* id: AI's unique identifier (UUID).
* customer_id: Customer's ID who owns this AI configuration.
* name: AI's display name.
* detail: AI's description or additional details.
* engine_type: AI's engine type (reserved for future use).
* engine_model: AI's LLM model. Format: ``<provider>.<model>``. See :ref:`Engine Models <ai-struct-ai-engine_model>`.
* engine_data: Provider-specific configuration data (JSON object).
* engine_key: API key for the LLM provider.
* init_prompt: Initial system prompt that defines the AI's behavior and persona.
* tts_type: Text-to-Speech provider. See :ref:`TTS Types <ai-struct-ai-tts_type>`.
* tts_voice_id: Voice ID for the selected TTS provider.
* stt_type: Speech-to-Text provider. See :ref:`STT Types <ai-struct-ai-stt_type>`.
* tool_names: List of enabled tool functions. See :ref:`Tool Functions <ai-struct-tool>`.

Example
+++++++

.. code::

    {
        "id": "a092c5d9-632c-48d7-b70b-499f2ca084b1",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Sales Assistant AI",
        "detail": "AI assistant for handling sales inquiries",
        "engine_type": "",
        "engine_model": "openai.gpt-4o",
        "engine_data": {},
        "engine_key": "sk-...",
        "init_prompt": "You are a friendly sales assistant. Help customers find the right products.",
        "tts_type": "elevenlabs",
        "tts_voice_id": "EXAVITQu4vr4xnSDxMaL",
        "stt_type": "deepgram",
        "tool_names": ["connect_call", "send_email", "stop_service"],
        "tm_create": "2024-02-09 07:01:35.666687",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

.. _ai-struct-ai-engine_model:

Engine Model
------------
The engine_model field specifies which LLM provider and model to use. Format: ``<provider>.<model>``.

**Supported Providers**

======================== ================================ =======================================
Provider                 Format                           Examples
======================== ================================ =======================================
OpenAI                   ``openai.<model>``               openai.gpt-4o, openai.gpt-4o-mini
Anthropic                ``anthropic.<model>``            anthropic.claude-3-5-sonnet
AWS Bedrock              ``aws.<model>``                  aws.claude-3-sonnet
Azure OpenAI             ``azure.<model>``                azure.gpt-4
Cerebras                 ``cerebras.<model>``             cerebras.llama3.1-8b
DeepSeek                 ``deepseek.<model>``             deepseek.deepseek-chat
Fireworks                ``fireworks.<model>``            fireworks.llama-v3-70b
Google Gemini            ``gemini.<model>``               gemini.gemini-1.5-pro
Grok                     ``grok.<model>``                 grok.grok-1
Groq                     ``groq.<model>``                 groq.llama3-70b-8192
Mistral                  ``mistral.<model>``              mistral.mistral-large
NVIDIA NIM               ``nvidia.<model>``               nvidia.llama3-70b
Ollama                   ``ollama.<model>``               ollama.llama3
OpenRouter               ``openrouter.<model>``           openrouter.meta-llama/llama-3-70b
Perplexity               ``perplexity.<model>``           perplexity.llama-3-sonar-large
Qwen                     ``qwen.<model>``                 qwen.qwen-max
SambaNova                ``sambanova.<model>``            sambanova.llama3-70b
Together AI              ``together.<model>``             together.meta-llama/Llama-3-70b
Dialogflow               ``dialogflow.<type>``            dialogflow.cx, dialogflow.es
======================== ================================ =======================================

**Common OpenAI Models**

==================== ======================================
Model                Description
==================== ======================================
gpt-4o               Latest GPT-4 Omni model (recommended)
gpt-4o-mini          Smaller, faster GPT-4 Omni variant
gpt-4-turbo          GPT-4 Turbo with vision capabilities
gpt-4                Original GPT-4 model
gpt-3.5-turbo        Fast and cost-effective model
o1                   OpenAI o1 reasoning model
o1-mini              Smaller o1 reasoning model
o3-mini              Latest o3 mini reasoning model
==================== ======================================

.. _ai-struct-ai-tts_type:

TTS Type
--------
Text-to-Speech provider for converting AI responses to audio.

================ =======================================
Type             Description
================ =======================================
elevenlabs       ElevenLabs high-quality voice synthesis (recommended)
deepgram         Deepgram Aura voices
openai           OpenAI TTS (alloy, echo, fable, etc.)
aws              AWS Polly voices
azure            Azure Cognitive Services TTS
google           Google Cloud Text-to-Speech
cartesia         Cartesia TTS
hume             Hume AI emotional TTS
playht           PlayHT voice synthesis
================ =======================================

**Default Voice IDs by TTS Type**

======================== ====================================
TTS Type                 Default Voice ID
======================== ====================================
elevenlabs               EXAVITQu4vr4xnSDxMaL (Rachel)
deepgram                 aura-2-thalia-en (Thalia)
openai                   alloy
aws                      Joanna
azure                    en-US-JennyNeural
google                   en-US-Wavenet-D
cartesia                 71a7ad14-091c-4e8e-a314-022ece01c121
======================== ====================================

.. _ai-struct-ai-stt_type:

STT Type
--------
Speech-to-Text provider for converting incoming audio to text.

================ =======================================
Type             Description
================ =======================================
deepgram         Deepgram speech recognition (recommended)
cartesia         Cartesia speech recognition
elevenlabs       ElevenLabs speech recognition
================ =======================================

.. _ai-struct-ai-tool_names:

Tool Names
----------
The tool_names field controls which tool functions the AI can invoke during conversations.

**Configuration Options**

======================== ========================================================
Value                    Description
======================== ========================================================
``["all"]``              Enable all available tool functions
``[]`` or ``null``       Disable all tool functions (AI can only converse)
``["tool1", "tool2"]``   Enable only the specified tools
======================== ========================================================

**Available Tools**

See :ref:`Tool Functions <ai-struct-tool>` for the complete list of tools and their descriptions.

Example configurations:

.. code::

    // Enable all tools
    "tool_names": ["all"]

    // Enable only call transfer and email
    "tool_names": ["connect_call", "send_email"]

    // Enable conversation control tools only
    "tool_names": ["stop_service", "stop_flow", "set_variables"]

    // Disable all tools (conversation-only AI)
    "tool_names": []
