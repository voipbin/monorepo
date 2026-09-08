# VOIP-1501 / VOIP-1498 구현 계획

- 설계: `docs/plans/2026-09-08-direct-token-resource-binding-design.md` (rev11, 디자인 리뷰 완료)
- 작성일: 2026-09-08
- 개정: rev12 (A 단계 구현 완료. 코드 리뷰 4라운드 반영)
- 상태: **A 단계 완료, PR 대기.** B 단계와 배포 게이트는 미착수

## 0. 수용 기준

1. direct 토큰이 자기에게 배정된 리소스 하나에만 접근한다. 같은 위젯을 연 다른 방문자의 리소스에 REST로도 WebSocket으로도 닿지 않는다.
2. `GET /aicalls`가 direct 신원에서 거부된다.
3. `customer_id:<cid>:aicall:` (trailing colon) 구독이 direct에서 거부되고, 관리자·매니저의 동일 문법 구독은 계속 동작한다.
4. direct의 `POST /aicalls`가 `reference_type`/`reference_id`를 무시하고 `none`/`Nil`로 강제한다.
5. 진행 중인 대화가 토큰 갱신(3시간 55분) 시점에 끊기지 않는다.
6. 해시 재발급·direct 삭제·고객사 동결이 다음 갱신에 반영된다.
7. **각 단계 경계에서 살아 있는 모든 토큰은 (a) 계속 동작하거나 (b) 관측 가능한 거부(401·403·409, 그리고 `scope_version >= 2`일 때에 한해 id 불일치)를 받으며, 배포된 클라이언트가 그 전부를 재부팅으로 전환하고, 재부팅이 상한 안에서 동작하는 대화로 수렴한다.**
   - rev1의 "조용히 망가지는 집단이 없다"는 검증 불가능했다.
   - rev2는 관측 가능하게 바꿨으나 **수렴 조건이 없어 무한 재부팅 루프를 만족시켜 버렸다**(라운드 2 CRITICAL). 수렴 절을 추가했고, id 불일치는 강제가 켜진 뒤에만 신호로 인정한다.

## 1. 전제와 결정

### 1.1 사전 결정 사항

| 항목 | 결정 | 근거 |
|---|---|---|
| 401 센티널 | `serviceerrors.ErrAuthenticationRequired` | **`ErrTokenInvalid`는 존재하지 않는다**(`pkg/serviceerrors/sentinels.go:14-24`). `server/error_translate.go:58-62`가 401로 매핑 |
| 403 센티널 | `serviceerrors.ErrPermissionDenied` | 동일 매핑 경로로 403 |
| 중복 사전 확인 | **하지 않는다.** 기본키 제약이 유일한 판정 | 설계 §4.2.2가 사전 확인을 "깨끗한 렌더링용"으로만 규정. 교차 테넌트 오라클 위험을 없앤다. **설계 §7의 "사전 확인을 통과한 경우에도 409" 테스트 항목은 이 결정으로 무효**이며, "PK 충돌이 409로 매핑되는지"로 대체한다 |
| OpenAPI | **변경 필요** (rev2의 "변경 없음"은 오판) | `/auth/boot`은 gin 수기 등록(`cmd/api-manager/main.go:299`)인 **동시에** 스펙에도 있다(`bin-openapi-manager/openapi/openapi.yaml:8416`, `paths/auth/boot.yaml:23` → `AuthBootResponse` at `openapi.yaml:8156`). **경로 추가는 `ServerInterface` 메서드를 생성한다**(`gens/openapi_server/gen.go:11114`, `:23184`). 응답 구조체가 수기인 것과 별개다. stub 없이는 `bin-api-manager`가 컴파일되지 않는다. 그리고 **공개 계약 문서가 stale해진다.** 설계 §4.2의 "OpenAPI에 `id`를 노출하지 않는다"는 내부 RPC DTO에 대한 서술이며 부팅 응답이나 신규 공개 라우트를 다루지 않았다. rev2가 이를 부당하게 일반화했다 |
| DB 마이그레이션 | **없음** | `PrepareFields`가 구조체 `ID`를 그대로 insert (`bin-ai-manager/pkg/dbhandler/aicall.go:26`, `bin-webchat-manager/pkg/dbhandler/session.go:42`) |
| 갱신 시 WS 소켓 | **재연결하지 않는다** | `subscriptionRun`(`pkg/websockhandler/subscription.go:31-38`)이 업그레이드 시 `*auth.AuthIdentity`를 캡처하고, 이후 프레임은 같은 포인터를 재사용한다(`:126-163`, `:165-181`). 토큰을 다시 파싱하는 경로가 없으므로 갱신이 살아 있는 소켓을 무효화하지 못한다. 재연결하면 구독과 in-flight 메시지만 잃는다 |

### 1.2 착륙 단계

**2026-09-08 대표님 결정: 서버는 한 PR로 간다.** rev10까지의 3단계 구성을 병합했다.

| 단계 | 저장소 | 성격 |
|---|---|---|
| A | monorepo | 발급 + 배관 + 409 경로 + **강제**. 한 PR |
| B | monorepo-javascript | 클라이언트 재부팅 계약 |

#### 병합이 가능한 이유 (롤아웃 완료 후 기준)

rev10은 발급과 강제를 함께 배포하면 "최대 4시간 동안 모든 위젯이 죽는다"고 서술했다. 과장이었다.
**롤아웃이 끝난 뒤에는** 새로 부팅하는 방문자가 구버전 프론트로도 정상 동작한다.

- 부팅이 `AllowedResourceID`를 발급한다
- `AIcallCreate`가 그 번호를 지정해 aicall을 만든다. 즉 `aicall.id == AllowedResourceID`
- 구버전 프론트는 `aicall.id`를 그대로 쓰므로(`useChatState.js:134`), 토픽도 경로 파라미터도 전부 일치한다
- `POST /aimessages`의 덮어쓰기도 같은 값이라 무해하다

**"롤아웃 완료 후"라는 단서가 핵심이다.** 롤아웃 중에는 성립하지 않는다. 아래 §1.2.1이 그 창을 다룬다.

#### 1.2.1 롤링 배포 창 (병합 착륙의 실제 비용)

세 서비스 모두 `replicas: 2`이고, CI가 서비스별 워크플로로 갈라져 각자 승인 게이트를 가진다
(`.circleci/config.yml:14-22`). **원자적 배포 단위가 없다.** 따라서 혼재 창이 반드시 생긴다.

혼재 창에서 방문자가 깨지는 경로는 둘이며, **401이 주 경로다**(설계 §9.2와 같은 순서).

| 경로 | 발생 | 결과 |
|---|---|---|
| **A. 구버전 api-manager에서 부팅** | 부팅이 구버전에 걸림 | 토큰에 `AllowedResourceID`가 없다. 다음 요청이 신버전에 닿는 순간 401 |
| **B. 신버전 부팅 + 구버전 처리** | 부팅은 신버전, 생성이 구버전 api-manager이거나 신버전 api-manager가 구버전 매니저에 RPC | 번호 불일치. 토픽 거부로 소켓 절단, `aimessages` 404 |

**rev11 초판이 이 창에 "약 22%"를 적은 것은 오류였다.** 그 값은 설계 §9.2가 **강제 전용 롤아웃**을,
그것도 **재부팅 계약이 이미 배포된 상태**로 계산한 것이다. 병합 착륙에는 두 전제가 모두 없다.

배포 중간 시점 기준으로 다시 계산하면(2 replica, 세션 어피니티 없음 가정):

- 부팅이 구버전 api-manager (1/2): 사실상 전부 깨짐
- 부팅이 신버전 api-manager (1/2):
  - 생성이 구버전 api-manager (1/2) → 번호 미전달 → 불일치
  - 생성이 신버전 api-manager (1/2) → 구버전 매니저로 RPC (1/2) → 불일치
  - 정상 확률 1/4

**중간 시점 P(깨짐) ≈ 7/8.** 창 전체로 적분해도 큰 비율이며, "일반적인 배포 끊김과 같은 수준"이 아니다.

**G1의 순서 강제가 이 값을 대략 절반으로 줄인다.** 매니저를 먼저 완전히 배포하면 위 B의 RPC 축이 사라진다.

**창의 길이는 수 분이고, 전부 새로고침으로 복구된다.** 감수하는 것은 "짧은 창 동안 그때 부팅한 방문자
다수가 수동 새로고침을 해야 한다"이지 "장시간 장애"가 아니다. B 단계가 배포되면 그 새로고침도 자동 재부팅으로 바뀐다.

#### 1.2.2 롤아웃 완료 후 남는 것 (B 단계가 닫는다)

| # | 깨지는 것 | 범위 | 증상 | 복구 |
|---|---|---|---|---|
| 1 | 배포 순간 열려 있던 대화 | 그 시점 접속자. 구토큰에 `AllowedResourceID`가 없어 401 | 전송 실패 | 새로고침 |
| 2 | 3시간 55분 넘긴 대화 | `scheduleRefresh`가 `/auth/boot`을 다시 불러 **새 번호**를 받는데 대화 번호는 그대로다(`auth.js:57`) | **조용하다.** `websocket.js`가 갱신 시 소켓을 유지하므로 표시는 계속 "Online"이고, 보내야 알게 된다 | 새로고침 |
| 3 | (해당 없음) | rev11 초판은 "종료 후 재시작 → 409"를 적었으나 **도달 불가다.** 두 클라이언트 모두 생성 전에 항상 부팅한다(§3.3 계기 2). 서버 409 경로는 방어적으로 구현하되 실사용 경로가 아니다 | | |

