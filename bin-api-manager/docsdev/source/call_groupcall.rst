.. _call-groupcall:

Groupcall
=========
The Groupcall feature in VoIPBIN is a blast calling functionality that enables team members to communicate in real-time. When a Group Call is initiated, an alert is sent to all team members, allowing them to join the call with a single click. The feature uses VoIP technology to enable high-quality audio communication and offers benefits such as increased productivity, improved collaboration, and enhanced connectivity.

.. note:: **AI Implementation Hint**

   Groupcalls create multiple simultaneous outbound calls, each of which is individually chargeable. The number of calls created equals the number of destinations in the groupcall request. Use ``call_ids`` in the groupcall response to track the status of each individual call via ``GET /calls/{call-id}``.

Ringall
-------
Ringall sends the dial request to all destinations simultaneously.

The ringall ring method is a way to make calls to multiple destinations simultaneously. When you initiate a groupcall using the ringall method, VoIPBIN will place calls to all of the destinations on your list at once.
This means that each destination's phone will start ringing simultaneously, and the person who answers first will begin executing the call flow specified for that groupcall, while all other destinations that have not yet been answered will be hung up immediately.
This ensures that only one call is active at a time and the call flow is executed by the person who answered first.

.. code::

    Client           VoIPBIN        Destination-1    Destination-2
    |                  |                |                |
    |    Groupcall     |                |                |
    |    request       |                |                |
    |----------------->|                |                |
    |                  |                |                |
    |                  | Dial           |                |
    |                  |--------------->|                |
    |                  |                |                |
    |                  | Dial           |                |
    |                  |-------------------------------->|
    |                  |                |                |
    |                  |                |         Answer |
    |                  |<--------------------------------|
    |                  |                |                |
    |                  | Cancel         |                |
    |                  |--------------->|                |

The diagram shows the sequence of events for a Group Call request in VoIPBIN with two destination endpoints, Destination-1 and Destination-2.
* The Client initiates the Group Call request by sending a request message to the VoIPBIN server.
* The server then sends a Dial message to Destination-1 and Destination-2 to establish the call.
* After Destination-2 answers the call, it sends an Answer message back to the server.
* The VoIPBIN cancels the call to Destination-1 by sending a Cancel message to Destination-1.

.. note:: **AI Implementation Hint**

   With ``ring_all``, if no destination answers within the dial timeout, all calls will be hung up with ``hangup_reason: "dialout"``. Set an appropriate ``dial_timeout`` (default: 30000 ms) to allow enough ring time. For PSTN destinations, 45000-60000 ms is recommended.

Linear
------
The linear ring method is a way to call a list of destinations one by one, in a specific order.

When you initiate a groupcall using the linear method, VoIPBIN will call the first destination on your list. If that destination does not answer, VoIPBIN will move on to the next destination on the list and call it instead.
This process will continue until one of the destinations answers the call, at which point the call flow specified for the groupcall will be executed. If all of the destinations on the list have been called and none of them have answered, the call will end without any further action.

The linear method is useful when you want to call a list of destinations in a specific order and don't want to simultaneously ring all destinations at once.

For example, you might use the linear method for a sales team to call potential clients one by one, in a specific order based on priority.

.. code::

    groupcall1            destinationA       destinationB        destinationC
        |----- ring ---------->|                   |                   |
        |<---- no answer ------|                   |                   |
        |                      |                   |                   |
        |----- ring ------------------------------>|                   |
        |<---- no answer --------------------------|                   |
        |                      |                   |                   |
        |----- ring -------------------------------------------------->|
        |<---- answer -------------------------------------------------|

.. note:: **AI Implementation Hint**

   With ``linear`` ring method, the total time to reach the last destination is the sum of all individual dial timeouts. For example, with 5 destinations and a 30-second timeout each, the last destination will not be tried until up to 150 seconds have elapsed. Plan your destination order and timeouts accordingly.

Nested groupcall
----------------
A nested groupcall is a groupcall that is included as one of the destinations in another groupcall. When a groupcall with a nested groupcall is initiated, the nested groupcall is also initiated, creating a "nested" groupcall within the main groupcall.

For example, let's say you have a groupcall with the following list of destinations: Destination A, Destination B, and Destination C. Destination C is a nested groupcall that includes its own list of destinations: Destination X and Destination Y.

When you initiate the main groupcall, VoIPBIN will begin calling Destination A and Destination B simultaneously according to the ring method you've specified (either ringall or linear). When it reaches Destination C, the nested groupcall is initiated and VoIPBIN will begin calling Destination X and Destination Y according to the ring method specified in the nested groupcall.

Once a destination in the nested groupcall has answered the call, the flow specified for that groupcall is executed. The child groupcall informs the master groupcall that it has answered call. The master groupcall then hangs up any remaining calls that have not yet been answered in the child groupcall, and stops calling the remaining destinations in the main groupcall list.

In VoIPBIN, the main groupcall is considered the "master" groupcall and the nested groupcall is considered a "chained" groupcall. Each chained groupcall is assigned a unique ID, and the IDs of all chained groupcalls are stored in a list within the master groupcall. This allows VoIPBIN to keep track of all nested groupcalls and their current status within the main call.

It is also possible to have chained groupcalls within chained groupcalls, creating multiple levels of nesting. This means that a nested groupcall can itself include another groupcall as one of its destinations, forming a chain of groupcalls. The nested groupcalls can continue to be chained in a cascading manner, allowing for complex call flows and routing scenarios.

For example, the main groupcall may include a chained groupcall as one of its destinations, and that chained groupcall may, in turn, include another chained groupcall within it. This nesting can extend to multiple levels, providing a highly flexible and customizable approach to call routing and management.

By allowing nested and chained groupcalls, VoIPBIN empowers users to design and implement intricate call flows that cater to their specific needs. This functionality opens up possibilities for applications such as multi-level call routing, call forwarding to different departments or teams, and advanced call handling scenarios.

The ability to include nested groupcalls within a main groupcall is a powerful feature that allows for more complex call flows and routing strategies. It can be used, for example, to create more sophisticated call routing trees that can handle a wide range of call scenarios and use cases.

.. code::

    groupcall1            destinationA        destinationB        destinationC
        |----- ring ---------->|                   |       (groupcall destination linear)
        |----- ring ------------------------------>|                   |                         groupcall2        destinationX        destinationY
        |----- ring -------------------------------------------------->|-- start a nested groupcall-->|                 |                   |
        |                      |                   |                                                  |----- ring ----->|                   |
        |                      |                   |                                                  |<-- no answer ---|                   |
        |                      |                   |                                                  |----- ring ------------------------->|
        |                      |                   |                                                  |<-- answer --------------------------|
        |<---------------------------------------- inform that groupcall2 got answered call ----------|
        |----- cancel -------->|                   |
        |----- cancel ---------------------------->|

.. note:: **AI Implementation Hint**

   Nested groupcalls multiply the number of outbound calls created. A master groupcall with 3 destinations, where one destination is a nested groupcall with 3 more destinations, can create up to 5 simultaneous calls (with ``ring_all``). Each call is individually chargeable. When the nested groupcall answers, the master cancels all remaining unanswered calls across all levels.
