# Voice & Real-Time Documentation Improvement Design

**Date:** 2026-02-01
**Status:** Draft
**Location:** `bin-api-manager/docsdev/source/`

## Overview

This design improves the six Voice & Real-Time documentation sections in the VoIPBIN developer documentation. The improvements are proportional based on current quality gaps, targeting a balanced audience of developers (practical examples) and architects (conceptual understanding).

### Goals

1. **Fill gaps** - Bring weaker sections up to the quality level of stronger ones
2. **Add practical examples** - More tutorials, real-world scenarios, working code
3. **Improve architecture clarity** - ASCII diagrams, sequence flows, system interactions

### Sections by Improvement Size

| Section | Current State | Change Type |
|---------|---------------|-------------|
| Recording | Very thin - just API descriptions | Major rewrite |
| Transcribe | Decent but missing architecture | Significant expansion |
| Mediastream | Basic overview, good tutorial | Significant expansion |
| Call | Comprehensive | Polish & enhance |
| Conference | Good | Polish & enhance |
| Queue | Good | Polish & enhance |

---

## Design Principle: ASCII Diagrams

All new diagrams will use ASCII art for:
- Consistent rendering in Sphinx, terminals, code editors
- Version control friendly (readable diffs)
- No external image dependencies
- Matching existing style in Call/Conference/Queue

**Diagram style:**
```
+-------------------+     Boxes for components
|    Component      |
+--------+----------+
         |              Lines for connections
         v
+-------------------+     Arrows for direction
|  Next Component   |
+-------------------+

State A -----> State B     Arrows for state transitions

    +-----+-----+          Branching for decisions
    |           |
    v           v
  Yes          No
```

---

## Section 1: Recording (Major Rewrite)

### Current Problems

- Only describes API capabilities (list, get, delete, download, export)
- No explanation of how recording actually works
- No lifecycle or state information
- No architecture diagrams
- No best practices or troubleshooting

### New Structure

```
recording_overview.rst (rewrite)
├── How Recording Works
│   ├── Architecture diagram
│   ├── Recording types (call vs conference)
│   └── Storage and formats
│
├── Recording Lifecycle
│   ├── State diagram
│   ├── State descriptions table
│   └── What happens at each stage
│
├── Starting and Stopping Recordings
│   ├── Via Flow Action
│   ├── Via API
│   └── Automatic recording
│
├── Recording Storage
│   ├── File formats supported
│   ├── Storage duration and retention
│   ├── Download vs streaming access
│   └── Bulk export capabilities
│
├── Common Scenarios
│   ├── Record entire call
│   ├── Record only after consent
│   ├── Record conference
│   └── Pause and resume
│
├── Best Practices
│   ├── Legal compliance
│   ├── Storage management
│   ├── Naming conventions
│   └── Retention policies
│
└── Troubleshooting
    ├── Recording not starting
    ├── Empty or corrupted files
    └── Download failures
```

### New ASCII Diagrams

**1. Recording Architecture**
```
+--------+        +----------------+        +---------+
|  Call  |--audio-->| Recording    |--file-->| Storage |
+--------+        |    Engine      |        | (GCS)   |
                  +----------------+        +---------+
+------------+           |
| Conference |--audio----+
+------------+
```

**2. Recording Lifecycle State Machine**
```
POST /recording_start
       |
       v
+------------+     recording     +------------+
|  starting  |------------------>|  recording |
+------------+                   +-----+------+
                                       |
                      POST /recording_stop or hangup
                                       |
                                       v
                                +------------+
                                |  stopped   |
                                +-----+------+
                                      |
                                      v (file processing)
                                +------------+
                                | available  |
                                +------------+
```

**3. Recording Start Sequence**
```
Your App              VoIPBIN               Storage
   |                     |                     |
   | POST recording_start|                     |
   +-------------------->|                     |
   |                     | Open audio stream   |
   |                     +-------------------->|
   |  recording_id       |                     |
   |<--------------------+                     |
   |                     |====audio write====>|
   |                     |                     |
```

**4. Download vs Stream Access**
```
Download:                          Stream:
+------+  GET /download  +-----+   +------+  GET /stream  +-----+
| Your |---------------->|File |   | Your |<- - - - - - ->|File |
| App  |<--full file-----+-----+   | App  |   chunks      +-----+
+------+                           +------+
```

---

## Section 2: Transcribe (Significant Expansion)

### Current Problems

