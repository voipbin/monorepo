.. _error-reason-catalog:

Error Reason Codes
==================

.. note:: **AI Context**

   Every 4xx/5xx response from the VoIPbin API contains an ``error.reason`` field in ``UPPER_SNAKE`` that identifies the specific cause. This page catalogues all published reasons grouped by ``domain``. Clients should branch on ``reason`` for self-healing behavior. ``status`` maps 1:1 to HTTP (see :doc:`restful_api`).

   Reason codes are append-only once published. Adding a new reason does not require a schema version bump; removing or renaming one does and triggers a deprecation window.

api-manager
-----------

.. list-table::
   :header-rows: 1
   :widths: 30 10 60

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``AUTHENTICATION_REQUIRED``
     - 401
     - No token or access key was supplied. Send a valid JWT in the ``Authorization: Bearer ...`` header, or a valid access key.
   * - ``INVALID_CREDENTIALS``
     - 401
     - The supplied token or access key is invalid or expired. Refresh the token via ``POST https://api.voipbin.net/v1.0/auth/login`` and retry.
   * - ``ACCOUNT_FROZEN``
     - 403
     - The customer account is frozen (typically because unregister was scheduled). Inspect ``error.details[0].recovery_endpoint`` and call it to restore the account, or wait past ``deletion_effective_at``.
   * - ``PERMISSION_DENIED``
     - 403
     - The authenticated user does not have permission for this resource.
   * - ``DIRECT_ACCESS_NOT_SUPPORTED``
     - 403
     - The endpoint requires a JWT login flow that the current direct access (access key) cannot satisfy. Re-authenticate via JWT.
   * - ``RATE_LIMIT_EXCEEDED``
     - 429
     - Client exceeded the rate limit for this endpoint. Back off and retry with exponential delay.
   * - ``RESOURCE_NOT_FOUND``
     - 404
     - The requested resource does not exist or does not belong to the authenticated customer. Verify the UUID was obtained from a recent ``GET`` list call.
   * - ``ROUTE_NOT_FOUND``
     - 404
     - The requested HTTP endpoint does not exist (wrong path or typo). Verify the URL against the API reference; check API version prefix (``/v1.0/``) and resource name.
   * - ``INVALID_ARGUMENT``
     - 400
     - The request body or a path/query parameter is invalid. Inspect ``error.message`` for details.
   * - ``INVALID_JSON_BODY``
     - 400
     - The request body is not valid JSON. Ensure ``Content-Type`` is ``application/json`` and the payload parses as a JSON object or array.
   * - ``INVALID_ID``
     - 400
     - A path or body parameter is not a valid UUID. Verify the ID was obtained from a recent ``GET`` list call and has the form ``a1b2c3d4-e5f6-7890-abcd-ef1234567890``.
   * - ``REQUEST_TIMEOUT``
     - 503
     - Upstream manager did not respond within the deadline. Retry with the same idempotency key after a short delay.
   * - ``REQUEST_CANCELED``
     - 503
     - The request was canceled before completion (typically the client disconnected). Usually safe to ignore; the server did not finish processing.
   * - ``SERVICE_UNAVAILABLE``
     - 503
     - An upstream manager service is temporarily unavailable. Retry the request with the same idempotency key after a short delay (exponential backoff).
   * - ``INTERNAL``
     - 500
     - Unexpected server error. Include ``error.request_id`` when opening a support ticket.

call-manager
------------

