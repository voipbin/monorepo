# Barge-In Audio Flush Design

## Problem

When a user speaks during AI TTS playback (barge-in), the AI voice doesn't stop immediately despite Pipecat detecting the interruption via VAD. The root cause: `UnpacedWebsocketClientOutputTransport` delivers TTS audio faster than real-time to Asterisk's `chan_websocket`, which buffers it internally. By the time Pipecat cancels its output task (`_start_interruption`), Asterisk already has seconds of audio queued for playback. There's no mechanism to flush that buffer.

## Solution

Send a burst of silence from Go to Asterisk when Pipecat signals an interruption. This overwrites the buffered TTS audio in Asterisk's playback queue.

### Signal Flow

```
User speaks → Pipecat VAD detects speech → Pipecat _start_interruption()
  → Python sends TextFrame(name="flush_audio") via protobuf WebSocket to Go
  → Go receives TextFrame with name "flush_audio"
  → Go writes 500ms of silence (16,000 zero bytes) to Asterisk WebSocket
  → Asterisk's buffer is overwritten with silence
  → User hears silence instead of stale TTS audio
```

### Python Changes (`scripts/pipecat/run.py`)

Override `_start_interruption()` in `UnpacedWebsocketClientOutputTransport` to send a flush signal after the base class cancels the audio queue:

```python
class UnpacedWebsocketClientOutputTransport(WebsocketClientOutputTransport):
    async def _write_audio_sleep(self):
        pass

    async def _start_interruption(self):
        await super()._start_interruption()
        # Send flush signal to Go so it can overwrite Asterisk's audio buffer
        frame = TextFrame(text="flush_audio")
        await self._websocket.send_bytes(
            self._params.serializer.serialize(frame)
        )
```

### Go Changes (`pkg/pipecatcallhandler/runner.go`)

In `RunnerWebsocketHandleOutput`, when receiving a TextFrame with name "flush_audio", write silence to the Asterisk WebSocket:

```go
case *pipecatframe.Frame_Text:
    if x.Text.Name == "flush_audio" || x.Text.Text == "flush_audio" {
        h.flushAsteriskAudioBuffer(se)
    }
```

New method:

```go
func (h *pipecatcallHandler) flushAsteriskAudioBuffer(se *pipecatcall.Session) {
    // 500ms of silence at 16kHz 16-bit mono = 16000 samples/sec * 0.5 sec * 2 bytes = 16000 bytes
    silence := make([]byte, defaultFlushSilenceBytes)
    h.websocketHandler.WriteMessage(se.ConnAst, websocket.BinaryMessage, silence)
}
```

### Constants

- `defaultFlushSilenceBytes = 16000` (500ms at 16kHz 16-bit mono PCM)

### Why 500ms?

- Asterisk's `chan_websocket` buffer is bounded. 500ms of silence is enough to overwrite any buffered audio from the unpaced delivery without introducing a noticeable gap.
- If the silence burst is too short, stale TTS audio may still play through.
- If too long, there's an unnecessary pause before the next AI response.

## Files Changed

1. `bin-pipecat-manager/scripts/pipecat/run.py` - Override `_start_interruption()` in `UnpacedWebsocketClientOutputTransport`
2. `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` - Handle "flush_audio" TextFrame, add `flushAsteriskAudioBuffer` method
3. `bin-pipecat-manager/pkg/pipecatcallhandler/run.go` - Add `defaultFlushSilenceBytes` constant

## Risks

- **Pipecat internals**: `_start_interruption()` is a framework method. Pin pipecat version and re-verify after upgrades.
- **Serialization**: The flush TextFrame must be serialized with the same protobuf serializer. Using `self._params.serializer.serialize()` ensures this.
- **Timing**: The flush arrives after the last audio frame. Since Go processes frames sequentially in the same loop, the silence write happens immediately after the last audio write.