- Language list dominates the page (70% of content)
- Missing architecture explanation for real-time flow
- No sequence diagrams for transcript delivery
- Limited integration examples
- No troubleshooting

### New Structure

```
transcribe_overview.rst (expand and reorganize)
├── How Transcription Works
│   ├── Architecture diagram
│   ├── Real-time vs post-call
│   └── Direction detection explained
│
├── Transcription Lifecycle
│   ├── State diagram
│   ├── Transcript delivery flow
│   └── Relationship to calls/conferences
│
├── Starting Transcription
│   ├── Via Flow Action (enhanced)
│   ├── Via API (enhanced)
│   └── When to use each method
│
├── Receiving Transcripts
│   ├── Webhook delivery
│   │   ├── Sequence diagram
│   │   └── Payload structure
│   ├── WebSocket subscription
│   │   ├── How to subscribe
│   │   └── Real-time event handling
│   └── Comparison table
│
├── Working with Transcripts
│   ├── Direction field explained
│   ├── Timestamp interpretation
│   ├── Combining into conversation
│   └── Storing and searching
│
├── Supported Languages (condensed)
│   ├── Common languages in main doc
│   └── Full table in collapsible/reference
│
├── Common Scenarios
│   ├── Real-time call transcription
│   ├── Conference with speaker ID
│   ├── Post-call from recording
│   └── Multi-language support
│
├── Best Practices
│   ├── Language selection
│   ├── Handling background noise
│   ├── High-volume processing
│   └── Storage and compliance
│
└── Troubleshooting
    ├── Transcription not starting
    ├── Poor accuracy
    ├── Missing transcripts
    └── Webhook delivery issues
```

### New ASCII Diagrams

**1. Transcription Architecture**
```
+--------+        +----------------+        +------------+
|  Call  |--audio-->|   STT        |--text-->|  Webhook   |
+--------+        |   Engine       |        |     or     |
                  +----------------+        | WebSocket  |
+------------+           |                  +------------+
| Conference |--audio----+                        |
+------------+                                    v
                                           +------------+
                                           |  Your App  |
                                           +------------+
```

**2. Real-Time Transcript Delivery**
```
Call Audio          VoIPBIN STT           Your App
    |                    |                    |
    |====audio chunk====>|                    |
    |                    | process            |
    |                    |----+               |
    |                    |<---+               |
    |                    |                    |
    |                    | transcript_created |
    |                    +------------------->|
    |                    |                    |
    |====audio chunk====>|                    |
    |                    | process            |
    |                    +------------------->|
    |                    |                    |
```

**3. Direction Detection**
```
+----------+                             +---------+
|  Caller  |-----> direction: "in" ----->| VoIPBIN |
|          |<---- direction: "out" <-----|         |
+----------+                             +---------+

Transcript output:
[in]  "Hello, I need help with my account"
[out] "Sure, I can help you with that"
[in]  "My account number is 12345"
```

**4. Webhook vs WebSocket Delivery**
```
Webhook (push):                    WebSocket (subscribe):
+-------+                          +-------+
|VoIPBIN|---POST /your-endpoint--->| Your  |
+-------+      transcript          |  App  |
                                   +---+---+
                                       |
+-------+                          +---+---+
|VoIPBIN|<====== websocket =======>| Your  |
+-------+    subscribe/events      |  App  |
                                   +-------+
```

---

## Section 3: Mediastream (Significant Expansion)

### Current Problems

- Overview is basic compared to good tutorial
- Missing architecture diagram
- No decision guide for encapsulation types
- Overview and tutorial have duplicate content

### New Structure

```
mediastream_overview.rst (expand)
├── What is Media Streaming?
│   ├── Concept explanation
│   ├── Architecture diagram
│   └── Comparison to traditional call handling
│
├── Streaming Modes
│   ├── Bi-directional streaming
│   │   ├── Diagram
│   │   ├── Use cases
│   │   └── API endpoint
│   ├── Uni-directional streaming
│   │   ├── Diagram
│   │   ├── Use cases
│   │   └── Flow action method
│   └── Comparison table
│
├── Encapsulation Types (enhanced)
│   ├── Decision flowchart
│   ├── RTP - detailed
│   │   ├── Packet structure
│   │   ├── Pros/cons
│   │   └── Best for
│   ├── SLN - detailed
│   │   ├── Format
│   │   ├── Pros/cons
│   │   └── Best for
│   ├── AudioSocket - detailed
│   │   ├── Protocol details
│   │   ├── Pros/cons
│   │   └── Best for
│   └── Comparison table
│
├── Audio Format Reference
│   ├── Codec specifications
│   ├── Sample rate, bit depth
│   └── Chunk size recommendations
│
├── Supported Resources
│   ├── Call media streaming
│   ├── Conference media streaming
│   └── Differences
│
├── Integration Patterns
│   ├── Real-time speech recognition
│   ├── AI voice assistant
│   ├── Custom recording/analysis
│   ├── Audio injection
│   └── Links to tutorial
│
└── Connection Lifecycle
    ├── State diagram
    ├── Error handling
    └── Reconnection strategies
```

