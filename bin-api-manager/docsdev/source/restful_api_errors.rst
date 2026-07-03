.. _error-reason-catalog:

Error Reason Codes
==================

.. note:: **AI Context**

   Every 4xx/5xx response from the VoIPbin API contains an ``error.reason`` field in ``UPPER_SNAKE`` that identifies the specific cause. This page catalogues all published reasons. Clients should branch on ``reason`` for self-healing behavior. ``status`` maps 1:1 to HTTP (see :doc:`restful_api`).

   The error envelope contains ``status`` / ``reason`` / ``message`` / ``request_id`` (and an optional ``details`` array). Reason codes are append-only once published — adding a new reason does not require a schema version bump; removing or renaming one does and triggers a deprecation window.

   Reasons are organized below into two groups:

   * **Generic / Cross-cutting Reasons** apply across every endpoint regardless of resource (authentication, authorization, validation, transport, billing pre-checks, generic state errors, generic not-found).
   * **Resource-Prefixed Reasons** carry an explicit resource prefix (``CALL_*``, ``FLOW_*``, ``RECORDING_*``, …) and only fire on endpoints that operate on that resource.

Generic / Cross-cutting Reasons
-------------------------------

These reasons are not tied to a specific resource and may be returned by any endpoint. They cover authentication, authorization, validation, transport-layer failures, and generic fallback reasons (``RESOURCE_NOT_FOUND``, ``STATE_INVALID``, ``INSUFFICIENT_BALANCE``, ``SERVICE_UNAVAILABLE``) that surface when the underlying condition does not carry a more specific resource-prefixed reason. Treat this group as the baseline catch-all for client error handling.

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
     - The requested resource does not exist or does not belong to the authenticated customer. Verify the UUID was obtained from a recent ``GET`` list call. Returned when the upstream condition does not carry a more specific resource-prefixed not-found reason.
   * - ``ROUTE_NOT_FOUND``
     - 404
     - The requested HTTP endpoint does not exist (wrong path or typo). Verify the URL against the API reference; check API version prefix (``/v1.0/``) and resource name.
   * - ``INVALID_ARGUMENT``
     - 400
     - The request body or a path/query parameter failed handler-level validation (e.g. a manually-parsed UUID typed as ``string`` in the OpenAPI spec). Inspect ``error.message`` for details. Distinct from ``INVALID_REQUEST_PARAMETER``/``MISSING_REQUEST_PARAMETER`` below, which fire earlier, at the wrapper/binding stage, for parameters typed with a specific OpenAPI format (e.g. ``uuid``).
   * - ``INVALID_JSON_BODY``
     - 400
     - The request body is not valid JSON. Ensure ``Content-Type`` is ``application/json`` and the payload parses as a JSON object or array.
   * - ``INVALID_ID``
     - 400
     - A path or body parameter is not a valid UUID. Verify the ID was obtained from a recent ``GET`` list call and has the form ``a1b2c3d4-e5f6-7890-abcd-ef1234567890``.
   * - ``INVALID_REQUEST_PARAMETER``
     - 400
     - A path or query parameter typed with a specific OpenAPI format (e.g. ``uuid``, integer) was present but its value failed to parse, caught at the wrapper/binding stage before any handler code ran. Inspect ``error.message`` for the offending parameter name.
   * - ``MISSING_REQUEST_PARAMETER``
     - 400
     - A required path or query parameter was absent entirely, caught at the same wrapper/binding stage as ``INVALID_REQUEST_PARAMETER``. **Fix:** Add the parameter named in ``error.message``.
   * - ``STATE_INVALID``
     - 409
     - The target resource is in a state that does not allow the requested operation. Check the current state via the resource's ``GET`` endpoint and retry only when the state matches the operation's prerequisite. Returned when the upstream condition does not carry a more specific resource-prefixed state reason.
   * - ``INSUFFICIENT_BALANCE``
     - 402
     - Customer balance is below the minimum required for a chargeable operation (e.g., ``POST /numbers``, ``POST /numbers/renew``, ``POST /messages``, ``POST /emails``). **Fix:** Top up the customer balance via ``POST /billing-accounts/{id}/balance-add`` (admin) or have the customer add credit, then retry.
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

Resource-Prefixed Reasons
-------------------------

These reasons carry an explicit resource prefix in the reason code (e.g., ``CALL_NOT_FOUND``, ``FLOW_STATE_INVALID``) and only surface on endpoints that operate on that resource. Branch on the reason code's prefix to scope your client-side error handling to the right resource.

Call Reasons
^^^^^^^^^^^^

