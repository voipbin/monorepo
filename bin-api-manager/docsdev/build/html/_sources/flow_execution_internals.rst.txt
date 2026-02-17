.. _flow-execution-internals:

Flow Execution Internals
========================

This section provides deep technical details about how VoIPBIN's flow engine executes actions internally, including the interaction between services and the state machine that drives flow execution.

.. note:: **AI Implementation Hint**

   This section describes internal architecture and is primarily useful for understanding how the system works, not for building flows. Key takeaways for API consumers: (1) Actions return one of four result types (``next``, ``wait``, ``block``, ``done``) which determines whether the flow continues immediately or pauses. (2) Variables are stored as a JSON column in the activeflow database record and resolved via ``${...}`` pattern substitution at execution time. (3) Concurrent events for the same activeflow are handled via database-level locking -- the first event wins, later events are discarded as stale.

Flow Manager Architecture
-------------------------

The flow-manager service is responsible for executing all flow logic:

.. code::

    Flow Manager Internal Architecture:

    +------------------------------------------------------------------+
    |                        bin-flow-manager                          |
    +------------------------------------------------------------------+
    |                                                                  |
    |  +------------------+    +------------------+    +-------------+ |
    |  | Listen Handler   |    | Flow Handler     |    | Action      | |
    |  | (RabbitMQ RPC)   |--->| (Orchestrator)   |--->| Executors   | |
    |  +------------------+    +------------------+    +-------------+ |
    |           |                      |                     |         |
    |           v                      v                     v         |
    |  +------------------+    +------------------+    +-------------+ |
    |  | Request Router   |    | Activeflow       |    | External    | |
    |  | (message types)  |    | State Machine    |    | Services    | |
    |  +------------------+    +------------------+    +-------------+ |
    |                                  |                               |
    +----------------------------------+-------------------------------+
                                       |
                                       v
    +------------------------------------------------------------------+
    |                    Database Layer (MySQL)                        |
    |  +------------------+    +------------------+    +-------------+ |
    |  | activeflows      |    | flows            |    | variables   | |
    |  | (runtime state)  |    | (definitions)    |    | (runtime)   | |
    +------------------------------------------------------------------+


Activeflow Lifecycle
--------------------

When a flow executes, an "activeflow" instance is created to track its state:

.. code::

    Activeflow State Transitions:

    +----------+     +-----------+     +-----------+     +----------+
    | created  |---->| executing |---->| waiting   |---->| ended    |
    +----------+     +-----------+     +-----------+     +----------+
         |                |                 |                 ^
         |                |                 |                 |
         |                v                 v                 |
         |           +-----------+     +-----------+          |
         |           | error     |     | blocked   |          |
         |           +-----------+     +-----------+          |
         |                |                 |                 |
         |                +-----------------+-----------------+
         |                                  |
         +----------------------------------+
                   (any state can end)


    Activeflow Database Record:
    +------------------------------------------------------------------+
    | id: uuid                    | Unique activeflow identifier       |
    | flow_id: uuid               | Reference to flow definition       |
    | customer_id: uuid           | Owner of this execution            |
    | reference_type: string      | "call", "message", "api"           |
    | reference_id: uuid          | Associated call/message ID         |
    | current_action_id: uuid     | Cursor position in flow            |
    | status: string              | Current execution state            |
    | variables: json             | Runtime variable storage           |
    | forward_action_id: uuid     | For nested flow returns            |
    | stack_depth: int            | Nested execution depth             |
    | execute_count: int          | Safety counter                     |
    +------------------------------------------------------------------+


Execution Loop Detail
---------------------

The core execution loop processes actions one at a time:

.. code::

    Flow Execution Loop:

    +------------------------------------------------------------------+
    |                     Execution Engine                             |
    +------------------------------------------------------------------+

    1. Get Next Action
    +-----------------+
    | Load activeflow |
    | from database   |
    +--------+--------+
             |
             v
    +------------------+
    | Get action by    |
    | current_action_id|
    +--------+---------+
             |
             v
    2. Execute Action
    +------------------+     +-----------------------------------+
    | Action Executor  |---->| Action-specific logic:            |
    | Factory          |     | - talk: Generate TTS, send media  |
    +------------------+     | - branch: Evaluate variable       |
                             | - connect: Create outbound call   |
                             +-----------------------------------+
             |
             v
    3. Process Result
    +------------------+
    | Execution Result |
    +--------+---------+
             |
             +------------------+------------------+
             |                  |                  |
             v                  v                  v
    +-------------+    +---------------+    +-------------+
    | result_next |    | result_wait   |    | result_done |
    | (continue)  |    | (async event) |    | (flow ends) |
    +------+------+    +-------+-------+    +------+------+
           |                   |                   |
           v                   v                   v
    Move cursor to      Keep cursor,         Mark activeflow
    next action         wait for event       as ended


