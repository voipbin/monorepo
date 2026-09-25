# 2026-09-26 monorepo 코드 컨벤션 강제 계층 복구 및 README 현행화

Status: Draft (design review loop 대기)
Branch: `NOJIRA-Monorepo-code-convention-and-readme-cleanup`
Worktree: `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Monorepo-code-convention-and-readme-cleanup`
Base: `origin/main` @ `fb8e8d58a`

---

## 1. Problem statement

monorepo(Go 37 서비스, 31,291 파일)의 코드 컨벤션이 서비스 연령에 따라 드리프트했다.
조사 결과 근본 원인은 "규칙 문서의 부재"가 아니라 **"선언된 규칙과 기계적으로 강제되는 규칙의 괴리"** 였다.

### 1.1 실측된 괴리 (증거)

루트 `CLAUDE.md:44-52`는 5단계 검증 워크플로우를 MANDATORY로 선언한다.
실제 CI(`.circleci/config_work.yml`)가 강제하는 것은 5단계 중 1개뿐이다.

| 선언된 단계 | CLAUDE.md 위치 | CI 실제 강제 여부 | 증거 |
|---|---|---|---|
| `go mod tidy` | `CLAUDE.md:48` | ❌ 미실행 | `config_work.yml` 내 `go mod tidy` 호출 없음 (vendor만 수행, L2220-2223) |
| `go mod vendor` | `CLAUDE.md:49` | ✅ 실행 | `config_work.yml:2223` |
| `go generate ./...` | `CLAUDE.md:50` | ⚠️ **부분 강제** | `go-test` command에는 없으나, `bin-openapi-manager-validate` job(L1498-1523)이 `go generate ./...` 후 `git diff --exit-code gens/models/gen.go`로 생성물 drift를 차단한다. **단 bin-openapi-manager 한정** |
| `go test ./...` | `CLAUDE.md:51` | ✅ 실행 | `config_work.yml:2237-2243` (`go test` 자체는 2241) |
| `golangci-lint run` | `CLAUDE.md:52` | ❌ **주석 처리** | `config_work.yml:2224-2236`, 동일 블록이 2267, 2315에도 존재 |

추가로 `go vet`이 같은 주석 블록(`config_work.yml:2236`)에 묻혀 함께 비활성화되어 있다.
주석 사유는 `# TODO: Re-enable golangci-lint after fixing OOM on small resource_class`로,
**`go vet`은 OOM과 무관한데 연좌로 죽었다.**

`.golangci.yml`은 저장소에 존재하지 않는다 (`git ls-files | grep -ci golangci` → 0).
즉 CLAUDE.md가 실행을 요구하는 `golangci-lint run`은 설정 파일조차 없는 상태다.

### 1.2 강제 부재의 결과

| 항목 | 실측치 |
|---|---|
| gofmt 미준수 파일 | **313개** (프로덕션 162 / 테스트 151, **35개** 서비스에 분포 — `voip-kamailio-proxy`·`voip-rtpengine-proxy` 포함) |
| 테스트 함수명 `TestXxx_Case` (규약 위반) | **626건 / 131파일** (`bin-*` 한정 599건 / 126파일, `voip-*` 27건) |
| testify 사용 (규약 위반, 외부 의존성) | **9파일** (api 5, ai 3, storage 1) |
| gomock 컨트롤러 변수 `ctrl` (규약 위반) | **194회 / 22파일** (`bin-*` 한정 171회 / 19파일) |
| 루트 README 서비스 표 누락 | 3건 (schedule / trigger-sender / webchat) |

집계 범위는 `bin-*/` + `voip-*/`이며 `vendor/`를 제외한다.
§6.4의 게이트 스크립트도 동일 범위를 검사하므로 수치와 게이트 대상이 일치한다.

참고로 `mockCtrl := gomock.NewController`는 저장소에 **0건**이다.
따라서 게이트 정규식에 `mockCtrl` 분기를 넣을 이유가 없다.

### 1.3 중요: 컨벤션 문서는 이미 완비되어 있다

`docs/conventions/testing.md`(383줄)는 위반된 규칙을 **이미 전부 정확히 명문화**하고 있다.

| 규칙 | testing.md 선언 위치 |
|---|---|
| `Test_<MethodName>` 함수명 | L128 `Use `Test_<MethodName>`:`, L132-134 예시 |
| `mc := gomock.NewController(t)` | L33, L272, L305 |
| `defer mc.Finish()` 페어링 | L380 (체크리스트 항목) |
| `reflect.DeepEqual` 어서션 | L116, L292 |
| `<source>_test.go` 1:1 파일명 | L342-344 (디렉터리 트리 예시) |
| `t.Run(tt.name, ...)` 서브테스트 | L32, L271 |

**따라서 "컨벤션을 문서에 명문화한다"는 조치는 이미 완료된 일을 반복하는 것이며, 드리프트를 막지 못함이 실증되었다.**
이번 작업은 문서 추가가 아니라 **기계적 강제 수단의 도입**에 집중한다.

### 1.4 조사에서 기각된 가설

착수 시 가설은 "AI 코딩 에이전트가 기존 컨벤션을 훼손했다"였으나, 프로덕션 코드 전수 조사 결과 **부분 기각**되었다.

- 프로덕션 코드 7개 축 중 5개 축에서 드리프트 없음. 생성자 패턴 280/280 준수, 파일 구조 위반 3건은 **전부 2024년 구(舊) 코호트**.
- 최대 결함인 에러 체인 파괴(`fmt.Errorf("...%v", err)` 1,602건)도 **64%(1,028건)가 구 코호트**.
- 드리프트가 실재하는 영역은 **테스트 코드에 한정**되며, 여기서는 서비스 연령과 상관관계가 뚜렷하다
  (`TestXxx_Case`가 bin-call/flow/queue/conference/agent-manager에서 정확히 0건, bin-timeline 127건·bin-rag 41건).

이 결론은 조치의 방향을 바꾼다: "AI 산출물 교정"이 아니라 **작성 주체와 무관한 기계적 게이트 설치**가 옳은 해법이다.

---

## 2. Goals

1. **G1.** CI에서 `go vet`을 복구한다. (OOM과 무관하므로 즉시 가능)
2. **G2.** 루트 `.golangci.yml`을 신설하고 CI에서 `golangci-lint`를 복구한다. 단 OOM을 유발하지 않아야 한다.
3. **G3.** gofmt 미준수 313개 파일을 일괄 정리하고, 이후 재발을 CI가 차단한다.
4. **G4.** `docs/conventions/testing.md`가 이미 규정한 테스트 규칙 중 린터로 잡히지 않는 3건
   (`TestXxx_Case` 함수명 / testify import / `ctrl` 변수명)을 **변경된 파일에 한해** CI가 차단한다.
5. **G5.** 루트 `README.md`의 서비스 표 누락 3건을 보완하고, 셀프호스팅 경로를 현행화한다.

각 목표는 §9 검증 계획에서 실행 가능한 명령으로 확인한다.

---

## 3. Decisions locked (2026-09-26, 대표님 확정)

| # | 결정 | 근거 |
|---|---|---|
| D1 | **단일 PR로 진행** | CLAUDE.md "Don't Split Work Into Multiple PRs Without Permission" 원칙 준수 |
| D2 | `.golangci.yml` 초기 강도는 **보수적** (현재 통과하는 표준 린터 + gofmt/goimports). 점진 강화 | 위반 수천 건 유발 시 PR이 리뷰 불가능해짐 |
| D3 | README 정비 범위는 **루트 README만**. 서비스별 37개 README는 제외 | 스코프 제어 |
| D4 | OOM 회피는 **변경된 서비스만 린트**. resource_class 상향하지 않음 | 비용 증가 최소화 |
| D5 | 테스트 컨벤션은 **기존 방식 유지**(`Test_FuncName`, `reflect.DeepEqual`, `mc`). Go 커뮤니티 표준으로 전환하지 않음 | 정통성 + 다수파 일치 (§1.3) |
| D6 | 존량 드리프트(테스트 함수명 599건 등)는 **이번 PR 범위 밖**. 점진 교체 방침 | 스코프 제어. 단 D7로 신규 유입은 차단 |
| D7 | 테스트 규칙 강제는 **grep 기반 경량 CI 게이트 스크립트**. 신규/변경 파일만 검사 | golangci-lint로는 함수명 규칙을 잡을 수 없음 |
| D8 | 에러 체인 복구(`fmt.Errorf` 1,602건)는 **이번 PR 범위 밖** | 별도 설계 필요. 64%가 구 코호트라 "AI 교정" 프레이밍이 부적절 |

---

## 4. Non-goals (명시적 스코프 컷)

| 제외 항목 | 규모 | 사유 / 후속 |
|---|---|---|
| 테스트 함수명 존량 수정 | 599건 / 126파일 | D6. 게이트 설치 후 서비스별 후속 PR |
| testify 제거 존량 | 9파일 | D6. 단 D7 게이트가 신규 유입 차단 |
| `ctrl` → `mc` 존량 리네이밍 | 171회 / 19파일 | D6 |
| `exepct` 오타 정정 | 179회 / 37파일 | D6. 정통 관용구의 오타이나 기능 영향 없음 |
| 에러 체인 복구 (`%v` → `%w`) | 1,602건 | D8. 별도 설계 문서 필요 |
| 서비스별 README 37개 표준화 | 16개는 `##` 섹션 0개 | D3 |
| 서비스별 CLAUDE.md 37개 정리 | 중복률 7.9% | 실질 모순 없음이 확인됨. 불필요 |
| `go mod tidy` / `go generate` CI 추가 | — | 본 PR은 lint 게이트에 집중. tidy/generate는 diff 검증 방식 설계가 별도로 필요 |
| `bin-trigger-sender` Go 1.25.3 → 1.27.1 통일 | 1개 모듈 | Dockerfile도 일관되게 1.25이므로 의도적일 가능성. 별도 확인 후 처리 |
| `.gitignore` vs `git add -f` 872파일 자기모순 | — | RST 빌드 산출물 정책과 얽혀 있어 별도 논의 필요 |
| `scripts/check-service-docs.sh` 죽은 코드 정리 | 110줄 | 별도 후속 |

---

## 5. Affected files

| 파일 | 변경 내용 | 목표 |
|---|---|---|
| `.golangci.yml` | **신규 생성** | G2, G3 |
| `.circleci/config_work.yml` | `commands:` 3곳의 lint 주석 해제 + 수정, `enable-lint` 파라미터 3곳, `run-lint-config-check` 파라미터·workflow·job 신설, `check-test-conventions` job·workflow 신설 | G1, G2, G4 |
| `.circleci/config.yml` | path-filtering mapping에 `\.golangci\.yml` 1행 추가 (§6.2.1.1) | G2 |
| `scripts/check-test-conventions.sh` | **신규 생성** (grep 게이트) | G4 |
| `README.md` | 서비스 표 3행 추가, 셀프호스팅 섹션 현행화, em dash 11건 정리, L110 배포 서술 수정 | G5 |
| `bin-*/`·`voip-*/` 하위 `*.go` (313 파일) | `gofmt -w` 일괄 적용. **313파일 중 4개는 `voip-*` 소속**이다(§1.2) | G3 |
| `bin-call-manager/pkg/dbhandler/json_expr.go` | doc comment를 탭 들여쓰기 코드블록으로 변환 (§6.3.2). 위 313파일에 포함되나 별도 수작업이 필요하다 | G3 |
| `docs/plans/2026-09-26-...md` | 본 설계 문서 | — |

> **주의:** `commands:` 블록은 3개 정의(`go-test`, `go-test-api-manager`, `go-test-pipecat-manager`)만 존재하고
> 37개 job이 이를 호출한다. 따라서 **3곳 수정으로 전량 반영**된다. job 37개를 개별 수정하지 않는다.
> 근거: `config_work.yml` 내 `go-test` 참조 40건 = 정의 3 + 호출 37.

