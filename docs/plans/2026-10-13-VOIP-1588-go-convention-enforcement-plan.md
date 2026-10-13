# VOIP-1588 Go 컨벤션 CI 집행 구현 계획

- 문서: 구현 계획. 설계는 `2026-10-13-VOIP-1588-go-convention-enforcement-design.md`(이하 설계 N.N)이며 이 문서는 설계를 바꾸지 않고 구현 순서와 설계가 계획 단계로 미룬 값만 정한다.
- 용어: 담당자와 후속 확인 담당은 설계 문서 서두의 용어 줄과 같은 뜻이다.
- 기준: origin/main b3c97aea8 (착수 시 fetch로 설계 전제 재확인: DTO import 위반 0, squirrel 허용 밖 import 0, config.yml 66/67행 존재).

## 1. 결정 사항 (설계가 계획으로 미룬 값)

| 항목 | 결정 |
|---|---|
| 스캐너 코드 크기 상한 | `check-structure-conventions.sh` 전체 600줄 이하, 그중 awk 프로그램 250줄 이하. 넘으면 규칙을 줄이지 않고 작업을 중단한 뒤 담당자에게 확인한다 |
| self-test 범위 상한 | 사례 90개 이하(awk 변형별 반복 실행은 사례 수에 세지 않고, 사례마다 ID `Sxx` 를 붙여 중복 열거를 막는다). 사례는 표 기반 헬퍼(`expect <id> <rc> <패턴> <<<fixture`)로 한 사례 3줄 이내로 쓴다. 코드 크기 상한은 awk 프로그램 250줄, 셸(래퍼, 허용 목록, 표 대조) 130줄, self-test 헬퍼와 픽스처 220줄로 나누어 세며 합계 600줄 이하다(셸과 self-test 사이 예산 이동은 합계 600줄 안에서 허용). 시간 상한: self-test 전체(두 변형) 60초 이내이며 임시 저장소는 사례마다 새로 만들지 않고 변형별 1개를 재사용해 파일만 바꾼다. 초과하면 규칙을 줄이지 않고 사례를 병합하거나, 병합으로도 안 되면 작업을 중단하고 담당자에게 확인한다. 임시 git 저장소는 `$TMPDIR`(없으면 `mktemp -d`) 아래에 만들고 종료 시 삭제. 네트워크/실제 저장소 의존 없음 |
| fetch 단계 복제 | 공용 command 로 빼지 않는다. 기존 `check-test-conventions` 의 "Fetch base branch" 와 같은 본문을 3개 command 와 lint-config-check 에 복제한다(기존 패턴 재사용, 오버엔지니어링 회피) |
| awk 변형 | self-test 는 `command -v awk` 와 `command -v mawk` 를 각각 실행하고 변형 이름과 실제 경로를 출력한다. CI(Ubuntu cimg)에서는 awk 가 mawk 여서 같은 바이너리일 수 있으며 중복 실행은 허용한다(gawk 호환은 로컬에서만 검증됨을 주석에 명시). 둘 중 하나라도 없으면 건너뛰지 않고 rc=1(의도된 동작, 스크립트 주석에 명시). 모든 awk 호출은 `LC_ALL=C` 로 고정한다 |
| awk 이식성 금지 목록 | `gensub`, `asort`, `\b`/`\w`/`\s`, 정규식 interval `{n}`, `switch`, `BEGINFILE`/`ENDFILE`, `PROCINFO`, `length(배열)` 를 쓰지 않는다. `getline < file` 반환값이 -1(읽기 실패)이면 열거된 Go 파일, 허용 목록, 파일 목록 모두 gate 실패 |
| R2 처리 범위 | 모호함을 없애기 위해 확정: R2 는 원문에 `listenhandler/models/` 또는 `Masterminds/squirrel` 문자열(백틱 import 경로도 이 문자열을 포함하므로 같은 필터로 잡힘)이 있는 파일(약 260개)에만 상태기계를 적용한다. 줄마다 "줄 시작 상태가 코드인가" 플래그를 기록하고, 플래그가 참인 줄에서만 `import` 를 인식한다(본문 raw string 이나 주석 속 `import "x"` 는 무시). import 블록은 `import (` 로 열고 코드 상태의 첫 `)` 로 닫으며, 닫히지 않은 블록은 gate 실패. import 구간 안의 `/* */` 는 상태기계가 제거한 뒤 처리한다 |
| awk 파일 핸들 | 파일마다 `close()`(목록, 허용 목록, Go 파일 모두)를 호출한다(mawk 의 열린 파일 수 제한 회피). CI 이미지의 mawk 는 로컬(1.3.4)보다 구버전(Ubuntu 20200120)일 수 있으므로 이식성 금지 목록을 지키고 CI 로그로 확인한다 |
| 성능 상한 | 실측: mawk 에서 전 파일 문자 루프는 약 23초이므로 상태기계는 필요한 파일에만 적용한다. 줄 단위 프리필터: R2 는 위 R2 처리 범위의 프리필터를 쓰고(백틱 단독 프리필터는 struct tag 때문에 1291개가 걸려 쓰지 않음), R3 는 `^bin-[^/]+/models/` 비테스트 파일(441개)만 상태기계로 처리한다. 파일을 한 문자열로 잇지 않고 줄 배열과 줄 내 인덱스로 처리한다. 전체 스캔 소요 상한: 실제 CI(small, mawk) 기준 60초 이내이며 측정은 로컬 mawk(`mawk` 명시 실행)로 하고 CI 로그에서 재확인한다. 초과하면 작업을 중단하고 담당자에게 확인한다 |
| mawk 확보 | gate job 과 lint-config-check 에 `sudo apt-get update && sudo apt-get install -y mawk` 단계(shell-tests 와 동일) |
| 열거 | `git ls-files '*.go'` 를 파일로 받아 종료 코드와 0건을 검사. 실패 또는 0건이면 rc=1 |
| 백틱 import | import 구간(import 문 안)에서 백틱을 만나면 rc=1 |
| 실행 환경 | 스크립트는 `git rev-parse --show-toplevel` 로 루트로 이동해 실행하므로 cwd 독립. 허용 목록 경로는 스크립트 위치 기준 |

