"""Tests for build_team_flow() conversation history injection."""

import sys
from unittest.mock import MagicMock

# conftest.py mocks team_flow itself (for run.py tests). Remove it so we
# can import the real module. conftest already stubs pipecat_flows, common,
# aiohttp, etc., which team_flow.py needs.
sys.modules.pop("team_flow", None)

from team_flow import build_team_flow  # noqa: E402


def _make_team(members, start_member_id):
    """Helper to build a minimal resolved_team dict."""
    return {
        "start_member_id": start_member_id,
        "members": members,
    }


def _make_member(member_id, name="Agent", init_prompt="Hello", tools=None, transitions=None):
    m = {
        "id": member_id,
        "name": name,
        "ai": {
            "engine_model": "openai.gpt-4o",
            "engine_key": "fake-key",
            "init_prompt": init_prompt,
        },
    }
    if tools:
        m["tools"] = tools
    if transitions:
        m["transitions"] = transitions
    return m


class TestBuildTeamFlowConversationHistory:
    """Tests for llm_messages injection into start node's task_messages."""

    def test_llm_messages_injected_into_start_node(self):
        """Conversation history should appear in start node's task_messages."""
        member = _make_member("m1", init_prompt="You are helpful.")
        team = _make_team([member], "m1")
        llm_messages = [
            {"role": "user", "content": "Hi there"},
            {"role": "assistant", "content": "Hello! How can I help?"},
        ]

        member_nodes, start_node = build_team_flow(
            team, "pc-1", MagicMock(), None, None,
            llm_messages=llm_messages,
        )

        assert start_node["task_messages"] == llm_messages
        # start_node is the same reference as in member_nodes
        assert member_nodes["m1"]["task_messages"] == llm_messages

    def test_empty_llm_messages_leaves_task_messages_empty(self):
        """Empty list should not modify task_messages."""
        member = _make_member("m1")
        team = _make_team([member], "m1")

        _, start_node = build_team_flow(
            team, "pc-1", MagicMock(), None, None,
            llm_messages=[],
        )

        assert start_node["task_messages"] == []

    def test_none_llm_messages_leaves_task_messages_empty(self):
        """None (default) should not modify task_messages."""
        member = _make_member("m1")
        team = _make_team([member], "m1")

        _, start_node = build_team_flow(
            team, "pc-1", MagicMock(), None, None,
            llm_messages=None,
        )

        assert start_node["task_messages"] == []

    def test_no_llm_messages_param_leaves_task_messages_empty(self):
        """Omitting llm_messages entirely should not modify task_messages."""
        member = _make_member("m1")
        team = _make_team([member], "m1")

        _, start_node = build_team_flow(
            team, "pc-1", MagicMock(), None, None,
        )

        assert start_node["task_messages"] == []

    def test_malformed_messages_filtered_out(self):
        """Messages missing role or content should be excluded."""
        member = _make_member("m1")
        team = _make_team([member], "m1")
        llm_messages = [
            {"role": "user", "content": "valid message"},
            {"role": "assistant"},                          # missing content
            {"content": "orphaned content"},                # missing role
            {},                                             # empty
            {"role": "", "content": "empty role"},          # empty string role
            {"role": "user", "content": ""},                # empty string content
            {"role": "assistant", "content": "also valid"},
        ]

        _, start_node = build_team_flow(
            team, "pc-1", MagicMock(), None, None,
            llm_messages=llm_messages,
        )

        assert start_node["task_messages"] == [
            {"role": "user", "content": "valid message"},
            {"role": "assistant", "content": "also valid"},
        ]

    def test_only_start_node_gets_history(self):
        """Non-start members should keep empty task_messages."""
        m1 = _make_member("m1", name="Sales", init_prompt="I am sales.")
        m2 = _make_member("m2", name="Support", init_prompt="I am support.")
        team = _make_team([m1, m2], "m1")
        llm_messages = [
            {"role": "user", "content": "I need help"},
        ]

        member_nodes, start_node = build_team_flow(
            team, "pc-1", MagicMock(), None, None,
            llm_messages=llm_messages,
        )

        assert start_node["task_messages"] == llm_messages
        assert member_nodes["m2"]["task_messages"] == []

    def test_role_messages_preserved_with_history(self):
        """System prompt in role_messages should not be affected by llm_messages injection."""
        member = _make_member("m1", init_prompt="You are a helpful assistant.")
        team = _make_team([member], "m1")
        llm_messages = [
            {"role": "user", "content": "Hello"},
        ]

        _, start_node = build_team_flow(
            team, "pc-1", MagicMock(), None, None,
            llm_messages=llm_messages,
        )

        assert start_node["role_messages"] == [
            {"role": "system", "content": "You are a helpful assistant."},
        ]
        assert start_node["task_messages"] == llm_messages
