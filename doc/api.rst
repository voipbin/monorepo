.. _api:

**********
API basics
**********

.. index:: Title, Purpose, Methods, Method, Required Permission, Call, Returns, Example

API description
===============

Title
-----
The main category of API. Normally, it represents general API URI.

Purpose
-------
The purpose of API.

Methods
-------
List of supported command with simple description.

Method: <command>
-----------------
Method description with command in detail.
It shown also added version.

Call
++++
Description of how to call this API. It will explain about method
parameters and data parameters.

::

  <method> <call URI>

  <required data>

Method parameters
* ``method``: API calling method. i.e. GET, PUT, POST, ...
* ``call URI``: URI. Uniform Resource Identifier

Data parameters
* ``required data``: Required data to call the API.

Returns
+++++++
Description of reply. Described in detail. These are common return
objects. This objects will not be explain again.

::

  {
    "status_code": <number>,
    "data_type": "<data type>",
    "data": <data>
  }

* ``status_code``: Status code of response. See detail https://developer.mozilla.org/en-US/docs/Web/HTTP/Status
* ``data_type``: Type of data.
* ``data`` Response data.

Example
+++++++
Simple example of how to use the API. It would be little bit different with real response.

Normally, to test the API curl is used. curl is a tool to transfer
data from or to a server, using one of the supported protocols. See
detail at link below.

::

  https://curl.haxx.se/docs/manpage.html

Some of APIs requires a returned uuid for the request. But
one must expect that these information are only valid within the user
sessions and are temporary.

***
API
***

/v1/calls/<call-id>
===================

Methods
-------
GET : Get specified call info.
POST: Create a call info with given call-id

Method: GET
-----------
Get specified call info.

Call
++++
::

    {
        "uri": "/v1/calls/<call-id>,
        "method": "GET",
        "data_type": "application/json",
        "data": ""
    }

* ``call-id``: call's id.

Returns
+++++++
::

   {
       $result,
       "data": {
           ...
       }
   }  

Example
+++++++
::

    # send a request
    {

    }

    # response
    {

    }

Method: POST
-----------
Create a call info with given call-id.
Making a call to given destination.

Call
++++
::

    {
        "uri": "/v1/calls/<call-id>,
        "method": "POST",
        "data_type": "application/json",
        "data": {
            "flow_id": "<flow-id>",
            "source": ...,
            "destination": ...
        }
    }

* ``call-id``: call's id.
* ``flow_id``: flow's id.
* ``source``: Source address. See detail :ref:call_address.
* ``destination``: Destination address. See detail :ref:call_address.

Returns
+++++++
::

   {
       $result,
       "data": {
           ...
       }
   }  

Example
+++++++
::

    # send a request
    {

    }

    # response
    {
        
    }