## 2. 파일별 구현

1. `scripts/check-structure-conventions.sh` (신규)
   - 모드: 인자 없음(저장소 검사), `--self-test`. 내부 함수 `run_check <repo> <awk> <allow> <dbdoc>` 를 두 모드가 공유한다(인자 없는 본 검사의 awk 는 mawk 가 있으면 mawk, 없으면 `awk`)(self-test 는 임시 저장소를 같은 함수로 검사).
   - awk 프로그램 한 개가 파일 목록을 `getline` 으로 읽어 파일 단위로 처리한다.
     (1) 파일 전체를 읽고 문자 단위 상태기계로 주석을 제거한다. 상태: 코드, 줄 주석, 블록 주석, 문자열, raw 문자열, rune. 주석 제거본 C[i](리터럴 유지)와 리터럴 내용을 비운 B[i](중괄호 계산용)를 만든다. 닫히지 않은 블록 주석/문자열/rune/raw 문자열은 gate 실패.
     (2) R2: C[i] 의 import 구간(`import "x"`, `import (` 블록, 한 줄 괄호 import)에서 따옴표 경로를 모두 추출한다. 별칭/`_`/`.` 허용. 백틱은 rc=1(import 구간 안에서만 판정하고, import 밖의 raw 문자열 태그는 영향 없음: 반례를 self-test 에 포함). 경로가 `^monorepo/<S>/pkg/listenhandler/models/(request|response)(/|$)` 이면 파일 경로가 `^[^/]+/pkg/listenhandler/`(요소 단위) 또는 `^bin-common-handler/pkg/requesthandler/` 일 때만 허용. 경로가 `github.com/Masterminds/squirrel` 이면 파일 경로가 `^bin-[^/]+/pkg/dbhandler/`, `^bin-common-handler/pkg/databasehandler/`, `^bin-timeline-manager/pkg/analysisdbhandler/` 일 때만 허용(요소 단위, 뒤에 `/`).
     (3) R3: 파일 경로가 `^bin-[^/]+/models/` 이고 `_test.go` 가 아니면 B[i] 깊이를 추적해 최상위 `type X struct {` 블록의 깊이 1 필드만 모은다. ID 줄(`(^|[^[:alnum:]_])ID[[:space:]]+[][*]*[[:alnum:]_]+\.UUID` 와 `db:"id[,"]`), CustomerID 줄(동일 형태, `db:"customer_id[,"]`)이 한 블록에 모두 있으면 키 `<디렉터리>:<타입명>` 을 수동 정의로 기록한다. 정의 줄이 인식된 블록 깊이 1에 귀속되지 않으면 gate 실패(`type (` 그룹, 중첩/익명 struct, 한 줄 struct, 제네릭). 한 줄 `struct{}` 는 열고 닫음이 한 줄이면 정상 처리한다. 블록 안에 또 `type ... struct {` 가 열리면 귀속 실패.
     (4) 교차 검증: 모델 범위 파일의 원문에서 `//` 로 시작하는 줄을 제외하고 정의 패턴 줄 수를 단순 패스로 세어 귀속 줄 수와 다르면 gate 실패.
     (5) 허용 목록(`scripts/structure-conventions-allow.txt`): 첫 필드 전체 일치. `#`/빈 줄 무시, 공백/탭/CR 구분. 중복 키, 읽기 실패, 유효 0개는 gate 실패. 발견 집합에 없는 허용 키(stale)는 gate 실패, 허용 밖 발견 키는 위반.
   - R2b 허용 경로 상수를 database.md 의 R2b 예외 표와 매 실행 grep 대조한다(표에 경로 문자열이 없으면 gate 실패).
   - 위반 메시지는 `file:line`, 규칙, 인용 문서 절(CLAUDE.md Layering, database.md 7.1/7.7, models.md 6.1)을 포함한다. 인용 절의 존재는 6장 검증에서 grep 한다.
   - 종료 코드: 0 통과, 1 위반 또는 gate 실패(메시지에서 구분). 위반/실패가 둘 다 있으면 모두 출력한다.
   - 주석에 "git add 하지 않은 새 파일은 검사되지 않음(CI 무관)" 과 `[*\[\]]` → `[][*]*` 표기를 남긴다.