---

## 6. Detailed design

### 6.1 `.golangci.yml` (신규)

golangci-lint v2 스키마를 사용한다 (로컬 확인 버전: v2.13.2, **CI 설치 대상: v2.14.0** — §6.2.2 및 §8 R2 참조).

```yaml
version: "2"

run:
  timeout: 10m

linters:
  default: standard
  exclusions:
    generated: lax
    paths:
      - vendor
    rules:
      - path: _test\.go
        linters:
          - errcheck

# Only gofmt is enabled. goimports is intentionally NOT enabled here: it flags
# 13 files beyond gofmt's 313, all of them cases where a third-party import is
# not separated into its own group. Enabling it also forces a decision on
# formatters.settings.goimports.local-prefixes, because without it goimports
# treats this monorepo's own packages as third-party. Both belong to a
# follow-up change, not to this one.
formatters:
  enable:
    - gofmt
  exclusions:
    generated: lax
    paths:
      - vendor
```

설계 근거:

- `default: standard` — errcheck / govet / ineffassign / staticcheck / unused. D2의 "보수적" 요구에 부합.
- `_test.go`에서 errcheck 제외 — 테스트의 `defer f.Close()` 관용구를 위반으로 잡지 않기 위함.
- v2에서 포맷터는 `linters`가 아닌 `formatters` 블록에 선언한다.
- **`goimports`는 활성화하지 않는다.** 실측상 `goimports -l` 326파일 vs `gofmt -l` 313파일로
  13파일이 추가 검출된다. 정리 명령(§6.3)과 검증(V1)이 gofmt 기준이므로,
  goimports를 켜면 **V1이 0을 반환하고도 6개 서비스의 CI가 실패**한다.
  또한 `local-prefixes` 정책 결정이 선행되어야 한다(미설정 시 monorepo 자기 패키지를 서드파티로 취급).
  이번 범위에서 제외하고 §4 Non-goals에 명시한다.
- **`concurrency` 키는 설정하지 않는다.** §6.1.2에서 실측으로 무효함이 확인되었다.
- 메모리 제어는 설정 파일이 아니라 **CI step의 환경변수**(`GOGC`, `GOMEMLIMIT`, `GOMAXPROCS`)로 수행한다. §6.2 참조.

> 이 스니펫은 커밋된 `.golangci.yml`과 **바이트 단위로 일치해야 한다.**
> 둘이 갈라지면 구현자가 어느 쪽을 정본으로 볼지 알 수 없다.

**실측 결과 (worktree에서 실제 실행, golangci-lint v2.13.2, 로컬 16코어):**

| 서비스 | Go 파일 | 위반 | 피크 RSS | 소요 |
|---|---|---|---|---|
| bin-tag-manager | 50 | gofmt 3 | — | — |
| bin-storage-manager | 88 | gofmt 1 | — | — |
| bin-transfer-manager | 47 | gofmt 1 | — | — |
| bin-common-handler | 295 | gofmt 3 | 0.81 GB | 23s |
| bin-call-manager | 361 | gofmt 3 | 0.93 GB | 45s |
| bin-ai-manager | 354 | gofmt 3 | 1.12 GB | 45s |
| **bin-api-manager** | **482** | gofmt 3 | **2.18 GB** | 60s |

**핵심 소견 1 — 린터 위반이 사실상 없다.** 7개 서비스 전부에서
errcheck / govet / ineffassign / staticcheck / unused 위반이 **0건**이며, gofmt만 검출되었다.
따라서 §6.3의 gofmt 일괄 적용 후 표준 린터는 위반 0에 수렴한다. D2(보수적 강도)의 도입 비용은 사실상 없다.

**핵심 소견 2 — OOM 리스크는 실재한다.** bin-api-manager 피크 2.18 GB는
CircleCI `small`(2 vCPU / 4 GB)에서 Go 빌드 캐시·벤더 다운로드와 경합하면 위험 구간이다.
과거 주석 처리 사유("OOM on small resource_class")가 이 수치로 설명된다.

**측정 무효 사례 (기록 목적).** `--concurrency 2 --GOMAXPROCS=2` 1차 측정이 0.80s / 135 MB로 나왔으나,
직전 동일 서비스 측정(60s / 2.18 GB) 대비 75배 빠른 것은 물리적으로 불가능하다.
golangci-lint 결과 캐시 히트였다. **이 수치는 채택하지 않는다.**

### 6.1.2 OOM 완화 수단 실측 (bin-api-manager, 콜드 캐시)

모든 측정은 `golangci-lint cache clean` + `rm -rf ~/.cache/golangci-lint` 직후 수행했다.

| 조건 | 피크 RSS | 소요 | 기본 대비 메모리 |
|---|---|---|---|
| 기본 (제한 없음, 대조군) | 2.40 GB | 43.0s | — |
| `concurrency=2` + `GOMAXPROCS=2` | 2.36 GB | 43.6s | **-1.8%** |
| **`GOGC=50` + `GOMAXPROCS=2`** | **1.66 GB** | 61.3s | **-31%** |
| `GOGC=20` + `GOMAXPROCS=2` | 1.44 GB | 94.7s | -40% |
| `GOMEMLIMIT=2GiB` + `GOGC=20` | 1.41 GB | 92.7s | -41% |

**결론 1 — `concurrency`는 OOM 대책이 될 수 없다.** 대조군 대비 1.8% 차이는 측정 노이즈 수준이다.
golangci-lint의 메모리는 린터 실행 병렬도가 아니라 **패키지 로딩과 타입 체크**가 지배하며,
`concurrency`는 후자를 제어하지 않는다. 초안 설계의 "`concurrency: 2`가 OOM 완화의 핵심"은 **오류였고 철회한다.**

**결론 2 — `GOGC=50`을 채택한다.** 4 GB 컨테이너에서 피크 1.66 GB는
Go 빌드 캐시·벤더 다운로드와 병존할 여유가 충분하며, 비용은 +18초에 그친다.
`GOGC=20`은 0.22 GB를 더 줄이기 위해 33초를 추가로 지불하므로 교환비가 나쁘다.

**결론 3 — `GOMEMLIMIT=3GiB`를 안전망으로 병기한다.** `GOGC=50` 단독으로 충분하지만,
향후 서비스가 커져 예상을 벗어날 때 `GOMEMLIMIT`은 프로세스를 OOM으로 죽이는 대신
GC를 강제해 **감속하되 완주**하게 만든다. 추가 비용이 없으므로 넣지 않을 이유가 없다.
값은 컨테이너 상한(4 GB)보다 낮되 정상 피크(1.66 GB)보다 충분히 높은 3 GiB로 둔다.

**결론 4 — resource_class 상향은 불필요하다. D4를 유지한다.**

### 6.2 `.circleci/config_work.yml` 수정

현재 상태 (`config_work.yml:2224-2236`, 동일 블록이 2267·2315에 반복):

```yaml
      # TODO: Re-enable golangci-lint after fixing OOM on small resource_class
      # - run:
      #     name: Install golangci-lint v2.5.0
      #     command: |
      #       curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v2.5.0
      #       sudo mv ./bin/golangci-lint /usr/local/bin/
      #       golangci-lint version
      # - run:
      #     name: Linting and vet
      #     command: |
      #       cd << parameters.source-directory >>
      #       golangci-lint run -v --timeout 5m
      #       go vet $(go list ./...)
```

변경 후:

```yaml
      - run:
          name: Install golangci-lint
          # Pinned to a release built with Go >= the go directive in our go.mod
          # files (1.27.1). golangci-lint REFUSES to run when the Go version it
          # was built with is older than the targeted Go version, so an older
          # pin (e.g. v2.5.0, built with go1.25.1) fails every job with:
          #   "the Go language version (go1.25) used to build golangci-lint is
          #    lower than the targeted Go version (1.27.1)"
          # When bumping the go directive, bump this pin to a matching release.
          #
          # The upstream install.sh is NOT used: it resolves the checksum from
          # the wrong entry in checksums.txt (picks the .sbom.json hash) and
          # exits 0 on verification failure, leaving no binary behind.
          environment:
            GOLANGCI_LINT_VERSION: "2.14.0"
          command: |
            cd /tmp
            base="https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}"
            tarball="golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64.tar.gz"
            curl -sSLf -o "${tarball}" "${base}/${tarball}"
            curl -sSLf -o checksums.txt "${base}/golangci-lint-${GOLANGCI_LINT_VERSION}-checksums.txt"
            grep " ${tarball}\$" checksums.txt | sha256sum -c -
            tar xzf "${tarball}"
            sudo mv "golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64/golangci-lint" /usr/local/bin/
            installed="$(golangci-lint version | grep -oE 'version [0-9]+\.[0-9]+\.[0-9]+' | awk '{print $2}')"
            if [ "${installed}" != "${GOLANGCI_LINT_VERSION}" ]; then
              echo "golangci-lint version mismatch: want ${GOLANGCI_LINT_VERSION}, got ${installed}"
              exit 1
            fi
      - run:
          name: Linting
          # GOGC/GOMEMLIMIT keep the peak RSS inside the 4 GB `small` container.
          # Measured on bin-api-manager (the largest service, 482 Go files):
          # default 2.40 GB / 43 s  ->  GOGC=50 1.66 GB / 61 s.
          # `--concurrency` was measured and does NOT reduce memory (-1.8%).
          #
          # GOMAXPROCS is pinned because the measurements above were taken with
          # GOMAXPROCS=2. Go's runtime reads the host CPU count, not the cgroup
          # limit, on CircleCI's docker executor, so without this the container
          # would run with far more Ps than the 2 vCPUs it actually has and the
          # measured footprint would not reproduce.
          environment:
            GOGC: "50"
            GOMEMLIMIT: 3GiB
            GOMAXPROCS: "2"
          command: |
            cd << parameters.source-directory >>
            golangci-lint run --timeout 10m
```

변경점과 근거:

1. **별도 `go vet` step을 두지 않는다 (초안에서 철회).**
   초안은 "lint와 분리하면 lint가 죽어도 vet은 살아남는다"는 이유로 별도 step을 두었으나,
   **govet은 `linters.default: standard`에 이미 포함**되므로 그 논리가 성립하지 않는다.
   lint가 죽으면 govet도 함께 죽는 것은 동일하다. §6.2.3의 실측 근거로 삭제한다.
2. **`-v` 플래그 제거.** verbose 출력은 메모리·로그 부담만 늘리고 실패 진단에 기여하지 않는다.
3. **`--timeout 5m` → `10m`.** `GOGC=50`은 실행 시간을 43s → 61s로 늘린다(§6.1.2).
   타임아웃은 메모리를 쓰지 않으므로 여유를 둔다.
4. **OOM 대응은 `GOGC=50` + `GOMEMLIMIT=3GiB`가 담당.** §6.1.2 실측 근거.
   resource_class는 상향하지 않는다 (D4 유지).
5. **`GOMAXPROCS: "2"` 고정.** §6.1.2의 측정이 `GOMAXPROCS=2` 조건에서 이루어졌는데,
   CircleCI docker executor에서 Go 런타임은 cgroup 제한이 아니라 **호스트 코어 수**를 보는 경우가 있다.
   명시하지 않으면 측정 조건이 CI에 재현되지 않는다.
6. **버전 핀 v2.5.0 → v2.14.0.** §6.2.2의 실측 근거. **원래 주석을 그대로 해제하면 37개 job이 전부 실패한다.**
7. **`install.sh` 의존 제거.** §6.2.2 참조. 세 가지 문제가 있다 —
   (a) 체크섬 오선택 버그, (b) 검증 실패 시에도 exit 0, (c) `master` 브랜치 참조로 설치가 재현 불가능.
