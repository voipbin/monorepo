# bin-registrar-manager 단위 테스트 커버리지 60% 계획 (VOIP-1647)

## 1. 티켓 검토 (이슈 분석)

판정: 유효. origin/main d68e695b5 기준 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외, 같은 블록 중복은 최대 count) 합계 56.31% (2058/3655, 독립 재계산 시 statement 1개 차이 가능), 원본 47.88%. 60% 까지 약 136 statement 필요.

| 패키지 | statement | 미커버 | 비고 |
|---|---|---|---|
| cmd/registrar-control | 922 | 718 | cobra 명령과 실제 DB, Redis, RabbitMQ 초기화에 묶임 |
| pkg/dbhandler | 926 | 190 | sqlmock 기반 기존 테스트 있음 |
| pkg/cachehandler | 184 | 184 | 테스트 0건. go-redis 클라이언트를 struct 필드로 직접 보유 |
| pkg/listenhandler | 627 | 161 | processV1ExtensionsIDGet 0%, 나머지는 NewListenHandler/Run |
| cmd/registrar-manager | 156 | 148 | main, 데몬 기동. 제외 |
| pkg/extensionhandler | 256 | 81 | Get 50%, GetByExtension 0%, 오류 경로 |
| pkg/trunkhandler | 157 | 41 | Get 50% |
| pkg/subscribehandler | 75 | 22 | customer_deleted 오류 경로 55.6% |

중복, 이미 해결 여부: registrar 커버리지 PR 은 과거 #510 (NOJIRA-Test-coverage-registrar-manager) 이고 #360 (low coverage 서비스 일괄) 도 있으나 둘 다 머지된 과거 변경이며 이후 pkg/cmd 변경은 lint, migration, ctx 전달뿐으로 전제가 바뀌지 않음.

## 2. 구조 분석

