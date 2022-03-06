.. _queue-overfiw: queue_overview

Overview
========

Call queueing allows calls to be placed on hold without handling the actual enquiries or transferring callers to the desired party.
While in the call queue, the caller is played pre-recorded music or messages. Call queues are often used in call centres when there are not enough staff to handle a large number of calls. Call centre operators generally receive information about the number of callers in the call queue and the duration of the waiting time. This allows them to respond flexibly to peak demand by deploying extra call centre staff.

The purpose of call queueing
----------------------------
Call queueing is intended to prevent callers from being turned away in the case of insufficient staff capacity. The purpose of the pre-recorded music or messages is to shorten the subjective waiting time. At the same time, call queues can be used for advertising products or services. As soon as the call can be dealt with, the caller is automatically transferred from the call queue to the member of staff responsible. If customer or contract data has to be requested in several stages, multiple downstream call queues can be used.

Waiting actions
---------------
In the VoIPBIN, the queued call will loop the queue's wating actions until find the available agents of the queue.

.. image:: _static/images/queue_overview_flow.png

Agent searching
---------------
If the call queued to the queue, then the call will loop the waiting action. Meanwhile, the queue is searching for available agents of the queue.
Every valid queue has the tags. When the queue is searching the agents, it picks the agent which the same tag with valid status.

.. image:: _static/images/queue_overview_agent.png