Reasons fired by call endpoints (``/calls``, ``/calls/{id}*``, ``/groupcalls``, ``/service_agents/calls*``).

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
   * - ``GROUPCALL_NOT_FOUND``
     - 404
     - Groupcall ID does not exist or belongs to another customer. Verify via ``GET /groupcalls``.

Recording Reasons
^^^^^^^^^^^^^^^^^

Reasons fired by recording endpoints (``/recordings``, ``/calls/{id}/recording-*``).

.. list-table::
   :header-rows: 1
   :widths: 30 10 60

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``RECORDING_NOT_FOUND``
     - 404
     - Recording ID does not exist or belongs to another customer. Verify via ``GET /recordings``.
   * - ``RECORDING_ALREADY_ACTIVE``
     - 409
     - A recording is already in progress on this call. Stop the existing recording with ``POST /calls/{id}/recording-stop`` before starting a new one.
   * - ``RECORDING_NOT_ACTIVE``
     - 409
     - No recording is active on this call. Start one with ``POST /calls/{id}/recording-start`` before attempting to stop.

Flow / Activeflow Reasons
^^^^^^^^^^^^^^^^^^^^^^^^^

Reasons fired by flow and activeflow endpoints (``/flows``, ``/flows/{id}``, ``/activeflows``, ``/activeflows/{id}*``).

.. note::

   ``POST /activeflows/{id}/stop`` is idempotent in production today — stopping an already-stopped activeflow returns 200 (no-op). The 409 ``ACTIVEFLOW_ALREADY_STOPPED`` response declared in the OpenAPI spec is forward-compatible for clients that prefer opt-in idempotency-aware semantics.

.. list-table::
   :header-rows: 1
   :widths: 30 10 60

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

Billing Reasons
^^^^^^^^^^^^^^^

Reasons fired by billing endpoints (``/billings/{id}``, ``/billing-accounts/{id}*``). The cross-cutting ``INSUFFICIENT_BALANCE`` (402) reason is documented in the Generic / Cross-cutting Reasons table above.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``BILLING_NOT_FOUND``
     - 404
     - Billing record ID does not exist or belongs to another customer. Fired by ``GET /billings/{billing_id}``. **Fix:** Verify the ID was obtained from a recent ``GET /billings`` list call.
   * - ``BILLING_ACCOUNT_NOT_FOUND``
     - 404
     - Billing account ID does not exist or belongs to another customer. Fired by the ``/billing-accounts/{id}*`` admin endpoints. **Fix:** Verify the ID was obtained from a recent ``GET /billing-accounts`` list call.

Number Reasons
^^^^^^^^^^^^^^

Reasons fired by number endpoints (``/numbers``, ``/numbers/{id}``, ``/numbers/renew``).

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

Trunk / Provider Reasons
^^^^^^^^^^^^^^^^^^^^^^^^

Admin-only reasons fired by the provisioning surfaces (``/providers``, ``/providers/setup``, ``/providercalls``, ``/trunks``, ``/routes`` and their sub-paths). Non-admin callers receive ``403 PERMISSION_DENIED``.

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
   * - ``PROVIDERCALL_NOT_FOUND``
     - 404
     - Provider-call ID does not exist. Verify via ``GET /providercalls``. Admin-only resource.

Storage Reasons
^^^^^^^^^^^^^^^

Reasons fired by storage endpoints (``/storage-accounts/{id}``, ``/storage_files``, ``/storage_files/{id}*``, and their ``/service_agents/files*`` agent-surface counterparts).

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

Conversation Reasons
^^^^^^^^^^^^^^^^^^^^

Reasons fired by conversation endpoints (``/conversation-accounts/{id}``, ``/conversations/{id}*``, ``/service-agents/conversations/{id}*``).

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

Message Reasons
^^^^^^^^^^^^^^^

Reasons fired by SMS message endpoints (``/messages``, ``/messages/{id}``). ``POST /messages`` is billing-sensitive — see ``INSUFFICIENT_BALANCE`` in the Generic / Cross-cutting Reasons table above.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``MESSAGE_NOT_FOUND``
     - 404
     - Message ID does not exist or belongs to another customer. Fired by ``GET /messages/{id}`` and ``DELETE /messages/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /messages`` list call.

Email Reasons
^^^^^^^^^^^^^

Reasons fired by email endpoints (``/emails``, ``/emails/{id}``). ``POST /emails`` is billing-sensitive — see ``INSUFFICIENT_BALANCE`` in the Generic / Cross-cutting Reasons table above.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``EMAIL_NOT_FOUND``
     - 404
     - Email ID does not exist or belongs to another customer. Fired by ``GET /emails/{id}`` and ``DELETE /emails/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /emails`` list call.

