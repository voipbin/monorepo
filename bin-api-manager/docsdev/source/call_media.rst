.. _call-media:

Call Media and Codecs
=====================

This section covers audio and video media handling in VoIPBIN, including codec support, quality considerations, and encryption.

Audio Codec Support
-------------------

VoIPBIN supports multiple audio codecs for different use cases:

.. code::

    Supported Audio Codecs:

    +----------+------------+-----------+------------------+------------------------+
    | Codec    | Bitrate    | Sample    | Quality          | Use Case               |
    |          |            | Rate      |                  |                        |
    +----------+------------+-----------+------------------+------------------------+
    | G.711    | 64 kbps    | 8 kHz     | Good (PSTN)      | PSTN calls, SIP trunks |
    | (ulaw/   |            |           |                  |                        |
    |  alaw)   |            |           |                  |                        |
    +----------+------------+-----------+------------------+------------------------+
    | G.722    | 64 kbps    | 16 kHz    | Excellent (HD)   | HD voice SIP calls     |
    +----------+------------+-----------+------------------+------------------------+
    | G.729    | 8 kbps     | 8 kHz     | Acceptable       | Low bandwidth links    |
    +----------+------------+-----------+------------------+------------------------+
    | Opus     | 6-510 kbps | 8-48 kHz  | Excellent        | WebRTC, adaptive       |
    |          | (adaptive) |           |                  |                        |
    +----------+------------+-----------+------------------+------------------------+
    | PCMU     | 64 kbps    | 8 kHz     | Good             | Same as G.711 ulaw     |
    +----------+------------+-----------+------------------+------------------------+
    | PCMA     | 64 kbps    | 8 kHz     | Good             | Same as G.711 alaw     |
    +----------+------------+-----------+------------------+------------------------+

**Codec Selection:**

.. code::

    Codec Selection by Call Type:

    PSTN Calls:
    +------------------------------------------+
    | Codec: G.711 (ulaw for US, alaw for EU)  |
    | Reason: Universal PSTN compatibility     |
    | Quality: Standard telephone quality      |
    +------------------------------------------+

    WebRTC Calls:
    +------------------------------------------+
    | Codec: Opus (primary)                    |
    | Reason: Adaptive bitrate, loss resilient |
    | Quality: HD voice up to 48 kHz           |
    +------------------------------------------+

    SIP Trunk (HD):
    +------------------------------------------+
    | Codec: G.722 (if supported)              |
    | Fallback: G.711                          |
    | Quality: Wideband audio (16 kHz)         |
    +------------------------------------------+

    Low Bandwidth:
    +------------------------------------------+
    | Codec: G.729                             |
    | Reason: Only 8 kbps required             |
    | Quality: Acceptable for voice            |
    +------------------------------------------+

Audio Quality Factors
---------------------

Several factors affect call audio quality:

.. code::

    Quality Factor: Network Latency

    +------------------------------------------+
    | < 150ms:  Excellent - natural conversation
    | 150-300ms: Good - slight delay noticeable
    | 300-500ms: Fair - conversation difficult
    | > 500ms:  Poor - echo, overlap issues
    +------------------------------------------+

    VoIPBIN Infrastructure:
    +------------------------------------------+
    | Global edge locations minimize latency   |
    | Typical added latency: < 30ms            |
    +------------------------------------------+

.. code::

    Quality Factor: Packet Loss

    +------------------------------------------+
    | 0%:    Perfect audio
    | 1-2%:  Minor artifacts, acceptable
    | 3-5%:  Noticeable degradation
    | > 5%:  Significant quality loss
    +------------------------------------------+

    VoIPBIN Mitigation:
    +------------------------------------------+
    | - Opus codec: Built-in packet loss       |
    |   concealment up to 15%                  |
    | - Jitter buffer: Smooths packet timing   |
    | - FEC: Forward Error Correction (Opus)   |
    +------------------------------------------+

.. code::

    Quality Factor: Jitter

    +------------------------------------------+
    | Definition: Variation in packet arrival  |
    |                                          |
    | < 20ms:  Excellent                       |
    | 20-50ms: Good (jitter buffer handles)    |
    | > 50ms:  Poor - buffer underruns         |
    +------------------------------------------+

    VoIPBIN Jitter Buffer:
    +------------------------------------------+
    | Type: Adaptive                           |
    | Range: 20-200ms                          |
    | Adapts to network conditions             |
    +------------------------------------------+

