# VOIP-1530: bin-api-manager 목록 핸들러 pagination-envelope 래핑 감사 및 수정

Status: APPROVED (design review 2R, 2 consecutive APPROVED, 2026-09-17)

## 1. Problem statement

VOIP-1529에서 recording 서브리소스 목록 엔드포인트(`GET /recordings/{id}/transcribes`, `/transcripts`)가
OpenAPI 스펙은 `allOf[CommonPagination, {result: array}]` envelope를 요구하는데도 핸들러가 raw JSON 배열을
반환하는 drift가 발견됐다. 프론트가 `response.result`를 소비하므로 목록을 읽지 못해 admin 콘솔 recording
상세의 Transcript 카드가 'No transcript yet'으로 표시됐다.

이 drift는 "스펙은 올바른데 핸들러 구현만 어긋난" 유형으로, 정적 스펙 리뷰나 상태코드만 검증하는 테스트로는
잡히지 않고 실제 UI 렌더에서만 드러난다. 동일 패턴(`GenerateListResponse` 래핑 누락)이 다른 목록 핸들러에도
남아 있을 수 있어 전수 감사가 필요하다.

## 2. 감사 결과 (근거 코드 포함)

`bin-api-manager/server/*.go`의 모든 `c.JSON(200/http.StatusOK, ...)` 호출 386건을 파싱해, 각 인자의
assignment를 역추적하고 (a) `GenerateListResponse` 래핑 여부, (b) 대응 serviceHandler 메서드의 반환이 slice인지,
(c) OpenAPI 스펙 200 응답 shape를 교차 대조했다. 대부분 핸들러는 단일 객체 Get/Create/Update/Delete로 대상이
아니며, slice를 반환하는 목록 핸들러만 정밀 검토했다.

### 2.1 순수 drift (스펙=envelope, 핸들러=raw slice) — 수정 대상

| # | 핸들러 (file:line) | serviceHandler 메서드 (반환 타입) | OpenAPI 스펙 200 | 현재 반환 |
|---|---|---|---|---|
| 1 | `contact_addresses.go:55` `GetContactAddresses` | `ContactAddressList` → `[]cmcontact.Address` (값 슬라이스) | `allOf[CommonPagination,{result:array}]` (`paths/contact_addresses/main.yaml`) | `c.JSON(200, res)` raw slice |
| 2 | `service_agents_contact_addresses.go:55` `GetServiceAgentsContactAddresses` | `ServiceAgentContactAddressList` → `[]cmcontact.Address` (값 슬라이스) | `allOf[CommonPagination,{result:array}]` (`paths/service_agents/contact_addresses.yaml`) | `c.JSON(200, res)` raw slice |

### 2.2 envelope 일치하나 헬퍼 미사용 (수동 struct) — GenerateListResponse로 통일

| # | 핸들러 (file:line) | serviceHandler 메서드 (반환 타입) | OpenAPI 스펙 200 | 현재 반환 |
|---|---|---|---|---|
| 4 | `timelines.go:85` `GetTimelinesResourceTypeResourceIdEvents` | `TimelineEventList` → `([]*TimelineEvent, string, error)` | `allOf[CommonPagination,{result:array}]` (`paths/timelines/resource_events.yaml`) | 수동 `struct{Result interface{}; NextPageToken string}` |
| 5 | `aggregated_events.go:82` `GetAggregatedEvents` | `AggregatedEventList` → `([]*TimelineEvent, string, error)` | `allOf[CommonPagination,{result:array}]` (`paths/timelines/aggregated_events.yaml`) | 수동 `struct{Result interface{}; NextPageToken string}` |

### 2.3 스코프 제외 (대표님 확정, 2026-09-17)

- **#3 `available_numbers.go:68` `GetAvailableNumbers`**: 실시간 번호 검색으로 커서 페이지네이션 개념이
  없음. 스펙은 `{result}` 래퍼는 있으나 `CommonPagination`이 없고(`next_page_token` 미정의), 핸들러도
  이미 `struct{Result []*...}`로 스펙과 일치(handler==spec). 즉 drift가 아니므로 변경 불필요이며, 오히려
  `GenerateListResponse`를 적용하면 스펙에 없는 `next_page_token`을 추가해 새 drift를 만들게 됨. **현행 유지**.
