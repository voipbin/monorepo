# AI Agent Evaluation Feature Design

**Date:** 2026-05-21
**Status:** Draft v2 (post-review)
**Author:** Hermes (CPO)
**Reviewers:** Independent subagent review (27 items identified, addressed below)

---

## 1. Problem Statement

VoIPBin AI Agent(AIcall)가 고객 응대를 수행한 후, 해당 응대의 품질을 사후에 평가하는 메커니즘이 없다. LLM 생성 Summary는 있지만 정량 평가가 부재하다.

평가 기능이 있으면:
- 고객사가 AI 응대 품질을 정량적으로 추적할 수 있다
- 특정 기준(목표 달성률, 감성 톤, 문제 해결 품질 등)에 따라 AI 설정 개선의 근거를 확보할 수 있다
- Flow에서 평가 결과를 조건 분기나 후속 액션 트리거로 활용할 수 있다

---

## 2. Scope

### In scope (Phase 1)
- `AIEvaluation` 도메인 모델 (bin-ai-manager 내 신규 모델 + handler)
- LLM 기반 자동 평가 생성 (engine_openai_handler 재사용, GPT-4o)
- REST API 노출 (OpenAPI 스펙)
- 평가 완료 후 on_end_flow 트리거
- Webhook 이벤트 발행
- Customer ownership 검증 (aicall_id가 해당 customer 소유인지 확인)

### Out of scope
- **Phase 2:** AI config의 `auto_evaluation` 플래그 + AIcall terminated 이벤트 기반 자동 트리거
- **Phase 2:** bin-flow-manager `run_evaluation` Flow action
- **Phase 2:** 수동 평가 입력 UI (square-admin)
- **Phase 3:** RLHF-style 자동 피드백 루프

### 보류 결정 사항
- **자동 트리거(Phase 2) 보류 이유:** 모든 AIcall 종료 시 GPT-4o를 호출하면 LLM 비용이 통제 불가. 평균 메시지 수 × 토큰 비용 × 콜 볼륨 기반 비용 추정 후 Phase 2에서 재검토. 고객사 선택적 적용이 적합.

---

## 3. Domain Model: AIEvaluation

### 3.1 모델 정의 (`bin-ai-manager/models/aievaluation/main.go`)

```go
type AIEvaluation struct {
    commonidentity.Identity  // id, customer_id

    ActiveflowID uuid.UUID `json:"activeflow_id" db:"activeflow_id,uuid"`
    OnEndFlowID  uuid.UUID `json:"on_end_flow_id" db:"on_end_flow_id,uuid"`

    // 평가 대상 AIcall ID
    AicallID uuid.UUID `json:"aicall_id" db:"aicall_id,uuid"`

    // 평가 수행 모델 (평가 시점의 모델명 기록)
    EngineModel string `json:"engine_model" db:"engine_model"`

    Status   Status `json:"status" db:"status"`
    Language string `json:"language" db:"language"`

    // 평가 결과 (LLM 생성, progressing 중 nil)
    Result *EvaluationResult `json:"result,omitempty" db:"result,json"`

    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
    TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
    TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

type Status string

const (
    // StatusNone: 사용하지 않음 — 빈 문자열 zero-value 충돌 방지를 위해 미정의
    StatusProgressing Status = "progressing"
    StatusDone        Status = "done"
    StatusFailed      Status = "failed"
)

// EvaluationResult — LLM이 채우는 구조화된 평가 결과
// 모든 Score.Value는 0.0~10.0 범위 (프롬프트에서 강제, unmarshal 후 검증)
type EvaluationResult struct {
    Version              int     `json:"version"`               // 스키마 버전 (현재: 1)
    OverallScore         float64 `json:"overall_score"`         // 종합 점수 (0.0~10.0), LLM 직접 산출
    GoalAchievement      Score   `json:"goal_achievement"`
    CustomerSentiment    Score   `json:"customer_sentiment"`
    ResolutionQuality    Score   `json:"resolution_quality"`
    CommunicationClarity Score   `json:"communication_clarity"`
    TopicAdherence       Score   `json:"topic_adherence"`
    Summary              string  `json:"summary"`
    Strengths            []string `json:"strengths"`
    Improvements         []string `json:"improvements"`
}

type Score struct {
    Value  float64 `json:"value"`  // 0.0~10.0 (범위 외 값은 unmarshal 후 검증에서 거부)
    Reason string  `json:"reason"`
}

// Validate checks all Score values are in [0.0, 10.0].
func (r *EvaluationResult) Validate() error {
    scores := []struct {
        name  string
        value float64
    }{
        {"overall_score", r.OverallScore},
        {"goal_achievement", r.GoalAchievement.Value},
        {"customer_sentiment", r.CustomerSentiment.Value},
        {"resolution_quality", r.ResolutionQuality.Value},
        {"communication_clarity", r.CommunicationClarity.Value},
        {"topic_adherence", r.TopicAdherence.Value},
    }
    for _, s := range scores {
        if s.value < 0 || s.value > 10 {
            return fmt.Errorf("score %s out of range [0,10]: %f", s.name, s.value)
        }
    }
    return nil
}
```

