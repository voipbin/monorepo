"""Real-library tests for the customer endpoint guard (VOIP-1599 Layer 2, design 7 items 6 and 8a).

Uses the real pipecat 1.12, httpx, httpcore, openai and a local TLS stub (trustme). Lives outside
scripts/pipecat/ for the same reason as the other files here (that conftest replaces pipecat with
MagicMocks). Not run by CI. Run manually in a venv with the pinned versions
(scripts/pipecat/requirements.txt) plus `pytest pytest-asyncio trustme`:

    pytest --noconftest bin-pipecat-manager/scripts/pipecat_realtest/test_custom_endpoint_real.py

No test dials a public address: the validating backend resolves to a stand-in public address and
its inner backend dials the local stub instead.
"""
import asyncio
import gzip
import json
import os
import resource
import ssl
import sys
import time
import uuid
import warnings
import zlib
from unittest.mock import AsyncMock

import pytest
import pytest_asyncio

pytest.importorskip("pipecat.flows")
trustme = pytest.importorskip("trustme")

HERE = os.path.dirname(__file__)
sys.path.insert(0, os.path.abspath(os.path.join(HERE, "..", "pipecat")))
for k in ("DEEPGRAM_API_KEY", "CARTESIA_API_KEY", "ELEVENLABS_API_KEY", "GOOGLE_API_KEY"):
    os.environ.setdefault(k, "x")
warnings.simplefilter("ignore")

import httpcore  # noqa: E402
import httpx  # noqa: E402
import openai  # noqa: E402
from httpcore import AsyncNetworkBackend  # noqa: E402
from httpcore._backends.auto import AutoBackend  # noqa: E402
from loguru import logger  # noqa: E402

import custom_endpoint as ce  # noqa: E402
import custom_endpoint_net as net  # noqa: E402
import run  # noqa: E402

HOST = "llm.example.test"
PUBLIC_A = "8.8.8.8"  # stand-ins for "public"; the inner backend never dials them
PUBLIC_B = "1.1.1.1"
KEY = "sk-REAL-endpoint-key-123"
LEGACY = "legacy-engine-key-DO-NOT-FORWARD"
CUSTOMER = str(uuid.uuid4())
MESSAGES = [{"role": "user", "content": "ping"}]

pytestmark = pytest.mark.asyncio


@pytest.fixture(autouse=True)
def _leave_a_current_event_loop():
    """pytest-asyncio leaves no current event loop behind; other tests of this directory call
    asyncio.get_event_loop() from sync code and would fail when they run after this file."""
    yield
    try:
        asyncio.get_event_loop_policy().get_event_loop()
    except RuntimeError:
        asyncio.set_event_loop(asyncio.new_event_loop())


# --- TLS stub ---------------------------------------------------------------------------------------


async def read_request(r):
    head = await r.readuntil(b"\r\n\r\n")
    lines = head.decode("latin-1").split("\r\n")
    method, path, _ = lines[0].split(" ", 2)
    headers = {}
    for line in lines[1:]:
        if ":" in line:
            n, v = line.split(":", 1)
            headers[n.strip().lower()] = v.strip()
    body = b""
    if headers.get("content-length"):
        body = await r.readexactly(int(headers["content-length"]))
    return method, path, headers, body


COMPLETION = json.dumps({
    "id": "c1", "object": "chat.completion", "created": 1, "model": "m",
    "choices": [{"index": 0, "finish_reason": "stop", "message": {"role": "assistant", "content": "pong"}}],
}).encode()


def http_response(status, body=b"", headers=None, reason="OK"):
    h = {"Content-Length": str(len(body)), "Content-Type": "application/json"}
    h.update(headers or {})
    head = "HTTP/1.1 %d %s\r\n" % (status, reason) + "".join("%s: %s\r\n" % kv for kv in h.items()) + "\r\n"
    return head.encode() + body


async def ok(req, w):
    w.write(http_response(200, COMPLETION))
    await w.drain()
    return True


