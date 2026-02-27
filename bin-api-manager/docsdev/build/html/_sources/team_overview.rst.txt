.. _team-overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Medium
   * **Cost:** Chargeable (credit deduction per AI session — each member consumes LLM, TTS, and STT credits while active)
   * **Async:** Yes. Team conversations run asynchronously during calls. Monitor via ``GET /calls/{id}`` or WebSocket events.

A **Team** models a multi-agent conversation as a directed graph. Each node in the graph is a **Member** backed by a reusable :ref:`AI configuration <ai-struct-ai>`. Directed edges between members are **Transitions** — LLM tool-functions that, when invoked by the currently active AI, hand the conversation to the next member.

.. note:: **AI Implementation Hint**

   A Team is a configuration resource, not a runtime entity. You create a Team via ``POST /teams``, then reference it in a flow action (e.g., ``ai_talk`` with ``team_id``) to activate it during a call. The Team itself does not start conversations — it defines the graph that the flow engine traverses.

How it works
============

Architecture
------------

::

    +-----------------------------------------------------------------------+
    |                     Team Conversation Architecture                    |
    +-----------------------------------------------------------------------+

    1. TEAM DEFINITION (design-time)

       +---------------------+
       |       Team          |
       | start_member_id: A  |
       +----------+----------+
                  |
       +----------v----------+       +---------------------+
       |   Member A           |       |   Member B           |
       |   "Receptionist"     |------>|   "Billing Agent"    |
       |   ai_id: <uuid-A>   | xfer  |   ai_id: <uuid-B>   |
       +-----------+----------+ _to_  +---------------------+
                   |            billing
                   |
                   |  xfer_to_support
                   v
       +---------------------+
       |   Member C           |
       |   "Support Agent"    |
       |   ai_id: <uuid-C>   |
       +---------------------+

    2. RUNTIME (during a call)

       Caller  ──>  Member A (Receptionist)
                       │
                       │ LLM invokes "xfer_to_billing"
                       v
                    Member B (Billing Agent)
                       │
                       │ LLM invokes "xfer_to_support"  (if defined)
                       v
                    Member C (Support Agent)

Graph Model
-----------
A Team is a directed graph where:

* **Nodes** are Members. Each member wraps an existing :ref:`AI configuration <ai-struct-ai>` (``ai_id``), inheriting its LLM, TTS, STT, prompt, and tools.
* **Edges** are Transitions. Each transition is an LLM tool-function (``function_name``) with a human-readable ``description`` the LLM uses to decide when to invoke it. When triggered, the conversation switches to ``next_member_id``.
* **Entry point** is the ``start_member_id`` on the Team. The conversation begins with this member when the flow action activates the team.

::

    +-----------------------------------------------------------------------+
    |                          Graph Model                                  |
    +-----------------------------------------------------------------------+

    Members (nodes)              Transitions (edges)
    +-----------+                +------------------------------+
    | id        |---<defines>---→| function_name                |
    | name      |                | description                  |
    | ai_id     |                | next_member_id → Member.id   |
    +-----------+                +------------------------------+

Conversation Flow
-----------------
When a flow action references a Team:

1. The system starts the conversation with the member identified by ``start_member_id``.
2. The active member's AI configuration (prompt, LLM, TTS, STT, tools) drives the conversation.
3. Transitions are injected into the LLM as additional tool-functions. The LLM's ``description`` field tells the model when to trigger each transition.
4. When the LLM invokes a transition's ``function_name``, the system seamlessly switches to ``next_member_id``. The new member's AI configuration takes over.
5. This repeats until the call ends, the flow stops, or no further transitions are available.

.. note:: **AI Implementation Hint**

   Transition descriptions act as instructions to the LLM. Write them as clear conditions: ``"Transfer to billing when the caller asks about invoices, payments, or account charges."`` Vague descriptions (``"Transfer to billing"`` without context) may cause the LLM to trigger transitions unexpectedly.

Use Cases
=========

* **Customer service routing** — A receptionist AI qualifies intent and routes to specialized agents (billing, support, sales).
* **Multi-step onboarding** — A greeting agent collects basic info, then hands off to a verification agent, then to an account setup agent.
* **Escalation chains** — A first-line support AI attempts resolution, then escalates to a senior agent if needed.
* **Survey + follow-up** — A survey agent collects responses, then transitions to a follow-up agent for scheduling or issue resolution.

Related Documentation
=====================

- :ref:`AI Configuration <ai-struct-ai>` — AI resource referenced by each member
- :ref:`AI Overview <ai-overview>` — How AI voice conversations work
- :ref:`Tool Functions <ai-struct-tool>` — Tools available to each member's AI
- :ref:`Flow Actions <flow-struct-action>` — How to reference a team in a flow
