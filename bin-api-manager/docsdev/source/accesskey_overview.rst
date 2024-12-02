.. _accesskey-overview:

Accesskey API Overview
======================
The Accesskey API provides a secure and efficient way to authenticate and interact with the VoIPBIN platform. It utilizes access keys as API tokens to enable authorized access to API endpoints. This ensures seamless integration while maintaining the security and integrity of your operations.

API Usage
---------
To use the Accesskey, pass the `token` in the query parameters or headers of your API requests. Below is an example usage:

Example Request
~~~~~~~~~~~~~~~
.. code-block:: bash

   curl --location 'https://api.voipbin.net/v1.0/accesskeys?accesskey=DuRWq5T4DAK32dw4' \
   --data ''

Description of Parameters
~~~~~~~~~~~~~~~~~~~~~~~~~
- **Accesskey**: The `accesskey` query parameter must contain the token issued for the customer. This token uniquely identifies and authorizes the request.

JSON Structure
--------------
Below is the JSON representation of the Accesskey resource:

.. code-block:: json

   {
      "id": "5f1f8f7e-9b3d-4c60-8465-b69e9f28b6db",
      "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
      "name": "My API Key",
      "detail": "For accessing reporting APIs",
      "token": "DuRWq5T4DAK32dw4",
      "tm_expire": "2025-04-28 01:41:40.503790",
      "tm_create": "2022-04-28 01:41:40.503790",
      "tm_update": "2022-04-28 01:41:40.503790",
      "tm_delete": "9999-01-01 00:00:00.000000"
   }

Authentication
--------------
The `token` serves as the authentication credential. Ensure you store it securely and avoid exposing it in client-side code or public repositories.

Lifecycles and Expiry
---------------------
Accesskeys come with lifecycle timestamps:

- **tm_create**: The creation timestamp.
- **tm_update**: The last updated timestamp.
- **tm_expire**: The expiration timestamp. Ensure you rotate keys before this date.
- **tm_delete**: The deletion timestamp, if applicable.

Notes
-----
- Always use HTTPS to ensure secure communication.
- Tokens should be rotated periodically to enhance security.
