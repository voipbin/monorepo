.. _domain-overview:

Overview
========
The SIP Domain resource in Voipbin describes a custom DNS hostname that can accept SIP (Session Initiation Protocol) traffic for your account. When a SIP request is made to this domain, such as sip:alice@example.sip.voipbin.net, it is routed to Voipbin. Voipbin then authenticates the request and forwards it to the specified voice_url associated with the SIP domain.

.. image:: _static/images/domain_overview_flow.png

The flow diagram illustrates the process of handling SIP traffic using the SIP Domain resource:

SIP Request: A SIP request is made to the custom DNS hostname, which points to the Voipbin platform.

Routing: The SIP request is routed to Voipbin for further processing.

Authentication: Voipbin authenticates the incoming SIP request to ensure its validity and security.

Voice URL: The authenticated SIP request is then forwarded to the voice_url associated with the specific SIP domain. The voice_url contains the endpoint where Voipbin should direct the call or communication.

The SIP Domain resource acts as a vital component in managing SIP traffic for your Voipbin account. It allows you to handle incoming SIP requests from various sources, enabling seamless communication and integration with your Voipbin services. This capability is particularly useful for businesses and developers looking to manage their SIP-based communications securely and efficiently. By leveraging the SIP Domain resource, organizations can create custom DNS hostnames, integrate Voipbin services into their existing systems, and build scalable and reliable SIP-based communication solutions.
