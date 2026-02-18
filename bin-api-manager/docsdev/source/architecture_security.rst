.. _architecture-security:

Security Architecture
=====================

.. note:: **AI Context**

   This page describes VoIPBIN's security architecture: four defense-in-depth layers (edge, API gateway, internal services, data), JWT and Access Key authentication, RBAC authorization model, transport security (TLS, SRTP), Kubernetes secrets management, network isolation (VPC, firewall rules, network policies), input validation, rate limiting, DDoS protection, audit logging, data protection, and compliance frameworks. Relevant when an AI agent needs to understand authentication flows, permission models, or security controls.

VoIPBIN implements defense-in-depth security across all layers, from API authentication to data encryption. This section details the security architecture, authentication flows, and protection mechanisms.

Security Overview
-----------------

.. code::

    Security Layers:

    +------------------------------------------------------------------+
    |                      External Clients                            |
    +------------------------------------------------------------------+
                                |
                                v
    +------------------------------------------------------------------+
    | Layer 1: Edge Security                                           |
    |  o TLS 1.3 encryption                                            |
    |  o DDoS protection (Cloud Armor)                                 |
    |  o WAF rules                                                     |
    +------------------------------------------------------------------+
                                |
                                v
    +------------------------------------------------------------------+
    | Layer 2: API Gateway (bin-api-manager)                           |
    |  o JWT/AccessKey authentication                                  |
    |  o Authorization checks                                          |
    |  o Rate limiting                                                 |
    |  o Input validation                                              |
    +------------------------------------------------------------------+
                                |
                                v
    +------------------------------------------------------------------+
    | Layer 3: Internal Services                                       |
    |  o Network isolation (VPC)                                       |
    |  o Service-to-service trust                                      |
    |  o No external exposure                                          |
    +------------------------------------------------------------------+
                                |
                                v
    +------------------------------------------------------------------+
    | Layer 4: Data Layer                                              |
    |  o Encryption at rest                                            |
    |  o Encrypted connections                                         |
    |  o Access controls                                               |
    +------------------------------------------------------------------+

Authentication Architecture
---------------------------

VoIPBIN supports two authentication methods: JWT tokens and Access Keys.

**Authentication Flow:**

.. code::

    JWT Authentication Flow:

    Client                  API Gateway              Auth Service
       |                        |                         |
       | POST /auth/login       |                         |
       | (username, password)   |                         |
       +----------------------->|                         |
       |                        |                         |
       |                        | Validate Credentials    |
       |                        +------------------------>|
       |                        |                         |
       |                        |<------------------------+
       |                        | User Valid + Permissions|
       |                        |                         |
       |                        | Generate JWT Token      |
       |                        | (HS256 signed)          |
       |                        |                         |
       |<-----------------------+                         |
       | { "token": "eyJ..." }  |                         |
       |                        |                         |
       |                        |                         |
       | GET /v1/calls          |                         |
       | Authorization: Bearer eyJ...                     |
       +----------------------->|                         |
       |                        |                         |
       |                        | Validate JWT:           |
       |                        | 1. Verify signature     |
       |                        | 2. Check expiration     |
       |                        | 3. Extract claims       |
       |                        |                         |
       |                        | Route to call-manager   |
       |                        | (with customer_id)      |
       |                        |                         |
       |<-----------------------+                         |
       | { calls: [...] }       |                         |
       |                        |                         |

**JWT Token Structure:**

.. code::

    JWT Token Claims:

    Header:
    {
      "alg": "HS256",
      "typ": "JWT"
    }

    Payload:
    {
      "customer_id": "uuid",      // Customer UUID
      "agent_id": "uuid",         // Agent UUID (optional)
      "permissions": [            // Permission list
        "customer_admin",
        "call_create",
        "call_read"
      ],
      "iat": 1706000000,          // Issued at
      "exp": 1706003600           // Expires (1 hour)
    }

    Signature:
    HMACSHA256(
      base64UrlEncode(header) + "." +
      base64UrlEncode(payload),
      secret
    )

