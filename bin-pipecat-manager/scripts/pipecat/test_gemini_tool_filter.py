"""Control-flow tests for gemini_tool_filter.drop_gemini_invalid_tools.

These run under the mocked conftest.py (pipecat is a MagicMock tree), so the
Gemini adapter and GenerateContentConfig are replaced by injected fakes and
ToolsSchema by a small recording class. The real-library check lives in
scripts/pipecat_realtest/ (outside this directory, so this conftest never
shadows real pipecat there).
"""

import sys
import types
from dataclasses import dataclass, field

import pytest

pydantic = pytest.importorskip("pydantic")

from loguru import logger  # noqa: E402

from gemini_tool_filter import drop_gemini_invalid_tools  # noqa: E402


@dataclass
class _Schema:
    """Stand-in for pipecat FunctionSchema; the filter only reads .name."""

    name: str
    properties: dict = field(default_factory=dict)


class _FakeToolsSchema:
    def __init__(self, standard_tools):
        self.standard_tools = list(standard_tools)


class _StrictModel(pydantic.BaseModel):
    value: int


def _validation_error() -> Exception:
    """Return a real pydantic.ValidationError."""
    try:
        _StrictModel(value="not-an-int")
    except pydantic.ValidationError as e:
        return e
    raise AssertionError("expected a ValidationError")


class _FakeAdapter:
    """Records every tool set it converts and passes the names through."""

    calls: list

    def __init__(self):
        self.calls = []

    def to_provider_tools_format(self, tools_schema):
        names = [s.name for s in tools_schema.standard_tools]
        self.calls.append(names)
        return names


def _config_factory_rejecting(*bad_names):
    """GenerateContentConfig stand-in: raises ValidationError when any bad name is present."""

    def factory(tools):
        if any(n in bad_names for n in tools):
            raise _validation_error()
        return {"tools": tools}

    return factory


@pytest.fixture(autouse=True)
def _fake_tools_schema(monkeypatch):
    mod = types.ModuleType("pipecat.adapters.schemas.tools_schema")
    mod.ToolsSchema = _FakeToolsSchema
    monkeypatch.setitem(sys.modules, "pipecat.adapters.schemas.tools_schema", mod)


@pytest.fixture
def logs():
    records = []
    sink_id = logger.add(lambda m: records.append(m.record), level="DEBUG")
    yield records
    logger.remove(sink_id)


def _run(schemas, adapter, config_factory, pipeline_id="pl-1"):
    return drop_gemini_invalid_tools(
        schemas,
        pipeline_id,
        adapter_factory=lambda: adapter,
        config_factory=config_factory,
    )


def test_empty_list_returns_immediately(logs):
    adapter = _FakeAdapter()

    res = _run([], adapter, _config_factory_rejecting())

    assert res == []
    assert adapter.calls == []
    assert logs == []


def test_fast_path_returns_input_unchanged_after_one_validation(logs):
    adapter = _FakeAdapter()
    schemas = [_Schema("a"), _Schema("b")]

    res = _run(schemas, adapter, _config_factory_rejecting())

    assert res is schemas
    assert adapter.calls == [["a", "b"]]
    assert [r for r in logs if r["level"].name in ("WARNING", "INFO")] == []


def _messages(logs, level):
    return [r["message"] for r in logs if r["level"].name == level]


def test_per_tool_path_drops_only_rejected_tools(logs):
    adapter = _FakeAdapter()
    good1, bad, good2 = _Schema("good1"), _Schema("bad"), _Schema("good2")

    res = _run([good1, bad, good2], adapter, _config_factory_rejecting("bad"), pipeline_id="pl-42")

    assert res == [good1, good2]
    # whole set, then each tool alone, then the kept set once more
    assert adapter.calls == [["good1", "bad", "good2"], ["good1"], ["bad"], ["good2"], ["good1", "good2"]]

    warns = _messages(logs, "WARNING")
    assert len(warns) == 1
    assert warns[0].startswith("Dropped tool 'bad' rejected by the Gemini schema validator: 1 error(s), first: ")
    assert "value" in warns[0]
    assert "pipeline id=pl-42" in warns[0]

    infos = _messages(logs, "INFO")
    assert infos == ["Gemini tool validation dropped 1 of 3 tools. pipeline id=pl-42"]


