# Recording start: default empty format to wav

## Problem and cause

A recording start request with an empty format is passed through unchanged. Call recordings
build filenames without an extension, put an empty `recording_format` into the snoop app
args, and the later ARI record call (callhandler/start.go) gets an empty format. Asterisk
rejects it with HTTP 422 (format unknown); the error is only logged and the row stays in
`initiating`. Confbridge recordings use the same empty value for the filename and the
bridge record call. No layer defaults or validates the field: api-manager casts it as is,
and the flow action `recording_start` option has an omitempty format.

## Decision

In `recordinghandler.Start` (the single entry for every caller: listenhandler, callhandler,
confbridgehandler, flow action, conference-manager), right after the metric increment and
before the reference type switch, replace an empty format with `recording.FormatWAV`.
Signatures do not change. `FormatWAV` is the only defined format constant. No OpenAPI, RST,
or migration change.

## Out of scope

- Failure handling when the ARI record call fails (row stays in `initiating`).
- Validation of non-empty unsupported formats (these still reach Asterisk).
- Correcting rows already stored with an empty format.
- Whitespace or case variants such as `" "` or `"WAV"`; only the exact empty string is defaulted.

## Verification

- Existing tests are unaffected (they already pass `FormatWAV` or call private functions).
- New cases in `start_test.go`, through `Start` with an empty format:
  - call: filenames end in `.wav`, snoop app args carry `recording_format=wav`, the
    stored recording has format `wav`.
  - confbridge: filename ends in `.wav`, the bridge record call receives `wav`.
- Both new cases fail before the change and pass after it.