2. `scripts/structure-conventions-allow.txt` (신규): 설계 3.3 의 11개 키와 사유 한 줄.
3. `.golangci-errorlint.yml` (신규): 설계 3.1 골격(`version: "2"`, `default: none`, errorlint `errorf: true`, `comparison: false`, `asserts: false`, `max-same-issues: 0`, `.golangci.yml` 과 같은 exclusions).
4. `.circleci/config_work.yml`
   - `go-test`, `go-test-api-manager`, `go-test-pipecat-manager` 3개 command 의 `when: enable-lint` 블록에서 기존 `Linting` 뒤에 (a) origin/main 확보 단계(check-test-conventions 의 본문 복제), (b) `Lint error wrapping on new lines` 단계(같은 GOGC/GOMEMLIMIT/GOMAXPROCS, `cd` 후 `golangci-lint run --timeout 10m -c "$(git rev-parse --show-toplevel)/.golangci-errorlint.yml" --new-from-merge-base=origin/main`).
   - `lint-config-check`: 저장소 루트에서 `golangci-lint config verify -c .golangci-errorlint.yml`, fetch 단계, mawk 설치, `bash -n scripts/check-structure-conventions.sh`, `--self-test`, 그리고 기존 `Lint one service` 뒤에 bin-common-handler 의 errorlint 단계.
   - 신규 job `check-structure-conventions`(jobs 의 `check-test-conventions` 정의(2301행) 바로 뒤, workflow 는 929행 `check-test-conventions` workflow 바로 뒤, cimg/base:2024.02, small, checkout, mawk 설치, 본 검사, `--self-test`)와 workflow(무조건 실행) 추가.
5. `.circleci/config.yml` path-filtering: `\.golangci-errorlint\.yml` 과 `\.circleci/config_work\.yml` 두 행을 `run-lint-config-check true` 로 추가.
6. 문서: error-handling.md 4.2 에 `%w` 해석 한 줄, models.md 6.1 에 설계 3.3 의 한 줄, database.md 7.1/7.7 에 R2b 해석 한 줄(대리 지표, raw SQL 미탐지 한계)과 R2b 예외 표(3 경로).
7. 두 계획/설계 문서 외의 서비스 소스는 수정하지 않는다.

## 3. self-test 사례 목록 (구현 시작 시 먼저 작성, 구현 전에 모두 실패하는 것을 확인)

