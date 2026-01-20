.. _architecture-tutorial:

Architecture Tutorial
=====================

This tutorial provides practical examples of working with VoIPBIN's architecture. These examples help developers understand how to build integrations, debug issues, and optimize performance.

Tutorial 1: Understanding Request Flow
---------------------------------------

Learn how a simple API request flows through the entire system.

**Objective**: Trace a call creation request from client to database and back.

**Step 1: Make the API Request**

.. code::

    curl --location --request POST 'https://api.voipbin.net/v1.0/calls' \
      --header 'Authorization: Bearer <YOUR_AUTH_TOKEN>' \
      --header 'Content-Type: application/json' \
      --data-raw '{
        "source": {
          "type": "tel",
          "target": "+15551234567"
        },
        "destinations": [{
          "type": "tel",
          "target": "+15559876543"
        }]
      }'

**Step 2: Follow the Flow**

.. code::

    Component Flow:

    1. Request hits Load Balancer
       └─ Routes to API Gateway instance

    2. API Gateway (bin-api-manager)
       ├─ Extracts JWT from header
       ├─ Validates token signature
       ├─ Checks customer permissions
       ├─ Validates request body
       └─ Sends RabbitMQ RPC message

    3. RabbitMQ
       ├─ Receives message in bin-manager.call.request queue
       └─ Routes to available Call Manager instance

    4. Call Manager (bin-call-manager)
       ├─ Receives RPC message
       ├─ Validates business logic
       ├─ Checks billing balance (via RPC to Billing Manager)
       ├─ Creates call record in MySQL
       ├─ Sends request to RTC Manager for SIP setup
       ├─ Publishes call.created event to RabbitMQ
       └─ Returns response via RabbitMQ

    5. API Gateway receives response
       └─ Returns HTTP 201 with call details

**Step 3: Verify Each Layer**

Check logs at each layer:

.. code::

    # API Gateway logs
    [INFO] customer-123 POST /v1.0/calls authenticated
    [INFO] Sending RPC to bin-call-manager

    # Call Manager logs
    [INFO] Creating call for customer-123
    [INFO] Call call-789 created successfully
    [INFO] Publishing call.created event

    # Database
    SELECT * FROM calls WHERE id='call-789';
    → Verify record exists

    # RabbitMQ
    # Check message was published to call.events exchange

Tutorial 2: Working with Events
--------------------------------

Learn how to subscribe to and process system events.

**Objective**: Listen to call events and process them in real-time.

**Step 1: Subscribe via WebSocket**

.. code::

    import websocket
    import json

    def on_message(ws, message):
        data = json.loads(message)
        print(f"Received event: {data['event_type']}")
        print(f"Call ID: {data['data']['call_id']}")

        if data['event_type'] == 'call.created':
            handle_call_created(data['data'])
        elif data['event_type'] == 'call.answered':
            handle_call_answered(data['data'])
        elif data['event_type'] == 'call.ended':
            handle_call_ended(data['data'])

    def on_open(ws):
        # Subscribe to all call events for this customer
        subscription = {
            "type": "subscribe",
            "topics": [
                "customer_id:<YOUR_CUSTOMER_ID>:call:*"
            ]
        }
        ws.send(json.dumps(subscription))
        print("Subscribed to call events")

    token = "<YOUR_AUTH_TOKEN>"
    ws_url = f"wss://api.voipbin.net/v1.0/ws?token={token}"

    ws = websocket.WebSocketApp(
        ws_url,
        on_open=on_open,
        on_message=on_message
    )

    ws.run_forever()

**Step 2: Handle Different Event Types**

.. code::

    def handle_call_created(call_data):
        """Process new call"""
        print(f"New call: {call_data['id']}")

        # Update dashboard
        update_dashboard_call_count()

        # Log to analytics
        track_call_created(call_data)

    def handle_call_answered(call_data):
        """Process answered call"""
        print(f"Call answered: {call_data['id']}")

        # Start timer for billing
        start_call_timer(call_data['id'])

        # Notify agents
        notify_agents(call_data)

    def handle_call_ended(call_data):
        """Process ended call"""
        print(f"Call ended: {call_data['id']}")

        # Calculate duration
        duration = call_data['duration']

        # Store in analytics
        store_call_record(call_data)

**Step 3: Handle Connection Issues**

