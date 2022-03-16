.. _call-struct: call-struct

Struct
======

.. _call-struct-call:

Call
----

.. code::

    {
        "id": "<string>",
        "flow_id": "<string>",
        "type": "<string>",
        "master_call_id": "<string>",
        "chained_call_ids": [
            "<string>",
            ...
        ],
        "recording_id": "<string>",
        "recording_ids": [
            "<string>",
            ...
        ],
        "source": {
            ...
        },
        "destination": {
            ...
        },
        "status": "<string>",
        "action": {
            ...
        },
        "direction": "<string>",
        "hangup_by": "<string>",
        "hangup_reason": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_progressing": "<string>",
        "tm_ringing": "<string>",
        "tm_hangup": "<string>"
    }

* *id*: Call's ID.
* *flow_id*: Call's flow id.
* *type*: Call's type. See detail :ref:`here <call-struct-type>`.
* *master_call_id*: Master call's id. If the master_call_id set, it follows master call's hangup.
* *chained_call_ids*: List of chained call ids. If the call hangs up, the chained call also will hangup.
* *recording_id*: Shows currently recording id.
* *recording_ids*: List of recording ids.
* *source*: Source address info. See detail :ref:`here <call-struct-address>`.
* *destination*: Destination address info. See detail :ref:`here <call-struct-address>`.
* *status*: Call's status. See detail :ref:`here <call-struct-status>`.
* *action*: Call's current action. See detail :ref:`here <flow-action>`.
* *direction*: Call's direction. See detail :ref:`here <call-struct-direction>`.
* *hangup_by*: Shows call's hangup end. See detail :ref:`here <call-struct-hangupby>`.
* *hangup_reason*: Show call's hangup reason. See detail :ref:`here <call-struct-hangupreason>`.

.. _call-struct-type:

Type
----
Call's type.

=========== ============
Type        Description
=========== ============
flow        Executing the call-flow
conference  Conference call.
sip-service sip-service call. Will execute the corresponding the pre-defined sip-service by the destination.
=========== ============

.. _call-struct-status:

Status
------
Call's status.

=========== ===================
Status      Description
=========== ===================
dialing     The call is created. We are dialing to the destination.
ringing     The destination has confirmed that the call is ringng.
progressing The call has answered. The both endpoints are talking to each other.
terminating The call is terminating.
canceling   The call originator is canceling the call.
hangup      The call has been completed.
=========== ===================

.. _call-struct-direction:

Direction
---------
Call's direction.

=========== ============
Direction   Description
=========== ============
incoming    Call is coming from outside from voipbin.
outgoing    Call is generating form the voipbin.
=========== ============

.. _call-struct-hangupby:

Hangup by
---------
The Hangup by shows which endpoint sent the hangup request first.

=========== ============
hangup by   Description
=========== ============
remote      The remote end hangup the call first.
local       The local end hangup the call first.
=========== ============

.. _call-struct-hangupreason:

Hangup reason
-------------
Shows why the call was hungup.

=========== ============
Reason      Description
=========== ============
normal      The call has ended after answer.
failed      The call attempt(signal) was not reached to the phone network.
busy        The destination is on the line with another caller.
cancel      Call was cancelled by the originator before it was answered.
timeout     Call reached max call duration after it was answered.
unanswer    Destination didn't answer until destination's timeout.
dialout     The call reached dialing timeout before it was answered. This timeout is fired by our time out(outgoing call).
=========== ============

.. _call-struct-address:

Address
-------
Defines target(source/destination) address.

.. code::

    {
        "type": "<string>",
        "target": "<string>",
        "target_name": "<string>",
        "name": "<string>",
        "detail": "<string>"
    }

* *type*: Address type. See detail :ref:`here <call-struct-address-type>`.
* *target*: address endpoint.
* *target_name*: address's name.
* *name*: Name.
* *detail*: Detail description.

.. _call-struct-address-type:

Address type
------------
Defines types of address.

=========== ============
Type        Description
=========== ============
agent       Used for calling to the agent
endpoint    Used for calling to endpoint(extension@domain)
sip         SIP type address.
tel         Telephone type address.
=========== ============

