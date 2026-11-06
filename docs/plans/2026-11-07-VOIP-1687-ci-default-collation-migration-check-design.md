# VOIP-1687 CI: service-owned migration default-collation check

## A. 이슈 분석

### A.1 문제

VOIP-1686: `bin-email-manager` migration `000002` 의 `JSON_TYPE(value_json) <> 'NULL'` 이 MariaDB 12.x 에서 `CHARACTER SET utf8` 만 지정해 만든 DB(기본 collation `utf8mb3_uca1400_ai_ci`)에서 Error 1267 로 실패했다. 기존 검증은 DB 를 `utf8 COLLATE utf8_general_ci` 로 명시 생성해 이 실패를 보지 못했다.

#1741(`d8ff9e9e3`)이 `scripts/verify-service-migrations.sh` 에 V13 단계(collation 미지정 DB 에서 up/down/up)를 추가했다. 이 스크립트는 `mariadb:12.3`, `mysql:8.0` 컨테이너를 docker 로 띄우는 수동 실행용이며 CircleCI 에서 호출되지 않는다(`grep -rn verify-service-migrations .circleci` 결과 없음).

### A.2 현재 CI 가 하는 일 (origin/main d8ff9e9e3 기준 확인)

- `migration-lint`: bin-dbscheme-manager 의 Alembic 정적 검사. DB 접속 없음.
- 서비스별 `bin-*-test`: `go mod vendor`, lint, `go test`. 서비스 `migrations/migrations_test.go` 는 파일 구조와 문자열 정적 검사만 하고(`migrationcheck.Check`, `Test_noStrayFiles` 등) DB 에 적용하지 않는다. `bin-common-handler` 의 migrationhandler 통합 테스트는 `MIGRATOR_TEST_DSN` 이 없으면 `t.Skip` 하며(`integration_test.go`) CI 에는 이 변수가 없다.
- path-filtering(`config.yml`)은 `bin-X/.*` 단위로 `run-bin-X` 만 켠다. migrations 변경만 따로 감지하는 매핑은 없다. 따라서 이 일감은 별도 매핑 한 줄과 별도 파라미터가 필요하다(`bin-common-handler` 변경은 아래 A.8 에서 다룬다).
- `bin-dbscheme-manager-test`: `mariadb:12.3` 보조 컨테이너를 쓰는 선례(`MARIADB_ALLOW_EMPTY_ROOT_PASSWORD`).
- 따라서 서비스 migration 을 실제 DB 에 적용하는 CI 단계는 없다.

### A.3 대상 범위 (코드로 확인)

`bin-*/migrations` 디렉터리를 가진 서비스 28개 중 MariaDB 서비스는 26개이다. 제외 대상은 서비스별로 다음과 같다. `bin-rag-manager`: PostgreSQL 이며 `migrations/embed.go` 와 `*-control` 이 없다. `bin-timeline-manager`: 부모 디렉터리 `migrations/` 는 ClickHouse DDL(`ENGINE = MergeTree`)이며 `migrations/embed.go` 가 없어 제외한다. 단 하위 디렉터리 `migrations/mysql/` 은 MySQL/MariaDB 분석 저장소 세트이고(`embed.go` 패키지 `mysqlmigrations`, VersionTable `timeline_schema_migrations`, `000001_baseline`) `timeline-control migrate-mysql` 로 적용되므로 별도 검사 대상이다. 실측: `CHARACTER SET utf8` DB 에 `migrate-mysql up` 성공, 환경변수는 `DATABASE_DSN` 만 필요. 따라서 검사 대상은 26개 서비스 디렉터리 + timeline 의 mysql 세트 = 27개이다. 대표님 범위(PostgreSQL/ClickHouse 제외)와 일치한다. 26개 사이의 차이: `bin-contact-manager` 는 `contact-control` 과 `case-control` 이 있고 migrate 명령은 `contact-control` 에만 있다. `bin-registrar-manager` 는 추가 환경변수가 필요하다(A.4). 서비스 목록은 하드코딩하지 않고 디렉터리 탐색으로 만든다.

### A.4 실측 (scratch `mariadb:12.3.x` 컨테이너, `CREATE DATABASE x CHARACTER SET utf8`, origin/main)

