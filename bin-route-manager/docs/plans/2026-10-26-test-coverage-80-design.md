# bin-route-manager unit test coverage 80% (VOIP-1657)

## 1. 이슈 분석 (티켓 검토)

판정: 유효. 기준선 `origin/main 161e9748b` 에서 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외, 같은 블록 중복은 최대 count) 합계를 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 로 재측정했다.

- 수기 statement 2502개 중 1929개 커버, 77.10%. 80.0%까지 73개, 80.5%까지 약 86개가 더 필요하다. 티켓 수치와 일치한다. init 고루틴이 실행하는 signalHandler 3 stmt 가 실행마다 커버될 수도 있어 기준선은 ±3 stmt 흔들린다(필요량 73~76).
- bin-route-manager 경로를 바꾸는 열린 PR 은 없고, 이 경로의 최근 변경(VOIP-1647, 1651, 1652)은 테스트, 테스트 의존성, 문서, 테스트용 SQL 스크립트뿐이다. 전제를 바꾸는 변경은 없다.
- 미커버 573 stmt 의 분포: cmd/route-control 315, cmd/route-manager 171(run 52, runService 100, runRequestListen 6, initProm 9, main 3, init 1), 그 밖의 패키지 87(dbhandler 40, listenhandler 32, providerhandler 6, models 3, telnyxclient 3, config 2, cachehandler 1).
- 그 밖 87 의 분류(stmt 단위): 도달 불가는 응답 JSON marshal 실패 32(16블록), 쿼리 빌드 14, Delete 와 고정 필드 Create(route, providercall)의 필드 준비 오류 5, telnyx marshal 3, 난수, 상수 기반, 자격 증명 생성 실패 분기 6, 설정 바인딩 2, 모델 webhook marshal 3, 캐시 marshal 1이고 RowsAffected 3 과 rows.Err 6 은 별도 테스트 드라이버가 필요해 제외한다. 도달 가능은 row scan 오류 9(엔티티당 3: FromRow, FromDB 래퍼, List 래퍼)와 Create/Update 의 필드 준비 오류 3이다. scan 오류는 id 가 아니라 다른 컬럼에 잘못된 값을 넣어야 도달한다(route 와 providercall 은 customer_id 에 16바이트가 아닌 값, provider 는 tech_headers 에 잘못된 JSON, List 는 tm_create 도 채움). 필드 준비 오류는 map 값에 marshal 불가 값(채널)을 넘기면 도달한다.
- 따라서 목표 달성은 cmd/ 안에서 외부 연결 없이 도달 가능한 함수를 테스트하는 데 달려 있고, dbhandler 의 도달 가능 분기는 보강한다.

도달 가능 상한 산정(카드의 분류 규칙 적용, statement 는 커버 프로파일의 함수별 집계):

| 대상 | 전체 stmt | 도달 가능 stmt |
|---|---|---|
| route-control cmd* 생성자 7개(27+12+18+15+27+12+15) | 126 | 126 |
| route-control initCommand (Bootstrap 오류 분기 제외) | 39 | 37 |
| route-control resolveUUID, printJSON, initCache | 7, 5, 4 | 16 |
| route-control initHandler (DB 연결 실패 분기까지) | 7 | 약 4 |
| route-control run* 7개 (함수당 initHandler 호출, if, return 3 stmt) | 106 | 21 |
| route-control initRouteHandler, main | 21 | 0 (외부 연결, main) |
| route-manager run(DB 연결 실패까지), initProm, runRequestListen | 67 | 약 23 (run 의 DB 실패 경로 8, initProm 9, runRequestListen 6) |
| route-manager runService, main | 103 | 0 |
| dbhandler row scan 9, 필드 준비 3 | 12 | 12 |

도달 가능 합계 약 239개이고 상한은 (1929+239)/2502 로 약 86.7%다. 80.5% 목표(약 86개 필요)는 충분히 달성 가능하므로 block 없이 진행한다. 계획은 위 도달 가능 항목 전부를 테스트하는 것이다.

## 2. 설계 (계획)

범위: `cmd/route-control/main_test.go` 신규, `cmd/route-manager/run_test.go` 신규, `pkg/dbhandler/` 에 오류 경로 테스트 추가. 제품 코드, 시그니처, go.mod 변경 없음(miniredis 는 이미 go.mod 에 있다). 외부 네트워크 호출 없음: DB 는 `tcp(127.0.0.1:1)` 로 즉시 연결 거부되는 DSN(자격 증명 없음), Redis 는 miniredis, RabbitMQ 는 gomock.

공통 격리:
- `config` 는 sync.Once 싱글톤이고 `config.Get()` 은 전역 구조체 포인터를 돌려준다. 패키지 TestMain 에서 환경변수(DATABASE_DSN 등)를 설정해 첫 `LoadGlobalConfig` 가 한 번만 실행되게 하고, 시나리오별 값(Redis 주소, 리슨 주소)은 `*config.Get()` 을 직접 바꾸고 `t.Cleanup` 으로 복원한다.
- route-manager 는 `init()` 이 이미 `Bootstrap(rootCmd)` 를 호출하므로 `viper.Reset()` 은 첫 Load 이후에만 한다(순서: env 설정, 첫 Load, 이후 Reset).
- cobra 는 매번 `initCommand()` 로 새 root 를 만들고 반드시 `SetArgs` 와 `SetOut`/`SetErr` 를 지정한다(go test 의 os.Args 를 읽지 않게 함). 각 테스트 끝에 `viper.Reset()`. stdout 캡처는 파이프 교체와 defer 원복, 병렬 실행 금지.
- 고정 포트와 긴 sleep 에 의존하지 않고 `127.0.0.1:0` 과 deadline 이 있는 polling 을 쓴다. `initProm` 은 DefaultServeMux 에 등록하므로 route-manager 의 `run` 은 테스트 전체에서 정확히 1회만 호출하며 `-count>1` 은 지원하지 않는다(CI 는 `-count=1`). 이 `run` 이 남기는 listen 고루틴은 프로세스 종료까지 유지되며 수용한다.
- dbhandler 의 기존 `newLiveTestHandler` 는 패키지 공유 sqlite(`dbTest`)를 쓴다. 불량 행은 `cleanupRows` 로 사전과 사후에 삭제하고 고정 uuid 를 쓰며 `t.Parallel` 은 쓰지 않는다(`newClosedDBHandler` 는 DB 를 닫아 두므로 행 삽입에 쓰지 않는다).
- 리슨 주소 점유: `127.0.0.1:0` 으로 먼저 listen 해 얻은 주소를 `config.Get().PrometheusListenAddress` 에 넣는다.

