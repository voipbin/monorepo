.. _transcribe-overview:

Overview
========
VoIPBIN's Transcription API converts spoken audio from calls and conferences into text in real-time. Whether you need transcripts for compliance, searchable call logs, AI analysis, or accessibility, the Transcription API delivers accurate text as conversations happen.

With the Transcription API you can:

- Transcribe calls and conferences in real-time
- Distinguish between incoming and outgoing speech
- Receive transcripts via webhooks or WebSocket
- Support 70+ languages and regional variants
- Integrate with AI systems for sentiment analysis and summarization


How Transcription Works
-----------------------
When you start transcription, VoIPBIN captures audio from the call or conference, sends it to a speech-to-text (STT) engine, and delivers the resulting text to your application.

**Transcription Architecture**

::

    +--------+        +----------------+        +------------+
    |  Call  |--audio-->|     STT      |--text-->|  Webhook   |
    +--------+        |    Engine      |        |     or     |
                      +----------------+        | WebSocket  |
    +------------+           |                  +------------+
    | Conference |--audio----+                        |
    +------------+                                    v
                                              +------------+
                                              |  Your App  |
                                              +------------+

**Key Components**

- **Audio Source**: The call or conference being transcribed
- **STT Engine**: Google Cloud Speech-to-Text for accurate recognition
- **Delivery**: Webhooks (push) or WebSocket (subscribe) to your application

**Transcription Types**

+---------------------+-------------------------------------------------------+
| Type                | Description                                           |
+=====================+=======================================================+
| Call Transcription  | Transcribes a single call with direction detection    |
+---------------------+-------------------------------------------------------+
| Conference          | Transcribes all participants (direction indicates     |
| Transcription       | speaker relative to conference)                       |
+---------------------+-------------------------------------------------------+


Transcription Lifecycle
-----------------------
Transcription runs continuously while active, generating transcript segments as speech is detected.

**Lifecycle Diagram**

::

    POST /transcribes or flow action
           |
           v
    +-------------+                         +-------------+
    |  starting   |------active------------>| transcribing|
    +-------------+                         +------+------+
                                                   |
                              POST /transcribe_stop, hangup, or timeout
                                                   |
                                                   v
                                            +-------------+
                                            |   stopped   |
                                            +-------------+

**State Descriptions**

+---------------+------------------------------------------------------------------+
| State         | What's happening                                                 |
+===============+==================================================================+
| starting      | Transcription initialization. STT engine is connecting.          |
+---------------+------------------------------------------------------------------+
| transcribing  | Actively processing audio. Transcripts are being generated.      |
+---------------+------------------------------------------------------------------+
| stopped       | Transcription has ended. No more transcripts will be generated.  |
+---------------+------------------------------------------------------------------+

**Transcript Delivery Flow**

::

    Call Audio          VoIPBIN STT           Your App
        |                    |                    |
        |====audio chunk====>|                    |
        |                    | process            |
        |                    |----+               |
        |                    |<---+               |
        |                    |                    |
        |                    | transcript_created |
        |                    +------------------->|
        |                    |                    |
        |====audio chunk====>|                    |
        |                    | process            |
        |                    +------------------->|
        |                    |                    |

Each transcript segment is delivered as soon as speech is recognized, enabling real-time processing.


Starting Transcription
----------------------
VoIPBIN provides two methods to start transcription based on your use case.

**Method 1: Via Flow Action**

Use ``transcribe_start`` and ``transcribe_stop`` actions in your call flow for automatic control.

::

    Your Flow                    VoIPBIN                     Your App
        |                           |                           |
        | transcribe_start action   |                           |
        +-------------------------->|                           |
        |                           | Initialize STT            |
        |                           |                           |
        |                           |<====audio stream====      |
        |                           |                           |
        |                           | transcript_created        |
        |                           +-------------------------->|
        |                           |                           |
        | transcribe_stop action    |                           |
        +-------------------------->|                           |
        |                           |                           |

**Example flow with transcription:**

.. code::

    {
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
                    "text": "Hello, how can I help you today?"
                }
            },
            {
                "type": "connect",
                "option": {
                    "destinations": [{"type": "tel", "target": "+15551234567"}]
                }
            },
            {
                "type": "transcribe_stop"
            }
        ]
    }