**침묵 여부를 정정한다.** rev11 초판은 "4번만 조용히 깨진다"고 적었으나 반대다.
불일치(§1.2.1의 B)는 `square-main`에서 **가장 눈에 띈다**. 소켓이 끊겨 "Reconnecting" 표시가 계속 남고
전송도 실패 표시가 된다. **조용한 쪽은 위 2번이다.**

**부수 효과: 유계 WS 재연결 폭풍.** 토픽 거부는 소켓을 끊고(`subscription.go:174-177` → `cancel()`),
클라이언트는 `activeTopics`에 낡은 토픽을 든 채 재연결하므로 다시 끊긴다.
10회·백오프 3s/6s/12s/24s로 약 3분간 반복된다. 유계이고 fail-closed이나 기록해 둔다.

**B 단계는 새로 로드되는 페이지에만 적용된다.** 위젯 번들이 `max-age=300`이고,
이미 열려 있는 탭은 구 `auth.js`를 무기한 들고 있다. 그 탭이 바로 위 2번을 맞는 집단이다.

#### 1.2.3 병합이 assurance에 미치는 영향

3단계 구성에서는 배관이 **보안 경계가 되기 전에** 프로덕션에서 무해하게 돌아 봤다.
병합하면 배관 결함(특히 §2.2가 경고하는 `scope`/`BootResponse` 분리)이 곧바로 가용성 사고로 나타난다.
따라서 **§2.2의 단일 출처 규칙과 §2.9의 "응답 == 클레임" 테스트가 병합 착륙에서 더 중요해진다.**

### 1.3 배포 게이트 (강제)

통과 여부와 확인 시각을 §5에 기록한다.

- [ ] **G1. 배포는 순서를 지킨다. 동시 배포가 아니라 순차 배포다.**

      1. `bin-ai-manager`와 `bin-webchat-manager`를 먼저 승인·배포한다.
         Komodo 배포가 terminal success이고 **구버전 replica가 모두 사라졌는지** 확인한다.
      2. 그 다음에 `bin-api-manager`를 승인한다.

      **매니저 먼저가 안전한 이유는 증명 가능하다.** 두 매니저의 요청 DTO에는 오늘 `ID` 필드가 없다.
      구버전 api-manager는 그 필드를 아예 보내지 않고, 신버전 매니저는 `uuid.Nil`로 읽어 기존 생성 경로를 탄다.
      즉 매니저만 먼저 나가는 구간에서는 **아무 변화가 없다.**
      (`PipecatcallID` 선례와 같은 형태다. `bin-ai-manager/pkg/listenhandler/models/request/aicalls.go:44-52`)

      **rev11 초판의 "같은 배포 단위로 나가야 한다"는 구현 불가능했다.**
      CI가 서비스별 path filter로 워크플로를 갈라 각자 승인 게이트와 배포 스크립트를 가진다
      (`.circleci/config.yml:14-22`). 공유 배포 단위가 존재하지 않는다.
      그리고 동시에 시작해도 `replicas: 2`의 순차 recreate 때문에 혼재 창은 남는다.
      **순서가 동시성보다 강하다.** 이 순서가 §1.2.1의 RPC 축을 통째로 제거한다.

- [ ] **G2. api-manager 배포 직후:** 신규 부팅이 정상 동작하는지 프로덕션에서 확인(§4의 첫 두 항목).
      실패하면 즉시 롤백한다. 단 롤백 비용은 §1.4를 먼저 읽는다.

- [ ] **G3. B는 A 직후에 착륙시킨다.** §1.2.2가 열려 있는 기간을 짧게 유지한다.
      A와 B 사이 소킹은 rev10의 3단계 구성에서만 의미가 있었다. 병합에서는 짧을수록 좋다.

### 1.4 롤백

| 단계 | 가용성 관점 | 보안 관점 |
|---|---|---|
| A | 되돌리면 강제와 필드가 함께 사라지고 병합 이전 동작으로 복귀한다. 롤백 배포 중에는 §1.2.1과 같은 혼재 창이 다시 생긴다 | **VOIP-1501과 VOIP-1498이 다시 열린다** |
| B | 클라이언트가 재부팅 계약을 잃을 뿐 서버는 그대로다. §1.2.2가 다시 열린다 | 변화 없음 |

**병합의 진짜 대가가 여기에 있다.** 3단계 구성에서는 발급을 남긴 채 **강제만** 되돌릴 수 있었다.
병합하면 레버가 하나뿐이고, 그것을 당기면 **미인증 토큰의 테넌트 전체 읽기·삭제·강제종료가 즉시 복구된다.**
장애 대응 중에 이 레버를 당기는 사람이 그 사실을 모르면 안 되므로 여기에 명시한다.

**B가 배포된 상태에서 A를 되돌릴 때.** 계기 6은 `scope_version >= 2`에서만 발동하고 되돌린 서버는 그 필드를
발급하지 않으므로 **롤백 완료 후에는** 잠든다. 다만 **롤백 창 동안에는 발동한다.**
아직 강제 중인 replica가 발급한 `scope_version: 2` 토큰의 생성이 되돌린 replica에 걸리면 불일치가 되기 때문이다.
재부팅 상한으로 유계이며, 이 경우 복구는 수동 새로고침이 아니라 **자동 재부팅**이다.

## 2. A 단계 (monorepo, 한 PR): 발급·배관·409·강제

브랜치: `VOIP-1501-Bind-direct-token-to-allowed-resource` (현재 워크트리)

### 2.1 스코프 필드 추가

- [x] `bin-api-manager/models/auth/auth.go` `DirectScope`(`:35-40`)에 5필드 추가
      `AllowedResourceID uuid.UUID` / `DirectID uuid.UUID` / `HashFingerprint string` / `BootExpire string` / `ScopeVersion int`
      json 태그: `allowed_resource_id`, `direct_id`, `hash_fingerprint`, `boot_expire`, `scope_version`
      **`omitempty`를 붙이지 않는다.** 설계 §8.1이 값 기준 술어를 요구한다

### 2.2 부팅에서 채우기

- [x] `bin-api-manager/pkg/servicehandler/boot.go` `AuthBoot`의 `scope` 생성부(`:121`)
      `AllowedResourceID = h.utilHandler.UUIDCreate()`
      `DirectID = d.ID`
      `HashFingerprint = directHashFingerprint(d.Hash)`
      `BootExpire = h.utilHandler.TimeGetCurTimeAdd(BootSessionMaxLifetime)`
      `ScopeVersion = DirectScopeVersionCurrent`
- [x] 상수: `BootSessionMaxLifetime = time.Hour * 24`, `DirectScopeVersionCurrent = 2`
      **처음부터 2를 발급한다.** 3단계 구성의 1→2 승격 절차는 병합 착륙에서 폐기됐다.
      1로 "고치지" 말 것. 근거는 설계 §9.1과 상수 선언 옆 주석에 있다
- [x] `directHashFingerprint(hash string) string`: **`HMAC(jwtKey, 도메인 접두사 || 해시)`의 hex 전체, 절단 없음**(설계 §3.6).
      초판은 키 없는 SHA-256을 지정했으나 그것으로 되돌리면 안 된다. 원본 해시가 48비트뿐이고
      토큰이 WebSocket 경로에서 통째로 로깅되므로, 키 없는 다이제스트는 로그에서 역산된다.
      `Test_directHashFingerprint_isKeyed`와 `_goldenVector`가 유도식을 고정한다.
      **`boot.go`에 둔다.** §2.3의 `boot_refresh.go`도 같은 패키지에서 호출한다. 중복 정의하지 말 것
- [x] `BootResponse`(`:27-46`)에 **두 필드** 추가하고 응답에 채운다
      `AllowedResourceID uuid.UUID \`json:"allowed_resource_id"\``
      `ScopeVersion int \`json:"scope_version"\``
      **두 값은 반드시 `scope.AllowedResourceID` / `scope.ScopeVersion`에서 읽는다.**
      상수나 지역 변수에서 따로 채우면 안 된다. 현재 `res := &BootResponse{...}`(`boot.go:139-146`)는
      전부 `d.*`(direct 레코드)에서 채우고 있어, 자연스럽게 따라 하면 `scope`와 분리된다.
      분리되면 클레임과 응답이 어긋나 **계기 6이 정확히 필요한 롤아웃 창에서 잠든 채로 있게 된다**(라운드 2 CRITICAL의 거울상).
      `AllowedResourceID`를 응답용으로 다시 `UUIDCreate()`하면 더 나쁘다. 모든 대화가 불일치가 된다
      **`ScopeVersion`을 여기에 노출하는 것이 라운드 2 CRITICAL의 해결책이다.** 클라이언트가 "강제가 켜졌는지"를 알 유일한 수단이며, JWT 클레임 안에만 있으면 읽을 수 없다. **서버 PR에 반드시 들어가야 한다**(B 단계 클라이언트가 이것을 읽는다)

### 2.3 갱신 엔드포인트

