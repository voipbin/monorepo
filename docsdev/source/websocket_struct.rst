.. _websocket_struct:

Struct
======

.. _websocket_struct_subscribe:

Subscribe
---------
.. code::

    {
        "type": "unsubscribe",
        "topics": [...]
    }

* topic: List of subscribe topics.

Example
+++++++
.. code::

    {
        "type": "subscribe",
        "topics": ["call"]
    }

.. _websocket_struct_unsubscribe:

Unsubscribe
-----------
.. code::

    {
        "type": "unsubscribe",
        "topics": [...]
    }

* topic: List of unsubscribe topics.

Example
+++++++
.. code::

    {
        "type": "subscribe",
        "topics": ["call"]
    }
