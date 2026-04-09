.. _speaking-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before using the Speaking API, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use your access key via ``?accesskey=<your-accesskey>``.
* An active call in ``progressing`` status. Create one via ``POST /calls`` and poll ``GET /calls/{id}`` until answered. Or an active conference via ``POST /conferences``.
* A language code in BCP47 format (e.g., ``en-US``, ``ko-KR``, ``ja-JP``).
* (Optional) A provider-specific voice ID. Defaults to the provider's default voice if omitted.

.. note:: **AI Implementation Hint**

   The call must be in ``progressing`` status before attaching a speaking session.
   Poll ``GET /calls/{id}`` until ``status`` is ``progressing``. If the call reaches
   ``hangup`` status, the call was not answered and you must retry. For conferences,
   ensure at least one participant has joined before attaching TTS.


Create a Speaking Session
-------------------------

Attach a TTS session to a live call. The session starts in ``initiating`` status while the provider connection is established.

.. code::

   $ curl --location --request POST 'https://api.voipbin.net/v1.0/speakings?token=<YOUR_AUTH_TOKEN>' \
   --header 'Content-Type: application/json' \
   --data-raw '{
       "reference_type": "call",
       "reference_id": "12f8f1c9-a6c3-4f81-93db-ae445dcf188f",
       "language": "en-US",
       "direction": "both"
   }'

   {
       "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
       "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
       "reference_type": "call",
       "reference_id": "12f8f1c9-a6c3-4f81-93db-ae445dcf188f",
       "language": "en-US",
       "provider": "elevenlabs",
       "voice_id": "",
       "direction": "both",
       "status": "initiating",
       "tm_create": "2025-06-15 14:30:00.123456",
       "tm_update": "2025-06-15 14:30:00.123456",
       "tm_delete": "9999-01-01 00:00:00.000000"
   }

Poll until the session becomes ``active``:

.. code::

   $ curl --location --request GET 'https://api.voipbin.net/v1.0/speakings/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'

   {
       "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
       "status": "active",
       ...
   }

Wait for ``active`` before proceeding. Typical setup time is 2-3 seconds.


Send Text to Speak
-------------------

Once the session is ``active``, send text to be synthesized and played into the call.

.. code::

   $ curl --location --request POST 'https://api.voipbin.net/v1.0/speakings/a1b2c3d4-e5f6-7890-abcd-ef1234567890/say?token=<YOUR_AUTH_TOKEN>' \
   --header 'Content-Type: application/json' \
   --data-raw '{
       "text": "Hello! This is your AI agent speaking. How can I help you today?"
   }'

   {
       "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
       "status": "active",
       "reference_type": "call",
       "reference_id": "12f8f1c9-a6c3-4f81-93db-ae445dcf188f"
   }

You can call ``/say`` multiple times to queue additional speech segments. Each segment is synthesized and played in order. Maximum text length per request is 5,000 characters.


Choose a TTS Provider
---------------------

Specify the ``provider`` field when creating a session to select a TTS engine. If omitted, ElevenLabs is used by default.

**ElevenLabs (default):**

.. code::

   {
       "reference_type": "call",
       "reference_id": "<call-id>",
       "language": "en-US",
       "provider": "elevenlabs",
       "direction": "both"
   }

ElevenLabs provides high-quality neural voices with natural intonation. This is the default provider—if ``provider`` is omitted, ElevenLabs is used.

**Google Cloud TTS:**

.. code::

   {
       "reference_type": "call",
       "reference_id": "<call-id>",
       "language": "en-US",
       "provider": "gcp",
       "direction": "both"
   }

Google Cloud TTS offers wide language support with WaveNet and Neural2 voices.

**Amazon Polly:**

.. code::

   {
       "reference_type": "call",
       "reference_id": "<call-id>",
       "language": "en-US",
       "provider": "aws",
       "direction": "both"
   }

Amazon Polly provides neural and standard voices with SSML support.


Select a Voice
--------------

Use the ``voice_id`` field to choose a specific voice from the provider's voice library.

.. code::

   $ curl --location --request POST 'https://api.voipbin.net/v1.0/speakings?token=<YOUR_AUTH_TOKEN>' \
   --header 'Content-Type: application/json' \
   --data-raw '{
       "reference_type": "call",
       "reference_id": "12f8f1c9-a6c3-4f81-93db-ae445dcf188f",
       "language": "en-US",
       "provider": "elevenlabs",
       "voice_id": "21m00Tcm4TlvDq8ikWAM",
       "direction": "both"
   }'

   {
       "id": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
       "provider": "elevenlabs",
       "voice_id": "21m00Tcm4TlvDq8ikWAM",
       "status": "initiating",
       ...
   }

