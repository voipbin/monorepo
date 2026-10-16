"""Unit tests for custom_endpoint_net.py (VOIP-1599 Layer 2, design 3.5, 3.8, 7 items 5 and 8a).

custom_endpoint_net.py does not import pipecat, so it is imported directly (conftest.py only
mocks pipecat). Tests that need the real httpx/httpcore stack use a scripted fake network
stream; tests with a real TLS server are in scripts/pipecat_realtest/test_custom_endpoint_real.py.
"""

import asyncio
import ipaddress
import json
import os
import ssl
import threading
import time
from concurrent.futures import ThreadPoolExecutor

import httpcore
import httpx
import openai
import pytest
from httpcore import AsyncNetworkBackend, AsyncNetworkStream

import custom_endpoint_net as net

CUSTOMER = "11111111-2222-3333-4444-555555555555"
OTHER_CUSTOMER = "99999999-2222-3333-4444-555555555555"
HOST = "llm.example.test"
URL = f"https://{HOST}/v1/x"
PUBLIC_A = "8.8.8.8"
PUBLIC_B = "1.1.1.1"


# --- shared fixture of the block table (parity with the Go test) -------------------------------


def _fixture_path():
    env = os.environ.get("ENDPOINT_BLOCKLIST_FIXTURE")
    if env:
        return env
    here = os.path.dirname(os.path.abspath(__file__))
    for _ in range(8):
        cand = os.path.join(here, "bin-ai-manager", "models", "ai", "testdata", "endpoint_blocklist.json")
        if os.path.exists(cand):
            return cand
        here = os.path.dirname(here)
    return "bin-ai-manager/models/ai/testdata/endpoint_blocklist.json"  # not found: the test fails


def _load_fixture():
    with open(_fixture_path()) as f:
        return json.load(f)


def _nets(values):
    return {ipaddress.ip_network(v) for v in values}


def test_block_tables_equal_shared_fixture():
    fx = _load_fixture()
    assert _nets(net.V4_BLOCKS) == _nets(fx["v4"])
    assert _nets(net.V6_BLOCKS) == _nets(fx["v6"])
    assert _nets(net.NAT64_BLOCKS) == _nets(fx["nat64"])
    # the tuples are what is_blocked uses
    assert _nets(net.V4_BLOCKS) == set(net._V4_NETS)
    assert _nets(net.V6_BLOCKS) == set(net._V6_NETS)


def test_block_corpus_equals_shared_fixture_cases():
    cases = _load_fixture()["cases"]
    assert len(cases) > 50
    wrong = [c["addr"] for c in cases if net.is_blocked(c["addr"]) != c["blocked"]]
    assert wrong == []


def test_every_table_network_is_enforced():
    """First and last address of every network is refused, unless a different rule decides it."""
    for n in net.V4_BLOCKS:
        network = ipaddress.ip_network(n)
        assert net.is_blocked(str(network[0])), n
        assert net.is_blocked(str(network[-1])), n
    for n in net.V6_BLOCKS:
        network = ipaddress.ip_network(n)
        if network.prefixlen == 96 and network.network_address == ipaddress.ip_address("::ffff:0:0"):
            continue  # IPv4-mapped: decided by the embedded IPv4 (covered below)
        assert net.is_blocked(str(network[0])), n
        assert net.is_blocked(str(network[-1])), n


@pytest.mark.parametrize(
    "addr",
    ["not-an-ip", "", " 8.8.8.8", "8.8.8.8 ", "8.8.8.8\n", "010.0.0.1", "1.2.3", "0x7f.1", "127.1", "2130706433", "[::1]", None, 5, b"8.8.8.8"],
)
def test_unparseable_is_blocked(addr):
    assert net.is_blocked(addr)


@pytest.mark.parametrize(
    "addr,blocked",
    [
        ("fe80::1%eth0", True),  # zone dropped, link local
        ("fe80::1%2", True),
        ("2606:4700:4700::1111%eth0", False),  # zone dropped, public
        ("2606:4700:4700::1111%", True),  # empty zone does not parse in Go either
        ("8.8.8.8%eth0", True),  # zone on IPv4 does not parse in Go either
        ("::ffff:10.0.0.1", True),
        ("::ffff:169.254.169.254", True),
        ("::ffff:8.8.8.8", False),  # mapped public address is allowed
        ("::ffff:0808:0808", False),
        ("64:ff9b::a9fe:a9fe", True),
        ("64:ff9b::7f00:1", True),
        ("64:ff9b::808:808", False),
        ("64:ff9b:1::1", True),  # local use NAT64 prefix is in the v6 table
        ("2002:a9fe:a9fe::", True),
        ("2002:7f00:1::", True),
        ("2002:808:808::", False),
        ("::ffff:0:808:808", True),  # SIIT
        ("::808:808", True),  # IPv4 compatible
        ("100.64.0.1", True),
        ("100.63.255.255", False),
        ("172.32.0.1", False),
        ("2001:4860:4860::8888", False),
    ],
)
def test_embedded_ipv4_and_zone_rules(addr, blocked):
    assert net.is_blocked(addr) is blocked


# --- scripted network ----------------------------------------------------------------------------


class FakeStream(AsyncNetworkStream):
    """items: bytes (data), number (sleep seconds) or an exception instance (raised by read)."""

    def __init__(self, items):
        self._items = iter(items)
        self._buf = b""
        self.writes = []
        self.read_sizes = []
        self.closed = False

    async def read(self, max_bytes, timeout=None):
        self.read_sizes.append(max_bytes)
        while not self._buf:
            try:
                item = next(self._items)
            except StopIteration:
                return b""
            if isinstance(item, BaseException):
                raise item
            if isinstance(item, (int, float)):
                await asyncio.sleep(item)
                continue
            self._buf = item
        out, self._buf = self._buf[:max_bytes], self._buf[max_bytes:]
        return out

    async def write(self, buffer, timeout=None):
        self.writes.append(bytes(buffer))

    async def aclose(self):
        self.closed = True

    async def start_tls(self, ssl_context, server_hostname=None, timeout=None):
        self.server_hostname = server_hostname
        return self

    def get_extra_info(self, info):
        return None


