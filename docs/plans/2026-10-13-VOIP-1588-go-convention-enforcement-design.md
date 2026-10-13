# VOIP-1588 Go 컨벤션 CI 집행 설계

- 단계: 1/3 설계 (코드 변경 없음). 이어서 계획(plan), 구현이 같은 브랜치/PR에서 진행된다.
- 기준 커밋: origin/main b3c97aea8. 측정 환경: Linux, go 1.25.3, 로컬 golangci-lint 2.13.2 (CI 고정 버전은 2.14.0, 본 문서 4.4 참조).
- 표기: "실측"은 이 세션에서 실제로 실행한 결과, "추정"은 근사치, "미확인"은 실행하지 못한 항목이다. 수치에는 측정 정의를 붙인다.
- 용어: QA-1은 PR 머지 전 qa 담당 검증 카드, QA-2는 머지 후 배포 검증 카드(이 변경은 배포가 없어 QA-2가 없을 수 있음)이다.
- 승인 배경: 2026-10-13 대표님이 린트 규칙 상세화 권고안(1단계 경계 규칙, 2단계 신규 라인 한정 집행, 기존 위반 정리는 별도)을 승인했다. 승인된 권고안의 규칙 표에는 경계 규칙(서비스 간 경계, DB 접근은 dbhandler 전담), 모델 Identity 임베딩 규칙, 그리고 "에러 체인 보존, `%w` 사용 (error-handling 4.2): `fmt.Errorf`에서 err를 `%v/%s`로 출력, 신규 라인 한정 집행"이 포함되어 있었다. 이 문서는 구현 방식을 실측 결과에 맞게 조정한 것이며 조정 사유는 2장에 있다.

## 1. 문제와 목표

docs/conventions 17개 문서가 규칙을 정의하지만 기계적으로 집행되는 것은 gofmt, golangci-lint `standard` 세트, `scripts/check-test-conventions.sh`(테스트 규칙 3건)뿐이다. 드리프트의 원인은 문서 부재가 아니라 집행 부재다(2026-09-26 계획 문서 1장의 결론과 같다).

목표: 문서에 이미 명문화된 규칙 중 정적 분석으로 탐지 가능하고 오탐이 낮은 것을 신규/수정 코드에 한해 CI에서 차단한다. 문서에 없는 규칙은 새로 만들지 않는다(예외: 승인된 R1의 해석 명시와 R2b의 대리 지표. 근거와 보고 항목은 본 문서 3.1, 3.2, 3.4).

비목표: 기존 위반 일괄 수정, 전체 코드에 즉시 error 적용, 문서와 코드가 어긋나는 규칙의 일방 선택(본 문서 3.4), monorepo-javascript.

## 2. 승인안과 달라진 점

| 승인안 | 조정 | 근거 |
|---|---|---|
| depguard로 서비스 간 pkg/ import 금지 | 문서화된 DTO 경계 규칙을 스크립트 gate로 집행(R2a). 일반적인 "서비스 간 pkg import 금지"는 제외(본 문서 3.4) | 실측: depguard `pkg`는 prefix 일치이며 glob(`monorepo/bin-*/pkg`)을 쓰면 0건으로 통과한다(fail-open). 자기 서비스 허용을 표현하려면 서비스별 규칙이 필요하다(`bin-*`/`voip-*` 디렉터리 40개). 또한 일반 금지는 문서 근거가 없다: docs/conventions와 CLAUDE.md에 명시 규정이 없고, docs/architecture/service-dependency-graph.md는 Core→Leaf 방향 등만 서술한다 |
| 2단계 gate를 모두 스크립트로 | 에러 래핑은 golangci-lint `errorlint`를 별도 설정 파일(`.golangci-errorlint.yml`)과 별도 실행 단계로 분리하고 그 단계에만 `--new-from-merge-base` 적용 | 실측(합성 저장소): `--new-from-merge-base`는 전역 옵션이라 기존 `standard` 실행에 붙이면 변경하지 않은 줄에 생기는 이슈(예: 마지막 호출부 삭제로 `unused`가 된 함수)가 `0 issues`로 사라진다. 그래서 기존 전체 실행은 그대로 두고 errorlint만 분리한다. 분리 실행의 동작: 기준 커밋의 위반은 제외하고 브랜치가 추가한 위반만 file:line으로 보고, detached HEAD 동작, 기준 ref 없음/shallow clone/무관 히스토리 ref는 전체 위반을 보고하며 rc=1(fail-open 아님), 순수 rename은 보고하지 않음, main과 같은 커밋이면 0건 |
| forbidigo로 logrus.Fatal/fmt.Print 금지 | 제외 | 문서에 금지 규정이 없다. 실측(비테스트, `.Fatal/.Fatalf/.Fatalln/.Panic/.Panicf(` 패턴): 80건이며 대부분 cmd/main.go와 internal/config의 부트스트랩이다. 정당 사용이 다수라 규칙화 근거가 없다 |
| goimports/gci import 5그룹 | 제외, 대표님 결정 항목 | 실측: gci를 imports.md 순서로 돌리면 bin-tag-manager 10/50, bin-agent-manager 23/79, bin-call-manager 115/366 파일이 차이. 실제 관행은 "표준, 서비스 로컬, common/cross, 서드파티" 순이어서 문서와 코드가 불일치 |
| gate 2개(경계, 모델)를 별도 job | 스크립트 1개, job 1개로 통합. R2와 R3 모두 전체 트리 검사(R3는 구조체 단위 + 허용 목록) | 둘 다 같은 구조의 가벼운 검사이며 문서 반영이 중복된다. 승인안의 "신규 라인 한정"은 R3에서 "전체 트리 + 허용 목록(기존 11개 구조체 고정) + stale 항목 실패(ratchet)"로 구현 방식이 바뀌었다(신규 구조체는 막고 기존 구조체는 건드릴 수 있다는 승인 취지는 동일). R3를 git diff 기반(추가 줄, 개수 비교)으로 했을 때 rename 시 개수 0, 같은 파일 안 상쇄 우회, ID 전용 모델 오탐, merge-base 의존이 리뷰에서 반복 확인되어 전체 트리 + 구조체 키 집합 판정으로 바꿨다 |

## 3. 규칙 카탈로그

### 3.1 R1 에러 체인 보존 (error-handling.md 4.2)