The ``voice_id`` is provider-specific. Obtain available voice IDs from your TTS provider's documentation. If omitted, the provider's default voice for the specified language is used.


Control Audio Direction
-----------------------

The ``direction`` field controls who hears the synthesized speech.

**Both directions (announcements):**

Use ``"direction": "both"`` when both parties should hear the speech. Suitable for announcements, greetings, or AI agent conversations.

.. code::

   {
       "direction": "both"
   }

**Outgoing only (agent coaching):**

Use ``"direction": "out"`` so only the local party (callee) hears the speech. The remote caller does not. Suitable for real-time agent coaching or whisper prompts.

.. code::

   {
       "direction": "out"
   }

**Incoming only (IVR replacement):**

Use ``"direction": "in"`` so only the remote caller hears the speech. The local party does not. Suitable for IVR-style prompts or one-sided announcements.

.. code::

   {
       "direction": "in"
   }


Flush the Speech Queue
----------------------

Clear queued text and stop the currently playing audio.

.. code::

   $ curl --location --request POST 'https://api.voipbin.net/v1.0/speakings/a1b2c3d4-e5f6-7890-abcd-ef1234567890/flush?token=<YOUR_AUTH_TOKEN>'

   {
       "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
       "status": "active",
       "reference_type": "call",
       "reference_id": "12f8f1c9-a6c3-4f81-93db-ae445dcf188f"
   }

Use flush to implement barge-in behavior—when the user starts speaking, flush the queue and listen instead. After flushing, you can send new text via ``/say`` to continue the conversation.


Attach Speaking to a Conference
-------------------------------

Attach TTS to a conference so all participants hear the synthesized speech.

.. code::

   $ curl --location --request POST 'https://api.voipbin.net/v1.0/speakings?token=<YOUR_AUTH_TOKEN>' \
   --header 'Content-Type: application/json' \
   --data-raw '{
       "reference_type": "confbridge",
       "reference_id": "c0d1e2f3-a4b5-6c7d-8e9f-0a1b2c3d4e5f",
       "language": "en-US",
       "provider": "elevenlabs",
       "direction": "both"
   }'

   {
       "id": "d4e5f6a7-b8c9-0123-def4-567890123456",
       "reference_type": "confbridge",
       "reference_id": "c0d1e2f3-a4b5-6c7d-8e9f-0a1b2c3d4e5f",
       "language": "en-US",
       "provider": "elevenlabs",
       "voice_id": "",
       "direction": "both",
       "status": "initiating",
       "tm_create": "2025-06-15 15:00:00.123456",
       "tm_update": "2025-06-15 15:00:00.123456",
       "tm_delete": "9999-01-01 00:00:00.000000"
   }

The ``reference_id`` is a conference ID obtained from ``GET /conferences``. All conference participants hear the synthesized speech when ``direction`` is ``both``.


Stop and Delete a Speaking Session
----------------------------------

When finished, stop the session first, then delete it.

**Stop the session:**

.. code::

   $ curl --location --request POST 'https://api.voipbin.net/v1.0/speakings/a1b2c3d4-e5f6-7890-abcd-ef1234567890/stop?token=<YOUR_AUTH_TOKEN>'

   {
       "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
       "status": "stopped",
       ...
   }

**Delete the session:**

.. code::

   $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/speakings/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'

Always stop a session before deleting it. If the call is hung up, the speaking session is automatically stopped, but you should still delete it to clean up resources.


Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** Invalid language code, empty text in ``/say``, or missing required fields (``reference_type``, ``reference_id``).
    * **Fix:** Verify ``language`` is a valid BCP47 code (e.g., ``en-US``). Ensure ``text`` in ``/say`` is non-empty and under 5,000 characters.

* **404 Not Found:**
    * **Cause:** The speaking session ID does not exist or belongs to another customer.
    * **Fix:** Verify the UUID was obtained from a recent ``POST /speakings`` or ``GET /speakings`` response.

* **409 Conflict:**
    * **Cause:** Another speaking session is already active on this call, or the call is not in ``progressing`` status.
    * **Fix:** Stop the existing session via ``POST /speakings/{id}/stop`` first. Verify the call status is ``progressing`` via ``GET /calls/{id}``.