`mariadb:12.3.3`(`@@collation_server=utf8mb4_uca1400_ai_ci`, 서비스마다 별도 DB 를 `CHARACTER SET utf8` 로 생성, 결과 DB 의 기본 collation 은 `utf8mb3_uca1400_ai_ci`)에서 26개 서비스 디렉터리의 `<name>-control migrate up` 과 timeline 의 `migrate-mysql up` 을 실행했다. 25개는 DSN 환경변수 `DATABASE_DSN` 만으로 성공했다. `bin-registrar-manager` 는 `DATABASE_DSN_BIN` 과 `DOMAIN_NAME_EXTENSION`, `DOMAIN_NAME_TRUNK` 이 필요해 `DATABASE_DSN` 만으로는 `could not set base domain names` 로 실패했고, 세 변수를 주면 성공했다. 즉 현재 main 의 모든 서비스 migration 은 기본 collation DB 에서 up 이 통과한다(과거 결함이 더 숨어 있지 않다).

### A.5 반증 시도

- "이미 CI 에서 한다": 아니다(A.2).
- "중복 티켓": Jira 에서 collation 관련 열린 티켓은 VOIP-1600(스키마 소유권 이관)뿐이며 이 일감과 무관하다.
- "스크립트를 그대로 CI 에 연결하면 된다": 스크립트는 Alembic venv, 두 엔진, Alembic E1/E2 비교 등 큰 검사 묶음이며 docker 중첩이 필요해 CI 비용이 크다. 대표님 지시는 V13 에 해당하는 최소 단계만이다.

### A.5.1 down 단계의 검증력

baseline 의 down 은 no-op(`SELECT 1`, "A baseline is never reverted")이고 up 은 `CREATE TABLE IF NOT EXISTS` 라서, baseline 만 있는 서비스(26개 중 대부분)의 up/down/up 은 사실상 첫 up 이 검증의 전부이고 down 이후 두 번째 up 은 baseline 을 다시 만들지 않는다. down 이 의미 있는 것은 non-baseline migration 이다(`000002` 이후, 일부 down 은 lossy 표시: ai, contact, email, number, talk). non-baseline migration 이 있는 서비스는 6개(ai, contact, email, number, schedule, talk)이다. `-- DESTRUCTIVE:` 마커가 있는 서비스(현재 schedule-manager)는 수동 스크립트와 같이 down/up 을 건너뛰고 첫 up 만 확인한다. 일부 baseline 은 `utf8mb4_0900_ai_ci` 를 표 옵션으로 명시한다(ai, call, talk). 그 표들은 기본 collation 경로를 타지 않으므로 이 검사가 보장하는 범위는 "기본 collation 에 의존하는 문장(함수 결과와 리터럴 비교 등)이 실패하지 않는다"이다.

### A.6 확인하지 못한 것

- CircleCI 에서 보조 컨테이너 job 의 실제 실행 시간(첫 실행 뒤 측정).
- 셀프호스팅 설치 경로의 DB 생성 collation(이번 범위 밖).
- 운영 DB 의 실제 collation(운영 열람 금지).

### A.7 검증 기준 (verify)

- collation 미지정 DB 에서 결함이 있는 migration 을 넣으면 job 이 실패한다(RED: VOIP-1686 수정 이전 `000002` 로 재현).
- 결함 없는 main 상태에서 job 이 통과한다.
- migrations 를 건드리지 않는 PR 에서는 job 이 실행되지 않는다.
- 모든 MariaDB 서비스에 같은 코드 경로가 적용된다.
- 서비스 목록은 `bin-*/migrations` 디렉터리 탐색으로 만들고, 제외 대상(rag, timeline 의 ClickHouse 부모 디렉터리)이 아닌 target 이 embed.go 또는 `<name>-control` 을 갖지 못하면 job 이 실패한다(조용히 건너뛰지 않는다).
- 실패 출력에 서비스 이름, 단계(up/down/up), 마지막 오류 줄이 나온다.
- MariaDB 이미지는 `mariadb:12.3`(minor 고정 태그, `bin-dbscheme-manager-test` 와 같음)이며 준비 대기에 상한(120초)이 있고 job 에 `no_output_timeout` 이 있다.
- RED 증거: VOIP-1686 수정 이전 `000002` 를 가진 scratch 복사본에서 같은 스크립트가 실패한다(로컬). CI 안의 RED 는 PR 의 job 결과로 남긴다.
- 변경된 서비스만 순차로 처리한다(보통 1~2개). 병렬 job 분할은 하지 않는다(실측 신호 없음, 트리거: 한 PR 에서 5개 이상 서비스 동시 변경이 반복되거나 job 시간이 10분을 넘을 때).

### A.8 영향 범위 판단

