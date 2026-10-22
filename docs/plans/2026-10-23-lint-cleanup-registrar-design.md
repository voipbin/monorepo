# VOIP-1642 bin-registrar-manager lint cleanup (design)

## 1. 티켓 검토
유효. origin/main d2dcae915 기준 임시 설정(gocyclo, nestif 제거, 상한 해제)으로 golangci-lint 2.13.2 측정 결과 93건 (errorlint 89, wastedassign 3, gosec 1)로 티켓 수치와 일치. 관련 열린/머지 PR 중복 없음.

## 2. 범위
errorlint 89, wastedassign 3, gosec G114 1. 스타일 린터, gocyclo, nestif 제외.

## 3. 변경 계획
- errorlint 89: `golangci-lint run --fix` 로 fmt.Errorf 의 `%v` 를 `%w` 로 변경 (11개 파일, 89건). 메시지 문자열은 동일.
- wastedassign 3: `pkg/dbhandler/extension.go` 2곳, `trunks.go` 1곳의 `res = nil` 삭제. 직후 `res` 가 재대입되므로(`res, err = ...`, `res = tmp[0]`) 동작 불변.
- gosec G114: `cmd/registrar-manager/main.go` initProm 의 `http.ListenAndServe` 를 `http.Server{ReadHeaderTimeout: 10s}` 로 교체. 메트릭 엔드포인트의 헤더 읽기만 제한하며 핸들러(DefaultServeMux)와 통화 경로에는 영향 없음. 요청 ctx 없는 백그라운드 고루틴.
- 신규 파일 `cmd/registrar-manager/main_test.go`: initProm 의 /metrics 200 응답(sync.Once 로 -count=N 대응)과 promReadHeaderTimeout > 0 확인.

## 4. %w 소비 지점 검증
errors.Is/As 소비 지점: dbhandler.ErrNotFound/ErrDuplicate(sentinel), 검증 오류 타입(listenhandler), mysql.MySQLError(isDuplicateEntryErr, 원본 err 에만 적용). 새로 도달 가능해진 내부 오류(드라이버 오류, rows.Err)는 이 sentinel/타입과 같지 않으므로 분기가 바뀌지 않음. `%w: %w` 2곳(CustomerDomainCreate/Update)은 기존 ErrDuplicate 래핑 유지, 드라이버 오류가 추가 도달 가능. 모든 Errorf 인자는 `err != nil` 분기 안이라 nil 출력 없음.

## 5. verify 항목
- 임시 설정 린트 0건, 저장소 기본 린트 0건, 새 줄 린트 0건
- go mod tidy/vendor/generate, go test ./... 통과
- 비-ASCII diff 변화 없음
- initProm 이 /metrics 에 200 응답, promReadHeaderTimeout > 0 (신규 테스트)
