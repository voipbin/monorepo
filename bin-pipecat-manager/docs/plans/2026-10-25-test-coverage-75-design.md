# 2026-10-25 bin-pipecat-manager 단위 테스트 커버리지 75% 설계 (VOIP-1652)

## 1. 티켓 검토 (이슈 분석)

판정: 유효. origin/main accc03e00 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 를 직접 실행해 카드 수치와 일치함을 확인했습니다(테스트 실패 0건, go.mod/go.sum 변경 없음).

- 수기 코드 71.60% (1727/2412 statement), 생성 코드와 mock 을 포함한 원본 65.12%.
- 제외 방법: `gens/` 경로, `mock_*.go`, 헤더에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일(frames.pb.go)을 빼고, 같은 블록의 중복은 최대 count 로 합칩니다.
- 75.0% 는 1809 statement(+82), 권장 76% 는 올림하여 1834 statement(2412 x 0.76 = 1833.12, +107)입니다. 목표는 +107 statement 이상입니다. 측정 기준 커밋 accc03e00 은 이 브랜치의 기반 HEAD 와 같습니다.
- 최근 변경 확인(`git log -- bin-pipecat-manager` 에서 6cccb1194 VOIP-1647, eed0006ba VOIP-1649 확인): 이 서비스는 VOIP-1647(60%)과 VOIP-1649(70%)로 이미 두 차례 올렸고 열린 중복 PR 은 없습니다(조회 결과: Python 의존성 bump 두 건뿐). 카드 전제를 바꾸는 변경은 없습니다.

패키지별 기준선(수기 statement 기준이며 `go test` 출력의 원본 기준 퍼센트와 다릅니다. models/pipecatframe 는 생성 코드, internal/logpolicy 와 migrations 와 listenhandler/models/request 는 statement 가 없어 표에서 뺐습니다):

| 패키지 | statement | 미커버 | 커버리지 |
|---|---|---|---|
| cmd/pipecat-control | 268 | 268 | 0.0% (CLI main) |
| cmd/pipecat-manager | 128 | 127 | 0.8% (main) |
| internal/config | 88 | 4 | 95.5% |
| internal/logtest | 49 | 1 | 98.0% |
| models/message | 3 | 0 | 100.0% |
| models/pipecatcall | 21 | 4 | 81.0% |
| pkg/cachehandler | 50 | 50 | 0.0% (redis 의존) |
| pkg/dbhandler | 130 | 12 | 90.8% |
| pkg/httphandler | 98 | 58 | 40.8% |
| pkg/listenhandler | 198 | 8 | 96.0% |
| pkg/pipecatcallhandler | 1332 | 153 | 88.5% |
| pkg/toolhandler | 47 | 0 | 100.0% |
| 합계 | 2412 | 685 | 71.60% |

미커버 구조 분석(실제 코드를 읽어 확인):