class FakeInner(AsyncNetworkBackend):
    def __init__(self, make=None, behaviors=None):
        self.make = make
        self.behaviors = list(behaviors or [])  # per dial: None (ok) or "fail" or ("sleep", secs)
        self.dialed = []
        self.timeouts = []
        self.streams = []

    async def connect_tcp(self, host, port, timeout=None, local_address=None, socket_options=None):
        self.dialed.append(host)
        self.timeouts.append(timeout)
        behavior = self.behaviors.pop(0) if self.behaviors else None
        if behavior == "fail":
            raise httpcore.ConnectError("refused")
        if isinstance(behavior, tuple) and behavior[0] == "sleep":
            await asyncio.sleep(behavior[1])
            raise httpcore.ConnectError("refused after sleep")
        if behavior == "hang":
            await asyncio.sleep(60)
        stream = self.make(host)
        self.streams.append(stream)
        return stream

    async def connect_unix_socket(self, path, timeout=None, socket_options=None):
        raise AssertionError("dialed unix")

    async def sleep(self, seconds):
        await asyncio.sleep(seconds)


def fixed(*answers):
    """Resolver returning the answers in turn (the last one repeats) and counting calls."""

    class R:
        calls = 0

        async def __call__(self, host, port):
            a = answers[min(R.calls, len(answers) - 1)]
            R.calls += 1
            self.last = (host, port)
            return list(a)

    return R()


OK = b"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello"


def make_client(items=None, resolver=None, inner=None, customer=CUSTOMER):
    inner = inner or FakeInner(lambda host: FakeStream(items if items is not None else [OK]))
    backend = net.ValidatingBackend(customer, inner=inner, resolver=resolver or fixed([PUBLIC_A]))
    return net.build_http_client(customer, backend), inner


async def guarded(coro, secs=8):
    return await asyncio.wait_for(coro, secs)


# --- ValidatingBackend -----------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_dns_rebinding_refused_at_connect():
    inner = FakeInner(lambda h: FakeStream([OK, OK]))
    res = fixed([PUBLIC_A], ["10.0.0.5"])
    b = net.ValidatingBackend(CUSTOMER, inner=inner, resolver=res)
    s = await b.connect_tcp(HOST, 443, timeout=5)
    assert isinstance(s, net.WireGuardStream)
    with pytest.raises(net.AddressBlocked):
        await b.connect_tcp(HOST, 443, timeout=5)
    assert inner.dialed == [PUBLIC_A]  # the private address was never dialed
    assert res.calls == 2  # re-resolved and re-validated on every connect


@pytest.mark.asyncio
@pytest.mark.parametrize("answer", [[PUBLIC_A, "192.168.1.9"], ["192.168.1.9", PUBLIC_A], [PUBLIC_A, PUBLIC_B, "::1"], [PUBLIC_A, "169.254.169.254"]])
async def test_mixed_answer_is_refused_whatever_the_position(answer):
    inner = FakeInner(lambda h: FakeStream([]))
    b = net.ValidatingBackend(CUSTOMER, inner=inner, resolver=fixed(answer))
    with pytest.raises(net.AddressBlocked):
        await b.connect_tcp(HOST, 443, timeout=5)
    assert inner.dialed == []


@pytest.mark.asyncio
async def test_every_address_is_validated_not_only_the_first():
    inner = FakeInner(lambda h: FakeStream([]))
    b = net.ValidatingBackend(CUSTOMER, inner=inner, resolver=fixed([PUBLIC_A, PUBLIC_B, "127.0.0.1"]))
    with pytest.raises(net.AddressBlocked):
        await b.connect_tcp(HOST, 443, timeout=5)


@pytest.mark.asyncio
async def test_empty_answer_and_resolver_errors_are_connect_errors():
    inner = FakeInner(lambda h: FakeStream([]))

    async def boom(host, port):
        raise OSError("secret-dns-detail")

    for res in (fixed([]), boom):
        b = net.ValidatingBackend(CUSTOMER, inner=inner, resolver=res)
        with pytest.raises(httpcore.ConnectError) as ei:
            await b.connect_tcp(HOST, 443, timeout=5)
        assert "secret" not in str(ei.value)
    assert inner.dialed == []


@pytest.mark.asyncio
async def test_connects_by_validated_ip_literal_and_wraps_the_stream():
    inner = FakeInner(lambda h: FakeStream([OK]))
    b = net.ValidatingBackend(CUSTOMER, inner=inner, resolver=fixed([PUBLIC_A]))
    s = await b.connect_tcp(HOST, 443, timeout=5)
    assert inner.dialed == [PUBLIC_A]
    assert isinstance(s, net.WireGuardStream)
    t = await s.start_tls(ssl.create_default_context(), server_hostname=HOST)
    assert isinstance(t, net.WireGuardStream)  # the result of start_tls is wrapped too
    assert inner.streams[0].server_hostname == HOST  # SNI / certificate name stay the original name


@pytest.mark.asyncio
async def test_unix_socket_is_refused():
    b = net.ValidatingBackend(CUSTOMER, inner=FakeInner(lambda h: FakeStream([])), resolver=fixed([PUBLIC_A]))
    with pytest.raises(net.AddressBlocked):
        await b.connect_unix_socket("/var/run/docker.sock")