- 도구: golangci-lint `errorlint`. 설정은 루트의 별도 파일 `.golangci-errorlint.yml`에 둔다. 골격: `version: "2"`, `linters.default: none`, `linters.enable: [errorlint]`, errorlint 설정 `errorf: true`/`comparison: false`/`asserts: false`, `issues.max-same-issues: 0`(기본 3이면 위반이 많은 PR에서 일부만 출력됨, 실측), 그리고 `.golangci.yml`과 같은 `linters.exclusions`(`generated: lax`, 경로 `(^|/)vendor(/|$)`). 기존 `.golangci.yml`에는 errorlint를 넣지 않는다(넣으면 기존 전체 실행이 기존 위반으로 실패하고, 플래그를 붙이면 `standard`가 약해진다). error-handling.md 4.2는 "`fmt.Errorf` with `%w` 또는 `errors.Wrap`/`errors.Wrapf`"를 사용하라고 규정하며 `%v`/`%s`로 error를 감싸는 것을 금지한다는 문구는 따로 없다. 해석: `fmt.Errorf`로 error를 포함할 때는 `%w`를 쓴다는 것이 4.2의 지시이며, 이 PR은 그 해석을 error-handling.md 4.2에 한 줄로 명시한다("`fmt.Errorf`에 error를 넘길 때는 `%v`/`%s`가 아니라 `%w`"). R1 집행은 위 승인안에 포함되어 있었고, 4.2 해석 한 줄 추가는 그 집행을 문서가 뒷받침하게 하는 해석 확정이다. 4.2가 `%v` 금지를 직접 쓰지 않았다는 점은 숨기지 않고 대표님 보고와 QA 보고의 해석 항목으로 적는다(본 문서 3.4). 계층 경계에서 원인 노출을 일부러 끊어야 하는 정당한 경우는 사유를 붙인 `//nolint:errorlint // <사유>`로 우회한다. error-handling.md 4.3 sentinel 비교는 범위 밖이다.
- 범위 제한: 별도 실행 단계에서만 `golangci-lint run -c .golangci-errorlint.yml --new-from-merge-base=origin/main`. 기존 `golangci-lint run --timeout 10m`(standard 전체 검사)은 변경하지 않는다. 기존 위반(추정 1,589~1,676건: `git ls-files '*.go'`에서 `fmt.Errorf(` 한 줄에 `%[vs]`와 `err`가 있는 grep, 테스트 포함, 패턴에 따라 다름. 정확한 값은 구현 단계 첫 실행에서 확정)은 보고되지 않는다.
- 테스트 파일: error-handling.md 4.2에 테스트 예외가 없고 `.golangci.yml`은 errcheck만 테스트에서 제외하므로, 신규 테스트 코드에도 적용한다(의도).
- 한계(실측 포함): 기존 위반이 있는 줄을 수정하면 그 줄은 신규로 취급된다(의도된 동작). errorlint는 `fmt.Errorf`의 verb와 error 인자만 보므로 `err.Error()` 문자열 결합은 탐지하지 못한다. 호출이 여러 줄이고 포맷 문자열 줄만 바뀌면 이슈가 인자 줄에 보고되어 비변경 줄로 걸러져 탐지되지 않는다(실측, 저장소의 여러 줄 `fmt.Errorf(` 호출은 매우 드묾). `//nolint:errorlint`로 우회할 수 있고(`nolintlint` 비활성) `Code generated ... DO NOT EDIT` 헤더 파일은 `generated: lax`로 제외되고, 이름 있는 빌드 태그 파일(`tools`, `builder_eval` 2개)은 기본 분석 범위 밖이다. 의도적 우회는 리뷰가 잡는 영역으로 둔다.

### 3.2 R2 계층/위치 경계 (신규 스크립트 `scripts/check-structure-conventions.sh`, R2와 3.3의 R3가 함께 쓰는 전체 트리 검사)

(R2는 R2a와 R2b의 총칭이다.) 현재 위반이 0이므로 baseline이 필요 없다.

파일 열거: `git ls-files '*.go'`(추적 파일만, vendor 제외). `find`/`grep -r`은 `.worktrees/`와 vendor를 포함하므로 쓰지 않는다. 파일의 소유 서비스는 경로의 첫 요소로 판정하고, import 경로의 서비스는 `monorepo/` 다음 첫 요소를 디렉터리명과 비교한다(서비스명 정규식을 쓰지 않는다. `bin-s3-manager` 같은 이름도 같은 방식으로 처리). import 추출은 한 줄 `import "..."`, 블록 import, 별칭/`_`/`.` import를 모두 처리하고 주석 줄은 제외한다.

- R2a DTO 경계 (CLAUDE.md "transport DTO packages" 절 241~258행(규정 241~252행, 예외 256행) "Exception — the RPC client"): `monorepo/<서비스>/pkg/listenhandler/models/(request|response)` (하위 패키지 포함: 경로가 `(/|")`로 이어지는 경우) 를 import할 수 있는 파일은 허용 표(3.2.1)의 둘뿐이다. 실측(전체 추적 Go 파일, 테스트 포함): 허용 import 184줄(자기 listenhandler 108, requesthandler 76), 위반 0.
- R2b squirrel 위치 (database.md 7.1 Mandatory squirrel, 7.7 DB 작업 위치 `pkg/dbhandler/`): `github.com/Masterminds/squirrel` import는 허용 표의 디렉터리에서만 가능하다. 테스트 파일도 R2a와 같이 검사 대상이고, 허용 경로 안의 테스트(`databasehandler/main_test.go`)는 허용된다. 7.7이 직접 금지하는 것은 "DB 작업의 위치"이고 squirrel import 위치는 그 대리 지표다. 한계: squirrel을 쓰지 않는 raw SQL 호출은 탐지하지 못한다(예: `bin-agent-manager/cmd/agent-control/normalize_addresses.go`의 `QueryRowContext`). 문서에 해석 한 줄을 추가하되 "7.7 집행"이라고 쓰지 않고 이 한계를 함께 적는다. 실측: 허용 경로 밖 비테스트 import 0건(허용 경로 안은 `databasehandler/main.go`, `analysisdbhandler/analysis.go` 2개 파일).
- `database/sql` import 제한은 제외한다. dbhandler 밖의 50개 파일 대부분이 `cmd/*/main.go`의 DB 연결 코드이고, 문서가 금지하는 것은 쿼리 위치다.

3.2.1 허용 예외 (R2b 경로는 스크립트 상수와 database.md 표를 매 실행 대조하고, R2a 두 행의 근거는 CLAUDE.md이다)