verify 항목:

1. route-control `resolveUUID`: 빈 값은 `is required` 포함 오류, 형식 오류는 `invalid format` 포함 오류와 입력값 표시, 정상 uuid 는 같은 값.
2. `printJSON`: 정상 값은 2칸 들여쓰기 JSON 을 stdout 에 출력(파이프 캡처 후 필드와 들여쓰기 확인), marshal 불가 값(채널)은 `failed to marshal JSON` 오류.
3. `initCommand`: route 하위 명령 7개와 migrate 가 등록되고, 명령별 RunE 함수, 플래그와 기본값(limit=100 등)이 CLI 계약과 일치. 하위 명령 Execute 가 PersistentPreRunE 를 거쳐 RunE 까지 도달한다(config 로드는 sync.Once 라 별도 단언하지 않는다).
4. run* 7개: DB 연결 불가 환경에서 Execute 가 `failed to initialize handlers` 와 `could not connect to the database` 를 포함한 오류를 반환. `migrate status` 는 DB 불가 시 오류 반환(출력에서 DSN 을 단언하지 않음).
5. `initCache`: miniredis 주소면 성공, 닫힌 miniredis 주소면 `could not connect to the cache` 오류. `initHandler` 는 DB 불가를 먼저 보고.
6. route-manager `run`(1회): DB 연결 불가면 오류를 반환하고 Prometheus 엔드포인트가 DefaultServeMux 에 등록됨. 리슨 주소를 미리 점유하고 logrus test hook 으로 `Could not start prometheus listener` 로그를 관찰한 뒤 점유를 해제하면 deadline(5초 이상) 안에 실제로 listen 이 시작됨.
7. `runRequestListen`: mock sockhandler 로 QueueCreate 성공 시 큐 이름 `QueueNameRouteRequest` 로 ConsumeRPC 가 consumer 이름 `route-manager`, 동시성 10 으로 호출되고(ConsumeRPC 의 DoAndReturn 에서 채널을 닫고 테스트는 deadline 이 있는 select 로 대기) nil 반환. QueueCreate 실패 시에도 nil 반환하며 ConsumeRPC 에 EXPECT 를 두지 않아 미호출을 검증.
8. dbhandler scan 오류: route 와 providercall 은 customer_id 에 3바이트 값, provider 는 tech_headers 에 잘못된 JSON 을 가진 행을 직접 삽입(List 용은 tm_create 도 채움, 오류 행은 customer 필터로 선택되지 않으므로 필터 없이 호출)한 뒤 Get 과 List 가 `copy to field failed` 계열 오류와 감싼 함수 이름(`providerGetFromDB`, `ProviderGets` 등)을 반환. 정상 행이 함께 있어도 오류를 삼키지 않음. 삽입 행은 cleanupRows 로 정리.
9. 필드 준비 오류: ProviderUpdate 와 RouteUpdate 는 채널을 담은 슬라이스나 맵을 필드 값으로 넘기고, ProviderCreate 는 Metadata 에 채널을 담은 값을 넣어 `JSON marshal failed` 를 반환하며 DB 행이 바뀌지 않음.
10. 변이 검사(scratch 복사본에서만): 아래 변이마다 대응 테스트가 실패하는지 확인하고 원복 후 통과를 확인한다.

| 변이 | 기대 실패 테스트 |
|---|---|
| resolveUUID 의 Nil 비교 제거 | 형식 오류 테스트 |
| printJSON 의 MarshalIndent 를 Marshal 로 | 들여쓰기 출력 테스트 |
| printJSON 의 오류 래핑 문구 변경 | printJSON 오류 테스트 |
| limit 기본값 100 변경 | initCommand 플래그 테스트 |
| 하위 명령의 RunE 를 다른 run 함수로 교체 | initCommand 테스트의 RunE 함수 비교 |
| runRequestListen 의 큐 이름 변경 | runRequestListen 성공 테스트 |
| initProm 재시도 분기의 continue 를 break 로 | Test_run 의 metrics 서버 polling |
| dbhandler 래퍼의 scan 오류 반환을 continue 로 | scan 오류 List 테스트 |
| dbhandler 래퍼의 %w 를 %v 로 | 감싼 오류 단언 테스트 |

11. 완료 조건: 수기 합계 80.5% 이상, `go test -race -count=1 ./...` 와 `golangci-lint` 통과, `git diff --stat origin/main...HEAD` 가 bin-route-manager/ 만 포함.

VOIP-1416 겹침: dbhandler 는 이미 90% 이상이라 80% 미만 패키지 상향 대상이 아니다. 겹침 여부는 PR 본문에 명시만 한다.
