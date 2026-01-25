.. _common-overview:

Overview
========

This section covers common data structures, patterns, and concepts used throughout the VoIPBIN API. Understanding these foundational elements will help you work more effectively with all VoIPBIN resources.

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

    GET /v1.0/calls?page_token=2022-06-17%2006%3A06%3A14.948432

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

=========== =====================================================
Code        Description
=========== =====================================================
200 OK      Request succeeded. Response contains requested data.
201 Created Resource created successfully.
204 No Content Request succeeded with no response body (e.g., DELETE).
=========== =====================================================

Client Error Codes
++++++++++++++++++

=========== =====================================================
Code        Description
=========== =====================================================
400 Bad Request Invalid request format or parameters.
401 Unauthorized Missing or invalid authentication token.
403 Forbidden Valid token but insufficient permissions.
404 Not Found Resource does not exist.
409 Conflict Resource state conflict (e.g., duplicate creation).
422 Unprocessable Entity Valid format but semantic errors.
429 Too Many Requests Rate limit exceeded. Retry after delay.
=========== =====================================================

Server Error Codes
++++++++++++++++++

=========== =====================================================
Code        Description
=========== =====================================================
500 Internal Server Error Unexpected server error.
502 Bad Gateway Upstream service unavailable.
503 Service Unavailable Service temporarily unavailable.
=========== =====================================================

Error Response Format
---------------------

When an error occurs, the API returns a JSON response with error details:

.. code::

    {
        "error": {
            "code": "invalid_parameter",
            "message": "The 'destination' field is required",
            "field": "destination"
        }
    }

**Error fields:**

* ``code``: Machine-readable error code
* ``message``: Human-readable error description
* ``field``: (Optional) Field that caused the error

Common Error Codes
++++++++++++++++++

======================= =====================================================
Code                    Description
======================= =====================================================
invalid_parameter       Request parameter is invalid or missing.
resource_not_found      Requested resource does not exist.
permission_denied       Insufficient permissions for this operation.
duplicate_resource      Resource with same identifier already exists.
invalid_state           Operation not allowed in current resource state.
rate_limit_exceeded     Too many requests. Wait before retrying.
======================= =====================================================

Request/Response Formats
------------------------

All API requests and responses use JSON format.

**Request headers:**

.. code::

    Content-Type: application/json
    Authorization: Bearer <YOUR_ACCESS_TOKEN>

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

Rate Limiting
-------------

VoIPBIN applies rate limits to protect service stability. When you exceed the rate limit, the API returns a ``429 Too Many Requests`` response.

**Rate limit headers:**

.. code::

    X-RateLimit-Limit: 100
    X-RateLimit-Remaining: 45
    X-RateLimit-Reset: 1609459200

**Handling rate limits:**

1. Check the ``Retry-After`` header for when to retry
2. Implement exponential backoff for retries
3. Cache responses when possible to reduce API calls
