# 전 서비스 정적 빌드 + distroless 런타임 전환 (VOIP-1277 플릿 롤아웃)

날짜: 2026-07-31
상태: 디자인 리뷰 루프 완료 (R1 RC → R2 A → R3 RC → R4 A → R5 A, 2연속 Approval)
선행: [2026-07-31-api-manager-static-build-distroless-design.md](2026-07-31-api-manager-static-build-distroless-design.md) (파일럿, 설계/코드 리뷰 루프 통과, 이미지 1.08GB → 139MB 실증)

## 1. 목표

파일럿(bin-api-manager)에서 검증된 패턴을 모노레포 전체 Go 서비스로 확대한다. 단일 PR로 진행한다(파일럿 PR #1151은 본 PR로 대체).

## 2. 전수 조사 결과 (37개 Dockerfile)

### 2.1 현행 패턴

- **표준 (30개)**: `golang:1.25-alpine` 빌드 → `alpine:latest` 런타임, `COPY --from=build /app/bin /app/bin`, CMD 없음(k8s가 exec-form command 제공).
- **변형**: bin-timeline-manager(runtime `apk add ca-certificates` + `CMD ["./timeline-manager"]` + migrations COPY), bin-rag-manager(migrations COPY), voip-kamailio-proxy(exec-form ENTRYPOINT), bin-api-manager(파일럿에서 이미 전환).
- **특수 (3개, 전환 제외)**: 아래 §3.2.

### 2.2 안전성 확인 (전 서비스 실측)

- **k8s `exec:` probe: 0건.** probe는 api-manager와 hook-manager의 `httpGet`(:443 HTTPS /ping) 2건뿐. initContainers/lifecycle hook 0건.
- **k8s command: 100% exec-list form.** 셸 불필요. (tts-manager의 `sh -c` 사이드카는 별도 python 이미지, pipecat의 사이드카는 §3.2에서 제외 처리)
- **voip 프록시 2종의 외부 소비자 검증(monorepo-voip 실측)**: voip-asterisk-proxy는 voip-asterisk-k8s-{call,conference,registrar}의 사이드카로 `command: ['./asterisk-proxy']` (exec form) 실행, probe 없음. voip-kamailio-proxy는 voip-kamailio-ansible의 docker-compose에서 command/healthcheck 없이 이미지 ENTRYPOINT(exec form, 본 전환에서 보존)로 실행되며 `cap_add: NET_RAW`는 프로세스 capability라 이미지 내용과 무관. 두 이미지를 `FROM`으로 파생하는 이미지는 monorepo-voip 전체에 없음. → 전환 안전.
- **비-vendor Go 코드의 `os/exec` 사용: voip-rtpengine-proxy 1건뿐** (tcpdump, §3.2에서 제외).
- **`time.LoadLocation`/`os/user` 사용: 0건.** tzdata/NSS 우려 없음.
- **`mattn/go-sqlite3`는 `_test.go`에서만 import** → 프로덕션 빌드에 cgo 불필요 (pipecat 제외).
- **`/tmp` 쓰기: 3건** — api-manager, hook-manager(TLS 인증서 파일), tts-control(`/tmp/tts-media` 기본값, CLI 바이너리). distroless/static의 `/tmp`는 world-writable, `readOnlyRootFilesystem` 미설정 → 동작.
- **런타임 상대 경로 파일 의존**: api-manager(docsdev/redoc, 파일럿에서 처리), rag-manager(`./migrations`, 메인 프로세스가 기동 시 실행), timeline-manager(`file://migrations`, timeline-control) → migrations COPY 보존 필수.
- **CA 인증서 일관성 확보**: 34개 alpine 런타임 중 timeline-manager 1개만 명시적으로 ca-certificates를 설치하며, 나머지는 alpine 베이스에 기본 포함된 `ca-certificates-bundle`에 암묵 의존. distroless/static은 CA 번들을 항상 포함하므로 명시/암묵 혼재가 단일 기준으로 정리됨.

## 3. 설계

### 3.1 전환 대상: 34개 (bin-* 32개 + voip-asterisk-proxy, voip-kamailio-proxy)

표준 변환 규칙 (기존 파일 대비 최소 diff):

1. 빌드 명령에 `CGO_ENABLED=0` 추가: `go mod vendor && CGO_ENABLED=0 go build -o /app/bin/ ./cmd/...`
2. 런타임 스테이지: `FROM public.ecr.aws/docker/library/alpine:latest` (또는 `alpine:latest`) → `FROM gcr.io/distroless/static-debian12`
3. 기존 LABEL, WORKDIR, COPY 구조 유지. migrations COPY(rag, timeline), CMD(timeline, exec form), ENTRYPOINT(kamailio-proxy, exec form) 그대로 보존.
4. timeline-manager의 `RUN apk --no-cache add ca-certificates` 삭제 (distroless가 CA 번들 포함).
5. bin-api-manager: 파일럿 상태 유지하되 빌드 베이스를 `golang:1.25-bookworm`(+무의미해진 base 스테이지)에서 표준 `golang:1.25-alpine` 단일 빌드 스테이지로 통일. (코드 리뷰 R3에서 vestigial base 스테이지로 지적된 항목의 해소이기도 함)

### 3.2 전환 제외: 3개

| 서비스 | 사유 | 처리 |
|---|---|---|
| bin-pipecat-manager | `github.com/zaf/resample`(libsoxr cgo) 필수 + 런타임이 Python 3.12(torch/ffmpeg) + 동일 이미지로 python 사이드카 실행 | 현행 유지. 별도 최적화는 독립 티켓 |
| bin-dbscheme-manager | Go 서비스 아님 (python/alembic 빌더 → mysql:8.0 시드 이미지) | 현행 유지 |
| voip-rtpengine-proxy | 런타임에 `tcpdump` 실행 파일 필요 (`exec.Command`, 동적 링크라 distroless/static 불가) | 현행(alpine) 유지. distroless/base+tcpdump 검토는 후속 |

### 3.3 베이스 이미지 정책

파일럿과 동일: `gcr.io/distroless/static-debian12` (저장소명으로 debian 세대 고정, 태그는 암묵 `:latest` 가변, digest 고정 없음 — 현행 `alpine:latest`와 동일한 가변성 수준이므로 후퇴 아님. digest 고정은 플릿 안정화 후 별도 검토).

## 4. 검증 계획

1. **전 서비스 `CGO_ENABLED=0 go build ./...`**: 34개 대상 서비스 전부 로컬 실행, 실패 0건 확인 (cgo 의존 잔존 검출).
2. **전 서비스(34개) docker build 로컬 완주**: 변환된 모든 Dockerfile을 머지 전에 로컬에서 빌드해 실패 0건 확인. (readinessProbe가 없는 32개 서비스는 배포 시 안전망이 약하므로, 머지 전 컨테이너 수준 검증을 전수로 수행)
3. **스모크 실행 샘플**: bin-call-manager(표준), bin-timeline-manager(migrations+CMD), voip-kamailio-proxy(ENTRYPOINT), bin-hook-manager(/tmp TLS+443), bin-rag-manager(기동 시 migrations 실행), bin-trigger-sender(CronJob 소비, 아래 §5 참고) 기동 확인(의존성 부재로 인한 연결 실패까지가 정상 범위).
4. **이미지 크기 비교**: 표준 서비스 1종의 전후 크기 기록 (alpine 런타임은 원래 작으므로 api-manager만큼 극적이지 않음. 주된 이득은 셸/패키지매니저 제거와 CA 번들 일관성).
5. Go 코드 무변경이므로 모노레포 검증 워크플로우는 원칙적으로 대상 외. 예외: voip-kamailio-proxy는 `go mod vendor`가 main에서 이미 깨져 있던 상태(x/sys 0.42.0 pin이 bin-common-handler의 0.45.0 요구와 불일치)라 go.mod/go.sum 동기화를 포함하며, 해당 서비스는 전체 검증 워크플로우(vendor/generate/build/test/lint)를 완주함.

## 5. 리스크

| 리스크 | 완화 |
|---|---|
| 특정 서비스의 미발견 런타임 의존 | §2.2 전수 조사(os/exec, 파일 경로, probe, tzdata)로 사전 배제. **전 서비스(34개) docker build를 머지 전 로컬 완주** + 전 서비스 CGO_ENABLED=0 빌드 + 샘플 스모크로 삼중 확인 |
| **readinessProbe 부재 (32/34 서비스)** | probe가 있는 서비스는 api-manager, hook-manager 2개뿐. 나머지는 기동 직후 프로세스가 종료하는 유형의 실패(start-then-exit)가 발생하면 롤아웃이 정지되지 않고 기존 pod를 대체해버릴 수 있음(장애 가능). **완화: 배포는 서비스별 CircleCI approval 게이트로 1개씩 진행하며, 각 서비스 승인 후 `kubectl rollout status` + 기동 로그 확인을 마친 뒤 다음 서비스를 승인**. 문제 발생 시 이미지 태그 롤백. probe 일괄 추가는 별도 티켓으로 후속 (이번 변경과 독립적인 기존 격차) |
| 배포 파이프라인 | 서비스별 CircleCI build-approval 수동 승인 구조 유지. 한 번에 전 서비스를 배포하지 않고 1개씩 승인·확인·진행 (위 readinessProbe 행의 절차). 이미지 태그 단위 롤백 가능 |
| **bin-trigger-sender 예외 (CronJob 자동 반영)** | 이 서비스만 release 잡이 없고 CronJob이 `:latest` + `imagePullPolicy: Always`로 다음 스케줄 실행 때 자동으로 새 이미지를 사용. `kubectl rollout status` 절차가 적용 불가하며, 실패 시 "번호 갱신 잡의 조용한 실패"가 됨. **완화: build approval 후 `kubectl create job --from=cronjob/...`로 1회 수동 실행해 로그를 확인**. 롤백은 태그 재지정이 아니라 이전 이미지의 `:latest` 재푸시 |
| sandbox healthcheck (wget/셸 기반) | sandbox는 digest lock이라 즉시 영향 없음. **단, 다음 lock 갱신 시 전 서비스 healthcheck가 일괄 영향을 받으므로**, sandbox 저장소에서 healthcheck 방식 교체(각 서비스 control 바이너리 또는 대체 방식)를 선행/동시 진행해야 함. 별도 sandbox PR로 처리 |
| pure-Go resolver | 33개 alpine 빌드 서비스는 이미 사실상 pure-Go resolver로 운영 중임(golang:alpine 이미지에 C 컴파일러가 없어 Go 1.20+가 cgo를 자동 비활성화 → 현행 프로덕션 바이너리가 이미 정적/netgo). `CGO_ENABLED=0` 명시는 이 상태를 고정하는 것이지 동작 변경이 아님. 실질 전환은 api-manager(bookworm 빌드) 1개뿐이며 파일럿에서 /etc/hosts + DNS 쿼리 동작 실증 완료 |