- [x] `bin-api-manager/pkg/servicehandler/boot_refresh.go` 신설: `AuthBootRefresh(ctx, a *auth.AuthIdentity) (*BootResponse, error)`
      설계 §3.6의 7단계를 그 순서대로:
      1: `!a.IsDirect()` → `ErrPermissionDenied` (403)
      2: `AllowedResourceID`/`DirectID` Nil 또는 `HashFingerprint` 빈 문자열 → `ErrAuthenticationRequired`
      3: `now > BootExpire` → `ErrAuthenticationRequired`
      4: `DirectV1DirectGet(DirectID)` 실패(not-found·RPC 오류 무관) → `ErrAuthenticationRequired`
      5: `directHashFingerprint(record.Hash) != HashFingerprint` → `ErrAuthenticationRequired`
      6: `CustomerV1CustomerGet` 실패 또는 `Status != StatusActive` → `ErrAuthenticationRequired`
         **`isBlockedAccountStatus`(`authenticate.go:251-253`)의 fail-open 분기를 재사용하지 말 것**
      7: 스코프 전체 복사 → 레코드로 `CustomerID`/`ResourceType`/`ResourceID` 덮어쓰기 →
         `AllowedResourceTypes = directResourceMapping[record.ResourceType]` (미스면 `ErrAuthenticationRequired`)
- [x] **만료는 duration 쪽에서 자른다.** `authJWTGenerateWithExpiration`은 절대 시각을 받지 않는다(호출부 `boot.go:133`)
      `expireAt, err := h.utilHandler.TimeParseWithError(scope.BootExpire)` → 실패 시 401
      (**패키지 함수가 아니라 주입된 메서드를 쓴다.** `bin-common-handler/pkg/utilhandler/time.go:43`. 상한 직전 갱신 테스트가 결정적이려면 mock이 필요하다)
      `duration := min(BootExpiration, expireAt.Sub(time.Now().UTC()))` → `duration <= 0`이면 401
- [x] 응답은 `BootResponse`에서 `ResourceData`만 뺀 형태. 나머지 필드는 전부 채운다
- [x] `ServiceHandler` 인터페이스(`pkg/servicehandler/main.go`) 추가 + mock 재생성
- [x] `bin-api-manager/lib/service/boot.go`에 `PostBootRefresh` 신설.
      **신원 추출은 `lib/service/auth_delegate.go:46-56`을 그대로 따른다.**
      `c.Get("auth_identity")` → 없으면 401 → `tmp.(*auth.AuthIdentity)` 타입 단언 실패나 nil이면 401.
      `c.MustGet`을 쓰면 패닉 경로가 생긴다. 기존 회귀 테스트 `auth_delegate_test.go:63`이 같은 형태다.
      `PostBoot`(`:44`)의 blanket `AbortWithStatus(400)`을 따르지 않는다. `ErrPermissionDenied` → 403, 그 외 → 401
- [x] `bin-api-manager/cmd/api-manager/main.go` `authProtected` 그룹(`:306-312`)에 `POST /boot/refresh` 등록

### 2.4 id 배관: aicall (설계 §4.2.1, 전부 inert)

- [x] `bin-common-handler/pkg/requesthandler/ai_aicalls.go:18` `AIV1AIcallStart`에 `id uuid.UUID` + 인터페이스(`main.go:283`) + mock
- [x] `bin-api-manager/pkg/servicehandler/aicall.go:93` 호출부 `uuid.Nil`
- [x] `bin-api-manager/pkg/servicehandler/serviceagent_aicall.go:284` 호출부 `uuid.Nil`
- [x] `bin-ai-manager/pkg/listenhandler/models/request/aicalls.go`에 `ID uuid.UUID \`json:"id,omitempty"\`` 추가
      **`omitempty`는 `[16]byte`에 효과가 없어 항상 zero UUID로 직렬화된다.** 무해하며 transcribe 선례와 동일. 나중에 "고치지" 말 것
- [x] `bin-ai-manager/pkg/listenhandler/v1_aicalls.go:91` `req.ID` 전달
- [x] `bin-ai-manager/pkg/aicallhandler/start.go:170` `Start`에 `id` + 인터페이스 + mock
- [x] `bin-ai-manager/pkg/aicallhandler/service.go:54`, `:96` 호출부 `uuid.Nil`
- [x] `startReferenceType*` 4개(`:204/:245/:405/:606`), `startAIcallByRealtime`(`:993`), `startAIcallByMessaging`(`:1050`)에 `id` 관통
- [x] `startAIcallByMessaging` 호출 4곳(`:286/:424/:622/:1121`), `startAIcallByRealtime` 호출 1곳(`:228`)
- [x] `bin-ai-manager/pkg/aicallhandler/db.go:41`, `:115`: `id == uuid.Nil`이면 `UUIDCreate()`, 아니면 그대로

### 2.5 id 배관: webchat session (호출자 조사 완료)

- [x] `bin-common-handler/pkg/requesthandler/webchat_session.go:17` `WebchatV1SessionCreate`에 `id` + 인터페이스(`main.go:1567`) + mock
- [x] `bin-api-manager/pkg/servicehandler/webchat_session.go:217` 호출부 `uuid.Nil`
- [x] `bin-webchat-manager/pkg/listenhandler/models/request` `V1DataSessionsPost`에 `ID` 추가
- [x] `bin-webchat-manager/pkg/listenhandler/v1_sessions.go:33` `req.ID` 전달
- [x] `bin-webchat-manager/pkg/sessionhandler/create.go:29` `Create`에 `id` + 인터페이스 + mock
- [x] `bin-webchat-manager/pkg/sessionhandler/create.go:37`: `id == uuid.Nil`이면 `UUIDCreate()`, 아니면 그대로

### 2.6 409 경로 (설계 §4.2.2)

**aicall:**

- [x] **`bin-ai-manager/pkg/dbhandler/aicall.go:38`의 `%v` 래핑을 유지한다.**
      `IsErrDuplicate`의 기존 소비자 2곳(`aicallhandler/start.go:444`, `aihandler/db.go:229`)이 문자열 매칭에 의존한다
- [x] 분류는 `aicallhandler.Start`(`start.go:170-201`)에서 **switch 이후 반환값**에 대해,
      `callerSpecifiedID && dbhandler.IsErrDuplicate(err)`일 때만.
      **현재 `Start`의 다섯 분기는 각자 곧바로 `return`한다.** 반환값에 분류를 걸려면
      전부 "대입 후 fall-through"로 바꿔야 한다. 단순 추가가 아니다.
      (에러 문자열은 살아남는다. `startReferenceTypeNone`(`:606`)이 `errors.Wrapf`로 감싸므로
      `IsErrDuplicate`의 문자열 매칭이 계속 동작한다)
- [x] `errAIcallIDAlreadyExists`: 드라이버 에러를 `%w`/`%v`로 감싸지 않은 새 `cerrors.AlreadyExists`

**webchat session: 판별 수단부터 만들어야 한다:**

- [x] **`bin-webchat-manager`에는 중복 판별 수단이 없다.** `IsErrDuplicate` 0건, db 센티널은 `ErrNotFound` 하나뿐(`pkg/dbhandler/main.go:52`).
      그리고 `errorResponse`(`pkg/listenhandler/main.go:102-123`)는 타입 없는 에러를 **500**으로 떨어뜨린다.
      아무것도 안 하면 **webchat 위젯의 계기 2가 죽는다.**
- [x] `bin-webchat-manager/pkg/dbhandler/main.go`에 `IsErrDuplicate` 신설.
      `bin-ai-manager/pkg/dbhandler/main.go:114`를 그대로 미러링(`"Duplicate entry"` / `"UNIQUE constraint failed"` 문자열 매칭).
      `SessionCreate`가 `%v`로 래핑해 드라이버 문자열을 보존하므로(`pkg/dbhandler/session.go:58`) 동작한다
- [x] **`SessionGet` 기반 사전 확인을 쓰지 말 것.** §1.1이 금지하며, transcribe 선례를 따라가면 설계가 명시적으로 거부한 교차 테넌트 존재 오라클을 재도입하게 된다
- [x] `sessionhandler.Create`에서 `callerSpecifiedID && IsErrDuplicate(err)`일 때 `cerrors.AlreadyExists` 반환.
      `errorResponse`의 `stderrors.As(&ve)` 분기가 409로 매핑한다
- [x] `webchat_sessions`는 기본키 외 유니크 인덱스가 없으므로(`sessions.sql:20`) 분류가 aicall보다 단순하다

### 2.7 강제 (rev10까지의 3단계를 병합)

(이 절은 §2의 일부다. 별도 브랜치를 만들지 않는다)

- [x] `AuthBoot`의 `ScopeVersion` 참조를 `DirectScopeVersionCurrent`(=2)로 변경. **`scope`와 `BootResponse` 양쪽에 동일하게 반영된다**(§2.2의 단일 출처 규칙 덕분에 자동)
- [x] `bin-api-manager/lib/middleware/authenticate.go` `buildJWTIdentity`(`:122-144`) direct 분기에 거부 2조건
      `scope.AllowedResourceID == uuid.Nil` → 401, `scope.ScopeVersion < 2` → 401
