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


class TestAudioSampleRateConfiguration:
    """Regression tests for audio sample rate configuration.

    Pipecat defaults to 24kHz output (PipelineParams.audio_out_sample_rate=24000).
    Asterisk chan_websocket uses 16kHz slin16. Without explicitly setting
    audio_out_sample_rate=16000, TTS generates 24kHz audio that Go's per-chunk
    resampler converts with boundary artifacts every 40ms — causing robotic audio.

    The pipeline input sample rate defaults to 16kHz, matching Asterisk.
    """

    def _parse_run_py(self):
        """Parse run.py into an AST."""
        run_path = os.path.join(os.path.dirname(__file__), "run.py")
        with open(run_path) as f:
            return ast.parse(f.read(), filename="run.py")

    def test_pipeline_params_sets_16khz_output(self):
        """PipelineParams must set audio_out_sample_rate=16000.

        Without this, Pipecat defaults to 24kHz output. The Go resampler
        creates a new instance per audio chunk (no filter state across
        boundaries), causing robotic/choppy audio artifacts.
        """
        tree = self._parse_run_py()
        found = False
        for node in ast.walk(tree):
            if not isinstance(node, ast.Call):
                continue
            func = node.func
            is_pipeline_params = (
                (isinstance(func, ast.Name) and func.id == "PipelineParams") or
                (isinstance(func, ast.Attribute) and func.attr == "PipelineParams")
            )
            if not is_pipeline_params:
                continue
            for kw in node.keywords:
                if kw.arg == "audio_out_sample_rate":
                    assert isinstance(kw.value, ast.Constant), (
                        "audio_out_sample_rate must be a constant value"
                    )
                    assert kw.value.value == 16000, (
                        f"audio_out_sample_rate must be 16000 (matching Asterisk slin16), "
                        f"got {kw.value.value}. Pipecat defaults to 24kHz which causes "
                        f"robotic audio due to Go's per-chunk resampling."
                    )
                    found = True
        assert found, (
            "PipelineParams must include audio_out_sample_rate=16000. "
            "Without it, Pipecat defaults to 24kHz and Go's per-chunk resampler "
            "creates boundary artifacts causing robotic/choppy audio."
        )

    def test_no_explicit_tts_sample_rate(self):
        """TTS services should not override sample_rate.

        When audio_out_sample_rate=16000 is set in PipelineParams, TTS services
        inherit it via the StartFrame. Explicitly setting a different sample_rate
        on TTS would bypass the 16kHz pipeline and re-introduce Go-side resampling.
        """
        tree = self._parse_run_py()
        tts_constructors = {"CartesiaTTSService", "ElevenLabsTTSService", "GoogleTTSService"}
        for node in ast.walk(tree):
            if not isinstance(node, ast.Call):
                continue
            func = node.func
            name = None
            if isinstance(func, ast.Name):
                name = func.id
            elif isinstance(func, ast.Attribute):
                name = func.attr
            if name not in tts_constructors:
                continue
            for kw in node.keywords:
                if kw.arg == "sample_rate":
                    if isinstance(kw.value, ast.Constant) and kw.value.value != 16000:
                        pytest.fail(
                            f"{name} sets sample_rate={kw.value.value} which differs from "
                            f"the pipeline's 16kHz. This would cause Go-side resampling "
                            f"with boundary artifacts. Remove the explicit sample_rate to "
                            f"inherit 16kHz from PipelineParams."
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
