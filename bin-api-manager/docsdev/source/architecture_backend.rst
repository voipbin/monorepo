.. _architecture-backend:

Backend Microservices
======================

VoIPBIN's backend consists of 30+ specialized Go microservices organized into functional domains. Each service owns its specific business logic and communicates with others through a message queue, enabling independent scaling, deployment, and development.

Microservices Organization
---------------------------

Services are organized by functional domain:

.. code::

    VoIPBIN Microservices Architecture

    ┌─────────────────────────────────────────────────────────────┐
    │                   Communication Services                    │
    ├─────────────────────────────────────────────────────────────┤
    │  bin-call-manager        │  Call lifecycle and routing      │
    │  bin-conference-manager  │  Conference bridge management    │
    │  bin-sms-manager         │  SMS messaging                   │
    │  bin-chat-manager        │  Real-time chat                  │
    │  bin-email-manager       │  Email campaigns                 │
    └─────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────┐
    │                      AI Services                            │
    ├─────────────────────────────────────────────────────────────┤
    │  bin-ai-manager          │  AI assistants and processing    │
    │  bin-transcribe-manager  │  Speech-to-text transcription    │
    │  bin-tts-manager         │  Text-to-speech synthesis        │
    │  bin-summary-manager     │  Call summarization              │
    └─────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────┐
    │                    Workflow Services                        │
    ├─────────────────────────────────────────────────────────────┤
    │  bin-flow-manager        │  Call flow and IVR orchestration │
    │  bin-queue-manager       │  Call queue management           │
    │  bin-campaign-manager    │  Outbound campaign automation    │
    │  bin-conversation-mgr    │  Conversation tracking           │
    └─────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────┐
    │                   Management Services                       │
    ├─────────────────────────────────────────────────────────────┤
    │  bin-agent-manager       │  Agent state and presence        │
    │  bin-billing-manager     │  Usage tracking and billing      │
    │  bin-customer-manager    │  Customer account management     │
    │  bin-webhook-manager     │  Webhook delivery                │
    │  bin-storage-manager     │  File and media storage          │
    │  bin-number-manager      │  Phone number management         │
    │  bin-accesskey-manager   │  API key management              │
    └─────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────┐
    │                  Integration Services                       │
    ├─────────────────────────────────────────────────────────────┤
    │  bin-talk-manager        │  Agent UI backend                │
    │  bin-config-manager      │  Configuration management        │
    │  bin-sentinel-manager    │  Health monitoring               │
    │  bin-record-manager      │  Call recording                  │
    │  bin-rtc-manager         │  Real-time communication control │
    └─────────────────────────────────────────────────────────────┘

Service Characteristics
-----------------------

Each microservice follows these design principles:

**Domain Isolation**

.. code::

    Service Boundary:

    ┌────────────────────────────────────────┐
    │         bin-call-manager               │
    │                                        │
    │  ┌──────────────────────────────────┐ │
    │  │   Domain Logic (Call Handling)   │ │
    │  └──────────────────────────────────┘ │
    │                                        │
    │  ┌──────────────────────────────────┐ │
    │  │   Data Access (Call Records)     │ │
    │  └──────────────────────────────────┘ │
    │                                        │
    │  ┌──────────────────────────────────┐ │
    │  │   RPC Handlers (Message Queue)   │ │
    │  └──────────────────────────────────┘ │
    └────────────────────────────────────────┘

* **Single Responsibility**: Each service owns one specific domain
* **Encapsulated Logic**: Business rules contained within the service
* **Data Ownership**: Service owns its database tables and schema
* **Clear Boundaries**: Well-defined interfaces and APIs

**Technology Stack**

All backend services share a common technology stack:

* **Language**: Go (Golang) 1.21+
* **HTTP Framework**: Gin for REST endpoints (when needed)
* **Database**: MySQL 8.0 via sqlx
* **Cache**: Redis 7.0 via go-redis
* **Message Queue**: RabbitMQ via bin-common-handler
* **Logging**: Structured logging with logrus
* **Monitoring**: Prometheus metrics