class Stub:
    """Local TLS HTTP/1.1 stub. Records SNI, requests and connections; `mode` writes the response."""

    def __init__(self, mode=ok, cert_host=HOST):
        self.ca = trustme.CA()
        cert = self.ca.issue_cert(cert_host)
        self.ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        cert.configure_cert(self.ctx)
        self.sni = []
        self.ctx.sni_callback = lambda s, name, c: self.sni.append(name)
        self.requests = []
        self.conns = 0
        self.mode = mode
        self.sent = 0
        self.writers = []

    async def start(self):
        self.server = await asyncio.start_server(self._h, "127.0.0.1", 0, ssl=self.ctx)
        self.port = self.server.sockets[0].getsockname()[1]
        return self

    async def stop(self):
        self.server.close()
        for w in list(self.writers):
            w.transport.abort()
        try:
            await asyncio.wait_for(self.server.wait_closed(), 5)
        except asyncio.TimeoutError:
            pass

    def client_ctx(self):
        c = ssl.create_default_context()
        self.ca.configure_trust(c)
        return c

    @property
    def url(self):
        return "https://%s:%d/v1" % (HOST, self.port)

    async def _h(self, r, w):
        self.conns += 1
        self.writers.append(w)
        try:
            while True:
                req = await read_request(r)
                self.requests.append(req)
                if not await self.mode(req, w):
                    break
        except (asyncio.IncompleteReadError, ConnectionError, ssl.SSLError):
            pass
        finally:
            try:
                w.close()
            except Exception:
                pass


class CountingStream(httpcore.AsyncNetworkStream):
    """Counts the plaintext bytes the guard asked its inner stream for (below the wire guard)."""

    def __init__(self, inner, counter):
        self._inner = inner
        self._counter = counter

    async def read(self, max_bytes, timeout=None):
        data = await self._inner.read(max_bytes, timeout=timeout)
        self._counter[0] += len(data)
        return data

    async def write(self, buffer, timeout=None):
        await self._inner.write(buffer, timeout=timeout)

    async def aclose(self):
        await self._inner.aclose()

    async def start_tls(self, ssl_context, server_hostname=None, timeout=None):
        return CountingStream(await self._inner.start_tls(ssl_context, server_hostname=server_hostname, timeout=timeout), self._counter)

    def get_extra_info(self, info):
        return self._inner.get_extra_info(info)


class DialStub(AsyncNetworkBackend):
    """Inner backend: records the validated addresses it is asked to dial and dials the stub."""

    def __init__(self, port):
        self.port = port
        self.dialed = []
        self.timeouts = []
        self.received = [0]
        self._real = AutoBackend()

    async def connect_tcp(self, host, port, timeout=None, **kw):
        self.dialed.append(host)
        self.timeouts.append(timeout)
        return CountingStream(await self._real.connect_tcp("127.0.0.1", self.port, timeout=timeout), self.received)

    async def sleep(self, s):
        await asyncio.sleep(s)


class Answers:
    def __init__(self, *answers):
        self.answers = list(answers)
        self.calls = 0

    async def __call__(self, host, port):
        a = self.answers[min(self.calls, len(self.answers) - 1)]
        self.calls += 1
        return list(a)


class Env:
    """The pieces of one test: stub, dial recorder, the hooks installed on the production classes."""

    def __init__(self, stub, answers):
        self.stub = stub
        self.dial = DialStub(stub.port)
        self.answers = answers
        self.backend = net.ValidatingBackend(CUSTOMER, inner=self.dial, resolver=answers)

    def service(self, key=KEY, pipeline="p-%s", customer=CUSTOMER):
        llm, _ = run.create_llm_service(
            "customendpoint.m", LEGACY, MESSAGES, [], pipeline_id=pipeline % uuid.uuid4().hex[:8],
            customer_id=customer, llm_base_url=self.stub.url, llm_base_key=key,
        )
        return llm


@pytest_asyncio.fixture
async def make_env(monkeypatch):
    stubs = []

    async def make(mode=ok, answers=None, cert_host=HOST, trust=True):
        stub = await Stub(mode, cert_host).start()
        stubs.append(stub)
        env = Env(stub, answers or Answers([PUBLIC_A]))
        monkeypatch.setattr(ce.CustomEndpointLLMService, "_http_backend", env.backend)
        if trust:
            monkeypatch.setattr(net.ValidatingTransport, "test_ssl_context", stub.client_ctx())
        return env

    monkeypatch.setattr(net, "SLOTS", net.SlotTable())
    monkeypatch.setattr(net, "_PIPELINES", {})
    monkeypatch.setattr(net, "_RESERVED", set())
    yield make
    for s in stubs:
        await s.stop()


async def chat(client, **kw):
    return await asyncio.wait_for(
        client.chat.completions.create(model="m", messages=MESSAGES, **kw), 20)