### New ASCII Diagrams

**1. Media Stream Architecture**
```
+--------+                              +----------+
|  Call  |<====== WebSocket ==========>| Your App |
+--------+      bi-directional          +----------+
                audio stream

Traditional:                    Media Stream:
+------+  SIP  +-------+        +------+  WS   +-------+
| Phone|<----->|VoIPBIN|        | Call |<====>| Your  |
+------+       +-------+        +------+      |  App  |
                                              +-------+
                                              Direct audio access!
```

**2. Bi-directional vs Uni-directional**
```
Bi-directional:                 Uni-directional:
+-------+                       +-------+
|VoIPBIN|<==== audio in ====   |VoIPBIN|
|       |==== audio out ===>   |       |==== audio out ===>
+-------+        +-------+     +-------+        +-------+
                 | Your  |                      | Your  |
                 |  App  |                      |  App  |
                 +-------+                      +-------+
You send AND receive            You only receive (or only send)
```

**3. Encapsulation Type Decision**
```
                Need audio streaming?
                       |
          +------------+------------+
          |                         |
     Standard VoIP?            Minimal overhead?
          |                         |
    +-----+-----+             +-----+-----+
    |           |             |           |
   Yes         No            Yes         No
    |           |             |           |
    v           |             v           |
  [RTP]         |           [SLN]         |
                |                         |
           Asterisk?                      |
                |                         |
          +-----+-----+                   |
          |           |                   |
         Yes         No                   |
          |           |                   |
          v           +-------------------+
    [AudioSocket]              |
                               v
                          [RTP default]
```

**4. WebSocket Connection Lifecycle**
```
Your App                    VoIPBIN
   |                           |
   | GET /calls/{id}/media_stream
   +-------------------------->|
   |                           |
   | 101 Switching Protocols   |
   |<--------------------------+
   |                           |
   |<====== audio frames =====>|  (bidirectional)
   |<====== audio frames =====>|
   |<====== audio frames =====>|
   |                           |
   | close()                   |
   +-------------------------->|
   |                           |
```

---

## Section 4: Call (Polish & Enhance)

### Current State

Already comprehensive with state diagrams, lifecycle, chaining, timestamps, failover.

### Enhancements

```
call_overview.rst (polish)
├── Add: Quick Reference Card
│   ├── Status values at a glance
│   ├── Hangup reasons cheat sheet
│   ├── Common API endpoints table
│   └── Key timestamps one-liners
│
├── Add: Cross-References Section
│   ├── Recording a call → Recording
│   ├── Transcribing a call → Transcribe
│   ├── Streaming call audio → Mediastream
│   └── Queuing a call → Queue
│
├── Enhance: Call Chaining
│   ├── 3+ party scenario diagram
│   └── Chaining vs conference decision guide
│
└── Minor fixes
    ├── Consistent terminology
    └── Anchor links for deep linking
```

### New ASCII Diagrams

**Quick Reference Status Flow**
```
dialing ---> ringing ---> progressing ---> hangup
   |            |              |
   +-----> canceling ----+    |
   |                     |    |
   +--------> hangup <---+----+
             (failed)
```

**Chaining with 3+ Parties**
```
+--------+       +---------+       +---------+
| Caller |<----->| VoIPBIN |<----->| Agent 1 |
+--------+       +----+----+       +---------+
                     |
                     +------------>+---------+
                     |             | Agent 2 |
                     |             +---------+
                     |
                     +------------>+------------+
                                   | Supervisor |
                                   +------------+

Master: Caller's call
Chained: Agent 1, Agent 2, Supervisor
```

---

## Section 5: Conference (Polish & Enhance)

### Current State

Good architecture, lifecycle, recording/transcription sections.

