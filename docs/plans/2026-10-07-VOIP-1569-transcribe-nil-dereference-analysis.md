# VOIP-1569 transcribe 참조 리소스 조회 실패 시 nil 역참조 패닉 이슈 분석

- 상태: Draft (이슈 분석 리뷰 4회차 대기, 3회차 반영)
- 티켓: VOIP-1569
- 기준: origin/main 49ec4a06d
- 단계: 이슈 확인과 분석 (코드 변경 없음)

## 1. 결론

- 이슈는 유효하다. 현재 main 에서 재현 가능한 코드 결함이며, 운영 로그에서 실제 패닉이 관측되었다.
- 진행이 타당하다. 수정은 분기당 한 줄(`break`)이고, 서비스 중단 위험이 없으며, 호출자에게 500 대신 정확한 4xx 가 반환된다.
- 범위: 티켓 본문은 `transcribe.go` 의 한 함수만 적었으나, 코드 재점검에서 같은 결함이 `serviceagent_transcribe.go` 의 에이전트용 함수에도 있음을 확인했다. 같은 결함 클래스이고 같은 PR 에서 함께 고치는 것이 맞아 범위에 포함한다(티켓에 코멘트로 반영).

## 2. 코드 확인 결과

### 2.1 결함 위치 (4곳, 2개 함수)

`bin-api-manager/pkg/servicehandler/transcribe.go` 의 `transcribeGetResourceInfo` (189행부터).

- conference 분기: 216행 `if tmpErr != nil { err = tmpErr }` 뒤에 `break` 나 `return` 없이 219행 `tmpCustomerID = tmpResource.CustomerID` 로 진행한다.
- recording 분기: 225행의 같은 패턴 뒤 228행 `tmpCustomerID = tmpResource.CustomerID`.
- call 분기는 207-208행에서 `err = tmpErr` 다음 `break` 가 있어 정상이다.

`bin-api-manager/pkg/servicehandler/serviceagent_transcribe.go` 의 `transcribeGetResourceInfoForAgent` (155행부터).

- conference 분기: 181-184행, recording 분기: 190-193행에 같은 결함이 있다. call 분기(171-174행)는 `break` 가 있다.
- 이 함수는 `service_agents` API(상담사 권한)의 `ServiceAgentTranscribeStart`(118행)가 호출한다. 일반 상담사가 도달하는 경로라 admin/manager 전용 경로보다 노출이 넓다.

### 2.2 패닉이 나는 조건

- `conferenceGet` (`conference.go:23` 선언, 33행) 은 RPC 에러 시 `return nil, err` 를 반환한다.
- `recordingGet` (`recording.go:30` 선언, 40행) 은 RPC 에러 시 `return nil, err`, 삭제된 recording(`TMDelete != nil`)이면 `return nil, serviceerrors.ErrNotFound` 를 반환한다(recording.go:44-47).
- 즉 존재하지 않는 id 뿐 아니라 삭제된 recording id, backend 일시 장애에서도 `tmpResource` 가 nil 이 되어 역참조 패닉이 난다.

### 2.3 패닉이 호출자에게 주는 결과

- gin Recovery 가 복구해 500 으로 응답한다(운영 로그에서 확인). 프로세스는 유지된다.
- 수정 후 기대 응답(조회 실패의 종류에 따라 404 등 해당 상태코드이며 항상 404 는 아니다. 예: backend 일시 장애는 503 계열로 매핑될 수 있다): `transcribeGetResourceInfo` 는 에러를 `fmt.Errorf("could not pass the reference validation: %w", err)` 로 감싸 반환하고, `server/error_translate.go:41` 의 `translateToVoipbinError` 는 1단계에서 `errors.As` 로 `*cerrors.VoipbinError` 를 그대로 통과시킨다. 따라서 call-manager 가 보낸 typed 에러(예: RECORDING_NOT_FOUND)는 래핑되어도 원래 4xx 로 응답된다. 삭제된 recording 의 `serviceerrors.ErrNotFound` 는 같은 함수의 2단계에서 404 `RESOURCE_NOT_FOUND` 로 매핑된다. 에이전트용 함수는 래핑 없이 에러를 반환하며 같은 매핑을 거친다.