| 규칙 | 허용 | 근거 |
|---|---|---|
| R2a | 파일 `<임의 서비스>/pkg/listenhandler/**` 가 임의 서비스의 `monorepo/<서비스>/pkg/listenhandler/models/(request\|response)` 를 import | CLAUDE.md 241행 "may only be referenced inside pkg/listenhandler"와 251행 "any non-listenhandler". 문서는 자기 서비스로 한정하지 않으므로 교차 서비스 listenhandler import는 집행하지 않는다(문서에 없는 금지를 새로 만들지 않음) |
| R2a | 파일 `bin-common-handler/pkg/requesthandler/**` 가 모든 서비스의 위 DTO를 import | CLAUDE.md 256행 "Exception — the RPC client". 실측 76줄(request 72, response 4), 72개 파일, 테스트/mock 없음 |
| R2b | 파일 경로가 `bin-<S>/pkg/dbhandler/**`, `bin-common-handler/pkg/databasehandler/**`, `bin-timeline-manager/pkg/analysisdbhandler/**` 인 경우 | `bin-*/pkg/dbhandler/`는 database.md 7.7이 정한 위치. `databasehandler`(공용 DB 헬퍼), `analysisdbhandler`는 문서 미기재 기존 예외로 현재 위반이 아니며 database.md 해석에 "기존 예외"로 명시한다. 실측: 저장소의 모든 dbhandler 디렉터리는 `bin-*/pkg/dbhandler/` 한 형태이며 나머지 2개가 위 2개 파일 |

정확 일치 규칙: 경로 요소 단위로 비교한다(`dbhandlerx`, `requesthandler2` 같은 이름은 허용되지 않는다).

### 3.3 R3 모델 Identity 임베딩 (models.md 6.1, 같은 스크립트, 전체 트리 + 구조체 단위 + 허용 목록)

