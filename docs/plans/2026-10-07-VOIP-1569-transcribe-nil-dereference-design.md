# VOIP-1569 transcribe 참조 리소스 조회 실패 시 nil 역참조 수정 설계

- 상태: Draft (설계 리뷰 2회차 대기, 1회차 반영)
- 티켓: VOIP-1569
- 선행 문서: docs/plans/2026-10-07-VOIP-1569-transcribe-nil-dereference-analysis.md (이슈 분석, 범위와 사실 확정)
- 범위: 모노레포 PR 하나(서버 수정과 테스트)와 monorepo-monitoring PR 하나(api-validator 허용 코드). 저장소가 달라 PR 이 둘이다.

## 1. 목표와 비목표

목표.

- conference, recording 참조 조회가 실패해도 패닉하지 않고 조회 에러를 호출자에게 반환한다(500 대신 조회 실패에 해당하는 상태코드).
- 같은 결함이 있는 두 함수(`transcribeGetResourceInfo`, `transcribeGetResourceInfoForAgent`)를 모두 고친다.
- 수정으로 빨갛게 되는 api-validator 테스트 5개를 함께 맞춘다.

비목표.

- 두 함수의 공통 헬퍼 통합, 에러 래핑 방식 통일(분석 문서 4절의 대안 C, 오버엔지니어링).
- OpenAPI 응답 코드(403, 404) 문서화(선재 공백, D4 참조).
- VOIP-1410(빈 conference 의 external-media 타임아웃)과 VOIP-1568(accesskey 쿼리 로그) 처리.
- api-validator 의 skip 마커 정리(`pytestmark` 재대입으로 무효가 된 skip), 허용 코드에서 500 제거.

## 2. 결정 사항

### D1. 서버 수정: 4개 분기에 `break` 추가

`switch` 안에서 조회 실패 시 call 분기와 같은 방식으로 `break` 를 넣는다. `break` 는 `switch` 를 벗어나 뒤의 `if err != nil` 블록이 기존대로 에러를 반환하게 한다.

| 위치 | 변경 |
|---|---|
| bin-api-manager/pkg/servicehandler/transcribe.go, `transcribeGetResourceInfo` conference 분기(216-218행) | `err = tmpErr` 다음에 `break` 추가 |
| 같은 함수 recording 분기(225-227행) | 같음 |
| bin-api-manager/pkg/servicehandler/serviceagent_transcribe.go, `transcribeGetResourceInfoForAgent` conference 분기(181-183행) | 같음 |
| 같은 함수 recording 분기(190-192행) | 같음 |

- 에러 처리 블록과 메시지, 반환 형태는 바꾸지 않는다. 관리자용 함수는 `could not pass the reference validation: %w` 로 래핑해 반환하고, 상담사용은 래핑 없이 반환한다(기존 동작 그대로).
- 로그 호출과 변수 이름은 바꾸지 않는다.

### D2. 테스트: 조회 실패 케이스 추가

기존 `Test_TranscribeStart` 와 `Test_ServiceAgentTranscribeStart` 는 정상 케이스 하나씩이라 구조가 단순하다. 같은 파일에 실패 전용 테스트를 새로 추가한다(기존 테스트의 성공 경로 단언과 섞지 않는다).

- `transcribe_test.go`: `Test_TranscribeStart_referenceLookupFailure`
- `serviceagent_transcribe_test.go`: `Test_ServiceAgentTranscribeStart_referenceLookupFailure`

두 테스트의 공통 구조는 기존과 같다. 테이블 테스트, `mc := gomock.NewController(t)`(저장소 컨벤션 게이트가 컨트롤러 이름 `mc` 를 요구), `serviceHandler{reqHandler: mockReq, dbHandler: mockDB}`.

케이스.

| 케이스 | 목 설정 | 기대 |
|---|---|---|
| conference RPC 에러 | `ConferenceV1ConferenceGet(ctx, id)` 가 `(nil, errFake)` 반환 | 에러 반환, `errors.Is(err, errFake)` 참, 결과 nil |
| recording RPC 에러 | `CallV1RecordingGet(ctx, id)` 가 `(nil, errFake)` 반환 | 같음 |
| recording 삭제됨 | `CallV1RecordingGet` 이 `TMDelete` 가 설정된 recording 반환 | 에러 반환, `errors.Is(err, serviceerrors.ErrNotFound)` 참 |
| recording typed 에러 | `CallV1RecordingGet` 이 `cerrors.NotFound(...)` 반환 | `var ve *cerrors.VoipbinError` 를 선언해 `errors.As(err, &ve)` 가 참(래핑 뒤에도 typed 에러 유지) |

- 모든 케이스에서 `TranscribeV1TranscribeStart` 의 기대를 설정하지 않는다. gomock 은 예상 밖 호출을 실패시키므로, 조회 실패 후 transcribe 요청이 나가지 않음을 별도 단언 없이 보장한다.
- 에이전트 신원은 해당 함수의 권한 검사를 통과하는 값을 쓴다(`TranscribeStart` 는 `PermissionCustomerAdmin`, `ServiceAgentTranscribeStart` 는 `PermissionCustomerAgent`, 기존 테스트와 같은 값).
- 패닉 검증: 수정 전 코드에서는 nil 역참조 패닉으로 테스트가 실패한다. 별도의 `recover` 를 두지 않는다(패닉이 곧 실패이며 원인이 로그에 그대로 보인다).
- 변이 시험(구현 단계에서 수행하고 PR 본문에 결과를 적는다): 4곳의 `break` 를 각각 제거해, 해당 함수의 해당 분기를 다루는 케이스가 실패하는지 확인한다. 기대 매핑은 다음과 같다. `transcribeGetResourceInfo` conference 는 conference 케이스, recording 은 recording 의 RPC 에러, 삭제됨, typed 에러 세 케이스가 모두 잡고(세 케이스는 변이 검출 관점에서는 서로 중복이며, 삭제됨과 typed 는 반환 에러의 종류를 확인하는 역할이다), 상담사용 함수도 같은 구조다. 패닉은 테스트 바이너리 전체를 중단시키므로 변이 시험은 케이스별로 `-run` 을 지정해 개별 확인하고, 결과를 PR 본문에 적는다.
- HTTP 상태코드로의 매핑(`translateToVoipbinError`)은 기존 서버 에러 변환 테스트가 다루는 영역이라 이 PR 에서 다시 테스트하지 않는다. 대신 위 typed 에러 케이스가 래핑 뒤에도 타입이 유지됨을 보장한다.