- cmd/pipecat-control(268), cmd/pipecat-manager(128) 는 main 과 외부 연결(RabbitMQ, DB, Redis) 초기화라 단위 테스트를 쓰지 않습니다(합계 396 statement, 미커버 395). 수기 합계의 분모에는 그대로 남기며 분모에서 빼지 않습니다.
- pkg/httphandler.Run: gin 라우터를 만들고 고정 주소 `localhost:8001` 에 listen 한 뒤 고루틴에서 Serve 합니다. 현재 테스트는 라우트를 테스트가 다시 등록해 핸들러만 검증하므로 Run 의 라우트 등록과 listen 실패 처리는 미커버입니다. loopback 고정 포트이므로 외부 네트워크 의존은 없습니다. 이 한 함수가 58 statement 입니다(coverprofile 의 블록 크기 기준). 분해: Listen 호출까지의 라우터 구성과 서버 생성 50, Listen 실패 분기 2, 고루틴과 반환 6 입니다. 포트가 상수라 테스트가 바꿀 수 없고 서버 핸들이 저장되지 않아 종료할 수 없으므로, 성공 경로는 호출하지 않고(listener 가 프로세스 끝까지 남음) 실패 경로만 시험합니다. 테스트가 같은 주소를 먼저 점유한 뒤 Run 을 호출하면 Listen 실패가 결정적으로 발생해 50+2=52 statement 가 커버됩니다. 점유는 Go 의 `localhost` listen 과 같은 IPv4 `127.0.0.1:8001` 로 시도하며, 점유가 이미 사용 중(EADDRINUSE)으로 실패하면 Run 의 Listen 도 똑같이 실패하므로 skip 하지 않고 오류 반환을 단언합니다. 이 경로는 라우터 구성까지만 실행하므로 라우트가 실제로 서빙되는지는 검증하지 못하며, 그 한계를 테스트 주석에 적습니다.
- pkg/pipecatcallhandler/start.go (미커버 59): Start 의 성공 반환(83행), startReferenceTypeCall 의 세션 생성 실패(106행)와 external media 중단 실패 로그(136행)와 연결 성공 처리(141-145행), 성공 경로 말미(167-182행, 미디어 핸들러와 수명주기 모니터 고루틴), startReferenceTypeAIcall 의 `ReferenceTypeCall` 분기 245-313행(40 statement: 실패 경로 245-275행과 284-287행 약 20, 연결 성공 276-280행 8, 대기 이후 291-295행 4, 미디어 핸들러와 모니터 고루틴 298-313행 8). `default:` 분기(315-327행)와 LLM 키 해석 실패 분기는 이미 커버되어 있습니다. 모두 requestHandler 와 websocketHandler 가 인터페이스로 mock 가능합니다.
- pkg/pipecatcallhandler/runner.go (미커버 55): RunnerWebsocketHandleOutput(65.7%)의 Upgrade 실패, 조회 실패, protobuf 해제 실패, 오디오 오류 누적 한계, 알 수 없는 프레임과 메시지 종류. runnerWebsocketHandleInputReceiver(75%)의 텍스트와 종료와 알 수 없는 종류. receiveMessageFrameTypeMessage(81.4%)의 일부 오류 분기.
- pkg/cachehandler 는 redis 클라이언트를 직접 생성해 miniredis 가 필요합니다. 이 서비스 go.mod 에 miniredis 가 없고 go.mod/go.sum 변경이 따르므로, 아래 예산이 모자랄 때만 쓰는 예비입니다(미사용이 기본).
- 테스트하기 어려운 구조: websocket 과 고루틴이 얽힌 start 성공 경로입니다. websocketHandler 와 pythonRunner 가 mock 가능한 인터페이스라 mock 의 완료 신호 채널로 고루틴 종료를 기다리는 방식이면 제품 코드 변경 없이 결정적으로 시험할 수 있다고 판단합니다. 시그니처나 노출 범위를 바꿀 필요는 없습니다.

VOIP-1416 과의 겹침: 이 서비스에서 80% 미만인 패키지(httphandler, cachehandler, cmd)와 겹칩니다. PR 본문에 명시만 하고 흡수하지 않습니다.

함수별 근거(`go tool cover -func` 기준): startReferenceTypeCall 68.4%, startReferenceTypeAIcall 51.2%, RunnerWebsocketHandleOutput 65.7%, runnerWebsocketHandleInputReceiver 75.0%, receiveMessageFrameTypeMessage 81.4%, httphandler.Run 0.0%.

예산(미커버 statement 기준, 어려운 고루틴 경로를 뺀 쉬운 항목만 계산):

