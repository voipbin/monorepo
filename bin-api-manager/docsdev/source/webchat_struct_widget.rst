.. _webchat-struct-widget:

Widget
======

.. _webchat-struct-widget-widget:

Widget
------
Widget struct

.. code::

    {
        "id": "<string>",
        "customer_id": "<string>",
        "name": "<string>",
        "status": "<string>",
        "flow_id": "<string>",
        "session_idle_timeout": <number>,
        "theme_config": {
            "primary_color": "<string>",
            "logo_url": "<string>",
            "position": "<string>"
        },
        "direct_hash": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The widget's unique identifier. Returned when creating a widget via ``POST /webchat_widgets`` or when listing widgets via ``GET /webchat_widgets``.
* ``customer_id`` (UUID): The customer who owns this widget. Obtained from the ``id`` field of ``GET /customers``.
* ``name`` (String): Human-readable name for the widget.
* ``status`` (enum string): The widget's status. See :ref:`Status <webchat-struct-widget-status>`.
* ``flow_id`` (UUID): The flow triggered on the visitor's first inbound message in each session. Obtained from the ``id`` field of ``GET /flows``.
* ``session_idle_timeout`` (Integer): Session idle timeout in seconds before an inactive session is automatically ended. Default ``1800`` (30 minutes).
* ``theme_config`` (Object, optional): Cosmetic widget appearance settings. See :ref:`Theme Config <webchat-struct-widget-theme-config>`. Omitted or ``null`` fields fall back to the platform default (blue bubble, no logo, bottom-right).
* ``direct_hash`` (String, optional): Hash used by the embed script (``data-hash`` attribute) to authenticate anonymous visitors via ``POST /auth/boot``. Returned on every response (``GET``, list, create, update, regenerate).
* ``tm_create`` (string, ISO 8601): Timestamp when the widget was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any widget property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the widget was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. _webchat-struct-widget-status:

Status
------
Defines the widget's operational state.

============= ================
Type          Description
============= ================
active        The widget is live; visitors can create sessions and send messages.
inactive       The widget is disabled; ``POST /webchat_sessions`` for this widget is rejected.
============= ================

.. _webchat-struct-widget-theme-config:

Theme Config
------------
Optional, customer-editable appearance settings for the floating chat bubble and panel.

.. code::

    {
        "primary_color": "<string>",
        "logo_url": "<string>",
        "position": "<string>"
    }

* ``primary_color`` (String, optional): Hex color code (e.g. ``#2563eb``) for the bubble and header. Defaults to the platform blue when omitted.
* ``logo_url`` (String, optional): HTTPS URL to a logo image shown in the panel header. No logo shown when omitted.
* ``position`` (enum string, optional): Where the floating bubble renders on the visitor's page. One of ``bottom_right`` (default) or ``bottom_left``.

.. note:: **AI Implementation Hint**

   Arbitrary CSS injection is deliberately not supported -- only the three fields above are configurable. This keeps the widget's attack surface bounded, since it renders on customer-controlled third-party pages.
