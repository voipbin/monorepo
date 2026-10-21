"""Network guard for customer specified OpenAI-compatible endpoints (VOIP-1599 Layer 2).

This module does NOT import pipecat, so the unit tests can import it directly
(scripts/pipecat/conftest.py replaces pipecat with mocks). The pipecat subclass
that uses it is custom_endpoint.py. Design: docs/plans/2026-10-16-custom-endpoint-layer2-design.md
section 3.8.

Pinned against httpx 0.28.1, httpcore 1.0.9, h11 0.16.0, anyio 4.14.2 and
openai 2.15.0 (requirements.txt, pyproject.toml, uv.lock). The code relies on
private attributes of httpx and httpcore (the pool of httpx.AsyncHTTPTransport,
the pool's ssl context and network backend), so verify_openai_client() checks
the result at runtime and the caller fails closed when a check does not hold.

Every message raised for the caller of a custom endpoint session is a fixed
text: the URL, the key and the response body of the customer's server must not
reach an exception text, a log line or a response.
"""

import asyncio
import functools
import ipaddress
import re
import socket
import ssl
import threading
import uuid
from concurrent.futures import ThreadPoolExecutor
from urllib.parse import urlsplit

import httpcore
import httpx
import openai
from httpcore import AsyncNetworkBackend, AsyncNetworkStream

# --- Fixed texts and names --------------------------------------------------

# The only text a caller of a custom endpoint session ever sees (ValueError detail,
# HTTP detail, log line).
FIXED_ERROR = "custom endpoint is not available"
SERVICE_NAME = "customendpoint"

# --- Client settings (constants, not flags) ----------------------------------

CONNECT_TIMEOUT = 5.0  # shared by every address of one answer
READ_TIMEOUT = 30.0
WRITE_TIMEOUT = 10.0
POOL_TIMEOUT = 5.0
TIMEOUT = httpx.Timeout(connect=CONNECT_TIMEOUT, read=READ_TIMEOUT, write=WRITE_TIMEOUT, pool=POOL_TIMEOUT)
MAX_CONNECTIONS = 10
KEEPALIVE_EXPIRY = 30.0
RETRIES = 0
RESOLVE_TIMEOUT = 5.0

# Wire guard / transport guard (R12, R15). Working assumptions: completions for a
# voice agent are small.
READ_CHUNK_MAX = 4096
WIRE_BYTES_MAX = 4 * 1024 * 1024
TOTAL_DEADLINE_SECS = 120.0

# Name resolution (R24).
LOOKUP_WORKERS = 8
LOOKUPS_PER_CUSTOMER = 2

# Slots (R18), per runner instance. Provisional, confirmed by the measurement of
# the pull request (design 3.8 and section 9).
SLOTS_PER_CUSTOMER = 16
SLOTS_TOTAL = 64


class AddressBlocked(httpcore.ConnectError):
    """A resolved address is in the block table (or a unix socket was requested).

    Subclasses httpcore.ConnectError so httpx maps it to httpx.ConnectError and
    the OpenAI SDK reports APIConnectionError ("Connection error.").
    """


class WireLimitExceeded(httpcore.ReadError):
    """More bytes than WIRE_BYTES_MAX were received for one request."""


class SelfCheckError(RuntimeError):
    """The built client is not the validating one (dependency drift, R8)."""


# --- Block table (appendix A, parity with bin-ai-manager endpoint_block.go) --

V4_BLOCKS = (
    "0.0.0.0/8",
    "10.0.0.0/8",
    "172.16.0.0/12",
    "192.168.0.0/16",
    "127.0.0.0/8",
    "169.254.0.0/16",
    "100.64.0.0/10",
    "198.18.0.0/15",
    "192.0.0.0/24",
    "224.0.0.0/4",
    "240.0.0.0/4",
    "192.0.2.0/24",
    "198.51.100.0/24",
    "203.0.113.0/24",
    "192.88.99.0/24",
    "192.175.48.0/24",
)
V6_BLOCKS = (
    "::/128",
    "::1/128",
    "fc00::/7",
    "fe80::/10",
    "ff00::/8",
    "64:ff9b:1::/48",
    "100::/64",
    "::/96",
    "::ffff:0:0:0/96",
    "fec0::/10",
    "2001::/32",
    "2001:db8::/32",
    "2001:2::/48",
    "3fff::/20",
)
# 64:ff9b::/96 is not refused as such: the embedded IPv4 decides. The same holds
# for 2002::/16 (6to4) and for IPv4-mapped addresses.
NAT64_BLOCKS = ("64:ff9b::/96",)