| 항목 | 예상 증가 |
|---|---|
| httphandler.Run 실패 경로 | 52 |
| runner.go: Output 의 Upgrade/조회 실패, protobuf 해제 실패, 오디오 오류 한계, Transcription/Message/알 수 없는 프레임과 메시지 종류, InputReceiver 의 종류별 분기(약 28), receiveMessageFrameTypeMessage 의 타입별 unmarshal 오류(12) | 약 40 |
| start.go 실패 경로: 106행 세션 생성 실패, 136행 orphan external media 중단 실패 로그, AIcall Call 분기 external media 시작 실패와 websocket 연결 실패 | 약 22 |
| 도달 가능한 오류 분기만: pythonrunner.go Start/Stop 의 HTTP 전송 오류와 본문 읽기 오류와 non-200 (약 5), SessionStop 의 python Stop 오류 로그(session.go 110행, 1) | 약 6 |
| 소계 | 약 120 |

소계 약 120 으로 필요량 107 을 넘고 여유는 약 13 입니다(재계산에서 session/audiosocket/pipecatframe 의 빈 구조체 생성자와 경합 분기, 정상 입력으로 도달할 수 없는 marshal 오류 분기는 제외해 재계산). httphandler.Run 실패 경로는 포트 상태와 무관하게 결정적입니다: 테스트가 점유하든, 다른 프로세스가 이미 점유하고 있든 Listen 은 실패하고 Run 이 오류를 반환하므로 오류 반환을 단언합니다(정상 환경에서 52 statement 확정. 점유 실패 원인이 EADDRINUSE 가 아닌 비정상 환경에서만 2절의 skip 이 적용되며 이는 최종 측정에서 PASS 로 확인).

추가 여유(성공 꼬리, 약 32): 141-145행 8, 167-182행 8, 276-280행 8, 298-313행 8 입니다(291-295행의 4 는 연결 실패 분기라 항목 4 의 실패 시험이 커버). 이 경로는 사전 조건이 많아 예산에 넣지 않았고, 필요할 때만 씁니다. 요건: (1) websocketAsteriskConnect 가 DialContext 결과에 `SetReadDeadline` 을 호출하고 SessionStop 이 `ConnAst.Close()` 를 부르므로 nil 이 아닌 실제 연결이 필요합니다(기존 `dialTestServer` 로 httptest websocket 서버에 접속). (2) runAsteriskReceivedMediaHandle 은 `ReadMessage` 를 반복 호출하고 종료 시 ConnAstDone 을 닫으므로 websocketAsteriskConnect 가 MEDIA_START 를 읽기 위해 ReadMessage 를 먼저 한 번 부르므로(`TextMessage` 를 반환해야 함) 첫 호출은 TextMessage, 이후 runAsteriskReceivedMediaHandle 의 호출은 오류를 반환하도록 순서를 두고 호출 종료를 기다려야 합니다. (3) 협력 객체는 흐름에 따라 다릅니다. Call 참조: RunnerStart 의 toolHandler.GetByNames 와 pythonRunner.Start(오류 반환으로 즉시 종료), terminate 의 CallV1CallGet(진행 중이 아니면 external media 중단 없음), notifyHandler.PublishEvent, SessionStop 의 pythonRunner.Stop. AICall 참조의 Call 분기: RunnerStart 가 AIV1AIcallGet, resolveTeamForPython, resolveAIFromAIcall, AIV1AIcallToolList 를 거치고, terminate 가 terminateReferenceTypeAICall 로 가서 AIV1AIcallGet 과 CallV1ExternalMediaStop 을 부릅니다. 호출마다 완료 신호 채널을 두고 모든 고루틴의 마지막 mock 호출을 기다린 뒤 종료해야 mc.Finish 이후 호출이 없습니다. 이 요건을 안정적으로 충족하지 못하면 쓰지 않고 부족분은 cmd/pipecat-control 의 순수 함수나 cachehandler(miniredis) 예비로 채웁니다.

