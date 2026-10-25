# VOIP-1652 bin-common-handler 단위 테스트 커버리지 75% 이상

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main accc03e 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile ./...` 를 다시 실측했고 테스트 실패 0건이다.
- 수기 합계(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 파일 제외, 같은 블록은 최대 count) 72.97% (7498/10276 statement), 원본 합계 51.96%. 티켓 수치(73.0%, 52.0%)와 일치한다.
- 76% 에는 수기 statement 7810개 이상이 필요하므로 약 312개를 더 커버해야 한다. 75% 는 7707개, 약 209개.
- 최근 변경 확인: 이 경로의 마지막 커버리지 PR 은 VOIP-1651 (#1578, 머지됨)이며 `pkg/requesthandler/rpc_table_test.go` 의 표 기반 하네스(`rpcCase`, `runRPCCases`, `runMarshalErrorCase`)를 남겼다. 열린 PR 중 이 경로를 바꾸는 것은 없다.
- 구조 분석: cmd/ 가 없는 라이브러리라 main 때문에 달성이 불가능한 부분이 없다. 미커버의 대부분은 `pkg/requesthandler` 의 RPC 클라이언트 래퍼(미커버 2232 statement)다. 이번 범위에서 제외하는 곳은 `rabbitmqhandler`(amqp 연결과 채널 필요, 미커버 272), `migrationhandler`(DSN 이 없으면 skip 되는 통합 테스트가 있어 CI 에서 미커버, 100), `sockhandler`(미커버 3, 총량에 영향 없음)이며 `databasehandler/mapping.go`(102)는 reflect 기반 순수 코드가 많아 테스트할 수 있으나 이번에는 비용 대비 효과로 제외하고 부족할 때의 보충 후보로 둔다.
- 0% 래퍼 파일(미커버 statement): contact_addresses 67, webchat_session 67, route_providercalls 62, timeline_analyses 56, webchat_message 56, timeline_sip 47, agent_password 38, contact_tags 31, contact_interactions 30. 합계 454. 모두 `sendRequest*` 와 `parseResponse` 만 쓰는 함수라 sockhandler mock 으로 결정적으로 테스트된다. 입력으로 도달할 수 없는 `json.Marshal` 오류 반환 약 11 statement 를 빼면 최대 약 443 이므로 9개 파일을 모두 채우면 약 77.3% 이고, 76% 까지 여유는 약 130 statement 다. 부족하면 `customer_customer.go`, `billing_accounts.go` 의 미커버(두 파일 합계 214 statement)와 `databasehandler/mapping.go` 의 순수 변환 함수를 보충한다.
- 목표: 인수 하한 75.0%, 권장 76% 이상(CI 소수점 차이 여유)이며 달성값은 실측 그대로 보고한다.
- 반증하지 못한 주장: 수치, "main 과 외부 의존 때문이 아니라 수기 래퍼 미커버가 대부분". 틀린 주장: 없음.

## 2. 설계

범위는 `bin-common-handler/pkg/requesthandler` 의 `_test.go` 신규 파일뿐이다. 제품 코드, go.mod, go.sum 은 바꾸지 않는다(새 모듈 불필요).

- 위 9개 파일마다 `<source>_test.go` 를 새로 만들고 기존 `runRPCCases` 로 래퍼별 success, transport error, non-2xx(404) 오류를 검증한다. 기대 request(queue, URI 전체, method, DataType, body JSON)와 기대 결과는 wire 계약에서 손으로 쓴 리터럴이고 응답을 그대로 기대값으로 되돌리는 어댑터는 쓰지 않는다. JSON body 의 map 키는 encoding/json 이 정렬하므로 결정적이다.
- 오류만 반환하는 `AgentV1PasswordForgot`, `AgentV1PasswordReset` 은 `(nil, err)` 어댑터로 감싼다. 성공 케이스의 의미는 queue, URI, body 리터럴 매칭에서 나온다(token, password 는 합성값).
- `ContactV1InteractionList`(반환 3개)는 `struct{Res []*PeerEvent; Next string}` 어댑터로 합친다. query 조합(peer, contact, address, size, token)마다 정렬된 URI 전체를 리터럴로 단언하고, nextToken 은 비-UTC 오프셋 timestamp 응답에 대해 UTC 포맷 리터럴로, 빈 결과는 body `[]` 와 빈 Data 두 형태로 검증한다.
- `ContactV1AddressUpdate` 는 fields 조합(target 만, is_primary 만, 둘 다, nil 또는 빈 맵, 타입 불일치, 빈 문자열 target)별로 body 리터럴(`{}` 포함)을 단언한다. `ContactV1AddressGet` 은 `result` 가 빈 배열일 때와 항목이 있을 때를 둘 다 검증한다.
- `RouteV1ProviderCallGets` 는 filter 2개 이상(`url.Values.Encode()` 키 정렬), nil filters, 빈 page_token 을 URI 리터럴로 고정한다. `WebchatV1SessionCreate` 는 id 가 `uuid.Nil` 일 때와 지정일 때, `WebchatV1SessionEnd` 는 POST 와 `ContentTypeNone` 과 빈 body 를 검증한다. 목록 계열 page_token 은 특수문자가 없는 값만 쓴다.
- `TimelineV1SIPPcapGet`(`[]byte`, `statusCustom`)은 합성 바이너리의 base64 리터럴 정상 케이스, 빈 data, status>=300, 본문 JSON 파손, base64 파손을 검증한다. `TimelineV1SIPAnalysisGet` body 는 키 정렬 리터럴이다.
- marshal 실패 경로는 실제로 도달 가능한 4곳만 `runMarshalErrorCase` 로 검증한다: `TimelineV1AnalysisList`, `WebchatV1MessageList`, `WebchatV1SessionList`(filters 에 chan 값), `RouteV1ProviderCallCreate`(`Action.Option` 맵에 chan 값). 나머지 `json.Marshal` 오류 분기는 uuid, string, bool, 고정 구조체 입력만 받아 도달할 수 없으므로 의도적으로 제외한다.
- 결함을 발견하면 고치지 않고 카드에만 적는다(결함을 고정하는 테스트는 만들지 않는다).

## 3. verify 항목

1. 수기 합계 75% 이상(권장 76% 이상), baseline 과 같은 명령과 제외 규칙으로 패키지별 전후 표를 PR 본문에 쓴다.
2. 루트 CLAUDE.md 의 5단계(`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run`)를 통과하고 go.mod, go.sum diff 가 없다. `git fetch origin main` 뒤 `scripts/check-test-conventions.sh`(Test_ 이름, testify 금지, gomock 컨트롤러 이름 mc)가 통과한다. 기존 테스트는 바꾸지 않는다. `git diff --stat origin/main...HEAD` 가 bin-common-handler/ 안의 신규 `_test.go` 와 이 문서뿐이다.
3. 변이 확인(scratch 사본에서만, worktree 금지): 파일마다 URI 한 글자, method, queue, 요청 필드 매핑 교체, `return nil, err` 를 `return nil, nil` 로, 분기 조건 반전(`>=300`, `len(res)>0`, 타입 단언 ok), InteractionList 의 `.UTC()` 제거와 포맷 변경, PcapGet 의 base64 디코딩을 변이하고 변이별 실패 테스트 이름을 표로 PR 본문에 남긴다. timeout 상수는 mock 이 ctx 를 검사하지 않으므로 검출 대상에서 제외한다.
4. 빈 호출과 자기비교 테스트가 없다(모든 기대값은 리터럴).
5. 테스트 값은 합성값만 쓴다. 고정 UUID 리터럴, 예약 도메인과 문서용 번호 대역만 쓰고 실제 전화번호, 이메일, 호스트, 비밀값, 주소 형태 DSN 은 쓰지 않는다.
