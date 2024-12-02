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

   curl --location 'https://api.voipbin.net/v1.0/calls?accesskey=DuRWq5T4DAK32dw4'

Description of Parameters
~~~~~~~~~~~~~~~~~~~~~~~~~
- **Accesskey**: The `accesskey` query parameter must contain the token issued for the customer. This token uniquely identifies and authorizes the request.

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
