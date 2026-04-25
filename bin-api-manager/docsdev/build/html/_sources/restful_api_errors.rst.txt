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

.. list-table::
   :header-rows: 1
   :widths: 30 10 60

   * - Reason
     - HTTP
     - Cause → Fix

   (Populated as migration PR 5 ships.)

Other Domains
-------------

Reason code sections for the remaining manager services — ``number-manager``, ``conversation-manager``, ``email-manager``, ``ai-manager``, ``transcribe-manager``, ``talk-manager``, ``agent-manager``, ``queue-manager``, ``conference-manager``, ``campaign-manager``, ``storage-manager``, ``tag-manager``, ``team-manager``, ``timeline-manager``, ``contact-manager``, ``rag-manager``, ``provider-manager``, ``route-manager``, ``trunk-manager`` — will be added as migration PRs 3–9 ship. See the design doc for the PR rollout.
