# VOIP-1642 린트 승격 설정 변경 설계

## 1. 배경과 목표

VOIP-1641 이 신규 줄 전용 게이트(`.golangci-errorlint.yml`, `--new-from-merge-base`)를 추가했고, VOIP-1642 가 기존 코드의 위반을 서비스 PR 19개로 정리했다. 이 변경은 정리가 끝난 린터를 전체 코드 대상 게이트(`.golangci.yml`)로 승격해, 기존 코드에서도 같은 위반이 다시 쌓이지 않게 한다.

## 2. 측정 근거 (최신 main, golangci-lint 2.14.0, CI 고정 버전)

- 서비스 PR 머지 후 main 재측정(gocyclo, nestif 제외, 이슈 상한 해제, 줄당 1건 제한 해제): 39개 모듈 합계 252건. bin-api-manager `server/` 의 생성 `ServerInterface` 메서드명 revive `var-naming` 251건(설정 예외 대상)과 bin-timeline-manager contextcheck 1건. 후자는 PR #1498 로 처리했고, 그 머지 후 이 변경의 설정으로 timeline 과 api-manager 가 0건이다.
- 이 변경의 `.golangci.yml` 로 39개 모듈 전체를 실행한 결과: 38개 모듈 0건이고, 남은 1개(timeline)는 #1498 반영 후 0건이다.
- gocyclo 와 nestif 는 대표님 결정으로 정리 범위에서 제외했다(서비스 PR 착수 시점 69건과 22건, 2026-10-23 main 재측정 70건과 21건. 합계 91건은 같고 분포만 다르다). 따라서 이 두 린터는 승격하지 않고 신규 줄 게이트에 남긴다.
- 스타일 린터(revive unused-parameter 와 exported 주석, shadow, perfsprint, unparam, nilnil, prealloc, goimports)는 2차 범위에서 제외되었고 승격하지 않는다.

## 3. 변경 내용

| 파일 | 변경 |
|------|------|
| `.golangci.yml` | `standard` 에 15개 린터 추가: errorlint, noctx, rowserrcheck, bodyclose, nilerr, wastedassign, predeclared, copyloopvar, gocritic, contextcheck, gosec, forbidigo, revive, usestdlibvars, nolintlint. 설정(revive 9규칙, gosec G101/G115/G117/G304 제외, errorlint errorf 만, nolintlint 설명 필수)과 테스트 파일, `cmd/*-control`, migrationhandler 예외를 신규 줄 게이트에서 그대로 옮긴다 |
| `.golangci.yml` (예외) | `^bin-api-manager/server/` 경로(설정 파일 기준 상대 경로)의 revive 에서 `^var-naming: method [A-Za-z0-9_]+ should be ` 메시지만 제외한다. 생성 인터페이스(`gens/openapi_server/gen.go`)와 이름이 일치해야 하는 메서드명이다. 같은 파일의 메서드 파라미터(`method parameter ...` 메시지)와 지역 변수 이름 검사는 유지된다. `issues.max-same-issues: 0` 도 이전 설정에서 가져온다 |
| `.circleci/config_work.yml` | 서비스 job 템플릿 3곳의 `Linting` 단계: `no_output_timeout: 20m` 추가, `golangci-lint run --timeout 10m` 을 `18m` 으로 변경(주석 포함). 다른 단계, 승인 게이트, 필터, resource_class 는 변경 없음 |
| `.golangci-errorlint.yml` | gocyclo 와 nestif 만 남긴다(임계값 20, 6 유지, 테스트 예외 유지). 파일 이름은 CI 경로 필터와 문서가 참조하므로 유지한다 |
| `.claude/scripts/check-magic-strings.sh` | 안내 메시지의 `// nolint:magicstring` 을 `//nolint:magicstring // <reason>` 으로 바꾼다(승격된 nolintlint 가 공백형을 거부하므로). 검사 로직(`nolint:magicstring` 부분 문자열 매칭)은 그대로다 |
| `docs/conventions/linting.md`, `error-handling.md`, `database.md`, `logging.md`, `direct-resource-types.md`, `README.md` | 두 패스의 역할, 전체 코드 규칙, 신규 줄 규칙, 예외, 알려진 한계를 현재 상태로 갱신한다. error-handling 4.2 와 database 7.2a 의 "신규 줄 전용" 서술은 "전체 코드"로 고친다 |

