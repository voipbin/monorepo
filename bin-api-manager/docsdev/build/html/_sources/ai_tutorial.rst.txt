.. _ai-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before using AI features, you need:

* A valid authentication token (String). Obtain via ``POST /auth/login`` or use an accesskey from ``GET /accesskeys``.
* A source phone number in E.164 format (e.g., ``+15551234567``). Obtain one owned by your account via ``GET /numbers``.
* A destination phone number in E.164 format or an internal extension.
* An LLM provider API key (String). Obtain from your provider's dashboard (e.g., OpenAI, Anthropic).
* (Optional) A pre-created AI configuration (UUID). Create one via ``POST /ais`` or use inline action settings.
* (Optional) A flow ID (UUID). Create one via ``POST /flows`` or obtain from ``GET /flows``.

.. note:: **AI Implementation Hint**

   AI features use three external services: an LLM (e.g., OpenAI), a TTS provider (e.g., ElevenLabs), and an STT provider (e.g., Deepgram). Each incurs costs on both VoIPBIN credits and the external provider's billing. Verify your VoIPBIN balance via ``GET /billing-accounts`` and your provider API key validity before creating AI calls.

Simple AI Voice Assistant
-------------------------

Create a basic AI voice assistant that answers questions during a call. The AI will listen to the user's speech, process it, and respond using text-to-speech.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ],
            "actions": [
                {
                    "type": "answer"
                },
                {
                    "type": "ai",
                    "option": {
                        "initial_prompt": "You are a helpful customer service assistant. Answer questions politely and concisely.",
                        "language": "en-US",
                        "voice_type": "female"
                    }
                }
            ]
        }'

Response:

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",   // Save this as call_id
        "status": "dialing",
        "source": {"type": "tel", "target": "+15551234567"},
        "destination": {"type": "tel", "target": "+15559876543"},
        "direction": "outgoing",
        "tm_create": "2026-02-18T10:30:00Z"
    }

This creates a call with an AI assistant that will:

1. Answer the incoming call
2. Listen to the user's speech using STT (Speech-to-Text)
3. Process the input through the AI engine with the given prompt
4. Respond using TTS (Text-to-Speech)

AI Talk with Real-Time Conversation
------------------------------------

Use AI Talk for more natural, low-latency conversations powered by ElevenLabs. This enables interruption detection where the AI stops speaking when the user starts talking.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ],
            "actions": [
                {
                    "type": "answer"
                },
                {
                    "type": "ai_talk",
                    "option": {
                        "initial_prompt": "You are an expert sales representative for VoIPBIN. Help customers understand our calling and messaging platform. Be enthusiastic but professional.",
                        "language": "en-US",
                        "voice_type": "male"
                    }
                }
            ]
        }'

Response:

.. code::

    {
        "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",   // Save this as call_id
        "status": "dialing",
        "source": {"type": "tel", "target": "+15551234567"},
        "destination": {"type": "tel", "target": "+15559876543"},
        "direction": "outgoing",
        "tm_create": "2026-02-18T10:31:00Z"
    }

AI Talk provides:

- **Interruption Detection**: Stops speaking when user talks
- **Low Latency**: Streams responses in chunks for faster perceived response time
- **Natural Voice**: Uses ElevenLabs for high-quality voice output
- **Context Retention**: Remembers previous conversation exchanges

.. note:: **AI Implementation Hint**

   The ``ai_talk`` action type (not ``ai``) enables real-time voice interaction with interruption detection. Use ``ai_talk`` for live conversational AI. The older ``ai`` action type uses a simpler request-response pattern without interruption support and is recommended only for basic Q&A use cases.

AI with Custom Voice ID
------------------------

Customize the AI voice by specifying an ElevenLabs Voice ID using variables.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ],
            "actions": [
                {
                    "type": "answer"
                },
                {
                    "type": "variable_set",
                    "option": {
                        "key": "voipbin.tts.elevenlabs.voice_id",
                        "value": "21m00Tcm4TlvDq8ikWAM"
                    }
                },
                {
                    "type": "ai_talk",
                    "option": {
                        "initial_prompt": "You are a friendly receptionist. Greet callers warmly and help them with their inquiries.",
                        "language": "en-US"
                    }
                }
            ]
        }'