See detail :ref:`here <flow-struct-action-transribe_start>`.

**Method 2: Via API (Interrupt Method)**

Start transcription on an active call or conference programmatically.

**Start transcription:**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/transcribes?token=<token>' \
        --header 'Content-Type: application/json' \
        --data '{
            "reference_type": "call",
            "reference_id": "8c71bcb6-e7e7-4ed2-8aba-44bc2deda9a5",
            "language": "en-US",
            "direction": "both"
        }'

**Parameters:**

+------------------+----------------------------------------------------------------+
| Parameter        | Description                                                    |
+==================+================================================================+
| reference_type   | Type of resource: ``call`` or ``conference``                   |
+------------------+----------------------------------------------------------------+
| reference_id     | UUID of the call or conference                                 |
+------------------+----------------------------------------------------------------+
| language         | Language code (e.g., ``en-US``, ``ko-KR``)                     |
+------------------+----------------------------------------------------------------+
| direction        | Which audio to transcribe: ``in``, ``out``, or ``both``        |
+------------------+----------------------------------------------------------------+

**When to Use Each Method**

+-------------------+----------------------------------------------------------------+
| Method            | Best for                                                       |
+===================+================================================================+
| Flow Action       | Automated transcription based on call flow logic               |
+-------------------+----------------------------------------------------------------+
| API (Interrupt)   | Dynamic control - start/stop based on external events          |
+-------------------+----------------------------------------------------------------+


Receiving Transcripts
---------------------
VoIPBIN delivers transcripts to your application via webhooks or WebSocket subscription.

**Webhook Delivery (Push)**

Configure a webhook URL in your customer settings to receive ``transcript_created`` events automatically.

::

    VoIPBIN                           Your App
        |                                 |
        | POST /your-webhook-endpoint     |
        | {transcript_created event}      |
        +-------------------------------->|
        |                                 |
        |            200 OK               |
        |<--------------------------------+
        |                                 |

**Webhook Payload:**

.. code::

    {
        "type": "transcript_created",
        "data": {
            "id": "9d59e7f0-7bdc-4c52-bb8c-bab718952050",
            "transcribe_id": "8c5a9e2a-2a7f-4a6f-9f1d-debd72c279ce",
            "direction": "out",
            "message": "Hello, this is transcribe test call.",
            "tm_transcript": "0001-01-01 00:00:08.991840",
            "tm_create": "2024-04-04 07:15:59.233415"
        }
    }

**WebSocket Subscription (Subscribe)**

Subscribe to transcript events via WebSocket for real-time streaming.

::

    Your App                          VoIPBIN
        |                                 |
        | WebSocket connect               |
        +-------------------------------->|
        |                                 |
        | Subscribe to transcript events  |
        +-------------------------------->|
        |                                 |
        |<======= transcript events =====>|
        |<======= transcript events =====>|
        |                                 |
        | Unsubscribe                     |
        +-------------------------------->|
        |                                 |

**Comparison: Webhook vs WebSocket**

+-------------------+--------------------------------+--------------------------------+
| Aspect            | Webhook                        | WebSocket                      |
+===================+================================+================================+
| Connection        | VoIPBIN initiates POST         | Your app maintains connection  |
+-------------------+--------------------------------+--------------------------------+
| Latency           | Higher (HTTP overhead)         | Lower (persistent connection)  |
+-------------------+--------------------------------+--------------------------------+
| Reliability       | Retry on failure               | Must handle reconnection       |
+-------------------+--------------------------------+--------------------------------+
| Best for          | Simple integration, batch      | Real-time UI, low-latency      |
|                   | processing                     | applications                   |
+-------------------+--------------------------------+--------------------------------+


.. _transcribe-overview-transcription:

Understanding Transcript Direction
----------------------------------
Each transcript includes a ``direction`` field indicating whether the speech was incoming or outgoing relative to VoIPBIN.

**Direction Detection**

::

    +----------+                             +---------+
    |  Caller  |-----> direction: "in" ----->| VoIPBIN |
    |          |                             |         |
    |          |<---- direction: "out" <-----|         |
    +----------+                             +---------+

**Example Conversation:**

::

    [in]  "Hello, I need help with my account"
    [out] "Sure, I can help you with that"
    [in]  "My account number is 12345"
    [out] "Let me look that up for you"