서비스 job 은 이미 `golangci-lint run` 을 전체로, `.golangci-errorlint.yml` 을 `--new-from-merge-base` 로 실행하므로 파이프라인 구조는 바꾸지 않는다. 다만 실제 CI 측정(4장 3항)에서 가장 큰 서비스의 린트 단계가 승격 전에도 10분 한도에 근접해 있음이 확인되어, `.circleci/config_work.yml` 의 서비스 job 템플릿 3곳의 `Linting` 단계에 `no_output_timeout: 20m` 을 추가하고 `golangci-lint run --timeout 10m` 을 `--timeout 18m` 으로 올린다(`no_output_timeout` 20분보다 짧게 두어 린터가 먼저 자체 deadline 을 보고하게 한다). 신규 줄 단계(`--new-from-merge-base`)와 `lint-config-check` job 의 시간 제한은 그대로 둔다.

## 3.1 검토한 대안

| 안 | 내용 | 판단 |
|----|------|------|
| A | 승격하지 않고 신규 줄 게이트만 유지 | 기각(대표님 결정 2026-10-23: 시간 한도를 올리고 승격 유지). 서비스 PR 19개로 정리한 비용이 회귀 방지로 이어지지 않고, 기존 파일을 건드리지 않는 변경에서는 새 위반이 다시 쌓일 수 있다 |
| B | `.golangci.yml` 하나로 합치고 gocyclo, nestif 를 끈다 | 기각. 91건 남은 복잡도 위반이 늘어나는 것을 막는 능력을 잃는다 |
| C | 생성 인터페이스 메서드명을 코드의 `//nolint` 로 처리 | 기각. 수기 작성 파일 `bin-api-manager/server/*.go`(생성 헤더 없음, 생성 파일은 `gens/openapi_server/gen.go` 의 인터페이스뿐)의 메서드 중 보고된 251곳에 `//nolint:revive // <사유>` 를 붙여야 하고, 새 `ServerInterface` 메서드를 추가할 때마다 반복해야 한다. 이름 자체는 생성 인터페이스와 맞아야 하므로 바꿀 수 없다. 설정 예외 한 줄이 낫다 |
| D | 같은 변경에서 CI 자원(resource_class)을 상향 | 보류. 1.57 GB / 2 GB 라 메모리는 한도 안이다. 로컬도 1 vCPU 로 77 s 였으므로 CPU 수가 CI 속도 차이의 원인이라는 증거가 없다(측정하지 않음). 대신 시간 한도를 올린다 |
| E | Go 빌드 캐시와 린트 캐시를 CircleCI 에서 save/restore | 범위 밖(5장). 로컬과 CI 의 약 9배 차이는 콜드 캐시가 가장 그럴듯한 가설이지만 이번 조사에서는 확인하지 못했다. 한도만 올리는 것이 최선이 아닐 수 있으므로 별도 변경으로 검토한다 |

표의 A~E 는 이 문서 안의 대안 번호이다. 대표님께 올린 권고안 A(승격 유지 + 시간 한도 상향)는 표의 어느 행도 아니라 이 문서의 변경 자체(승격 15개 린터, gocyclo 와 nestif 는 신규 줄 게이트 유지, 시간 한도 상향)이다.

## 4. 위험과 검증 (verify 항목)