`bin-common-handler/pkg/migrationhandler` 변경 시 이 검사를 켤지: 켜지 않는다(기능은 migrationhandler 단위 테스트가 담당하고, 수동 `HELPER_TESTS=1` 이 있다). 이 검사는 서비스 SQL 의 collation 의존 오류가 대상이다. 트리거는 `bin-*/migrations/` 하위 변경이다.

## B. 디자인 (방식 선택)

목표: migrations 가 바뀐 PR 에서, 변경된 MariaDB 서비스의 migration 을 collation 미지정 `mariadb:12.3` DB 에 up/down/up 한다. 공통 메커니즘 하나로 26개 서비스를 덮는다.

### B.1 선택지

| 안 | 내용 | 평가 |
|---|---|---|
| 1 | 26개 서비스 workflow 마다 migration job 추가 | 26곳 복제, 서비스 추가 때 누락 위험. 기각 |
| 2 (권고) | path-filtering 에 `bin-[^/]+/migrations/.*` 정규식 매핑 1줄과 파라미터 1개, workflow 1개, job 1개, 스크립트 1개. 스크립트가 git diff 로 변경된 서비스를 찾아 순차 처리 | 변경 지점 5개, 서비스 추가에 코드 변경 없음. 기존 `bin-dbscheme-manager-test` 의 보조 컨테이너 방식과 `shell-tests` 의 apt 설치 방식 재사용 |
| 3 | `verify-service-migrations.sh` 전체를 CI 에서 실행 | Alembic venv, 두 엔진, docker 중첩 필요. 비용 큼. 대표님 지시(최소 단계)에 반함. 기각 |

### B.2 결정 사항