**Access Key Authentication:**

.. code::

    Access Key Flow:

    Client                  API Gateway
       |                        |
       | GET /v1/calls          |
       | Authorization: AccessKey ak_xxxxx
       +----------------------->|
       |                        |
       |                        | Lookup Access Key:
       |                        | 1. Find in database
       |                        | 2. Verify not expired
       |                        | 3. Check permissions
       |                        | 4. Get customer_id
       |                        |
       |                        | Route to call-manager
       |                        | (with customer_id)
       |                        |
       |<-----------------------+
       | { calls: [...] }       |
       |                        |

**Access Key Structure:**

.. code::

    Access Key:
    +------------------------------------------+
    | Format: ak_<32-character-random-string>  |
    | Example: ak_a1b2c3d4e5f6g7h8i9j0k1l2m3n4|
    +------------------------------------------+

    Database Record:
    +------------------------------------------+
    | id:           UUID                       |
    | customer_id:  UUID                       |
    | key_hash:     SHA256 hash of key         |
    | permissions:  JSON array                 |
    | tm_expire:    Expiration timestamp       |
    | tm_create:    Creation timestamp         |
    +------------------------------------------+

Authorization Model
-------------------

VoIPBIN uses role-based access control (RBAC):

**Permission Hierarchy:**

.. code::

    Permission Structure:

    Customer Level:
    +------------------------------------------+
    | customer_admin                           |
    |  o Full access to all customer resources|
    |  o Can manage agents and access keys    |
    |  o Can view billing                     |
    +------------------------------------------+

    Manager Level:
    +------------------------------------------+
    | customer_manager                         |
    |  o Most customer operations             |
    |  o Cannot access billing                |
    |  o Cannot delete customer               |
    +------------------------------------------+

    Agent Level:
    +------------------------------------------+
    | agent_user                               |
    |  o Access own resources only            |
    |  o Can handle calls/chats               |
    |  o Limited to assigned queues           |
    +------------------------------------------+

**Authorization Check Flow:**

.. code::

    Authorization in API Gateway:

    Request Arrives
         |
         v
    +------------------+
    | Parse Auth Header|
    | (JWT or AccessKey)|
    +--------+---------+
             |
             v
    +------------------+
    | Validate Token   |
    | Extract Claims   |
    +--------+---------+
             |
             v
    +------------------+
    | Get Resource     |
    | from Backend     |
    +--------+---------+
             |
             v
    +------------------+
    | Check Permission:|
    | resource.customer_id
    | == token.customer_id?
    +--------+---------+
             |
        +----+----+
        |         |
        v         v
     Allowed   Forbidden
     (200 OK)  (403)

**CRITICAL: Auth at Gateway Only**

.. code::

    Authentication Boundary:

    External                 API Gateway            Internal Services
       |                         |                        |
       |  With Auth Token        |                        |
       +------------------------>|                        |
       |                         | Validate               |
       |                         | Token                  |
       |                         |                        |
       |                         | RPC (no token)         |
       |                         +----------------------->|
       |                         |                        |
       |                         | Internal services      |
       |                         | TRUST API Gateway      |
       |                         |                        |
       |                         |<-----------------------+
       |                         | Response               |
       |<------------------------+                        |
       |                         |                        |

    IMPORTANT:
    +------------------------------------------------+
    | o JWT validation happens ONLY in api-manager   |
    | o Internal services DO NOT validate tokens     |
    | o Internal services TRUST customer_id from RPC |
    | o This simplifies internal service logic       |
    +------------------------------------------------+

Transport Security
------------------

All communication encrypted:

**External TLS:**

