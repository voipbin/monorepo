# VOIP-1663 bin-call-manager unit test coverage to 90%

## 1. 티켓 검토
유효. origin/main 0bfbe2a1d 에서 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외) 11738 statement 중 10228 이 커버되어 87.14% 입니다. 90.0% 까지 약 337, 90.5% 까지 약 395 statement 가 더 필요합니다. 열린 중복 PR 은 없고 병합된 커버리지 PR(#1603, #1648)은 기준선에 포함되어 있습니다.

## 2. 범위
- 신규 `*_cov2*_test.go` 파일 추가만 합니다. 기존 `_test.go`, 비테스트 .go, 시그니처, go.mod 는 바꾸지 않습니다.
- 목표: 수기 합계 90.5% 이상 (최소 90.0%). 90.5% 에 못 미치면 사유를 PR 본문에 보고합니다.

## 3. 접근과 패키지별 목표 증가량(statement)
기존 테스트 패턴(gomock 오류 주입, `newTXTestHandler` sqlmock)을 따릅니다. 외부 연결이 필요한 생성자, 설정 바인딩, 직렬화 실패, 쿼리 빌더 실패는 대상이 아닙니다.
- pkg/callhandler 약 +190 (action, start, outgoing_call, hangup, digit, media, recovery 의 오류 반환과 분기)
- pkg/dbhandler 약 +90 (쿼리 실행, 행 스캔과 반복 오류, ErrNotFound)
- pkg/confbridgehandler 약 +60, channelhandler 약 +45, arieventhandler 약 +30, recordinghandler 약 +25, groupcallhandler 약 +20, listenhandler 약 +20
- pkg/testhelper +13, internal/config 와 cmd/* 의 외부 연결 없는 함수 약 +20

## 4. 테스트 규칙
- check-test-conventions.sh: 함수명 `Test_` 접두, gomock controller 변수명 `mc`, 비교는 `reflect.DeepEqual` + `t.Errorf`, testify 금지.
- 여러 시나리오는 테이블 드리븐(행마다 `name`)으로 쓰고, sqlmock 은 `ExpectationsWereMet` 를 확인합니다.
- 반환값 또는 mock 호출 인자를 기대값과 비교합니다. 같은 값을 만들어 비교하는 테스트와 빈 호출 테스트는 쓰지 않습니다.
- `time.Sleep`, 실제 네트워크, 환경과 시간 의존을 쓰지 않고 합성값만 씁니다.
- 현재 동작이 결함으로 보이면 테스트로 고정하지 않고 PR 본문 또는 별도 이슈로 보고합니다.

## 5. Verify 항목
1. `go test -count=1 ./...` 전부 통과, 변경 패키지는 `-race -count=3` 통과.
2. 측정: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 후 gens/, mock_*.go, DO NOT EDIT 헤더 파일을 제외하고 같은 블록은 최대 count 로 합쳐 수기 합계를 계산하며 패키지별 전후 표를 PR 본문에 적습니다.
3. 변이 검사: scratch 복사본에서 제품 코드의 오류 처리 분기를 제거하면 신규 테스트가 실패함을 표본 확인.
4. `git diff --stat origin/main...HEAD` 가 bin-call-manager/ 안의 신규 `_cov2*_test.go` 와 이 문서뿐임.
5. golangci-lint, scripts/check-test-conventions.sh 통과.