class LoopLag:
    """Largest delay of a 5 ms timer on the event loop."""

    def __init__(self):
        self.max = 0.0
        self._t = None

    async def _run(self):
        loop = asyncio.get_running_loop()
        while True:
            t = loop.time()
            await asyncio.sleep(0.005)
            self.max = max(self.max, loop.time() - t - 0.005)

    def __enter__(self):
        self._t = asyncio.get_running_loop().create_task(self._run())
        return self

    def __exit__(self, *a):
        self._t.cancel()


# --- the subclass is what create_llm_service builds --------------------------------------------------


async def test_create_llm_service_builds_the_subclass_and_it_completes_a_request(make_env, monkeypatch):
    for k, v in (("OPENAI_API_KEY", "operator-key"), ("OPENAI_ORG_ID", "org-operator"), ("OPENAI_PROJECT_ID", "proj-operator")):
        monkeypatch.setenv(k, v)
    env = await make_env()
    llm = env.service()
    from pipecat.services.openai.llm import OpenAILLMService
    assert type(llm) is ce.CustomEndpointLLMService and isinstance(llm, OpenAILLMService)
    assert net.SLOTS.held(CUSTOMER) == 1
    net.verify_openai_client(llm._client)
    r = await chat(llm._client)
    assert r.choices[0].message.content == "pong"
    method, path, headers, body = env.stub.requests[0]
    assert (method, path) == ("POST", "/v1/chat/completions")
    assert headers["authorization"] == "Bearer " + KEY
    assert "openai-organization" not in headers and "openai-project" not in headers
    assert headers["accept-encoding"] == "identity"
    assert LEGACY not in json.dumps(headers) and "operator" not in json.dumps(headers)
    assert env.stub.sni == [HOST] and env.dial.dialed == [PUBLIC_A]
    assert len(env.dial.timeouts) == 1 and 4.9 < env.dial.timeouts[0] <= 5.0  # the shared connect deadline
    await llm.cleanup()
    assert net.SLOTS.held() == 0


async def test_keyless_endpoint_sends_no_authorization_header(make_env, monkeypatch):
    monkeypatch.setenv("OPENAI_API_KEY", "operator-key")
    env = await make_env()
    llm = env.service(key="")
    assert llm._client.api_key == ""
    await chat(llm._client)
    assert "authorization" not in env.stub.requests[0][2]
    await llm.cleanup()


async def test_environment_has_no_effect(make_env, monkeypatch, tmp_path):
    hits = []

    async def proxy(r, w):
        hits.append(1)
        w.close()

    px = await asyncio.start_server(proxy, "127.0.0.1", 0)
    pport = px.sockets[0].getsockname()[1]
    for k in ("HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy", "HTTP_PROXY"):
        monkeypatch.setenv(k, "http://127.0.0.1:%d" % pport)
    monkeypatch.setenv("OPENAI_BASE_URL", "https://evil.example.test/v1")
    monkeypatch.setenv("OPENAI_API_KEY", "operator-key")
    try:
        env = await make_env()
        llm = env.service()
        await chat(llm._client)
        assert hits == [] and env.dial.dialed == [PUBLIC_A] and env.stub.conns == 1
        # control: a default httpx client honours the variable
        ctl = httpx.AsyncClient(verify=env.stub.client_ctx())
        with pytest.raises(httpx.HTTPError):
            await ctl.get("https://%s:%d/x" % (HOST, env.stub.port))
        assert hits == [1]
        await ctl.aclose()
        await llm.cleanup()
    finally:
        px.close()


async def test_ssl_cert_file_does_not_extend_trust(make_env, monkeypatch, tmp_path):
    env = await make_env(trust=False)  # production trust store: the stub CA is not in it
    pem = tmp_path / "ca.pem"
    pem.write_bytes(env.stub.ca.cert_pem.bytes())
    monkeypatch.setenv("SSL_CERT_FILE", str(pem))
    monkeypatch.setenv("SSL_CERT_DIR", str(tmp_path))
    net._default_ssl_context.cache_clear()  # build the context under this environment
    try:
        llm = env.service()
        with pytest.raises(openai.APIConnectionError):
            await chat(llm._client)
        assert env.stub.requests == []
        await llm.cleanup()
    finally:
        net._default_ssl_context.cache_clear()


async def test_certificate_for_another_name_is_rejected_and_no_request_is_sent(make_env):
    env = await make_env(cert_host="other.example.test")
    llm = env.service()
    with pytest.raises(openai.APIConnectionError) as ei:
        await chat(llm._client)
    assert "Connection error." == str(ei.value)
    assert env.stub.requests == [] and env.stub.sni == [HOST]
    await llm.cleanup()


