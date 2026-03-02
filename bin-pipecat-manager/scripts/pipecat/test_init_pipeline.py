"""Tests for pipeline init error handling.

Verifies that init_pipeline raises exceptions for invalid configurations
instead of silently failing in a background task.
"""
import pytest
from unittest.mock import patch, MagicMock, AsyncMock

from run import init_pipeline, init_team_pipeline, init_single_ai_pipeline


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
    with pytest.raises(ValueError, match="Unknown member_id"):
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
