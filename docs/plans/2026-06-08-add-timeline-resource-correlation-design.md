# Timeline Resource Correlation (internal RPC)

Status: Approved (v2.1, 2 review rounds)
Service: bin-timeline-manager
Date: 2026-06-08

## 1. Problem Statement

운영/디버깅 중 특정 리소스(예: Call) 하나의 ID만 가지고 있을 때, 그 통화 흐름에 함께 엮인 모든 리소스(Activeflow, AICall, Message, Recording, Transcribe, ConferenceCall, QueueCall 등)의 ID를 한 번에 파악할 방법이 현재 없다.

오늘날 가능한 것은 두 가지로 분절되어 있다.

- `EventList` (RPC): `publisher` + `resource_id` + `events` 필터로 한 리소스의 이벤트 로그만 조회.
- `AggregatedList` (RPC): `activeflow_id`를 **이미 알고 있어야** 그 activeflow에 엮인 이벤트를 평면 리스트로 조회.

즉 "리소스 ID 하나 → 엮인 모든 리소스 ID" 라는 단일 진입 흐름이 없다. 운영자는 직접 Call을 조회해 activeflow_id를 추출하고, 다시 AggregatedList를 호출하고, 결과 이벤트를 수작업으로 묶어야 한다.

이 설계는 `id` 하나만 입력받아 그 리소스가 속한 activeflow의 엮인 리소스들을 type별로 묶어 반환하는 **내부 RPC 메서드**를 추가한다.

## 2. Scope

### In scope (Phase 1)

- bin-timeline-manager에 신규 RPC listener 메서드 1개 추가.
- 입력: 리소스 `id` (UUID) 하나. type 불필요.
- 동작: ClickHouse 2단계 쿼리.
  1. `id`(resource_id) → activeflow_id 역추출.
  2. activeflow_id로 엮인 리소스를 publisher 그룹핑 + resource_id dedupe 집계.
- 출력: ID 그래프 (publisher별 묶음, resource_id dedupe, event_types 집합, first_seen/last_seen).
- 조회 성능을 위한 ClickHouse bloom_filter skip index 2개 추가 (`resource_id`, `activeflow_id`) + 과거 파트 백필.

### Out of scope (Phase 1)

- 외부 REST 엔드포인트 / OpenAPI 스펙 (api-manager 경유). 내부 RPC만.
- LLM tool 노출 (bin-ai-manager toolhandler 등록).
- 각 매니저로의 fan-out을 통한 리소스 본문(full object) 채우기. ID 그래프만 반환.
- message-manager(SMS), transfer-manager 리소스. (3절 한계 참조)

### Rationale

- 신규 manager를 만들지 않는다. 적재 파이프라인(27큐 구독 → ClickHouse, activeflow_id/resource_id MATERIALIZED 추출)이 timeline-manager에 이미 존재하므로 read 경로만 추가한다.
- 1차는 내부 RPC로 한정하여 blast radius를 최소화한다. 외부 노출/LLM tool화는 후속 페이즈에서 별도 결정.

## 3. Known Limitation: message / transfer 미지원

correlation은 ClickHouse `events.activeflow_id` MATERIALIZED 컬럼에 의존한다. 이 컬럼은 이벤트 payload(`data`)에서 추출된다.

```sql
-- migrations/000003_add_activeflow_id_column.up.sql
activeflow_id String MATERIALIZED
  if(event_type LIKE 'activeflow_%',
     JSONExtractString(data, 'id'),
     JSONExtractString(data, 'activeflow_id'))
```

코드 확인 결과 (2026-06-08):

| 리소스 | activeflow_id 보유 | correlation 포착 |
|---|---|---|
| call, recording, confbridge (call-manager) | O | O |
| aicall, message(AI 대화), summary (ai-manager) | O | O |
| transcribe (transcribe-manager) | O | O |
| conferencecall (conference-manager) | O | O |
| queuecall (queue-manager) | O | O |
| **message (message-manager, SMS)** | **X (모델/payload에 없음)** | **X** |
| **transfer (transfer-manager)** | **X (모델/payload에 없음)** | **X** |

