# Sort SIP messages by numeric timestamp

## Background
GetSIPMessages sorted the formatted RFC3339Nano strings. Trailing zero omission makes the string order differ from the time order (whole-second vs fractional values, e.g. .1Z vs .15Z). Callers (siphandler, two sites) only consume the returned order; docs do not describe the sort.

## Change
- Sort the Homer messages by the integer MicroTS (milliseconds) with sort.SliceStable before conversion; equal times keep the response order.
- Remove the string-based sort. Timestamp format and all response fields are unchanged. The 50-message cut stays after sorting.

## Verify
- Table test: mixed fractional digit counts, whole second vs fractional. Fails before the change, passes after.
- Stability test: 40 messages over two times keep response order within each time (catches an unstable sort).
- Existing sort/cap test and the full service suite and lint pass.
