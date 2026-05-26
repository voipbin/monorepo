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
        "engine_model": "<string>",
        "parameter": "<object>",
        "engine_key": "<string>",
        "rag_id": "<string>",
        "init_prompt": "<string>",
        "current_prompt_history_id": "<string>",
        "tts_type": "<string>",
        "tts_voice_id": "<string>",
        "stt_type": "<string>",
        "stt_language": "<string>",
        "vad_config": {
            "confidence": <number>,
            "start_secs": <number>,
            "stop_secs": <number>,
            "min_volume": <number>
        },
        "smart_turn_enabled": <boolean>,
        "tool_names": ["<string>"],
        "direct_hash": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The AI configuration's unique identifier. Returned when creating an AI via ``POST /ais`` or when listing AIs via ``GET /ais``.
* ``customer_id`` (UUID): The customer that owns this AI configuration. Obtained from the ``id`` field of ``GET /customers``.
* ``name`` (String, Required): A human-readable name for the AI configuration (e.g., ``"Sales Assistant"``).
* ``detail`` (String, Optional): A description of the AI's purpose or additional notes.
* ``engine_model`` (String, Required): The LLM provider and model. Format: ``<provider>.<model>`` (e.g., ``openai.gpt-4o``, ``anthropic.claude-3-5-sonnet``). See :ref:`Engine Models <ai-struct-ai-engine_model>`.
* ``parameter`` (Object, Optional): Custom key-value parameter data for the AI configuration. Supports flow variable substitution at runtime. Typically left as ``{}``.
* ``engine_key`` (String, Required): The API key for the LLM provider. Must be a valid key from the provider's dashboard.
* ``rag_id`` (UUID, Optional): The knowledge base ID for the ``search_knowledge`` tool. Obtained from the ``id`` field of ``GET https://api.voipbin.net/v1.0/rags``. When set, the AI assistant can search this knowledge base during voice calls. Set to ``00000000-0000-0000-0000-000000000000`` or omit to disable.
* ``init_prompt`` (String, Required): The system prompt that defines the AI's behavior, persona, and instructions. No enforced length limit.
* ``current_prompt_history_id`` (string/UUID): UUID of the most-recent ``ai_ai_prompt_histories``
  entry for this AI. Included in webhook events so callers can correlate each AI event with the
  exact prompt version that was active. Zero UUID (``00000000-0000-0000-0000-000000000000``) means
  no versioned prompt history has been recorded yet.
* ``tts_type`` (enum string, Required): Text-to-Speech provider. See :ref:`TTS Types <ai-struct-ai-tts_type>`.
* ``tts_voice_id`` (String, Optional): Voice ID for the selected TTS provider. If omitted, the default voice for the chosen TTS type is used. See default voices in :ref:`TTS Types <ai-struct-ai-tts_type>`.
* ``stt_type`` (enum string, Required): Speech-to-Text provider. See :ref:`STT Types <ai-struct-ai-stt_type>`.
* ``stt_language`` (String, Optional): STT language in BCP-47 format (e.g., ``ko-KR``, ``en-US``). Controls which language the Speech-to-Text engine listens for. When set, the STT provider is configured to recognize this specific language, improving accuracy for non-English calls. Empty string or omitted means auto-detect (provider default).
* ``vad_config`` (Object, Optional): Voice Activity Detection configuration. All fields are optional — omitted fields use Pipecat defaults. See :ref:`VAD Config <ai-struct-ai-vad_config>`.
* ``smart_turn_enabled`` (Boolean, Optional): Enable smart turn detection using Pipecat's LocalSmartTurnAnalyzerV3 for more natural turn-taking. When ``true``, the VAD ``stop_secs`` parameter is automatically forced to ``0.2`` regardless of ``vad_config`` settings. Defaults to ``false``. See :ref:`Smart Turn <ai-struct-ai-smart_turn>`.
* ``tool_names`` (Array of String, Optional): List of enabled tool functions. Use ``["all"]`` to enable all tools, ``[]`` to disable all tools, or list specific tool names. See :ref:`Tool Functions <ai-struct-tool>`.
* ``direct_hash`` (String): Hash for direct AI access. Empty string when direct access is disabled. When enabled, this hash forms the direct SIP URI: ``sip:direct.<hash>@sip.voipbin.net``. Regenerate via ``POST /ais/{id}/direct-hash-regenerate``.
* ``tm_create`` (String, ISO 8601): Timestamp when the AI configuration was created.
* ``tm_update`` (String, ISO 8601): Timestamp when the AI configuration was last updated.
* ``tm_delete`` (String, ISO 8601): Timestamp when the AI configuration was deleted, if applicable.