- 문서 규정: "ID와 고객 소유를 가진 모델은 `commonidentity.Identity`를 임베딩한다". 위반 예시(WRONG)는 `db:"id,uuid"` 태그가 달린 영속 모델이다(탐지 패턴은 `db:"id[,\"]"` 형태로 쉼표 없는 태그도 포함한다).
- 실측 신호(승인 근거 보강): 최근 180일에 새로 추가된 persisted 모델 파일 중 수동 정의 4개(outboundconfig, casenote, kase, providercall) 대비 Identity 임베딩 16~18개로, 드리프트가 지금도 신규 모델에서 발생한다(측정: `git log origin/main --since="180 days ago" --diff-filter=A --name-only`로 추가된 파일 중 정규식 `^bin-[^/]+/models/`에 맞는 비테스트 `.go` 파일을 `origin/main` 내용으로 판정,  신규 비테스트 파일 102개 중. 임베딩 수는 별칭 포함 여부에 따라 16~18개).
- 스캔 범위: `git ls-files` 결과 중 정규식 `^bin-[^/]+/models/`에 맞는 비테스트 Go 파일(441개). 정규식이 `pkg/listenhandler/models/**`와 `bin-openapi-manager/gens/models`를 원래 포함하지 않는다.
- 탐지: 위 범위의 Go 파일에서, 최상위 `type X struct {` 블록 하나 안에 수동 정의 ID 줄(`ID <패키지>.UUID` 타입, `db:"id,..."` 태그)과 수동 정의 CustomerID 줄(`CustomerID <패키지>.UUID` 타입, `db:"customer_id,..."` 태그)이 모두 있으면 "수동 정의 구조체"로 판정한다. 이 구조체의 키는 `<디렉터리>:<타입명>`(파일명 제외)이다. 키가 허용 목록에 없으면 위반이다.
- 허용 목록: `scripts/structure-conventions-allow.txt`(키와 사유 한 줄). 초기 항목은 실측한 11개 구조체로 고정한다: 기존 위반 10개(`bin-call-manager/models/outboundconfig:OutboundConfig`, `bin-contact-manager/models/casenote:CaseNote`, `.../kase:Case`, `bin-customer-manager/models/accesskey:Accesskey`, `bin-rag-manager/models/chunk:Chunk`, `.../document:Document`, `.../rag:Rag`, `bin-route-manager/models/providercall:ProviderCall`, `.../route:Route`, `bin-storage-manager/models/account:Account`)와 정의처 `bin-common-handler/models/identity:Identity`. 허용 목록에 있으나 더 이상 수동 정의가 아닌 항목(stale)은 실패로 처리해, 임베딩으로 전환한 PR이 항목을 삭제하도록 한다(ratchet).
- 문서 반영: models.md 6.1에 한 줄을 추가한다("신규 모델은 `commonidentity.Identity`를 임베딩한다. 기존 수동 정의 구조체는 `scripts/structure-conventions-allow.txt`에 고정되어 있으며 임베딩으로 전환할 때 항목을 삭제한다").
- 이 설계가 해소하는 것: 신규 모델은 허용 목록 변경 없이는 막힌다. 기존 모델의 태그 수정, 줄 이동, 파일 rename은 키가 같아 통과한다. 개수 상쇄 우회는 집합 판정이라 불가능하다. ID만 있는 모델은 구조체 안에 두 줄이 모두 있지 않아 대상이 아니다. 패키지 이동처럼 키가 바뀌는 경우는 허용 목록을 같은 PR에서 고치며, 그 diff가 리뷰에서 보이는 정당한 우회 수단이다. git diff와 merge-base에 의존하지 않으므로 fetch 단계가 필요 없다.
- fail-closed 불변식: 범위 안 파일에서 위 두 패턴(타입은 `[*\[\]]*\w+\.UUID`로 포인터/슬라이스 포함, 태그는 `db:"id[,"]`/`db:"customer_id[,"]`로 별칭 import와 태그 변형을 포함)에 맞는 모든 줄(정의: 한 줄에 필드명 `ID`와 UUID 타입과 `db:"id[,\"]"` 태그가 함께 있거나, 필드명 `CustomerID`와 UUID 타입과 `db:"customer_id[,\"]"` 태그가 함께 있는 줄. 필드명은 `ID`, `CustomerID`로 고정)은 인식된 최상위 `type X struct {` 블록에 귀속되어야 한다. 귀속되지 않는 줄이 하나라도 있으면(예: `type ( ... )` 그룹 선언 안의 구조체, 익명/중첩 구조체, 제네릭 타입 선언 등 파서가 해석하지 못하는 형태) 위반이 아니라 gate 실패(rc=1)로 처리하고 파일과 줄을 보고한다. 판정 전에 문자열(`"..."`), raw string(`` `...` ``), rune(`'...'`) 리터럴을 인식하는 스캐너로 `/* */`와 `//` 주석을 제거한다(리터럴 안의 `/*`, `//`, 그리고 struct 태그(raw string) 안의 `//`는 주석이 아니다). 닫히지 않은 `/*`나 닫히지 않은 리터럴은 gate 실패다. 원문에서 위 정의에 맞는 줄의 수를 스캐너와 독립된 단순 패스(`//`로 시작하는 줄만 제외하는 한 줄 grep)로 세어, 구조체에 귀속된 줄의 수와 다르면 gate 실패다(스캐너 자체가 구조체를 주석으로 오판해도 양쪽이 같이 0이 되지 않게 하는 교차 검증). `/* */` 블록 안에 있는 정의 줄은 단순 패스에는 세어지고 스캐너에는 제거되므로 보수적으로 gate 실패가 된다. 중괄호 깊이를 추적하되(리터럴 안의 `{`, `}`는 깊이 계산에서 제외한다: `'{'`나 태그 안의 `a{`로 정상 모델이 오탐되지 않게) 최상위 블록의 깊이 1 필드만 귀속으로 인정한다(깊이 2 이상의 중첩/익명 구조체 안의 패턴 줄은 귀속 실패). 블록이 열린 상태에서 또 다른 `type ... struct {`가 나와도 귀속 실패다. 이 규칙으로 판정 불가 형태가 조용히 통과하지 않는다. 현재 저장소에는 귀속되지 않는 줄이 0건이고(실측, 완화된 패턴으로도 수동 정의 구조체는 동일한 11개), `string` 타입 ID(`bridge`, `channel`의 `db:"id"`)는 패턴에 걸리지 않는다.
- 허용 목록 형식: 줄마다 `<키>` 뒤에 공백으로 구분한 사유를 쓴다. `#`로 시작하는 줄과 빈 줄은 무시한다. 키는 첫 공백 구분 필드를 전체 일치로 비교한다(부분 일치 금지: `rag:Rag`가 `rag:RagX`나 사유 문구에 걸리지 않는다). 중복 키, 파일이 없거나 읽기 실패, 유효 항목이 0개인 경우는 모두 gate 실패다.
- 오탐 방지: `db` 태그가 없는 필터(`filters.go`, `field.go`의 `filter:"id"`), 인증/이벤트 DTO, 수동 WebhookMessage는 대상이 아니다.
- 실측(정의: `git ls-files`, 비테스트 `bin-*/models/**/*.go`, 구조체 블록 단위 매칭): 수동 정의 구조체 11개(파일당 하나), 구조체에 ID만 있는 경우 5개와 CustomerID만 있는 경우 2개(판정 대상 아님). Identity 임베딩 파일 수(줄 시작이 `commonidentity.Identity`인 파일 96개)는 판정에 쓰지 않는다.
- 한계: gate job 이미지(`cimg/base`)에는 Go가 없어 `go/ast` 대안은 이 PR의 범위 밖이며, 스캐너의 코드 크기 상한은 계획 단계에서 정한다. import 경로는 큰따옴표 형태만 추출하되, R2 대상 파일의 import 구간에 백틱 경로가 있으면 rc=1로 처리한다(현재 0건, 6장 2번 음성 시험에 포함). 이 규칙은 3.2의 R2에도 적용된다. 허용 목록의 공백 구분은 공백, 탭, CR을 포함한다. 구현은 mawk 호환을 위해 `\b`, `\w` 대신 `(^|[^[:alnum:]_])ID[[:space:]]`, `[[:alnum:]_]+\.UUID` 같은 POSIX ERE를 쓰고 6장 7번의 mawk `--self-test`에 포함한다. `ID` 또는 `CustomerID`가 필드명 목록의 마지막 요소가 아닌 필드 병기(`ID, X uuid.UUID`, `X, ID, Y uuid.UUID`, `CustomerID, X uuid.UUID`)는 패턴(`ID`/`CustomerID` 바로 뒤에 공백)에 맞지 않아 조용히 통과하는 미탐지 형태로 확정한다(현재 0건. mawk 실측: 마지막 요소인 `X, ID uuid.UUID`, `X, CustomerID uuid.UUID`만 탐지된다). 필드명 변형(`Id`, `CustomerId`)과 해석된 문자열 태그(`"db:\"id,uuid\""`)는 탐지하지 못하며 현재 0건이다. raw string 안의 코드 템플릿은 리터럴로 처리되어 구조체로 인식되지 않고, 그 안의 정의 줄은 독립 grep 개수와 귀속 개수의 불일치로 fail-closed 실패한다. 스캐너 코드 크기와 `--self-test` 범위의 상한은 계획 단계에서 정한다. 로컬 별칭 타입(`type ID = uuid.UUID`)과 dot-import(`ID UUID`)는 패턴에 걸리지 않으며 현재 범위 안에 해당 형태가 없다(`astaor` 등의 `ID *string`은 UUID 타입이 아니라 대상이 아니다). 범위 밖의 정의(예: `bin-contact-manager/pkg/dbhandler/address.go`, `address_ownership.go`)는 탐지하지 못한다. 태그가 `ID`와 다른 줄로 나뉜 형태는 패턴에 걸리지 않아 불변식으로도 잡히지 않으며 현재 0건이다. ID/CustomerID 정의 패턴을 포함한 한 줄 구조체는 fail-closed(5장)이고 필드 병기는 위에서 확정한 미탐지 형태다. 기존 위반 모델을 Identity 임베딩으로 바꾸는 리팩터링은 이 카드의 범위가 아니다.

### 3.4 제외 항목과 대표님 결정 항목

