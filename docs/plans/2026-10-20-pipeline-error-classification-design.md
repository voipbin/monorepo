# VOIP-1633 pipeline_error 분류 확장과 장문 입력 timeout 알림

기준: origin/main 4d1084ad3.

## 1. 티켓 검토 (부분 유효)

- 분류 위치: `bin-pipecat-manager/pkg/pipecatcallhandler/pipelineerror.go` `classifyPipelineError`(순수 함수, 소문자 문자열 매칭), 호출은 `runner.go:770`. ai-manager는 category를 그대로 받아 고정 문구를 붙이고(`bin-ai-manager/pkg/messagehandler/pipeline_error.go:34-40`), enum 밖 값은 unknown으로 저장한다(같은 파일 71-79). 러너 Python은 분류하지 않는다.
- (d) 재현: pipecat-ai 1.12.0 + openai 2.15.0, 로컬 가짜 서버(비용 없음)에 openai SDK로 실제 요청해 예외 텍스트를 얻고, pipecat 접두사 `Error during completion: {e}` 는 pipecat 소스(`base_llm.py:653`)로 확인했다. 404는 `Error during completion: Error code: 404 - {...}`, 500/503은 `... Error code: 50x - {...}`, 스트림 중간 끊김은 두 형태다. 소켓 RST는 httpx.ReadError 로 `Error during completion: `(본문 비어 있음), 서버가 정상 종료한 chunked 미완결은 httpx.RemoteProtocolError 로 `Error during completion: peer closed connection without sending complete message body (incomplete chunked read)`. 현재 정규식은 401/403/429만 잡으므로 셋 다 unknown이다. 증거(scratch venv 의 pipecat-ai 1.12.0, openai 2.15.0, 가짜 서버; 정상 종료형 chunked 미완결은 별도 가짜 서버로 재현, Content-Length 부족형은 `(received N bytes, expected M)` 로 끝나지만 앵커가 `peer closed connection without sending complete message body` 까지라 같이 매칭, 테스트에 포함): 404 openai.NotFoundError, 500/503 openai.InternalServerError, 끊김 httpx.ReadError.
- (e) `defaultAITaskTimeout=30000`(`aicallhandler/main.go:112`)은 `start.go:415,661,1256`, `send.go:144`에서 지연 종료에 쓰인다. 지연 종료는 pipecat-manager의 일반 stop(`listenhandler/v1_pipecatcalls.go` `Terminate`)과 같아 pipeline_error를 내지 않는다. 그래서 응답 없이 끝나면 침묵이다. 단 conversation 타입은 `EventPMPipecatcallTerminated` backstop(`messagehandler/event.go:436-499`)이 이미 고정 문구로 응답한다. 침묵인 쪽은 contact_case(Insight, 테스트 에이전트 채팅)다. voice는 사용자 종료와 구분 불가라 대상이 아니다. task는 사용자 메시지가 없어 해당 없다.
- 외부 계약: OpenAPI에는 없고(notification content는 문자열), RST `bin-api-manager/docsdev/source/ai_struct_message.rst`의 category 표와 webhook `aimessage_created`의 content.category로 공개된다. 새 값 3개는 additive(기존 값 동작 불변)다.
- 중복 PR/Jira 없음. 관련 머지: VOIP-1542, VOIP-1599(#1399, #1438).

## 2. 설계 (d) 분류

새 category: `not_found`, `server_error`, `stream_interrupted`.

- 매칭은 기존 tier 0~3 뒤(tier 4)에서만 한다. 기존 4개 분류 결과는 불변이고 unknown이던 것만 재분류된다.
- 오탐 방지로 LLM 경로 접두사에 앵커한다. 소문자 텍스트가 `error during completion: error code: 404` 로 시작하면 not_found, 정규식 `^error during completion: error code: 5\d\d\b` 면 server_error, `strings.TrimSpace(s) == "error during completion:"` 이거나 `error during completion: peer closed connection without sending complete message body` 로 시작하면 stream_interrupted(두 끊김 형태 모두). 음성 세션 STT 일시 오류는 이 접두사가 없어 unknown 그대로다.
- `shouldNotifyPipelineError`: 세 값 모두 connection과 같이 항상 발행(LLM 경로 한정이므로).
- 고정 문구(평문, 원문과 키 미포함):
  - not_found: "The AI language model or its endpoint was not found. Verify that the model name is correct and that the model is available at the configured address or provider."
  - server_error: "The AI language model provider returned a server error. Try again later."
  - stream_interrupted: "The connection to the AI language model was interrupted before the response completed. Try again."
- 범위와 한계: OpenAI 호환 LLM 경로(접두사 일치)에 한정한다. customendpoint 세션은 `custom_endpoint_net.py:596-615 sanitize_error_text` 가 본문을 지워 `Error during completion: Error code: NNN` 만 남기므로 not_found/server_error 는 그대로 분류되나, 본문이 빈 끊김은 `Error during completion: error`(unknown)로 바뀌어 stream_interrupted 에 도달하지 않는다(스트림 읽기 중 전송 오류는 `APIConnectionError` 로 바뀌어 이미 connection). sanitizer 는 변경하지 않고 NNN 통과를 Python 테스트(`test_custom_endpoint_net.py:950` 표)에 404/5xx 케이스로 고정한다. 5xx 본문 우선순위와 504 HTML 본문 테스트는 플랫폼 관리 경로 전용이다. Gemini 경로(`Unknown error occurred: 404/500`), 400/413/422(컨텍스트 길이 초과 등)는 계속 unknown이다. 5xx 정규식은 OpenAI SDK 경유 5xx를 server_error로 분류하되, 본문에 `504 gateway timeout` 문구가 있으면 tier 3 timeout이 이기고 HTML 본문의 `gateway time-out` 은 server_error가 된다(Gemini 504는 기존대로 timeout). 두 경우를 테스트로 고정한다. 4xx 본문에 `'code': 401` 이 섞이면 기존 tier 2가 이기며 이것도 테스트 한 줄로 고정한다. 앵커는 `strings.TrimSpace(s)` 후 HasPrefix, 404 에도 `\b`. Anthropic 등 다른 SDK 형식은 계속 unknown이다. 본문이 빈 `Error during completion: ` 는 SDK가 빈 메시지 예외를 낼 때 모두 해당하나, pipecat은 httpx timeout을 `LLM completion timeout` 으로 먼저 분기하므로 timeout과는 겹치지 않는다(실측 예외 클래스는 probe.py 출력에 기록).
- tier 4는 tier 1~3 뒤이므로 본문에 기존 문구(예: `rate limit`)가 섞인 5xx는 기존 분류가 이긴다(이전에도 unknown이 아니었던 것은 불변). 이 우선순위를 테스트로 고정한다.
- 변경 파일: `pipelineerror.go`, `bin-pipecat-manager/models/message/pipeline_error.go`(상수와 상단 주석), ai-manager `pipeline_error.go`(문구 맵), 단위 테스트(`Test_shouldNotifyPipelineError` categories 에 connection 과 새 3개 추가), RST `ai_struct_message.rst`(표, 문구 단락, timeout/unknown 설명), `ai_struct_ai.rst`(Failures 단락 278행, OpenRouter ZDR 항목 234행: ZDR 미지원 모델의 상태 오류(404 등)는 이제 not_found/server_error 로 보고될 수 있음, 문구는 "404 등 상태 오류"로 단정을 피함. 줄 번호보다 문구로 찾는다), `ai_overview.rst:178` 점검, `self_hosting_providers.rst`(connection 설명 두 곳 점검), docs: pipecat `domain.md:55-59`, `operations.md:44`(category 목록), ai `domain.md:143`, `architecture.md`, `operations.md:101`(backstop 라벨), 코드 주석(`metrics_backstop.go` 라벨 목록, `EventPMPipecatcallTerminated` 상단 주석). RST는 clean rebuild 후 build 를 같이 커밋. mock 재생성 없음(인터페이스 불변).
- 호환: 구 ai-manager가 새 category를 받으면 unknown으로 저장한다(문구는 일반 문구). 배포 순서 무관, 롤백 안전. 신 ai-manager가 구 pipecat 이벤트를 받으면 기존 값만 온다.

## 3. 설계 (e) timeout 침묵 종료 알림 (aicall 측 판단)

`defaultAITaskTimeout` 값은 바꾸지 않는다. `EventPMPipecatcallTerminated`에서 `ReferenceTypeContactCase` aicall에 한해 다음을 모두 만족하면 기존 timeout 문구 notice를 만든다. 하나라도 아니면 아무것도 하지 않는다(현 동작 유지).

캐시된 aicall 로 상태(Terminating/Terminated)와 pipecatcall id 를 먼저 걸러(오래된 캐시는 알림 생략으로만 이어짐) 대기 goroutine 을 줄인 뒤, `backstopGraceSleep(backstopGraceDelay)` 로 진행 중 BotLLM 처리를 기다린 뒤, `AIV1AIcallGetSkipCache` 로 aicall 을 다시 읽어(캐시 불신) 판정한다. 근거: `SendReferenceTypeOthers` 는 `interruptPreviousPipecatcall` 후 `UpdatePipecatcallID` 순이라 이전 pipecatcall 의 terminated 가 새 id 반영 전에 도착할 수 있고, 이때 새 user 행이 최신이라 가짜 알림이 난다. sleep 후 재조회로 이 창을 닫는다. 재조회 실패 시 알림하지 않는다(현 동작 유지). 실패 정책: 재조회, 조건 3의 `MessageList`/`MessageAssistantReplyExists` 오류는 알림 없이 nil(fail-closed), 조건 4 스캔 오류는 알림 진행(fail-open, 기존 `pipelineErrorNoticeRecent` 와 같음). 각각 음성 테스트를 둔다.
1. 재조회한 `ac.Status`가 Terminating/Terminated가 아님(명시 종료 제외).
2. `ac.PipecatcallID == evt.ID`(오래된 pipecatcall 제외).
3. `MessageList(size=1, token="", aicall_id, deleted=false, role NotEq string(notification))` 의 최신 행이 role=user, 그리고 `MessageAssistantReplyExists(evt.ID)` 가 false. 도구를 실행한 턴은 RoleTool 행이 남아 최신 행이 user 가 아니므로 제외된다(미탐 허용). tool 행 없이 `run_llm=false` 로 끝난 턴은 허용된 오탐이다.
4. 새 헬퍼 `pipelineErrorNoticeBlocks(ctx, aicallID, pipecatcallID, category)` 한 번의 스캔(`pipelineErrorNoticeRecent` 와 같은 필터, notification 행만 `pipelineErrorNoticeScanSize`=50건)으로: 같은 pipecatcall_id 의 pipeline_error 가 어느 category 든 있으면 중단(이미 오류로 알림됨), 같은 aicall 의 timeout 알림이 10분 창 안에 있으면 중단. 후자는 category 단위라 진짜 timeout 알림 직후의 침묵 종료도, 같은 aicall 의 연속 두 턴 timeout 의 두 번째도 억제된다(기존 10분 창 규칙과 같은 의도된 동작). 스캔 50건 밖은 조회되지 않아 알림이 중복될 수 있다(fail-open).

메트릭: contact_case 를 `skipped_voice` 로 세지 않도록 contact_case 분기를 early return 앞에 두고 `promBackstopReplyTotal` 에 `timeout_notice_created`, `timeout_notice_failed`, `timeout_notice_skipped` 라벨을 추가한다(docs 갱신). 별도 메트릭 신설은 하지 않는다.

notice: category=timeout, fatal=true(세션이 중단됨), pipecatcall_id=evt.ID, 기존 timeout 문구. 생성 실패나 조회 실패는 로그만 남기고 nil을 반환한다(재큐 방지).

오탐 조건과 완화: (a) LLM이 빈 텍스트를 내고 끝나거나(run_llm=false, 도구만 쓴 턴 포함) 최신 행이 user 로 남으면 3번이 참이 되어 "응답 시간 초과"가 표시될 수 있다. 사용자 관점에서 응답이 없었다는 사실은 같고 Python 쪽 판정 없이 구분할 수 없어 허용한다. 문구는 사실(응답 없음)만 말한다. (b) 사용자가 채팅창을 닫아 정리된 경우는 aicall 종료 상태로 1번에서 제외된다. (c) 정상 응답은 assistant 행이 user 행 뒤에 있어 3번에서 제외. (d) 재조회 후 2번이 어긋나면 알림이 생략될 뿐(현 동작)이다. 표시 위험이 크다고 리뷰에서 판단되면 알림을 달지 않고 block 한다.

한계(허용): 초기 턴처럼 최신 행이 user 가 아닌 경로, interrupt 후 `UpdatePipecatcallID`/시작이 실패한 잔여 경쟁, 같은 terminated 이벤트의 동시 재전달(조건 4 동시 통과), 모든 contact_case terminated 마다 3초 grace 동안 대기하는 goroutine 이 이벤트 수만큼 생기는 점(구독 루프는 이벤트마다 `go` 로 처리해 막히지 않음, 구현 후 규모 확인). RST에는 같은 aicall 에서 10분 안의 반복 timeout 은 한 번만 기록된다고 적는다. 메트릭 의미 변화(contact_case 가 `skipped_voice` 에서 새 라벨로 이동)는 `metrics_backstop.go` 주석과 ai `operations.md` 에 적는다.

테스트(구현 시 작성): tool 행이 최신인 경우 무알림, 3초 grace 대기 goroutine 규모는 구현 후 확인(문제 시 sleep 전 캐시로 조건 1,2 선필터), RST 에는 fatal 의미를 약속하지 않음, listen 턴(foreign pipecatcall) 무알림, sleep 스텁 후 id 변경 시나리오, MessageList 인자(size=1, token 빈 문자열, NotEq, deleted=false) 검증, 라벨별 메트릭 증가, conversation backstop 회귀 없음, 분류기 테이블(404/500/503/스트림 끊김 양성, 기존 401/403/429/timeout/connection 불변, STT 접두사 없는 문장 unknown), `shouldNotify`, 문구 맵, 알림 발행 4조건 각각의 음성/양성, 10분 창, 오류 경로.
