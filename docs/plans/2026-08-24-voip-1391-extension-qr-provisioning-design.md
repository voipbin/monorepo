# VOIP-1391: Extension QR Provisioning (Linphone) — Design

- Date: 2026-08-24
- Ticket: VOIP-1391
- Status: design review loop 대상
- 선행 문서: 이슈 분석(리뷰 4회, 2연속 Approval 통과). 본 문서는 그 분석의 미결정 항목 7건을 확정한다.

## 1. 목표

square-admin extension 상세 페이지에서 QR 코드를 생성하고, Linphone 모바일 앱으로 스캔하면
타이핑 없이 SIP 계정(도메인/사용자명/비밀번호/전송)이 자동 등록되게 한다.

비목표 (v1 제외):
- Zoiper/Grandstream 등 타 소프트폰 지원 (전용 형식이라 불가 판정)
- ha1 해시 기반 프로비저닝 (후속 개선 후보)
- TLS transport 기본값 (후속: *.reg.voipbin.net 인증서 검증 후)

## 2. 전체 흐름

```
[square-admin]                  [bin-api-manager]                [Redis]        [registrar-manager]
 Generate QR 버튼
  → POST /v1.0/extensions/{id}/provisioning-token   (관리자 인증 + 소유권 검증)
                                 토큰 생성(32B hex)
                                 token→extension_id 저장(TTL 10분) →
  ← { token, url, expire }
 QR 렌더링(qrcode.react)

[Linphone 앱] QR 스캔 → GET https://api.voipbin.net/provisioning/extension?token=...
                                 regex 검증 → Redis 조회 ──────────→
                                 RegistrarV1ExtensionGet(extension_id) ───────→ (기존 RPC)
                                 lpconfig XML 렌더링
  ← application/xml (proxy_0 + auth_info_0 + misc)
 계정 자동 등록 완료
```

## 3. 확정 결정 사항 (이슈 분석 미결정 항목 1~7)

### 3.1 토큰 저장 아키텍처: A안 (api-manager 캐시) — 항목 6

- `bin-api-manager/pkg/cachehandler`에 도메인 메서드 2개 추가:
  `ProvisioningTokenSet(ctx, token, extensionID, ttl)`, `ProvisioningTokenGet(ctx, token) (uuid.UUID, error)`
  (bin-agent-manager PasswordResetToken 패턴 준용. Delete는 v1에서 호출처가 없으므로 추가하지 않고,
  §6의 후속(비밀번호 변경 시 토큰 무효화)에서 필요해질 때 도입한다).
- Redis key: `api-manager.provisioning_token.<token>`, value: extension UUID 문자열, TTL 10분.
- 자격증명 조회는 기존 `RegistrarV1ExtensionGet` 재사용. registrar-manager 무변경, 신규 RPC 0개.
- 배선 참고: cachehandler 인스턴스는 현재 dbhandler에만 주입되므로, servicehandler 생성자에
  cachehandler를 추가 주입한다 (생성자 시그니처 변경 1건).
- B안(registrar-manager 저장, 전례 일치) 대비 선택 이유: 스코프 최소(단일 서비스 변경),
  토큰은 API 계층의 단명 관심사로 자격증명 원본 저장과 성격이 다름.

### 3.2 토큰 정책 — 항목 1, 3

- 토큰: `crypto/rand` 32바이트 → 64자 hex. TTL 10분(상수 `ProvisioningTokenTTL`).
- **TTL 내 다회 사용 허용 (단일 사용 아님)**. 근거:
  - Linphone은 URL 저장 후 앱 재시작 시 재접속하므로 1회용이면 스캔 직후부터 실패 경고 유발.
  - 스캔 실패/재시도(카메라 초점 등) 현실 UX.
  - 보안 가중치는 10분 창 + 64 hex 랜덤 + RateLimit + 로그 마스킹으로 상쇄. 유출 시 피해가
    발신 과금으로 직결되는 점은 인지하되, 10분 창 내 토큰 URL 유출은 QR 화면 유출과 등가이며
    이는 기존 상세 화면(비밀번호 평문 표시) 유출과 동일한 운영 리스크 수준.
- **XML에 `misc/transient_provisioning=1` 포함** — 항목 3. 최신 SDK는 프로비저닝 URI를 영구 저장하지
  않게 되어 TTL 만료 후 재접속 실패 경고를 예방. 구버전 클라이언트는 이 키를 무시(병합 의미론상 무해)
  하며, 그 경우 앱 재시작 후 1회 실패 경고가 뜰 수 있음을 UI 도움말에 명시.

