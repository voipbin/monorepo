.. voice_basic:

Basic
=====

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


Direction
---------
Call's direction.

=========== ============
Direction   Description
=========== ============
incoming    Call is coming from outside from voipbin.
outgoing    Call is generating form the voipbin.
=========== ============


Hangup by
---------
The Hangup by shows which endpoint sent the hangup request first.

=========== ============
hangup by   Description
=========== ============
remote      The remote end hangup the call first.
local       The local end hangup the call first.
=========== ============


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