R2a: 비listenhandler 의 DTO import 위반 / 자기 listenhandler 통과 / 타 서비스 DTO 를 listenhandler 가 import 통과 / requesthandler 통과 / `listenhandlerx` 위반 / `requesthandler2` 위반 / DTO 하위 패키지 위반 / 신규 서비스 디렉터리 위반 / 한 줄, 블록, 별칭, `_`, 한 줄 괄호 import / 주석 속 경로 무시 / 문자열 속 경로 무시 / 백틱 import rc=1.
R2b: 허용 밖 squirrel 위반 / dbhandler, databasehandler, analysisdbhandler 통과 / `dbhandlerx` 위반 / 허용 경로 안 테스트 파일 통과 / 허용 밖 테스트 파일 위반 / database.md 표에서 경로 삭제 시 실패.
R3: 신규 수동 구조체(새 파일, 기존 파일의 새 구조체) 위반 / Identity 임베딩 통과 / ID 전용, `filters.go`, WebhookMessage, DTO 통과 / 허용된 구조체의 태그 수정, 줄 이동, 파일 rename 통과 / 임베딩 전환 + 새 수동 정의 상쇄 위반 / stale 허용 항목 실패 / 허용 목록 읽기 실패, 중복 키, 빈 목록 실패 / 부분 일치(`rag:RagX`) 비허용 / 한 줄 struct 패턴 포함 gate 실패 / `type ( )` 그룹, 중첩 struct, 익명 struct gate 실패 / 주석 속 가짜 `type X struct {` 무시 후 실제 구조체 판정 / `"a/*b"` 뒤의 구조체, 태그 속 `//` 탐지 / 리터럴 `'{'` 와 태그 속 `{` 로 깊이 오판 없음 / 닫히지 않은 `/*`, 문자열 gate 실패 / 원문 줄 수와 귀속 줄 수 불일치 gate 실패 / 별칭 UUID 타입, 포인터/슬라이스, `db:"id,omitempty"` 탐지 / 필드 병기 미탐지(문서화된 한계 고정) / 한 줄 `struct{}` 정상 / 스캐너 고장 주입 rc=1.
R2 import 블록 내부 `/* */` 처리 / 닫히지 않은 import 블록 gate 실패 / 본문 raw string 안의 `import "x"` 줄 무시 / R2b 스캐너 고장 주입 rc=1 / 블록이 열린 상태에서 또 `type ... struct {` 귀속 실패 / 제네릭 타입 선언 gate 실패 / 허용 목록 CR, 탭 구분 인식 / 허용 목록의 사유 없는 줄 허용 / 허용 목록 안 구조체 안의 중첩 struct gate 실패 / `X, ID uuid.UUID`(마지막 요소) 탐지 / 쉼표 없는 `db:"id"` 탐지 / 닫히지 않은 raw string, rune gate 실패 / 한 줄 괄호 별칭 import / import 밖 raw string 태그는 백틱 rc=1 에 걸리지 않음 / 한 줄 `struct{}` 정상.
공통: 열거된 파일 읽기 실패 rc=1, 열거 0건 rc=1, `git ls-files` 실패 rc=1, awk 변형 누락 rc=1, 위반과 gate 실패 동시 보고.
전부 awk 와 mawk 양쪽에서 실행한다. 스캐너 고장 주입은 awk 바이너리를 항상 실패하는 스텁으로 바꿔 rc=1 임을 본다.

## 4. 검증 절차 (설계 6장 대응)

