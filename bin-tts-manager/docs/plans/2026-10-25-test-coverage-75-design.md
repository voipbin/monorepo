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
| models/message | 1 | 1 | 100.0 |
| models/speaking | 9 | 8 | 88.9 |
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
- 테스트 가능: 미커버 오류 경로가 다수인 listenhandler(약 95), speakinghandler(약 23), dbhandler(11), buckethandler OSFileExist(11), cachehandler(50, miniredis), streaminghandler 의 websocket/start 오류 분기(약 10).
- 75% 는 제외 대상을 분모에 포함한 수기 합계 기준이다. cmd/tts-control(미커버 57)의 initCommand 등은 보충 후보.
- 합계 추정 약 190 stmt 로 필요량 133 을 넘는 여유가 있으며, 부족하면 gcp 스트리머의 `closeStreamLocked`/keepalive 분기 중 가짜 stream 으로 외부 호출 없이 닿는 분기에 한해 보충한다. 선행 계획(VOIP-1647)에서 시간에 민감한 기존 테스트로 약 9 stmt 의 측정 편차가 있었으므로 최종 수치는 반복 측정해 기록한다.

## 4. 추가할 테스트 (모두 로컬 전용, 외부 네트워크 없음)

도달 가능 stmt 추정(미커버 프로파일 기준. json.Marshal 실패 분기, squirrel 빌드 실패 분기처럼 입력으로 만들 수 없는 분기는 제외):

| 항목 | 추정 stmt |
|---|---|
| listenhandler v1_speakings (파싱/하위 오류/필터, Marshal 실패 분기 7개 제외) | 약 28 |
| listenhandler v1_streamings, v1_speeches (오류 전파) | 약 29 |
| listenhandler main (Run, processRequest 오류 변환) | 약 15 |
| speakinghandler | 약 20 |
| cachehandler (miniredis) | 약 45 |
| buckethandler OSFileExist | 11 |
| dbhandler (sqlmock 스캔/행/쿼리 오류, ApplyFields 오류, 캐시 오류 전파) | 약 6 |
| streaminghandler websocket/start | 약 4 |
| 합계 | 약 156 (필요 133, 여유 약 23). 부족하면 cmd/tts-control initCommand(구조 검사)와 gcp 스트리머 분기로 보충 |

1. listenhandler (`processRequest` 표 기반, 엔드포인트별로 기대값을 명시):
   - v1_speakings 는 모든 하위 오류를 응답으로 바꾼다(`nil` 오류 반환): 잘못된 JSON POST 400, Create 실패 500, 잘못된 목록 URI 400, 잘못된 id URI 400(`url.Parse` 는 `%zz` 같은 잘못된 퍼센트 인코딩에서만 실패하고 라우팅 정규식은 앵커가 없으므로 `/v1/speakings/<uuid>%zz` 형태 URI 로 구성), Get 실패 404(어떤 오류든), SayAdd/Flush/Stop/Delete 실패 404, JSON 파싱 실패 400. 목록 조회는 page_size 가 미지정(0), 1000(상한 초과), 숫자가 아닌 값일 때 모두 100 으로, 5 는 그대로 5 로 mock 인자에 전달되고(상한 비교 변이를 잡기 위함) mock 호출 인자로 전달됨, reference_type/reference_id 필터가 mock 인자로 전달됨, 조회 실패 500.
   - v1_streamings, v1_speeches 는 `nil, err` 로 오류를 전파한다: 잘못된 JSON 과 하위 핸들러 오류에서 `res == nil` 이고 반환 오류가 mock 이 준 오류와 같음. 잘못된 URI 도 오류 반환.
   - processRequest 의 변환 분기(v1_speakings 는 오류를 삼키므로 speeches/streamings 라우트로만 도달, `ToResponse` 실패 분기는 도달 불가라 제외): 하위 핸들러가 `VoipbinError` 를 반환하면 `cerrors.ToResponse` 결과 응답, `dbhandler.ErrNotFound` 면 404, 그 외 오류는 그대로 전파(`errorResponse` 자체는 기존 `error_response_test.go` 가 검증하므로 중복하지 않고 processRequest 경유 변환만).
   - `Run`: mock sockHandler 의 QueueCreate 실패 시 오류 반환(ConsumeRPC 미호출), 성공 시 nil 반환과 ConsumeRPC 호출(고루틴이므로 채널로 동기화).