### 2.4 운영 증거

- bm-nyc-01 api-manager 컨테이너 로그: `panic recovered` 스택이 `transcribe.go:228`, `transcribe.go:159`, `server/transcribes.go:73` 을 가리킨다. 2026-10-06 22:49 UTC 부터 약 20초 동안 레플리카당 약 25회(api-validator 실행 중).
- 같은 컨테이너 로그에 `RECORDING_NOT_FOUND` 에러가 있다. 시각 대조는 하지 않았으므로 관련성은 추정이다.
- 호출 주체: api-validator 로 보인다. `monorepo-monitoring/api-validator/tests/scenarios/stt/test_stt_api.py` 등 `stt/` 의 모듈 4개(`test_stt_accuracy.py`, `test_stt_api.py`, `test_stt_events.py`, `test_stt_languages.py`)는 파일 앞부분에 `pytestmark = pytest.mark.skip(...)` 가 있으나 같은 파일 아래에서 `pytestmark = [pytest.mark.stt, ...]` 로 다시 대입되어 skip 이 무효이고 실제로 실행된다. 실제로 skip 되는 것은 `generated/test_transcribes_generated.py` 하나뿐이다. 실행되는 테스트 중 존재하지 않는 recording id(무작위 uuid 또는 nil uuid)로 POST /transcribes 를 보내는 테스트가 이 패닉 경로를 탄다(5절의 목록). 22:49 UTC 의 패닉 시각이 검증기 실행과 부합하는 정황이나 시각 대조는 하지 않았다.

### 2.5 기존 테스트 현황

- `transcribe_test.go` 의 `Test_TranscribeStart` 는 정상 케이스만 있고(call 레퍼런스) 조회 실패 케이스가 없다.
- `serviceagent_transcribe_test.go` 의 `Test_ServiceAgentTranscribeStart` 도 에러 케이스가 없다. 이 때문에 결함이 테스트에 걸리지 않았다.

## 3. 범위

포함.

- `transcribeGetResourceInfo`, `transcribeGetResourceInfoForAgent` 의 conference, recording 분기 4곳에 조회 실패 시 `break` 추가.
- 조회 실패 케이스 테스트 추가(두 함수, conference 와 recording 각각).
- api-validator(monorepo-monitoring) 의 위 5개 테스트 허용 코드에 404 추가(5절의 목록). 저장소가 달라 별도 PR 이다(구조적으로 필요한 분리).

제외.

- VOIP-1410(존재하는 빈 conference 에서 call-manager 응답 타임아웃으로 인한 500). 원인이 다르며 별도 티켓이다.
- VOIP-1568(accesskey 쿼리 로깅). 패닉 덤프에 쿼리가 찍히는 점은 같은 로그에서 비롯되지만 별개 이슈이며 다른 세션에서 처리 중이다.
- 같은 서비스의 다른 `err = tmpErr` 패턴: `grep` 결과 api-manager 에서 `transcribe.go` 와 `serviceagent_transcribe.go` 외에는 없다.

## 4. 대안과 판단

- 대안 A(채택): 각 분기에 `break` 추가. call 분기와 같은 방식이라 일관적이고 변경이 최소다.
- 대안 B: 분기마다 `return` 으로 즉시 반환. 동작은 같으나 현재 함수 구조(switch 뒤 단일 에러 처리 블록)와 일관되지 않아 채택하지 않는다.
- 대안 C: 공통 헬퍼로 두 함수를 통합. 두 함수는 권한 검사와 에러 래핑이 다르다(admin/manager 전용 대 상담사). 이번 결함과 무관한 리팩터링이라 오버엔지니어링으로 판단해 하지 않는다.