### D3. api-validator 수정 (monorepo-monitoring 별도 PR)

분석 문서 5절의 5개 테스트의 허용 코드에 404 를 추가한다. 500 은 제거하지 않는다.

- `tests/scenarios/stt/test_stt_api.py`: `test_create_transcribe_requires_reference_id`, `test_create_transcribe_requires_language`, `test_create_transcribe_requires_direction`
- `tests/scenarios/stt/test_stt_languages.py`: `test_empty_language_rejected`, `test_invalid_direction_rejected`
- PR 운영: monorepo-monitoring 의 CLAUDE.md 에는 브랜치와 PR 규칙이 없으므로 전역 규칙을 따른다. 작업은 해당 저장소의 worktree 에서 하고, 브랜치는 `VOIP-1569-Allow-404-in-transcribe-validator-tests`, PR 제목은 브랜치명과 같게, 본문은 서술 한 문단과 `api-validator:` 접두 불릿(마크다운 헤더, Test plan, AI 속성 없음), 커밋 작성자는 pchero21@gmail.com 으로 한다. main 에 직접 푸시하지 않는다.
- 허용 목록은 `[400, 404, 422, 500]` 로 한다. 주석은 "조회 실패는 404, 검증 실패는 400, 서버 수정 이전의 구버전은 500" 이라는 사실에 맞게 한 줄로 갱신한다.
- 500 을 남기는 이유: 서버 수정 배포와 검증기 배포 순서에 관계없이 검증기가 빨갛게 되지 않게 한다. 500 의 제거는 서버 수정이 운영에 반영된 뒤 별도로 판단할 일이며 이번 범위가 아니다(오버엔지니어링 지양).
- 이 변경은 검증기가 서버 버그를 허용하도록 하는 것이 아니라 이미 허용 중인 500 에 올바른 응답을 추가하는 것이다. 이 저장소에는 `stt` 테스트 전반이 `[... 404, 500]` 형태를 이미 쓰고 있어 일관적이다.

### D4. OpenAPI 와 문서

- `bin-openapi-manager` 의 POST /transcribes, POST /service_agents/transcribes 응답에는 403, 404 가 없다. 이 공백은 이번 결함과 무관하게 기존부터 있었다(403 도 문서화되어 있지 않다).
- 이번 PR 에서는 OpenAPI 를 바꾸지 않는다. 바꾸면 생성 코드(`go generate`)와 docsdev 재빌드(Sphinx, `git add -f`)가 필요해 변경 범위가 크게 늘고, 이번 결함의 수정과 독립적이다. 공백은 이 설계 문서와 PR 본문에 기록만 하고, 필요하면 별도 티켓으로 다룬다.

### D5. 배포와 호환

- 모노레포 PR 은 api-manager 한 서비스의 동작 변경이다. API 계약, 이벤트, DB 변경이 없다.
- 동작 변화: 존재하지 않는 recording, conference id 와 삭제된 recording 으로의 POST /transcribes 가 500(패닉)에서 조회 실패 상태코드(typed 에러는 404 등, 일시 장애는 503 계열)로 바뀐다.
- 배포 순서: 제약이 없다. 검증기 PR 이 먼저 나가면 허용 목록이 넓어질 뿐이라 안전하다. 서버 수정이 먼저 나가면 검증기 5개 테스트가 반환 코드를 404 로 받아 실패하므로, 검증기 PR 을 먼저 머지하거나 같은 시점에 배포한다(권고 순서).
- 롤백: 서버 수정을 되돌려도 허용 목록에 500 이 남아 있어 검증기가 영향을 받지 않는다.

## 3. 변경 파일 목록

모노레포.

- bin-api-manager/pkg/servicehandler/transcribe.go
- bin-api-manager/pkg/servicehandler/serviceagent_transcribe.go
- bin-api-manager/pkg/servicehandler/transcribe_test.go
- bin-api-manager/pkg/servicehandler/serviceagent_transcribe_test.go
- docs/plans 의 분석 문서와 이 설계 문서
- 변경 없음: OpenAPI, 생성물, docsdev

monorepo-monitoring.

- api-validator/tests/scenarios/stt/test_stt_api.py
- api-validator/tests/scenarios/stt/test_stt_languages.py

## 4. 위험

- 검증기가 반환 코드를 404 로 받는다는 전제: 분석 문서 5절에서 call-manager 의 recording 조회 경로를 추적해 nil UUID 와 무작위 UUID 모두 typed `RECORDING_NOT_FOUND`(404)가 됨을 확인했다. 일시 장애 시 503 이 나올 수 있으나 검증기 목록에 503 은 없다. 이는 장애 시에만 발생하며 이번 변경 이전의 500 과 같은 성격이라 수용한다.
- 같은 PR 에서 5곳을 한 번에 확인할 수 있도록 PR 본문에 변이 시험 결과를 적는다.
- 검증기 저장소의 skip 마커 무효화(재대입) 문제는 이번 범위가 아니며 발견만 기록한다.