# --- rebinding, redirects, pooled reuse -----------------------------------------------------------------


async def test_rebinding_is_refused_at_connect(make_env):
    async def close_each(req, w):
        w.write(http_response(200, COMPLETION, {"Connection": "close"}))
        await w.drain()
        return False

    env = await make_env(close_each, Answers([PUBLIC_A], ["10.0.0.5"]))
    llm = env.service()
    assert (await chat(llm._client)).choices[0].message.content == "pong"
    with pytest.raises(openai.APIConnectionError) as ei:
        await chat(llm._client)
    assert isinstance(ei.value.__cause__.__cause__, net.AddressBlocked)
    assert env.dial.dialed == [PUBLIC_A] and env.answers.calls == 2
    await llm.cleanup()


async def test_mixed_answer_is_refused(make_env):
    env = await make_env(answers=Answers([PUBLIC_A, "192.168.1.9"]))
    llm = env.service()
    with pytest.raises(openai.APIConnectionError):
        await chat(llm._client)
    assert env.dial.dialed == [] and env.stub.conns == 0
    await llm.cleanup()


async def test_redirect_is_not_followed(make_env):
    async def redirect(req, w):
        w.write(http_response(302, b"", {"Location": "https://%s:%d/evil" % (HOST, env.stub.port)}, "Found"))
        await w.drain()
        return True

    env = await make_env(redirect)
    llm = env.service()
    with pytest.raises(openai.APIStatusError) as ei:
        await chat(llm._client)
    assert ei.value.status_code == 302
    assert [r[1] for r in env.stub.requests] == ["/v1/chat/completions"]
    await llm.cleanup()


async def test_pooled_connection_is_reused_without_resolving_again(make_env):
    env = await make_env(answers=Answers([PUBLIC_A], ["10.0.0.5"]))
    llm = env.service()
    await chat(llm._client)
    await chat(llm._client)  # same socket: no connect, no resolve
    assert env.stub.conns == 1 and env.answers.calls == 1
    pool = llm._client._client._transport._pool
    for c in list(pool.connections):  # the connection goes away: the next request connects again
        await c.aclose()
    with pytest.raises(openai.APIConnectionError):
        await chat(llm._client)
    assert env.answers.calls == 2 and env.dial.dialed == [PUBLIC_A]
    await llm.cleanup()


@pytest.mark.parametrize("url", [
    "https://localhost/v1", "https://127.0.0.1/v1", "https://[::1]/v1", "https://2130706433/v1", "https://0x7f.1/v1",
    "https://017700000001/v1", "https://127.1/v1", "https://[::ffff:127.0.0.1]/v1", "https://169.254.169.254/v1",
    "https://10.0.0.1/v1", "https://[64:ff9b::a9fe:a9fe]/v1", "https://0/v1", "https://localhost./v1",
])
async def test_real_resolver_refuses_loopback_and_odd_forms(url, monkeypatch):
    class NoDial(AsyncNetworkBackend):
        async def connect_tcp(self, *a, **k):
            raise AssertionError("dialed")

        async def sleep(self, s):
            pass

    monkeypatch.setattr(net, "SLOTS", net.SlotTable())
    client = net.build_openai_client(api_key="k", base_url=url, customer_id=CUSTOMER, backend=net.ValidatingBackend(CUSTOMER, inner=NoDial()))
    with pytest.raises(openai.APIConnectionError) as ei:
        await asyncio.wait_for(client.chat.completions.create(model="m", messages=MESSAGES), 15)
    cause = ei.value.__cause__.__cause__
    assert isinstance(cause, httpcore.ConnectError) and not isinstance(cause, AssertionError), repr(cause)
    await client.close()


# --- response guards ---------------------------------------------------------------------------------------------


async def test_gzip_bomb_is_refused_without_inflating(make_env):
    comp = zlib.compressobj(wbits=31)
    parts = [comp.compress(b"\0" * (1 << 20)) for _ in range(256)] + [comp.flush()]  # 256 MiB when inflated
    bomb = b"".join(parts)

    async def gz(req, w):
        w.write(http_response(200, bomb, {"Content-Encoding": "gzip"}))
        await w.drain()
        return True

    env = await make_env(gz)
    llm = env.service()
    before = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
    with pytest.raises(openai.APIConnectionError):
        await chat(llm._client)
    grown_mb = (resource.getrusage(resource.RUSAGE_SELF).ru_maxrss - before) / 1024
    assert len(bomb) < 1 << 20 and grown_mb < 60, grown_mb
    await llm.cleanup()