- [x] `bin-api-manager/lib/middleware/direct_resource_scope.go` 신설. 설계 §4.3의 8단계
- [x] **미들웨어에 주석으로 2차 파라미터 공백을 남긴다**(설계 §4.3 요구).
      `/aicalls/:id/<child>/:child_id` 형태가 추가되면 `:child_id`를 아무도 검사하지 않는다
- [x] `cmd/api-manager/main.go`: `RegisterHandlersWithOptions`(`:336`) **이전에** `v1.Use(middleware.DirectResourceScope())`
- [x] 핸들러 6곳 덮어쓰기(설계 §4.4 표). 전부 덮어쓰기 직전 `AllowedResourceID == uuid.Nil` 가드.
      `WebchatMessageList`의 기존 `sessionID == uuid.Nil` 가드(`webchat_message.go:108-113`)는 **제거하지 않는다**
- [x] `AIcallCreate` direct 분기(`aicall.go:66-72`)에 `referenceType = ReferenceTypeNone`, `referenceID = uuid.Nil` 강제
- [x] `AIcallGetsByCustomerID`(`aicall.go:137`) direct 분기를 전면 거부로
- [x] `pkg/websockhandler/etc.go` `validateTopics`: nil 가드 + 정규 UUID 검사 + 배정 대조(설계 §4.6)
- [x] `etc.go:39-42` 주석 고쳐쓰기. **현재 주석이 "리소스 id를 검증하지 않는다"고 명시해 변경 후 거짓이 된다**
- [x] `validateTopic`(단수, `etc.go:87-150`) 제거하고 11개 케이스를 `Test_validateTopics`로 이관
- [x] RST: direct 토큰 접근 범위(`GET /aicalls` 차단 포함). 클린 빌드 + `git add -f build/`
- [x] **롤아웃 관측 수단.** `ScopeVersion < 2` 401과 미들웨어 403에 각각 카운터 또는 구조화 로그를 붙인다.
      **A 단계 배포 중 go/no-go와 롤백 판단의 입력이다.**
      병합 착륙에서는 강제 코드가 A와 함께 나가므로 배포 순간부터 값이 올라간다.
      구형 토큰이 소진되면 자연히 0으로 수렴하며, 그 수렴이 롤아웃 완료의 신호다
- [x] `bin-api-manager/docs/architecture.md`·`auth.md`: 미들웨어와 강제 규칙


### 2.8 A 단계 문서 (외부 노출 표면이 여기서 생긴다)

**OpenAPI (§1.1 정정 반영):**

- [x] `bin-openapi-manager/openapi/paths/auth/boot-refresh.yaml` 신설. 기존 6개 `paths/auth/*.yaml` 형식을 따른다.
      **200 응답 스키마는 기존 `AuthBootResponse`를 재사용한다.** `resource_data`가 이미
      `omitempty`이므로(`boot.go:45`) 별도 스키마가 필요 없다. 명시하지 않으면 구현자가
      `AuthBootRefreshResponse`를 새로 만들고 곧바로 drift가 시작된다.
      **`operationId: postAuthBootRefresh`로 고정한다.** 이 값이 생성될 메서드 이름 `PostAuthBootRefresh`를 결정하며,
      stub 파일명·메서드·규약 테스트 행이 전부 여기에 물린다(`paths/auth/boot.yaml:10`의 `postAuthBoot`가 전례)
- [x] `bin-openapi-manager/openapi/openapi.yaml`의 paths에 `/auth/boot/refresh` 엔트리 추가(`:8416` 인근)
- [x] `AuthBootResponse`(`openapi.yaml:8156`)에 `allowed_resource_id`, `scope_version` 추가
- [x] `bin-openapi-manager`에서 `go generate ./...` → `gens/models/gen.go` 커밋 → 이어서 `bin-api-manager`
- [x] **`bin-api-manager/server/auth_boot_refresh.go` 신설.** `server/auth_boot.go:13-19` 형태의 stub으로
      `ROUTE_NOT_FOUND`를 반환한다. 실제 라우트는 `/auth` 그룹이므로 `/v1.0/auth/boot/refresh`는 도달 불가여야 한다.
      **이 stub이 없으면 `RegisterHandlersWithOptions`(`cmd/api-manager/main.go:336`)에서 컴파일이 실패한다**
- [x] `bin-api-manager/server/auth_stubs_test.go:24` 표에 행 추가

**RST:**

- [x] `POST /auth/boot/refresh`, `allowed_resource_id`, `scope_version`을 `bin-api-manager/docsdev/source/`에 반영
- [x] 클린 빌드: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build` 후 `git add -f bin-api-manager/docsdev/build/`

**서비스 docs (존재하는 파일만. rev2는 없는 파일 2개를 지목했다):**

- [x] `bin-api-manager/docs/routing.md`: 새 라우트
- [x] `bin-api-manager/docs/architecture.md`: `cmd/api-manager/main.go` 변경분
- [x] `bin-api-manager/docs/auth.md`: `DirectScope` 필드와 갱신 절차
      (**`bin-api-manager/docs/domain.md`는 존재하지 않는다.** 있는 파일은 `architecture.md`/`auth.md`/`operations.md`/`routing.md`뿐.
      §2.1이 `bin-api-manager/models/auth/auth.go`를 건드리므로 `scripts/check-service-docs.sh:60`이
      `docs/domain.md`를 요구하는 경고를 낸다. **경고에 반응해 `domain.md`를 만들지 말 것.**
      해당 내용은 `auth.md`로 간다)
- [x] `bin-ai-manager/docs/architecture.md`: 요청 DTO 필드 추가분
      (**`bin-webchat-manager/docs/`에는 `plans/`뿐이라 대상 파일이 없다.** 새로 만들지 않는다)
- [x] 라우팅 자체는 두 매니저 모두 변경 없음(기존 DTO에 필드만 추가). "라우팅 변경"으로 서술하지 말 것

### 2.9 발급·배관 테스트

**설계 §7 중 발급·배관에 해당하는 항목 전부.** 명시적으로:

- [x] `AuthBoot`이 5필드를 채우는지. `AllowedResourceID`가 매 부팅마다 다른지. `BootExpire`가 부팅+24h인지. `ScopeVersion == DirectScopeVersionCurrent`인지
- [x] `BootResponse`에 `allowed_resource_id`와 **`scope_version`**이 실려 나가는지
- [x] **부팅 응답의 `allowed_resource_id`·`scope_version`이 발급된 JWT의 `direct` 클레임 값과 동일한지.**
      단일 출처 규칙(§2.2)의 회귀 고정
- [x] 갱신 검사 1: agent/accesskey/delegate 각각 403이고 **패닉·500이 아닌지**
- [x] 갱신: **인증 없이는 거부되는지**
- [x] 갱신: **본문에 `allowed_resource_id`를 넣어도 무시되는지**
- [x] 갱신 검사 2: `AllowedResourceID`/`DirectID` Nil, `HashFingerprint` 빈 문자열 각각 401
- [x] 갱신 검사 3: `BootExpire` 경과 시 401. 파싱 실패 시 401
- [x] 갱신 검사 4: direct 레코드 삭제 시 401. **RPC 오류일 때도 401**
- [x] 갱신 검사 5: **해시 재발급 후 기존 토큰의 갱신이 401** (무효화가 갱신을 이기는지, 핵심 케이스)
- [x] 갱신 검사 6: 동결·삭제 고객사 401. **RPC 오류일 때도 401**
- [x] 갱신 7단계: `directResourceMapping` 미스 시 401
- [x] 갱신 7단계: **2회 연속 갱신 후 9필드가 `expire` 외 전부 동일**
- [x] 갱신 후 토큰으로 **WS 토픽 구독이 여전히 통과하는지** (`CustomerID` 유실의 행위적 탐지기)
- [x] 갱신: `BootExpire` 직전 갱신의 `expire`가 상한을 넘지 않는지. **duration 쪽에서 잘렸는지**
- [x] 갱신: `ScopeVersion`이 증가하지 않고 복사되는지
- [x] 갱신 응답에 `resource_data`가 없고 나머지 `BootResponse` 필드는 전부 있는지
- [x] 배관: 기존 호출자 전부가 `uuid.Nil`로 서버 생성 경로를 그대로 타는지
- [x] 배관: 지정 id가 실제로 그 id로 저장되는지 (aicall·webchat 양쪽)
- [x] 409: aicall·**webchat 양쪽**에서 PK 충돌이 409로 RPC 경계를 넘어오는지
- [x] **`IsErrDuplicate` 비회귀**: `aicallhandler/start.go:444`와 `aihandler/db.go:229`가 그대로 동작
- [x] 센티널이 드라이버 에러를 감싸지 않아 `IsErrDuplicate`에 걸리지 않는지
- [x] 시그니처 변경으로 깨지는 기존 테스트 수정: `requesthandler/ai_aicalls_test.go:82`,
      `servicehandler/aicall_test.go:167`, `serviceagent_aicall_test.go:505`,
      `aicallhandler/start_test.go:2445,:2799`, `db_test.go:238`,
      `sessionhandler/create_test.go:68,:152,:208,:249`, 양쪽 listenhandler 테스트
- [x] **`bin-api-manager/pkg/servicehandler/boot_test.go` 수정.** 시그니처 변경이 아니라 mock 기대값 문제다.
      `AuthBoot`이 `UUIDCreate()` 호출 1건과 `TimeGetCurTimeAdd(BootSessionMaxLifetime)` 호출 1건을 추가로 하는데,
      현재 기대는 `TimeGetCurTimeAdd(BootExpiration)` 하나뿐이다(`:324`). gomock이 예상 밖 호출로 실패한다

### 2.10 강제 테스트

**설계 §7의 서버 항목 중 강제에 해당하는 전부.** 명시적으로:

- [x] 미들웨어 0단계: `auth_identity` 없음·타입 불일치 시 401 abort
- [x] 미들웨어 1단계: **direct가 아닌 신원(agent/accesskey/delegate)은 그대로 통과**
- [x] 미들웨어 2단계: **경로 파라미터가 없으면 통과**(생성·전송 계열)
- [x] 미들웨어: **경로 파라미터 일치 시 통과, 불일치 시 403** (기본 케이스)
- [x] 미들웨어 3단계: 파라미터는 있는데 `id`가 아닌 경우 403 (fail-closed)
- [x] 미들웨어 4단계: 파싱 불가 문자열 403
- [x] 미들웨어 5단계: `DirectScope` nil 403, `AllowedResourceID` Nil + 경로 id 전부 0 → 403
- [x] 미들웨어 6단계: 대소문자·중괄호 UUID가 핸들러와 동일 판정
- [x] 미들웨어: **`GET /aimessages/{message_id}`가 자동 거부**되는지(종류 불일치)
- [x] 미들웨어 등록 위치 회귀(`RegisterHandlersWithOptions` 이전)
- [x] 6개 핸들러: 남의 id를 보내도 자기 리소스에 기록되는지
- [x] 6개 핸들러: 덮어쓰기 직전 `AllowedResourceID == uuid.Nil` 가드가 동작하는지
- [x] `WebchatMessageList`의 기존 `sessionID == uuid.Nil` 가드가 **제거되지 않았는지**
- [x] `AIcallCreate`: `reference_type: contact_case`가 `none`으로 강제되는지
- [x] `AIcallCreate`: 임의 `reference_id`가 `uuid.Nil`로 강제되는지 (**별도 케이스**)
- [x] `AIcallGetsByCustomerID`가 direct를 거부하는지
- [x] `buildJWTIdentity`: `AllowedResourceID` Nil 토큰, `ScopeVersion < 2` 토큰 각각 401.
      v1.0·authProtected·WS 업그레이드 세 경로 모두
- [x] `validateTopics`: 빈 문자열·부분 문자열·비정규 표기 거부, 배정 id 통과, 타 id 거부
- [x] `validateTopics`: **관리자·매니저의 4조각 trailing colon 구독이 여전히 통과**
- [x] `validateTopics`: **`webchat_widget` direct 토큰이 `aicall` 타입 구독을 여전히 거부**
- [x] `validateTopics`: `DirectScope` nil 시 패닉 없이 거부
- [x] `validateTopic`(단수) 제거 후 11개 케이스가 이관되어 통과하는지
- [x] **`AuthBoot` 응답의 `scope_version`이 2인지.** 클레임만 2로 바뀌고 응답이 1로 남는 회귀를 잡는다

### 2.11 A 단계 검증

**빌드 순서를 지킨다.** local `replace` 지시자 때문에 `bin-common-handler`의 인터페이스와 mock이 먼저 착륙해야 나머지가 컴파일된다.

`bin-openapi-manager` → `bin-common-handler` → `bin-ai-manager` → `bin-webchat-manager` → `bin-api-manager`

`bin-openapi-manager`가 맨 앞인 이유를 정확히 적는다. `gens/openapi_server/gen.go`는
**`bin-api-manager` 자신의 generate 지시자**가 스펙 YAML을 직접 읽어 만든다
(`bin-api-manager/openapi/config_server/generate.go:3`). `bin-api-manager`가
`bin-openapi-manager/gens/models`를 import하지는 않는다.
실제 전제는 **스펙 YAML 편집과, 다른 소비자를 위한 `bin-openapi-manager/gens/models/gen.go` 재생성·커밋**이며,
`bin-openapi-manager/CLAUDE.md`가 이 순서를 명령한다. 나중에 "필요 없으니 빼자"가 되지 않도록 적어 둔다.

각 디렉토리에서:

```
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [x] 5개 디렉토리 전부 통과. `go.mod`/`go.sum` 변경분을 함께 커밋
      (`go build`만으로는 `go.sum` staleness를 못 잡아 Docker 빌드가 깨진 전례가 있다)

