# bin-pipecat-manager 단위 테스트 커버리지 80% 설계 (VOIP-1657)

## 목표와 범위
- 수기 코드(gens/, mock_*.go, DO NOT EDIT 파일 제외) statement 커버리지를 77.03%(2412 중 1858)에서 80.5% 이상(커버 1942 이상, 증분 84 이상)으로 올린다.
- 단위 테스트와 테스트 파일 안의 헬퍼만 추가한다. 제품 코드 동작, 시그니처, 노출 범위는 바꾸지 않는다. 변경 파일은 bin-pipecat-manager/ 안으로 한정한다.
- 측정: 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, 수기 합계는 생성 파일을 뺀 값이며 같은 블록의 중복은 최대 count 로 합친다.

## 추가할 테스트 (패키지별, 큰 것부터)
1. pkg/cachehandler (미커버 50): miniredis 로 NewHandler, Connect 성공과 실패(서버 Close 후), PipecatcallSet 후 Get 왕복(필드 값 비교, TTL 24h 확인), Get 미존재 키 오류(redis.Nil), 손상된 JSON 값 오류, 취소된 context 로 Set 오류. miniredis v2.36.1 을 go.mod 에 추가(테스트에서만 import). 증분이 크므로 이 영역만으로 49 를 확보한다. 왕복 비교 픽스처는 문자열 값만 쓰고(map[string]any 숫자는 float64 로 바뀜) 시각은 UTC 로 만든 값(monotonic 없음)을 쓰고 저장 키 `pipecatcall:<id>` 와 TTL 도 miniredis 로 확인한다. 시점 의존이 없는 확정분(cachehandler 49 + cmd* 생성자 90 = 139)만으로 증분 84 의 1.6배다.
2. cmd/pipecat-control: TestMain 이 miniredis 를 띄우고 환경변수(DATABASE_DSN 은 `invalid-dsn`, REDIS_ADDRESS 는 miniredis 주소, PROMETHEUS_ENDPOINT, PROMETHEUS_LISTEN_ADDRESS, 그리고 RABBITMQ_ADDRESS, REDIS_PASSWORD, REDIS_DATABASE, LOG_LEVEL, LOCAL_LLM_MODELS 는 명시적으로 비움 또는 고정값으로 설정해 개발자 셸 값이 새지 않게 한다)를 세팅한 뒤 `config.Bootstrap(&cobra.Command{})` 와 `config.LoadGlobalConfig()` 를 직접 호출해 appConfig 를 1회 채운다(Execute 를 거치지 않는 initCache 등이 빈 설정으로 도는 순서 의존을 막는다). resolveUUID(빈 값, 잘못된 형식, 정상), printJSON(정상 출력은 os.Stdout 교체로 캡처해 JSON 확인, 복구는 t.Cleanup, t.Parallel 금지, Marshal 불가 값은 오류), cmd* 생성자의 플래그 이름과 기본값, initCommand 의 서브커맨드 구성과 migrate 등록, Execute(SetArgs 지정, SetOut/SetErr 은 io.Discard, SilenceUsage)로 PersistentPreRunE 와 run* 의 initHandler 실패 래핑 메시지 확인, initCache 의 성공과 실패(miniredis Close 후 같은 주소에서 Restart, Restart 실패는 t.Fatal, 다음 테스트가 서버 상태에 의존하지 않게 t.Cleanup 에서 Restart. Set 오류 분기는 서버를 닫지 않고 이미 취소된 context 로 결정적으로 만든다. cachehandler 의 Connect 실패는 별도 miniredis 를 띄워 닫은 주소로 만들어 전역 서버를 건드리지 않고, 전역 서버 Close/Restart 는 initCache 실패 테스트 한 곳에서만 파일 마지막에 두며, appConfig 를 리셋할 수 없어 같은 주소 재바인딩 실패의 극히 낮은 가능성을 감수한다). run* 에서 Execute 로 도달하는 오류는 initHandler 의 DB 연결 실패뿐이다(DB 연결이 resolveUUID 보다 먼저). resolveUUID 이후의 필수값 분기는 도달 불가로 제외한다.
3. cmd/pipecat-manager: initSignal(chSigs 에 시그널을 주입하면 chDone 으로 신호가 옴. -count 반복에서 goroutine 이 쌓이지 않도록 매 테스트가 끝날 때 chSigs 로 종료시키고 chDone 을 비움), initProm(DefaultServeMux 중복 등록 panic 을 피하려고 각 테스트가 http.DefaultServeMux 를 새 ServeMux 로 바꾸고 t.Cleanup 으로 원복하므로 중복 등록 panic 이 없고 -count 반복에도 안전하다. runDaemon 오류 경로도 같은 방식. 엔드포인트는 `/metrics` 처럼 `/` 로 시작하는 값, 주소는 127.0.0.1:0. initSignal 은 호출마다 goroutine 이 쌓이고 signal.Notify 를 등록하므로 호출 수만큼 chSigs 로 종료를 주입하고 t.Cleanup 에서 signal.Stop(chSigs) 로 해제한다), runHTTP(HTTPHandler 의 mock 이 없으므로 테스트 파일 안의 로컬 fake 로 Run 호출을 채널로 동기화하고, runHTTP 는 항상 nil 을 반환하므로 fake 의 Run 오류 시 goroutine 안 로그 줄 커버는 목표 산정에서 제외), runListen(기존 mock SockHandler 와 mock PipecatcallHandler. 오류 경로는 QueueCreate 오류로 goroutine 없이 검증하고, 성공 경로는 QueueCreate 를 Times(2) 로 기대하고 ConsumeRPC 를 DoAndReturn 으로 받아 WaitGroup 으로 2회 호출을 기다린 뒤 컨트롤러를 종료해 늦은 호출이 없게 한다), createDBHandler 와 run 의 깨진 DSN 오류 경로. 구현 시 프로파일로 분기 수를 재확인한다.
4. pkg/dbhandler: 기존 테스트는 sqlite 이며 closed-DB 테스트가 이미 있으므로 프로파일로 미커버 줄을 다시 확인해 중복을 만들지 않는다. sqlite 로 만들 수 없는 오류 5개(pipecatcalls.go 25 ScanRow, 68, 120 row.Err, 127, 210 RowsAffected)를 go-sqlmock v1.5.2(다른 서비스 16곳에서 사용, 테스트에서만 import)로 만든다. go.mod 변경 허용은 miniredis 와 go-sqlmock 두 모듈이다. PrepareFields, ToSql 오류는 도달 불가로 제외.
5. pkg/pipecatcallhandler: start.go 의 call/aicall 분기(실패 경로, 빈 입력), session.go SessionStop 분기, audiosocket, pipecatframe 오류 분기, pythonrunner 중 결정적으로 도달하는 것. goroutine 타이밍에 의존하는 줄은 제외한다.
6. models/pipecatcall SetConnAst, internal/config, httphandler, listenhandler 의 남은 오류 분기(소량).

