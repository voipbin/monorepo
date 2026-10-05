#!/usr/bin/env python3
"""Manual check: every OpenRouter slug in the Go model catalog has a ZDR endpoint with tools.

Reads `UpstreamSlug: "..."` entries from bin-ai-manager/models/ai/catalog.go by regex,
fetches the public ZDR endpoint list and exits non-zero when a slug has no ZDR endpoint
whose `supported_parameters` contains `tools`. Slugs are compared by exact `model_id`
equality (never substring: `...-v3.2` must not match `...-v3.2-exp`).

Not wired into CI (network dependency). Run before releases and when OpenRouter retires
a slug:

    python verify_openrouter_catalog.py [--catalog PATH]

No API key is needed or read: the ZDR endpoint list is public.
"""
import argparse
import json
import re
import sys
import urllib.request
from pathlib import Path

ZDR_URL = "https://openrouter.ai/api/v1/endpoints/zdr"
DEFAULT_CATALOG = Path(__file__).resolve().parents[3] / "bin-ai-manager" / "models" / "ai" / "catalog.go"

# Tolerant of gofmt alignment padding after the colon.
_SLUG_RE = re.compile(r'UpstreamSlug:\s*"([^"]+)"')


def parse_catalog_slugs(go_source: str) -> list[str]:
    """Return unique UpstreamSlug values in file order (full-line // comments ignored)."""
    seen: list[str] = []
    for line in go_source.splitlines():
        if line.lstrip().startswith("//"):
            continue
        for slug in _SLUG_RE.findall(line):
            if slug not in seen:
                seen.append(slug)
    return seen


def check_slugs(slugs: list[str], zdr: dict) -> list[str]:
    """Return a list of problem descriptions (empty means all slugs verified)."""
    by_model: dict[str, list[dict]] = {}
    for ep in zdr.get("data", []):
        by_model.setdefault(ep.get("model_id", ""), []).append(ep)

    problems = []
    for slug in slugs:
        endpoints = by_model.get(slug, [])  # exact equality only
        if not endpoints:
            problems.append(f"{slug}: no ZDR endpoint")
        elif not any("tools" in (ep.get("supported_parameters") or []) for ep in endpoints):
            problems.append(f"{slug}: ZDR endpoints exist but none supports 'tools'")
    return problems


def _fetch_json(url: str) -> dict:
    req = urllib.request.Request(url, headers={"User-Agent": "voipbin-verify-openrouter-catalog"})
    with urllib.request.urlopen(req, timeout=30) as resp:  # noqa: S310 (fixed https URL)
        return json.load(resp)


def main(argv=None, fetch=_fetch_json) -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--catalog", default=str(DEFAULT_CATALOG), help="path to catalog.go")
    args = ap.parse_args(argv)

    try:
        source = Path(args.catalog).read_text()
    except OSError as e:
        print(f"cannot read catalog: {e}", file=sys.stderr)
        return 2

    slugs = parse_catalog_slugs(source)
    if not slugs:
        print(f"no UpstreamSlug entries found in {args.catalog}", file=sys.stderr)
        return 1

    problems = check_slugs(slugs, fetch(ZDR_URL))
    for slug in slugs:
        status = "FAIL" if any(p.startswith(slug + ":") for p in problems) else "ok"
        print(f"{status}  {slug}")
    for p in problems:
        print(p, file=sys.stderr)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