.. note:: **AI Implementation Hint**

   The ``engine_key`` field contains the LLM provider's API key. This key is write-only: it is accepted on ``POST /ais`` and ``PUT /ais`` but is **never returned** in ``GET`` responses for security. If you need to change the key, send a full ``PUT`` update with the new key.

.. note:: **AI Implementation Hint**

   A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` indicates the AI configuration has not been deleted and is still active. This sentinel value is used across all VoIPBIN resources to represent "not yet occurred."

Example
+++++++

.. code::

    {
        "id": "a092c5d9-632c-48d7-b70b-499f2ca084b1",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "name": "Sales Assistant AI",
        "detail": "AI assistant for handling sales inquiries",
        "engine_model": "openai.gpt-4o",
        "parameter": {},
        "engine_key": "sk-...",
        "rag_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "init_prompt": "You are a friendly sales assistant. Help customers find the right products.",
        "tts_type": "elevenlabs",
        "tts_voice_id": "EXAVITQu4vr4xnSDxMaL",
        "stt_type": "deepgram",
        "stt_language": "en-US",
        "vad_config": {
            "stop_secs": 0.5
        },
        "smart_turn_enabled": true,
        "tool_names": ["connect_call", "send_email", "stop_service"],
        "direct_hash": "",
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

.. _ai-struct-ai-stt_language:

STT Language
------------
The ``stt_language`` field specifies which language the Speech-to-Text engine should recognize. The value must be in BCP-47 format (e.g., ``en-US``, ``ko-KR``, ``ja-JP``).

When set, the STT provider is explicitly configured for the specified language, which improves recognition accuracy — especially for non-English conversations. When omitted or set to an empty string, the STT provider uses its default auto-detection behavior.

**Common BCP-47 Language Codes**

======================== ====================================
Language Code            Language
======================== ====================================
en-US                    English (United States)
en-GB                    English (United Kingdom)
ko-KR                    Korean
ja-JP                    Japanese
zh-CN                    Chinese (Simplified)
de-DE                    German
fr-FR                    French
es-ES                    Spanish (Spain)
pt-BR                    Portuguese (Brazil)
it-IT                    Italian
nl-NL                    Dutch
ru-RU                    Russian
ar-SA                    Arabic
hi-IN                    Hindi
pl-PL                    Polish
======================== ====================================

.. note:: **AI Implementation Hint**

   The ``stt_language`` is configured on the AI resource itself, not per-call. If you need different STT languages for different call scenarios, create separate AI configurations — one per language — and reference the appropriate ``ai_id`` in each flow action.

.. _ai-struct-ai-vad_config:

VAD Config
----------
Voice Activity Detection configuration for tuning speech detection sensitivity and timing.

All fields are optional. Omitted fields use Pipecat's native defaults.

================ ======== ===== ===== ====================================
Field            Default  Min   Max   Description
================ ======== ===== ===== ====================================
confidence       0.7      0.0   1.0   Minimum confidence threshold to detect voice.
start_secs       0.2      0.0   30.0  Duration in seconds of continuous speech needed to confirm speaking started.
stop_secs        0.2      0.0   30.0  Duration in seconds of silence needed to confirm speaking stopped.
min_volume       0.6      0.0   1.0   Minimum audio volume for voice detection.
================ ======== ===== ===== ====================================

.. note:: **AI Implementation Hint**

   When ``vad_config`` is ``null`` or omitted, Pipecat's native defaults apply (confidence=0.7, start_secs=0.2, stop_secs=0.2, min_volume=0.6). To keep the AI responsive but avoid cutting off speech mid-sentence, increase ``stop_secs`` (e.g., 0.5). To make the AI more patient before responding, increase both ``stop_secs`` and ``start_secs``.

.. _ai-struct-ai-smart_turn:

Smart Turn
----------
When ``smart_turn_enabled`` is ``true``, the Pipecat pipeline uses ``LocalSmartTurnAnalyzerV3`` — a local ONNX model that analyzes speech and transcription context to detect when the user has truly finished their turn, rather than pausing mid-sentence. This results in more natural conversations with fewer premature interruptions.

.. note:: **AI Implementation Hint**

   Smart Turn detection requires VAD ``stop_secs=0.2``. When ``smart_turn_enabled`` is ``true``, any ``stop_secs`` value in ``vad_config`` is silently overridden to ``0.2``. This value matches the model's training data and allows Smart Turn to dynamically adjust timing.

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