.. code::

    TLS Configuration:

    api.voipbin.net:
    +------------------------------------------+
    | Protocol:     TLS 1.3 (minimum TLS 1.2)  |
    | Cipher:       ECDHE-RSA-AES256-GCM-SHA384|
    | Certificate:  Let's Encrypt (auto-renew)|
    | HSTS:         Enabled (max-age=31536000) |
    +------------------------------------------+

    SIP TLS (sip.voipbin.net:5061):
    +------------------------------------------+
    | Protocol:     TLS 1.2+                   |
    | Certificate:  Let's Encrypt              |
    | Client Auth:  Optional                   |
    +------------------------------------------+

**Internal Encryption:**

.. code::

    Internal Communications:

    Kubernetes Pod-to-Pod:
    +------------------------------------------+
    | Network Policies enforce isolation       |
    | Internal traffic within VPC only         |
    | No TLS required (trusted network)        |
    +------------------------------------------+

    Database Connections:
    +------------------------------------------+
    | Cloud SQL:    SSL required               |
    | Redis:        In-transit encryption      |
    | RabbitMQ:     TLS between nodes          |
    +------------------------------------------+

**SRTP for Media:**

.. code::

    Media Encryption:

    WebRTC Calls:
    +------------------------------------------+
    | Protocol:     SRTP (DTLS-SRTP)           |
    | Key Exchange: DTLS 1.2                   |
    | Cipher:       AES_CM_128_HMAC_SHA1_80    |
    +------------------------------------------+

    SIP TLS Calls:
    +------------------------------------------+
    | Signaling:    SIP over TLS               |
    | Media:        SRTP (negotiated via SDP)  |
    +------------------------------------------+

    PSTN Calls:
    +------------------------------------------+
    | Internal:     SRTP within VoIPBIN        |
    | To Carrier:   Depends on carrier support |
    +------------------------------------------+

Secrets Management
------------------

Kubernetes secrets store sensitive data:

**Secret Types:**

.. code::

    Kubernetes Secrets:

    Database Credentials:
    +------------------------------------------+
    | Secret: db-credentials                   |
    | Keys:                                    |
    |   - dsn: mysql://user:pass@host/db       |
    |   - username: voipbin_app                |
    |   - password: <encrypted>                |
    +------------------------------------------+

    JWT Signing Key:
    +------------------------------------------+
    | Secret: jwt-secret                       |
    | Keys:                                    |
    |   - key: <256-bit random key>            |
    +------------------------------------------+

    API Keys (External Services):
    +------------------------------------------+
    | Secret: external-api-keys                |
    | Keys:                                    |
    |   - deepgram_api_key                     |
    |   - openai_api_key                       |
    |   - twilio_api_key                       |
    +------------------------------------------+

    SSL Certificates:
    +------------------------------------------+
    | Secret: ssl-certs                        |
    | Keys:                                    |
    |   - tls.crt: <certificate>               |
    |   - tls.key: <private key>               |
    +------------------------------------------+

**Secret Injection:**

.. code::

    Pod Secret Configuration:

    spec:
      containers:
      - name: bin-api-manager
        env:
        - name: DSN
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: dsn
        - name: JWT_KEY
          valueFrom:
            secretKeyRef:
              name: jwt-secret
              key: key
        volumeMounts:
        - name: ssl-certs
          mountPath: /etc/ssl/voipbin
          readOnly: true
      volumes:
      - name: ssl-certs
        secret:
          secretName: ssl-certs

**Base64 Encoding:**

.. code::

    CLI Flag Pattern:

    Some services accept base64-encoded secrets via CLI:
    +------------------------------------------+
    | -ssl_cert_base64=<base64-encoded-cert>   |
    | -ssl_private_base64=<base64-encoded-key> |
    +------------------------------------------+

    Why Base64:
    +------------------------------------------+
    | o Allows passing binary data via env vars|
    | o Avoids special character issues        |
    | o Decoded at runtime in application      |
    +------------------------------------------+

Network Security
----------------

VPC and firewall protection:

**Network Isolation:**

