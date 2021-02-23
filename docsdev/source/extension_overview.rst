.. _extension-overview: extension_overview

Overview
========

In order for your SIP endpoint to receive calls from Voipbin, you need a domain, extension and registration.

Calling a registered SIP endpoint works the same as calling any other SIP URI, only you will now be using the AOR of your registered SIP extension(endpoint).

When calling your registered SIP extension(endpoint), you should use the general SIP domain URI, omitting the Voipbin's sip URI i.e.:

`{extension}@{your voipbin domain}.sip.voipbin.net`
