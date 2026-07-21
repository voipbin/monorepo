Creating a Flow
================

.. index:: single: Admin console; Flow
.. index:: single: Admin console; Flow editor
.. index:: single: Admin console; Action Graph

Flows are VoIPBin's programmable call-handling workflows. In the admin
console they live under **Platform -> Flows**.

.. image:: _static/images/admin_console_flow_list.png
   :alt: Flows list page
   :width: 700px
   :align: center

The Flows list shows every flow saved in your account. Click **Create
Flow** to start a new one.

.. note:: **AI Implementation Hint**

   A Flow in the console maps directly to the ``flows`` resource in the
   REST API (see :ref:`Flow <flow-main>`). Flows you build visually here
   can also be created or updated programmatically through
   ``POST /v1.0/flows``.

Starting from a template
-------------------------

.. image:: _static/images/admin_console_flow_choose_template.png
   :alt: Choose a Flow Template dialog
   :width: 700px
   :align: center

Creating a flow opens a template picker so you do not have to start from
a blank canvas. Common starting points include an IVR menu, a simple
answer-and-play greeting, or a queue-routing flow. Pick the closest match
and customize it, or choose **Blank Flow** to start empty.

The Flow editor (Action Graph)
-------------------------------

.. image:: _static/images/admin_console_flow_editor.png
   :alt: Flow editor canvas with node palette
   :width: 700px
   :align: center

The editor is a drag-and-drop canvas (built on top of the visual
Action Graph). The left panel is the node palette, grouped by category:

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Category
     - Example nodes
   * - **Call control**
     - Answer, Hangup, Call, Connect, Branch, Goto, Stop
   * - **Media**
     - Play, Beep, Echo, Recording Start/Stop, External Media Start/Stop,
       Stream Echo
   * - **AI**
     - AI Talk, AI Summary, AI Task
   * - **Messaging**
     - Message Send, Email Send, Conversation Send
   * - **Digits & Variables**
     - Digits Receive, Digits Send, Variable Set
   * - **Routing**
     - Queue Join, Conference Join, Confbridge Join, Case Create
   * - **Transcription**
     - Transcribe Start/Stop, Transcribe Recording
   * - **Integration**
     - Fetch, Fetch Flow, Webhook Send
   * - **Utility**
     - AMD (answering machine detection), Sleep

Drag a node onto the canvas, connect it to the previous node with an
edge, and configure its fields in the panel that opens when you click the
node. Every flow starts from a single **Start** node; execution follows
the edges you draw until it reaches a **Hangup** or **Stop** node.

Save your changes with the save action in the editor toolbar. The editor
also persists your last canvas layout (node positions, zoom level) per
flow, so returning to a flow later reopens it the way you left it.

Activeflows: watching a flow run
----------------------------------

.. index:: single: Admin console; Activeflow

Every time a flow is triggered (an inbound call, an API call, and so on),
VoIPBin creates an **Activeflow**, a live execution instance of that
flow. **Platform -> Flows -> Activeflows** lists these executions so you
can see which flow ran, when, and its current node. This is useful for
debugging why a call did not behave as expected: open the Activeflow tied
to that call and step through the node history.