@pytest.mark.asyncio
async def test_connect_deadline_is_shared_by_all_addresses(monkeypatch):
    monkeypatch.setattr(net, "CONNECT_TIMEOUT", 0.6)
    # A fails after 0.3 s, B hangs: B only gets what is left, C is never dialed
    inner = FakeInner(lambda h: FakeStream([]), behaviors=[("sleep", 0.3), "hang", None])
    b = net.ValidatingBackend(CUSTOMER, inner=inner, resolver=fixed([PUBLIC_A, PUBLIC_B, "9.9.9.9"]))
    t0 = time.monotonic()
    with pytest.raises(httpcore.ConnectTimeout):
        await guarded(b.connect_tcp(HOST, 443, timeout=5))
    elapsed = time.monotonic() - t0
    assert elapsed < 0.9  # not 2 x 0.6 or 3 x 0.6
    assert inner.dialed == [PUBLIC_A, PUBLIC_B]
    assert inner.timeouts[1] < 0.4  # the second address got the remainder, not a fresh budget


@pytest.mark.asyncio
async def test_falls_back_to_next_address_within_the_deadline():
    inner = FakeInner(lambda h: FakeStream([OK]), behaviors=["fail", None])
    b = net.ValidatingBackend(CUSTOMER, inner=inner, resolver=fixed([PUBLIC_A, PUBLIC_B]))
    await b.connect_tcp(HOST, 443, timeout=5)
    assert inner.dialed == [PUBLIC_A, PUBLIC_B]


@pytest.mark.asyncio
async def test_connect_timeout_is_capped_at_the_constant():
    inner = FakeInner(lambda h: FakeStream([OK]))
    b = net.ValidatingBackend(CUSTOMER, inner=inner, resolver=fixed([PUBLIC_A]))
    await b.connect_tcp(HOST, 443, timeout=100)
    assert inner.timeouts[0] <= net.CONNECT_TIMEOUT
    await b.connect_tcp(HOST, 443, timeout=None)
    assert inner.timeouts[1] <= net.CONNECT_TIMEOUT


# --- resolve_host: absolute names, literals, executor and limits -------------------------------


@pytest.fixture
def lookup_state():
    """Replaces the executor/lookup seam and fails if a lookup counter leaks."""
    events = []
    seen = []
    orig = (net._getaddrinfo, net._EXECUTOR)

    def fake(name, port, type=0):
        seen.append(name)
        ev = threading.Event()
        events.append(ev)
        if name.startswith("slow"):
            assert ev.wait(10)
        return [(2, 1, 6, "", (PUBLIC_A, port))]

    pool = ThreadPoolExecutor(max_workers=net.LOOKUP_WORKERS)
    net._getaddrinfo = fake
    net._EXECUTOR = pool
    yield type("S", (), {"events": events, "seen": seen})
    for ev in events:
        ev.set()
    deadline = time.monotonic() + 5
    while time.monotonic() < deadline and (net._LOOKUPS or net._LOOKUP_SEM._value != net.LOOKUP_WORKERS):
        time.sleep(0.01)
    leaked = (dict(net._LOOKUPS), net._LOOKUP_SEM._value)
    pool.shutdown(wait=True)
    net._getaddrinfo, net._EXECUTOR = orig
    assert leaked == ({}, net.LOOKUP_WORKERS)


@pytest.mark.asyncio
async def test_resolve_uses_absolute_name_and_literals_are_never_looked_up(lookup_state):
    assert await net.resolve_host("example.com", 443, CUSTOMER) == [PUBLIC_A]
    assert await net.resolve_host("127.1", 443, CUSTOMER) == [PUBLIC_A]  # not a literal: looked up
    assert await net.resolve_host("example.com.", 443, CUSTOMER) == [PUBLIC_A]  # no second dot
    assert lookup_state.seen == ["example.com.", "127.1.", "example.com."]
    assert await net.resolve_host("127.0.0.1", 443, CUSTOMER) == ["127.0.0.1"]
    assert await net.resolve_host("::1", 443, CUSTOMER) == ["::1"]
    assert lookup_state.seen == ["example.com.", "127.1.", "example.com."]  # literals: no lookup


@pytest.mark.asyncio
async def test_resolve_failure_is_a_fixed_connect_error(lookup_state):
    def failing(name, port, type=0):
        raise OSError("secret-resolver-text")

    net._getaddrinfo = failing
    with pytest.raises(httpcore.ConnectError) as ei:
        await net.resolve_host("example.com", 443, CUSTOMER)
    assert "secret" not in str(ei.value) and ei.value.__cause__ is None and ei.value.__suppress_context__


@pytest.mark.asyncio
async def test_per_customer_lookup_limit_fails_closed_without_queueing(lookup_state):
    slow = [asyncio.create_task(net.resolve_host(f"slow{i}.test", 443, CUSTOMER)) for i in range(2)]
    await asyncio.sleep(0.1)
    t0 = time.monotonic()
    with pytest.raises(httpcore.ConnectError):
        await net.resolve_host("slow-third.test", 443, CUSTOMER)  # third lookup of one customer
    assert time.monotonic() - t0 < 0.3
    assert await net.resolve_host("fast.test", 443, OTHER_CUSTOMER) == [PUBLIC_A]  # other customers unaffected
    for ev in lookup_state.events:
        ev.set()
    assert [await t for t in slow] == [[PUBLIC_A], [PUBLIC_A]]


@pytest.mark.asyncio
async def test_semaphore_is_not_blocking(lookup_state):
    customers = [f"00000000-0000-0000-0000-{i:012d}" for i in range(9)]
    slow = [asyncio.create_task(net.resolve_host(f"slow{i}.test", 443, customers[i])) for i in range(8)]
    await asyncio.sleep(0.2)
    assert net._LOOKUP_SEM._value == 0
    t0 = time.monotonic()
    with pytest.raises(httpcore.ConnectError):
        await guarded(net.resolve_host("fast.test", 443, customers[8]), 3)
    assert time.monotonic() - t0 < 0.5  # failed at once instead of waiting for a worker
    for ev in lookup_state.events:
        ev.set()
    await asyncio.gather(*slow)


