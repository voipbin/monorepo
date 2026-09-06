"""Tests for pipeline init error handling.

Verifies that init_pipeline raises exceptions for invalid configurations
instead of silently failing in a background task.
"""
import pytest
from types import SimpleNamespace
from unittest.mock import call, patch, MagicMock, AsyncMock

from run import init_pipeline, init_team_pipeline, init_single_ai_pipeline


def _two_google_members():
    """Two members with Google TTS/STT, mirroring the production team d414dc93."""
    return [
        {
            "id": "member-1",
            "name": "Sales Assistant",
            "ai": {
                "engine_model": "gemini.gemini-2.5-flash",
                "engine_key": "",
                "tts_type": "google",
                "tts_voice_id": "en-US-Chirp3-HD-Gacrux",
                "stt_type": "google",
            },
            "tools": [],
            "transitions": [],
        },
        {
            "id": "member-2",
            "name": "General Assistant",
            "ai": {
                "engine_model": "gemini.gemini-2.5-flash",
                "engine_key": "",
                "tts_type": "google",
                "tts_voice_id": "",
                "stt_type": "google",
            },
            "tools": [],
            "transitions": [],
        },
    ]


async def _run_team_init(resolved_team, pipeline_id="test-gate", **kwargs):
    """Await init_team_pipeline on the success-path mock harness.

    Reuses the mock set from test_init_team_pipeline_swaps_flowmanager_llm_to_router
    and additionally patches the TTS/STT factories, the routing TTS/STT services and
    the VAD/transport factories so the audio-mode gate can be observed.

    Returns (ctx, mocks) where mocks exposes the patched factories.
    """
    mock_llm = MagicMock()

    mock_routing = MagicMock()
    mock_routing.active_service = mock_llm
    mock_routing.cleanup = AsyncMock()

    mock_task = MagicMock()
    mock_task.cancel = AsyncMock()
    mock_task.queue_frames = AsyncMock()

    mock_transport = MagicMock()
    mock_transport.cleanup = AsyncMock()

    flow_manager_stub = MagicMock()
    flow_manager_stub._llm = MagicMock()
    flow_manager_stub.initialize = AsyncMock()

    with patch("run.create_llm_service", return_value=(mock_llm, MagicMock())) as mock_create_llm, \
         patch("run.RoutingLLMService", return_value=mock_routing), \
         patch("run.create_tts_service") as mock_create_tts, \
         patch("run.create_stt_service") as mock_create_stt, \
         patch("run.RoutingTTSService") as mock_routing_tts, \
         patch("run.RoutingSTTService") as mock_routing_stt, \
         patch("run.SileroVADAnalyzer"), \
         patch("run.build_vad_params"), \
         patch("run.create_websocket_transport", return_value=mock_transport) as mock_create_transport, \
         patch("run.Pipeline"), \
         patch("run.PipelineTask", return_value=mock_task), \
         patch("run.build_team_flow", return_value=({}, MagicMock())), \
         patch("run.FlowManager", return_value=flow_manager_stub), \
         patch("run.task_manager") as mock_task_mgr:
        mock_task_mgr.add = AsyncMock()
        mock_task_mgr.remove = AsyncMock()

        ctx = await init_team_pipeline(
            id=pipeline_id,
            resolved_team=resolved_team,
            **kwargs,
        )

    mocks = SimpleNamespace(
        create_llm_service=mock_create_llm,
        create_tts_service=mock_create_tts,
        create_stt_service=mock_create_stt,
        RoutingTTSService=mock_routing_tts,
        RoutingSTTService=mock_routing_stt,
        create_websocket_transport=mock_create_transport,
    )
    return ctx, mocks


@pytest.mark.asyncio
async def test_init_single_ai_pipeline_unsupported_llm_type():
    """Unsupported LLM service name raises ValueError."""
    with pytest.raises(ValueError, match="Unsupported LLM service"):
        await init_single_ai_pipeline(
            id="test-1",
            llm_type="unsupported.model",
            llm_key="fake-key",
        )


