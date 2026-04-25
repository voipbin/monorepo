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
   ``BILLING_NOT_FOUND`` and ``BILLING_ACCOUNT_NOT_FOUND`` are reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reasons will be emitted directly once the billing-manager typed-error migration ships.
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

storage-manager
---------------

.. note::

   ``STORAGE_ACCOUNT_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the storage-manager typed-error migration ships.
   The admin-gated ``POST /storage-accounts`` endpoint returns **403 PERMISSION_DENIED** for non-admin callers via the standard ``"no permission"`` translator pattern. The OpenAPI spec declares 403 on this path to reflect runtime behavior.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``STORAGE_ACCOUNT_NOT_FOUND``
     - 404
     - Storage account ID does not exist or belongs to another customer. Fired by ``GET /storage-accounts/{id}`` and ``DELETE /storage-accounts/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /storage-accounts`` list call.

conversation-manager
--------------------

.. note::

   ``CONVERSATION_ACCOUNT_NOT_FOUND`` and ``CONVERSATION_NOT_FOUND`` are reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reasons will be emitted directly once the conversation-manager typed-error migration ships.
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

   ``MESSAGE_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the message-manager typed-error migration ships.
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

   ``EMAIL_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the email-manager typed-error migration ships.
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
   The ``*_NOT_FOUND`` reasons listed below are reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reasons will be emitted directly once the ai-manager typed-error migration ships.
   ``POST /aimessages``, ``POST /aicalls``, and ``POST /aisummaries`` are conceptually billing-sensitive (LLM token cost, voice AI minutes, summary generation cost), but bin-ai-manager has **no balance pre-check today** — the 402 ``INSUFFICIENT_BALANCE`` contract is **not reachable** for AI resource creation. Wiring the pre-check in ai-manager is deferred to a follow-up PR; no 402 declarations were added in PR 7 to match runtime behavior.
   Team write surfaces (``POST /teams``, ``PUT /teams/{id}``, ``DELETE /teams/{id}``, ``POST /teams/{id}/direct_hash_regenerate``) are admin-gated; non-admin callers receive 403 ``PERMISSION_DENIED`` via the translator's ``"no permission"`` substring fallback.

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
   * - ``AISUMMARY_NOT_FOUND``
     - 404
     - AI summary ID does not exist or belongs to another customer. Fired by ``GET /aisummaries/{id}`` and ``DELETE /aisummaries/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /aisummaries`` list call.
   * - ``TEAM_NOT_FOUND``
     - 404
     - Team ID does not exist or belongs to another customer. Fired by ``GET /teams/{id}``, ``PUT /teams/{id}``, ``DELETE /teams/{id}``, and ``POST /teams/{id}/direct_hash_regenerate``. **Fix:** Verify the ID was obtained from a recent ``GET /teams`` list call.

rag-manager
-----------

.. note::

   ``RAG_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the rag-manager typed-error migration ships.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``RAG_NOT_FOUND``
     - 404
     - RAG knowledge-base ID does not exist or belongs to another customer. Fired by ``GET /rags/{id}``, ``PUT /rags/{id}``, ``DELETE /rags/{id}``, ``POST /rags/{id}/sources``, and ``DELETE /rags/{id}/sources/{source_id}``. **Fix:** Verify the ID was obtained from a recent ``GET /rags`` list call.

tts-manager
-----------

.. note::

   ``SPEAKING_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the tts-manager typed-error migration ships.
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

   ``TRANSCRIBE_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the transcribe-manager typed-error migration ships.
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
   ``AGENT_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the agent-manager typed-error migration ships.
   Agent write surfaces (``POST /agents``, ``PUT /agents/{id}``, ``PUT /agents/{id}/permission``, ``PUT /agents/{id}/password``, ``PUT /agents/{id}/addresses``, ``PUT /agents/{id}/tag_ids``, ``PUT /agents/{id}/status``, ``DELETE /agents/{id}``, ``POST /agents/{id}/direct_hash_regenerate``) are admin-gated; non-admin callers receive 403 ``PERMISSION_DENIED`` via the translator's ``"no permission"`` substring fallback.

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
   ``TAG_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the tag-manager typed-error migration ships.
   Tag write surfaces (``POST /tags``, ``PUT /tags/{id}``, ``DELETE /tags/{id}``) are admin-gated; non-admin callers receive 403 ``PERMISSION_DENIED`` via the translator's ``"no permission"`` substring fallback.

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
   ``ACCESSKEY_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the customer-manager typed-error migration ships.
   Access-key write surfaces (``POST /accesskeys``, ``PUT /accesskeys/{id}``, ``DELETE /accesskeys/{id}``) are admin-gated; non-admin callers receive 403 ``PERMISSION_DENIED`` via the translator's ``"no permission"`` substring fallback.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``ACCESSKEY_NOT_FOUND``
     - 404
     - Access-key ID does not exist or belongs to another customer. Fired by ``GET /accesskeys/{id}``, ``PUT /accesskeys/{id}``, and ``DELETE /accesskeys/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /accesskeys`` list call.

campaign-manager
----------------

.. note::

   PR 10 introduced this section for the campaign (``/campaigns``), campaign-call (``/campaigncalls``), and outplan (``/outplans``) resources. The outplan resource is owned by ``bin-campaign-manager`` (alongside campaigns and campaigncalls), so it lives in this section rather than under outdial-manager.
   The ``*_NOT_FOUND`` reasons listed below are reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reasons will be emitted directly once the campaign-manager typed-error migration ships.
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
   The ``*_NOT_FOUND`` reasons listed below are reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reasons will be emitted directly once the outdial-manager typed-error migration ships.
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
   The ``*_NOT_FOUND`` reasons listed below are reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reasons will be emitted directly once the conference-manager typed-error migration ships.
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
   The ``*_NOT_FOUND`` reasons listed below are reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reasons will be emitted directly once the queue-manager typed-error migration ships.
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
   The ``*_NOT_FOUND`` reasons listed below are reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reasons will be emitted directly once the contact-manager typed-error migration ships.
   ``POST /contacts``, ``POST /contacts/{id}/phone-numbers``, ``POST /contacts/{id}/emails``, and ``POST /contacts/{id}/tags`` are not billing-sensitive — contact records and their sub-resources are stored without per-operation charges. No 402 declarations apply to contact resource creation.

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
   ``EXTENSION_NOT_FOUND`` is reachable today via the translator's ``"not found"`` substring fallback (currently surfacing as the api-manager generic ``RESOURCE_NOT_FOUND``); the typed reason will be emitted directly once the registrar-manager typed-error migration ships.
   ``POST /extensions`` is not billing-sensitive — extension records are stored without per-operation charges. No 402 declarations apply to extension creation.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``EXTENSION_NOT_FOUND``
     - 404
     - Extension ID does not exist or belongs to another customer. Fired by ``GET /extensions/{id}``, ``PUT /extensions/{id}``, ``DELETE /extensions/{id}``, and ``POST /extensions/{id}/direct_hash_regenerate``. **Fix:** Verify the ID was obtained from a recent ``GET /extensions`` list call.

Other Domains
-------------

Reason code sections for the remaining manager services — ``talk-manager``, ``timeline-manager`` — will be added as future migration PRs ship. See the design doc for the PR rollout.