**설계 결정:**
- `StatusNone = ""` 미정의 — zero-value 충돌(JSON omitempty, DB NULL) 방지
- `EvaluationResult.Version` 필드 포함 — 향후 스키마 변경 시 하위 호환
- `Score.Value` 범위: 0.0~10.0 고정, Validate() 메서드로 unmarshal 후 검증

### 3.2 Status lifecycle

```
[Start() called]
      |
      v
  progressing   ──── LLM call timeout(120s) or error ──► failed
      |
      v
    done
```

- `progressing`: DB 생성 후 goroutine에서 LLM 호출 중
- `done`: 평가 완료, Result 채워짐
- `failed`: LLM 오류, 타임아웃, 빈 메시지 히스토리, 결과 검증 실패

---

## 4. Database Schema

### 4.1 Alembic migration (bin-dbscheme-manager)

```sql
CREATE TABLE ai_evaluations (
    id              BINARY(16)    NOT NULL,
    customer_id     BINARY(16)    NOT NULL,
    activeflow_id   BINARY(16)    NOT NULL DEFAULT 0x00000000000000000000000000000000,
    on_end_flow_id  BINARY(16)    NOT NULL DEFAULT 0x00000000000000000000000000000000,

    aicall_id       BINARY(16)    NOT NULL,
    engine_model    VARCHAR(255)  NOT NULL DEFAULT '',
    status          VARCHAR(50)   NOT NULL DEFAULT '',
    language        VARCHAR(50)   NOT NULL DEFAULT 'en',
    result          JSON          DEFAULT NULL,

    tm_create  DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',
    tm_update  DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',
    tm_delete  DATETIME(6)  NOT NULL DEFAULT '9999-01-01 00:00:00.000000',

    PRIMARY KEY (id),
    KEY idx_aievaluations_customer_id_tm_create (customer_id, tm_create),
    KEY idx_aievaluations_customer_id_aicall_id (customer_id, aicall_id),
    KEY idx_aievaluations_customer_id_tm_delete (customer_id, tm_delete)
);
```

**설계 결정:**
- `tm_delete DEFAULT '9999-01-01 00:00:00.000000'` — VoIPBin soft-delete 컨벤션 준수
- 인덱스 3개: 목록 조회(customer_id + tm_create), aicall 기준 조회(customer_id + aicall_id), soft-delete 필터(customer_id + tm_delete)
- `language DEFAULT 'en'` — 미입력 시 기본값

### 4.2 참조 관계

- `aicall_id`는 `ai_aicalls.id` soft reference (VoIPBin 관례상 FK 없음)
- 동일 `aicall_id`에 대해 여러 평가 가능 (dedup 없음, Summary와 다름)
- 동시 `progressing` 평가 허용 — 클라이언트가 관리 책임

---

## 5. Handler Interface

