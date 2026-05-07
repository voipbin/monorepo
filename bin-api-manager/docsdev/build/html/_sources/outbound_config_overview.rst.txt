.. _outbound_config_overview:

OutboundConfig Overview
=======================

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Free (no billing impact)
   * **Async:** No

OutboundConfig controls per-customer outbound PSTN call behaviour. It has two fields:

- ``destination_whitelist`` — ISO 3166 alpha-2 country codes permitted for outbound PSTN calls. **Always enforced** — an empty list denies all PSTN calls. Customers must populate this before making any outbound PSTN call.
- ``codecs`` — Comma-separated codec preference list (e.g. ``PCMU,PCMA,G729``). Empty string = server default.

There is exactly one OutboundConfig per customer. It is created explicitly via ``POST /v1/outbound_configs`` and updated via ``PUT /v1/outbound_configs/{id}``.

.. warning::

   **Deploy-day behaviour:** Any customer without an OutboundConfig row will have all outbound PSTN calls rejected immediately. Customers must create their OutboundConfig and populate ``destination_whitelist`` before placing PSTN calls.

.. note:: **AI Implementation Hint**

   Before placing an outbound PSTN call on behalf of a customer, verify the destination country is in ``destination_whitelist``. If not, add it via ``PUT https://api.voipbin.net/v1.0/outbound_configs/{id}`` before attempting the call. A missing or empty whitelist causes a ``400 Bad Request`` with the message ``"outbound destination country not whitelisted"``.

Troubleshooting
---------------

* **400 Bad Request** — ``"outbound destination country not whitelisted"``:
    * **Cause:** Destination country not in ``destination_whitelist``, or whitelist is empty.
    * **Fix:** Add the country code via ``PUT /v1/outbound_configs/{id}`` with ``{"destination_whitelist": ["us", "gb"]}``.

* **409 Conflict** on ``POST /v1/outbound_configs``:
    * **Cause:** An OutboundConfig already exists for this customer.
    * **Fix:** Use ``GET /v1/outbound_configs?customer_id={id}`` to retrieve it, then update with ``PUT``.