Run 실패 경로 시험의 주의: 점유 리스너는 `t.Cleanup` 으로 닫고, 점유 실패 원인이 EADDRINUSE 가 아니면(예: 권한) Run 의 결과를 예측할 수 없고 성공하면 리스너를 남기므로 그 경우에 한해 Run 을 호출하지 않고 skip 합니다(환경 이상일 때만이며 일반 CI 에서는 발생하지 않음). Go 코드에서 8001 을 listen 하는 곳은 이 상수 하나뿐입니다. VOIP-1647 은 고정 포트 listen 을 이유로 Run 을 제외했으나, 이번에는 서버를 남기지 않는 실패 경로만 시험하는 것으로 입장을 바꿉니다. Run 52 가 예산의 절반이므로 이 시험이 막히면 목표가 줄어드는 위험이 있고, 그 경우 위 예비를 씁니다. 모든 statement 수는 go.mod 가 지정한 툴체인의 coverprofile 기준이며(한 문장짜리 블록을 10 으로 세는 경우가 있음), 기준선 실행의 테스트 실패 0건과 go.mod/go.sum 무변경은 이 문서 작성자가 직접 실행한 결과입니다.

반증하지 못한 주장: 카드의 모든 수치와 제외 방법. 틀린 주장: 없음(재검토 과정에서 문서의 76% 필요 statement 1833 을 1834 로, start.go 미커버 서술을 실제 위치로 정정).

## 2. 계획

기준: 브랜치 기반 accc03e00. 제품 코드는 바꾸지 않고 `_test.go` 만 추가합니다. 새 테스트 이름은 `Test_` 로 시작하고 gomock 컨트롤러는 `mc`, testify 는 쓰지 않습니다. 목표는 수기 합계 76.04% 이상(1834 statement, +107)이며 실제 증가량은 구현 후 coverprofile 로 측정해 보고합니다. 설계 단계라 테스트 코드는 쓰지 않고 verify 항목만 적습니다. 아래 행 번호는 accc03e00 기준 coverprofile 의 미커버 블록에서 가져온 값이며, 구현 시 커버 여부를 다시 coverprofile 로 확인해 이미 커버된 분기는 시험하지 않습니다.

1. pkg/httphandler/main_test.go (기존 파일에 추가, 약 +31): `Test_Run_listenFailure`. 테스트가 `127.0.0.1:8001` 을 먼저 점유하고(`t.Cleanup` 으로 닫음) `Run()` 을 호출해 오류가 반환되고 메시지에 `failed to listen on localhost:8001` 이 포함됨을 단언합니다. 점유 실패 원인이 EADDRINUSE 이면 그대로 Run 오류를 단언하고, 그 외 원인(권한이나 IPv6 전용 환경 등)이면 skip 합니다. 성공 경로는 서버를 종료할 수 없어 호출하지 않으며, 이 시험은 라우트가 실제로 서빙되는지 검증하지 못한다는 한계를 주석에 적습니다.
2. pkg/pipecatcallhandler/runner_output_test.go (신규, runner.go 약 +28). 기존 `Test_logPolicy_RunnerWebsocketHandleOutput_frames`(Text/Transcription/TextMessage 커버)와 겹치지 않는 분기만 시험합니다. 전제: Output/Input 은 `defer ws.Close()` 와 `c.Request` 를 쓰므로 `Upgrade` mock 은 nil 이 아닌 실제 `*websocket.Conn`(기존 logpolicy_test.go 처럼 httptest websocket 서버에 접속)을 반환하고 gin 컨텍스트는 `gin.CreateTestContext` 로 만들고 `c.Request` 는 `httptest.NewRequest` 로 설정합니다(로그 필드가 `c.Request.RemoteAddr` 를 읽음).
   - RunnerWebsocketHandleOutput: Upgrade 실패(372-373행), `db.PipecatcallGet` 실패(382-383행), ReadMessage 오류 종류별 분기(400, 402행: `CloseNormalClosure` 는 정상 종료, 정상이 아닌 종료 코드는 비정상 종료, 일반 오류) 후 루프 종료, 파싱 불가한 protobuf 후 계속(413-414행), 오디오 프레임 처리 오류가 이어져도 루프가 계속되고 연속 오류가 501회째에 nil 로 반환(423-424, 429-439행. `NumChannels` 가 1 이 아닌 프레임이면 mock 없이 `runnerWebsocketHandleAudio` 가 오류를 반환하며, ReadMessage 는 501회 오디오 프레임을 돌려주고 반환 후 추가 ReadMessage 호출이 없음을 단언), Frame_Message 의 JSON 이 잘못되어 처리 오류 로그 후 계속(445-447행), 인식할 수 없는 프레임(450행), CloseMessage 에서 nil 반환(457-458행), Ping/Pong/알 수 없는 메시지 종류(465-469행). 각 케이스는 ReadMessage 순서와 `PublishEvent(EventTypeInitialized)` 인자를 정확한 값으로 단언하고, 반환값과 소비한 메시지 수를 확인합니다. 오디오 오류 몇 회 뒤 성공 프레임(`NumChannels` 1, 기본 샘플레이트, 빈 `Audio` 바이트열이면 `len(audioData)==0` 에서 mock 없이 nil 반환하므로 협력 객체가 필요 없음)이 오면 카운터가 0 으로 리셋(438-439행, 미커버)되어 이후 오류가 다시 1부터 누적되고 501 도달 시점이 밀림을 단언하는 케이스를 포함합니다.
   - runnerWebsocketHandleInputReceiver: 미커버인 Close(338-339행), Ping(346행), Pong(348행)만 시험합니다(Text 와 알 수 없는 종류는 이미 커버). Close 에서 반환하는 것과 읽기 오류에서 반환하는 것을 ReadMessage 호출 횟수로 구분해 단언합니다. 같은 파일에서 RunnerWebsocketHandleInput 의 Upgrade 실패(284-285행)도 시험합니다.
