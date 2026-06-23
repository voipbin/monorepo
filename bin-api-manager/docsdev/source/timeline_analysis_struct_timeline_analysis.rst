.. _timeline-analysis-struct-timeline-analysis:

Timeline analysis
=================

.. _timeline-analysis-struct-timeline-analysis-timeline-analysis:

Timeline analysis
-----------------

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "activeflow_id": "<string>",
        "status": "<string>",
        "result": { ... },
        "error": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>"
    }

* ``id`` (UUID): The analysis's unique identifier. Returned when creating via ``POST /timeline-analyses`` or listing via ``GET /timeline-analyses``.
* ``customer_id`` (UUID): The customer who owns this analysis. Obtained from the ``id`` field of ``GET /customers``.
* ``activeflow_id`` (UUID): The ended activeflow this analysis was produced for. Obtained from the ``id`` field of ``GET /activeflows``.
* ``status`` (enum string): The analysis lifecycle state. See :ref:`Status <timeline-analysis-struct-timeline-analysis-status>`.
* ``result`` (object): The structured verdict. Present only when ``status`` is ``completed``. See :ref:`Result <timeline-analysis-struct-timeline-analysis-result>`.
* ``error`` (string): A sanitized, operator-safe failure reason. Present only when ``status`` is ``failed``.
* ``tm_create`` (string, ISO 8601): Timestamp when this analysis was created.
* ``tm_update`` (string, ISO 8601): Timestamp when this analysis was last updated.

.. _timeline-analysis-struct-timeline-analysis-status:

Status
------

All possible values for the ``status`` field:

=========== ===========
Status      Description
=========== ===========
progressing The analysis chain is running. No ``result`` yet.
completed   The structured verdict has been produced and stored in ``result``.
failed      The analysis chain failed. ``error`` carries a sanitized reason.
=========== ===========

.. _timeline-analysis-struct-timeline-analysis-result:

Result
------

The structured verdict produced for a ``completed`` analysis.

.. code::

    {
        "version": 1,
        "overall_status": "<string>",
        "input_reduced": false,
        "resources_used": [
            {"type": "<string>", "count": 0}
        ],
        "narrative": "<string>",
        "issues": [
            {
                "severity": "<string>",
                "area": "<string>",
                "summary": "<string>",
                "evidence": [
                    {
                        "evidence_index": 0,
                        "event_type": "<string>",
                        "timestamp": "<string>",
                        "resource_id": "<string>"
                    }
                ]
            }
        ]
    }

* ``version`` (integer): The verdict schema version.
* ``overall_status`` (enum string): The holistic verdict. One of ``ok``, ``warning``, ``error``.
* ``input_reduced`` (boolean): True when the analyzed input was reduced (very large flow) before the verdict was produced.
* ``resources_used`` (array): The resources involved in the flow, with per-type counts.
* ``narrative`` (string): A human-readable summary of what happened in the flow.
* ``issues`` (array): Detected problems. Empty when ``overall_status`` is ``ok``.

  * ``severity`` (enum string): One of ``info``, ``warning``, ``error``.
  * ``area`` (string): The functional area the issue relates to (e.g. ``media``, ``flow``).
  * ``summary`` (string): A short description of the issue.
  * ``evidence`` (array): The events that support this issue. Each entry points to a specific event in the flow's timeline.

Example
-------

.. code::

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "5e4a0680-804e-11ec-8477-2fea5968d85b",
        "activeflow_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "status": "completed",
        "result": {
            "version": 1,
            "overall_status": "warning",
            "input_reduced": false,
            "resources_used": [
                {"type": "call", "count": 2},
                {"type": "transcribe", "count": 1}
            ],
            "narrative": "Two inbound calls were answered and transcribed; one call's media quality degraded mid-conversation.",
            "issues": [
                {
                    "severity": "warning",
                    "area": "media",
                    "summary": "Call media quality (MOS) degraded to 2.8.",
                    "evidence": [
                        {
                            "evidence_index": 42,
                            "event_type": "call_hangup",
                            "timestamp": "2024-03-01T10:05:12.000000Z",
                            "resource_id": "c3d4e5f6-a7b8-9012-cdef-123456789012"
                        }
                    ]
                }
            ]
        },
        "error": "",
        "tm_create": "2024-03-01T10:06:00.000000Z",
        "tm_update": "2024-03-01T10:06:20.000000Z"
    }
