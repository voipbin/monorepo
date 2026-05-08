.. _providercall-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Medium
   * **Cost:** Chargeable. Each ``POST /v1/providercalls`` creates one or more real outbound calls. Billing and outbound rate-limiting apply identically to customer-initiated calls; the admin's own customer account is charged.
   * **Async:** Yes. The ``POST`` response is a ``ProviderCall`` audit record that contains the IDs of the created Call and Groupcall records. Per-call state (dialing / ringing / answered / hangup reason) must be observed via ``GET https://api.voipbin.net/v1.0/calls/{id}`` or the normal Call webhooks.
   * **Access:** Admin-only. Requires ``PermissionProjectSuperAdmin``. Customer-tier users (``PermissionCustomerAdmin``, ``PermissionCustomerManager``, ``PermissionCustomerAgent``) cannot call this endpoint.

VoIPBIN's ProviderCall API lets platform super-admins place a real outbound call through a specific SIP provider, forcing the routing decision instead of letting the normal customer / default dialroute merge choose one. Use cases:

- **Provider onboarding.** Verify that a newly-configured provider actually accepts calls, routes to the PSTN, and returns a sensible SIP response — before live customer traffic touches it.
- **Routing debugging.** When a provider is suspected of misbehaving, isolate it from normal dialroute priority so the admin can exercise it directly.
- **Post-config verification.** After changing ``hostname`` / ``tech_prefix`` / ``tech_postfix`` / ``tech_headers`` on an existing provider, confirm the new configuration actually works.

The endpoint **triggers** a call — it does not produce a pass/fail verdict. Admins interpret the resulting Call's status, hangup reason, and SIP trace themselves.

How ProviderCalls Work
----------------------

bin-api-manager is a thin gateway: it authenticates, verifies the ``provider_id`` exists, and forwards the full request to bin-route-manager. The orchestration happens inside route-manager (which owns the ``ProviderCall`` entity):

1. **Optional temp-flow creation** — when the admin supplies inline ``actions`` without a ``flow_id``, route-manager creates a temporary flow via ``FlowV1FlowCreate`` and passes that flow's id to call-manager. If any downstream step fails, the temp flow is cleaned up.

2. **Server-side metadata construction** — two internal keys are attached to the Call:

   - ``route_provider_ids`` — tells call-manager / route-manager to return a synthetic dialroute that points at exactly the specified provider, bypassing the normal customer / default merge.
   - ``skip_source_validation`` — tells call-manager to preserve the admin-supplied source number verbatim, instead of falling back to the OutboundConfig's ``default_outgoing_source_number_id`` (or rejecting the call if no default is set) when the source is not owned by the customer. Necessary because providers commonly reject INVITEs whose ``From`` / ``P-Asserted-Identity`` doesn't match a pre-allowlisted caller ID.

3. **Create the underlying Call(s)** — route-manager issues ``CallV1CallsCreate`` synchronously. Call-manager persists the Call(s), reads ``route_provider_ids`` in ``getDialroutes`` and forwards them to ``DialrouteList``, honors ``skip_source_validation`` in ``getValidatedSourceForOutgoingCall``.

4. **Persist the ProviderCall audit record** — route-manager saves the admin's request info (``customer_id``, ``provider_id``, ``flow_id``, ``source``, ``destinations``, ``anonymous``) alongside the IDs of the Call and Groupcall records that step 3 produced.

The response is the persisted ``ProviderCall.WebhookMessage`` — an atomic record (IDs only, no embedded Call/Groupcall objects, per the VoIPBIN atomic-API rule). Admin retrieves per-call state separately.

**Internal-only metadata (not a customer-facing API field)**

``Call.Metadata`` is server-side-only on internal trust paths. No customer-facing ``POST`` endpoint accepts a ``metadata`` field in the request body. Both ``route_provider_ids`` and ``skip_source_validation`` are built server-side by the admin endpoint (never passed in by clients). The call-manager listen handler rejects unknown metadata keys with HTTP 400, so new keys must be declared in the typed-registry before they can be set.

.. note:: **AI Implementation Hint**

   After ``POST /v1/providercalls`` returns, the ProviderCall record's ``call_ids`` array contains one entry per destination. To observe per-destination outcome, iterate ``call_ids`` and call ``GET https://api.voipbin.net/v1.0/calls/{id}`` (or subscribe to webhooks) for each. Do not poll the ProviderCall record itself for call state — it is a summary, not a live status snapshot.

Provider Validation
-------------------

Before the underlying call is created, the handler validates that the supplied ``provider_id`` exists and is not soft-deleted (via ``RouteV1ProviderGet``). This fail-fast check prevents the scenario where a Call record is created with metadata pointing at a non-existent provider — route-manager would then return an error at dial-time, leaving behind a dead Call record.

Soft-Delete Semantics
---------------------

``DELETE /v1/providercalls/{id}`` is a soft-delete — it sets ``tm_delete`` on the audit record and returns the deleted ``ProviderCall`` object. The underlying Call records referenced by ``call_ids`` are **not** affected. Hard deletion of the historical audit trail is not supported.

.. note:: **AI Implementation Hint**

   Deleting a ProviderCall is purely an audit-trail operation. If you need to terminate a call that's currently in progress, use ``POST /v1/calls/{id}/hangup`` against the ``call_ids`` entry instead — the ProviderCall DELETE does not hang up or cancel anything on the wire.


Related Documentation
---------------------

- :ref:`Provider Overview <provider-overview>` — Provider configuration (hostname, tech prefix/postfix, headers)
- :ref:`Call Overview <call-overview>` — Observing per-call progress and final state
- :ref:`Route Overview <route-overview>` — How normal dialroute selection works (ProviderCall bypasses this)
