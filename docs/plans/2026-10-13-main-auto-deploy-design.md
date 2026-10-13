# main 자동 배포 전환 설계 v7 (VOIP-1586 monorepo, SQUARE-101 monorepo-javascript)

작성: 2026-10-13. 대표님 지시: "main 은 머지되면 자동 배포되도록 변경, 티켓 만들고 진행". 방침: 위험이 낮은 서비스부터, 실측 신호 없는 선제 확장 반대, 기존 도구 재사용, 연쇄 후속 티켓 금지(CI 변경은 VOIP-1586, SQUARE-101, 지시문 변경은 ETC-24 범위). R1 결과(A 승인, B와 C 수정 요구)와 R2 결과(2명 승인), R3 결과(A+B 승인, C 수정 요구: MAJOR 2), R4 결과(둘 다 수정 요구), R5 결과(A+B 승인, C 수정 요구: 긴급 범주와 연장 규칙 모순, 알리지 않은 직접 머지 잔여 위험 미명시)를 반영했다.

## 1. 현황 (직접 확인한 사실)
- F1. monorepo `config_work.yml` 은 서비스마다 workflow 하나이고 deploy 가 있는 34개는 `build-approval(approval) -> <svc>-test -> <svc>-build -> <svc>-deploy(main 필터)` 로 균일하다(YAML 파싱으로 확인). 나머지 workflow(bin-common-handler 는 test 만, bin-openapi-manager 는 validate 만, bin-dbscheme-manager 는 lint 와 main 수동 checkpoint, bin-trigger-sender, voip-asterisk-proxy, voip-rtpengine-proxy 는 deploy 없이 build 까지)는 deploy 가 없어 변환 대상이 아니다. 변환 불가 정의 없음(`name:` 지정 job 없음, api-manager 의 `when: or` 는 workflow 레벨). deploy 는 `render-image-tag.sh` 와 `komodo-api-deploy.sh`(set -euo pipefail) 를 쓴다.
- F2. monorepo-javascript 구조는 다르다: `lint` 와 `bats-tests` 는 승인 없이 항상 돌고, `release-approval` 은 `image-build` 만 막는다(`image-build requires [release-approval, lint]`), 이후 `komodo-deploy`(main 필터, `komodo-deploy.sh` 호출). workflow 는 5개이며 square-dev 는 lint 만 있어 배포 대상은 admin, main, meet, talk 4개다. `config.yml` 매핑에서 `.circleci/.*` 변경은 5개 앱 workflow 전부를 만든다.
- F3. 프로젝트 설정 `autocancel_builds=false`, `setup_workflows=true`.
- F4. 승인 없이 배포되면 안 되는 서비스: bin-timeline-manager, bin-rag-manager(기동 시 마이그레이션), bin-trigger-sender, voip-asterisk-proxy, voip-rtpengine-proxy(deploy 없이 build 가 `:latest` 를 push), voip-kamailio-proxy(통화 SIP 엣지).

## 2. CircleCI 동작 실험 결과 (임시 브랜치, echo job, 삭제 완료. R1 A 가 재현 확인)
- E1. 같은 이름 job 두 개를 `requires` 하면 모호성 오류 -> 사용 불가.
- E2. 필터로 빠진 approval 에만 의존하는 job 은 오류 없이 건너뜀.
- E3. `when: matches (pipeline.git.branch)` 가 continuation 에서 동작.
- E4. 한 workflow 에서 `approval(ignore main)`, `test-gated(requires approval, ignore main)`, `test-auto(only main)`, `build(requires [test-gated, test-auto])`, `deploy(only main, requires build)` 구조 동작. main 대역 브랜치는 승인 없이 이어졌고 비-main 은 approval 에서 on_hold, deploy job 은 생성되지 않음. 실험의 "main 대역" 은 정규식 브랜치였고 실제 `only: main` 필터는 시험하지 못했다(구현 PR 과 파일럿 머지에서 검증. 최악의 경우 test-auto 가 안 돌아 build 가 막히는 안전한 방향 실패. 반대로 test-auto 필터가 넓으면 PR 브랜치가 승인 없이 production context 로 build 하므로 정적 검증이 필터가 정확히 `only: main` 인지 검사한다).
- E5. `serial-group` 동작(두 번째 파이프라인이 첫 번째 deploy 종료까지 대기). 직렬화는 serial-start 도착 순서(FIFO)라서 오래된 커밋이 늦게 build 를 끝내면 최신을 덮어쓰는 역전 위험은 구조상 존재(미재현).
- 미검증: monorepo-javascript 에서의 동작(구조가 달라 E4 가 그대로 적용되지 않음. `release-approval` 에 `ignore main` 만 걸고 `image-build requires [release-approval, lint]` 에서 필터로 빠진 approval 이 있는 상태로 image-build 가 도는지 구현 PR 임시 브랜치에서 실측). 체크아웃이 full clone 인지(guard 의 git log 전제)도 구현 PR 에서 실측.

