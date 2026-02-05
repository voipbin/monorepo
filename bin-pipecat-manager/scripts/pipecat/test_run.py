import pytest
from unittest.mock import patch, MagicMock, call
import os
import ast
import inspect


class TestCreateLLMService:
    """Tests for create_llm_service function."""

    @patch("run.OpenAILLMService")
    @patch("run.OpenAILLMContext")
    def test_openai_service_creation(self, mock_context, mock_service):
        """Test OpenAI service is created with correct parameters."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        llm, aggregator = create_llm_service(
            type="openai.gpt-4o",
            key="test-key",
            messages=[{"role": "user", "content": "hello"}],
            tools=[]
        )

        mock_service.assert_called_once_with(api_key="test-key", model="gpt-4o")

    @patch("run.OpenAILLMService")
    @patch("run.OpenAILLMContext")
    def test_grok_service_creation(self, mock_context, mock_service):
        """Test Grok service is created with xAI base URL."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        llm, aggregator = create_llm_service(
            type="grok.grok-3",
            key="xai-test-key",
            messages=[{"role": "user", "content": "hello"}],
            tools=[]
        )

        mock_service.assert_called_once_with(
            api_key="xai-test-key",
            model="grok-3",
            base_url="https://api.x.ai/v1"
        )

    @patch("run.OpenAILLMService")
    @patch("run.OpenAILLMContext")
    def test_grok_uses_env_var_fallback(self, mock_context, mock_service):
        """Test Grok falls back to XAI_API_KEY env var."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        with patch.dict(os.environ, {"XAI_API_KEY": "env-xai-key"}):
            llm, aggregator = create_llm_service(
                type="grok.grok-3-mini",
                key="",
                messages=[],
                tools=[]
            )

        mock_service.assert_called_once_with(
            api_key="env-xai-key",
            model="grok-3-mini",
            base_url="https://api.x.ai/v1"
        )

    def test_unsupported_service_raises_error(self):
        """Test unsupported service raises ValueError."""
        from run import create_llm_service

        with pytest.raises(ValueError, match="Unsupported LLM service"):
            create_llm_service(
                type="unsupported.model",
                key="key",
                messages=[],
                tools=[]
            )

    def test_invalid_format_raises_error(self):
        """Test invalid format without dot raises ValueError."""
        from run import create_llm_service

        with pytest.raises(ValueError, match="Wrong LLM format"):
            create_llm_service(
                type="invalidformat",
                key="key",
                messages=[],
                tools=[]
            )


class TestNoManualRTVISetup:
    """Regression tests to prevent duplicate RTVI message emission.

    PipelineTask automatically sets up RTVIProcessor and RTVIObserver when
    enable_rtvi=True (the default since pipecat-ai v0.0.101). Manually adding
    them causes every RTVI message (bot-llm-text, etc.) to be emitted twice,
    resulting in duplicated/interleaved assistant response text.
    """

    def _parse_run_py(self):
        """Parse run.py into an AST."""
        run_path = os.path.join(os.path.dirname(__file__), "run.py")
        with open(run_path) as f:
            return ast.parse(f.read(), filename="run.py")

    def test_no_rtvi_processor_import(self):
        """RTVIProcessor must not be imported — PipelineTask creates it automatically."""
        tree = self._parse_run_py()
        for node in ast.walk(tree):
            if isinstance(node, ast.ImportFrom):
                for alias in node.names:
                    assert alias.name != "RTVIProcessor", (
                        "RTVIProcessor must not be imported in run.py. "
                        "PipelineTask automatically creates RTVIProcessor when enable_rtvi=True (default). "
                        "Manually adding it causes duplicate bot-llm-text messages."
                    )

    def test_no_rtvi_observer_import(self):
        """RTVIObserver must not be imported — PipelineTask registers it automatically."""
        tree = self._parse_run_py()
        for node in ast.walk(tree):
            if isinstance(node, ast.ImportFrom):
                for alias in node.names:
                    assert alias.name != "RTVIObserver", (
                        "RTVIObserver must not be imported in run.py. "
                        "PipelineTask automatically registers RTVIObserver when enable_rtvi=True (default). "
                        "Manually adding it causes duplicate bot-llm-text messages."
                    )

    def test_pipeline_task_not_called_with_observers(self):
        """PipelineTask must not receive an 'observers' argument with RTVIObserver.

        Scans the AST for PipelineTask(...) calls and verifies none pass
        an 'observers' keyword argument, which would duplicate the auto-registered observer.
        """
        tree = self._parse_run_py()
        for node in ast.walk(tree):
            if not isinstance(node, ast.Call):
                continue
            # Match PipelineTask(...) calls
            func = node.func
            is_pipeline_task = (
                (isinstance(func, ast.Name) and func.id == "PipelineTask") or
                (isinstance(func, ast.Attribute) and func.attr == "PipelineTask")
            )
            if not is_pipeline_task:
                continue
            for kw in node.keywords:
                assert kw.arg != "observers", (
                    "PipelineTask must not be called with 'observers' argument in run.py. "
                    "PipelineTask automatically registers RTVIObserver when enable_rtvi=True (default). "
                    "Passing observers=[RTVIObserver(...)] causes duplicate bot-llm-text messages."
                )

    def test_no_rtvi_processor_in_pipeline_stages(self):
        """Pipeline stages must not include an RTVIProcessor instance.

        Scans the AST for pipeline_stages.append(...) calls and verifies
        none append an RTVIProcessor (or a variable likely holding one).
        """
        tree = self._parse_run_py()
        for node in ast.walk(tree):
            if not isinstance(node, ast.Call):
                continue
            func = node.func
            # Match pipeline_stages.append(...)
            is_stages_append = (
                isinstance(func, ast.Attribute) and
                func.attr == "append" and
                isinstance(func.value, ast.Name) and
                func.value.id == "pipeline_stages"
            )
            if not is_stages_append:
                continue
            for arg in node.args:
                # Check for RTVIProcessor() direct instantiation
                if isinstance(arg, ast.Call):
                    callee = arg.func
                    if isinstance(callee, ast.Name) and callee.id == "RTVIProcessor":
                        pytest.fail(
                            "pipeline_stages must not include RTVIProcessor(). "
                            "PipelineTask handles RTVI setup automatically."
                        )
                # Check for a variable named 'rtvi' being appended
                if isinstance(arg, ast.Name) and arg.id == "rtvi":
                    pytest.fail(
                        "pipeline_stages must not include 'rtvi' variable. "
                        "PipelineTask handles RTVI setup automatically. "
                        "Manually adding RTVIProcessor causes duplicate bot-llm-text messages."
                    )
