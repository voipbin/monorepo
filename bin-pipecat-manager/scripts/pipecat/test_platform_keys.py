"""Platform keys: the platform_keys of a /run request and how the runner uses them.

Source of a key (the same table as docs/operations.md):
- openai, grok, gemini: the key of the customer, then platform_keys, then the environment variable
  of the runner container. None of them: not configured.
- cartesia, elevenlabs, deepgram: platform_keys, then the environment variable.
- platform_openrouter: platform_keys, then the environment variable. The key of the customer is ignored.
- local: platform_keys["local_llm.endpoints"] of the request, else the variable LOCAL_LLM_ENDPOINTS,
  parsed for that session only; the key of the customer is ignored.
An empty value or an unresolved placeholder (starts with "[[") counts as not set. A key that is not
there is rejected before the vendor SDK object is built, because the SDK clients fall back to their
own environment variable when they get None or an empty key.

All keys here are fake. No test calls a vendor.
"""
import asyncio
import json
import os
import subprocess
import sys
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient
from loguru import logger

import main
import run

from test_init_pipeline import _member, _run_team_init

SENTINEL = "PLATFORM-SENTINEL-7f3a"
SECRET_URL = "http://llm-secret-host.example.invalid:8000/v1"
FIXED_LOCAL = "local engine model is not available"


@pytest.fixture
def logs():
    lines = []
    sink = logger.add(lines.append, level="DEBUG")
    yield lines
    logger.remove(sink)


def _text(lines):
    return "".join(str(m) for m in lines)


def _llm(type_, key="", platform_keys=None, service="OpenAILLMService"):
    """Build an LLM service with the vendor class patched. Returns the mock of the class."""
    with patch("run.LLMContextAggregatorPair"), patch("run.LLMContext"), \
         patch("run.drop_gemini_invalid_tools", side_effect=lambda t, _id: t), \
         patch(f"run.{service}") as svc:
        run.create_llm_service(type_, key, [{"role": "user", "content": "hi"}], [], platform_keys=platform_keys)
    return svc


# --- priority: llm ----------------------------------------------------------------------------------


def _llm_rejected(type_, key="", platform_keys=None, service="OpenAILLMService"):
    """The factory must raise before the vendor class is called. Returns (error, mock of the class)."""
    with patch("run.LLMContextAggregatorPair"), patch("run.LLMContext"), \
         patch("run.drop_gemini_invalid_tools", side_effect=lambda t, _id: t), \
         patch(f"run.{service}") as svc:
        with pytest.raises(ValueError) as e:
            run.create_llm_service(type_, key, [{"role": "user", "content": "hi"}], [], platform_keys=platform_keys)
    return e.value, svc


