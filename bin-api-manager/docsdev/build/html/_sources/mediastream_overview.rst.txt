.. _mediastream-overview:

Overview
========
VoIPBIN's Media Stream API provides direct access to call and conference audio via WebSocket connections. Instead of relying on SIP signaling for media control, you can stream audio bidirectionally with your applications for real-time processing, AI integration, custom IVR, and more.

With the Media Stream API you can:

- Stream live call audio to your application in real-time
- Inject audio into calls and conferences
- Build AI voice assistants with direct audio access
- Create custom speech recognition pipelines
- Implement real-time audio analysis and monitoring


How Media Streaming Works
-------------------------
When you connect to a media stream, VoIPBIN establishes a WebSocket connection that carries audio data directly between the call/conference and your application.

**Media Stream Architecture**

::

    Traditional VoIP:                   VoIPBIN Media Stream:
    +-------+   SIP   +-------+         +-------+   WebSocket  +----------+
    | Phone |<------->|VoIPBIN|         | Call  |<============>| Your App |
    +-------+         +-------+         +-------+              +----------+
         (signaling only)                    (direct audio access)

**Key Differences from Traditional VoIP**

+------------------------+----------------------------------+----------------------------------+
| Aspect                 | Traditional SIP                  | Media Streaming                  |
+========================+==================================+==================================+
| Audio Access           | Via RTP to SIP endpoints         | Direct WebSocket to your app     |
+------------------------+----------------------------------+----------------------------------+
| Control                | SIP signaling                    | API and WebSocket                |
+------------------------+----------------------------------+----------------------------------+
| Integration            | Requires SIP stack               | Simple WebSocket client          |
+------------------------+----------------------------------+----------------------------------+
| Use Cases              | Phone-to-phone calls             | AI, custom IVR, analysis         |
+------------------------+----------------------------------+----------------------------------+

**System Components**

::

    +--------+                                              +-----------+
    |  Call  |<------- RTP ------->+                        |           |
    +--------+                     |                        |  Your     |
                              +----+-----+                  |  App      |
                              | VoIPBIN  |<== WebSocket ===>|           |
                              | Media    |                  | - AI/ML   |
                              | Bridge   |                  | - STT/TTS |
    +------------+            +----+-----+                  | - IVR     |
    | Conference |<-- RTP --->+                             |           |
    +------------+                                          +-----------+

The Media Bridge handles protocol conversion between RTP (VoIP standard) and WebSocket (web standard), enabling any WebSocket-capable application to process call audio.


Streaming Modes
---------------
VoIPBIN supports two streaming modes based on your application's needs.

**Bi-Directional Streaming**

Your application both receives and sends audio through the same WebSocket connection.

::

    +----------+                              +----------+
    |          |======= audio IN ============>|          |
    | VoIPBIN  |                              | Your App |
    |          |<====== audio OUT ============|          |
    +----------+                              +----------+

**Initiate via API:**

.. code::

    GET https://api.voipbin.net/v1.0/calls/<call-id>/media_stream?encapsulation=rtp&token=<token>

**Use cases:**
- AI voice assistants (listen and respond)
- Interactive IVR systems
- Real-time audio processing with feedback
- Call bridging to custom systems

**Uni-Directional Streaming**

VoIPBIN receives audio from your server and plays it to the call. Your app sends audio but doesn't receive call audio.

::

    +----------+                              +----------+
    |          |                              |          |
    | VoIPBIN  |<====== audio only ===========| Your App |
    |          |                              |          |
    +----------+                              +----------+

**Initiate via Flow Action:**

.. code::

    {
        "type": "external_media_start",
        "option": {
            "url": "wss://your-server.com/audio",
            "encapsulation": "audiosocket"
        }
    }

See detail :ref:`here <flow-struct-action-external_media_start>`.

**Use cases:**
- Custom music on hold
- Pre-recorded message playback
- Text-to-speech from external service
- Audio announcements

**Mode Comparison**

+---------------------+----------------------------------+----------------------------------+
| Aspect              | Bi-Directional                   | Uni-Directional                  |
+=====================+==================================+==================================+
| Audio Direction     | Both send and receive            | Send only (to call)              |
+---------------------+----------------------------------+----------------------------------+
| Initiation          | API call (GET /media_stream)     | Flow action (external_media_start)|
+---------------------+----------------------------------+----------------------------------+
| Connection          | Your app connects to VoIPBIN     | VoIPBIN connects to your server  |
+---------------------+----------------------------------+----------------------------------+
| Best for            | Interactive applications         | Playback applications            |
+---------------------+----------------------------------+----------------------------------+


Encapsulation Types
-------------------
VoIPBIN supports three encapsulation types for different integration scenarios.

**Decision Guide**

::

                    What's your use case?
                           |
          +----------------+----------------+
          |                                 |
    Standard VoIP                      Simple audio
    integration?                       processing?
          |                                 |
    +-----+-----+                     +-----+-----+
    |           |                     |           |
   Yes         No                    Yes         No
    |           |                     |           |
    v           |                     v           |
  [RTP]         |                   [SLN]         |
                |                                 |
           Asterisk                               |
           integration?                           |
                |                                 |
          +-----+-----+                           |
          |           |                           |
         Yes         No                           |
          |           |                           |
          v           +---------------------------+
    [AudioSocket]                 |
                                  v
                            [RTP default]

