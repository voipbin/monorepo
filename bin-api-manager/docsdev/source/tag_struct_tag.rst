.. _tag-struct-tag:

Tag
======

.. _tag-struct-tag-tag:

Tag
---

.. code::

    {
        "id": "<string>",
        "name": "<string>",
        "detail": "<string>",
        "tm_create": "<string>",
        "tm_update": "<string>",
        "tm_delete": "<string>"
    }

* ``id`` (UUID): The tag's unique identifier. Returned when creating via ``POST /tags`` or listing via ``GET /tags``.
* ``name`` (String): Human-readable name for the tag. Must be unique per customer account. Used for matching agents to queues.
* ``detail`` (String): Detailed description of what this tag represents.
* ``tm_create`` (string, ISO 8601): Timestamp when the tag was created.
* ``tm_update`` (string, ISO 8601): Timestamp of the last update to any tag property.
* ``tm_delete`` (string, ISO 8601): Timestamp when the tag was deleted. Set to ``9999-01-01 00:00:00.000000`` if not deleted.

.. note:: **AI Implementation Hint**

   Tag names must be unique within a customer account. When assigning tags to agents via ``PUT /agents/{id}/tag_ids``, use the tag's ``id`` (UUID), not the tag name. A ``tm_delete`` value of ``9999-01-01 00:00:00.000000`` is a sentinel meaning the resource has **not** been deleted.

