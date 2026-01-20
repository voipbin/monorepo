.. _mediastream-tutorial:

Tutorial
========

Bi-Directional Media Streaming for Calls
-----------------------------------------

Connect to a call's media stream via WebSocket to send and receive audio in real-time. This allows you to build custom audio processing applications without SIP signaling.

**Establish WebSocket Connection:**

.. code::

    GET https://api.voipbin.net/v1.0/calls/<call-id>/media_stream?encapsulation=rtp&token=<YOUR_AUTH_TOKEN>

**Example:**

.. code::

    GET https://api.voipbin.net/v1.0/calls/652af662-eb45-11ee-b1a5-6fde165f9226/media_stream?encapsulation=rtp&token=<YOUR_AUTH_TOKEN>

This creates a bi-directional WebSocket connection where you can:
- **Receive audio** from the call (what the other party is saying)
- **Send audio** to the call (inject audio into the conversation)

Bi-Directional Media Streaming for Conferences
-----------------------------------------------

Access a conference's media stream to monitor or participate in the conference audio.

.. code::

    GET https://api.voipbin.net/v1.0/conferences/<conference-id>/media_stream?encapsulation=rtp&token=<YOUR_AUTH_TOKEN>

**Example:**

.. code::

    GET https://api.voipbin.net/v1.0/conferences/1ed12456-eb4b-11ee-bba8-1bfb2838807a/media_stream?encapsulation=rtp&token=<YOUR_AUTH_TOKEN>

This allows you to:
- Listen to all conference participants
- Inject audio into the conference
- Build custom conference recording or analysis tools

Encapsulation Types
-------------------

VoIPBIN supports three encapsulation types for media streaming:

**1. RTP (Real-time Transport Protocol)**

Standard protocol for audio/video over IP networks.

.. code::

    ?encapsulation=rtp

**Use cases:**
- Standard VoIP integration
- Compatible with most audio processing tools
- Industry-standard protocol

**2. SLN (Signed Linear Mono)**

Raw audio stream without headers or padding.

.. code::

    ?encapsulation=sln

**Use cases:**
- Minimal overhead needed
- Simple audio processing
- Direct PCM audio access

**3. AudioSocket**

Asterisk-specific protocol for simple audio streaming.

.. code::

    ?encapsulation=audiosocket

**Use cases:**
- Asterisk integration
- Low-overhead streaming
- Simple audio applications

**Codec:** All formats use **16-bit, 8kHz, mono** audio (ulaw for RTP/SLN, PCM little-endian for AudioSocket)

WebSocket Client Examples
--------------------------

**Python Example (RTP Streaming):**

.. code::

    import websocket
    import struct

    def on_message(ws, message):
        """Receive audio data from the call"""
        # message contains RTP packets
        print(f"Received {len(message)} bytes of audio")

        # Process audio here
        # - Save to file
        # - Run speech recognition
        # - Analyze audio
        process_audio(message)

    def on_open(ws):
        """Connection established, can start sending audio"""
        print("Media stream connected")

        # Send audio to the call
        # audio_data should be RTP packets
        audio_data = generate_audio()
        ws.send(audio_data, opcode=websocket.ABNF.OPCODE_BINARY)

    def on_error(ws, error):
        print(f"Error: {error}")

    def on_close(ws, close_status_code, close_msg):
        print(f"Connection closed: {close_status_code}")

    # Connect to media stream
    call_id = "652af662-eb45-11ee-b1a5-6fde165f9226"
    token = "<YOUR_AUTH_TOKEN>"
    ws_url = f"wss://api.voipbin.net/v1.0/calls/{call_id}/media_stream?encapsulation=rtp&token={token}"

    ws = websocket.WebSocketApp(
        ws_url,
        on_open=on_open,
        on_message=on_message,
        on_error=on_error,
        on_close=on_close
    )

    ws.run_forever()

    def process_audio(rtp_packet):
        """Process received RTP audio"""
        # Extract payload from RTP packet
        # RTP header is typically 12 bytes
        payload = rtp_packet[12:]

        # Save or process audio
        with open('received_audio.raw', 'ab') as f:
            f.write(payload)

    def generate_audio():
        """Generate RTP packets to send"""
        # This is a simplified example
        # In production, properly construct RTP packets

        # Read audio file
        with open('audio_to_inject.raw', 'rb') as f:
            audio_data = f.read(160)  # 20ms of 8kHz audio

        # Construct RTP header (simplified)
        # In production, use a proper RTP library
        return audio_data

