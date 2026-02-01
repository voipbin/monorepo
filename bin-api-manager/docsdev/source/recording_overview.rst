.. _recording-overview:

Overview
========
VoIPBIN's Recording API enables you to capture, store, and manage audio from calls and conferences. Whether you need recordings for compliance, quality assurance, training, or analytics, the Recording API provides a complete solution for managing call audio throughout its lifecycle.

With the Recording API you can:

- Record calls and conferences in real-time
- Start and stop recordings programmatically or via flow actions
- Download recordings as audio files
- Export recordings in bulk for archival
- Manage recording lifecycle and storage


How Recording Works
-------------------
When you start a recording, VoIPBIN captures the audio stream and writes it to cloud storage. The recording continues until you stop it, the call ends, or the maximum duration is reached.

**Recording Architecture**

::

    +--------+        +----------------+        +-----------+
    |  Call  |--audio-->|   Recording  |--file-->|  Storage  |
    +--------+        |    Engine      |        |   (GCS)   |
                      +----------------+        +-----------+
    +------------+           |
    | Conference |--audio----+
    +------------+

**Key Components**

- **Audio Source**: The call or conference being recorded
- **Recording Engine**: Captures audio in real-time, handles encoding
- **Storage**: Google Cloud Storage for reliable, scalable file storage

**Recording Types**

+------------------+-------------------------------------------------------+
| Type             | Description                                           |
+==================+=======================================================+
| Call Recording   | Records a single call's audio (both directions mixed) |
+------------------+-------------------------------------------------------+
| Conference       | Records all participant audio mixed together          |
| Recording        |                                                       |
+------------------+-------------------------------------------------------+

**File Format**

- Format: WAV (PCM)
- Sample Rate: 8 kHz
- Channels: Mono (all audio mixed)
- Bit Depth: 16-bit


Recording Lifecycle
-------------------
Every recording moves through a predictable set of states from creation to availability.

**State Diagram**

::

    POST /recording_start or flow action
           |
           v
    +------------+                        +------------+
    |  starting  |------recording-------->|  recording |
    +------------+                        +-----+------+
                                                |
                           POST /recording_stop, hangup, or max duration
                                                |
                                                v
                                         +------------+
                                         |  stopping  |
                                         +-----+------+
                                               |
                                               v (file processing)
                                         +------------+
                                         | available  |
                                         +------------+

**State Descriptions**

+------------+------------------------------------------------------------------+
| State      | What's happening                                                 |
+============+==================================================================+
| starting   | Recording initialization. Audio capture is being set up.         |
+------------+------------------------------------------------------------------+
| recording  | Actively capturing audio. File is being written to storage.      |
+------------+------------------------------------------------------------------+
| stopping   | Recording is ending. Final audio is being flushed to storage.    |
+------------+------------------------------------------------------------------+
| available  | Recording is complete. File is ready for download or streaming.  |
+------------+------------------------------------------------------------------+

**Key Behaviors**

- States only move forward, never backward
- A recording in "available" state cannot be modified
- If the call/conference ends, active recordings automatically stop
- Maximum recording duration is 24 hours


Starting and Stopping Recordings
--------------------------------
VoIPBIN provides multiple ways to control recordings based on your use case.

**Method 1: Via Flow Action**

Use ``recording_start`` and ``recording_stop`` actions in your call flow for automatic recording control.

::

    Your Flow                    VoIPBIN                     Storage
        |                           |                           |
        | recording_start action    |                           |
        +-------------------------->|                           |
        |                           | Initialize recording      |
        |                           +-------------------------->|
        |                           |                           |
        |                           |<=====audio stream=======>|
        |                           |                           |
        | recording_stop action     |                           |
        +-------------------------->|                           |
        |                           | Finalize recording        |
        |                           +-------------------------->|
        |                           |                           |

**Example flow with recording:**