---

## 3. B 단계 (monorepo-javascript): 클라이언트 재부팅 계약

브랜치: `VOIP-1501-Bind-direct-token-to-allowed-resource`
**선행: 게이트 G2. A 직후에 착륙시킨다(G3)**

### 3.1 선행 리팩터링: 상태 코드 보존

계기 2·3·5가 HTTP 상태를 봐야 하는데 두 앱 모두 버린다.

- [ ] `square-main/src/lib/api.js:26-30`: 상태 코드를 담은 에러 타입으로 교체
- [ ] `square-admin/src/webchat-widget-runtime/client.js:303-306`: 상태를 문자열에 섞지 말고 구조화

### 3.2 공용 재부팅 핸들러

- [ ] 각 앱에 재부팅 핸들러 하나. 계기 1·2·3·5·6이 전부 호출
- [ ] **square-admin은 핸들러의 소유 모듈을 정한다.** 핸들러가 client 상태(`_rebooting`, `start()`, `sessionId`)와
      widget DOM(`_clearConnectingIndicator`·`_clearTypingIndicator`는 `widget.js:630`에서만 도달 가능)을 **둘 다** 필요로 하는데,
      계기 2·3·5·6은 `client.js`에서, 계기 1은 `widget.js:464-469`에서 발동한다.
      **widget이 소유하고 client에는 `onSessionEnded`와 같은 형태의 콜백으로 노출한다.**
      둘로 쪼개면 플래그와 표시 상태가 어긋난다
- [ ] 재부팅 후 동작은 **실패 요청 재생이 아니라 대화 재초기화**
- [ ] square-admin의 재초기화는 대화 상태만으로 부족하다. **소켓 닫기, 재연결 카운터 리셋,
      `_closeIntentional`·`_startPromise` 초기화, `sessionId` 비우기, `start()` 재실행**까지 포함
- [ ] **square-main의 재초기화 경로를 명시한다. 라운드 6에서 두 리뷰어가 서로 반대편을 지적했고, 둘 다 옳다.**

      - 한쪽: **`cleanup()`을 거쳐야 한다.** 이유를 정확히 적는다.
        rev7은 "`initializingRef`를 되돌리는 유일한 지점이 `:218`"이라고 썼는데 **사실이 아니다.**
        `initialize()`의 `finally`(`:147-149`)도 되돌리며, 계기 6에서 즉시 반환하면 그 `finally`가 실행된다.
        따라서 재진입 가드는 문제가 아니다.
        `cleanup()`이 필요한 진짜 이유는 **abort(`:196-198`), 구독 해제(`:200-203`),
        리스너 제거(`:204-211`), `ws.disconnect()`(`:216`), `clearBoot()`(`:217`), 대화 내용 비우기(`:219`)**다.
        특히 abort는 §3.2의 "재부팅 이후 발행한" 판정을 square-main에서 공짜로 얻게 해 주는 장치이므로,
        **`cleanup()`을 생략하면 그 판정이 무너진다.**
        (rev7의 틀린 근거를 그대로 두면, 나중에 `:148`을 발견한 사람이 "`cleanup()`은 불필요하다"고
        결론지어 abort까지 함께 제거할 수 있다. 그래서 근거를 고쳐 적는다)
      - 다른 쪽: **`cleanup()`만 부르면 백오프가 무력화된다.** `cleanup()`이 `setStatus('idle')`(`:221`)을
        하는데, `ChatWidget.jsx:31-35`의 effect가 `isOpen && status === 'idle'`에서 곧바로 `initialize()`를
        부른다. 이것이 기존 재시도 메커니즘이며 `handleRetry`(`:56-58`)에 그렇게 문서화돼 있다.
        백오프가 지나기 전에 React가 다음 커밋에서 초기화를 시작하고,
        핸들러의 백오프 후 호출과 겹치면 **부팅과 생성이 두 번** 나간다(각자 새 토큰이라 409도 아니다).
        고아 aicall과 덮어써진 `topicRef.current`(구독 누수)가 남는다

      따라서 재부팅 핸들러는 이 순서를 따른다.

      1. `cleanup()` 호출 (참조·abort·`initializingRef` 정리)
      2. **즉시 `idle`이 아닌 상태로 전환**해 effect의 자동 발화를 막는다(예: `rebooting` 상태 신설).
         **1번과 같은 동기 블록에서, 사이에 `await`를 두지 않는다.**
         React 18 자동 배칭(`square-main/src/main.jsx:7` `createRoot`)이 두 `setStatus`를 한 커밋으로 합쳐
         `'idle'`이 관측되지 않게 해 주는데, 중간에 microtask가 끼면 `'idle'` 커밋이 한 번 나가고
         `ChatWidget.jsx:31-35`가 초기화를 선점한다. 이 순서가 존재하는 이유 자체가 무너진다
         **`rebooting`은 렌더링 분기에 반드시 편입한다.** `ChatWindow.jsx:47`의 `isLoading`과
         `useChatState.js:249`의 `isLoading`이 현재 `booting`/`connecting`만 본다.
         `rebooting`이 어디에도 안 걸리고 `cleanup()`이 메시지도 비우므로(`:219`),
         그대로 두면 백오프 최대 4초 동안 **헤더만 있고 본문이 빈 패널**이 보인다.
         설계 §9.2가 약속한 "짧은 재연결"이 아니다. 기존 로더를 재사용하면 한 줄이다
      3. 백오프 대기
      4. 핸들러가 **직접** `initialize()` 호출

      **재부팅 핸들러와 `initialize`가 서로를 참조하므로** 평범한 `useCallback` 두 개로는 표현되지 않는다.
      ref 간접 참조가 필요하다. 설계 문제가 아니라 구현 형태의 문제이며,
      square-main의 lint 게이트(`--max-warnings 0`, §3.5)가 구현 시점에 잡아 준다

