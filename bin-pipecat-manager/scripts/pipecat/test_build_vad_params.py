"""Tests for build_vad_params in run.py."""

from unittest.mock import MagicMock, patch
from run import build_vad_params


def test_none_input_returns_default():
    """None config should return VADParams with no arguments."""
    with patch("run.VADParams") as mock_vad:
        build_vad_params(None)
        mock_vad.assert_called_once_with()


def test_empty_dict_returns_default():
    """Empty dict should return VADParams with no arguments."""
    with patch("run.VADParams") as mock_vad:
        build_vad_params({})
        mock_vad.assert_called_once_with()


def test_full_config():
    """All four fields set should pass all kwargs."""
    config = {
        "confidence": 0.8,
        "start_secs": 0.3,
        "stop_secs": 0.5,
        "min_volume": 0.6,
    }
    with patch("run.VADParams") as mock_vad:
        build_vad_params(config)
        mock_vad.assert_called_once_with(
            confidence=0.8,
            start_secs=0.3,
            stop_secs=0.5,
            min_volume=0.6,
        )


def test_partial_config_stop_secs_only():
    """Only stop_secs set should pass only stop_secs."""
    config = {"stop_secs": 0.8}
    with patch("run.VADParams") as mock_vad:
        build_vad_params(config)
        mock_vad.assert_called_once_with(stop_secs=0.8)


def test_partial_config_confidence_only():
    """Only confidence set should pass only confidence."""
    config = {"confidence": 0.5}
    with patch("run.VADParams") as mock_vad:
        build_vad_params(config)
        mock_vad.assert_called_once_with(confidence=0.5)


def test_explicit_zero_values_are_passed():
    """Explicit 0.0 values should be passed, not treated as absent."""
    config = {
        "confidence": 0.0,
        "start_secs": 0.0,
        "stop_secs": 0.0,
        "min_volume": 0.0,
    }
    with patch("run.VADParams") as mock_vad:
        build_vad_params(config)
        mock_vad.assert_called_once_with(
            confidence=0.0,
            start_secs=0.0,
            stop_secs=0.0,
            min_volume=0.0,
        )


def test_none_values_in_dict_are_skipped():
    """Fields explicitly set to None in the dict should be skipped."""
    config = {
        "confidence": 0.7,
        "stop_secs": None,
    }
    with patch("run.VADParams") as mock_vad:
        build_vad_params(config)
        mock_vad.assert_called_once_with(confidence=0.7)


def test_unknown_fields_are_ignored():
    """Unknown fields should not be passed to VADParams."""
    config = {
        "confidence": 0.9,
        "unknown_field": 42,
    }
    with patch("run.VADParams") as mock_vad:
        build_vad_params(config)
        mock_vad.assert_called_once_with(confidence=0.9)
