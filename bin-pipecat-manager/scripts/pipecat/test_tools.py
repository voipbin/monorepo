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
_strip_unsupported_schema_fields = _tools._strip_unsupported_schema_fields


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

    def test_strips_additional_properties(self):
        """Gemini rejects additionalProperties — must be stripped."""
        tools = [
            {
                "name": "set_variables",
                "description": "Save variables",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "variables": {
                            "type": "object",
                            "description": "Key-value pairs",
                            "additionalProperties": {"type": "string"},
                        },
                    },
                    "required": ["variables"],
                },
            },
        ]
        result = convert_to_gemini_format(tools)
        decl = result[0]["function_declarations"][0]
        variables_schema = decl["parameters"]["properties"]["variables"]
        assert "additionalProperties" not in variables_schema
        assert variables_schema["type"] == "object"
        assert variables_schema["description"] == "Key-value pairs"

    def test_strips_default_field(self):
        """Gemini rejects default — must be stripped."""
        tools = [
            {
                "name": "test_tool",
                "description": "A tool",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "count": {"type": "integer", "default": 10},
                    },
                    "required": [],
                },
            },
        ]
        result = convert_to_gemini_format(tools)
        count_schema = result[0]["function_declarations"][0]["parameters"]["properties"]["count"]
        assert "default" not in count_schema
        assert count_schema["type"] == "integer"


class TestStripUnsupportedSchemaFields:
    def test_returns_non_dict_unchanged(self):
        assert _strip_unsupported_schema_fields("hello") == "hello"
        assert _strip_unsupported_schema_fields(42) == 42
        assert _strip_unsupported_schema_fields(None) is None

    def test_strips_top_level(self):
        schema = {"type": "object", "additionalProperties": {"type": "string"}}
        result = _strip_unsupported_schema_fields(schema)
        assert result == {"type": "object"}

    def test_strips_nested_properties(self):
        schema = {
            "type": "object",
            "properties": {
                "inner": {
                    "type": "object",
                    "additionalProperties": {"type": "number"},
                    "default": {},
                },
            },
        }
        result = _strip_unsupported_schema_fields(schema)
        assert "additionalProperties" not in result["properties"]["inner"]
        assert "default" not in result["properties"]["inner"]
        assert result["properties"]["inner"]["type"] == "object"

    def test_strips_inside_array_items(self):
        schema = {
            "type": "array",
            "items": {
                "type": "object",
                "additionalProperties": True,
            },
        }
        result = _strip_unsupported_schema_fields(schema)
        assert "additionalProperties" not in result["items"]
        assert result["items"]["type"] == "object"

    def test_preserves_supported_fields(self):
        schema = {
            "type": "object",
            "description": "A schema",
            "properties": {"name": {"type": "string"}},
            "required": ["name"],
        }
        result = _strip_unsupported_schema_fields(schema)
        assert result == schema
