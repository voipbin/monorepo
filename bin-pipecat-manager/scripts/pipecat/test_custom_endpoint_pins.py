"""The versions the custom endpoint guard was written and tested against are pinned (VOIP-1599, design 3.8).

custom_endpoint_net.py swaps the httpcore pool of httpx.AsyncHTTPTransport and reads private
attributes of httpx and httpcore, so a floating version could break the guard silently. This test
fails when requirements.txt, pyproject.toml, uv.lock or the installed packages drift apart.
"""

import os
import re
import tomllib
from importlib import metadata

import pytest

HERE = os.path.dirname(os.path.abspath(__file__))
PINS = {"httpx": "0.28.1", "httpcore": "1.0.9", "h11": "0.16.0", "anyio": "4.12.1", "openai": "2.15.0"}


def _requirement_pins():
    pins = {}
    with open(os.path.join(HERE, "requirements.txt")) as f:
        for line in f:
            m = re.match(r"^([A-Za-z0-9_.-]+)==([^\s#]+)\s*(#.*)?$", line.strip())
            if m:
                pins[m.group(1).lower()] = m.group(2)
    return pins


def test_requirements_txt_pins_exactly():
    pins = _requirement_pins()
    for name, version in PINS.items():
        assert pins.get(name) == version, name


def test_pyproject_pins_exactly():
    with open(os.path.join(HERE, "pyproject.toml"), "rb") as f:
        deps = tomllib.load(f)["project"]["dependencies"]
    found = {}
    for dep in deps:
        m = re.match(r"^([A-Za-z0-9_.-]+)==(\S+)$", dep)
        if m:
            found[m.group(1).lower()] = m.group(2)
    for name, version in PINS.items():
        assert found.get(name) == version, name


def test_uv_lock_resolves_the_same_versions():
    with open(os.path.join(HERE, "uv.lock"), "rb") as f:
        lock = tomllib.load(f)
    locked = {p["name"]: p["version"] for p in lock["package"]}
    for name, version in PINS.items():
        assert locked.get(name) == version, name
    project = next(p for p in lock["package"] if p["name"] == "pipecat-runner")
    spec = {d["name"]: d.get("specifier") for d in lock["package"][lock["package"].index(project)]["metadata"]["requires-dist"]}
    for name, version in PINS.items():
        assert spec.get(name) == f"=={version}", name


@pytest.mark.parametrize("name,version", sorted(PINS.items()))
def test_installed_version_equals_the_pin(name, version):
    assert metadata.version(name) == version
