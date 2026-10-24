# bin-storage-manager 단위 테스트 커버리지 상향 (VOIP-1647)

## 1. 목표와 기준선

- 목표: 비테스트 코드를 바꾸지 않고 단위 테스트만 추가해 수기 코드 커버리지를 60% 이상(권장 61% 이상)으로 올린다.
- 수기 코드: `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일을 제외한 코드. 같은 블록의 중복은 최대 count 로 합친다.
- 측정: 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 후 위 규칙으로 집계한다.
- 기준선(origin/main d68e695b5): 수기 49.39% (1258/2547 statement), 원본 43.32%. 60% 에는 271개(⌈0.6×2547⌉-1258), 61% 에는 296개가 더 필요하다.
- `cmd/*` 의 main(670 statement)은 기동 시 DB, Redis, RabbitMQ, GCS 에 연결하므로 단위 테스트 대상에서 제외하고 `pkg/*` 에서 채운다.

| 패키지 | 기준선 | statement |
|---|---|---|
| cmd/storage-control | 0.0% | 490 |
| cmd/storage-manager | 0.0% | 180 |
| internal/config | 90.4% | 73 |
| models/account, models/file | 88.2%, 87.5% | 17, 16 |
| pkg/accounthandler | 74.8% | 155 |
| pkg/cachehandler | 0.0% | 87 |
| pkg/dbhandler | 83.5% | 346 |
| pkg/filehandler | 60.0% | 520 |
| pkg/listenhandler | 65.8% | 401 |
| pkg/storagehandler | 68.9% | 190 |
| pkg/subscribehandler | 70.8% | 72 |

## 2. 구조적 제약과 선택

- cachehandler 는 구체 `*redis.Client` 를 필드로 가진다. `Dial`, `ProcessHook`, `ProcessPipelineHook` 를 구현한 go-redis 훅이 get/set/del/ping 을 메모리 맵으로 처리하게 한다. 미존재 키는 훅이 `redis.Nil` 을 반환해야 한다. 네트워크를 쓰지 않는다.
- filehandler 는 구체 GCS 클라이언트를 쓴다. 기존 방식(httptest 서버와 `STORAGE_EMULATOR_HOST`)을 확장한 fake 가 객체 속성 조회와 삭제를 처리한다. 환경변수를 바꾸므로 이 테스트는 `t.Parallel` 을 쓰지 않는다. 서명 URL 은 테스트 안에서 만든 RSA 키로 만든다.
- 객체 본문 읽기, zip 업로드, 파형 계산은 fake 확장 비용이 커서 1단계 측정 결과가 61% 에 못 미칠 때만 추가한다.
- 나머지는 gomock 으로 충분하다. 새 의존성을 추가하지 않는다(`go.mod`, `go.sum` 불변).
- 제품 동작을 바꾸는 단언은 추가하지 않는다.

## 3. 테스트 항목

공통: 새 함수는 `Test_` 접두어(`scripts/check-test-conventions.sh`), gomock 컨트롤러 변수명 `mc`, 기존 이름과 겹치지 않게 `Test_<대상>_<시나리오>`, 합성값만 사용, 외부 호출 없음.

- cachehandler: 키와 TTL(`["set", key, []byte(JSON), "ex", int64(86400)]`), 저장 후 조회의 필드별 왕복 비교, 삭제 후 `redis.Nil`, 오류 전파, 깨진 JSON, NewHandler 와 Connect 성공/실패.
- listenhandler: 13개 라우트별 (메서드, URI) 가 올바른 핸들러 mock 만 호출하는지, 알 수 없는 URI 와 허용되지 않은 메서드의 404 와 `notfound` 지표 증가분(+1), 타입 오류와 `ErrNotFound` 의 응답 변환 및 그 외 오류 전파, 잘못된 본문과 필터 변환 실패의 400, 페이지 파라미터와 필터 전달, `Run`, NewListenHandler.
- storagehandler: `RecordingPeaks`(캐시 적중, nil 맵은 미스, 파일 하나의 계산 실패는 그 파일만 빈 값, 캐시 쓰기 실패 무시, 목록 오류), 파일/녹음/압축 경로의 오류 전파와 호출 중단, NewStorageHandler.
- accounthandler: 중복 계정, 생성/삭제/증감 실패와 재조회 실패, NotFound 의 타입 오류 변환, 10GB 경계(정확히 한도는 허용, 1 바이트 초과는 거부), 고객 이벤트 실패 경로, NewAccountHandler.
- subscribehandler: `Run` 의 각 실패 단계와 이후 호출 없음, 알 수 없는 이벤트, 잘못된 JSON, 파일 핸들러 실패 시 계정 핸들러 미호출(순서), `processEventRun` 비동기 처리.
- filehandler(1단계): `Delete` 의 단계별 실패(DB 삭제 실패와 재조회 실패는 이후 호출 없음, 버킷 삭제 실패와 계정 감소 실패는 계속), `DeleteBucketfile`, `IsExist`, `DownloadURIRefresh`(서명 형식, 갱신 필드), `DownloadURIGet`, 키 없음의 타입 오류, `EventCustomerDeleted`(한 파일 실패가 나머지를 막지 않음).
- dbhandler: 닫힌 DB 에서의 오류 전파(성공이나 not found 로 바뀌지 않음), 캐시 적중 시 DB 미조회, 캐시 실패 허용, 필드 없는 갱신, NewHandler.

## 4. 수용 기준과 검증

- 수기 합계 61.0% 이상(CI 방식 측정). 한 블록의 비결정성으로 측정이 ±1 statement 흔들릴 수 있다.
- `go test ./...` 통과, 신규 테스트의 `-race -count=3`, `golangci-lint run`, `go mod tidy` 후 `go.mod`/`go.sum` 불변, `git diff --stat origin/main...HEAD` 가 bin-storage-manager/ 안에만 있음, `scripts/check-test-conventions.sh` 통과.
- 변이 검사(별도 복사본에서만): 쿼터 비교, 캐시 키와 TTL, 라우터 상태 코드와 라우트 메서드, 오류 변환 분기, 연쇄 삭제의 `continue`, 구독 핸들러의 호출 순서, 서명 URL 필드, 캐시 분기 등 표본을 제품 코드에 넣어 신규 테스트가 실패하는지 확인한다.
- 금지: 빈 호출 테스트, 입력으로 기대값을 만드는 자기비교.
