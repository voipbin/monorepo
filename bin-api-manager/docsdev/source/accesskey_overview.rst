.. _accesskey-overview:

Accesskey API Overview
======================

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free (no charges for accesskey operations)
   * **Async:** No. All accesskey operations are synchronous and return immediately.

The Accesskey API provides a secure and efficient way to authenticate and interact with the VoIPBIN platform. It utilizes access keys as API tokens to enable authorized access to API endpoints. This ensures seamless integration while maintaining the security and integrity of your operations.

.. note:: **AI Implementation Hint**

   Use the ``accesskey`` query parameter (not the ``token`` parameter) when authenticating with an access key. The ``token`` parameter is used for session-based authentication, while ``accesskey`` is for API key-based authentication. Do not confuse the two.

API Usage
---------
To use the Accesskey, pass the ``accesskey`` in the query parameters of your API requests. Below is an example usage:

Example Request
~~~~~~~~~~~~~~~
.. code::

   curl --location 'https://api.voipbin.net/v1.0/calls?accesskey=vb_a3Bf9xKmPq2nR7sT4wYzLp8mN5qR1xWe'

Description of Parameters
~~~~~~~~~~~~~~~~~~~~~~~~~
- **Accesskey**: The ``accesskey`` query parameter must contain the token issued for the customer. This token uniquely identifies and authorizes the request. Tokens are prefixed with ``vb_`` for identification.

Authentication
--------------
The ``accesskey`` serves as the authentication credential. Tokens are hashed server-side using SHA-256 before storage. The full token is only shown once at creation time. Store it securely and avoid exposing it in client-side code or public repositories.

Lifecycles and Expiry
---------------------
Accesskeys come with lifecycle timestamps:

- **tm_create**: The creation timestamp.
- **tm_update**: The last updated timestamp.
- **tm_expire**: The expiration timestamp. Ensure you rotate keys before this date.
- **tm_delete**: The deletion timestamp, if applicable.

.. note:: **AI Implementation Hint**

   When creating an accesskey, the ``expire`` field is specified in seconds (e.g., ``31536000`` for one year). The API returns ``tm_expire`` as an ISO 8601 timestamp calculated from the creation time plus the expiry duration. Always check ``tm_expire`` before using a key to avoid authentication failures with expired tokens.

Notes
-----
- Always use HTTPS to ensure secure communication.
- Tokens should be rotated periodically to enhance security.
- If a token is lost, delete the key and create a new one. The full token cannot be retrieved after creation.