## 5. 위험과 수용 사항

- 동작 변화: 실패 조회가 500(패닉)에서 4xx 로 바뀐다. 이 변화는 의도한 것이며 클라이언트가 500 에 의존할 이유가 없다.
- api-validator 영향(확인함, 이번 수정이 검증기를 빨갛게 만든다): 아래 5개 테스트는 존재하지 않는 recording 으로 POST /transcribes 를 보내고 허용 코드가 `[400, 422, 500]` 이다. 현재는 이 버그의 500 으로 통과하고 있다. 수정 후에는 조회 실패 상태코드(call-manager 의 typed 에러면 404)가 반환되어 실패한다.
  - `stt/test_stt_api.py`: `test_create_transcribe_requires_reference_id`, `test_create_transcribe_requires_language`, `test_create_transcribe_requires_direction`. 본문 검증이 `c.BindJSON` 이라 누락 필드가 요청을 거부하지 않고 빈 값이나 기본값으로 서비스까지 도달한다(direction 은 handler 가 `both` 로 기본 보정). 그 뒤 참조 조회에서 패닉이 난다.
  - `stt/test_stt_languages.py`: `test_empty_language_rejected`, `test_invalid_direction_rejected`. 후자는 잘못된 direction 을 handler 가 거부하지 않고 `Normalize()` 로 `both` 로 보정해 계속 진행하므로 조회 단계에 도달한다(참조: `server/transcribes.go` 의 direction 처리).
  - 확정 방법: 검증기 저장소의 `tests/` 전체에서 `post("/transcribes"` 를 호출하는 함수를 AST 로 모두 추출해 허용 코드 목록을 대조했다. 404 를 허용하지 않고 500 을 허용하는 8개 중 3개(`test_create_transcribe_requires_reference_type`, `test_create_with_empty_body_returns_error`, `test_invalid_reference_type_rejected`)는 reference_type 이 비었거나 잘못된 값이라 `default` 분기의 `ErrInvalidArgument`(400)로 끝나 영향이 없고, 나머지 5개가 위 목록이다.
  - 나머지 테스트는 404 를 허용하거나(예: `test_create_transcribe_with_invalid_reference_returns_error`, `test_invalid_language_code_rejected`) 200 이면 정리하는 형태라 영향이 없다. `test_create_transcribe_requires_reference_type` 은 빈 reference_type 이 `default` 분기의 `ErrInvalidArgument`(400)로 처리되어 영향이 없다.
  - 따라서 api-validator 저장소(monorepo-monitoring)의 위 5개 테스트 허용 코드에 404 를 추가하는 변경이 필요하다. 저장소가 달라 별도 PR 이며, 서버 수정 배포 전후로 검증기가 빨갛게 되지 않도록 두 PR 의 순서를 조정한다(검증기 PR 을 먼저 머지해도 500 과 404 를 모두 허용하므로 안전하다).
- 다른 서비스 영향 없음: api-manager 한 곳의 변경이며 API 계약, 이벤트, DB 변경이 없다.

## 6. 설계 단계로 넘길 사항

- 테스트 케이스 설계(두 함수 4분기의 실패 케이스, 패닉이 아니라 에러 반환임을 단언, `break` 제거 시 패닉으로 실패하는지).
- 반환 에러가 상위(`TranscribeStart`, `ServiceAgentTranscribeStart`)에서 4xx 로 매핑되는지 확인하는 테스트의 수준.
- OpenAPI: `bin-openapi-manager/openapi/paths/transcribes/main.yaml` 의 POST 응답은 200/400/401/409/500 이고 403, 404 가 없다. `service_agents/transcribes.yaml` 의 POST 응답도 200/400/401/409/500 이며 403, 404 가 없다. 기존 403 도 미문서화인 선재 공백이라 이번 PR 에서 응답 코드를 추가할지 설계 단계에서 결정한다.