@pytest.mark.asyncio
async def test_init_single_ai_pipeline_bad_llm_format():
    """LLM type without separator raises ValueError."""
    with pytest.raises(ValueError, match="Wrong LLM format"):
        await init_single_ai_pipeline(
            id="test-2",
            llm_type="no-separator",
            llm_key="fake-key",
        )


@pytest.mark.asyncio
async def test_init_single_ai_pipeline_unsupported_tts_type():
    """Unsupported TTS type raises ValueError."""
    with pytest.raises(ValueError, match="Unsupported TTS service"):
        await init_single_ai_pipeline(
            id="test-3",
            llm_type="openai.gpt-4o",
            llm_key="fake-key",
            tts_type="unsupported",
        )


@pytest.mark.asyncio
async def test_init_single_ai_pipeline_unsupported_stt_type():
    """Unsupported STT type raises ValueError."""
    with pytest.raises(ValueError, match="Unsupported STT service"):
        await init_single_ai_pipeline(
            id="test-4",
            llm_type="openai.gpt-4o",
            llm_key="fake-key",
            stt_type="unsupported",
        )


@pytest.mark.asyncio
async def test_init_team_pipeline_start_member_not_found():
    """start_member_id not in members raises ValueError."""
    resolved_team = {
        "id": "team-1",
        "start_member_id": "nonexistent-member",
        "members": [
            {
                "id": "member-1",
                "name": "Agent A",
                "ai": {
                    "engine_model": "openai.gpt-4o",
                    "engine_key": "fake-key",
                },
                "tools": [],
                "transitions": [],
            }
        ],
    }
    with pytest.raises(ValueError, match="start_member_id .* not found"):
        await init_team_pipeline(
            id="test-5",
            resolved_team=resolved_team,
        )


@pytest.mark.asyncio
async def test_init_team_pipeline_unsupported_member_llm():
    """Unsupported LLM type in a team member raises ValueError."""
    resolved_team = {
        "id": "team-2",
        "start_member_id": "member-1",
        "members": [
            {
                "id": "member-1",
                "name": "Agent A",
                "ai": {
                    "engine_model": "unsupported.model",
                    "engine_key": "fake-key",
                },
                "tools": [],
                "transitions": [],
            }
        ],
    }
    with pytest.raises(ValueError, match="Unsupported LLM service"):
        await init_team_pipeline(
            id="test-6",
            resolved_team=resolved_team,
        )


@pytest.mark.asyncio
@patch("run.RoutingLLMService")
async def test_init_team_pipeline_empty_members(mock_routing_llm):
    """Empty members list fails at start_member_id lookup."""
    resolved_team = {
        "id": "team-3",
        "start_member_id": "member-1",
        "members": [],
    }
    with pytest.raises(ValueError, match="start_member_id .* not found"):
        await init_team_pipeline(
            id="test-7",
            resolved_team=resolved_team,
        )


@pytest.mark.asyncio
async def test_init_team_pipeline_active_service_none_guard():
    """active_service returning None raises ValueError before FlowManager init.

    This guards against RoutingLLMService.active_service being None when
    the routing service is constructed but set_active_member somehow fails
    or the member ID doesn't match any service.
    """
    mock_llm = MagicMock()
    mock_llm.create_context_aggregator.return_value = MagicMock()

    mock_routing = MagicMock()
    mock_routing.active_service = None  # Simulate None active_service
    mock_routing.cleanup = AsyncMock()  # awaited in the except cleanup path

    resolved_team = {
        "id": "team-4",
        "start_member_id": "member-1",
        "members": [
            {
                "id": "member-1",
                "name": "Agent A",
                "ai": {
                    "engine_model": "openai.gpt-4o",
                    "engine_key": "fake-key",
                },
                "tools": [],
                "transitions": [],
            }
        ],
    }

    mock_task = MagicMock()
    mock_task.cancel = AsyncMock()

    mock_transport = MagicMock()
    mock_transport.cleanup = AsyncMock()

    with patch("run.create_llm_service") as mock_create_llm, \
         patch("run.RoutingLLMService", return_value=mock_routing), \
         patch("run.create_websocket_transport", return_value=mock_transport), \
         patch("run.PipelineTask", return_value=mock_task), \
         patch("run.Pipeline"), \
         patch("run.build_team_flow", return_value=({}, MagicMock())), \
         patch("run.task_manager") as mock_task_mgr:
        mock_create_llm.return_value = (mock_llm, MagicMock())
        mock_task_mgr.add = AsyncMock()
        mock_task_mgr.remove = AsyncMock()

        with pytest.raises(ValueError, match="No active LLM service"):
            await init_team_pipeline(
                id="test-8",
                resolved_team=resolved_team,
            )