@pytest.mark.parametrize("type_, pk, env, service, vendor", [
    ("openai.gpt-5", run.PK_OPENAI, "OPENAI_API_KEY", "OpenAILLMService", "OpenAI"),
    ("grok.grok-4", run.PK_XAI, "XAI_API_KEY", "OpenAILLMService", "xAI"),
    ("gemini.gemini-2.5-flash", run.PK_GOOGLE, "GOOGLE_API_KEY", "GoogleLLMService", "Google"),
])
class TestCustomerKeyServices:
    def test_customer_key_wins(self, type_, pk, env, service, vendor, monkeypatch):
        monkeypatch.setenv(env, "env-value")
        svc = _llm(type_, key="customer-value", platform_keys={pk: "platform-value"}, service=service)
        assert svc.call_args.kwargs["api_key"] == "customer-value"

    def test_customer_key_does_not_look_up_the_platform_key(self, type_, pk, env, service, vendor):
        with patch("run._platform_key", side_effect=AssertionError("must not be called")):
            svc = _llm(type_, key="customer-value", platform_keys=None, service=service)
        assert svc.call_args.kwargs["api_key"] == "customer-value"

    def test_platform_key_is_used_without_a_customer_key(self, type_, pk, env, service, vendor, monkeypatch):
        monkeypatch.setenv(env, "env-value")
        svc = _llm(type_, platform_keys={pk: "platform-value"}, service=service)
        assert svc.call_args.kwargs["api_key"] == "platform-value"

    @pytest.mark.parametrize("platform_keys", [None, {}, "empty"])
    def test_env_is_used_without_a_request_key(self, type_, pk, env, service, vendor, platform_keys, monkeypatch):
        monkeypatch.setenv(env, "  env-value  ")
        if platform_keys == "empty":
            platform_keys = {pk: ""}
        svc = _llm(type_, platform_keys=platform_keys, service=service)
        assert svc.call_args.kwargs["api_key"] == "env-value"

    @pytest.mark.parametrize("value", [None, "", "   ", "[[BIN_MANAGER__X]]", " [[BIN_MANAGER__X]]"])
    def test_unset_empty_and_placeholder_are_rejected_before_the_sdk(self, type_, pk, env, service, vendor, value, monkeypatch):
        if value is not None:
            monkeypatch.setenv(env, value)
        err, svc = _llm_rejected(type_, platform_keys=None, service=service)
        assert str(err) == f"{vendor} is not configured"
        assert "BIN_MANAGER" not in str(err)
        svc.assert_not_called()

    def test_a_placeholder_does_not_hide_a_request_key(self, type_, pk, env, service, vendor, monkeypatch):
        monkeypatch.setenv(env, "[[BIN_MANAGER__X]]")
        svc = _llm(type_, platform_keys={pk: "platform-value"}, service=service)
        assert svc.call_args.kwargs["api_key"] == "platform-value"

    def test_a_variable_of_another_vendor_is_not_used(self, type_, pk, env, service, vendor, monkeypatch):
        for other_env in ("OPENAI_API_KEY", "XAI_API_KEY", "GOOGLE_API_KEY"):
            if other_env != env:
                monkeypatch.setenv(other_env, "other-value")
        other = {k: "other-value" for k in (run.PK_OPENAI, run.PK_XAI, run.PK_GOOGLE) if k != pk}
        err, svc = _llm_rejected(type_, platform_keys=other, service=service)
        assert str(err) == f"{vendor} is not configured"
        svc.assert_not_called()


class TestOpenRouter:
    def test_platform_key_beats_env_and_customer_key_is_ignored(self, monkeypatch):
        monkeypatch.setenv("OPENROUTER_API_KEY", "env-or")
        with patch("run._build_openrouter_llm") as build, patch("run.LLMContextAggregatorPair"), patch("run.LLMContext"):
            run.create_llm_service("platform_openrouter.anthropic/claude-haiku-4.5", "customer-or",
                                   [{"role": "user", "content": "hi"}], [], platform_keys={run.PK_OPENROUTER: "platform-or"})
        assert build.call_args.args[0] == "platform-or"
        assert "customer-or" not in repr(build.call_args)

    @pytest.mark.parametrize("platform_keys", [None, {}, {run.PK_OPENROUTER: ""}])
    def test_env_is_used_without_a_request_key(self, platform_keys, monkeypatch):
        monkeypatch.setenv("OPENROUTER_API_KEY", "env-or")
        with patch("run._build_openrouter_llm") as build, patch("run.LLMContextAggregatorPair"), patch("run.LLMContext"):
            run.create_llm_service("platform_openrouter.m", "customer-or", [], [], platform_keys=platform_keys)
        assert build.call_args.args[0] == "env-or"
        assert "customer-or" not in repr(build.call_args)

    @pytest.mark.parametrize("value", [None, "", "  ", "[[BIN_MANAGER__OPENROUTER_API_KEY]]"])
    def test_unset_empty_and_placeholder_are_rejected_before_the_sdk(self, value, monkeypatch):
        if value is not None:
            monkeypatch.setenv("OPENROUTER_API_KEY", value)
        with patch("run._build_openrouter_llm") as build, patch("run.LLMContextAggregatorPair"), patch("run.LLMContext"):
            with pytest.raises(ValueError) as e:
                run.create_llm_service("platform_openrouter.m", "customer-or", [], [], platform_keys=None)
        assert str(e.value) == "OpenRouter is not configured"
        build.assert_not_called()

    def test_customer_openrouter_never_reads_platform_or_env_keys(self, monkeypatch):
        monkeypatch.setenv("OPENROUTER_API_KEY", "env-or")
        pk = {run.PK_OPENROUTER: "platform-or"}
        with patch("run._build_openrouter_llm") as build, patch("run.LLMContextAggregatorPair"), patch("run.LLMContext"):
            with pytest.raises(ValueError, match="An API key is required"):
                run.create_llm_service("openrouter.anthropic/claude", "", [], [], platform_keys=pk)
            run.create_llm_service("openrouter.anthropic/claude", " customer-or ", [], [], platform_keys=pk)
        assert build.call_args.args[0] == "customer-or"


