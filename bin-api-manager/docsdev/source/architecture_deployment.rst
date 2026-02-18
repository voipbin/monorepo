.. _architecture-deployment:

Deployment Architecture
=======================

.. note:: **AI Context**

   This page describes VoIPBIN's production deployment on Google Cloud Platform: GKE cluster configuration, Kubernetes deployment patterns, HPA auto-scaling, VoIP VM infrastructure (Kamailio, Asterisk, RTPEngine), database and cache infrastructure, network architecture (VPC, firewall rules), CI/CD pipeline (CircleCI), and disaster recovery. Relevant when an AI agent needs to understand infrastructure sizing, scaling limits, deployment strategies, or network topology.

VoIPBIN runs on Google Cloud Platform (GCP) using Google Kubernetes Engine (GKE) for container orchestration. This section details the deployment topology, scaling strategies, and infrastructure components.

Infrastructure Overview
-----------------------

.. code::

    VoIPBIN Production Infrastructure:

    +------------------------------------------------------------------+
    |                    Google Cloud Platform                          |
    +------------------------------------------------------------------+
    |                                                                    |
    |  +------------------------+    +------------------------+         |
    |  |   GKE Cluster          |    |   Cloud SQL (MySQL)   |         |
    |  |   (Kubernetes)         |    |   - Primary           |         |
    |  |                        |    |   - Read Replicas (3) |         |
    |  |  30+ Microservices     |    +------------------------+         |
    |  |  2 replicas each       |                                       |
    |  +------------------------+    +------------------------+         |
    |                                |   Memorystore (Redis) |         |
    |  +------------------------+    |   - 16 GB Instance    |         |
    |  |   Compute Engine VMs   |    +------------------------+         |
    |  |   - Kamailio (3)       |                                       |
    |  |   - Asterisk (6+)      |    +------------------------+         |
    |  |   - RTPEngine (3)      |    |   Cloud Storage       |         |
    |  +------------------------+    |   - Recordings        |         |
    |                                |   - Media files       |         |
    |  +------------------------+    +------------------------+         |
    |  |   RabbitMQ Cluster     |                                       |
    |  |   (3-node)             |    +------------------------+         |
    |  +------------------------+    |   Cloud Load Balancer |         |
    |                                +------------------------+         |
    |                                                                    |
    +------------------------------------------------------------------+

Kubernetes Architecture
-----------------------

All Go microservices run in Kubernetes:

.. code::

    GKE Cluster Configuration:

    +----------------------------------------------------------------+
    |                     GKE Cluster                                 |
    +----------------------------------------------------------------+
    |                                                                  |
    |  Namespace: production                                           |
    |  +-----------------------------------------------------------+  |
    |  |                                                           |  |
    |  |  Deployment: bin-api-manager (2 replicas)                 |  |
    |  |  +----------------+  +----------------+                   |  |
    |  |  | Pod 1          |  | Pod 2          |                   |  |
    |  |  | - api-manager  |  | - api-manager  |                   |  |
    |  |  | - Port: 443    |  | - Port: 443    |                   |  |
    |  |  | - Port: 9000   |  | - Port: 9000   |                   |  |
    |  |  | - Port: 2112   |  | - Port: 2112   |                   |  |
    |  |  +----------------+  +----------------+                   |  |
    |  |                                                           |  |
    |  |  Deployment: bin-call-manager (2 replicas)                |  |
    |  |  +----------------+  +----------------+                   |  |
    |  |  | Pod 1          |  | Pod 2          |                   |  |
    |  |  +----------------+  +----------------+                   |  |
    |  |                                                           |  |
    |  |  ... (28 more deployments, each with 2 replicas)          |  |
    |  |                                                           |  |
    |  +-----------------------------------------------------------+  |
    |                                                                  |
    +----------------------------------------------------------------+

**Standard Deployment Pattern:**

All services follow the same deployment pattern:

.. code::

    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: bin-call-manager
    spec:
      replicas: 2
      selector:
        matchLabels:
          app: bin-call-manager
      template:
        spec:
          containers:
          - name: bin-call-manager
            image: gcr.io/voipbin/bin-call-manager:latest
            ports:
            - containerPort: 8080      # Health check
            - containerPort: 2112      # Prometheus metrics
            resources:
              requests:
                memory: "256Mi"
                cpu: "100m"
              limits:
                memory: "512Mi"
                cpu: "500m"
            livenessProbe:
              httpGet:
                path: /health
                port: 8080
              initialDelaySeconds: 30
              periodSeconds: 10
            readinessProbe:
              httpGet:
                path: /ready
                port: 8080
              initialDelaySeconds: 5
              periodSeconds: 5
            env:
            - name: DSN
              valueFrom:
                secretKeyRef:
                  name: db-credentials
                  key: dsn
            - name: RABBIT_ADDR
              valueFrom:
                configMapKeyRef:
                  name: app-config
                  key: rabbit_addr

**Pod Ports:**

.. code::

    Service Port Configuration:

    bin-api-manager:
    +------------------------------------------+
    | Port 443   - HTTPS REST API (external)   |
    | Port 9000  - Audiosocket (media stream)  |
    | Port 2112  - Prometheus metrics          |
    +------------------------------------------+

    Other Services:
    +------------------------------------------+
    | Port 8080  - Health/Ready endpoints      |
    | Port 2112  - Prometheus metrics          |
    +------------------------------------------+

Service Scaling
---------------

VoIPBIN scales services based on demand:

**Horizontal Pod Autoscaler (HPA):**

.. code::

    HPA Configuration:

    apiVersion: autoscaling/v2
    kind: HorizontalPodAutoscaler
    metadata:
      name: bin-call-manager-hpa
    spec:
      scaleTargetRef:
        apiVersion: apps/v1
        kind: Deployment
        name: bin-call-manager
      minReplicas: 2
      maxReplicas: 10
      metrics:
      - type: Resource
        resource:
          name: cpu
          target:
            type: Utilization
            averageUtilization: 70
      - type: Resource
        resource:
          name: memory
          target:
            type: Utilization
            averageUtilization: 80

**Scaling Triggers:**

.. code::

    Auto-Scaling Strategy:

    +---------------------------------------------+
    |  Metric          | Threshold | Action       |
    +---------------------------------------------+
    |  CPU > 70%       | Scale up  | +1 replica   |
    |  CPU < 30%       | Scale down| -1 replica   |
    |  Memory > 80%    | Scale up  | +1 replica   |
    |  Queue depth >100| Scale up  | +2 replicas  |
    +---------------------------------------------+

    Scaling Limits:
    +---------------------------------------------+
    |  Service             | Min | Max  | Notes   |
    +---------------------------------------------+
    |  bin-api-manager     | 2   | 20   | Gateway |
    |  bin-call-manager    | 2   | 10   | Core    |
    |  bin-flow-manager    | 2   | 10   | Core    |
    |  bin-ai-manager      | 2   | 5    | GPU     |
    |  bin-pipecat-manager | 2   | 8    | AI      |
    |  Other services      | 2   | 5    | Standard|
    +---------------------------------------------+

VoIP Infrastructure
-------------------

VoIP components run on dedicated VMs for performance:

.. code::

    VoIP Component Topology:

    +------------------------------------------------------------------+
    |                      External Traffic                            |
    +------------------------------------------------------------------+
                                |
                                | SIP (UDP/TCP 5060-5061)
                                v
    +------------------------------------------------------------------+
    |                   Cloud Load Balancer                            |
    |                   (L4 - TCP/UDP)                                  |
    +------------------------------------------------------------------+
                |               |               |
                v               v               v
    +----------------+  +----------------+  +----------------+
    |  Kamailio-1    |  |  Kamailio-2    |  |  Kamailio-3    |
    |  (SIP Proxy)   |  |  (SIP Proxy)   |  |  (SIP Proxy)   |
    |                |  |                |  |                |
    |  n1-standard-4 |  |  n1-standard-4 |  |  n1-standard-4 |
    |  4 vCPU, 15GB  |  |  4 vCPU, 15GB  |  |  4 vCPU, 15GB  |
    +-------+--------+  +-------+--------+  +-------+--------+
            |                   |                   |
            +-------------------+-------------------+
                                |
                                v
    +------------------------------------------------------------------+
    |                   Internal Load Balancer                          |
    +------------------------------------------------------------------+
                |               |               |
                v               v               v
    +----------------+  +----------------+  +----------------+
    |  Asterisk-1    |  |  Asterisk-2    |  |  Asterisk-3    |
    |  (Call Farm)   |  |  (Call Farm)   |  |  (Conf Farm)   |
    |                |  |                |  |                |
    |  n1-standard-8 |  |  n1-standard-8 |  |  n1-standard-8 |
    |  8 vCPU, 30GB  |  |  8 vCPU, 30GB  |  |  8 vCPU, 30GB  |
    +-------+--------+  +-------+--------+  +-------+--------+
            |                   |                   |
            +-------------------+-------------------+
                                |
                                v
    +------------------------------------------------------------------+
    |                     RTPEngine Farm                                |
    +------------------------------------------------------------------+
    |  +----------------+  +----------------+  +----------------+       |
    |  |  RTPEngine-1   |  |  RTPEngine-2   |  |  RTPEngine-3   |       |
    |  |  n1-highcpu-8  |  |  n1-highcpu-8  |  |  n1-highcpu-8  |       |
    |  +----------------+  +----------------+  +----------------+       |
    +------------------------------------------------------------------+

**VM Specifications:**

.. code::

    VoIP VM Sizing:

    Kamailio Nodes (SIP Proxy):
    +-------------------------------------------+
    | Machine Type:  n1-standard-4              |
    | vCPUs:         4                          |
    | Memory:        15 GB                      |
    | Disk:          100 GB SSD                 |
    | Network:       10 Gbps                    |
    | Capacity:      ~5,000 concurrent calls    |
    +-------------------------------------------+

    Asterisk Nodes (Media Server):
    +-------------------------------------------+
    | Machine Type:  n1-standard-8              |
    | vCPUs:         8                          |
    | Memory:        30 GB                      |
    | Disk:          200 GB SSD                 |
    | Network:       10 Gbps                    |
    | Capacity:      ~500 concurrent calls each |
    +-------------------------------------------+

    RTPEngine Nodes (Media Proxy):
    +-------------------------------------------+
    | Machine Type:  n1-highcpu-8               |
    | vCPUs:         8                          |
    | Memory:        7.2 GB                     |
    | Disk:          50 GB SSD                  |
    | Network:       10 Gbps                    |
    | Capacity:      ~2,000 media streams each  |
    +-------------------------------------------+

Database Infrastructure
-----------------------

Cloud SQL for MySQL provides managed database:

.. code::

    Cloud SQL Configuration:

    Primary Instance:
    +-------------------------------------------+
    | Instance:      db-custom-8-32768          |
    | vCPUs:         8                          |
    | Memory:        32 GB                      |
    | Storage:       1 TB SSD                   |
    | High Avail:    Regional (failover)        |
    | Backups:       Daily automatic            |
    +-------------------------------------------+

    Read Replicas (3):
    +-------------------------------------------+
    | Instance:      db-custom-4-16384          |
    | vCPUs:         4                          |
    | Memory:        16 GB                      |
    | Storage:       1 TB SSD                   |
    | Region:        Same as primary            |
    +-------------------------------------------+

    Replication Architecture:
    +-------------------------------------------+
    |                                           |
    |  +-----------+                            |
    |  |  Primary  |<-- All Writes              |
    |  +-----------+                            |
    |       |                                   |
    |       | Async Replication                 |
    |       |                                   |
    |  +----+----+----+                         |
    |  |    |    |    |                         |
    |  v    v    v    v                         |
    |  R1   R2   R3   (Backups)                 |
    |  ^    ^    ^                              |
    |  |    |    |                              |
    |  +----+----+                              |
    |       |                                   |
    |       +--- Read Traffic                   |
    |                                           |
    +-------------------------------------------+

