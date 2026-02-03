import pytest
from unittest.mock import patch, MagicMock
import os


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