# --- priority: tts and stt --------------------------------------------------------------------------


@pytest.mark.parametrize("name, pk, env, cls, vendor", [
    ("cartesia", run.PK_CARTESIA, "CARTESIA_API_KEY", "CartesiaTTSService", "Cartesia"),
    ("elevenlabs", run.PK_ELEVENLABS, "ELEVENLABS_API_KEY", "ElevenLabsTTSService", "ElevenLabs"),
])
def test_tts_request_key_wins_then_env_then_rejected(name, pk, env, cls, vendor, monkeypatch):
    monkeypatch.setenv(env, "env-value")
    with patch(f"run.{cls}") as svc, patch("run._keep_usable", side_effect=lambda s: s):
        run.create_tts_service(name, voice_id="v", platform_keys={pk: "platform-value"})
        assert svc.call_args.kwargs["api_key"] == "platform-value"
        for platform_keys in (None, {}, {pk: ""}):
            run.create_tts_service(name.upper(), voice_id="v", platform_keys=platform_keys)
            assert svc.call_args.kwargs["api_key"] == "env-value"
        svc.reset_mock()
        for value in ("", "  ", "[[BIN_MANAGER__X]]"):
            monkeypatch.setenv(env, value)
            with pytest.raises(ValueError) as e:
                run.create_tts_service(name.upper(), voice_id="v", platform_keys=None)
            assert str(e.value) == f"{vendor} is not configured"
        monkeypatch.delenv(env)
        with pytest.raises(ValueError) as e:
            run.create_tts_service(name, voice_id="v")
        assert str(e.value) == f"{vendor} is not configured"
        svc.assert_not_called()


def test_stt_request_key_wins_then_env_then_rejected(monkeypatch):
    monkeypatch.setenv("DEEPGRAM_API_KEY", "env-value")
    with patch("run.DeepgramSTTService") as svc, patch("run._keep_usable", side_effect=lambda s: s), \
         patch("run._harden_deepgram", side_effect=lambda s: s), patch("run.LiveOptions"):
        run.create_stt_service("deepgram", language="en-US", platform_keys={run.PK_DEEPGRAM: "platform-value"})
        assert svc.call_args.kwargs["api_key"] == "platform-value"
        for platform_keys in (None, {}, {run.PK_DEEPGRAM: ""}):
            run.create_stt_service("deepgram", language="en-US", platform_keys=platform_keys)
            assert svc.call_args.kwargs["api_key"] == "env-value"
        svc.reset_mock()
        for value in ("", "  ", "[[BIN_MANAGER__DEEPGRAM_API_KEY]]"):
            monkeypatch.setenv("DEEPGRAM_API_KEY", value)
            with pytest.raises(ValueError) as e:
                run.create_stt_service("deepgram", language="en-US", platform_keys=None)
            assert str(e.value) == "Deepgram is not configured"
        svc.assert_not_called()