.. code::

    Network Segmentation:

    +------------------------------------------------------------------+
    |                     VPC: voipbin-prod                            |
    +------------------------------------------------------------------+
    |                                                                   |
    |  DMZ (Public Subnet):                                             |
    |  +-------------------------------------------------------------+ |
    |  |  Cloud Load Balancer (External IP)                          | |
    |  |  - Only port 443 (HTTPS)                                    | |
    |  |  - Only port 5060/5061 (SIP)                                | |
    |  +-------------------------------------------------------------+ |
    |                              |                                    |
    |                              | Internal Only                      |
    |                              v                                    |
    |  Application Subnet:                                              |
    |  +-------------------------------------------------------------+ |
    |  |  GKE Pods (No external IPs)                                 | |
    |  |  VoIP VMs (Internal IPs)                                    | |
    |  |  - Outbound via NAT Gateway only                            | |
    |  +-------------------------------------------------------------+ |
    |                              |                                    |
    |                              v                                    |
    |  Data Subnet:                                                     |
    |  +-------------------------------------------------------------+ |
    |  |  Cloud SQL (Private IP only)                                | |
    |  |  Memorystore (Private IP only)                              | |
    |  |  RabbitMQ (Private IP only)                                 | |
    |  +-------------------------------------------------------------+ |
    |                                                                   |
    +------------------------------------------------------------------+

**Firewall Rules:**

.. code::

    Cloud Firewall:

    Ingress (Allow):
    +------------------------------------------+
    | Rule: allow-https                        |
    | Source: 0.0.0.0/0                        |
    | Target: Load Balancer                    |
    | Ports: TCP 443                           |
    +------------------------------------------+
    | Rule: allow-sip                          |
    | Source: Carrier IPs (whitelist)          |
    | Target: Kamailio VMs                     |
    | Ports: UDP/TCP 5060, TCP 5061            |
    +------------------------------------------+
    | Rule: allow-rtp                          |
    | Source: 0.0.0.0/0                        |
    | Target: RTPEngine VMs                    |
    | Ports: UDP 10000-60000                   |
    +------------------------------------------+

    Egress (Default Allow):
    +------------------------------------------+
    | All outbound traffic allowed             |
    | NAT Gateway for external access          |
    +------------------------------------------+

    Deny (Default):
    +------------------------------------------+
    | All other ingress denied by default      |
    +------------------------------------------+

**Kubernetes Network Policies:**

.. code::

    Pod Network Policy:

    apiVersion: networking.k8s.io/v1
    kind: NetworkPolicy
    metadata:
      name: api-manager-policy
    spec:
      podSelector:
        matchLabels:
          app: bin-api-manager
      policyTypes:
      - Ingress
      - Egress
      ingress:
      - from:
        - ipBlock:
            cidr: 10.0.0.0/16    # Internal VPC only
        ports:
        - port: 443
        - port: 9000
        - port: 2112
      egress:
      - to:
        - ipBlock:
            cidr: 10.0.0.0/16    # Internal VPC
      - to:
        - ipBlock:
            cidr: 0.0.0.0/0      # External (for webhooks)
        ports:
        - port: 443

Input Validation
----------------

All inputs validated at API boundary:

**Validation Layers:**

.. code::

    Input Validation Stack:

    1. OpenAPI Schema Validation:
    +------------------------------------------+
    | - Required fields present                |
    | - Field types correct (string, int, etc)|
    | - Enum values valid                      |
    | - String length limits                   |
    +------------------------------------------+

    2. Business Logic Validation:
    +------------------------------------------+
    | - Phone number format (+E.164)           |
    | - UUID format                            |
    | - Resource exists                        |
    | - Sufficient balance                     |
    +------------------------------------------+

    3. SQL Injection Prevention:
    +------------------------------------------+
    | - Parameterized queries only             |
    | - No string concatenation for SQL        |
    | - ORM with escaping (Squirrel)           |
    +------------------------------------------+

