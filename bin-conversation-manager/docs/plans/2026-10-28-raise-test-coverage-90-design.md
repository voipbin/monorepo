# bin-conversation-manager 단위 테스트 커버리지 90% 상향 (VOIP-1663)

## 1. 이슈 분석 요약
- 측정 대상: origin/main 9649841c7 (bin-common-handler 테스트 주입점 머지 포함). `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 후 수기 집계(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 파일 제외, 같은 블록은 최대 count). 기준선 3277/3719 = 88.12%, 테스트 실패 0건. 90.0% 까지 71, 90.5% 까지 89 statement 가 더 필요하다.
- 미커버 442 중 cmd/conversation-control 184, cmd/conversation-manager 126 이 대부분이다. 두 패키지의 시작 경로는 메시지 브로커, 데이터베이스, 마이그레이션 엔진 연결을 요구했으나 이제 `sockhandler.SetRabbitFactory`, `databasehandler.SetConnectFunc`, `migrationhandler.SetEngineFunc` 로 대체할 수 있다. 이 구간을 도달 가능으로 보면 두 패키지의 시작 경로만으로 필요량을 충분히 넘길 수 있다(기대 증가량은 PR 본문에서 실측값으로 대체한다).
- 다른 작업과 겹치는 패키지(수기 80% 미만)는 PR 본문에 명시만 하고 흡수하지 않는다.

## 2. 목표와 범위
- 합격선은 수기 합계 90.5% 이상이며 90.0% 는 최소선이다.
- 비테스트 .go 변경 없음, go.mod 와 go.sum 변경 없음(sqlite3, purse, miniredis 는 이미 사용 중). 새 테스트 파일은 기존 선례를 따라 `*_coverage_test.go` 로 둔다. 기존 테스트 파일은 순서 의존을 없애는 최소 수정만 한다: manager 의 기존 runMain 테스트는 `config.LoadGlobalConfig()` 선호출 뒤 `DatabaseDSN` 과 메트릭 주소(`127.0.0.1:0`)를 직접 대입하고 복원하며 Once 소진 여부 검사를 삭제한다.
- 외부 연결, 시간, 네트워크, 고정 포트 바인딩, 호스트 파일에 의존하지 않고 합성값만 쓴다(기존 테스트가 loopback 의 거부되는 주소로 접속하는 실패 경로는 그대로 둔다). 새 헬퍼, 상수, 가짜 엔진 타입의 이름이 패키지의 기존 식별자(`unreachableDSN`, `errBrokerTest`, `resetState`, `captureStdout` 등)와 겹치지 않는지 확인한다. manager 에서는 기존 `runmain_coverage_test.go` 가 runMain 의 DB 실패와 signalHandler 를, 새 `run_coverage_test.go` 가 run, runMain 의 나머지 경로, initProm 과 viper 복원 헬퍼를 가진다. 자기비교식과 빈 호출 테스트는 금지한다.

## 3. 격리 방침(전역 상태)
모든 새 테스트는 병렬 실행하지 않는다. 주입점과 전역 교체의 복원은 `t.Cleanup`(후입선출)으로만 하고, gomock 컨트롤러는 규약대로 `defer mc.Finish()` 로 짝짓는다.
- 주입점: DB 는 케이스마다 새 sqlite `:memory:`(`SetMaxOpenConns(1)`, 케이스 간 상태가 공유되는 `cache=shared` 는 쓰지 않음)와 `scripts/database_scripts_test` 의 스키마(기존 dbhandler 테스트와 같은 purse 방식)를 쓰고, 마이그레이션은 가짜 엔진, 캐시는 miniredis 설정 주소를 쓴다. sock 은 `sockhandler.MockSockHandler` 에 Connect 와 글로벌 이벤트 토픽 생성을 기대 호출로 둔다.
- 변경성 명령의 notify 호출(동기화): 변경성 명령(계정 생성, 갱신, 삭제, 메시지 삭제) 1회당 sock 호출은 `EventPublish` 1회와 `RequestPublish` 1회(`(nil, nil)` 반환 허용)뿐이며, 시드와 입력은 항상 고정 문자열의 non-nil customer_id 를 쓴다(nil 이면 웹훅 호출이 없어 대기가 타임아웃한다). 두 호출이 도착할 때까지 카운터로 타임아웃 있는 대기를 하고, 이어서 이전 호출의 goroutine 이 전역 메트릭 상태를 읽으므로 그 호출 전용 registry 의 요청 처리 시간 히스토그램 샘플 수가 1 이 될 때까지 타임아웃 있는 폴링으로 기다린 뒤에만 다음 호출로 넘어간다.
- 한 케이스 안에서 `initHandlers` 를 여러 번 부를 때: DB 주입점은 케이스 안에서 항상 같은 `*sql.DB` 를 돌려주고(`initHandlers` 는 DB 를 닫지 않고 연결이 1개라 새 DB 는 시드를 잃는다), Connect 와 글로벌 토픽 생성은 호출 횟수에 맞춰 기대하거나 `AnyTimes` 로 둔다. manager 의 `run()` 에서 글로벌 토픽 생성 2회는 notify 생성자 1회와 subscribe 1회이며, listen 실패 케이스는 subscribe 단계에 도달하지 않으므로 기대 호출 집합이 다르다.
- 변경성 명령의 notify 호출(단정): 이벤트 종류(created, updated, deleted)와 routing key 에 리소스 id 가 포함됨을 단정하고, 이 두 호출 외의 sock 호출은 허용하지 않아 sms 계정이 외부 호출을 하지 않음도 단정한다. Consume 류 goroutine 은 소비 시작 호출 도달을 신호로 받은 뒤에 컨트롤러를 끝낸다.
- 출력 픽스처: 시드와 입력의 secret 과 token 은 명백한 가짜 값만 쓴다(출력에 평문으로 나온다). list 시드의 `tm_create` 는 과거 고정값으로 두고, 실제 시각이 들어가는 필드(`tm_update`, `tm_delete` 등)는 값이 아니라 null 여부로 단정한다.
- prometheus: 요청 핸들러 생성자는 가드 없이 등록하므로 중복 호출 시 panic 하고 회로 차단 핸들러도 기본 registerer 에 등록한다. `initHandlers` 나 `run()` 을 호출할 때마다(한 케이스 안의 여러 호출 포함) `prometheus.DefaultRegisterer` 와 `DefaultGatherer` 를 새 registry 로 바꾸고 복원하는 헬퍼를 쓴다. notify 핸들러는 자체 가드가 있어 해당 없다. 패키지 `init` 에서 등록된 메트릭은 `DefaultGatherer` 로 단정하지 않는다. 히스토그램 폴링은 그 호출 전용 registry 를 직접 `Gather` 하며 대상은 요청 핸들러의 요청 처리 시간 하나뿐이다(notify 메트릭은 프로세스에서 처음 만든 registry 에만 등록되므로 폴링하지 않는다).
- config: `LoadGlobalConfig` 는 `sync.Once` 이고 manager 의 `config.Init` 과 control 의 `PersistentPreRunE` 가 모두 그것을 거치므로, 모든 케이스는 시작 시 먼저 `config.LoadGlobalConfig()` 를 호출해 Once 를 소진하고 필요한 필드(DSN, Redis, RabbitMQ, 메트릭 경로 `/metrics`, 메트릭 주소 `127.0.0.1:0`)를 직접 설정하며 전체를 복원한다. viper 는 control 은 기존 `resetState` 로 복원하고, manager 의 새 테스트는 viper 값을 바꾸지 않는다(빈 cobra 명령으로 `runMain` 을 호출). cobra `Execute` 는 `SetArgs` 로 인자를 지정한다.
- 메트릭 서버: `initProm` 은 loopback 의 임시 포트 리스너를 열고 종료 수단이 없어 프로세스 종료까지 남는다. 이를 허용하고, `DefaultServeMux` 교체와의 `-race` 충돌을 피하려 요청은 보내지 않는다. 케이스마다 `http.DefaultServeMux` 를 새 mux 로 바꾸고 복원하며 등록 verify 는 mux 조회로 한다. `ListenAndServe` 줄은 스케줄이 비결정적이므로 합격선 계산에 넣지 않는다.
- 시그널: 실제 신호를 보내지 않는다. `initSignal` 은 호출마다 `signalHandler` goroutine 을 남기지만 어느 goroutine 이 `chSigs` 값을 받아도 `chDone` 에 값 하나가 전달되므로 관측 결과가 같다. 케이스 시작과 끝에 `chDone` 을 비우고 끝에 `signal.Stop(chSigs)` 로 구독을 해제한다. `runMain` 은 `run` 의 오류를 무시하고 신호를 기다리므로 listen 과 subscribe 실패는 `run()` 을 직접 호출해 확인한다.
- sqlite 도달성: list 필터, 계정과 메시지 갱신과 삭제는 기존 dbhandler 테스트와 같은 스키마와 쿼리를 따른다. 계정 시드는 sqlite 가 지원하지 않는 기본값 함수를 피하도록 해당 컬럼을 명시하거나 dbhandler 의 생성 함수로 한다. 구현 중 sqlite 에서 막히는 명령이 있으면 제외하고 4절 3번으로 넘긴다.

## 4. 계획과 verify
1. cmd/conversation-control (새 `run_handlers_coverage_test.go`, 기대 증가량 약 170): 주입점으로 `initHandlers` 가 끝까지 성공하는 환경에서 run* 10개를 실행한다. 정상 출력은 JSON 의 id, customer_id, type, name 등 주요 필드를 단정한다. 없는 id 의 조회 오류 래핑, 입력 검증(필수 값 누락, 잘못된 UUID, 잘못된 conversation-id, 갱신 필드 없음, type 와 secret 과 token 필수, 지원하지 않는 type 은 외부 호출 없이 오류)을 확인한다. list 필터(type, direction, status, conversation-id)는 시드에 일치 행과 불일치 행을 함께 두고 기대 id 집합 전체(걸러진 것과 남은 것)를 단정한다. 변경성 명령은 sms 계정으로 삭제와 갱신 뒤 DB 행의 상태를 단정하고, 잘못된 입력에서는 DB 행 수가 변하지 않고 `EventPublish` 호출 수가 0 임을 단정한다. `initCommand` 의 `PersistentPreRunE` 와 migrate 클로저는 가짜 엔진과 `SetArgs` 로 `Execute` 하여 도달한다.
2. cmd/conversation-manager (새 `run_coverage_test.go`, 기대 증가량 약 110): `run()` 의 sock 기대 호출(Connect 1, 글로벌 이벤트 토픽 생성 2, 큐 생성 2, 바인딩 4, 소비 시작 2)을 열거해 성공 경로와 listen 실패, subscribe 실패를 확인한다. `runMain` 은 DB 연결 실패(엔진과 sock 팩토리 호출 수 0), 마이그레이션 실패(가짜 엔진의 Up 오류, 캐시는 종료한 miniredis 로 두고 반환 오류가 마이그레이션 오류이며 캐시 오류가 아님을 단정, sock 팩토리 호출 수 0), 캐시 연결 실패(엔진 `Up` 호출 수 1 과 `Close` 호출, sock 팩토리 호출 수 0), 성공 후 종료 신호를 확인한다. 종료 검증은 `runMain` 을 별도 goroutine 에서 돌려 `run()` 완료(소비 시작 두 호출 도달)를 동기화하고, 상한이 있는 짧은 고정 대기(수백 밀리초) 동안 반환하지 않음을 확인한 뒤(이 대기는 변이를 놓치는 거짓 음성만 낼 수 있고 실패를 만들지 않으며, 임의로 늘리거나 빼지 않는다) `chSigs` 에 값을 넣어 반환함을 타임아웃 있는 select 로 확인한다(신호 대기를 지운 변이를 잡기 위해). `initProm` 은 3절의 방식으로 확인한다.
3. 보완(필요할 때만, 합격선 90.5% 에 도달하면 중단): 도달 가능하고 외부 호출이 없는 소규모 분기를 표 기반으로 추가한다. 대상 패키지는 필요 시점에 측정으로 정한다. 선택 근거 수치는 PR 본문에 남긴다.

## 5. 검증
- 규약(`docs/conventions/testing.md`): 함수 이름 `Test_<대상>`, gomock 컨트롤러 변수 `mc` 와 `defer mc.Finish()`, 표 기반, testify 금지, 단정은 `reflect.DeepEqual` 과 `t.Errorf("Wrong match...")` 형식, `t.Fatalf` 는 선행 조건 실패에만 쓴다. 새 테스트 함수 이름이 같은 패키지의 기존 이름과 겹치지 않는지 확인한다.
- 실행: `go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`, `-race -shuffle=on -count=15`(레이스 0건, 기존 runMain 테스트가 순서와 무관하게 통과)를 확인한다. `scripts/check-test-conventions.sh` 는 커밋된 변경만 보므로 커밋한 뒤 실행하고 출력의 검사 파일 수가 (새 테스트 파일 수 + 수정한 기존 테스트 파일 수)와 같음을 확인한다. 이 스크립트가 강제하는 것은 `Test_` 이름, testify 금지, `ctrl :=` 금지뿐이므로 나머지 규약은 별도로 확인한다.
- 변이 표본(제품 코드를 임시 복사본에서 바꿔 새 테스트가 실패하는지 확인): run* 의 오류 래핑 문구, 변경성 명령 뒤 발행 호출 제거, runMain 의 마이그레이션 호출 순서와 오류 반환, runMain 의 신호 대기 제거, run() 의 subscribe 실패 반환, list 필터 구성, 계정 필수 값 검사.
- PR 본문: 패키지별 전후 표, 수기 합계와 원본 합계, 제외 방법과 측정 명령과 플래그(최종 HEAD 에서 같은 플래그), 다른 작업과 겹치는 패키지 명시.
- push 전 확인: `git diff --stat origin/main...HEAD` 와 `git diff --name-only origin/main...HEAD | grep -v '_test.go$'` 의 출력이 `bin-conversation-manager/` 안이고 이 설계 문서(커밋 대상) 한 줄뿐이며, `go mod tidy` 와 `go mod vendor` 뒤 go.mod 와 go.sum 에 diff 가 없다. 롤백은 PR 되돌림으로 충분하다.