def test_google_tts_and_stt_take_no_platform_key():
    pk = {k: SENTINEL for k in (run.PK_CARTESIA, run.PK_ELEVENLABS, run.PK_DEEPGRAM, run.PK_GOOGLE)}
    with patch("run.GoogleTTSService") as tts, patch("run.GoogleSTTService") as stt, patch("run._keep_usable", side_effect=lambda s: s):
        run.create_tts_service("google", voice_id="en-US-Chirp3-HD-Charon", platform_keys=pk)
        run.create_stt_service("google", language="en-US", platform_keys=pk)
    assert SENTINEL not in repr(tts.call_args) and SENTINEL not in repr(stt.call_args)


# --- local endpoints --------------------------------------------------------------------------------


def _endpoints(**over):
    data = {"vllm-a": {"base_url": SECRET_URL, "api_key": "local-key-1"}}
    data.update(over)
    return json.dumps(data)


def _local(platform_keys, type_="local.vllm-a.qwen3-8b", key="customer-key"):
    with patch("run.LLMContextAggregatorPair"), patch("run.LLMContext"), patch("run.OpenAILLMService") as svc:
        run.create_llm_service(type_, key, [{"role": "user", "content": "hi"}], [], platform_keys=platform_keys)
    return svc


class TestLocalEndpoints:
    def test_request_value_is_used(self):
        svc = _local({run.PK_LOCAL_ENDPOINTS: _endpoints()})
        assert svc.call_args.kwargs == {"api_key": "local-key-1", "model": "qwen3-8b", "base_url": SECRET_URL}

    def test_request_value_has_only_its_own_endpoints(self):
        with pytest.raises(ValueError, match=FIXED_LOCAL):
            _local({run.PK_LOCAL_ENDPOINTS: _endpoints()}, type_="local.only-env.m")

    @pytest.mark.parametrize("platform_keys", [None, {}, {run.PK_LOCAL_ENDPOINTS: ""}])
    def test_environment_is_used_without_a_request_value(self, platform_keys, monkeypatch):
        monkeypatch.setenv("LOCAL_LLM_ENDPOINTS", "  " + _endpoints() + "  ")
        svc = _local(platform_keys)
        assert svc.call_args.kwargs == {"api_key": "local-key-1", "model": "qwen3-8b", "base_url": SECRET_URL}

    def test_request_value_wins_over_the_environment(self, monkeypatch):
        monkeypatch.setenv("LOCAL_LLM_ENDPOINTS", _endpoints(**{"vllm-a": {"base_url": "http://env.example.invalid/v1"}}))
        svc = _local({run.PK_LOCAL_ENDPOINTS: _endpoints()})
        assert svc.call_args.kwargs["base_url"] == SECRET_URL

    @pytest.mark.parametrize("value", [None, "", "   ", "[[BIN_MANAGER__LOCAL_LLM_ENDPOINTS]]"])
    def test_unset_empty_and_placeholder_environment_give_the_fixed_error(self, value, monkeypatch):
        if value is not None:
            monkeypatch.setenv("LOCAL_LLM_ENDPOINTS", value)
        with pytest.raises(ValueError) as e:
            _local(None)
        assert str(e.value) == FIXED_LOCAL

    def test_a_bad_environment_value_is_the_fixed_error_without_the_value(self, monkeypatch, logs):
        monkeypatch.setenv("LOCAL_LLM_ENDPOINTS", "{" + SECRET_URL + SENTINEL)
        with pytest.raises(ValueError) as e:
            _local(None)
        assert str(e.value) == FIXED_LOCAL
        assert SECRET_URL not in _text(logs) and SENTINEL not in _text(logs)

    def test_the_environment_is_read_for_each_call(self, monkeypatch):
        """No module state: a changed variable is seen by the next call."""
        monkeypatch.setenv("LOCAL_LLM_ENDPOINTS", _endpoints(**{"vllm-a": {"base_url": "http://one.example.invalid/v1"}}))
        assert _local(None).call_args.kwargs["base_url"] == "http://one.example.invalid/v1"
        monkeypatch.setenv("LOCAL_LLM_ENDPOINTS", _endpoints(**{"vllm-a": {"base_url": "http://two.example.invalid/v1"}}))
        assert _local(None).call_args.kwargs["base_url"] == "http://two.example.invalid/v1"
        monkeypatch.delenv("LOCAL_LLM_ENDPOINTS")
        with pytest.raises(ValueError, match=FIXED_LOCAL):
            _local(None)

    def test_the_module_has_no_environment_registry(self):
        assert not hasattr(run, "load_local_endpoints") and not hasattr(run, "_local_endpoints")

    def test_importing_the_module_does_not_read_the_environment(self):
        """Fresh interpreter: the variable is set before the import and removed before the call. A
        value read at import time would still be served."""
        here = Path(__file__).parent
        code = (
            "import os\n"
            "import conftest, run\n"
            "from unittest.mock import patch\n"
            "del os.environ['LOCAL_LLM_ENDPOINTS']\n"
            "with patch('run.LLMContextAggregatorPair'), patch('run.LLMContext'), patch('run.OpenAILLMService') as svc:\n"
            "    try:\n"
            "        run.create_llm_service('local.vllm-a.qwen3-8b', '', [], [], platform_keys=None)\n"
            "    except ValueError as e:\n"
            "        print('REJECTED:' + str(e))\n"
            "    print('CALLED:' + str(svc.called))\n"
        )
        out = subprocess.run([sys.executable, "-c", code], cwd=here, capture_output=True, text=True, timeout=120,
                             env={**os.environ, "LOCAL_LLM_ENDPOINTS": _endpoints()})
        assert out.returncode == 0, out.stderr[-500:]
        assert f"REJECTED:{FIXED_LOCAL}" in out.stdout and "CALLED:False" in out.stdout

    @pytest.mark.parametrize("raw", ["not json", "[]", "{}", '{"vllm-a": {"base_url": "ftp://h/v1"}}', '{"A": {"base_url": "http://h/v1"}}'])
    def test_bad_request_value_is_the_fixed_error(self, raw):
        with pytest.raises(ValueError) as e:
            _local({run.PK_LOCAL_ENDPOINTS: raw})
        assert str(e.value) == FIXED_LOCAL
        assert SECRET_URL not in str(e.value)

    def test_customer_key_is_never_used(self):
        svc = _local({run.PK_LOCAL_ENDPOINTS: _endpoints()}, key="customer-key")
        assert "customer-key" not in repr(svc.call_args)

    def test_two_concurrent_sessions_do_not_mix(self):
        a = _endpoints(**{"vllm-a": {"base_url": "http://a.example.invalid/v1", "api_key": "key-a"}})
        b = _endpoints(**{"vllm-a": {"base_url": "http://b.example.invalid/v1", "api_key": "key-b"}})

        async def one(raw):
            await asyncio.sleep(0)
            svc = _local({run.PK_LOCAL_ENDPOINTS: raw})
            await asyncio.sleep(0)
            return svc.call_args.kwargs["base_url"]

        async def both():
            return await asyncio.gather(one(a), one(b), one(a), one(b))

        assert asyncio.run(both()) == ["http://a.example.invalid/v1", "http://b.example.invalid/v1"] * 2

    def test_parse_returns_the_loaded_endpoints_and_the_skipped_count(self):
        raw = _endpoints(**{"b": {"base_url": "https://x.example.invalid/v1"}, "C": {"base_url": "http://h/v1"}})
        loaded, skipped = run.parse_local_endpoints(raw)
        assert sorted(loaded) == ["b", "vllm-a"] and skipped == 1

    def test_the_longest_stored_text_is_parsed(self):
        prefix = '{"vllm-a":{"base_url":"http://h.example.invalid:8000/v1","api_key":"'
        suffix = '"}}'
        raw = prefix + "k" * (2020 - len(prefix) - len(suffix)) + suffix
        assert len(raw.encode()) == 2020
        assert list(run.parse_local_endpoints(raw)[0]) == ["vllm-a"]

    def test_parse_log_has_no_value(self, logs):
        run.parse_local_endpoints("{" + SECRET_URL + SENTINEL)
        run.parse_local_endpoints(_endpoints())
        assert SECRET_URL not in _text(logs) and SENTINEL not in _text(logs)


