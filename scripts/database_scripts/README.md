# Test
```
$ mysql -u call-manager -p call_manager -h 10.126.80.5 < ./table_channels.sql
```


# Call
status
```
* started: The call is created.
* ringing: The destination has confirmed that the call is ringing.
* answered: The call has answred.
* completed: The call has been terminated after answered.

* failed: The call attempt(signal) was not reached to the phone network.
* busy: The destination is on the line with another caller.
* cancelling: The call is being cancel. - hide?
* cancelled: The call was cancelled by the originator before it was answered.
* timeout: The call timed out before it was answered. This timeout is fired by our time out.(outgoing call)
* noanswer: The destination didn't answer until destination's timeout.
```

# NOTE
id: call-manager
pass: 47f94686-8184-11ea-bfe8-e791e06ef5ef

database: call_manager