```go
// bin-ai-manager/pkg/aievaluationhandler/main.go

type AIEvaluationHandler interface {
    // CRUD
    Get(ctx context.Context, id uuid.UUID) (*aievaluation.AIEvaluation, error)
    List(ctx context.Context, size uint64, token string, filters map[aievaluation.Field]any) ([]*aievaluation.AIEvaluation, error)
    Delete(ctx context.Context, id uuid.UUID) (*aievaluation.AIEvaluation, error)

    // 평가 시작
    // - aicall_id ownership 검증 (customerID 일치 확인)
    // - 메시지 히스토리 사전 검증 (user/assistant 메시지 0개면 status=failed 즉시 반환)
    // - DB insert (status=progressing)
    // - goroutine 내 LLM 호출 (context + 120s timeout, recover() 포함)
    // - 완료 시 DB update (status=done/failed)
    Start(
        ctx context.Context,
        customerID uuid.UUID,
        activeflowID uuid.UUID,
        onEndFlowID uuid.UUID,
        aicallID uuid.UUID,
        language string,
    ) (*aievaluation.AIEvaluation, error)

    // Flow Service 액션 진입점 (Phase 2 준비)
    ServiceStart(
        ctx context.Context,
        customerID uuid.UUID,
        activeflowID uuid.UUID,
        onEndFlowID uuid.UUID,
        aicallID uuid.UUID,
        language string,
    ) (*commonservice.Service, error)
}
```

### 5.1 Start() 내부 흐름

```
1. reqHandler.AIcallV1Get(aicallID)
   → 404: return error
   → aicall.CustomerID != customerID: return 403 error
   → aicall.TMDelete != 9999-...: return 404 error (soft-deleted)

2. messagehandler.List(aicallID)  — user/assistant 메시지 수 확인
   → count == 0: Create(status=failed), publish event, return

3. Create(status=progressing) → publish aievaluation.created event

4. go func() {
       defer func() { if r := recover(); r != nil { update(status=failed) } }()
       ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
       defer cancel()
       result, err := engineOpenaiHandler.Evaluate(ctx, initPrompt, messages, language)
       if err != nil || result.Validate() != nil {
           update(status=failed)
       } else {
           update(status=done, result=result)
       }
       publish aievaluation.updated event
       startOnEndFlow() — 실패 시 로그만 (on_end_flow 실패는 평가 status에 영향 없음)
   }()

5. return progressing 상태의 AIEvaluation
```

---

## 6. LLM Evaluation Logic

### 6.1 engine_openai_handler 재사용

`summaryhandler`와 동일하게 `engine_openai_handler`를 통해 OpenAI API 호출. 직접 SDK 호출 금지.

### 6.2 평가 모델 설정

- **기본 모델:** `gpt-4o` (고정)
- **설정 소스:** 환경 변수 `AI_EVALUATION_ENGINE_MODEL` (미설정 시 `gpt-4o`)
- **rationale:** 평가 일관성 확보를 위해 고정. 개별 AI config 모델 상속 안 함.

### 6.3 입력 데이터

1. **AIcall 메시지 히스토리** — role IN (user, assistant) 메시지 전체, 시간순 정렬
2. **AI init_prompt** — AIcall → assistanceID → AI 설정의 init_prompt 조회
3. **언어** — POST body의 `language` 파라미터

### 6.4 평가 프롬프트

```
System:
You are an expert call center QA evaluator.
Evaluate the following AI agent conversation strictly based on the criteria below.
Respond ONLY with a valid JSON object. Do not include any text outside the JSON.
All Score values must be in range 0.0 to 10.0.
Use "{language}" for all text fields (summary, reason, strengths, improvements).

Required JSON schema:
{
  "version": 1,
  "overall_score": <float 0.0-10.0>,
  "goal_achievement":      { "value": <float 0.0-10.0>, "reason": "<string>" },
  "customer_sentiment":    { "value": <float 0.0-10.0>, "reason": "<string>" },
  "resolution_quality":    { "value": <float 0.0-10.0>, "reason": "<string>" },
  "communication_clarity": { "value": <float 0.0-10.0>, "reason": "<string>" },
  "topic_adherence":       { "value": <float 0.0-10.0>, "reason": "<string>" },
  "summary":     "<string>",
  "strengths":   ["<string>", ...],
  "improvements": ["<string>", ...]
}

AI Agent Role and Goal:
{init_prompt}

Conversation:
{formatted_messages}  // "User: ...\nAssistant: ...\n" 형식
```