async def test_response_over_four_mebibytes_is_cut_off_without_starving_the_loop(make_env):
    async def endless_sse(req, w):
        w.write(b"HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\nTransfer-Encoding: chunked\r\n\r\n")
        line = b"data: " + b"x" * 1000 + b"\n"  # never an empty line: one endless unterminated event
        chunk = ("%x\r\n" % len(line)).encode() + line + b"\r\n"
        try:
            while True:
                w.write(chunk)
                env.stub.sent += len(chunk)
                await w.drain()
        except (ConnectionError, ssl.SSLError):
            pass
        return False

    env = await make_env(endless_sse)
    llm = env.service()
    with LoopLag() as lag:
        with pytest.raises(openai.APIConnectionError):
            stream = await chat(llm._client, stream=True)
            async for _ in stream:
                pass
    # the guard stopped the read right after the 4 MiB cap (the server wrote more into socket buffers)
    assert 4 * 1024 * 1024 < env.dial.received[0] <= 4 * 1024 * 1024 + 4096, env.dial.received[0]
    assert lag.max < 0.25, lag.max
    await llm.cleanup()


async def test_one_byte_chunks_and_1xx_flood_do_not_starve_the_loop(make_env, monkeypatch):
    async def tiny_chunks(req, w):
        w.write(b"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n")
        try:
            while True:
                w.write(b"1;ext=" + b"e" * 20 + b"\r\nx\r\n" * 1)
                await w.drain()
        except (ConnectionError, ssl.SSLError):
            pass
        return False

    async def flood_103(req, w):
        try:
            while True:
                w.write(b"HTTP/1.1 103 Early Hints\r\nLink: </x>\r\n\r\n")
                await w.drain()
        except (ConnectionError, ssl.SSLError):
            pass
        return False

    monkeypatch.setattr(net, "WIRE_BYTES_MAX", 1024 * 1024)  # keep the test short; the cap logic is the same
    for mode in (tiny_chunks, flood_103):
        env = await make_env(mode)
        llm = env.service()
        with LoopLag() as lag:
            with pytest.raises(openai.APIConnectionError):
                await chat(llm._client)
        assert lag.max < 0.25, (mode.__name__, lag.max)
        await llm.cleanup()


async def test_slow_drip_is_stopped_by_the_total_deadline(make_env, monkeypatch):
    monkeypatch.setattr(net, "TOTAL_DEADLINE_SECS", 1.0)

    async def drip(req, w):
        w.write(b"HTTP/1.1 200 OK\r\nContent-Length: 100000\r\n\r\n")
        try:
            while True:
                w.write(b"x")
                await w.drain()
                await asyncio.sleep(0.1)
        except (ConnectionError, ssl.SSLError):
            pass
        return False

    env = await make_env(drip)
    llm = env.service()
    t0 = time.monotonic()
    with pytest.raises((openai.APITimeoutError, httpx.ReadTimeout)):
        await chat(llm._client)
    assert time.monotonic() - t0 < 6
    await llm.cleanup()


async def _sse_then_reset(req, w):
    w.write(b"HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\nTransfer-Encoding: chunked\r\n\r\n")
    first = b'data: {"id":"c","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":"he"}}]}\n\n'
    w.write(("%x\r\n" % len(first)).encode() + first + b"\r\n")
    await w.drain()
    await asyncio.sleep(0.05)
    w.transport.abort()  # reset in the middle of the stream
    return False


async def test_reset_in_the_middle_of_a_stream_is_a_connection_error(make_env):
    env = await make_env(_sse_then_reset)
    llm = env.service()
    seen = []
    with pytest.raises(openai.APIConnectionError) as ei:
        stream = await chat(llm._client, stream=True)
        async for chunk in stream:
            seen.append(chunk)
    assert str(ei.value) == "Connection error." and len(seen) == 1
    await llm.cleanup()


async def test_pipecat_reports_the_reset_as_connection_error_and_sanitizes_bodies(make_env):
    """Through the real OpenAILLMService._process_context and the push_error override."""
    env = await make_env(_sse_then_reset)
    llm = env.service()
    pushed = []
    llm.push_frame = AsyncMock()
    from pipecat.processors.aggregators.llm_context import LLMContext
    from pipecat.frames.frames import LLMContextFrame
    from pipecat.processors.frame_processor import FrameDirection
    llm.start_processing_metrics = AsyncMock()
    llm.stop_processing_metrics = AsyncMock()
    orig = ce.OpenAILLMService.push_error

    async def spy(self, error_msg, exception=None, **kw):
        pushed.append((error_msg, exception, kw.get("category")))

    ce.OpenAILLMService.push_error = spy
    try:
        await llm.process_frame(LLMContextFrame(LLMContext(messages=MESSAGES)), FrameDirection.DOWNSTREAM)
    finally:
        ce.OpenAILLMService.push_error = orig
    assert pushed and pushed[0][0] == "Error during completion: Connection error."
    assert pushed[0][1] is None  # the exception (and its text) is not forwarded
    await llm.cleanup()