- 방식 2. 트리거는 `bin-*/migrations/` 하위 변경이다. `bin-common-handler` 변경은 트리거하지 않는다(A.8). path-filtering 매핑은 정규식이며 기존 선례(`bin-agent-manager/.*`)와 같은 형식의 한 줄 `bin-[^/]+/migrations/.*   run-service-migration-collation true` 를 쓴다(전체 일치와 부분 일치 어느 쪽이어도 같은 파일 집합에 맞는 패턴). 한 파일이 여러 패턴에 맞는 것은 선례가 있다(`config_work.yml` 이 두 파라미터를 함께 켠다). 매핑 외에 파라미터 선언, workflow, job, 스크립트가 더 필요하다.
- 브랜치 필터 없음: `migration-lint`, `shell-tests` 선례처럼 모든 브랜치에서 실행한다. `check-auto-deploy-config.rb` 의 `check_kept_workflow` 는 auto-deploy 목록에 없는 workflow 의 `branches.ignore` 필터를 오류로 보므로(`has an ignore branch filter but ... is not auto-deploy`) non-main 전용 필터는 쓰지 않는다. main 에서 두 PR 이 각자 통과한 뒤 합쳐진 결합 결과도 검사된다. 서비스 build/deploy job 의 `requires` 에는 넣지 않으므로 main 의 실패는 배포를 막지 못하고 신호 역할이다(배포 경로를 바꾸지 않기 위함). PR 단계의 머지 차단은 GitHub 필수 체크 설정에 달려 있고 그 설정은 이 변경 범위 밖이며, QA 머지 전에 이 job 의 상태를 확인하는 것을 QA-1 본문에 적는다.
- 실행 환경: `docker: [cimg/go:1.27.1 (기존 go_image), mariadb:12.3 (bin-dbscheme-manager-test 와 같은 보조 컨테이너, MARIADB_ALLOW_EMPTY_ROOT_PASSWORD)]`. DB 생성에 필요한 클라이언트는 apt `mariadb-client` 를 `shell-tests` 와 같은 `sudo apt-get install` 방식으로 설치한다. 변경 서비스의 `<name>-control` 만 `go build` 하고(별도 `go mod vendor`/`download` step 없이 `go build` 가 모듈을 받는다. 첫 실행에서 시간을 측정한다) 전체 서비스를 빌드하지 않는다.
- 변경 서비스 판정: `git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'`, merge-base 가 없으면 `--unshallow` 또는 `--deepen=1000` 로 보강(config_work.yml 의 check-test-conventions 선례와 같은 절차). PR 브랜치는 `origin/main...HEAD` 삼중점 diff, main 브랜치(`CIRCLE_BRANCH=main`)는 `HEAD~1..HEAD`. merge-base 를 끝내 구하지 못하면 변경 서비스 판정을 포기하고 전체 서비스를 검사한다(fail-safe).
- 판정 규칙: 검사 대상은 target 단위이다(서비스 디렉터리의 `migrations/`, 그리고 `bin-timeline-manager/migrations/mysql` 은 control 명령 `migrate-mysql` 를 쓰는 별도 target 으로 스크립트 안에 한 줄로 등록). 제외 목록은 `bin-rag-manager`(PostgreSQL)와 `bin-timeline-manager` 의 ClickHouse 부모 디렉터리뿐이며, 변경된 파일이 그 제외 대상에만 속하면 "skipped: not MariaDB" 로 건너뛴다. 그 밖의 target 이 `embed.go` 또는 `cmd/<name>-control` 을 갖지 못하면 job 이 실패한다. 변경 파일의 target 디렉터리가 HEAD 에 존재하지 않으면(서비스 삭제) 그 target 은 건너뛴다(파일 상태가 아니라 디렉터리 존재로 판단하므로 `*.down.sql` 만 삭제한 PR 도 target 으로 잡힌다). DESTRUCTIVE 마커 target 은 첫 up 만 하고 skip 으로 표시한다. 정의: "변경 target 목록이 빈다" 는 제외와 삭제 필터를 적용한 뒤의 상태이다.
- 빈 목록 smoke 의 의미: 기준 서비스 `bin-tag-manager`(N=1, baseline 뿐, DESTRUCTIVE 없음)로 job 정의, 컨테이너, 스크립트 배관만 확인한다(down 검증이 아님). 비용(sidecar, apt, `go build` 의 모듈 다운로드, tag-control 빌드)은 `config_work.yml` 이 바뀔 때마다 발생하며 의도된 비용이다. A.7 의 "migrations 를 건드리지 않으면 실행되지 않음" 은 "서비스 target 검사를 하지 않음" 으로 읽고, 스크립트나 `config_work.yml` 변경 때만 smoke 로 실행된다.
- main 브랜치 경로: `HEAD~1` 이 얕은 clone 에서 해석되지 않을 수 있으므로 해석 실패 시 `git fetch --deepen=1` 을 하고, 그래도 실패하면 전체 검사로 간다. `set -e` 로 죽는 경로가 없어야 한다(모든 git 호출의 실패를 분기로 처리).
- 스크립트는 `.circleci/scripts/` 에 두고(`run-shell-tests` 가 켜짐) bats 테스트(`.circleci/tests/`)로 서비스 판정 로직(diff 모드, 제외, 실패 조건)을 DB 없이 검증한다. DB 를 쓰는 부분의 RED 는 로컬 scratch 실측으로 증거를 남긴다.
- 서비스별 추가 환경변수: 스크립트는 모든 서비스에 대해 `DATABASE_DSN` 과 `DATABASE_DSN_BIN` 을 같은 scratch DSN 으로, `DOMAIN_NAME_EXTENSION=registrar.example.com`, `DOMAIN_NAME_TRUNK=trunk.example.com`(가짜 값)을 함께 준다. 서비스별 하드코딩 분기를 두지 않는다(A.4 에서 registrar 가 이 세 변수로 성공하고 다른 서비스는 영향 없음을 확인). 이 변수들로도 부족한 새 서비스는 job 이 실패하고, 실패 출력에 서비스 이름과 마지막 오류가 나오므로 그때 스크립트에 변수를 추가한다.
- 각 단계의 정의는 수동 스크립트 V13 과 같다: `CREATE DATABASE x CHARACTER SET utf8`(collation 미지정) 후 `up`(버전이 마지막 번호 N 인지 확인), DESTRUCTIVE 가 아니면 up 파일 수 N 만큼 `down`(버전 0 확인), 다시 `up`(버전 N 확인).
- 자기 자신의 변경 검증: 이 job 을 추가하거나 이후 스크립트와 `config_work.yml` 만 고치는 PR 에서도 job 이 실행되도록 매핑에 `\.circleci/scripts/check-service-migration-default-collation\.sh` 와 `\.circleci/config_work\.yml` 를 같은 파라미터로 더한다. 이 경우 diff 에 서비스 migrations 가 없으므로 빈 목록 규칙을 둔다: 변경 서비스 목록이 비면 기준 서비스 `bin-tag-manager`(service-owned migrations 의 reference service)로 smoke 검사한다. 이로써 job 정의 자체가 CI 에서 실행되어 PR 의 job 결과가 RED/GREEN 증거가 된다.
- 시간: 보조 컨테이너 준비 대기 상한 120초, 스크립트 실행 step 에 `no_output_timeout: 20m`. fail-safe 전체 검사(27 target)는 서비스당 `go build` 와 migrate 를 순차로 하므로 가장 느리며, job 시간이 10분을 넘으면 병렬 분할을 재검토한다(A.7 트리거와 같음).
- 순차 처리, 병렬 분할 없음(A.7).
- 이 변경이 막는 것: migration SQL 이 DB 기본 collation 에 의존하는 오류. 막지 않는 것: 두 엔진 차이(MySQL 8.0), Alembic 대비 스키마 동일성, 운영 DB 와의 차이. 이 부분은 수동 `verify-service-migrations.sh` 에 남는다.
- 이 변경의 되돌림: `.circleci` 변경 revert 로 충분하다(운영 영향 없음, 배포 없음).
- QA 하드룰에 V13 명시는 qa SOUL.md 변경이므로 이 PR 범위 밖이며 별도 ETC 티켓으로 제안한다.
- job 은 `resource_class: medium`(기본) 으로 하고 target 마다 새 DB(`CHARACTER SET utf8`)를 만든다. 서비스명에서 control 이름을 도출하는 규칙(`bin-` 과 `-manager` 제거 후 `<name>-control`)과 timeline mysql 의 `migrate-mysql` 예외 등록은 bats 로 고정한다. CI 에서 `go build` 가 `go mod vendor` 없이 되는지는 첫 실행 뒤 측정한다(A.6).