| 항목 | 상태 |
|---|---|
| R1 해석(error-handling 4.2에 `%w` 지시를 명시) | 승인안에 포함된 항목이며 반영 후 대표님 보고 항목으로 올린다. 불승인 시 error-handling.md 4.2의 한 줄과 errorlint 단계를 되돌린다(영향 CI 정의와 문서뿐) |
| R3 구현 방식 변경 | 신규 라인 한정에서 전체 트리 + 허용 목록(11개) + stale 시 실패로 조정. stale 실패가 과하다고 판단하시면 경고로 낮출 수 있다(작은 변경). 대표님 보고 항목 |
| 문서 명문화 추가(models.md 6.1 해석, database.md 7.1/7.7 해석과 `databasehandler`/`analysisdbhandler` 기존 예외) | 문서에 없던 기존 예외를 표로 굳히는 것이므로 대표님 보고 항목 |
| CI 트리거 확장(`\.circleci/config_work\.yml`과 `\.golangci-errorlint\.yml` 변경이 `run-lint-config-check`를 켬) | 승인안에 없던 부수 변경이다. 후속 PR이 검증 없이 머지되는 것을 막기 위한 작은 변경이며 매핑 행 두 줄로 되돌릴 수 있다. 대표님 보고 항목 |
| import 5그룹 순서 | 문서와 관행 불일치. 문서를 관행에 맞출지 코드를 문서에 맞출지 결정 필요. 이 카드는 건드리지 않는다 |
| 일반적인 서비스 간 pkg/cmd/internal import 금지 | 현재 위반 0이나 문서에 명시 규정이 없다(`bin-common-handler`를 모든 서비스가 import하는 것은 service-dependency-graph.md가 허용). 규칙을 새로 만드는 일이므로 도입하려면 문서화 결정이 선행되어야 한다 |
| logrus.Fatal/fmt.Print 금지 | 문서에 금지 규정 없음 |
| 에러 sentinel 비교(error-handling.md 4.3), 로깅 레벨 | 정적 분석 불가 또는 근거 부족 |

## 4. CI 변경 (.circleci/config_work.yml, .circleci/config.yml)

이 PR의 전체 변경 파일: `.golangci-errorlint.yml`(신규), `scripts/check-structure-conventions.sh`(신규), `scripts/structure-conventions-allow.txt`(신규, R3 허용 목록), `.circleci/config_work.yml`, `.circleci/config.yml`(path-filtering mapping 행 추가), `docs/conventions/error-handling.md`, `models.md`, `database.md`(해석과 예외 표 반영), 이 설계 문서와 계획 문서.

### 4.1 서비스별 lint 단계
`go-test`(2421행), `go-test-api-manager`(2491행), `go-test-pipecat-manager`(2562행) 3개 command의 `when: enable-lint` 블록 안에서 기존 `Linting` 단계는 그대로 두고, 그 뒤에 `Lint error wrapping on new lines` 단계를 추가한다(같은 GOGC/GOMEMLIMIT/GOMAXPROCS 환경). 명령은 서비스 디렉터리로 `cd`한 뒤 실행하므로 설정 경로를 cwd 독립적으로 쓴다: `golangci-lint run --timeout 10m -c "$(git rev-parse --show-toplevel)/.golangci-errorlint.yml" --new-from-merge-base=origin/main`(실측: 서비스 디렉터리에서 상대경로 `-c .golangci-errorlint.yml`은 `no such file or directory`로 실패하고 `-c ../.golangci-errorlint.yml`은 동작). 이 단계 앞에 origin/main 확보 단계를 같은 `when` 블록 안에 둔다(fetch 단계의 4곳 복제를 공용 command로 뺄지는 계획 단계에서 결정). `check-test-conventions` job의 "Fetch base branch" 절차(`--depth` 금지, 필요 시에만 deepen)를 재사용한다. 실측으로 shallow clone에서 이 단계가 없으면 merge-base 해석이 실패해 전체 위반이 보고되므로 필수다.

### 4.2 lint-config-check job (정의 2336행, `golangci-lint run` 줄 2370행)
기존 `bin-common-handler` 전체 린트는 변경하지 않아 이 job의 목적(설정 변경을 전체 코드에서 검증)이 유지된다. 추가로 (a) 저장소 루트 cwd에서 `golangci-lint config verify -c .golangci-errorlint.yml`, (b) fetch 단계와, 기존 `Lint one service against the config` 단계의 `go mod vendor` 이후 `bin-common-handler` 디렉터리에서 본 문서 4.1과 같은 명령의 errorlint 별도 실행 단계를 넣는다. 경로 필터: `.circleci/config.yml`의 path-filtering mapping(66행 `.golangci.yml`, 67행 `scripts/.*`)은 `\.golangci\.yml`만 `run-lint-config-check`에 연결하고 `.golangci-errorlint.yml`은 정규식에 맞지 않으므로(그대로 두면 이 파일만 바꾸는 PR은 검증 없이 머지된다) `\.golangci-errorlint\.yml  run-lint-config-check true` 행을 추가한다. 같은 이유로 errorlint 단계가 들어가는 `\.circleci/config_work\.yml` 변경은 현재 `run-shell-tests`만 켜므로 `\.circleci/config_work\.yml  run-lint-config-check true` 행도 추가한다(이 단계의 플래그/경로를 고치는 후속 PR이 검증 없이 머지되는 것을 막는다). 따라서 이 PR의 변경 대상에 `.circleci/config.yml`이 포함된다. 이 PR의 CI에서는 `scripts/` 변경이 `run-lint-config-check`를 켜므로 R1의 설정 로드, 플래그 수용, fetch 경로가 머지 전에 한 번 실행된다. 이 PR은 Go diff가 없어 `bin-common-handler`에서 errorlint가 항상 0건으로 통과하므로, 탐지 동작 자체는 6장 1번의 로컬 위반 주입이 검증한다. 반면 3개 서비스 command의 새 단계는 서비스 경로를 건드리는 PR에서만 실행되므로 첫 서비스 PR에서 처음 관찰된다(5장).
- 신규 스크립트에도 `bash -n`과 `--self-test` 실행 단계를 추가하며, `--self-test` 앞에 mawk 확보 단계(`sudo apt-get update && sudo apt-get install -y mawk`, `shell-tests`와 동일 방식)를 둔다. 이미지에 이미 있으면 확인만 하고 없으면 설치한다. `scripts/.*` 변경은 path-filtering(`.circleci/config.yml`)에서 이미 `run-lint-config-check`를 켠다.
- 같은 job의 "design doc snippet 일치" 단계는 2026-09-26 문서 전용이며 이 설계 문서는 해당하지 않는다(스크립트 본문을 문서에 복사하지 않는다).