- **#6 `service_agents_talk.go:353` `GetServiceAgentsTalkChatsIdParticipants`**: 스펙 자체가 bare
  `type: array`(envelope 아님)이고 핸들러도 스펙과 일치. 이건 "핸들러가 스펙을 어긴" drift가 아니라
  스펙을 envelope로 바꾸고 프론트도 함께 고쳐야 하는 별개 성격의 작업. **별도 티켓으로 분리**(본 PR 범위 외).

## 3. Decisions locked (2026-09-17)

1. 수정 스코프: #1, #2 (순수 drift) + #4, #5 (헬퍼 통일). #6은 별도 티켓 분리, #3은 현행 유지.
2. #6 별도 티켓은 본 PR 머지 후 별도 design-first 후속으로 처리(범위확장: 미병합 PR 없으므로 새 티켓).

## 4. Goals (numbered, testable)

1. `GetContactAddresses`가 `{result:[...], next_page_token}` envelope를 반환한다.
2. `GetServiceAgentsContactAddresses`가 동일 envelope를 반환한다.
3. `GetTimelinesResourceTypeResourceIdEvents`가 `GenerateListResponse` 헬퍼로 envelope를 생성한다(수동 struct 제거).
4. `GetAggregatedEvents`가 `GenerateListResponse` 헬퍼로 envelope를 생성한다(수동 struct 제거).
5. 각 핸들러 테스트에 응답 본문(envelope shape) assertion과 빈 목록 `result:[]` 회귀 케이스를 추가한다.
6. 전체 검증 워크플로우(go mod tidy/vendor, go generate, go test, golangci-lint) 통과.

## 5. Non-goals

- #3 available_numbers 변경 (의도된 설계, 현행 유지).
- #6 talk participants envelope 전환 (별도 티켓, 스펙+프론트 동시 수정 필요).
- OpenAPI 스펙 변경 (대상 4개 엔드포인트는 이미 스펙이 envelope로 올바름 — 스펙-핸들러 drift를 핸들러 쪽 수정으로 해소).
- 프론트(square-admin) 변경 (스펙이 이미 `result`를 정의하고 있어 프론트는 이미 `result`를 소비하도록 작성됨).

## 6. Affected files

