.. _storage_overview:

Overview
========
The Storage API on VoIPBin is designed to help users manage their stored files and monitor their storage usage. This feature allows users to efficiently oversee their data within their allocated storage quota.

The API provides endpoints for various file operations, such as uploading, retrieving, and deleting files, as well as monitoring overall storage usage. These functionalities enable users to manage their data effectively.

File Management
---------------
The Files API enables comprehensive file management on the platform. It includes the following features:

* **Retrieve All Files:** Fetch a list of all files stored by the authenticated user. This helps users view and organize their stored files efficiently.

* **Upload a File:** Allows users to upload new files to the platform. This endpoint is essential for adding new data to the storage.

* **Retrieve a Single File:** Get detailed information about a specific file using its unique identifier. This is useful for accessing file metadata and content, including the download URL.

* **Download a File:** Download a file using the download URL.

* **Delete a File:** Enables users to delete a specific file from the storage by its unique identifier. This helps manage storage space by removing unnecessary files.

Storage Account Management
--------------------------
The Storage Account API provides insights into the user's storage usage. Users can retrieve current storage usage details, including:

* Total amount of stored file size.
* Count of stored files.
* Storage quota.

Quota
----------
Each user has a storage quota, currently set at 10GB, to ensure fair usage and efficient resource allocation. The Storage API helps users keep track of their stored files and make informed decisions about their storage needs.

By utilizing the Storage API, users can seamlessly manage their data and ensure optimal use of their allocated storage space.