AI Reasons
^^^^^^^^^^

Reasons fired by AI endpoints (``/aimessages``, ``/ais``, ``/aicalls``, ``/aisummaries``, ``/teams``).

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

RAG Reasons
^^^^^^^^^^^

Reasons fired by RAG endpoints (``/rags``, ``/rags/{id}*``).

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

Speaking Reasons
^^^^^^^^^^^^^^^^

Reasons fired by TTS speaking endpoints (``/speakings``, ``/speakings/{id}*``). ``POST /speakings/{id}/stop`` is idempotent today — stopping an already-stopped speaking session returns success (no-op).

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``SPEAKING_NOT_FOUND``
     - 404
     - Speaking session ID does not exist or belongs to another customer. Fired by ``GET /speakings/{id}``, ``DELETE /speakings/{id}``, ``POST /speakings/{id}/flush``, ``POST /speakings/{id}/say``, and ``POST /speakings/{id}/stop``. **Fix:** Verify the ID was obtained from a recent ``GET /speakings`` list call or from the response of ``POST /speakings``.

Transcribe Reasons
^^^^^^^^^^^^^^^^^^

Reasons fired by STT transcribe endpoints (``/transcribes``, ``/transcribes/{id}*``). ``POST /transcribes/{id}/stop`` is idempotent today — stopping an already-stopped transcription returns success (no-op).

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``TRANSCRIBE_NOT_FOUND``
     - 404
     - Transcribe ID does not exist or belongs to another customer. Fired by ``GET /transcribes/{id}``, ``DELETE /transcribes/{id}``, and ``POST /transcribes/{id}/stop``. **Fix:** Verify the ID was obtained from a recent ``GET /transcribes`` list call or from the response of ``POST /transcribes``.

Agent Reasons
^^^^^^^^^^^^^

Reasons fired by agent endpoints (``/agents``, ``/agents/{id}*``, ``/service_agents/agents*``). Agent write surfaces are admin-gated; non-admin callers receive ``403 PERMISSION_DENIED``.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``AGENT_NOT_FOUND``
     - 404
     - Agent ID does not exist or belongs to another customer. Fired by ``GET /agents/{id}``, ``PUT /agents/{id}``, ``DELETE /agents/{id}``, ``PUT /agents/{id}/addresses``, ``PUT /agents/{id}/tag_ids``, ``PUT /agents/{id}/status``, ``PUT /agents/{id}/permission``, ``PUT /agents/{id}/password``, and ``POST /agents/{id}/direct_hash_regenerate``. **Fix:** Verify the ID was obtained from a recent ``GET /agents`` list call.

Tag Reasons
^^^^^^^^^^^

Reasons fired by tag endpoints (``/tags``, ``/tags/{id}``, ``/service_agents/tags*``). Tag write surfaces are admin-gated; non-admin callers receive ``403 PERMISSION_DENIED``.

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``TAG_NOT_FOUND``
     - 404
     - Tag ID does not exist or belongs to another customer. Fired by ``GET /tags/{id}``, ``PUT /tags/{id}``, and ``DELETE /tags/{id}``. **Fix:** Verify the ID was obtained from a recent ``GET /tags`` list call.

Customer / Accesskey Reasons
^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Reasons fired by customer and access-key endpoints (``/customers/{id}``, ``/accesskeys``, ``/accesskeys/{id}``). Access-key write surfaces and admin customer endpoints are admin-gated; non-admin callers receive ``403 PERMISSION_DENIED``.

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
     - Customer ID does not exist. Fired by admin endpoints ``GET /customers/{id}``, ``PUT /customers/{id}``, and ``DELETE /customers/{id}``. **Fix:** Verify the customer ID was obtained from a recent ``GET /customers`` list call.

Campaign Reasons
^^^^^^^^^^^^^^^^

Reasons fired by campaign, campaign-call, and outplan endpoints (``/campaigns``, ``/campaigns/{id}*``, ``/campaigncalls``, ``/outplans``, ``/outplans/{id}*``). Campaign state-transition operations are idempotent today — for example, stopping an already-stopped campaign returns success (no-op).

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

Outdial Reasons
^^^^^^^^^^^^^^^

Reasons fired by outdial endpoints (``/outdials``, ``/outdials/{id}*``, ``/outdials/{id}/targets*``).

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

Conference Reasons
^^^^^^^^^^^^^^^^^^