**JavaScript Example (Browser):**

.. code::

    const callId = '652af662-eb45-11ee-b1a5-6fde165f9226';
    const token = '<YOUR_AUTH_TOKEN>';
    const wsUrl = `wss://api.voipbin.net/v1.0/calls/${callId}/media_stream?encapsulation=rtp&token=${token}`;

    const ws = new WebSocket(wsUrl);
    ws.binaryType = 'arraybuffer';

    ws.onopen = function() {
        console.log('Media stream connected');

        // Send audio to the call
        const audioData = generateAudio();
        ws.send(audioData);
    };

    ws.onmessage = function(event) {
        // Receive audio from the call
        const audioData = event.data;
        console.log(`Received ${audioData.byteLength} bytes`);

        // Process audio
        processAudio(new Uint8Array(audioData));
    };

    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
    };

    ws.onclose = function() {
        console.log('Media stream closed');
    };

    function processAudio(audioBuffer) {
        // Process received audio
        // - Play through Web Audio API
        // - Run speech recognition
        // - Visualize audio
    }

    function generateAudio() {
        // Generate audio to send
        // Returns ArrayBuffer with RTP packets
        return new ArrayBuffer(172); // RTP packet size
    }

**Node.js Example (AudioSocket):**

.. code::

    const WebSocket = require('ws');
    const fs = require('fs');

    const callId = '652af662-eb45-11ee-b1a5-6fde165f9226';
    const token = '<YOUR_AUTH_TOKEN>';
    const wsUrl = `wss://api.voipbin.net/v1.0/calls/${callId}/media_stream?encapsulation=audiosocket&token=${token}`;

    const ws = new WebSocket(wsUrl);

    ws.on('open', function() {
        console.log('AudioSocket connected');

        // Send audio file
        const audioFile = fs.readFileSync('audio.pcm');

        // Send in chunks (20ms = 320 bytes for 16-bit 8kHz mono)
        const chunkSize = 320;
        for (let i = 0; i < audioFile.length; i += chunkSize) {
            const chunk = audioFile.slice(i, i + chunkSize);
            ws.send(chunk);
        }
    });

    ws.on('message', function(data) {
        // Receive audio from call
        console.log(`Received ${data.length} bytes`);

        // Save received audio
        fs.appendFileSync('received_audio.pcm', data);
    });

    ws.on('error', function(error) {
        console.error('Error:', error);
    });

    ws.on('close', function() {
        console.log('AudioSocket closed');
    });

Uni-Directional Streaming with Flow Action
-------------------------------------------

For sending audio to a call without receiving audio back, use the ``external_media_start`` flow action.

**Create Call with External Media:**

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
                    "type": "external_media_start",
                    "option": {
                        "url": "wss://your-media-server.com/audio-stream",
                        "encapsulation": "audiosocket"
                    }
                }
            ]
        }'

This creates a uni-directional stream where VoIPBIN:
1. Establishes the call
2. Connects to your media server via WebSocket
3. Receives audio from your server
4. Plays that audio to the call participant

**Your media server receives:**

.. code::

    WebSocket connection from VoIPBIN
    → Send audio chunks (PCM format for AudioSocket)
    → VoIPBIN plays audio to call

Common Use Cases
----------------

**1. Real-Time Speech Recognition:**

.. code::

    # Python example
    def on_message(ws, message):
        # Extract audio from RTP packet
        audio = extract_audio(message)

        # Send to speech recognition API
        text = speech_to_text(audio)
        print(f"Recognized: {text}")

        # Store transcription
        save_transcription(text)

**2. Audio Injection / IVR Replacement:**

