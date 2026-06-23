.. _timeline_analysis-tutorial:

Tutorial
========

Prerequisites
+++++++++++++

Before working with timeline analysis, you need:

* An authentication token. Obtain one via ``POST /auth/login`` or use an access key from ``GET /accesskeys``.
* An **ended** activeflow. Obtain the activeflow ID via ``GET /activeflows`` and confirm its ``status`` is ``ended``.

.. note:: **AI Implementation Hint**

   Analysis can only be triggered on an activeflow whose ``status`` is ``ended``. Triggering on a ``running`` activeflow returns HTTP ``409``. Triggering for an activeflow your customer account does not own returns HTTP ``404`` (the API does not reveal whether a foreign activeflow exists).

Trigger an Analysis
-------------------

Start an analysis for an ended activeflow. The request returns immediately with status ``progressing``.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/timeline-analyses?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "activeflow_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901"
        }'

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "12345678-1234-1234-1234-123456789012",
        "activeflow_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "status": "progressing",
        "tm_create": "2026-01-20 12:00:00.000000",
        "tm_update": "9999-01-01 00:00:00.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

Triggering again for the same activeflow returns the existing analysis (it does not start a second run) unless you request a re-analysis.

Poll for the Result
-------------------

Poll the analysis by ID until ``status`` becomes ``completed`` or ``failed``.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/timeline-analyses/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'

    {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "customer_id": "12345678-1234-1234-1234-123456789012",
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
                            "timestamp": "2026-01-20T12:05:12.000000Z",
                            "resource_id": "c3d4e5f6-a7b8-9012-cdef-123456789012"
                        }
                    ]
                }
            ]
        },
        "tm_create": "2026-01-20 12:00:00.000000",
        "tm_update": "2026-01-20 12:00:20.000000",
        "tm_delete": "9999-01-01 00:00:00.000000"
    }

List Analyses
-------------

List the analyses for your customer account, optionally filtered by ``activeflow_id`` or ``status``.

.. code::

    $ curl --location --request GET 'https://api.voipbin.net/v1.0/timeline-analyses?token=<YOUR_AUTH_TOKEN>&page_size=10'

Re-analyze
----------

Re-run the analysis for an activeflow. The previous verdict is archived internally and replaced in place.

.. code::

    $ curl --location --request POST 'https://api.voipbin.net/v1.0/timeline-analyses?token=<YOUR_AUTH_TOKEN>' \
        --header 'Content-Type: application/json' \
        --data-raw '{
            "activeflow_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
            "reanalyze": true
        }'

.. note::

   Re-analysis is rate limited. Re-analyzing the same activeflow within the cooldown window, or exceeding the per-customer limit of in-flight analyses, returns HTTP ``429``.

Delete an Analysis
------------------

Delete an analysis you no longer need.

.. code::

    $ curl --location --request DELETE 'https://api.voipbin.net/v1.0/timeline-analyses/a1b2c3d4-e5f6-7890-abcd-ef1234567890?token=<YOUR_AUTH_TOKEN>'