# --- the start log of the variables ------------------------------------------------------------------

ENV_NAMES = ("OPENAI_API_KEY", "DEEPGRAM_API_KEY", "CARTESIA_API_KEY", "XAI_API_KEY",
             "ELEVENLABS_API_KEY", "GOOGLE_API_KEY", "OPENROUTER_API_KEY", "LOCAL_LLM_ENDPOINTS")


class TestLogEnvState:
    def _records(self):
        records = []
        sink = logger.add(lambda m: records.append((m.record["level"].name, m.record["message"])), level="DEBUG")
        try:
            run.log_env_state()
        finally:
            logger.remove(sink)
        return records

    def test_one_line_per_variable_with_its_state_and_never_a_value(self, monkeypatch):
        monkeypatch.setenv("OPENAI_API_KEY", SENTINEL)
        monkeypatch.setenv("DEEPGRAM_API_KEY", "[[BIN_MANAGER__DEEPGRAM_API_KEY]]")
        monkeypatch.setenv("CARTESIA_API_KEY", "   ")
        monkeypatch.setenv("LOCAL_LLM_ENDPOINTS", _endpoints())
        records = self._records()
        by_name = {n: [r for r in records if n in r[1]] for n in ENV_NAMES}
        assert all(len(v) == 1 for v in by_name.values()), by_name
        assert "set" in by_name["OPENAI_API_KEY"][0][1] and "not set" not in by_name["OPENAI_API_KEY"][0][1]
        assert "unresolved placeholder" in by_name["DEEPGRAM_API_KEY"][0][1]
        assert by_name["DEEPGRAM_API_KEY"][0][0] == "WARNING"
        assert "not set" in by_name["CARTESIA_API_KEY"][0][1]
        assert "not set" in by_name["XAI_API_KEY"][0][1]
        text = _text(m for _, m in records)
        assert SENTINEL not in text and SECRET_URL not in text and "local-key-1" not in text

    def test_it_reads_the_environment_at_the_time_of_the_call(self, monkeypatch):
        assert "not set" in [m for _, m in self._records() if "OPENAI_API_KEY" in m][0]
        monkeypatch.setenv("OPENAI_API_KEY", SENTINEL)
        assert "not set" not in [m for _, m in self._records() if "OPENAI_API_KEY" in m][0]

    def test_main_calls_it_once_after_load_dotenv(self):
        """Fresh interpreter: the order of load_dotenv and log_env_state while main is imported."""
        here = Path(__file__).parent
        code = (
            "import conftest, run, dotenv\n"
            "from unittest.mock import patch\n"
            "calls = []\n"
            "with patch('dotenv.load_dotenv', side_effect=lambda **kw: calls.append(('dotenv', kw))), \\\n"
            "     patch('run.log_env_state', side_effect=lambda: calls.append(('log', None))):\n"
            "    import main\n"
            "print('CALLS:' + repr([c[0] for c in calls]) + repr([c[1] for c in calls if c[0] == 'dotenv']))\n"
        )
        out = subprocess.run([sys.executable, "-c", code], cwd=here, capture_output=True, text=True, timeout=120)
        assert out.returncode == 0, out.stderr[-500:]
        assert "CALLS:['dotenv', 'log'][{'override': True}]" in out.stdout


