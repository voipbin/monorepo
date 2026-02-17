# Multi-Provider Streaming TTS Design

## Problem Statement

The streaming TTS system (`bin-tts-manager`) currently only supports ElevenLabs as the streaming provider. The `/speakings` endpoint already accepts a `provider` parameter, but the streaming handler ignores it and always dispatches to ElevenLabs. We need to add GCP Cloud TTS (StreamingSynthesize) and AWS Polly (SynthesizeSpeech) as additional streaming providers.

## Current Architecture

- `pkg/streaminghandler/main.go` defines a `streamer` interface with methods: `Init`, `Run`, `SayStop`, `SayAdd`, `SayFlush`, `SayFinish`
- `pkg/streaminghandler/elevenlabs.go` is the only implementation
- `streamingHandler` struct holds a single `elevenlabsHandler streamer` field
- `runStreamer()` in `run.go` iterates a hardcoded map of handlers and uses the first success, ignoring `st.Provider`
- The `Streaming` model already has `Provider` and `VoiceID` fields
- `speakinghandler` defaults provider to `"elevenlabs"` if empty

## Approach: Direct Provider Dispatch

Add `gcpHandler` and `awsHandler` fields to `streamingHandler` (same pattern as `elevenlabsHandler`). Modify `runStreamer()` to dispatch based on `st.Provider`.

## Provider Specifications

| Aspect | ElevenLabs (existing) | GCP (new) | AWS (new) |
|--------|----------------------|-----------|-----------|
| Protocol | WebSocket | gRPC bidirectional | HTTP request-response |
| SDK | gorilla/websocket | cloud.google.com/go/texttospeech/apiv1 | github.com/aws/aws-sdk-go-v2/service/polly |
| Audio output | PCM 16kHz | PCM 8kHz (requested via sample_rate_hertz) | PCM 8kHz |
| Downsampling | Yes (2x to 8kHz) | No | No |
| Streaming model | Send text, receive audio on same WebSocket | Send text, receive audio on same gRPC stream | Each SayAdd = separate SynthesizeSpeech call |
| Keep-alive | Custom ping every 10s | gRPC built-in keepalive | Not needed |
| Voice restriction | All ElevenLabs voices | Chirp 3: HD voices only (Pre-GA) | All Polly voices |
| API maturity | GA | Pre-GA | GA |

## Files to Change

### Model Changes

**`models/streaming/streaming.go`** - Add vendor name constants:
```go
VendorNameGCP VendorName = "gcp"
VendorNameAWS VendorName = "aws"
```

### New Handler Files

**`pkg/streaminghandler/gcp.go`** - GCP Cloud TTS StreamingSynthesize handler:
- Implements `streamer` interface
- `Init()`: Creates gRPC client, starts `StreamingSynthesize` call with `StreamingAudioConfig{AudioEncoding: PCM, SampleRateHertz: 8000}`
- `Run()`: Reads PCM chunks from gRPC stream, wraps in AudioSocket frames, writes to Asterisk
- `SayAdd()`: Sends text input to the gRPC stream
- `SayFlush()`: Closes current gRPC stream, reinitializes for next message
- `SayStop()`: Cancels context
- `SayFinish()`: Closes gRPC stream cleanly
- Voice selection: explicit voice_id > flow variable `voipbin.tts.gcp.voice_id` > language+gender map (Chirp 3 HD voices) > default

**`pkg/streaminghandler/aws.go`** - AWS Polly SynthesizeSpeech handler:
- Implements `streamer` interface
- `Init()`: Creates Polly client with credentials
- `Run()`: Background goroutine reads from current audio stream, writes AudioSocket frames
- `SayAdd()`: Calls `SynthesizeSpeech` with `OutputFormat=pcm, SampleRate=8000`, reads response body as stream. Each call is a separate HTTP request.
- `SayFlush()`: Cancels current response reader
- `SayStop()`: Cancels context
- `SayFinish()`: Drains remaining audio
- Voice selection: explicit voice_id > flow variable `voipbin.tts.aws.voice_id` > language+gender map > default

### Dispatch Changes

**`pkg/streaminghandler/run.go`** - `runStreamer()`:
- Replace the iterate-all-handlers map with direct lookup based on `st.Provider`
- Map: `"elevenlabs"` -> `h.elevenlabsHandler`, `"gcp"` -> `h.gcpHandler`, `"aws"` -> `h.awsHandler`

**`pkg/streaminghandler/main.go`**:
- Add `gcpHandler streamer` and `awsHandler streamer` fields to `streamingHandler`
- Update `NewStreamingHandler()` to accept AWS credentials and instantiate all three handlers
- Add flow variable constants: `voipbin.tts.gcp.voice_id`, `voipbin.tts.aws.voice_id`

### Upstream Changes

**`cmd/tts-manager/main.go`** (or wherever `NewStreamingHandler` is called):
- Pass AWS credentials to `NewStreamingHandler`

## Audio Pipeline

### GCP (8kHz PCM, no downsampling)
```
gRPC StreamingSynthesize -> PCM 8kHz bytes -> AudioSocket wrap -> 320-byte frames -> Asterisk
```

### AWS (8kHz PCM, no downsampling)
```
SynthesizeSpeech -> io.ReadCloser (PCM 8kHz) -> read chunks -> AudioSocket wrap -> 320-byte frames -> Asterisk
```

## Voice Mapping

Each provider has its own voice mapping table following the same 3-tier fallback:
1. Explicit `voice_id` on the Streaming struct
2. Flow variable (`voipbin.tts.<provider>.voice_id`)
3. Language+gender lookup table with provider-specific voices
4. Default voice

### GCP Chirp 3 HD Voices (8 personalities x 31 locales)
- Male: Puck, Charon, Fenrir, Orus
- Female: Aoede, Kore, Leda, Zephyr
- Format: `<locale>-Chirp3-HD-<name>` (e.g., `en-US-Chirp3-HD-Charon`)

### AWS Polly Voices
- Reuse the same default mapping pattern from batch TTS `audiohandler/aws.go`

## Configuration

- **GCP**: Application Default Credentials (ADC) - no new env vars
- **AWS**: Existing `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` env vars
- **New flow variables**: `voipbin.tts.gcp.voice_id`, `voipbin.tts.aws.voice_id`

## AWS Polly Trade-off: Per-Request Latency

Each `SayAdd()` triggers a separate `SynthesizeSpeech` HTTP call. For AI conversations with incremental token streaming, this means ~100-300ms latency per text chunk. This is inherent to Polly's request-response model. Acceptable for most use cases but worth noting.

## Error Handling

If a requested provider's handler is nil (credentials not configured), return a clear error: `"<provider> streaming TTS is not configured"`. No fallback to another provider.