1. 로컬 golangci-lint 2.13.2 로 설계 6.1 전 항목(신규 파일 위반 file:line 정확성, 기존 위반 파일 미변경 시 통과, 기존 위반 줄 수정 시 보고, 기준 ref 삭제 시 rc=1, 사유 있는 `//nolint:errorlint // 사유` 제외)과 6.6(서비스 디렉터리에서 4.1 명령의 설정 로드와 main 동일 커밋 0건, 기존 `golangci-lint run` 이 변경 없이 비변경 줄의 `unused` 를 계속 보고함)을 합성 저장소와 bin-tag-manager 에서 실행. 2.14.0 은 바이너리를 임시 디렉터리에 내려받아(체크섬 검증) 재확인한다. 내려받지 못하면 미확인으로 보고한다.
2. 실제 저장소: 스크립트 rc=0, 발견 구조체 집합이 허용 목록 11개와 일치, 허용 import 184줄 집계(자기 listenhandler 108, requesthandler 76).
3. 소요 시간 측정(`time`, 기존 Linting 단계 대비 errorlint 단계 추가 소요 시간 포함, self-test 전체 시간 포함, 로컬 값이며 small 소요시간은 CI 로그로 확인), bin-api-manager 의 errorlint 를 `systemd-run --user -p MemoryMax=2G` 가 쓸 수 없으면 `docker run --memory=2g` 로 대체 시도, 둘 다 불가하면 미확인으로 보고.
4. CI: `circleci` CLI 가 없으므로 공식 바이너리를 임시 디렉터리에 내려받아 `circleci config validate/process` 를 시도하고, 불가하면 ruby YAML 파싱 + `.circleci/scripts/check-auto-deploy-config.rb` 로 대체하고 미확인으로 보고한다. path-filtering 정규식은 스크립트로 두 파일명이 `run-lint-config-check` 에 매핑됨을 확인한다. 같은 패턴 `\.circleci/config_work\.yml` 이 `run-shell-tests` 와 `run-lint-config-check` 두 행에 걸리므로, CircleCI path-filtering 은 매핑 행마다 독립 평가하여 일치하는 모든 파라미터를 켠다는 공식 문서 근거를 확인하고(문서 확인 불가하면 미확인으로 보고) 이 PR 의 CI 에서 `lint-config-check` workflow 가 실제로 실행되는지 관찰한다.
5. 기타: `scripts/check-docs.sh`, `scripts/check-service-docs.sh`, `scripts/check-test-conventions.sh`, `ruby .circleci/scripts/check-auto-deploy-config.rb`, error-handling.md 4.3 직접 비교 6곳 위치 기록, gate 메시지 인용 절 grep.
   처리된 설정에서 이 PR이 배포 workflow 를 새로 켜지 않음(5장 주장 근거)과 gate job 의 mawk 설치 단계 렌더링도 확인한다.
6. 브랜치 push 후 CircleCI 결과(lint, test, build) 확인. 이 PR 은 `.circleci` 를 바꾸므로 build-approval 은 수동 승인하지 않고 둔다. 승인 없이 수집 가능한 workflow(`check-structure-conventions`, `lint-config-check`, `check-test-conventions`, `shell-tests`)만 확인한다.

## 4a. 머지 전 검증 확인 항목 (완료 요약에 기재)

머지 전 검증에서 확인할 항목: required status check 에 `check-structure-conventions` 하나가 등록되어 있는지(저장소 설정), `lint-config-check` 는 등록하지 않음, processed config 에서 배포 workflow 가 새로 켜지지 않음.

## 5. 변경 공지 항목 (설계 7장 + 구현 차이)

완료 요약에 번호로 열거하고 담당자에게 공지한다: (1) R1 해석, (2) R3 구현 방식 변경(허용 목록, stale 실패)과 R2a/R2b 가 신규 라인 한정이 아닌 전체 트리 검사이며 baseline 이 없음, (3) 문서 명문화 추가, (4) CI 트리거 확장, (5) 초기 제안 대비 축소(depguard 대체, forbidigo 제외, DB 경계 대리 지표), (6) R1 신규 테스트 코드 적용과 R2 테스트 파일 검사, (7) R2a 교차 서비스 listenhandler import 미금지, (8) R3 미탐지 형태 전부, (9) R1 은 go-test* 3개 command 사용 서비스에만 적용, (10) 4.3 sentinel 충돌 가능성과 직접 비교 6곳 위치, (11) lint-config-check 비차단 한계, (12) required status check 등록 확인 항목(저장소 설정, 등록 대상은 `check-structure-conventions` 하나), (13) 머지 후 첫 서비스 PR errorlint 관찰(후속 확인 담당), (14) 보류 항목: import 순서, 일반 서비스 간 import 금지.

구현 중 설계와 달라진 점이 있으면 완료 요약에 목록으로 적는다. 설계와 충돌하거나 설계에 없는 사안은 임의 해석하지 않고 작업을 중단하고 담당자에게 확인한다.