**Parameterized Query Example:**

.. code::

    Safe Query Pattern:

    // CORRECT - Parameterized
    query := sq.Select("*").
        From("calls").
        Where(sq.Eq{"customer_id": customerID}).
        Where(sq.Eq{"id": callID})

    // Generated SQL:
    // SELECT * FROM calls
    // WHERE customer_id = ? AND id = ?
    // Parameters: [customerID, callID]

    // WRONG - String concatenation (NEVER DO THIS)
    // query := "SELECT * FROM calls WHERE id = '" + callID + "'"

Rate Limiting
-------------

Protect against abuse:

**Rate Limit Configuration:**

.. code::

    Rate Limiting Strategy:

    Global Limits (per customer):
    +------------------------------------------+
    | Endpoint              | Limit            |
    +------------------------------------------+
    | API requests          | 1000/minute      |
    | Call creation         | 100/minute       |
    | SMS sending           | 100/minute       |
    | Login attempts        | 10/minute        |
    +------------------------------------------+

    Burst Handling:
    +------------------------------------------+
    | Token bucket algorithm                   |
    | Bucket size: 2x rate limit               |
    | Refill rate: Rate limit per second       |
    +------------------------------------------+

    Response on Limit:
    +------------------------------------------+
    | Status: 429 Too Many Requests            |
    | Header: Retry-After: 60                  |
    | Body: {"error": "rate_limit_exceeded"}   |
    +------------------------------------------+

**DDoS Protection:**

.. code::

    Cloud Armor Configuration:

    WAF Rules:
    +------------------------------------------+
    | Rule: block-known-attackers              |
    | - Block IPs from threat intelligence     |
    +------------------------------------------+
    | Rule: rate-limit-by-ip                   |
    | - 10,000 requests/minute per IP          |
    +------------------------------------------+
    | Rule: geo-restrict (optional)            |
    | - Allow specific countries only          |
    +------------------------------------------+

    Adaptive Protection:
    +------------------------------------------+
    | - ML-based attack detection              |
    | - Automatic rule suggestions             |
    | - Alert on anomalies                     |
    +------------------------------------------+

Audit Logging
-------------

Complete audit trail:

**Logged Events:**

.. code::

    Audit Log Events:

    Authentication:
    +------------------------------------------+
    | o Login success/failure                  |
    | o Logout                                 |
    | o Token refresh                          |
    | o Access key creation/revocation         |
    +------------------------------------------+

    Resource Operations:
    +------------------------------------------+
    | o Create (who, what, when)               |
    | o Update (who, what, old, new, when)     |
    | o Delete (who, what, when)               |
    +------------------------------------------+

    Security Events:
    +------------------------------------------+
    | o Permission denied attempts             |
    | o Rate limit exceeded                    |
    | o Invalid token attempts                 |
    | o Suspicious activity patterns           |
    +------------------------------------------+

**Log Format:**

.. code::

    Audit Log Entry:

    {
      "timestamp": "2026-01-20T12:00:00.000Z",
      "event_type": "resource_created",
      "customer_id": "uuid",
      "agent_id": "uuid",
      "resource_type": "call",
      "resource_id": "uuid",
      "action": "create",
      "source_ip": "192.168.1.100",
      "user_agent": "VoIPBIN-SDK/1.0",
      "request_id": "uuid",
      "details": {
        "source": "+15551234567",
        "destination": "+15559876543"
      }
    }

Data Protection
---------------

Protecting sensitive data:

**Data Classification:**

