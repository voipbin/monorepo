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

    @patch("run.GoogleLLMService")
    @patch("run.OpenAILLMContext")
    def test_gemini_service_creation(self, mock_context, mock_service):
        """Test Gemini service is created with GoogleLLMService and correct parameters."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        llm, aggregator = create_llm_service(
            type="gemini.gemini-2.5-flash",
            key="google-test-key",
            messages=[{"role": "user", "content": "hello"}],
            tools=[]
        )

        mock_service.assert_called_once_with(api_key="google-test-key", model="gemini-2.5-flash")

    @patch("run.GoogleLLMService")
    @patch("run.OpenAILLMContext")
    def test_gemini_uses_env_var_fallback(self, mock_context, mock_service):
        """Test Gemini falls back to GOOGLE_API_KEY env var when key is empty."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        with patch.dict(os.environ, {"GOOGLE_API_KEY": "env-google-key"}):
            llm, aggregator = create_llm_service(
                type="gemini.gemini-1.5-pro",
                key="",
                messages=[],
                tools=[]
            )

        mock_service.assert_called_once_with(api_key="env-google-key", model="gemini-1.5-pro")

    @patch("run.GoogleLLMService")
    @patch("run.OpenAILLMContext")
    def test_gemini_with_colon_separator(self, mock_context, mock_service):
        """Test Gemini service works with colon separator format."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        llm, aggregator = create_llm_service(
            type="gemini:gemini-flash",
            key="test-key",
            messages=[],
            tools=[]
        )

        mock_service.assert_called_once_with(api_key="test-key", model="gemini-flash")

    @patch("run.GoogleLLMService")
    @patch("run.OpenAILLMContext")
    def test_gemini_case_insensitive(self, mock_context, mock_service):
        """Test Gemini service name matching is case-insensitive."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        llm, aggregator = create_llm_service(
            type="Gemini.gemini-2.5-flash",
            key="test-key",
            messages=[],
            tools=[]
        )

        mock_service.assert_called_once_with(api_key="test-key", model="gemini-2.5-flash")

    @patch("run.GoogleLLMService")
    @patch("run.OpenAILLMContext")
    def test_gemini_model_name_with_dots_preserved(self, mock_context, mock_service):
        """Test model names containing dots are preserved (split on first dot only)."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        llm, aggregator = create_llm_service(
            type="gemini.gemini-1.5-pro-latest",
            key="test-key",
            messages=[],
            tools=[]
        )

        mock_service.assert_called_once_with(api_key="test-key", model="gemini-1.5-pro-latest")

    @patch("run.GoogleLLMService")
    @patch("run.OpenAILLMContext")
    def test_gemini_filters_invalid_messages(self, mock_context, mock_service):
        """Test messages without role or content are filtered before passing to context."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        messages = [
            {"role": "system", "content": "You are helpful."},
            {"role": "", "content": "no role"},
            {"content": "missing role key"},
            {"role": "user"},
            {"role": "user", "content": ""},
            {"role": "user", "content": "valid message"},
        ]

        create_llm_service(
            type="gemini.gemini-2.5-flash",
            key="test-key",
            messages=messages,
            tools=[]
        )

        mock_context.assert_called_once()
        passed_messages = mock_context.call_args[1]["messages"]
        assert len(passed_messages) == 2
        assert passed_messages[0] == {"role": "system", "content": "You are helpful."}
        assert passed_messages[1] == {"role": "user", "content": "valid message"}

    @patch("run.GoogleLLMService")
    @patch("run.OpenAILLMContext")
    def test_gemini_passes_tools_to_context(self, mock_context, mock_service):
        """Test tools list is correctly passed to OpenAILLMContext."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_llm.create_context_aggregator.return_value = MagicMock()

        tools = [{"type": "function", "function": {"name": "connect_call", "parameters": {}}}]

        create_llm_service(
            type="gemini.gemini-2.5-flash",
            key="test-key",
            messages=[],
            tools=tools
        )

        mock_context.assert_called_once_with(messages=[], tools=tools)

    @patch("run.GoogleLLMService")
    @patch("run.OpenAILLMContext")
    def test_gemini_returns_llm_and_aggregator(self, mock_context, mock_service):
        """Test Gemini returns the correct (llm, aggregator) tuple."""
        from run import create_llm_service

        mock_llm = MagicMock()
        mock_service.return_value = mock_llm
        mock_aggregator = MagicMock()
        mock_llm.create_context_aggregator.return_value = mock_aggregator

        mock_ctx_instance = MagicMock()
        mock_context.return_value = mock_ctx_instance

        llm, aggregator = create_llm_service(
            type="gemini.gemini-2.5-flash",
            key="test-key",
            messages=[],
            tools=[]
        )

        assert llm is mock_llm
        assert aggregator is mock_aggregator
        mock_llm.create_context_aggregator.assert_called_once_with(mock_ctx_instance)

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


class TestParseLanguage:
    """Tests for _parse_language helper function."""

    def test_valid_en_us(self):
        from run import _parse_language
        from pipecat.transcriptions.language import Language

        assert _parse_language("en-US") == Language.EN_US

    def test_valid_fr_fr(self):
        from run import _parse_language
        from pipecat.transcriptions.language import Language

        assert _parse_language("fr-FR") == Language.FR_FR

    def test_valid_ja_jp(self):
        from run import _parse_language
        from pipecat.transcriptions.language import Language

        assert _parse_language("ja-JP") == Language.JA_JP

    def test_valid_de_de(self):
        from run import _parse_language
        from pipecat.transcriptions.language import Language

        assert _parse_language("de-DE") == Language.DE_DE

    def test_unknown_language_falls_back(self):
        from run import _parse_language
        from pipecat.transcriptions.language import Language

        assert _parse_language("xx-YY") == Language.EN_US

    def test_empty_string_falls_back(self):
        from run import _parse_language
        from pipecat.transcriptions.language import Language

        assert _parse_language("") == Language.EN_US


class TestCreateTTSService:
    """Tests for create_tts_service function."""

    @patch("run.GoogleTTSService")
    def test_google_tts_service_creation(self, mock_service):
        """Test Google TTS service is created with voice_id and no explicit credentials."""
        from run import create_tts_service

        create_tts_service("google", voice_id="en-US-Chirp3-HD-Charon")

        mock_service.assert_called_once_with(voice_id="en-US-Chirp3-HD-Charon")

    @patch("run.GoogleTTSService")
    def test_google_tts_default_voice(self, mock_service):
        """Test Google TTS uses default voice when none specified."""
        from run import create_tts_service

        create_tts_service("google")

        mock_service.assert_called_once_with(voice_id="default_voice_id")

    @patch("run.CartesiaTTSService")
    def test_cartesia_still_works(self, mock_service):
        """Test existing Cartesia provider is not broken."""
        from run import create_tts_service

        with patch.dict(os.environ, {"CARTESIA_API_KEY": "test-key"}):
            create_tts_service("cartesia", voice_id="test-voice", language="en")

        mock_service.assert_called_once_with(
            api_key="test-key",
            voice_id="test-voice",
            language="en",
        )

    def test_unsupported_tts_raises_error(self):
        """Test unsupported TTS provider raises ValueError."""
        from run import create_tts_service

        with pytest.raises(ValueError, match="Unsupported TTS service"):
            create_tts_service("nonexistent")


class TestCreateSTTService:
    """Tests for create_stt_service function."""

    @patch("run.GoogleSTTService")
    def test_google_stt_service_creation(self, mock_service):
        """Test Google STT service is created with language mapped to Language enum."""
        from run import create_stt_service
        from pipecat.transcriptions.language import Language

        create_stt_service("google", language="en-US")

        # Verify InputParams was called with the correct language
        mock_service.InputParams.assert_called_once()
        ip_kwargs = mock_service.InputParams.call_args[1]
        assert ip_kwargs["languages"] == [Language.EN_US]
        assert ip_kwargs["model"] == "latest_long"
        assert ip_kwargs["enable_automatic_punctuation"] is True
        assert ip_kwargs["enable_interim_results"] is True

        # Verify GoogleSTTService was called with the params result
        mock_service.assert_called_once_with(params=mock_service.InputParams.return_value)

    @patch("run.GoogleSTTService")
    def test_google_stt_default_language(self, mock_service):
        """Test Google STT defaults to EN_US when no language provided."""
        from run import create_stt_service
        from pipecat.transcriptions.language import Language

        create_stt_service("google")

        ip_kwargs = mock_service.InputParams.call_args[1]
        assert ip_kwargs["languages"] == [Language.EN_US]

    @patch("run.GoogleSTTService")
    def test_google_stt_unknown_language_fallback(self, mock_service):
        """Test Google STT falls back to EN_US for unknown language strings."""
        from run import create_stt_service
        from pipecat.transcriptions.language import Language

        create_stt_service("google", language="xx-YY")

        ip_kwargs = mock_service.InputParams.call_args[1]
        assert ip_kwargs["languages"] == [Language.EN_US]

    @patch("run.GoogleSTTService")
    def test_google_stt_non_english_language(self, mock_service):
        """Test Google STT correctly maps non-English language strings."""
        from run import create_stt_service
        from pipecat.transcriptions.language import Language

        create_stt_service("google", language="fr-FR")

        ip_kwargs = mock_service.InputParams.call_args[1]
        assert ip_kwargs["languages"] == [Language.FR_FR]

    @patch("run.GoogleSTTService")
    def test_google_stt_empty_string_language(self, mock_service):
        """Test Google STT defaults to EN_US when language is empty string."""
        from run import create_stt_service
        from pipecat.transcriptions.language import Language

        create_stt_service("google", language="")

        ip_kwargs = mock_service.InputParams.call_args[1]
        assert ip_kwargs["languages"] == [Language.EN_US]

    @patch("run.DeepgramSTTService")
    def test_deepgram_still_works(self, mock_service):
        """Test existing Deepgram provider is not broken."""
        from run import create_stt_service

        with patch.dict(os.environ, {"DEEPGRAM_API_KEY": "test-key"}):
            create_stt_service("deepgram", language="en")

        mock_service.assert_called_once()

    def test_unsupported_stt_raises_error(self):
        """Test unsupported STT provider raises ValueError."""
        from run import create_stt_service

        with pytest.raises(ValueError, match="Unsupported STT service"):
            create_stt_service("nonexistent")
