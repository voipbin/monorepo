# VOIP-1652 bin-agent-manager 단위 테스트 커버리지 75% 이상 계획과 결과

## 1. 배경과 측정 방법

- 수기 코드 커버리지는 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 의 프로파일에서 `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고, 같은 블록이 여러 패키지 실행으로 중복되면 최대 count 로 합산해 계산합니다. `go test` 가 패키지별로 출력하는 퍼센트는 mock 과 생성 코드를 분모에 포함하므로 이 문서의 수치와 다릅니다.
- 시작 시점의 수기 합계는 74.23% (2221/2992 statement) 이고, 생성 코드와 mock 을 포함한 원본은 66.14% 입니다. 75% 는 2244, 76% 는 2274 statement 입니다.
- 패키지별 시작 값 (covered/total): cmd/agent-control 441/646, cmd/agent-manager 1/103, internal/config 57/65, models/agent 35/36, pkg/agenthandler 595/750, pkg/cachehandler 60/61, pkg/dbhandler 483/637, pkg/listenhandler 473/601, pkg/metricshandler 0/1, pkg/subscribehandler 76/92.
- cmd/agent-manager 의 main 은 외부 연결을 직접 만들어 단위 테스트 대상에서 제외합니다. 이 서비스는 그 제외 후에도 산술적으로 목표에 도달할 수 있습니다.
- 가장 큰 미커버는 pkg/listenhandler/v1_agents.go 의 요청 검증과 오류 응답 분기, pkg/agenthandler 의 db.go, direct_hash.go, event.go 의 오류 경로였습니다.

## 2. 범위

제품 코드는 바꾸지 않고 `bin-agent-manager` 안의 테스트 파일과 이 문서만 추가합니다. 정상적인 입력으로는 도달할 수 없는 분기(응답 직렬화 실패, 난수 생성 실패)는 대상에서 제외합니다.

## 3. 추가한 테스트

1. `pkg/listenhandler/v1_agents_validation_test.go`: 핸들러를 직접 호출합니다(라우터가 URI 형태를 먼저 걸러 URI 항목 수 검사에 닿지 않기 때문). 목록 조회의 URI 파싱 실패, 필터 파싱과 변환 실패, 목록 오류 응답(typed 오류, 일반 오류, not found), 잘못된 UUID, 각 핸들러의 경계 URI(항목 수 1개 부족, 본문은 유효), 잘못된 JSON 본문, 핸들러 오류의 상태 코드와 응답 본문. 호출이 없어야 하는 케이스는 mock 에 기대를 두지 않고, 호출이 있는 케이스는 인자를 정확히 일치시킵니다(ctx 만 `gomock.Any()`).
2. `pkg/agenthandler/db_error_test.go`: dbCreate(지원하지 않는 ring method, 해시 생성 실패, direct 생성 실패, 저장 실패 시 고아 direct 정리와 원 오류 유지, 중복 주소의 typed 오류, 저장 후 조회 실패), dbDelete(조회 실패, 삭제 실패, 삭제 후 조회 실패, direct 삭제 실패에도 계속 진행), dbLogin 의 두 오류, dbUpdateInfo, dbUpdatePassword, dbUpdatePermission, dbUpdateTagIDs, dbUpdateAddresses(중복 주소 변환 포함), dbUpdateStatus 의 저장 실패와 재조회 실패. 선언하지 않은 호출(이벤트 발행 등)은 mock 이 실패시킵니다.
3. `pkg/agenthandler/direct_hash_error_test.go`: DirectHashRegenerate 의 단계별 오류와 래핑 메시지, 마지막 재조회 실패는 래핑 없이 반환됨.
4. `pkg/agenthandler/event_error_test.go`: 그룹콜 진행 이벤트에서 대상이 에이전트가 아님, UUID 가 아님, 조회 실패, 상태 갱신 실패 뒤에도 다음 대상이 처리됨. 고객 삭제 이벤트의 목록 실패와 한 에이전트 삭제 실패 후 계속 진행. 고객 생성 이벤트의 관리자 에이전트 생성 실패.

테스트 컨벤션은 `Test_` 이름, gomock 컨트롤러 이름 `mc`, testify 미사용, table-driven, 합성 테스트 값입니다.

## 4. 결과

- 수기 합계 79.08% (2366/2992 statement), 원본 70.46%. pkg/agenthandler 91.2%, pkg/listenhandler 87.9%, 나머지 패키지는 시작 값과 같습니다(pkg/subscribehandler 는 타이밍에 따라 ±1 statement 변동).
- 변이 검사: 제품 코드의 분기를 복사본에서 바꾸는 변이 26건(URI 항목 수 경계, 잘못된 UUID 검사 제거, 400 과 500 응답 교체, 오류 응답 경로 교체, 필터 변환 응답 교체, 본문 파싱 오류 처리 제거, 고아 direct 정리 제거, 중복 주소 변환 제거와 비교 방식 변경, 로그인 오류 메시지 교환, direct 삭제 오류 시 중단, direct 삭제 가드 제거, 각 저장 오류 삼킴, 계속 진행 분기를 반환으로 변경, 래핑 변경)을 새 테스트 파일만 실행해 확인했고 26건 모두 테스트 실패로 검출되었습니다.
- 게이트: 전체 테스트 통과, `golangci-lint run` 0건, `scripts/check-test-conventions.sh` 통과, go.mod 와 go.sum 변경 없음, 변경 파일은 모두 bin-agent-manager/ 안에 있습니다.