.. code::

    {
        "actions": [
            {
                "type": "answer"
            },
            {
                "type": "recording_start",
                "option": {
                    "format": "wav"
                }
            },
            {
                "type": "talk",
                "option": {
                    "text": "This call is being recorded for quality purposes."
                }
            },
            {
                "type": "connect",
                "option": {
                    "destinations": [{"type": "tel", "target": "+15551234567"}]
                }
            },
            {
                "type": "recording_stop"
            }
        ]
    }

**Method 2: Via API (Interrupt Method)**

Start or stop recording on an active call or conference programmatically.

**Start recording on a call:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/calls/<call-id>/recording_start?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "format": "wav"
        }'

**Stop recording on a call:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/calls/<call-id>/recording_stop?token=<token>'

**Start recording on a conference:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/conferences/<conference-id>/recording_start?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "format": "wav"
        }'

**When to Use Each Method**

+-------------------+----------------------------------------------------------------+
| Method            | Best for                                                       |
+===================+================================================================+
| Flow Action       | Automated recording based on call flow logic                   |
+-------------------+----------------------------------------------------------------+
| API (Interrupt)   | Dynamic control - start/stop based on external events          |
+-------------------+----------------------------------------------------------------+


Recording Storage
-----------------
Recordings are stored securely in Google Cloud Storage and accessible via the VoIPBIN API.

**Storage Architecture**

::

    Download:                               Stream:
    +------+   GET /recordings/{id}/file    +------+   GET (Range header)
    | Your |------------------------------->| Your |<- - - - - - - - - - ->
    | App  |<--------full file--------------| App  |       chunks
    +------+              |                 +------+           |
                          v                                    v
                   +-------------+                      +-------------+
                   |  Recording  |                      |  Recording  |
                   |    File     |                      |    File     |
                   +-------------+                      +-------------+

**Accessing Recordings**

*List all recordings:*

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/recordings?token=<token>'

*Get recording metadata:*

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/recordings/<recording-id>?token=<token>'

*Download recording file:*

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/recordings/<recording-id>/file?token=<token>' \
        --output recording.wav

**Storage Details**

+---------------------+----------------------------------------------------------+
| Aspect              | Details                                                  |
+=====================+==========================================================+
| Storage Location    | Google Cloud Storage (GCS)                               |
+---------------------+----------------------------------------------------------+
| Retention Period    | Configurable per customer (default: 30 days)             |
+---------------------+----------------------------------------------------------+
| Maximum Duration    | 24 hours per recording                                   |
+---------------------+----------------------------------------------------------+
| File Size           | ~1 MB per minute (8 kHz mono WAV)                        |
+---------------------+----------------------------------------------------------+

**Bulk Export**

For exporting large numbers of recordings, use the asynchronous bulk export API:

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/recordings/export?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "recording_ids": [
                "recording-id-1",
                "recording-id-2",
                "recording-id-3"
            ]
        }'

The export runs asynchronously and you'll receive a webhook when complete.


Recording and Calls/Conferences
-------------------------------
Understanding how recordings relate to calls and conferences helps you track and manage them effectively.

**Call Recording Relationship**

::

    +-------------------------------------------------------------------+
    |                           Call                                    |
    |                                                                   |
    |   recording_id: "abc-123"     <- Currently active recording       |
    |                                                                   |
    |   recording_ids: [            <- All recordings from this call    |
    |       "abc-123",                                                  |
    |       "def-456",                                                  |
    |       "ghi-789"                                                   |
    |   ]                                                               |
    |                                                                   |
    +-------------------------------------------------------------------+

- ``recording_id``: The currently active recording (empty if not recording)
- ``recording_ids``: History of all recordings made during this call

**Multiple Recordings Per Call**

You can start and stop recording multiple times during a single call:

::

    Call Timeline:

    |-----|=========|-----|=========|-----|=========|----->
       ^       ^       ^       ^       ^       ^
       |       |       |       |       |       |
    start   stop    start   stop    start   stop
    rec 1   rec 1   rec 2   rec 2   rec 3   rec 3

    Result: 3 separate recording files


Common Scenarios
----------------

