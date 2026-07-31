# api-manager 정적 빌드 + distroless 런타임 파일럿 (VOIP-1277)

날짜: 2026-07-31
상태: 디자인 리뷰 루프 완료 (R1 Request Changes → R2 Approve → R3 Approve, 2연속 Approval)
선행: VOIP-1276 (ZeroMQ/cgo 제거, PR #1150 머지 완료)

## 1. 목표

VOIP-1277의 파일럿으로 `bin-api-manager` 하나만 `CGO_ENABLED=0` 정적 빌드 + 최소 런타임 이미지(distroless)로 전환한다. 패턴이 검증되면 나머지 서비스에 일괄 적용한다(별도 작업).

## 2. 사전 조사 결과 (실측 기반)

### 2.1 실행 환경 제약

| 환경 | 실행 방식 | 셸 필요 여부 | 영향 |
|---|---|---|---|
| GKE (`bin-api-manager/k8s/deployment.yml`) | `command: ['./api-manager']` (exec form), probe는 `httpGet /ping :443 HTTPS` | 불필요 | 변경 없음. distroless 호환 |
| sandbox (`~/gitvoipbin/sandbox/docker-compose.yml`) | `command: ./api-manager` (compose 문자열은 shlex 분해되어 exec form으로 실행, 셸 불필요) | command는 불필요. 단 **healthcheck가 `wget` 사용** → distroless에 없음 | sandbox가 새 이미지 digest로 lock 갱신하는 시점에 healthcheck 교체 필요 (sandbox 저장소, 별도 PR) |
| CircleCI (`docker-build` command) | repo 루트 컨텍스트로 `docker build -f bin-api-manager/Dockerfile .` 후 push | 무관 | Dockerfile 변경만으로 충분, CI 수정 불필요 |

### 2.2 런타임 요구사항 vs distroless/static-debian12 (이미지 실측: docker export로 확인)

| 요구 | 근거 | distroless/static 충족 여부 |
|---|---|---|
| `/tmp` 쓰기 | `cmd/api-manager/main.go`가 `/tmp/ssl_privkey.pem`, `/tmp/ssl_cert.pem` 기록 | ✓ `/tmp` 존재, `drwxrwxrwt` |
| CA 인증서 | GCS 접근, 외부 HTTPS 호출 | ✓ `/etc/ssl/certs/ca-certificates.crt` 포함 |
| tzdata | Go `time.LoadLocation` 사용처 없음(비-vendor 코드 grep 0건). 있어도 무방 | ✓ zoneinfo 1,308개 파일 포함 |
| 정적 자산 서빙 | `docsdev/` HTML, `gens/openapi_redoc` — 바이너리가 `WORKDIR /app/bin` 기준 상대 경로로 서빙 | COPY로 동일 레이아웃 유지 |
| 포트 443 바인딩 | HTTPS 리스너가 443 사용 | distroless/static 기본 태그는 root 실행이므로 가능. `:nonroot` 태그는 443 바인딩 불가 → 이번 파일럿에서는 기본(root) 유지 |
| cgo 미필요 | VOIP-1276에서 `CGO_ENABLED=0 go build ./cmd/...` 성공 실증 | ✓ |
| DNS resolve | RabbitMQ/MySQL/Redis/GCS 호스트명 해석. `CGO_ENABLED=0`이면 glibc `getaddrinfo` 대신 **pure-Go resolver**가 `/etc/resolv.conf`, `/etc/hosts`, `/etc/nsswitch.conf`를 직접 파싱한다(모두 distroless/static에 존재). 이미지 교체가 아닌 **런타임 동작 변화**이므로 스모크 테스트에서 의존 호스트명 해석을 명시 확인 | ✓ (단, §5 리스크에 기재) |

### 2.3 현행 Dockerfile 문제

- 런타임 스테이지가 `FROM base`(= golang:1.25-bookworm, 약 800MB+)를 그대로 사용. 컴파일러 툴체인 전체가 프로덕션 컨테이너에 포함됨.
- `CGO_ENABLED` 미명시 → 기본값(cgo 활성)으로 동적 링크 바이너리 생성.

## 3. 설계

### 3.1 Dockerfile 변경 (bin-api-manager만)

```dockerfile
# base
FROM public.ecr.aws/docker/library/golang:1.25-bookworm AS base

# build
FROM base AS build
WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-api-manager && go mod vendor && CGO_ENABLED=0 go build -o /app/bin/ ./cmd/...

# run
FROM gcr.io/distroless/static-debian12
WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
COPY --from=build /app/bin-api-manager/docsdev/build/html /app/bin/docsdev/build/html
COPY --from=build /app/bin-api-manager/gens/openapi_redoc /app/bin/gens/openapi_redoc
```

결정 사항:
- **런타임 베이스: `gcr.io/distroless/static-debian12` (기본/root 태그)**. `:nonroot`는 443 바인딩이 막히므로 이번 범위에서 제외. 비루트 전환은 포트 구성 변경(8443 + Service 매핑)과 함께 별도 검토.
- **베이스 이미지 태그 정책**: `gcr.io/distroless/static-debian12`에서 `debian12`는 태그가 아니라 저장소 이름의 일부이며, 태그 미지정 시 해당 저장소의 `:latest`(가변 태그)를 pull한다. 저장소 이름으로 debian 세대 고정은 보장되지만 pull 자체가 불변인 것은 아니다. digest 고정은 하지 않는다. 현행 golang 베이스(`golang:1.25-bookworm`)도 동일한 수준의 가변 태그이므로 기준을 유지하는 것이다.
- **docsdev COPY 축소**: 바이너리는 `docsdev/build/html`만 서빙한다(main.go의 gin Static 경로). RST 소스 등 비런타임 파일 약 5.4MB를 제외하고 `build/html`만 동일 상대 경로로 복사한다.
- ENTRYPOINT/CMD는 기존과 동일하게 미지정 (k8s/compose가 command 제공).
- **빌드 경로에 gcr.io 추가**: distroless pull은 익명 접근 가능하며 CI 수정이 불필요하다. 다만 모노레포 빌드 경로에 처음 추가되는 gcr.io 의존이다(빌드 타임 한정 — GKE/sandbox는 최종 이미지를 Docker Hub에서 pull하므로 런타임 gcr.io 의존은 없음).
- 빌드 스테이지 구조(base→build)와 빌드 명령은 기존 유지, `CGO_ENABLED=0`만 추가.

### 3.2 이번 범위에서 하지 않는 것

- 다른 36개 서비스 전환 (파일럿 검증 후 별도 진행)
- `:nonroot` 전환 (포트 구성 변경 필요)
- sandbox healthcheck 교체 (sandbox 저장소 소관, 새 이미지 digest로 lock 갱신 시점에 `wget` → 다른 방식으로 교체. 후보: api-control 기반 CMD probe 또는 metrics 포트 노출 유지 방식 재검토)
- k8s 매니페스트 변경 (변경 불필요함을 확인했으므로)

## 4. 검증 계획

1. **로컬 이미지 빌드**: repo 루트에서 `docker build -f bin-api-manager/Dockerfile -t api-manager-distroless-test .` 성공.
2. **바이너리 정적 링크 확인**: 빌드 스테이지 산출물에 대해 `file`/`ldd`로 정적 바이너리 확인 (또는 `docker run` 기동으로 갈음).
3. **컨테이너 기동 스모크 테스트**: 새 이미지를 `docker run` (RabbitMQ 등 없는 환경이므로 dependency 연결 실패로 종료하는 것까지가 정상 동작 확인 범위 — 바이너리가 exec되고 로그가 나오는지, "no such file"류 링킹/경로 오류가 없는지 확인). **DNS 해석 확인 포함**: 실패 로그가 "호스트명 해석 실패"가 아니라 "연결/인증 실패"인지 구분 확인(pure-Go resolver 검증). 가능하면 sandbox 스택에 이미지를 끼워 `/metrics` 응답과 docsdev/redoc 정적 서빙까지 확인.
3b. **api-control 실행 확인**: `go build ./cmd/...`는 `api-manager`와 `api-control` 두 바이너리를 생성하며, sandbox healthcheck 교체 후보가 api-control이므로 distroless 이미지에서 api-control도 exec되는지 확인.
4. **이미지 크기 비교**: 변경 전후 `docker images` 기록.
5. 모노레포 검증 워크플로우는 Go 코드 무변경이므로 build/test만 재확인.
6. **배포 후 검증** (머지 → CircleCI build-approval 수동 승인 → 프로덕션 배포 파이프라인. 스테이징 없음, 단 readinessProbe + 2 replica + maxUnavailable=0 구조라 기동 불능 이미지는 롤아웃이 정지될 뿐 장애로 이어지지 않음): `kubectl rollout status deploy/api-manager` 확인 후 `/ping`, `/docs`, `/redoc/index.html` 응답 확인.

## 5. 리스크

| 리스크 | 완화 |
|---|---|
| distroless에 없는 런타임 파일 의존 발견 | §2.2에서 알려진 요구사항 전수 실측 확인 완료. 스모크 테스트에서 기동 로그로 재확인 |
| pure-Go resolver 동작 차이 (ndots 검색 도메인 확장, NSS 플러그인 부재 등) | GKE kube-dns 표준 구성에서는 동작 동일 범위. 스모크 테스트에서 의존 호스트명 해석 확인. 문제 시 이미지 롤백으로 즉시 복구 |
| sandbox healthcheck 파손 | sandbox는 digest lock이라 이번 머지로 즉시 영향 없음. lock 갱신 PR에서 healthcheck 교체를 함께 진행 (§3.2에 명시) |
| 프로덕션 롤백 | 이미지 태그 단위 롤백 가능 (이전 digest로 재배포). Go 코드 무변경이므로 이미지만 되돌리면 됨 |
