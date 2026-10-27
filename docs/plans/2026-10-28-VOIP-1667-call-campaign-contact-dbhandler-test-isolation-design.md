# VOIP-1667 bin-call-manager, bin-campaign-manager, bin-contact-manager pkg/dbhandler 테스트 격리 설계

## 1. 티켓 검토
유효. origin/main 9b798099b 에서 세 서비스의 `go test -count=2 ./pkg/dbhandler` 와 `-shuffle=on` 이 실패한다. 원인은 VOIP-1655 와 같다. 패키지 전체가 공유하는 in-memory sqlite(TestMain 1회 스키마)에 고정 id 와 고정 업무 키로 INSERT 하고 정리하지 않는다. 제품 코드는 원인이 아니다.

수정 전 실측(`-test.count=1 -test.shuffle=on` 10회, 컴파일된 테스트 바이너리):
- call: 통과 6, 실패 4(그중 panic 4). `-count=3` FAIL 40줄.
- campaign: 통과 5, 실패 5. `-count=3` FAIL 58줄.
- contact: 통과 5, 실패 5. `-count=3` FAIL 152줄.
즉 shuffle 은 순서 의존이라 통과하는 시드가 있어 flake 로 보이며, 인수 조건 검증은 여러 시드로 반복해야 한다.

call 의 panic 은 `channel_test.go` 의 목록 비교 실패 메시지가 `res[0]`, `tt.expectRes[0]` 를 직접 읽는 데서 난다(결과가 비면 index out of range). 같은 패턴이 campaign 의 `campaigncall_test.go` 4곳과 `campaign_test.go` 1곳에 있다. 나머지 `res[i]` 접근은 앞선 `len(res)` Fatalf 로 보호됨을 확인한다.

## 2. 범위
제품 코드 무변경. 세 서비스의 `pkg/dbhandler/*_test.go` 와 이 문서만 변경. 나머지 서비스는 건드리지 않는다. 별도 드라이버나 private DB 를 쓰는 테스트(contact `ctx_rows_test.go` 등)는 공유 DB 를 쓰지 않으므로 영향이 없고 호출해도 무해하다.

## 3. 설계
- 각 서비스 `main_test.go` 에 VOIP-1655 와 같은 `resetTestDB(t *testing.T)` 를 추가한다. `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'` 로 전 테이블을 읽고(새 테이블 자동 포함, sqlite 내부 테이블 제외), rows 를 `rows.Err()` 확인 뒤 즉시 닫고(MaxOpenConns(1) 데드락 방지), 각 테이블을 `DELETE FROM` 한다. 실패하면 `t.Fatalf`.
- `TestMain` 을 제외한 모든 `func Test...(t *testing.T)` 첫 줄에서 예외 없이 호출한다(VOIP-1655 와 같음. sqlmock, private DB 테스트는 공유 DB 를 쓰지 않아 무해하고, 파일별 판별보다 안전하다). 시작 시점 정리라 선행 실패에 강하다. 기존 DELETE, 단언은 변경하지 않는다. 삭제되는 줄은 아래 6개 `t.Errorf` 줄뿐이다.
- 실패 메시지 panic 정정: call 과 campaign 의 위 6곳에서 `[0]` 접근을 nil 안전한 요약 출력(`describeList`, 해당 서비스 `main_test.go`)으로 바꾼다. 비교 조건(`reflect.DeepEqual`)은 그대로이므로 단언은 약해지지 않는다.
- 알려진 한계: 격리 단위는 Test 함수이며 함수 안 t.Run 사이는 격리하지 않는다. `-shuffle` 은 최상위 Test 순서만 섞고 서브테스트 순서는 섞지 않는다(현 순서에서 통과 상태 유지). 신규 Test 가 호출을 빼먹는 것은 구조로 막지 않는다(과설계 회피).
- 대안 (b) 테스트별 이름 DB: 마지막 커넥션이 닫히면 shared DB 소멸 위험. (c) 고유 id: 고정 id 기대값 테스트가 많아 변경량 과다.

## 4. 검증 기준
- 전제: `t.Parallel` 0건 확인.
- 호출 누락: `grep -A1 '^func Test' *_test.go` 에서 다음 줄이 `resetTestDB(t)` 가 아닌 함수가 TestMain 뿐이다(주 기준: `resetTestDB(t)` 호출 수 = `^func Test` 수 − 1. call 146, campaign 41, contact 143).
- 비테스트 소스 무변경: `git diff --name-only origin/main` 에 `_test.go` 와 이 문서 외 파일이 없다. `git diff --numstat` 의 삭제 컬럼 합계는 6(6개 `t.Errorf` 줄이 `describeList` 를 쓰는 줄로 바뀜)이다.
- 서비스별 `-count=3`, `-shuffle=on -count=2`, `-count=20 -shuffle=<1|7|42|99|12345>` 와 임의 시드 다회 통과, `-race -count=5`(dbhandler 패키지 한정). 서비스 `./...` 전체 `-shuffle=on -count=2` 도 확인한다(다른 패키지 영향 없음).
- 음성 확인(커밋 금지): reset 에서 한 테이블 제외 또는 SQL 오류를 넣어 실패 확인 후 복원. `res` 와 `expectRes` 가 각각 비었을 때 메시지 생성이 panic 하지 않음을 임시 변경으로 확인. 음성 확인 뒤 `git diff` 로 복원을 확인한다.
- 서비스별 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`, `scripts/check-test-conventions.sh`, CI.
- 범위 밖 기록: bin-contact-manager 의 `internal/config`, `pkg/casehandler` 는 변경 전 main 에서도 `-count=2` 에 실패한다(dbhandler 아님, 이번 PR 은 수정하지 않는다).
- 검증 보강: `git diff -U0 -- '*_test.go'` 로 삭제 줄이 6개 `t.Errorf` 뿐임을 직접 확인한다.