async def test_http_error_body_never_reaches_the_error_text_or_the_logs(make_env):
    async def echo_401(req, w):
        body = json.dumps({"error": {"message": "bad key: " + req[2].get("authorization", ""), "type": "invalid_request_error"}}).encode()
        w.write(http_response(401, body, reason="Unauthorized"))
        await w.drain()
        return True

    env = await make_env(echo_401)
    llm = env.service()
    llm.push_frame = AsyncMock()
    lines = []
    sink = logger.add(lines.append, level="DEBUG")
    try:
        with pytest.raises(openai.AuthenticationError) as ei:
            await chat(llm._client)
        raw = "Error during completion: %s" % ei.value
        assert KEY in raw  # the SDK text does carry the echoed body: that is what the override removes
        await llm.push_error(error_msg=raw, exception=ei.value)
    finally:
        logger.remove(sink)
    frame = llm.push_frame.call_args.args[0]
    assert frame.error == "Error during completion: Error code: 401" and frame.exception is None
    assert frame.category.value == "authentication"  # category worked out before the exception was dropped
    text = "".join(str(m) for m in lines)
    assert KEY not in text and "bad key" not in text and "Error code: 401" in text
    await llm.cleanup()


async def test_422_body_is_dropped_too(make_env):
    async def unprocessable(req, w):
        w.write(http_response(422, json.dumps({"detail": "echo " + json.dumps(req[3].decode())}).encode(), reason="Unprocessable"))
        await w.drain()
        return True

    env = await make_env(unprocessable)
    llm = env.service()
    llm.push_frame = AsyncMock()
    with pytest.raises(openai.UnprocessableEntityError) as ei:
        await chat(llm._client)
    await llm.push_error(error_msg="Error during completion: %s" % ei.value, exception=ei.value)
    assert llm.push_frame.call_args.args[0].error == "Error during completion: Error code: 422"
    await llm.cleanup()


# --- push_error keeps the phrases the Go classifier needs ----------------------------------------------------


@pytest.mark.parametrize("text", [
    "LLM completion timeout", "Error during completion: Request timed out.", "Error during completion: Connection error.",
])
async def test_push_error_keeps_fixed_phrases(make_env, text):
    env = await make_env()
    llm = env.service()
    llm.push_frame = AsyncMock()
    await llm.push_error(error_msg=text)
    assert llm.push_frame.call_args.args[0].error == text
    await llm.cleanup()


# --- slots, release and cleanup --------------------------------------------------------------------------------------


async def test_release_and_cleanup_are_idempotent(make_env):
    env = await make_env()
    llm = env.service()
    other = env.service()  # same customer: a second give-back of `llm` would take its slot
    assert net.SLOTS.held() == 2
    real_close = llm._client.close
    closes = []

    async def counting_close():
        closes.append(1)
        await real_close()

    llm._client.close = counting_close
    await llm.cleanup()
    await llm.cleanup()
    await llm.release()
    llm.release_slot()
    llm.release_slot()
    assert net.SLOTS.held() == 1  # exactly once: the other service's slot is untouched
    assert len(closes) == 1  # and the client is closed once
    assert llm._client.is_closed()
    await other.cleanup()
    assert net.SLOTS.held() == 0


async def test_cleanup_gives_the_slot_back_when_super_cleanup_fails(make_env, monkeypatch):
    env = await make_env()
    llm = env.service()

    async def boom(self):
        raise RuntimeError("cleanup failed")

    monkeypatch.setattr(ce.OpenAILLMService, "cleanup", boom)
    with pytest.raises(RuntimeError):
        await llm.cleanup()
    assert net.SLOTS.held() == 0