## C. 설계 (구현 명세)

### C.1 변경 파일

| 파일 | 변경 |
|---|---|
| `.circleci/config.yml` | path-filtering `mapping` 에 3줄: `bin-[^/]+/migrations/.*`, `\.circleci/scripts/check-service-migration-default-collation\.sh`, `\.circleci/config_work\.yml` 를 모두 `run-service-migration-collation true` 로 |
| `.circleci/config_work.yml` | `parameters` 에 `run-service-migration-collation`(boolean, default false) 선언, `workflows` 에 `service-migration-collation`(`when: << pipeline.parameters.run-service-migration-collation >>`, job 하나, 필터 없음, `requires` 없음), `jobs` 에 `service-migration-collation` |
| `.circleci/scripts/check-service-migration-default-collation.sh` | 새 스크립트 |
| `.circleci/tests/check-service-migration-default-collation.bats` | 새 bats (DB 없는 판정 로직 테스트) |
| `docs/workflows/service-owned-migrations.md` | 기존 V13 문장에 "CI job `service-migration-collation` 이 변경된 target 에 같은 검사를 자동으로 실행한다" 한 줄 추가 |

### C.2 job 정의

```yaml
  service-migration-collation:
    docker:
      - image: cimg/go:1.27.1          # *go_image 와 같은 이미지
      - image: mariadb:12.3
        command: ["--innodb-flush-log-at-trx-commit=0", "--skip-log-bin"]
        environment:
          MARIADB_ALLOW_EMPTY_ROOT_PASSWORD: "1"
    steps:
      - checkout
      - run:
          name: Install the MariaDB client
          command: |
            sudo apt-get update
            sudo apt-get install -y mariadb-client
      - run:
          name: Check migrations on a default-collation database
          no_output_timeout: 20m
          command: bash .circleci/scripts/check-service-migration-default-collation.sh
```

`*go_image` 앵커는 리스트이므로 보조 컨테이너를 더하려고 같은 이미지를 job 안에 직접 쓴다(`bin-dbscheme-manager-test` 가 `python:3.11-slim` 을 직접 쓰는 선례). fetch/merge-base/deepen 절차는 별도 step 이 아니라 스크립트 안에 둔다(main 경로와 PR 경로를 한 곳에서 분기하기 위함). resource_class 는 기본값.

### C.3 스크립트 동작

> C.5 와 C.6 이 이 절에 우선한다(git 범위, DESTRUCTIVE 탐지, 단계 순서, FAIL 정의, 안전장치).

환경: `DB_HOST`(127.0.0.1), `DB_PORT`(3306), `DB_USER`(root), `TARGETS`(공백 구분 target 목록 override, bats 와 로컬 용), `LIST_ONLY=1`(목록만 출력하고 종료, bats 용).

target 표기: `<service-dir>` 또는 `<service-dir>:<migrations-subdir>:<migrate-command>`. 기본 target 은 `<service-dir>` 이고 `migrations/`, 명령 `migrate`. 스크립트 안의 등록표(case 문 한 곳)에 `bin-timeline-manager:migrations/mysql:migrate-mysql` 한 줄이 있고, 제외 대상은 `bin-rag-manager` 와 timeline 의 ClickHouse 부모 디렉터리(`bin-timeline-manager/migrations/` 직속 파일)뿐이다.