- [ ] **`_startPromise` 정리 순서에 주의한다.** `client.js:329-331`이 `.finally()`에서 이를 null로 만든다.
      계기 6 재부팅을 `_doStart()` **안에서** 호출하면, 아직 진행 중인 P1의 `finally`가
      새로 만든 P2의 핸들을 null로 지워 이후 `start()`가 **동시 `POST /webchat_sessions`를 두 번** 보낸다.
      `_doStart`에서 먼저 빠져나온 뒤 재부팅하거나, `finally`에 동일성 검사를 건다
- [ ] **"재부팅 이후 발행한"을 판정할 수단을 둔다.** 이 구절이 §3.2 규칙 전체의 핵심인데,
      square-main은 공짜로 얻는다. `cleanup()`이 abort controller를 끊고(`useChatState.js:196-198`)
      생성 요청이 signal을 받으므로(`:132`), 재부팅 이전 생성은 `AbortError`로 끝나 2xx가 될 수 없다.
      **square-admin의 `POST /webchat_sessions`(`client.js:372-378`)는 signal을 넘기지 않는다.**
      재부팅과 동시에 떠 있던 생성이 나중에 2xx로 resolve하면 **새 주기의 카운터를 리셋**할 수 있다.
      부팅 epoch 값을 하나 두고 리셋 지점에서 대조한다
- [ ] 루프 방지: 연속 3회 상한, 지수 백오프(0s/1s/4s)
- [ ] **리셋 조건.** 설계 §3.6의 규칙을 그대로 쓰되 예외 하나만 둔다.

      **설계 §3.6의 리셋 규칙을 대체한다.** 설계 초판의 "인증된 요청 1회 성공"은 아래 이유로 폐기됐고, 설계 rev10이 같은 내용으로 갱신됐다.

      > 리셋 = **재부팅 이후 발행한 리소스 생성 요청(`POST /aicalls` 또는 `POST /webchat_sessions`)이
      > 2xx이고, 그 요청이 계기 6을 발동시키지 않았을 때 0.**
      >
      > **그 외 어떤 요청도 리셋하지 않는다.**

      **화이트리스트로 뒤집은 이유.** rev4는 "인증된 요청 1회 성공"(설계 초판과 동일), rev5는 "구조화된 HTTP 계층의 2xx"였다.
      둘 다 블랙리스트였고, 세 라운드 연속으로 새 문이 뚫렸다.

      - 라운드 4: **WebSocket 연결이 생성보다 먼저** 일어난다(`useChatState.js:122` vs `:129`).
        WS 성공이 리셋하면 부팅 → WS 성공(리셋) → 불일치 생성(계기 6) → 재부팅이 무한 반복된다.
      - 라운드 5: **재초기화의 정리 요청이 리셋한다.** `cleanup()`이 `DELETE /aicalls/<불일치 id>`를
        보내는데(`useChatState.js:213`), 구버전 replica에는 미들웨어가 없고 `AIcallDelete`가
        고객사만 확인하므로 **2xx**다. 계기 6 분기가 확정 증가가 아니라 동전 던지기가 되어 상한에 못 닿는다.
        square-admin의 `end()`(`client.js:1068-1073`)도 같다.
        **이력 조회는 더 나쁘다.** `GET /webchat_messages?session_id=<낡은 값>`(`client.js:793-796`)은
        설계 §3.4대로 조용히 치환되므로 **강제된 replica에서도 2xx**다.

      생성 요청 하나만 리셋 자격을 갖게 하면 이 계열이 통째로 닫힌다.
      정리 삭제·이력 조회·WS 연결·부팅·갱신은 애초에 후보가 아니다.

      **rev3의 "일치하는 생성"과는 다르다.** id 일치를 요구하지 않는다.
      `scope_version < 2`에서는 계기 6이 잠들어 있으므로 성공한 생성이 그대로 리셋하고,
      2단계 창의 계기 1 복구가 살아 있다. rev3이 깨뜨린 지점이 바로 여기였다.

      **rev2와 rev3이 양쪽으로 틀렸다.**
      - rev2("인증 요청 1회 성공")는 예외가 없어, 계기 6에서 `POST /aicalls`가 **200으로 성공**하고
        본문 id만 틀린 경우 매번 리셋되어 **무한 루프**가 됐다(라운드 2 CRITICAL).
      - rev3("배정 번호와 일치하는 생성이 성공했을 때")는 과교정이었다. 그 술어는 **3단계 이후에만 참이 될 수 있다.**
        그 전에는 서버가 항상 자기 id를 만들기 때문이다(`aicallhandler/db.go:41`, `sessionhandler/create.go:37`).
        결과적으로 G3가 요구하는 24시간 내내 리셋이 한 번도 일어나지 않아,
        **대화를 네 번 종료한 방문자(계기 1, 정상 경로)가 영구 에러 화면**을 보게 된다.
        설계 §3.6이 명시적으로 막으라고 한 실패 모드다.
        또한 설계 §9.2의 소진 확률이 1/64도 1/8도 아닌 **약 42%**가 되어 수용 기준 7이 거짓이 된다.

      좁은 예외만으로 양쪽이 다 닫힌다. `scope_version < 2`에서는 성공한 생성이 리셋하므로
      계기 1 복구가 살아 있고, `scope_version >= 2`에서는 불일치 생성이 리셋하지 않으므로
      라운드 2의 무한 루프도 닫힌 채로 남는다.

      **A 단계 롤링 배포 창의 소진 확률은 약 22%(2/9)다.** rev6이 적은 1/8은 과소평가였다.

      한 주기의 결과가 셋이다. replica 2개 기준으로 (부팅, 생성) 조합이 넷인데,
      - **종단 성공 1/4** (신-신): id 일치, 이후 어느 replica에 가도 동작한다. 흡수 상태
      - **증가 1/2** (신-구는 계기 6, 구-신은 401)
      - **리셋 후 재시도 1/4** (구-구): 생성이 성공해 리셋되지만 토큰이 `scope_version: 1`이므로
        다음 요청이 신버전에 닿는 순간 401이다. 흡수되지 않고 주기로 돌아온다

      1/8은 **세 번째 분기를 빼먹은 값**이다. `f(c) = ½f(c+1) + ¼f(0)`, `f(3) = 1`을 풀면 `f(0) = 2/9`다.

      수용 기준 7(유계·관측 가능·새로고침 복구)은 그대로 성립하고 재설계도 불필요하다.
      다만 이 값이 3단계 go/no-go 근거로 인용되므로 정확해야 한다.
      **설계 §9.2의 1/8·1/64와 같다고 주장하지 말 것.** 둘 다 이 예외 아래에서는 성립하지 않는다.

      **상한 민감도.** 상한을 3에서 5로 올리면 약 6%(2/33)로 떨어진다.
      영향 모수가 롤아웃 수 분간 부팅한 방문자로 한정되고 새로고침으로 복구되므로 **3을 유지한다.**
      22%가 과하다고 판단되면 상한만 올리면 되며, 다른 변경은 필요 없다
- [ ] 카운터는 **페이지 로드 단위 메모리**. `localStorage` 금지
- [ ] 429는 백오프 후 재시도하며 상한을 공유

### 3.3 계기 연결

- [ ] 계기 1 (square-admin): `widget.js:464-469`의 `onSessionEnded`에서 메시지 표시 후 재부팅
- [ ] 계기 1 (square-main): `useChatState.js`의 `aicall_id` 가드(`:47`) **앞에** `aicall_status_terminated` 분기 추가.
      **가드를 완화하지 않는다**(`:36-45`에 목적이 문서화되어 있다)
- [ ] 계기 2: `POST /aicalls`·`POST /webchat_sessions`의 409 → 재부팅.
      **현재 두 클라이언트 모두 생성 전에 항상 부팅하므로**(`useChatState.js:118`→`:129`,
      `client.js:345`→`:372`) 한 토큰으로 두 번 생성하는 경로가 없다. 즉 계기 2는 실사용에서 도달 불가다.
      그래도 서버 409 경로는 설계 §3.1의 계약이고 webchat은 그것 없이는 500이 나므로 구현한다.
      **테스트는 한 토큰으로 두 번 생성하는 합성 케이스로 만든다.**
      나중에 "죽은 코드"로 판단해 제거하지 않도록 여기 적어 둔다