**LLM 호출 파라미터:**
- `response_format`: `json_schema` (structured outputs, GPT-4o 지원) — schema mismatch 방지
  - fallback: `json_object` (structured outputs 미지원 모델 대비)
- `temperature`: `0` — 비결정성 최소화 (단, non-determinism 완전 제거 불가, API 문서에 명시 필요)

### 6.5 실패 처리

| 실패 원인 | 결과 |
|---|---|
| AIcall not found / 다른 customer 소유 | Start()에서 즉시 error return (DB insert 없음) |
| 메시지 없음 (user/assistant 0개) | status=failed, result=nil, on_end_flow 실행 |
| LLM API 오류 (rate limit, network) | status=failed, result=nil, on_end_flow 실행 |
| JSON schema mismatch / Validate() 실패 | status=failed, result=nil, on_end_flow 실행 |
| 120s timeout | goroutine cancel, status=failed |
| goroutine panic | recover(), status=failed |

---

## 7. REST API

### 7.1 Endpoints

```
GET    /v1/aievaluations              List evaluations (paginated, filterable)
POST   /v1/aievaluations              Start evaluation
GET    /v1/aievaluations/{id}         Get evaluation
DELETE /v1/aievaluations/{id}         Delete evaluation (soft-delete)
```

### 7.2 POST /v1/aievaluations (Request Body)

```json
{
  "activeflow_id":  "uuid (optional)",
  "on_end_flow_id": "uuid (optional)",
  "aicall_id":      "uuid (required)",
  "language":       "ko (optional, default: en, BCP-47)"
}
```

**설계 결정:**
- `activeflow_id` 포함 — summaryhandler 패턴 일치
- `language` 생략 시 서버 기본값 `"en"` 적용
- `aicall_id`: customer ownership 서버 측 검증 (403 if mismatch)

### 7.3 POST /v1/aievaluations (Response: 200)

```json
{
  "id": "uuid",
  "customer_id": "uuid",
  "activeflow_id": "uuid",
  "on_end_flow_id": "uuid",
  "aicall_id": "uuid",
  "engine_model": "openai.gpt-4o",
  "status": "progressing",
  "language": "ko",
  "result": null,
  "tm_create": "2026-05-21T10:00:00.000000Z",
  "tm_update": "2026-05-21T10:00:00.000000Z",
  "tm_delete": "9999-01-01T00:00:00.000000Z"
}
```

### 7.4 GET /v1/aievaluations (Query Parameters)

| Parameter | Type | Description |
|---|---|---|
| `page_size` | int | 페이지 크기 (default: 20) |
| `page_token` | string | 커서 기반 페이지네이션 토큰 |
| `aicall_id` | uuid | 특정 AIcall의 평가만 조회 |
| `status` | string | `progressing`, `done`, `failed` 필터 |

### 7.5 GET /v1/aievaluations/{id} (done 상태 예시)

```json
{
  "id": "uuid",
  "status": "done",
  "result": {
    "version": 1,
    "overall_score": 8.2,
    "goal_achievement":      { "value": 9.0, "reason": "고객 문의를 성공적으로 해결했습니다." },
    "customer_sentiment":    { "value": 7.5, "reason": "고객이 중립적에서 긍정적으로 변화했습니다." },
    "resolution_quality":    { "value": 8.5, "reason": "정확한 정보를 제공하고 문제를 해결했습니다." },
    "communication_clarity": { "value": 8.0, "reason": "응답이 명확하고 간결했습니다." },
    "topic_adherence":       { "value": 9.0, "reason": "주제에서 벗어나지 않았습니다." },
    "summary": "AI 에이전트가 고객의 결제 문의를 성공적으로 처리했습니다.",
    "strengths": ["신속한 응답", "정확한 정보 제공"],
    "improvements": ["공감 표현 부족", "추가 안내 부재"]
  }
}
```

---

## 8. Webhook Events

`bin-ai-manager/models/aievaluation/webhook.go`:

| Event key | 발행 시점 |
|---|---|
| `ai_manager.aievaluation.created` | DB insert 직후 |
| `ai_manager.aievaluation.updated` | status done/failed 전환 시 |
| `ai_manager.aievaluation.deleted` | Delete() 호출 시 |