.. code::

    import time

    def on_error(ws, error):
        print(f"WebSocket error: {error}")

    def on_close(ws, close_status_code, close_msg):
        print("Connection closed, reconnecting in 5s...")
        time.sleep(5)
        reconnect()

    def reconnect():
        # Recreate WebSocket connection
        ws = websocket.WebSocketApp(
            ws_url,
            on_open=on_open,
            on_message=on_message,
            on_error=on_error,
            on_close=on_close
        )
        ws.run_forever()

Tutorial 3: Optimizing with Cache
----------------------------------

Learn how to use Redis caching to improve performance.

**Objective**: Implement cache-aside pattern for frequently accessed data.

**Step 1: Setup Cache Client**

.. code::

    import redis
    import json
    import requests

    # Initialize Redis client
    redis_client = redis.Redis(
        host='localhost',
        port=6379,
        db=0,
        decode_responses=True
    )

    # VoIPBIN API configuration
    API_BASE = "https://api.voipbin.net/v1.0"
    AUTH_TOKEN = "<YOUR_AUTH_TOKEN>"

**Step 2: Implement Cache-Aside Pattern**

.. code::

    def get_call(call_id):
        """Get call with cache-aside pattern"""

        # 1. Check cache first
        cache_key = f"call:{call_id}"
        cached_data = redis_client.get(cache_key)

        if cached_data:
            print("Cache HIT")
            return json.loads(cached_data)

        print("Cache MISS - fetching from API")

        # 2. Cache miss - fetch from API
        response = requests.get(
            f"{API_BASE}/calls/{call_id}",
            headers={"Authorization": f"Bearer {AUTH_TOKEN}"}
        )

        if response.status_code != 200:
            return None

        call_data = response.json()

        # 3. Store in cache for next time
        redis_client.setex(
            cache_key,
            300,  # 5 minutes TTL
            json.dumps(call_data)
        )

        return call_data

**Step 3: Invalidate Cache on Updates**

.. code::

    def update_call_status(call_id, new_status):
        """Update call status and invalidate cache"""

        # 1. Update via API
        response = requests.patch(
            f"{API_BASE}/calls/{call_id}",
            headers={"Authorization": f"Bearer {AUTH_TOKEN}"},
            json={"status": new_status}
        )

        if response.status_code != 200:
            return False

        # 2. Invalidate cache
        cache_key = f"call:{call_id}"
        redis_client.delete(cache_key)

        print(f"Cache invalidated for {call_id}")
        return True

**Step 4: Measure Cache Performance**

.. code::

    import time

    def measure_cache_performance():
        """Compare cached vs uncached performance"""

        call_id = "call-789"

        # First call (cache miss)
        start = time.time()
        get_call(call_id)
        uncached_time = time.time() - start
        print(f"Uncached request: {uncached_time:.3f}s")

        # Second call (cache hit)
        start = time.time()
        get_call(call_id)
        cached_time = time.time() - start
        print(f"Cached request: {cached_time:.3f}s")

        speedup = uncached_time / cached_time
        print(f"Speedup: {speedup:.1f}x faster")

    # Example output:
    # Cache MISS - fetching from API
    # Uncached request: 0.095s
    # Cache HIT
    # Cached request: 0.012s
    # Speedup: 7.9x faster

Tutorial 4: Building a Webhook Handler
---------------------------------------

Learn how to receive and process webhooks from VoIPBIN.

**Objective**: Build a webhook endpoint that processes call events.

**Step 1: Create Webhook Server**

.. code::

    from flask import Flask, request, jsonify
    import hmac
    import hashlib

    app = Flask(__name__)
    WEBHOOK_SECRET = "<YOUR_WEBHOOK_SECRET>"

    @app.route('/webhooks/voipbin', methods=['POST'])
    def voipbin_webhook():
        # 1. Verify signature
        if not verify_signature(request):
            return jsonify({'error': 'Invalid signature'}), 403

        # 2. Parse payload
        payload = request.get_json()
        event_type = payload.get('event_type')

        # 3. Process event
        if event_type == 'call.created':
            handle_call_created(payload['data'])
        elif event_type == 'call.ended':
            handle_call_ended(payload['data'])
        elif event_type == 'sms.sent':
            handle_sms_sent(payload['data'])

        # 4. Acknowledge receipt
        return jsonify({'status': 'received'}), 200

    def verify_signature(request):
        """Verify webhook signature for security"""
        signature = request.headers.get('X-VoIPBIN-Signature')
        if not signature:
            return False

        # Compute expected signature
        expected = hmac.new(
            WEBHOOK_SECRET.encode(),
            request.data,
            hashlib.sha256
        ).hexdigest()

        return hmac.compare_digest(signature, expected)

