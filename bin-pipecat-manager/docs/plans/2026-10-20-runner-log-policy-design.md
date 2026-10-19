# VOIP-1633 Pipecat runner log policy

## Problem
The Python runner logs at loguru's default DEBUG level, and pipecat-ai 1.12.0 prints the whole
conversation at DEBUG (`Generating chat from context [...]`, openai/google/aws/anthropic LLM
services). Some runner lines also print tool arguments and tool response bodies.

## Policy
- Default level is INFO. `LOG_LEVEL` (DEBUG, INFO, WARNING, ERROR, CRITICAL, case-insensitive) changes
  it. Unset or invalid means INFO. No on/off flag. It is read from the process environment at startup;
  `.env` is not consulted because `load_dotenv` runs after the imports (stated in the docs row).
- Library records (`pipecat`, `pipecat.*`) are limited to `max(LOG_LEVEL, INFO)`, so the library
  never prints its DEBUG conversation lines and never prints more than the runner level.
- Library INFO records pass: every reachable `logger.info/success` call with an interpolation was
  read in pipecat 1.12.0 (services, processors, pipeline, transports, flows) and none prints text.
- Library WARNING and above fail closed. Sites that print content exist and are not enumerable
  (`frames.py` `text: [..]` and `thought text: ..`, `tts_service.py:1428`,
  `aggregated_frame_sequencer.py:671`, `cartesia/stt.py:464`, `ai_service.py:285-297`,
  `openai/base_llm.py:620`, `flows/manager.py` `str(e)`), so the message is replaced by
  `library message withheld (len=N)` and `record.exception` is dropped. Module, function, line and
  level stay in the JSON record. Allowlisted (message kept): `pipecat.pipeline.worker`
  (ErrorFrame lines, where a provider failure is reported once whichever service raised it) and
  `pipecat.transports.websocket.client` (uri and exception text of the socket to the Go side). Other `{frame}` worker lines (shutdown and wait frames, worker.py:826, 1201, 1221) hold no conversation. Allowlisted ErrorFrame text is
  provider error text and may rarely echo request fragments; this is accepted and listed below.
  Lost: the body of other library WARNING and above lines (reconnect attempts, per-service
  warnings, `frame_processor` error line); only module, function, line, level and length remain.
  `operations.md:67` (Pipeline error section) is corrected: the full text is the worker record.
- The filter treats a missing or `None` `record["name"]` as non-library, matches `pipecat` exactly
  or with a dot boundary (not `pipecat_x`), and on any exception inside the filter withholds the
  message (fail closed). The runner runs as `__main__`. Only `message` and `exception` are replaced:
  `record.extra` is not, which holds because pipecat 1.12.0 never `bind`s content (re-check on a pin bump).
- Runner lines that carried content now carry lengths and identifiers only:
  tools.py:148 (arg keys after `run_llm` is popped, plus args length), tools.py:162 (body length),
  tools.py:180 and team_flow.py:242, 281 (status and response length). Tool name, tool_call_id,
  pipeline id, status and exception class stay.

## Change
- New `log_config.py`: `resolve_level`, `setup(sink)` with one sink at TRACE and a filter
  function (level, library cap, library WARNING+ withholding); sink stays JSON (`serialize=True`, `diagnose=False`).
  `main.py` calls it before importing `run`, so import time lines follow the policy; the old
  bottom block is removed. No global `logger.configure` patcher.
- `tools.py`, `team_flow.py`: the lines above. `docs/operations.md`: `LOG_LEVEL` row.

## Verify
1. Level resolution (unset, invalid, lowercase, TRACE and SUCCESS rejected); default emits INFO, not DEBUG.
2. `LOG_LEVEL=DEBUG`: runner DEBUG line emitted; `pipecat.*` DEBUG with a test sentence not emitted;
   `LOG_LEVEL=WARNING` blocks `pipecat.*` INFO. 3. `pipecat.*` WARNING and ERROR records from a
   non-allowlisted module (sentence with `]`, newline, `, language:`, and exception attached): sentence
   absent in the JSON (message, text, exception), length present; allowlisted module kept.
4. Real pipecat OpenAILLMService call in a subprocess: sentence absent for default and DEBUG.
5. tool_execute and team_flow capture tests at DEBUG: sentence absent; tool name, tool_call_id,
   status, lengths present. 6. Full runner pytest (default collection), local result in the PR.
   The INFO site audit (AST walk of `logger.<level>(` calls, plus a grep for `logger.opt(` and
   aliased imports) is re-run when the `<1.13` pin moves. 3 also covers `name=None`, `__main__`,
   `pipecat` exact and `pipecat_x`, and a real ErrorFrame through the worker module (kept).

## Not covered
- `str(e)` of runner ERROR lines and tracebacks (`main.py:147, 203, 209`, `run.py:337, 390, 460, 1132`,
  `tools.py:134, 208, 215`, `team_flow.py:296`), allowlisted ErrorFrame lines and websocket client
  exception text: provider or validation error text, not audited for echoed content. The openai SDK
  error text is the provider response body (`_base_client.py:413-420`), not the request. A pydantic
  ValidationError text would carry `input_value`; whether `init_pipeline` can raise one is not verified.
- stdlib `logging` (uvicorn, openai, google-genai, websockets) is outside loguru and not audited
  beyond URL and id lines.
- Go services (logrus fixed at debug). CI inclusion of runner pytest.
- Compose: `LOG_LEVEL` is not wired into `komodo/docker-compose.yml` (a Komodo variable would be
  needed first); the default INFO needs no change, and enabling DEBUG needs a literal
  `LOG_LEVEL=DEBUG` runner env entry and redeploy by infra.
