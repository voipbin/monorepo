# bin-conference-manager unit test coverage to 70% (VOIP-1651)

## 1. 티켓 검토 (분석)

판정: 유효. origin/main 99b59c386 에서 재측정했고 카드 수치와 일치합니다.

- 수기 합계 69.38% (1894/2730 statement), 원본 61.45%. 70% 까지 21개, 71% 권장선까지 45개를 더 커버해야 합니다.
- 제외 방법: 경로에 `/gens/` 가 있거나 파일명이 `mock_` 로 시작하거나 파일 헤더(package 줄 이전)에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일. 같은 블록의 중복은 최대 count 로 합칩니다.
- 패키지별 전(수기): cmd/conference-control 0.0 (510), cmd/conference-manager 0.0 (162), internal/config 97.8 (135), models/conference 97.1 (35), models/conferencecall 90.0 (20), pkg/cachehandler 0.0 (59), pkg/conferencecallhandler 98.5 (271), pkg/conferencehandler 96.8 (467), pkg/dbhandler 89.9 (425), pkg/listenhandler 93.7 (569), pkg/subscribehandler 98.7 (77).
- 구조 확인: cmd/ 두 main(672)은 cobra 와 외부 연결(DB, Redis, RabbitMQ)을 엮는 main 이라 대상에서 제외합니다. pkg/cachehandler(59)는 go-redis 클라이언트를 직접 쓰지만 miniredis v2.36.1 이 main 의 다른 서비스 10곳에 이미 있어 테스트 전용 의존성으로 추가할 수 있습니다(이전 VOIP-1647 에서는 부재로 제외했던 대상).
- 중복 여부: 이 서비스 대상으로 열린 커버리지 PR 없음.
- 달성 산술: 제외 672 를 빼도 cachehandler 59 만으로 1953/2730=71.5%.

## 2. 설계

범위는 `_test.go` 추가와 go.mod/go.sum 의 miniredis 추가뿐입니다(제품 코드 무변경). 단언은 `reflect.DeepEqual` 과 `t.Errorf`, 신규 함수는 `Test_` 접두, gomock 컨트롤러 변수는 `mc`. 네트워크는 miniredis 의 루프백 인스턴스만 씁니다.

- pkg/cachehandler (신규 handler_test.go): 각 Set/Get 왕복(Conference, Conferencecall)과 24시간 TTL 설정 확인, 키 부재 시 오류, 저장값이 JSON 이 아닐 때 오류, 서버 종료 후 Set 오류, ConferencecallGetByReferenceID 는 사전 주입한 키 조회와 부재 오류, NewHandler 와 Connect 성공 및 서버 종료 후 실패.
- pkg/conferencehandler (remove_test.go 에 케이스 추가): RemoveConferencecallID 의 DB 제거 실패와 재조회 실패 시 `nil, err` 및 webhook 미발행, removeConferencecallIDTypeConference 의 Destroy 실패 전파.
- 위로 71% 가 모자라면 dbhandler/listenhandler 오류 경로를 추가합니다.

## 3. verify 항목

- `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 수기 합계 71% 이상.
- 변이 확인: cachehandler 의 TTL 값과 키 접두를 바꾸면, remove.go 의 오류 반환을 nil 로 바꾸면 해당 테스트가 실패해야 합니다.
- `go mod tidy` 후 go.mod/go.sum 변경이 miniredis 와 그 전이 의존성뿐이고, 제품 코드 diff 가 없음. golangci-lint 통과.
- VOIP-1416 과의 중복: pkg/cachehandler, pkg/conferencehandler, pkg/dbhandler 가 80% 미만 대상일 수 있어 PR 본문에 명시만 합니다.