3. pkg/pipecatcallhandler/runner_message_test.go (신규, 약 +10~16): receiveMessageFrameTypeMessage 의 타입별 unmarshal 실패(bot-transcription, user-transcription, user-llm-text, bot-llm-text, bot-llm-stopped, error, error-response 중 미커버인 분기. 611-655행), 비최종 user-transcription 은 이벤트 없이 nil(638행), bot-llm-text 토큰 채널이 가득 차면 경고 후 계속(693행), user-llm-text 성공 시 PublishEvent 고루틴(649행, `message.EventTypeUserLLM` 과 본문 텍스트를 정확히 단언하고 고루틴 완료를 채널로 기다림), 알 수 없는 label 과 RTVI type 은 오류가 아닌 nil 반환. 기존 `Test_receiveMessageFrameTypeMessage` 와 겹치지 않는 케이스만 추가하고 오류 접두어를 정확히 단언합니다.
4. pkg/pipecatcallhandler/start_callpath_test.go (신규, start.go 약 +26: external media 시작 실패 13, websocket 연결 실패 추가 11, 106행과 136행 각 1):
   - startReferenceTypeCall 의 세션 생성 실패(106행): 같은 id 세션을 미리 넣어 `SessionCreate` 가 실패하게 합니다. LLM 타입은 RPC 가 필요 없는 `openai.gpt-5` 로 두고(기존 테스트와 동일), `CallV1CallGet` 은 호출되지만 `CallV1ExternalMediaStart` 이후는 호출되지 않음을 단언합니다.
   - orphan external media 중단 실패 로그(136행): DialContext 실패 후 `CallV1ExternalMediaStop` 도 오류를 반환해도 시작 오류가 `asterisk setup failed` 로 반환되고 세션이 삭제됨을 단언.
   - startReferenceTypeAIcall 의 `ReferenceTypeCall` 분기: external media 시작 실패(245-264행), websocket 연결 실패와 정리(267-274행, `CallV1ExternalMediaStop` 오류 로그 270-272행 포함. `DialContext` mock 이 오류를 반환하게 하며 네트워크 호출은 없음)와 대기 후 실패 처리(291-295행). 두 경우 모두 반환 오류 문구와 세션 삭제를 단언합니다.
   - 고루틴 규칙: 이 경로들은 `astErrCh` 를 기다리기 전에 RunnerStart 고루틴을 띄우고, AICall 참조에서는 그 고루틴이 `AIV1AIcallGet`, `resolveTeamForPython`, `toolHandler.GetByNames`, `pythonRunner.Start/Stop` 을 호출하며 시점이 테스트 본문과 경합합니다. (1) `startReferenceTypeAIcall` 본문의 동기 `AIV1AIcallGet` 1회는 정확히 기대하고, 고루틴이 부르는 mock 은 별도 `AnyTimes()` 로 분리해 호출 횟수를 단언 대상에서 제외합니다. `AIV1AIcallGet` 은 반환 값을 비우면 nil, nil 로 nil 역참조가 나므로 유효한 `AIcall` 또는 오류를 명시적으로 반환하게 합니다. (2) 시나리오마다 고루틴의 마지막 mock 호출을 구체적으로 정해(RunnerStart 는 `Stop` 을 부르지 않으므로 `AIV1AIcallGet` 이 오류를 반환하면 그 호출, 그렇지 않으면 `pythonRunner.Start`) 그 호출에 완료 채널을 두고, 테스트는 세션 맵이 비고 세션 컨텍스트가 취소된 뒤 그 완료를 기다려 `mc.Finish` 이후 mock 호출이 없게 합니다. (3) `go test -race -count=20` 으로 반복해 간헐 실패가 없음을 확인합니다. 기존 `Test_startReferenceTypeCall_*` 는 수정하지 않습니다.
