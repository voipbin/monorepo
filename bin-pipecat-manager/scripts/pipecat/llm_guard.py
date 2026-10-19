"""Runner guards against silent LLM failures (VOIP-1633).

Design: docs/plans/2026-10-20-runner-silent-failures-design.md.

Two pieces, both reusing the existing ErrorFrame -> RTVI "error" -> pipeline_error path:

* EmptyOutputObserver: a response that ends with nothing usable (empty completion, or tool calls
  that pipecat dropped as malformed) becomes a non fatal ErrorFrame with a fixed text.
* normalize_tool_call_ids: a tool call without id gets a generated one before pipecat records it,
  and a call without a name is dropped, so neither poisons the following turns.

No log line here carries user, prompt or model text: only the pipecatcall id, counts and an exception class name.
The fixed error texts must not contain a token of classifyPipelineError (Go), so they stay
"unknown"; a Go test pins that.
"""

import uuid

from loguru import logger
from pipecat.frames.frames import (
    CancelFrame,
    EagerEndOfTurnCancelFrame,
    EndFrame,
    ErrorFrame,
    FunctionCallsStartedFrame,
    InterruptionFrame,
    LLMContextFrame,
    LLMFullResponseEndFrame,
    LLMFullResponseStartFrame,
    LLMTextFrame,
)
from pipecat.observers.base_observer import BaseObserver, FramePushed
from pipecat.services.llm_service import LLMService

EMPTY_OUTPUT_ERROR = "LLM returned no usable output"


def _tail_role_is_tool(frame: LLMContextFrame) -> bool:
    """True when the context handed to the LLM ends with a tool result (read errors: False)."""
    try:
        last = frame.context.get_messages()[-1]
        role = last.get("role") if isinstance(last, dict) else getattr(last, "role", None)
        return role == "tool"
    except Exception:
        return False


def _is_output(frame) -> bool:
    """Frames that show an LLM response was not silently empty."""
    if isinstance(frame, LLMTextFrame):
        return bool(frame.text and frame.text.strip())
    if isinstance(frame, ErrorFrame):
        # an STT or TTS error says nothing about the LLM answer
        return isinstance(frame.processor, LLMService)
    return isinstance(
        frame,
        (FunctionCallsStartedFrame, InterruptionFrame, CancelFrame, EndFrame, EagerEndOfTurnCancelFrame),
    )


class EmptyOutputObserver(BaseObserver):
    """Reports an LLM response that closes with nothing usable. See the module doc."""

    def __init__(self, pipecatcall_id: str = ""):
        # each frame once (by frame id): a later re-push of the same End frame is not seen again
        super().__init__(observe_every_push=False)
        self._pipecatcall_id = pipecatcall_id
        self._open = False
        self._produced = False
        self._after_tool = False
        self._tail_is_tool = False

    async def on_push_frame(self, data: FramePushed):
        # An exception must not escape: it would end the observer's task and stop all later detection.
        try:
            await self._observe(data)
        except Exception as e:
            logger.error(f"[empty_completion] observer failed. pipecatcall_id={self._pipecatcall_id} error_type={type(e).__name__}")

    async def _observe(self, data: FramePushed):
        frame = data.frame
        if isinstance(frame, LLMContextFrame):
            self._tail_is_tool = _tail_role_is_tool(frame)
        elif isinstance(frame, LLMFullResponseStartFrame):
            self._open = True
            self._produced = False
            self._after_tool = self._tail_is_tool
            self._tail_is_tool = False
        elif isinstance(frame, LLMFullResponseEndFrame):
            await self._close(data)
        elif self._open and _is_output(frame):
            self._produced = True

    async def _close(self, data: FramePushed):
        if not self._open:
            return
        self._open = False
        if self._produced:
            return
        if self._after_tool:
            # saying nothing after a tool action is a normal model choice: log only
            logger.warning(f"[empty_completion] after_tool=true pipecatcall_id={self._pipecatcall_id}")
            return
        logger.error(f"[empty_completion] pipecatcall_id={self._pipecatcall_id}")
        await data.source.push_error(error_msg=EMPTY_OUTPUT_ERROR)


def normalize_tool_call_ids(llm_service, pipecatcall_id: str = ""):
    """Wrap run_function_calls of one LLM service instance.

    A call with an empty id gets a generated one (before pipecat records it in the context and
    before tool_execute posts it). A call with an empty name is dropped. Nothing else changes: a
    registered tool runs as before, an unregistered name still goes to pipecat's missing handler.
    """
    original = llm_service.run_function_calls

    async def run_function_calls(function_calls):
        kept = []
        dropped = 0
        for fc in function_calls:
            if not fc.function_name:
                dropped += 1
                continue
            if not fc.tool_call_id:
                fc.tool_call_id = f"call_{uuid.uuid4().hex}"
            kept.append(fc)
        if dropped:
            logger.warning(f"[tool_call] dropped calls without a name. pipecatcall_id={pipecatcall_id} count={dropped}")
        return await original(kept if dropped else function_calls)

    llm_service.run_function_calls = run_function_calls