Action Execution Results
------------------------

Each action executor returns one of these result types:

.. code::

    Action Result Types:

    +------------------------------------------------------------------+
    |                         result_next                              |
    +------------------------------------------------------------------+
    | Meaning: Action completed, move to next action immediately       |
    | Used by: goto, branch, variable_set, condition_*, answer         |
    |                                                                  |
    | Flow continues without waiting:                                  |
    | +-------+     +-------+     +-------+                           |
    | | goto  |---->| talk  |---->| next  |                           |
    | +-------+     +-------+     +-------+                           |
    |     ^                                                            |
    |     |                                                            |
    |  result_next                                                     |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                         result_wait                              |
    +------------------------------------------------------------------+
    | Meaning: Action started async operation, wait for completion     |
    | Used by: talk, play, digits_receive, connect, queue_join         |
    |                                                                  |
    | Execution pauses, waiting for event:                             |
    | +-------+     +-----------------+     +-------+                  |
    | | talk  |---->| (waiting...)    |---->| next  |                  |
    | +-------+     | TTS playing     |     +-------+                  |
    |               +-----------------+                                |
    |                       ^                                          |
    |                       |                                          |
    |               Media complete event                               |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                         result_block                             |
    +------------------------------------------------------------------+
    | Meaning: Flow is blocked until external API call resumes it      |
    | Used by: block                                                   |
    |                                                                  |
    | Execution stops until API trigger:                               |
    | +-------+     +-------------------+     +-------+                |
    | | block |---->| BLOCKED           |---->| next  |                |
    | +-------+     | (waiting for API) |     +-------+                |
    |               +-------------------+                              |
    |                       ^                                          |
    |                       |                                          |
    |           POST /activeflows/{id}/execute                         |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    |                         result_done                              |
    +------------------------------------------------------------------+
    | Meaning: Flow execution is complete                              |
    | Used by: stop, hangup (when call ends), end of actions           |
    |                                                                  |
    | Flow terminates:                                                 |
    | +-------+     +-------+                                          |
    | | stop  |---->| END   |                                          |
    | +-------+     +-------+                                          |
    +------------------------------------------------------------------+


Inter-Service Communication During Execution
--------------------------------------------

Flow execution involves multiple services communicating via RabbitMQ:

.. code::

    Talk Action Execution Flow:

    flow-manager          call-manager         asterisk-proxy        Asterisk
         |                     |                     |                   |
         | Execute talk        |                     |                   |
         | action              |                     |                   |
         |                     |                     |                   |
         |--(RPC)------------->|                     |                   |
         | TalkStart request   |                     |                   |
         |                     |                     |                   |
         |                     |--(RPC)------------->|                   |
         |                     | Media command       |                   |
         |                     |                     |                   |
         |                     |                     |--(ARI)----------->|
         |                     |                     | Play media        |
         |                     |                     |                   |
         |                     |                     |<-(Event)----------|
         |                     |                     | PlaybackFinished  |
         |                     |                     |                   |
         |                     |<-(Event)------------|                   |
         |                     | MediaComplete       |                   |
         |                     |                     |                   |
         |<-(Event)------------|                     |                   |
         | ActionComplete      |                     |                   |
         |                     |                     |                   |
         | Continue to         |                     |                   |
         | next action         |                     |                   |


.. code::

    Connect Action Execution Flow:

    flow-manager          call-manager         asterisk-proxy        Asterisk
         |                     |                     |                   |
         | Execute connect     |                     |                   |
         |                     |                     |                   |
         |--(RPC)------------->|                     |                   |
         | Connect request     |                     |                   |
         | (destinations)      |                     |                   |
         |                     |                     |                   |
         |                     | Create outbound     |                   |
         |                     | call record         |                   |
         |                     |                     |                   |
         |                     |--(RPC)------------->|                   |
         |                     | Originate request   |                   |
         |                     |                     |                   |
         |                     |                     |--(ARI)----------->|
         |                     |                     | channels/create   |
         |                     |                     |                   |
         |                     |                     |<-(Event)----------|
         |                     |                     | ChannelCreated    |
         |                     |                     |                   |
         |                     |                     |<-(Event)----------|
         |                     |                     | ChannelAnswered   |
         |                     |                     |                   |
         |                     |<-(Event)------------|                   |
         |                     | CallAnswered        |                   |
         |                     |                     |                   |
         |                     | Bridge calls        |                   |
         |                     | together            |                   |
         |                     |                     |                   |
         |<-(Event)------------|                     |                   |
         | ConnectComplete     |                     |                   |
         |                     |                     |                   |
         | Flow waits for      |                     |                   |
         | call to end         |                     |                   |


