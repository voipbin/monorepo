.. _ai-tutorial:

Tutorial
========

Simple AI Voice Assistant
-------------------------

Create a basic AI voice assistant that answers questions during a call. The AI will listen to the caller's speech, process it, and respond using text-to-speech.

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

This creates a call with an AI assistant that will:
1. Answer the incoming call
2. Listen to the caller's speech using STT (Speech-to-Text)
3. Process the input through the AI engine with the given prompt
4. Respond using TTS (Text-to-Speech)

AI Talk with Real-Time Conversation
------------------------------------

Use AI Talk for more natural, low-latency conversations powered by ElevenLabs. This enables interruption detection where the AI stops speaking when the caller starts talking.

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

AI Talk provides:
- **Interruption Detection**: Stops speaking when caller talks
- **Low Latency**: Streams responses in chunks for faster perceived response time
- **Natural Voice**: Uses ElevenLabs for high-quality voice output
- **Context Retention**: Remembers previous conversation exchanges

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
                    "type": "wait",
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
- Store the summary in ``${voipbin.ai_summary.result}``
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
- Match the ``language`` parameter with caller's expected language
- AI can detect and respond in multiple languages if not constrained

**Context Retention:**
- AI remembers conversation history within the same call
- Variables set during the call are available to AI
- Use context to build multi-turn conversations

**Error Handling:**
- Always include fallback actions after AI actions
- Handle cases where AI may not understand the input
- Provide clear instructions to callers about what they can ask

For more details on AI features and configuration, see :ref:`AI Overview <ai-overview>`.
