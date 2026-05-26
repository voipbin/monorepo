.. _aicall-struct-aicall:

AIcall
======

.. _aicall-struct-aicall-aicall:

AIcall
------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "assistance_type": "<string>",
        "assistance_id": "<string>",
        "ai_engine_model": "<string>",
        "ai_tts_type": "<string>",
        "ai_tts_voice_id": "<string>",
        "ai_stt_type": "<string>",
        "ai_vad_config": {},
        "ai_smart_turn_enabled": "<boolean>",
        "parameter": {},
        "activeflow_id": "<string>",
        "reference_type": "<string>",
        "reference_id": "<string>",
        "confbridge_id": "<string>",
        "current_member_id": "<string>",
        "status": "<string>",
        "stt_language": "<string>",
        "metadata": {},
        "tm_end": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The AI call's unique identifier. Returned when creating via flow action or listing via ``GET /aicalls``.
* ``customer_id`` (UUID): The customer who owns this AI call. Obtained from the ``id`` field of ``GET /customers``.
* ``assistance_type`` (enum string): The type of AI assistance. See :ref:`Assistance Type <aicall-struct-aicall-assistance-type>`.
* ``assistance_id`` (UUID): The ID of the AI configuration or team used for this call. Obtained from the ``id`` field of ``GET /ais`` or ``GET /ai-teams``.
* ``ai_engine_model`` (string): The LLM engine and model used (e.g., ``openai.gpt-4o``, ``anthropic.claude-3-5-sonnet``).
* ``ai_tts_type`` (string): The text-to-speech provider type (e.g., ``openai``, ``elevenlabs``, ``deepgram``, ``cartesia``).
* ``ai_tts_voice_id`` (string): The voice identifier used for text-to-speech output.
* ``ai_stt_type`` (string): The speech-to-text provider type (e.g., ``deepgram``, ``cartesia``).
* ``ai_vad_config`` (object): Voice Activity Detection configuration settings. May be null if using defaults.
* ``ai_smart_turn_enabled`` (boolean): Whether smart turn-taking is enabled for natural conversation flow.
* ``parameter`` (object): Additional key-value parameters passed to the AI engine.
* ``activeflow_id`` (UUID): The ID of the active flow associated with this AI call. Obtained from the ``id`` field of ``GET /activeflows``. Set to ``00000000-0000-0000-0000-000000000000`` if no active flow.
* ``reference_type`` (enum string): The type of resource this AI call is attached to. See :ref:`Reference Type <aicall-struct-aicall-reference-type>`.
* ``reference_id`` (UUID): The ID of the referenced resource (call, conversation, or task).
* ``confbridge_id`` (UUID): The conference bridge ID used for audio mixing. Obtained from the ``id`` field of ``GET /conferences``.
* ``current_member_id`` (UUID): The ID of the current member in the conference bridge.
* ``status`` (enum string): The AI call's current status. See :ref:`Status <aicall-struct-aicall-status>`.
* ``stt_language`` (string): The BCP47 language code used for speech-to-text (e.g., ``en-US``, ``ko-KR``).
* ``metadata`` (object): Generic key-value store attached to this AIcall. At call start time the
  key ``prompt_snapshots`` holds an array of ``PromptSnapshot`` objects (one per AI participant).
  Additional audit or operational data may appear under other keys in future releases.

  **PromptSnapshot fields:**

  * ``ai_id`` (string/UUID): ID of the AI configuration.
  * ``prompt_history_id`` (string/UUID): ID of the ``ai_ai_prompt_histories`` entry in effect
    at call start. Zero UUID if no history entry exists yet.
  * ``prompt`` (string): Final variable-substituted ``init_prompt`` as sent to the LLM.
  * ``member_id`` (string/UUID): Team member UUID for team calls; zero UUID for single-AI calls.
* ``tm_end`` (string, ISO 8601): Timestamp when the AI call ended.
* ``tm_create`` (string, ISO 8601): Timestamp when this AI call was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to this AI call.
* ``tm_delete`` (string, ISO 8601): Timestamp when this AI call was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. _aicall-struct-aicall-assistance-type:

Assistance Type
---------------

All possible values for the ``assistance_type`` field:

====== ===========
Type   Description
====== ===========
ai     A single AI agent configuration
team   An AI team configuration with multiple agents
====== ===========

.. _aicall-struct-aicall-reference-type:

Reference Type
--------------

All possible values for the ``reference_type`` field:

============== ===========
Type           Description
============== ===========
call           The AI call is attached to a phone call
conversation   The AI call is attached to a chat conversation
task           The AI call is running as a background task
============== ===========

.. _aicall-struct-aicall-status:

Status
------

All possible values for the ``status`` field:

============= ===========
Status        Description
============= ===========
initiating    The AI call is being initialized
progressing   The AI call is active and processing
pausing       The AI call is being paused
resuming      The AI call is resuming from a paused state
terminating   The AI call is being terminated
terminated    The AI call has ended. Final state.
============= ===========

Example
-------

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "assistance_type": "ai",
        "assistance_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "ai_engine_model": "openai.gpt-4o",
        "ai_tts_type": "elevenlabs",
        "ai_tts_voice_id": "21m00Tcm4TlvDq8ikWAM",
        "ai_stt_type": "deepgram",
        "ai_vad_config": null,
        "ai_smart_turn_enabled": true,
        "parameter": {},
        "activeflow_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "reference_type": "call",
        "reference_id": "d4e5f6a7-b8c9-0123-defa-234567890123",
        "confbridge_id": "e5f6a7b8-c9d0-1234-efab-345678901234",
        "current_member_id": "f6a7b8c9-d0e1-2345-fabc-456789012345",
        "status": "progressing",
        "stt_language": "en-US",
        "tm_end": "9999-01-01T00:00:00.000000Z",
        "tm_create": "2024-03-01T10:00:00.000000Z",
        "tm_update": "2024-03-01T10:00:05.000000Z",
        "tm_delete": "9999-01-01T00:00:00.000000Z"
    }
