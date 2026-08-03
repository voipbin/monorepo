.. _trunk-overview-domain_name:

Domain name
============

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free. SIP domains are automatically generated as part of trunk and extension configuration.
   * **Async:** No. Domain names are generated immediately when creating trunks or extensions.

The SIP Domain resource in VoIPBIN entails a personalized DNS hostname designed to accept SIP (Session Initiation Protocol) traffic for your account. When a SIP request is directed to this domain, like sip:alice@example.trunk.voipbin.net, it is directed to VoIPBIN. Subsequently, VoIPBIN authenticates the request against the trunk's configured ``auth_types`` (basic username/password and/or the source IP against ``allowed_ips``) and, once authorized, routes the call out to its destination.

This pivotal component facilitates the management of SIP traffic within your VoIPBIN account. It accommodates incoming SIP requests from diverse sources, ensuring seamless communication and integration with VoIPBIN services. Businesses and developers can leverage the SIP Domain resource to create bespoke DNS hostnames, seamlessly integrate VoIPBIN services into existing systems, and construct scalable, reliable SIP-based communication solutions. This capability is particularly beneficial for organizations seeking to manage SIP-based communications securely and efficiently.

.. note:: **AI Implementation Hint**

   VoIPBIN uses two domain patterns: ``{customer-id}.registrar.voipbin.net`` for SIP device registration (extensions), and ``{domain-name}.trunk.voipbin.net`` for SIP trunking (outbound calls). The domain is auto-generated -- you do not need to create or manage DNS records yourself.
