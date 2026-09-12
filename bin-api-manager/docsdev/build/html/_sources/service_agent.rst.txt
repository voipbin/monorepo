.. _service_agent-main:

***************
Agent Console
***************
The dedicated API surface for human-agent-facing consoles (``talk.voipbin.net``, square-talk), served entirely under the ``/service_agents/*`` path prefix. It scopes every request to the caller's own agent identity and customer, and adds console-specific workflows on top of the resources documented elsewhere: case management, agent self-service, and public channel discovery.

**API Reference:** `Service Agent endpoints <https://api.voipbin.net/redoc/#tag/Service-Agent>`_

.. toctree::
   :maxdepth: 2

   service_agent_overview
   service_agent_case_overview
   service_agent_case_struct
   service_agent_tutorial