RTP and Media Transport
-----------------------

Real-time Transport Protocol (RTP) carries audio:

.. code::

    RTP Packet Structure:

    +-----------------------------------+
    | V=2|P|X|CC |M| PT |  Sequence #  |   12 bytes
    |           Timestamp              |   header
    |             SSRC                 |
    +-----------------------------------+
    |            Payload               |   Audio
    |        (codec-encoded audio)     |   data
    +-----------------------------------+

    Header Fields:
    +------------------------------------------+
    | V:  Version (always 2)                   |
    | P:  Padding flag                         |
    | X:  Extension flag                       |
    | CC: CSRC count                           |
    | M:  Marker bit (frame boundary)          |
    | PT: Payload type (codec identifier)      |
    | Sequence: Packet ordering                |
    | Timestamp: Sampling instant              |
    | SSRC: Synchronization source ID          |
    +------------------------------------------+

**RTP Port Ranges:**

.. code::

    VoIPBIN RTP Ports:

    Media Servers (Asterisk):
    +------------------------------------------+
    | Range: 10000-20000 UDP                   |
    | Per call: 2 ports (RTP + RTCP)           |
    +------------------------------------------+

    RTPEngine (Media Proxy):
    +------------------------------------------+
    | Range: 20000-60000 UDP                   |
    | Handles NAT traversal                    |
    | Provides encryption bridging             |
    +------------------------------------------+

    Client Requirements:
    +------------------------------------------+
    | Outbound UDP to VoIPBIN ports required   |
    | If blocked: WebRTC with TURN as fallback |
    +------------------------------------------+

Media Encryption
----------------

VoIPBIN supports encrypted media for security:

**SRTP (Secure RTP):**

.. code::

    SRTP Encryption:

    +------------------------------------------+
    | Algorithm: AES-128-CM                    |
    | Authentication: HMAC-SHA1-80             |
    | Key exchange: DTLS-SRTP or SDES          |
    +------------------------------------------+

    SRTP Packet:
    +-----------------------------------+
    | RTP Header (not encrypted)        |
    +-----------------------------------+
    | Encrypted Payload                 |
    | (AES-128 Counter Mode)            |
    +-----------------------------------+
    | Authentication Tag (10 bytes)     |
    +-----------------------------------+

**Encryption by Call Type:**

.. code::

    Encryption Matrix:

    +------------------+-------------+-----------+
    | Call Type        | Signaling   | Media     |
    +------------------+-------------+-----------+
    | WebRTC           | WSS (TLS)   | SRTP      |
    | SIP over TLS     | TLS         | SRTP*     |
    | SIP over UDP     | None        | RTP       |
    | PSTN             | N/A         | RTP**     |
    +------------------+-------------+-----------+

    * SRTP if negotiated via SDES or DTLS
    ** PSTN segment is unencrypted (carrier network)

**End-to-End Encryption:**

.. code::

    E2E Encryption Consideration:

    WebRTC to WebRTC:
    +------------------------------------------+
    | Full SRTP encryption possible            |
    | Keys never leave endpoints               |
    +------------------------------------------+

    WebRTC to PSTN:
    +------------------------------------------+
    | WebRTC leg: SRTP encrypted               |
    | VoIPBIN: Decrypts to mix/process         |
    | PSTN leg: Unencrypted (carrier limit)    |
    +------------------------------------------+

    Note: VoIPBIN must decrypt media for:
    - Transcoding between codecs
    - Recording
    - Transcription
    - Conferencing (mixing)

DTMF Handling
-------------

Dual-Tone Multi-Frequency (DTMF) for IVR input:

.. code::

    DTMF Methods:

    RFC 2833 (RTP Events):
    +------------------------------------------+
    | DTMF sent as special RTP packets         |
    | Payload type: 101 (commonly)             |
    | Most reliable for VoIP                   |
    | VoIPBIN default method                   |
    +------------------------------------------+

    In-band (Audio):
    +------------------------------------------+
    | DTMF tones in audio stream               |
    | Can be compressed/distorted              |
    | Fallback for legacy systems              |
    +------------------------------------------+

    SIP INFO:
    +------------------------------------------+
    | DTMF in SIP signaling messages           |
    | Not affected by audio path               |
    | Less common                              |
    +------------------------------------------+