Reasons fired by conference endpoints (``/conferences``, ``/conferences/{id}*``, ``/conferencecalls``, ``/conferencecalls/{id}``). Conference state-mutation operations are idempotent today — stopping an already-stopped recording or transcription returns success (no-op).

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

Queue Reasons
^^^^^^^^^^^^^

Reasons fired by queue endpoints (``/queues``, ``/queues/{id}*``, ``/queuecalls``, ``/queuecalls/{id}*``). Queue-call kick operations are idempotent today — kicking an already-ended queue-call returns success (no-op).

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

Contact Reasons
^^^^^^^^^^^^^^^

Reasons fired by contact endpoints (``/contacts``, ``/contacts/{id}*`` and the ``/service_agents/contacts*`` agent-surface counterparts) and their nested phone-number, email, and tag sub-resources.

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

Extension Reasons
^^^^^^^^^^^^^^^^^

Reasons fired by extension endpoints (``/extensions``, ``/extensions/{id}*``, ``/service_agents/extensions*``).

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``EXTENSION_NOT_FOUND``
     - 404
     - Extension ID does not exist or belongs to another customer. Fired by ``GET /extensions/{id}``, ``PUT /extensions/{id}``, ``DELETE /extensions/{id}``, and ``POST /extensions/{id}/direct_hash_regenerate``. **Fix:** Verify the ID was obtained from a recent ``GET /extensions`` list call.

Talk Reasons
^^^^^^^^^^^^

Reasons fired by the agent-scoped talk surface (chats, channels, messages, participants, reactions) under ``/service_agents/talk_chats*``, ``/service_agents/talk_channels``, and ``/service_agents/talk_messages*``. The talk surface is internal agent collaboration only — it does not deduct balance.

.. note::

   ``TALK_MESSAGE_NOT_FOUND`` is intentionally distinct from ``MESSAGE_NOT_FOUND`` in the **Message Reasons** section above (which covers SMS messages under ``/messages``); the two share no code path despite the similar shape.

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
     - Talk message ID does not exist or belongs to another customer. Fired by ``GET /service_agents/talk_messages/{id}``, ``DELETE /service_agents/talk_messages/{id}``, and ``POST /service_agents/talk_messages/{id}/reactions``. **Fix:** Verify the ID was obtained from a recent ``GET /service_agents/talk_messages?chat_id={chat_id}`` list call. Distinct from ``MESSAGE_NOT_FOUND`` (SMS messages).
   * - ``PARTICIPANT_NOT_FOUND``
     - 404
     - Participant ID is not currently associated with the supplied chat. Fired by ``DELETE /service_agents/talk_chats/{id}/participants/{participant_id}``. **Fix:** Verify the participant ID was obtained from a recent ``GET /service_agents/talk_chats/{id}/participants`` response against the same chat.

Timeline Reasons
^^^^^^^^^^^^^^^^

Reasons fired by the timeline gateway endpoints: ``GET /aggregated-events``, ``GET /timelines/{resource_type}/{resource_id}/events``, ``GET /timelines/calls/{call_id}/sip-analysis``, and ``GET /timelines/calls/{call_id}/pcap``. These endpoints do not yet emit timeline-specific not-found reasons of their own — generic not-found, permission, and validation conditions surface as the cross-cutting reasons (``RESOURCE_NOT_FOUND``, ``PERMISSION_DENIED``, ``INVALID_ARGUMENT``, ``INTERNAL``) documented in the Generic / Cross-cutting Reasons table above.

The aggregated-events and resource-events endpoints additionally emit ``INVALID_ARGUMENT`` with the following resource-specific sub-reasons for syntactically invalid query-param or path-enum input:

.. list-table::
   :header-rows: 1
   :widths: 35 10 55

   * - Reason
     - HTTP
     - Cause → Fix
   * - ``INVALID_ACTIVEFLOW_ID``
     - 400
     - The ``activeflow_id`` query parameter on ``GET /aggregated-events`` is not a valid UUID. **Fix:** Verify the ID has the form ``a1b2c3d4-e5f6-7890-abcd-ef1234567890``.
   * - ``INVALID_CALL_ID``
     - 400
     - The ``call_id`` query parameter on ``GET /aggregated-events`` is not a valid UUID. **Fix:** Verify the ID has the form ``a1b2c3d4-e5f6-7890-abcd-ef1234567890``.
   * - ``INVALID_RESOURCE_TYPE``
     - 400
     - The ``{resource_type}`` path parameter on ``GET /timelines/{resource_type}/{resource_id}/events`` is not a recognized resource-type enum value. **Fix:** Use one of the documented resource-type enum values.
