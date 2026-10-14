"""Local (operator-registered, self-hosted) LLM endpoints: runner branch and hygiene."""
import json
import os
from unittest.mock import MagicMock, patch

import pytest
from loguru import logger

import run

SECRET_URL = "http://llm-secret-host.internal:8000/v1"
SECRET_KEY = "sk-local-secret-123"
FIXED = "local engine model is not available"


@pytest.fixture(autouse=True)
def _restore_endpoints():
    saved = run._local_endpoints
    yield
    run._local_endpoints = saved


def _load(obj):
    return run.load_local_endpoints(json.dumps(obj))


class TestLoadLocalEndpoints:
    def test_valid(self):
        assert _load({"vllm-a": {"base_url": SECRET_URL, "api_key": ""}, "b": {"base_url": "https://x.example/v1"}}) == 2

    @pytest.mark.parametrize("raw", ["", "  ", "{}", "[]", "not json", "[1,2]", "null", '"s"'])
    def test_empty_or_malformed_loads_nothing(self, raw):
        assert run.load_local_endpoints(raw) == 0

    @pytest.mark.parametrize("base_url", [
        "ftp://h/v1", "h/v1", "http://", "http://u:p@h/v1", "http://h/v1?x=1", "http://h/v1#f", "http://h:99999/v1", 5, None,
    ])
    def test_rule_violating_base_url_skipped(self, base_url):
        assert _load({"a": {"base_url": base_url}}) == 0

    @pytest.mark.parametrize("name", ["A", "a.b", "", "-a", "a" * 33])
    def test_bad_endpoint_name_skipped(self, name):
        assert _load({name: {"base_url": SECRET_URL}}) == 0

    def test_non_string_key_skipped(self):
        assert _load({"a": {"base_url": SECRET_URL, "api_key": 5}}) == 0

    def test_reads_environment(self):
        with patch.dict(os.environ, {"LOCAL_LLM_ENDPOINTS": json.dumps({"a": {"base_url": SECRET_URL}})}):
            assert run.load_local_endpoints() == 1

    def test_load_log_has_no_value(self):
        sink = []
        hid = logger.add(sink.append, format="{message}")
        try:
            run.load_local_endpoints("{" + SECRET_URL + SECRET_KEY)
            _load({"a": {"base_url": SECRET_URL, "api_key": SECRET_KEY}})
        finally:
            logger.remove(hid)
        text = "".join(str(m) for m in sink)
        assert SECRET_URL not in text and SECRET_KEY not in text
        assert "loaded: 1" in text


class TestLocalBranch:
    def _create(self, type_, key="customer-key"):
        with patch("run.LLMContextAggregatorPair"), patch("run.LLMContext"), patch("run.OpenAILLMService") as svc:
            res = run.create_llm_service(type=type_, key=key, messages=[{"role": "user", "content": "hi"}], tools=[])
        return svc, res

    def test_builds_openai_service_with_base_url(self):
        _load({"vllm-a": {"base_url": SECRET_URL, "api_key": SECRET_KEY}})
        svc, (llm, _) = self._create("local.vllm-a.qwen3-8b")
        assert svc.call_args.kwargs == {"api_key": SECRET_KEY, "model": "qwen3-8b", "base_url": SECRET_URL}
        assert llm is svc.return_value

    def test_model_id_with_dots_slash_colon(self):
        _load({"a": {"base_url": SECRET_URL}})
        svc, _ = self._create("local.a.Org/qwen2.5-7b:v1.2")
        assert svc.call_args.kwargs["model"] == "Org/qwen2.5-7b:v1.2"

    def test_empty_key_becomes_placeholder_never_env_key(self):
        _load({"a": {"base_url": SECRET_URL, "api_key": ""}})
        with patch.dict(os.environ, {"OPENAI_API_KEY": "env-openai-key"}):
            svc, _ = self._create("local.a.m", key="customer-key")
        key = svc.call_args.kwargs["api_key"]
        assert key and key not in ("env-openai-key", "customer-key")

    def test_customer_key_is_never_used(self):
        _load({"a": {"base_url": SECRET_URL, "api_key": SECRET_KEY}})
        svc, _ = self._create("local.a.m", key="customer-key")
        assert "customer-key" not in repr(svc.call_args)

    @pytest.mark.parametrize("type_", ["local.missing.m", "local.a", "local.a.", "local:a", "local.", "LOCAL.missing.m"])
    def test_unavailable_raises_fixed_message(self, type_):
        _load({"a": {"base_url": SECRET_URL, "api_key": SECRET_KEY}})
        with pytest.raises(ValueError) as e:
            self._create(type_)
        assert str(e.value) == FIXED
        assert SECRET_URL not in str(e.value) and SECRET_KEY not in str(e.value)

    def test_no_endpoints_configured(self):
        run.load_local_endpoints("")
        with pytest.raises(ValueError, match=FIXED):
            self._create("local.a.m")