_V4_NETS = tuple(ipaddress.ip_network(n) for n in V4_BLOCKS)
_V6_NETS = tuple(ipaddress.ip_network(n) for n in V6_BLOCKS)
_NAT64_NETS = tuple(ipaddress.ip_network(n) for n in NAT64_BLOCKS)


def _blocked_v4(a: ipaddress.IPv4Address) -> bool:
    return any(a in n for n in _V4_NETS)


def is_blocked(addr) -> bool:
    """True when the connect target must be refused.

    Steps: anything unparseable is blocked; a zone id is dropped (IPv6 only, a
    zone on an IPv4 address or an empty zone does not parse in the Go validator
    either); an IPv4-mapped address, 64:ff9b::/96 and 2002::/16 are decided by
    the embedded IPv4; everything else by the v4 or v6 table.
    """
    if not isinstance(addr, str):
        return True
    text = addr
    zoned = False
    if "%" in text:
        text, zone = text.split("%", 1)
        if not zone:
            return True
        zoned = True
    try:
        a = ipaddress.ip_address(text)
    except ValueError:
        return True
    if a.version == 4:
        return True if zoned else _blocked_v4(a)
    if a.ipv4_mapped is not None:
        return _blocked_v4(a.ipv4_mapped)
    if any(a in n for n in _NAT64_NETS):
        return _blocked_v4(ipaddress.IPv4Address(int(a) & 0xFFFFFFFF))
    if a.sixtofour is not None:
        return _blocked_v4(a.sixtofour)
    return any(a in n for n in _V6_NETS)


# --- Name resolution on a dedicated executor (R24) ---------------------------

_getaddrinfo = socket.getaddrinfo  # test seam
_EXECUTOR = ThreadPoolExecutor(max_workers=LOOKUP_WORKERS, thread_name_prefix="custom-endpoint-dns")
_LOOKUP_SEM = threading.Semaphore(LOOKUP_WORKERS)
_LOOKUP_LOCK = threading.Lock()
_LOOKUPS: dict = {}  # customer id -> lookups in flight


def _take_lookup(customer_id: str) -> bool:
    """Take one worker and one per customer lookup without blocking."""
    with _LOOKUP_LOCK:
        if _LOOKUPS.get(customer_id, 0) >= LOOKUPS_PER_CUSTOMER:
            return False
        if not _LOOKUP_SEM.acquire(blocking=False):
            return False
        _LOOKUPS[customer_id] = _LOOKUPS.get(customer_id, 0) + 1
        return True


def _give_lookup(customer_id: str) -> None:
    with _LOOKUP_LOCK:
        left = _LOOKUPS.get(customer_id, 0) - 1
        if left > 0:
            _LOOKUPS[customer_id] = left
        else:
            _LOOKUPS.pop(customer_id, None)  # an entry at zero is deleted
        _LOOKUP_SEM.release()


class _Once:
    """Call fn at most once; thread safe (the thread and the event loop both call it)."""

    def __init__(self, fn):
        self._fn = fn
        self._lock = threading.Lock()
        self._done = False

    def __call__(self):
        with self._lock:
            if self._done:
                return
            self._done = True
        self._fn()


def _lookup_worker(name: str, port: int) -> list:
    infos = _getaddrinfo(name, port, type=socket.SOCK_STREAM)
    out = []
    for info in infos:
        a = info[4][0]
        if a not in out:
            out.append(a)
    return out


def _ip_literal(host: str):
    try:
        return ipaddress.ip_address(host)
    except ValueError:
        return None


