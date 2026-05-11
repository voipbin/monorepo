.. _speaking-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Chargeable (per TTS synthesis request)
   * **Async:** Yes. ``POST https://api.voipbin.net/v1.0/speakings`` returns immediately with status ``initiating``. Poll ``GET https://api.voipbin.net/v1.0/speakings/{id}`` until status is ``active`` before calling ``POST https://api.voipbin.net/v1.0/speakings/{id}/say``.

The Speaking API enables you to inject synthesized speech into live calls and conferences in real-time. You can choose from multiple TTS providers, select specific voices, control audio direction (who hears the speech), and queue multiple speech segments for continuous playback.

Key capabilities:

- Inject synthesized speech into live calls and conferences
- Choose from multiple TTS providers (ElevenLabs, Google Cloud, AWS)
- Select specific voices or use provider defaults
- Control audio direction (caller only, callee only, or both)
- Queue multiple speech segments with flush control


How Speaking Works
------------------

The Speaking API synthesizes text-to-speech in real-time and delivers it to the specified audio target—either a call or a conference bridge.

::

   +--------+        +----------------+        +-------------+
   |  Call  |<-audio-|  TTS Engine    |<-text--|  Your App   |
   +--------+        +----------------+        | POST /say   |
                            |                  +-------------+
   +------------+           |
   | Conference |<--audio---+
   +------------+

**Key components:**

- **Audio Target:** A live call or conference that will receive the synthesized audio
- **TTS Engine:** The voice synthesis provider (ElevenLabs, Google Cloud, or AWS)
- **Your App:** Sends text via ``POST https://api.voipbin.net/v1.0/speakings/{id}/say`` to be synthesized and played


Speaking Lifecycle
------------------

Each speaking session progresses through a series of states from creation through termination.

::

   POST /speakings
        |
        v
   +-------------+                    +-------------+
   | initiating  |----setup done----->|   active    |
   +-------------+                    +------+------+
                                            |
                          POST /speakings/{id}/stop or call hangup
                                            |
                                            v
                                     +-------------+
                                     |   stopped   |
                                     +-------------+

**Status values:**

=========== ============
Status      Description
=========== ============
initiating  TTS session is being set up. Provider connection is being established. Do not call ``/say`` in this state.
active      TTS session is ready. You can send text via ``POST https://api.voipbin.net/v1.0/speakings/{id}/say``. Audio is being injected into the call.
stopped     TTS session has ended. Either stopped explicitly via ``POST https://api.voipbin.net/v1.0/speakings/{id}/stop`` or the call was hung up.
=========== ============

.. note:: **AI Implementation Hint**

   Always poll ``GET https://api.voipbin.net/v1.0/speakings/{id}`` until ``status`` is ``active`` before calling ``POST https://api.voipbin.net/v1.0/speakings/{id}/say``. Sending text while status is ``initiating`` will fail. Typical setup time is 2-3 seconds. Only one active speaking session per call is allowed -- create a new session only after the previous one is ``stopped``.


Providers
---------

The Speaking API supports multiple TTS providers, each with distinct voice libraries and pricing.

=========== ============
Provider    Description
=========== ============
elevenlabs  ElevenLabs TTS. High-quality neural voices with natural intonation. Default provider if omitted.
gcp         Google Cloud Text-to-Speech. Wide language support with WaveNet and Neural2 voices.
aws         Amazon Polly. Neural and standard voices with SSML support.
=========== ============

The ``provider`` field is optional and defaults to ``elevenlabs`` if omitted.


Direction
---------

Control who hears the synthesized speech by specifying the audio direction.

=========== ============
Direction   Description
=========== ============
in          Audio injected toward the caller (remote party hears it, local party does not).
out         Audio injected toward the callee/local side (local party hears it, remote party does not).
both        Audio injected to both sides of the call. Both parties hear the synthesized speech.
=========== ============


Reference Types
---------------

Attach the speaking session to either a call or a conference.

=========== ============
Type        Description
=========== ============
call        Attach TTS to a live call. The ``reference_id`` is a call ID from ``GET https://api.voipbin.net/v1.0/calls``.
confbridge  Attach TTS to a live conference. The ``reference_id`` is a conference ID from ``GET https://api.voipbin.net/v1.0/conferences``.
=========== ============


Best Practices
--------------

- Always wait for ``active`` status before sending text via ``POST https://api.voipbin.net/v1.0/speakings/{id}/say``
- Keep individual say requests under 5,000 characters to avoid timeout and latency
- Use ``flush`` to interrupt current speech when user speaks (barge-in scenarios)
- Clean up sessions explicitly with ``POST https://api.voipbin.net/v1.0/speakings/{id}/stop`` and ``DELETE https://api.voipbin.net/v1.0/speakings/{id}`` when finished
- Choose direction based on use case: ``both`` for announcements, ``out`` for agent coaching, ``in`` for IVR prompts
- Monitor session status actively—if the underlying call hangs up, the speaking session auto-stops


Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** Invalid language code or missing required fields.
    * **Fix:** Verify ``language`` is a valid BCP47 code (e.g., ``en-US``). Ensure ``reference_type`` and ``reference_id`` are provided.

* **404 Not Found:**
    * **Cause:** The speaking session ID does not exist or belongs to another customer.
    * **Fix:** Verify the UUID was obtained from a recent ``POST https://api.voipbin.net/v1.0/speakings`` or ``GET https://api.voipbin.net/v1.0/speakings`` response.

* **409 Conflict:**
    * **Cause:** Another speaking session is already active on this call, or the call is not in ``progressing`` status.
    * **Fix:** Stop the existing session via ``POST https://api.voipbin.net/v1.0/speakings/{id}/stop`` first. Verify the call status is ``progressing`` via ``GET https://api.voipbin.net/v1.0/calls/{id}``.


Related Documentation
---------------------

- :ref:`Call Overview <call-overview>` - Attaching TTS to calls
- :ref:`Conference Overview <conference-overview>` - Attaching TTS to conferences
- :ref:`Transcribe Overview <transcribe-overview>` - Speech-to-text (the listen counterpart)
- :ref:`Quickstart: Real-Time Voice <quickstart-realtime>` - End-to-end speaking and transcription example
