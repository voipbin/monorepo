.. _storage_overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Chargeable (per GB stored; default quota is 10 GB per customer)
   * **Async:** No. File uploads and downloads are synchronous. ``POST /storage_files`` returns immediately with the file metadata. ``GET /storage_files/{id}`` returns file details including a time-limited ``uri_download``.

VoIPBIN's Storage API provides file management capabilities for storing and retrieving files associated with your communications. From call recordings to media attachments, the Storage API enables secure file operations within your allocated quota.

With the Storage API you can:

- Upload files to cloud storage
- Retrieve and download stored files
- Delete files to manage quota
- Monitor storage usage
- Access file metadata and URLs


How Storage Works
-----------------
VoIPBIN storage provides cloud-based file management with quota limits.

**Storage Architecture**

::

    +-----------------------------------------------------------------------+
    |                         Storage System                                |
    +-----------------------------------------------------------------------+

    +-------------------+
    |  Storage Account  |
    |   (quota: 10GB)   |
    +--------+----------+
             |
             | contains
             v
    +--------+----------+--------+----------+--------+----------+
    |                   |                   |                   |
    v                   v                   v                   v
    +----------+   +----------+   +----------+   +----------+
    | Recording|   |   Media  |   |  Export  |   |  Other   |
    |  files   |   |   files  |   |  files   |   |  files   |
    +----------+   +----------+   +----------+   +----------+
         |              |              |              |
         v              v              v              v
    +---------+    +---------+    +---------+    +---------+
    | .wav    |    | .jpg    |    | .csv    |    | .pdf    |
    | .mp3    |    | .png    |    | .json   |    | .txt    |
    +---------+    +---------+    +---------+    +---------+

**Key Components**

- **Storage Account**: Your allocated storage space with quota
- **Files**: Individual stored items with metadata
- **Quota**: Maximum storage limit (currently 10GB)
- **Download URL**: Secure link to access file content


File Operations
---------------
Core file management operations.

.. note:: **AI Implementation Hint**

   The ``uri_download`` field in file responses contains a time-limited signed URL. Always check ``tm_download_expire`` before using the URL. If the URL has expired, fetch fresh file details via ``GET /storage_files/{id}`` to get a new download URL. Do not cache or persist download URLs for long-term use.

**Upload a File**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/storage_files?token=<token>' \
        --header 'Content-Type: multipart/form-data' \
        --form 'file=@/path/to/file.pdf' \
        --form 'name=document.pdf'

**Response:**

.. code::

    {
        "id": "file-uuid-123",
        "name": "document.pdf",
        "size": 102400,
        "mime_type": "application/pdf",
        "download_url": "https://storage.voipbin.net/...",
        "tm_create": "2024-01-15T10:30:00Z"
    }

**List All Files**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/storage_files?token=<token>'

**Get File Details**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/storage_files/<file-id>?token=<token>'

**Download a File (Direct)**

.. code::

    $ curl -L -X GET 'https://api.voipbin.net/v1.0/storage_files/<file-id>/file?token=<token>' \
        --output downloaded_file.pdf

.. note:: **AI Implementation Hint**

   ``GET https://api.voipbin.net/v1.0/storage_files/{id}/file`` returns an HTTP **307 Temporary Redirect** to a time-limited signed Google Cloud Storage URL. The ``Location`` header contains the actual download URL. Most HTTP clients (curl with ``-L``, browsers, ``requests`` with ``allow_redirects=True``) follow the redirect automatically. The ``{id}`` parameter is the file's UUID, obtained from the ``id`` field of ``GET https://api.voipbin.net/v1.0/storage_files``. This endpoint requires **CustomerAdmin** or **CustomerManager** permission. If the stored download URL has expired, the server automatically refreshes it before redirecting -- so unlike ``uri_download`` from ``GET /storage_files/{id}``, you never need to worry about expiration.

**Response:**

The server responds with HTTP 307 and a ``Location`` header:

.. code::

    HTTP/1.1 307 Temporary Redirect
    Location: https://storage.googleapis.com/bucket-name/storage/file-uuid?X-Goog-Signature=...&X-Goog-Expires=...