async def resolve_host(host: str, port: int, customer_id: str = "") -> list:
    """Return the addresses of host (no cache, no single flight).

    An IP literal is parsed and never looked up. A name is looked up as an
    absolute name (trailing dot) so resolver search lists and legacy IPv4
    spellings such as 127.1 cannot be expanded. The lookup runs on the dedicated
    executor; a lookup that cannot take a worker or the per customer share fails
    closed instead of queueing. Both are given back by the done callback of the
    concurrent future: when the thread ends, or at once when a lookup that no
    thread has started yet is cancelled. They are NOT given back when the
    awaiting coroutine times out (the thread is still busy).
    """
    if _ip_literal(host) is not None:
        return [host]
    name = host if host.endswith(".") else host + "."
    if not _take_lookup(customer_id):
        raise httpcore.ConnectError("name lookup limit reached")
    give = _Once(lambda: _give_lookup(customer_id))
    try:
        cf = _EXECUTOR.submit(_lookup_worker, name, port)
        cf.add_done_callback(lambda _f: give())
    except BaseException:
        give()
        raise
    try:
        return await asyncio.wait_for(asyncio.wrap_future(cf), RESOLVE_TIMEOUT)
    except (asyncio.TimeoutError, OSError):
        raise httpcore.ConnectError("name lookup failed") from None


# --- Wire guard (R12, R15) ----------------------------------------------------


class WireGuardStream(AsyncNetworkStream):
    """Wraps the network stream: small reads, loop yield, per request byte cap.

    Counts every byte received for the current request (status line, interim 1xx
    responses, headers, chunk framing and body). The counter is reset by write(),
    which starts a request on a pooled connection (a server cannot cause a client
    write).
    """

    def __init__(self, inner: AsyncNetworkStream):
        self._inner = inner
        self._received = 0

    async def read(self, max_bytes: int, timeout=None) -> bytes:
        data = await self._inner.read(min(max_bytes, READ_CHUNK_MAX), timeout=timeout)
        self._received += len(data)
        if self._received > WIRE_BYTES_MAX:
            raise WireLimitExceeded("response too large")
        await asyncio.sleep(0)
        return data

    async def write(self, buffer: bytes, timeout=None) -> None:
        self._received = 0
        await self._inner.write(buffer, timeout=timeout)

    async def aclose(self) -> None:
        await self._inner.aclose()

    async def start_tls(self, ssl_context, server_hostname=None, timeout=None):
        tls = await self._inner.start_tls(ssl_context, server_hostname=server_hostname, timeout=timeout)
        return WireGuardStream(tls)

    def get_extra_info(self, info: str):
        return self._inner.get_extra_info(info)


class ValidatingBackend(AsyncNetworkBackend):
    """Resolve, validate EVERY address, connect to a validated IP literal.

    httpcore calls start_tls(server_hostname=<origin host>) on the returned
    stream, so SNI and certificate verification keep using the original name.
    """

    def __init__(self, customer_id: str = "", inner=None, resolver=None):
        if inner is None:
            from httpcore._backends.auto import AutoBackend

            inner = AutoBackend()
        self._inner = inner
        self._customer_id = customer_id
        self._resolver = resolver

    async def _resolve(self, host, port):
        try:
            if self._resolver is not None:
                return await self._resolver(host, port)
            return await resolve_host(host, port, self._customer_id)
        except httpcore.ConnectError:
            raise
        except Exception:
            raise httpcore.ConnectError("name lookup failed") from None

    async def connect_tcp(self, host, port, timeout=None, local_address=None, socket_options=None):
        addrs = await self._resolve(host, port)
        if not addrs:
            raise httpcore.ConnectError("no address")
        for a in addrs:  # any blocked answer refuses the host
            if is_blocked(a):
                raise AddressBlocked("address not allowed")
        loop = asyncio.get_running_loop()
        budget = CONNECT_TIMEOUT if timeout is None else min(timeout, CONNECT_TIMEOUT)
        deadline = loop.time() + budget  # one deadline for all addresses
        last = None
        for a in addrs:
            remaining = deadline - loop.time()
            if remaining <= 0:
                raise last if last is not None else httpcore.ConnectTimeout("connect deadline exceeded")
            try:
                stream = await asyncio.wait_for(
                    self._inner.connect_tcp(
                        a, port, timeout=remaining, local_address=local_address, socket_options=socket_options
                    ),
                    remaining,
                )
            except asyncio.TimeoutError:
                last = httpcore.ConnectTimeout("connect deadline exceeded")
                continue
            except (httpcore.ConnectError, httpcore.ConnectTimeout) as e:
                last = e
                continue
            return WireGuardStream(stream)
        raise last

    async def connect_unix_socket(self, path, timeout=None, socket_options=None):
        raise AddressBlocked("unix sockets not allowed")

    async def sleep(self, seconds):
        await self._inner.sleep(seconds)


