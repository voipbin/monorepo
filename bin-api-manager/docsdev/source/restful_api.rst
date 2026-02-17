.. _restful_api:

**********************************************
RESTful API(Application Programming Interface)
**********************************************
The VoIPBIN RESTful API offers a robust set of endpoints designed to facilitate seamless Voice over IP (VoIP) communications. By leveraging the OpenAPI specification, the API ensures a standardized, well-documented, and comprehensive interface for developers. This allows for easy integration and interaction with the VoIPBIN service, enabling efficient management of VoIP functionalities and services.

API Reference Documentation
============================

For complete API endpoint documentation including request/response schemas, authentication requirements, and interactive examples, see:

* **ReDoc (Recommended)**: https://api.voipbin.net/redoc/index.html
* **Swagger UI**: https://api.voipbin.net/swagger/index.html

Getting Started with the API
=============================

Authentication
--------------
All API requests must be authenticated using either:

* **JWT Token**: Include in ``Authorization: Bearer <token>`` header
* **Access Key**: Include as ``?accesskey=<key>`` query parameter

For details, see :ref:`Authentication <quickstart_authentication>`.

Base URL
--------
All API endpoints are prefixed with::

    https://api.voipbin.net/v1.0/

API Conventions
===============

Request Format
--------------
* All POST and PUT requests must use ``Content-Type: application/json``
* Request bodies must be valid JSON
* URL parameters must be URL-encoded

Response Format
---------------
All API responses return JSON with consistent structure:

Success Response (2xx)::

    {
        "id": "resource-id",
        "field1": "value1",
        ...
    }

List Response::

    {
        "result": [...],
        "next_page_token": "token-for-next-page"   // Pass as ?page_token= to get the next page
    }

.. note:: **AI Implementation Hint**

   List endpoints return paginated results. If ``next_page_token`` is non-empty, pass it as the ``page_token`` query parameter in the next request to retrieve subsequent pages. When ``next_page_token`` is empty or absent, you have reached the last page.

Error Response (4xx, 5xx)::

    {
        "error": "error_code",
        "message": "Human-readable error description"
    }

Common HTTP Status Codes
------------------------

.. note:: **AI Implementation Hint**

   When an API call returns an error, parse the JSON response body for the ``message`` field which contains a human-readable error description. Use the HTTP status code to determine the error category (4xx = client error, 5xx = server error) and the ``message`` field for specific fix guidance.

* **200 OK**: Request succeeded. Response body contains the requested resource.
* **201 Created**: Resource created successfully. Response body contains the new resource with its assigned ``id`` (UUID).
* **400 Bad Request**: Invalid request parameters. Check request body JSON syntax, required fields, and value formats (E.164 for phone numbers, UUID for IDs).
    * **Common causes:** Missing required fields, invalid phone number format, malformed UUID, invalid enum value.
    * **Fix:** Validate all required fields and data types before sending the request.
* **401 Unauthorized**: Missing or invalid authentication token.
    * **Cause:** JWT token expired (older than 7 days), malformed token, or missing ``Authorization`` header.
    * **Fix:** Generate a new token via ``POST /auth/login`` or use a valid accesskey.
* **402 Payment Required**: Insufficient account balance for the requested operation.
    * **Cause:** Operations like calls, messages, and AI sessions deduct credits.
    * **Fix:** Check balance via ``GET /billing-accounts``. Top up the account before retrying.
* **403 Forbidden**: Insufficient permissions for the requested resource.
    * **Cause:** The resource belongs to a different customer account, or the agent lacks the required permission level.
    * **Fix:** Verify the resource belongs to your account. Check agent permission level (admin vs. manager).
* **404 Not Found**: Resource does not exist or has been deleted.
    * **Cause:** The UUID is invalid, the resource was deleted, or it belongs to another customer.
    * **Fix:** Verify the UUID was obtained from a recent ``GET`` list call.
* **429 Too Many Requests**: Rate limit exceeded.
    * **Fix:** Implement exponential backoff and retry after the duration specified in the ``Retry-After`` header.
* **500 Internal Server Error**: Server-side error.
    * **Fix:** Retry the request. If the error persists, contact support with the request details.

For detailed endpoint documentation, parameter descriptions, and response schemas, visit the API reference documentation linked above.