**Step 2: Handle Events with Retry**

.. code::

    import sqlite3
    from datetime import datetime

    def handle_call_ended(call_data):
        """Process call ended event with database storage"""
        try:
            # Store in database
            conn = sqlite3.connect('calls.db')
            cursor = conn.cursor()

            cursor.execute('''
                INSERT INTO call_history
                (call_id, duration, customer_id, cost, timestamp)
                VALUES (?, ?, ?, ?, ?)
            ''', (
                call_data['id'],
                call_data['duration'],
                call_data['customer_id'],
                calculate_cost(call_data['duration']),
                datetime.now()
            ))

            conn.commit()
            conn.close()

            # Send notification
            send_notification(call_data)

            return True

        except Exception as e:
            print(f"Error processing call ended: {e}")
            # Will be retried by VoIPBIN
            raise

**Step 3: Test Webhook Locally**

.. code::

    # Use ngrok to expose local server
    # ngrok http 5000

    # Configure webhook in VoIPBIN:
    curl --location --request POST \
      'https://api.voipbin.net/v1.0/webhooks' \
      --header 'Authorization: Bearer <YOUR_AUTH_TOKEN>' \
      --header 'Content-Type: application/json' \
      --data-raw '{
        "name": "My Webhook",
        "uri": "https://abc123.ngrok.io/webhooks/voipbin",
        "method": "POST",
        "event_types": [
          "call.created",
          "call.ended",
          "sms.sent"
        ]
      }'

**Step 4: Monitor Webhook Health**

.. code::

    from collections import defaultdict
    from datetime import datetime, timedelta

    webhook_stats = defaultdict(int)
    last_received = {}

    @app.route('/webhooks/voipbin', methods=['POST'])
    def voipbin_webhook():
        payload = request.get_json()
        event_type = payload.get('event_type')

        # Track statistics
        webhook_stats[event_type] += 1
        last_received[event_type] = datetime.now()

        # Process event...
        return jsonify({'status': 'received'}), 200

    @app.route('/webhooks/stats', methods=['GET'])
    def webhook_stats_endpoint():
        """Endpoint to view webhook statistics"""
        stats = {
            'total_received': sum(webhook_stats.values()),
            'by_type': dict(webhook_stats),
            'last_received': {
                k: v.isoformat()
                for k, v in last_received.items()
            }
        }
        return jsonify(stats)

Tutorial 5: Implementing Retry Logic
-------------------------------------

Learn how to implement robust retry logic for API calls.

**Objective**: Build a resilient API client with exponential backoff.

**Step 1: Basic Retry with Exponential Backoff**

.. code::

    import time
    import requests
    from requests.adapters import HTTPAdapter
    from requests.packages.urllib3.util.retry import Retry

    def create_api_client():
        """Create API client with retry logic"""

        session = requests.Session()

        # Configure retry strategy
        retry_strategy = Retry(
            total=5,  # Max retries
            backoff_factor=1,  # Wait 1, 2, 4, 8, 16 seconds
            status_forcelist=[500, 502, 503, 504],  # Retry on these status codes
            allowed_methods=["GET", "POST", "PUT", "DELETE"]
        )

        adapter = HTTPAdapter(max_retries=retry_strategy)
        session.mount("https://", adapter)
        session.mount("http://", adapter)

        return session

    # Usage
    api_client = create_api_client()

    response = api_client.post(
        "https://api.voipbin.net/v1.0/calls",
        headers={"Authorization": f"Bearer <YOUR_AUTH_TOKEN>"},
        json={
            "source": {"type": "tel", "target": "+15551234567"},
            "destinations": [{"type": "tel", "target": "+15559876543"}]
        }
    )

**Step 2: Custom Retry Logic with Logging**