1. **기존 코드의 숨은 위반**: 전체 39개 모듈을 새 설정으로 실행해 0건임을 확인했다(위 측정). `uniq-by-line` 기본값(줄당 1건)에서는 한 줄의 두 번째 위반이 첫 번째를 고친 뒤에 보이므로, 측정은 줄당 제한 해제로 별도 수행해 252건의 내역을 확인했다.
2. **예외 범위**: 경로를 `^bin-api-manager/server/` 로 좁혀 다른 모듈의 `server/` 디렉터리(현재 저장소에는 이 하나뿐)를 가리지 않게 했고, 메시지 정규식은 메서드명(`method <이름> should be`)만 매칭하고 `method parameter ...` 메시지는 매칭하지 않는다. 경로는 설정 파일 위치(저장소 루트) 기준이다(`run.relative-path-mode` 기본값). 서비스 CI 는 서비스 디렉터리로 이동해 `golangci-lint run` 을 실행하고 설정을 상위 디렉터리에서 찾으므로 이 전제가 맞다. 설정을 모듈 안으로 복사하거나 `-c` 로 다른 위치의 파일을 쓰면 경로가 어긋나 예외가 적용되지 않는다. 경로 규칙이 실제로 작동하는지는 경로를 존재하지 않는 값으로 바꾼 음성 대조 실행에서 위반이 다시 나타나는 것으로 확인했다(예외 적용 시 0건, 미적용 시 revive 위반 발생).
3. **CI 자원(메모리, 시간)**: CI 린트 job 은 resource_class small(1 vCPU, 2 GB), `GOGC=50`, `GOMEMLIMIT=1500MiB`, `GOMAXPROCS=1` 이다. 가장 큰 서비스 bin-api-manager 에서 린트 캐시를 비우고 같은 환경변수로 측정했다(Go 빌드 캐시는 따뜻한 상태, 로컬 측정).

   | 설정 | 최대 메모리 | 소요 시간 |
   |------|-------------|-----------|
   | 이전 `.golangci.yml` | 약 1.20 GB | 52 s |
   | 승격한 `.golangci.yml` | 약 1.57 GB | 77 s |

   2 GB 한도 안이지만 여유가 줄었고, 새 설정의 최대 메모리는 GOMEMLIMIT(1500MiB)를 이미 넘는다(GOMEMLIMIT 는 소프트 한도이고 컨테이너 한도는 2 GB 다). 이 값은 단일 시행이며 Go 빌드 캐시가 따뜻한 로컬 측정이고, `time -v` 의 최대 상주 메모리는 가장 큰 프로세스 하나의 값이다. CI 주석의 이전 설정 콜드 캐시 측정값(1.08 GB)과 조건이 다르므로 단순 비교는 하지 않는다. 신규 줄 패스는 승격된 린터를 더 이상 실행하지 않으므로(gocyclo, nestif 만) 서비스 job 의 총 실행 시간 증가는 위 차이보다 작다. **실제 CI 측정(2026-10-23, 임시 PR 2회 + 승격 전 기준선 조회)**: 임시 PR 은 bin-api-manager 에서 1회차 `Linting` 600.5 s 에 CircleCI 무출력 한도(10분)로 timeout, 2회차 523.6 s 통과(린트 0건)였다. 승격 전 기준선은 같은 서비스 main 6건에서 `Linting` 405~509 s(평균 466 s)이고 `Lint new-line conventions` 28~72 s(평균 45 s), 두 단계 합계 433~580 s(평균 511 s), job 전체 566~727 s 이다. 승격 후 관측은 두 건뿐이다: 1회차(job #102345)는 `Linting` 이 600.5 s 에서 한도로 중단된 값이라 실제 소요는 600.5 s 이상이다. 이후 `Lint new-line` 단계는 실행되지 않아 합계는 알 수 없지만, `Linting` 단독으로 이미 승격 전 두 단계 합계 최대(580 s)를 넘는다. 이 회차는 `Spin up environment` 도 24.3 s 로 2회차(8.5 s)와 승격 전 main(약 3~12 s)보다 길어 호스트 편차가 있었을 가능성이 있다(단서일 뿐 확인하지 못했다. 승격 전 PR 실행 한 건에도 `Spin up` 23.1 s 가 있었으나 그 실행의 `Linting` 은 483.2 s 로 정상 범위였으므로 단서는 약하다). 2회차(job #102351)는 `Linting` 523.6 s + `Lint new-line` 5.7 s = 529 s, job 전체 691 s 로 기준선 범위 안이다. 두 건만으로는 승격이 합계를 얼마나 늘리는지 확정할 수 없다(승격 전 기준선의 일 단위 변동 폭 405~509 s 가 1, 2회차 차이만큼 크다). 로컬은 +45%(53 s 에서 77 s)이고 CI 통과 1건은 평균 대비 +12% 로 서로 다르며 이 차이는 설명하지 못했다. 분명한 사실은 승격 전에도 `Linting` 단계가 10분 한도에 최대 약 91 s 여유뿐이었다는 점이고(call, flow 서비스는 약 200 s), 승격으로 그 여유가 더 줄거나 사라질 수 있다. 로컬 측정(약 53 s)과 CI 의 약 9배 차이의 원인은 확인하지 못했다(메모리 사용량은 CircleCI API 로 조회할 수 없고, 로그에 OOM, Killed, 137 흔적은 없다). 이 단계에 `no_output_timeout` 이 없어 기본 10분이 적용되고 `--timeout 10m` 도 같은 값이다. 그래서 단계 한도를 올린다(무출력 20분, `--timeout` 18분). 20분은 관측 최대(600 s 이상)의 약 2배이고, 한도를 올리면 실제로 멈춘 job 의 감지가 최대 10분 늦어지는 비용이 있다. `lint-config-check` job(bin-common-handler 대상)은 이 PR 의 첫 파이프라인(job 102340)에서 승격 설정 린트 단계가 약 126 s 였고 시간 한도(10분)는 그대로 둔다. 이 변경은 승격과 한 쌍이므로 같은 PR 에 둔다(따로 머지하면 사이에 timeout 위험이 있는 상태가 생긴다).

**수정한 머지 선행 조건**: 처음 정한 "7분 이하" 기준은 승격 전 기준선(평균 약 8분)도 넘으므로 잘못 잡은 기준이었다. 새 기준의 목적은 승격 후 린트 단계가 새 한도(20분)에 충분한 여유를 두고 끝나는지 확인하는 것이다. 판정은 하나의 임계값으로 한다: 같은 커밋에서 CI 를 2회 실행해 **두 회차 모두** (1) 린트 단계가 새 한도 안에서 통과하고, (2) `Linting` 과 `Lint new-line conventions` 의 합계가 720 s(12분, 무출력 한도 20분의 60%) 이하이며, (3) 로그에 OOM, Killed, 종료 코드 137 흔적이 없어야 한다. 2회 모두 충족하면 통과한다. 1회만 어기면(실행 편차에 따른 거짓 불합격을 피하려고) 1회를 더 실행해 3회 중 2회 이상 충족이면 통과로 보고, 어긴 회차의 값을 PR 본문에 그대로 기록한다. 2회 이상 어기거나 OOM 흔적이 있으면 머지하지 않고 단계별 시간과 로그 요지를 모아 대표님께 올려 판단을 받는다. 승격 후 관측이 적어(위 두 건) 분포는 알려지지 않았음을 전제로 한 기준이다. 결과(실행 횟수, 단계별 소요 시간, 종료 상태, 임시 PR 번호)는 머지 전 이 PR 본문에 기록한다. 임시 PR 은 이 커밋 위에서 `bin-api-manager/` 아래 파일 하나(주석 한 줄)를 바꾸는 draft PR 이어야 서비스 job 이 켜진다(`.circleci/config.yml` 의 path filter 는 `config_work.yml` 변경만으로는 서비스 job 을 켜지 않는다). 같은 커밋에서 워크플로를 두 번 실행한다. 임시 PR 은 `build-approval` 승인이 필요하고 test 가 통과하면 이어지는 build job 이 SHA 태그 이미지를 push 한다(`latest` 와 deploy 는 main 전용이라 운영 영향은 없음). 린트 결과만 필요하면 test 완료 뒤 워크플로를 취소해도 된다.
4. **머지 순서**: 이 변경은 서비스 PR 19개(#1498 timeline-connect 포함)가 모두 머지된 main 위에서만 머지한다. 그 전에 머지하면 해당 서비스의 CI 가 실패한다.
5. **`lint-config-check` job**: 이 변경은 `.golangci.yml`, `.golangci-errorlint.yml` 을 바꾸므로 `run-lint-config-check` 가 켜진다. 이 job 은 `golangci-lint config verify` 와 bin-common-handler 린트(두 패스)를 실행한다. 두 설정 모두 로컬에서 `config verify` 를 통과했다.
6. **문서와 CI 주석 불일치**: `.circleci/config_work.yml` 의 "Lint new-line conventions" 단계 주석은 `%w`, noctx 를 설명하고 있어 낡았다. 낡은 주석을 정리하는 `.circleci` 변경은 이번 시간 한도 변경과 무관하므로 별도로 두고, 이 변경에서는 시간 한도 줄만 바꾼다(주석은 새 한도 설명만 추가).

## 5. 범위 밖

- CI 가 로컬보다 약 9배 느린 원인 규명과 린트 캐시의 CI 저장 같은 속도 개선(별도 변경).
- gocyclo, nestif 의 기존 위반 정리와 승격.
- 스타일 린터 승격.
- `.circleci` 의 위 한도 변경 외 변경(resource_class 상향, 낡은 "Lint new-line conventions" 단계 주석 정리).
- `server/` 예외를 다른 모듈의 생성 인터페이스로 일반화하는 것.

## 6. 롤백

`.golangci.yml`, `.golangci-errorlint.yml`, `.circleci/config_work.yml`(린트 한도), 문서를 이전 커밋으로 되돌리는 revert 한 번이면 된다. 승격과 시간 한도 상향은 한 쌍이므로 한도만 되돌리고 승격을 유지하면 timeout 위험이 다시 생긴다. 서비스 코드는 영향을 받지 않는다.