WebhookMessage 구조체:
```go
type WebhookMessage struct {
    ID           uuid.UUID              `json:"id"`
    CustomerID   uuid.UUID              `json:"customer_id"`
    AicallID     uuid.UUID              `json:"aicall_id"`
    EngineModel  string                 `json:"engine_model"`
    Status       aievaluation.Status    `json:"status"`
    Language     string                 `json:"language"`
    Result       *aievaluation.EvaluationResult `json:"result,omitempty"`
    TMCreate     *time.Time             `json:"tm_create"`
    TMUpdate     *time.Time             `json:"tm_update"`
    TMDelete     *time.Time             `json:"tm_delete"`
}
```

---

## 9. Flow Variable Integration

`on_end_flow` 실행 시 activeflow variables에 주입:

| Variable Key | 타입 | 설명 |
|---|---|---|
| `voipbin.ai_evaluation.id` | string (uuid) | evaluation UUID |
| `voipbin.ai_evaluation.aicall_id` | string (uuid) | 평가된 AIcall UUID |
| `voipbin.ai_evaluation.status` | string | done / failed |
| `voipbin.ai_evaluation.overall_score` | string | "0.0"~"10.0" (float → string) |
| `voipbin.ai_evaluation.summary` | string | 평가 요약 |
| `voipbin.ai_evaluation.goal_achievement` | string | "0.0"~"10.0" |
| `voipbin.ai_evaluation.customer_sentiment` | string | "0.0"~"10.0" |

**on_end_flow 실패 처리:** summaryhandler 패턴과 동일하게 에러 로그만 남기고 evaluation status에는 영향 없음.

---

## 10. RabbitMQ Integration

Phase 1에서 `ServiceStart()` 진입점은 bin-flow-manager Phase 2 연동을 위해 인터페이스로만 정의. RabbitMQ action 이름은 Phase 2 설계 시 확정.

Phase 1 내부 서비스 호출:
- AIcall 조회: `requesthandler.AIcallV1Get(aicallID)` — RabbitMQ RPC
- Message 목록 조회: `requesthandler.AIManagerV1MessageList(aicallID)` — RabbitMQ RPC

---

## 11. Observability

### 11.1 Prometheus Metrics

```go
// aievaluationhandler/metrics.go

var (
    // 평가 시작 카운터
    promEvaluationStartTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Namespace: "ai_manager", Name: "evaluation_start_total"},
        []string{},
    )

    // 평가 완료 카운터 (status별)
    promEvaluationDoneTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Namespace: "ai_manager", Name: "evaluation_done_total"},
        []string{"status"},  // "done" | "failed"
    )

    // LLM 호출 지연 히스토그램
    promEvaluationLLMDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: "ai_manager",
            Name:      "evaluation_llm_duration_seconds",
            Buckets:   prometheus.DefBuckets,
        },
        []string{},
    )
)
```

### 11.2 Structured Logging

goroutine 내 LLM 호출 시 trace ID propagation:
```go
// Start() 호출 시점의 traceID를 goroutine에 전달
traceID := utilhandler.GetTraceID(ctx)
go func() {
    gCtx, cancel := context.WithTimeout(
        context.WithValue(context.Background(), utilhandler.TraceIDKey, traceID),
        120*time.Second,
    )
    defer cancel()
    // ...
}()
```

---

## 12. Security & Compliance Considerations

### 12.1 Customer Ownership Validation

`Start()` 진입 시 aicall_id의 customer_id가 요청 customer_id와 일치하는지 반드시 검증. 불일치 시 `403 Forbidden` 반환.

### 12.2 PII / GDPR 고려사항 (CEO/CTO 결정 필요)

평가 수행 시 AIcall 전체 대화 내용(사용자 발화 포함)이 OpenAI API로 전송된다. 이는:
- GDPR / CCPA 적용 대상일 수 있음 (고객 위치, 계약에 따라)
- Enterprise 고객의 경우 데이터 처리 계약(DPA) 확인 필요

**대응 방안 제안 (대표님 결정 필요):**
1. 현재 VoIPBin-OpenAI DPA 상태 확인
2. 단기: 문서(API 가이드)에 "외부 LLM 처리 포함" 명시
3. 중기: 고객별 `allow_external_ai_processing` 동의 플래그 추가