## 3. 설계
### D1-a. monorepo 서비스 workflow 변환 (자동 전환 대상, 31개 중 파일럿부터)
- `build-approval` 에 `filters.branches.ignore: main`.
- `<svc>-test` 에도 `ignore: main` 을 걸고(requires build-approval), `name: <svc>-test-auto`, `filters.branches.only: main` 인 같은 job 을 추가(requires 없음).
- `<svc>-build` 는 `requires: [<svc>-test, <svc>-test-auto]`.
- `<svc>-deploy` 는 기존 `deploy_filters` 유지, `serial-group: << pipeline.project.slug >>/deploy-<svc>` 추가, 앞에 guard 단계(D2).
- 변환은 텍스트 기반 스크립트(PyYAML, ruamel 미설치이고 주석, 앵커 보존이 필요. 재작성 도구는 쓰지 않음). `bin-call-manager` 의 "build-approval 한 번이면 운영 반영" 주석은 전환 서비스에서는 갱신한다.
### D1-b. monorepo-javascript (SQUARE-101)
- test 분리는 필요 없다. 자동 전환 앱의 `release-approval` 에 `ignore: main` 만 걸고 `image-build` 의 requires 는 `[release-approval, lint]` 그대로(필터로 빠진 approval 이 있어도 image-build 가 도는지 구현 PR 에서 실측. 안 돌면 `release-approval` 의존을 main 전용 `image-build-auto` 로 분리).
- `.circleci/.*` 변경은 5개 앱 workflow 전부를 만든다. 자동 전환 앱은 그 변경으로도 main 에서 현재 main 코드를 한 번 배포한다. 전환 PR 자체가 파일럿 앱의 첫 자동 배포다. 이를 알려진 동작으로 명시하고, Phase 1 은 파일럿 앱 1개만 전환해 영향 범위를 한 앱으로 제한한다.
- `komodo-deploy.sh` 에도 같은 guard 를 env 플래그로 넣는다(앱 경로 기준).
### D2. 역전 배포 방지 guard (fail-closed, 건너뛰기를 success 로 숨기지 않음)
- 위치: 각 deploy 의 shared script(`komodo-api-deploy.sh`, js 는 `komodo-deploy.sh`)에 env 플래그 `DEPLOY_REQUIRE_LATEST_FOR_PATHS="<경로들>"` 를 추가(기본 off 이므로 제외 서비스와 kamailio, 기존 bats 영향 없음). 켜면 명시 refspec fetch(필요 시 unshallow, 5절 T1)로 `origin/main` 을 확보한 후 `git log -1 --format=%H origin/main -- <경로>` 가 `$CIRCLE_SHA1` 과 다르면 이 deploy 를 실패 종료한다("superseded by <sha>"). 인라인 step 34개를 늘리지 않는다.
- 경로: `<svc>` 디렉터리(bin-api-manager 는 bin-openapi-manager 포함, 단 파일럿에서는 제외), js 는 `<앱 디렉터리> .circleci` (js 의 `.circleci/.*` 매핑이 앱 workflow 를 만들기 때문. `.circleci` 만 바꾼 커밋도 최신 경로 커밋이 되어 guard 를 통과하며, 전환 PR 이 파일럿 앱의 첫 자동 배포가 된다는 서술과 일치한다. monorepo 는 `.circleci` 가 서비스 workflow 를 만들지 않으므로 서비스 디렉터리만).
- 실패로 종료하는 이유: 건너뛰기를 success 로 처리하면 CircleCI 상 배포된 것으로 보여 QA 의 "마지막 성공 deploy"와 QA-2 확인이 오염된다. 대신 구 커밋 workflow 가 red 로 남지만 truthful 하다(그 커밋은 배포되지 않았고, 더 새 커밋이 같은 서비스 경로를 포함해 배포한다). QA-2 는 `DEPLOY_SUPERSEDED`(QA 는 로그가 아니라 git 으로 판정, T8) 를 보면 그 sha 의 deploy 로 확인 대상을 옮긴다.
- fail-closed: fetch, log 실패, shallow 로 경로 이력 판정 불가도 deploy 실패(구현 PR 에서 full clone 여부 실측하고 bats 에 케이스 추가). 수동 rerun 한 구 커밋도 guard 가 같은 이유로 실패한다(의도).
- 한계: 최신 커밋 B 가 test 에서 실패하면 구 커밋 A 도 이미 guard 로 실패/미배포이므로 서비스는 구 버전에 머문다. 이는 안전한 방향의 정지이며 QA-2 가 deploy 미성공을 감지해 긴급 요청한다(근거: 위 red/실패 신호).
### D3. 사람 승인 유지(자동 전환 제외)
- bin-timeline-manager, bin-rag-manager, bin-trigger-sender, voip-asterisk-proxy, voip-rtpengine-proxy, voip-kamailio-proxy. bin-dbscheme-manager, bin-common-handler, bin-openapi-manager 는 deploy 가 없어 변환 대상이 아니다. 제외 서비스 workflow 는 현재 구조 유지.
- 스키마 의존 변경: 자동 전환 서비스에서는 승인 보류(11(g)(h))가 사라지므로 QA 규칙을 "머지 보류"로 승격한다(ETC-24). 기계적 기준: 해당 서비스 PR 이 `bin-dbscheme-manager` 를 건드리면 머지 불가(11(a) 유지), main 에 "적용 완료" 표시가 없는 dbscheme 변경이 있으면 자동 전환 서비스 PR 의 머지를 보류한다. 순서는 dbscheme 머지 -> 대표님 적용 -> '적용 완료' -> 코드 머지. 이는 대표님 직접 머지에도 적용되는 프로세스 규칙이며 CI 는 막지 못한다(위험으로 명시).
### D4. 단계적 전환과 성공 기준
- Phase 1 파일럿: monorepo 3개 서비스와 js 1개 앱. 선정 기준(대표님 지정): 통화 핵심 경로 제외, 변경 빈도와 영향 범위가 작을 것. 추가 필수 조건(구현 전 dry-run 으로 확인): (i) 서비스 경로의 마지막 성공 deploy revision 과 origin/main 사이에 미배포(보류) 커밋이 없을 것, (ii) 해당 서비스의 migrations 나 dbscheme 대기 변경이 없을 것, (iii) bin-api-manager 는 bin-openapi-manager 결합으로 제외. 최근 측정에서 경로 최신 커밋이 이미 deploy 성공인 서비스는 agent, call, flow, ai, conference, campaign, contact, queue, storage, webhook, pipecat, talk, number 이며 이 중 통화 경로가 아닌 후보는 bin-agent-manager, bin-contact-manager, bin-storage-manager. js 후보는 square-meet. 최종 지정은 대표님이 한다. 변환 스크립트는 allowlist 로만 서비스를 건드린다.
- 성공 기준(모두 확인): (1) main 머지 후 승인 없이 test-auto, build, deploy 가 이어진다, (2) 비-main 브랜치에서 deploy 가 생성되지 않는다, (3) 같은 서비스 연속 머지에서 guard 가 구 커밋을 막고 최신만 배포한다, (4) test 실패 시 build 와 deploy 가 막힌다, (5) QA-2 가 `superseded` 와 정상 deploy 를 구분한다, (5b) 파일럿 서비스에 대해 대표님 직접 머지 1건을 실제로 수행해 default 의 부모 없는 QA-2 생성과 배포 후 검증이 이어지고 test 실패 시 QA-2 가 [URGENT] 로 이어진다, (6) 제외 서비스 workflow 가 변하지 않았다(정적 검증).
- js 의 deploy 필터도 `deploy_filters: &deploy_filters` 앵커(`only: main`)를 쓴다(monorepo 와 같은 이름). 구현은 이 앵커를 재사용하고 정적 검증은 풀어낸 값이 정확히 `only: main` 인지 본다. 구현 PR 전에 파일럿 서비스의 `git log -1 origin/main -- <svc>` 와 `CIRCLE_SHA1` 정합을 dry-run 으로 한 번 확인한다.
- 구현 PR 에 js `image-build` 의 approval 필터 제외 시 동작 실측 결과와 checkout full clone 실측 결과를 남긴다(파일럿 머지 전).
- 파일럿 머지 1건(README 한 줄 등)은 실제 운영 배포가 된다. 의도된 동작이다.
- Phase 2: 파일럿 이상 없음이 확인되면 제외 목록을 뺀 나머지를 같은 스크립트로 전환.
### D4b. 배포 감시와 실패 감지 (Phase 1 전제, 지시문은 ETC-24 에서 D6 항목으로 확정)
- 감시 트리거(직접 머지 포함): 파일럿 서비스 PR 은 QA-1 카드 경유 머지가 원칙이다. 대표님이 파일럿 서비스 PR 을 직접 머지했을 때의 트리거는 '대표님이 default 에게 머지했다고 알리는 것' 하나로 정한다(대표님께 파일럿 서비스 직접 머지 후 한 줄 알림을 요청). default 는 알림을 받으면 지시가 있는 것으로 보고 QA-2 카드를 부모 없이 생성한다(입력: 대상 서비스, 머지 SHA, PR 번호 를 본문에 명시). default 가 직접 머지한 경우도 같다. 이 트리거에 cron 을 쓰지 않는다. 점검 cron 은 읽기 전용 Komodo 키 발급 후 가동되며 가동되면 CircleCI GET 으로 파일럿 서비스 main 최신 커밋의 deploy 상태를 추가 확인한다(보조 안전망일 뿐 Phase 1 의 전제가 아니다).
- 지시문 변경 필수 항목: (a) QA 하드룰 13 에 '부모 없는 QA-2 는 default 가 대상 서비스, 머지 SHA 를 본문에 명시해 만든 경우에만 시작한다' 한 줄, (b) QA 하드룰 14 와 default SOUL 긴급 처리 문구에 '자동 배포 서비스의 main 최신 커밋 deploy 미성공(failed, canceled, 파이프라인 부재(머지 직후 30초 간격 GET 3회 재조회 뒤에도 없는 경우), deploy job failed)'을 '배포 후 확인된 장애 한 가지'와 같은 범주로 명시한다. 상한을 넘겨도 running 인 경우는 장애 범주가 아니며 NEEDS-HUMAN 이다. test-auto 실패는 flaky 여부와 별개로 '서비스가 배포되지 않았다'는 QA-2 확인 결과이므로 하드룰 14 의 '불안정 테스트는 예외 아님'과 충돌하지 않는다. 자동 서비스인데 on_hold 인 approval 이 보이는 오설정도 NEEDS-HUMAN 이다.
- QA-2 대기 창: 자동 서비스는 머지 직후부터 test-auto, build, deploy 를 기다린다. 실측: 최근 main 3건에서 bin-agent-manager 는 test 206/291/222초, build 75~76초, deploy 21~25초(합계 약 5~7분). 파일럿 서비스별 최근 main 실행 3건 합계의 최대치의 3배(상한 30분)를 60초 간격 GET 폴링 창으로 한다(현행 방식 유지).
- 타임아웃과 실패 구분: [URGENT] 는 workflow 가 failed 또는 canceled 이거나, 파이프라인이 없거나, deploy job 이 failed(guard 의 superseded 로 인한 실패는 새 커밋으로 대상 이동 후 판정) 인 경우로 한정한다. 상한을 넘겨도 running 이면 [URGENT] 가 아니라 한 번만 같은 길이로 연장하고, 그래도 끝나지 않으면 NEEDS-HUMAN(kanban_block)으로 보고한다(큐 지연이나 serial-group 대기 오탐 방지).
- superseded 후 대상 이동 종료 조건: 이동한 최신 커밋의 deploy 가 없거나 연속 실패면 위 창과 연장 규칙으로 NEEDS-HUMAN 에 수렴한다(지시문 반영 시 ETC-24 에서 한 줄로 명시).
- 잔여 위험(수용): 대표님이 default 에 알리지 않은 직접 머지는 실패 감지가 없다. 이전에는 build-approval 이 on_hold 로 남아 신호가 됐으나 자동 서비스에서는 사라진다. 파일럿 범위가 서비스 3개와 앱 1개로 작고 실패해도 구 버전이 유지되는 안전한 방향이며 위험은 감지 지연이다. 점검 cron 가동 후 보조 안전망이 생긴다. CircleCI 프로젝트 기본 실패 알림을 켜는 것은 별도 도구가 필요 없는 대안으로 남겨 둔다.
### D5. 정적 검증과 테스트 (구현 PR 범위)
- ruby 로 YAML 을 풀어(앵커 해제) 검사하는 스크립트를 `.circleci/scripts` 에 두고 `shell-tests` 에서 실행: allowlist 서비스는 상기 구조(필터가 정확히 `only: main`/`ignore: main`, serial-group, requires)를 만족하고, 제외 서비스 workflow 는 변경 전과 동일. 호스트에 `circleci` CLI 가 없으므로 CI 에서 `circleci config validate` 가능 여부는 구현 PR 에서 확인.
- bats: guard 케이스(최신/구 커밋, fetch 실패, 경로 없음, 플래그 off 기본), 기존 `komodo-api-deploy.bats` 유지.
### D6. 지시문 변경 (ETC-24, 모드 판별은 config 에서 파생)
- 서비스의 모드(자동/승인)는 SOUL 에 목록을 쓰지 않고 workflow 의 job 목록에 `<svc>-test-auto` 가 있는지로 판별한다(단일 근거).
- 변경 줄(qa/SOUL.md): 3(승인 문구: 자동 서비스는 승인 없음, 제외 서비스는 현행), 11(f)~(i)(제외 서비스에 한정, (g)(h) 는 자동 서비스에서 머지 보류), 12(자동 서비스는 범위 검사 대신 deploy 성공 대기와 guard 실패 대응, 제외 서비스는 현행), 13(자동 서비스 QA-1 은 머지 직후 complete, QA-2 폴링 기준은 머지 후), '머지 전 정밀 검증자' 불릿(.circleci 검토; 하드룰 번호 아님)에 build-approval `ignore main`, test-auto `only main`, serial-group 존재 추가), 15(4)(승인 호출 절차는 제외 서비스에만).
- backend/frontend SOUL 4번 (b): `deploy_filters` 외에 test-auto 와 build-approval 필터 검사 추가. default SOUL: 승인 대기 알림 외에 자동 배포 실패/superseded 알림 경로.
- 적용 순서: 지시문 먼저(모드 판별로 현 config 에서도 안전: test-auto 가 없으면 현행 규칙), config 나중.
### D7. 알려진 위험(설계가 막지 못하는 것)
- 머지가 곧 배포: build-approval 이 대표님 직접 머지나 QA 를 거치지 않은 머지에 대한 2차 관문이었는데 자동 전환 서비스에서는 사라진다. main 보호 설정은 확인하지 않았다(프로세스로만 통제).
- 교차 서비스 배포 순서: 한 PR 이 여러 서비스를 바꾸면 서비스 간 병렬 배포. 계약 추가 PR 은 피호출자를 먼저 머지하는 지침을 QA 규칙에 한 줄 추가.
- 공유 모듈 드리프트: bin-common-handler 나 타 서비스 모델 변경은 해당 서비스를 배포시키지 않으며 다음 배포 때 함께 나간다(현행과 동일, 이번 설계에서 개선하지 않음).
- 롤백 자동화 없음(긴급 수정 정책 유지).
- 권한 분리: CIRCLECI_API_KEY, gh(pchero 단일 계정), 브랜치 보호 없음이 모든 프로필에서 공유된다. 승인이 기술적 분리가 아니었으므로 악화 폭은 '머지 후 승인' 2단계가 1단계로 줄어드는 정도이며, 2차 방어는 개발자 SOUL 의 '.circleci 변경 PR 은 승인하지 않음'과 QA 의 .circleci 검토다. 이 두 항목을 D6 필수 항목으로 한다. 근본 완화(브랜치 보호, 프로필별 키)는 이번 범위 밖.
- `:latest` 역전: main 의 build 단계는 `:latest` 도 push 하며 build 는 serial-group 대상이 아니다. 자동 전환 서비스의 compose 는 `__IMAGE_TAG__`(SHA) 만 쓰므로 영향이 없고, `:latest` 로 소비되는 제외 서비스는 전환하지 않는다.
- js guard 경로에 `.circleci` 를 포함하므로 파일럿이 아닌 앱의 bats 나 스크립트 변경 커밋도 파일럿 앱의 구 커밋 deploy 를 superseded 로 실패시킬 수 있다(안전한 방향, red 증가).
- guard 범위가 서비스 디렉터리 전체라서 README, docs 커밋도 '더 새 커밋'이 되어 구 코드 커밋 deploy 가 실패(red)할 수 있다. 신 커밋이 더 새 코드를 포함해 배포하므로 안전하며 red 가 늘어나는 것만 감수한다.
- 최신 커밋 test 실패로 서비스가 구 버전에 머무는 경우는 D4b 의 QA-2 창과 [URGENT] 연결로 감지한다(15분 제한은 자동 서비스에서 D4b 창으로 대체).
- 수동 rerun runbook: 최신 커밋의 deploy 는 rerun 허용, 구 커밋은 guard 로 실패(의도).
- main 브랜치 보호는 설정되어 있지 않음을 확인했다(`branches/main/protection` 404). 머지가 곧 배포라는 위험은 QA 규칙으로만 완화된다.
## 4. 대표님 결정 필요
- 파일럿 monorepo 3개, js 1개 지정, 제외 목록 확정(voip-kamailio-proxy 제외 동의).