# --- Transport guard (R12, R15) -----------------------------------------------


class _GuardedBody(httpx.AsyncByteStream):
    """Response body with the total deadline; transport errors become APIConnectionError.

    The OpenAI SDK wraps only a failure before the response headers; an error raised
    while a stream is read would reach pipecat as "Error during completion: " with an
    empty or odd text. A read timeout is left alone (pipecat reports a timeout).
    """

    def __init__(self, inner, deadline: float, request: httpx.Request):
        self._inner = inner
        self._deadline = deadline
        self._request = request

    async def __aiter__(self):
        it = self._inner.__aiter__()
        while True:
            try:
                async with asyncio.timeout_at(self._deadline):
                    chunk = await it.__anext__()
            except StopAsyncIteration:
                return
            except TimeoutError:
                raise httpx.ReadTimeout("response deadline exceeded", request=self._request) from None
            except httpx.TimeoutException:
                raise
            except httpx.TransportError as e:
                raise openai.APIConnectionError(request=self._request) from e
            yield chunk

    async def aclose(self) -> None:
        await self._inner.aclose()


@functools.lru_cache(maxsize=None)
def _default_ssl_context() -> ssl.SSLContext:
    """The production trust store: certifi only (trust_env=False ignores SSL_CERT_FILE and
    SSL_CERT_DIR), certificate required, host name checked.

    One context for the whole process: building one loads the CA bundle (about 0.7 MB of memory per
    session, measured), and an SSLContext that is only used for connecting is safe to share.
    """
    return httpx.create_ssl_context(verify=True, trust_env=False)


class ValidatingTransport(httpx.AsyncHTTPTransport):
    """httpx transport whose httpcore pool uses ValidatingBackend.

    httpx.AsyncHTTPTransport has no network_backend parameter, so the pool is
    swapped after construction (private attribute _pool; the versions are pinned and
    verify_openai_client() asserts that the swap took effect).

    Certificate verification and host name checking are not parameters. The only way
    to use another trust store is the class attribute test_ssl_context, which production
    code never sets; verify_openai_client() still requires CERT_REQUIRED and
    check_hostname.
    """

    test_ssl_context = None  # test hook; production code never sets it

    def __init__(self, backend: AsyncNetworkBackend):
        ctx = type(self).test_ssl_context
        if ctx is None:
            ctx = _default_ssl_context()
        super().__init__(verify=ctx, http2=False, trust_env=False)
        self._pool = httpcore.AsyncConnectionPool(
            ssl_context=ctx,
            max_connections=MAX_CONNECTIONS,
            max_keepalive_connections=MAX_CONNECTIONS,
            keepalive_expiry=KEEPALIVE_EXPIRY,
            http1=True,
            http2=False,
            retries=RETRIES,
            network_backend=backend,
        )

    async def handle_async_request(self, request: httpx.Request) -> httpx.Response:
        # httpx would decompress gzip/deflate/br: a 1 MiB gzip bomb expands to ~2 GB.
        request.headers["Accept-Encoding"] = "identity"
        deadline = asyncio.get_running_loop().time() + TOTAL_DEADLINE_SECS
        try:
            async with asyncio.timeout_at(deadline):  # header phase (endless 1xx)
                response = await super().handle_async_request(request)
        except TimeoutError:
            raise httpx.ReadTimeout("response deadline exceeded", request=request) from None
        for value in response.headers.get_list("content-encoding"):
            for token in value.split(","):
                if token.strip().lower() not in ("", "identity"):
                    try:
                        await response.aclose()
                    except Exception:
                        pass
                    raise httpx.ReadError("unexpected content encoding", request=request)
        return httpx.Response(
            status_code=response.status_code,
            headers=response.headers,
            stream=_GuardedBody(response.stream, deadline, request),
            extensions=response.extensions,
        )


def build_http_client(customer_id: str = "", backend=None) -> httpx.AsyncClient:
    """The only HTTP client a custom endpoint uses.

    Verification is not a parameter. follow_redirects=False (the SDK default client
    follows), trust_env=False (no proxy variables, no SSL_CERT_FILE, no netrc).
    """
    if backend is None:
        backend = ValidatingBackend(customer_id)
    return httpx.AsyncClient(
        transport=ValidatingTransport(backend),
        timeout=TIMEOUT,
        follow_redirects=False,
        trust_env=False,
        http2=False,
    )