.. code::

    # Node.js example
    ws.on('open', function() {
        // Play custom audio prompts
        const prompt1 = fs.readFileSync('welcome.pcm');
        ws.send(prompt1);

        // Wait for DTMF or speech
        // Then play next prompt
    });

**3. Conference Recording:**

.. code::

    # Python example
    def on_message(ws, message):
        # Save all conference audio
        with open(f'conference_{conference_id}.raw', 'ab') as f:
            f.write(extract_audio(message))

**4. Real-Time Audio Analysis:**

.. code::

    def on_message(ws, message):
        audio = extract_audio(message)

        # Detect emotion
        emotion = analyze_emotion(audio)

        # Detect keywords
        if detect_keyword(audio, ['help', 'urgent']):
            alert_supervisor()

        # Calculate audio quality
        quality = measure_quality(audio)

**5. Custom Music on Hold:**

.. code::

    ws.on('open', function() {
        // Play custom music or messages
        const music = fs.readFileSync('hold_music.pcm');

        // Loop music while call is on hold
        setInterval(() => {
            ws.send(music);
        }, 1000);
    });

**6. AI-Powered Voice Assistant:**

.. code::

    ws.on('message', function(data) {
        // Receive customer audio
        const audio = extractAudio(data);

        // Send to AI for processing
        const response = await aiProcess(audio);

        // Convert AI response to audio
        const responseAudio = textToSpeech(response);

        // Send back to call
        ws.send(responseAudio);
    });

Audio Format Details
--------------------

**RTP Format:**
- Codec: ulaw (G.711 μ-law)
- Sample rate: 8 kHz
- Bits: 16-bit
- Channels: Mono
- Packet size: 160 bytes payload (20ms audio)

**SLN Format:**
- Raw PCM audio
- No headers or padding
- Sample rate: 8 kHz
- Bits: 16-bit signed
- Channels: Mono

**AudioSocket Format:**
- PCM little-endian
- Sample rate: 8 kHz
- Bits: 16-bit
- Channels: Mono
- Chunk size: 320 bytes (20ms of audio)

Best Practices
--------------

**1. Buffer Management:**
- Maintain audio buffers to handle jitter
- Send audio in consistent 20ms chunks
- Don't send too fast or too slow

**2. Error Handling:**
- Implement reconnection logic
- Handle WebSocket disconnections gracefully
- Log errors for debugging

**3. Audio Quality:**
- Use proper RTP packet construction
- Maintain correct timing for audio chunks
- Monitor for packet loss

**4. Resource Management:**
- Close WebSocket when done
- Don't leave connections open indefinitely
- Clean up audio buffers and files

**5. Testing:**
- Test with various network conditions
- Verify audio quality with real calls
- Monitor latency and packet loss

**6. Security:**
- Use WSS (secure WebSocket) in production
- Validate authentication tokens
- Encrypt sensitive audio data

Connection Lifecycle
--------------------

**1. Establish Connection:**

.. code::

    GET /v1.0/calls/<call-id>/media_stream?encapsulation=rtp&token=<token>

**2. WebSocket Upgrade:**

.. code::

    HTTP/1.1 101 Switching Protocols
    Upgrade: websocket
    Connection: Upgrade

**3. Bi-Directional Communication:**

.. code::

    Client ←→ VoIPBIN
    - Send audio: Binary frames with RTP packets
    - Receive audio: Binary frames with RTP packets

**4. Close Connection:**

.. code::

    ws.close()

Troubleshooting
---------------

**Common Issues:**

**No audio received:**
- Check WebSocket connection is established
- Verify call is active and answered
- Ensure correct encapsulation type

**Audio quality poor:**
- Check network latency
- Verify audio format matches requirements
- Monitor packet loss

**Connection drops:**
- Implement reconnection logic
- Check firewall rules for WebSocket
- Verify authentication token is valid

**Can't send audio:**
- Ensure binary frames are used (not text)
- Verify audio format is correct
- Check audio chunk size (typically 20ms)

For more information about media stream configuration, see :ref:`Media Stream Overview <extension-overview>`.
