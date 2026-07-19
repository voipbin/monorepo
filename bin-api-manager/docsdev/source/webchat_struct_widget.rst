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
        "session_flow_id": "<string>",
        "message_flow_id": "<string>",
        "session_idle_timeout": <number>,
        "theme_config": {
            "primary_color": "<string>",
            "secondary_color": "<string>",
            "header_background_color": "<string>",
            "header_text_color": "<string>",
            "logo_url": "<string>",
            "position": "<string>",
            "theme_mode": "<string>",
            "header_title": "<string>",
            "header_subtitle": "<string>",
            "connecting_indicator_enabled": <boolean>,
            "connecting_indicator_text": "<string>",
            "typing_indicator_enabled": <boolean>,
            "border_radius": "<string>",
            "font_size": "<string>"
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
* ``session_flow_id`` (UUID): The flow triggered once per Session, at Session creation. Obtained from the ``id`` field of ``GET /flows``.
* ``message_flow_id`` (UUID, optional): An independent flow triggered on every inbound message. Obtained from the ``id`` field of ``GET /flows``. Omit to leave inbound messages un-triggered.
* ``session_idle_timeout`` (Integer): Session idle timeout in seconds before an inactive session is automatically ended. Default ``1800`` (30 minutes).
* ``theme_config`` (Object, optional): Cosmetic widget appearance settings. See :ref:`Theme Config <webchat-struct-widget-theme-config>`. Omitted or ``null`` fields fall back to the platform default (blue bubble, no logo, bottom-right, light theme).
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
        "secondary_color": "<string>",
        "header_background_color": "<string>",
        "header_text_color": "<string>",
        "logo_url": "<string>",
        "position": "<string>",
        "theme_mode": "<string>",
        "header_title": "<string>",
        "header_subtitle": "<string>",
        "connecting_indicator_enabled": <boolean>,
        "connecting_indicator_text": "<string>",
        "typing_indicator_enabled": <boolean>,
        "border_radius": "<string>",
        "font_size": "<string>"
    }

* ``primary_color`` (String, optional): Hex color code (e.g. ``#2563eb``) for the bubble and, in light mode with no explicit ``header_background_color``, the header background. Defaults to the platform blue when omitted.
* ``secondary_color`` (String, optional): Hex color code for accent/text-contrast elements. Defaults to ``#fff`` in light mode, ``#1f2937`` in dark mode.
* ``header_background_color`` (String, optional): Hex color code for the widget header bar's background. Falls back to ``primary_color`` in light mode, or a fixed dark surface color (``#111827``) in dark mode, when omitted.
* ``header_text_color`` (String, optional): Hex color code for the widget header bar's text. Defaults to ``#fff`` in light mode, ``#f9fafb`` in dark mode.
* ``logo_url`` (String, optional): HTTPS URL to a logo image shown in the panel header. No logo shown when omitted.
* ``position`` (enum string, optional): Where the floating bubble renders on the visitor's page. One of ``bottom_right`` (default) or ``bottom_left``.
* ``theme_mode`` (enum string, optional): Light/dark/auto rendering of the widget panel. One of ``light`` (default), ``dark``, or ``auto`` (follows the visitor's OS ``prefers-color-scheme``, resolved once at widget load -- not live-reactive to a mid-session OS theme change). An explicit ``header_background_color``/``header_text_color``/``secondary_color`` always wins over the ``theme_mode``-resolved default.
* ``header_title`` (String, optional): Widget header text, max 100 characters. Defaults to ``"Chat with us"`` when omitted.
* ``header_subtitle`` (String, optional): Widget header subtext shown below ``header_title``, max 200 characters. No subtitle row rendered when omitted.
* ``connecting_indicator_enabled`` (Boolean, optional): Whether to show a system message in the panel while the visitor's session is being created. Omitted/``null`` falls back to enabled (``true``). To disable, the request must send an explicit ``false`` -- omitting the field never disables it.
* ``connecting_indicator_text`` (String, optional): Text shown while the visitor's session is being created, max 100 characters. Defaults to ``"Connecting…"`` when omitted.
* ``typing_indicator_enabled`` (Boolean, optional): Whether to show the three-dot "waiting for response" animation after the visitor sends a message. Omitted/``null`` falls back to enabled (``true``). No text-label variant is supported; this is a pure on/off toggle over the existing dot animation.
* ``border_radius`` (enum string, optional): Corner rounding applied to the bubble, panel, message bubbles, input field, and send button as a coordinated set. One of ``sharp``, ``rounded`` (default), or ``pill``.
* ``font_size`` (enum string, optional): Base font-size scale applied to the widget's header text and message text. One of ``compact``, ``default`` (default), or ``large``.

.. note:: **AI Implementation Hint**

   Arbitrary CSS injection is deliberately not supported -- only the fourteen fields above are configurable. This keeps the widget's attack surface bounded, since it renders on customer-controlled third-party pages. Hex color fields (``primary_color``, ``secondary_color``, ``header_background_color``, ``header_text_color``), the ``theme_mode`` enum, and the ``border_radius``/``font_size`` enums are validated server-side (regex/enum check) at the ``bin-api-manager`` handler boundary before persistence; requests with malformed values are rejected with ``INVALID_THEME_CONFIG``. ``connecting_indicator_enabled``/``typing_indicator_enabled`` are nullable booleans: omitting the field preserves the platform default (enabled) on every subsequent update, while an explicit ``false`` disables it -- clients must send ``false`` explicitly to turn an indicator off, not merely omit the field.