- [ ] 계기 3: 401 → 재부팅 (두 앱 모두)
- [ ] 계기 5: 403 → 재부팅
- [ ] **계기 6: `boot.scope_version >= 2`일 때에 한해, 생성된 리소스 id가 `boot.allowed_resource_id`와 다르면 재부팅**
      삽입 지점: `square-main/src/components/widget/useChatState.js:134`(`aicallIdRef.current = aicall.id`),
      webchat은 `client.js:380`의 `sessionId` 대입부.

      **대조는 대입 "이전"에 수행한다.** 대입 후에 검사하면 `aicallIdRef.current`가 채워진 채로
      재초기화가 돌아 `cleanup()`이 `DELETE /aicalls/<불일치 id>`를 보낸다.
      발행 지점이 **두 곳**임에 주의한다. `cleanup()`(`useChatState.js:213`)과
      언마운트 effect(`:232-233`)다. 둘 다 `aicallIdRef.current` 가드를 쓰므로
      대입 이전 검사가 양쪽을 함께 덮는다. 한쪽만 고치지 말 것.
      square-admin도 같다. `end()`(`client.js:1068`)와 `_backfillMessages`(`:793`)가
      `this.sessionId` 가드를 쓰므로 `:380` 이전 검사가 둘 다 건너뛴다.

      **계기 6이 발동하면 초기화 함수의 나머지를 실행하지 않고 즉시 반환한다.**
      - square-main은 `useChatState.js:137-141`에 도달하면 안 된다. 도달하면
        **고아 id로 토픽을 구독**하고, 3단계에서 그 토픽은 거부되어 설계 §8.3대로
        **소켓 전체가 끊긴다.** 방금 재부팅으로 만든 새 대화의 WS까지 죽는다.
        `topicRef.current`도 고아를 가리킨 채 남아 이후 unsubscribe가 엉뚱한 토픽에 걸린다
      - square-admin은 `client.js:385`(`onSessionStart`)·`:387`(`_connectWs`)에 도달하면 안 된다.
        **다만 조기 반환은 `start()`를 정상 resolve시킨다.** 호출자가 그대로 진행한다.
        `widget.js:604-607`(`_handleSend`)는 곧바로 `sendMessage(text)`로 가고,
        `widget.js:637-669`(`open()`)은 `finally`(`:657-667`)에서 연결 표시를 지운다(`:665`).
        `sendMessage`는 `client.js:982-987`에서 fail-closed로 막히므로 안전하지만,
        방문자에게는 **표시가 사라진 빈 패널**로 보인다.
        조기 반환을 폴백스루로 "고치지" 말 것

- [ ] **square-admin에 재부팅 진행 플래그를 둔다.** square-main은 `rebooting` 상태가 그 역할을 하는데
      square-admin에는 등가물이 없다. 두 앱의 재부팅 계약이 여기서만 비대칭이다.
      - **`start()` 재진입 차단.** 백오프(최대 4초) 동안 방문자가 메시지를 보내면
        `widget.js:604`가 `this.client.sessionId`가 비어 있는 것을 보고(계기 6 사전 검사 때문에 정상)
        스스로 `start()`를 부른다. `_startPromise` 중복 제거(`client.js:323-333`)는
        **진행 중인 시도만** 덮으므로, 방문자의 `start()`가 백오프보다 먼저 끝나면
        핸들러의 `start()`가 **두 번째 세션을 만들고** 첫 번째가 고아가 된다.
        `_rebooting` 플래그를 두고 `start()`가 그 동안 short-circuit하게 한다.
        **해제 시점을 못 박는다. 백오프가 끝나는 즉시, 핸들러가 자기 `start()`를 부르기 직전에 해제한다.**
        `await start()` 이후에 해제하면 **핸들러의 `start()`가 자기 플래그에 막혀** 위젯이 새로고침까지 죽는다.
        백오프가 0초여도 `_doStart`에서 먼저 빠져나온 뒤 호출한다(§3.2의 `_startPromise` 함정과 같은 계열)
      - **연결 표시 해제.** rev8은 "재부팅 중 지우지 않는다"만 적고 **언제 지울지를 안 적었다.**
        `_clearConnectingIndicator()`는 `open()`의 `finally`(`widget.js:657-666`)·`close()`·`destroy()`에서만
        도달 가능한데, 재부팅 핸들러는 `open()`이 아니라 `start()`를 직접 부른다.
        따라서 **재부팅이 성공해도 "Connecting…"이 남는다.** 핸들러가 settle 시 직접 지운다
      - **드롭된 메시지를 표시한다.** `_rebooting` 차단 중 방문자가 보내면 `widget.js:580`이 낙관적 말풍선을
        먼저 붙이고, `sendMessage`가 `client.js:982-987`에서 fail-closed로 던지며, `widget.js:608-611`은
        `console.error`만 한다. **말풍선은 남고 텍스트는 사라진다.**
        square-main은 낙관적 메시지에 `_error: true`를 붙이지만(`useChatState.js:186-191`) square-admin에는 등가물이 없다.
        재부팅 시작 시 시스템 메시지를 넣어 방문자가 알 수 있게 한다.
        (재전송은 하지 않는다. §3.2의 no-replay 결정이다)
      - **타이핑 표시 해제.** 계기 6 경로에서 `sendMessage`가 던지면 `widget.js:608-611`이
        `console.error`만 하므로 `:596-600`의 타이핑 표시가 30초 타임아웃까지 남는다. 함께 지운다

      **불일치로 생성된 리소스는 지우지 않고 남긴다.** 지우는 행위가 §3.2 누출의 원인이었고,
      남겨 두는 것은 설계 §3.4와 일관된다(고아 리소스는 만료로 정리된다).

      **게이팅이 필수인 이유(라운드 2 CRITICAL).** 무조건 대조하면 서버가 아직 배정 번호를 쓰지 않는 동안
      서버가 항상 자기 id를 생성하므로(`aicallhandler/db.go:41`, `sessionhandler/create.go:37`)
      **모든 대화가 매번 불일치**가 된다. G3가 요구하는 24시간 내내 두 위젯이 전면 장애가 나며,
      §3.2의 리셋 때문에 상한에도 걸리지 않아 무한 부팅 루프가 된다.

      **계기 6이 여전히 필요한 이유.** A 단계 롤링 배포 창에서 신버전이 발급한 토큰(`scope_version: 2`)의
      생성 요청을 구버전 인스턴스가 처리하면 서버 생성 id로 대화가 만들어지는데,
      그때 계기 1~5 중 **어느 것도 걸리지 않는다.** WS는 소켓이 끊기고 같은 토픽으로 재연결만 반복하며,
      `POST /aimessages`는 401/403/409가 아니라 **404**다. 403은 언마운트 시점에야 나온다.
      반대로 구버전이 발급한 토큰(`scope_version: 1`)은 계기 6이 잠들고, 신버전 인스턴스에 닿는 순간 401(계기 3)로 수렴한다
- [ ] 계기 4: `square-main/src/lib/auth.js:57`의 `bootAuth(directHash)`를 `refreshAuth()`로 교체.
      **`refreshAuth()`는 `cachedBoot` 갱신과 `scheduleRefresh` 재무장을 반드시 포함한다**(`auth.js:25-28`)
- [ ] **계기 4 실패는 상태 코드와 무관하게 계기 3으로 강등**한다.
      1→2단계 롤아웃 창에서는 갱신이 401이 아니라 **404**를 받을 수 있다
- [ ] 계기 4 (square-admin): 갱신 타이머가 **아예 없다**. 신설한다
      - `client.js:349-351`은 `token`/`customer_id`/`resource_id`만 저장한다.
        **`expire`, `allowed_resource_id`, `scope_version`을 함께 인스턴스 필드로 포착한다.**
        `expire` 없이는 타이머가 성립하지 않고, 뒤의 둘 없이는 `client.js:380`의 계기 6 대조가 성립하지 않는다
      - **갱신은 `this.token`을 교체하고, 갱신 응답의 새 `expire`로 자기 타이머를 다시 건다.**
        square-main의 `refreshAuth()`와 같은 요구사항이다. 재무장을 빠뜨리면 갱신이 1회로 끝나고
        4시간 절벽이 돌아온다. `allowed_resource_id`와 `scope_version`은 갱신으로 바뀌지 않으므로
        다시 잡을 필요가 없다. `expire`만 갱신한다
      - **갱신 시 WebSocket을 재연결하지 않는다**(§1.1 근거)

### 3.4 B 단계 테스트