## 5. 구현 계획 v2 (VOIP-1586, 이 PR 범위. SQUARE-101 은 monorepo-javascript 별도 PR)
이 PR 은 Phase 1 파일럿 3개 서비스(bin-agent-manager, bin-contact-manager, bin-storage-manager)만 전환한다. 워크트리 `.worktrees/VOIP-1586-Auto-deploy-on-main-pilot`, 브랜치와 PR 제목 `VOIP-1586-Auto-deploy-on-main-pilot`. 계획 리뷰 R1(A, B 모두 수정 요구: shallow 와 origin/main 확보, 정적 검증기 방식, config 변경 시 검증 미실행, 검증 방법)을 반영했다.

- T0. 사전 읽기 전용 확인(dry-run, 코드 변경 없음): 파일럿 3개 서비스의 `git log -1 origin/main -- <svc>` 와 마지막 성공 main deploy revision 의 정합(미배포 보류 커밋 없음), 최근 main 실행 3건의 test, build, deploy 소요 시간(QA-2 창 근거), 두 결과를 PR 설명과 ETC-24 에 기록한다.
- T1. `.circleci/scripts/komodo-api-deploy.sh` 에 guard 추가. env `DEPLOY_REQUIRE_LATEST_FOR_PATHS`(공백 구분 경로 목록, 기본 비어 있음=off, off 일 때 기존 동작과 요청 형태 동일하며 git 을 호출하지 않는다). 켜진 경우 인자 검증 직후 Komodo 호출 전에 수행한다: (1) `CIRCLE_SHA1` 없으면 실패, (2) 경로 목록을 배열로 분리하고 빈 항목, `-` 로 시작, `:` 포함 항목은 거부, (3) 저장소 안인지 확인(`git rev-parse --git-dir`), (4) 이미 `check-test-conventions` job(config_work.yml 2255 행 부근)에서 검증된 패턴으로 확보: `git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'` 후 `git rev-parse --is-shallow-repository` 가 true 면 `git fetch --no-tags --unshallow origin '+refs/heads/main:refs/remotes/origin/main'` 를 시도하고 실패하면 `--deepen=1000`, 그래도 shallow 이면 실패(`--depth` 는 쓰지 않는다), (5) `git log -1 --format=%H origin/main -- "${paths[@]}"` 의 종료 코드가 0 이 아니거나 결과가 비어 있으면 실패, (6) 결과가 `CIRCLE_SHA1` 과 다르면 `DEPLOY_SUPERSEDED sha=<CIRCLE_SHA1> latest=<결과>` 를 출력하고 실패. 모든 실패는 exit 1 이며 Komodo 를 호출하지 않는다.
- T2. guard 케이스 추가(구현은 별도 파일 `.circleci/tests/komodo-api-deploy-guard.bats` 에 23개, 기존 `komodo-api-deploy.bats` 는 불변)(bats 1.5 이상 전용 기능 금지, 임시 디렉터리는 기존 `setup_test_env` 재사용, fixture 커밋은 `GIT_AUTHOR_*`/`GIT_COMMITTER_*` 환경변수로 작성자 지정, 모두 실제 임시 git 저장소와 bare origin 사용, `PATH` 의 fake curl 로그로 Komodo 미호출 단언): off 기본(기존 테스트 불변)과 off 일 때 git 저장소 밖에서도 통과하고 git 미호출, 일치 통과, 뒤에 다른 경로 커밋만 있어도 통과, 서비스 경로를 바꾼 더 새 커밋이 있으면 실패, 최신 커밋 rerun 통과, 오래된 커밋 rerun 실패, 경로 접두 혼동(`bin-agent-manager` 와 `bin-agent-manager-x`), 다중 경로(앞 경로만 변경), shallow clone 에서 시작해 unshallow 로 통과, unshallow 불가면 실패, fetch 실패, origin/main refspec 이 없는 clone 에서 명시 refspec fetch 로 통과, 경로 이력 없음(빈 결과) 실패, `git log` 자체 실패(저장소 아님, rc 128)와 빈 결과 구분, CIRCLE_SHA1 미설정 실패, 경로 항목 거부(`-x`, `a:b`, 빈 항목).
- T3. `.circleci/config_work.yml` 에 앵커 `non_main_filters: &non_main_filters`(branches ignore main)를 `deploy_filters` 옆에 추가하고 파일럿 3개 workflow 를 설계 D1-a 구조로 변환한다(`build-approval` 에 `<<: *non_main_filters`, `<svc>-test` 에 `<<: *non_main_filters`, `<svc>-test` 를 `name: <svc>-test-auto` 로 한 번 더 두고 `<<: *deploy_filters`(requires 없음), build `requires [<svc>-test, <svc>-test-auto]`, deploy 에 `serial-group: << pipeline.project.slug >>/deploy-<svc>`). deploy job 의 deploy step 에서 `CC_KOMODO_...` 환경변수와 같은 명령에 `DEPLOY_REQUIRE_LATEST_FOR_PATHS="<svc>"` 를 붙인다. 상단 주석에 자동 배포 모드와 승인 유지 모드를 설명한다.
- T4. `.circleci/config.yml` path-filtering 매핑에 `\.circleci/config_work\.yml` 과 `\.circleci/auto-deploy-services\.txt` 를 `run-shell-tests true` 로 추가한다(config 만 바꾼 변경에서도 정적 검증이 돈다. 다른 workflow 는 만들지 않는다).
- T5. 정적 검증기: `.circleci/auto-deploy-services.txt`(allowlist)와 `.circleci/scripts/check-auto-deploy-config.rb`(ruby, YAML 을 풀어 anchor 와 merge key 를 해제한 구조를 검사. 주석과 들여쓰기 오배치는 YAML 파서가 처리. 호환 shim: `YAML.respond_to?(:unsafe_load_file) ? YAML.unsafe_load_file(f) : YAML.load_file(f)`)를 추가하고 `.circleci/tests/check-auto-deploy-config.bats` 가 실제 config 와 fixture 로 검증한다. shell-tests job 의 apt 설치 한 줄에 `ruby` 를 추가한다(mawk 와 같은 줄). 규칙: (a) allowlist 서비스 workflow 구조(위 T3 의 모든 항목, `<svc>-test-auto` 에 requires 가 없음, build requires 가 정확히 두 job, deploy 환경변수 값이 정확히 `<svc>` 이고 `komodo-api-deploy.sh` 와 같은 명령에 있음), (b) allowlist 에 없는 workflow 에 `-test-auto` 이름이나 `build-approval` 의 ignore 필터가 있으면 실패(주석은 무시), (c) 앵커 풀린 deploy 필터 값이 정확히 `only: main`, non_main 은 정확히 `ignore: main`(`[main, /.*/]` 변조 감지), (d) allowlist 중복 또는 config 에 없는 항목 실패. 음성 fixture 는 규칙마다 1개 이상: build-approval 필터 없음 또는 only 로 잘못됨, test-auto 에 requires 있음, test-auto 가 only main 아님, build requires 에 test-auto 빠짐, deploy serial-group 없음, 환경변수 없음 또는 값 불일치, 앵커 값 변조, allowlist 중복과 누락, 비-allowlist 에 test-auto, 양성 fixture 로 주석 안의 test-auto 무시.
- T6. 검증: 로컬 `bats .circleci/tests/*.bats` 전체. 변환 diff 가 파일럿 3개 workflow 와 앵커와 상단 주석 외에는 없음을 `origin/main` 의 `config_work.yml`(기준)과 비교해 비파일럿 workflow 블록이 바이트 동일함을 스크립트로 확인. PR 브랜치 파이프라인에서 config 오류 없음(errored 여부)과 shell-tests 통과를 CircleCI GET 으로 확인. 비-main 에서 deploy 미생성과 build-approval on_hold 는 PR 에 임시 커밋(`bin-agent-manager/` 안 주석 한 줄)을 올려 서비스 workflow 가 만들어진 뒤 GET 으로 확인하고 즉시 revert 한다(승인은 하지 않는다. 최종 diff 에서 빠진다). checkout 의 shallow 여부와 `origin/main` 존재, 명시 refspec fetch 성공은 임시 실험 브랜치(echo job, `cimg/base:2024.02` 에서 `checkout` 후 `--is-shallow-repository`, `git config remote.origin.fetch`, T1 의 fetch 와 `git log -1 -- <svc>` 실행)로 실측해 PR 에 기록하고 실험 브랜치는 원격과 로컬에서 삭제한다.
- T7. 머지 후 확인(QA-2): 이 PR 의 머지는 config_work.yml, config.yml, scripts, tests, docs 변경이라 파일럿 서비스 workflow 를 만들지 않아 배포가 없다(config.yml 매핑 확인됨. T4 의 매핑 추가분은 shell-tests 만 켠다). 첫 실제 자동 배포는 파일럿 서비스의 다음 머지이며 설계 D4 성공 기준 (1)~(6) 을 그때 확인한다.
- T8. ETC-24 범위의 지시문 정정(이 PR 밖, 별도 티켓 없음, 이 PR 머지 전에 적용): CircleCI v2 API 는 단계 로그를 주지 않으므로 QA 하드룰 16(e)의 deploy 로그 문구(`superseded by <sha>`) 의존을 제거하고, deploy 가 failed 이면 `git log -1 --format=%H origin/main -- <서비스 경로>` 가 머지 SHA 와 다를 때 superseded 로 보고 그 SHA 의 파이프라인으로 대상을 옮기도록 정정한다. 설계 D2 의 문구도 같은 내용으로 맞춘다.
- 범위 밖: SQUARE-101(js), Phase 2, 나머지 28개 서비스, 점검 cron.