**Common Structure**

All services follow a consistent directory structure:

.. code::

    bin-<service>-manager/
    ├── cmd/
    │   └── <service>-manager/
    │       └── main.go                 # Entry point
    ├── pkg/
    │   ├── <domain>handler/            # Business logic
    │   ├── dbhandler/                  # Database operations
    │   ├── cachehandler/               # Redis operations
    │   └── listenhandler/              # RabbitMQ RPC handlers
    ├── models/
    │   └── <resource>/                 # Data models
    └── go.mod                          # Dependencies

API Gateway - bin-api-manager
------------------------------

The API Gateway serves as the single entry point for all external requests, handling authentication, authorization, and request routing to backend services.

**Gateway Responsibilities**

.. code::

    API Gateway Layer:

    External Clients
    (Web, Mobile, Server)
         │
         │ HTTPS
         ▼
    ┌────────────────────────────────────────┐
    │        bin-api-manager                 │
    │                                        │
    │  1. ┌────────────────────────────┐    │
    │     │  Authentication (JWT)      │    │
    │     └────────────────────────────┘    │
    │                                        │
    │  2. ┌────────────────────────────┐    │
    │     │  Authorization (Permissions)│   │
    │     └────────────────────────────┘    │
    │                                        │
    │  3. ┌────────────────────────────┐    │
    │     │  Rate Limiting / Throttling│    │
    │     └────────────────────────────┘    │
    │                                        │
    │  4. ┌────────────────────────────┐    │
    │     │  Request Routing (RabbitMQ)│    │
    │     └────────────────────────────┘    │
    │                                        │
    │  5. ┌────────────────────────────┐    │
    │     │  Response Aggregation      │    │
    │     └────────────────────────────┘    │
    └────────────────────────────────────────┘
         │
         │ RabbitMQ RPC
         ▼
    Backend Services

**Authentication Flow**

.. code::

    JWT Authentication:

    Client                    API Gateway              Backend Service
      │                            │                          │
      │  POST /auth/login          │                          │
      ├───────────────────────────▶│                          │
      │  {user, pass}              │                          │
      │                            │                          │
      │                            │  Verify credentials      │
      │                            │                          │
      │  JWT Token                 │                          │
      │◀───────────────────────────┤                          │
      │                            │                          │
      │                            │                          │
      │  GET /calls?token=xyz      │                          │
      ├───────────────────────────▶│                          │
      │                            │  1. Validate JWT         │
      │                            │  2. Extract customer_id  │
      │                            │  3. Check permissions    │
      │                            │                          │
      │                            │  RPC: GetCalls(ctx)      │
      │                            ├─────────────────────────▶│
      │                            │                          │
      │                            │  [Call List]             │
      │                            │◀─────────────────────────┤
      │                            │                          │
      │  [Call List]               │  4. Return response      │
      │◀───────────────────────────┤                          │
      │                            │                          │

**Authentication Components:**

* **JWT Validation**: Validates token signature and expiration
* **Customer Extraction**: Extracts customer_id from JWT claims
* **Permission Check**: Verifies user has required permissions
* **Context Propagation**: Passes auth context to backend services

**Authorization Pattern**

VoIPBIN implements authorization at the API Gateway, NOT in backend services:

.. code::

    Authorization Check:

    ┌─────────────────────────────────────────────────────┐
    │              bin-api-manager (Gateway)              │
    │                                                     │
    │  1. Fetch Resource                                  │
    │     ├──────▶ bin-call-manager.GetCall(call_id)     │
    │     │                                               │
    │  2. Check Authorization                             │
    │     │  if call.customer_id != jwt.customer_id:     │
    │     │      return 404 (not 403, for security)      │
    │     │                                               │
    │  3. Return Resource                                 │
    │     └──────▶ return call                           │
    │                                                     │
    └─────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────┐
    │           bin-call-manager (Backend)                │
    │                                                     │
    │  • NO authentication logic                          │
    │  • NO customer_id validation                        │
    │  • Just process RPC requests                        │
    │  • Return requested data                            │
    │                                                     │
    └─────────────────────────────────────────────────────┘