**RTP (Real-time Transport Protocol)**

The standard protocol for audio/video over IP networks.

::

    +------------------+------------------+
    |   RTP Header     |   Audio Payload  |
    |   (12 bytes)     |   (160 bytes)    |
    +------------------+------------------+

+-------------------+------------------------------------------------+
| Specification     | Value                                          |
+===================+================================================+
| Protocol          | RTP over WebSocket                             |
+-------------------+------------------------------------------------+
| Codec             | G.711 μ-law (ulaw)                             |
+-------------------+------------------------------------------------+
| Sample Rate       | 8 kHz                                          |
+-------------------+------------------------------------------------+
| Bit Depth         | 16-bit                                         |
+-------------------+------------------------------------------------+
| Channels          | Mono                                           |
+-------------------+------------------------------------------------+
| Packet Size       | 172 bytes (12 header + 160 payload = 20ms)     |
+-------------------+------------------------------------------------+

**Best for:** Standard VoIP tools, industry compatibility, existing RTP processing pipelines.

**SLN (Signed Linear)**

Raw audio without protocol overhead.

::

    +----------------------------------+
    |   Raw PCM Audio Data             |
    |   (no headers, no padding)       |
    +----------------------------------+

+-------------------+------------------------------------------------+
| Specification     | Value                                          |
+===================+================================================+
| Format            | Raw PCM, signed linear                         |
+-------------------+------------------------------------------------+
| Sample Rate       | 8 kHz                                          |
+-------------------+------------------------------------------------+
| Bit Depth         | 16-bit signed                                  |
+-------------------+------------------------------------------------+
| Channels          | Mono                                           |
+-------------------+------------------------------------------------+
| Byte Order        | Native                                         |
+-------------------+------------------------------------------------+

**Best for:** Minimal overhead, simple audio processing, direct PCM access without parsing.

**AudioSocket**

Asterisk-specific protocol designed for simple audio streaming.

::

    +------------------+------------------+
    | AudioSocket Hdr  |   PCM Audio      |
    +------------------+------------------+

+-------------------+------------------------------------------------+
| Specification     | Value                                          |
+===================+================================================+
| Protocol          | Asterisk AudioSocket                           |
+-------------------+------------------------------------------------+
| Format            | PCM little-endian                              |
+-------------------+------------------------------------------------+
| Sample Rate       | 8 kHz                                          |
+-------------------+------------------------------------------------+
| Bit Depth         | 16-bit                                         |
+-------------------+------------------------------------------------+
| Channels          | Mono                                           |
+-------------------+------------------------------------------------+
| Chunk Size        | 320 bytes (20ms of audio)                      |
+-------------------+------------------------------------------------+

**Best for:** Asterisk integration, simple streaming with minimal overhead.

See `Asterisk AudioSocket Documentation <https://docs.asterisk.org/Configuration/Channel-Drivers/AudioSocket/>`_ for protocol details.

**Encapsulation Comparison**

+-------------------+------------------+------------------+------------------+
| Aspect            | RTP              | SLN              | AudioSocket      |
+===================+==================+==================+==================+
| Headers           | 12 bytes         | None             | Protocol header  |
+-------------------+------------------+------------------+------------------+
| Compatibility     | Industry standard| Simple           | Asterisk         |
+-------------------+------------------+------------------+------------------+
| Overhead          | Low              | Minimal          | Low              |
+-------------------+------------------+------------------+------------------+
| Parsing Required  | Yes (RTP)        | No               | Yes (AudioSocket)|
+-------------------+------------------+------------------+------------------+


Supported Resources
-------------------
Media streaming is available for both calls and conferences.

**Call Media Streaming**

Stream audio from a single call.

.. code::

    GET https://api.voipbin.net/v1.0/calls/<call-id>/media_stream?encapsulation=<type>&token=<token>

**Audio contains:** Both parties' audio mixed together.

**Conference Media Streaming**

Stream audio from a conference with multiple participants.

.. code::

    GET https://api.voipbin.net/v1.0/conferences/<conference-id>/media_stream?encapsulation=<type>&token=<token>

**Audio contains:** All participants' audio mixed together.

**Resource Comparison**

+-------------------+----------------------------------+----------------------------------+
| Aspect            | Call                             | Conference                       |
+===================+==================================+==================================+
| Audio Source      | Two-party conversation           | Multi-party conversation         |
+-------------------+----------------------------------+----------------------------------+
| Audio Mix         | Caller + callee                  | All participants                 |
+-------------------+----------------------------------+----------------------------------+
| Audio Injection   | Heard by both parties            | Heard by all participants        |
+-------------------+----------------------------------+----------------------------------+
| Use Case          | 1:1 AI assistant                 | Conference monitoring/recording  |
+-------------------+----------------------------------+----------------------------------+


