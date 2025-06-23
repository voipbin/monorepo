.. _accesskey-struct:

Struct
======

.. _accesskey-struct-accesskey:

Accesskey
---------

.. code-block:: json

   {
      "id": "5f1f8f7e-9b3d-4c60-8465-b69e9f28b6db",
      "customer_id": "a1d9b2cd-4578-4b23-91b6-5f5ec4a2f840",
      "name": "My API Key",
      "detail": "For accessing reporting APIs",
      "token": "DuRWq5T4DAK32dw4",
      "tm_expire": "2025-04-28 01:41:40.503790",
      "tm_create": "2022-04-28 01:41:40.503790",
      "tm_update": "2022-04-28 01:41:40.503790",
      "tm_delete": "9999-01-01 00:00:00.000000"
   }

- **id**:  
  A unique identifier (`UUID`) for the access key. This is auto-generated when the key is created.

- **customer_id**:  
  The `UUID` representing the customer to whom the access key belongs. This links the key to a specific customer account.

- **name**:  
  An optional string field that allows users to assign a human-readable name to the access key. This is useful for identification purposes in multi-key environments.

- **detail**:  
  An optional string field for additional information about the access key, such as its intended use or description.

- **token**:  
  The API token string. This is the credential used to authenticate API requests. It must be treated as sensitive information and stored securely.

- **tm_expire**:  
  A timestamp (`ISO 8601 format`) indicating when the access key will expire. After this time, the key will no longer be valid.

- **tm_create**:  
  A timestamp (`ISO 8601 format`) indicating when the access key was created.

- **tm_update**:  
  A timestamp (`ISO 8601 format`) showing the last time the access key was updated. This is useful for tracking modifications to the key.

- **tm_delete**:  
  A timestamp (`ISO 8601 format`) indicating when the key was marked for deletion. If the value is set to a far-future date (e.g., `"9999-01-01 00:00:00.000000"`), it implies the key is not deleted.

