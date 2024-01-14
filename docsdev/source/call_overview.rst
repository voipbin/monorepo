.. _call-overview:

Overview
========
The VoIPBIN call API provides a straightforward and convenient way to develop high-quality call applications in the Cloud. With the VoIPBIN API, developers can leverage familiar web technologies to build scalable and feature-rich call applications, giving them the power to control inbound and outbound call flows using JSON-based VoIPBIN actions. Additionally, the API offers capabilities to record and store inbound or outbound calls, create conference calls, and send text-to-speech messages in multiple languages with different gender and accents.

With the VoIPBIN API you can:

- Build apps that scale with the web technologies you are already using.
- Control the flow of inbound and outbound calls in JSON with VoIPBIN's actions.
- Record and store inbound or outbound calls.
- Create conference calls.
- Send text-to-speech messages in 50 languages with different gender and accents.

Protocol
--------
VoIPBIN offers support for various call/video protocols, enabling users to join the same conference room and communicate with one another seamlessly. The flexibility in protocol options ensures efficient and reliable communication between different devices and platforms.

.. image:: _static/images/call_overview_protocol.png

PSTN/Phone Number Format
------------------------
In the VoIPBIN APIs, all PSTN/Phone numbers must adhere to the `+E164 <https://en.wikipedia.org/wiki/E.164>`_ format. This format standardizes the representation of phone numbers to facilitate smooth communication and interoperability across different systems.

Key requirements for phone numbers within the VoIPBIN APIs:

* The phone number must have the '+' symbol at the beginning.
* The number should not contain any special characters, such as spaces, parentheses, or hyphens.

For example, a US phone number should be represented as +16062067563, and a Korea phone number should be represented as +821021656521.

Extension Number Format
-----------------------
The extension numbers used in the VoIPBIN system can be customized according to specific requirements. However, they must adhere to the following limitation:

* Extension numbers should not contain any special characters, such as spaces, parentheses, or hyphens.

The absence of special characters ensures consistent and reliable processing of extension numbers within the VoIPBIN system, promoting smooth communication and interaction.

With the VoIPBIN call API's robust features and support for various protocols, developers can create versatile and efficient call applications, enabling seamless communication and collaboration for end-users.

Incoming call
-------------
The VoIPBin system provides the functionality to receive incoming calls from external parties. This feature allows users to accept and handle incoming calls through their VoIP services. Incoming calls are crucial for various communication applications and call center setups as they enable users to receive inquiries, provide support, and engage with customers, clients, or other users.
When an incoming call is received, the VoIPBin system processes the call request and prepares for call handling based on the specified parameters and configurations.

Execution of Call Flow for incoming call
----------------------------------------
The execution of the call flow for incoming calls involves a simple yet effective sequence of actions:

* Call Verification: When an incoming call is received, the VoIPBin system verifies the call's authenticity and checks for any potential security risks, such as spoofed or fraudulent calls. This verification process ensures that legitimate calls are allowed to proceed.
* Determine Call Flow: After successful verification, the system determines the appropriate call flow based on the destination of the incoming call. The call flow includes a set of predefined actions and configurations tailored to handle calls directed to a specific user, department, or interactive voice response (IVR) system.
* Execute Call Flow: Once the call flow is determined, the system proceeds to execute it without delay. The call flow actions are triggered in accordance with the predefined configuration for the call destination.
* End the Call: After executing the call flow actions, the system initiates the process of ending the call. The call is terminated, and the connection with the external party is disconnected.

By following this streamlined call flow process, the VoIPBin system efficiently handles incoming calls, ensures their secure and verified handling, and executes the appropriate flow actions based on the call destination. After executing the call flow, the system promptly ends the call, completing the call handling process for the incoming call. Customizable flow actions allow users to tailor the call handling process according to their application's needs, optimizing user experience and call management efficiency.

.. image:: _static/images/call_incoming.png

Outgoing call
-------------
The VoIPBin system offers the outgoing call feature, enabling users to initiate calls to external parties through their VoIP services. This feature is commonly used in various communication applications and call center setups to establish connections with customers, clients, or other users outside the organization.
To utilize the outgoing call feature, users need to provide the necessary call parameters, such as the destination phone number, caller ID information, and any additional call settings. These parameters are submitted to the VoIPBin system, which then processes the request and attempts to establish a connection with the specified destination.

Execution of Call Flow for outgoing call
----------------------------------------
Once the outgoing call request is initiated, the VoIPBin system starts the process of connecting to the destination phone number. During this phase, the system waits for the called party to answer the call. The call flow refers to the sequence of actions and events that occur from the moment the call is initiated until it is successfully answered or terminated.

The call flow execution occurs as follows:

* Initiation: The user triggers the outgoing call request, providing the necessary call parameters.
* Call Setup: The VoIPBin system processes the request and establishes a connection with the destination phone number.
* Wait for Call Answer: After the call setup, the system waits for the called party to answer the call. This waiting period involves ringing the called party's phone and monitoring the call status.
* Call Answered: Once the called party answers the outgoing call, the system proceeds to execute the predefined call flow actions.
* Flow Actions Execution: The call flow actions are a set of customizable operations that are executed upon call answer. These actions can include call recording, call routing, call analytics, notifications, and post-call actions, among others.

The call flow execution is critical for ensuring a smooth and efficient communication experience. By customizing the flow actions, users can tailor the call handling process to meet the specific requirements of their application or service, enhancing user engagement and overall call management.

.. image:: _static/images/call_outgoing.png

Error handling and Termination
------------------------------
During the incoming/outgoing call process, various errors may occur, such as call failures or network issues.
The VoIPBin system have robust error handling mechanisms to gracefully manage such situations. In case of a failed call attempt or call rejection, the system log relevant information for further analysis or reporting purposes.

Call concept
-------------
The concept of a call in Voipbin departs from the traditional 1:1 call model. Here's an overview:

In Voipbin, a call includes source, destination, and additional metadata. Moreover, the call can be associated with multiple other calls, creating a dynamic journey that goes beyond the standard 1:1 connection. Envision a call's trajectory as it connects to an agent and then diverges to another destination.

In Voipbin, the conventional call scenario A -> B is delineated by two distinct calls:

.. code::

    A            Voipbin            B
    |<-- Call 1 --->|               |
    |               |<--- Call 2 -->|
    |<-----RTP----->|<-----RTP----->|

Comparison: Traditional Call Concept vs Voipbin Call Concept
++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

Traditional Call Concept

* Follows a 1:1 model where a call is a direct connection between a source and a destination.
* Typically involves a straightforward flow from the caller to the recipient.
* Limited in handling complex call journeys or interactions with multiple parties.

Voipbin Call Concept

* Deviates from the traditional 1:1 model, allowing for more intricate call structures.
* Encompasses source, destination, and additional metadata in a call.
* Permits connections to multiple other calls, creating dynamic call journeys.
* Visualizes a call's path, which may involve connecting to an agent and branching to additional destinations.

In summary, while the traditional call concept adheres to a simple point-to-point model, the Voipbin call concept introduces a more flexible and multifaceted approach, accommodating diverse call scenarios and interactions.