# --- the keys travel down the call chain -------------------------------------------------------------


def _platform():
    return {run.PK_OPENAI: SENTINEL + "-1", run.PK_DEEPGRAM: SENTINEL + "-2"}


@pytest.mark.asyncio
async def test_single_pipeline_hands_the_same_dict_to_stt_tts_and_llm():
    pk = _platform()
    transport = MagicMock()
    transport.cleanup = AsyncMock()
    task = MagicMock()
    task.cancel = AsyncMock()
    task.queue_frames = AsyncMock()
    llm = MagicMock()
    llm.cleanup = AsyncMock()

    with patch("run.create_llm_service", return_value=(llm, MagicMock())) as c_llm, \
         patch("run.create_tts_service") as c_tts, patch("run.create_stt_service") as c_stt, \
         patch("run.SileroVADAnalyzer"), patch("run.build_vad_params"), \
         patch("run.create_websocket_transport", return_value=transport), \
         patch("run.Pipeline"), patch("run.PipelineTask", return_value=task), \
         patch("run.tool_register"), patch("run.register_missing_tool_logging"), patch("run.normalize_tool_call_ids"), \
         patch("run.task_manager") as tm:
        tm.add = AsyncMock()
        tm.remove = AsyncMock()
        ctx = await run.init_pipeline(
            "pl-keys", "openai.gpt-5", "", [], "deepgram", "en-US", "cartesia", "en-US", "voice", [],
            platform_keys=pk,
        )

    assert c_stt.call_args.kwargs["platform_keys"] is pk
    assert c_tts.call_args.kwargs["platform_keys"] is pk
    assert c_llm.call_args.kwargs["platform_keys"] is pk
    assert SENTINEL not in repr(ctx)


