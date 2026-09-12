.. _timeline-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free
   * **Async:** No. All Timeline endpoints are synchronous reads.

VoIPBin's Timeline API exposes the raw event history that backend services publish as resources change state. Every event is stored with its original timestamp, its event type, and the full resource payload (in the same shape you would receive on a webhook), so you can reconstruct exactly what happened to a resource, or across an entire flow execution, after the fact.

There are two ways to query events:

- **Resource events** (``GET /v1.0/timelines/{resource_type}/{resource_id}/events``): every event for a single resource (a call, conference, flow, or activeflow), filtered to events published by that resource's owning service.
- **Aggregated events** (``GET /v1.0/aggregated-events``): every event tied to a single activeflow execution, across every resource type the flow touched (calls, conferences, recordings, transcripts, and so on). Query by ``activeflow_id`` directly, or by ``call_id`` (VoIPBin resolves the call to its owning activeflow for you).

In addition, calls carry two SIP-signalling-specific endpoints for deep troubleshooting:

- **SIP analysis** (``GET /v1.0/timelines/calls/{call_id}/sip-analysis``): the SIP messages and RTCP quality stats captured for a call.
- **PCAP download** (``GET /v1.0/timelines/calls/{call_id}/pcap``): the same call's SIP signalling as a downloadable ``.pcap`` file.

Resource Events
----------------

::

    GET /v1.0/timelines/{resource_type}/{resource_id}/events

``resource_type`` must be one of:

.. list-table::
   :header-rows: 1

   * - resource_type
     - Owning service
     - Matched event types
   * - calls
     - call-manager
     - ``call_*``
   * - conferences
     - conference-manager
     - ``conference_*``
   * - flows
     - flow-manager
     - ``flow_*``
   * - activeflows
     - flow-manager
     - ``activeflow_*``

VoIPBin verifies that the ``resource_id`` belongs to your customer account before returning any events. Requesting events for a resource type other than the four listed above, or for a resource that does not exist or is not yours, returns an error.

Results are paginated and returned in ``result``, newest first, with ``next_page_token`` set when more pages are available.

Aggregated Events
------------------

::

    GET /v1.0/aggregated-events?activeflow_id={activeflow_id}
    GET /v1.0/aggregated-events?call_id={call_id}

Exactly one of ``activeflow_id`` or ``call_id`` must be provided; passing both, or neither, returns an error. When you pass ``call_id``, VoIPBin looks up the call, takes its ``activeflow_id``, and returns the same result as if you had queried by that activeflow directly.

Aggregated events cover every resource type an activeflow touched during its execution, not just one. Unlike resource events, the event's ``data`` field can contain any of VoIPBin's webhook-shaped resources (accesskey, agent, AI, call, campaign, conference, contact, conversation, email, extension, file, flow, groupcall, message, number, outdial, provider, queue, recording, route, speaking, summary, tag, team, transcript, transcribe, transfer, trunk, and more), keyed by the ``event_type`` prefix (for example ``call_answered`` carries a call payload, ``flow_created`` carries a flow payload). Event types VoIPBin does not yet know how to convert to a webhook payload are silently omitted from the results rather than leaking internal fields.

SIP Analysis and PCAP (Calls Only)
------------------------------------

::

    GET /v1.0/timelines/calls/{call_id}/sip-analysis
    GET /v1.0/timelines/calls/{call_id}/pcap

Both endpoints pull from VoIPBin's SIP capture layer (Homer) for the exact time window the call was active, and both filter out internal-to-internal signalling (where source and destination IPs are both RFC 1918 private addresses) so you only see what actually left or entered your call path.

``sip-analysis`` returns:

- ``sip_messages``: an ordered array of captured SIP messages (timestamp, method, source/destination IP and port, and the raw message text).
- ``rtcp_stats``: call quality metrics (MOS, jitter, packet loss percentage, round-trip time, and RTP/RTCP byte and packet counts) parsed from the ``X-RTP-Stat`` header of the call's ``BYE`` message. ``null`` if no RTCP stats were captured for the call.

``pcap`` streams the same signalling as a ``.pcap`` file (``Content-Type: application/vnd.tcpdump.pcap``), suitable for opening directly in Wireshark.

Both endpoints require the call to have an associated channel with a captured SIP Call-ID; calls that never reached the SIP layer (for example, calls that failed before dialing) return a not-found error.

Access Control
--------------

All Timeline endpoints require an authenticated agent with the ``customer_admin`` or ``customer_manager`` permission on the resource's owning customer. Direct/API-key access (accesskey-based requests) is not supported for any Timeline endpoint.
