.. _quickstart-signup:

Signup
------
To use the VoIPBIN production API, you need your own account. There are two ways to sign up: via the Admin Console (browser) or via the API (headless, for automated systems).

Sign up via Admin Console
~~~~~~~~~~~~~~~~~~~~~~~~~
1. Go to the `admin console <https://admin.voipbin.net>`_.
2. Click **Sign Up**.
3. Enter your email address and submit.
4. Check your inbox for a verification email.
5. Click the verification link in the email to verify your address.
6. You will receive a welcome email with instructions to set your password.

Once your password is set, you can log in to the `admin console <https://admin.voipbin.net>`_ and start making API requests.

Sign up via API
~~~~~~~~~~~~~~~~~~~~~~~~~~
For automated systems and AI agents, use the API signup flow. This is a single API call that returns an access key immediately.

.. note:: **AI Implementation Hint**

   The ``POST /auth/signup`` response includes an ``accesskey`` with an API token that can be used immediately — no need to call ``POST /auth/login`` separately. Note that ``POST /auth/signup`` always returns HTTP 200 regardless of success to prevent email enumeration. An empty response body (``{}``) may indicate the email is already registered or invalid.

**Initiate signup**

Send a ``POST`` request to ``/auth/signup`` with your email address (String, Required) and ``accepted_tos`` (Boolean, Required, must be ``true``). All other fields are optional.

.. code::

    $ curl --request POST 'https://api.voipbin.net/auth/signup' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "email": "your-email@example.com",
            "name": "Your Company Name",
            "accepted_tos": true
        }'

Response (always HTTP 200):

.. code::

    {
        "customer": {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "name": "Your Company Name",
            "email": "your-email@example.com",
            ...
        },
        "accesskey": {
            "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "token": "your-api-access-key-token"
        }
    }

The ``accesskey.token`` (String) is your API key. Use it immediately for authentication — see :ref:`Authentication <quickstart-authentication>`.

A verification email is also sent to the email address you provided. Click the link in the email to verify your address and activate your account. After verification, you will receive a welcome email with instructions to set your password.

Troubleshooting
~~~~~~~~~~~~~~~

* **200 OK but empty response (``{}``):**
    * **Cause:** The email may already be registered, or the email format is invalid.
    * **Fix:** Try logging in via ``POST /auth/login`` with the email. If the account exists, use the existing credentials. ``POST /auth/signup`` always returns 200 to prevent email enumeration.

* **400 Bad Request:**
    * **Cause:** Missing required fields (``email`` or ``accepted_tos``) or ``accepted_tos`` is ``false``.
    * **Fix:** Ensure the request body includes ``"email"`` and ``"accepted_tos": true``.