## 격리 원칙
- 외부 네트워크, 시간, 순서에 의존하지 않는다. initProm 의 goroutine 안 ListenAndServe 줄 같은 시점 의존 줄은 목표 산정에서 제외하고, 그 줄 없이도 증분이 84 를 10% 이상 넘도록 여유를 둔다. sleep 대신 채널과 cleanup 을 쓴다. miniredis 와 httptest 는 loopback 임시 포트만 쓴다.
- 전역 상태: internal/config 의 appConfig 는 cmd 테스트에서 리셋할 수 없으므로 TestMain 에서 1회만 세팅하고 LoadGlobalConfig 를 직접 호출해 채운다. viper 는 서브커맨드 테스트 사이에 키가 남으므로 키를 지우는 API 가 없고 viper.Set 의 override 는 남으므로 resolveUUID 테스트는 매번 사용하는 모든 키를 명시 세팅한다. cobra Execute 는 항상 SetArgs 를 지정한다. initProm 을 부르는 테스트는 DefaultServeMux 를 교체/원복한다.
- DSN 은 슬래시가 없는 깨진 값(예: `invalid-dsn`)만 쓴다. 빈 문자열은 드라이버가 기본 설정(로컬 3306 dial)으로 해석하므로 금지하고, 정상 형식 DSN 도 dial 을 시도하므로 금지한다. cmd/pipecat-manager 테스트에도 TestMain 으로 같은 방식(DSN, PROMETHEUS_ENDPOINT, PROMETHEUS_LISTEN_ADDRESS, REDIS 환경변수 + Bootstrap + LoadGlobalConfig 1회)을 적용하고, DB 경로 테스트는 DSN 이 비어 있지 않음을 먼저 확인한다. 비밀번호 형태 리터럴을 쓰지 않고 합성값만 쓴다.
- 테이블 기반 테스트, 테스트 이름 `Test_` 규약, gomock 컨트롤러 변수는 `mc`(scripts/check-test-conventions.sh 규약).

## 검증 (verify)
- 수기 합계 80.5% 이상을 프로파일에서 생성 파일(gens/, mock_*.go, DO NOT EDIT 헤더 파일)을 빼고 같은 블록은 최대 count 로 합산해 실측하고 PR 본문에 패키지별 전후 표를 쓴다.
- `go mod tidy`, `go mod vendor`, `go generate ./...`, `go test -race -count=3 ./...`, `golangci-lint run -v --timeout 5m` 통과, check-test-conventions 게이트 통과.
- 변이 검사 표본: 각 영역(cachehandler 의 TTL 값, cmd resolveUUID 와 printJSON 의 Marshal 오류 분기, cmd* 플래그 이름과 기본값 변경, initCommand 서브커맨드 하나 제거, runListen 큐 이름 변경, initSignal 의 종료 신호 전달 제거, runHTTP 의 Run 호출 제거, dbhandler sqlmock 분기 중 ScanRow 와 RowsAffected 두 곳 이상, start.go 분기)에서 제품 코드를 scratch 복사본에서 1곳씩 훼손해 새 테스트가 실패하는지 확인하고 훼손 diff 와 결과를 기록하는 것을 체크리스트로 한다.
- 기준선(origin/main)에서 `go test -race -count=3 ./...` 가 이미 통과하는지 먼저 확인해, 실패 시 이 PR 의 원인이 아님을 구분한다.
- `go test -count=1` 을 3번 돌려 수기 합계가 같은지 확인한다.
- 제품 코드 변경 없음: `git diff --stat origin/main...HEAD` 가 bin-pipecat-manager/ 의 *_test.go, go.mod, go.sum, docs/plans 만 포함하고 vendor/ 는 포함하지 않는다.
- 공개 저장소 위생: 합성값만 쓰고 비밀번호 리터럴, 실제 호스트명, DSN 형태 문자열이 없는지 diff 를 grep 한다.
- 테스트 작성 중 결함을 발견해도 이 PR 에서 고치지 않는다(요청 범위 밖).

## 영향도
- 비테스트 코드와 스키마 변경 없음. 운영 영향 없음. 롤백은 PR revert.
