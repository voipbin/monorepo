# VOIP-1724 CI: service-migration-collation job 에 libsoxr 설치

## A. 이슈 분석

### A.1 문제

`service-migration-collation` job(VOIP-1687)은 변경된 서비스의 `<svc>-control` 을 `go build` 한 뒤 up/down/up 을 실행한다. `bin-pipecat-manager/migrations` 를 바꾸는 PR 은 이 job 의 빌드 단계에서 `Package soxr was not found in the pkg-config search path` 로 실패한다. 첫 사례는 VOIP-1723 의 PR(#1770)이다.

### A.2 원인 (origin/main 748883de7 기준 확인)

- `bin-pipecat-manager/cmd/pipecat-control/main.go` 는 `pkg/pipecatcallhandler` 를 포함하는 의존 그래프를 갖고, 그 안의 `audiosocket.go` 가 `github.com/zaf/resample`(cgo, libsoxr)을 임포트한다.
- job 은 `cimg/go:1.27.1` 에서 `mariadb-client` 만 설치한다. 같은 파일의 `go-test-pipecat-manager` command 는 `pkg-config`, `libsoxr-dev` 를 이미 설치한다.
- 로컬 재현: 빈 `PKG_CONFIG_LIBDIR` 와 새 `GOCACHE` 에서 `go build ./cmd/pipecat-control` 은 위 오류로 실패하고, 정상 환경에서는 성공한다.

### A.3 다른 target 영향 범위

- `go list -deps ./cmd/<name>-control` 을 MariaDB target 의 control 바이너리 전부(`bin-*/migrations` 를 가진 서비스의 `cmd/*-control`, PostgreSQL 인 bin-rag-manager 와 ClickHouse 인 bin-timeline-manager 기본 경로 제외)에 실행한 결과 `zaf/resample` 을 포함하는 것은 `pipecat-control` 하나다. 소스 전체에서 `soxr`, `zaf/resample` 을 쓰는 Go 파일도 `bin-pipecat-manager/pkg/pipecatcallhandler/audiosocket.go` 하나다. `import "C"` 를 직접 쓰는 비-vendor 소스는 없다.
- 따라서 이 한 단계로 현재 모든 target 의 빌드가 해결된다. 앞으로 다른 서비스 control 이 cgo 라이브러리를 끌어오면 그때 같은 방식으로 추가한다(선제 확장 없음).

### A.4 배포, 승인 job 영향

- `service-migration-collation` workflow 는 `when: << pipeline.parameters.run-service-migration-collation >>` 로만 켜지고, 어떤 build, deploy, approval job 도 이 job 을 `requires` 하지 않는다.
- `.circleci/config_work.yml` 변경은 path-filtering 매핑에 의해 `run-shell-tests`, `run-lint-config-check`, `run-service-migration-collation` 을 켠다. 이 변경에서 job 정의 외의 부분은 건드리지 않는다.
- `.circleci/config_work.yml` 변경 PR 이므로 job 자체가 smoke target(`bin-tag-manager`)으로 실행된다. 이 PR 의 CI 로는 pipecat 빌드가 직접 검증되지 않는다(A.5).

### A.5 검증 한계와 보완

smoke target 이 `bin-tag-manager` 라서 이 PR 의 job 통과만으로 pipecat 빌드 성공은 증명되지 않는다. 보완은 두 가지다: (1) `bin-pipecat-manager-test` job 이 `*go_image`(`cimg/go:1.27.1`, 이 job 과 같은 이미지)에서 `go-test-pipecat-manager` 로 같은 두 패키지를 설치하고 pipecat 모듈을 빌드하는 선례, (2) VOIP-1723 의 PR 이 이 수정 머지 뒤 rebase 되면 같은 job 이 `bin-pipecat-manager` target 으로 실행되어 최종 검증이 된다.

## B. 디자인

- 방식: job 의 `Install the MariaDB client` 단계 바로 다음에 별도 `run` 단계 `Install libsoxr build dependencies` 를 추가한다. 기존 단계를 고치지 않는다.
- 대안 검토: (1) 기존 단계에 패키지를 합치기: 단계 이름과 목적이 흐려진다. (2) 새 command 로 추출해 두 곳에서 재사용: 두 곳뿐이고 선제 추상화가 된다. (3) `CGO_ENABLED=0` 으로 빌드: resample 이 cgo 필수라 빌드 불가. 채택하지 않는다.
- 명령은 같은 job 의 MariaDB 단계와 같은 두 줄 형태(`sudo apt-get update`, `sudo apt-get install -y pkg-config libsoxr-dev`)로 하고, 마지막에 `pkg-config --exists soxr` 로 설치 결과를 직접 확인한다.

## C. 설계

변경 파일은 `.circleci/config_work.yml` 하나와 이 문서다.

```yaml
      - run:
          name: Install libsoxr build dependencies
          # bin-pipecat-manager's pipecat-control links libsoxr through cgo
          # (github.com/zaf/resample); the other services need nothing here.
          command: |
            sudo apt-get update
            sudo apt-get install -y pkg-config libsoxr-dev
            pkg-config --exists soxr
```

- 검증 기준(verify): (1) 변경 후 YAML 이 파싱된다. (2) `shell-tests` 의 `check-auto-deploy-config.rb` 와 bats 가 통과한다. (3) PR 의 `service-migration-collation` job 이 통과한다. (4) 설치 단계 끝의 `pkg-config --exists soxr` 가 CI 에서 성공한다. pipecat-control 빌드 자체는 PR 에 로컬 재현(빈 PKG_CONFIG_LIBDIR 에서 실패, 정상 환경에서 성공)을 기록하고, 머지 뒤 VOIP-1723 rebase 의 job 실행으로 최종 확인한다. 그 실행이 실패하면 설치 실패인지 다른 cgo 의존인지 로그로 구분한다.
- 롤백: 이 단계를 되돌리면 이전 동작(pipecat migrations PR 실패)으로 돌아가며 다른 영향은 없다.
