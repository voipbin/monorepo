# 이슈 분석: recording 서브리소스 transcribe/transcript 목록 응답이 스펙과 불일치 (raw 배열)

작성일: 2026-09-17
작성자: Hermes (CPO)
브랜치: VOIP-NEW-Wrap-recording-transcribe-transcript-list-responses
상태: Draft (이슈 분석 리뷰 대기)

## 1. 문제 요약

admin 콘솔 recording 상세 화면에서 transcribe가 정상 완료(status=done)됐고 transcript도
정상 저장(offset_ms 포함)됐는데도, Transcript 카드가 계속 "No transcript yet"을 표시한다.

## 2. 근본 원인 (코드로 확정)

`bin-api-manager/server/recordings.go`의 두 핸들러가 serviceHandler의 슬라이스를
**래핑 없이 raw 배열로 반환**한다.

- `GetRecordingsIdTranscribes` (recordings.go:208): `c.JSON(200, res)` — res는 `[]*tmtranscribe.WebhookMessage`
- `GetRecordingsIdTranscripts` (recordings.go:276): `c.JSON(200, res)` — res는 `[]*tmtranscript.WebhookMessage`

따라서 응답이 `[{...}]` (raw 배열)로 나간다.

반면 같은 파일의 `GetRecordings` (recordings.go:69)는
`GenerateListResponse(tmps, nextToken)`로 `{result:[...], next_page_token:"..."}` 형태로 래핑한다.
두 서브리소스 핸들러만 이 래핑이 누락됐다.

### 2.1 프론트 소비 계약과의 불일치
- square-admin `TranscriptCard.js`(recording 상세 Transcript 카드): recording 서브리소스
  `recordings/{id}/transcribes`, `recordings/{id}/transcripts`를 직접 호출하고
  `const items = (listRes && listRes.result) || []` / `(res && res.result) || []`로 `res.result`를 읽음
  → raw 배열엔 `.result`가 없어 항상 `[]` → transcribes.length===0 → "No transcript yet"
- 이 컴포넌트가 **유일한 버그 노출 대상**이다.

- 참고(영향 없음): Case 상세 `CaseCallTranscriptPanel.js`는 recording 서브리소스가 아니라
  `caseLinkedApi.js`를 통해 `service_agents/transcribes`, `service_agents/transcripts`를 호출한다.
  이 두 service_agents 핸들러는 이미 GenerateListResponse로 래핑돼 있으므로
  (`service_agents_transcribes.go:78`, `service_agents_transcripts.go:60`)
  `response.result`/페이지네이션이 정상 동작하며 이 버그의 영향을 받지 않는다.

### 2.2 OpenAPI 스펙과의 불일치 (스펙-핸들러 drift)
- `bin-openapi-manager/openapi/paths/recordings/id_transcribes.yaml`
- `bin-openapi-manager/openapi/paths/recordings/id_transcripts.yaml`

두 스펙 모두 200 응답을 `allOf: [CommonPagination, {result: array}]`로 정의한다.
즉 **스펙은 이미 올바르게 `{result, next_page_token}` 형태**를 요구하고 있고,
핸들러 구현이 이 계약을 어기고 raw 배열을 반환하는 것이다.
스펙은 정확하므로 스펙 변경은 불필요하고, 핸들러만 스펙에 맞추면 된다.

## 3. 실측 증거 (프로덕션, 2026-09-16)

recording `b3347c17-f338-42b6-9d67-da7993efeb2d`에 신규 transcribe 실행 후:

- `GET /recordings/{id}/transcribes` →
  `[{"id":"17df5d2c...","status":"done","language":"en-US",...}]` (raw 배열, `.result` 없음)
- `GET /recordings/{id}/transcripts?transcribe_id=17df5d2c...` →
  `[{"...","message":"...","offset_ms":6600,...},{...,"offset_ms":2000,...}]` (raw 배열)

백엔드 데이터/offset_ms는 완전 정상. 응답 래핑만 누락.

## 4. 영향 범위

- **버그 노출(유일)**: admin 콘솔 recording 상세 Transcript 카드(`TranscriptCard.js`).
  recording 서브리소스 목록 응답이 raw 배열이라 transcript/transcribe를 표시하지 못하고
  "No transcript yet"을 띄운다.
- **영향 없음**: Case 상세 Linked Conversation 패널(`CaseCallTranscriptPanel.js`)은
  `service_agents/*` 엔드포인트(이미 GenerateListResponse로 래핑됨)를 소비하므로 정상 동작한다.