See :ref:`Built-in ElevenLabs Voice IDs <ai-overview>` for available voice options.

AI Summary for Call Transcription
----------------------------------

Generate an AI-powered summary of a call transcription. This is useful for post-call analysis and record-keeping.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ],
            "actions": [
                {
                    "type": "answer"
                },
                {
                    "type": "transcribe_start",
                    "option": {
                        "language": "en-US"
                    }
                },
                {
                    "type": "talk",
                    "option": {
                        "text": "Hello! This call is being transcribed and summarized. Please tell me about your experience with our service.",
                        "language": "en-US"
                    }
                },
                {
                    "type": "sleep",
                    "option": {
                        "duration": 30000
                    }
                },
                {
                    "type": "ai_summary",
                    "option": {
                        "source_type": "transcribe",
                        "source_id": "${voipbin.transcribe.id}"
                    }
                },
                {
                    "type": "talk",
                    "option": {
                        "text": "Thank you for your feedback. We have recorded and summarized your call.",
                        "language": "en-US"
                    }
                }
            ]
        }'

The AI summary will:
- Process the transcription from ``transcribe_start``
- Generate a structured summary of key points
- Store the summary in ``${voipbin.ai_summary.content}``
- Can be accessed via webhook or API after the call

Real-Time AI Summary
--------------------

Get AI summaries while the call is still active. Useful for live call monitoring and agent assistance.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/calls?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "source": {
                "type": "tel",
                "target": "+15551234567"
            },
            "destinations": [
                {
                    "type": "tel",
                    "target": "+15559876543"
                }
            ],
            "actions": [
                {
                    "type": "answer"
                },
                {
                    "type": "transcribe_start",
                    "option": {
                        "language": "en-US",
                        "real_time": true
                    }
                },
                {
                    "type": "ai_summary",
                    "option": {
                        "source_type": "call",
                        "source_id": "${voipbin.call.id}",
                        "real_time": true
                    }
                },
                {
                    "type": "connect",
                    "option": {
                        "source": {
                            "type": "tel",
                            "target": "+15551234567"
                        },
                        "destinations": [
                            {
                                "type": "tel",
                                "target": "+15551111111"
                            }
                        ]
                    }
                }
            ]
        }'

Real-time summaries provide:
- **Live Updates**: Summary updates as conversation progresses
- **Agent Assistance**: Provides context to agents joining mid-call
- **Call Monitoring**: Enables supervisors to quickly understand ongoing calls

Best Practices
--------------

**Initial Prompt Design:**
- Be specific about the AI's role and behavior
- Include constraints (e.g., "Keep responses under 30 seconds")
- Define the tone (professional, friendly, technical, etc.)

**Language Support:**
- AI supports multiple languages (see :ref:`supported languages <transcribe-overview-supported_languages>`)
- Match the ``language`` parameter with the user's expected language
- AI can detect and respond in multiple languages if not constrained

**Context Retention:**
- AI remembers conversation history within the same call
- Variables set during the call are available to AI
- Use context to build multi-turn conversations

**Error Handling:**
- Always include fallback actions after AI actions
- Handle cases where AI may not understand the input
- Provide clear instructions to users about what they can ask

Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** Invalid ``engine_model`` format or missing required action fields.
    * **Fix:** Verify ``engine_model`` uses ``<provider>.<model>`` format (e.g., ``openai.gpt-4o``). Ensure ``initial_prompt`` is provided.

* **402 Payment Required:**
    * **Cause:** Insufficient VoIPBIN account balance.
    * **Fix:** Check balance via ``GET /billing-accounts``. Top up before retrying.

* **AI not responding during call:**
    * **Cause:** LLM provider API key is invalid or rate-limited.
    * **Fix:** Verify the ``engine_key`` in your AI configuration. Check the provider's status page and rate limits.

* **No audio from AI:**
    * **Cause:** TTS provider credentials are invalid or the voice ID does not exist.
    * **Fix:** Verify ``tts_type`` and ``tts_voice_id``. Try using a default voice (omit ``tts_voice_id``).

For more details on AI features and configuration, see :ref:`AI Overview <ai-overview>`.