- cachehandler: `handler{Cache *redis.Client}` 를 테스트에서 직접 구성할 수 있어 제품 코드 변경 없이 miniredis 로 실제 Redis 프로토콜 동작(키 형식, TTL, 직렬화 왕복, 오류 전파)을 검증 가능. 가장 큰 단일 이득(184 statement).
- cmd/*: main 과 cobra 실행이 외부 연결(DB, Redis, RabbitMQ) 생성에 묶여 있어 제품 코드 변경 없이는 단위 테스트가 부적절하므로 이번 범위에서 제외.
- miniredis 는 이 모듈 go.mod 에 없음. 테스트 전용 의존성(github.com/alicebob/miniredis/v2 v2.36.1, 다수 서비스가 쓰는 v2.36.1 과 같은 버전)으로 go.mod/go.sum 에 추가. 제품 코드 동작 변경 없음.

## 3. 테스트 계획 (코드는 구현 단계에서 작성)

1. pkg/cachehandler/handler_test.go (miniredis, 같은 패키지에서 `handler{Cache: redis.NewClient(...)}` 직접 구성)
   - Set 후 Get 왕복: Ast Endpoint/Auth/AOR, Extension, Trunk, CustomerDomain, AstContacts. Extension 은 3개 키(ID, EndpointID, CustomerID+Extension), Trunk 는 2개 키(ID, DomainName)를 각각 개별 Get 으로 확인.
   - 키 형식 고정: 기대 키 문자열을 테스트에 리터럴로 쓰고(제품 Sprintf 재사용 금지) miniredis 에 해당 키가 있는지 확인. TTL 은 Ast Endpoint/Auth/AOR 와 AstContacts 가 정확히 3분, 나머지 정확히 24h 인지 각각 단언(범위 비교 금지).
   - 삭제 후 Get 이 redis.Nil, TrunkDel 은 두 키 모두 제거, 미존재 키 삭제는 오류 아님.
   - 오류 경로: go-redis Hook(`AddHook`)으로 N번째 set/del 명령만 실패시켜 다중 키 쓰기/삭제 중 일부 명령이 실패하는 경우와 각 Del/Set 함수의 오류 전파를 검증. Hook 은 명령 이름(set, del)으로 걸러 센 카운터를 사용(Ping 이 섞이지 않음). 서버 종료 케이스는 Connect 와 소수 대표 함수에만 사용.
   - Get 오류: 손상된 JSON 과 타입 불일치(객체 대상에 배열, 배열 대상 AstContactsGet 에는 객체)를 각 Get 함수에서 unmarshal 오류로 전파하는지 확인.
   - AstContactsGet 은 서로 다른 contact 여러 건이 각각 복원되는지 확인. AstContactsSet 에 nil 슬라이스를 넣으면 JSON null 이 저장되므로 Get 이 빈 슬라이스를 반환하는 경계 확인.
   - 테스트는 ID 가 채워진 입력을 사용(제품 코드 변경 금지).
   - NewHandler 는 지연 연결이라 네트워크 없이 Addr, DB, Password 반영을 확인.
   - Connect: 정상 Ping 과 서버 종료 후 오류.
2. extensionhandler: 기존 Test_Get, Test_ExtensionGet 과 이름이 겹치지 않도록 Test_Get_notFound, Test_Get_dbError, Test_GetByExtension 등 새 이름을 쓴다. Get 의 ErrNotFound 가 NotFound 오류로 변환(코드 단언), 일반 오류는 NotFound 로 변환되지 않고 그대로 전달, GetByExtension 도 정상, NotFound, 일반 오류.
3. trunkhandler: 기존 Test_Get, Test_GetByDomainName 과 이름이 겹치지 않게 Test_Get_notFound, Test_Get_dbError, Test_GetByDomainName_notFound, Test_GetByDomainName_dbError 로 Get 과 GetByDomainName 의 ErrNotFound 변환과 일반 오류 비변환.
4. listenhandler: processV1ExtensionsIDGet 정상(200, JSON 본문과 id 파싱), 핸들러 오류 전파, 잘못된 uuid 경로(파싱 불가 id)의 경계값.
5. subscribehandler: customer_deleted 에서 trunk, extension, customerdomain 핸들러 각각 실패 시 오류 전파와 이후 핸들러 미호출(gomock 호출 횟수로 단언). 잘못된 JSON 입력의 unmarshal 오류(customer_deleted, customer_created 둘 다)와 customer_created 핸들러 오류 전파도 포함.
6. dbhandler(sqlmock, 기존 패턴, 필수): ExtensionCountByCustomerID, TrunkCountByCustomerID, TrunkDelete 에 대해 정상, DB 오류, 0건 경계값. TrunkDelete 는 성공 후 캐시를 갱신하므로 sqlmock 에 UPDATE 뒤 SELECT 와 cache mock 기대가 필요하고 캐시 갱신 오류는 호출 결과에 영향을 주지 않는다(오류 확인은 prepare/Exec 실패로). Count 쿼리의 소프트 삭제 조건은 sqlmock ExpectQuery 를 `tm_delete IS NULL` 포함 정규식 또는 QueryMatcherEqual 로 고정해 검출. 추가로 AstContactsSetToCache, AstContactDeleteFromCache 와 *ToCache/*FromCache 류의 캐시 오류 전파를 cache mock 이 오류를 돌려주게 하여 확인.

예상 증분: cachehandler 약 175 + 나머지 약 55 = 약 225 statement, 수기 합계 약 62%. 목표 61% 이상 미달 시 구현 단계에서 dbhandler, extensionhandler 오류 경로를 추가한다.

## 4. verify 항목

- 수기 합계 61% 이상. 계산 규칙은 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 파일 제외, 같은 블록 중복은 최대 count 이며 scratch 의 계산 스크립트(PR 본문에 규칙을 서술)로 재현. 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile`.
- 기존 테스트 전부 통과, `go mod tidy`, `go vet`, `golangci-lint run`. 로컬에서 `git fetch origin main` 후 `scripts/check-test-conventions.sh` 통과(Test_ 이름, 값 비교는 reflect.DeepEqual 과 t.Errorf(testify 금지), gomock 컨트롤러 이름 mc 는 gomock 을 쓰는 신규 테스트에 해당).
- 변이 검사: 핵심 동작(키 형식, TTL, 키 단위 쓰기/삭제, NotFound 변환과 일반 오류 전달, 소프트 삭제 조건, 오류 전파)을 임시 복사본에서 각각 변형해 새 테스트가 실패하는지 확인.
- 변경 파일은 bin-registrar-manager/ 안의 `_test.go`, docs/plans, go.mod, go.sum 뿐(`git diff --stat origin/main...HEAD`).
- 합성 값만 사용, 네트워크와 외부 비용 호출 없음.