@pytest.mark.asyncio
async def test_team_pipeline_hands_the_same_dict_to_every_member_service():
    pk = _platform()
    team = {
        "start_member_id": "m1",
        "members": [
            _member("m1", "openai.gpt-5", llm_type="openai.gpt-5", tts_type="cartesia", stt_type="deepgram"),
            _member("m2", "grok.grok-4", llm_type="grok.grok-4", tts_type="elevenlabs", stt_type="deepgram"),
        ],
    }
    ctx, mocks = await _run_team_init(team, tts_type="cartesia", stt_type="deepgram", platform_keys=pk)

    assert [c.kwargs["platform_keys"] for c in mocks.create_llm_service.call_args_list] == [pk, pk]
    assert [c.kwargs["platform_keys"] for c in mocks.create_tts_service.call_args_list] == [pk, pk]
    assert [c.kwargs["platform_keys"] for c in mocks.create_stt_service.call_args_list] == [pk, pk]
    assert SENTINEL not in repr(ctx)


@pytest.mark.asyncio
async def test_team_text_mode_builds_no_tts_or_stt_whatever_the_keys():
    team = {"start_member_id": "m1", "members": [_member("m1", "openai.gpt-5", llm_type="openai.gpt-5", tts_type="cartesia", stt_type="deepgram")]}
    _, mocks = await _run_team_init(team, platform_keys=_platform())
    assert mocks.create_tts_service.call_count == 0 and mocks.create_stt_service.call_count == 0
    assert mocks.create_llm_service.call_args.kwargs["platform_keys"] == _platform()


@pytest.mark.asyncio
async def test_init_pipeline_without_platform_keys_passes_none():
    team = {"start_member_id": "m1", "members": [_member("m1", "openai.gpt-5", llm_type="openai.gpt-5")]}
    _, mocks = await _run_team_init(team)
    assert mocks.create_llm_service.call_args.kwargs["platform_keys"] is None


@pytest.mark.asyncio
async def test_a_real_factory_call_through_the_team_path_uses_the_request_key(monkeypatch):
    """The request key reaches the SDK constructor of a member through the real create_llm_service
    (the environment value is set and is not used)."""
    monkeypatch.setenv("OPENAI_API_KEY", "env-value")
    real = run.create_llm_service
    with patch("run.OpenAILLMService") as openai_cls, patch("run.LLMContext"), patch("run.LLMContextAggregatorPair"):
        team = {"start_member_id": "m1", "members": [_member("m1", "openai.gpt-4o", "", llm_type="openai.gpt-4o")]}
        await _run_team_init(team, llm_side_effect=real, platform_keys={run.PK_OPENAI: "platform-value"})
    assert openai_cls.call_args.kwargs["api_key"] == "platform-value"