Variable System Internals
-------------------------

Variables are stored and resolved during flow execution:

.. code::

    Variable Storage Architecture:

    +------------------------------------------------------------------+
    |                     Activeflow Variables                         |
    +------------------------------------------------------------------+
    |                                                                  |
    | Storage: JSON column in activeflows table                        |
    |                                                                  |
    | {                                                                |
    |   "voipbin.activeflow.id": "abc-123",                           |
    |   "voipbin.activeflow.reference_id": "call-456",                |
    |   "voipbin.call.id": "call-456",                                |
    |   "voipbin.call.source.target": "+15551234567",                 |
    |   "voipbin.call.destination.target": "+15559876543",            |
    |   "voipbin.call.digits": "123",                                 |
    |   "voipbin.call.status": "progressing",                         |
    |   "voipbin.recording.id": "rec-789",                            |
    |   "customer.tier": "premium",                 <- custom          |
    |   "order.id": "ORD-12345"                     <- custom          |
    | }                                                                |
    |                                                                  |
    +------------------------------------------------------------------+


.. code::

    Variable Resolution Process:

    Input: "Hello ${customer.name}, your order ${order.id} is ready"

    +------------------------------------------------------------------+
    |                     Variable Resolver                            |
    +------------------------------------------------------------------+

    Step 1: Find all ${...} patterns
    +------------------+
    | Pattern scanner  |---> ["${customer.name}", "${order.id}"]
    +------------------+

    Step 2: Look up each variable
    +------------------+     +------------------+
    | customer.name    |---->| activeflow.vars  |---> "John Smith"
    +------------------+     +------------------+

    +------------------+     +------------------+
    | order.id         |---->| activeflow.vars  |---> "ORD-12345"
    +------------------+     +------------------+

    Step 3: Substitute values
    +------------------------------------------------------------------+
    | Output: "Hello John Smith, your order ORD-12345 is ready"        |
    +------------------------------------------------------------------+


    Variable Set Timing:

    +------------------------------------------------------------------+
    | When Variables Are Set                                           |
    +------------------------------------------------------------------+
    |                                                                  |
    | At activeflow creation:                                          |
    |   - voipbin.activeflow.id                                        |
    |   - voipbin.activeflow.reference_id                              |
    |   - voipbin.activeflow.reference_type                            |
    |                                                                  |
    | When call starts:                                                |
    |   - voipbin.call.id                                              |
    |   - voipbin.call.source.target                                   |
    |   - voipbin.call.destination.target                              |
    |   - voipbin.call.direction                                       |
    |                                                                  |
    | After digits_receive action:                                     |
    |   - voipbin.call.digits                                          |
    |                                                                  |
    | After recording_start action:                                    |
    |   - voipbin.recording.id                                         |
    |                                                                  |
    | After transcribe completes:                                      |
    |   - voipbin.transcribe.text                                      |
    |                                                                  |
    | By variable_set action:                                          |
    |   - Any custom variable                                          |
    +------------------------------------------------------------------+


Flow Forking and Stack Management
---------------------------------

Nested flows use a stack-based execution model:

.. code::

    Flow Fork Mechanism:

    Main Flow (stack depth 0)
    +------------------------------------------------------------------+
    | action 1: answer                                                 |
    | action 2: talk "Welcome"                                         |
    | action 3: queue_join  ----+                                      |
    | action 4: talk "Goodbye"  |                                      |
    +---------------------------|--------------------------------------+
                                |
                                | Fork creates new stack frame
                                v
    Forked Flow (stack depth 1) - Wait Flow
    +------------------------------------------------------------------+
    | action 1: talk "Please hold..."                                  |
    | action 2: play music.mp3                                         |
    | action 3: goto action 1 (loop)                                   |
    +------------------------------------------------------------------+
                                |
                                | When queue_join completes
                                | (agent answers or caller hangs up)
                                v
    Returns to Main Flow
    +------------------------------------------------------------------+
    | action 4: talk "Goodbye"  <-- continues here                     |
    +------------------------------------------------------------------+


