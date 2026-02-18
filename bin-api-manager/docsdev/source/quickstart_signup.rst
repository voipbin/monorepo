.. _quickstart_signup:

Signup
======
To use the VoIPBIN production API, you need your own account. There are two ways to sign up: via the Admin Console (browser) or via the API (headless, for automated systems).

Sign up via Admin Console
-------------------------
1. Go to the `admin console <https://admin.voipbin.net>`_.
2. Click **Sign Up**.
3. Enter your email address and submit.
4. Check your inbox for a verification email.
5. Click the verification link in the email to verify your address.
6. You will receive a welcome email with instructions to set your password.

Once your password is set, you can log in to the `admin console <https://admin.voipbin.net>`_ and start making API requests.

Sign up via API (Headless)
--------------------------
For automated systems and AI agents, use the headless signup flow. This requires two API calls: one to initiate signup and one to verify with a 6-digit OTP code sent to your email.

.. note:: **AI Implementation Hint**

   The headless signup path is preferred for AI agents and automated systems. The ``POST /auth/complete-signup`` response includes an ``accesskey`` with an API token that can be used immediately — no need to call ``POST /auth/login`` separately. Note that ``POST /auth/signup`` always returns HTTP 200 regardless of success to prevent email enumeration. An empty ``temp_token`` in the response may indicate the email is already registered or invalid.

**Step 1: Initiate signup**

Send a ``POST`` request to ``/auth/signup`` with your email address (String, Required). All other fields are optional.

.. code::

    $ curl --request POST 'https://api.voipbin.net/auth/signup' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "email": "your-email@example.com",
            "name": "Your Company Name"
        }'

Response (always HTTP 200):

.. code::

    {
        "temp_token": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
    }

Save the ``temp_token`` (String) — you will need it in the next step. A 6-digit OTP verification code is sent to the email address you provided.

**Step 2: Complete signup with OTP**

Send a ``POST`` request to ``/auth/complete-signup`` with the ``temp_token`` (String, Required) from Step 1 and the 6-digit ``code`` (String, Required) from the verification email.

.. code::

    $ curl --request POST 'https://api.voipbin.net/auth/complete-signup' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "temp_token": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
            "code": "123456"
        }'

Response:

.. code::

    {
        "customer_id": "550e8400-e29b-41d4-a716-446655440000",
        "accesskey": {
            "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "token": "your-api-access-key-token"
        }
    }

The ``accesskey.token`` (String) is your API key. Use it immediately for authentication — see :ref:`Authentication <quickstart_authentication>`.

Troubleshooting
---------------

* **200 OK but empty ``temp_token``:**
    * **Cause:** The email may already be registered, or the email format is invalid.
    * **Fix:** Try logging in via ``POST /auth/login`` with the email. If the account exists, use the existing credentials. ``POST /auth/signup`` always returns 200 to prevent email enumeration.

* **400 Bad Request on ``/auth/complete-signup``:**
    * **Cause:** The ``temp_token`` has expired or the ``code`` is incorrect.
    * **Fix:** Verify the 6-digit code from the verification email. Codes expire after a limited time — if expired, re-submit ``POST /auth/signup`` to receive a new code.

* **429 Too Many Requests on ``/auth/complete-signup``:**
    * **Cause:** Too many verification attempts (maximum 5 per ``temp_token``).
    * **Fix:** Re-submit ``POST /auth/signup`` with the same email to receive a new ``temp_token`` and verification code, then retry.