---

## 13. Affected Services

| Service | 변경 내용 | Phase |
|---|---|---|
| `bin-ai-manager` | 신규 모델(`aievaluation`), 신규 핸들러(`aievaluationhandler`), listenhandler 라우팅 | 1 |
| `bin-dbscheme-manager` | Alembic 마이그레이션: `ai_evaluations` 테이블 | 1 |
| `bin-openapi-manager` | `AIManagerAIEvaluation` 스키마, `/v1/aievaluations` 경로 | 1 |
| `bin-api-manager` | `/v1/aievaluations` REST 라우팅 | 1 |
| `bin-flow-manager` | `run_evaluation` Flow action | 2 |
| `square-admin` | 평가 결과 목록/상세 UI | 2 |

---

## 14. Implementation Order

1. **DB 마이그레이션** (bin-dbscheme-manager) — `ai_evaluations` 테이블 생성
2. **모델 정의** — `bin-ai-manager/models/aievaluation/` (main.go, field.go, filters.go, event.go, webhook.go)
3. **DB 핸들러** — `aievaluationhandler/db.go` (Create, Get, List, Update, Delete)
4. **평가 로직** — `aievaluationhandler/start.go` (ownership 검증, 메시지 사전 검증, LLM 호출, goroutine 패턴)
5. **이벤트/웹훅** — `aievaluationhandler/event.go`
6. **OpenAPI 스펙** — `bin-openapi-manager` (스키마 + 경로 + codegen)
7. **API 라우팅** — `bin-api-manager` listen + HTTP handler
8. **단위 테스트** — 각 핸들러 (gomock, table-driven)

---

## 15. Open Questions (결정 보류)

| # | 질문 | 권장 | 결정 필요자 |
|---|---|---|---|
| Q1 | `overall_score` 계산 방법 | LLM 직접 산출 (컨텍스트 고려 가중치), temperature=0 | 대표님 확인 |
| Q2 | 동일 aicall_id 재평가 허용 여부 | 허용 (Summary와 달리 dedup 없음) | 대표님 확인 |
| Q3 | OpenAI DPA / PII 처리 동의 현황 | 내부 확인 후 문서화 | 대표님 결정 필수 |
| Q4 | reference_type/reference_id 확장 전략 | Phase 1: aicall_id 직접 참조. Phase 2: 컬럼 추가 마이그레이션(aicall_id 유지, reference_type/id 추가)으로 비파괴 확장 | 대표님 확인 |
| Q5 | Phase 2 LLM 비용 cap 정책 | 평균 메시지 수 × 토큰 비용 × 콜 볼륨 추정 후 월별 cap 설계 | 대표님 결정 필수 |

---

## 16. Review Summary

초안 대비 v2에서 반영된 주요 변경:

| 항목 | 변경 내용 |
|---|---|
| Security | aicall_id customer ownership 검증 명시 |
| PII | OpenAI 전송 리스크 섹션 신설 (대표님 결정 요청) |
| Goroutine | context + 120s timeout + recover() 패턴 명시 |
| Status enum | `StatusNone=""` 제거 (zero-value 충돌 방지) |
| Score 범위 | 0.0~10.0 고정, Validate() 메서드 정의 |
| JSON schema | response_format=json_schema + 사후 Validate() 이중 검증 |
| result 버전 | `version: 1` 필드 포함 (향후 스키마 진화 대비) |
| POST body | `activeflow_id` 추가 (summaryhandler 패턴 일치) |
| List API | 필터 파라미터 (aicall_id, status, 페이지네이션) 정의 |
| DB | `tm_delete DEFAULT 9999-...` 명시, 인덱스 3개 구체화 |
| Observability | Prometheus metrics 3종 + trace ID propagation 명시 |
| engine_model | 환경 변수 `AI_EVALUATION_ENGINE_MODEL` 명시 |
| engine_openai_handler | 직접 재사용 명시 (raw SDK 호출 금지) |
| on_end_flow failure | log-only, evaluation status 무관 명시 |
| Phase 2 cost | LLM 비용 추정 및 cap 정책 Open Question으로 격상 |