**Scenario 1: Record Entire Call**

Start recording immediately when the call is answered.

.. code::

    {
        "actions": [
            {"type": "answer"},
            {"type": "recording_start"},
            {"type": "connect", "option": {"destinations": [...]}},
            {"type": "recording_stop"}
        ]
    }

**Scenario 2: Record Only After Consent**

Play a consent message, then start recording.

::

    Call answered
         |
         v
    +------------------+
    | "Press 1 to      |
    |  consent to      |
    |  recording"      |
    +--------+---------+
             |
        +----+----+
        |         |
     Pressed 1   Other
        |         |
        v         v
    Start       Continue
    Recording   without
        |       recording
        v
    Continue
    call

**Scenario 3: Conference Recording**

Record all participants in a conference.

::

    Conference
    +-------------------------------------------------------+
    |  +------+    +------+    +------+                     |
    |  |User A|    |User B|    |User C|                     |
    |  +--+---+    +--+---+    +--+---+                     |
    |     |           |           |                         |
    |     +-----+-----+-----+-----+                         |
    |           |                                           |
    |           v                                           |
    |     +----------+                                      |
    |     |Recording |  <- All audio mixed together         |
    |     |  File    |                                      |
    |     +----------+                                      |
    +-------------------------------------------------------+

**Scenario 4: Pause and Resume Recording**

Stop and start recording to skip sensitive information.

::

    Recording active
         |
         v
    "Please provide your credit card number"
         |
         v
    STOP recording
         |
         v
    Customer provides card number (not recorded)
         |
         v
    START recording (new recording file)
         |
         v
    Continue call


Best Practices
--------------

**1. Legal Compliance**

- **Consent**: Many jurisdictions require consent before recording. Always announce recordings.
- **Retention**: Define clear retention policies. Delete recordings when no longer needed.
- **Access Control**: Limit who can access recordings. Use VoIPBIN's permission system.

**2. Announcement Examples**

.. code::

    {
        "type": "talk",
        "option": {
            "text": "This call may be recorded for quality and training purposes.",
            "language": "en-US"
        }
    }

**3. Storage Management**

- Set appropriate retention periods for your use case
- Use bulk export for long-term archival
- Monitor storage usage to manage costs
- Delete recordings that are no longer needed

**4. Recording Quality**

- Ensure good network connectivity for clear audio
- Monitor for recording failures via webhooks
- Test recording playback regularly


Troubleshooting
---------------

**Recording Not Starting**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| No recording_id returned  | Verify call/conference is in "progressing"     |
|                           | status before starting recording               |
+---------------------------+------------------------------------------------+
| Permission denied         | Check API token has recording permissions      |
+---------------------------+------------------------------------------------+
| Recording starts but      | Check if another recording is already active   |
| immediately stops         | (only one active recording per call)           |
+---------------------------+------------------------------------------------+

**Empty or Corrupted Files**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| File size is 0 bytes      | Recording may have been stopped immediately    |
|                           | after starting. Ensure audio is flowing.       |
+---------------------------+------------------------------------------------+
| File won't play           | Verify file format. VoIPBIN uses WAV format.   |
|                           | Some players may not support 8kHz mono.        |
+---------------------------+------------------------------------------------+
| Audio is silent           | Check that audio was flowing during recording. |
|                           | Muted calls produce silent recordings.         |
+---------------------------+------------------------------------------------+

**Download Failures**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| 404 Not Found             | Recording may still be processing. Wait for    |
|                           | "available" status before downloading.         |
+---------------------------+------------------------------------------------+
| Timeout on large files    | Use streaming download with Range headers      |
|                           | for large recordings.                          |
+---------------------------+------------------------------------------------+
| Recording expired         | Recording exceeded retention period. Check     |
|                           | your retention settings.                       |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Call Overview <call-overview>` - Recording calls
- :ref:`Conference Overview <conference-overview>` - Recording conferences
- :ref:`Transcribe Overview <transcribe-overview>` - Transcribing recordings