**Key Authorization Principles:**

* **Gateway-Only Auth**: All authorization logic in bin-api-manager
* **Fetch-Then-Check**: Fetch resource first, then verify ownership
* **Return 404, Not 403**: Return "not found" for unauthorized access (security)
* **Backend Trust**: Backend services trust the gateway

**Request Routing**

The gateway routes requests to appropriate backend services:

.. code::

    Routing Decision:

    HTTP Request          Gateway Router          Backend Service
        │                      │                        │
        │  GET /v1.0/calls     │                        │
        ├─────────────────────▶│                        │
        │                      │  Parse: "calls"        │
        │                      │  → bin-call-manager    │
        │                      │                        │
        │                      │  RPC Request           │
        │                      ├───────────────────────▶│
        │                      │                        │
        │                      │  RPC Response          │
        │                      │◀───────────────────────┤
        │                      │                        │
        │  JSON Response       │                        │
        │◀─────────────────────┤                        │
        │                      │                        │

**Routing Table:**

===============================  ========================
HTTP Endpoint                    Backend Service
===============================  ========================
/v1.0/calls                      bin-call-manager
/v1.0/conferences                bin-conference-manager
/v1.0/sms                        bin-sms-manager
/v1.0/chats                      bin-chat-manager
/v1.0/agents                     bin-agent-manager
/v1.0/queues                     bin-queue-manager
/v1.0/campaigns                  bin-campaign-manager
/v1.0/flows                      bin-flow-manager
/v1.0/billings                   bin-billing-manager
/v1.0/webhooks                   bin-webhook-manager
/v1.0/transcribes                bin-transcribe-manager
/v1.0/numbers                    bin-number-manager
===============================  ========================

Service Independence
--------------------

VoIPBIN's microservices architecture enables true service independence:

**Independent Deployment**

.. code::

    Service Deployment:

    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
    │  Service A   │  │  Service B   │  │  Service C   │
    │  v1.2.3      │  │  v2.0.1      │  │  v1.5.0      │
    └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
           │                 │                 │
           │                 │  Deploy v2.1.0  │
           │                 │  (no impact)    │
           │                 ▼                 │
           │          ┌──────────────┐         │
           │          │  Service B   │         │
           │          │  v2.1.0      │         │
           │          └──────────────┘         │
           │                 │                 │
           └─────────────────┴─────────────────┘
                       Message Queue

* **No Downtime**: Services update without affecting others
* **Version Independence**: Each service has its own version
* **Gradual Rollout**: Can deploy to subset of instances
* **Quick Rollback**: Easy to revert problematic deployments

**Independent Scaling**

.. code::

    Horizontal Scaling:

    Normal Load:              High Call Load:
    ┌──────────┐              ┌──────────┐ ┌──────────┐ ┌──────────┐
    │   Call   │              │   Call   │ │   Call   │ │   Call   │
    │  Manager │              │ Manager  │ │ Manager  │ │ Manager  │
    │   x1     │              │   x1     │ │   x2     │ │   x3     │
    └──────────┘              └──────────┘ └──────────┘ └──────────┘
    ┌──────────┐              ┌──────────┐
    │   SMS    │              │   SMS    │
    │  Manager │              │  Manager │
    │   x1     │              │   x1     │
    └──────────┘              └──────────┘

    Scale only what needs scaling

* **Targeted Scaling**: Scale only services experiencing load
* **Cost Optimization**: Don't over-provision underutilized services
* **Auto-Scaling**: Kubernetes HPA scales based on metrics
* **Resource Efficiency**: Better resource utilization

**Independent Development**

