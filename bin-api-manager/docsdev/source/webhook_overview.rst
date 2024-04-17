.. _webhook-overview:

Overview
========
Webhooks, a robust feature offered by VoIPBIN, empower users to receive real-time event data for their calls and associated resources directly on their servers. By establishing custom endpoints, users can seamlessly configure their servers to receive timely notifications and updates related to VoIPBIN resources, thereby enhancing control, customization, and real-time visibility within their communication workflows.

Notification Mechanism
----------------------
Webhook events act as notifications sent by VoIPBIN, triggering when specific events or actions unfold within the system, such as call events, message events, or changes to resources like queues or agents. Configured webhook endpoints receive these events, ensuring users promptly receive pertinent data related to their VoIPBIN resources.

.. image:: _static/images/webhook_overview_notification.png

Types of Webhooks
-----------------
VoIPBIN tailors webhooks for each resource type, ensuring users receive granular progress and updates for various events. This resource-specific approach allows users to monitor their VoIPBIN resources with precision, obtaining notifications and data tailored to each resource type. For instance, call-specific webhook events furnish details on call status, duration, and caller ID, while message-specific events offer insights into SMS or MMS messages, including content, sender ID, and delivery status.

Custom Endpoints
----------------
To harness webhooks, users must configure custom webhook endpoints on their servers. These endpoints, serving as URLs, dictate where VoIPBIN transmits webhook events. Upon an event occurrence, VoIPBIN initiates an HTTP request to the configured endpoint, incorporating relevant data in the payload. This empowers users to process and respond to events according to their unique requirements.

Benefits of Webhooks
--------------------
Webhooks deliver a range of advantages for VoIPBIN users:

* Real-Time Updates: Offering immediate event notifications, webhooks keep users abreast of real-time changes to their VoIPBIN resources.
* Customization: Users can tailor webhook endpoints and process data as per their specific needs, facilitating the creation of customized workflows and integrations.
* Automated Actions: Webhooks enable users to automate actions based on event data, such as record updates, notifications, or the initiation of additional processes.
* Enhanced Monitoring: Providing a proactive monitoring solution, webhooks empower users to track and respond promptly to changes within the VoIPBIN system, ensuring informed decision-making.