**DTMF in API:**

.. code::

    Sending DTMF:
    POST /v1/calls/{call-id}/dtmf
    {
        "digits": "1234#",
        "duration": 250,      // ms per digit
        "interval": 100       // ms between digits
    }

    Receiving DTMF (in flow):
    {
        "type": "digits_receive",
        "option": {
            "length": 4,          // Expected digits
            "duration": 10000,    // Timeout ms
            "terminator": "#"     // Optional end char
        }
    }

Recording Formats
-----------------

VoIPBIN supports multiple recording formats:

.. code::

    Recording Format Options:

    +--------+------------+-----------+------------------+
    | Format | Codec      | Quality   | File Size        |
    +--------+------------+-----------+------------------+
    | WAV    | PCM        | Lossless  | ~960 KB/min      |
    | MP3    | MP3        | Good      | ~128 KB/min      |
    | OGG    | Opus       | Excellent | ~96 KB/min       |
    +--------+------------+-----------+------------------+

    Default: MP3 (balance of quality and size)

**Recording Configuration:**

.. code::

    Recording Options:

    {
        "type": "record_start",
        "option": {
            "direction": "both",    // "in", "out", "both"
            "format": "mp3",        // "wav", "mp3", "ogg"
            "channels": "mixed",    // "mixed", "stereo"
            "sample_rate": 16000    // Hz
        }
    }

    Direction Explained:
    +------------------------------------------+
    | "in":   Record only incoming audio       |
    |         (what caller says)               |
    |                                          |
    | "out":  Record only outgoing audio       |
    |         (what system/agent says)         |
    |                                          |
    | "both": Record entire conversation       |
    |         (default, recommended)           |
    +------------------------------------------+

    Channels:
    +------------------------------------------+
    | "mixed":  Single track, both parties     |
    |           combined. Smaller file.        |
    |                                          |
    | "stereo": Two tracks, parties separated  |
    |           Left = inbound, Right = outbound|
    |           Better for analysis/transcription|
    +------------------------------------------+

Text-to-Speech (TTS)
--------------------

TTS converts text to spoken audio:

.. code::

    TTS Providers:

    Google Cloud TTS (default):
    +------------------------------------------+
    | Voices: 200+ in 40+ languages            |
    | Quality: Neural and Standard             |
    | SSML: Supported                          |
    +------------------------------------------+

    AWS Polly:
    +------------------------------------------+
    | Voices: 60+ in 30+ languages             |
    | Quality: Neural and Standard             |
    | SSML: Supported                          |
    +------------------------------------------+

    Provider Fallback:
    +------------------------------------------+
    | If the selected provider fails, the      |
    | system falls back to the alternative     |
    | provider with the default voice for the  |
    | language.                                |
    +------------------------------------------+

    TTS Action Example:
    {
        "type": "talk",
        "option": {
            "text": "Hello, welcome to our service.",
            "language": "en-US",
            "provider": "gcp",                  // Optional: "gcp" or "aws"
            "voice_id": "en-US-Neural2-C"       // Optional: provider-specific voice
        }
    }

**SSML Support:**

.. code::

    SSML (Speech Synthesis Markup Language):

    Basic Example:
    {
        "type": "talk",
        "option": {
            "text": "<speak>Your balance is <say-as interpret-as='currency'>$123.45</say-as></speak>",
            "language": "en-US"
        }
    }

    Common SSML Tags:
    +------------------------------------------+
    | <break time='500ms'/>  - Pause           |
    | <emphasis>            - Stress word      |
    | <prosody rate='slow'> - Speed control    |
    | <say-as>              - Format numbers   |
    | <phoneme>             - Pronunciation    |
    +------------------------------------------+

Speech-to-Text (STT)
--------------------

Real-time transcription of audio:

.. code::

    STT Configuration:

    {
        "type": "transcribe_start",
        "option": {
            "language": "en-US",
            "direction": "both",      // "in", "out", "both"
            "interim_results": false  // Real-time partials
        }
    }

    Supported Languages:
    +------------------------------------------+
    | 70+ languages and regional variants      |
    | See: transcribe_overview for full list   |
    +------------------------------------------+

**STT Accuracy Tips:**