.. code::

    Development Isolation:

    Team A              Team B              Team C
       │                   │                   │
       │  bin-call-        │  bin-flow-        │  bin-ai-
       │  manager          │  manager          │  manager
       │                   │                   │
       │  • Go codebase    │  • Go codebase    │  • Go codebase
       │  • Own git        │  • Own git        │  • Own git
       │    branch         │    branch         │    branch
       │  • Own CI/CD      │  • Own CI/CD      │  • Own CI/CD
       │  • Own tests      │  • Own tests      │  • Own tests
       │                   │                   │
       └───────────────────┴───────────────────┘
              Coordinate only via:
              • Message contracts
              • Database schema
              • API contracts

* **Team Autonomy**: Teams work independently
* **Faster Development**: No coordination bottleneck
* **Technology Flexibility**: Can use different libraries
* **Clear Ownership**: Each team owns specific domains

Service Communication Patterns
-------------------------------

Services communicate primarily through RabbitMQ RPC:

**Synchronous RPC (Request-Response)**

.. code::

    RPC Communication:

    API Gateway                RabbitMQ              Call Manager
         │                         │                      │
         │  1. Call Request        │                      │
         ├────────────────────────▶│                      │
         │  Queue: bin-manager.    │                      │
         │         call.request    │                      │
         │                         │  2. Dequeue Request  │
         │                         ├─────────────────────▶│
         │                         │                      │
         │                         │  3. Process Request  │
         │                         │      (create call)   │
         │                         │                      │
         │                         │  4. Send Response    │
         │                         │◀─────────────────────┤
         │  5. Response            │                      │
         │◀────────────────────────┤                      │
         │                         │                      │

**Asynchronous Events (Pub/Sub)**

.. code::

    Event Broadcasting:

    Call Manager          RabbitMQ Exchange        Subscribers
         │                      │                       │
         │  1. Call Created     │                       │
         │  (publish event)     │                       │
         ├─────────────────────▶│                       │
         │                      │                       │
         │                      │  2. Broadcast         │
         │                      │      to all           │
         │                      ├──────────┬────────────┤
         │                      │          │            │
         │                      │          ▼            ▼
         │                      │    ┌──────────┐ ┌──────────┐
         │                      │    │ Billing  │ │ Webhook  │
         │                      │    │ Manager  │ │ Manager  │
         │                      │    └──────────┘ └──────────┘
         │                      │                       │
         │                      │    Process event      │
         │                      │    independently      │

**Communication Patterns Used:**

* **RPC (Synchronous)**: For request-response operations (GET, POST, DELETE)
* **Pub/Sub (Asynchronous)**: For event notifications (call.created, sms.sent)
* **Webhooks**: For external system notifications
* **WebSocket**: For real-time client updates

Service Discovery and Configuration
------------------------------------

VoIPBIN uses a hybrid approach for service discovery:

**Queue-Based Discovery**

.. code::

    Service Registration:

    ┌────────────────────────────────────────────────┐
    │            RabbitMQ Queue Naming               │
    │                                                │
    │  bin-manager.<service>.<operation>             │
    │                                                │
    │  Examples:                                     │
    │  • bin-manager.call.request                    │
    │  • bin-manager.conference.request              │
    │  • bin-manager.sms.request                     │
    │                                                │
    │  Services listen on their named queues         │
    │  Clients send to known queue names             │
    └────────────────────────────────────────────────┘

* **Convention-Based**: Queue names follow predictable pattern
* **No Registry**: No central service registry needed
* **Self-Registering**: Services create queues on startup
* **Load Balanced**: Multiple instances share same queue

**Configuration Management**

Services receive configuration through multiple sources:

.. code::

    Configuration Sources:

    ┌────────────────┐
    │   Service      │
    └────┬───────────┘
         │
         ├──────▶ Environment Variables
         │       • Database connection
         │       • RabbitMQ address
         │       • Redis address
         │
         ├──────▶ Command-Line Flags
         │       • Port number
         │       • Log level
         │
         ├──────▶ bin-config-manager
         │       • Feature flags
         │       • Business logic config
         │
         └──────▶ Database
                 • Dynamic configuration
                 • Customer-specific settings

Health Monitoring
-----------------

All services expose health check endpoints:

