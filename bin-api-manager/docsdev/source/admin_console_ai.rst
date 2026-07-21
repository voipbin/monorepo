AI Assistant & Team
=====================

.. index:: single: Admin console; AI Assistant
.. index:: single: Admin console; Team (AI)
.. index:: single: Admin console; RAG

AI configuration lives under **AI Services** in the sidebar. It covers
your Assistants and Teams plus supporting resources: RAGs, AI Calls,
Summaries, Audits, and Proposals.

Assistants
----------

.. image:: _static/images/admin_console_ai_assistants_list.png
   :alt: AI Assistants list page
   :width: 700px
   :align: center

**AI Services -> Engines** lists your AI Assistants. An Assistant bundles
an engine model (for example Gemini 2.5 Flash or GPT-4o), voice/TTS-STT
settings, and the tools it is allowed to call. Click **Create Assistant**
to configure a new one, or click a row to edit an existing Assistant's
model, prompt, and tools.

.. note:: **AI Implementation Hint**

   This maps to the ``ais`` resource in the REST API (see
   :ref:`AI <ai-main>`). A Flow's **AI Talk** node references an
   Assistant by ID to hand a call over to it.

Teams
-----

.. index:: single: Admin console; Team graph

**AI Services -> Teams** is where you compose multiple Assistants into a
multi-agent Team using a visual graph (the Team Graph editor). Each node
in the graph is either an AI member (with its own model/voice/tools) or a
routing rule between members. Use a Team when a single Assistant's tool
set or persona is not enough to cover the whole conversation, for example
a triage Assistant that hands off to a billing specialist Assistant.

.. note:: **AI Implementation Hint**

   This maps to the ``teams`` resource in the REST API (see
   :ref:`Team <team-main>`).

RAGs
----

**AI Services -> RAGs** manages Retrieval-Augmented Generation knowledge
bases. Create a RAG, upload or link source documents, and attach the RAG
to an Assistant so its responses are grounded in your own content instead
of the model's general knowledge.

.. note:: **AI Implementation Hint**

   This maps to the ``rags`` resource in the REST API (see
   :ref:`RAG <rag-main>`).

AI Calls and Summaries
-----------------------

**AI Services -> Calls** lists past calls that were handled (fully or
partially) by an AI Assistant, with the transcript and any tool calls the
Assistant made during the call. **AI Services -> Summaries** lists
AI-generated summaries produced after a call or conversation ends, useful
for a quick recap without replaying the whole transcript.

Audits and Proposals
----------------------

**AI Services -> Audits** lists a history of changes made to your AI
Assistants and Teams, useful when reviewing who changed a prompt or model
setting and when. **AI Services -> Proposals** lists AI-suggested prompt
changes awaiting your review; accept a proposal to apply it to the
Assistant, or dismiss it if it does not fit.