SMS와 transfer는 activeflow 없이도 발생할 수 있어 구조적으로 빠진다. Phase 1에서 명시적으로 제외하고, 응답에 잡히지 않음을 문서화한다. 향후 해당 매니저가 이벤트 payload에 activeflow_id를 싣게 되면 코드 변경 없이 자동 포착된다 (MATERIALIZED 특성).

## 4. Domain Model

신규 DB 엔티티 없음. ClickHouse `events` 테이블만 읽는다. 응답 전용 구조체만 정의한다.

### 응답 구조체 (response 패키지)

```go
// V1DataResourceCorrelationGet represents a correlation graph for a resource.
type V1DataResourceCorrelationGet struct {
    ResourceID    uuid.UUID                 `json:"resource_id"`
    ResourceFound bool                      `json:"resource_found"` // 입력 resource_id 이벤트 존재 여부
    ActiveflowID  uuid.UUID                 `json:"activeflow_id"`  // 못 찾으면 uuid.Nil
    Truncated     bool                      `json:"truncated"`      // LIMIT 초과로 잘렸는지
    Resources     []*PublisherGroup         `json:"resources"`      // publisher별 그룹 (정렬된 slice)
}

// PublisherGroup groups correlated resources by publisher (service name).
type PublisherGroup struct {
    Publisher string                `json:"publisher"` // 예: "call-manager"
    Resources []*CorrelatedResource `json:"resources"`
}

// CorrelatedResource is one deduplicated resource derived from events.
type CorrelatedResource struct {
    ID         uuid.UUID `json:"id"`          // resource_id
    DataType   string    `json:"data_type"`   // resource kind label
    EventTypes []string  `json:"event_types"` // distinct event types (sorted, stable)
    FirstSeen  time.Time `json:"first_seen"`  // min(timestamp)
    LastSeen   time.Time `json:"last_seen"`   // max(timestamp)
}
```

