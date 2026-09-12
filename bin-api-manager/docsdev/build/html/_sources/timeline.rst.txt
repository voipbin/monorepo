.. _timeline-main:

********
Timeline
********
Raw event history for VoIPBin resources. Every state change VoIPBin's backend services publish (call started, flow executed, conference joined, and so on) is recorded to a searchable event store. The Timeline API lets you fetch that history for a single resource, or fetch every event tied to one activeflow execution across every resource it touched.

This is different from :ref:`timeline_analysis-main`, which produces an AI-generated diagnostic verdict. The Timeline API returns the raw, ordered events themselves with no AI interpretation.

**API Reference:** `Timeline endpoints <https://api.voipbin.net/redoc/#tag/Timeline>`_

.. toctree::
   :maxdepth: 2

   timeline_overview
   timeline_struct_event
