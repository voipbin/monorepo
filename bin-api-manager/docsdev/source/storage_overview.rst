.. _storage_overview:

Overview
========

.. note:: **AI Context**

   * **Complexity:** Low
   * **Cost:** Chargeable (per GB stored; default quota is 10 GB per customer)
   * **Async:** No. File uploads and downloads are synchronous. ``POST /files`` returns immediately with the file metadata. ``GET /files/{id}`` returns file details including a time-limited ``uri_download``.

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

   The ``uri_download`` field in file responses contains a time-limited signed URL. Always check ``tm_download_expire`` before using the URL. If the URL has expired, fetch fresh file details via ``GET /files/{id}`` to get a new download URL. Do not cache or persist download URLs for long-term use.

**Upload a File**

.. code::

    $ curl -X POST 'https://api.voipbin.net/v1.0/files?token=<token>' \
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

    $ curl -X GET 'https://api.voipbin.net/v1.0/files?token=<token>'

**Get File Details**

.. code::

    $ curl -X GET 'https://api.voipbin.net/v1.0/files/<file-id>?token=<token>'

**Delete a File**

.. code::

    $ curl -X DELETE 'https://api.voipbin.net/v1.0/files/<file-id>?token=<token>'


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
       GET /files?type=recording
       -> List of recording files

    3. Download recording
       GET /files/{id}
       -> download_url in response

    4. Clean up old recordings
       DELETE /files/{id}
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
       GET /files?sort=size&order=desc
       -> Find largest files

       GET /files?sort=tm_create&order=asc
       -> Find oldest files

    3. Delete unnecessary files
       DELETE /files/{id}
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

+---------------------------+------------------------------------------------+
| Symptom                   | Solution                                       |
+===========================+================================================+
| Download URL expired      | Get fresh file details; URLs are time-limited  |
+---------------------------+------------------------------------------------+
| File not found            | Verify file ID; file may have been deleted     |
+---------------------------+------------------------------------------------+

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

