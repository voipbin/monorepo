.. _timeline-struct-event:

Timeline event
==============

.. _timeline-struct-event-timeline-event:

Timeline event
--------------

The shape returned by both ``GET /timelines/{resource_type}/{resource_id}/events`` and ``GET /aggregated-events``, in the ``result`` array.

.. code::

    {
        "timestamp": "<string>",
        "event_type": "<string>",
        "data": { ... }
    }

* ``timestamp`` (string, ISO 8601): When the event occurred.
* ``event_type`` (string): The event's type, e.g. ``call_created``, ``call_answered``, ``flow_updated``, ``activeflow_created``. The prefix before the first underscore identifies which resource ``data`` contains.
* ``data`` (object): The resource's webhook payload at the time of the event, in the same shape as the resource's own webhook message (see, for example, :ref:`call-struct-call`, :ref:`flow-struct-flow`, :ref:`activeflow-struct-activeflow`). The exact shape depends on ``event_type``.

.. _timeline-struct-event-sip-analysis:

SIP analysis
------------

The response of ``GET /timelines/calls/{call_id}/sip-analysis``.

.. code::

    {
        "sip_messages": [
            {
                "timestamp": "<string>",
                "method": "<string>",
                "src_ip": "<string>",
                "src_port": 0,
                "dst_ip": "<string>",
                "dst_port": 0,
                "raw": "<string>"
            }
        ],
        "rtcp_stats": {
            "mos": 0.0,
            "jitter": 0,
            "packet_loss_pct": 0.0,
            "rtt": 0,
            "rtp_bytes": 0,
            "rtp_packets": 0,
            "rtp_errors": 0,
            "rtcp_bytes": 0,
            "rtcp_packets": 0,
            "rtcp_errors": 0
        }
    }

* ``sip_messages`` (array): The captured SIP messages for the call, in chronological order, with internal-to-internal messages filtered out.

  * ``timestamp`` (string, ISO 8601): When the message was captured.
  * ``method`` (string): The SIP method (e.g. ``INVITE``, ``BYE``, ``200``).
  * ``src_ip`` (string): Source IP address.
  * ``src_port`` (integer): Source port.
  * ``dst_ip`` (string): Destination IP address.
  * ``dst_port`` (integer): Destination port.
  * ``raw`` (string): The full raw SIP message text.

* ``rtcp_stats`` (object, nullable): Call quality metrics parsed from the ``X-RTP-Stat`` header of the call's ``BYE`` message. ``null`` if no RTCP stats were captured.

  * ``mos`` (number): Mean Opinion Score (1.0-5.0).
  * ``jitter`` (integer): Jitter in milliseconds.
  * ``packet_loss_pct`` (number): Packet loss percentage.
  * ``rtt`` (integer): Round-trip time in microseconds, as reported by RTPEngine (divide by 1000 for milliseconds).
  * ``rtp_bytes`` (integer): Total RTP bytes transferred.
  * ``rtp_packets`` (integer): Total RTP packets transferred.
  * ``rtp_errors`` (integer): Total RTP errors.
  * ``rtcp_bytes`` (integer): Total RTCP bytes transferred.
  * ``rtcp_packets`` (integer): Total RTCP packets transferred.
  * ``rtcp_errors`` (integer): Total RTCP errors.

.. note::

   ``GET /timelines/calls/{call_id}/pcap`` returns the same underlying SIP capture as a binary ``.pcap`` file (``Content-Type: application/vnd.tcpdump.pcap``) rather than a JSON body.

Example
-------

.. code::

    {
        "result": [
            {
                "timestamp": "2026-01-15T09:30:00.000Z",
                "event_type": "call_created",
                "data": {
                    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                    "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                    "status": "dialing"
                }
            },
            {
                "timestamp": "2026-01-15T09:30:05.000Z",
                "event_type": "call_answered",
                "data": {
                    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                    "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
                    "status": "progressing"
                }
            }
        ],
        "next_page_token": ""
    }