2. speakinghandler (mock dbhandler): Create 의 기존 세션 조회 실패(`SpeakingGets` 오류) 전파, 동시 세션 경합. id 는 `uuid.NewV4` 라 고정할 수 없으므로 재조회 결과에 `uuid.Nil`(항상 더 작음) active 세션을 주어 후순위로 중단(SpeakingUpdate stopped 호출 확인)되는 경우와 `ffff...` 세션을 주어 중단되지 않는 반대 경우를 둘 다 검증, 재조회 오류 시 계속 진행. Get 의 `ErrNotFound` 가 NotFound 타입 오류로 변환됨, Gets/Stop/Delete 의 DB 오류 전파, Stop/Delete 후 재조회 실패 시 기존 값 반환.
3. dbhandler (sqlmock): 컬럼 수가 다른 행으로 `speakingGetFromRow` 스캔 오류, `SpeakingGet`/`SpeakingGets` 의 `RowError` 와 스캔 오류. 기존 `streaming_test.go` 가 이미 캐시 미스를 다루므로 추가하지 않는다. `ApplyFields` 오류, Query/Exec 오류와 `streamingGetFromCache` 의 non-Nil 캐시 오류 전파는 도달 가능하면 추가. 쿼리 빌드 실패 분기는 입력으로 만들 수 없어 제외.
4. buckethandler: `OSFileExist` 를 `t.TempDir()` 의 존재 파일(true)과 없는 경로(false)로 확인.
5. cachehandler: miniredis(origin/main 의 17개 서비스가 쓰는 v2.36.1(다른 1개는 v2.38.0), 테스트에서만 import)로 StreamingSet/Get 왕복(JSON 태그가 있는 필드를 필드 단위로 비교하고 `json:"-"` 필드와 Mutex 는 비교/복사하지 않음), 키 `tts:streaming:<id>` 와 miniredis `TTL(key)` 가 정확히 24시간, 미존재 키는 `redis.Nil` 이 그대로 반환됨(ErrNotFound 변환은 dbhandler 몫), 손상된 JSON 값은 파싱 오류, Connect 성공과 서버 종료 뒤 실패.
6. streaminghandler: 기존 `newMediaStartServer` 헬퍼를 재사용해 서버가 MEDIA_START 전에 즉시 연결을 닫는 읽기 오류 분기만 추가(바이너리 분기는 기존 테스트가 커버, 데드라인 설정 오류 분기는 도달 불가). `Start` 는 mock dbhandler 의 StreamingCreate 오류로 Create 실패가 그대로 전파됨.

## 5. verify 항목

- 수기 합계 측정: 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 후 scratch 의 계산 스크립트(gens/, mock_*.go, DO NOT EDIT 헤더 제외, 같은 블록은 최대 count)로 패키지별 전후 표와 수기/원본 합계를 산출. 최종은 `-count=3` 으로 3회 측정해 최소값이 76% 이상임을 기록.
- 수기 합계 하한 75.0%, 목표 76% 이상, 기존 테스트 전부 통과, `golangci-lint run` 통과, `-race` 로 신규 테스트 통과.
- 각 테스트는 기대 응답 코드, 반환 오류 동일성, 호출 인자, 호출되지 않아야 하는 mock(미설정 EXPECT)로 검증하며 호출만 하는 테스트와 자기비교 테스트를 금지.
- 변이 검사(scratch 복사본, 결과는 PR 본문에 요약): listenhandler 오류 상태 코드 변경, `Run` 의 QueueCreate 오류 분기 제거, cachehandler 키 접두어 변경, cachehandler TTL 24시간→1시간 변경, page_size 상한 100 제거, speakinghandler 경합 비교 연산자 반전, `OSFileExist` 반환값 반전 각각에서 신규 테스트가 실패해야 함.
- go.mod/go.sum: miniredis 와 그 전이 의존성 추가 외 변경 없음. `git diff go.mod` 로 go-redis 등 기존 require 버전이 오르지 않았음을 확인하고, `go mod tidy` 후 추가 diff 없음. 비테스트 .go 변경 없음. 파일 목록은 bin-tts-manager/ 안.
- 비밀번호 리터럴은 별도 문장에서 대입하고 `user:pass@host` DSN 을 쓰지 않는다.
