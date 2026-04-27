.. _error-reason-catalog:

Error Reason Codes
==================

.. note:: **AI Context**

   Every 4xx/5xx response from the VoIPbin API contains an ``error.reason`` field in ``UPPER_SNAKE`` that identifies the specific cause. This page catalogues all published reasons grouped by ``domain``. Clients should branch on ``reason`` for self-healing behavior. ``status`` maps 1:1 to HTTP (see :doc:`restful_api`).

   Reason codes are append-only once published. Adding a new reason does not require a schema version bump; removing or renaming one does and triggers a deprecation window.

.. note:: **Rollout status (2026-04, complete)**

   All ``bin-api-manager/server/*.go`` files uniformly emit the canonical error envelope. Every 4xx/5xx response from the API gateway carries the ``status`` / ``reason`` / ``domain`` / ``message`` / ``request_id`` fields documented on this page. The api-manager servicehandler layer emits typed sentinels (``serviceerrors.Err*``) and the legacy substring-fallback translator step has been removed — any unmatched error correctly degrades to ``500 INTERNAL`` via the default branch. Domain-specific reasons (``CALL_NOT_FOUND``, ``FLOW_NOT_FOUND``, etc.) flow through end-to-end when the upstream manager emits ``*cerrors.VoipbinError`` with the domain-specific reason: the api-manager translator's typed-passthrough step (``errors.As``) recovers the upstream typed error even when wrapped by ``pkg/errors.Wrapf``. When the upstream returns a generic / untyped error, the api-manager servicehandler maps it to a sentinel and the translator surfaces the api-manager generic reason (``RESOURCE_NOT_FOUND``, ``STATE_INVALID``, ``INSUFFICIENT_BALANCE``, etc.).

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

   The reasons ``CALL_NOT_FOUND``, ``RECORDING_NOT_FOUND``, and ``GROUPCALL_NOT_FOUND`` are emitted today by call-manager (via ``cerrors.NotFound("call-manager", ...)``) and flow through end-to-end on Get-by-ID paths. The remaining reasons (``CALL_ALREADY_HANGUP``, ``RECORDING_ALREADY_ACTIVE``, ``RECORDING_NOT_ACTIVE``, ``INSUFFICIENT_BALANCE``) define the platform's planned state-transition / billing-precheck typed-error contract; until call-manager emits them directly, the api-manager servicehandler maps the underlying conditions to sentinels (``serviceerrors.ErrStateInvalid``, ``serviceerrors.ErrInsufficientBalance``) and the translator surfaces them as the api-manager generic equivalents (``STATE_INVALID``, ``INSUFFICIENT_BALANCE``).
   PR 13 added the agent-surface read endpoints (``GET /service_agents/calls`` and ``GET /service_agents/calls/{id}``); these reuse ``CALL_NOT_FOUND`` for not-found semantics on the by-ID read.

flow-manager
------------

.. note::

   The reasons ``FLOW_NOT_FOUND`` and ``ACTIVEFLOW_NOT_FOUND`` are emitted today by flow-manager (via ``cerrors.NotFound("flow-manager", ...)``) and flow through end-to-end on Get-by-ID paths. The state-transition reasons ``ACTIVEFLOW_ALREADY_STOPPED`` and ``FLOW_STATE_INVALID`` define the platform's planned typed-error contract; until flow-manager emits them directly, the api-manager servicehandler maps the underlying conditions to ``serviceerrors.ErrStateInvalid`` and the translator surfaces them as the api-manager generic ``STATE_INVALID``.

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

   ``INSUFFICIENT_BALANCE`` is reachable today as **402 PAYMENT_REQUIRED** with reason ``INSUFFICIENT_BALANCE`` in the api-manager domain.
   ``BILLING_NOT_FOUND`` and ``BILLING_ACCOUNT_NOT_FOUND`` are emitted today by billing-manager (via ``cerrors.NotFound("billing-manager", ...)``) and surface end-to-end on Get-by-ID lookups for non-existent resources.
   The full billing-manager typed-error contract (deeper reasons such as ``ACCOUNT_SUSPENDED``) remains scheduled for a future migration.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``INSUFFICIENT_BALANCE``
     - 402
     - Customer balance is below the minimum required for a chargeable operation. Currently fired by ``POST /numbers`` (number purchase), ``POST /numbers/renew`` (number renewal), ``POST /messages`` (SMS send), and ``POST /emails`` (email send). **Fix:** Top up the customer balance via ``POST /billing-accounts/{id}/balance-add`` (admin) or have the customer add credit, then retry. Future endpoints (deferred): ``POST /aimessages``, ``POST /conversations/{id}/messages``, ``POST /service-agents/conversations/{id}/messages``, ``POST /speakings`` (TTS character cost), ``POST /transcribes`` (STT second cost) — pending balance pre-check wiring in ai-manager, conversation-manager, tts-manager, and transcribe-manager.
   * - ``BILLING_NOT_FOUND``
     - 404
     - Billing record ID does not exist or belongs to another customer. Fired by ``GET /billings/{billing_id}``. **Fix:** Verify the ID was obtained from a recent ``GET /billings`` list call.
   * - ``BILLING_ACCOUNT_NOT_FOUND``
     - 404
     - Billing account ID does not exist or belongs to another customer. Fired by the ``/billing-accounts/{id}*`` admin endpoints. **Fix:** Verify the ID was obtained from a recent ``GET /billing-accounts`` list call.

number-manager
--------------

.. note::

   ``NUMBER_NOT_FOUND`` is emitted today by number-manager (via ``cerrors.NotFound("number-manager", ...)``) and surfaces end-to-end on Get-by-ID lookups for non-existent resources. For soft-deleted entries, the api-manager servicehandler intercepts via a post-fetch ``TMDelete`` check (``pkg/servicehandler/numbers.go``) and re-emits ``serviceerrors.ErrNotFound``, which surfaces as the api-manager generic ``RESOURCE_NOT_FOUND`` — the typed ``NUMBER_NOT_FOUND`` is not reached on that path.
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

   ``ROUTE_NOT_FOUND``, ``PROVIDER_NOT_FOUND``, and ``PROVIDERCALL_NOT_FOUND`` are emitted today by route-manager, and ``TRUNK_NOT_FOUND`` is emitted today by registrar-manager (via ``cerrors.NotFound("<service>", ...)``); all surface end-to-end on Get-by-ID lookups for non-existent resources.
   These resources are admin-gated. Non-admin callers receive **403 PERMISSION_DENIED** via ``serviceerrors.ErrPermissionDenied``.
   Admin-gated endpoints (``/providers``, ``/providers/setup``, ``/providercalls``, ``/trunks``, ``/routes`` and their sub-paths) return 403 ``PERMISSION_DENIED`` for non-admin callers. The OpenAPI spec declares 403 on these paths to reflect this runtime behavior.

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
   * - ``PROVIDERCALL_NOT_FOUND``
     - 404
     - Provider-call ID does not exist. Verify via ``GET /providercalls``. Admin-only resource.

storage-manager
---------------

.. note::

   ``ACCOUNT_NOT_FOUND`` and ``FILE_NOT_FOUND`` are emitted today by storage-manager (via ``cerrors.NotFound("storage-manager", ...)``) and surface end-to-end on Get-by-ID lookups for non-existent resources. For soft-deleted entries, the api-manager servicehandler intercepts via a post-fetch ``TMDelete`` check (``pkg/servicehandler/storage_file.go``) and re-emits ``serviceerrors.ErrNotFound``, which surfaces as the api-manager generic ``RESOURCE_NOT_FOUND`` — the typed ``FILE_NOT_FOUND`` is not reached on that path.
   The admin-gated ``POST /storage-accounts`` endpoint returns **403 PERMISSION_DENIED** for non-admin callers via ``serviceerrors.ErrPermissionDenied``. The OpenAPI spec declares 403 on this path to reflect runtime behavior.
   PR 13 added the agent-surface file endpoints (``/service_agents/files`` and ``/service_agents/files/{id}*``); these reuse ``FILE_NOT_FOUND`` for not-found semantics on agent-scoped reads, deletes, and downloads.
   PR 15 added the remaining public file endpoints (``/storage_files``, ``/storage_files/{id}``, and ``/storage_files/{id}/file``) under the same ``FILE_NOT_FOUND`` semantics for not-found cases on the customer-scoped read, delete, and download surfaces.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``ACCOUNT_NOT_FOUND``
     - 404
     - Storage account ID does not exist or belongs to another customer. Fired by ``GET /storage-accounts/{id}`` and ``DELETE /storage-accounts/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /storage-accounts`` list call.
   * - ``FILE_NOT_FOUND``
     - 404
     - Storage file ID does not exist or belongs to another customer. Fired by ``GET /storage_files/{id}``, ``DELETE /storage_files/{id}``, ``GET /storage_files/{id}/file``, and the agent-surface counterparts ``GET /service_agents/files/{id}``, ``DELETE /service_agents/files/{id}``, and ``GET /service_agents/files/{id}/file``. **Fix:** Verify the ID was obtained from a recent ``GET /storage_files`` (admin) or ``GET /service_agents/files`` (agent) list call.

conversation-manager
--------------------

.. note::

   ``CONVERSATION_NOT_FOUND``, ``CONVERSATION_MESSAGE_NOT_FOUND``, and ``CONVERSATION_ACCOUNT_NOT_FOUND`` are emitted today by conversation-manager (via ``cerrors.NotFound("conversation-manager", ...)``) and surface end-to-end on Get-by-ID lookups for non-existent resources.
   The admin-gated ``POST /conversation-accounts`` endpoint returns **403 PERMISSION_DENIED** for non-admin callers via the standard ``"no permission"`` translator pattern. The OpenAPI spec declares 403 on this path to reflect runtime behavior.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``CONVERSATION_ACCOUNT_NOT_FOUND``
     - 404
     - Conversation account ID does not exist or belongs to another customer. Fired by ``GET /conversation-accounts/{id}``, ``PUT /conversation-accounts/{id}``, and ``DELETE /conversation-accounts/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /conversation-accounts`` list call.
   * - ``CONVERSATION_NOT_FOUND``
     - 404
     - Conversation ID does not exist or belongs to another customer. Fired by ``GET /conversations/{id}``, ``PUT /conversations/{id}``, ``GET /conversations/{id}/messages``, ``POST /conversations/{id}/messages``, and the ``/service-agents/conversations/{id}*`` agent-surface endpoints. **Fix:** Verify the ID was obtained from a recent ``GET /conversations`` list call.

message-manager
---------------

.. note::

   ``MESSAGE_NOT_FOUND`` is emitted today by message-manager (via ``cerrors.NotFound("message-manager", ...)``) and surfaces end-to-end on Get-by-ID lookups for non-existent resources. For soft-deleted entries, the api-manager servicehandler intercepts via a post-fetch ``TMDelete`` check (``pkg/servicehandler/message.go``) and re-emits ``serviceerrors.ErrNotFound``, which surfaces as the api-manager generic ``RESOURCE_NOT_FOUND`` — the typed ``MESSAGE_NOT_FOUND`` is not reached on that path.
   ``POST /messages`` is billing-sensitive — see the ``billing-manager`` section above for the ``INSUFFICIENT_BALANCE`` (402) contract that applies to SMS send.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``MESSAGE_NOT_FOUND``
     - 404
     - Message ID does not exist or belongs to another customer. Fired by ``GET /messages/{id}`` and ``DELETE /messages/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /messages`` list call.

email-manager
-------------

.. note::

   ``EMAIL_NOT_FOUND`` is emitted today by email-manager (via ``cerrors.NotFound("email-manager", ...)``) and surfaces end-to-end on Get-by-ID lookups for non-existent resources. For soft-deleted entries, the api-manager servicehandler intercepts via a post-fetch ``TMDelete`` check (``pkg/servicehandler/email.go``) and re-emits ``serviceerrors.ErrNotFound``, which surfaces as the api-manager generic ``RESOURCE_NOT_FOUND`` — the typed ``EMAIL_NOT_FOUND`` is not reached on that path.
   ``POST /emails`` is billing-sensitive — see the ``billing-manager`` section above for the ``INSUFFICIENT_BALANCE`` (402) contract that applies to email send.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``EMAIL_NOT_FOUND``
     - 404
     - Email ID does not exist or belongs to another customer. Fired by ``GET /emails/{id}`` and ``DELETE /emails/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /emails`` list call.

ai-manager
----------

.. note::

   PR 6 introduced this section for the AI-message resource (``/aimessages``); PR 7 extended it with the AI configuration (``/ais``), AI calls (``/aicalls``), and AI summaries (``/aisummaries``) resources; PR 9 extends it with the AI team resource (``/teams``).
   ``AI_NOT_FOUND``, ``AICALL_NOT_FOUND``, ``SUMMARY_NOT_FOUND``, and ``TEAM_NOT_FOUND`` are emitted today by ai-manager (via ``cerrors.NotFound("ai-manager", ...)``) and surface end-to-end on Get-by-ID lookups for non-existent resources. ``AIMESSAGE_NOT_FOUND`` defines the planned typed-error contract for the aimessage sub-resource; until ai-manager emits it directly, the api-manager servicehandler maps the underlying not-found condition to ``serviceerrors.ErrNotFound`` and the translator surfaces the api-manager generic ``RESOURCE_NOT_FOUND``.
   ``POST /aimessages``, ``POST /aicalls``, and ``POST /aisummaries`` are conceptually billing-sensitive (LLM token cost, voice AI minutes, summary generation cost), but bin-ai-manager has **no balance pre-check today** — the 402 ``INSUFFICIENT_BALANCE`` contract is **not reachable** for AI resource creation. Wiring the pre-check in ai-manager is deferred to a follow-up PR; no 402 declarations were added in PR 7 to match runtime behavior.
   Team write surfaces (``POST /teams``, ``PUT /teams/{id}``, ``DELETE /teams/{id}``, ``POST /teams/{id}/direct_hash_regenerate``) are admin-gated; non-admin callers receive 403 ``PERMISSION_DENIED`` via ``serviceerrors.ErrPermissionDenied``.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``AIMESSAGE_NOT_FOUND``
     - 404
     - AI message ID does not exist or belongs to another customer. Fired by ``GET /aimessages/{id}`` and ``DELETE /aimessages/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /aimessages`` list call.
   * - ``AI_NOT_FOUND``
     - 404
     - AI configuration ID does not exist or belongs to another customer. Fired by ``GET /ais/{id}``, ``PUT /ais/{id}``, ``DELETE /ais/{id}``, and ``POST /ais/{id}/direct_hash_regenerate``. **Fix:** Verify the ID was obtained from a recent ``GET /ais`` list call.
   * - ``AICALL_NOT_FOUND``
     - 404
     - AI call ID does not exist or belongs to another customer. Fired by ``GET /aicalls/{id}`` and ``DELETE /aicalls/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /aicalls`` list call.
   * - ``SUMMARY_NOT_FOUND``
     - 404
     - AI summary ID does not exist or belongs to another customer. Fired by ``GET /aisummaries/{id}`` and ``DELETE /aisummaries/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /aisummaries`` list call.
   * - ``TEAM_NOT_FOUND``
     - 404
     - Team ID does not exist or belongs to another customer. Fired by ``GET /teams/{id}``, ``PUT /teams/{id}``, ``DELETE /teams/{id}``, and ``POST /teams/{id}/direct_hash_regenerate``. **Fix:** Verify the ID was obtained from a recent ``GET /teams`` list call.

rag-manager
-----------

.. note::

   ``RAG_NOT_FOUND`` and ``DOCUMENT_NOT_FOUND`` are emitted today by rag-manager (via ``cerrors.NotFound("rag-manager", ...)``) and surface end-to-end on Get-by-ID lookups (the api-manager servicehandler does not intercept soft-deleted entries for this resource type).

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``RAG_NOT_FOUND``
     - 404
     - RAG knowledge-base ID does not exist or belongs to another customer. Fired by ``GET /rags/{id}``, ``PUT /rags/{id}``, ``DELETE /rags/{id}``, ``POST /rags/{id}/sources``, and ``DELETE /rags/{id}/sources/{source_id}``. **Fix:** Verify the ID was obtained from a recent ``GET /rags`` list call.
   * - ``DOCUMENT_NOT_FOUND``
     - 404
     - RAG document/source ID does not exist or belongs to another customer. Fired by ``DELETE /rags/{id}/sources/{source_id}`` when the source is missing. **Fix:** Verify the source ID was obtained from a recent ``GET /rags/{id}`` response.

tts-manager
-----------

.. note::

   ``SPEAKING_NOT_FOUND`` is emitted today by tts-manager (via ``cerrors.NotFound("tts-manager", ...)``) and surfaces end-to-end on Get-by-ID lookups (the api-manager servicehandler does not intercept soft-deleted entries for this resource type).
   ``POST /speakings/{id}/stop`` is **idempotent** in tts-manager today — stopping an already-stopped speaking session returns success (no-op). The session-state-restriction reason ``SPEAKING_STATE_INVALID`` (409) is **not declared** on ``/speakings/{id}/stop`` because the underlying handler does not surface a state error. A forward-compatible 409 declaration may be added once the typed-error migration ships and explicit state-transition typing is introduced.
   ``POST /speakings`` is conceptually billing-sensitive (TTS character cost) but tts-manager has **no balance pre-check today** — the 402 ``INSUFFICIENT_BALANCE`` contract is **not reachable** for speaking-session creation. Wiring the pre-check is deferred to a follow-up PR; see the ``billing-manager`` section above for the deferred list.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``SPEAKING_NOT_FOUND``
     - 404
     - Speaking session ID does not exist or belongs to another customer. Fired by ``GET /speakings/{id}``, ``DELETE /speakings/{id}``, ``POST /speakings/{id}/flush``, ``POST /speakings/{id}/say``, and ``POST /speakings/{id}/stop``. **Fix:** Verify the ID was obtained from a recent ``GET /speakings`` list call or from the response of ``POST /speakings``.

transcribe-manager
------------------

.. note::

   ``TRANSCRIBE_NOT_FOUND`` is emitted today by transcribe-manager (via ``cerrors.NotFound("transcribe-manager", ...)``) and surfaces end-to-end on Get-by-ID lookups (the api-manager servicehandler does not intercept soft-deleted entries for this resource type).
   ``POST /transcribes/{id}/stop`` is **idempotent** in transcribe-manager today — stopping an already-stopped transcription returns success (no-op). The session-state-restriction reason ``TRANSCRIBE_STATE_INVALID`` (409) is **not declared** on ``/transcribes/{id}/stop`` because the underlying handler does not surface a state error. A forward-compatible 409 declaration may be added once the typed-error migration ships and explicit state-transition typing is introduced.
   ``POST /transcribes`` is conceptually billing-sensitive (STT second cost) but transcribe-manager has **no balance pre-check today** — the 402 ``INSUFFICIENT_BALANCE`` contract is **not reachable** for transcription creation. Wiring the pre-check is deferred to a follow-up PR; see the ``billing-manager`` section above for the deferred list.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``TRANSCRIBE_NOT_FOUND``
     - 404
     - Transcribe ID does not exist or belongs to another customer. Fired by ``GET /transcribes/{id}``, ``DELETE /transcribes/{id}``, and ``POST /transcribes/{id}/stop``. **Fix:** Verify the ID was obtained from a recent ``GET /transcribes`` list call or from the response of ``POST /transcribes``.

agent-manager
-------------

.. note::

   PR 9 introduced this section for the agent resource (``/agents``).
   ``AGENT_NOT_FOUND`` is emitted today by agent-manager (via ``cerrors.NotFound("agent-manager", ...)``) and surfaces end-to-end on Get-by-ID lookups (the api-manager servicehandler does not intercept soft-deleted entries for this resource type).
   Agent write surfaces (``POST /agents``, ``PUT /agents/{id}``, ``PUT /agents/{id}/permission``, ``PUT /agents/{id}/password``, ``PUT /agents/{id}/addresses``, ``PUT /agents/{id}/tag_ids``, ``PUT /agents/{id}/status``, ``DELETE /agents/{id}``, ``POST /agents/{id}/direct_hash_regenerate``) are admin-gated; non-admin callers receive 403 ``PERMISSION_DENIED`` via ``serviceerrors.ErrPermissionDenied``.
   PR 13 added the agent-surface read endpoints (``GET /service_agents/agents`` and ``GET /service_agents/agents/{id}``); these reuse ``AGENT_NOT_FOUND`` for not-found semantics on the by-ID read.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``AGENT_NOT_FOUND``
     - 404
     - Agent ID does not exist or belongs to another customer. Fired by ``GET /agents/{id}``, ``PUT /agents/{id}``, ``DELETE /agents/{id}``, ``PUT /agents/{id}/addresses``, ``PUT /agents/{id}/tag_ids``, ``PUT /agents/{id}/status``, ``PUT /agents/{id}/permission``, ``PUT /agents/{id}/password``, and ``POST /agents/{id}/direct_hash_regenerate``. **Fix:** Verify the ID was obtained from a recent ``GET /agents`` list call.

tag-manager
-----------

.. note::

   PR 9 introduced this section for the tag resource (``/tags``).
   ``TAG_NOT_FOUND`` is emitted today by tag-manager (via ``cerrors.NotFound("tag-manager", ...)``) and surfaces end-to-end on Get-by-ID lookups (the api-manager servicehandler does not intercept soft-deleted entries for this resource type).
   Tag write surfaces (``POST /tags``, ``PUT /tags/{id}``, ``DELETE /tags/{id}``) are admin-gated; non-admin callers receive 403 ``PERMISSION_DENIED`` via ``serviceerrors.ErrPermissionDenied``.
   PR 13 added the agent-surface read endpoints (``GET /service_agents/tags`` and ``GET /service_agents/tags/{id}``); these reuse ``TAG_NOT_FOUND`` for not-found semantics on the by-ID read.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``TAG_NOT_FOUND``
     - 404
     - Tag ID does not exist or belongs to another customer. Fired by ``GET /tags/{id}``, ``PUT /tags/{id}``, and ``DELETE /tags/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /tags`` list call.

customer-manager
----------------

.. note::

   PR 9 introduced this section for the access-key resource (``/accesskeys``).
   ``ACCESSKEY_NOT_FOUND`` and ``CUSTOMER_NOT_FOUND`` are emitted today by customer-manager (via ``cerrors.NotFound("customer-manager", ...)``) and surface end-to-end on Get-by-ID lookups for non-existent resources. For soft-deleted accesskeys, the api-manager servicehandler intercepts via a post-fetch ``TMDelete`` check (``pkg/servicehandler/accesskeys.go``) and re-emits ``serviceerrors.ErrStateInvalid``, which surfaces as 409 ``STATE_INVALID`` in the api-manager domain.
   Access-key write surfaces (``POST /accesskeys``, ``PUT /accesskeys/{id}``, ``DELETE /accesskeys/{id}``) are admin-gated; non-admin callers receive 403 ``PERMISSION_DENIED`` via ``serviceerrors.ErrPermissionDenied``.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``ACCESSKEY_NOT_FOUND``
     - 404
     - Access-key ID does not exist or belongs to another customer. Fired by ``GET /accesskeys/{id}``, ``PUT /accesskeys/{id}``, and ``DELETE /accesskeys/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /accesskeys`` list call.
   * - ``CUSTOMER_NOT_FOUND``
     - 404
     - Customer ID does not exist. Fired by admin endpoints ``GET /customers/{id}``, ``PUT /customers/{id}``, and ``DELETE /customers/{id}``. **Fix:** Verify the customer ID was obtained from a recent ``GET /customers`` list call. (Note: the ``POST /auth/boot`` direct-hash flow currently re-wraps a missing-customer condition with the api-manager generic ``RESOURCE_NOT_FOUND`` rather than passing through ``CUSTOMER_NOT_FOUND``; that is tracked as a follow-up to preserve the typed reason end-to-end.)

campaign-manager
----------------

.. note::

   PR 10 introduced this section for the campaign (``/campaigns``), campaign-call (``/campaigncalls``), and outplan (``/outplans``) resources. The outplan resource is owned by ``bin-campaign-manager`` (alongside campaigns and campaigncalls), so it lives in this section rather than under outdial-manager.
   ``CAMPAIGN_NOT_FOUND`` and ``CAMPAIGNCALL_NOT_FOUND`` are emitted today by campaign-manager (via ``cerrors.NotFound("campaign-manager", ...)``) and surface end-to-end on Get-by-ID lookups for non-existent resources.
   Campaign state-transition operations (``PUT /campaigns/{id}/status``) are **idempotent** in bin-campaign-manager today — for example, stopping an already-stopped campaign returns success (no-op). The state-restriction reason ``CAMPAIGN_STATE_INVALID`` (409) is **not declared** on these endpoints because the underlying handler does not surface a state error. A forward-compatible 409 declaration may be added once the typed-error migration ships and explicit state-transition typing is introduced.
   ``POST /campaigns`` and ``POST /outplans`` are not directly billing-sensitive — per-call charges happen downstream when individual outbound calls are dialed. No 402 declarations apply to campaign or outplan creation.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``CAMPAIGN_NOT_FOUND``
     - 404
     - Campaign ID does not exist or belongs to another customer. Fired by ``GET /campaigns/{id}``, ``PUT /campaigns/{id}``, ``DELETE /campaigns/{id}``, ``PUT /campaigns/{id}/status``, ``PUT /campaigns/{id}/service_level``, ``PUT /campaigns/{id}/actions``, ``PUT /campaigns/{id}/resource_info``, ``PUT /campaigns/{id}/next_campaign_id``, and ``GET /campaigns/{id}/campaigncalls``. **Fix:** Verify the ID was obtained from a recent ``GET /campaigns`` list call.
   * - ``CAMPAIGNCALL_NOT_FOUND``
     - 404
     - Campaign-call ID does not exist or belongs to another customer. Fired by ``GET /campaigncalls/{id}`` and ``DELETE /campaigncalls/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /campaigncalls`` list call or from the response of ``GET /campaigns/{id}/campaigncalls``.
   * - ``OUTPLAN_NOT_FOUND``
     - 404
     - Outplan ID does not exist or belongs to another customer. Fired by ``GET /outplans/{id}``, ``PUT /outplans/{id}``, ``DELETE /outplans/{id}``, and ``PUT /outplans/{id}/dial_info``. **Fix:** Verify the ID was obtained from a recent ``GET /outplans`` list call.

outdial-manager
---------------

.. note::

   PR 10 introduced this section for the outdial (``/outdials``) and outdial-target (``/outdials/{id}/targets``) resources.
   ``OUTDIAL_NOT_FOUND``, ``OUTDIAL_TARGET_NOT_FOUND``, and ``OUTPLAN_NOT_FOUND`` are emitted today by outdial-manager (via ``cerrors.NotFound("outdial-manager", ...)``) and surface end-to-end on Get-by-ID lookups for non-existent resources.
   ``POST /outdials`` and ``POST /outdials/{id}/targets`` are not directly billing-sensitive — per-call charges happen downstream when individual outbound calls are dialed. No 402 declarations apply to outdial or outdial-target creation.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``OUTDIAL_NOT_FOUND``
     - 404
     - Outdial ID does not exist or belongs to another customer. Fired by ``GET /outdials/{id}``, ``PUT /outdials/{id}``, ``DELETE /outdials/{id}``, ``PUT /outdials/{id}/campaign_id``, ``PUT /outdials/{id}/data``, ``GET /outdials/{id}/targets``, and ``POST /outdials/{id}/targets``. **Fix:** Verify the ID was obtained from a recent ``GET /outdials`` list call.
   * - ``OUTDIAL_TARGET_NOT_FOUND``
     - 404
     - Outdial-target ID does not exist or does not belong to the supplied outdial. Fired by ``GET /outdials/{id}/targets/{target_id}`` and ``DELETE /outdials/{id}/targets/{target_id}``. **Fix:** Verify the target ID was obtained from a recent ``GET /outdials/{id}/targets`` list call against the same outdial.

conference-manager
------------------

.. note::

   PR 11 introduced this section for the conference (``/conferences``) and conference-call (``/conferencecalls``) resources.
   ``CONFERENCE_NOT_FOUND`` and ``CONFERENCECALL_NOT_FOUND`` are emitted today by conference-manager (via ``cerrors.NotFound("conference-manager", ...)``) and surface end-to-end on Get-by-ID lookups for non-existent resources.
   Conference state-mutation operations (``POST /conferences/{id}/recording_stop``, ``POST /conferences/{id}/transcribe_stop``, etc.) are **idempotent** in bin-conference-manager today — stopping an already-stopped recording or transcription returns success (no-op). The state-restriction reason ``CONFERENCE_STATE_INVALID`` (409) is **not declared** on these endpoints because the underlying handler does not surface a state error. A forward-compatible 409 declaration may be added once the typed-error migration ships and explicit state-transition typing is introduced.
   ``POST /conferences`` is not directly billing-sensitive — per-call charges happen downstream when individual participants join. No 402 declarations apply to conference creation.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``CONFERENCE_NOT_FOUND``
     - 404
     - Conference ID does not exist or belongs to another customer. Fired by ``GET /conferences/{id}``, ``PUT /conferences/{id}``, ``DELETE /conferences/{id}``, ``POST /conferences/{id}/recording_start``, ``POST /conferences/{id}/recording_stop``, ``POST /conferences/{id}/transcribe_start``, ``POST /conferences/{id}/transcribe_stop``, ``GET /conferences/{id}/media_stream``, and ``POST /conferences/{id}/direct_hash_regenerate``. **Fix:** Verify the ID was obtained from a recent ``GET /conferences`` list call.
   * - ``CONFERENCECALL_NOT_FOUND``
     - 404
     - Conference-call ID does not exist or belongs to another customer. Fired by ``GET /conferencecalls/{id}`` and ``DELETE /conferencecalls/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /conferencecalls`` list call.

queue-manager
-------------

.. note::

   PR 11 introduced this section for the queue (``/queues``) and queue-call (``/queuecalls``) resources.
   ``QUEUE_NOT_FOUND`` and ``QUEUECALL_NOT_FOUND`` are emitted today by queue-manager (via ``cerrors.NotFound("queue-manager", ...)``) and surface end-to-end on Get-by-ID lookups for non-existent resources.
   Queue-call kick operations (``POST /queuecalls/{id}/kick``, ``POST /queuecalls/reference_id/{id}/kick``, ``DELETE /queuecalls/{id}``) are **idempotent** in bin-queue-manager today — kicking an already-ended queue-call returns success (no-op). The state-restriction reason ``QUEUECALL_STATE_INVALID`` (409) is **not declared** on these endpoints because the underlying handler does not surface a state error. A forward-compatible 409 declaration may be added once the typed-error migration ships and explicit state-transition typing is introduced.
   ``POST /queues`` is not directly billing-sensitive — per-call charges happen downstream when individual calls enter the queue. No 402 declarations apply to queue creation.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``QUEUE_NOT_FOUND``
     - 404
     - Queue ID does not exist or belongs to another customer. Fired by ``GET /queues/{id}``, ``PUT /queues/{id}``, ``DELETE /queues/{id}``, ``PUT /queues/{id}/tag_ids``, ``PUT /queues/{id}/routing_method``, and ``POST /queues/{id}/direct_hash_regenerate``. **Fix:** Verify the ID was obtained from a recent ``GET /queues`` list call.
   * - ``QUEUECALL_NOT_FOUND``
     - 404
     - Queue-call ID (or reference ID) does not exist or belongs to another customer. Fired by ``GET /queuecalls/{id}``, ``DELETE /queuecalls/{id}``, ``POST /queuecalls/{id}/kick``, and ``POST /queuecalls/reference_id/{id}/kick``. **Fix:** Verify the ID was obtained from a recent ``GET /queuecalls`` list call.

contact-manager
---------------

.. note::

   PR 12 introduced this section for the contact (``/contacts``) resource and its nested phone-number, email, and tag sub-resources.
   ``CONTACT_NOT_FOUND`` is emitted today by contact-manager (via ``cerrors.NotFound("contact-manager", ...)``) and surfaces end-to-end on Get-by-ID lookups for non-existent contacts. ``CONTACT_EMAIL_NOT_FOUND``, ``CONTACT_PHONE_NUMBER_NOT_FOUND``, and ``CONTACT_TAG_NOT_FOUND`` define the planned typed-error contract for the contact-email / phone-number / tag sub-resources; until contact-manager emits them directly, the api-manager servicehandler maps the underlying not-found conditions to ``serviceerrors.ErrNotFound`` and the translator surfaces the api-manager generic ``RESOURCE_NOT_FOUND``.
   ``POST /contacts``, ``POST /contacts/{id}/phone-numbers``, ``POST /contacts/{id}/emails``, and ``POST /contacts/{id}/tags`` are not billing-sensitive — contact records and their sub-resources are stored without per-operation charges. No 402 declarations apply to contact resource creation.
   PR 13 added the agent-surface counterparts under ``/service_agents/contacts*`` (full read/write parity, including dual-ID nested phone-number, email, and tag sub-paths). These endpoints reuse the same ``CONTACT_NOT_FOUND``, ``CONTACT_PHONE_NUMBER_NOT_FOUND``, ``CONTACT_EMAIL_NOT_FOUND``, and ``CONTACT_TAG_NOT_FOUND`` reasons listed below.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``CONTACT_NOT_FOUND``
     - 404
     - Contact ID does not exist or belongs to another customer. Fired by ``GET /contacts/{id}``, ``PUT /contacts/{id}``, ``DELETE /contacts/{id}``, ``GET /contacts/lookup``, ``POST /contacts/{id}/phone-numbers``, ``PUT /contacts/{id}/phone-numbers/{phone_number_id}``, ``DELETE /contacts/{id}/phone-numbers/{phone_number_id}``, ``POST /contacts/{id}/emails``, ``PUT /contacts/{id}/emails/{email_id}``, ``DELETE /contacts/{id}/emails/{email_id}``, ``POST /contacts/{id}/tags``, and ``DELETE /contacts/{id}/tags/{tag_id}``. **Fix:** Verify the ID was obtained from a recent ``GET /contacts`` list call.
   * - ``CONTACT_PHONE_NUMBER_NOT_FOUND``
     - 404
     - Phone-number ID does not exist on the supplied contact. Fired by ``PUT /contacts/{id}/phone-numbers/{phone_number_id}`` and ``DELETE /contacts/{id}/phone-numbers/{phone_number_id}``. **Fix:** Verify the phone-number ID was obtained from a recent ``GET /contacts/{id}`` response against the same contact.
   * - ``CONTACT_EMAIL_NOT_FOUND``
     - 404
     - Email ID does not exist on the supplied contact. Fired by ``PUT /contacts/{id}/emails/{email_id}`` and ``DELETE /contacts/{id}/emails/{email_id}``. **Fix:** Verify the email ID was obtained from a recent ``GET /contacts/{id}`` response against the same contact.
   * - ``CONTACT_TAG_NOT_FOUND``
     - 404
     - Tag ID is not currently associated with the supplied contact. Fired by ``DELETE /contacts/{id}/tags/{tag_id}``. **Fix:** Verify the tag ID is present in the ``tag_ids`` field of a recent ``GET /contacts/{id}`` response against the same contact.

registrar-manager
-----------------

.. note::

   PR 12 introduced this section for the extension (``/extensions``) resource.
   ``EXTENSION_NOT_FOUND`` is emitted today by registrar-manager (via ``cerrors.NotFound("registrar-manager", ...)``) and surfaces end-to-end on Get-by-ID lookups for non-existent extensions. (registrar-manager also emits ``TRUNK_NOT_FOUND`` — see the provisioning-admin section above.)
   ``POST /extensions`` is not billing-sensitive — extension records are stored without per-operation charges. No 402 declarations apply to extension creation.
   PR 13 added the agent-surface read endpoints (``GET /service_agents/extensions`` and ``GET /service_agents/extensions/{id}``); these reuse ``EXTENSION_NOT_FOUND`` for not-found semantics on the by-ID read.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``EXTENSION_NOT_FOUND``
     - 404
     - Extension ID does not exist or belongs to another customer. Fired by ``GET /extensions/{id}``, ``PUT /extensions/{id}``, ``DELETE /extensions/{id}``, and ``POST /extensions/{id}/direct_hash_regenerate``. **Fix:** Verify the ID was obtained from a recent ``GET /extensions`` list call.

talk-manager
------------

.. note::

   PR 14 introduced this section for the agent-scoped talk surface (chats, channels, messages, participants, reactions) under ``/service_agents/talk_chats*``, ``/service_agents/talk_channels``, and ``/service_agents/talk_messages*``.
   ``CHAT_NOT_FOUND`` is emitted today by talk-manager (via ``cerrors.NotFound("talk-manager", ...)``) and surfaces end-to-end on Get-by-ID lookups for non-existent chats. ``TALK_MESSAGE_NOT_FOUND`` and ``PARTICIPANT_NOT_FOUND`` define the planned typed-error contract for the message / participant sub-resources; until talk-manager emits them directly, the api-manager servicehandler maps the underlying not-found conditions to ``serviceerrors.ErrNotFound`` and the translator surfaces the api-manager generic ``RESOURCE_NOT_FOUND``.
   The talk surface is internal agent collaboration only — it does not deduct balance and has no idempotent state-transition contracts. No 402 or 409 declarations apply to talk endpoints.
   ``TALK_MESSAGE_NOT_FOUND`` is intentionally distinct from PR 6's ``MESSAGE_NOT_FOUND`` reason in the message-manager (which covers SMS messages under ``/messages``); the two share no code path despite the similar shape.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``CHAT_NOT_FOUND``
     - 404
     - Chat ID does not exist or belongs to another customer. Fired by ``GET /service_agents/talk_chats/{id}``, ``PUT /service_agents/talk_chats/{id}``, ``DELETE /service_agents/talk_chats/{id}``, ``POST /service_agents/talk_chats/{id}/join``, ``GET /service_agents/talk_chats/{id}/participants``, ``POST /service_agents/talk_chats/{id}/participants``, ``DELETE /service_agents/talk_chats/{id}/participants/{participant_id}``, and ``GET /service_agents/talk_messages?chat_id={chat_id}``. **Fix:** Verify the ID was obtained from a recent ``GET /service_agents/talk_chats`` or ``GET /service_agents/talk_channels`` list call.
   * - ``TALK_MESSAGE_NOT_FOUND``
     - 404
     - Talk message ID does not exist or belongs to another customer. Fired by ``GET /service_agents/talk_messages/{id}``, ``DELETE /service_agents/talk_messages/{id}``, and ``POST /service_agents/talk_messages/{id}/reactions``. **Fix:** Verify the ID was obtained from a recent ``GET /service_agents/talk_messages?chat_id={chat_id}`` list call. Distinct from PR 6's ``MESSAGE_NOT_FOUND`` (SMS messages).
   * - ``PARTICIPANT_NOT_FOUND``
     - 404
     - Participant ID is not currently associated with the supplied chat. Fired by ``DELETE /service_agents/talk_chats/{id}/participants/{participant_id}``. **Fix:** Verify the participant ID was obtained from a recent ``GET /service_agents/talk_chats/{id}/participants`` response against the same chat.

timeline-manager
----------------

.. note::

   PR 15 finalized the canonical error envelope on the timeline gateway endpoints: ``GET /aggregated-events``, ``GET /timelines/{resource_type}/{resource_id}/events``, ``GET /timelines/calls/{call_id}/sip-analysis``, and ``GET /timelines/calls/{call_id}/pcap``.
   These endpoints do **not** yet emit typed timeline-specific reasons of their own. All 4xx/5xx responses route through the translator's typed-sentinel path and surface as the api-manager generic reasons (``RESOURCE_NOT_FOUND`` via ``serviceerrors.ErrNotFound``, ``PERMISSION_DENIED`` via ``serviceerrors.ErrPermissionDenied``, ``INVALID_ARGUMENT`` via ``serviceerrors.ErrInvalidArgument``, ``INTERNAL`` for unmatched errors). The aggregated-events endpoint additionally returns ``INVALID_ARGUMENT`` with the reasons ``INVALID_ACTIVEFLOW_ID`` and ``INVALID_CALL_ID`` for syntactically invalid query-param UUIDs, and the resource-events endpoint returns ``INVALID_ARGUMENT`` with reason ``INVALID_RESOURCE_TYPE`` when the path enum is rejected.
   Domain-specific reasons such as ``TIMELINE_NOT_FOUND``, ``SIP_ANALYSIS_NOT_AVAILABLE``, or ``PCAP_NOT_AVAILABLE`` will be appended here once the timeline-manager service-side typed-error migration ships.

Other Domains
-------------

The PR 4-15 rollout landed the canonical error envelope across every ``bin-api-manager/server/*.go`` file. The reason-code catalog above covers the published reasons emitted today via the api-manager servicehandler typed sentinels. As of PR 15 the broader-sweep grep (matching ``c.AbortWithStatus(JSON)?(...)`` and ``c.JSON(http.Status...)`` in non-test ``server/*.go`` files) returns zero matches — the migration is structurally complete. Remaining manager services (e.g., ``timeline-manager`` listed above) do not yet emit typed reasons of their own; those will be appended here as the corresponding service-side typed-error migrations ship. See the design doc for the next planned rollout.
