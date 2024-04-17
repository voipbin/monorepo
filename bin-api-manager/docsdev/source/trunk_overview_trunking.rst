.. _trunk-overview-trunking: trunk-overview-trunking

Trunking
========
SIP trunking is a technology that enables organizations to make telephone calls over the internet, rather than traditional phone lines. Instead of using physical phone lines to connect to the public switched telephone network (PSTN), SIP trunking uses an internet connection to carry voice and other communications. SIP trunking is cost-effective, scalable and provides a range of features and benefits, such as better call quality, enhanced disaster recovery capabilities, and the ability to easily manage multiple locations or remote workers. Additionally, SIP trunking allows organizations to consolidate their voice and data networks, reducing the complexity and cost of their telecommunications infrastructure.

Server address
--------------
Once you created trunk, the voipbin generates the trunk server address for you.

`sip:{your voipbin trunk domain name}.trunk.voipbin.net`

Authentication
--------------
Currently, The VoIPBin's trunking authentication supports only the Basic authentication.

* Basic authentication
* IP base authentication(WIP)

Basic authentication
++++++++++++++++++++
To make a SIP outgoing call through a VoIPBin using basic authentication, you need to follow a few steps:

1. Choose a SIP client: You can use a software-based SIP client, such as Zoiper or X-Lite, or a hardware-based SIP phone, such as a Cisco or Grandstream phone.
2. Configure your SIP client: You need to configure your SIP client with VoIPBin credentials, such as your name, extension, password, domain info.
3. Set up your outgoing call settings: In your SIP client, you need to specify the destination address(phone number or extension) you want to call and set any additional options, such as the call type, call quality, or call duration.
4. Initiate the call: Once you have configured your SIP client and set up your outgoing call settings, you can initiate the call by clicking on the call button or using a keypad command.
5. Authenticate your credentials: When you initiate the call, your SIP client sends your authentication credentials to the VoIPBin, using the basic authentication method. The VoIPBin then verifies your credentials and authorizes the call.
6. Make the call: Once your credentials are verified, the VoIPBin establishes the call and connects you with the destination address.

By following these steps, you can make a SIP outgoing call through VoIPBin using basic authentication. This process can be used for a variety of business and personal applications, such as remote work, conferencing, and customer support.

.. code::

    UA                                   VoIPBin                                 Destination

    |                                        |                                        |
    |---------------- INVITE --------------->|                                        |
    |<-- 407 Proxy Authentication Required --|                                        |
    |---------------- ACK ------------------>|                                        |
    |                                        |                                        |
    |----- INVITE with Authorization ------->|                                        |
    |                                        |---------------- INVITE --------------->|
    |                                        |                                        |
    |                                        |<--------------- 100 Trying ------------|
    |<------------- 100 Trying --------------|                                        |
    |                                        |                                        |
    |                                        |<--------------- 180 Ringing -----------|
    |<------------- 180 Ringing -------------|                                        |
    |                                        |                                        |
    |                                        |<---------------- 200 OK ---------------|
    |<------------- 200 OK ------------------|                                        |
    |-------------- ACK -------------------->|                                        |
    |                                        |----------------- ACK ----------------->|
    |                                        |                                        |
    |                                        |                                        |
    |-------------- BYE -------------------->|                                        |
    |                                        |----------------- BYE ----------------->|
    |                                        |<---------------- 200 OK ---------------|
    |<------------- 200 OK ------------------|                                        |


Call handle
-------------------
Unlike the normal VoIPBin's normal call handle, the VoIPBin handles trunking outbound call in a different way. The VoIPBin executes special flow for the trunking call.
It executes the follow features:

* Enable the early media.
* Relay the hangup cause.

Early media handle
++++++++++++++++++
The VoIPBin enables the the early-media feature for the trunking outbound call.

.. code::

    UA                                   VoIPBin                                 Destination

    |                                        |                                        |
    ===================================================================================
    |----- INVITE with Authorization ------->|                                        |
    |                                        |---------------- INVITE --------------->|
    |                                        |<--------------- 100 Trying ------------|
    |<------------- 100 Trying --------------|                                        |
    |                                        |<--------------- 183 Ringing -----------|
    |<------------- 183 Ringing -------------|                                        |
    |<------------- RTP Media ---------------|<---------------- RTP Media ------------|


Realy hangup cause
++++++++++++++++++
The VoIPBin delivers the hangup cause code from the outgoing call.

.. code::

    UA                                   VoIPBin                                 Destination

    |                                        |                                        |
    ===================================================================================
    |----- INVITE with Authorization ------->|                                        |
    |                                        |---------------- INVITE --------------->|
    |                                        |<--------------- 100 Trying ------------|
    |<------------- 100 Trying --------------|                                        |
    |                                        |<--------------- 404 NOT FOUND ---------|
    |<------------- 404 NOT FOUND -----------|                                        |