### Enhancements

```
conference_overview.rst (polish)
├── Add: Quick Reference Card
│   ├── Conference status values
│   ├── Participant status values
│   ├── Common API endpoints table
│   └── Type comparison
│
├── Add: Cross-References Section
│   ├── Conference recording → Recording
│   ├── Conference transcription → Transcribe
│   ├── Conference media streaming → Mediastream
│   └── Call joins conference → Call
│
├── Enhance: Feature Integration
│   ├── Recording + Transcription together
│   └── Combined scenarios
│
└── Minor fixes
    ├── Note "leaved" is intentional API term
    └── Anchor links
```

### New ASCII Diagrams

**Quick Reference Status Flow**
```
starting ---> progressing ---> terminating ---> terminated
                   ^                |
                   |                |
                   +-- still has ---+
                       participants
```

**Recording + Transcription Together**
```
+-------------------------------------------------------+
|                    Conference                          |
|  +------+  +------+  +------+                         |
|  |User A|  |User B|  |User C|                         |
|  +--+---+  +--+---+  +--+---+                         |
|     |         |         |                             |
|     +----+----+----+----+                             |
|          |         |                                  |
|          v         v                                  |
|    +----------+  +-------------+                      |
|    |Recording |  |Transcription|                      |
|    |  File    |  |   Stream    |                      |
|    +----------+  +------+------+                      |
+-------------------------------------------------------+
                          |
                          v
                   +-----------+
                   | Your App  |
                   | (webhook) |
                   +-----------+
```

---

## Section 6: Queue (Polish & Enhance)

### Current State

Good agent matching, timeout handling, metrics.

### Enhancements

```
queue_overview.rst (polish)
├── Add: Quick Reference Card
│   ├── Queuecall status values
│   ├── Agent status values
│   ├── Key timeout fields
│   └── Common API endpoints
│
├── Add: Cross-References Section
│   ├── Queue uses conference bridge → Conference
│   ├── Recording queued calls → Recording
│   ├── Agent management → Agent
│   └── Queue flow actions → Flow
│
├── Enhance: Queue + Agent Integration
│   ├── Agent status transitions diagram
│   └── Multi-queue agent scenarios
│
└── Minor fixes
    ├── Example tag structures
    └── Anchor links
```

### New ASCII Diagrams

**Quick Reference Status Flow**
```
initiating --> waiting --> connecting --> service --> done
                  |            |            |
                  v            v            v
              abandoned    abandoned    abandoned
```

**Agent Status Transitions**
```
+----------+                     +----------+
|  offline |------ login ------->| available|
+----------+                     +-----+----+
     ^                                 |
     |                           receive call
     |                                 |
     |                                 v
     |                           +----------+
     +-------- logout -----------|   busy   |
     |                           +-----+----+
     |                                 |
     |                            call ends
     |                                 |
     |                                 v
     |                           +----------+
     +-------- logout -----------| wrap-up  |
                                 +-----+----+
                                       |
                                  complete wrap-up
                                       |
                                       v
                                 +----------+
                                 | available|
                                 +----------+
```

---

## Summary

### Total Changes

| Section | Files Modified | New Diagrams | Estimated Lines |
|---------|----------------|--------------|-----------------|
| Recording | 1 (rewrite) | 4 | ~400 |
| Transcribe | 1 (expand) | 4 | ~300 |
| Mediastream | 1 (expand) | 4 | ~250 |
| Call | 1 (enhance) | 2 | ~100 |
| Conference | 1 (enhance) | 2 | ~100 |
| Queue | 1 (enhance) | 2 | ~100 |
| **Total** | **6 files** | **18 diagrams** | **~1250 lines** |

### Implementation Order

1. **Recording** - Largest gap, sets quality standard
2. **Transcribe** - Build on Recording patterns
3. **Mediastream** - Complete the major rewrites
4. **Call** - Quick wins with cross-references
5. **Conference** - Quick wins with cross-references
6. **Queue** - Complete the polish phase

### Success Criteria

- [ ] All sections have architecture diagrams
- [ ] All sections have lifecycle/state diagrams
- [ ] All sections have cross-references to related sections
- [ ] All sections have quick reference cards
- [ ] All sections have troubleshooting (where applicable)
- [ ] All sections have best practices (where applicable)
- [ ] Documentation builds without warnings
- [ ] ASCII diagrams render correctly in HTML output