1. 목록 결정(`TARGETS` 비었을 때): 
   - `CIRCLE_BRANCH=main` 이면 base 를 `HEAD~1`, 해석 실패 시 `git fetch --no-tags --deepen=1 origin main` 후 재시도, 그래도 실패하면 전체. 그 밖에는 `git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'`, `git merge-base origin/main HEAD` 실패 시 `--unshallow` 또는 `--deepen=1000`, 그래도 실패하면 전체.
   - 변경 파일: `git diff --name-only <base>...HEAD`(main 은 `<base> HEAD`). 모든 git 호출은 `if` 로 실패를 분기한다(`set -e` 로 죽지 않음).
   - 변경 파일에서 `^(bin-[^/]+)/migrations/(.+)$` 를 target 으로 매핑: `bin-timeline-manager` 의 `migrations/mysql/` 이하는 mysql target, 같은 서비스의 `migrations/` 직속은 제외, `bin-rag-manager` 는 제외(둘 다 "skipped: not MariaDB" 출력). 나머지는 서비스 target. 중복 제거, 정렬.
   - 서비스 target 디렉터리가 HEAD 에 없으면 건너뜀.
   - 전체 모드는 `bin-*/migrations` 디렉터리 전수 탐색 + mysql target, 같은 제외.
   - 목록이 비면 smoke target `bin-tag-manager`.
2. 검증(target 마다): `migrations/embed.go` 와 `cmd/<name>-control` 이 없으면 FAIL(제외 대상 제외). `name` 은 `bin-` 과 `-manager` 를 뗀 값.
3. 준비: mysql 클라이언트로 접속 가능할 때까지 최대 120초 대기. 실패하면 exit 1.
4. target 마다 순차: `CREATE DATABASE <db> CHARACTER SET utf8`(collation 지정 없음, db 이름은 `c_<name>` 에서 `-` 를 `_`), `(cd <service-dir> && go build -o $WORK/ ./cmd/<name>-control)`, 환경 `DATABASE_DSN`, `DATABASE_DSN_BIN`(같은 DSN `root:@tcp(host:port)/db?parseTime=true&loc=Local`), `DOMAIN_NAME_EXTENSION=registrar.example.com`, `DOMAIN_NAME_TRUNK=trunk.example.com` 으로 `<control> <cmd> up`, 버전 N(up 파일의 최대 번호) 확인, `-- DESTRUCTIVE:` 마커가 없으면 up 파일 수 N 만큼 `down`, 버전 0 확인, 다시 `up`, 버전 N 확인. 마커가 있으면 `SKIP down/up (DESTRUCTIVE)` 출력. 버전은 `<cmd> status` 의 `Current version:` 줄.
5. 출력: target/단계마다 `PASS`/`FAIL`/`SKIP` 한 줄, FAIL 은 해당 단계의 마지막 5줄을 이어서 출력. 하나가 FAIL 이어도 나머지를 계속 검사하고 마지막에 요약(`checked N, failed M`)을 출력, 실패가 있으면 exit 1.
6. 마지막에 DB 와 임시 디렉터리를 정리한다(trap).

### C.4 verify 계획

> C.6 이 C.4 에 우선한다(C.4 의 (i) target 수는 fixture 에서 센 값, `TARGETS` 규칙과 요약 줄/exit code 검증 위치는 C.6).

1. bats(DB 없음): `TARGETS` 와 `LIST_ONLY=1` 로, 임시 git 저장소에서 (a) 서비스 migrations 파일 변경 → 그 서비스 1개, (b) timeline `migrations/mysql` 변경 → mysql target, (c) timeline `migrations/` 직속과 rag 변경 → 제외 메시지와 smoke target, (d) 두 서비스 동시 변경, (e) 서비스 migrations 가 아닌 파일 변경(스크립트 변경) → smoke, (f) 서비스 디렉터리 삭제 → 건너뜀, (g) 이름 도출 규칙(`bin-call-manager` → `call-control`), (h) `*.down.sql` 만 삭제 → target 으로 잡힘, (i) merge-base 없음 → 전체(27 target 이며 제외 대상은 목록에 없음), (j) main 경로에서 `HEAD~1` 없음 → 전체.
2. 로컬 실측(scratch `mariadb:12.3` 컨테이너): 전체 모드 27 target 통과(GREEN). VOIP-1686 수정 이전 `bin-email-manager` `000002` 를 가진 scratch 복사본에서 같은 스크립트가 해당 target 을 FAIL 로 만든다(RED, Error 1267 출력 확인).
3. `ruby .circleci/scripts/check-auto-deploy-config.rb` 와 `bats .circleci/tests/*.bats docs/reference/tests/*.bats` 통과.
4. CI: PR 의 `service-migration-collation` job 결과(smoke target 으로 실행됨, `.circleci` 변경이므로)와 첫 실행 시간을 PR 에 기록.

