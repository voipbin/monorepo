# Test
```
$ mysql -u <db-user> -p call_manager -h <db-host> < ./table_channels.sql
```


# Call
status
```
* dialing: The call is created. We are dialing to the destination.
* ringing: The destination has confirmed that the call is ringng.
* progressing: The call has answered. The both endpoints are talking to each other.
* terminating: The call is terminating.
* canceling: The call originator is canceling the call.
* hangup: The call has been terminated.
```

hangup reason
```
* normal: The call has ended after answer.
* failed: The call attempt(signal) was not reached to the phone network.
* busy: The destination is on the line with another caller.
* cancel: The call was cancelled by the originator before it was answered.
* timeout: The call reached max call duration after it was answered. This timeout is fired by our time out.(outgoing call)
* unanswer: The destination didn't answer until destination's timeout.
```


```
* started: The call is created.
* ringing: The destination has confirmed that the call is ringing.
* answered: The call has answred.
* 
* hangup: The call has been terminated after answered.

hangup cause
* failed: The call attempt(signal) was not reached to the phone network.
* busy: The destination is on the line with another caller.
* cancelling: The call is being cancel. - hide?
* cancelled: The call was cancelled by the originator before it was answered.
* timeout: The call reached max call duration after it was answered.
* noanswer: The destination didn't answer until destination's timeout.
* dialout: The call reached timeout before it was answered. This timeout is fired by our time out.(outgoing call)
```

# NOTE
id: `<db-user>`
pass: `<db-password>`

database: call_manager