### 4.3 신규 gate job
`check-structure-conventions`가 스크립트 본 검사와 함께 `--self-test`도 실행하고(필수 gate 자체가 fail-open이 되는 변경을 비차단인 `lint-config-check`에만 맡기지 않기 위함). `cimg/base`에는 Go도 mawk 보장도 없으므로 `shell-tests`와 같은 mawk apt 설치 단계를 둔다. 이 job을 `check-test-conventions`와 같은 위치와 구성(매 pipeline 무조건 실행, resource_class small. 전체 트리 검사라 origin/main fetch 단계는 필요 없다)으로 추가한다. `.circleci/scripts/check-auto-deploy-config.rb`(`-test-auto` 워크플로와 allowlist 검사, 배포 job이 있는 워크플로만 승인 단계 요구)와 충돌하지 않는지 구현 단계에서 `ruby .circleci/scripts/check-auto-deploy-config.rb`를 실행해 확인한다. 또한 신규 job이 GitHub required status check에 등록되어야 머지를 막으므로 이는 저장소 설정(대표님)이며, QA-1 카드 본문에 확인 항목으로 적는다. 실제 job 구성은 `circleci config process`(파라미터 ON)로 확인한다.

### 4.4 버전 차이
합성 실험은 로컬 2.13.2에서 수행했다. CI 고정 버전은 2.14.0이므로 `new-from-merge-base`와 errorlint 동작은 구현 단계에서 2.14.0으로 재확인한다(미확인).

## 5. 리소스와 위험

| 위험 | 대응 |
|---|---|
| errorlint가 small(1 vCPU/2GB)에서 메모리를 추가로 쓸 수 있음 | 미확인. 구현 단계에서 가장 큰 서비스(bin-api-manager, `git ls-files 'bin-api-manager/*.go'` 기준 497개 Go 파일)를 기존과 같은 cgroup 상한(MemoryMax=2G)에서 실행해 oom_kill 0을 확인한다. 초과 시 해당 규칙을 보류한다 |
| 기준 ref 없음 | 실측상 전체 위반을 보고하며 rc=1(fail-closed). 단 기존 위반이 0인 모듈에서는 rc=0이며 신규 위반이 숨지 않는다 |
| main 브랜치 pipeline | errorlint 별도 단계는 merge-base가 HEAD라 신규 issue 0으로 통과(실측). 기존 standard 전체 실행은 변경이 없어 main에서도 그대로 전체 검사한다 |
| 서비스 command의 새 단계가 이 PR의 CI에서 실행되지 않음 | 서비스 경로를 건드리지 않는 PR이라 3개 command는 실행되지 않는다. 동일한 단계가 lint-config-check에서 실행되어 설정, 플래그, fetch 경로는 검증되지만 서비스 job 자체는 머지 후 첫 서비스 PR에서 처음 관찰된다. 이 카드는 QA-2가 없을 수 있고 첫 서비스 PR은 이 카드 밖의 미래 PR이므로, 확인 담당은 오케스트레이터(default)로 명시한다: 머지 후 첫 서비스 PR의 CI 결과를 확인하고 이상이 있으면 대표님께 보고한다(별도 카드나 티켓은 만들지 않는다) |
| R1이 error-handling.md 4.3(sentinel 직접 비교)과 어긋남 | error-handling.md 4.3은 dbhandler sentinel의 직접 비교를 지시하고 wrapping이 있을 때만 `errors.Is`를 쓰라고 한다. 실측: dbhandler는 이미 `%w`로 314곳을 감싸고, 서비스 코드의 직접 비교(`err`/`selErr` `==`·`!=` `dbhandler.Err*`, 비테스트)는 6곳, `errors.Is(…dbhandler.Err…)`는 168곳이다. 충돌 경로는 "sentinel을 품은 err를 신규 `fmt.Errorf(...%w)`로 감싸고 상위 호출자가 4.3대로 `==` 비교"인 경우이며 현재 해당 코드는 없다(미래 위험, error-handling.md 4.3이 감싼 경우 `errors.Is`를 쓰라고 이미 지시). R1은 신규 코드의 `fmt.Errorf`에만 적용되고 기존 코드는 변경하지 않는다. 구현 단계에서 직접 비교 6곳의 위치를 PR 본문에 기록한다 |
| go-test command를 쓰지 않는 모듈 | 예: `bin-openapi-manager`(Go 파일 2개)는 CI에 go-test job이 없어 R1 범위 밖이다(기존 구조, 변경하지 않음) |
| 허용 예외 표가 문서와 스크립트에서 어긋남(6장 5번) | 스크립트가 허용 경로를 상수로 갖고, 매 실행(매 pipeline에서 도는 `check-structure-conventions` job)마다 database.md의 R2b 예외 표에 같은 경로 문자열이 모두 있는지 grep으로 대조해 없으면 실패한다. database.md만 고치는 PR도 이 job이 잡으므로 별도 path-filter 연결이 필요 없다 |
| 전체 트리 스캔 시간 | R2는 `git ls-files '*.go'` 4천여 파일을 매 pipeline 스캔한다. resource_class small에서 소요 시간을 구현 단계에 측정하고 기록한다 |
| gate가 required status check에 미등록 | 본 문서 4.3. 등록 전에는 gate가 실패해도 머지를 막지 못한다. 대표님 설정 항목(7장)이며 QA 카드 본문에 확인 항목으로 둔다(6장 9번). 반대로 조건부로 도는 `lint-config-check`를 등록하면 대부분의 PR이 pending에 걸리므로 등록하지 않는다. 그 결과 설정/스크립트 변경 검증(`lint-config-check`)은 비차단으로 남는 한계가 있고 이는 수용한다 |
| errorlint 단계가 서비스 lint 시간을 늘림 | 단계마다 `--timeout 10m`을 두고 6장 7번에서 가장 큰 서비스의 추가 소요 시간을 측정한다. 기존 Linting 단계와 같은 환경 변수를 쓴다 |
| 대규모 기계적 변경(일괄 치환, rename)에서 R1이 대량 실패 | 수정한 줄은 신규로 취급되므로(본 문서 3.1) 기존 위반 줄을 건드린 PR은 그 줄이 보고된다. 우회: 줄마다 사유를 붙인 `//nolint:errorlint // <사유>`. 6장 1번의 "기존 위반 줄 수정 시 보고"로 동작을 확인한다 |
| 신규 서비스 추가 시 R2 누락 | R2는 `git ls-files` 전체를 경로 요소 규칙으로 순회하므로 서비스별 규칙 추가가 필요 없다 |
| gate가 조용히 통과(fail-open) | 파일 열거(`git ls-files`) 실패나 열거 0건은 rc=1(`set -o pipefail`, 열거 결과를 파일로 받아 종료 코드 확인), 필수 awk 변형(awk, mawk)이 없으면 self-test는 건너뛰지 않고 rc=1, mawk 설치 단계를 gate job에 두고, `lint-config-check`(go_image)에도 mawk 존재 확인 또는 apt 설치 단계를 둔다(4.2)(`shell-tests`의 `apt-get install mawk` 단계와 같은 방식). ID/CustomerID 정의 패턴을 포함한 한 줄 구조체는 귀속 불가로 처리하며(스캐너가 블록으로 인식하지 않음) 독립 grep 개수와 귀속 개수 불일치로 fail-closed이다. 문서의 `[*\[\]]*`는 개념 표기이며 구현은 POSIX bracket `[][*]*`로 쓰고 주석으로 남긴다. 로컬 실행 시 `git add`하지 않은 새 파일은 `git ls-files`에 없어 검사되지 않는다(CI는 무관, 스크립트 주석에 명시). 한 줄 `struct{}`와, 패턴을 포함한 한 줄 구조체(``type X struct{ ID uuid.UUID `db:"id"` }``, gate 실패 사례)를 self-test에 넣는다. 한 줄 `struct{}`(예: `bin-ai-manager/models/engine_openai/main.go:3`)는 열고 닫음이 한 줄에서 끝나는 형태로 처리하고 self-test 사례에 넣는다. 스크립트는 `--self-test` 모드(위반 주입, 정상 입력, 예외 경계, awk 변형 mawk 포함, 스캐너 고장 주입 시 rc=1)를 내장하고 필수 gate job(`check-structure-conventions`)과 `lint-config-check` 양쪽에서 실행한다. 6장 2, 3, 4, 7, 8번. 상세는 계획 단계 |
| R3 오탐과 우회 | 3.3의 구조체 단위 판정으로 필터, DTO, WebhookMessage, ID 전용 모델은 대상이 아니고, 기존 위반 모델의 무관한 수정과 rename은 키가 같아 통과한다. 개수 상쇄는 집합 판정이라 불가능하다. 우회 수단은 허용 목록 편집(리뷰에서 보임)뿐이다. 알려진 한계는 `models/` 밖 정의, 파싱 불가 형태, 3.3 한계에 적은 조용히 통과하는 미탐지 형태다. 검증 계획 4번 |

