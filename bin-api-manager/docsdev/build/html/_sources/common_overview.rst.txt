.. _common-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low -- These are shared data structures and conventions used across all VoIPBIN APIs. No API calls are specific to this section.
   * **Cost:** Free. Common structures are reference documentation only; no operations are performed.
   * **Async:** N/A. This section documents conventions, not API endpoints.

This section covers common data structures, patterns, and concepts used throughout the VoIPBIN API. Understanding these foundational elements will help you work more effectively with all VoIPBIN resources.

.. note:: **AI Implementation Hint**

   All VoIPBIN timestamps use the format ``YYYY-MM-DD HH:MM:SS.microseconds`` in UTC. A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` means the resource has **not** been deleted (sentinel value). When filtering by time ranges, use URL-encoded timestamps as query parameters (e.g., ``?page_token=2022-06-17%2006%3A06%3A14.948432``).

Common Data Structures
----------------------

Address Structure
+++++++++++++++++
The Address structure is used throughout VoIPBIN to represent communication endpoints, including phone numbers, SIP addresses, and extension numbers. See the detailed structure documentation at :ref:`here <common-struct-address-address>`.

Timestamp Format
++++++++++++++++
All timestamps in VoIPBIN follow the format ``YYYY-MM-DD HH:MM:SS.microseconds`` and are in UTC timezone unless otherwise specified.

Example: ``2022-05-01 15:10:38.785510878``

UUID Format
+++++++++++
VoIPBIN uses UUIDs (Universally Unique Identifiers) to identify resources. All resource IDs follow the standard UUID v4 format.

Example: ``d9d32881-12fd-4b19-a6b2-6d5b6b6acf76``

Empty UUID
++++++++++
The empty UUID ``00000000-0000-0000-0000-000000000000`` is used to represent null or unset references.

Common Patterns
---------------

Pagination
++++++++++
List endpoints support pagination using the ``next_page_token`` parameter. When a response includes a ``next_page_token`` field, pass this value in your next request to retrieve the next page of results.

**Response format:**

.. code::

    {
        "result": [
            { ... },
            { ... }
        ],
        "next_page_token": "2022-06-17 06:06:14.948432"
    }

**Fetching the next page:**

.. code::

    GET https://api.voipbin.net/v1.0/calls?page_token=2022-06-17%2006%3A06%3A14.948432

When ``next_page_token`` is empty or not present, you have reached the last page.

Filtering and Searching
++++++++++++++++++++++++
Many list endpoints support filtering and searching through query parameters. Common filter parameters include:

* ``status``: Filter by resource status
* ``customer_id``: Filter by customer
* ``tm_create``: Filter by creation time range

Check the specific endpoint documentation for available filter options.

Soft Deletion
+++++++++++++
VoIPBIN uses soft deletion for most resources. Deleted resources have their ``tm_delete`` timestamp set to the deletion time. Non-deleted resources have ``tm_delete`` set to ``9999-01-01 00:00:00.000000``.

**Non-deleted resource:**

.. code::

    "tm_delete": "9999-01-01 00:00:00.000000"

**Deleted resource:**

.. code::

    "tm_delete": "2024-03-15 10:30:00.000000"

HTTP Status Codes
-----------------

VoIPBIN API uses standard HTTP status codes to indicate success or failure.

Success Codes
+++++++++++++

.. list-table::
   :header-rows: 1

   * - Code
     - Description
   * - 200 OK
     - Request succeeded. Response contains requested data.
   * - 201 Created
     - Resource created successfully.
   * - 204 No Content
     - Request succeeded with no response body (e.g., DELETE).

Client Error Codes
++++++++++++++++++

.. list-table::
   :header-rows: 1

   * - Code
     - Description
   * - 400 Bad Request
     - Invalid request format or parameters.
   * - 401 Unauthorized
     - Missing or invalid authentication token.
   * - 403 Forbidden
     - Valid token but insufficient permissions.
   * - 404 Not Found
     - Resource does not exist.
   * - 409 Conflict
     - Resource state conflict (e.g., duplicate creation).
   * - 422 Unprocessable Entity
     - Valid format but semantic errors.
   * - 429 Too Many Requests
     - Rate limit exceeded. Retry after delay.

Server Error Codes
++++++++++++++++++

.. list-table::
   :header-rows: 1

   * - Code
     - Description
   * - 500 Internal Server Error
     - Unexpected server error.
   * - 502 Bad Gateway
     - Upstream service unavailable.
   * - 503 Service Unavailable
     - Service temporarily unavailable.

Error Responses
---------------

All 4xx and 5xx responses share a single canonical envelope shape. See
:ref:`error-response-envelope` in the RESTful API page for the field
definitions, status-to-HTTP mapping, and AI implementation hints.

The reason-code catalog grouped by originating manager is at
:doc:`restful_api_errors`.

Request/Response Formats
------------------------

All API requests and responses use JSON format.

**Request headers:**

.. code::

    Content-Type: application/json
    Authorization: Bearer <YOUR_ACCESS_TOKEN>

.. note:: **AI Implementation Hint**

   Besides the ``Authorization: Bearer <token>`` header, VoIPBIN also accepts the token as a ``token`` query parameter (e.g., ``?token=<token>``, used throughout this documentation for ``curl`` examples and required for the WebSocket endpoint since it cannot send custom headers) or as a ``token`` cookie.

**Single resource response:**

.. code::

    {
        "id": "d9d32881-12fd-4b19-a6b2-6d5b6b6acf76",
        "name": "Example Resource",
        "status": "active",
        "tm_create": "2022-05-01 15:10:38.785510",
        "tm_update": "2022-05-01 15:10:38.785510",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

**List response:**

.. code::

    {
        "result": [
            { ... },
            { ... }
        ],
        "next_page_token": "..."
    }

.. _rate-limiting:

Rate Limiting
-------------

VoIPBIN applies a per-client-IP rate limit to all API requests, enforced in three tiers:

* Unauthenticated ``/auth/*`` endpoints (signup, login/boot, password reset, etc. — see :ref:`Auth <auth-main>` for the full list): up to approximately 10 requests/second, burst of 20.
* Authenticated account-management endpoints (``/auth/unregister``, ``/auth/delegate``): up to approximately 10 requests/second, burst of 20.
* The rest of the authenticated ``/v1.0/*`` API surface (calls, messages, and the rest of the platform API): up to approximately 200 requests/second, burst of 400.

These figures are per-IP and per-server-instance, not an exact guaranteed ceiling — the effective limit for a given client can vary with backend scaling, so treat them as an order-of-magnitude guide rather than a fixed contract. When a limit is exceeded, the API returns a ``429 Too Many Requests`` response using the same canonical error envelope described in :ref:`error-response-envelope`, with reason code ``RATE_LIMIT_EXCEEDED``.

.. code::

    {
        "error": {
            "status": "RESOURCE_EXHAUSTED",
            "reason": "RATE_LIMIT_EXCEEDED",
            "message": "Too many requests. Please try again later."
        }
    }

.. note:: **AI Implementation Hint**

   The rate limit response does **not** include ``X-RateLimit-*`` or ``Retry-After`` headers -- there is no way to read the remaining quota or reset time from the response. Detect rate limiting purely from the ``429`` status code and ``RATE_LIMIT_EXCEEDED`` reason, and back off blindly (exponential backoff) rather than relying on a reset timestamp.

**Handling rate limits:**

1. Detect the ``429`` status code and ``RATE_LIMIT_EXCEEDED`` reason
2. Implement exponential backoff for retries (no reset time is provided)
3. Cache responses when possible to reduce API calls