.. code::

    Stack Frame Storage:

    Activeflow record during nested execution:

    +------------------------------------------------------------------+
    | Main Activeflow                                                  |
    +------------------------------------------------------------------+
    | id: "main-af-123"                                                |
    | current_action_id: "queue-join-action"                           |
    | status: "executing"                                              |
    | stack_depth: 0                                                   |
    | forward_action_id: "goodbye-action"  <- return point             |
    +------------------------------------------------------------------+

    +------------------------------------------------------------------+
    | Forked Activeflow (Wait Flow)                                    |
    +------------------------------------------------------------------+
    | id: "forked-af-456"                                              |
    | parent_activeflow_id: "main-af-123"                              |
    | current_action_id: "play-music-action"                           |
    | status: "executing"                                              |
    | stack_depth: 1                                                   |
    +------------------------------------------------------------------+


Event-Driven Execution
----------------------

Many actions wait for external events to continue:

.. code::

    Event Types and Sources:

    +------------------------------------------------------------------+
    |                     Event Processing                             |
    +------------------------------------------------------------------+

    Media Events (from Asterisk via asterisk-proxy):
    +----------------------+------------------------------------------+
    | Event                | Triggers                                 |
    +----------------------+------------------------------------------+
    | PlaybackFinished     | talk/play action completion              |
    | DTMFReceived         | digits_receive completion                |
    | RecordingFinished    | recording_stop or silence/timeout        |
    +----------------------+------------------------------------------+

    Call Events (from call-manager):
    +----------------------+------------------------------------------+
    | Event                | Triggers                                 |
    +----------------------+------------------------------------------+
    | CallAnswered         | connect action (destination answered)    |
    | CallHangup           | Flow termination                         |
    | BridgeDestroyed      | connect completion (party disconnected)  |
    +----------------------+------------------------------------------+

    Queue Events (from queue-manager):
    +----------------------+------------------------------------------+
    | Event                | Triggers                                 |
    +----------------------+------------------------------------------+
    | AgentAnswered        | queue_join completion (success)          |
    | QueueTimeout         | queue_join completion (timeout)          |
    | QueueEmpty           | queue_join completion (no agents)        |
    +----------------------+------------------------------------------+

    AI Events (from ai-manager):
    +----------------------+------------------------------------------+
    | Event                | Triggers                                 |
    +----------------------+------------------------------------------+
    | AITalkComplete       | ai_talk completion                       |
    | TranscribeResult     | Real-time transcription result           |
    +----------------------+------------------------------------------+


.. code::

    Event Routing to Activeflow:

    +------------------------------------------------------------------+
    |                     Event Router                                 |
    +------------------------------------------------------------------+

    1. Event arrives at flow-manager
    +------------------+
    | RabbitMQ Event   |
    | {                |
    |   type: "media"  |
    |   call_id: "xyz" |
    |   event: "done"  |
    | }                |
    +--------+---------+
             |
             v
    2. Look up activeflow by reference_id
    +------------------+
    | SELECT * FROM    |
    | activeflows      |
    | WHERE ref_id =   |
    | "xyz"            |
    +--------+---------+
             |
             v
    3. Resume execution
    +------------------+
    | Execute next     |
    | action           |
    +------------------+


Safety Mechanisms
-----------------

VoIPBIN includes safeguards to prevent runaway flows:

.. code::

    Execution Limits:

    +------------------------------------------------------------------+
    |                     Safety Counters                              |
    +------------------------------------------------------------------+

    Per-Cycle Iteration Limit:
    +------------------------------------------+
    | Max iterations: 1000                     |
    | Counted: Each action in one cycle        |
    | Reset: When flow waits for async event   |
    +------------------------------------------+

    Example:
    +-------+  +-------+  +-------+     +-------+
    | goto  |->| goto  |->| goto  |...->| ERROR |
    +-------+  +-------+  +-------+     +-------+
       1          2          3    ...    1000
                                   (stops here)

    Total Execution Limit:
    +------------------------------------------+
    | Max total executions: 100                |
    | Counted: Each time flow resumes          |
    | Never reset during activeflow lifetime   |
    +------------------------------------------+

    Example:
    Call with 100 DTMF interactions would hit this limit

    On-Complete Chain Limit:
    +------------------------------------------+
    | Max chain depth: 5                       |
    | Counted: on_complete_flow_id triggers    |
    +------------------------------------------+

    Flow A -> Flow B -> Flow C -> Flow D -> Flow E -> STOP
      0         1         2         3         4        5(blocked)