설계 결정 (1차 리뷰 반영):
- **map 대신 정렬된 slice.** `map[publisher][]`는 Go map 순회가 무순서라 ClickHouse `ORDER BY`를 무효화한다. `[]*PublisherGroup`으로 바꿔 publisher는 알파벳순, 그룹 내 리소스는 `first_seen ASC`로 안정 정렬한다.
- **그룹핑 키 = publisher** (service name 그대로). 별도 type 매핑 테이블을 두지 않아 유지보수 지점을 만들지 않는다. (문서 전반의 "type별 묶음"은 publisher 단위 묶음을 의미한다.)
- **`ResourceFound`**: 입력 resource_id의 이벤트가 하나도 없으면 false. "리소스 자체가 없음"과 "리소스는 있으나 activeflow 없음"을 구분한다 (1차 리뷰 #18).
- **`Truncated`**: LIMIT 초과로 결과가 잘렸음을 호출측에 알린다 (1차 리뷰 #13).
- **`EventTypes`는 API 경계에서 정렬**하여 호출 간 순서를 안정화한다 (`groupUniqArray`는 무순서, #11).
- **한 리소스가 여러 publisher로 등장 가능** (#8): `GROUP BY publisher, resource_id`이므로 같은 resource_id가 서로 다른 publisher 그룹에 중복 등장할 수 있다. 이는 의도된 동작으로, 동일 ID를 여러 서비스가 이벤트로 발행한 경우를 그대로 노출한다. 호출측은 publisher 컨텍스트로 구분한다.

## 5. Database (ClickHouse) Schema Change

신규 테이블 없음. 기존 `events` 테이블에 조회 성능용 인덱스를 추가한다. 현재 `ORDER BY (event_type, timestamp)` 이므로 `resource_id` 및 `activeflow_id` 단독 조회는 풀스캔이 된다.

### 1차 리뷰 반영: projection 폐기, data-skipping index 채택

v1은 `ADD PROJECTION (SELECT * ORDER BY ...)` 2개를 제안했으나 폐기한다. 이유:
- `SELECT *` projection은 테이블 전체를 정렬 복사하므로 base + 복사본 2개 = **스토리지 약 3배** (1년 TTL 고려 시 비용 과다, 리뷰 #1).
- 본 기능의 두 쿼리는 모두 **point lookup**(특정 resource_id/activeflow_id 등치 필터)이다. 이 패턴에는 full projection이 아니라 **bloom_filter data-skipping index**가 정확한 도구다 (리뷰 #2). 인덱스는 파트별 granule을 건너뛰게 해주며 스토리지 오버헤드가 작다.

### Migration 000004: skip index 2개 + 백필

```sql
-- up
-- 0) (선행) 과거 파트 materialized 컬럼 백필. 000003 이전 INSERT 된 파트는
--    resource_id / activeflow_id 가 비어 있을 수 있다 (리뷰 #6, Q5).
ALTER TABLE events MATERIALIZE COLUMN resource_id;
ALTER TABLE events MATERIALIZE COLUMN activeflow_id;

-- 1) skip index 추가
ALTER TABLE events ADD INDEX IF NOT EXISTS idx_resource_id   resource_id   TYPE bloom_filter GRANULARITY 1;
ALTER TABLE events ADD INDEX IF NOT EXISTS idx_activeflow_id activeflow_id TYPE bloom_filter GRANULARITY 1;

-- 2) 기존 파트에 인덱스 구체화
ALTER TABLE events MATERIALIZE INDEX idx_resource_id;
ALTER TABLE events MATERIALIZE INDEX idx_activeflow_id;

-- down
ALTER TABLE events DROP INDEX IF EXISTS idx_resource_id;
ALTER TABLE events DROP INDEX IF EXISTS idx_activeflow_id;
```

### 마이그레이션 실행 계획 (리뷰 #4, #6, Q5/Q6)

- `MATERIALIZE COLUMN` / `MATERIALIZE INDEX`는 기존 파트를 백그라운드 머지로 재작성한다. 무거운 I/O이며 대용량 테이블에서 장시간 소요된다.
- 적용 전 운영 `events` 테이블 크기(파트 수, 디스크 사용량)를 `system.parts`로 측정하고 소요 시간을 추정한다.
- 가능하면 트래픽이 낮은 시간대에 실행한다. 진행 상황은 `system.mutations`(`is_done`, `parts_to_do`)로 모니터링한다.
- 적용 완료 전에는 신규 INSERT 파트만 인덱스/백필이 반영된다 (조회 정확성은 유지, 성능만 점진 개선).
- 롤백: `DROP INDEX` (즉시). `MATERIALIZE COLUMN`은 되돌릴 필요가 없다 (값 자체는 정상 추출이며 down에서 무동작).
- bloom_filter 인덱스가 `MATERIALIZED` 컬럼 위에서 동작하는지 대상 ClickHouse 버전에서 사전 검증한다 (리뷰 #5). 검증 실패 시 대안: `data` JSON 직접 `JSONExtractString` 필터 + `minmax` 인덱스, 또는 Q2 한정 aggregating projection.

### 선택 대안 (Q2 고빈도화 시): aggregating projection

correlation 조회가 고빈도가 되면 Q2 집계 전용 projection을 검토한다. 단 `SELECT *`가 아니라 **필요 컬럼만** 집계한다 (리뷰 #3).
```sql
ALTER TABLE events ADD PROJECTION proj_correlation (
    SELECT publisher, resource_id, any(data_type), min(timestamp), max(timestamp), groupUniqArray(event_type)
    GROUP BY activeflow_id, publisher, resource_id
);
```
Phase 1에서는 적용하지 않는다 (진단 빈도면 skip index로 충분).

## 6. Handler Interface

### dbhandler (신규 메서드 3개)

```go
// ResourceActiveflowIDGet returns the activeflow_id for a given resource_id.
// Returns "" (no error) when no matching event with a non-empty activeflow_id exists.
func (h *dbHandler) ResourceActiveflowIDGet(ctx context.Context, resourceID string) (string, error)

// ResourceExists reports whether any event row exists for the resource_id.
// Used to distinguish "resource never seen" from "resource has no activeflow".
func (h *dbHandler) ResourceExists(ctx context.Context, resourceID string) (bool, error)

// CorrelatedResourceList returns deduplicated resources grouped by (publisher, resource_id)
// for a given activeflow_id, aggregated at the ClickHouse layer. limit caps the row count.
func (h *dbHandler) CorrelatedResourceList(ctx context.Context, activeflowID string, limit int) ([]*event.CorrelatedRow, error)
```

`event.CorrelatedRow`는 집계 쿼리 행 스캔용 내부 구조체. ClickHouse 드라이버 제약(LowCardinality(String)을 커스텀 타입으로 스캔 불가)에 따라 `Publisher`, `DataType`을 **string**으로 스캔하고, resource_id/activeflow_id도 string으로 받아 API 경계에서 uuid 변환한다 (리뷰 #17).

### eventhandler (신규 메서드 1개)

```go
// ResourceCorrelationGet resolves a resource id to its activeflow and returns
// the correlation graph of all resources sharing that activeflow.
func (h *eventHandler) ResourceCorrelationGet(ctx context.Context, resourceID uuid.UUID) (*response.V1DataResourceCorrelationGet, error)
```

### 내부 흐름 (pseudocode)

```
ResourceCorrelationGet(ctx, resourceID):
    if resourceID == uuid.Nil:
        return error("resource_id is required")

    # 1. resource_id -> activeflow_id 역추출 (결정적)
    activeflowID = db.ResourceActiveflowIDGet(ctx, resourceID.String())
    if activeflowID == "":
        # 이벤트가 없거나(미존재) activeflow 미보유.
        # 빈 그래프 200 반환, ResourceFound 로 구분 (Q2).
        found = db.ResourceExists(ctx, resourceID.String())
        return &V1DataResourceCorrelationGet{
            ResourceID: resourceID, ResourceFound: found,
            ActiveflowID: uuid.Nil, Resources: [],
        }

    # 2. activeflow_id -> 집계된 리소스 목록 (LIMIT+1 로 truncation 감지)
    rows = db.CorrelatedResourceList(ctx, activeflowID, maxResources+1)
    truncated = len(rows) > maxResources
    if truncated: rows = rows[:maxResources]

    # 3. publisher 그룹핑 + 안정 정렬
    groups = groupByPublisher(rows)        # publisher 알파벳순
    for g in groups: sort g.Resources by FirstSeen ASC
    for r in rows: sort r.EventTypes       # 안정 정렬

    return &V1DataResourceCorrelationGet{
        ResourceID: resourceID, ResourceFound: true,
        ActiveflowID: parse(activeflowID), Truncated: truncated,
        Resources: groups,
    }
```

uuid.Parse 실패 처리 (리뷰 #16): 행 단위로 resource_id/activeflow_id 파싱이 실패하면 해당 행을 skip + warn 로그. 한 개 불량 payload가 전체 호출을 500으로 만들지 않는다.

### ClickHouse 쿼리

**1단계: 역추출 (결정적, 리뷰 #7)**
```sql
SELECT activeflow_id
FROM events
WHERE resource_id = ?
  AND activeflow_id != ''
ORDER BY timestamp ASC
LIMIT 1
SETTINGS max_execution_time = 5
```
`ORDER BY timestamp ASC`로 결정성을 보장한다. 한 리소스가 여러 activeflow에 엮인 비정상 케이스에서도 가장 이른 activeflow를 일관되게 반환한다 (단일 activeflow 가정은 정상 동작이며, 위반 시에도 결정적). `max_execution_time`으로 폭주를 막는다 (리뷰 #20).

**존재 확인 (ResourceExists, Q2 구분용)**
```sql
SELECT 1 FROM events WHERE resource_id = ? LIMIT 1
SETTINGS max_execution_time = 5
```

**2단계: 집계 (LIMIT + 타임아웃, 리뷰 #13/#20)**
```sql
SELECT
    publisher,
    resource_id,
    min(data_type)             AS data_type,
    groupUniqArray(event_type) AS event_types,
    min(timestamp)             AS first_seen,
    max(timestamp)             AS last_seen
FROM events
WHERE activeflow_id = ?
  AND resource_id != ''
GROUP BY publisher, resource_id
ORDER BY first_seen ASC
LIMIT ?                        -- maxResources + 1 (truncation 감지)
SETTINGS max_execution_time = 10
```
`groupUniqArray`로 event_type 중복 제거, `min/max(timestamp)`로 시간 범위를 접는다. `min(data_type)`은 같은 (publisher, resource_id)에서 data_type이 상수라는 가정 하에 결정적 값을 고른다 (`any()`는 비결정적이므로 회피, 2차 리뷰 #6). `LIMIT`으로 대용량 activeflow 그래프를 제한하고, 초과 시 응답 `Truncated=true`로 알린다. `max_execution_time`으로 풀스캔 폭주를 차단한다.

`maxResources` 기본값: 1000 (config 노출). 정렬은 ClickHouse `ORDER BY first_seen`에서 1차, Go에서 publisher 그룹핑 후 그룹 내 재정렬로 안정화한다.

**Truncation 의미 (2차 리뷰 #5)**: LIMIT은 `first_seen ASC` 전역 정렬 후 적용되므로, 잘릴 때 전 publisher에 걸쳐 **가장 늦게 등장한** 리소스부터 누락된다. publisher별로는 일부 그룹이 불완전해 보일 수 있으나 결정적이다. 연속 토큰(continuation token)은 없다. maxResources를 초과하는 그래프는 cap 이상 조회 불가하며, 필요 시 maxResources config 상향으로 대응한다.

## 7. RabbitMQ Integration

### Request URI

기존 timeline listenhandler 라우팅 규칙을 따른다. 신규 URI:

```
GET /v1/resource_correlations/<resource_id>
```

또는 기존 패턴(POST + body)과의 일관성을 위해:

```
POST /v1/resource_correlations   body: { "resource_id": "<uuid>" }
```

→ Open Question Q1 (GET path param vs POST body). 기존 `aggregated_events`는 POST + body, `events`도 POST + body 패턴을 사용한다. 일관성상 POST + body가 유력.

### 신규 파일

- `pkg/listenhandler/models/request/resource_correlation.go` — 요청 DTO
- `pkg/listenhandler/models/response/resource_correlation.go` — 응답 DTO (4절 구조체)
- `pkg/listenhandler/v1_resource_correlations.go` — listen 핸들러
- `pkg/listenhandler/main.go` — 라우팅 등록

### requesthandler (bin-common-handler)

다른 서비스가 이 RPC를 호출할 수 있도록 `bin-common-handler/pkg/requesthandler`에 클라이언트 메서드 추가 (예: `TimelineV1ResourceCorrelationGet`). Phase 1에서 실제 소비자가 없다면 이 단계는 선택. → Open Question Q3.

## 8. Observability

Prometheus 메트릭 3개 추가 (timeline-manager 기존 메트릭 네이밍 규칙 + `initPrometheus()` 중복 등록 확인 후):

- Counter: `timeline_resource_correlation_total{result="success|not_found|empty|error"}` (label 값 고정, 카디널리티 제한, 리뷰 #21)
- Histogram: `timeline_resource_correlation_process_time` (쿼리 2단계 합산 레이턴시)
- Histogram: `timeline_resource_correlation_graph_size` (반환 리소스 수, 용량 인사이트, 리뷰 #21)

로깅: `logrus.WithFields`에 `resource_id`, `activeflow_id` 포함. 기존 핸들러 컨벤션 준수.

## 9. Security & Compliance

- timeline-manager의 RPC listen 경로는 read-only (domain.md 명시). 이 메서드도 read-only를 유지한다.
- customer ownership 검증: 현재 timeline events 테이블에는 customer_id 컬럼이 없다 (publisher/resource_id/activeflow_id 기반). 내부 RPC이고 호출 주체가 신뢰된 내부 서비스이므로 Phase 1에서는 ownership 필터를 적용하지 않는다. 외부 REST 노출(후속 페이즈) 시점에 api-manager가 customer 경계를 강제해야 한다. → Open Question Q4.
- 위협 모델 명시 (리뷰 #19): events에 customer_id가 없으므로, 임의 UUID를 가진 내부 호출자는 어떤 activeflow 그래프든 열거할 수 있다. 이 RPC listener가 외부에서 도달 불가함을 전제로 한다 (RabbitMQ 내부 큐, api-manager 미경유). 외부 노출 전까지 이 전제를 유지한다.
- 쿼리 자원 한계 (리뷰 #20): 모든 쿼리에 `max_execution_time` 설정. 2단계 집계에 `LIMIT maxResources+1`. 메모리/시간 폭주 차단.
- PII: 응답은 리소스 ID + event_type + timestamp만 포함한다. 대화 내용/전화번호 등 payload 본문은 반환하지 않으므로 PII 노출이 없다.
- 데이터 보존 한계 (리뷰 #23): events 테이블 TTL이 1년이므로 1년 이상 지난 이벤트는 correlation에 잡히지 않는다. 기능적 한계로 명시한다.

## 10. Affected Services

| Service | Change | Phase |
|---|---|---|
| bin-timeline-manager | 신규 RPC 메서드, dbhandler 3메서드, skip index migration(000004), 메트릭 3 | 1 |
| bin-common-handler | requesthandler 클라이언트 메서드 (소비자 있을 때) | 1 (optional) |
| bin-message-manager | (후속) 이벤트 payload에 activeflow_id 추가 시 자동 포착 | future |
| bin-transfer-manager | (후속) 동일 | future |

## 11. Implementation Order

1. Migration 000004 (skip index 2개 + MATERIALIZE COLUMN 백필) 작성 + `timeline-control migrate up` 검증. 운영 적용 전 테이블 크기 측정.
2. `event.CorrelatedRow` / response DTO(`V1DataResourceCorrelationGet`, `PublisherGroup`, `CorrelatedResource`) 정의.
3. dbhandler `ResourceActiveflowIDGet`, `ResourceExists`, `CorrelatedResourceList` + 단위 테스트 (쿼리 문자열 검증 + 행 스캔).
4. eventhandler `ResourceCorrelationGet` + 단위 테스트 (mock dbhandler, 그룹핑/truncation/not-found/empty 분기, uuid 파싱 실패 skip).
5. listenhandler `v1_resource_correlations.go` + 라우팅 등록 + 테스트.
6. Prometheus 메트릭 3개 등록 (중복 등록 panic 주의).
7. (optional) requesthandler 클라이언트 메서드.
8. 전체 검증 워크플로우 (go mod tidy/vendor/generate/test/lint).

## 12. Open Questions

| # | Question | Recommendation | Owner |
|---|---|---|---|
| Q1 | RPC 형태: GET path param vs POST body | 기존 events/aggregated_events 일관성상 POST + body | CTO |
| Q2 | activeflow 못 찾을 때: error vs 빈 그래프 200 | 빈 그래프(200) + `resource_found`/`activeflow_id=nil`로 구분 (v2 반영) | CPO/CTO |
| Q3 | requesthandler 클라이언트 메서드를 Phase 1에 포함? | 실제 소비자(향후 admin/디버그 도구) 확정 시 추가. Phase 1은 listener까지 | CTO |
| Q4 | customer ownership 미적용을 명시적으로 수용? | 내부 RPC 한정이므로 수용. 외부 노출 시 api-manager 강제 | CTO |
| Q5 | MATERIALIZED 백필 (000003 이전 과거 이벤트 공백) | migration 000004에 `MATERIALIZE COLUMN` 선행 단계 포함 (v2 반영). 운영 적용 비용 측정 필요 | CTO |
| Q6 | skip index MATERIALIZE 적용 비용 (대용량 events) | 운영 테이블 크기 확인 후 적용 시간 추정. 저트래픽 시간대 실행, system.mutations 모니터링 (v2 반영) | CTO |
| Q7 | bloom_filter 인덱스가 대상 CH 버전에서 MATERIALIZED 컬럼 위 정상 동작? | 구현 착수 전 사전 검증. 실패 시 minmax 인덱스 또는 JSON 직접 필터 대안 (v2 반영) | CTO |
| Q8 | maxResources 기본값 1000 적절? | config 노출로 운영 조정 가능하게. 1000은 진단 충분 추정 | CPO/CTO |

## 13. Review Summary (v1 → v2)

1차 독립 리뷰(CHANGES REQUESTED)의 Critical/High 항목을 반영했다.

- **[Critical #1/#2] projection `SELECT *` 폐기 → bloom_filter skip index 채택.** point lookup에 맞는 도구로 교체, 스토리지 3배 문제 제거. (§5 전면 개정)
- **[High #6/#7 Q5] 000003 materialized 컬럼 의존 해결**: migration 000004에 `MATERIALIZE COLUMN` 백필 선행 단계 추가. (§5)
- **[High #7] Q1 쿼리 결정성**: `ORDER BY timestamp ASC` 추가. (§6)
- **[High #13/#20] 페이지네이션/타임아웃 부재**: `LIMIT maxResources+1` + `Truncated` 플래그 + `max_execution_time`. (§6, §9)
- **[High #12] map 그룹핑이 ORDER BY 무효화**: `map[publisher][]` → `[]*PublisherGroup` 정렬 slice. (§4)
- **[Medium #16] uuid 파싱 실패 처리**: 행 skip + warn, 전체 500 방지. (§6)
- **[Medium #18 Q2] resource_found 구분**: `ResourceExists` + `ResourceFound` 필드. (§4, §6)
- **[Medium #19] 위협 모델 명시**, **[Low #21] 메트릭 label 고정 + graph_size 추가**, **[Low #23] TTL 1년 한계 명시**. (§8, §9)
- **[Medium #8] 다중 publisher 등장 의도 명시**, **[Low #11] EventTypes API 경계 정렬**, **[Medium #9] any(data_type) 불변 가정 명시**. (§4, §6)
- 신규 Open Question Q7(bloom_filter+MATERIALIZED 버전 검증), Q8(maxResources 기본값) 추가.

## 14. Review Summary (v2 → v2.1, 2차 리뷰)

2차 독립 리뷰 결과 **APPROVE** (1차 Critical/High 8건 모두 실질 해결 확인, 신규 Critical/High 없음). non-blocking 항목 중 즉시 반영:

- **[2차 #4] ResourceExists 쿼리에 `max_execution_time = 5` 추가.** (§6)
- **[2차 #6] `any(data_type)` → `min(data_type)`** (결정성). (§6)
- **[2차 #5] Truncation 의미 명시**: 전역 first_seen 정렬 후 cap, 늦게 등장한 리소스부터 누락, continuation token 없음. (§6)

구현 PR 단계 추적 항목 (production enable 전 close):
- [2차 #1] bloom_filter skip index 실효성 벤치마크 (`EXPLAIN indexes=1`, 대표 데이터). 테이블이 `ORDER BY (event_type, timestamp)`라 UUID가 granule 전반에 흩어져 skip 효과가 보장되지 않음.
- [2차 #2] migrate 성공 != 백필 완료. `system.mutations` 완료를 feature enable 게이트로 둘 것.
- [2차 #3, Q7] 대상 ClickHouse 버전 확정 + bloom_filter on MATERIALIZED 동작 검증.
- [2차 #8, Q4] customer_id 부재 하 ownership 미적용 명시적 sign-off.
