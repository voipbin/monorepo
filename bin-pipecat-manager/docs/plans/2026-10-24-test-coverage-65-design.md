# bin-pipecat-manager 단위 테스트 커버리지 65% 상향 (VOIP-1649)

## 1. 현황 분석

범위는 테스트 추가만이며 제품 코드는 바꾸지 않는다.

기준선 (origin/main bffdd69e3 로 rebase 후 재측정, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`, 테스트 실패 0건). 이 서비스는 직전 VOIP-1647 (`2026-10-24-test-coverage-60-design.md`)이 httphandler, listenhandler, dbhandler 의 오류 경로를 이미 올렸고, 이 문서는 그 이후 남은 부분만 다룬다.

- 수기 코드 64.1% (1547/2412 statement), 원본 58.7% (1646/2804). 수기 = `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일(`frames.pb.go` 포함)을 제외하고 같은 블록은 최대 count 로 합친 값이다.
- 다른 방식으로 측정한 60.3% / 2595 는 `frames.pb.go`(183 statement, 커버 19)를 수기에 포함한 값이다((1547+19)/2595 = 60.3%). 즉 두 방식은 같은 프로파일의 제외 방법 차이다. 2412 기준은 65% 까지 +21, 66% 까지 +45 이고, 2595 기준은 65% 까지 +121, 66% 까지 +147 이다. 수기(2412)와 포함 방식(2595) 두 방식 모두 66% 이상이 되도록 +160 statement 를 목표로 한다(수기 70.8%, 포함 방식 66.5%). 제외 없는 원본(2804) 기준은 66% 에 +205 가 필요해 목표로 삼지 않는다.

| 패키지 | 수기 전 | 미커버 성격 |
|---|---|---|
| cmd/pipecat-control | 0.0% (0/268) | main.go 전체. 연결 없이 가능한 부분(명령 생성, UUID 파싱 오류)이 있으나 이번에는 보류하고 목표 계산에 쓰지 않음 |
| cmd/pipecat-manager | 0.8% (1/128) | main.go 대부분(`logStartup` 1 statement 만 커버). 데몬 기동과 DB 연결이 필요해 이번에는 보류하고 목표 계산에 쓰지 않음 |
| pkg/httphandler | 40.8% (40/98) | 핸들러 3개는 VOIP-1647 에서 커버됨. 남은 58 statement 는 전부 고정 포트를 리슨하는 `Run`(main.go 41~72)이라 제외 |
| pkg/cachehandler | 0.0% (0/50) | Redis 클라이언트. go.mod 에 miniredis 가 없어 의존성 추가가 필요하므로 이번 범위에서 제외 |
| pkg/listenhandler | 96.0% (190/198) | 거의 완료. 잔여 8 |
| pkg/dbhandler | 90.8% (118/130) | `ToSql`, `PrepareFields` 실패 분기는 sqlite 인메모리 방식으로 도달하기 어려워 제외(7 statement). 나머지 5 statement(스캔 실패 2, 캐시 갱신 오류 1, 행 오류 1, 영향 행 수 오류 1)는 이번 목표 계산에 쓰지 않음 |
| pkg/pipecatcallhandler | 75.0% (999/1332) | start.go 115, runner.go 136, pipecatframe.go 34, websocket.go 15 미커버. 주 대상 |
| 그 밖 | config 95.5%, logtest 98.0%, message 100%, pipecatcall 81.0%, toolhandler 100% | - |

구조 판단: 이번 +160 은 pipecatcallhandler 의 네 파일(미커버 합계 300)에서 채운다. gomock, gin 테스트 컨텍스트, 포트를 쓰지 않는 `httptest` 루프백으로 도달 가능한 오류 경로를 우선한다. 제외하거나 보류하는 것은 cmd 두 패키지(목표 계산에 쓰지 않음), httphandler.Run, cachehandler 이다. pipecatcallhandler 에서 +160 을 채울 수 없으면 범위를 재조정한다. 도달 가능 여부는 2절에서 함수별로 확인한다.

## 2. 설계 (테스트 계획)

새 `_test.go` 파일만 추가하고, 같은 패키지에 이미 있는 테스트 이름(`Test_terminate_publishesTerminatedEventOnce`, `Test_terminate_callsFlushBeforeStop`, `Test_websocketAsteriskConnect`, `Test_startReferenceTypeCall_websocketDialFailure` 등)과 겹치지 않도록 새 이름에는 `_errorPaths` 같은 접미를 붙인다. 제품 코드의 시그니처와 동작은 바꾸지 않는다. 기존 mock(gomock), gin 테스트 컨텍스트, 기존 테스트가 이미 쓰는 `httptest.NewServer`(루프백 임시 포트, 고정 포트와 외부 네트워크 없음)만 쓴다. 테스트 값은 `a1b2c3d4-...` 형태의 합성 UUID 만 쓰고 호스트 이름 조회가 필요한 값은 쓰지 않으며(`DialContext` 실패는 스킴이 잘못된 URL 로만 만든다) 로그를 PR 에 붙이지 않는다. 핸들러 인스턴스는 테스트마다 새로 만들고 `pipecatcall.Session` 을 직접 만들 때는 `Ctx` 와 `Cancel` 을 반드시 채우며(nil `Ctx` 는 패닉) 전역 `upgrader` 와 Prometheus 전역 카운터의 절대값은 단언하지 않는다.

### 2.1 대상, 파일, verify 항목

| 대상 | 파일 | verify (실제 결함을 잡는 단언) |
|---|---|---|
| `RunnerWebsocketHandle` (input, output, invalid direction) | runner_handle_test.go | invalid direction 에러 문자열. input 과 output 은 세션 없음으로 `SessionGet` 오류까지 도달하며 에러를 확인한다. |
| `RunnerToolHandle` | runner_handle_test.go | 세션 없음, `Get` 실패, AICall 이 아닌 참조 유형, `BindJSON` 실패(에러와 `w.Code` 400)가 각각 에러이며 하위 RPC(`AIV1AIcallToolExecute`) 미호출(기대값 미설정으로 증명). RPC 실패 전파. 정상 시 인자(reference id, tool id, type, `pc.ID`)와 응답 본문, HTTP 200. |
| `RunnerMemberSwitchedHandle` | runner_handle_test.go | 같은 계열의 오류 경로에서 `PublishEvent` 미호출. 정상 시 200 과 본문 `{"status":"ok"}`, `Content-Type`, 발행 이벤트가 `*message.MemberSwitchedEvent` 포인터이며 필드(고객, pipecatcall, 참조, 전환 함수, 전후 멤버)가 일치함(`DoAndReturn` 안에서 타입 단언). |
| `Terminate` | start_terminate_errorpaths_test.go | `Get` 실패 전파(원인 에러 포함). 정상 시에는 세션을 미리 등록해 두고 호출 뒤 `SessionGet` 이 실패(세션 제거. `SessionStop` 이 `pythonRunner.Stop` 을 호출하므로 mock 기대값 필요)하며 `PublishEvent` 가 1회임을 단언해 `terminate` 가 실제로 실행됐음을 증명한다(세션이 없으면 `terminate` 는 즉시 반환하므로 반환값만으로는 호출 여부를 알 수 없음). |
| `terminate` | start_terminate_errorpaths_test.go | `terminate` 는 반환값이 없다. 전제: `terminatedPublished` 맵 초기화와 `notifyHandler` mock 기대값. 참조 유형이 Call 인 경우와 AICall 인 경우 각각에서 정리 실패(`start.go` 의 두 return 분기)와 정리 성공(진입) 두 경우를 모두 다룬다. 정리가 실패하면 세션이 남고(`SessionGet(pc.ID)` 성공, `Cancel` 미호출) `pythonRunner.Stop` 이 호출되지 않음. 성공 시 세션 제거(`SessionStop` 이 `pythonRunner.Stop` 을 호출하므로 mock 기대값 필요). 이벤트 발행은 정확히 1회. |
| `terminateReferenceTypeCall` | start_terminate_errorpaths_test.go | 조회 실패 전파. 상태가 progressing 이 아니면 `CallV1ExternalMediaStop` 미호출. progressing 이면 `pc.ID` 로 호출(인자 고정). 중지 실패 전파. |
| `terminateReferenceTypeAICall` | start_terminate_errorpaths_test.go | 조회 실패 전파. AICall 참조가 call 이면 `pc.ID` 로 중지, 실패 전파, 그 밖이면 무동작(RPC 미호출). |
| `Start` 의 `Create` 실패, `startReferenceTypeAIcall` 의 조회 실패와 `SessionCreate` 중복 ID | start_error_test.go | `Start` 에서 DB 생성 실패 시 에러 래핑과 이후 단계 미진행. `Create` 가 성공한 뒤 AICall 조회가 실패하면 `Start` 의 래핑 문구까지 전파(`Start` 경유). 이미 같은 ID 세션이 있으면 `SessionCreate` 오류 전파(call 분기와 default 분기 각각, 세션 맵의 키는 두 분기 모두 세션 ID 인 `pc.ID` 이며(default 분기가 넘기는 `uuid.Nil` 은 스트리밍 ID 일 뿐 키가 아니다) 사전 등록 세션도 이 키로 둔다). 거부 모델과 잘못된 참조 유형은 기존 `llmresolve_test.go` 가 이미 다루므로 추가하지 않는다. |
| `pushFrame`, `RunSender` | pipecatframe_sender_test.go | 채널이 가득 차고 수신자가 없으면 타임아웃 뒤 `DroppedFrames` 가 1 증가하고 프레임이 적재되지 않음. 채널이 가득 찬 상태에서만 ctx 취소 케이스를 써서 `DroppedFrames==0` 과 미적재로 단언(경과 시간 단언 금지, 취소와 여유가 동시에 참인 비결정 조합 금지). `RunSender` 는 고루틴 없이 동기로 호출하고, ctx 취소 케이스는 빈 채널과 열린 채널로, 채널 닫힘 케이스는 취소되지 않은 ctx 로 써서 두 case 가 동시에 준비되는 무작위 선택을 피하며 종료 조건 세 가지(쓰기 실패, 채널 닫힘, ctx 취소)가 반환으로 끝남. nil 프레임은 건너뛰고 뒤 프레임은 전송됨. |
| websocket.go 래퍼 4종, `NewWebsocketHandler`, `CheckOrigin`, `websocketAsteriskConnect` 의 `SetReadDeadline` 오류, `ReadMessage` 오류, 데드라인 해제 오류 | websocket_wrapper_test.go | `httptest.NewServer` 의 업그레이드 왕복(쓰기 후 읽기 일치), 업그레이드 실패(웹소켓 요청이 아님) 에러 전달, `DialContext` 실패. `SetReadDeadline` 오류는 닫힌 연결로, 데드라인 해제 오류는 mock `ReadMessage` 의 `DoAndReturn` 안에서 소켓을 닫아 도달한다. |

비동기 구간: `Start` 의 AICall 성공 경로와 `startReferenceTypeAIcall` 의 call 참조 구간은 `RunnerStart` 고루틴을 띄우므로 기본 범위에서 제외한다. 필요해 추가할 때는 기존 `Test_startReferenceTypeAIcall_runnerHandoff` 처럼 `started` 채널로 동기화하고 `se.Cancel()` 과 `t.Cleanup` 으로 고루틴을 정리한다(정리하지 않으면 종료 뒤 mock 호출로 비결정 실패).

### 2.2 함수별 미커버와 확보량

함수별 미커버(기준 프로파일 합산, 네 파일 합계 300)와 확보량이다. 대부분 행은 해당 함수의 미커버 전 분기를 확보한다고 가정했고, 보수적으로 줄여 잡은 행은 `startReferenceTypeAIcall`, `Start`, `pushFrame`/`RunSender` 이다.

| 함수 | 미커버 | 확보 |
|---|---|---|
| RunnerMemberSwitchedHandle | 41 | 41 |
| RunnerToolHandle | 33 | 33 |
| RunnerWebsocketHandle | 5 | 5 |
| Terminate, terminate | 11 | 11 |
| terminateReferenceTypeCall | 20 | 20 |
| terminateReferenceTypeAICall | 17 | 17 |
| startReferenceTypeAIcall (조회 실패, 중복 세션) | 45 | 5 |
| Start (`Create` 실패 래핑 등 동기 분기) | 4 | 2 |
| pushFrame, RunSender (대기 중 취소와 대기 중 송신 성공 2 statement 는 결정적으로 만들 수 없어 제외) | 30 | 28 |
| websocket.go (래퍼 9, `websocketAsteriskConnect` 6) | 15 | 15 |
| 합계 | | 약 177 (목표 +160 에 약 17 여유) |

보류 후보와 추가 기준: RunnerWebsocketHandleOutput 24, receiveMessageFrameTypeMessage 16, startReferenceTypeCall 18. 코딩 중 중간 측정에서 확보량이 +120 미만이면 이 순서로 추가한다. 추가해도 +160 에 못 미치면 범위를 재조정한다.

### 2.3 검증과 변이 검사

- 완료 조건: 수기(2412)와 포함 방식(2595) 두 방식 모두 66% 이상. `go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m` 5단계 통과와 `go.mod`, `go.sum` 변경 없음. `go test -race -count=3 ./pkg/pipecatcallhandler/` 통과(변경 전 기준선도 통과). `scripts/check-test-conventions.sh`(`git diff MERGE_BASE HEAD` 기반이므로 커밋 뒤에 실행)와 `scripts/check-structure-conventions.sh`(테스트 파일도 대상, `listenhandler/models` 와 squirrel import 금지) 통과(`Test_` 명명, testify 금지로 `reflect.DeepEqual` 와 `t.Errorf` 사용, gomock 컨트롤러 변수명 `mc`). `git diff --stat origin/main...HEAD` 가 bin-pipecat-manager/ 안만 가리킴.
- 변이 검사: 변이 전에 원본이 통과함을 먼저 확인하고, 임시 복사본에서만 변이를 적용해 짝지은 테스트가 실패하는지 본다. (1) `terminateReferenceTypeCall` 의 progressing 판정 반전 -> 중지 미호출 테스트 실패. (2) `terminateReferenceTypeCall` 과 `terminateReferenceTypeAICall` 두 곳의 `CallV1ExternalMediaStop(ctx, pc.ID)` 인자를 각각 `pc.ReferenceID` 로 변경 -> 인자 테스트 실패(테스트에서 `pc.ID` 와 `pc.ReferenceID` 를 서로 다른 값으로 둠). (3) `RunnerToolHandle` 의 참조 유형 가드 삭제 -> 하위 RPC 가 호출되어 실패(비 AICall 테스트는 유효한 JSON 본문을 보내야 변이가 살아남지 않음). (4) `pushFrame` 의 `DroppedFrames.Add(1)` 을 `Add(0)` 으로 변경 -> 드롭 카운트 테스트 실패. (5) `terminate` 의 Call 과 AICall 정리 실패 후 `return` 을 각각 삭제 -> 세션 유지 단언 실패. (6) `RunnerMemberSwitchedHandle` 의 비 AICall 가드 삭제 -> 유효한 JSON 본문에서 `PublishEvent` 가 호출되어 실패. (7) `RunSender` 의 nil 프레임 건너뜀 삭제 -> 역참조로 실패. (8) `startReferenceTypeAIcall` 의 `SessionCreate` 오류 반환 삭제를 call 분기와 default 분기에 각각 적용 -> 세션이 nil 인 채 이어져 패닉 또는 중복 세션 테스트 실패(탐지가 단언이 아니라 패닉에 의존하므로 짧은 `-timeout` 으로 실행). (9) `websocketAsteriskConnect` 의 `SetReadDeadline` 오류 처리 삭제 -> 닫힌 연결 테스트 실패. (10) member switched 이벤트를 값 타입으로 발행 -> 포인터 수신자 인터페이스 때문에 컴파일 단계에서 거부되므로 타입 단언은 보조 확인이다. 변이는 임시 복사본에서만 적용하고 작업 트리는 변이하지 않는다.
- 규약: 가능한 곳은 table-driven 으로 쓰고 UUID 는 합성 값을 상수나 테이블 필드로 둔다.
- 금지: 빈 호출과 자기비교 테스트. 오류 경로 테스트는 해당 분기에 실제로 도달하며 특정 에러 문자열, 인자, 호출 부재로 확인한다.