async def test_slot_is_given_back_before_the_client_is_closed(make_env):
    env = await make_env()
    llm = env.service()
    started = asyncio.Event()
    gate = asyncio.Event()

    finished = []

    async def slow_close():
        started.set()
        await gate.wait()
        finished.append(1)

    llm._client.close = slow_close
    task = asyncio.ensure_future(llm.release())
    await asyncio.wait_for(started.wait(), 5)
    assert net.SLOTS.held() == 0  # already back while the close is still running
    task.cancel()  # a cancellation while the client closes
    with pytest.raises(asyncio.CancelledError):
        await task
    assert net.SLOTS.held() == 0
    gate.set()
    await asyncio.sleep(0.05)
    assert finished == [1]  # shielded: the close ran to the end although the caller was cancelled


async def test_constructor_failure_gives_the_slot_back(make_env, monkeypatch):
    env = await make_env()

    def boom(self, **kw):
        raise RuntimeError("constructor failed")

    monkeypatch.setattr(ce.CustomEndpointLLMService, "__init__", boom)
    with pytest.raises(ValueError) as ei:
        env.service()
    assert str(ei.value) == "custom endpoint is not available" and ei.value.__cause__ is None
    assert net.SLOTS.held() == 0 and not net._PIPELINES


async def test_test_hooks_are_unset_in_production():
    assert ce.CustomEndpointLLMService._http_backend is None
    assert net.ValidatingTransport.test_ssl_context is None


async def test_close_errors_are_swallowed(make_env):
    env = await make_env()
    llm = env.service()
    llm._client.close = AsyncMock(side_effect=RuntimeError("close failed"))
    await llm.release()
    assert net.SLOTS.held() == 0


async def test_release_pipeline_releases_all_services_of_a_pipeline(make_env):
    env = await make_env()
    pid = "shared-pipeline"
    svcs = []
    for _ in range(3):
        llm, _ = run.create_llm_service("customendpoint.m", "", MESSAGES, [], pipeline_id=pid, customer_id=CUSTOMER,
                                         llm_base_url=env.stub.url, llm_base_key=KEY)
        svcs.append(llm)
    assert net.SLOTS.held() == 3
    await net.release_pipeline(pid)
    assert net.SLOTS.held() == 0 and all(s._client.is_closed() for s in svcs)
    await net.release_pipeline(pid)


async def test_per_customer_and_total_limits(make_env):
    env = await make_env()
    held = [env.service(customer=CUSTOMER) for _ in range(16)]
    with pytest.raises(ValueError) as ei:
        env.service(customer=CUSTOMER)
    assert str(ei.value) == "custom endpoint is not available"
    assert net.SLOTS.held() == 16
    others = []
    for i in range(3):
        c = str(uuid.uuid4())
        others += [env.service(customer=c) for _ in range(16)]
    assert net.SLOTS.held() == 64
    with pytest.raises(ValueError):
        env.service(customer=str(uuid.uuid4()))
    await held[0].release()
    env.service(customer=str(uuid.uuid4()))  # one slot freed: allowed again


@pytest.mark.parametrize("customer", [None, "", "not-a-uuid", "1234"])
async def test_missing_or_malformed_customer_is_refused_without_a_slot(make_env, customer):
    env = await make_env()
    with pytest.raises(ValueError) as ei:
        run.create_llm_service("customendpoint.m", "", MESSAGES, [], pipeline_id="p", customer_id=customer,
                               llm_base_url=env.stub.url, llm_base_key=KEY)
    assert str(ei.value) == "custom endpoint is not available" and ei.value.__cause__ is None
    assert net.SLOTS.held() == 0


@pytest.mark.parametrize("url", [None, "", "http://llm.example.test/v1", "https://user:pw@llm.example.test/v1", "ftp://x/y"])
async def test_runner_scheme_and_userinfo_check(make_env, url):
    with pytest.raises(ValueError) as ei:
        run.create_llm_service("customendpoint.m", "", MESSAGES, [], pipeline_id="p", customer_id=CUSTOMER,
                               llm_base_url=url, llm_base_key=KEY)
    assert str(ei.value) == "custom endpoint is not available"
    assert net.SLOTS.held() == 0


async def test_uppercase_customer_id_counts_as_the_same_customer(make_env):
    env = await make_env()
    for _ in range(16):
        env.service(customer=CUSTOMER.upper())
    with pytest.raises(ValueError):
        env.service(customer=CUSTOMER.replace("-", ""))


async def test_self_check_failure_fails_closed_and_gives_the_slot_back(make_env, monkeypatch):
    env = await make_env()
    monkeypatch.setattr(net, "build_http_client", lambda *a, **k: httpx.AsyncClient())  # not the validating client
    with pytest.raises(ValueError) as ei:
        env.service()
    assert str(ei.value) == "custom endpoint is not available"
    assert net.SLOTS.held() == 0 and not net._PIPELINES