5. 도달 가능한 오류 분기(약 +7): pythonrunner.go Start/Stop 의 HTTP 전송 오류(141, 188행), 응답 본문 읽기 오류(149, 196행), Stop 이 200 으로 성공 반환(203행)을 기존 `endpoint_test.go` 와 같이 전역 `httpClient` 를 교체(`roundTripFunc`)하고 `defer` 로 복원하며 `t.Parallel` 없이 시험합니다. 오류 접두어를 정확히 단언합니다. session.go 의 두 미커버 블록: 드롭된 프레임이 있으면 경고하고 정상 종료(110-111행, `DroppedFrames` 를 미리 올린 세션), `pythonRunner.Stop` 이 오류를 반환해도 세션이 삭제됨(160-161행). 필드 없는 생성자(NewAudiosocketHandler, NewPythonRunner)와 동시성 경합이 필요한 SessionStop 분기(141-143행), 정상 입력으로는 도달할 수 없는 marshal 오류 분기는 시험하지 않습니다.
6. 예비(1-5번으로 +107 이 안 되면): (a) start.go 성공 꼬리(1절 요건 충족 시, 약 32: 141-145, 167-182, 276-280, 298-313행), (b) cmd/pipecat-control 의 resolveUUID 와 printJSON(각각 몇 statement 에 불과해 효과 작음), (d) runner.go 의 runnerStartScript 실패 분기(121-122, 130-131행)와 오디오 리샘플/쓰기 실패(1031-1032, 1042, 1050-1051행)처럼 mock 만으로 도달 가능한 분기, (c) cachehandler + miniredis(약 +50, go.mod/go.sum 변경이 필요하고 같은 버전이 이미 다른 서비스에 있음을 확인해야 함). 선택 기준은 mock 만으로 닿는 (d) 를 가장 먼저, 그다음 제품 코드 무변경과 결정성이 확실한 (a), 그다음 효과가 큰 (c), (b)는 마지막입니다.

예산 재계산(미커버 statement 기준): 1번 52 + 2번 약 28 + 3번 약 10 + 4번 약 26 + 5번 약 7 = 약 123. 필요량 107 에 대해 여유는 약 16 입니다. 1절 표의 약 120(start 22, runner 40, python 6)은 초기 추정이고 2절의 값(start 26, runner 38, python 7)이 최신이며 3번 항목은 실제 미커버가 약 16 이라 10 은 보수적 값입니다. 예비는 위 6번입니다.