@pytest.mark.asyncio
async def test_timed_out_lookup_keeps_both_until_the_thread_ends(lookup_state, monkeypatch):
    monkeypatch.setattr(net, "RESOLVE_TIMEOUT", 0.1)
    with pytest.raises(httpcore.ConnectError):
        await net.resolve_host("slow.test", 443, CUSTOMER)
    # the awaiting coroutine gave up, the thread is still busy: neither is given back
    assert net._LOOKUPS == {CUSTOMER: 1}
    assert net._LOOKUP_SEM._value == net.LOOKUP_WORKERS - 1
    lookup_state.events[0].set()
    deadline = time.monotonic() + 3
    while time.monotonic() < deadline and net._LOOKUPS:
        await asyncio.sleep(0.01)
    assert net._LOOKUPS == {}  # an entry at zero is deleted
    assert net._LOOKUP_SEM._value == net.LOOKUP_WORKERS


@pytest.mark.asyncio
async def test_lookup_cancelled_before_a_thread_started_gives_both_back(lookup_state, monkeypatch):
    monkeypatch.setattr(net, "RESOLVE_TIMEOUT", 0.1)
    one = ThreadPoolExecutor(max_workers=1)
    gate = threading.Event()
    one.submit(gate.wait, 10)  # the only worker is busy: the lookup stays queued
    net._EXECUTOR = one
    try:
        with pytest.raises(httpcore.ConnectError):
            await net.resolve_host("example.com", 443, CUSTOMER)
        # no thread ever ran it, so a finally inside the worker would never have run
        assert net._LOOKUPS == {}
        assert net._LOOKUP_SEM._value == net.LOOKUP_WORKERS
    finally:
        gate.set()
        one.shutdown(wait=True)


@pytest.mark.asyncio
async def test_submit_failure_gives_both_back(lookup_state):
    class Broken:
        def submit(self, *a, **k):
            raise RuntimeError("cannot schedule new futures after shutdown")

    net._EXECUTOR = Broken()
    with pytest.raises(RuntimeError):
        await net.resolve_host("example.com", 443, CUSTOMER)
    assert net._LOOKUPS == {}
    assert net._LOOKUP_SEM._value == net.LOOKUP_WORKERS


@pytest.mark.asyncio
async def test_unrelated_lookup_is_not_starved_by_a_slow_name(lookup_state):
    slow = [asyncio.create_task(net.resolve_host(f"slow{i}.test", 443, CUSTOMER)) for i in range(2)]
    await asyncio.sleep(0.1)
    t0 = time.monotonic()
    assert await guarded(net.resolve_host("localhost-like.test", 443, OTHER_CUSTOMER), 3) == [PUBLIC_A]
    assert time.monotonic() - t0 < 0.5
    for ev in lookup_state.events:
        ev.set()
    await asyncio.gather(*slow)


def test_executor_has_a_dedicated_pool_of_eight():
    assert net.LOOKUP_WORKERS == 8 and net.LOOKUPS_PER_CUSTOMER == 2
    assert net._EXECUTOR._max_workers == 8
    assert net._EXECUTOR is not getattr(asyncio.get_event_loop_policy(), "_default_executor", None)
    assert isinstance(net._LOOKUP_SEM, type(threading.Semaphore()))


# --- wire guard ----------------------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_wire_guard_caps_every_read_at_4096():
    inner = FakeStream([b"x" * 100000])
    s = net.WireGuardStream(inner)
    data = await s.read(65536)
    assert len(data) == 4096 and inner.read_sizes == [4096]
    await s.read(10)
    assert inner.read_sizes[-1] == 10  # a smaller request is kept


@pytest.mark.asyncio
async def test_wire_guard_yields_to_the_event_loop_after_each_read():
    inner = FakeStream([b"x" * 100000])
    s = net.WireGuardStream(inner)
    ticks = 0

    async def ticker():
        nonlocal ticks
        while True:
            ticks += 1
            await asyncio.sleep(0)

    t = asyncio.create_task(ticker())
    await asyncio.sleep(0)
    before = ticks
    for _ in range(50):
        await s.read(4096)  # FakeStream.read never suspends by itself
    t.cancel()
    assert ticks - before >= 45


@pytest.mark.asyncio
async def test_wire_guard_limit_counts_all_bytes_and_fails_the_read(monkeypatch):
    monkeypatch.setattr(net, "WIRE_BYTES_MAX", 10000)
    s = net.WireGuardStream(FakeStream([b"x" * 20000]))
    await s.read(4096)
    await s.read(4096)
    with pytest.raises(net.WireLimitExceeded):
        await s.read(4096)  # 12288 > 10000
    assert isinstance(net.WireLimitExceeded("x"), httpcore.ReadError)


def test_wire_cap_is_four_mebibytes_and_chunk_cap_4096():
    assert net.WIRE_BYTES_MAX == 4 * 1024 * 1024 and net.READ_CHUNK_MAX == 4096 and net.TOTAL_DEADLINE_SECS == 120.0


@pytest.mark.asyncio
async def test_wire_counter_is_reset_by_write(monkeypatch):
    monkeypatch.setattr(net, "WIRE_BYTES_MAX", 10000)
    s = net.WireGuardStream(FakeStream([b"x" * 8000, b"y" * 8000, b"z" * 8000]))
    for _ in range(3):
        await s.write(b"GET / HTTP/1.1\r\n\r\n")  # start of a request on a pooled connection
        got = 0
        while got < 8000:
            got += len(await s.read(4096))
    # without the reset the second round would have crossed 10000


@pytest.mark.asyncio
async def test_wire_guard_delegates():
    class Inner(FakeStream):
        def get_extra_info(self, info):
            return {"is_readable": True, "ssl_object": "tls"}.get(info)

    s = net.WireGuardStream(Inner([]))
    assert s.get_extra_info("is_readable") is True and s.get_extra_info("ssl_object") == "tls"
    await s.aclose()
    assert s._inner.closed