**Direction Values**

+-------------+----------------------------------------------------------------+
| Direction   | Meaning                                                        |
+=============+================================================================+
| in          | Audio from the caller/remote party toward VoIPBIN             |
+-------------+----------------------------------------------------------------+
| out         | Audio from VoIPBIN toward the caller/remote party             |
+-------------+----------------------------------------------------------------+

**Transcript Data Structure:**

.. code::

    [
        {
            "id": "06af78f0-b063-48c0-b22d-d31a5af0aa88",
            "transcribe_id": "bbf08426-3979-41bc-a544-5fc92c237848",
            "direction": "in",
            "message": "Hi, good to see you. How are you today.",
            "tm_transcript": "0001-01-01 00:01:04.441160",
            "tm_create": "2024-04-01 07:22:07.229309"
        },
        {
            "id": "3c95ea10-a5b7-4a68-aebf-ed1903baf110",
            "transcribe_id": "bbf08426-3979-41bc-a544-5fc92c237848",
            "direction": "out",
            "message": "Welcome to the transcribe test scenario.",
            "tm_transcript": "0001-01-01 00:00:43.116830",
            "tm_create": "2024-04-01 07:17:27.208337"
        }
    ]


Working with Transcripts
------------------------

**Timestamp Fields**

+----------------+----------------------------------------------------------------+
| Field          | Description                                                    |
+================+================================================================+
| tm_transcript  | Time offset within the call when speech occurred               |
+----------------+----------------------------------------------------------------+
| tm_create      | Absolute timestamp when transcript was created                 |
+----------------+----------------------------------------------------------------+

**Combining Transcripts into Conversation**

To reconstruct a conversation, sort transcripts by ``tm_transcript``:

::

    Transcripts received (order of delivery):
    [out] 00:00:05 "Welcome to VoIPBIN support"
    [in]  00:00:12 "Hi, I have a billing question"
    [out] 00:00:18 "I'd be happy to help"
    [in]  00:00:08 "Hello?"

    Sorted by tm_transcript:
    [out] 00:00:05 "Welcome to VoIPBIN support"
    [in]  00:00:08 "Hello?"
    [in]  00:00:12 "Hi, I have a billing question"
    [out] 00:00:18 "I'd be happy to help"

**Storing Transcripts**

For long-term storage, consider:

- Store raw transcripts with all metadata
- Index by ``transcribe_id`` to group by session
- Use ``direction`` for speaker attribution
- Create searchable text indexes on ``message`` field


Common Scenarios
----------------

**Scenario 1: Real-Time Call Transcription**

Transcribe a call from start to finish with webhook delivery.

::

    Call starts
         |
         v
    +--------------------+
    | transcribe_start   |
    | language: "en-US"  |
    +--------+-----------+
             |
             v
    +===================+
    | Call in progress  |------> transcript_created events
    +===================+           to your webhook
             |
             v
    +--------------------+
    | Call ends          |
    | (auto-stop)        |
    +--------------------+

**Scenario 2: Conference with Multiple Speakers**

Transcribe all participants in a conference.

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
    |     +------------+                                    |
    |     |Transcription|----> transcript_created events    |
    |     +------------+       (direction indicates speaker)|
    +-------------------------------------------------------+

**Scenario 3: AI Integration**

Send transcripts to an AI system for real-time analysis.

::

    VoIPBIN                Your App               AI Service
        |                      |                      |
        | transcript_created   |                      |
        +--------------------->|                      |
        |                      | Analyze sentiment    |
        |                      +--------------------->|
        |                      |                      |
        |                      | sentiment: positive  |
        |                      |<---------------------+
        |                      |                      |
        |                      | Update agent UI      |
        |                      |                      |

**Scenario 4: Compliance Recording with Transcription**

Combine recording and transcription for complete call documentation.

.. code::

    {
        "actions": [
            {"type": "answer"},
            {"type": "recording_start"},
            {"type": "transcribe_start", "option": {"language": "en-US"}},
            {"type": "connect", "option": {"destinations": [...]}},
            {"type": "transcribe_stop"},
            {"type": "recording_stop"}
        ]
    }


.. _transcribe-overview-supported_languages:

Supported Languages
-------------------
VoIPBIN supports transcription in 70+ languages and regional variants. Specify the language using the ``language`` option (e.g., ``en-US``, ``ko-KR``).