def test_drop_warning_is_capped_and_never_logs_the_schema(logs):
    adapter = _FakeAdapter()
    long_name = "t" * 500
    bad = _Schema(long_name, properties={"secret_schema_marker": {"type": "weird"}})

    _run([_Schema("ok"), bad], adapter, _config_factory_rejecting(long_name), pipeline_id="pl-7")

    warns = _messages(logs, "WARNING")
    assert len(warns) == 1
    assert "secret_schema_marker" not in warns[0]
    assert warns[0].endswith("pipeline id=pl-7")
    assert len(warns[0]) <= 300 + len(". pipeline id=pl-7")


def test_fail_open_when_filtered_set_still_fails(logs):
    """Every tool passes alone but the set fails: return the input unchanged."""
    adapter = _FakeAdapter()
    schemas = [_Schema("a"), _Schema("b")]

    def config_factory(tools):
        if len(tools) > 1:
            raise _validation_error()
        return {"tools": tools}

    res = _run(schemas, adapter, config_factory, pipeline_id="pl-3")

    assert res is schemas
    warns = _messages(logs, "WARNING")
    assert len(warns) == 1
    assert "pipeline id=pl-3" in warns[0]
    assert _messages(logs, "INFO") == []


def test_fail_open_when_filtered_set_still_fails_after_a_drop(logs):
    adapter = _FakeAdapter()
    schemas = [_Schema("a"), _Schema("bad"), _Schema("b")]

    def config_factory(tools):
        if "bad" in tools or len(tools) > 1:
            raise _validation_error()
        return {"tools": tools}

    res = _run(schemas, adapter, config_factory, pipeline_id="pl-4")

    assert res is schemas
    assert _messages(logs, "INFO") == []
    assert all("pipeline id=pl-4" in m for m in _messages(logs, "WARNING"))


@pytest.mark.parametrize("fail_on_call", [1, 2, 3])
def test_fail_open_on_non_pydantic_exception(logs, fail_on_call):
    """A validator failure (not a ValidationError) at any step never removes tools.

    Call 1 is the fast path, call 2 the per-tool loop, call 3 the re-check.
    """
    adapter = _FakeAdapter()
    schemas = [_Schema("bad"), _Schema("ok")]
    count = {"n": 0}

    def config_factory(tools):
        count["n"] += 1
        if count["n"] == fail_on_call:
            raise TypeError("adapter api changed")
        if "bad" in tools:
            raise _validation_error()
        return {"tools": tools}

    res = _run(schemas, adapter, config_factory, pipeline_id="pl-5")

    assert res is schemas
    warns = _messages(logs, "WARNING")
    assert any("TypeError" in m and "pipeline id=pl-5" in m for m in warns)
    assert _messages(logs, "INFO") == []


def test_fail_open_on_adapter_factory_exception(logs):
    schemas = [_Schema("a")]

    def broken_factory():
        raise RuntimeError("boom")

    res = drop_gemini_invalid_tools(
        schemas, "pl-6", adapter_factory=broken_factory, config_factory=_config_factory_rejecting()
    )

    assert res is schemas
    assert any("RuntimeError" in m and "pipeline id=pl-6" in m for m in _messages(logs, "WARNING"))


def test_fail_open_on_import_error(logs, monkeypatch):
    """The real adapter import fails: warn once and return the input unchanged."""
    monkeypatch.setitem(sys.modules, "pipecat.adapters.services.gemini_adapter", None)
    schemas = [_Schema("a"), _Schema("b")]

    res = drop_gemini_invalid_tools(schemas, "pl-8", config_factory=_config_factory_rejecting("a"))

    assert res is schemas
    warns = _messages(logs, "WARNING")
    assert len(warns) == 1
    assert "pipeline id=pl-8" in warns[0]


def test_fail_open_on_config_import_error(logs, monkeypatch):
    monkeypatch.setitem(sys.modules, "google.genai.types", None)
    schemas = [_Schema("a")]

    res = drop_gemini_invalid_tools(schemas, "pl-9", adapter_factory=_FakeAdapter)

    assert res is schemas
    warns = _messages(logs, "WARNING")
    assert len(warns) == 1
    assert "pipeline id=pl-9" in warns[0]


def test_injected_factories_skip_real_imports(logs, monkeypatch):
    """With both factories injected, a broken real adapter/config import is never touched."""
    monkeypatch.setitem(sys.modules, "pipecat.adapters.services.gemini_adapter", None)
    monkeypatch.setitem(sys.modules, "google.genai.types", None)
    adapter = _FakeAdapter()
    good, bad = _Schema("good"), _Schema("bad")

    res = _run([good, bad], adapter, _config_factory_rejecting("bad"))

    assert res == [good]