### C.5 설계 보정 (설계 1회차 반영)

> bats 의 "요약 줄과 exit code" 항목은 C.6 으로 이동(로컬 실측).

- 변경 파일 목록: `git diff -z --no-renames --name-only <범위>` 를 쓰고 `-z`(NUL 구분)로 읽는다. 따옴표 처리된 비ASCII 경로와 rename 의 old path 누락으로 target 이 조용히 빠지는 fail-open 을 막는다.
- DESTRUCTIVE 탐지는 수동 스크립트와 같이 `<migrations 디렉터리>/*.up.sql` 의 줄 시작 `-- DESTRUCTIVE:` 만 본다(`migrations_test.go` 등 다른 파일은 보지 않는다).
- 안전장치: 기본 `DB_HOST` 는 127.0.0.1 이고, 루프백이 아닌 호스트는 `ALLOW_REMOTE_DB=1` 이 없으면 거부한다. `CREATE DATABASE` 는 `IF NOT EXISTS`/`DROP` 없이 실행하고 이미 있으면 FAIL 한다. trap 은 이 실행이 만든 DB(성공한 CREATE 만 기록)만 지운다. DB 이름은 backtick 으로 감싸고 `c_<name>`(`-` 를 `_`)이다. mysql 클라이언트는 `mysql --protocol=tcp -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER"` 로 붙는다.
- 단계와 FAIL 분류: create, build, up, version, down, version, up, version 의 각 단계 실패와 빈 버전 출력은 모두 FAIL 이다. 요약 `checked N, failed M`, 실패가 있으면 exit 1.
- timeline 규칙: `bin-timeline-manager/migrations/` 아래에서 `mysql/` 밖의 모든 경로는 제외(ClickHouse). 나중에 `migrations/<다른>/` 이 생겨도 서비스 target 으로 오인하지 않는다.
- `LIST_ONLY=1` 은 mysql 클라이언트와 go 사전 점검보다 먼저 종료한다(`shell-tests` 는 `cimg/base` 이미지라 mysql, go 가 없다).
- bats 추가 케이스: `embed.go` 또는 `<name>-control` 없음 FAIL, DESTRUCTIVE 탐지(`*.up.sql` 만, `migrations_test.go` 제외), 비ASCII 경로와 rename 이 target 으로 잡힘, timeline `migrations/<다른>/` 제외, 요약 줄과 exit code. 임시 git 저장소의 커밋은 `git -c user.name=test -c user.email=test@example.com commit` 으로 한다(CI 에 identity 가 없음). (i) merge-base 없음과 (j) `HEAD~1` 없음은 원격이 없는 임시 저장소(커밋 1개)로 재현한다: fetch 와 deepen 이 모두 실패해 전체 모드로 간다.
- CI 안의 RED/GREEN: PR 에 임시 커밋 하나로 `bin-email-manager` `000002` 를 VOIP-1686 수정 이전 내용으로 되돌려 `service-migration-collation` job 이 빨갛게 되는 것을 확인해 PR 코멘트에 기록한 뒤 그 커밋을 revert 해 GREEN 을 확인한다(최종 PR 에는 revert 로 남고 squash 되므로 결과 diff 에 영향 없음). smoke GREEN 은 이 단계 이전의 배관 확인이다.

### C.6 설계 보정 (설계 2회차 반영, C.3 와 C.5 에 우선)