되돌림: CI 플래그와 설정 항목, gate job 제거로 즉시 가능하다. 서비스 배포와 데이터에 영향이 없다(CI 정의와 스크립트만 변경). 머지가 배포를 일으키지 않는다는 점은 QA 검증 단계에서 processed config로 재확인한다.

## 6. 검증 계획 (구현 시점에 코드로 옮긴다)

1. R1: 위반 주입 시 file:line 정확히 보고(신규 파일), 기존 위반 파일을 건드리지 않으면 통과, 기존 위반 줄을 수정하면 보고, 기준 ref 삭제 시 rc=1, 사유를 붙인 `//nolint:errorlint // <사유>` 줄은 제외됨, 2.14.0 재확인.
2. R2a: 새 서비스 디렉터리(예: `bin-new-manager`)가 DTO를 import해도 규칙이 적용됨, 허용 표 두 경로에서 통과, 다른 서비스의 DTO를 listenhandler가 import해도 통과(CLAUDE.md가 금지하지 않음), 비listenhandler가 DTO를 import하면 위반, 이름만 비슷한 경로(`listenhandlerx`, `requesthandler2`)에서 위반, 한 줄/블록/별칭 import 형태, 주석 속 경로는 무시, 백틱 import는 rc=1, 스캐너 고장 주입 시 rc=1.
3. R2b: 허용 밖 squirrel import 위반, 허용 경로 통과, `dbhandlerx` 위반, 스캐너 고장 주입 시 rc=1.
4. R3: 허용 목록에 없는 새 수동 구조체(새 파일과 기존 파일의 새 구조체 둘 다)는 위반, 임베딩 시 통과, ID만 있는 구조체/`filters.go`/WebhookMessage/DTO는 통과, 허용 목록의 기존 위반 10개 중 하나의 태그 줄 수정/줄 이동/파일 rename은 통과, 같은 파일에서 한 구조체를 임베딩으로 바꾸고 다른 구조체를 새로 수동 정의하는 상쇄 시도는 위반, 허용 목록의 stale 항목은 실패, 허용 목록을 읽지 못하면 rc=1, 스캐너 고장 주입 시 rc=1, 실제 저장소에서 위반 0이고 허용 목록과 구조체 집합이 정확히 일치(11개).
5. CI: `circleci config validate`, `config process`(파라미터 ON)로 3개 command의 새 단계(`when: enable-lint` 안, `-c` 절대 경로 해석)와 신규 job 렌더링, lint-config-check의 `config verify`/errorlint/`bash -n`/mawk 확보/`--self-test` 단계 확인, gate job의 mawk 설치 단계 렌더링 확인, path-filtering mapping에 `.golangci-errorlint.yml` 행이 있고 그 파일만 바꾼 변경이 `run-lint-config-check`를 켜는지 확인, 서비스 디렉터리(예: bin-tag-manager)에서 4.1 명령이 설정을 로드하는지 확인, `\.circleci/config_work\.yml`만 바꾼 변경이 `run-lint-config-check`를 켜는지 확인, 처리된 설정에서 이 PR이 배포 workflow를 새로 켜지 않음을 확인(머지가 배포를 일으키지 않는다는 5장 주장의 검증).
   - 허용 표 drift 음성 시험(5번에 포함): database.md의 R2b 예외 표에서 경로 하나를 지우면 gate와 `--self-test`가 모두 실패한다.
