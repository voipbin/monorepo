.. _timeline_analysis-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Chargeable (each analysis runs one or more AI model calls)
   * **Async:** Yes. ``POST https://api.voipbin.net/v1.0/timeline-analyses`` returns immediately with status ``progressing``. Poll ``GET https://api.voipbin.net/v1.0/timeline-analyses/{id}`` until ``status`` becomes ``completed`` or ``failed``.

VoIPBIN's Timeline Analysis API turns a finished communication flow into a structured, human-readable diagnosis. After an activeflow ends, you can ask VoIPBIN to analyze everything that happened (calls, conferences, transcripts, errors) and return a verdict describing the outcome and any problems, with each problem backed by the exact timeline events that support it.

With the Timeline Analysis API you can:

- Get an at-a-glance ``ok`` / ``warning`` / ``error`` verdict for an ended flow
- Read a narrative summary of what happened across all resources in the flow
- See detected issues with severity, area, and the supporting timeline events
- Re-run the analysis on demand after a flow is reprocessed or updated

How Analysis Works
------------------

When you trigger an analysis, VoIPBIN gathers the ended activeflow's timeline of events and any available transcripts, runs an AI analysis over them, validates the result, and stores a single structured verdict per activeflow.

::

    +-------------+      +------------------+      +-----------+
    | Ended       |--->  | Timeline events  |--->  |    AI     |
    | activeflow  |      | + transcripts    |      | analysis  |
    +-------------+      +------------------+      +-----+-----+
                                                        |
                                                        v
                                                 +-------------+
                                                 |  Structured |
                                                 |   verdict   |
                                                 +-------------+

**Key Points**

- **One live analysis per activeflow.** Triggering again returns the existing analysis unless you request a re-analysis.
- **Eligibility.** Only an ``ended`` activeflow can be analyzed. Triggering on a still-running flow is rejected.
- **Grounded results.** Every non-``ok`` issue cites the specific timeline events it is based on, so the verdict is traceable rather than speculative.

Analysis Lifecycle
------------------

::

    POST /v1.0/timeline-analyses
           |
           v
    +--------------+      success      +-----------+
    | progressing  |------------------>| completed |
    +--------------+                   +-----------+
           |
           | failure
           v
       +--------+
       | failed |
       +--------+

**State Descriptions**

.. list-table::
   :header-rows: 1

   * - State
     - What's happening
   * - progressing
     - The analysis chain is running. No ``result`` yet. Poll until it changes.
   * - completed
     - The structured verdict is available in ``result``.
   * - failed
     - The analysis could not produce a result. ``error`` carries a sanitized reason.

Re-analysis and Rate Limits
---------------------------

You can re-run an analysis with ``reanalyze: true``. To control cost, VoIPBIN applies:

- A short per-activeflow cooldown between re-analyses (a too-frequent re-analyze returns HTTP ``429``).
- A per-customer cap on the number of analyses running at once (triggering past the cap returns HTTP ``429``).

When you re-analyze, the previous verdict is discarded and the analysis is replaced in place by the new run.