8. **설치 후 버전 일치 확인 가드 추가.** 조용한 설치 실패 시 이전 버전으로 통과하는 사태를 막는다.
9. **변경분만 린트되는 구조는 이미 존재.** `.circleci/config.yml`의 `path-filtering/filter`가
   변경된 서비스의 job만 트리거하므로, 추가 작업 없이 D4가 충족된다.
10. **`check-test-conventions.sh`는 이 command에 넣지 않는다.** 37개 job에서 중복 실행되기 때문이다.
    §6.2.1의 독립 job으로 1회만 실행한다.

### 6.2.3 `go vet` 별도 step을 두지 않는 근거 (실측)

초안은 G1을 "`go vet` 복구"로 정의하고 별도 step을 두었다. 이는 불필요하다.

**analyzer 집합 비교:** `go tool vet help`는 35개 analyzer를 등록한다.
golangci-lint의 govet은 `linters.default: standard`에서 기본 활성화되며,
집합 차이는 `loopclosure` 1개뿐이다. 그런데 이 analyzer는 **이 저장소에서 무의미하다**:

- 공식 문서: "An iteration variable can only outlive a loop iteration in Go versions **<=1.21**."
- 이 저장소 모듈 전수: `go 1.27.1` 38개 + `go 1.25.3` 1개. **1.21 이하 모듈 0개.**

**실제 검출 동작 비교** (printf 위반 2건 + unreachable 1건, 테스트 파일 포함):

```
$ go vet ./...
./b.go:11:2: unreachable code
./b.go:6:14: fmt.Printf format %d has arg "not-an-int" of wrong type string
./c_test.go:9:14: fmt.Printf format %d has arg "bad-in-test" of wrong type string

$ golangci-lint run   (default: none, enable: govet)
b.go:11:2: unreachable: unreachable code (govet)
b.go:6:14: printf: ... (govet)
c_test.go:9:14: printf: ... (govet)
```

**완전히 동일하다.** `run.tests`가 기본 true이므로 golangci-lint도 `_test.go`를 검사한다.
`go vet`이 더 넓게 검사하는 영역은 없다.

**비용:** bin-api-manager에서 `go vet` 단독 실행은 약 55초 / 1.18 GB(`GOMAXPROCS=2`)다.
이 step은 `GOGC`/`GOMEMLIMIT`가 걸리지 않아 메모리 제어도 없다.
**37개 job에서 0건의 추가 검출을 위해 이 비용을 지불할 이유가 없다.**

따라서 G1은 별도 step이 아니라 **golangci-lint 복구(G2)에 흡수**된다.
이에 따라 §10 Q4(`go vet` 복구 시 신규 위반 처리 방침)도 자동 해소된다.

### 6.2.2 golangci-lint 버전·설치 방식 실측 (CI 전면 실패를 막은 검증)

기존 주석 블록은 `v2.5.0`을 `install.sh`로 설치한다. **이 조합은 현재 저장소에서 동작하지 않는다.**
아래는 전부 실제 실행 결과다.

| # | 검증 | 명령 | 결과 |
|---|---|---|---|
| 1 | v2.5.0 스키마 검증 | `golangci-lint config verify` (v2.5.0) | **EXIT=0 통과** |
| 2 | v2.5.0 실제 실행 | `golangci-lint run` (v2.5.0, bin-tag-manager) | **exit 3 실패** |
| 3 | v2.14.0 install.sh 설치 | `install.sh \| sh -s v2.14.0` | **실패하나 exit 0** |
| 4 | v2.14.0 공식 tarball 체크섬 | `sha256sum -c` | **OK** |
| 5 | v2.14.0 실제 실행 | `golangci-lint run` (v2.14.0) | **정상, gofmt 3건 검출, 398 MB / 6.3s** |

**발견 1 — v2.5.0은 실행 불가.** #2의 실패 메시지:

```
Error: can't load config: the Go language version (go1.25) used to build
golangci-lint is lower than the targeted Go version (1.27.1)
```

v2.5.0은 go1.25.1로 빌드되었고, 38개 모듈의 go directive는 1.27.1이다.
golangci-lint는 빌드 Go 버전 < 타깃 Go 버전 조합을 거부한다.
**기존 주석을 그대로 해제하면 37개 서비스 job이 전부 즉시 실패한다.**

**발견 2 — `config verify`는 이 결함을 잡지 못한다.** #1이 EXIT=0으로 통과했음에도 #2가 실패했다.
verify는 설정 스키마만 확인하고 툴체인 호환성은 보지 않는다.
따라서 **검증 계획에서 `config verify`만으로 충분하다고 판단해서는 안 된다**(§9 V5 수정).

**발견 3 — 공식 `install.sh`가 깨져 있다.** #3의 출력:

```
err hash_sha256_verify checksum for '.../golangci-lint-2.14.0-linux-amd64.tar.gz' did not verify
  0cff1e23f6d9... vs ab90aeb7b066...
```

