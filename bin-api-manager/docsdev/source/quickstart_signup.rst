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

If the link has expired (verification links are valid for 24 hours, and an unverified signup is left open for 72 hours), the page it opens will offer to send you a new one. Enter your email address there and follow the fresh link.

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

If that email is lost or the link expires, request a new one with ``POST /auth/email-verify-resend`` (body: ``{"email": "your-email@example.com"}``). Like signup, it always returns HTTP 200 with an empty body, so the response does not tell you whether an email was sent. Sends are limited to one per 60 seconds and 5 per 24 hours per account. See :ref:`Auth Overview <auth-overview>` for details.

Troubleshooting
~~~~~~~~~~~~~~~

* **200 OK but empty response (``{}``):**
    * **Cause:** The email may already be registered, or the email format is invalid.
    * **Fix:** Try logging in via ``POST /auth/login`` with the email. If the account exists, use the existing credentials. ``POST /auth/signup`` always returns 200 to prevent email enumeration.

* **400 Bad Request:**
    * **Cause:** Missing required fields (``email`` or ``accepted_tos``) or ``accepted_tos`` is ``false``.
    * **Fix:** Ensure the request body includes ``"email"`` and ``"accepted_tos": true``.

* **Verification link expired / verification email never arrived:**
    * **Cause:** Verification tokens are valid for 24 hours, and the email may have been filtered or lost.
    * **Fix:** Call ``POST /auth/email-verify-resend`` with the registered email address, or use the resend form on the page the expired link opens. An account that already expired for want of verification can still be recovered this way.