- **VOIP-1528(offset_ms)과의 관계**: 완전 별개. offset_ms 재설계는 정상 배포·동작 확인됨.
  이 버그는 응답 envelope(래핑) 누락이고 offset_ms는 데이터 필드 값 문제로 계층이 다르다.
  offset_ms 수정으로 데이터가 정상화되자 비로소 "표시 자체가 안 되는" 이 UI 버그가 드러났다.

## 5. 수정 대상 (예정)

| 파일 | 변경 |
|------|------|
| bin-api-manager/server/recordings.go | 두 핸들러(208, 276)를 GetRecordings와 동일하게 nextToken 생성 + GenerateListResponse 래핑 |
| bin-api-manager/server/recordings_test.go | 두 핸들러 테스트에 응답 본문(`{result:[...],next_page_token:...}`) 검증 추가 |

- 현행 `Test_GetRecordingsIdTranscribes`, `Test_GetRecordingsIdTranscripts`는 **상태코드(200)만
  검증하고 본문(w.Body)을 assert하지 않는다.** 따라서 래핑 변경만으로는 기존 테스트가 깨지지
  않는다. 테스트 갱신은 "기대값 수정"이 아니라, 래핑된 envelope를 **회귀 검증하기 위한 본문
  assertion 추가**가 목적이다(향후 다시 raw 배열로 회귀하는 것을 테스트가 잡도록).

- OpenAPI 스펙: 변경 불필요(이미 올바름). 신규 엔드포인트 아님 → RST 구조 갱신 불필요.
- serviceHandler(recording.go): 변경 불필요(슬라이스 반환 유지, 래핑은 server 계층 책임).
- nextToken 생성: GetRecordings 패턴 동일 — 마지막 요소의 TMCreate를
  `2006-01-02T15:04:05.000000Z` 포맷. 두 WebhookMessage 모두 TMCreate(*time.Time) 보유 확인됨.

## 7. 구현 설계

### 7.1 GetRecordingsIdTranscribes (recordings.go)
serviceHandler 호출 직후, `c.JSON(200, res)` 앞에 GetRecordings(:62-70)와 동일한
nextToken 생성 + 래핑을 삽입한다. 반환 변수명을 `res` → `tmps`로 바꾼다.

변경 전:
```go
	res, err := h.serviceHandler.RecordingTranscribeList(c.Request.Context(), a, target, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get transcribes for the recording. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
```
변경 후:
```go
	tmps, err := h.serviceHandler.RecordingTranscribeList(c.Request.Context(), a, target, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get transcribes for the recording. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
```

### 7.2 GetRecordingsIdTranscripts (recordings.go)
7.1과 동일 패턴. serviceHandler 호출은 `RecordingTranscriptList(...)`,
반환 슬라이스는 `[]*tmtranscript.WebhookMessage`(TMCreate 보유). 나머지 삽입 코드 동일.

### 7.3 GenerateListResponse 안전성 (변경 없음, 확인만)
`main.go:91`의 제네릭 `GenerateListResponse[T any](tmps []*T, nextToken string)`는
nil 슬라이스를 `[]`로 coerce하고(빈 목록도 `result:[]`로 보장), len==0이면 next_page_token을
빈 문자열로 둔다. 두 WebhookMessage 타입에 그대로 적용 가능.

### 7.4 프론트(square-admin) 무변경 (의도됨)
TranscriptCard.js는 이미 `res.result` / `res.next_page_token`을 기대하도록 작성돼 있다.
백엔드가 스펙대로 래핑하면 프론트는 **코드 변경 없이 자동 정상화**된다. 이 PR은 백엔드
(monorepo) 단일 PR이며 프론트 변경은 없다.

### 7.5 테스트 (recordings_test.go)
`Test_GetRecordingsIdTranscribes` / `Test_GetRecordingsIdTranscripts`는 현재 상태코드만
검증한다. 응답 본문을 파싱해 `result` 배열 길이·요소 id와 `next_page_token`을 assert하는
회귀 검증을 추가한다(향후 raw 배열로 회귀 시 실패하도록). mock이 반환하는 슬라이스의
마지막 요소 TMCreate로 계산되는 next_page_token 기대값도 함께 검증한다.

## 8. 진행 타당성

- 유효한 프로덕션 UI 버그이며 실측으로 재현됨.
- 스펙이 이미 정답을 정의하고 있어 수정 방향이 명확하고 위험 낮음(핸들러 2곳 + 테스트).
- 다른 정상 핸들러(GetRecordings)의 확립된 패턴을 그대로 따르므로 일관성 확보.
- 진행 타당.