`0cff1e23...`는 `checksums.txt`상 **`.sbom.json` 파일의 해시**이고,
tarball의 실제 해시는 `ab90aeb7...`다 (#4에서 `sha256sum -c` OK로 확인).
스크립트가 `checksums.txt`에서 엉뚱한 행을 집는 버그다.
더 위험한 것은 **검증 실패 후에도 exit 0을 반환**하여 바이너리 없이 "성공"으로 끝난다는 점이다.
CI에서 이 경우 `sudo mv`가 실패하거나, 이전 버전 바이너리가 남아 있으면 **핀이 무시된 채 조용히 통과**한다.

**결론:** `install.sh`를 쓰지 않고 릴리스 tarball을 직접 받아 체크섬을 검증한다.
설치 후 `golangci-lint version`으로 기대 버전과 대조하는 가드를 둔다.
버전 핀은 go directive와 연동되므로, 그 이유를 config에 주석으로 남긴다.

### 6.2.1 테스트 컨벤션 게이트를 독립 job으로 분리

`scripts/check-test-conventions.sh`는 저장소 전역의 변경 파일을 보므로 서비스별로 반복할 이유가 없다.
`go-test` command에 넣으면 최대 37회 중복 실행된다. 기존 `run-shell-tests` job과 같은 층위에 둔다.

```yaml
  check-test-conventions:
    docker:
      - image: cimg/go:1.27.1
    resource_class: small
    steps:
      - checkout
      - run:
          name: Fetch base branch
          # CircleCI's checkout only fetches the current branch's refspec, so
          # origin/main may not exist in the clone. The gate is fail-closed and
          # would abort without this.
          #
          # Do NOT add --depth here. `--depth` marks the whole repository
          # shallow, and a shallow boundary breaks ancestry computation even
          # when the base commit object is present: measured, `git cat-file -t`
          # reports the commit while `git merge-base` exits 1. With the gate
          # running unconditionally and fail-closed, that turns every PR whose
          # base is behind the fetched window red regardless of its contents.
          # 21 of the 23 live remote branches are already more than 200 commits
          # behind, and main moves ~121 commits a month.
          #
          # The clone may also arrive shallow from checkout itself, so a plain
          # fetch is not always enough; deepen only when the merge base cannot
          # be resolved, to avoid paying for full history on every run.
          command: |
            git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'
            if ! git merge-base origin/main HEAD >/dev/null 2>&1; then
              echo "Merge base unreachable; deepening the clone."
              git fetch --no-tags --unshallow origin '+refs/heads/main:refs/remotes/origin/main' \
                || git fetch --no-tags --deepen=1000 origin '+refs/heads/main:refs/remotes/origin/main'
            fi
      - run:
          name: Check test conventions
          command: bash scripts/check-test-conventions.sh
```

workflow에는 `when:` 없이 등록하여 무조건 실행한다(경량 grep이므로 path-filtering 불필요).

**주의:** `config_work.yml`의 기존 41개 workflow는 **전부 `when:` 게이트**를 갖는다.
`when` 없는 workflow는 continued config에서 항상 실행되므로 문법상 유효하나,
이 저장소에서는 최초 사례가 된다. 구현 시 이 점을 PR 설명에 명시한다.

#### 6.2.1.1 `.circleci/config.yml` path-filtering mapping 보강 (필수)

현재 mapping에는 `.golangci.yml`과 `scripts/` 항목이 **없다**
(`.circleci/scripts/.*` → `run-shell-tests`만 존재).

이 때문에 **`.golangci.yml`만 수정하는 후속 PR은 어떤 lint job도 트리거하지 않는다.**
이번 PR은 35개 서비스에 `.go` 변경이 있어 우연히 넓게 돌지만,
이후 린터 튜닝 PR은 검증 없이 머지된다. 이는 조용한 사각지대다.

**초안의 mapping 두 줄은 그대로 쓸 수 없다. 실측으로 확인된 문제:**

| 초안 | 문제 | 실측 근거 |
|---|---|---|
| `.golangci.yml → run-lint-config-check true` | 이 파이프라인 파라미터가 **선언되어 있지 않다.** path-filtering이 미선언 파라미터를 continuation에 넘기면 continuation 자체가 실패한다 | `grep -c run-lint-config-check .circleci/config_work.yml` → `0` |
| `scripts/.* → run-shell-tests true` | 이 파라미터가 켜는 `shell-tests` job은 **bats만 실행**한다. 신설 스크립트를 전혀 검증하지 않으므로 R9 완화 근거가 성립하지 않는다 | `config_work.yml:2202` = `bats .circleci/tests/*.bats docs/reference/tests/*.bats` |

따라서 파라미터 선언·workflow·job을 **함께 신설**해야 한다.

> **먼저 읽을 것:** 아래 (1)~(4) 중 `run-convention-scripts` / `convention-scripts` /
> `scripts/.*` 관련 부분은 **채택하지 않는다**(이유는 이 절 끝의 "결정" 참조).
> 실제 추가분은 `run-lint-config-check` 파라미터 + mapping 1행 + workflow 1개 + job 1개다.

**(1) `config_work.yml` 파라미터 선언** (기존 `run-shell-tests` L186 옆에 추가)

```yaml
  run-lint-config-check:
    type: boolean
    default: false
```

**(2) `config.yml` mapping** (기존 L59-62 블록 옆)

기존 행들과 동일한 `<정규식><공백정렬><파라미터> true` 단일 라인 형식이며,
`docs/reference/extractor\.sh run-shell-tests true`(L61)처럼 `\.` 이스케이프 선례가 있다.

```
            \.golangci\.yml             run-lint-config-check true
```

**(3) `config_work.yml` workflow**

```yaml
  lint-config-check:
    when: << pipeline.parameters.run-lint-config-check >>
    jobs:
      - lint-config-check
```

**(4) `lint-config-check` job** — `.golangci.yml` 변경 시 대표 서비스 1개에서 실제 실행한다.
`config verify`만으로는 불충분함이 이미 증명되었다(v2.5.0이 verify 통과 후 run에서 exit 3).

```yaml
  lint-config-check:
    docker:
      - image: cimg/go:1.27.1
    resource_class: small
    steps:
      - checkout
      - run:
          name: Install golangci-lint
          command: |
            # same pinned install block as the lint step in 6.2.2
      - run:
          name: Verify config and run against a representative service
          # bin-tag-manager is the smallest service (50 Go files, 398 MB peak,
          # 6.3 s) yet exercises the full config. Running it catches schema
          # errors that `config verify` alone does not: v2.5.0 passed verify
          # and then failed `run` with exit 3 on this repo's Go version.
          environment:
            GOGC: "50"
            GOMEMLIMIT: 3GiB
            GOMAXPROCS: "2"
          command: |
            golangci-lint config verify
            cd bin-tag-manager
            go mod vendor
            golangci-lint run --timeout 10m
```

**주의:** `check-test-conventions` job은 §6.2.1에서 이미 "무조건 실행"으로 등록된다.
위 (3)의 `convention-scripts` workflow는 **스크립트 자체가 수정될 때** 추가로 도는 것이 아니라,
`check-test-conventions`가 이미 무조건 실행되므로 **중복이다.**
→ **결정: (1)의 `run-convention-scripts`와 (3)의 `convention-scripts` workflow는 만들지 않는다.**
`scripts/.*` mapping 행도 추가하지 않는다. 게이트가 항상 돌기 때문에 트리거가 불필요하다.
R9는 "mapping 추가로 해소"가 아니라 **"게이트를 무조건 실행으로 등록하여 해소"**로 근거를 바꾼다.

즉 실제 추가분은 **`run-lint-config-check` 파라미터 + mapping 1행 + workflow 1개 + job 1개**다.

### 6.3 gofmt 일괄 적용

```bash
gofmt -w $(gofmt -l bin-*/ voip-*/ | grep -v /vendor/)
```

313개 파일. **vendor/ 제외는 필수** — vendor는 서드파티 코드이며 `.gitignore` 대상이다.

#### 6.3.1 diff 성격 실측 (초안의 "대부분 빈 줄" 서술은 오류였다)

전체 diff 9,550줄을 집계한 결과(`git diff` 기본 알고리즘 기준. `diff` 유틸리티로 세면
1,708/1,704로 다르게 나오나 비공백이 대부분이라는 결론은 동일하다):

| 분류 | 추가(+) | 삭제(-) |
|---|---|---|
| **비공백 라인** | **2,210** | **2,206** |
| 빈 줄 | 63 | 116 |

**비공백 라인 변경이 98%이고 빈 줄은 2%다.** 지배적 변경은 struct 필드/태그 정렬 재계산과
import 블록 정렬이며, 초안이 예시로 든 "불필요한 빈 줄 제거"는 소수 사례였다.

공백을 전부 제거한 뒤 바이트 비교로 313파일을 전수 검사하면 **57개 파일이 비공백 변경**을 갖는다:

- 54개 — 문자 다중집합이 보존됨 (= import 그룹 재정렬)
- 3개 — doc comment 재작성

#### 6.3.2 ⚠️ gofmt가 주석 내용을 실제로 손상시키는 사례 (구현 시 필수 확인)

`bin-call-manager/pkg/dbhandler/json_expr.go`에서 **문자가 치환된다**:

```diff
-// `json_remove(<column>, replace(json_search(<column>, 'one', ?), '"', ''))`,
+// `json_remove(<column>, replace(json_search(<column>, 'one', ?), '"', ”))`,
```

바이트 확인: `27 27`(ASCII `''`) → `E2 80 9D`(U+201D RIGHT DOUBLE QUOTATION MARK).
Go 1.19+ doc comment 포맷터의 스마트쿼트 변환이다.

컴파일 의미는 바뀌지 않지만 **문서화된 SQL 식이 복사·붙여넣기 불가 상태로 손상**된다.
9,550줄 diff 안에서 리뷰어가 이를 발견할 확률은 사실상 0이므로, 기계적 확인이 필요하다.

**조치 (실험으로 확정, 조건부가 아니다):**

초안은 "ASCII로 되돌린다. 되돌린 상태가 gofmt를 통과하지 못하면 코드 블록으로 바꾼다"고
조건부로 기술했으나, **ASCII 복원은 항상 실패한다.** 실측:

```
$ gofmt -w json_expr.go          # L40에 U+201D 도입
$ sed -i "s/”/''/g" json_expr.go # ASCII로 복원
$ gofmt -l json_expr.go
json_expr.go                     # <- 다시 미준수. gofmt가 재차 치환한다
```

즉 ASCII 복원 분기는 **무한 반복**이며 채택할 수 없다. 구현자가 첫 번째 분기를 시도하면
CI와 로컬이 영구히 어긋난다. 따라서 **코드 블록 변환만이 유일한 조치**다.

백틱 인라인 코드는 doc comment 포맷터의 스마트쿼트 변환 대상이다.
탭 들여쓰기 코드 블록으로 바꾸면 대상에서 제외된다:

```go
// exprJSONArrayRemoveByValue builds
//
//	json_remove(<column>, replace(json_search(<column>, 'one', ?), '"', ''))
//
// deleting the first array element equal to the given value.
```

검증 완료 (스크래치 사본에서 실행):

| 확인 | 결과 |
|---|---|
| `gofmt -l` (변환 직후) | 출력 없음 = 이미 준수 |
| `gofmt -w` 후 ASCII `''` 보존 | L41에 `'"', ''` 그대로 유지 |
| 2회차 `gofmt -l` (멱등성) | 출력 없음 = 안정, 루프 없음 |
| `gofmt -e` 구문 검사 | 통과 |

**손상되는 곳은 이 파일의 L40 한 곳뿐이다.** 같은 식이 L58에도 나오지만
그쪽은 **이미 탭 들여쓰기 코드블록**(`//\tIF(json_search(...`)이라 doc comment 포맷터의
스마트쿼트 변환 대상이 아니다. 313파일 전수 스캔에서 신규 비ASCII가 1건뿐인 이유가 이것이며,
동시에 위 조치(백틱 인라인 → 코드블록)가 옳은 방향임을 저장소 자신이 보여주는 사례다.

**V15의 단일 파일 범위 근거:** 313파일 전수에서 gofmt 전후의 **비ASCII 바이트 다중집합**을
비교한 결과, 변하는 파일은 `json_expr.go` **1건뿐**이다.

```
$ for f in $(cat gofmt_list); do
    a=$(LC_ALL=C grep -oP '[\x80-\xFF]' "$f" | sort | uniq -c | md5sum)
    b=$(gofmt "$f" | LC_ALL=C grep -oP '[\x80-\xFF]' | sort | uniq -c | md5sum)
    [ "$a" != "$b" ] && echo "MULTISET-CHANGED: $f"
  done
MULTISET-CHANGED: bin-call-manager/pkg/dbhandler/json_expr.go
```

`LC_ALL=C`는 생략하면 안 된다. UTF-8 로캘에서 `[\x80-\xFF]`는 **바이트가 아니라 문자**로
해석되어 비교가 무의미해진다.

다중집합 비교를 쓰는 이유는 **"문자가 새로 생겼는가"와 "기존 문자가 이동했는가"를 구분**하기
위해서다. 단순 라인 카운트로는 `mcp_tool_test.go`·`convert.go`·`stop.go` 3개가 증가한 것처럼
보이지만, 이는 구조체 필드 정렬로 비ASCII를 포함한 줄이 재배치된 결과이며 새 문자는 없다.
따라서 V15가 `json_expr.go` 하나만 검사해도 충분하다.

§9 V15에 검증 항목을 둔다.

나머지 2건은 무해하다:
- `bin-common-handler/pkg/databasehandler/convert.go` — 코드블록 들여쓰기 재구성
- `bin-transcribe-manager/pkg/transcribehandler/stop.go` — 빈 주석 줄 삽입

**build tag / `//go:` 지시자 변경은 0건**이다(diff 전체에서 `grep -cE '//go:|\+build'` → 0).
이 부분은 안전하다.

#### 6.3.3 멱등성

`gofmt(gofmt(f)) == gofmt(f)`가 313파일 전부에서 성립한다(json_expr.go 포함).
따라서 §8 R3의 완화책인 "머지 직전 재포맷"은 안전하다.

### 6.4 `scripts/check-test-conventions.sh` (신규)

`docs/conventions/testing.md`가 규정하지만 어떤 린터로도 잡히지 않는 3개 규칙을 강제한다.
**변경된 파일만 검사**하여 존량(D6)에 걸리지 않게 한다.

#### 6.4.1 Rule 1의 범위 축소 및 검사 단위 (실측에 따른 결정)

**검사 단위: 변경된 "파일"이 아니라 변경된 "라인"이다.**

초안은 변경된 파일 전체를 grep했다. 이 방식은 **이번 PR 자체를 실패시킨다.**
G3(gofmt 313파일)이 그 중 **테스트 파일 151개**를 변경 목록에 올리는데,
그 파일들은 브랜치가 작성하지 않은 존량 위반을 이미 갖고 있다. 실측:

| 규칙 | 위반 | 파일 |
|---|---|---|
| Rule 1 (`TestXxx_Case`) | 122 | 20 |
| Rule 2 (testify) | 2 | 2 |
| Rule 3 (`ctrl :=`) | 54 | 4 |
| **합계** | **178** | — |

게이트는 `when:` 없이 무조건 실행되므로 첫 파이프라인에서 확정적으로 적색이 된다.
D6("존량은 범위 밖")과 G3+G4 동시 진행이 정면으로 충돌한다.

**해결: `git diff -U0`의 추가(`+`) 라인만 검사한다.** 실측으로 확인했다.

| 입력 | 기존(파일 단위) | 변경(라인 단위) |
|---|---|---|
| gofmt 전용 변경 | **178건 위반, exit 1** | **0건, exit 0** |
| 실제 위반 신규 추가 | 검출 | **3종 전부 검출, exit 1** |

gofmt는 함수 시그니처 줄이나 import 줄 자체를 바꾸지 않으므로, 포맷 전용 변경은
추가 라인에 위반 패턴을 만들지 않는다. "브랜치가 실제로 쓴 것만 본다"는
게이트의 본래 의도에 오히려 더 부합한다.

이로써 D6(존량 미수정)과 G4(신규 차단)가 양립한다. **선택지 1·2(포맷 범위 축소, PR 분리)는
채택하지 않는다.** 전자는 V1과 모순되고 후자는 D1(단일 PR)을 깬다.

**이 방식의 한계 (알고 채택한다):**

라인 단위 검사는 다음 두 경우를 놓친다. 실측으로 확인했다.

| 우회 경로 | 동작 | 빈도 |
|---|---|---|
| 기존 위반 함수의 **본문만** 수정 | 함수 선언 줄이 `+`에 없으므로 통과 | 상시 가능 |
| 위반 함수가 있는 파일을 **rename** | `git diff -U0`이 rename을 유사도로 압축해 `+` 라인을 내지 않음 | 최근 180일 `*_test.go` rename **6건** |

둘 다 **G4를 손상시키지 않는다고 판단한다.** 전자는 새 위반을 만드는 것이 아니라 존량을
건드리는 것이며 D6과 일관된다. 후자는 빈도가 낮고 우회하려면 의도적이어야 한다.
반대로 파일 **분할**은 새 경로가 추가(`A`)로 잡혀 존량 위반이 검출되는데,
이는 "새 파일에는 규약을 지켜라"는 게이트 취지에 부합하므로 그대로 둔다.

존량 정리는 애초에 이번 범위가 아니다(D6). 게이트의 목적은 **유입 차단**이지 소급 청소가 아니다.

**Rule 1의 매칭 범위:**

초안의 정규식 `^func Test[A-Z]`는 저장소 전역에서 **1,755건**을 매치한다. 분류하면:

| 분류 | 건수 | 판정 |
|---|---|---|
| `Test_<Method>` 정통 | 5,433 | 통과 |
| `TestXxx_Case` (밑줄 있음) | **626** | **위반 — 게이트 대상** |
| `TestXxx` (밑줄 없음) | 1,098 | **규정 없음 — 게이트 대상 아님** |
| `TestMain` | 31 | Go 표준 엔트리포인트 — 정당 |

`TestXxx` 1,098건에는 `TestFieldConstants`(46), `TestEventTypeConstants`(43),
`TestGoldenRoutingKeys`(26)처럼 **메서드가 아닌 대상(상수군·골든파일·부트스트랩)을 검증하는 테스트**가 다수 포함된다.
`testing.md` §13.6은 `Test_<MethodName>`만 규정하며, 이런 비메서드 테스트의 표기를 정의하지 않는다
(`grep -i 'constant|golden|TestMain|non-method|package-level' docs/conventions/testing.md` → **0건**).

**결정(대표님 확정): Rule 1은 `TestXxx_Case` 형태만 잡는다.**
문서가 규정하지 않은 것을 게이트가 강제해서는 안 된다. `TestXxx` 표기의 정통성 여부는
testing.md 개정이 선행되어야 하므로 이번 범위 밖이다(§4 Non-goals).

이 축소로 `TestMain` 예외 처리도 자동으로 불필요해진다(`TestMain`에는 밑줄이 없다).

#### 6.4.2 스크립트

```bash
#!/usr/bin/env bash
#
# Enforces the test conventions declared in docs/conventions/testing.md
# that no Go linter can express. Only inspects files changed relative to
# the merge base with main, so the existing backlog does not fail the build.
#
# Fail-closed: if the merge base cannot be resolved the script exits non-zero.
# The CI job is responsible for fetching origin/main before running this
# (CircleCI's checkout only fetches the current branch's refspec).
#
set -uo pipefail

BASE_REF="origin/main"

if ! MERGE_BASE="$(git merge-base "${BASE_REF}" HEAD 2>/dev/null)" || [ -z "${MERGE_BASE}" ]; then
  echo "check-test-conventions: FAILED to resolve merge base with ${BASE_REF}." >&2
  echo "" >&2
  echo "The job must fetch the base branch before running this script:" >&2
  echo "  git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'" >&2
  echo "" >&2
  echo "Refusing to pass silently: a skipped gate is indistinguishable from" >&2
  echo "a passing one, which is how conventions drift in the first place." >&2
  exit 1
fi

# Resolve the changed test files. `git diff` failure must NOT be swallowed:
# a shallow clone can resolve the merge base and still fail to diff it, and an
# empty array is indistinguishable from "nothing changed". Capture the status
# explicitly instead of relying on mapfile's, which reflects the redirect and
# is always 0.
#
# core.quotePath=false keeps non-ASCII paths usable; git would otherwise emit
# them octal-escaped and quoted, and the name would not resolve on disk.
if ! DIFF_OUT="$(git -c core.quotePath=false diff --name-only --diff-filter=d "${MERGE_BASE}" HEAD -- '*_test.go')"; then
  echo "check-test-conventions: FAILED to diff ${MERGE_BASE}..HEAD." >&2
  echo "The repository may be a shallow clone missing the base commit's objects." >&2
  exit 1
fi

CHANGED=()
while IFS= read -r line; do
  [ -n "${line}" ] || continue
  case "${line}" in vendor/*|*/vendor/*) continue ;; esac
  CHANGED+=("${line}")
done <<< "${DIFF_OUT}"

if [ "${#CHANGED[@]}" -eq 0 ]; then
  echo "check-test-conventions: no changed test files."
  exit 0
fi

# Inspect ADDED LINES ONLY, not whole files.
#
# Checking whole files would make any repo-wide reformat fail this gate: a
# gofmt-only pass rewrites 151 test files, and those files carry 178
# pre-existing violations that the branch never introduced. Scoping to added
# lines keeps the gate on what the branch actually wrote, which is what
# "changed files only" was meant to express in the first place.
#
# -U0 emits no context lines, so every '+' line is genuinely new.
# vendor/ is excluded here too: the pathspec, not the CHANGED array, is what
# bounds this diff. Filtering only the file list would let a vendored test
# file's added lines reach the rules.
if ! DIFF_U0="$(git -c core.quotePath=false diff -U0 "${MERGE_BASE}" HEAD \
  -- '*_test.go' ':(exclude)vendor/**' ':(exclude)*/vendor/**')"; then
  echo "check-test-conventions: FAILED to produce a unified diff." >&2
  exit 1
fi

fail=0

report() {
  # $1 = rule label, $2 = doc anchor, $3 = matches
  echo ""
  echo "✖ ${1}"
  echo "  Convention: docs/conventions/testing.md${2}"
  echo "${3}" | sed 's/^/    /'
  fail=1
}

# Report added-line violations with file:line. `git diff -U0` hunk headers
# (@@ -a,b +c,d @@) carry the new-file line number, so walk the diff and keep
# a running counter; a bare grep over added lines would lose the location.
scan() {
  # $1 = ERE to match against added lines.
  # Note the patterns below are POSIX EREs as awk understands them: no \b, no
  # \<, no \s. awk warns about (and ignores) unknown escapes, which silently
  # disables a rule -- Rule 3 was lost this way during review.
  awk -v pat="$1" '
    /^\+\+\+ b\// { file = substr($0, 7); next }
    /^@@ / {
      # @@ -old,cnt +new,cnt @@
      split($3, a, ",")
      line = a[1]; sub(/^\+/, "", line)
      next
    }
    /^\+/ {
      body = substr($0, 2)
      if (body ~ pat) printf "%s:%d:%s\n", file, line, body
      line++
    }
  ' <<< "${DIFF_U0}"
}

# Rule 1 — Test_<MethodName>, not TestXxx_Case. See 6.4.1 for why bare
# TestXxx (no underscore) is deliberately NOT matched.
m="$(scan '^func Test[A-Z][A-Za-z0-9]*_')"
[ -n "${m}" ] && report \
  "Test function must be named Test_<MethodName> (got TestXxx_Case)." \
  " (13.6 Test Function Naming)" "${m}"

# Rule 2 — assertions use reflect.DeepEqual + t.Errorf, not testify.
m="$(scan '"github[.]com/stretchr/testify')"
[ -n "${m}" ] && report \
  "testify is not used in this repository; use reflect.DeepEqual + t.Errorf." \
  " (13.5 Assertion Pattern)" "${m}"

# Rule 3 — the gomock controller variable is named mc.
m="$(scan '(^|[^A-Za-z0-9_])ctrl[ \t]*:=[ \t]*gomock[.]NewController')"
[ -n "${m}" ] && report \
  "Name the gomock controller 'mc' (mc := gomock.NewController(t))." \
  " (13.3 Test Structure Conventions)" "${m}"

if [ "${fail}" -ne 0 ]; then
  echo ""
  echo "Test convention check failed. These rules are documented in"
  echo "docs/conventions/testing.md and are enforced only on files this"
  echo "branch changes; pre-existing violations elsewhere are untouched."
  exit 1
fi

echo "check-test-conventions: OK (${#CHANGED[@]} file(s) checked)"
```

#### 6.4.3 초안에서 수정된 결함

| # | 결함 | 수정 |
|---|---|---|
| 1 | `CIRCLE_MAIN_BRANCH`는 CircleCI 내장 변수가 아니다 (공식 변수 목록에 없음). 항상 `main`으로 폴백되는 죽은 표현 | 제거하고 `origin/main` 고정 |
| 2 | merge base 실패 시 `exit 0` (fail-open). CircleCI `checkout`은 현재 브랜치 refspec만 fetch하므로 **상시 무력화 가능** → G4가 미달성인데 달성된 것으로 기록됨 | **fail-closed(`exit 1`)로 전환** + job에 명시적 fetch step 추가(§6.2.1) |
| 3 | `${CHANGED}`를 비인용 변수로 전개하여 단어분할에 의존 | `mapfile` + `"${CHANGED[@]}"` 배열로 변경 |
| 4 | 파일이 1개일 때 `grep -n`이 파일명을 출력하지 않아 위반 위치를 알 수 없음 | `grep -H` 추가 |
| 5 | 마지막 줄 `(${CHANGED} checked)`가 파일 목록 전체를 개행 포함 출력 | `${#CHANGED[@]}` 개수로 변경 |
| 6 | Rule 3의 `mockCtrl` 분기가 사문 (저장소 내 **0건**) | 제거 |
| 7 | Rule 1이 `TestMain` 31건과 비메서드 테스트 1,098건을 오탐 | §6.4.1대로 `TestXxx_Case`만 매치하도록 축소 |
| 8 | `mapfile -t CHANGED < <(... \|\| true)`가 **fail-open을 되살림**. `mapfile`의 종료 상태는 프로세스 치환 내부 파이프라인이 아니라 리다이렉트 결과라 항상 0이다. `git diff`가 실패해도(shallow clone에서 merge base는 풀렸으나 objects 미확보 등) 배열이 비어 `exit 0`으로 조용히 통과 | `DIFF_OUT="$(git diff ...)"` 로 분리하여 상태를 명시 검사. `\|\| true`는 `set -e`가 없어 애초에 죽은 표현이었다 |
| 9 | `mapfile`은 bash 4+ 전용이며 CI 이미지의 bash 버전을 확인하지 못함 | `while IFS= read -r` 루프로 대체하여 의존 제거. vendor 필터도 `case` 문으로 옮겨 `grep` 프로세스 하나를 줄였다 |

**수정된 스크립트는 실행으로 검증했다** (§6.4.4).

참고로 §10 Q6이 예고했던 수정안 `grep -vE '^func TestMain\('`은 **동작하지 않는다**.
`grep -n`은 다중 파일에서 `경로:행번호:본문`을 출력하므로 `^func` 앵커가 결코 매치되지 않는다
(실행 확인). Rule 1 축소로 이 필터 자체가 불필요해졌다.

설계 근거:

- **정규식 `^func Test[A-Z][A-Za-z0-9]*_`** — 정통 `Test_Create`는 `Test` 뒤가 `_`이므로 매치되지 않는다.
  위반 `TestCreate_Success`는 `Test` 뒤가 대문자이고 이후 `_`가 있어 매치된다.
  밑줄 없는 `TestCreate`·`TestMain`은 의도적으로 매치하지 않는다(§6.4.1).
  실제 정통 파일 100개(bin-call/flow/queue-manager)에 강제 실행하여 **오탐 0건**을 확인했다.
- **`--diff-filter=d`** — 삭제된 파일을 grep 대상에서 제외한다 (파일 부재로 인한 오류 방지).
- **merge base 해석 실패 시 exit 1 (fail-closed)** — 초안은 exit 0이었으나 철회했다.
  CircleCI `checkout`은 현재 브랜치의 refspec만 fetch하므로 `origin/main`이 없을 수 있고,
  그 경우 게이트가 **상시 조용히 통과**하여 G4가 미달성인 채 달성된 것으로 기록된다.
  이는 현상 유지가 아니라 잘못된 보증이다. 대신 job에 명시적 fetch step을 둔다(§6.2.1).
- **`set -uo pipefail`이며 `-e`는 쓰지 않는다** — grep이 매치 없을 때 exit 1을 반환하므로
  `-e`를 쓰면 정상 경로에서 스크립트가 죽는다.

#### 6.4.4 스크립트 실행 검증 (설계 단계에서 완료)

임시 git 저장소를 만들어 실제로 실행한 결과:

| 케이스 | 입력 | 결과 |
|---|---|---|
| 오탐 검사 | `TestMain` + `TestFieldConstants` + `Test_Get` | **OK, exit 0** |
| 위반 검출 | `TestCreate_EmptyName` + testify import + `ctrl :=` | **3종 전부 검출, exit 1** |
| 단일 파일 리포트 | 변경 파일 1개 | `pkg/d_test.go:3:` — **파일명 출력됨** |
| fail-closed | `origin/main` ref 삭제 | **exit 1 + 조치 안내 출력** |
| 실저장소 오탐 | 정통 테스트 파일 100개 | **Rule 1/2/3 전부 0건** |

**라운드 3 수정(§6.4.3 #8·#9) 후 재검증 (`while read` 방식):**

| 케이스 | 결과 |
|---|---|
| 위반 3종 검출 | **3건 전부, 파일명·행번호 정상, exit 1** |
| 정통 파일 오탐 (`TestMain`·`TestGoldenRoutingKeys`) | **OK (1 file(s) checked), exit 0** |
| merge base 해석 실패 | **exit 1 + 조치 안내** |

`mapfile` fail-open도 별도 재현으로 확인했다: `git diff`를 잘못된 ref로 실행해도
`mapfile`의 종료 상태는 **0**, 배열 길이 **0**, 스크립트 최종 **exit 0**이었다.
수정본은 이 경로에서 exit 1을 반환한다.

**라운드 4 수정(hunk 기반 전환) 후 재검증.**
실제 monorepo의 gofmt 미준수 테스트 파일 60개를 임시 저장소에 복사해 실행:

| 케이스 | 기대 | 결과 |
|---|---|---|
| **gofmt 전용 변경 (B1 시나리오)** | exit 0 | **OK (60 file(s) checked), exit 0** |
| 위반 3종 신규 추가 | exit 1 | **3종 전부, 행번호 정확(L5/L3/L6), exit 1** |
| 정통 표기(`TestMain`·`TestFieldConstants`·`TestGoldenRoutingKeys`) | exit 0 | **OK, exit 0** |
| vendor 경로 테스트 파일 | 무시 | **OK, exit 0** |
| merge base 해석 실패 | exit 1 | **exit 1 + 안내** |

이 과정에서 **추가 결함 2건을 자체 발견**했다(§12에 기록):
awk가 `\<`·`\.`를 무시해 **Rule 3이 조용히 비활성화**되어 있었고,
hunk 전환으로 **vendor 필터가 무력화**되어 있었다. 둘 다 수정 후 위 표로 재확인했다.
설계 리뷰나 코드 읽기로는 어느 쪽도 드러나지 않았다.

### 6.5 루트 `README.md`

#### (a) 서비스 표 누락 3건 보완

현재 표(L63-101)에 다음 3개 서비스가 없다. 알파벳 순 위치에 삽입한다.

| 삽입 위치 | 추가 행 |
|---|---|
| `bin-route-manager` 다음 (L89 뒤) | `| \`bin-schedule-manager\`     | Scheduled job execution                       |` |
| `bin-transcribe-manager` 다음 (L95 뒤) | `| \`bin-trigger-sender\`       | Scheduled trigger dispatch                    |` |
| `bin-tts-manager` 다음 (L97 뒤) | `| \`bin-webchat-manager\`      | Web chat widget backend                       |` |

> 각 행의 Purpose 문구는 해당 서비스의 `README.md` / `CLAUDE.md` 1줄 요약에서 도출한다.
> 구현 시 실제 파일을 읽어 확정하며, 위 문구는 잠정안이다.

역방향 확인 완료: 표에 있으나 실재하지 않는 "유령 항목"은 **0건**이다.

#### (b) 셀프호스팅 경로 현행화

현재 L145-146:

```
### Deploy to Kubernetes
You'll need a Kubernetes manifest or Helm chart per service. For now, these are maintained privately or in internal repositories — contact us if you'd like access to deployment blueprints.
```

**이 서술은 사실과 다르다.** 셀프호스팅 경로가 공개되어 있다:

| 저장소 | 상태 | 근거 |
|---|---|---|
| `voipbin/voipbin` `install/` | **현행 정본** | `voipbin/voipbin` README가 셀프호스팅 정본으로 명시. Docker Compose 단일 서버 방식 |
| `voipbin/install` (GCP/K8s) | **deprecated** | 해당 저장소 README: "This repository is deprecated... will be archived" |
| `voipbin/sandbox` | **deprecated** | 해당 저장소 README: 동일 문구, `voipbin/voipbin` `install/`로 이관 |

변경안:

```
### Deploy

The supported self-hosting path is the single-server Docker Compose installer in
[voipbin/voipbin](https://github.com/voipbin/voipbin/tree/main/install). It brings up the full
platform (backend services, SIP/media infrastructure, and the web applications) on one host.

This monorepo contains the backend service sources only. It is built and published as container
images that the installer consumes; it is not deployed directly from a checkout of this repository.
```

브랜드 규칙 준수 확인: em/en 대시 미사용, "opensource" 한 단어, 타사 언급 없음, 개인 연락처 노출 없음.

#### (c) 기존 em dash 11건 정리 (대표님 확정)

README에 em dash가 **11건** 존재한다: L3, 21, 39, 45, 52, 53, 54, 55, 110, 146, 153.
(b)가 손대는 것은 L146 하나뿐이므로 나머지 10건이 남는다.

**결정: 11건 전부 정리한다.** 작성 규칙상 em/en 대시는 금지이며,
일부만 고치면 규칙 준수 여부가 파일 내에서 일관되지 않는다.

치환 방침: 마침표·쉼표·괄호·콜론으로 대체하며 문장 의미를 바꾸지 않는다. 예시:

```
- 3:   ...monorepo — it contains backend services...
+ 3:   ...monorepo. It contains backend services...

- 52:  🔧 [Admin Console](...) — Manage everything visually
+ 52:  🔧 [Admin Console](...): Manage everything visually
```

아울러 L3·L23의 `open-source`(하이픈 표기)를 `opensource` 한 단어로 통일한다.

#### (d) L110 "Kubernetes-based deployments" 서술 수정 (대표님 확정)

현재 L110:

```
> It's a platform composed of multiple microservices, SIP/media infrastructure
> (e.g., Asterisk, RTPEngine), Kubernetes-based deployments, and pluggable
> third-party integrations ... You don't just "run it" — you assemble and deploy it
> based on your architecture.
```

**(b)에서 셀프호스팅 정본을 Docker Compose 단일 서버로 현행화하는데, 이 문장은 K8s를 전제한다.**
같은 문서 안에서 배포 모델이 충돌하므로 함께 고친다.
또한 "you assemble and deploy it"은 설치 관리자가 제공되는 현 상태와 맞지 않는다.

변경안:

```
> It's a platform composed of multiple microservices, SIP/media infrastructure
> (e.g., Asterisk, RTPEngine), and pluggable third-party integrations for telephony,
> AI, and other backends. The installer brings these up for you on a single host;
> larger deployments can split the services across hosts.
```

> 이 문구는 잠정안이다. 구현 시 `voipbin/voipbin` README의 현행 서술과 대조하여 확정한다.

---

## 7. 왜 "컨벤션 문서 추가"를 하지 않는가

당초 계획에는 "CLAUDE.md에 테스트 7규칙 명문화"가 포함되어 있었으나 **철회**한다.

- §1.3에서 확인했듯 `docs/conventions/testing.md`가 7규칙을 이미 정확히 규정하고 있다.
- 그럼에도 599건의 위반이 발생했다. 즉 **문서화는 이미 시도되었고 실패한 대책**이다.
- 동일한 내용을 다른 파일에 한 번 더 쓰는 것은 중복 서술을 늘리고, 두 사본이 갈라질 위험만 만든다.
- 올바른 대책은 §6.4의 기계적 게이트다.

루트 CLAUDE.md에는 규칙 본문을 복제하지 않고, **§6.4 게이트의 존재만** 1줄 링크로 안내한다.

---

## 8. Rollout / risk

| # | 리스크 | 가능성 | 영향 | 완화 |
|---|---|---|---|---|
| R1 | CI에서 golangci-lint가 여전히 OOM | 낮 | lint job 실패 | §6.1.2 실측으로 `GOGC=50` 채택(2.40→1.66 GB). 실패 시 `GOGC=20`(1.44 GB)으로 강화하거나, §11의 `enable-lint` 파라미터를 `false`로 되돌려 즉시 무력화 |
| R2 | golangci-lint 버전이 go.mod의 go directive보다 낮게 빌드되어 실행 거부 | **발생 확인됨** | **전 job 실패** | §6.2.2에서 실측. v2.5.0 → v2.14.0으로 교체하고 실제 `run` 성공 확인. 핀 변경 이유를 config 주석에 명시하여 재발 방지 |
| R2b | 향후 go directive를 1.28+로 올릴 때 lint 핀을 함께 올리지 않아 전 job 실패 | 중 | 전 job 실패 | config 주석에 "go directive를 올리면 이 핀도 올릴 것" 명시. 근본 해결은 아니나 다음 작업자에게 원인을 즉시 알려준다 |
| R2c | 릴리스 tarball URL/체크섬 파일명 규칙이 상류에서 바뀌면 설치 step 실패 | 낮 | 전 job 실패 | 설치 실패는 즉시 빨간불로 드러나며 조용한 오작동이 아니다. 버전 일치 가드가 추가 방어선 |
| R3 | gofmt 313파일 일괄 변경이 진행 중인 다른 PR과 충돌 | 높 | 머지 충돌 | **"의미 변화 없음" 근거는 철회**(§6.3.1: 비공백 변경 98%, 그중 54파일이 import 재정렬). import 블록은 기능 PR이 거의 반드시 건드리는 지점이라 자동 해소가 안 될 수 있다. 완화: ① 교집합 브랜치 소유자에게 사전 통지, ② 머지 순서 조정, ③ 머지 직전 재포맷(gofmt 멱등성 확인됨, §6.3.3) |
| R3b | 살아있는 worktree와의 실제 교집합 | 확인됨 | 머지 충돌 | 실측 5파일 — `pr1285-review` 4건(bin-ai-manager 3, aicallhandler 1), `VOIP-1444` 1건(bin-campaign-manager/pkg/dbhandler/main.go). 구현 시 재확인 후 해당 브랜치 소유자에게 통지 |
| R4 | `check-test-conventions.sh`가 CI에서 merge base 해석 실패 | 중 | **게이트가 상시 무력화** | **fail-open → fail-closed로 전환**(§6.4.3). CircleCI checkout이 `origin/main`을 보장하지 않으므로 job에 명시적 fetch step 추가(§6.2.1). 조용한 통과는 "현상 유지"가 아니라 G4 미달성을 달성으로 오기록하는 것이므로 허용하지 않는다 |
| ~~R5~~ | ~~`go vet` 복구로 신규 위반 검출~~ | — | — | **해소됨**: §6.2.3에서 별도 `go vet` step을 삭제(govet이 golangci-lint에 이미 포함, 검출 결과 동일 확인) |
| R6 | 313파일 포맷 변경이 diff를 키워 리뷰 부담 | 높 | 리뷰 품질 저하 | 포맷 커밋을 **별도 커밋으로 분리**(PR은 단일 유지, D1). **추가 필수 조치**: 비공백 변경 57파일 목록을 PR 본문에 첨부하여 리뷰어가 순수 정렬 변경을 건너뛸 수 있게 한다(§6.3.1). 이 목록이 없으면 §6.3.2의 주석 손상류 변경이 통과한다 |
| R7 | README 서비스 설명 문구가 실제 서비스 역할과 불일치 | 낮 | 문서 오류 | 구현 시 각 서비스 README/CLAUDE.md에서 직접 인용 |
| R8 | 새 lint step이 일부 서비스에서 한 번도 실행되지 않은 채 머지됨 | 중 | 머지 후 첫 변경 시 실패 | `bin-email-manager`·`bin-sentinel-manager`는 gofmt 변경이 없어 이번 PR에서 job이 트리거되지 않는다(실측). 구현 시 두 서비스에 대해 **로컬에서 `golangci-lint run`을 선실행**하여 위반 0을 확인한다 |
| R9 | `.golangci.yml` 단독 변경 PR이 어떤 lint job도 트리거하지 않음 | 중 | 튜닝 PR이 미검증 머지 | §6.2.1.1에서 `run-lint-config-check` 파라미터·workflow·`lint-config-check` job을 **신설**하고 mapping 1행을 추가하여 해소. 초안의 `scripts/.* → run-shell-tests` 매핑은 해당 job이 bats만 돌리므로 근거가 되지 못했다(철회). 게이트 스크립트 쪽은 §6.2.1에서 **무조건 실행**으로 등록되므로 트리거가 불필요하다 |
| R10 | `GOMAXPROCS`가 CI에서 호스트 코어 수를 보아 측정 조건과 달라짐 | 중 | 메모리 초과 | Linting step environment에 `GOMAXPROCS: "2"` 명시(§6.2 변경점 5) |
| R11 | **pre-commit hook이 gofmt 일괄 커밋을 거부** | 높 | 구현 착수 즉시 막힘 | 실측: `core.hooksPath`가 설정되어 있고 hook이 활성이다. gofmt가 `models/*/webhook.go` 6개를 건드리면 "WebhookMessage model changed without RST documentation update"로 커밋이 거부된다. **이 변경은 포맷 전용이므로 RST 갱신 대상이 아니다.** 포맷 커밋에 한해 `--no-verify`를 사용하고, PR 본문에 사유(포맷 전용, 필드 변경 없음)를 명시한다. `git diff --stat`으로 webhook.go 변경이 공백뿐임을 함께 첨부한다 |
| R12 | lint가 영구히 적용되지 않는 서비스가 존재 | 낮 | 사각지대 | `bin-openapi-manager`(Go 2파일)와 `voip-asterisk-proxy`(21파일)는 **`go-test` command 호출부 자체가 없다**(호출 37 = go-test 35 + api 1 + pipecat 1, 이 둘은 목록에 없음). §6.2의 "3곳 수정으로 전량 반영"이 커버하지 못한다. 이번 범위에서는 손대지 않고 §4 Non-goals로 기록한다. 두 서비스에 test job을 신설하는 것은 별개 과제다 |
| R13 | **shallow clone이 merge base 계산을 끊어 게이트가 무관한 PR을 죽임** | 높 | 머지 후 전면 발현 | 초안의 `--depth=200`은 저장소를 shallow로 전환한다. 실측: base 커밋 객체는 존재(`git cat-file -t` → `commit`)하는데 `git merge-base`는 **exit 1**이다. 게이트가 fail-closed·무조건 실행이므로 base가 fetch 윈도를 벗어난 브랜치는 **위반 0건이어도 CI 적색**이다. 현재 살아있는 원격 브랜치 23개(main 제외) 중 **21개가 200커밋 이상 뒤져** 있고 main은 월 ~121커밋으로 움직여, 약 7주만 지나면 걸린다. **조치: `--depth` 제거 + merge base 해석 실패 시에만 deepen**(§6.2.1). `--depth` 제거만으로는 checkout 자체가 shallow인 경우를 못 막는다는 점도 실측 확인했다. **V8b(CI 로그 확인)로는 걸러지지 않는다** — 이번 PR의 base는 main 팁 바로 아래라 그냥 통과하기 때문이다. 이 결함만을 겨냥해 V8c를 신설했다 |

Risk: None이 아니다. R2(발생 확인됨)·R3·R4가 실질 리스크이며, R2는 이미 설계에서 해소했다.

---

## 9. Verification plan

| # | 항목 | 명령 | 통과 기준 |
|---|---|---|---|
| V1 | gofmt 잔여 0 | `gofmt -l bin-*/ voip-*/ \| grep -v /vendor/ \| wc -l` | `0` |
| V2 | 표준 린터 위반 0 | 서비스별 `golangci-lint run --timeout 10m --max-issues-per-linter 0 --max-same-issues 0 ./...` | `0 issues` |
| V2b | **미트리거 서비스 선검증** (R8) | `bin-email-manager`·`bin-sentinel-manager`에서 `golangci-lint run` | `0 issues` |
| ~~V3~~ | ~~`go vet` 전 서비스 통과~~ | — | **삭제**: §6.2.3에서 별도 `go vet` step을 제거했으므로 불필요 (govet은 V2에 포함) |
| V4 | 전 서비스 테스트 통과 | 서비스별 `go test ./...` | PASS |
| V5 | **CI 버전으로 실제 실행** (스키마 검증만으로는 불충분) | 핀한 버전을 설치 후 대표 서비스에서 `golangci-lint run` | 정상 종료. **`config verify` 통과만으로 합격 처리하지 않는다** (§6.2.2 발견 2) |
| V5b | 설치 체크섬 검증 | `sha256sum -c` | OK |
| V5c | 설치 후 버전 일치 가드 동작 | 의도적으로 다른 버전을 PATH에 둔 뒤 step 실행 | exit 1로 실패 |
| V6 | 게이트 스크립트 오탐 없음 | 정통 파일 100개(bin-call/flow/queue-manager)를 대상으로 강제 실행 | 위반 0 **(완료: §6.4.4)** |
| V7 | 게이트 스크립트 검출 동작 | 위반 3종을 담은 임시 저장소로 실행 | 3건 모두 검출, exit 1 **(완료: §6.4.4)** |
| V8 | **게이트 fail-closed 동작** | `origin/main` ref 삭제 후 실행 | **exit 1** + 조치 안내 출력 **(완료: §6.4.4)** |
| V8b | **게이트가 CI에서 실제로 merge base를 해석했는지** | CI 로그에서 `check-test-conventions: OK (N file(s) checked)` 확인 | N ≥ 1이며 skip/실패 메시지가 아님. **이 확인 없이는 G4 달성으로 간주하지 않는다** |
| V8c | **오래된 base 브랜치에서도 merge base가 풀리는지** (§8 R13) | 임시 저장소에서 base를 main보다 250커밋 뒤로 둔 뒤 fetch step을 그대로 실행 | `git merge-base` exit 0. **V8b로는 이 결함이 잡히지 않는다** — 이번 PR의 base는 main 팁 바로 아래라 어떤 fetch 방식이든 통과하기 때문이다. shallow clone 상태에서도 deepen 후 exit 0이어야 한다. **재현 시 `git clone --depth=1`의 대상은 반드시 `file://` URL이어야 한다** — 로컬 경로로 clone하면 git이 `--depth`를 조용히 무시해(`warning: --depth is ignored in local clones`) shallow가 아닌 저장소가 만들어지고, 검증이 통과한 것처럼 보인다 |
| V9 | README 서비스 표 완전성 | `for d in bin-*/ voip-*/; do grep -q "\`${d%/}\`" README.md \|\| echo MISSING $d; done` | 출력 없음 |
| V10 | README 역방향(유령 항목) | 표의 각 항목에 대응 디렉터리 존재 확인 | 전부 존재 |
| V11 | 브랜드 규칙 — em/en 대시 | `grep -cE '—\|–' README.md` | `0` (§6.5(c)에 따라 11건 전부 정리) |
| V11b | 브랜드 규칙 — opensource 표기 | `grep -ciE 'open.source' README.md` 결과 중 `open-source`/`open source` | `0` (전부 `opensource` 한 단어) |
| V11c | 타사 언급 없음 | `grep -niE 'twilio\|vonage\|plivo\|messagebird\|fonoster' README.md` | 출력 없음 |
| V11d | 배포 서술 일관성 (§6.5(d)) | `grep -n 'Kubernetes' README.md` | 셀프호스팅 정본(Docker Compose)과 모순되는 서술이 없음 |
| V12 | CI 설정 전개 검증 | `circleci config process .circleci/config_work.yml > /dev/null` | 성공. **`yaml.safe_load`로 대체하지 않는다** — 그것은 파싱만 볼 뿐 `when:` 블록·파라미터 참조·미선언 파라미터(§6.2.1.1)를 잡지 못한다. 이 환경에는 `circleci` CLI가 없으므로 구현자가 설치해야 한다 |
| V12b | 파이프라인 파라미터 선언 확인 | `grep -c 'run-lint-config-check' .circleci/config_work.yml` | `≥ 1`. mapping이 참조하는 파라미터가 선언되어 있지 않으면 continuation이 통째로 실패한다 |
| V13 | 주석 잔여 확인 | `! grep -q 'Re-enable golangci-lint' .circleci/config_work.yml` | exit 0 (매치 없음) |
| V14 | lint step 반영 범위 | `grep -cE '^[[:space:]]+name: Linting' .circleci/config_work.yml` | `3` (commands 정의 3개). **주석을 세지 않는 패턴이어야 한다** — `grep -c 'name: Linting'`은 현재도 3(전부 `#     name: Linting and vet`)이라 주석 해제 여부를 판별하지 못한다. 실측: 현재 비주석 매치 `0` |
| V14b | **롤백 파라미터 반영 범위** (§11.2) | `grep -c 'enable-lint:' .circleci/config_work.yml` | `3` (선언만 셈. `enable-lint` 로 세면 참조 `<< parameters.enable-lint >>`까지 포함되어 6이 나오므로 판정 불가) |
| V15 | **gofmt 주석 손상 확인** (§6.3.2) | `grep -c '”' bin-call-manager/pkg/dbhandler/json_expr.go` | `0` (스마트쿼트가 도입되지 않음) |
| V15b | **게이트가 이 PR 자체를 통과하는지** (§6.4.1 B1) | 브랜치에서 `bash scripts/check-test-conventions.sh` | **exit 0.** gofmt가 테스트 파일 151개를 변경 목록에 올리고 그 안에 존량 위반 178건이 있으므로, 파일 단위 검사였다면 반드시 실패한다. 이 검증 없이 머지하지 않는다 |
| V15c | 게이트가 신규 위반은 여전히 잡는지 | 임시 커밋으로 위반 3종을 추가한 뒤 실행 | 3종 전부 검출, exit 1. 확인 후 임시 커밋 폐기 |
| V16 | **PR 브랜치 CI에서 lint step 실제 통과** | PR 생성 후 CircleCI 결과 확인 | 새 Linting step이 초록. path-filtering setup workflow도 PR 브랜치의 `config_work.yml`을 읽으므로 확인 가능 |

---

## 10. Open questions

**Q1 — 해결됨 (§6.1.2).** `concurrency`는 메모리를 줄이지 못함이 실측되었고(-1.8%),
`GOGC=50` + `GOMEMLIMIT=3GiB`로 2.40 GB → 1.66 GB(-31%)를 달성했다. D4(상향 없음) 유지 확정.

**Q2 — 해결됨 (§6.2.1).** 게이트를 독립 job으로 분리해 37회 중복 실행을 제거했다.

**Q3 — 해결됨 (§8 R6).** 포맷 커밋 분리만으로는 부족하다.
비공백 변경 57파일 목록을 PR 본문에 첨부하는 것을 필수 조치로 승격했다.

**Q4 — 해소됨 (§6.2.3).** 별도 `go vet` step 자체를 삭제했으므로 질문이 성립하지 않는다.

**Q5 — 해결됨 (§6.2.2).** v2.5.0은 스키마는 통과하나 **실행이 불가능**했다(go1.25 빌드 vs 1.27.1 타깃).
v2.14.0(go1.27.0 빌드)으로 교체하여 실제 `run` 성공을 확인했다. `install.sh`도 버그가 있어 제거했다.

**Q6 — 해소됨 (§6.4.1).** Rule 1을 `TestXxx_Case`만 매치하도록 축소하여 `TestMain` 오탐이 사라졌다.
예고했던 필터 `grep -vE '^func TestMain\('`은 애초에 동작하지 않는 코드였다(§6.4.3).

**Q7 — 해결됨 (§6.5(c), 대표님 확정).** README의 em dash 11건을 **전부 정리**한다.
`open-source`(하이픈) 2건도 `opensource` 한 단어로 통일한다.

**Q8 — 해결됨 (§6.5(d), 대표님 확정).** L110의 "Kubernetes-based deployments" 서술을
같은 PR에서 수정한다. 셀프호스팅 정본 변경과 같은 문서 안에서 충돌하기 때문이다.

남은 질문: 없음. 구현 착수 가능 상태다.

---

## 11. 단계적 롤아웃과 롤백

### 11.1 Blast radius

`commands:` 정의 3곳 수정이 **37개 job에 즉시 전파**된다. 이는 도입 효율의 근거인 동시에
**실패 전파 경로**이기도 하다. `.golangci.yml` 파싱 오류 1건이면 37개 job이 동시에 적색이 된다.

§6.2.2에서 실제로 그런 상태(v2.5.0 + Go 1.27.1 타깃)를 발견했으므로 가상의 우려가 아니다.

### 11.2 `enable-lint` 파라미터 (채택)

`go-test` 계열 command에 boolean 파라미터를 두어 job 단위로 lint를 끌 수 있게 한다.

```yaml
  go-test:
    parameters:
      source-directory:
        type: string
      enable-lint:
        type: boolean
        default: true
    steps:
      # ... 기존 steps ...
      - when:
          condition: << parameters.enable-lint >>
          steps:
            - run:
                name: Install golangci-lint
                # ...
            - run:
                name: Linting
                # ...
```

채택 이유:

- **롤백 비용이 1줄로 낮아진다.** R1(OOM)이 실제로 터지면 `default: true` → `false` 한 줄로
  전체를 즉시 무력화할 수 있고, 313파일 포맷을 되돌릴 필요가 없다.
- **문제 서비스만 격리 가능하다.** 특정 서비스만 지속 실패하면 그 job에서 `enable-lint: false`를
  주어 나머지 36개의 강제는 유지한다.
- 비용은 YAML 파라미터 1개와 `when` 블록 1개뿐이다.

**반드시 3개 command 전부에 적용한다.** §5에서 확인했듯 command 정의는
`go-test`(35 호출) / `go-test-api-manager`(1) / `go-test-pipecat-manager`(1) 3개이고,
§6.2의 lint step도 3곳에 동일 적용된다(V14 기준 = 3).
`go-test`에만 넣으면 **bin-api-manager와 bin-pipecat-manager가 롤백 불가**가 된다.
하필 `bin-api-manager`는 §6.1.2에서 피크 2.18~2.40 GB로 측정된 **유일한 OOM 후보**다.
롤백 수단이 정확히 가장 필요한 곳에서만 빠지는 구성이 되므로, V14b로 기계 확인한다.

**37개 호출부는 수정하지 않는다.** `default: true`가 있으므로 기존 호출부
(`config_work.yml:879-880` 등 `source-directory`만 전달)는 그대로 유효하다.
`when`은 command의 steps 안에서 쓸 수 있는 logic step이며 boolean 파라미터를 condition으로 받는다.

단계적 롤아웃(처음에 `default: false`로 두고 1~2개만 true)은 **채택하지 않는다.**
7개 서비스 실측에서 gofmt 외 위반이 0이고(§6.1), 가장 큰 미지수였던 OOM과 버전 호환성이
모두 실측으로 해소되었으므로, 점진 도입의 추가 정보 획득량이 적다.
대신 위 파라미터로 **되돌릴 수 있는 상태**를 확보하는 것으로 갈음한다.

### 11.3 Rollback plan

| 상황 | 조치 | 비고 |
|---|---|---|
| lint가 다수 서비스에서 실패/OOM | `enable-lint`의 `default`를 `false`로 변경 | 1줄. **포맷 변경은 유지됨** |
| 특정 서비스만 실패 | 해당 job에 `enable-lint: false` 전달 | 나머지 36개 강제 유지 |
| 테스트 컨벤션 게이트가 오탐 | `check-test-conventions` job을 workflow에서 제거 | 게이트는 독립 job이라 격리가 쉽다 |
| 전면 되돌리기 | `git revert <squash-sha>` | ⚠️ **313파일 포맷까지 전부 되돌아간다** |

**중요 비대칭:** squash merge 정책상 이 PR은 main에 단일 커밋으로 남는다.
따라서 `git revert`는 CI 변경과 포맷 변경을 분리하지 못한다.
"lint만 끄고 포맷은 유지"는 revert가 아니라 §11.2의 파라미터로 처리해야 한다.
이 점이 포맷 커밋을 별도 커밋으로 분리해야 하는(R6) 실질적 이유이기도 하다.

**롤백 판단 주체와 기준:** 머지 후 첫 파이프라인에서 3개 이상 서비스의 lint step이 실패하면
원인 분석 전에 먼저 `enable-lint: false`로 되돌리고, 원인을 규명한 뒤 재활성화한다.

---

## 12. Approval status

Draft — Design Review 루프 진행 중.

| 라운드 | 관점 | 판정 | 반영 |
|---|---|---|---|
| 1 | 사실 정확성 | CHANGES_REQUESTED | 반영 완료 (§1.1 `go generate` 부분강제 정정, §1.2 집계범위 정정, §6.4 스크립트 결함 7건 수정) |
| 2 | 운영 안전성·회귀 리스크 | CHANGES_REQUESTED | 반영 완료 (C1 goimports 제외, C2 주석 손상 문서화, C3 fail-closed 전환, M2 `go vet` 삭제, §11 롤백 신설) |
| 3 | 반영 검증·신규 결함 | CHANGES_REQUESTED | 반영 완료 (10건) |
| 4 | 반영 검증·CI 실행 가능성 | CHANGES_REQUESTED | 반영 완료 (B1 + M1~M3) |
| 5 | 반영 검증·게이트 환경 의존성 | CHANGES_REQUESTED | 반영 완료 |
| 6 | 반영 검증·fetch 안전성 | **APPROVED** | 비차단 지적 5건 반영 |
| 7 | 최종 확인·구현 준비도 | **APPROVED** | **2연속 APPROVE 달성. 설계 단계 종료** |

라운드 7에서 게이트 스크립트를 mawk로도 실행해 gawk 의존이 없음을 확인했고(CI `cimg/go`
이미지 대비), §6.1 스니펫과 `.golangci.yml`의 바이트 일치, §/R/V/G/D/Q 상호참조 전수 대조
(미정의 0건, 중복 0건)를 마쳤다. N5의 브랜치 수는 `git branch -r`이 `origin/HEAD` 별칭과
stale tracking ref를 포함한다는 지적을 받아 "살아있는 원격 브랜치 23개(main 제외) 중 21개"로
재정정했다. 하중을 지는 21은 어느 기준으로도 동일하다.

**구현 착수 조건 충족.** 남은 미검증 항목은 V12(`circleci config process`) 하나이며,
이 환경에 CLI가 없어 구현 시 설치해 실행해야 한다. 미선언 파라미터로 continuation 전체가
죽는 경로(§6.2.1.1)를 잡는 유일한 검증이므로 생략하지 않는다.

**라운드 6 조치 내역** (전부 비차단, 정확성 보강):

| # | 지적 | 조치 |
|---|---|---|
| N1 | §6.3.2 명령이 UTF-8 로캘에서 재현 불가(`[\x80-\xFF]`가 바이트가 아닌 문자로 해석) | `LC_ALL=C` 추가 + 생략 불가 사유 명시 |
| N2 | 게이트 스크립트의 testing.md 앵커 오기 | Rule 2 → 13.5 Assertion Pattern, Rule 3 → 13.3 Test Structure Conventions로 정정 |
| N3 | R13의 "§9의 어떤 항목으로도 걸러지지 않는다"가 V8c 신설로 낡음 | "V8b로는 걸러지지 않는다"로 정정 |
| N4 | V8c 재현 함정: 로컬 경로 clone은 `--depth`를 조용히 무시 | `file://` URL 필수임을 명시 |
| N5 | 브랜치 수 63 → 실제 64 | 정정(21개라는 핵심 수치는 일치) |

라운드 6에서 fetch step을 5개 환경(full clone / shallow clone / merge base 해석 불가 /
양쪽 fetch 실패 / 기존 `--depth=200`)에서 실행 검증했고 전부 의도대로 동작했다.

2회 연속 APPROVE 시 종료 (최대 20라운드). **현재 1회 연속.**

**라운드 5 조치 내역:**

| # | 지적 | 조치 |
|---|---|---|
| B1 | **`--depth=200` fetch가 저장소를 shallow로 만들어 merge base 계산을 끊는다.** base가 200커밋 이상 뒤진 브랜치(63개 중 21개)는 위반 0건이어도 CI 적색 | `--depth` 제거 + 실패 시에만 deepen(§6.2.1). R13 신설, V8c 신설. **3환경 전부 실측 검증**: 비shallow(1차 fetch로 충분) / shallow clone(unshallow 후 OK) / depth=200(재현된 실패) |
| M1 | hunk 방식의 한계를 문서가 인정하지 않음 | §6.4.1에 한계표 추가(본문만 수정·rename 통과). G4 손상 아님을 근거와 함께 명시 |
| M2 | §6.3.2 근거 파일 목록 부정확 | 바이트 다중집합 비교로 재측정. 결론(`json_expr.go` 1건)은 유지, 근거를 실행 명령과 함께 교체 |
| M3 | V14가 주석/비주석을 구분하지 못해 판정력 없음(변경 전후 모두 3) | `grep -cE '^[[:space:]]+name: Linting'`으로 교체. 실측 현재값 `0` |


**라운드 4 조치 내역:**

| # | 지적 | 조치 |
|---|---|---|
| B1 | **게이트가 이 PR 자체를 실패시킴.** gofmt가 테스트 151파일을 변경 목록에 올리고, 그 안에 존량 위반 178건이 있다 | 검사 단위를 **파일 → 추가 라인(`git diff -U0`)**으로 전환(§6.4.1). 실측: gofmt 전용 변경 178건→**0건**, 신규 위반은 여전히 3종 검출 |
| M1 | §6.2.1.1 결론과 예시 YAML 모순 | 철회된 `run-convention-scripts`·`convention-scripts`·`scripts/.*` 블록 삭제 |
| M2 | "L60의 두 번째 occurrence도 같은 형태" 부정확 | 실제 L58이며 **이미 코드블록**이라 gofmt 대상이 아님을 명시 |
| M3 | V14b 기준이 기계 판정 불가(`enable-lint` = 6) | `enable-lint:`(선언만) = `3`으로 고정 |

**라운드 4에서 자체 발견한 결함:**

| 결함 | 발견 경위 | 조치 |
|---|---|---|
| **pre-commit hook이 gofmt 커밋을 거부** | B1 재현 중 실제 커밋 시도에서 발생 | R11 신설. 포맷 전용 커밋에 `--no-verify` + 사유 명시 |
| **awk가 `\b`/`\<`/`\s`를 무시하여 Rule 3이 조용히 비활성화** | hunk 전환 후 실행 검증에서 Rule 3만 누락 | POSIX ERE로 교체(`(^\|[^A-Za-z0-9_])ctrl...`). 경고 소멸 및 검출 확인 |
| **hunk 전환으로 vendor 필터 무력화** | vendor 경로 테스트에서 위반 검출됨 | `git diff` pathspec에 `:(exclude)vendor/**` 추가 |

2회 연속 APPROVE 시 종료 (최대 20라운드).

### 설계 단계에서 실행 검증으로 차단한 결함

문서 리뷰만으로는 발견할 수 없었고 **실제 실행으로만 드러난** 항목들:

| 결함 | 발견 방법 | 머지 시 영향 |
|---|---|---|
| golangci-lint v2.5.0이 Go 1.27.1 타깃을 거부 | 실제 `run` 실행 (`config verify`는 통과했음) | **37개 job 전량 실패** |
| 공식 `install.sh`의 체크섬 오선택 + 실패 시 exit 0 | 실제 설치 시도 | 바이너리 없이 "성공", 버전 핀 무력화 |
| `concurrency`가 메모리를 줄이지 못함 (-1.8%) | 콜드 캐시 재측정 | OOM 미해결 상태로 배포 |
| `goimports`가 gofmt보다 13파일 더 잡음 | `goimports -l` 대조 | 6개 서비스 job 실패 |
| `gofmt`가 SQL 주석을 스마트쿼트로 치환 | 전체 diff 비ASCII 스캔 | 문서화된 SQL 식 손상 |
| Rule 1 정규식이 1,755건 매치 (의도는 626건) | 실제 저장소 grep | 정당한 테스트 1,129건 오탐 |
| `grep -n` 다중 파일 출력 형식으로 필터 무효화 | 임시 저장소 실행 | 예외 처리가 조용히 동작 안 함 |