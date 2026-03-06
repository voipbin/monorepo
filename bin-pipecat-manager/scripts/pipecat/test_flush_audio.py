"""Tests for the barge-in audio flush signal.

Since the test environment mocks all pipecat modules (via conftest.py),
UnpacedWebsocketClientOutputTransport inherits from MagicMock which swallows
method definitions. These tests use AST inspection to verify the source code
contains the correct patterns — the same approach used by existing tests
(e.g., test_run.py::TestPipelineParamsAudioSampleRate).
"""

import ast
import inspect
import textwrap


def _get_run_source():
    """Read run.py source code for AST analysis."""
    import pathlib
    return pathlib.Path(__file__).parent.joinpath("run.py").read_text()


def _get_class_node(tree, class_name):
    """Find a class definition by name in the AST."""
    for node in ast.walk(tree):
        if isinstance(node, ast.ClassDef) and node.name == class_name:
            return node
    return None


def _get_method_node(class_node, method_name):
    """Find a method definition by name in a class AST node."""
    for node in class_node.body:
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)) and node.name == method_name:
            return node
    return None


class TestUnpacedOutputTransportFlush:
    """Verify UnpacedWebsocketClientOutputTransport has correct flush logic."""

    def test_process_frame_override_exists(self):
        """UnpacedWebsocketClientOutputTransport must define process_frame."""
        source = _get_run_source()
        tree = ast.parse(source)
        cls = _get_class_node(tree, "UnpacedWebsocketClientOutputTransport")
        assert cls is not None, "UnpacedWebsocketClientOutputTransport class not found in run.py"

        method = _get_method_node(cls, "process_frame")
        assert method is not None, (
            "process_frame not defined on UnpacedWebsocketClientOutputTransport. "
            "The barge-in flush override may have been removed."
        )
        assert isinstance(method, ast.AsyncFunctionDef), "process_frame must be async"

    def test_process_frame_calls_super(self):
        """process_frame must call super().process_frame() before flush logic."""
        source = _get_run_source()
        tree = ast.parse(source)
        cls = _get_class_node(tree, "UnpacedWebsocketClientOutputTransport")
        method = _get_method_node(cls, "process_frame")
        assert method is not None, "process_frame not found"

        # Check that the first expression is await super().process_frame(...)
        method_source = ast.get_source_segment(source, method)
        assert "await super().process_frame" in method_source, (
            "process_frame must call await super().process_frame() to preserve "
            "base class interruption handling (cancel audio tasks, clear buffers)."
        )

    def test_process_frame_checks_interruption_frame(self):
        """process_frame must check isinstance(frame, InterruptionFrame)."""
        source = _get_run_source()
        tree = ast.parse(source)
        cls = _get_class_node(tree, "UnpacedWebsocketClientOutputTransport")
        method = _get_method_node(cls, "process_frame")
        assert method is not None, "process_frame not found"

        method_source = ast.get_source_segment(source, method)
        assert "isinstance(frame, InterruptionFrame)" in method_source, (
            "process_frame must check isinstance(frame, InterruptionFrame) "
            "to only send flush signal on barge-in events."
        )

    def test_process_frame_sends_flush_audio_text_frame(self):
        """process_frame must send TextFrame(text='flush_audio') on interruption."""
        source = _get_run_source()
        tree = ast.parse(source)
        cls = _get_class_node(tree, "UnpacedWebsocketClientOutputTransport")
        method = _get_method_node(cls, "process_frame")
        assert method is not None, "process_frame not found"

        method_source = ast.get_source_segment(source, method)
        assert 'TextFrame(text="flush_audio")' in method_source, (
            "process_frame must call _write_frame(TextFrame(text='flush_audio')) "
            "to signal Go to flush Asterisk's audio buffer."
        )

    def test_write_audio_sleep_is_noop(self):
        """_write_audio_sleep must be a no-op (pass only)."""
        source = _get_run_source()
        tree = ast.parse(source)
        cls = _get_class_node(tree, "UnpacedWebsocketClientOutputTransport")
        assert cls is not None

        method = _get_method_node(cls, "_write_audio_sleep")
        assert method is not None, "_write_audio_sleep not defined"
        assert isinstance(method, ast.AsyncFunctionDef), "_write_audio_sleep must be async"

        # Body should be a single 'pass' statement
        body = [n for n in method.body if not isinstance(n, ast.Expr) or not isinstance(n.value, ast.Constant)]
        assert len(body) == 1 and isinstance(body[0], ast.Pass), (
            "_write_audio_sleep must contain only 'pass'. "
            "Any sleep logic defeats the unpaced transport purpose."
        )

    def test_flush_audio_import_interruption_frame(self):
        """run.py must import InterruptionFrame from pipecat.frames.frames."""
        source = _get_run_source()
        assert "InterruptionFrame" in source, (
            "InterruptionFrame not imported. Required for barge-in detection."
        )

    def test_flush_audio_import_text_frame(self):
        """run.py must import TextFrame from pipecat.frames.frames."""
        source = _get_run_source()
        assert "TextFrame" in source, (
            "TextFrame not imported. Required for sending flush signal."
        )
