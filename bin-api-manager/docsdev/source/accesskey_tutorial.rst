.. _accesskey-tutorial:

Tutorial
========

Create, Retrieve, and Manage Accesskeys
---------------------------------------
This tutorial demonstrates how to create an access key, retrieve a list of access keys, and retrieve a specific access key using the API. All requests must include the `accesskey` query parameter for authentication.

1. **Create an Accesskey**

   Use the following command to create a new access key. The `expire` parameter specifies the duration in seconds before the key expires.

   .. code::

      $ curl -k --location --request POST 'https://api.voipbin.net/v1.0/accesskeys?accesskey=<your-access-key>' \
      --header 'Content-Type: application/json' \
      --data-raw '{
          "name": "My New Accesskey",
          "detail": "This key is used for reporting",
          "expire": 31536000
      }'

   Example Response:

   .. code-block:: json

      {
          "id": "2f1f8f7e-9b3d-4c60-8465-b69e9f28b6dc",
          "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
          "name": "My New Accesskey",
          "detail": "This key is used for reporting",
          "token": "3yTqk5F4ABCD2xy9",
          "tm_expire": "2025-12-01 10:15:30.123456",
          "tm_create": "2024-12-01 10:15:30.123456",
          "tm_update": "2024-12-01 10:15:30.123456",
          "tm_delete": "9999-01-01 00:00:00.000000"
      }

2. **Get a List of Accesskeys**

   Retrieve all existing access keys associated with your account. Include the `accesskey` query parameter in the request URL for authentication.

   .. code::

      $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/accesskeys?accesskey=<your-access-key>'

   Example Response:

   .. code-block:: json

      {
          "result": [
              {
                  "id": "2f1f8f7e-9b3d-4c60-8465-b69e9f28b6dc",
                  "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
                  "name": "My New Accesskey",
                  "detail": "This key is used for reporting",
                  "token": "3yTqk5F4ABCD2xy9",
                  "tm_expire": "2025-12-01 10:15:30.123456",
                  "tm_create": "2024-12-01 10:15:30.123456",
                  "tm_update": "2024-12-01 10:15:30.123456",
                  "tm_delete": "9999-01-01 00:00:00.000000"
              }
          ],
          "next_page_token": null
      }

3. **Get a Specific Accesskey**

   Retrieve details of a specific access key using its unique ID. Include the `accesskey` query parameter for authentication.

   .. code::

      $ curl -k --location --request GET 'https://api.voipbin.net/v1.0/accesskeys/2f1f8f7e-9b3d-4c60-8465-b69e9f28b6dc?accesskey=<your-access-key>'

   Example Response:

   .. code-block:: json

      {
          "id": "2f1f8f7e-9b3d-4c60-8465-b69e9f28b6dc",
          "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
          "name": "My New Accesskey",
          "detail": "This key is used for reporting",
          "token": "3yTqk5F4ABCD2xy9",
          "tm_expire": "2025-12-01 10:15:30.123456",
          "tm_create": "2024-12-01 10:15:30.123456",
          "tm_update": "2024-12-01 10:15:30.123456",
          "tm_delete": "9999-01-01 00:00:00.000000"
      }