### 3.3 공개 엔드포인트 — 항목 4, 5, 7

- 발급(인증): `POST /v1.0/extensions/{id}/provisioning-token` — 항목 5.
  - 핸들러: `server/extensions.go`의 `PostExtensionsIdDirectHashRegenerate`와 동일 구조.
  - servicehandler: 기존 extension 메서드들과 동일하게 `a.IsDirect()`면 즉시 거부
    (`ErrDirectAccessNotSupported`. direct 토큰 소지자가 자격증명 URL을 발급하는 경로 차단) 후,
    `extensionGet(ctx, id)` → `hasPermission(CustomerAdmin|CustomerManager)` 2단계 소유권 검증,
    토큰 생성/저장.
  - 응답: `{ "token": "...", "url": "https://api.voipbin.net/provisioning/extension?token=...", "expire": "<RFC3339>" }`
    (프론트가 URL을 재조립하지 않게 서버가 완성형 제공)
  - 공개 베이스 URL은 bin-api-manager config에 **신규 필드**로 추가한다 (현재 부재).
    전례: bin-agent-manager의 `PasswordResetBaseURL`(기본값 `https://api.voipbin.net`, env 오버라이드).
    동일하게 flag + env `API_PUBLIC_BASE_URL` 바인딩, 코드 기본값 `https://api.voipbin.net`.
    프로덕션은 코드 기본값을 사용하므로 배포 env 선등록 불필요(셀프호스팅만 오버라이드).
- 공개(무인증): `GET /provisioning/extension?token=<64hex>` — 항목 7: 쿼리 파라미터.
  - `runListenHTTP()`에서 `provisioning := app.Group("/provisioning")` + 전용 RateLimit
    (신규 티어 `provisioning_public`, 코드 기본값 rps 5 / burst 10, env 오버라이드 가능). Authenticate 없음.
  - 토큰 regex `^[0-9a-f]{64}$` 사전 검증(패키지 레벨 컴파일). 실패/미존재/만료 모두 동일한
    bare `c.AbortWithStatus(400)` (열거 방지, GetPasswordReset 전례).
  - Redis 조회 성공 후 `RegistrarV1ExtensionGet` 응답의 `tm_delete`가 설정된(삭제된) extension이면
    동일한 bare 400 (TTL 창 내 삭제된 extension의 자격증명 서빙 방지).
  - 성공: `c.Data(200, "application/xml; charset=utf-8", xml)`.
  - `X-Linphone-Provisioning` 헤더는 요구하지 않음(구버전 호환). 값이 있으면 로깅만.
- **로그 마스킹** — 항목 7: gin 로거는 핸들러 체인 실행 전에 `RawQuery`를 로컬 변수로 캡처하므로
  그룹 미들웨어로는 마스킹이 불가능하다. 현재 `cmd/api-manager/main.go`는 `gin.Default()`를 사용하므로
  **엔진 레벨 변경**이 필요하다: `gin.New()` + `gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths:
  []string{"/provisioning/extension", "/v1.0/provisioning/extension"}})` + `gin.Recovery()`로 교체해
  Default()와 동일 동작을 유지하되 해당 경로만 액세스 로그에서 제외한다 (404 스텁 경로
  `/v1.0/provisioning/extension`도 쿼리가 로깅되므로 함께 제외). 이 변경은 전체 라우트의 로거 구성을
  건드리므로 구현 시 기존 로그 출력 형식이 유지되는지 확인하고, 프로비저닝 요청 시 토큰이 stdout에
  남지 않음을 검증 항목(§4)으로 확인한다.
- SkipPaths는 해당 경로의 액세스 로그를 통째로 제거하므로(관측 공백), 공개 핸들러가 **자체 구조화
  로그 1줄**(extension_id, 결과 코드, 클라이언트 IP. 토큰 미포함)을 남겨 관측성을 보전한다.
  이는 수용된 트레이드오프로 기록한다.