Connection Lifecycle
--------------------
Understanding the WebSocket connection lifecycle helps build robust streaming applications.

**Connection Flow**

::

    Your App                         VoIPBIN
        |                               |
        | GET /calls/{id}/media_stream  |
        +------------------------------>|
        |                               |
        | 101 Switching Protocols       |
        |<------------------------------+
        |                               |
        |<======= audio frames ========>|  (bi-directional)
        |<======= audio frames ========>|
        |<======= audio frames ========>|
        |                               |
        | close() or call ends          |
        +------------------------------>|
        |                               |

**Connection States**

::

    connecting ---> open ---> streaming ---> closing ---> closed
                                   |
                                   v
                              (call ends)
                                   |
                                   v
                                closed

**State Descriptions**

+---------------+------------------------------------------------------------------+
| State         | What's happening                                                 |
+===============+==================================================================+
| connecting    | WebSocket handshake in progress                                  |
+---------------+------------------------------------------------------------------+
| open          | Connection established, ready for audio                          |
+---------------+------------------------------------------------------------------+
| streaming     | Audio frames being sent/received                                 |
+---------------+------------------------------------------------------------------+
| closing       | Graceful shutdown initiated                                      |
+---------------+------------------------------------------------------------------+
| closed        | Connection terminated                                            |
+---------------+------------------------------------------------------------------+

**Connection Termination**

The WebSocket connection closes when:

- Your application closes the connection
- The call or conference ends
- Network failure occurs
- Authentication token expires


Integration Patterns
--------------------
Common patterns for integrating media streaming with your applications.

**Pattern 1: AI Voice Assistant**

::

    Call Audio         Your App           AI Service
        |                  |                   |
        |====audio====>    |                   |
        |                  | STT               |
        |                  +------------------>|
        |                  |                   |
        |                  | AI response       |
        |                  |<------------------+
        |                  |                   |
        |                  | TTS               |
        |    <====audio====+                   |
        |                  |                   |

**Pattern 2: Real-Time Monitoring**

::

    Call Audio         Your App           Dashboard
        |                  |                   |
        |====audio====>    |                   |
        |                  | analyze           |
        |                  +------------------>|
        |                  |    sentiment,     |
        |                  |    keywords,      |
        |                  |    quality        |
        |                  |                   |

**Pattern 3: Custom IVR**

::

    Call Audio         Your App           Logic Engine
        |                  |                   |
        |====audio====>    |                   |
        |                  | detect DTMF/speech|
        |                  +------------------>|
        |                  |                   |
        |                  | next action       |
        |                  |<------------------+
        |                  |                   |
        |    <====prompt===+                   |
        |                  |                   |

**Pattern 4: Recording with Processing**

::

    Call Audio         Your App           Storage
        |                  |                   |
        |====audio====>    |                   |
        |                  | process           |
        |                  | (filter, enhance) |
        |                  |                   |
        |                  | store             |
        |                  +------------------>|
        |                  |                   |

For working code examples of these patterns, see the :ref:`Media Stream Tutorial <mediastream-tutorial>`.


Best Practices
--------------

**1. Audio Timing**

- Send audio in consistent 20ms chunks
- Maintain proper timing to avoid audio gaps or overlaps
- Buffer incoming audio to handle network jitter

**2. Connection Management**

- Implement automatic reconnection for dropped connections
- Handle the ``onclose`` event gracefully
- Close connections when no longer needed

**3. Resource Efficiency**

- Process audio asynchronously to avoid blocking
- Use appropriate buffer sizes (typically 320 bytes for 20ms)
- Monitor memory usage for long-running streams

**4. Error Handling**

- Log connection errors for debugging
- Implement exponential backoff for reconnection attempts
- Handle authentication failures gracefully


Troubleshooting
---------------

**Connection Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Connection refused        | Verify call/conference is active and           |
|                           | in "progressing" status                        |
+---------------------------+------------------------------------------------+
| 401 Unauthorized          | Check API token is valid and has permissions   |
+---------------------------+------------------------------------------------+
| Connection drops          | Implement reconnection logic; check network    |
|                           | stability                                      |
+---------------------------+------------------------------------------------+

**Audio Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| No audio received         | Verify call is answered and audio is flowing;  |
|                           | check encapsulation type                       |
+---------------------------+------------------------------------------------+
| Audio quality poor        | Check network latency; verify correct audio    |
|                           | format; monitor packet loss                    |
+---------------------------+------------------------------------------------+
| Audio choppy              | Implement jitter buffer; send in consistent    |
|                           | 20ms chunks; check CPU usage                   |
+---------------------------+------------------------------------------------+
| Can't send audio          | Use binary WebSocket frames; verify audio      |
|                           | format matches encapsulation type              |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Media Stream Tutorial <mediastream-tutorial>` - Code examples and use cases
- :ref:`Call Overview <call-overview>` - Call lifecycle and states
- :ref:`Conference Overview <conference-overview>` - Conference management
- :ref:`Flow Actions <flow-struct-action-external_media_start>` - External media flow action