### 5.1 계획 v3 보강 (R2 MINOR 반영, 위 T0~T8 에 우선)
- T1: `git log` 는 `git --literal-pathspecs log -1 --format=%H origin/main -- "${paths[@]}"` 로 호출해 glob 을 리터럴로 취급한다. 경로 배열이 0개이면(공백뿐인 값 포함) 실패한다(빈 문자열 전체는 off).
- T2: shallow 케이스의 clone 은 `file://` URL 과 `--depth 1` 로 만들고 arrange 단계에서 `--is-shallow-repository` 가 true 임을 단언한다. `git log` 실패와 `--unshallow`, `--deepen` 실패 분기는 PATH 의 `git` shim 이 해당 서브커맨드만 실패시키는 방식으로 테스트한다(`rev-parse --git-dir` 가 비저장소를 먼저 거부하므로 실제 저장소로는 도달 불가). 빈 항목 케이스는 "경로 0개이면 실패"로 바꾼다. glob 문자(`bin-*`) 리터럴 취급 케이스를 추가한다.
- T3: Phase 1 은 3개 서비스 수동 변환과 정적 검증으로 한다(변환 스크립트는 Phase 2 에서 필요성이 확인되면 도입. 설계 D1-a 의 "기계 변환"은 Phase 2 기준).
- T4 보충: `.circleci/tests/voip-kamailio-proxy-komodo.bats` 헤더 주석의 "shell-tests installs only bats and mawk", "KNOWN TRIGGER GAP" 서술을 ruby 추가와 매핑 추가 후의 사실에 맞게 갱신한다. 이 파일이 `config_work.yml` 을 awk 로 읽으므로 변환 후에도 통과하는지 로컬 bats 전체로 확인한다.
- T5 규칙 추가: (e) allowlist 에 없는 workflow 에 `-deploy` job 이 있으면 `build-approval` 이 필터 없는 `type: approval` 이고 `<svc>-test` 가 이를 requires 하며 deploy 가 승인 없이 이어지지 않음(승인 유지 서비스 불변을 상시 검증), (f) timeline, rag, trigger-sender, asterisk-proxy, rtpengine-proxy, kamailio-proxy 가 allowlist 에 있으면 실패(하드코딩 거부 목록). 음성 fixture 추가: `<svc>-test`(gated)의 ignore main 누락 또는 오류, build requires 가 3개 이상이거나 다른 job 포함, serial-group 값이 다른 서비스의 것, deploy job 에 개별 `filters:` 가 있어 앵커를 덮어씀, env 플래그가 `komodo-api-deploy.sh` 와 다른 명령(줄)에 있음, `komodo-api-deploy.sh` 첫 인자가 `<svc>` 가 아님, 비-allowlist 의 build-approval 삭제 또는 필터 추가, 거부 목록 서비스를 allowlist 에 추가.
- T6 보충: 임시 실험은 `cimg/base:2024.02` 와 storage deploy 가 쓰는 `gcp_image` 두 이미지로 각각 수행해 shallow 여부, 클론 크기와 깊이를 기록한다(이미 full 이면 unshallow 비용 없음). 비-main 브랜치 checkout 은 main 의 checkout 을 100% 재현하지 못하므로 main 에서 구 SHA 재실행 경우는 T2 의 로컬 재현으로 보완함을 PR 에 적는다. 비파일럿 불변 비교 기준은 PR 의 merge-base 시점 `config_work.yml` 로 한다.

