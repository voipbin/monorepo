.. _quickstart_authentication:

Authentication
--------------
Every API request must be authenticated using either a **Token** (JWT string) or an **Accesskey** (API key string). Both serve the same purpose — choose whichever fits your workflow.

.. note:: **AI Implementation Hint**

   Use **Token** (JWT) for short-lived, session-based authentication (valid for 7 days). Use **Accesskey** for long-lived, programmatic access with custom expiration. For automated systems and AI agents, Accesskey is recommended because it does not require periodic re-authentication.

Generate a Token
~~~~~~~~~~~~~~~~
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

Use the token in subsequent API requests. See `Using Credentials in API Requests`_ below for usage examples.

Generate an Accesskey
~~~~~~~~~~~~~~~~~~~~~
For long-lived authentication, generate an access key. You can create one from the `admin console <https://admin.voipbin.net>`_ or via the API.

**Via Admin Console:**

.. image:: _static/images/quickstart_authentication_accesskey.png

**Via API:**

.. code::

    $ curl --request POST 'https://api.voipbin.net/v1.0/accesskeys?token=<your-token>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "name": "my-api-key",
            "detail": "API key for automated access"
        }'

Response:

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "token": "your-access-key-token",
        "name": "my-api-key",
        "detail": "API key for automated access",
        ...
    }

.. note:: **AI Implementation Hint**

   If you signed up via the headless flow (``POST /auth/complete-signup``), the response already includes an ``accesskey`` with a valid token — you do not need to create another one. Use ``POST /v1.0/accesskeys`` only if you need additional keys or want to set a custom expiration via the ``expire`` (Integer, Optional) field.

Using Credentials in API Requests
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
The query parameter name depends on which credential type you are using:

+-------------------+-----------------------------------------------------+
| Credential        | Query Parameter                                     |
+===================+=====================================================+
| Token (JWT)       | ``?token=<your-token>``                             |
+-------------------+-----------------------------------------------------+
| Accesskey         | ``?accesskey=<your-accesskey>``                     |
+-------------------+-----------------------------------------------------+

**Token example:**

.. code::

    $ curl --request GET 'https://api.voipbin.net/v1.0/calls?token=<your-token>'

**Accesskey example:**

.. code::

    $ curl --request GET 'https://api.voipbin.net/v1.0/calls?accesskey=<your-accesskey>'

Tokens can also be passed via the ``Authorization`` header:

.. code::

    $ curl --request GET 'https://api.voipbin.net/v1.0/calls' \
        --header 'Authorization: Bearer <your-token>'

.. note:: **AI Implementation Hint**

   The parameter names are **not interchangeable** — use ``token=`` for JWT tokens and ``accesskey=`` for access keys. Using the wrong parameter name results in ``401 Unauthorized``. All quickstart examples use ``?token=<your-token>`` — if you are using an accesskey instead, replace ``token=`` with ``accesskey=`` in every request.

Troubleshooting
~~~~~~~~~~~~~~~

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