Cache Infrastructure
--------------------

Memorystore for Redis provides caching:

.. code::

    Memorystore Configuration:

    +-------------------------------------------+
    | Tier:          Standard                   |
    | Capacity:      16 GB                      |
    | Version:       Redis 6.x                  |
    | High Avail:    Yes (failover replica)     |
    | Max Conn:      65,000                     |
    | Network:       Private VPC                |
    +-------------------------------------------+

    Cache Distribution:
    +-------------------------------------------+
    | Data Type         | Approx Size | TTL     |
    +-------------------------------------------+
    | Session tokens    | 2 GB        | 1 hour  |
    | Call records      | 4 GB        | 24 hours|
    | Agent status      | 1 GB        | 5 min   |
    | Configuration     | 500 MB      | 1 hour  |
    | Queue stats       | 500 MB      | 1 min   |
    | Flow definitions  | 2 GB        | 1 hour  |
    | Other             | 6 GB        | varies  |
    +-------------------------------------------+

Message Queue Infrastructure
----------------------------

RabbitMQ cluster for messaging:

.. code::

    RabbitMQ Cluster:

    +-------------------------------------------+
    |  Node 1 (Primary)                         |
    |  +----------------------------------+     |
    |  |  Queues: 50% of messages         |     |
    |  |  CPU: 4 vCPU                     |     |
    |  |  Memory: 16 GB                   |     |
    |  |  Disk: 100 GB SSD                |     |
    |  +----------------------------------+     |
    |                                           |
    |  Node 2 (Mirror)                          |
    |  +----------------------------------+     |
    |  |  Queues: Mirrored from Node 1    |     |
    |  |  CPU: 4 vCPU                     |     |
    |  |  Memory: 16 GB                   |     |
    |  +----------------------------------+     |
    |                                           |
    |  Node 3 (Mirror)                          |
    |  +----------------------------------+     |
    |  |  Queues: Mirrored from Node 1    |     |
    |  |  CPU: 4 vCPU                     |     |
    |  |  Memory: 16 GB                   |     |
    |  +----------------------------------+     |
    +-------------------------------------------+

    Queue Mirroring Policy:
    +-------------------------------------------+
    | Pattern: bin-manager.*                    |
    | ha-mode: all                              |
    | ha-sync-mode: automatic                   |
    +-------------------------------------------+

Network Architecture
--------------------

VPC network isolates components:

.. code::

    VPC Network Design:

    +------------------------------------------------------------------+
    |                         VPC: voipbin-prod                        |
    +------------------------------------------------------------------+
    |                                                                   |
    |  Subnet: public (10.0.0.0/24)                                     |
    |  +-------------------------------------------------------------+ |
    |  |  Cloud Load Balancer                                        | |
    |  |  NAT Gateway                                                 | |
    |  +-------------------------------------------------------------+ |
    |                                                                   |
    |  Subnet: kubernetes (10.0.1.0/24)                                 |
    |  +-------------------------------------------------------------+ |
    |  |  GKE Cluster (all pods)                                     | |
    |  |  Internal Load Balancers                                    | |
    |  +-------------------------------------------------------------+ |
    |                                                                   |
    |  Subnet: voip (10.0.2.0/24)                                       |
    |  +-------------------------------------------------------------+ |
    |  |  Kamailio VMs                                               | |
    |  |  Asterisk VMs                                               | |
    |  |  RTPEngine VMs                                              | |
    |  +-------------------------------------------------------------+ |
    |                                                                   |
    |  Subnet: data (10.0.3.0/24)                                       |
    |  +-------------------------------------------------------------+ |
    |  |  Cloud SQL (private IP)                                     | |
    |  |  Memorystore (private IP)                                   | |
    |  |  RabbitMQ Cluster                                           | |
    |  +-------------------------------------------------------------+ |
    |                                                                   |
    +------------------------------------------------------------------+

**Firewall Rules:**

