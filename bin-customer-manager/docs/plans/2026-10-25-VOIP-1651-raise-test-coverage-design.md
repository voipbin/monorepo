# VOIP-1651: bin-customer-manager unit test coverage to 70%

## 1. 이슈 분석 (착수 전 티켓 검토)
- 기준선(origin/main 99b59c386, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): 수기 2132/3095 = 68.89%, 원본 2132/3460 = 61.62%, 테스트 실패 0건. 수기 제외 기준은 `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일이며 같은 블록은 최대 count 로 합친다. 70% 까지 34개, 71% 까지 약 65개 부족.
- 미커버 분포(수기, statement): cmd/customer-control 594, cmd/customer-manager 99, pkg/dbhandler 83, pkg/listenhandler 101, pkg/customerhandler 55, pkg/accesskeyhandler 15, internal/config 8.
- cmd 는 `run*` 이 `initHandler` 로 실제 DB, Redis, RabbitMQ 연결을 열어 단위 테스트가 불가하다. 순수 헬퍼 `resolveUUID`, `resolveString`, `printJSON` 과 명령 트리 구성만 대상으로 한다.
- 서비스 핸들러와 listenhandler 의 미커버는 대부분 도달 가능한 오류 경로(하위 호출 오류 전파, 잘못된 요청 본문 400, 오류 매핑 404/400)다. dbhandler 는 `ToSql`/`PrepareFields` 실패처럼 도달 불가한 분기를 제외하면 DB 실행 오류와 캐시 오류 경로가 도달 가능하다.
- 도달 불가로 제외: Marshal 실패 500, squirrel ToSql 실패, `crypto/rand` 실패.

## 2. 설계
테스트 파일만 추가한다. 제품 코드, 시그니처, 노출 범위는 바꾸지 않는다. 합성값만 쓰고 외부 호출과 네트워크는 쓰지 않는다.

1. pkg/customerhandler/error_paths_test.go: Get 의 NotFound 가 `CUSTOMER_NOT_FOUND` 로 래핑되고 그 외 오류는 그대로 전달, Create 의 secret 생성, CustomerCreate, 재조회 실패, Delete 의 Get/dbDelete/재조회 실패, dbUpdateBasicInfo, dbUpdateBillingAccountID, UpdateMetadata 의 갱신과 재조회 실패(재조회 실패는 불투명 오류), email 중복 검사의 List 오류, agent 조회 오류, 이미 사용 중인 email 이 false, Signup 의 secret 생성과 재조회 실패, 이메일 검증의 락 획득 오류가 내부 오류로 변환, 토큰 삭제 실패와 비밀번호 재설정 요청 실패가 비치명이라 verified 결과가 유지됨. 실패 분기마다 이후 호출(예: Update 후 Publish)이 없음을 mock 기대 부재로 단언한다.
2. pkg/accesskeyhandler/error_paths_test.go: List 오류, Get 의 NotFound 가 `ACCESSKEY_NOT_FOUND` 로 래핑되고 그 외는 전달, 고객별 목록 오류, 토큰 생성 오류, 생성 후 재조회 오류, Delete 의 삭제와 재조회 실패(불투명 유지, `errors.Is(ErrNotFound)` 가 거짓), Update 재조회 실패.
3. pkg/listenhandler/error_paths_test.go: customers 와 accesskeys 각 엔드포인트의 잘못된 JSON 본문 400, 핸들러 오류가 errorResponse 로 매핑(typed VoipbinError 와 ErrNotFound 는 해당 상태, 일반 오류는 500 또는 기존 동작을 먼저 코드로 확인), 잘못된 필터 400, `processRequest` 의 미지원 URI 오류 매핑(typed, ErrNotFound, 기타 400), `Run` 의 QueueCreate 실패 오류 래핑과 성공 시 ConsumeRPC 시작(채널로 동기화해 레이스 없이 검증).
4. pkg/dbhandler/error_paths_test.go: 닫힌 sqlite 연결을 가진 핸들러로 Customer/Accesskey 의 Create, Get, List, Update, Delete 등 쿼리 실행 오류 전파, 캐시 Set 오류 전파, 조회 후 캐시 갱신 오류, 행 스캔 오류. 공유 인메모리 DB 를 닫지 않도록 테스트 전용 별도 연결을 쓴다.
5. cmd/customer-control/main_test.go: resolveUUID(빈 값, 잘못된 형식, 정상), resolveString(빈 값, 정상), printJSON(정상 출력 캡처, Marshal 불가 값 오류), initCommand 의 하위 명령 이름과 필수 안내 구성. viper 전역 상태는 각 테스트에서 설정하고 정리한다.
6. internal/config: LoadGlobalConfig 가 viper 값을 Config 로 옮기는지(once 전역이므로 한 번만 호출). 부족하면 cmd/customer-manager 의 순수 헬퍼.

## 3. verify 항목
- 수기 합계 71% 이상(목표), 최소 70.0%. 패키지별 전후 표, 원본 합계, 제외 방법을 PR 본문에 기재.
- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, `-race` 로 신규 패키지 테스트 실행, `scripts/check-test-conventions.sh`, `git diff --stat origin/main...HEAD` 가 bin-customer-manager/ 안에만 있음.
- 변이 검사 표본(scratch 복사본): NotFound 래핑 제거, 재조회 실패를 오류 전파로 변경, 400 반환 변경, 락 오류 변환 제거, dbhandler 오류 감싸기 제거를 각각 적용해 새 테스트가 실패하는지 확인.
- 테스트 중 발견한 결함은 고치지 않고 카드 코멘트와 완료 요약에만 기록.
- VOIP-1416 과 겹치는 패키지는 PR 본문에 명시만 한다.