**Common Languages**

+----------------+---------------------------+
| Language Code  | Language                  |
+================+===========================+
| en-US          | English (United States)   |
+----------------+---------------------------+
| en-GB          | English (United Kingdom)  |
+----------------+---------------------------+
| es-ES          | Spanish (Spain)           |
+----------------+---------------------------+
| es-MX          | Spanish (Mexico)          |
+----------------+---------------------------+
| fr-FR          | French (France)           |
+----------------+---------------------------+
| de-DE          | German (Germany)          |
+----------------+---------------------------+
| it-IT          | Italian (Italy)           |
+----------------+---------------------------+
| pt-BR          | Portuguese (Brazil)       |
+----------------+---------------------------+
| ja-JP          | Japanese (Japan)          |
+----------------+---------------------------+
| ko-KR          | Korean (South Korea)      |
+----------------+---------------------------+
| zh-CN          | Chinese (Mandarin)        |
+----------------+---------------------------+
| ar-SA          | Arabic (Saudi Arabia)     |
+----------------+---------------------------+
| hi-IN          | Hindi (India)             |
+----------------+---------------------------+
| nl-NL          | Dutch (Netherlands)       |
+----------------+---------------------------+
| ru-RU          | Russian (Russia)          |
+----------------+---------------------------+

VoIPBIN supports 70+ languages including regional variants for Arabic, Spanish, English, and more. Contact support for the complete language list.

To ensure optimal transcription results, choose the code that best matches your speaker's language and dialect.


Best Practices
--------------

**1. Language Selection**

- Use the most specific regional variant (e.g., ``en-AU`` not just ``en-US`` for Australian speakers)
- Mismatched language codes significantly reduce accuracy
- For multi-language calls, consider separate transcription sessions

**2. Audio Quality**

- Clear audio produces better transcripts
- Reduce background noise when possible
- Avoid overlapping speech in group calls

**3. Handling High Volume**

- Use WebSocket for real-time applications with many concurrent calls
- Batch process webhooks for analytics workloads
- Index transcripts for efficient searching

**4. Storage and Compliance**

- Define retention policies for transcript data
- Store transcripts with call metadata for context
- Consider encryption for sensitive conversations


Troubleshooting
---------------

**Transcription Not Starting**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| No transcribe_id returned | Verify call/conference is in "progressing"     |
|                           | status before starting transcription           |
+---------------------------+------------------------------------------------+
| Permission denied         | Check API token has transcription permissions  |
+---------------------------+------------------------------------------------+
| Invalid language code     | Verify language code is in supported list      |
+---------------------------+------------------------------------------------+

**Poor Accuracy**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Words frequently wrong    | Check language code matches speaker's dialect  |
+---------------------------+------------------------------------------------+
| Missing words             | Check audio quality - background noise or      |
|                           | low volume reduces accuracy                    |
+---------------------------+------------------------------------------------+
| Technical terms wrong     | STT may not recognize domain-specific terms;   |
|                           | consider post-processing                       |
+---------------------------+------------------------------------------------+

**Missing Transcripts**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Webhook not receiving     | Verify webhook URL is configured in customer   |
|                           | settings and is publicly accessible            |
+---------------------------+------------------------------------------------+
| WebSocket disconnects     | Implement reconnection logic; check for        |
|                           | network issues                                 |
+---------------------------+------------------------------------------------+
| Gaps in transcript        | Silence or unclear audio produces no           |
|                           | transcripts - this is expected behavior        |
+---------------------------+------------------------------------------------+

**Webhook Delivery Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Events delayed            | Check webhook endpoint response time;          |
|                           | should respond within 5 seconds                |
+---------------------------+------------------------------------------------+
| Duplicate events          | Implement idempotency using transcript ``id``  |
+---------------------------+------------------------------------------------+
| Events out of order       | Sort by ``tm_transcript`` to reconstruct       |
|                           | conversation order                             |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Call Overview <call-overview>` - Transcribing calls
- :ref:`Conference Overview <conference-overview>` - Transcribing conferences
- :ref:`Recording Overview <recording-overview>` - Recording and transcribing together
- :ref:`Flow Actions <flow-struct-action-transribe_start>` - Transcribe flow actions