# --- transport guard (identity, deadlines, error mapping) over the real httpx/httpcore stack ---


@pytest.mark.asyncio
async def test_request_asks_for_identity_encoding():
    http, inner = make_client()
    r = await guarded(http.get(URL, headers={"Accept-Encoding": "gzip, deflate, br"}))
    assert r.status_code == 200 and r.content == b"hello"
    assert b"accept-encoding: identity" in inner.streams[0].writes[0].lower()
    assert b"gzip" not in inner.streams[0].writes[0].lower()


@pytest.mark.asyncio
@pytest.mark.parametrize("value", ["gzip", "br", "deflate", "identity, gzip", "GZIP", "zstd"])
async def test_other_content_encoding_is_refused(value):
    body = b"HTTP/1.1 200 OK\r\nContent-Encoding: " + value.encode() + b"\r\nContent-Length: 5\r\n\r\nhello"
    http, inner = make_client([body])
    with pytest.raises(httpx.ReadError):
        await guarded(http.get(URL))
    assert inner.streams[0].closed  # the connection is not kept


@pytest.mark.asyncio
async def test_two_content_encoding_headers_are_checked_each():
    body = b"HTTP/1.1 200 OK\r\nContent-Encoding: identity\r\nContent-Encoding: gzip\r\nContent-Length: 5\r\n\r\nhello"
    http, _ = make_client([body])
    with pytest.raises(httpx.ReadError):
        await guarded(http.get(URL))


@pytest.mark.asyncio
async def test_identity_content_encoding_is_accepted():
    body = b"HTTP/1.1 200 OK\r\nContent-Encoding: identity\r\nContent-Length: 5\r\n\r\nhello"
    http, _ = make_client([body])
    assert (await guarded(http.get(URL))).content == b"hello"


def _early_hints():
    while True:
        yield b"HTTP/1.1 103 Early Hints\r\nLink: </x>\r\n\r\n"


def _early_hints_slow(secs):
    while True:
        yield b"HTTP/1.1 103 Early Hints\r\nLink: </x>\r\n\r\n"
        yield secs


@pytest.mark.asyncio
async def test_endless_1xx_is_stopped_by_the_total_deadline(monkeypatch):
    monkeypatch.setattr(net, "TOTAL_DEADLINE_SECS", 0.4)
    http, _ = make_client(_early_hints_slow(0.01))
    t0 = time.monotonic()
    with pytest.raises(httpx.ReadTimeout):
        await guarded(http.get(URL), 5)
    assert time.monotonic() - t0 < 2


@pytest.mark.asyncio
async def test_1xx_flood_is_stopped_by_the_wire_byte_cap(monkeypatch):
    monkeypatch.setattr(net, "WIRE_BYTES_MAX", 200000)
    http, _ = make_client(_early_hints())
    with pytest.raises(httpx.ReadError):
        await guarded(http.get(URL), 5)


@pytest.mark.asyncio
async def test_slow_drip_body_is_stopped_by_the_total_deadline(monkeypatch):
    monkeypatch.setattr(net, "TOTAL_DEADLINE_SECS", 0.4)

    def drip():
        yield b"HTTP/1.1 200 OK\r\nContent-Length: 100000\r\n\r\n"
        while True:
            yield b"x"
            yield 0.05

    http, _ = make_client(drip())
    t0 = time.monotonic()
    with pytest.raises(httpx.ReadTimeout):
        r = await guarded(http.get(URL), 5)
        _ = r.content
    assert time.monotonic() - t0 < 2


@pytest.mark.asyncio
async def test_one_byte_chunk_flood_is_stopped_by_the_wire_cap(monkeypatch):
    monkeypatch.setattr(net, "WIRE_BYTES_MAX", 100000)

    def chunks():
        yield b"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n"
        while True:
            yield b"1\r\nx\r\n"

    http, inner = make_client(chunks())
    with pytest.raises(openai.APIConnectionError):
        r = await guarded(http.send(http.build_request("GET", URL), stream=True), 5)
        async for _ in r.aiter_bytes():
            pass
    assert max(inner.streams[0].read_sizes) <= 4096


@pytest.mark.asyncio
async def test_single_response_above_the_cap_fails(monkeypatch):
    monkeypatch.setattr(net, "WIRE_BYTES_MAX", 50000)
    body = b"HTTP/1.1 200 OK\r\nContent-Length: 100000\r\n\r\n" + b"x" * 100000
    http, _ = make_client([body])
    with pytest.raises(openai.APIConnectionError):
        await guarded(http.get(URL))


@pytest.mark.asyncio
async def test_wire_counter_resets_on_a_reused_connection(monkeypatch):
    monkeypatch.setattr(net, "WIRE_BYTES_MAX", 300000)
    big = b"HTTP/1.1 200 OK\r\nContent-Length: 100000\r\n\r\n" + b"x" * 100000
    http, inner = make_client([big] * 5)  # 500000 bytes > the cap, over 5 requests
    for _ in range(5):
        r = await guarded(http.get(URL))
        assert len(r.content) == 100000
    assert len(inner.dialed) == 1  # one pooled connection carried all of them


@pytest.mark.asyncio
async def test_mid_stream_reset_becomes_connection_error():
    items = [b"HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\npartial", httpcore.ReadError("connection reset secret-host")]
    http, _ = make_client(items)
    with pytest.raises(openai.APIConnectionError) as ei:
        await guarded(http.get(URL))
    assert str(ei.value) == "Connection error."
    assert isinstance(ei.value.__cause__, httpx.ReadError)