.. code::

    import logging

    logging.basicConfig(level=logging.INFO)
    logger = logging.getLogger(__name__)

    def make_api_call_with_retry(url, method="GET", max_retries=5, **kwargs):
        """Make API call with custom retry logic and logging"""

        for attempt in range(max_retries):
            try:
                logger.info(f"Attempt {attempt + 1}/{max_retries}: {method} {url}")

                response = requests.request(method, url, **kwargs)

                # Check for success
                if 200 <= response.status_code < 300:
                    logger.info(f"Success on attempt {attempt + 1}")
                    return response

                # Check if we should retry
                if response.status_code in [500, 502, 503, 504]:
                    # Transient error - retry
                    if attempt < max_retries - 1:
                        wait_time = 2 ** attempt  # Exponential backoff
                        logger.warning(
                            f"Retryable error {response.status_code}. "
                            f"Waiting {wait_time}s before retry."
                        )
                        time.sleep(wait_time)
                        continue

                # Permanent error or max retries - don't retry
                logger.error(f"Failed with status {response.status_code}")
                return response

            except requests.exceptions.RequestException as e:
                # Network error - retry
                if attempt < max_retries - 1:
                    wait_time = 2 ** attempt
                    logger.warning(f"Network error: {e}. Retrying in {wait_time}s...")
                    time.sleep(wait_time)
                else:
                    logger.error(f"Max retries exceeded. Last error: {e}")
                    raise

        return None

**Step 3: Circuit Breaker Pattern**

.. code::

    from enum import Enum
    from datetime import datetime, timedelta

    class CircuitState(Enum):
        CLOSED = "closed"  # Normal operation
        OPEN = "open"      # Failing, reject requests
        HALF_OPEN = "half_open"  # Testing if recovered

    class CircuitBreaker:
        def __init__(self, failure_threshold=5, timeout=60):
            self.failure_threshold = failure_threshold
            self.timeout = timeout  # seconds
            self.failure_count = 0
            self.state = CircuitState.CLOSED
            self.last_failure_time = None

        def call(self, func, *args, **kwargs):
            """Execute function with circuit breaker"""

            # Check if circuit is open
            if self.state == CircuitState.OPEN:
                # Check if timeout has elapsed
                if datetime.now() - self.last_failure_time > timedelta(seconds=self.timeout):
                    self.state = CircuitState.HALF_OPEN
                    print("Circuit HALF_OPEN - testing...")
                else:
                    raise Exception("Circuit breaker is OPEN")

            try:
                # Execute function
                result = func(*args, **kwargs)

                # Success - reset if half-open
                if self.state == CircuitState.HALF_OPEN:
                    self.reset()
                    print("Circuit CLOSED - service recovered")

                return result

            except Exception as e:
                self.record_failure()
                raise

        def record_failure(self):
            """Record failure and potentially open circuit"""
            self.failure_count += 1
            self.last_failure_time = datetime.now()

            if self.failure_count >= self.failure_threshold:
                self.state = CircuitState.OPEN
                print(f"Circuit OPEN - too many failures ({self.failure_count})")

        def reset(self):
            """Reset circuit breaker"""
            self.failure_count = 0
            self.state = CircuitState.CLOSED

    # Usage
    breaker = CircuitBreaker(failure_threshold=5, timeout=60)

    def make_api_call():
        response = requests.get(
            "https://api.voipbin.net/v1.0/calls",
            headers={"Authorization": f"Bearer <YOUR_AUTH_TOKEN>"}
        )
        return response.json()

    # Make calls through circuit breaker
    try:
        result = breaker.call(make_api_call)
        print(f"Success: {result}")
    except Exception as e:
        print(f"Failed: {e}")

Tutorial 6: Performance Optimization
-------------------------------------

Learn how to optimize your VoIPBIN integration for performance.

**Objective**: Reduce latency and improve throughput.

**Technique 1: Batch Operations**

.. code::

    # SLOW: Create calls one by one
    def create_calls_sequential(call_list):
        results = []
        for call_data in call_list:
            response = requests.post(
                "https://api.voipbin.net/v1.0/calls",
                headers={"Authorization": f"Bearer <YOUR_AUTH_TOKEN>"},
                json=call_data
            )
            results.append(response.json())
        return results

    # Time for 100 calls: ~10 seconds

    # FAST: Create calls in parallel
    import concurrent.futures

    def create_call(call_data):
        response = requests.post(
            "https://api.voipbin.net/v1.0/calls",
            headers={"Authorization": f"Bearer <YOUR_AUTH_TOKEN>"},
            json=call_data
        )
        return response.json()

    def create_calls_parallel(call_list):
        with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
            results = list(executor.map(create_call, call_list))
        return results

    # Time for 100 calls: ~2 seconds (5x faster)

**Technique 2: Connection Pooling**