def build_openai_client(*, api_key, base_url, customer_id: str = "", backend=None) -> openai.AsyncOpenAI:
    """AsyncOpenAI for a customer endpoint: the validating http client and nothing from the environment.

    api_key is the stored key or "" and is never None: an empty key sends no Authorization header
    and does not fall back to OPENAI_API_KEY. organization and project are None and their headers are
    Omit(): the SDK would otherwise send OPENAI_ORG_ID / OPENAI_PROJECT_ID of the operator.
    max_retries=0 (the SDK default of 2 would triple every attempt).
    """
    return openai.AsyncOpenAI(
        api_key=api_key if isinstance(api_key, str) else "",
        base_url=base_url,
        organization=None,
        project=None,
        default_headers={"OpenAI-Organization": openai.Omit(), "OpenAI-Project": openai.Omit()},
        http_client=build_http_client(customer_id, backend),
        max_retries=0,
        timeout=TIMEOUT,
    )


def verify_openai_client(client, expected_base_url: str = None) -> None:
    """Self-check (R8): raise SelfCheckError unless the client is the validating one."""

    def need(cond):
        if not cond:
            raise SelfCheckError("client self-check failed")

    need(isinstance(client, openai.AsyncOpenAI))
    need(client.max_retries == 0)
    need(isinstance(client.api_key, str))  # never None: the SDK would read OPENAI_API_KEY
    custom = getattr(client, "_custom_headers", None) or {}
    need(isinstance(custom.get("OpenAI-Organization"), openai.Omit))
    need(isinstance(custom.get("OpenAI-Project"), openai.Omit))
    need(client.timeout == TIMEOUT)
    if expected_base_url is not None:
        need(str(client.base_url).rstrip("/") == expected_base_url.rstrip("/"))
    http = client._client
    need(isinstance(http, httpx.AsyncClient))
    need(http.follow_redirects is False)
    need(http.trust_env is False)
    need(not http._mounts)  # no proxy mounts
    need(http.timeout == TIMEOUT)
    transport = http._transport
    need(isinstance(transport, ValidatingTransport))
    pool = transport._pool
    need(isinstance(pool, httpcore.AsyncConnectionPool))
    need(isinstance(pool._network_backend, ValidatingBackend))
    need(pool._max_connections == MAX_CONNECTIONS)
    need(pool._keepalive_expiry == KEEPALIVE_EXPIRY)
    need(pool._retries == RETRIES)
    need(pool._http2 is False)
    ctx = pool._ssl_context
    need(isinstance(ctx, ssl.SSLContext))
    need(ctx.verify_mode == ssl.CERT_REQUIRED)
    need(ctx.check_hostname is True)


# --- Request level checks (the runner takes requests that did not pass Go) ----

_UUID_RE = re.compile(r"^[0-9a-fA-F-]{32,36}$")


def parse_customer_id(value) -> str:
    """Canonical lower case UUID string, or ValueError(FIXED_ERROR).

    Canonical form so that case or hyphen variants of one customer cannot bypass the
    per customer limits.
    """
    if not isinstance(value, str) or not _UUID_RE.match(value):
        raise ValueError(FIXED_ERROR)
    try:
        parsed = uuid.UUID(value)
    except ValueError:
        raise ValueError(FIXED_ERROR) from None
    if parsed.int == 0:
        # The nil UUID would put every caller that sends it into one bucket.
        raise ValueError(FIXED_ERROR)
    return str(parsed)


def check_runner_base_url(value) -> str:
    """https only, no userinfo, a host (design 3.8); no port policy.

    Also refuses control characters, spaces, backslashes and non-ASCII (the parsers
    of urlsplit and httpx must agree on what the host is).
    """
    if not isinstance(value, str) or not value or len(value) > 2048:
        raise ValueError(FIXED_ERROR)
    if any(ord(c) <= 0x20 or ord(c) >= 0x7F or c == "\\" for c in value):
        raise ValueError(FIXED_ERROR)
    try:
        parts = urlsplit(value)
        parts.port  # raises ValueError on an invalid port
    except ValueError:
        raise ValueError(FIXED_ERROR) from None
    if parts.scheme != "https" or not parts.hostname or "@" in parts.netloc:
        raise ValueError(FIXED_ERROR)
    return value


