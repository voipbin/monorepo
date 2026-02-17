.. _quickstart_authentication:

Authentication
==============
Every API request must be authenticated using either a **Token** (JWT string) or an **Accesskey** (API key string). Both serve the same purpose — choose whichever fits your workflow.

.. note:: **AI Implementation Hint**

   Use **Token** (JWT) for short-lived, session-based authentication (valid for 7 days). Use **Accesskey** for long-lived, programmatic access with custom expiration. For automated systems and AI agents, Accesskey is recommended because it does not require periodic re-authentication.

Generate a Token
----------------
Send a ``POST`` request to ``/auth/login`` with your username (String) and password (String) to receive a JWT token. The token is valid for 7 days.

.. code::

    $ curl --request POST 'https://api.voipbin.net/auth/login' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "username": "your-voipbin-username",
            "password": "your-voipbin-password"
        }'

Response:

.. code::

    {
        "username": "your-voipbin-username",
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }

Use the token in subsequent requests via the ``Authorization`` header:

.. code::

    $ curl -k --request GET 'https://api.voipbin.net/v1.0/accesskeys' \
        --header 'Authorization: Bearer <your-token>'

Or as a query parameter:

.. code::

    $ curl -k --request GET 'https://api.voipbin.net/v1.0/accesskeys?token=<your-token>'

Generate an Accesskey
---------------------
For long-lived authentication, generate an access key from the `admin console <https://admin.voipbin.net>`_. You can set a custom expiration when creating it.

.. image:: _static/images/quickstart_authentication_accesskey.png

Use the access key as a query parameter:

.. code::

    $ curl -k --request GET 'https://api.voipbin.net/v1.0/accesskeys?accesskey=<your-accesskey>'

Troubleshooting
---------------

* **401 Unauthorized:**
    * **Cause:** Token has expired (older than 7 days) or is malformed.
    * **Fix:** Generate a new token via ``POST /auth/login``. Ensure the ``Authorization`` header uses the format ``Bearer <token>`` (with a space after "Bearer").

* **401 Unauthorized (Accesskey):**
    * **Cause:** Accesskey has expired (past ``tm_expire``) or has been deleted.
    * **Fix:** Check the accesskey's expiration via ``GET /accesskeys``. Generate a new one from the admin console if expired.

* **403 Forbidden:**
    * **Cause:** The authenticated user lacks permission for the requested resource.
    * **Fix:** Verify the resource belongs to your customer account. Check your permission level (admin vs. manager).

For more details, see the full :ref:`Accesskey tutorial <accesskey-main>`.