@pytest.mark.asyncio
async def test_remote_protocol_error_mid_body_becomes_connection_error():
    http, _ = make_client([b"HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\npartial", b""])  # EOF before 100 bytes
    with pytest.raises(openai.APIConnectionError) as ei:
        await guarded(http.get(URL))
    assert isinstance(ei.value.__cause__, httpx.RemoteProtocolError)


@pytest.mark.asyncio
async def test_read_timeout_mid_body_is_left_alone():
    items = [b"HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\npartial", httpcore.ReadTimeout("t")]
    http, _ = make_client(items)
    with pytest.raises(httpx.ReadTimeout):
        await guarded(http.get(URL))


@pytest.mark.asyncio
async def test_redirect_is_not_followed():
    redirect = b"HTTP/1.1 302 Found\r\nLocation: https://169.254.169.254/\r\nContent-Length: 0\r\n\r\n"
    http, inner = make_client([redirect])
    r = await guarded(http.get(URL))
    assert r.status_code == 302 and len(inner.dialed) == 1


@pytest.mark.asyncio
async def test_redirect_flag_and_trust_env_are_off_even_with_environment(monkeypatch):
    monkeypatch.setenv("HTTPS_PROXY", "http://127.0.0.1:1")
    monkeypatch.setenv("ALL_PROXY", "http://127.0.0.1:1")
    monkeypatch.setenv("SSL_CERT_FILE", "/nonexistent")
    http, inner = make_client()
    assert http.follow_redirects is False and http.trust_env is False and not http._mounts
    assert (await guarded(http.get(URL))).status_code == 200
    assert inner.dialed == [PUBLIC_A]


@pytest.mark.asyncio
async def test_pooled_connections_are_per_host_and_new_connects_revalidate():
    closing = b"HTTP/1.1 200 OK\r\nConnection: close\r\nContent-Length: 5\r\n\r\nhello"
    inner = FakeInner(lambda h: FakeStream([closing]))
    res = fixed([PUBLIC_A], ["10.1.1.1"])
    backend = net.ValidatingBackend(CUSTOMER, inner=inner, resolver=res)
    http = net.build_http_client(CUSTOMER, backend)
    assert (await guarded(http.get(URL))).status_code == 200
    with pytest.raises(httpx.ConnectError) as ei:
        await guarded(http.get(URL))
    assert isinstance(ei.value.__cause__, net.AddressBlocked)
    assert inner.dialed == [PUBLIC_A] and res.calls == 2


# --- client settings, TLS, self-check ----------------------------------------------------------------------


def test_client_settings():
    http = net.build_http_client(CUSTOMER)
    t = http.timeout
    assert (t.connect, t.read, t.write, t.pool) == (5.0, 30.0, 10.0, 5.0)
    assert http.follow_redirects is False and http.trust_env is False
    pool = http._transport._pool
    assert isinstance(http._transport, net.ValidatingTransport)
    assert isinstance(pool._network_backend, net.ValidatingBackend)
    assert pool._max_connections == 10 and pool._keepalive_expiry == 30.0 and pool._retries == 0
    assert pool._http2 is False and pool._http1 is True
    assert net.RESOLVE_TIMEOUT == 5.0 and net.CONNECT_TIMEOUT == 5.0


def test_certificate_verification_cannot_be_switched_off():
    """R16: the production context requires and checks the certificate; no parameter relaxes it."""
    assert net.ValidatingTransport.test_ssl_context is None  # production code never sets the hook
    http = net.build_http_client(CUSTOMER)
    ctx = http._transport._pool._ssl_context
    assert ctx.verify_mode == ssl.CERT_REQUIRED and ctx.check_hostname is True
    import inspect

    for fn in (net.build_http_client, net.build_openai_client, net.ValidatingTransport.__init__):
        assert "verify" not in inspect.signature(fn).parameters


def test_environment_does_not_change_the_trust_store(monkeypatch, tmp_path):
    other = tmp_path / "ca.pem"
    other.write_text("not a certificate")
    monkeypatch.setenv("SSL_CERT_FILE", str(other))
    monkeypatch.setenv("SSL_CERT_DIR", str(tmp_path))
    net._default_ssl_context.cache_clear()  # build the context under this environment
    try:
        ctx = net.build_http_client(CUSTOMER)._transport._pool._ssl_context
        assert ctx.verify_mode == ssl.CERT_REQUIRED and ctx.check_hostname
    finally:
        net._default_ssl_context.cache_clear()


def test_default_ssl_context_is_shared_by_all_clients():
    net._default_ssl_context.cache_clear()
    a = net.build_http_client(CUSTOMER)._transport._pool._ssl_context
    b = net.build_http_client(OTHER_CUSTOMER)._transport._pool._ssl_context
    assert a is b  # the CA bundle is loaded once, not once per session


def good_client(**kw):
    return net.build_openai_client(api_key="k", base_url=f"https://{HOST}/v1", customer_id=CUSTOMER, **kw)


def test_openai_client_matches_the_design():
    c = good_client()
    net.verify_openai_client(c, f"https://{HOST}/v1")
    assert c.max_retries == 0 and c.api_key == "k"
    assert isinstance(c._custom_headers["OpenAI-Organization"], openai.Omit)
    assert isinstance(c._custom_headers["OpenAI-Project"], openai.Omit)
    assert c.timeout == net.TIMEOUT


def test_key_is_a_string_never_none_and_env_key_is_not_used(monkeypatch):
    monkeypatch.setenv("OPENAI_API_KEY", "operator-key")
    for given in (None, ""):
        c = net.build_openai_client(api_key=given, base_url=f"https://{HOST}/v1")
        assert c.api_key == "" and "operator-key" not in repr(c._custom_headers)
    assert good_client().api_key == "k"


