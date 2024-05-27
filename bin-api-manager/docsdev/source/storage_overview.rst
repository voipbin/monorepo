.. _storage_overview:

Overview
========
The Storage API on VoIPBin is designed to help users manage their stored files and keep track of their storage usage. The API provides endpoints to handle file operations such as uploading, retrieving, and deleting files, as well as monitoring the overall storage usage. This allows users to efficiently manage their data within their allocated storage quota. The key components of the Storage API include:

File
----
The Files API facilitates file management on the platform. It includes the following features.

* Retrieve All Files: Fetch a list of all files stored by the authenticated user. This helps users view and manage their stored files easily.

* Upload a File: Allows users to upload new files to the platform. This endpoint is essential for adding new data to the storage.

* Retrieve a Single File: Get detailed information about a specific file using its unique identifier. This is useful for accessing file metadata and content.

* Delete a File: Enables users to delete a specific file from the storage by its unique identifier. This helps in managing storage space by removing unnecessary files.

Endpoint: DELETE /files/:id

Account
-------
The Storage Account API provides insights into the user's storage usage. It includes the following endpoint:

Retrieve Storage Account Details: Fetches the current storage usage details, including the total amount of stored file size, the count of files, and the storage quota. This endpoint helps users monitor and manage their storage utilization effectively.

Endpoint: GET /storage_account

Limitation
+++++++++++
Each user has a storage quota, currently set at 1GB, to ensure fair usage and manage resource allocation efficiently. 
By using the Storage API, users can seamlessly keep track of their stored files and make informed decisions about their storage needs.