def is_custom_endpoint_type(llm_type) -> bool:
    """True for a type or engine model whose service name is customendpoint."""
    if not isinstance(llm_type, str):
        return False
    head = llm_type.replace(":", ".", 1).split(".", 1)[0]
    return head.lower() == SERVICE_NAME


_COMPLETION_PREFIX = "Error during completion: "
_STATUS_RE = re.compile(r"^Error code: (\d{3})(?!\d)")
_KEPT_TEXTS = ("Connection error.", "Request timed out.")


def sanitize_error_text(text) -> str:
    """Drop the response body of the customer's server from a pipecat error text.

    Keeps the fixed phrases the Go classifier needs (bin-pipecat-manager
    pipelineerror.go): "LLM completion timeout", "Error during completion:
    Connection error.", "...: Request timed out." and the status phrase "Error code:
    NNN". Everything else becomes a fixed text.
    """
    if not isinstance(text, str):
        return _COMPLETION_PREFIX + "error"
    if text == "LLM completion timeout":
        return text
    if text.startswith(_COMPLETION_PREFIX):
        rest = text[len(_COMPLETION_PREFIX) :]
        if rest in _KEPT_TEXTS:
            return text
        m = _STATUS_RE.match(rest)
        if m:
            return _COMPLETION_PREFIX + "Error code: " + m.group(1)
    return _COMPLETION_PREFIX + "error"


# --- Slots and pipeline registry (R18) ----------------------------------------


class SlotTable:
    """Custom endpoint services per customer and in total, per runner instance."""

    def __init__(self):
        self._lock = threading.Lock()
        self._per: dict = {}
        self._total = 0

    def acquire(self, customer_id: str) -> bool:
        with self._lock:
            if self._per.get(customer_id, 0) >= SLOTS_PER_CUSTOMER or self._total >= SLOTS_TOTAL:
                return False
            self._per[customer_id] = self._per.get(customer_id, 0) + 1
            self._total += 1
            return True

    def release(self, customer_id: str) -> None:
        with self._lock:
            left = self._per.get(customer_id, 0) - 1
            if left < 0:
                return  # never below zero
            if left == 0:
                self._per.pop(customer_id, None)
            else:
                self._per[customer_id] = left
            self._total -= 1

    def held(self, customer_id: str = None) -> int:
        with self._lock:
            return self._total if customer_id is None else self._per.get(customer_id, 0)


SLOTS = SlotTable()

_PIPELINES: dict = {}  # pipeline id -> custom services created for it
_RESERVED: set = set()  # pipeline ids a /run request has claimed (kept until release_pipeline)
_PIPELINES_LOCK = threading.Lock()


def reserve_pipeline(pipeline_id) -> bool:
    """Claim a pipeline id for a custom endpoint request; False if it is already claimed or in use.

    Synchronous (no await between check and claim), so two requests with the same id cannot both
    pass.
    """
    if not isinstance(pipeline_id, str) or not pipeline_id:
        return False
    with _PIPELINES_LOCK:
        if pipeline_id in _RESERVED or pipeline_id in _PIPELINES:
            return False
        _RESERVED.add(pipeline_id)
        return True


def pipeline_registered(pipeline_id) -> bool:
    with _PIPELINES_LOCK:
        return pipeline_id in _RESERVED or pipeline_id in _PIPELINES


def register_service(pipeline_id: str, service) -> None:
    with _PIPELINES_LOCK:
        _PIPELINES.setdefault(pipeline_id, []).append(service)


def unregister_service(pipeline_id, service) -> None:
    with _PIPELINES_LOCK:
        services = _PIPELINES.get(pipeline_id)
        if services is not None and service in services:
            services.remove(service)
            if not services:
                del _PIPELINES[pipeline_id]


async def release_pipeline(pipeline_id) -> None:
    """Release every custom service recorded for the pipeline and forget the id.

    The safety net of the cleanup chain (a sequence of awaits in which an earlier failure or
    cancellation skips the later steps). All slots are given back synchronously before the first
    await, so a cancellation cannot keep one.
    """
    with _PIPELINES_LOCK:
        services = _PIPELINES.pop(pipeline_id, None) or []
        _RESERVED.discard(pipeline_id)
    for service in services:
        service.release_slot()
    for service in services:
        try:
            await service.release()
        except Exception:
            pass
