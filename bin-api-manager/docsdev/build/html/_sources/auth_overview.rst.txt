.. _auth-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low to Medium (unregister/delegate carry authorization and lifecycle side effects)
   * **Cost:** Free. No auth endpoint incurs platform billing.
   * **Async:** No. All auth operations are synchronous, except that signup/password-forgot/unregister trigger emails sent out-of-band.

All endpoints in this group are served directly at ``/auth/<name>`` — **not** under the ``/v1.0`` prefix used by every other resource in the API. A generated stub exists for each path under ``/v1.0/auth/...`` that always returns ``404 ROUTE_NOT_FOUND`` with a message pointing at the correct un-prefixed path; this is intentional routing scaffolding, not a bug.

Endpoint Summary
----------------

.. list-table::
   :header-rows: 1
   :widths: 10 22 10 58

   * - Method
     - Path
     - Auth
     - Purpose
   * - POST
     - ``/auth/signup``
     - None
     - Self-service customer registration.
   * - POST
     - ``/auth/email-verify``
     - None
     - Verify the signup email token; provisions the first access key.
   * - GET
     - ``/auth/email-verify``
     - None
     - Serves an HTML page that auto-submits the verification token (link target from the verification email).
   * - POST
     - ``/auth/login``
     - Username/password
     - Exchange agent credentials for a 7-day JWT. See :ref:`Authentication quickstart <quickstart-authentication>`.
   * - POST
     - ``/auth/boot``
     - None (direct hash)
     - Exchange a direct hash for a 4-hour resource-scoped JWT.
   * - POST
     - ``/auth/password-forgot``
     - None
     - Request a password reset email.
   * - GET
     - ``/auth/password-reset``
     - None
     - Serves the HTML password reset form (link target from the reset email).
   * - POST
     - ``/auth/password-reset``
     - None (reset token)
     - Complete the password reset using the emailed token.
   * - POST
     - ``/auth/unregister``
     - Token or Accesskey
     - Freeze the caller's own account and schedule (or immediately execute) deletion.
   * - DELETE
     - ``/auth/unregister``
     - Token or Accesskey
     - Cancel a scheduled deletion and restore the account to ``active``.
   * - POST
     - ``/auth/delegate``
     - Token (ProjectSuperAdmin)
     - Issue a short-lived support/investigation token scoped to another customer. See :ref:`Authentication quickstart <quickstart-authentication>`.

