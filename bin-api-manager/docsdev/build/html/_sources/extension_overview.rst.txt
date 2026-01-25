.. _extension-overview: extension_overview

Overview
========
To enable your SIP endpoint to receive calls from VoIPBIN, you must set up a domain, extension, and registration.

Calling a registered SIP endpoint follows the same process as calling any other SIP URI. However, you will now use the Address of Record (AOR) of your registered SIP extension (endpoint).

Registration address
--------------------
When calling your registered SIP extension (endpoint), you should use the general SIP domain URI, omitting the VoIPBIN's SIP URI. Specifically, the format to call your registered SIP extension is as follows:

`{extension}@{your voipbin customer id}.registrar.voipbin.net`

By using this format, VoIPBIN can route the call to your registered SIP endpoint based on the extension and domain information provided. This allows you to receive incoming calls on your SIP endpoint and handle them using your SIP-enabled system.

The setup of domain, extension, and registration is crucial for integrating your SIP-based communication infrastructure with VoIPBIN. Once properly configured, you can seamlessly communicate with your SIP endpoint through VoIPBIN's SIP services, ensuring efficient and reliable voice communication for your business or application needs. This integration empowers businesses and developers to create robust and scalable communication solutions, streamlining call handling and enhancing overall communication experiences.

.. _extension-overview-registration: extension-overview-registration

Registration
------------
The extension registration is the process of identifying and authenticating a user or device on a SIP network. When a device wants to make or receive calls or messages on the network, it must first register with the VoIPBIN by sending a REGISTER message.

This message includes information about the user or device, such as its address, phone number, and authentication credentials (e.g., username and password). The VoIPBIN verifies the user's credentials and then responds with a confirmation message, such as a 200 OK response.

Once the device is registered, it can make and receive calls and messages on the SIP network. SIP registration is important for ensuring secure and reliable communication between devices on the network, as well as for enabling advanced features like call forwarding, voicemail, and presence.

407 Proxy Authentication Required
---------------------------------
When a user or device sends a SIP REGISTER message to a VoIPBIN with basic authentication, the VoIPBIN will typically respond with a 407 Proxy Authentication Required message if the user's credentials are not valid or not provided.

Here is an example of this process:

1. The user or device sends a REGISTER message to the VoIPBIN with its identification information.
2. The VoIPBIN checks the user's credentials, and if they are not valid or not provided, it responds with a 407 Proxy Authentication Required message.
3. The 407 response includes a "Nonce" value, which is a unique and random number used to help prevent replay attacks. The user's device must use this "Nonce" value, along with the username, password, and other information, to create an "Authorization" header for the next REGISTER message.
4. The user's device sends a second REGISTER message with the Authorization header, including the "Nonce" value and other authentication information.
5. The VoIPBIN verifies the user's credentials using the basic authentication method, and if they are valid, it sends a 200 OK response, confirming the registration.

Once the user's device is registered, it can receive calls and messages on the SIP network.

.. code::

    UA                                   VoIPBin

    |                                        |
    |-------------- REGISTER --------------->|
    |                                        |
    |<-- 407 Proxy Authentication Required --|
    |---------------- ACK ------------------>|
    |                                        |
    |----- REGISTER with Authorization ----->|
    |                                        |
    |<------------- 200 OK ------------------|
    |-------------- ACK -------------------->|