@pytest.mark.asyncio
async def test_init_team_pipeline_flowmanager_missing_llm_attr_guard():
    """FlowManager without a _llm attribute raises RuntimeError.

    The runner swaps flow_manager._llm = routing_llm so register_function fans
    out to all team members. If a future pipecat-ai-flows renames that private
    attribute, the swap would silently no-op (non-start members get no tool
    registrations). The hasattr guard must fail loudly instead.
    """
    mock_llm = MagicMock()

    mock_routing = MagicMock()
    mock_routing.active_service = mock_llm  # valid -> proceeds to FlowManager
    mock_routing.cleanup = AsyncMock()

    resolved_team = {
        "id": "team-5",
        "start_member_id": "member-1",
        "members": [
            {
                "id": "member-1",
                "name": "Agent A",
                "ai": {"engine_model": "openai.gpt-4o", "engine_key": "fake-key"},
                "tools": [],
                "transitions": [],
            }
        ],
    }

    mock_task = MagicMock()
    mock_task.cancel = AsyncMock()
    mock_transport = MagicMock()
    mock_transport.cleanup = AsyncMock()

    # FlowManager stub WITHOUT a _llm attribute. Use spec=[] so hasattr(_, "_llm")
    # is False (a bare MagicMock would auto-create the attribute and pass).
    flow_manager_stub = MagicMock(spec=[])

    with patch("run.create_llm_service", return_value=(mock_llm, MagicMock())), \
         patch("run.RoutingLLMService", return_value=mock_routing), \
         patch("run.create_websocket_transport", return_value=mock_transport), \
         patch("run.PipelineTask", return_value=mock_task), \
         patch("run.Pipeline"), \
         patch("run.build_team_flow", return_value=({}, MagicMock())), \
         patch("run.FlowManager", return_value=flow_manager_stub), \
         patch("run.task_manager") as mock_task_mgr:
        mock_task_mgr.add = AsyncMock()
        mock_task_mgr.remove = AsyncMock()

        with pytest.raises(RuntimeError, match="no longer exposes _llm"):
            await init_team_pipeline(
                id="test-9",
                resolved_team=resolved_team,
            )