The client follows the redirect and receives the file content directly from the storage backend.

**Delete a File**

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/storage_files/<file-id>?token=<token>'


Service Agent File Download
---------------------------
Service agents have their own file storage scoped to the authenticated agent. Files uploaded via ``POST /service_agents/files`` (e.g., talk attachments) can be downloaded using the same redirect pattern as storage files.

**Download a Service Agent File**

.. code::

    $ curl -L -X GET 'https://api.voipbin.net/v1.0/service_agents/files/<file-id>/file?token=<token>' \
        --output downloaded_file.pdf

.. note:: **AI Implementation Hint**

   ``GET https://api.voipbin.net/v1.0/service_agents/files/{id}/file`` behaves identically to the storage file download: it returns an HTTP **307 Temporary Redirect** to a time-limited signed URL. The ``{id}`` parameter is the file's UUID, obtained from the ``id`` field of ``GET https://api.voipbin.net/v1.0/service_agents/files``. Unlike ``GET /storage_files/{id}/file`` which requires CustomerAdmin or CustomerManager permission, this endpoint requires only that the file belongs to the same customer as the authenticated agent. If the download URL has expired, the server refreshes it automatically before redirecting.

**Response:**

The server responds with HTTP 307 and a ``Location`` header:

.. code::

    HTTP/1.1 307 Temporary Redirect
    Location: https://storage.googleapis.com/bucket-name/storage/file-uuid?X-Goog-Signature=...&X-Goog-Expires=...


Storage Account Management
--------------------------
Monitor and manage your storage usage.

**Get Storage Usage**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/storage_accounts?token=<token>'

**Response:**

.. code::

    {
        "id": "storage-uuid-123",
        "customer_id": "customer-uuid-456",
        "total_size": 524288000,
        "file_count": 150,
        "quota": 10737418240,
        "tm_create": "2024-01-01T00:00:00Z"
    }

**Usage Breakdown**