- OpenAPI 스펙 — 항목 4: 두 엔드포인트 모두 선언한다.
  - `openapi/paths/extensions/id_provisioning_token.yaml` (발급. 스펙에는 security 스킴이 전역적으로
    없으므로 기존 관례대로 401/403 응답 선언으로 표현)
  - `openapi/paths/provisioning/extension.yaml` (공개. `auth/password-reset.yaml` 전례처럼 설명 문구로
    "unauthenticated" 명시, 응답 `application/xml`. `security:` 키는 스펙 전체에 존재하지 않는 관례이므로
    도입하지 않음)
  - 공개 경로는 codegen이 `/v1.0/provisioning/extension` 중복 라우트를 만들므로
    `server/provisioning.go`에 404 스텁 추가 (auth_boot.go 전례).
  - 스키마: `ApiManagerExtensionProvisioningToken` (token/url/expire). 응답 예시는 실값 형태.
    참고: 기존 스키마 접두어는 모두 `<OwningService>Manager<Entity>` 형태이며 `ApiManager` 접두어는
    최초 도입이다. 이 리소스는 registrar-manager가 아닌 api-manager가 소유하는 API 계층 개념이므로
    의도적 신규 네임스페이스로 선언한다.

### 3.4 XML 내용 — 항목 2 (transport: UDP)

serving 값: `RegistrarV1ExtensionGet`은 **내부 모델**(`rmextension.Extension`)을 반환하므로
`DomainName`을 직접 읽지 말고 반드시 `ConvertWebhookMessage()`를 거친(또는 동일한 Realm 우선 규칙을
적용한) 값을 사용한다. 레거시 행은 `DomainName`에 고객 UUID가 들어 있어 원시 서빙 금지.
사용 필드: `domain_name`(Realm 우선), `extension`, `username`, `password`.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<config xmlns="http://www.linphone.org/xsds/lpconfig.xsd">
  <section name="misc">
    <entry name="transient_provisioning" overwrite="true">1</entry>
  </section>
  <section name="proxy_0">
    <entry name="reg_proxy" overwrite="true">&lt;sip:{domain_name};transport=udp&gt;</entry>
    <entry name="reg_identity" overwrite="true">sip:{extension}@{domain_name}</entry>
    <entry name="reg_expires" overwrite="true">3600</entry>
    <entry name="reg_sendregister" overwrite="true">1</entry>
    <entry name="publish" overwrite="true">0</entry>
  </section>
  <section name="auth_info_0">
    <entry name="username" overwrite="true">{username}</entry>
    <entry name="domain" overwrite="true">{domain_name}</entry>
    <entry name="passwd" overwrite="true">{password}</entry>
    <entry name="realm" overwrite="true">{domain_name}</entry>
  </section>
</config>
```

- transport=udp 명시. 근거: Kamailio 외부 리스너는 udp/tcp 5060 + tls 443/5061이 있으나,
  고객 도메인(*.reg.voipbin.net)에 대한 TLS 인증서 유효성이 미검증이며 현재 실증된 등록 경로는 UDP.
  TLS 전환은 후속 티켓.
- 값 이스케이프: 위 XML은 **최종 출력 형태**다(꺾쇠가 엔티티로 이스케이프된 상태). 구현은
  Go `encoding/xml` 마샬링 단일 경로로 하며, 구조체 필드에는 원시 값(`<sip:...;transport=udp>`,
  비밀번호 원문)을 넣어 인코더가 이스케이프를 전담한다. 수동 이스케이프와 혼용하지 않는다
  (이중 이스케이프 방지). 비밀번호에 `<`, `&`, `"`가 포함된 케이스를 골든 테스트로 고정.
- proxy_0/auth_info_0 인덱스 충돌(기존 계정 보유 앱에서 스캔 시 계정 0 대체) 동작은 UI 도움말에 명시.
- `sip/default_proxy`는 lpconfig가 기본 계정 부재 시 자동 설정하므로 XML에 포함하지 않는다.
  신규 프로비저닝된 계정이 활성 계정이 되는지는 §4 실기기 검증에서 명시적 확인 항목으로 둔다.

### 3.5 프론트엔드 (square-admin)

- `extensions_detail.js`의 Extension Configuration 카드 하단에 "Softphone QR Provisioning" 섹션 추가
  (DirectHashSection과 유사한 독립 컴포넌트 `ProvisioningQrSection.js`).
- 동작: "Generate QR" 버튼 → `Post('extensions/{id}/provisioning-token')` → 응답 url로
  QR 렌더링 + 만료 카운트다운(10:00) 표시 + 만료 시 QR 흐림 처리 및 재생성 버튼.
