# 2026-10-24 bin-pipecat-manager 단위 테스트 커버리지 60% 설계 (VOIP-1647)

## 1. 티켓 검토

판정: 유효. origin/main d68e695b5 에서 수기 코드 59.9% (1444/2412), 원본 55.0% 를 재측정해 카드 수치와 일치함을 확인했습니다.
제외 방법: gens/, mock_*.go, `Code generated ... DO NOT EDIT` 헤더 파일을 빼고 같은 블록은 최대 count 로 합칩니다.

| 패키지 | 수기 statement | 미커버 |
|---|---|---|
| pkg/pipecatcallhandler | 1332 | 333 (75.0%) |
| cmd/pipecat-control | 268 | 268 (0%, main/CLI) |
| cmd/pipecat-manager | 128 | 127 (main) |
| pkg/httphandler | 98 | 98 (0%) |
| pkg/listenhandler | 198 | 54 (72.7%) |
| pkg/cachehandler | 50 | 50 (0%, redis 의존) |
| pkg/dbhandler | 130 | 29 (77.7%) |

## 2. 계획

기준: 브랜치 HEAD 2fce50a(origin/main d68e695b5 의 후손, bin-pipecat-manager 변경은 README 1줄뿐이라 수치 동일, 이후 커밋은 이 문서뿐). 제품 코드는 바꾸지 않고 `_test.go` 만 추가합니다. 목표 61%는 1472 statement 이상(기준 1444 대비 +28)이며 아래 합계 약 +65 를 계획합니다. 실제 증가량은 구현 후 coverprofile 로 측정해 보고합니다. 새 테스트 이름은 `Test_` 로 시작하고 gomock 컨트롤러는 `mc` 입니다.

1. pkg/httphandler/main_test.go (신규, 약 +25): Run() 의 라우트(`GET /:id/ws`, `POST /:id/tools`, `POST /:id/member-switched`)와 같은 패턴으로 gin 라우터를 구성하고 httptest 로 호출합니다. 세 핸들러 각각에 대해 (a) 잘못된 id: 400, 본문 비어 있음, 핸들러 미호출, (b) 핸들러 오류: DoAndReturn 에서 id 를 정확한 UUID 로 매칭해 호출 확인, 400 과 본문이 정확히 `{"error":"<오류 문구>"}`, (c) 성공: 정확한 UUID 로 호출, 상태 200. 라우터를 테스트가 구성하므로 메서드 불일치 단언은 쓰지 않으며, 라우트 패턴은 Run() 과 같게 맞춘다는 한계를 코드 주석에 적습니다. 인자에 gomock.Any() 를 쓰지 않습니다. Run() 은 고정 포트 listen 이라 제외합니다.
2. pkg/listenhandler (기존 테스트와 중복 없이 실제 미커버만, 약 +25): 중복 방지를 위해 TestProcessRequest_NotFound, Post 의 InvalidJSON 과 오류, IDGet 과 IDStopPost 의 InvalidURI, Ping 오류, errorResponse 6케이스는 이미 있으므로 추가하지 않습니다.
   - processRequest 를 통한 호출(디스패처 오류 매핑 main.go 267-281): VoipbinError 는 해당 상태 코드, dbhandler.ErrNotFound 는 404, 일반 오류는 400. 혼합 케이스(ErrNotFound 를 감싼 VoipbinError)는 순서 고정이 아니라 "VoipbinError 상태 코드가 우선한다"는 회귀 단언입니다(두 case 가 모두 errorResponse 를 호출해 순서 교체는 동등 변이). IDGet 과 IDStopPost 의 핸들러 오류(`return nil, err`)가 디스패처에서 이 매핑으로 이어지는지 단언합니다.
   - Ping 라우트 분기(main.go 246-247)와 processV1PingGet 의 marshal 실패 분기(v1_ping.go 30-32): PingResult.Timestamp 연도를 10000 으로 주면 time.Time.MarshalJSON 이 실패하므로 500 을 단언.
   - runListenQueue, runListenQueueVolatile, Run(main.go 131-192): MockSockHandler 로 QueueCreate 와 ConsumeRPC 기대. ConsumeRPC 는 고루틴에서 불리므로 mock 의 Do 와 done 채널로 호출을 기다린 뒤 mc.Finish 하여 -race 와 flaky 를 피한다(143-145행 오류 분기도 동일). 오류 반환 시 전파 확인. errorResponse 의 ToResponse 실패 분기는 도달 불가라 제외.
3. pkg/dbhandler (기존 파일에 추가, 약 +15): 대상은 pipecatcallGetFromDB 의 없는 id ErrNotFound(119-122, PipecatcallGet 경유 호출), PipecatcallDelete 의 rowsAffected==0 ErrNotFound, 캐시 오류 시 DB 폴백과 캐시 Set 실패 무시, PipecatcallUpdate 빈 fields 의 nil 반환, 별도로 sql.Open 한 DB 를 닫아 Exec 와 Query 오류(TestMain 이 쓰는 공유 dbTest 는 닫지 않음), NewHandler(main.go 40-46). squirrel ToSql 실패 분기는 도달 불가라 제외. 패키지 수치는 이 문서 전체에서 수기 코드 기준(dbhandler 77.7%)이며 mock 포함 원본 값은 63.5% 입니다.
4. 여유분 후보(필요 시): pkg/pipecatcallhandler 의 미커버 오류 경로. 구체 함수는 구현 단계에서 coverprofile 의 함수별 값으로 고릅니다. cmd/ 와 cachehandler 는 제외.

## 3. verify 항목

- 모든 단언은 기대값을 독립적으로 적고 상태 코드, 본문, mock 호출 인자를 확인합니다. 자기비교 금지.
- CLAUDE.md 5단계(go mod tidy, vendor, generate, test, golangci-lint) 통과 후 go.mod 와 go.sum 변경 없음. `go test -race -count=1 ./...` 통과, 수기 합계 61% 이상(coverprofile 실측).
- scripts/check-test-conventions.sh 통과(git fetch origin main 선행): 새 테스트 `Test_` 이름, 컨트롤러명 mc, testify 미사용.
- 변이 표본(scratch 복사본에서만): httphandler id 검증 제거, 응답 코드 400 변경, processRequest 의 ErrNotFound case 삭제(404 가 400 이 되어야 실패)와 VoipbinError case 삭제(타입 상태 코드가 400 이 되어야 실패), PipecatcallDelete 의 rowsAffected 검사 제거 각각에서 새 테스트가 실패함을 확인합니다.
- `git diff --stat origin/main...HEAD` 가 bin-pipecat-manager/ 안의 `_test.go` 와 이 문서만 포함.