.. list-table::
   :header-rows: 1
   :widths: 30 10 60

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``CALL_NOT_FOUND``
     - 404
     - Call ID does not exist or belongs to another customer. Verify the ID was obtained from a recent ``GET /calls`` list call.
   * - ``CALL_ALREADY_HANGUP``
     - 409
     - Operation invalid because the call has already ended. Check current status via ``GET /calls/{id}`` before retrying.
   * - ``CALL_STATE_INVALID``
     - 409
     - Operation invalid for the current call state (e.g., transfer from a call that is not yet progressing). Check ``status`` via ``GET /calls/{id}`` and retry when the state matches the operation's prerequisite.
   * - ``RECORDING_NOT_FOUND``
     - 404
     - Recording ID does not exist or belongs to another customer. Verify via ``GET /recordings``.
   * - ``RECORDING_ALREADY_ACTIVE``
     - 409
     - A recording is already in progress on this call. Stop the existing recording with ``POST /calls/{id}/recording-stop`` before starting a new one.
   * - ``RECORDING_NOT_ACTIVE``
     - 409
     - No recording is active on this call. Start one with ``POST /calls/{id}/recording-start`` before attempting to stop.
   * - ``INSUFFICIENT_BALANCE``
     - 402
     - Customer balance is below the minimum required for this operation. Top up via the customer billing portal, then retry.
   * - ``GROUPCALL_NOT_FOUND``
     - 404
     - Groupcall ID does not exist or belongs to another customer. Verify via ``GET /groupcalls``.

.. note::

   The reasons in this section define the platform's planned typed-error contract.
   Reasons that match the translator's case-insensitive substring fallback (``CALL_NOT_FOUND``, ``RECORDING_NOT_FOUND``, ``GROUPCALL_NOT_FOUND``, ``CALL_ALREADY_HANGUP``, ``RECORDING_ALREADY_ACTIVE``, ``RECORDING_NOT_ACTIVE``, ``INSUFFICIENT_BALANCE`` via ``"already"`` / ``"deleted"`` / ``"not active"`` / ``"insufficient"`` / ``"not found"`` patterns) surface today.
   Domain-specific reason codes will be emitted directly once the servicehandler typed-error migration ships; until then, the translator routes the underlying ``fmt.Errorf`` strings through the substring fallback to the closest canonical reason (typically the api-manager generic equivalent, e.g., ``RESOURCE_NOT_FOUND`` rather than ``CALL_NOT_FOUND``).

flow-manager
------------

.. note::

   The reasons in this section define the platform's planned typed-error contract.
   Reasons reachable today via the translator's case-insensitive substring fallback:
   ``FLOW_NOT_FOUND`` and ``ACTIVEFLOW_NOT_FOUND`` (via ``"not found"`` pattern → currently surface as ``RESOURCE_NOT_FOUND`` in the api-manager domain),
   ``ACTIVEFLOW_ALREADY_STOPPED`` (via ``"already"`` pattern added in PR 2 → currently surfaces as ``STATE_INVALID`` in the api-manager domain).
   ``FLOW_STATE_INVALID`` (the generic state-restriction reason) is not currently reachable via fallback — it requires the typed-error migration to emit directly.
   Domain-specific reason codes will be emitted directly once the servicehandler typed-error migration ships.

   Note: ``POST /activeflows/{id}/stop`` is idempotent in production today — stopping an already-stopped activeflow returns 200 (no-op).
   The 409 ``ACTIVEFLOW_ALREADY_STOPPED`` response declared in the OpenAPI spec is forward-compatible for clients that prefer opt-in idempotency-aware semantics; it will be emitted once the typed-error migration ships.

.. list-table::
   :header-rows: 1
   :widths: 25 10 65

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``FLOW_NOT_FOUND``
     - 404
     - Flow ID does not exist or belongs to another customer. Verify the ID was obtained from a recent ``GET /flows`` list call.
   * - ``ACTIVEFLOW_NOT_FOUND``
     - 404
     - Activeflow ID does not exist or belongs to another customer. Verify via ``GET /activeflows``.
   * - ``FLOW_STATE_INVALID``
     - 409
     - Operation invalid for the current activeflow state (e.g., stop on an already-stopped activeflow). Check current status via ``GET /activeflows/{id}`` before retrying.
   * - ``ACTIVEFLOW_ALREADY_STOPPED``
     - 409
     - Stop requested on an activeflow that has already terminated. Idempotent — treat as success or check status before retrying.