.. note:: **AI Implementation Hint**

   All unauthenticated ``/auth/*`` endpoints (``login``, ``signup``, ``email-verify``, ``boot``, ``password-forgot``, ``password-reset``) share a single IP-based rate limiter: **up to approximately 10 requests/second per client IP, burst 20**. ``/auth/unregister`` and ``/auth/delegate`` require authentication first, then are subject to their own, separately-tracked IP-based limiter at the same approximate rate (**10 requests/second, burst 20**). Exceeding either limiter returns ``429`` with reason ``RATE_LIMIT_EXCEEDED`` (see :ref:`Error Reason Codes <error-reason-catalog>`).

.. note:: **AI Implementation Hint — response body shape differs by endpoint**

   ``/auth/unregister`` and ``/auth/delegate`` sit behind the shared ``Authenticate()`` middleware, so authentication/authorization failures at that layer (missing token, expired token, frozen account) return the standard structured error envelope described in :ref:`Error Reason Codes <error-reason-catalog>` (``{"error": {"status", "reason", "message", "request_id", ...}}``). However, validation failures raised *inside* the auth handlers themselves — e.g. a malformed JSON body on any endpoint, a wrong password on ``/auth/unregister``, an invalid ``confirmation_phrase``, a bad ``accepted_tos`` on signup — return a bare HTTP status code (400) with an **empty response body**, not the JSON error envelope. Do not assume ``error.reason`` is present on every 4xx from this group; check for an empty body first.


Auth & Account Lifecycle
------------------------

::

    signup          email-verify        (set password)        login / boot
    +----------+   +--------------+    +----------------+    +--------------+
    | initial  |-->|   active     |--->|    active      |--->|  JWT / key   |
    +----------+   +--------------+    +----------------+    +--------------+
         |                                    |
         | (no verification                   | POST /auth/unregister
         |  within timeout)                   v
         v                              +----------------+
    +----------+                        |    frozen      |
    | expired  |                        | (30-day grace) |
    +----------+                        +----------------+
                                          |             |
                            DELETE /auth/unregister      (grace expires,
                                          |               or immediate:true)
                                          v                       v
                                    +----------------+     +----------------+
                                    |    active      |     |    deleted     |
                                    +----------------+     +----------------+

For the full customer status lifecycle (including what gets cascade-deleted), see :ref:`Customer — Account Deletion Lifecycle <customer-main>`.


Signup — ``POST /auth/signup``
-------------------------------
Creates an unverified customer account and triggers a verification email. See the :ref:`Signup quickstart <quickstart-signup>` for a walkthrough.

**Request body**

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Field
     - Type
     - Description
   * - ``email``
     - String, Required
     - Email address for the new account. Must be unique across all customers.
   * - ``accepted_tos``
     - Boolean, Required
     - Must be ``true``. Missing or ``false`` is rejected with ``400``.
   * - ``name``
     - String, Optional
     - Display name for the customer account.
   * - ``detail``
     - String, Optional
     - Additional description.
   * - ``phone_number``
     - String, Optional
     - Contact phone number (E.164 format recommended).
   * - ``address``
     - String, Optional
     - Mailing address.
   * - ``webhook_method``
     - String, Optional
     - HTTP method for webhook delivery: ``POST``, ``GET``, ``PUT``, or ``DELETE``.
   * - ``webhook_uri``
     - String, Optional
     - URI where webhook events will be delivered.

**Response — 200 OK (success)**

.. code::

    {
        "customer": { "id": "...", "email": "...", "status": "initial", ... },
        "accesskey": { "id": "...", "token": "vb_...", "name": "...", ... }
    }

**Response — 200 OK (silent failure, e.g. duplicate email)**

.. code::

    {}

.. note:: **AI Implementation Hint**

   Signup always returns HTTP 200, even on failure, to prevent email-enumeration attacks — a duplicate or invalid email produces an empty ``{}`` body rather than an error. The only true ``400`` is a structurally invalid request (missing ``email``, missing/false ``accepted_tos``, or malformed JSON). The returned ``accesskey.token`` is usable immediately; no separate ``POST /auth/login`` call is required. Signup also auto-provisions an empty ``OutboundConfig`` (blocking all outbound PSTN calls until explicitly configured) and auto-rolls back the entire signup (deletes the customer) if that provisioning step permanently fails.

**Errors**

.. list-table::
   :header-rows: 1
   :widths: 15 85

   * - Status
     - Cause
   * - 400
     - ``email`` missing, ``accepted_tos`` missing/``false``, or malformed JSON body.


Email Verify — ``POST`` / ``GET /auth/email-verify``
------------------------------------------------------
Validates the verification token from the signup email, marks the customer's email as verified, and transitions the account from ``initial`` to ``active``.

**POST request body**

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Field
     - Type
     - Description
   * - ``token``
     - String, Required
     - 64-character lowercase hex verification token from the signup email.

**Response — 200 OK**

.. code::

    {
        "customer": { "id": "...", "email": "...", "email_verified": true, "status": "active", ... }
    }

**Errors**

.. list-table::
   :header-rows: 1
   :widths: 15 85

   * - Status
     - Cause
   * - 400
     - ``token`` is missing, invalid, expired, or already used; or the request body is malformed JSON.

**GET /auth/email-verify**

Serves a static HTML confirmation page (the link target embedded in the verification email). The page reads ``token`` from the query string and, when the visitor clicks "Verify Email", submits it to ``POST /auth/email-verify`` via client-side JavaScript. This is a convenience UI for humans following the email link — API integrations should call ``POST /auth/email-verify`` directly with the token extracted from the email/webhook.

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Parameter
     - Location
     - Description
   * - ``token``
     - Query, Required
     - 64-character lowercase hex string. An invalid/missing token returns ``400`` before the page is rendered.


Login (Token) — ``POST /auth/login``
--------------------------------------
Exchanges an agent's ``username``/``password`` for a JWT valid for 7 days. Fully documented with request/response examples in the :ref:`Authentication quickstart <quickstart-authentication>`; summarized here for completeness.

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Field
     - Type
     - Description
   * - ``username``
     - String, Required
     - The agent's username.
   * - ``password``
     - String, Required
     - The agent's password.

Response: ``{"username": "...", "token": "eyJ..."}``. The token is also set as an ``HttpOnly``, ``Secure``, ``SameSite=Strict`` cookie named ``token`` on the response. Errors: ``400`` on missing fields, malformed JSON, or invalid credentials (login failures are not distinguished from bad requests — both return a bare ``400``).


Boot (Direct Token) — ``POST /auth/boot``
--------------------------------------------
Resolves a direct hash into a short-lived, resource-scoped JWT, for use cases where the end user has no VoIPBin account (e.g. an embedded AI voice widget on a customer's public website). Also documented in the :ref:`Authentication quickstart <quickstart-authentication>`.

**Request body**

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Field
     - Type
     - Description
   * - ``direct_hash``
     - String, Required
     - The direct hash link, e.g. ``direct.a1b2c3d4e5f6``. Obtained from a direct link URL or a resource's ``direct_hash`` field (e.g. ``GET /v1.0/webchat_widgets/{id}``).

**Response — 200 OK**

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Field
     - Type
     - Description
   * - ``token``
     - String
     - JWT for API/WebSocket use. Pass as ``Bearer <token>``.
   * - ``type``
     - String
     - Always ``"direct"``.
   * - ``resource_type``
     - String
     - The resource type this token is scoped to (e.g. ``"ai"``, ``"ai_team"``, ``"webchat_widget"``).
   * - ``resource_id``
     - UUID
     - The scoped resource's ID.
   * - ``customer_id``
     - UUID
     - The owning customer's ID.
   * - ``expire``
     - String (ISO 8601)
     - Token expiry. Valid for 4 hours from issuance.
   * - ``resource_data``
     - Object, Optional
     - Present only for resource types with a registered public-display fetcher (currently ``webchat_widget``). Contains a ``public_display_config`` key with anonymous-visitor-safe display settings (e.g. theme). Omitted entirely when there is nothing to report; a fetch failure never fails the boot request itself.

**Errors**

.. list-table::
   :header-rows: 1
   :widths: 15 85

   * - Status
     - Cause
   * - 400
     - ``direct_hash`` missing/empty, does not start with ``direct.``, does not resolve to any resource, resolves to an unsupported resource type, or the owning customer is not ``active``.

.. note:: **AI Implementation Hint**

   Boot tokens are scoped: they only grant access to the resource types listed in the token (e.g. ``aicall`` for an ``ai``/``ai_team`` direct hash, ``webchat_session`` for a ``webchat_widget`` direct hash). For WebSocket subscriptions, boot tokens may only subscribe to 4-part topics (``customer_id:<uuid>:<resource_type>:<resource_id>``); broader topics are rejected.


Password Forgot — ``POST /auth/password-forgot``
----------------------------------------------------
Generates a password reset token and emails a reset link to the agent, valid for 1 hour.

**Request body**

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Field
     - Type
     - Description
   * - ``username``
     - String, Required
     - The agent's username (email address).

**Response — 200 OK**: always ``{}``, regardless of whether the username exists, to prevent username-enumeration attacks. The underlying call to bin-agent-manager is made best-effort; a lookup failure is logged but never surfaced to the caller.

**Errors**

.. list-table::
   :header-rows: 1
   :widths: 15 85

   * - Status
     - Cause
   * - 400
     - ``username`` missing or malformed JSON body.


Password Reset — ``GET`` / ``POST /auth/password-reset``
--------------------------------------------------------------
Completes a password reset using the token emailed by ``POST /auth/password-forgot``.

**GET /auth/password-reset** — serves the HTML reset form (the link target in the reset email).

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Parameter
     - Location
     - Description
   * - ``token``
     - Query, Required
     - 64-character lowercase hex string. An invalid/missing token returns ``400`` before the form is rendered.

**POST /auth/password-reset** — request body

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Field
     - Type
     - Description
   * - ``token``
     - String, Required
     - 64-character lowercase hex reset token from the email. Single-use, expires 1 hour after issuance.
   * - ``password``
     - String, Required
     - New password. Minimum 8 characters.

**Response — 200 OK**: ``{}``.

**Errors**

.. list-table::
   :header-rows: 1
   :widths: 15 85

   * - Status
     - Cause
   * - 400
     - ``token`` missing/invalid/expired, ``password`` missing or under 8 characters, malformed JSON body, or (per upstream agent-manager policy) the token belongs to a guest agent — guest agents cannot reset their password.


Unregister (Self-Service Deletion) — ``POST`` / ``DELETE /auth/unregister``
-------------------------------------------------------------------------------
Self-service account freeze/deletion and recovery. Requires authentication (Token or Accesskey) and ``PermissionCustomerAdmin`` on the caller's own customer account; direct (boot) tokens cannot call this endpoint. A condensed narrative version of this section, including the full status-lifecycle diagram and the cascade-delete resource list, lives in :ref:`Customer — Account Deletion Lifecycle <customer-main>`; this section is the field-level contract.

**POST /auth/unregister — request body**

.. list-table::
   :header-rows: 1
   :widths: 22 12 66

   * - Field
     - Type
     - Description
   * - ``password``
     - String, Conditional
     - Re-authentication password for password-based accounts. Mutually exclusive with ``confirmation_phrase`` — exactly one of the two must be supplied.
   * - ``confirmation_phrase``
     - String, Conditional
     - Must be exactly ``"DELETE"``. Used for SSO/accesskey-authenticated requests that have no password to re-check.
   * - ``immediate``
     - Boolean, Optional
     - Default ``false``. If ``true``, skips the 30-day grace period: the account is frozen and then permanently deleted (PII anonymized, all resources cascade-deleted) in the same request. Irreversible.

**Query parameters (both POST and DELETE)**

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Parameter
     - Location
     - Description
   * - ``accesskey``
     - Query, Optional
     - Access-key token, as an alternative to a ``Bearer`` token / ``token=`` credential.

**Response — 200 OK**: the updated ``Customer`` object (see :ref:`Customer <customer-main>`), reflecting ``status: "frozen"`` (or ``"deleted"`` if ``immediate`` was ``true``).

**Errors**

.. list-table::
   :header-rows: 1
   :widths: 15 85

   * - Status
     - Cause
   * - 400
     - Neither or both of ``password``/``confirmation_phrase`` supplied; password re-authentication failed; ``confirmation_phrase`` is not exactly ``"DELETE"``; malformed JSON body; caller lacks ``PermissionCustomerAdmin``; caller is authenticated via a direct (boot) token (``DIRECT_ACCESS_NOT_SUPPORTED``); or the downstream freeze/delete call failed. The handler returns a bare ``400`` for all of these cases — it does not distinguish permission/direct-access failures with a ``403``, unlike ``POST /auth/delegate``.
   * - 401
     - Missing/invalid/expired token or access key (returned by the shared ``Authenticate()`` middleware, as a structured error envelope — see the response-shape note above).

**DELETE /auth/unregister** — cancels a scheduled deletion and restores ``active`` status. No request body. Only works while the account is ``frozen`` within the 30-day grace period.

**Response — 200 OK**: the restored ``Customer`` object.

**Errors**

.. list-table::
   :header-rows: 1
   :widths: 15 85

   * - Status
     - Cause
   * - 400
     - Account is not currently in ``frozen`` state; caller lacks ``PermissionCustomerAdmin``; or caller is authenticated via a direct (boot) token. As with ``POST /auth/unregister`` above, the handler returns a bare ``400`` for all of these cases rather than a ``403``.
   * - 401
     - Missing/invalid/expired token or access key.

.. note:: **AI Implementation Hint — frozen accounts are also enforced globally**

   Once a customer account is ``frozen``, every other authenticated ``/v1.0/*`` request from that customer (except from a ``PermissionProjectSuperAdmin``, and except direct/boot tokens, which skip the check entirely) is rejected with ``403 ACCOUNT_FROZEN`` by the shared authentication middleware — not just calls to resource endpoints that would otherwise mutate data. The error's ``details[0]`` carries ``deletion_scheduled_at``, ``deletion_effective_at`` (30 days after scheduling), and ``recovery_endpoint: "DELETE /auth/unregister"`` so client UIs (admin/talk consoles) can render a consistent "account frozen, recover here" screen. ``POST`` and ``DELETE /auth/unregister`` themselves are explicitly exempted from this block so a frozen customer can still self-recover.


Delegate (Superadmin Support Access) — ``POST /auth/delegate``
--------------------------------------------------------------------
Lets a platform superadmin (``PermissionProjectSuperAdmin``) issue a short-lived, audit-logged token that grants ``PermissionCustomerAdmin``-equivalent access scoped to a specific target customer, for support/investigation without needing that customer's own credentials. Fully documented with request/response examples in the :ref:`Authentication quickstart <quickstart-authentication>`; summarized here for completeness.

**Request body**

.. list-table::
   :header-rows: 1
   :widths: 20 12 68

   * - Field
     - Type
     - Description
   * - ``customer_id``
     - String (UUID), Required
     - Target customer to act on behalf of.
   * - ``reason``
     - String, Required
     - Justification, written to the audit log. Must be 10-200 printable-ASCII characters (0x20-0x7E), no control characters.

**Response — 200 OK**

.. code::

    {
        "token": "eyJ...",
        "customer_id": "...",
        "expire": "2026-05-19T06:00:00.000000Z"
    }

The delegate token is valid for **8 hours** and grants customer-admin-equivalent access scoped only to ``customer_id`` — no project-level permissions are carried over.

**Errors**

.. list-table::
   :header-rows: 1
   :widths: 15 85

   * - Status
     - Cause
   * - 400
     - Malformed or missing request body.
   * - 401
     - Caller is not authenticated.
   * - 403
     - Caller lacks ``PermissionProjectSuperAdmin``; or the caller is itself authenticated via a delegate token (recursive delegation is blocked).
   * - 404
     - Target customer does not exist or is already ``deleted``.
   * - 422
     - ``customer_id`` is not a valid UUID, or ``reason`` fails length/character validation.

.. note:: **AI Implementation Hint**

   Every delegate token issuance (and denial) is written to the audit log with ``audit=true``, the issuer's agent ID, target customer, reason, and expiry — this is the audit trail for superadmin account access. Do not build tooling that calls this endpoint without a genuine, specific support reason; the ``reason`` field is not free-form filler, it is the compliance record.


Related Documentation
----------------------

- :ref:`Signup quickstart <quickstart-signup>` — narrative walkthrough with example requests
- :ref:`Authentication quickstart <quickstart-authentication>` — token/accesskey/boot/delegate walkthrough
- :ref:`Customer Overview <customer-main>` — full account status lifecycle and cascade-delete details
- :ref:`Accesskey <accesskey-main>` — long-lived programmatic credentials
- :ref:`Error Reason Codes <error-reason-catalog>` — full catalogue of ``error.reason`` values