@pytest.mark.asyncio
async def test_init_team_pipeline_swaps_flowmanager_llm_to_router():
    """On success, flow_manager._llm is swapped to the routing service.

    This is the load-bearing behavior that makes register_function /
    unregister_function fan out to ALL team members (not just the start member).
    """
    mock_llm = MagicMock()

    mock_routing = MagicMock()
    mock_routing.active_service = mock_llm
    mock_routing.cleanup = AsyncMock()

    resolved_team = {
        "id": "team-6",
        "start_member_id": "member-1",
        "members": [
            {
                "id": "member-1",
                "name": "Agent A",
                "ai": {"engine_model": "openai.gpt-4o", "engine_key": "fake-key"},
                "tools": [],
                "transitions": [],
            }
        ],
    }

    mock_task = MagicMock()
    mock_task.cancel = AsyncMock()
    mock_task.queue_frames = AsyncMock()
    mock_transport = MagicMock()
    mock_transport.cleanup = AsyncMock()

    # FlowManager stub WITH a _llm attribute (so the guard passes) and an async
    # initialize() so the success path completes.
    flow_manager_stub = MagicMock()
    flow_manager_stub._llm = MagicMock()
    flow_manager_stub.initialize = AsyncMock()

    with patch("run.create_llm_service", return_value=(mock_llm, MagicMock())), \
         patch("run.RoutingLLMService", return_value=mock_routing), \
         patch("run.create_websocket_transport", return_value=mock_transport), \
         patch("run.Pipeline"), \
         patch("run.PipelineTask", return_value=mock_task), \
         patch("run.build_team_flow", return_value=({}, MagicMock())), \
         patch("run.FlowManager", return_value=flow_manager_stub), \
         patch("run.task_manager") as mock_task_mgr:
        mock_task_mgr.add = AsyncMock()
        mock_task_mgr.remove = AsyncMock()

        await init_team_pipeline(
            id="test-10",
            resolved_team=resolved_team,
        )

    assert flow_manager_stub._llm is mock_routing


@pytest.mark.asyncio
async def test_init_team_pipeline_preserves_paired_tool_call_messages():
    """init_team_pipeline routes llm_messages through filter_valid_messages.

    VOIP-1460: reuses the working success-path harness from
    test_init_team_pipeline_swaps_flowmanager_llm_to_router and adds
    patch("run.LLMContext") so the message list the shared LLMContext is
    built with can be asserted. The production shape (assistant message with
    content="" + tool_calls, followed by its role="tool" result) must survive
    intact; an unpaired tool-call request must still be dropped.
    """
    mock_llm = MagicMock()

    mock_routing = MagicMock()
    mock_routing.active_service = mock_llm
    mock_routing.cleanup = AsyncMock()

    resolved_team = {
        "id": "team-7",
        "start_member_id": "member-1",
        "members": [
            {
                "id": "member-1",
                "name": "Agent A",
                "ai": {
                    "engine_model": "openai.gpt-4o",
                    "engine_key": "fake-key",
                    "init_prompt": "You are helpful.",
                },
                "tools": [],
                "transitions": [],
            }
        ],
    }

    tool_call_request = {
        "role": "assistant",
        "content": "",
        "tool_calls": [
            {
                "id": "call_abc",
                "type": "function",
                "function": {"name": "get_resource", "arguments": "{}"},
            }
        ],
    }
    tool_call_result = {
        "role": "tool",
        "content": '{"tool_call_id": "call_abc", "result": "success"}',
        "tool_call_id": "call_abc",
    }
    unpaired_request = {
        "role": "assistant",
        "content": "",
        "tool_calls": [
            {
                "id": "call_orphan",
                "type": "function",
                "function": {"name": "unknown_tool", "arguments": "{}"},
            }
        ],
    }

    llm_messages = [
        {"role": "user", "content": "What is the status?"},
        tool_call_request,
        tool_call_result,
        unpaired_request,
        {"role": "user", "content": "and now?"},
    ]

    mock_task = MagicMock()
    mock_task.cancel = AsyncMock()
    mock_task.queue_frames = AsyncMock()
    mock_transport = MagicMock()
    mock_transport.cleanup = AsyncMock()

    flow_manager_stub = MagicMock()
    flow_manager_stub._llm = MagicMock()
    flow_manager_stub.initialize = AsyncMock()

    with patch("run.create_llm_service", return_value=(mock_llm, MagicMock())), \
         patch("run.RoutingLLMService", return_value=mock_routing), \
         patch("run.create_websocket_transport", return_value=mock_transport), \
         patch("run.Pipeline"), \
         patch("run.PipelineTask", return_value=mock_task), \
         patch("run.build_team_flow", return_value=({}, MagicMock())), \
         patch("run.FlowManager", return_value=flow_manager_stub), \
         patch("run.LLMContext") as mock_context, \
         patch("run.task_manager") as mock_task_mgr:
        mock_task_mgr.add = AsyncMock()
        mock_task_mgr.remove = AsyncMock()

        await init_team_pipeline(
            id="test-11",
            resolved_team=resolved_team,
            llm_messages=llm_messages,
        )

    mock_context.assert_called_once()
    passed_messages = mock_context.call_args[1]["messages"]
    assert passed_messages == [
        {"role": "system", "content": "You are helpful."},
        {"role": "user", "content": "What is the status?"},
        tool_call_request,
        tool_call_result,
        {"role": "user", "content": "and now?"},
    ]