- [ ] 계기별 재부팅 호출 검증 (1·2·3·5·6)
- [ ] 계기 5: 실패 요청을 재생하지 않고 재초기화하는지
- [ ] 계기 6: `scope_version >= 2`이고 id가 다를 때 재부팅하는지
- [ ] **계기 6: `scope_version < 2`이면 id가 달라도 재부팅하지 않는지.** 프로덕션에 도달하면 안 되는 회귀
- [ ] 루프 방지: 3회에서 멈추는지, 백오프가 적용되는지
- [ ] 루프 방지 리셋: **재부팅 후 생성 요청 2xx 시** 카운터가 0이 되는지
- [ ] 루프 방지 리셋 예외: **계기 6이 발동한 200 응답으로는 리셋되지 않는지**
- [ ] 루프 방지 리셋: **`scope_version < 2`에서 생성 성공이 카운터를 리셋하는지**(2단계 창에서 계기 1 복구가 살아 있는지)
- [ ] 루프 방지 리셋: **WebSocket 연결 성공은 카운터를 리셋하지 않는지.** 라운드 4가 지적한 옆문
- [ ] 루프 방지 리셋: **`POST /auth/boot` 성공은 리셋하지 않는지**
- [ ] 루프 방지 리셋: **정리 삭제(`DELETE /aicalls/...`)나 세션 종료가 2xx여도 리셋하지 않는지**
- [ ] 루프 방지 리셋: **이력·목록 조회가 2xx여도 리셋하지 않는지**
- [ ] 계기 6: 대조가 **id 대입 이전**에 수행되어 재초기화가 정리 삭제를 보내지 않는지
- [ ] 계기 6: 발동 시 **`ws.subscribe`가 호출되지 않는지**(square-main), **`onSessionStart`·`_connectWs`가 실행되지 않는지**(square-admin)
- [ ] 계기 6 재부팅이 실제로 **새 생성 요청을 발행하는지**. 재진입 가드에 삼켜지지 않는지
- [ ] 재부팅 백오프가 지나기 전에 `initialize()`가 실행되지 않는지(square-main의 `status === 'idle'` effect 선점 방지).
      **이 테스트는 `ChatWidget`을 마운트해야 한다.** `useChatState`만 `renderHook`으로 돌리면
      `ChatWidget.jsx:31-35`의 effect가 아예 실행되지 않아 **무조건 통과**한다(§3.5의 빈 기준선과 같은 함정).
      단 **`React.StrictMode`로 감싸지 않는다**(`main.jsx:8`이 실제 앱에서 쓰는 것과 달리).
      effect가 두 번 호출되어 검증 대상과 무관한 `initialize()` 호출이 섞인다
- [ ] `rebooting` 상태에서 로더가 렌더링되는지. 빈 패널이 보이지 않는지
- [ ] 메시지 전송(`POST /aimessages`, `POST /webchat_messages`)의 2xx가 리셋하지 않는지.
      규칙 본문의 "리소스 생성 요청"이 메시지 생성을 포함하는 것으로 오독될 여지를 고정한다
- [ ] 루프 방지: 대화를 두 번 종료한 방문자가 계속 동작하는지
- [ ] 계기 4 (square-main): 갱신 후 **대화 id가 유지되는지**, 타이머가 재무장되는지
- [ ] 계기 4 (square-admin): 갱신이 동작하는지, **소켓이 유지되는지**
- [ ] 계기 4: 404·500 등 임의 실패에서도 계기 3으로 강등되는지
- [ ] 계기 1: 상태 객체가 말풍선으로 렌더링되지 않는지
- [ ] 429에서 즉시 재시도하지 않는지

### 3.5 B 단계 검증

`monorepo-javascript/CLAUDE.md`의 PR 전 게이트를 앱마다 수행한다. 예외 없다.
**두 앱의 테스트 러너가 다르므로 명령이 다르다.** 같은 명령을 쓰면 한쪽 검증이 무력화된다.

| 앱 | 러너 | 기준선·사후 비교 명령 |
|---|---|---|
| `square-main` | **Vitest** (`package.json:11` `vitest run`) | `npm test 2>&1 \| tail -20` |
| `square-admin` | **CRA/Jest** (`package.json:19` `react-scripts test`) | `npm test -- --watchAll=false --forceExit 2>&1 \| grep "Tests:"` |

**square-main에 Jest 문법을 쓰면 안 된다.** `--watchAll=false --forceExit`는 Vitest 옵션이 아니고,
Vitest 요약은 `Tests  N passed (N)`처럼 콜론이 없어 `grep "Tests:"`가 아무것도 잡지 못한다.
기준선과 사후가 **둘 다 빈 값**이 되어 비교가 절대 실패하지 않는다.

절차:

1. 워크트리라면 먼저 `npm install` (`monorepo-javascript/CLAUDE.md:107`)
2. 변경 전 기준선 확보 (작업분을 치워 두고 위 명령)
3. 변경 후 같은 명령으로 **기준선과 비교**. 새로 깨진 테스트가 없어야 한다
4. `npm run build`

- [ ] `square-main` 전부 통과. 추가로 `npm run lint`(`package.json:14`, `--max-warnings 0`)
- [ ] `square-admin` 전부 통과
- [ ] **워크트리에서 `npm run dev`가 안 되는 것은 `square-admin`만**이다(CRA `ModuleScopePlugin`).
      수동 확인이 필요하면 `npm run build && npx serve -s build -l 3000`.
      `square-main`은 Vite라 `npm run dev`가 정상 동작한다

---

## 4. 프로덕션 E2E

- [ ] 브라우저 2개로 같은 위젯을 열고 교차 접근이 불가한지
- [ ] 세션 종료 후 재부팅으로 새 대화가 시작되는지
- [ ] **갱신 시점을 앞당긴 빌드로** 갱신 전후 대화 연속성 확인(4시간을 기다리지 않는다)
- [ ] `GET /aicalls`가 direct 토큰에서 거부되는지
- [ ] `customer_id:<cid>:aicall:` 구독이 거부되는지

## 5. 티켓과 기록

- [ ] VOIP-1501·VOIP-1498에 2단계(A/B) 착륙 계획과 병합 결정 경위 코멘트
- [ ] B 단계용 신규 티켓(monorepo-javascript) 생성
- [ ] **설계 §8.8의 후속 10건 등록** (rev2는 7건으로 잘못 셌다)
- [ ] G1/G2/G3의 통과 여부와 확인 시각을 해당 티켓에 코멘트로 남긴다

## 6. 열린 항목

- **없음.** rev10까지 열려 있던 PR 분할 승인은 2026-09-08 대표님 결정으로 해소됐다.
  서버는 한 PR로 간다(§1.2). monorepo와 monorepo-javascript로 나뉘는 것은 저장소가 달라 구조적이며 승인 대상이 아니다.
- 병합 결정의 대가는 §1.4에 기록했다. 강제만 되돌리는 레버가 없어졌다는 점이 유일한 실질적 손실이다.

---

## 7. 결과 (A 단계)

**완료: §2 전부.** monorepo 한 브랜치, 커밋 15개.

| 영역 | 내용 |
|---|---|
| 발급 | `DirectScope`에 5필드. 부팅이 배정 번호와 무효화 앵커를 채움 |
| 갱신 | `POST /auth/boot/refresh`. 7단계 검사, duration 쪽 절단 |
| 배관 | 호출 지점 15곳. 기존 호출자는 전부 `uuid.Nil`로 무변경 |
| 409 | aicall·webchat 양쪽. webchat은 중복 판별 함수부터 신설 |
| 강제 | `buildJWTIdentity` 거부 + `DirectResourceScope` + 핸들러 6곳 덮어쓰기 |
| 목록 차단 | `GET /aicalls` direct 거부 |
| WebSocket | 정규 UUID + 배정 대조. `validateTopic` 중복 제거 |
| 문서 | OpenAPI, RST, 서비스 문서 3종 |

**검증.** 5개 디렉토리에서 `go mod tidy` → `vendor` → `generate` → `test` → `golangci-lint`. 테스트 6,316개 통과, 린트 0건. Sphinx 클린 빌드.

**미착수 (의도된 것).**

- **§1.3 G1~G3**: 배포 시점 행동. G1(매니저 선행)이 가장 중요하며 어기면 전면 장애
- **§3 B 단계**: monorepo-javascript. 별도 저장소·별도 PR
- **§4 프로덕션 E2E**: 배포 후
- **§5 티켓**: 후속 10건 등록은 머지 후

**리뷰에서 배운 것.** 코드 리뷰 5라운드에서 기능 결함은 0건이었고, 블로킹은 두 갈래였다.

첫째는 **테스트 품질**이다. 라운드 1~3의 블로킹 3건이 여기 속하고 셋 다 같은 유형이다. 테스트가 통과하지만 아무것도 검증하지 않는 상태였고, 매번 리뷰어의 변이 테스트로만 드러났다. 그중 하나는 검증한다는 주석까지 달려 있어 커버된 것처럼 읽혔다.

교훈은 두 가지다. **통과했다는 사실 자체는 근거가 아니다** (해당 줄을 지워 보고 실패하는지 확인해야 한다). 그리고 **필터링한 출력으로 통과를 판단하면 안 된다** (이 작업에서 `grep` 패턴 때문에 빌드 실패를 통과로, `-run` 대소문자 때문에 미실행을 통과로 읽은 적이 각각 있다).

둘째는 **문서 드리프트**다. 라운드 4~5의 블로킹이 여기 속한다. 코드를 고치면서 그 근거를 담은 문서를 같이 안 고쳤고, 그래서 문서가 **코드와 반대되는 지시**를 담게 됐다. 지문 유도식이 그랬다. 설계 문서를 고친 뒤에도 같은 서술이 계획서에 남아 있어 한 라운드가 더 걸렸다. 교훈은 **결정을 뒤집으면 그 결정을 담은 문서를 전부 찾아야 한다**는 것이다. 한 곳만 고치면 나머지가 되돌리라고 말한다.
