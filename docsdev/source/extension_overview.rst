.. _extension-overview: extension_overview

Overview
========
To enable your SIP endpoint to receive calls from Voipbin, you must set up a domain, extension, and registration.

Calling a registered SIP endpoint follows the same process as calling any other SIP URI. However, you will now use the Address of Record (AOR) of your registered SIP extension (endpoint).

When calling your registered SIP extension (endpoint), you should use the general SIP domain URI, omitting the Voipbin's SIP URI. Specifically, the format to call your registered SIP extension is as follows:

`{extension}@{your voipbin domain}.sip.voipbin.net`

By using this format, Voipbin can route the call to your registered SIP endpoint based on the extension and domain information provided. This allows you to receive incoming calls on your SIP endpoint and handle them using your SIP-enabled system.

The setup of domain, extension, and registration is crucial for integrating your SIP-based communication infrastructure with Voipbin. Once properly configured, you can seamlessly communicate with your SIP endpoint through Voipbin's SIP services, ensuring efficient and reliable voice communication for your business or application needs. This integration empowers businesses and developers to create robust and scalable communication solutions, streamlining call handling and enhancing overall communication experiences.