.. code::

    Improve Transcription Accuracy:

    1. Correct Language:
       +------------------------------------------+
       | Specify exact locale: "en-US" vs "en-GB" |
       | Affects vocabulary and accent models     |
       +------------------------------------------+

    2. Audio Quality:
       +------------------------------------------+
       | Clear audio = better transcription       |
       | Minimize background noise                |
       | Use headsets for agents                  |
       +------------------------------------------+

    3. Sample Rate:
       +------------------------------------------+
       | Higher sample rate (16kHz+) helps        |
       | VoIPBIN resamples automatically          |
       +------------------------------------------+

Video Calls (WebRTC)
--------------------

VoIPBIN supports video via WebRTC:

.. code::

    Video Codec Support:

    +--------+------------+-----------------+
    | Codec  | Resolution | Use Case        |
    +--------+------------+-----------------+
    | VP8    | Up to 720p | Default WebRTC  |
    | VP9    | Up to 1080p| Higher quality  |
    | H.264  | Up to 1080p| Hardware accel  |
    +--------+------------+-----------------+

    Video Constraints:
    +------------------------------------------+
    | Max resolution: 1280x720 (720p)          |
    | Max framerate: 30 fps                    |
    | Max bitrate: 2 Mbps                      |
    +------------------------------------------+

**Video Conferencing:**

.. code::

    Video Conference Layout:

    2 Participants:
    +-------------+-------------+
    |             |             |
    |     A       |      B      |
    |             |             |
    +-------------+-------------+

    4 Participants:
    +------+------+------+------+
    |  A   |  B   |  C   |  D   |
    +------+------+------+------+

    Active Speaker:
    +---------------------------+
    |                           |
    |     Active Speaker        |
    |                           |
    +-------+-------+-------+---+
    | P1    | P2    | P3    | ...|
    +-------+-------+-------+---+

Bandwidth Requirements
----------------------

Plan network capacity based on call types:

.. code::

    Bandwidth per Call:

    Audio Only (G.711):
    +------------------------------------------+
    | RTP payload: 64 kbps                     |
    | + Headers: ~15 kbps                      |
    | = Total: ~80 kbps per direction          |
    | Bidirectional: ~160 kbps per call        |
    +------------------------------------------+

    Audio Only (Opus):
    +------------------------------------------+
    | Adaptive: 6-128 kbps                     |
    | Typical: 24-32 kbps                      |
    | + Headers: ~15 kbps                      |
    | Bidirectional: ~100 kbps per call        |
    +------------------------------------------+

    Video (720p):
    +------------------------------------------+
    | Video: 1-2 Mbps                          |
    | Audio: ~100 kbps (Opus)                  |
    | Bidirectional: ~4 Mbps per call          |
    +------------------------------------------+

**Capacity Planning:**

.. code::

    Example: 100 Concurrent Calls

    Audio Only (G.711):
    +------------------------------------------+
    | 100 calls × 160 kbps = 16 Mbps           |
    | Recommended: 20 Mbps (headroom)          |
    +------------------------------------------+

    Audio Only (Opus):
    +------------------------------------------+
    | 100 calls × 100 kbps = 10 Mbps           |
    | Recommended: 15 Mbps (headroom)          |
    +------------------------------------------+

    Video (720p):
    +------------------------------------------+
    | 100 calls × 4 Mbps = 400 Mbps            |
    | Recommended: 500 Mbps (headroom)         |
    +------------------------------------------+

Quality Monitoring
------------------

Monitor call quality with metrics:

.. code::

    Available Metrics (via API/Webhooks):

    Call Level:
    +------------------------------------------+
    | duration         - Total call time       |
    | ringing_duration - Time ringing          |
    | talk_duration    - Connected time        |
    | hangup_reason    - Why call ended        |
    +------------------------------------------+

    Media Level (when available):
    +------------------------------------------+
    | jitter           - Packet timing variance|
    | packet_loss      - % packets lost        |
    | rtt              - Round-trip time       |
    | mos              - Mean Opinion Score    |
    +------------------------------------------+

    MOS Score Reference:
    +------------------------------------------+
    | 4.3-5.0: Excellent (HD quality)          |
    | 4.0-4.3: Good (toll quality)             |
    | 3.6-4.0: Fair (acceptable)               |
    | 3.1-3.6: Poor (degraded)                 |
    | < 3.1:   Bad (unacceptable)              |
    +------------------------------------------+