async def test_relaxed_certificate_verification_fails_the_self_check(make_env, monkeypatch):
    env = await make_env()
    relaxed = ssl.create_default_context()
    relaxed.check_hostname = False
    relaxed.verify_mode = ssl.CERT_NONE
    monkeypatch.setattr(net.ValidatingTransport, "test_ssl_context", relaxed)
    with pytest.raises(ValueError):
        env.service()
    assert net.SLOTS.held() == 0


async def test_default_context_requires_the_certificate(make_env):
    env = await make_env(trust=False)
    llm = env.service()
    ctx = llm._client._client._transport._pool._ssl_context
    assert ctx.verify_mode == ssl.CERT_REQUIRED and ctx.check_hostname is True
    await llm.cleanup()


async def test_client_limits_of_the_session_client(make_env):
    env = await make_env()
    llm = env.service()
    c = llm._client
    assert c.max_retries == 0 and c.timeout == net.TIMEOUT
    pool = c._client._transport._pool
    assert pool._max_connections == 10 and pool._keepalive_expiry == 30.0 and pool._retries == 0
    assert c._client.follow_redirects is False and c._client.trust_env is False
    await llm.cleanup()


# --- a failed /run releases what the request created ----------------------------------------------------------------------


async def test_team_above_the_per_customer_limit_fails_with_400_and_gives_everything_back(make_env, monkeypatch):
    import main

    env = await make_env()
    members = [{"id": "m%d" % i, "name": "m", "ai": {
        "engine_model": "x.y", "engine_key": LEGACY, "llm_type": "customendpoint.m",
        "llm_base_url": env.stub.url, "llm_base_key": KEY}} for i in range(17)]
    req = main.PipelineRequest(id="team-1", customer_id=CUSTOMER, resolved_team={"id": "t", "start_member_id": "m0", "members": members})
    with pytest.raises(main.HTTPException) as ei:
        await main.run_pipeline_endpoint(req)
    assert ei.value.status_code == 400 and ei.value.detail == "custom endpoint is not available"
    assert net.SLOTS.held() == 0 and not net.pipeline_registered("team-1")  # the 16 earlier members were released


async def test_rejected_team_member_gives_back_earlier_members(make_env):
    import main

    env = await make_env()
    good = {"engine_model": "x.y", "engine_key": "", "llm_type": "customendpoint.m", "llm_base_url": env.stub.url, "llm_base_key": KEY}
    bad = dict(good, llm_type="")
    members = [{"id": "a", "name": "a", "ai": good}, {"id": "b", "name": "b", "ai": good}, {"id": "c", "name": "c", "ai": bad}]
    req = main.PipelineRequest(id="team-2", customer_id=CUSTOMER, resolved_team={"id": "t", "start_member_id": "a", "members": members})
    with pytest.raises(main.HTTPException) as ei:
        await main.run_pipeline_endpoint(req)
    assert ei.value.status_code == 400 and net.SLOTS.held() == 0


async def test_duplicate_pipeline_id_is_refused_and_the_running_pipeline_keeps_its_slots(make_env):
    import main

    env = await make_env()
    run_ctx = {"type": "single"}
    req = main.PipelineRequest(id="dup-1", llm_type="customendpoint.m", customer_id=CUSTOMER, llm_base_url=env.stub.url, llm_base_key=KEY)

    async def fake_init(id, llm_type, *a, **k):
        llm, _ = run.create_llm_service(llm_type, "", [], [], pipeline_id=id, customer_id=k["customer_id"],
                                        llm_base_url=k["llm_base_url"], llm_base_key=k["llm_base_key"])
        return run_ctx

    main.init_pipeline, orig_init = fake_init, main.init_pipeline
    main.execute_pipeline, orig_exec = AsyncMock(), main.execute_pipeline
    try:
        assert (await main.run_pipeline_endpoint(req))["status"] == "ok"
        assert net.SLOTS.held() == 1
        with pytest.raises(main.HTTPException) as ei:
            await main.run_pipeline_endpoint(req)
        assert ei.value.status_code == 409
        assert net.SLOTS.held() == 1  # the refused request did not release the running pipeline's slot
        await asyncio.sleep(0)
        await net.release_pipeline("dup-1")
        assert net.SLOTS.held() == 0
    finally:
        main.init_pipeline, main.execute_pipeline = orig_init, orig_exec