# --- the request model and the endpoint --------------------------------------------------------------


def test_request_model_accepts_the_field_and_keeps_it_out_of_repr():
    req = main.PipelineRequest(id="p", llm_type="openai.gpt-5", platform_keys={run.PK_OPENAI: SENTINEL})
    assert req.platform_keys == {run.PK_OPENAI: SENTINEL}
    assert SENTINEL not in repr(req) and SENTINEL not in str(req)


def test_request_without_the_field_is_valid():
    """A request from a Go service that does not send the field."""
    req = main.PipelineRequest(id="p", llm_type="openai.gpt-5")
    assert req.platform_keys is None


@pytest.fixture
def client():
    return TestClient(main.app, raise_server_exceptions=True)


def test_run_passes_the_keys_to_init_pipeline_and_logs_none_of_them(client, logs):
    init = AsyncMock(return_value={"type": "single"})
    with patch.object(main, "init_pipeline", init), patch.object(main, "execute_pipeline", AsyncMock()):
        r = client.post("/run", json={
            "id": "p1", "llm_type": "openai.gpt-5", "llm_key": "", "llm_messages": [{"role": "user", "content": "hi"}],
            "platform_keys": {run.PK_OPENAI: SENTINEL, run.PK_LOCAL_ENDPOINTS: _endpoints()},
        })
    assert r.status_code == 200
    assert init.call_args.kwargs["platform_keys"] == {run.PK_OPENAI: SENTINEL, run.PK_LOCAL_ENDPOINTS: _endpoints()}
    assert SENTINEL not in _text(logs) and SECRET_URL not in _text(logs) and SENTINEL not in r.text


def test_a_wrongly_typed_value_is_a_fixed_422_without_the_value(client, logs):
    r = client.post("/run", json={"id": "p2", "llm_type": "openai.gpt-5", "platform_keys": {run.PK_OPENAI: 12345678, "x": SENTINEL, "y": [SENTINEL]}})
    assert r.status_code == 422
    assert r.json() == {"detail": "invalid request"}
    assert SENTINEL not in r.text and "12345678" not in r.text
    assert SENTINEL not in _text(logs) and "12345678" not in _text(logs)


def test_platform_keys_that_is_not_a_map_is_a_fixed_422(client, logs):
    r = client.post("/run", json={"id": "p3", "llm_type": "openai.gpt-5", "platform_keys": SENTINEL})
    assert r.status_code == 422 and r.json() == {"detail": "invalid request"}
    assert SENTINEL not in r.text and SENTINEL not in _text(logs)


def test_a_validation_failure_of_init_is_a_400_without_the_keys(client, logs):
    """The real factory with a bad local value: the answer and the logs hold the fixed text only."""
    r = client.post("/run", json={
        "id": "p4", "llm_type": "local.vllm-a.m", "llm_messages": [{"role": "user", "content": "hi"}],
        "platform_keys": {run.PK_LOCAL_ENDPOINTS: "{" + SECRET_URL + SENTINEL},
    })
    assert r.status_code == 400
    assert r.json() == {"detail": FIXED_LOCAL}
    assert SENTINEL not in r.text and SECRET_URL not in r.text
    assert SENTINEL not in _text(logs) and SECRET_URL not in _text(logs)


def test_an_unsupported_service_error_holds_no_key(client, logs):
    r = client.post("/run", json={
        "id": "p5", "llm_type": "openai.gpt-5", "llm_messages": [{"role": "user", "content": "hi"}],
        "tts_type": "nosuchvendor", "platform_keys": {run.PK_OPENAI: SENTINEL, run.PK_CARTESIA: SENTINEL},
    })
    assert r.status_code == 400
    assert SENTINEL not in r.text and SENTINEL not in _text(logs)


def test_the_environment_is_untouched_by_a_request():
    before = dict(os.environ)
    _llm("openai.gpt-5", platform_keys={run.PK_OPENAI: SENTINEL})
    assert dict(os.environ) == before
