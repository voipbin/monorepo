# bin-tts-manager unit test coverage 75% (VOIP-1652)

## 1. 티켓 검토

- 검토 결과: 유효. origin/main accc03e00 에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 을 재측정한 결과 수기 합계 71.39% (2054/2877), 원본 66.40% (2095/3155). 수기 기준은 gens/, mock_*.go, `Code generated ... DO NOT EDIT` 헤더 파일을 빼고 같은 블록은 최대 count 로 합친 값.
- 75% 까지 필요 statement 104개, 76% 권장 목표는 133개.
- 직전 VOIP-1647 계획서(60% 목표) 이후 streaminghandler 가 33%에서 77.7%로 올라 있음. 같은 목적의 열린 PR 과 중복 없음. 제품 코드 변경 없음, 테스트 파일만 추가.

## 2. 기준선 (수기)

| 패키지 | stmt | 커버 | % |
|---|---|---|---|
| cmd/tts-control | 110 | 53 | 48.2 |
| cmd/tts-manager | 142 | 4 | 2.8 |
| internal/config | 77 | 74 | 96.1 |
| pkg/audiohandler | 216 | 90 | 41.7 |
| pkg/buckethandler | 35 | 24 | 68.6 |
| pkg/cachehandler | 50 | 0 | 0.0 |
| pkg/dbhandler | 102 | 90 | 88.2 |
| pkg/listenhandler | 490 | 392 | 80.0 |
| pkg/speakinghandler | 177 | 154 | 87.0 |
| pkg/streaminghandler | 1276 | 992 | 77.7 |
| pkg/ttshandler | 192 | 172 | 89.6 |

## 3. 구조 분석

- 제외(테스트 어려움): `cmd/tts-manager` 의 main, run, runDaemon 등(프로세스 기동, 실제 DB/Redis/RabbitMQ, 138 stmt), `audiohandler/gcp.go` 와 `streaminghandler` 의 gcp/aws/elevenlabs `connect`/`Init`(외부 클라우드 인증과 네트워크).
- 테스트 가능: 미커버 오류 경로가 다수인 listenhandler(약 95), speakinghandler(약 23), dbhandler(11), buckethandler OSFileExist(8), cachehandler(약 44, miniredis), streaminghandler 의 websocket/start 오류 분기(약 10).
- 합계 추정 약 190 stmt 로 필요량 133 을 넘는 여유가 있으며, 부족하면 gcp 스트리머의 `closeStreamLocked`/keepalive 분기를 가짜 stream 으로 보충한다.

## 4. 추가할 테스트 (모두 로컬 전용, 외부 네트워크 없음)

1. listenhandler: `processRequest` 표 기반으로 speakings/streamings/speeches 각 엔드포인트의 잘못된 JSON(400 또는 오류), 하위 핸들러 오류(404/500 또는 오류 전파), 잘못된 URI, 페이지 크기 기본값 100, reference_type/reference_id 필터 전달, 타입 있는 오류와 ErrNotFound 의 응답 변환. `Run` 은 mock sockHandler 로 QueueCreate 실패와 성공(ConsumeRPC 호출)을 확인.
2. speakinghandler: Create 의 기존 세션 조회 실패, 동시 세션 경합에서 후순위 세션 중단, Get 의 NotFound 변환, Gets/Stop/Delete 의 DB 오류와 재조회 실패 시 기존 값 반환. 모두 mock dbhandler.
3. dbhandler: 기존 sqlite/mock 하네스로 SpeakingGet/Gets/Create/Update 의 쿼리 빌드 및 스캔 오류 경로, streaming 캐시 미스(ErrNotFound).
4. buckethandler: `OSFileExist` 를 임시 디렉터리의 존재 파일과 없는 경로로 확인.
5. cachehandler: miniredis(main 의 17개 서비스가 쓰는 v2.36.1, 테스트에서만 import)로 StreamingSet/Get 왕복, 키 형식 `tts:streaming:<id>`, TTL, 미존재 키 오류, 손상된 JSON 오류, Connect 성공과 서버 종료 후 실패.
6. streaminghandler: `websocketConnect` 의 MEDIA_START 읽기 오류, `Start` 의 생성 오류 전파.

## 5. verify 항목

- 수기 합계 76% 이상(최소 75.0%), 기존 테스트 전부 통과, `golangci-lint run` 통과.
- 각 테스트는 기대 응답 코드나 오류 문자열, 호출되지 않아야 하는 mock(미설정 EXPECT)로 검증하며 호출만 하는 테스트와 자기비교 테스트를 금지.
- 변이 검사: 표본으로 listenhandler 오류 분기의 상태 코드, cachehandler 키 접두어, speakinghandler 경합 비교 연산자를 scratch 복사본에서 바꿔 새 테스트가 실패하는지 확인.
- go.mod/go.sum 은 miniredis 추가 외 변경 없음, `go mod tidy` 후 diff 없음. 비테스트 .go 변경 없음. 파일 목록은 bin-tts-manager/ 안.
- 비밀번호 리터럴은 별도 문장에서 대입하고 `user:pass@host` DSN 을 쓰지 않는다.
