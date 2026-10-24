# bin-registrar-manager 단위 테스트 커버리지 70% 계획 (VOIP-1651)

## 1. 티켓 검토 (이슈 분석)

판정: 유효. origin/main 99b59c386 기준 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외, 같은 블록 중복은 최대 count) 합계 69.00% (2522/3655, 같은 코드에서도 subscribehandler/main.go 117행 블록 때문에 ±1 statement 변동 가능: 68.97%/69.00%), 원본 58.7% (측정 기준은 HEAD 와 같은 트리이며 수치는 `go test` 패키지별 출력이 아니라 생성 코드와 mock 을 뺀 계산 스크립트 기준이라 `go test` 출력과 다름, 예: dbhandler 65.5% 대 84.0%). 71% 까지 약 74 statement 필요. 직전 레인(VOIP-1647, VOIP-1649, #1506, #1553)이 이미 머지되어 cachehandler 는 99.5% 이고, 열린 중복 PR 은 없음. 전제 변화 없음.

| 패키지 | statement | 현재 | 비고 |
|---|---|---|---|
| cmd/registrar-control | 922 | 25.5% | main.go 580 중 564 미커버(cobra + 실제 연결), domain_migrate.go 342 중 123 미커버 |
| cmd/registrar-manager | 156 | 5.1% | main 데몬 기동, 제외 |
| pkg/dbhandler | 926 | 84.0% | sqlmock 기반. 오류 경로 148 statement 미커버 |
| pkg/listenhandler | 627 | 88.5% | 잘못된 입력과 핸들러 오류 경로 |
| pkg/customerdomainhandler | 162 | 88.3% | 오류 경로 |
| internal/config | 84 | 76.2% | Bootstrap 과 loadGlobalConfig(once.Do)는 전역 상태에 묶임, bindConfig 는 새 cobra.Command 로 도달 가능 |
| pkg/subscribehandler | 75 | 88.0% | main.go 10 미커버. 117행 블록은 고루틴 타이밍으로 측정마다 커버 여부가 달라짐 |
| pkg/contacthandler, extensionhandler, trunkhandler | 33, 256, 157 | 75.8%, 96.9%, 94.9% | 생성자와 customer_deleted 이벤트 오류 분기 각 4~8 |
| pkg/cachehandler, redacthandler, models/* | 184, 12, 61 | 99.5%, 100%, 90% 안팎 | 증분 미미 |

파일별 미커버 상위(statement): cmd/registrar-control/main.go 564, cmd/registrar-manager/main.go 148, domain_migrate.go 123, dbhandler/extension.go 44, trunks.go 34, customerdomains.go 25, sipauths.go 14, ast_contact.go 10, listenhandler/main.go 22(Run/NewListenHandler 약 15, processRequest 오류 매핑 분기 8), v1_trunks.go 18, v1_extensions.go 17, customerdomainhandler/customerdomain.go 10, internal/config/main.go 20(Bootstrap 은 cobra 전역 상태에 묶여 제외). main.go 두 개(전체 736, 미커버 712)는 분모에 남지만 제외 항목(main.go 두 개의 기동 경로와 config 일부)을 뺀 도달 가능 미커버가 약 400 이어서 71% 에는 그중 약 20% 만 채우면 된다.

도달 불가(제품 코드 변경 없이는 단위 테스트로 못 닿음, 목표에서 제외): errorResponse 의 `cerrors.ToResponse` 실패(main.go 121, 2), v1_count.go 의 Marshal 실패(구조체에 float 등이 없음, 4), dbhandler 의 ToSql/ApplyFields/PrepareFields 오류 분기(입력으로 유도 불가, 약 20: extension 8, trunks 6, customerdomains 5, sipauths 4 로 증분 70 계산에서 이미 제외), listenhandler v1_*.go 의 응답 `json.Marshal` 실패 분기(json 태그가 붙은 구체 도메인 타입이라 오류 주입 불가, 약 32. 예외로 v1_contacts.go 의 응답은 float64 필드가 있어 NaN 으로 도달 가능하므로 contacts 테스트 1건 추가), customerdomainhandler 의 라벨 생성 오류 분기(label.go 45~49, 60~61 과 customerdomain.go 96~98, 231 의 generateLabel 오류 전파. `generateLabel` 이 패키지 함수라 주입 불가. 이벤트 오류 전파 event.go 42~44, createShortDomain 의 조회 오류 113~115 등 나머지는 도달 가능), main.go 두 개의 cobra 기동과 데몬 경로(단 registrar-control/main.go 의 플래그 생성자와 헬퍼는 아래 예비 풀로 도달 가능).

구조 분석: dbhandler 는 `handler{db, cache, utilHandler}` 를 테스트에서 직접 구성할 수 있고 sqlmock 과 MockCacheHandler 가 이미 쓰이므로 제품 코드 변경 없이 쿼리 실패, 스캔 실패, 캐시 실패 경로를 검증할 수 있다. domain_migrate 의 run* 함수는 입력 검증이 DB 연결 생성보다 앞서므로 연결 없이 검증 오류를 확인할 수 있다. 연결 생성 이후와 main.go 는 제외한다.

## 2. 테스트 계획 (코드는 구현 단계에서 작성)

1. dbhandler (sqlmock + MockCacheHandler, 신규 `*_errors_test.go`): 쿼리 실패, 스캔 실패(열 불일치), 캐시 읽기/쓰기 실패 시 오류 전파 또는 DB 대체 조회, 그리고 캐시 쓰기 실패가 호출 결과를 바꾸지 않는 경로를 함수별로 단언. 대상: ast_aor, ast_auth, ast_endpoint, ast_contact, extension, trunks, customerdomains, sipauths 의 Get/List/Update/Delete 와 *ToCache/*FromCache. 오류 메시지 문자열이 아니라 `errors.Is` 와 ErrNotFound 구분, 호출 횟수(mock 기대)로 단언.
2. listenhandler: 핸들러 오류의 상태코드 경로는 기존 listen_error_paths_test.go 와 error_response_test.go 가 이미 덮으므로 새로 쓰지 않는다. 실제 잔여 도달 가능 분기만 대상: `url.Parse` 실패 분기(잘못된 `%zz` 이스케이프 URI, 약 10곳)와 그 오류가 processRequest 의 기본 오류 응답으로 매핑되는 경로, URI 항목 부족 분기(4블록 5 statement, realm 빈 값 분기 포함), NewListenHandler 가 주입한 의존성을 유지하는지. 증분은 약 15~25.
3. customerdomainhandler(Get/Update/Delete 오류 전파와 생성자, 약 10~12) / contacthandler / extensionhandler / trunkhandler / subscribehandler: 핸들러 오류 전파, 이벤트 처리 오류 경로, 생성자가 반환하는 구현이 인터페이스 기대 필드를 갖는지(생성자 0% 인 NewXHandler 는 반환값이 nil 이 아니고 주입한 의존성을 유지하는지로 확인).
4. cmd/registrar-control/domain_migrate.go: run* 의 검증 오류(dry-run 과 execute 동시 지정, 잘못된 customer-id, execute 인데 --log 없음, 열 수 없는 로그 경로, rollback 의 --log 누락과 파일 없음과 파싱 실패), cmd* 의 플래그 이름과 기본값, writeMigrationLogLine 의 쓰기/동기화 실패 전파, buildExtensionMigrationRecord 의 빈 Realm 경계, domainMigrate/domainMigrateRollback/reprovisionExtension 의 미커버 오류 분기.
5. models/common 의 isValidDomain 및 getBaseDomainName* 경계값(이미 92.1% 이므로 증분은 미미, 우선순위 낮음).

예상 증분(도달 불가 분기 제외): dbhandler 약 70(미커버 148 중 file 별 상위 extension 44, trunks 34, customerdomains 25, sipauths 14 와 ast_* 약 30 의 오류 분기), listenhandler 약 20, domain_migrate 약 40(연결 이후 분기 제외로 낙관적일 수 있어 중간 측정으로 확인하고 부족하면 dbhandler 를 더 채움), 기타 약 20, 합계 약 150 statement 로 수기 합계 약 73%. 70% 통과선은 +37, 권장 71% 는 +74 이며 목표는 71% 이상.

## 3. verify 항목

- 예비 풀(증분 부족 시): cmd/registrar-control/main.go 의 cmdExtension*/cmdTrunk* 플래그 정의 생성자와 resolveUUID, resolveString, printJSON 은 연결 없이 닿으며 약 100 statement 로 추정. listenhandler 도달 가능 잔여는 약 35 로 추정되어 본문 20 은 보수적.

- 구현 주의: domain_migrate run* 테스트는 viper 전역 상태를 쓰므로 테스트마다 키를 설정하고 `t.Cleanup` 으로 되돌려 격리한다. 중간 측정(패키지 묶음마다)으로 71% 도달을 확인한다.

- 수기 합계 71% 이상(계산: 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile`, 위 제외 규칙, 같은 블록 중복은 최대 count).
- 기존 테스트 전부 통과, `go mod tidy` 후 go.mod/go.sum 변경 없음(신규 의존성 없음), `go vet`, `golangci-lint run`, `scripts/check-test-conventions.sh`(Test_ 이름, reflect.DeepEqual 과 t.Errorf, gomock 컨트롤러 이름 mc).
- 변이 검사: 임시 복사본에서 오류 전파 제거, 분기 조건 반전, 상태코드 변경 표본을 적용해 새 테스트가 실패하는지 확인.
- 변경 파일은 bin-registrar-manager/ 안의 `_test.go` 와 docs/plans 뿐(`git diff --stat origin/main...HEAD`). 비테스트 .go 변경 없음. 합성 값만 사용, 네트워크와 외부 비용 호출 없음.