.. code::

    Loop Detection:

    Goto Loop Counter:
    +------------------------------------------------------------------+
    | The goto action has a built-in loop_count parameter              |
    |                                                                  |
    | {                                                                |
    |   "type": "goto",                                                |
    |   "option": {                                                    |
    |     "target_id": "action-123",                                   |
    |     "loop_count": 3   <- Maximum times to execute this goto     |
    |   }                                                              |
    | }                                                                |
    |                                                                  |
    | Execution:                                                       |
    | Loop 1: goto -> action-123                                       |
    | Loop 2: goto -> action-123                                       |
    | Loop 3: goto -> action-123                                       |
    | Loop 4: goto SKIPPED -> continue to next action                  |
    +------------------------------------------------------------------+


Concurrent Execution Handling
-----------------------------

Multiple events can arrive for the same activeflow:

.. code::

    Concurrency Control:

    +------------------------------------------------------------------+
    |                     Execution Locking                            |
    +------------------------------------------------------------------+

    Problem: Two events arrive simultaneously
    +-------------+                    +-------------+
    | Event A:    |                    | Event B:    |
    | DTMF "1"    |                    | Timeout     |
    +------+------+                    +------+------+
           |                                  |
           v                                  v
    +------------------------------------------------------------------+
    |           Both try to resume activeflow                          |
    +------------------------------------------------------------------+

    Solution: Database-level locking
    +------------------------------------------------------------------+
    | 1. Transaction starts                                            |
    | 2. SELECT ... FOR UPDATE on activeflow                           |
    | 3. Check current status                                          |
    | 4. Process event if valid                                        |
    | 5. Update status and cursor                                      |
    | 6. Commit transaction                                            |
    +------------------------------------------------------------------+

    Result:
    - Event A wins (arrives first, gets lock)
    - Event B sees activeflow already moved
    - Event B is discarded as stale


RabbitMQ Message Patterns
-------------------------

Flow-manager uses specific message patterns:

.. code::

    Queue Names:

    +------------------------------------------------------------------+
    | bin-manager.flow-manager.request                                 |
    | - Incoming RPC requests (create activeflow, execute, stop)       |
    +------------------------------------------------------------------+
    | bin-manager.flow-manager.event                                   |
    | - Incoming events (media complete, call hangup, etc.)            |
    +------------------------------------------------------------------+


    Request Message Format:
    +------------------------------------------------------------------+
    | {                                                                |
    |   "uri": "/v1/activeflows",                                      |
    |   "method": "POST",                                              |
    |   "data": {                                                      |
    |     "flow_id": "flow-123",                                       |
    |     "reference_type": "call",                                    |
    |     "reference_id": "call-456"                                   |
    |   }                                                              |
    | }                                                                |
    +------------------------------------------------------------------+


    Event Message Format:
    +------------------------------------------------------------------+
    | {                                                                |
    |   "type": "call",                                                |
    |   "event": "hangup",                                             |
    |   "reference_id": "call-456",                                    |
    |   "data": {                                                      |
    |     "hangup_cause": "normal_clearing"                            |
    |   }                                                              |
    | }                                                                |
    +------------------------------------------------------------------+


Database Schema
---------------

Key tables involved in flow execution:

.. code::

    flows table:
    +------------------------------------------------------------------+
    | Column               | Type          | Description               |
    +----------------------+---------------+---------------------------+
    | id                   | uuid          | Flow definition ID        |
    | customer_id          | uuid          | Owner                     |
    | name                 | varchar(255)  | Display name              |
    | detail               | text          | Description               |
    | actions              | json          | Array of action objects   |
    | on_complete_flow_id  | uuid          | Chain to next flow        |
    | tm_create            | datetime      | Created timestamp         |
    | tm_update            | datetime      | Updated timestamp         |
    | tm_delete            | datetime      | Soft delete timestamp     |
    +------------------------------------------------------------------+

    activeflows table:
    +------------------------------------------------------------------+
    | Column               | Type          | Description               |
    +----------------------+---------------+---------------------------+
    | id                   | uuid          | Activeflow instance ID    |
    | customer_id          | uuid          | Owner                     |
    | flow_id              | uuid          | Flow definition reference |
    | reference_type       | varchar(32)   | "call", "message", "api"  |
    | reference_id         | uuid          | Associated resource ID    |
    | current_action_id    | uuid          | Cursor position           |
    | status               | varchar(32)   | Execution state           |
    | variables            | json          | Runtime variables         |
    | forward_action_id    | uuid          | Return point for forks    |
    | parent_activeflow_id | uuid          | Parent for nested flows   |
    | stack_depth          | int           | Nesting level             |
    | execute_count        | int           | Safety counter            |
    | complete_count       | int           | On-complete chain counter |
    | tm_create            | datetime      | Created timestamp         |
    | tm_update            | datetime      | Last execution timestamp  |
    | tm_end               | datetime      | Completion timestamp      |
    +------------------------------------------------------------------+