@pytest.mark.asyncio
async def test_environment_credentials_never_reach_the_wire(monkeypatch):
    monkeypatch.setenv("OPENAI_API_KEY", "operator-key")
    monkeypatch.setenv("OPENAI_ORG_ID", "org-operator")
    monkeypatch.setenv("OPENAI_PROJECT_ID", "proj-operator")
    monkeypatch.setenv("OPENAI_BASE_URL", "https://evil.example.test/v1")
    inner = FakeInner(lambda h: FakeStream([OK, OK]))
    backend = net.ValidatingBackend(CUSTOMER, inner=inner, resolver=fixed([PUBLIC_A]))
    for key, expect_auth in (("", False), ("stored-key", True)):
        c = net.build_openai_client(api_key=key, base_url=f"https://{HOST}/v1", customer_id=CUSTOMER, backend=backend)
        try:
            await guarded(c.chat.completions.create(model="m", messages=[{"role": "user", "content": "x"}]))
        except Exception:
            pass  # the scripted body is not a completion; only the request matters
        wire = inner.streams[-1].writes[0].lower()
        assert (b"authorization: bearer stored-key" in wire) is expect_auth
        if not expect_auth:
            assert b"authorization" not in wire
        assert b"operator" not in wire and b"openai-organization" not in wire and b"openai-project" not in wire
    assert set(inner.dialed) == {PUBLIC_A}


def _broken_clients():
    base = f"https://{HOST}/v1"
    backend = net.ValidatingBackend(CUSTOMER, inner=FakeInner(lambda h: FakeStream([])), resolver=fixed([PUBLIC_A]))
    yield "plain sdk client", openai.AsyncOpenAI(api_key="k", base_url=base)
    yield "retries", openai.AsyncOpenAI(
        api_key="k", base_url=base, http_client=net.build_http_client(CUSTOMER, backend), max_retries=2, timeout=net.TIMEOUT,
        default_headers={"OpenAI-Organization": openai.Omit(), "OpenAI-Project": openai.Omit()},
    )
    yield "no omit headers", openai.AsyncOpenAI(
        api_key="k", base_url=base, http_client=net.build_http_client(CUSTOMER, backend), max_retries=0, timeout=net.TIMEOUT
    )
    yield "redirects", _with_http(httpx.AsyncClient(transport=net.ValidatingTransport(backend), timeout=net.TIMEOUT, follow_redirects=True, trust_env=False))
    yield "trust_env", _with_http(httpx.AsyncClient(transport=net.ValidatingTransport(backend), timeout=net.TIMEOUT, follow_redirects=False, trust_env=True))
    yield "mounts", _with_http(
        httpx.AsyncClient(
            transport=net.ValidatingTransport(backend), mounts={"all://": httpx.AsyncHTTPTransport()}, timeout=net.TIMEOUT,
            follow_redirects=False, trust_env=False,
        )
    )
    yield "default transport", _with_http(httpx.AsyncClient(timeout=net.TIMEOUT, follow_redirects=False, trust_env=False))
    yield "timeout", _with_http(httpx.AsyncClient(transport=net.ValidatingTransport(backend), timeout=600, follow_redirects=False, trust_env=False))
    c = _with_http(net.build_http_client(CUSTOMER, backend))
    c._client._transport._pool._keepalive_expiry = 600.0
    yield "keepalive", c
    c = _with_http(net.build_http_client(CUSTOMER, backend))
    c._client._transport._pool._network_backend = httpcore.AsyncNetworkBackend()
    yield "pool backend", c
    c = _with_http(net.build_http_client(CUSTOMER, backend))
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    c._client._transport._pool._ssl_context = ctx
    yield "tls verification off", c
    c = _with_http(net.build_http_client(CUSTOMER, backend))
    c._client._transport._pool._max_connections = 1000
    yield "max connections", c


def _with_http(http):
    return openai.AsyncOpenAI(
        api_key="k",
        base_url=f"https://{HOST}/v1",
        http_client=http,
        max_retries=0,
        timeout=net.TIMEOUT,
        default_headers={"OpenAI-Organization": openai.Omit(), "OpenAI-Project": openai.Omit()},
    )


@pytest.mark.parametrize("name,client", list(_broken_clients()), ids=lambda v: v if isinstance(v, str) else "")
def test_self_check_fails_closed(name, client):
    with pytest.raises(net.SelfCheckError):
        net.verify_openai_client(client, f"https://{HOST}/v1")


def test_self_check_fails_on_wrong_base_url():
    with pytest.raises(net.SelfCheckError):
        net.verify_openai_client(good_client(), "https://other.example.test/v1")


# --- request level checks ------------------------------------------------------------------------------------------


@pytest.mark.parametrize(
    "value", [CUSTOMER, CUSTOMER.upper(), CUSTOMER.replace("-", ""), "{" + CUSTOMER + "}"]
)
def test_customer_id_is_canonical(value):
    if value.startswith("{"):
        with pytest.raises(ValueError):
            net.parse_customer_id(value)
    else:
        assert net.parse_customer_id(value) == CUSTOMER


@pytest.mark.parametrize("value", [None, "", "x", 5, "11111111-2222-3333-4444-55555555555", "urn:uuid:" + CUSTOMER, CUSTOMER + "zz", "g" * 36, "00000000-0000-0000-0000-000000000000", "0" * 32])
def test_customer_id_missing_or_malformed_is_refused(value):
    with pytest.raises(ValueError) as ei:
        net.parse_customer_id(value)
    assert str(ei.value) == net.FIXED_ERROR


@pytest.mark.parametrize("url", ["https://llm.example.com/v1", "https://llm.example.com:8443/v1", "HTTPS://llm.example.com/v1", "https://[2606:4700::1111]/v1"])
def test_runner_accepts_https_without_userinfo(url):
    assert net.check_runner_base_url(url) == url