.. code::

    Firewall Configuration:

    Ingress (External):
    +-------------------------------------------+
    | Rule              | Ports    | Source     |
    +-------------------------------------------+
    | allow-https       | 443      | 0.0.0.0/0  |
    | allow-sip         | 5060-5061| 0.0.0.0/0  |
    | allow-rtp         | 10000-60000| 0.0.0.0/0|
    +-------------------------------------------+

    Internal:
    +-------------------------------------------+
    | Rule              | Ports    | Source     |
    +-------------------------------------------+
    | allow-k8s-internal| all      | 10.0.1.0/24|
    | allow-voip-internal| all     | 10.0.2.0/24|
    | allow-db-access   | 3306,6379| 10.0.1.0/24|
    | allow-rabbit      | 5672     | 10.0.1.0/24|
    +-------------------------------------------+

Load Balancing
--------------

Multiple load balancers route traffic:

.. code::

    Load Balancer Architecture:

    External (L7 - HTTPS):
    +-------------------------------------------+
    |  api.voipbin.net                          |
    |  +----------------------------------+     |
    |  |  Cloud Load Balancer (HTTPS)    |     |
    |  |  - SSL termination              |     |
    |  |  - Path routing                 |     |
    |  |  - Health checks                |     |
    |  +----------------------------------+     |
    |           |                              |
    |           v                              |
    |  +----------------------------------+     |
    |  |  GKE Ingress -> api-manager     |     |
    |  +----------------------------------+     |
    +-------------------------------------------+

    External (L4 - SIP):
    +-------------------------------------------+
    |  sip.voipbin.net                          |
    |  +----------------------------------+     |
    |  |  Network Load Balancer (TCP/UDP)|     |
    |  |  - Port 5060 (UDP/TCP)          |     |
    |  |  - Port 5061 (TLS)              |     |
    |  +----------------------------------+     |
    |           |                              |
    |           v                              |
    |  +----------------------------------+     |
    |  |  Kamailio Farm                  |     |
    |  +----------------------------------+     |
    +-------------------------------------------+

    Internal (Services):
    +-------------------------------------------+
    |  Kubernetes Service (ClusterIP)          |
    |  - bin-call-manager:8080                 |
    |  - bin-flow-manager:8080                 |
    |  - ...                                   |
    +-------------------------------------------+

Monitoring Stack
----------------

Prometheus and Grafana for observability:

.. code::

    Monitoring Architecture:

    +-------------------------------------------+
    |             Grafana Dashboard             |
    |  +----------------------------------+     |
    |  |  Service Health                 |     |
    |  |  Call Metrics                   |     |
    |  |  Queue Depths                   |     |
    |  |  Error Rates                    |     |
    |  +----------------------------------+     |
    +-------------------------------------------+
                        ^
                        |
    +-------------------------------------------+
    |               Prometheus                  |
    |  +----------------------------------+     |
    |  |  Scrape interval: 15s           |     |
    |  |  Retention: 30 days             |     |
    |  |  Storage: 100 GB                |     |
    |  +----------------------------------+     |
    +-------------------------------------------+
                        ^
                        |
    +-------+-------+-------+-------+-------+
    |       |       |       |       |       |
    v       v       v       v       v       v
    api     call    flow    ai      voip   db
    :2112   :2112   :2112   :2112   :9100  :9104

**Key Metrics Collected:**

.. code::

    Prometheus Metrics:

    Service Metrics (port 2112):
    +-------------------------------------------+
    | voipbin_http_requests_total               |
    | voipbin_http_request_duration_seconds     |
    | voipbin_rpc_requests_total                |
    | voipbin_rpc_request_duration_seconds      |
    | voipbin_active_calls_gauge                |
    | voipbin_queue_depth_gauge                 |
    +-------------------------------------------+

    Infrastructure Metrics:
    +-------------------------------------------+
    | container_cpu_usage_seconds_total         |
    | container_memory_usage_bytes              |
    | mysql_global_status_threads_connected     |
    | redis_connected_clients                   |
    | rabbitmq_queue_messages                   |
    +-------------------------------------------+

Deployment Pipeline
-------------------

CI/CD with CircleCI:

.. code::

    Deployment Pipeline:

    Developer        GitHub         CircleCI         GKE
        |               |              |              |
        | Push          |              |              |
        +-------------->|              |              |
        |               | Webhook      |              |
        |               +------------->|              |
        |               |              |              |
        |               |              | 1. Checkout  |
        |               |              | 2. Test      |
        |               |              | 3. Lint      |
        |               |              | 4. Build     |
        |               |              | 5. Push Image|
        |               |              |              |
        |               |              | (if main)    |
        |               |              | 6. Deploy    |
        |               |              +------------->|
        |               |              |              |
        |               |              |              | Rolling
        |               |              |              | Update
        |               |              |              |
        |               |              |<-------------+
        |               |              | Deploy Done  |
        |               |              |              |

**Rolling Update Strategy:**

.. code::

    Deployment Strategy:

    spec:
      strategy:
        type: RollingUpdate
        rollingUpdate:
          maxSurge: 1        # Create 1 new pod at a time
          maxUnavailable: 0  # Never have 0 ready pods

    Update Flow:
    +-------------------------------------------+
    | 1. New pod created with new version       |
    | 2. Wait for readiness probe success       |
    | 3. Remove old pod from service            |
    | 4. Terminate old pod                      |
    | 5. Repeat for all replicas                |
    +-------------------------------------------+

    Zero-Downtime:
    - At least 1 pod always ready
    - No traffic to terminating pods
    - Graceful shutdown (SIGTERM)

Disaster Recovery
-----------------

Multi-region resilience:

.. code::

    DR Strategy:

    Primary Region: us-central1
    +-------------------------------------------+
    |  GKE Cluster (Active)                     |
    |  Cloud SQL Primary                        |
    |  Memorystore                              |
    |  VoIP VMs                                 |
    +-------------------------------------------+

    DR Region: us-east1 (Standby)
    +-------------------------------------------+
    |  GKE Cluster (Warm Standby)               |
    |  Cloud SQL Replica                        |
    |  Memorystore (Separate)                   |
    |  VoIP VMs (Ready to scale)                |
    +-------------------------------------------+

    Recovery Objectives:
    +-------------------------------------------+
    | RTO (Recovery Time):    < 30 minutes      |
    | RPO (Data Loss):        < 5 minutes       |
    +-------------------------------------------+

**Failover Procedure:**

.. code::

    DR Failover Steps:

    1. Detect Failure
       +----------------------------------+
       | - Monitor alerts trigger         |
       | - Confirm region outage          |
       +----------------------------------+

    2. Database Failover
       +----------------------------------+
       | - Promote DR replica to primary  |
       | - Update connection strings      |
       +----------------------------------+

    3. Traffic Redirect
       +----------------------------------+
       | - Update DNS (Cloud DNS)         |
       | - Route traffic to DR region     |
       +----------------------------------+

    4. Scale DR Resources
       +----------------------------------+
       | - Scale GKE deployments          |
       | - Start additional VoIP VMs      |
       +----------------------------------+

    5. Verify Services
       +----------------------------------+
       | - Health checks pass             |
       | - Test critical paths            |
       +----------------------------------+

Cost Optimization
-----------------

Resource efficiency strategies:

.. code::

    Cost Optimization:

    Committed Use Discounts:
    +-------------------------------------------+
    | GKE nodes:      3-year commitment (57%)   |
    | Cloud SQL:      3-year commitment (57%)   |
    | Memorystore:    1-year commitment (25%)   |
    +-------------------------------------------+

    Preemptible VMs (non-critical):
    +-------------------------------------------+
    | CI/CD runners:  Preemptible (80% savings) |
    | Batch jobs:     Preemptible               |
    +-------------------------------------------+

    Right-sizing:
    +-------------------------------------------+
    | Monthly review of resource utilization    |
    | Downsize underutilized instances          |
    | Upsize bottlenecked services              |
    +-------------------------------------------+

    Auto-scaling:
    +-------------------------------------------+
    | Scale down during off-peak hours          |
    | Scale up only when needed                 |
    | Set appropriate min/max replicas          |
    +-------------------------------------------+