.. code::

    Health Check Architecture:

    Kubernetes              Service Health          Dependencies
         │                       │                       │
         │  1. Health Check      │                       │
         ├──────────────────────▶│                       │
         │  GET /health          │                       │
         │                       │  2. Check MySQL       │
         │                       ├──────────────────────▶│
         │                       │     (ping)            │
         │                       │                       │
         │                       │  3. Check Redis       │
         │                       ├──────────────────────▶│
         │                       │     (ping)            │
         │                       │                       │
         │                       │  4. Check RabbitMQ    │
         │                       ├──────────────────────▶│
         │                       │     (connection)      │
         │                       │                       │
         │  200 OK / 503 Error   │                       │
         │◀──────────────────────┤                       │
         │                       │                       │
         │  5. Restart if failed │                       │
         │  (after retries)      │                       │

**Health Check Components:**

* **Liveness Probe**: Is the service running?
* **Readiness Probe**: Is the service ready to accept traffic?
* **Dependency Checks**: Are database, cache, queue healthy?
* **Auto-Recovery**: Kubernetes restarts unhealthy pods

Error Handling and Resilience
------------------------------

Services implement multiple resilience patterns:

**Circuit Breaker**

.. code::

    Circuit Breaker States:

    Closed (Normal)         Open (Failed)          Half-Open (Testing)
         │                       │                       │
         │  Requests pass        │  Requests rejected    │  Limited requests
         │  through              │  immediately          │  allowed
         │                       │                       │
         │  ───────────▶         │  ────────X            │  ───────────▶
         │                       │                       │
         │  If failures          │  After timeout        │  If success
         │  exceed threshold     │  period               │  threshold met
         │                       │                       │
         └──────────────────────▶│                       │
                                 │◀──────────────────────┤
                                 │                       │
                                 └──────────────────────▶│
                                   If still failing      │
                                                         │
                                                         └──────▶ Closed

* **Prevent Cascade Failures**: Stop calling failed services
* **Fast Fail**: Return error immediately when circuit open
* **Auto-Recovery**: Periodically test if service recovered

**Retry with Backoff**

.. code::

    Exponential Backoff:

    Attempt 1: Immediate
         │
         │ Failed
         ▼
    Attempt 2: Wait 1s
         │
         │ Failed
         ▼
    Attempt 3: Wait 2s
         │
         │ Failed
         ▼
    Attempt 4: Wait 4s
         │
         │ Failed
         ▼
    Attempt 5: Wait 8s
         │
         │ Failed
         ▼
    Give up, return error

* **Transient Failures**: Retry on temporary failures
* **Backoff Strategy**: Increase wait time between retries
* **Max Attempts**: Limit total number of retries
* **Idempotency**: Ensure operations safe to retry

**Timeouts**

All RPC calls have strict timeouts:

* **Default Timeout**: 30 seconds for most operations
* **Long Operations**: 120 seconds for complex workflows
* **Streaming**: No timeout for streaming operations
* **Context Propagation**: Timeout passed through call chain

Deployment Architecture
-----------------------

Services deploy to Kubernetes on Google Cloud Platform:

.. code::

    Kubernetes Deployment:

    ┌─────────────────────────────────────────────────────────┐
    │                      GKE Cluster                        │
    │                                                         │
    │  ┌───────────────────────────────────────────────────┐ │
    │  │              Namespace: production                │ │
    │  │                                                   │ │
    │  │  ┌─────────────────────────────────────────────┐ │ │
    │  │  │  Deployment: bin-call-manager               │ │ │
    │  │  │  ┌─────────┐  ┌─────────┐  ┌─────────┐     │ │ │
    │  │  │  │  Pod 1  │  │  Pod 2  │  │  Pod 3  │     │ │ │
    │  │  │  └─────────┘  └─────────┘  └─────────┘     │ │ │
    │  │  │  Replicas: 3    HPA: 3-10                   │ │ │
    │  │  └─────────────────────────────────────────────┘ │ │
    │  │                                                   │ │
    │  │  ┌─────────────────────────────────────────────┐ │ │
    │  │  │  Deployment: bin-api-manager                │ │ │
    │  │  │  ┌─────────┐  ┌─────────┐  ┌─────────┐     │ │ │
    │  │  │  │  Pod 1  │  │  Pod 2  │  │  Pod 3  │     │ │ │
    │  │  │  └─────────┘  └─────────┘  └─────────┘     │ │ │
    │  │  │  Replicas: 3    HPA: 3-20                   │ │ │
    │  │  └─────────────────────────────────────────────┘ │ │
    │  │                                                   │ │
    │  │  ... 30+ more deployments                         │ │
    │  │                                                   │ │
    │  └───────────────────────────────────────────────────┘ │
    │                                                         │
    │  ┌───────────────────────────────────────────────────┐ │
    │  │         Shared Resources (same cluster)           │ │
    │  │  • MySQL StatefulSet                              │ │
    │  │  • Redis StatefulSet                              │ │
    │  │  • RabbitMQ StatefulSet                           │ │
    │  │  • Prometheus Monitoring                          │ │
    │  └───────────────────────────────────────────────────┘ │
    └─────────────────────────────────────────────────────────┘

**Deployment Characteristics:**

* **Container-Based**: Each service runs in Docker containers
* **Replica Sets**: Multiple instances for high availability
* **Auto-Scaling**: HPA (Horizontal Pod Autoscaler) based on CPU/memory
* **Rolling Updates**: Zero-downtime deployments
* **Resource Limits**: CPU and memory limits per container
* **Health Probes**: Automatic restart of failed containers

Monitoring and Observability
-----------------------------

Comprehensive monitoring across all services:

**Metrics Collection**

.. code::

    Metrics Pipeline:

    Services                Prometheus              Grafana
    (30+ services)              │                      │
         │                      │                      │
         │  Expose /metrics     │                      │
         │  endpoint            │                      │
         │                      │                      │
         │  Scrape every 15s    │                      │
         ├─────────────────────▶│                      │
         │                      │                      │
         │                      │  Time-series DB      │
         │                      │  stores metrics      │
         │                      │                      │
         │                      │  Query metrics       │
         │                      ├─────────────────────▶│
         │                      │                      │
         │                      │  Visualize           │
         │                      │  dashboards          │
         │                      │                      │

**Key Metrics:**

* **Request Rate**: Requests per second per service
* **Error Rate**: Failed requests percentage
* **Latency**: P50, P95, P99 response times
* **Resource Usage**: CPU, memory, disk per pod
* **Queue Depth**: RabbitMQ queue backlogs
* **Database Connections**: Active connections per service

**Logging**

All services use structured logging:

.. code::

    {
      "timestamp": "2026-01-20T12:00:00.000Z",
      "level": "info",
      "service": "bin-call-manager",
      "instance": "pod-xyz",
      "message": "Call created successfully",
      "call_id": "abc-123-def",
      "customer_id": "customer-789",
      "duration_ms": 45
    }

* **Structured Format**: JSON logs for easy parsing
* **Centralized Collection**: All logs aggregated in one place
* **Searchable**: Full-text search across all services
* **Correlation IDs**: Track requests across services

Best Practices
--------------

VoIPBIN's backend follows these best practices:

**Service Design:**

* One service, one responsibility
* Services communicate via messages, not direct calls
* Shared database, but logical isolation by tables
* Idempotent operations for safe retries

**Error Handling:**

* Always return errors, never panic
* Use context for timeouts and cancellation
* Implement circuit breakers for external dependencies
* Log errors with full context

**Performance:**

* Use connection pooling for database and Redis
* Implement caching for frequently accessed data
* Use batch operations where possible
* Monitor and optimize hot paths

**Security:**

* No authentication logic in backend services
* Trust the API gateway for auth decisions
* Validate all inputs at service boundaries
* Use parameterized queries to prevent SQL injection

**Testing:**

* Unit tests for business logic
* Integration tests with mock dependencies
* End-to-end tests for critical flows
* Load tests before production deployment