.. code::

    from requests.adapters import HTTPAdapter
    from requests.packages.urllib3.util.pool import HTTPConnectionPool

    # Configure connection pool
    session = requests.Session()

    adapter = HTTPAdapter(
        pool_connections=10,  # Number of connection pools
        pool_maxsize=20,      # Max connections per pool
        max_retries=3,
        pool_block=False
    )

    session.mount("https://", adapter)

    # Reuse session for all requests
    for i in range(100):
        response = session.get(
            f"https://api.voipbin.net/v1.0/calls/{call_id}",
            headers={"Authorization": f"Bearer <YOUR_AUTH_TOKEN>"}
        )

    # Connection pooling saves ~20ms per request

**Technique 3: Smart Caching**

.. code::

    from functools import lru_cache
    from datetime import datetime, timedelta

    class CachedAPIClient:
        def __init__(self):
            self.cache = {}
            self.cache_ttl = {}

        def get_with_cache(self, endpoint, ttl_seconds=300):
            """Get data with time-based caching"""

            # Check cache
            if endpoint in self.cache:
                cache_time = self.cache_ttl[endpoint]
                if datetime.now() - cache_time < timedelta(seconds=ttl_seconds):
                    print(f"Cache HIT: {endpoint}")
                    return self.cache[endpoint]

            # Cache miss - fetch from API
            print(f"Cache MISS: {endpoint}")
            response = requests.get(
                f"https://api.voipbin.net/v1.0{endpoint}",
                headers={"Authorization": f"Bearer <YOUR_AUTH_TOKEN>"}
            )

            data = response.json()

            # Store in cache
            self.cache[endpoint] = data
            self.cache_ttl[endpoint] = datetime.now()

            return data

    # Usage
    client = CachedAPIClient()

    # First call - cache miss
    calls = client.get_with_cache("/calls", ttl_seconds=60)

    # Second call within 60s - cache hit
    calls = client.get_with_cache("/calls", ttl_seconds=60)

**Technique 4: Pagination**

.. code::

    def get_all_calls_paginated(customer_id):
        """Efficiently fetch all calls using pagination"""

        all_calls = []
        page_size = 100
        offset = 0

        while True:
            response = requests.get(
                "https://api.voipbin.net/v1.0/calls",
                headers={"Authorization": f"Bearer <YOUR_AUTH_TOKEN>"},
                params={
                    "customer_id": customer_id,
                    "limit": page_size,
                    "offset": offset
                }
            )

            data = response.json()
            calls = data.get('result', [])

            if not calls:
                break

            all_calls.extend(calls)
            offset += page_size

            print(f"Fetched {len(all_calls)} calls so far...")

        return all_calls

Best Practices Summary
----------------------

**Authentication:**

* Store tokens securely
* Refresh tokens before expiration
* Handle 401 responses

**Error Handling:**

* Retry on 5xx errors with exponential backoff
* Don't retry on 4xx errors (except 429 rate limit)
* Implement circuit breakers for failing services
* Log all errors with context

**Performance:**

* Use connection pooling
* Implement caching with appropriate TTLs
* Use pagination for large datasets
* Make parallel requests when possible

**Monitoring:**

* Track API response times
* Monitor error rates
* Alert on circuit breaker opens
* Log all API calls

**Security:**

* Use HTTPS only
* Validate webhook signatures
* Store secrets in environment variables
* Rotate tokens regularly

Debugging Tips
--------------

**1. Use Correlation IDs**

.. code::

    import uuid

    correlation_id = str(uuid.uuid4())

    response = requests.post(
        "https://api.voipbin.net/v1.0/calls",
        headers={
            "Authorization": f"Bearer <YOUR_AUTH_TOKEN>",
            "X-Request-ID": correlation_id
        },
        json=call_data
    )

    print(f"Track this request: {correlation_id}")

**2. Enable Detailed Logging**

.. code::

    import logging
    import http.client as http_client

    http_client.HTTPConnection.debuglevel = 1
    logging.basicConfig(level=logging.DEBUG)
    requests_log = logging.getLogger("requests.packages.urllib3")
    requests_log.setLevel(logging.DEBUG)
    requests_log.propagate = True

**3. Test in Sandbox First**

.. code::

    # Use test credentials for development
    TEST_API_BASE = "https://sandbox-api.voipbin.net/v1.0"
    TEST_TOKEN = "<YOUR_TEST_TOKEN>"

    # Test your integration thoroughly before production

Further Reading
---------------

* :ref:`Backend Architecture <architecture-backend>` - Microservices design
* :ref:`Communication Patterns <architecture-communication>` - Messaging architecture
* :ref:`Data Architecture <architecture-data>` - Database and caching
* :ref:`System Flows <architecture-flow>` - Complete request flows
