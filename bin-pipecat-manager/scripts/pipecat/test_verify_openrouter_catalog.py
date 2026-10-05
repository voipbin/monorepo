"""Unit tests for verify_openrouter_catalog.py (no network, fixtures only)."""
import pytest

import verify_openrouter_catalog as v

CATALOG_GO = '''
package ai

var catalog = []ModelEntry{
	{ID: "gemini.gemini-2.5-flash", Label: "Gemini 2.5 Flash", Route: RouteDirect},
	{
		ID:           "anthropic.claude-haiku-4.5",
		Route:        RouteOpenRouter,
		UpstreamSlug: "anthropic/claude-haiku-4.5",
	},
	{ID: "deepseek.deepseek-v3.2", Route: RouteOpenRouter, UpstreamSlug:     "deepseek/deepseek-v3.2"},
	// UpstreamSlug: "commented/out" is a full-line comment and must be ignored
	{ID: "x", Route: RouteOpenRouter, UpstreamSlug: "anthropic/claude-haiku-4.5"},
}
'''


def _ep(model_id, params, provider="P"):
    return {"model_id": model_id, "supported_parameters": params, "provider_name": provider}


def test_parse_slugs_tolerates_gofmt_alignment_and_dedupes():
    assert v.parse_catalog_slugs(CATALOG_GO) == [
        "anthropic/claude-haiku-4.5",
        "deepseek/deepseek-v3.2",
    ]


def test_parse_slugs_empty_when_none():
    assert v.parse_catalog_slugs("package ai\n") == []


def test_exact_match_not_substring():
    zdr = {"data": [_ep("deepseek/deepseek-v3.2-exp", ["tools"])]}
    problems = v.check_slugs(["deepseek/deepseek-v3.2"], zdr)
    assert len(problems) == 1 and "deepseek/deepseek-v3.2" in problems[0]


def test_ok_when_a_zdr_endpoint_supports_tools():
    zdr = {"data": [
        _ep("a/b", ["temperature"], "NoTools"),
        _ep("a/b", ["temperature", "tools", "tool_choice"], "WithTools"),
    ]}
    assert v.check_slugs(["a/b"], zdr) == []


def test_problem_when_zdr_endpoints_lack_tools():
    zdr = {"data": [_ep("a/b", ["temperature"])]}
    problems = v.check_slugs(["a/b"], zdr)
    assert len(problems) == 1 and "tools" in problems[0]


def test_problem_when_no_zdr_endpoint():
    problems = v.check_slugs(["a/b"], {"data": []})
    assert len(problems) == 1 and "no ZDR endpoint" in problems[0]


def test_missing_supported_parameters_is_treated_as_no_tools():
    assert len(v.check_slugs(["a/b"], {"data": [{"model_id": "a/b"}]})) == 1


def test_main_fails_when_catalog_has_no_slugs(tmp_path, capsys):
    f = tmp_path / "catalog.go"
    f.write_text("package ai\n")
    assert v.main(["--catalog", str(f)], fetch=lambda url: {"data": []}) == 1


def test_main_exit_codes_with_injected_fetch(tmp_path):
    f = tmp_path / "catalog.go"
    f.write_text(CATALOG_GO)
    good = {"data": [_ep(s, ["tools"]) for s in ("anthropic/claude-haiku-4.5", "deepseek/deepseek-v3.2")]}
    assert v.main(["--catalog", str(f)], fetch=lambda url: good) == 0
    bad = {"data": [_ep("anthropic/claude-haiku-4.5", ["tools"])]}
    assert v.main(["--catalog", str(f)], fetch=lambda url: bad) == 1


def test_main_missing_catalog_file_returns_error(tmp_path):
    assert v.main(["--catalog", str(tmp_path / "nope.go")], fetch=lambda url: {"data": []}) == 2