@pytest.mark.asyncio
async def test_init_team_pipeline_text_mode_skips_member_tts_stt():
    """VOIP-1481 (A1): text-only session builds no per-member TTS/STT.

    Both request-level stt_type/tts_type are absent (the shape ai-manager sends
    for conversation / task / contact_case listen turns), so the members' own
    Google TTS/STT configuration must be ignored: no service construction, no
    routing TTS/STT, no audio input transport. The output transport is still
    built because a text session delivers the LLM reply through it.
    """
    resolved_team = {
        "id": "team-text",
        "start_member_id": "member-1",
        "members": _two_google_members(),
    }

    ctx, mocks = await _run_team_init(resolved_team)

    mocks.create_tts_service.assert_not_called()
    mocks.create_stt_service.assert_not_called()
    mocks.RoutingTTSService.assert_not_called()
    mocks.RoutingSTTService.assert_not_called()

    assert mocks.create_llm_service.call_count == 2
    assert mocks.create_websocket_transport.call_count == 1
    assert mocks.create_websocket_transport.call_args[0][0] == "output"

    assert ctx["routing_tts"] is None
    assert ctx["routing_stt"] is None
    assert ctx["transport_input"] is None


@pytest.mark.asyncio
async def test_init_team_pipeline_voice_mode_builds_member_tts_stt_from_member_config():
    """VOIP-1481 (A2): voice session keeps today's per-member construction.

    The request-level values ("elevenlabs"/"deepgram") only signal the mode; each
    member's service is still built from that member's own vendor and voice id,
    and a member without a tts_type is skipped exactly as before.
    """
    resolved_team = {
        "id": "team-voice",
        "start_member_id": "member-1",
        "members": [
            {
                "id": "member-1",
                "name": "Agent A",
                "ai": {
                    "engine_model": "openai.gpt-4o",
                    "engine_key": "fake-key",
                    "tts_type": "google",
                    "tts_voice_id": "v-a",
                    "stt_type": "google",
                },
                "tools": [],
                "transitions": [],
            },
            {
                "id": "member-2",
                "name": "Agent B",
                "ai": {
                    "engine_model": "openai.gpt-4o",
                    "engine_key": "fake-key",
                    # "whisper" is deliberately an unimplemented-but-distinct
                    # vendor string: create_stt_service is mocked here, and the
                    # test needs three mutually distinct STT values (member-1
                    # "google", request-level "deepgram", member-2 this one).
                    "stt_type": "whisper",
                },
                "tools": [],
                "transitions": [],
            },
        ],
    }

    # Members use different vendors from each other (mixed-vendor team) and the
    # request-level types are deliberately different from every member's own
    # vendor, so that both "wrong member's vendor" and "request-level value
    # leaked into a member factory call" are visible.
    ctx, mocks = await _run_team_init(
        resolved_team,
        stt_type="deepgram",
        tts_type="elevenlabs",
        stt_language="en-US",
        tts_language="en-US",
    )

    mocks.create_tts_service.assert_called_once_with(
        "google", voice_id="v-a", language="en-US",
    )
    assert mocks.create_stt_service.call_args_list == [
        call("google", language="en-US"),
        call("whisper", language="en-US"),
    ]
    # The request-level TTS/STT types must never reach a member's service factory.
    for tts_call in mocks.create_tts_service.call_args_list:
        assert "elevenlabs" not in tts_call.args
        assert "elevenlabs" not in tts_call.kwargs.values()
    for stt_call in mocks.create_stt_service.call_args_list:
        assert "deepgram" not in stt_call.args
        assert "deepgram" not in stt_call.kwargs.values()

    mocks.RoutingTTSService.assert_called_once()
    mocks.RoutingSTTService.assert_called_once()

    assert mocks.create_websocket_transport.call_count == 2
    assert [c.args[0] for c in mocks.create_websocket_transport.call_args_list] == [
        "input", "output",
    ]
    assert ctx["transport_input"] is not None