::

    Storage Account:
    +-----------------------------------------------------------------------+
    |                                                                       |
    |  Quota:      10 GB (10,737,418,240 bytes)                            |
    |  Used:       500 MB (524,288,000 bytes)                              |
    |  Available:  9.5 GB                                                  |
    |  File Count: 150 files                                               |
    |                                                                       |
    |  Usage:      [#####------------------------------] 4.9%              |
    |                                                                       |
    +-----------------------------------------------------------------------+


File Types and Sources
----------------------
Storage holds various file types from different sources.

**Common File Types**

+-------------------+------------------+----------------------------------------+
| Category          | Extensions       | Source                                 |
+===================+==================+========================================+
| Call Recordings   | .wav, .mp3       | Automatic recording feature            |
+-------------------+------------------+----------------------------------------+
| Voicemail         | .wav, .mp3       | Voicemail recordings                   |
+-------------------+------------------+----------------------------------------+
| Media Attachments | .jpg, .png, .gif | MMS and email attachments              |
+-------------------+------------------+----------------------------------------+
| Transcripts       | .txt, .json      | Call transcription exports             |
+-------------------+------------------+----------------------------------------+
| Reports           | .csv, .pdf       | Generated reports and exports          |
+-------------------+------------------+----------------------------------------+

**File Sources**

::

    +-------------------+
    |  Call Recording   |---------------+
    +-------------------+               |
                                        |
    +-------------------+               v
    |  Transcription    |-------> +----------+
    +-------------------+         | Storage  |
                                  | Account  |
    +-------------------+         +----------+
    |  MMS Attachment   |-------->      |
    +-------------------+               |
                                        v
    +-------------------+         +----------+
    |  Manual Upload    |-------->| Files    |
    +-------------------+         +----------+


Quota Management
----------------
Manage storage within your allocated quota.

**Quota Check Flow**

::

    Upload Request
          |
          v
    +-------------------+
    | Check quota       |
    | Used + New file   |
    +--------+----------+
             |
             v
    +-------------------+     Within limit
    | Under quota?      |----------------------> Upload succeeds
    +--------+----------+
             |
             | Exceeds quota
             v
    +-------------------+
    | Upload rejected   |
    | Error: quota      |
    | exceeded          |
    +-------------------+

**Managing Quota**

- Delete unnecessary files to free space
- Export and archive old recordings
- Monitor usage regularly
- Request quota increase if needed


Common Scenarios
----------------

**Scenario 1: Managing Call Recordings**

Store and access call recordings.

::

    1. Call with recording enabled
       +--------------------------------------------+
       | Call completed                             |
       | Recording saved to storage                 |
       | File: call-2024-01-15-abc123.wav          |
       +--------------------------------------------+

    2. Access recording
       GET /storage_files?type=recording
       -> List of recording files

    3. Download recording
       GET /storage_files/{id}/file
       -> HTTP 307 redirect to signed download URL
       (or GET /storage_files/{id} for metadata + download_url)

    4. Clean up old recordings
       DELETE /storage_files/{id}
       -> Free storage space

**Scenario 2: Storage Cleanup**

Free up space when approaching quota.

::

    1. Check current usage
       GET /storage_accounts
       +--------------------------------------------+
       | Used: 9.2 GB / 10 GB (92%)                |
       | Status: Near quota limit                  |
       +--------------------------------------------+

    2. Identify large/old files
       GET /storage_files?sort=size&order=desc
       -> Find largest files

       GET /storage_files?sort=tm_create&order=asc
       -> Find oldest files

    3. Delete unnecessary files
       DELETE /storage_files/{id}
       (repeat for multiple files)

    4. Verify freed space
       GET /storage_accounts
       +--------------------------------------------+
       | Used: 5.8 GB / 10 GB (58%)                |
       | Status: OK                                |
       +--------------------------------------------+

**Scenario 3: Media File Management**

Handle media attachments from messages.

::

    Inbound MMS with image
         |
         v
    +--------------------------------------------+
    | Image saved to storage                     |
    | File: mms-attachment-xyz.jpg               |
    | Size: 2.5 MB                               |
    +--------------------------------------------+
         |
         v
    Access via API or webhook
    +--------------------------------------------+
    | download_url in message webhook            |
    | GET file for processing                    |
    +--------------------------------------------+


Best Practices
--------------

**1. Quota Management**

- Monitor usage regularly
- Set up alerts for high usage
- Clean up old files periodically
- Archive important files externally

**2. File Organization**

- Use descriptive file names
- Track file purpose via metadata
- Group related files logically

**3. Security**

- Download URLs are time-limited
- Don't share URLs publicly
- Delete sensitive files when no longer needed

**4. Performance**

- Delete files you no longer need
- Don't upload unnecessarily large files
- Use appropriate file formats


Troubleshooting
---------------

**Upload Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Upload fails              | Check file size; verify quota not exceeded;    |
|                           | ensure file format is supported                |
+---------------------------+------------------------------------------------+
| Quota exceeded error      | Delete old files; check storage usage;         |
|                           | request quota increase                         |
+---------------------------+------------------------------------------------+

**Download Issues**

+---------------------------+---------------------------------------------------+
| Symptom                   | Solution                                          |
+===========================+===================================================+
| Download URL expired      | Use ``GET /storage_files/{id}/file`` for          |
|                           | automatic refresh and redirect; or fetch fresh    |
|                           | details via ``GET /storage_files/{id}``           |
+---------------------------+---------------------------------------------------+
| 307 redirect not followed | Ensure your HTTP client follows redirects         |
|                           | (curl ``-L``, requests ``allow_redirects=True``)  |
+---------------------------+---------------------------------------------------+
| File not found            | Verify file ID; file may have been deleted        |
+---------------------------+---------------------------------------------------+

**Management Issues**

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| File count mismatch       | Allow time for sync; some operations are       |
|                           | eventually consistent                          |
+---------------------------+------------------------------------------------+
| Cannot delete file        | Check file ID; verify permissions; file may    |
|                           | be in use                                      |
+---------------------------+------------------------------------------------+


Related Documentation
---------------------

- :ref:`Recording Overview <recording-overview>` - Call recordings
- :ref:`Transcribe Overview <transcribe-overview>` - Transcription files
- :ref:`Message Overview <message-overview>` - Media attachments
- :ref:`Customer Overview <customer-overview>` - Account management