단계 순서(번호 고정):
1. target 결정(C.3 1항, C.5 의 `-z --no-renames`). `TARGETS` override 도 같은 제외 규칙과 존재 검사를 거친다(제외 대상이 들어오면 skipped, 존재하지 않는 디렉터리는 skipped).
2. target 검증: `migrations/embed.go`(mysql target 은 `migrations/mysql/embed.go`)와 `cmd/<name>-control` 존재, `*.up.sql` 이 1개 이상이고 번호가 000001부터 빠짐없이 이어짐(서비스 migrations 규약, `docs/workflows/service-owned-migrations.md`)을 확인한다. N = 가장 큰 번호 = up 파일 수. `*.up.sql` 에 줄 시작 `-- DESTRUCTIVE:` 가 있으면 destructive=1. 위반은 그 target 의 `validate` FAIL.
3. `LIST_ONLY=1`: 2단계까지 수행하고 target 마다 `target=<dir> n=<N> destructive=<0|1> validate=<ok|FAIL>` 한 줄을 출력, 검증 FAIL 이 있으면 exit 1, 없으면 exit 0. mysql 클라이언트, go, DB 가 필요 없다(`shell-tests` 에서 실행 가능).
4. 사전 점검: `mysql` 또는 `mariadb` 클라이언트(`command -v mysql || command -v mariadb`, 둘 다 없으면 명확한 메시지로 exit 1), `go`, 루프백 DB(`127.0.0.1`, `localhost`, `::1`; DSN 에서 `::1` 은 `[::1]`). 루프백이 아니면 `ALLOW_REMOTE_DB=1` 필요. 접속 대기 120초.
5. target 마다 순차, 첫 실패 단계에서 그 target 을 중단하고 `FAIL <target> <step>` 을 기록, 다음 target 계속: create(이미 있으면 FAIL, 메시지에 수동 정리 `DROP DATABASE` 안내) → precondition(`information_schema.SCHEMATA.DEFAULT_COLLATION_NAME` 이 `uca1400` 을 포함하지 않으면 FAIL; 검사의 전제가 깨진 것을 조용히 통과시키지 않기 위함) → build → up → version(비어 있으면 FAIL, N 과 다르면 FAIL) → [destructive=0 일 때만] down 을 N회(첫 실패에서 중단) → version(0) → up → version(N). destructive=1 이면 `SKIP down/up (DESTRUCTIVE)`.
6. 요약: `checked <target 수>, failed <실패한 target 수>`, 실패가 있으면 exit 1. trap 은 이 실행이 만든 DB 와 임시 디렉터리만 정리.

테스트 보정:
- bats(DB 없음)는 `LIST_ONLY=1` 로 목록, 제외, 검증 FAIL(embed.go 또는 control 없음, up 파일 없음, 번호 gap), destructive 탐지(`*.up.sql` 만), 이름 도출, 비ASCII 경로와 rename, timeline 하위 규칙, 삭제 건너뜀, `TARGETS` override 를 검증한다. git 범위 분기는 임시 저장소로: 성공 경로(`file://` 원격이 있는 저장소의 PR 삼중점 diff, `--depth=1` 클론에서 `--deepen` 으로 merge-base 해석, main 경로 `HEAD~1` 성공)와 fail-safe 경로(원격 없는 저장소: fetch, deepen, `HEAD~1` 모두 실패 → 전체). 전체 모드의 target 수 단언은 fixture 에서 센 개수와 비교한다. 임시 저장소의 커밋은 `git -c user.name=test -c user.email=test@example.com` 으로 한다.
- 요약과 exit code, 단계 순서, precondition 은 DB 가 필요하므로 bats 가 아니라 로컬 실측으로 검증한다: 전체 27 target GREEN, 수정 전 `000002` 로 RED(해당 target 이 `up` 단계에서 1267 로 FAIL, 요약 failed=1, exit 1), 이미 존재하는 DB 로 create FAIL, 가짜 collation(`--collation-server` 지정 서버)으로 precondition FAIL.
- CI 안 RED 의 인정 기준: 실패 로그에 `bin-email-manager`, 단계 `up`, `1267` 이 모두 있어야 한다(네트워크, 빌드 오류로 빨간 것은 RED 증거가 아니다).
- 첫 CI 실행에서 확인하는 항목: `mariadb-client` 가 `mysql` 이름을 제공하는지, 27 target 순차 `go build` 시간, `go build` 가 별도 vendor 없이 되는지.

### C.7 설계 보정 (설계 3회차 비차단 지적 반영)

- 비-LIST 모드에서 validate FAIL 인 target 은 failed 로 집계하고 C.6 5항을 건너뛴다.
- `TARGETS` override 사용 시 smoke 로 대체하지 않는다. 모두 skipped 이면 `checked 0, failed 0` 과 메시지를 출력하고 exit 0.
- bats 의 임시 저장소는 `git init -b main` 으로 만든다.
- `LOCAL_SCRATCH_DB` 는 두지 않는다(C.6 4항에서 삭제). 루프백 기본값과 `ALLOW_REMOTE_DB` 만 유지한다.
- 첫 CI 실행 확인 항목에 클라이언트의 TLS 기본 검증 여부를 더한다.
- `config_work.yml` 매핑의 비용이 크면 그 매핑만 떼는 것을 첫 실행 시간 기록 뒤 판단한다.