.. code::

    Data Sensitivity Levels:

    Public:
    +------------------------------------------+
    | o API documentation                      |
    | o Service status                         |
    +------------------------------------------+

    Internal:
    +------------------------------------------+
    | o Call metadata (IDs, timestamps)        |
    | o Flow definitions                       |
    | o Configuration                          |
    +------------------------------------------+

    Confidential:
    +------------------------------------------+
    | o Customer PII (names, emails)           |
    | o Phone numbers                          |
    | o Call recordings                        |
    | o Chat transcripts                       |
    +------------------------------------------+

    Restricted:
    +------------------------------------------+
    | o Passwords (hashed, never stored plain) |
    | o API keys                               |
    | o JWT signing keys                       |
    | o Database credentials                   |
    +------------------------------------------+

**Encryption at Rest:**

.. code::

    Data Encryption:

    Cloud SQL:
    +------------------------------------------+
    | Encryption: AES-256                      |
    | Key Management: Google-managed           |
    | Automatic encryption of all data         |
    +------------------------------------------+

    Cloud Storage (Recordings):
    +------------------------------------------+
    | Encryption: AES-256                      |
    | Key Management: Customer-managed (CMEK)  |
    | Per-object encryption                    |
    +------------------------------------------+

    Redis (Memorystore):
    +------------------------------------------+
    | Encryption: AES-256                      |
    | In-transit encryption enabled            |
    +------------------------------------------+

**Data Retention:**

.. code::

    Retention Policies:

    +------------------------------------------+
    | Data Type        | Retention | Deletion  |
    +------------------------------------------+
    | Call records     | 2 years   | Soft      |
    | Call recordings  | 90 days   | Hard      |
    | Chat messages    | 1 year    | Soft      |
    | Audit logs       | 7 years   | Hard      |
    | Session tokens   | 1 hour    | Automatic |
    +------------------------------------------+

    Soft Delete:
    +------------------------------------------+
    | tm_delete set to deletion time           |
    | Data remains in DB but not returned      |
    | Can be restored if needed                |
    +------------------------------------------+

Compliance
----------

Security standards adherence:

**Security Standards:**

.. code::

    Compliance Framework:

    SOC 2 Type II:
    +------------------------------------------+
    | o Security controls documented           |
    | o Annual audit                           |
    | o Continuous monitoring                  |
    +------------------------------------------+

    GDPR:
    +------------------------------------------+
    | o Data subject rights supported          |
    | o Data portability APIs                  |
    | o Right to deletion implemented          |
    | o EU data residency option               |
    +------------------------------------------+

    HIPAA (Optional):
    +------------------------------------------+
    | o BAA available for healthcare customers |
    | o PHI handling procedures                |
    | o Audit controls                         |
    +------------------------------------------+

    PCI DSS:
    +------------------------------------------+
    | o No credit card data stored             |
    | o Payment via Stripe (PCI compliant)     |
    +------------------------------------------+

Security Best Practices
-----------------------

Development and operations security:

**Development:**

.. code::

    Secure Development:

    Code Review:
    +------------------------------------------+
    | o All changes peer-reviewed              |
    | o Security checklist for PRs             |
    | o Automated security scanning (SAST)     |
    +------------------------------------------+

    Dependency Management:
    +------------------------------------------+
    | o Regular dependency updates             |
    | o Vulnerability scanning (Snyk/Dependabot)|
    | o No known vulnerable dependencies       |
    +------------------------------------------+

    Secret Handling:
    +------------------------------------------+
    | o No secrets in code or git              |
    | o Environment variables for config       |
    | o Secret rotation procedures             |
    +------------------------------------------+

**Operations:**

.. code::

    Security Operations:

    Access Control:
    +------------------------------------------+
    | o Least privilege principle              |
    | o MFA for all admin access               |
    | o Regular access reviews                 |
    +------------------------------------------+

    Incident Response:
    +------------------------------------------+
    | o Documented incident procedures         |
    | o On-call rotation                       |
    | o Post-incident reviews                  |
    +------------------------------------------+

    Monitoring:
    +------------------------------------------+
    | o Real-time security alerts              |
    | o Failed login monitoring                |
    | o Anomaly detection                      |
    +------------------------------------------+

