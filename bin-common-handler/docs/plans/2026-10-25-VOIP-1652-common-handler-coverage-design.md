# VOIP-1652 bin-common-handler 단위 테스트 커버리지 75% 이상

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main accc03e 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile ./...` 를 다시 실측했고 테스트 실패 0건이다.
- 수기 합계(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 파일 제외, 같은 블록은 최대 count) 72.97% (7498/10276 statement), 원본 합계 51.96%. 티켓 수치(73.0%, 52.0%)와 일치한다.
- 76% 에는 수기 statement 7810개 이상이 필요하므로 약 312개를 더 커버해야 한다. 75% 는 7707개, 약 209개.
- 최근 변경 확인: 이 경로의 마지막 커버리지 PR 은 VOIP-1651 (#1578, 머지됨)이며 `pkg/requesthandler/rpc_table_test.go` 의 표 기반 하네스(`rpcCase`, `runRPCCases`, `runMarshalErrorCase`)를 남겼다. 열린 PR 중 이 경로를 바꾸는 것은 없다.
- 구조 분석: cmd/ 가 없는 라이브러리라 main 때문에 달성이 불가능한 부분이 없다. 미커버의 대부분은 `pkg/requesthandler` 의 RPC 클라이언트 래퍼(미커버 2232 statement)이며 `rabbitmqhandler`, `migrationhandler`, `databasehandler/mapping.go` 는 외부 브로커나 DB 가 필요해 이번 범위에서 제외한다.
- 0% 래퍼 파일(미커버 statement): contact_addresses 67, webchat_session 67, route_providercalls 62, timeline_analyses 56, webchat_message 56, timeline_sip 47, agent_password 38, contact_tags 31, contact_interactions 30. 합계 454. 모두 `sendRequest*` 와 `parseResponse` 만 쓰는 순수 함수라 sockhandler mock 으로 결정적으로 테스트된다.
- 반증하지 못한 주장: 수치, "main 과 외부 의존 때문이 아니라 수기 래퍼 미커버가 대부분". 틀린 주장: 없음.

## 2. 설계

범위는 `bin-common-handler/pkg/requesthandler` 의 `_test.go` 신규 파일뿐이다. 제품 코드, go.mod, go.sum 은 바꾸지 않는다(miniredis 등 새 모듈 불필요).

- 위 9개 파일마다 `<source>_test.go` 를 새로 만들고 기존 `runRPCCases` 로 래퍼별 success, transport error(RequestPublish 오류 전파, ErrNotFound 로 오분류되지 않음), non-2xx(404) status 오류를 검증한다. 기대 request(queue, URI, method, DataType, body JSON)와 기대 결과는 wire 계약에서 손으로 쓴 리터럴이다.
- 오류만 반환하는 `AgentV1PasswordForgot`, `AgentV1PasswordReset` 은 `call` 이 `(nil, err)` 를 돌려주는 어댑터로 감싼다.
- 입력별 분기: `ContactV1InteractionList` 는 query 조립(peer, contact, address, 페이지 인자 유무)과 nextToken 계산(빈 결과, 마지막 timestamp UTC 포맷), `ContactV1AddressUpdate` 는 fields 맵의 타입 불일치(문자열이 아닌 target, bool 이 아닌 is_primary)가 무시되는 경계, `TimelineV1SIPPcapGet` 은 status>=300, 본문 JSON 파손, base64 파손 오류를 각각 검증한다. `RouteV1ProviderCallGets` 는 filter 가 한 개일 때 query 가 결정적이다.
- marshal 실패 경로: `json.Marshal` 이 실패하는 입력(chan 값을 가진 filters)을 `runMarshalErrorCase` 로 확인한다(List 계열).
- 결함을 발견하면 고치지 않고 카드에만 적는다.

## 3. verify 항목

1. 수기 합계 76% 이상(목표 77%), 패키지별 전후 표를 PR 본문에 쓴다. 측정은 baseline 과 같은 명령과 제외 규칙.
2. `go test ./...`, `golangci-lint run`, `scripts/check-test-conventions.sh` 통과, 기존 테스트 변경 없음, `git diff --stat origin/main...HEAD` 가 bin-common-handler/ 안의 `_test.go` 와 이 문서뿐.
3. 변이 확인(scratch 사본): 래퍼의 URI, method, 필드 매핑, 오류 전파 한 줄씩을 바꿨을 때 새 테스트가 실패해야 한다.
4. 빈 호출이나 자기비교 테스트가 없음(모든 기대값이 리터럴).
5. 테스트 값은 합성값만 쓰고 비밀값, 주소 형태 DSN 이 없다.