### 5.2 계획 v4 보강 (R3 MINOR 반영)
- T2: 플래그 on 상태에서 저장소 밖일 때 거부, `--unshallow` 실패 후 `--deepen=1000` 성공으로 통과, deepen 후에도 shallow 이면 실패 케이스를 추가한다. git shim 은 `--literal-pathspecs` 같은 앞선 옵션 때문에 `$1` 이 서브커맨드가 아니므로 인자 전체에서 서브커맨드(`log`, `fetch`)를 찾아 실패를 주입한다.
- T5 (e): requires 그래프로 검증한다(`deploy` 의 requires 가 정확히 `[<svc>-build]`, `build` 가 정확히 `[<svc>-test]`, `<svc>-test` 가 `build-approval` 을 requires, deploy 에 개별 `filters:` 없음, deploy 에서 `build-approval` 까지 도달 가능). fixture 추가: `<svc>-test` 의 requires 에서 build-approval 누락. (c) fixture 는 deploy 필터 앵커 변조와 `non_main_filters` 변조로 나눈다. (d) 는 allowlist 항목이 config 에 없음과, `-test-auto` 가 있는 workflow 가 allowlist 에 없음 두 방향 모두 실패.

### 5.3 실측 결과 (T0, T6)
- T0 (CircleCI GET, 최근 main 성공 실행 3건): test 최대 bin-agent-manager 291초, bin-contact-manager 271초, bin-storage-manager 233초. build 약 75~77초(storage 128~131초). deploy 19~35초. 서비스별 test, build, deploy 합계 최대치는 약 6.2분에서 6.7분이며 3배는 약 18분에서 20분이다(QA-2 창의 하한 10분과 상한 30분 안). 세 서비스의 경로 최신 커밋(a0f6b7942, 9f0c97af7, ec3d0790e)은 모두 deploy 성공 실행으로 확인되어 미배포 보류 커밋이 없다(파이프라인 vcs 필드가 비어 있어 실행 순서로 판단).
- T6 checkout 실측(임시 실험 브랜치, cimg/base:2024.02 와 cimg/gcp:2026.02 두 이미지에서 동일, 실험 후 삭제): checkout 은 shallow 가 아니며(`--is-shallow-repository` false, 6000 커밋, `.git` 30MB, fetch refspec `+refs/heads/*:refs/remotes/origin/*`) `origin/main` 이 이미 있다. 명시 refspec fetch 는 rc 0 이며 unshallow 는 필요 없었다(방어용으로만 남음). 실제 CI 에서 오래된 SHA 로 guard 를 실행하면 `DEPLOY_SUPERSEDED sha=<old> latest=a0f6b7942...` 로 실패하고 Komodo 를 호출하지 않는다. 같은 실험 파이프라인에서 shell-tests(ruby 설치, 검증기, 신규 bats 포함, 측정 시점의 140개)도 성공했다.
- T6 비-main 확인(실제 PR 브랜치, 임시 커밋 `bin-agent-manager/` 마커 파일, 확인 후 제거, 승인은 하지 않음): 파이프라인 #5205 에서 bin-agent-manager workflow 는 `build-approval` on_hold, `bin-agent-manager-test` blocked, `bin-agent-manager-build` blocked 이고 `test-auto` 와 `deploy` job 은 만들어지지 않았다. 같은 파이프라인의 shell-tests 는 성공했다.
- 참고: CircleCI v1.1 API(`/api/v1.1/project/<circleci 슬러그>/<job번호>`)는 단계 로그의 output_url 을 주므로 로그 확인이 필요한 경우 가능하다. QA 의 superseded 판정은 git 기준이다(T8).

