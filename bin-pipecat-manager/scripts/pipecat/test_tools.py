"""Tests for tool format conversion functions."""

import importlib
import importlib.util
import os

# Import the real tools module, bypassing the conftest sys.modules mock.
_tools_path = os.path.join(os.path.dirname(__file__), "tools.py")
_spec = importlib.util.spec_from_file_location("tools_real", _tools_path)
_tools = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_tools)

convert_to_openai_format = _tools.convert_to_openai_format
convert_to_gemini_format = _tools.convert_to_gemini_format


SAMPLE_TOOLS = [
    {
        "name": "connect_call",
        "description": "Connect to another number",
        "parameters": {
            "type": "object",
            "properties": {"number": {"type": "string"}},
            "required": ["number"],
        },
    },
    {
        "name": "send_email",
        "description": "Send an email",
        "parameters": {
            "type": "object",
            "properties": {"to": {"type": "string"}, "body": {"type": "string"}},
            "required": ["to", "body"],
        },
    },
]


class TestConvertToOpenaiFormat:
    def test_empty_list(self):
        assert convert_to_openai_format([]) == []

    def test_none_returns_empty(self):
        assert convert_to_openai_format(None) == []

    def test_converts_to_openai_format(self):
        result = convert_to_openai_format(SAMPLE_TOOLS)
        assert len(result) == 2
        assert result[0] == {
            "type": "function",
            "function": {
                "name": "connect_call",
                "description": "Connect to another number",
                "parameters": {
                    "type": "object",
                    "properties": {"number": {"type": "string"}},
                    "required": ["number"],
                },
            },
        }
        assert result[1]["function"]["name"] == "send_email"


class TestConvertToGeminiFormat:
    def test_empty_list(self):
        assert convert_to_gemini_format([]) == []

    def test_none_returns_empty(self):
        assert convert_to_gemini_format(None) == []

    def test_converts_to_gemini_format(self):
        result = convert_to_gemini_format(SAMPLE_TOOLS)
        assert len(result) == 1
        assert "function_declarations" in result[0]
        declarations = result[0]["function_declarations"]
        assert len(declarations) == 2
        assert declarations[0] == {
            "name": "connect_call",
            "description": "Connect to another number",
            "parameters": {
                "type": "object",
                "properties": {"number": {"type": "string"}},
                "required": ["number"],
            },
        }
        assert declarations[1]["name"] == "send_email"

    def test_gemini_format_structure(self):
        """Verify the exact structure that GenerateContentConfig expects."""
        result = convert_to_gemini_format(SAMPLE_TOOLS)
        # Must be a list with one dict containing function_declarations
        assert isinstance(result, list)
        assert len(result) == 1
        assert isinstance(result[0], dict)
        assert list(result[0].keys()) == ["function_declarations"]
        # Each declaration has name, description, parameters
        for decl in result[0]["function_declarations"]:
            assert "name" in decl
            assert "description" in decl
            assert "parameters" in decl
