.. _chatbot-overview: chatbot-overview

Overview
========
A chatbot is a computer program designed to simulate conversation with human users, typically over messaging platforms. Chatbots use natural language processing (NLP) and artificial intelligence (AI) technologies to understand and interpret user requests, and then respond with appropriate information or actions.

Chatbots can be used for a variety of tasks, such as answering customer inquiries, providing support and guidance, making reservations, or even just for entertainment. They can be implemented on websites, messaging apps, social media platforms, and other digital channels, and can help to streamline customer service and support processes by providing quick and accurate responses to common questions and issues.

In the VoIPBin, you can set your chatbot and connect the call to the chatbot. So you can implement the voice chatbot talk easily.

Init prompt
===========
An initial prompt to a chatbot is the starting point or the first message given to a chatbot system to initiate a conversation or request information. It serves as the input or query that sets the context for the chatbot to generate a response. The prompt can be in the form of a question, statement, or any text-based instruction provided by a user.

When interacting with a chatbot, the initial prompt plays a crucial role in determining the quality and relevance of the chatbot's response. It should ideally be clear, concise, and specific enough to convey the desired information or task to the chatbot. The prompt can be tailored to a specific domain or topic, and it can be open-ended or structured, depending on the desired outcome.

The chatbot system processes the initial prompt using its underlying algorithms and models, which have been trained on large datasets to understand and generate human-like text. It analyzes the prompt's content, context, and intent, and then generates a response based on its learned knowledge and patterns.

It's important to note that the chatbot's response is generated based on its understanding of the prompt and the information available in its training data. The chatbot does not possess true understanding or consciousness but rather mimics human-like responses based on statistical patterns and correlations. The quality and accuracy of the chatbot's response depend on the training data, the sophistication of the chatbot model, and the complexity of the prompt provided.

Overall, the initial prompt is the starting point of interaction with a chatbot and serves as the input that triggers the chatbot's response generation process. It sets the stage for a conversation or inquiry and guides the chatbot in providing relevant information or completing a requested task.

You can provide any string and here's very simple example to getting the list actions array in a JSON format.

.. code::

    Pretend you are an expert customer service agent.

    Please response it kindly.

    But, if you received a request to connect to the agent,
    response the next message in a json format.
    Do not include any explanations in the response. Only provide a RFC8259 compliant JSON response following this format without deviation.

    [
        {
            "action": "connect",
            "option": {
                "source": {
                    "type": "tel",
                    "target": "+821100000001"
                },
                "destinations": [
                    {
                        "type": "tel",
                        "target": "+821100000002"
                    }
                ]
            }
        }
    ]


Response message
================
When a chatbot generates a response, it can be presented in various forms depending on the implementation.

In the case of a normal text string response, the chatbot will provide the response message as text, which can be read or played as an audio using Text-to-Speech technology. This type of response is suitable for regular conversational interactions.

However, there are scenarios where the chatbot's response takes the form of a JSON format containing a list of :ref:`action <flow-struct-action-action>` objects. This indicates that the chatbot aims to provide instructions for executing specific :ref:`actions <flow-struct-action-action>`.

The JSON format allows for structured data representation and facilitates communication between the chatbot and the systems it interacts with. Within the JSON response, the list of action objects represents the actions to be performed.

Each action object within the list contains relevant details and parameters necessary for executing the intended action. These details may include instructions, commands, or data inputs required for the action's execution.

When the chatbot generates this type of response, it signifies that the chatbot's role extends beyond providing textual information. It actively interacts with external systems or services by conveying instructions for triggering specific actions.

The recipient of the chatbot's response, typically a service or system, processes the list of action objects and executes the indicated actions accordingly. This seamless integration enables the chatbot to influence the behavior and functionality of the connected service.

It's important to note that the implementation and interpretation of action objects may vary depending on the chatbot platform, the connected service, and the communication protocol agreed upon.

In summary, when a chatbot generates a response, it can be a normal text string or a JSON format containing a list of action objects. The former is suitable for regular conversation, while the latter indicates the chatbot's intention to provide instructions for specific actions. The JSON response enables seamless integration with external systems, allowing the chatbot to influence and trigger actions within those systems.