## 3. verify 항목

- 모든 단언은 기대값을 독립적으로 적고 반환 오류 문구, 상태, mock 호출 인자를 확인합니다. 자기비교와 빈 호출 금지.
- CLAUDE.md 5단계(go mod tidy, vendor, generate, test, golangci-lint) 통과, go.mod/go.sum 변경 없음(예비 (c) 를 쓰는 경우에만 예외로 보고). `go test -race -count=1 ./...` 와 `-count` 반복 실행 통과(4번 항목은 `-race -count=20`).
- 수기 합계 76% 이상을 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 로 실측해(수기 합계 = coverprofile 에서 `gens/` 경로, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일의 블록을 빼고 같은 블록은 최대 count 로 합친 뒤 count>0 블록의 statement 합 / 전체 statement 합) 패키지별 전후 표를 PR 본문에 적습니다.
- `scripts/check-test-conventions.sh` 통과(`git fetch origin main` 선행).
- 변이 표본(scratch 복사본에서만): Run 의 Listen 오류 wrap 문구 변경, Output 의 audioErrors 500 한계 제거, receiveMessageFrameTypeMessage 의 unmarshal 오류 반환 제거, Output 의 CloseMessage `return nil` 제거(457-458행), Output 의 오디오 오류 카운터 리셋(`audioErrors = 0`) 제거, pythonrunner Stop 의 non-200 이 아닌 200 성공 반환 조건 변경, Stop 오류 시 `sessionDelete` 건너뛰는 변이, startReferenceTypeAIcall 의 `sessionDelete` 제거 각각에서 새 테스트가 실패함을 확인합니다.
- 최종 측정에서 `go test -v -run Test_Run_listenFailure ./pkg/httphandler/` 로 skip 없이 PASS 임을 확인합니다(skip 되면 52 statement 가 조용히 빠지므로 목표 달성 판정에 포함하지 않습니다). 점유 리스너는 `-count` 반복마다 닫혀야 합니다.
- `git diff --stat origin/main...HEAD` 가 bin-pipecat-manager/ 안의 `_test.go` 와 이 문서만 포함.

## 4. 구현 결과

수기 합계 71.60% (1727/2412) → 77.03% (1858/2412), 원본 65.12% → 69.79%. 측정은 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 이며 `go mod tidy` 후 go.mod/go.sum 변경은 없습니다.

| 패키지 | 전 | 후 |
|---|---|---|
| cmd/pipecat-control | 0.0% | 0.0% |
| cmd/pipecat-manager | 0.8% | 0.8% |
| internal/config | 95.5% | 95.5% |
| internal/logtest | 98.0% | 98.0% |
| models/message | 100.0% | 100.0% |
| models/pipecatcall | 81.0% | 81.0% |
| pkg/cachehandler | 0.0% | 0.0% |
| pkg/dbhandler | 90.8% | 90.8% |
| pkg/httphandler | 40.8% | 93.9% |
| pkg/listenhandler | 96.0% | 96.0% |
| pkg/pipecatcallhandler | 88.5% | 94.4% |
| pkg/toolhandler | 100.0% | 100.0% |

변이 표본(별도 scratch worktree에서만 적용): Run 의 listen 오류 문구, 오디오 오류 한계 500, 오디오 오류 카운터 리셋, Output 의 CloseMessage 반환, user-transcription unmarshal 오류 반환, 비최종 user-transcription 필터, AIcall Call 분기와 Call 경로의 sessionDelete, pythonrunner Stop 의 200 판정, SessionStop 의 드롭 프레임 경고 조건, 토큰 채널 가득 참 경고 11건 모두에서 새 테스트가 실패했습니다.
