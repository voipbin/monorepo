.. _accesskey-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before managing accesskeys, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an existing access key from ``GET /accesskeys``.
* (For creation) The ``expire`` duration in seconds (e.g., ``31536000`` for one year).

.. note:: **AI Implementation Hint**

   The ``expire`` field in the create request is in seconds, not a timestamp. For example, use ``86400`` for a one-day key, ``2592000`` for 30 days, or ``31536000`` for one year. The API calculates the actual expiration timestamp and returns it in the ``tm_expire`` field.

Create, Retrieve, and Manage Accesskeys
---------------------------------------
This tutorial demonstrates how to create an access key, retrieve a list of access keys, and retrieve a specific access key using the API. All requests must include the ``accesskey`` query parameter for authentication.

1. **Create an Accesskey**

   Use the following command to create a new access key. The ``expire`` parameter specifies the duration in seconds before the key expires.

   .. code::

      $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/accesskeys?accesskey=<your-access-key>' \
      --header 'Content-Type: application/json' \
      --data-raw '{
          "name": "My New Accesskey",
          "detail": "This key is used for reporting",
          "expire": 31536000
      }'

   Example Response:

   .. code::

      {
          "id": "2f1f8f7e-9b3d-4c60-8465-b69e9f28b6dc",
          "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
          "name": "My New Accesskey",
          "detail": "This key is used for reporting",
          "token": "vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xW",
          "token_prefix": "vb_a3Bf9xK",
          "tm_expire": "2027-12-01T10:15:30.123456Z",
          "tm_create": "2026-12-01T10:15:30.123456Z",
          "tm_update": "2026-12-01T10:15:30.123456Z",
          "tm_delete": "9999-01-01T00:00:00.000000Z"
      }

   .. note:: **AI Implementation Hint**

      The ``token`` field is only returned in this creation response. **Store it securely and immediately.** You will not be able to retrieve the full token again. If the token is lost, delete the key via ``DELETE /accesskeys/{id}`` and create a new one. Use ``token_prefix`` to identify which key is which in subsequent requests.

2. **Get a List of Accesskeys**

   Retrieve all existing access keys associated with your account. Include the ``accesskey`` query parameter in the request URL for authentication.

   .. code::

      $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/accesskeys?accesskey=<your-access-key>'

   Example Response:

   .. code::

      {
          "result": [
              {
                  "id": "2f1f8f7e-9b3d-4c60-8465-b69e9f28b6dc",
                  "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
                  "name": "My New Accesskey",
                  "detail": "This key is used for reporting",
                  "token_prefix": "vb_a3Bf9xK",
                  "tm_expire": "2027-12-01T10:15:30.123456Z",
                  "tm_create": "2026-12-01T10:15:30.123456Z",
                  "tm_update": "2026-12-01T10:15:30.123456Z",
                  "tm_delete": "9999-01-01T00:00:00.000000Z"
              }
          ],
          "next_page_token": null
      }

3. **Get a Specific Accesskey**

   Retrieve details of a specific access key using its unique ID. Include the ``accesskey`` query parameter for authentication.

   .. code::

      $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/accesskeys/2f1f8f7e-9b3d-4c60-8465-b69e9f28b6dc?accesskey=<your-access-key>'

   Example Response:

   .. code::

      {
          "id": "2f1f8f7e-9b3d-4c60-8465-b69e9f28b6dc",
          "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
          "name": "My New Accesskey",
          "detail": "This key is used for reporting",
          "token_prefix": "vb_a3Bf9xK",
          "tm_expire": "2027-12-01T10:15:30.123456Z",
          "tm_create": "2026-12-01T10:15:30.123456Z",
          "tm_update": "2026-12-01T10:15:30.123456Z",
          "tm_delete": "9999-01-01T00:00:00.000000Z"
      }