6. 로컬 대표 서비스(측정 환경의 Go 1.25.3이 서비스 go.mod의 1.27.1과 달라 막히면 합성 저장소 결과로 대체했음을 보고에 명시): bin-tag-manager, bin-common-handler, bin-api-manager의 각 서비스 디렉터리에서 본 문서 4.1의 명령(`golangci-lint run --timeout 10m -c "$(git rev-parse --show-toplevel)/.golangci-errorlint.yml" --new-from-merge-base=origin/main`, 상대경로 `-c`는 실패하므로 쓰지 않는다)이 main과 같은 커밋에서 0건이고, 기존 `golangci-lint run`은 변경 없이 전체 이슈를 보고한다(합성 저장소에서 변경하지 않은 줄의 `unused`가 여전히 보고됨을 확인).
7. `scripts/check-docs.sh`(CLAUDE.md 줄 수 상한, README 존재)가 이 변경에서 영향을 받지 않음을 확인, 스크립트 `--self-test`를 mawk로도 실행해 통과, R2/R3 gate의 `resource_class: small` 소요 시간 측정, 가장 큰 서비스(bin-api-manager, `git ls-files 'bin-api-manager/*.go'` 기준 497개, 비테스트 263개)를 cgroup MemoryMax=2G에서 errorlint 단계로 실행해 `oom_kill` 0 확인과 기존 Linting 단계 대비 추가 소요 시간 측정, error-handling.md 4.3 직접 비교 6곳의 위치 기록, `ruby .circleci/scripts/check-auto-deploy-config.rb`가 OK이고 `scripts/check-service-docs.sh`가 이 변경(bin-* 소스 변경 없음)에서 경고하지 않음을 확인.
8. R3 fail-closed 음성 시험(4번에 포함): ID/CustomerID 정의 패턴을 포함한 한 줄 구조체는 gate 실패, `type ( ... )` 그룹 선언, 중첩/익명 구조체 안의 태그 줄(허용 목록에 있는 구조체 안의 중첩 포함), 주석 속 가짜 `type X struct {` 줄은 무시되고 그 뒤의 실제 수동 구조체가 정상 판정됨(허용 목록 밖이면 위반, 안이면 통과), 문자열 리터럴 `"a/*b"` 뒤의 수동 구조체와 `example:"http://a" db:"id,uuid"` 같은 태그 안의 `//`가 있어도 탐지되거나 gate 실패(조용히 통과하지 않음), 닫히지 않은 `/*`는 gate 실패, 원문 패턴 줄 수와 귀속 줄 수 불일치는 gate 실패, 별칭 UUID 타입과 `db:"id,omitempty"` 같은 태그 변형은 탐지됨, 허용 목록 부분 일치 방지(`rag:RagX`), 중복 키/빈 목록/읽기 실패는 gate 실패.
9. 문서와 QA 확인: error-handling.md 4.2(`%w` 해석 한 줄), models.md 6.1, database.md 7.1/7.7(R2b 해석, 한계, 본 문서 3.2.1의 R2b 허용 예외 표 포함)에 집행 방식을 반영하고, gate 메시지가 인용하는 절이 실제 존재함을 grep으로 확인한다. 또한 QA 카드 본문에 required status check 등록 여부(대표님 설정, `check-structure-conventions` 하나만)를 확인 항목으로 둔다. 머지 후 첫 서비스 PR의 errorlint 단계 실행 결과는 QA 카드(머지 전)가 볼 수 없으므로 QA 항목이 아니라 오케스트레이터(default) 담당이다(본 문서 5장).

## 7. 비범위 재확인과 후속 제안(생성하지 않음)

- 기존 위반 정리(에러 래핑 추정 1,589~1,676건 등)는 이 카드 범위 밖이다. 필요하면 규칙 단위 PR로 별도 지시를 받아 진행한다.
- 보고 항목(승인안에 포함 또는 부수 변경, 반영 후 보고): R1 해석, R3 구현 방식 변경(허용 목록, stale 실패), 문서 명문화 추가, CI 트리거 확장(본 문서 3.4). R1을 신규 테스트 코드에도 적용하고 R2a/R2b가 테스트 파일까지 검사하는 점(승인안은 테스트 포함 여부를 명시하지 않았음), R2a가 listenhandler의 교차 서비스 DTO import를 금지하지 않는 점(CLAUDE.md가 자기 서비스로 한정하지 않아 문서 문구 그대로 집행), R3가 `bin-*/models/` 밖의 정의(`bin-contact-manager/pkg/dbhandler/address*.go`)를 탐지하지 못하는 점(보고 대상은 조용히 통과하는 미탐지 형태 전부다: 필드명 변형 `Id`/`CustomerId`, 필드 병기(`ID`/`CustomerID`가 이름 목록의 마지막이 아닌 `ID, X uuid.UUID`류), 태그가 다른 줄로 나뉜 형태, 로컬 별칭 타입, dot-import, 해석된 문자열 태그. 이 형태들은 현재 0건이고 3.3 한계에 기재(별도로 `models/` 밖 정의는 실제로 2개 파일 존재). 반대로 `type ( )` 그룹, 중첩/익명 struct, 정의 패턴을 포함한 한 줄 구조체처럼 불변식이 실제로 막는 형태는 보고 대상이 아님), R1이 `go-test*` 3개 command를 쓰는 서비스에만 적용되는 점(`bin-openapi-manager` 등 제외), R1이 error-handling.md 4.3의 sentinel 직접 비교 지침과 미래에 충돌할 수 있는 점, R2a/R2b가 신규 라인 한정이 아니라 전체 트리 검사이며 기존 위반이 0이라 baseline이 없는 점(승인안의 "신규 라인 한정"에서의 구현 방식 조정)과, `lint-config-check`가 비차단으로 남는 한계도 보고한다. 승인안 대비 축소도 여기에 포함한다: depguard를 문서화된 DTO 경계 gate(R2a)로 대체하고 일반적인 서비스 간 import 금지는 제외, forbidigo(`logrus.Fatal`/`fmt.Print` 금지) 제외, DB 접근 경계를 squirrel import 위치(R2b)라는 대리 지표로 집행하며 raw SQL은 탐지하지 못함(2장, 본 문서 3.2).
- 대표님 설정 확인 항목: 신규 gate job(`check-structure-conventions`) 하나를 GitHub required status check에 등록해야 머지가 실제로 차단된다. 저장소 설정이라 대표님만 할 수 있고, 등록 전에는 gate가 비차단이다. 등록 대상은 매 pipeline 무조건 실행되는 `check-structure-conventions` 하나로 한정한다. `lint-config-check`는 path-filter 조건부 workflow라 required로 등록하면 해당 경로를 건드리지 않은 PR이 pending에 걸리므로 등록하지 않는다(그 job은 비차단 검증으로 남는다).
- 머지 후 관찰 항목: 서비스 3개 command의 errorlint 단계는 머지 후 첫 서비스 PR에서 처음 실행된다. 담당은 default이고 별도 카드나 티켓은 만들지 않는다(본 문서 5장).
- 결정 대기 항목: import 순서 불일치(본 문서 3.4), 일반 서비스 간 import 금지의 문서화 여부(본 문서 3.4).