- 라이브러리: `qrcode.react` (신규 의존성 1개, SVG 렌더, 유지보수 활발).
- 문구: "Scan with the Linphone mobile app. The QR code expires in 10 minutes."
  + 주의: "Scanning on a Linphone app that already has an account will replace account 0."
- 접근성/상태: 로딩 스피너, 실패 시 ActionFeedback 사용(기존 규칙 준수, alert 금지).

## 4. 테스트 전략

백엔드 (Go, 각 대상 서비스에서 `go test ./...`):
- cachehandler: ProvisioningToken Set/Get + TTL 동작 (miniredis 또는 기존 테스트 관례 준용).
- servicehandler: 토큰 발급 성공 / 권한 없음 / 타 고객 extension 거부 / extension 미존재.
- XML 렌더러: 필드 매핑, 특수문자 이스케이프(`<`, `&`, `"` 포함 비밀번호), 골든 XML 비교.
- 공개 핸들러: 유효 토큰 200+XML / 형식 불일치 400 / 미존재 400 / 만료 400 / 삭제된 extension(tm_delete
  설정) 400 (모든 실패 케이스의 응답 본문 동일성 확인).
- 발급 핸들러(server 계층): 정상 / 인증 없음 401.
- OpenAPI: go generate 후 빌드 통과, 404 스텁 동작.

프론트엔드 (Jest):
- ProvisioningQrSection: 생성 호출/렌더/만료 카운트다운/에러 상태.
- extensions_detail 통합: 섹션 렌더 확인. 기존 테스트 무회귀.

수동 검증 (실기기):
- 실제 extension으로 QR 생성 → Linphone(Android/iOS 최신) 스캔 → REGISTER 성공 확인(kamailio 로그).
- 프로비저닝된 계정이 기본(활성) 계정으로 설정되는지 확인.
- TTL 만료 후 접속 400 확인. api-manager stdout 액세스 로그에 토큰 미노출 확인(SkipPaths 동작 검증).

## 5. 산출물/PR 구성

- monorepo 1 PR (브랜치 VOIP-1391-extension-qr-provisioning):
  - bin-openapi-manager: paths 2건 + 스키마 + go generate 산출물
  - bin-api-manager: cachehandler 메서드, servicehandler 발급 로직, lib/service 공개 핸들러,
    XML 렌더러(pkg 위치는 구현 시 lib/service 하위 파일로), 라우트 등록(+로거 교체), 404 스텁,
    config 신규 키 3건(RateLimit rps/burst 2건 + 공개 베이스 URL 1건),
    RST 문서(extension_overview/tutorial/struct + quickstart 해당분) + Sphinx 빌드 산출물,
    `docs/routing.md` 공개 엔드포인트 표 갱신, `docs/operations.md` config 표 갱신 (root CLAUDE.md 필수 규칙)
  - 검증: go generate / go test / golangci-lint / Sphinx 클린 빌드
- monorepo-javascript 1 PR (브랜치 VOIP-1391-extension-qr-provisioning):
  - square-admin: ProvisioningQrSection 컴포넌트 + extensions_detail 통합 + qrcode.react 의존성 + 테스트
  - 검증: npm test 기준선 비교 + npm run build

## 6. 리스크와 롤백

- 공개 엔드포인트는 신규 경로라 기존 트래픽에 영향 없음. 문제 시 라우트 등록 1줄 제거로 비활성화 가능.
- gin 로거 교체(gin.Default() → gin.New()+LoggerWithConfig+Recovery)는 전체 라우트의 로깅 경로를
  건드리는 유일한 광역 변경. §3.3의 형식 유지 확인 + §4 stdout 검증으로 완화하며, 문제 시 SkipPaths만
  제거하면 Default()와 동일 동작으로 복귀.
- 잔여 노출: SkipPaths는 api-manager stdout만 가린다. Cloudflare 등 엣지 계층 로그에는 쿼리 스트링
  포함 전체 URL이 남을 수 있음(10분 TTL로 완화, 인지된 잔여 리스크로 기록).
- 프론트 섹션은 독립 컴포넌트라 장애 시 해당 카드만 영향.
- 비밀번호 변경 후 TTL 잔여 창 동안 이전 자격증명 서빙 가능(10분 한정, 문서화). 필요 시 후속으로
  비밀번호 변경 시 해당 extension 토큰 무효화(토큰 역인덱스) 추가 가능하나 v1 스코프 제외.