| File | 변경 이유 |
|---|---|
| `bin-api-manager/server/contact_addresses.go` | `GetContactAddresses` raw slice → `GenerateListResponse` 래핑 (#1) |
| `bin-api-manager/server/service_agents_contact_addresses.go` | `GetServiceAgentsContactAddresses` raw slice → 래핑 (#2) |
| `bin-api-manager/server/timelines.go` | 수동 struct → `GenerateListResponse` (#4) |
| `bin-api-manager/server/aggregated_events.go` | 수동 struct → `GenerateListResponse` (#5) |
| `bin-api-manager/server/contact_addresses_test.go` | `Test_GetContactAddresses` 추가 (envelope + 빈 목록) |
| `bin-api-manager/server/service_agents_contact_addresses_test.go` | `Test_GetServiceAgentsContactAddresses` 추가 |
| `bin-api-manager/server/timelines_test.go` | 기존 테스트의 기대 응답 문자열을 envelope로 갱신 + 빈 목록 케이스 |
| `bin-api-manager/server/aggregated_events_test.go` | 동일 |

## 7. 참조 패턴

`GenerateListResponse[T any](tmps []*T, nextTokenValue string)` (`server/main.go:91`):
- `Result []*T json:"result"` + `openapi_server.CommonPagination`(NextPageToken `*string`).
- 빈/ nil 슬라이스는 `[]*T{}`로 coerce → `"result":[]` (null 아님).
- `len(tmps)==0`이면 nextToken은 빈 문자열. `NextPageToken`은 항상 non-nil 포인터라 `"next_page_token":""`로 직렬화됨.

nextToken 계산(recording/`GetRecordings` 선례, `TMCreate`가 `*time.Time`일 때):
```go
nextToken := ""
if len(tmps) > 0 {
    if tmps[len(tmps)-1].TMCreate != nil {
        nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
    }
}
res := GenerateListResponse(tmps, nextToken)
c.JSON(200, res)
```

## 8. Exact changes (per file)

### 8.1 `contact_addresses.go` (#1) — 값 슬라이스 → 포인터 슬라이스 변환 필수

`ContactAddressList`는 `[]cmcontact.Address`(값)를 반환하나 `GenerateListResponse[T]`는 `[]*T` 필요.
값 슬라이스를 포인터 슬라이스로 변환한 뒤 래핑한다. `cmcontact.Address.TMCreate`는 `*time.Time`.

```go
// 기존:
	res, err := h.serviceHandler.ContactAddressList(c.Request.Context(), a, filters, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not list contact addresses. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)

// 변경:
	tmps, err := h.serviceHandler.ContactAddressList(c.Request.Context(), a, filters, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not list contact addresses. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	items := make([]*cmcontact.Address, len(tmps))
	for i := range tmps {
		items[i] = &tmps[i]
	}

	nextToken := ""
	if len(items) > 0 {
		if items[len(items)-1].TMCreate != nil {
			nextToken = items[len(items)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(items, nextToken)
	c.JSON(200, res)
```
- import에 `cmcontact "monorepo/bin-contact-manager/models/contact"` 추가 필요(현재 파일에 없음 — 확인함).
- `&tmps[i]` 패턴 사용: 루프 변수 주소 함정 회피(range 변수 `v`의 주소가 아니라 슬라이스 원소의 주소를 취함).

### 8.2 `service_agents_contact_addresses.go` (#2)

8.1과 동일 패턴. `ServiceAgentContactAddressList` → `[]cmcontact.Address` 변환 후 래핑.
import에 `cmcontact` 추가 필요(현재 파일 확인 후 반영).

### 8.3 `timelines.go` (#4) — 이미 포인터 슬라이스 + nextToken 별도 반환

```go
// 기존:
	events, nextPageToken, err := h.serviceHandler.TimelineEventList(...)
	...
	// Build response
	res := struct {
		Result        interface{} `json:"result"`
		NextPageToken string      `json:"next_page_token,omitempty"`
	}{
		Result:        events,
		NextPageToken: nextPageToken,
	}
	c.JSON(http.StatusOK, res)

// 변경:
	events, nextPageToken, err := h.serviceHandler.TimelineEventList(...)
	...
	res := GenerateListResponse(events, nextPageToken)
	c.JSON(http.StatusOK, res)
```
- `events`는 이미 `[]*TimelineEvent`이므로 변환 불필요.
- nextToken은 serviceHandler가 계산해 반환한 값을 그대로 전달(재계산 안 함 — 기존 동작 보존).
  단 `GenerateListResponse`는 `len==0`이면 nextToken을 빈 문자열로 강제하므로, 빈 목록에서 non-empty
  nextPageToken을 반환하던 경우 동작이 바뀔 수 있음 → §11 위험 참조.
- `net/http` import는 `c.JSON(http.StatusOK, ...)`가 남으므로 유지.
- 익명 struct 제거로 사용 안 하게 되는 import 없음(확인).

### 8.4 `aggregated_events.go` (#5)

8.3과 동일. `AggregatedEventList` → `([]*TimelineEvent, string, error)` 그대로 `GenerateListResponse(events, nextPageToken)`.

## 9. 테스트 계획

각 핸들러에 대해 (recording `Test_GetRecordingsIdTranscribes` 선례 재사용):
1. **정상 목록 케이스**: mock이 N개 원소 반환 → `w.Body.String()`이 정확히
   `{"result":[...],"next_page_token":"<TMCreate 마이크로초 포맷>"}`인지 assert.
2. **빈 목록 케이스**: mock이 빈 슬라이스 반환 → 본문이 정확히 `{"result":[],"next_page_token":""}`인지 assert
   (nil-coercion + raw-array 회귀 동시 방지).

주의(직렬화 함정, recording 선례에서 확인됨):
- 원소의 `tm_create`는 Go 기본 `time.Time` JSON = RFC3339Nano(후행 0 절삭, 예 `...995Z`).
- `next_page_token`은 명시적 `Format("...000000Z")` = 고정 6자리 마이크로초(예 `...995000Z`).
- 같은 인스턴스라도 두 필드의 문자열 표현이 다르므로, 기대 문자열은 실제 실행 1회 후 `w.Body.String()`을
  복사해 확정한다(손으로 추측 금지).

timelines/aggregated_events는 기존 테스트가 있으므로 **기존 기대 문자열을 envelope 형태로 갱신** +
빈 목록 케이스 추가. contact_addresses 2개는 신규 `Test_Get*` 함수 추가.

## 10. Verification plan

`bin-api-manager`에서:
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
- `go generate`는 코드 생성만 — 스펙 변경 없으므로 gen.go diff 0 기대(변경 시 조사).
- grep: `grep -n "c.JSON(200, res)\|c.JSON(http.StatusOK, res)" server/contact_addresses.go server/service_agents_contact_addresses.go server/timelines.go server/aggregated_events.go` → 4개 파일에서 raw-slice/수동-struct 반환이 사라졌는지 확인.

## 11. Rollout / risk

- **위험 낮음.** 4개 엔드포인트 모두 스펙이 이미 envelope를 정의하고 있어 프론트는 이미 `result`를 소비하도록
  작성돼 있음(스펙이 SoT). 핸들러를 스펙에 맞추는 방향이라 프론트 변경 불필요.
  - **프론트 소비 실증 (square-admin, 직접 확인 2026-09-17)**: contact_addresses 목록의 모든 소비 지점이
    `.result` 우선 + raw-array fallback으로 방어적으로 작성됨:
    - `hooks/usePaginatedList.js:57`: `result && result.result ? result.result : (Array.isArray(result) ? result : [])`
      (`ResourceListPage` → `contact_addresses_list.js`가 사용)
    - `views/contacts/UnresolvedAddressPicker.js:100`: `(result && (result.result || result.items)) || []`
    - `views/contacts/contact_cases_detail.js:474`: `result?.result || result?.items || (Array.isArray(result) ? result : [])`
    즉 현재 백엔드가 raw array를 반환하는데도 프론트는 `.result`를 먼저 찾다 fallback으로 raw array를 읽는 중.
    envelope로 고치면 `.result` 정상 경로가 작동하고 fallback은 그대로 안전 → **수정이 프론트를 정상화**.
- **동작 변화 (#1, #2)**: 응답이 `[{...}]` → `{"result":[{...}],"next_page_token":"..."}`로 바뀜.
  기존에 raw 배열을 파싱하던 클라이언트가 있다면 깨질 수 있으나, 스펙상 원래 envelope가 계약이므로
  스펙 준수 클라이언트에는 영향 없음. contact_addresses는 신규 리소스(VOIP-1207 계열)로 raw-array 소비자
  가능성 낮음.
- **동작 변화 (#4, #5)**: `next_page_token`이 `omitempty`(빈 문자열 시 생략) → 항상 포함(빈 문자열)으로 바뀜.
  이는 플랫폼 전반 envelope 표준(GenerateListResponse)과 일치시키는 것으로 recording과 동일. 빈 목록에서
  기존엔 `next_page_token` 필드가 없었고 이제 `""`로 나타남. 프론트는 `res.next_page_token || ''` 형태로
  방어적으로 읽으므로 영향 없음.
  - **미묘한 회귀 가능성**: `GenerateListResponse`는 `len(tmps)==0`이면 nextToken을 빈 문자열로 강제한다.
    현재 `TimelineEventList`/`AggregatedEventList`가 빈 목록에서도 non-empty nextPageToken을 반환하는
    경우가 있다면 그 토큰이 사라진다. 커서 페이지네이션에서 빈 페이지의 nextToken은 통상 "더 없음"을
    의미하므로 실질 영향은 없을 것으로 판단하나, 설계 리뷰에서 serviceHandler의 빈-목록 nextToken 동작을
    확인하도록 함(§15 open question).

## 12. Approval status

Draft.

## 14. 사전 검증 완료 (설계 전 확인)

- **Open question #1 해소**: `TimelineEventList`(`pkg/servicehandler/timeline.go`)와 `AggregatedEventList`는
  nextToken을 자체 계산하지 않고 백엔드 응답 `resp.NextPageToken`을 그대로 반환한다
  (`return result, resp.NextPageToken, nil`). 빈 목록이면 백엔드도 통상 빈 토큰을 반환하므로
  `GenerateListResponse`의 `len==0 → nextToken=""` 강제는 실질 무해. 다만 이론상 백엔드가 빈 페이지에
  non-empty 토큰을 주는 경우 그 토큰이 사라지는데, 이는 커서 종료 신호와 동일 의미라 회귀 아님.
- **cmcontact import 확인**: `contact_addresses.go`, `service_agents_contact_addresses.go` 둘 다 현재
  `cmcontact` 미import → 8.1/8.2에서 추가 필요(확정).

## 15. Open questions (리뷰어 확인)

1. contact_addresses의 `[]cmcontact.Address` → `[]*cmcontact.Address` 변환에서 `&tmps[i]` 패턴이
   올바른가(range 변수 주소 함정 회피)? (§8.1)