### 5.4 구현 반영 사항 (코드 리뷰 R1~R4 반영, 위 T 항목에 우선)
- 구현 파일: guard 는 `komodo-api-deploy.sh`, guard bats 23개는 `komodo-api-deploy-guard.bats`, 검증기는 `check-auto-deploy-config.rb` 와 bats 46개(`check-auto-deploy-config.bats`), 허용 목록 `auto-deploy-services.txt`. 변환은 파일럿 3개 수동 변환(5.1).
- T1 추가: 값에 줄바꿈이 있으면 첫 줄만 읽고 나머지를 무시하지 않고 실패한다.
- T5 구현 규칙 보강: (a) deploy job 의 명령은 `VAR=value` 대입들 다음에 `komodo-api-deploy.sh` 가 오는 형태여야 하며(주석, `env -u`, `true;` 앞붙임 거부) `DEPLOY_REQUIRE_LATEST_FOR_PATHS` 대입이 정확히 한 번이고 값이 `<svc>` 이며 첫 인자가 `<svc>`(접두만 같은 이름 거부)이어야 한다. `run:` 축약형(`- run: <문자열>`)도 읽어 deploy 호출이 두 번 숨는 것을 막는다. deploy 는 production context 를 써야 한다. 허용 목록에 있는데 `jobs` 목록이 없는 workflow 는 거부한다. (e) 승인 유지 workflow 의 deploy 판별은 job 이름이 `-deploy` 로 끝나는 것뿐 아니라 키가 `-deploy` 로 끝나는 것(`name:` 으로 이름을 바꾼 경우)도 포함한다.
- 코드 리뷰 R3 반영: 가드 대입은 토큰화해 변수 이름이 정확히 일치할 때만 인정하고(다른 변수 값 안에 숨긴 경우 거부), 자동 배포 workflow 의 deploy 항목은 키가 `<svc>-deploy` 여야 한다(`name:` 으로 다른 job 정의를 가리키는 경우 거부).
- 코드 리뷰 R4 반영: guard bats 에 detached HEAD 체크아웃(CircleCI 가 CIRCLE_SHA1 을 detached 로 체크아웃하므로 `origin/main` 이 아니라 `HEAD` 를 보면 guard 가 항상 자기 자신을 최신으로 판정해 무력화됨)과 낡은 로컬 `origin/main` 갱신 케이스를 추가했다(변형 시험으로 확인). 승인 유지 workflow 는 deploy 가 의존하는 job(test, build 등 승인 체인 위의 job)에 branch 필터가 있으면 거부하고(main 에서 건너뛰어져 승인 없이 이어질 수 있음), deploy 는 `only: main` 외의 필터를 거부하며, deploy job 판별에 job 정의가 `komodo-api-deploy.sh` 를 호출하는지도 포함한다. 체인 밖 job(예: bin-dbscheme-manager 의 main 전용 migration-applied-checkpoint)은 제한하지 않는다.
- 한계(수용): 검증기는 단순 실수와 흔한 변형을 막는 장치이고, 악의적 우회(별도 gate 이름 등)를 전부 막지는 않는다. 변경은 diff 에서 리뷰어에게 보인다.