class TestMemberGuard:
    def test_none_llm_type_rejects_local(self):
        for em in ("local.a.m", "Local.a.m", "local:a.m"):
            with pytest.raises(ValueError, match="engine model is not available"):
                run._member_llm_type({"engine_model": em})

    def test_go_resolved_local_is_used(self):
        assert run._member_llm_type({"engine_model": "local.a.m", "llm_type": "local.a.m"}) == "local.a.m"

    def test_team_member_path_does_not_forward_engine_key_to_local(self):
        _load({"a": {"base_url": SECRET_URL, "api_key": ""}})
        with patch("run.LLMContextAggregatorPair"), patch("run.LLMContext"), patch("run.OpenAILLMService") as svc:
            run.create_llm_service(run._member_llm_type({"engine_model": "local.a.m", "llm_type": "local.a.m"}), "", [], [])
        assert svc.call_args.kwargs["api_key"]


class TestRunEndpoint:
    """The /run handler returns str(e) as detail and logs the error: neither may carry secrets."""

    def _post(self, llm_type):
        from fastapi.testclient import TestClient
        import main

        sink = []
        hid = logger.add(sink.append, format="{message}\n{exception}", diagnose=False)
        try:
            resp = TestClient(main.app).post("/run", json={"id": "p1", "llm_type": llm_type, "llm_key": "customer-key"})
        finally:
            logger.remove(hid)
        return resp, "".join(str(m) for m in sink)

    @pytest.mark.parametrize("raw", ["{" + SECRET_URL, json.dumps({"a": {"base_url": "http://u:" + SECRET_KEY + "@h/v1"}})])
    def test_broken_or_invalid_config_leaks_nothing(self, raw):
        run.load_local_endpoints(raw)
        resp, logs = self._post("local.a.m")
        assert resp.status_code == 400
        assert resp.json()["detail"] == FIXED
        for secret in (SECRET_URL, SECRET_KEY, "customer-key"):
            assert secret not in resp.text and secret not in logs

    def test_missing_endpoint(self):
        _load({"a": {"base_url": SECRET_URL, "api_key": SECRET_KEY}})
        resp, logs = self._post("local.other.m")
        assert resp.status_code == 400 and resp.json()["detail"] == FIXED
        assert SECRET_URL not in logs and SECRET_KEY not in logs


def test_main_log_sink_does_not_print_locals():
    """loguru diagnose=True prints local variables (endpoint URL, key) in tracebacks."""
    import ast

    tree = ast.parse(open(os.path.join(os.path.dirname(__file__), "main.py")).read())
    adds = [
        n for n in ast.walk(tree)
        if isinstance(n, ast.Call) and isinstance(n.func, ast.Attribute) and n.func.attr == "add"
        and isinstance(n.func.value, ast.Name) and n.func.value.id == "logger"
    ]
    assert adds, "main.py must configure the logger sink"
    for call in adds:
        kw = {k.arg: k.value for k in call.keywords}
        assert isinstance(kw.get("diagnose"), ast.Constant) and kw["diagnose"].value is False
