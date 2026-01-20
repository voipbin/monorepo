.. _architecture-data:

Data Architecture
=================

VoIPBIN uses a shared data layer with MySQL for persistent storage and Redis for caching and session management. This architecture provides consistency across services while enabling high-performance data access.

Data Layer Overview
-------------------

VoIPBIN's data architecture consists of three layers:

.. code::

    Data Architecture:

    ┌─────────────────────────────────────────────────────────┐
    │                   Application Layer                     │
    │            (30+ Microservices)                          │
    └────────────────────┬───────────────────┬────────────────┘
                         │                   │
                         │                   │
         ┌───────────────▼──────┐   ┌────────▼───────────┐
         │                      │   │                    │
         │   Redis Cache        │   │   MySQL Database   │
         │   (Hot Data)         │   │   (Persistent)     │
         │                      │   │                    │
         │  • Sessions          │   │  • All entities    │
         │  • Frequently read   │   │  • Relationships   │
         │  • Temporary data    │   │  • Audit logs      │
         │                      │   │                    │
         └──────────────────────┘   └────────────────────┘

         Cache-Aside Pattern:
         1. Check cache first
         2. If miss, query database
         3. Store in cache for next time

MySQL Database
--------------

VoIPBIN uses a single shared MySQL database accessed by all services.

**Database Characteristics**

.. code::

    Shared Database Pattern:

    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
    │   Service A  │  │   Service B  │  │   Service C  │
    │              │  │              │  │              │
    │  call-mgr    │  │  flow-mgr    │  │  agent-mgr   │
    └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
           │                 │                 │
           │   Connection    │                 │
           │   Pooling       │                 │
           └────────┬────────┴─────────────────┘
                    │
                    ▼
         ┌────────────────────────────┐
         │      MySQL Database        │
         │                            │
         │  ┌──────────────────────┐  │
         │  │  calls table         │  │
         │  │  conferences table   │  │
         │  │  agents table        │  │
         │  │  flows table         │  │
         │  │  customers table     │  │
         │  │  ... 100+ tables     │  │
         │  └──────────────────────┘  │
         └────────────────────────────┘

* **Shared Schema**: All services access same database
* **Logical Separation**: Services own specific tables
* **ACID Transactions**: Strong consistency guarantees
* **Connection Pooling**: Each service maintains pool

**Schema Organization**

Tables are logically grouped by domain:

.. code::

    Table Organization:

    Communication Domain:
    • calls                - Call records
    • conferences          - Conference bridges
    • sms                  - SMS messages
    • chats                - Chat messages
    • emails               - Email records

    Workflow Domain:
    • flows                - Call flow definitions
    • flow_actions         - Flow action steps
    • queues               - Call queues
    • campaigns            - Campaign definitions

    Management Domain:
    • customers            - Customer accounts
    • agents               - Agent records
    • billings             - Billing records
    • webhooks             - Webhook configurations
    • accesskeys           - API keys

    Resource Domain:
    • numbers              - Phone numbers
    • recordings           - Call recordings
    • transcribes          - Transcription jobs
    • transcripts          - Transcript segments

**Common Table Pattern**

All tables follow a consistent structure:

.. code::

    Standard Table Schema:

    CREATE TABLE resource (
        id              VARCHAR(36) PRIMARY KEY,    -- UUID
        customer_id     VARCHAR(36) NOT NULL,       -- Ownership

        -- Resource-specific fields
        name            VARCHAR(255),
        status          VARCHAR(50),
        detail          TEXT,

        -- Timestamps
        tm_create       DATETIME(6) NOT NULL,       -- Creation time
        tm_update       DATETIME(6) NOT NULL,       -- Last update
        tm_delete       DATETIME(6) NOT NULL,       -- Soft delete

        -- Indexes
        INDEX idx_customer (customer_id),
        INDEX idx_status (status),
        INDEX idx_tm_create (tm_create),
        INDEX idx_tm_delete (tm_delete)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

**Key Design Patterns:**

* **UUID Primary Keys**: Globally unique identifiers
* **Customer Ownership**: Every resource has customer_id
* **Soft Deletes**: tm_delete = '9999-01-01' for active records
* **Microsecond Timestamps**: DATETIME(6) for precise ordering
* **UTF8MB4**: Full Unicode support including emojis

**Data Access Patterns**

Services access data through consistent patterns:

.. code::

    Data Access Flow:

    Service Handler
         │
         │  1. Validate Input
         ▼
    ┌──────────────────────┐
    │  Business Logic      │
    └──────┬───────────────┘
           │  2. Check Cache
           ▼
    ┌──────────────────────┐
    │  Cache Handler       │
    │  (Redis)             │
    └──────┬───────────────┘
           │  Cache Miss
           │  3. Query DB
           ▼
    ┌──────────────────────┐
    │  DB Handler          │
    │  (MySQL)             │
    └──────┬───────────────┘
           │  4. Store in Cache
           ▼
    ┌──────────────────────┐
    │  Return Result       │
    └──────────────────────┘

**Transaction Handling**

VoIPBIN uses transactions for consistency:

.. code::

    Transaction Example:

    BEGIN TRANSACTION
        │
        │  1. Create Call Record
        ├──▶ INSERT INTO calls ...
        │
        │  2. Update Customer Stats
        ├──▶ UPDATE customers SET total_calls = total_calls + 1 ...
        │
        │  3. Create Billing Entry
        ├──▶ INSERT INTO billings ...
        │
        │  If all succeed:
        │    COMMIT
        │  If any fails:
        │    ROLLBACK
        │
    END TRANSACTION

* **ACID Guarantees**: Atomic, Consistent, Isolated, Durable
* **Rollback on Error**: All changes reverted if any step fails
* **Isolation Levels**: READ COMMITTED for most operations
* **Lock Timeout**: 30 seconds to prevent deadlocks

**Query Optimization**

VoIPBIN optimizes queries for performance:

.. code::

    Query Optimization Strategies:

    1. Proper Indexing:
       ┌─────────────────────────────────┐
       │ INDEX idx_customer_status       │
       │ ON calls (customer_id, status)  │
       └─────────────────────────────────┘

       SELECT * FROM calls
       WHERE customer_id = ? AND status = 'active'
       → Uses index, fast lookup

    2. Avoid SELECT *:
       ┌─────────────────────────────────┐
       │ SELECT id, status, tm_create    │
       │ FROM calls WHERE ...            │
       └─────────────────────────────────┘
       → Only retrieve needed columns

    3. Pagination:
       ┌─────────────────────────────────┐
       │ SELECT * FROM calls             │
       │ WHERE customer_id = ?           │
       │ LIMIT 50 OFFSET 0               │
       └─────────────────────────────────┘
       → Limit result size

    4. Connection Pooling:
       ┌─────────────────────────────────┐
       │ Pool Size: 10-50 connections    │
       │ Max Idle: 5 minutes             │
       │ Max Lifetime: 1 hour            │
       └─────────────────────────────────┘
       → Reuse connections

Database Migrations
-------------------

Schema changes are managed through Alembic migrations:

.. code::

    Migration Workflow:

    Development                 Migration Script              Production
         │                            │                           │
         │  1. Schema Change          │                           │
         │     Needed                 │                           │
         ▼                            │                           │
    ┌─────────────┐                   │                           │
    │ Create      │                   │                           │
    │ Migration   │──────────────────▶│                           │
    │ Script      │                   │                           │
    └─────────────┘                   │                           │
         │                            │                           │
         │  2. Test Locally           │                           │
         ▼                            │                           │
    ┌─────────────┐                   │                           │
    │ Run         │                   │                           │
    │ Migration   │◀──────────────────│                           │
    │ (dev DB)    │                   │                           │
    └─────────────┘                   │                           │
         │                            │                           │
         │  3. Commit to Git          │                           │
         ▼                            │                           │
    ┌─────────────┐                   │                           │
    │ Code Review │                   │                           │
    │ & Approval  │                   │                           │
    └─────────────┘                   │                           │
         │                            │                           │
         │  4. Deploy                 │                           │
         │                            │  5. Manual Execution      │
         │                            │     (by human)            │
         │                            ├──────────────────────────▶│
         │                            │                           │
         │                            │  alembic upgrade head     │
         │                            │                           │

**Migration Best Practices:**

* **Version Control**: All migrations in git
* **Forward Only**: Never modify existing migrations
* **Backward Compatible**: Support gradual rollout
* **Manual Execution**: Humans run migrations, not automation
* **Testing**: Test on staging before production

Redis Cache
-----------

Redis provides fast access to frequently used data:

**Cache Architecture**

.. code::

    Redis Cache Pattern:

    Application Request
         │
         │  1. Generate Cache Key
         │     key = "call:123"
         ▼
    ┌────────────────────┐
    │  Check Redis       │
    │  GET call:123      │
    └────┬───────────────┘
         │
         ├─ Cache Hit ────────┐
         │                    │
         │                    ▼
         │              ┌────────────────┐
         │              │ Return Cached  │
         │              │ Data (fast)    │
         │              └────────────────┘
         │
         ├─ Cache Miss ───────┐
         │                    │
         │                    ▼
         │              ┌────────────────┐
         │              │ Query MySQL    │
         │              └────┬───────────┘
         │                   │
         │                   ▼
         │              ┌────────────────┐
         │              │ Store in Redis │
         │              │ SET call:123   │
         │              │ EX 300 (5 min) │
         │              └────┬───────────┘
         │                   │
         │                   ▼
         │              ┌────────────────┐
         │              │ Return Data    │
         │              └────────────────┘

**Cache Key Patterns**

VoIPBIN uses structured cache keys:

.. code::

    Key Naming Convention:
    <resource>:<id>[:<field>]

    Examples:
    • call:abc-123              → Full call record
    • agent:xyz-789:status      → Agent status only
    • customer:customer-456     → Customer record
    • queue:queue-999:stats     → Queue statistics
    • flow:flow-111:definition  → Flow definition

    Advantages:
    • Predictable keys
    • Easy to invalidate
    • Pattern matching for bulk operations

**Data Structures**

Redis supports multiple data structures:

.. code::

    Redis Data Structures:

    1. String (Simple Values):
       SET call:123:status "active"
       GET call:123:status
       → "active"

    2. Hash (Object Fields):
       HSET call:123 status "active" duration "120"
       HGET call:123 status
       → "active"
       HGETALL call:123
       → {"status": "active", "duration": "120"}

    3. List (Ordered Collection):
       LPUSH queue:456:waiting call:123
       LPUSH queue:456:waiting call:789
       LRANGE queue:456:waiting 0 -1
       → [call:789, call:123]

    4. Set (Unique Collection):
       SADD conference:999:participants agent:111
       SADD conference:999:participants agent:222
       SMEMBERS conference:999:participants
       → [agent:111, agent:222]

    5. Sorted Set (Scored Collection):
       ZADD leaderboard 100 agent:111
       ZADD leaderboard 95 agent:222
       ZRANGE leaderboard 0 -1 WITHSCORES
       → [(agent:111, 100), (agent:222, 95)]

**Cache Expiration**

All cached data has Time-To-Live (TTL):

.. code::

    TTL Strategy:

    Data Type              TTL        Reason
    ─────────────────────────────────────────────
    Session tokens         1 hour     Security
    User profiles          5 min      Frequently updated
    Call records           1 min      Real-time changes
    Configuration          1 hour     Rarely changes
    Static data            24 hours   Almost never changes

    Set TTL:
    SET key value EX 300   # 5 minutes
    SETEX key 300 value    # Same as above
    EXPIRE key 300         # Set TTL on existing key

**Cache Invalidation**

VoIPBIN invalidates cache on updates:

.. code::

    Cache Invalidation Flow:

    Update Request
         │
         │  1. Update Database
         ▼
    ┌────────────────────┐
    │  UPDATE calls      │
    │  SET status='ended'│
    │  WHERE id='123'    │
    └────┬───────────────┘
         │
         │  2. Invalidate Cache
         ▼
    ┌────────────────────┐
    │  DEL call:123      │
    └────┬───────────────┘
         │
         │  3. Return Success
         ▼
    ┌────────────────────┐
    │  Response to Client│
    └────────────────────┘

    Next Read:
    • Cache miss
    • Fetch from DB
    • Store in cache with new data

**Cache Patterns**

.. code::

    Common Cache Patterns:

    1. Cache-Aside (Read Through):
       App checks cache → Cache miss → Query DB → Store in cache

    2. Write-Through:
       App writes to cache → Cache writes to DB → Return success

    3. Write-Behind (Async):
       App writes to cache → Return success → Cache writes to DB later

    VoIPBIN primarily uses Cache-Aside for simplicity and consistency.

Session Management
------------------

Redis stores session data for authenticated users:

**Session Structure**

.. code::

    Session Data in Redis:

    Key: session:<token-hash>
    Type: Hash
    TTL: 1 hour (refreshed on activity)

    Data:
    ┌─────────────────────────────────────┐
    │ customer_id    : customer-123       │
    │ agent_id       : agent-456          │
    │ permissions    : ["admin", "call"]  │
    │ login_time     : 2026-01-20 12:00   │
    │ last_activity  : 2026-01-20 12:30   │
    │ ip_address     : 192.168.1.100      │
    │ user_agent     : Mozilla/5.0 ...    │
    └─────────────────────────────────────┘

**Session Lifecycle**

.. code::

    Session Flow:

    1. Login:
       ┌────────────────────────────┐
       │ Generate JWT token         │
       │ Hash token → session_key   │
       │ Store session in Redis     │
       │ SET session:xyz {...}      │
       │ EXPIRE session:xyz 3600    │
       └────────────────────────────┘

    2. Request:
       ┌────────────────────────────┐
       │ Extract token from header  │
       │ Hash token → session_key   │
       │ GET session:xyz            │
       │ Validate session data      │
       │ EXPIRE session:xyz 3600    │  ← Refresh TTL
       └────────────────────────────┘

    3. Logout:
       ┌────────────────────────────┐
       │ Extract token from header  │
       │ Hash token → session_key   │
       │ DEL session:xyz            │
       └────────────────────────────┘

Data Consistency
----------------

VoIPBIN ensures consistency across data layers:

**Consistency Model**

.. code::

    Consistency Strategy:

    Strong Consistency:        Eventual Consistency:
    ┌──────────────┐           ┌──────────────┐
    │   MySQL      │           │   Redis      │
    │  (Source of  │           │  (May be     │
    │   Truth)     │           │   stale)     │
    └──────┬───────┘           └──────┬───────┘
           │                          │
           │  Always consistent       │  May lag behind
           │  ACID transactions       │  Best effort
           │                          │
           └──────────┬───────────────┘
                      │
                Database is authoritative

**Write Path**

.. code::

    Write Flow (Strong Consistency):

    1. Write Request
       │
       ▼
    2. Update Database First
       ├─ BEGIN TRANSACTION
       ├─ UPDATE table ...
       ├─ COMMIT
       │
       ▼
    3. Invalidate Cache
       ├─ DEL cache_key
       │
       ▼
    4. Publish Event
       ├─ Notify subscribers
       │
       ▼
    5. Return Success

    Database updated before cache invalidation
    ensures consistency.

**Read Path**

.. code::

    Read Flow (Eventual Consistency Acceptable):

    1. Read Request
       │
       ▼
    2. Check Cache
       ├─ Cache Hit → Return (may be slightly stale)
       ├─ Cache Miss → Continue
       │
       ▼
    3. Query Database
       ├─ SELECT * FROM table WHERE ...
       │
       ▼
    4. Store in Cache
       ├─ SET cache_key value EX ttl
       │
       ▼
    5. Return Result

Data Backup and Recovery
-------------------------

VoIPBIN implements comprehensive backup strategy:

**Backup Architecture**

.. code::

    Backup Strategy:

    Production Database
         │
         │  Continuous Replication
         ▼
    ┌────────────────────┐
    │  Read Replica      │  ← Used for backups
    └────┬───────────────┘    (no production impact)
         │
         │  Daily Full Backup
         ▼
    ┌────────────────────┐
    │  Backup Storage    │
    │  (Google Cloud)    │
    │                    │
    │  • Daily: 30 days  │
    │  • Weekly: 1 year  │
    │  • Monthly: 7 years│
    └────────────────────┘

**Backup Schedule**

.. code::

    Backup Timeline:

    Daily (3 AM UTC):
    ┌──────────────────────────────┐
    │ Full database dump           │
    │ Stored for 30 days           │
    │ ~100 GB compressed           │
    └──────────────────────────────┘

    Weekly (Sunday 3 AM):
    ┌──────────────────────────────┐
    │ Full database dump           │
    │ Stored for 1 year            │
    │ Long-term retention          │
    └──────────────────────────────┘

    Continuous:
    ┌──────────────────────────────┐
    │ Binary logs (point-in-time)  │
    │ Stored for 7 days            │
    │ For recovery between backups │
    └──────────────────────────────┘

**Recovery Procedures**

.. code::

    Recovery Scenarios:

    1. Recent Data Loss (< 7 days):
       ┌────────────────────────────┐
       │ Restore latest daily backup│
       │ Apply binary logs          │
       │ Point-in-time recovery     │
       └────────────────────────────┘
       Recovery time: 1-2 hours

    2. Older Data Loss (< 1 year):
       ┌────────────────────────────┐
       │ Restore weekly backup      │
       │ No binary logs available   │
       └────────────────────────────┘
       Recovery time: 2-4 hours

    3. Disaster Recovery:
       ┌────────────────────────────┐
       │ Failover to replica        │
       │ Promote to primary         │
       │ Restore from backup        │
       └────────────────────────────┘
       Recovery time: 15 minutes

Performance Monitoring
----------------------

VoIPBIN monitors data layer performance:

**Database Metrics**

.. code::

    Key Database Metrics:

    Query Performance:
    ┌─────────────────────────────────────┐
    │ Slow queries (> 1 second): 0.1%     │
    │ Average query time: 5ms             │
    │ P95 query time: 50ms                │
    │ P99 query time: 200ms               │
    └─────────────────────────────────────┘

    Connection Pool:
    ┌─────────────────────────────────────┐
    │ Active connections: 45/50           │
    │ Idle connections: 5/50              │
    │ Wait time: < 1ms                    │
    └─────────────────────────────────────┘

    Table Size:
    ┌─────────────────────────────────────┐
    │ calls:        50 million rows       │
    │ conferences:  5 million rows        │
    │ agents:       10,000 rows           │
    │ Total size:   500 GB                │
    └─────────────────────────────────────┘

**Cache Metrics**

.. code::

    Redis Performance:

    Hit Rate:
    ┌─────────────────────────────────────┐
    │ Cache hits:   95%                   │
    │ Cache misses: 5%                    │
    │ Target:       > 90%                 │
    └─────────────────────────────────────┘

    Memory Usage:
    ┌─────────────────────────────────────┐
    │ Used memory: 8 GB / 16 GB           │
    │ Peak memory: 12 GB                  │
    │ Eviction:    LRU policy             │
    └─────────────────────────────────────┘

    Latency:
    ┌─────────────────────────────────────┐
    │ P50: 0.5ms                          │
    │ P95: 2ms                            │
    │ P99: 5ms                            │
    └─────────────────────────────────────┘

Scalability Considerations
---------------------------

As VoIPBIN scales, data layer adapts:

**Database Scaling**

.. code::

    Scaling Strategy:

    Current (< 1M customers):
    ┌──────────────────────────┐
    │   Single Primary         │
    │   + Read Replicas (3)    │
    └──────────────────────────┘

    Future (> 1M customers):
    ┌──────────────────────────┐
    │   Sharding by Customer   │
    │                          │
    │   Shard 1: customers A-M │
    │   Shard 2: customers N-Z │
    └──────────────────────────┘

**Cache Scaling**

.. code::

    Redis Scaling:

    Current:
    ┌──────────────────────────┐
    │  Single Redis Instance   │
    │  16 GB Memory            │
    └──────────────────────────┘

    Future:
    ┌──────────────────────────┐
    │  Redis Cluster           │
    │  • Multiple nodes        │
    │  • Automatic sharding    │
    │  • High availability     │
    └──────────────────────────┘

Best Practices
--------------

**Database:**

* Use indexes for all WHERE clauses
* Avoid SELECT *, specify columns
* Use connection pooling
* Set appropriate timeouts
* Monitor slow queries
* Regular ANALYZE TABLE for statistics

**Cache:**

* Set appropriate TTLs
* Invalidate on updates
* Use structured keys
* Monitor hit rates
* Handle cache failures gracefully
* Don't store large objects (> 1MB)

**Security:**

* Use parameterized queries (prevent SQL injection)
* Encrypt sensitive data at rest
* Use SSL/TLS for connections
* Rotate database credentials regularly
* Audit database access
* Restrict network access

**Monitoring:**

* Track query performance
* Monitor connection pool utilization
* Alert on cache hit rate < 90%
* Alert on slow queries
* Monitor disk space
* Track replication lag