@pytest.mark.asyncio
async def test_init_team_pipeline_partial_request_types_gate_independently():
    """VOIP-1481 (A3): TTS and STT are gated by their own request-level field."""
    resolved_team = {
        "id": "team-partial",
        "start_member_id": "member-1",
        "members": _two_google_members(),
    }

    ctx, mocks = await _run_team_init(
        resolved_team,
        stt_type="deepgram",
        tts_type=None,
        stt_language="en-US",
        tts_language="en-US",
    )

    assert mocks.create_stt_service.call_count == 2
    mocks.create_tts_service.assert_not_called()

    assert ctx["routing_stt"] is not None
    assert ctx["routing_tts"] is None
    assert ctx["transport_input"] is not None


@pytest.mark.asyncio
async def test_init_team_pipeline_partial_request_types_tts_only_builds_no_stt():
    """VOIP-1481 (A3, mirror case): tts_type alone builds member TTS but no STT.

    With no request-level stt_type there is no STT router and therefore no
    input transport; only the output transport is created.
    """
    resolved_team = {
        "id": "team-partial-tts",
        "start_member_id": "member-1",
        "members": _two_google_members(),
    }

    ctx, mocks = await _run_team_init(
        resolved_team,
        stt_type=None,
        tts_type="elevenlabs",
        stt_language="en-US",
        tts_language="en-US",
    )

    assert mocks.create_tts_service.call_count == 2
    mocks.create_stt_service.assert_not_called()

    assert ctx["routing_tts"] is not None
    assert ctx["routing_stt"] is None
    assert ctx["transport_input"] is None
    assert mocks.create_websocket_transport.call_count == 1
    assert mocks.create_websocket_transport.call_args_list[0].args[0] == "output"


@pytest.mark.asyncio
async def test_init_pipeline_forwards_request_types_to_team_branch():
    """VOIP-1481 (A4): init_pipeline forwards the request-level types to the team branch.

    Production calls init_pipeline positionally (main.py), so the assertion is on
    what the team branch receives. A fresh dict is returned per call because
    init_pipeline mutates the returned ctx.
    """
    resolved_team = {"id": "t", "start_member_id": "m", "members": []}

    with patch("run.init_team_pipeline", new=AsyncMock(side_effect=lambda *a, **k: {})) as mock_team:
        await init_pipeline(
            "id", "openai.gpt-4o", "k", [],
            "deepgram", "en-US", "elevenlabs", "en-US", "voice", [],
            resolved_team=resolved_team,
        )

        mock_team.assert_awaited_once()
        assert mock_team.call_args.kwargs["stt_type"] == "deepgram"
        assert mock_team.call_args.kwargs["tts_type"] == "elevenlabs"

    with patch("run.init_team_pipeline", new=AsyncMock(side_effect=lambda *a, **k: {})) as mock_team_text:
        await init_pipeline(
            "id", "openai.gpt-4o", "k", [],
            None, None, None, None, None, [],
            resolved_team=resolved_team,
        )

        mock_team_text.assert_awaited_once()
        assert mock_team_text.call_args.kwargs["stt_type"] is None
        assert mock_team_text.call_args.kwargs["tts_type"] is None