billing-manager
---------------

.. note::

   ``INSUFFICIENT_BALANCE`` is reachable today via the translator's case-insensitive ``"insufficient"`` substring fallback — billing-manager surfaces credit-shortfall errors as ``fmt.Errorf("insufficient balance")`` (or similar) and the api-manager translator maps that to **402 PAYMENT_REQUIRED** with reason ``INSUFFICIENT_BALANCE`` in the api-manager domain.
   The full billing-manager typed-error contract (deeper reasons such as ``BILLING_ACCOUNT_NOT_FOUND``, ``ACCOUNT_SUSPENDED``) is scheduled for migration PR 5.

.. list-table::
   :header-rows: 1
   :widths: 30 10 60

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``INSUFFICIENT_BALANCE``
     - 402
     - Customer balance is below the minimum required for a chargeable operation. Currently fired by ``POST /numbers`` (number purchase) and ``POST /numbers/renew`` (number renewal); migration PR 6 will extend the list to ``POST /messages`` and ``POST /emails``. **Fix:** Top up the customer balance via ``POST /billing-accounts/{id}/balance-add`` (admin) or have the customer add credit, then retry.

number-manager
--------------

.. note::

   ``NUMBER_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the number-manager typed-error migration ships.
   ``IDENTITY_VERIFICATION_REQUIRED`` is wired in PR 4 via the dedicated ``"identity verification required"`` translator pattern and is reachable today.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``NUMBER_NOT_FOUND``
     - 404
     - Number ID does not exist or belongs to another customer. Verify the ID was obtained from a recent ``GET /numbers`` list call.
   * - ``IDENTITY_VERIFICATION_REQUIRED``
     - 403
     - Customer identity verification is required for this operation. Fired by ``POST /numbers`` for non-virtual purchases when the customer's ``identity_verification_status`` is not ``verified``. **Fix:** Complete the customer identity verification flow (see Customer overview), then retry the purchase. Virtual numbers do not require verification.

provisioning-admin
------------------

The ``provider``, ``trunk``, and ``route`` resources are admin-gated and share a common reason-code pattern. Listed together to avoid repetition.

.. note::

   The ``*_NOT_FOUND`` reasons in this section are reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reasons will be emitted directly once the corresponding manager's typed-error migration ships.
   These resources are admin-gated. Non-admin callers receive **403 PERMISSION_DENIED** via the standard ``"no permission"`` translator pattern (the call site falls back to the api-manager generic ``PERMISSION_DENIED`` reason — there is no resource-specific permission reason for these admin endpoints).
   Admin-gated endpoints (``/providers``, ``/providers/setup``, ``/providercalls``, ``/trunks``, ``/routes`` and their sub-paths) return 403 ``PERMISSION_DENIED`` for non-admin callers via the standard ``"no permission"`` translator pattern. The OpenAPI spec declares 403 on these paths to reflect this runtime behavior.

.. list-table::
   :header-rows: 1
   :widths: 30 10 60

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``PROVIDER_NOT_FOUND``
     - 404
     - Provider ID does not exist. Verify the ID was obtained from a recent ``GET /providers`` list call. Admin-only resource.
   * - ``TRUNK_NOT_FOUND``
     - 404
     - Trunk ID does not exist. Verify via ``GET /trunks``. Admin-only resource.
   * - ``ROUTE_NOT_FOUND``
     - 404
     - Route ID does not exist. Verify via ``GET /routes``. Admin-only resource.

Other Domains
-------------

Reason code sections for the remaining manager services — ``conversation-manager``, ``email-manager``, ``ai-manager``, ``transcribe-manager``, ``talk-manager``, ``agent-manager``, ``queue-manager``, ``conference-manager``, ``campaign-manager``, ``storage-manager``, ``tag-manager``, ``team-manager``, ``timeline-manager``, ``contact-manager``, ``rag-manager`` — will be added as migration PRs 5–9 ship. See the design doc for the PR rollout.
