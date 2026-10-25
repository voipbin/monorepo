# bin-route-manager unit test coverage 75% (VOIP-1652)

## 1. 이슈 분석 (티켓 검토)

판정: 유효. 기준선 `origin/main accc03e00`을 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 로 직접 재측정했다(작업 브랜치가 오래된 base 라서 먼저 origin/main 으로 rebase 했다). 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외, 같은 블록 중복은 최대 count) 합계는 2502 statement 중 1876개 커버, 74.98%. 원본(생성 코드와 mock 포함) 합계는 66.5%. 75.0%까지 최소 1개, 76%까지 25개(1901), 77%까지 약 50개가 더 필요하다.

- 관련 PR/브랜치: 이 경로의 선행 작업은 VOIP-1647(60%)과 VOIP-1651(70%) 설계 문서로 이미 머지되었다. 같은 경로를 바꾸는 열린 PR 은 없다(`gh pr list --state all --search "route-manager coverage"` 로 확인).
- 제품 코드 변경은 하지 않는다. 미커버 중 도달 불가 또는 후순위: cmd/route-control 315, cmd/route-manager 171(데몬 기동과 외부 연결 필요), listenhandler 와 telnyxclient 의 `json.Marshal` 실패 블록(고정된 구조체라 실패하지 않음), `rand.Read` 실패 등 약 3개. 이 구조 때문에 수기 합계 상한은 약 79%((2502-486-35-3)/2502)이므로 목표는 산술적으로 달성 가능하다.
- cmd/ 밖 도달 가능한 미커버(약 100개): dbhandler 의 provider/route/providercall 오류 경로(약 68), providerhandler/provider.go 와 setup.go(약 19), routehandler(5), providercallhandler(5).

반증하지 못한 주장: 기준선 수치(74.98%). 틀린 주장: 없음. VOIP-1416(80% 미만 패키지 일반 상향)이 적은 dbhandler(84.5%)는 80% 이상이라 겹치지 않는다. 80% 미만인 패키지는 cmd/ 두 개와 일부 models 뿐이다.

## 2. 설계 (계획)

범위: `*_test.go` 추가뿐이다(제품 코드, go.mod, go.sum 변경 없음). 외부 호출과 네트워크 의존 없음. 모든 대상은 기존 mock 과 sqlite 인메모리로 주입된다.

대상과 verify 항목(이 단계는 코드 없음, 구현 시작 후 테스트 작성):

1. `pkg/dbhandler` (provider, route, providercall 각각, 약 +45)
   - 닫힌 sqlite 연결(`sql.Open` 후 `Close`)을 쓰는 핸들러로 Create/Update/Delete/Get(DB 조회)/List 의 실행 오류 경로가 `could not execute query`/`exec failed`/`could not query` 문구를 포함한 오류를 반환하고 캐시 mock 은 호출되지 않는다.
   - List 에서 token 이 빈 문자열이면 `utilHandler.TimeGetCurTime()` 값이 사용되어 그 시각 이후 행이 제외된다.
   - 캐시 `ProviderSet`/`RouteSet` 이 오류를 반환하면 `providerUpdateToCache`/`routeUpdateToCache` 가 오류를 반환한다(Create 는 그 오류를 무시하므로 반환값은 nil).
2. `pkg/providerhandler`: Get 이 `dbhandler.ErrNotFound` 를 NotFound 도메인 오류(`PROVIDER_NOT_FOUND`)로 바꾸고 다른 오류는 그대로 전파, Create/Update 에서 잘못된 codecs 는 `INVALID_CODECS` 인수 오류이며 db 호출이 없다, Update 의 현재 provider 조회 실패는 오류 전파와 update 미호출. setup.go 의 캐리어 자격 생성 실패, 잘못된 PSTN 게이트웨이 포트, 롤백 시 Telnyx 삭제 실패 로그 경로는 기존 setup 테스트 패턴(httptest)으로 가능한 것만 추가한다.
3. `pkg/routehandler`: Get 의 NotFound 변환(`ROUTE_NOT_FOUND`), Dialroute 병합과 ListByTarget 의 provider 중복 시 customer 경로 우선(기본 경로 중복 제외, 순서 보존).
4. `pkg/providercallhandler`: 그룹콜 목록의 nil 요소 건너뜀, 정리용 FlowDelete 실패 시에도 원 오류를 반환.
5. 후속: 위로 77% 미만이면 setup.go, listenhandler 도달 가능 분기를 추가한다.

검증 방법:

- 수기 합계 76% 이상(권장), 75.0% 이상(인수). 측정은 bin-route-manager 에서 위 `go test` 명령과 후처리 스크립트(제외 필터, 같은 블록 최대 count)로 하고 PR 본문에 패키지별 전후 표와 수기/원본 합계를 적는다.
- 루트 CLAUDE.md 5단계(`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run`)와 `bash scripts/check-test-conventions.sh`를 실행하고 go.mod/go.sum 변경이 없음을 `git diff` 로 확인한다.
- 단언은 하드코딩한 기대값과 gomock 엄격 매칭으로 한다. 호출만 하거나 자기 비교인 테스트는 쓰지 않는다. 변이 검사는 격리 복사본에서 표본으로 하고 새 테스트만 실패해야 인정한다.
- 변경 파일은 bin-route-manager/ 안(`*_test.go`, 본 문서)뿐임을 `git diff --stat origin/main...HEAD` 로 확인한다.
- 공개 저장소 위생: 합성 값만 쓰고 자격증명과 주소를 같은 리터럴에 두지 않는다.