@pytest.mark.parametrize(
    "url",
    [
        None, "", "http://llm.example.com/v1", "ftp://llm.example.com", "llm.example.com/v1", "https:///v1", "//llm.example.com",
        "https://user@llm.example.com/v1", "https://user:pw@llm.example.com/v1", "https://:pw@llm.example.com/v1",
        "https://llm.example.com\\@evil.com/v1", "https://llm.example.com/v1 x", "https://llm.example.com/v\n1",
        "https://llm.example.com:99999/v1", "https://llm.example.com:abc/v1", "https://llm.exämple.com/v1", "https://llm.example.com/" + "a" * 3000,
    ],
)
def test_runner_refuses_other_urls_with_the_fixed_text(url):
    with pytest.raises(ValueError) as ei:
        net.check_runner_base_url(url)
    assert str(ei.value) == net.FIXED_ERROR
    assert ei.value.__cause__ is None and (url is None or "llm.example.com" not in str(ei.value))


@pytest.mark.parametrize(
    "t,expected",
    [("customendpoint.gpt", True), ("CustomEndpoint.gpt", True), ("customendpoint:gpt", True), ("customendpoint", True), ("openai.gpt-4o", False),
     ("local.a.b", False), ("platform_openrouter.x", False), ("", False), (None, False), ("x.customendpoint.y", False)],
)
def test_is_custom_endpoint_type(t, expected):
    assert net.is_custom_endpoint_type(t) is expected


@pytest.mark.parametrize(
    "text,expected",
    [
        ("LLM completion timeout", "LLM completion timeout"),
        ("Error during completion: Connection error.", "Error during completion: Connection error."),
        ("Error during completion: Request timed out.", "Error during completion: Request timed out."),
        ("Error during completion: Error code: 401 - {'error': 'Bearer sk-abc leaked'}", "Error during completion: Error code: 401"),
        ("Error during completion: Error code: 429 - body", "Error during completion: Error code: 429"),
        ("Error during completion: Error code: 4011 - body", "Error during completion: error"),
        ("Error during completion: ", "Error during completion: error"),
        ("Error during completion: secret body text", "Error during completion: error"),
        ("something secret", "Error during completion: error"),
        (None, "Error during completion: error"),
    ],
)
def test_sanitize_error_text_keeps_classifier_phrases_and_drops_bodies(text, expected):
    assert net.sanitize_error_text(text) == expected


# --- slots and the pipeline registry ----------------------------------------------------------------------------------


@pytest.fixture
def slots(monkeypatch):
    table = net.SlotTable()
    monkeypatch.setattr(net, "SLOTS", table)
    monkeypatch.setattr(net, "_PIPELINES", {})
    monkeypatch.setattr(net, "_RESERVED", set())
    return table


def test_slot_limits_per_customer_and_total(slots, monkeypatch):
    assert net.SLOTS_PER_CUSTOMER == 16 and net.SLOTS_TOTAL == 64
    assert all(slots.acquire(CUSTOMER) for _ in range(16))
    assert not slots.acquire(CUSTOMER)  # per customer limit
    for i in range(3):
        c = f"00000000-0000-0000-0000-{i:012d}"
        assert all(slots.acquire(c) for _ in range(16))
    assert slots.held() == 64
    assert not slots.acquire("00000000-0000-0000-0000-000000000099")  # total limit
    slots.release(CUSTOMER)
    assert slots.acquire("00000000-0000-0000-0000-000000000099")
    assert slots.held(CUSTOMER) == 15


def test_slot_release_never_goes_below_zero(slots):
    slots.release(CUSTOMER)
    assert slots.held() == 0
    assert slots.acquire(CUSTOMER)
    slots.release(CUSTOMER)
    slots.release(CUSTOMER)
    assert slots.held() == 0 and slots.held(CUSTOMER) == 0


class Svc:
    def __init__(self, customer=CUSTOMER, pipeline="p1", table=None):
        self.customer, self.pipeline = customer, pipeline
        self.slot, self.closed, self.calls = True, 0, []
        self.table = table

    def release_slot(self):
        if self.slot:
            self.slot = False
            self.table.release(self.customer)
            net.unregister_service(self.pipeline, self)

    async def release(self):
        self.calls.append("release")
        self.release_slot()
        self.closed += 1


def test_pipeline_reservation_refuses_duplicates_and_bad_ids(slots):
    assert net.reserve_pipeline("p1") is True
    assert net.pipeline_registered("p1")
    assert net.reserve_pipeline("p1") is False
    for bad in (None, "", 5):
        assert net.reserve_pipeline(bad) is False


@pytest.mark.asyncio
async def test_release_pipeline_releases_every_recorded_service_and_forgets(slots):
    svcs = [Svc(table=slots) for _ in range(3)]
    for s in svcs:
        assert slots.acquire(CUSTOMER)
        net.register_service("p1", s)
    await net.release_pipeline("p1")
    assert slots.held() == 0 and all(s.closed == 1 for s in svcs)
    assert not net.pipeline_registered("p1")
    await net.release_pipeline("p1")  # idempotent
    await net.release_pipeline("unknown")


@pytest.mark.asyncio
async def test_release_pipeline_gives_slots_back_before_the_first_await(slots):
    s = Svc(table=slots)
    assert slots.acquire(CUSTOMER)
    net.register_service("p1", s)
    coro = net.release_pipeline("p1")
    task = asyncio.ensure_future(coro)
    await asyncio.sleep(0)  # runs up to the first suspension only
    task.cancel()
    try:
        await task
    except asyncio.CancelledError:
        pass
    assert slots.held() == 0


@pytest.mark.asyncio
async def test_release_pipeline_continues_after_a_failing_release(slots):
    class Bad(Svc):
        async def release(self):
            raise RuntimeError("boom")

    a, b = Bad(table=slots), Svc(table=slots)
    for s in (a, b):
        assert slots.acquire(CUSTOMER)
        net.register_service("p1", s)
    await net.release_pipeline("p1")
    assert slots.held() == 0 and b.closed == 1
