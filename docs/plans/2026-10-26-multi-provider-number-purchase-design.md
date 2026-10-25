# VOIP-1654 Multi-provider number purchase (bin-number-manager)

## 1. 착수 전 티켓 검토 (적대적 재확인) : 부분 유효

origin/main 6bbe6b7fd 기준으로 직접 읽어 확인했습니다.

- 유효(반증 못함): `numberhandler.Create`(`pkg/numberhandler/number.go`)는 구입을 `h.numberHandlerTelnyx.NumberPurchase`로 고정하고, 롤백 release와 태그 갱신도 Telnyx 고정입니다. 반환(`Delete`)만 번호의 `provider_name`으로 분기합니다(telnyx, twilio). provider 설정 테이블 없음(`migrations/`는 `number_numbers` 베이스라인 하나). 열린 PR과 중복 티켓 없음(`gh pr list --state all --search "number provider"`에 무관한 PR만).
- 확인하지 못했던 것의 결과: Twilio 구입은 구현되어 있지 않습니다(`numberhandlertwilio.CreateNumber`와 `ReleaseNumber`는 "unimplemented" 에러 반환, 검색 `GetAvailableNumbers`만 구현). 따라서 이번 카드의 Twilio는 인터페이스 적합화까지이며 구입 시 항상 실패하여 폴백 대상이 됩니다(기본 비활성). api-manager 어드민 모델은 `hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin)`과 `a.IsDirect()` 거부 패턴입니다(`pkg/servicehandler/provider.go`). 최상위 `/providers`는 이미 route-manager의 SIP 캐리어 리소스이므로 이름 충돌을 피해 `/number_providers`를 씁니다.
- 틀린 주장: 없음. 단 "ai-manager와 bin-common-handler를 건드리지 않는다"는 암호화 공용화에 대한 지시로 해석합니다. api-manager가 number-manager를 호출하려면 `bin-common-handler/pkg/requesthandler`에 RPC 메서드 추가가 불가피합니다(기존 모든 리소스와 동일한 방식, 암호화 코드는 포함하지 않음).
- 숨은 가정: (a) Telnyx 주문 성공 후 확인 조회가 실패해도 폴백하면 이중 구입이 됩니다. 5절에서 폴백 중단 조건으로 처리합니다. (b) 구입 후 DB 등록 실패 시 롤백 release는 구입한 provider로 해야 합니다.

## 2. 범위와 비범위

범위: number-manager(provider 레지스트리, 설정 테이블, 라우팅, 암호화, RPC), bin-common-handler requesthandler 메서드, bin-openapi-manager 계약, bin-api-manager 어드민 엔드포인트, docsdev RST, komodo와 k8s 환경변수 이름, 문서(docs/domain, operations, architecture).
비범위: square-admin, 신규 provider 구현체, 고객 대상 API, 운영 값 주입, 재암호화 배치.

## 3. Provider 인터페이스와 레지스트리

`pkg/providerhandler`(신규)가 모든 provider 접근을 소유합니다.

```go
type Provider interface {
    GetAvailableNumbers(countryCode string, limit uint) ([]*availablenumber.AvailableNumber, error)
    NumberPurchase(num string) (*providernumber.ProviderNumber, error)
    NumberRelease(ctx context.Context, num *number.Number) error
    NumberUpdateTags(ctx context.Context, num *number.Number, tags []string) error
}
type Credentials struct { Secret string; Config map[string]string }
type Factory func(c Credentials) (Provider, error)
```

- 레지스트리는 `map[number.ProviderName]Descriptor{Factory, RequiredConfig []string, DefaultEnabled bool, DefaultPriority int, EnvCredentials func() Credentials}`. 코드에 등록된 provider만 DB 행이 의미를 가집니다(알 수 없는 provider_name은 API가 404).
- telnyx: 비밀=token, config=`connection_id`,`profile_id`. 기존 `numberhandlertelnyx`를 Provider에 맞춰 생성자가 Credentials를 받도록 바꿉니다(외부 HTTP 코드는 그대로).
- twilio: 비밀=token, config=`sid`. Purchase, Release, Tags는 기존처럼 unimplemented 에러를 반환(`ErrNotImplemented`), 검색만 동작.
- 기본값(행이 없을 때): telnyx enabled=true priority=10, twilio enabled=false priority=20. 어드민 PUT으로 행이 처음 만들어질 때 요청에 없는 필드는 DDL 기본값이 아니라 이 레지스트리 기본값으로 채웁니다(예: 행 없는 telnyx에 `priority`만 보내도 `enabled=true` 유지). usable provider가 0개가 되는 변경은 허용하되 warn 로그를 남깁니다. 따라서 배포 직후 환경변수만 있으면 Telnyx 단독으로 기존과 동일하게 동작합니다. 시드 마이그레이션(데이터 INSERT)은 두지 않습니다.
- 새 provider 추가 = 레지스트리에 Descriptor 한 건과 구현체 추가(마이그레이션 불필요).

### 자격 해석(Resolve)

provider 하나에 대해: 행이 있으면 enabled, priority는 행 값, 없으면 기본값. config 각 키는 행의 config 값, 없으면 환경변수. 비밀은 행에 `secret_ciphertext`가 있으면 복호화한 값(DB 우선), 없으면 환경변수. 행에 암호문이 있는데 복호화 실패(키 버전 없음, 변조)이면 환경변수로 폴백하지 않고 "사용 불가"로 처리하며 provider_name과 key_version만 로그에 남깁니다(다른 계정으로 조용히 바뀌는 것을 막기 위함, fail closed). `usable = enabled && 필수 비밀과 필수 config가 모두 있음`.

자격은 소스 단위로 원자적으로 해석합니다(혼합 금지): 행에 `secret_ciphertext`가 있으면 소스는 `db`이며 필수 config도 행의 `config`에서만 읽고 하나라도 없으면 사용 불가입니다. 행에 비밀이 없으면 소스는 `env`이며 비밀과 config 모두 환경변수에서만 읽습니다(행의 config는 무시). 이로써 DB의 새 계정 token과 환경변수의 옛 connection_id가 섞이지 않습니다. 복호화 실패 정책의 영향 범위: 해당 provider의 구입, 반환(`Delete`: 잔액 부족 정리 `renew.go`, 고객 삭제 정리 `event.go`), 검색이 모두 실패합니다. 이는 의도된 fail closed이며 복구 경로는 키 목록 복원 또는 `DELETE /number_providers/{name}/secret`(DB 비밀 삭제, 환경변수 소스로 복귀)입니다. 이 영향과 복구 절차를 `docs/operations.md`에 적습니다. 키가 없는 provider는 enabled여도 구입 후보에서 제외됩니다.

환경변수 `TELNYX_*`, `TWILIO_*`는 그대로 유지합니다.

## 4. 데이터 모델과 마이그레이션

`migrations/000002_number_providers_create_table_all.up.sql`/`.down.sql` (명명 규칙 `NNNNNN_<table>_<verb>_<type>_<items>`, items=all) (파일당 DDL 한 문장, 비파괴, 데이터 INSERT 없음):

```sql
CREATE TABLE number_providers (
  provider_name varchar(32) NOT NULL,
  enabled tinyint(1) NOT NULL DEFAULT 0,
  priority int NOT NULL DEFAULT 100,
  config json DEFAULT NULL,
  secret_ciphertext varbinary(2048) DEFAULT NULL,
  key_version int NOT NULL DEFAULT 0,
  updated_by binary(16) DEFAULT NULL,
  secret_updated_by binary(16) DEFAULT NULL,
  tm_secret_update datetime(6) DEFAULT NULL,
  tm_create datetime(6) DEFAULT NULL,
  tm_update datetime(6) DEFAULT NULL,
  PRIMARY KEY (provider_name)
);
```

down은 `DROP TABLE IF EXISTS number_providers`로 데이터를 복원하지 못하므로 첫 줄에 `-- DOWN-LOSSY:`를 씁니다(up은 비파괴이므로 `[DESTRUCTIVE]` 대상 아님). 조회는 항상 컬럼을 명시합니다. 테스트용 sqlite 스키마는 기존 관례대로 `scripts/database_scripts_test/table_number_providers.sql`(디렉터리 전체가 로드됨, `dbhandler/main_test.go`)에 추가합니다. `verify-service-migrations.sh`의 Alembic head(E1) 비교는 신규 테이블이 head에 없으므로 차이가 이 테이블 하나뿐임을 확인하고 `E1_ALLOWED_DIFF` 사용 여부를 스크립트 규칙대로 PR 본문에 남깁니다. `config`는 비밀이 아닌 값만 담으며 API가 비밀 키 이름을 받지 않도록 허용 키를 Descriptor의 `RequiredConfig`로 제한합니다.

감사: `updated_by`/`tm_update`는 enabled, priority, config 변경의 마지막 변경자와 시각, `secret_updated_by`/`tm_secret_update`는 비밀 변경의 변경자와 시각입니다. 변경마다 info 로그(변경자, provider_name, 바뀐 필드 이름만)를 남깁니다.

### 영향도 분석 (하드룰 17)

- 파괴적 연산: DROP, TRUNCATE, UPDATE, DELETE, INSERT SELECT, 타입 변경, 기본값 제거, 이름 변경, 기존 데이터에 대한 NOT NULL, UNIQUE, FK, CHECK 추가 모두 해당 없음. upgrade는 신규 테이블 CREATE 한 문장뿐입니다(기존 테이블 무변경, NOT NULL 컬럼은 모두 DEFAULT가 있고 신규 빈 테이블). enum 값 제거 해당 없음.
- 구/신 호환: 구 코드는 새 테이블을 참조하지 않으므로 신 스키마에서 동작합니다. 신 코드는 기동 시 마이그레이션이 리스닝 전에 실행되어 구 스키마를 보는 구간이 없습니다. `number_providers` 참조 코드는 신규뿐입니다(main에서 grep 결과 0건).
- 락과 소요: 신규 테이블 CREATE는 기존 테이블 락이 없고 즉시 끝납니다. 행 수는 provider 종류 수(한 자리)입니다.
- 롤백: 마이그레이터는 전진 호환이며 이전 코드는 신규 테이블을 사용하지 않으므로 이미지를 되돌려도 동작합니다. 테이블은 남습니다. 필요하면 후속 마이그레이션으로 DROP.
- 데이터 손실: upgrade는 없음. down은 provider 설정 행(비밀 암호문 포함)을 잃으므로 DOWN-LOSSY.
- scratch의 MariaDB와 MySQL 8.0에서 upgrade, downgrade, upgrade와 기존 데이터가 있는 DB 적용, Alembic head와의 덤프 비교(`scripts/verify-service-migrations.sh`)를 실행하고 PR 본문에 남깁니다.

## 5. 구입 라우팅

`numberHandler.Create`는 잔액 검사 후 `providerHandler.PurchaseCandidates(ctx)`로 usable provider를 priority 오름차순(동률은 provider_name 순)으로 받아 순서대로 시도합니다.

- 구입 전에 `FieldDeleted=false, FieldNumber=num` 중복 검사를 먼저 합니다(이미 등록된 번호를 구입한 뒤 Register가 실패해 누수되는 것을 막음).
- 폴백 전제: 고객이 검색에서 고른 번호(telnyx 재고)를 다른 provider가 가지고 있다는 보장은 없습니다. 폴백은 같은 번호를 보유한 provider가 있을 때만 의미가 있고, 다른 provider에 그 번호가 없으면 정상적인 확정 실패로 보고 다음 후보로 넘어갑니다. 이 한계는 문서화하고(검색의 provider 통합은 비범위) 테스트에 넣습니다.
- 오류 분류 표(이중 구입 방지, 보수적 기본값): provider는 구입 에러를 두 종류로 반환합니다. 확정 실패 `ErrPurchaseRejected`(요청이 수신되어 4xx로 거절됨: 재고 없음, 번호 오류, 주문 status가 실패로 확정)만 다음 후보로 폴백합니다. 그 밖의 모든 에러(네트워크 오류, 타임아웃(Telnyx 클라이언트 30초), 5xx, 주문 접수 후 조회 실패, 응답 파싱 실패)는 주문이 접수됐을 수 있는 불명확 실패로 보고 `ErrPurchaseUnconfirmed`로 감싸 폴백 없이 중단합니다(로그에 provider와 번호만, 운영자 확인). 불명확 실패는 provider 장애 때 폴백이 안 되는 단점이 있으나 이중 구입 비용 위험을 우선합니다(어드민이 provider를 비활성화하여 우회). 후보가 없으면 `could not purchase: no usable provider` 에러.
- `Register` 실패 시 롤백 release는 실제로 구입한 provider로 합니다. 롤백 release가 실패하면(예: provider가 release 미구현) error 레벨로 provider와 번호, `ROLLBACK_RELEASE_FAILED`를 남겨 운영 확인 대상으로 합니다. 태그 갱신도 같은 provider입니다. 번호의 `provider_name`에는 실제 구입한 provider 이름이 들어갑니다.
- `Delete`(반환)는 번호의 `provider_name`으로 레지스트리에서 provider를 해석하되 enabled 여부와 무관하게 자격만 있으면 해석합니다(비활성 provider 번호도 반환 가능). 자격이 없으면 에러(기존과 동일하게 반환 실패). `ProviderNameNone`(virtual)과 알 수 없는 이름의 기존 동작은 유지합니다.
- `RenewNumbers`(`renew.go`)는 provider를 직접 호출하지 않고 잔액 부족 시 `h.Delete`만 호출하므로(확인함) Delete 규칙을 따릅니다. `event.go`의 고객 삭제 정리도 `Delete`를 호출합니다.
- `cmd/number-control/main.go`도 `NewNumberHandler`를 조립하고 `number delete`가 provider를 호출하므로, 같은 secretcrypto와 providerhandler 조립을 적용하고 키 환경변수는 동일한 `NUMBER_SECRET_ENCRYPTION_KEYS`를 읽습니다. 두 `NewNumberHandler` 시그니처 변경과 mock 재생성이 따릅니다.
- 검색(`GetAvailableNumbers`, 고객 대상 API)은 telnyx 고정을 유지하되 해석된 telnyx 자격을 씁니다(DB 비밀 반영). 고객 대상 응답은 변경하지 않습니다. 단 telnyx가 db 소스에서 복호화 실패이면 검색도 실패합니다(위 영향 범위).
- provider 설정은 구입과 반환 시점에 DB에서 읽습니다(호출 빈도가 낮아 Redis 캐시를 두지 않으므로 캐시에 비밀이 남지 않고 어드민 변경이 즉시 반영). 복호화한 비밀은 해당 요청의 Provider 인스턴스 안에만 있고 로그와 에러 메시지에 넣지 않습니다.

## 6. 비밀값 암호화

`pkg/secretcrypto`(number-manager 안, ai-manager의 `SecretCrypto` 패턴을 별도 구현). 공용화는 세 번째 사용처가 생길 때(루트 CLAUDE.md 승인 규칙 3곳 이상).

- 환경변수 `NUMBER_SECRET_ENCRYPTION_KEYS`: `<버전>:<base64 32바이트>` 쉼표 나열. 최신(최대) 버전으로 암호화하고 행의 `key_version`으로 복호화. 형식 오류, 중복 버전, 버전<=0, 32바이트 아님은 파싱 에러이며 `cmd/number-manager`가 부팅 실패합니다. 빈 값은 허용(키 없음): 암호화는 항상 에러(비밀 저장 거부), DB 비밀 복호화 불가, 환경변수 자격으로만 동작.
- 암호화: AES-256-GCM, 12바이트 랜덤 nonce, AAD=provider_name(행 바꿔치기 방지). 저장 형식은 `nonce || ciphertext`를 `secret_ciphertext` 한 컬럼에 담습니다. 평문 상한 1024자.
- 모델 `Provider.SecretCiphertext []byte`는 `json:"-"`. 조회 응답은 `has_secret`, `key_version`만 노출. 에러 메시지와 로그에 평문, 암호문, 키를 넣지 않습니다. 이벤트는 발행하지 않습니다(provider 설정은 이벤트 대상이 아님). Config 설정 로그(`logStartup`)에 신규 키를 출력하지 않습니다.
- 키 교체 절차(재암호화 배치 불필요 판단 근거: 행이 provider 종류 수뿐이고 비밀은 어드민이 가진 외부 값을 다시 입력하면 되며, 서버가 평문을 일괄로 다루는 경로를 만드는 것이 오히려 노출면을 키움): (1) 새 버전을 목록에 추가하고 이전 버전을 유지하여 배포, (2) 어드민이 provider별로 비밀을 다시 입력(새 버전으로 재암호화, 응답의 `key_version`으로 확인), (3) 모든 provider의 key_version이 최신인 것을 확인한 뒤 이전 버전을 제거. 이전 버전을 먼저 빼면 그 행은 복호화되지 않아 provider가 사용 불가가 되고(환경변수 폴백 없음) 로그에 provider_name과 key_version이 남습니다. 절차는 `docs/operations.md`에 문서화합니다.

## 7. 어드민 API

number-manager RPC (listenhandler, 모두 `/v1/number_providers...`):

| 메서드 | 경로 | 동작 |
|---|---|---|
| GET | `/v1/number_providers` | 레지스트리의 모든 provider 해석 결과 목록(행이 없어도 기본값으로 표시) |
| GET | `/v1/number_providers/{name}` | 단건 |
| PUT | `/v1/number_providers/{name}` | `enabled`, `priority`, `config` 부분 갱신(포인터 필드, 행이 없으면 레지스트리 기본값으로 생성, `config`는 키 단위 병합이며 값이 빈 문자열이면 그 키 삭제), 요청에 `actor_id` |
| PUT | `/v1/number_providers/{name}/secret` | 비밀 쓰기 전용(`secret`, `actor_id`) |
| DELETE | `/v1/number_providers/{name}/secret` | DB 비밀 삭제(환경변수 소스로 복귀, 복구용, `actor_id`) |

listenhandler 정규식은 `^`/`$` 앵커를 두고(`/v1/number_providers$`, `/v1/number_providers/<name>$`, `/v1/number_providers/<name>/secret$`) 서로 겹치지 않게 합니다. `{name}`은 소문자 영숫자 `[a-z0-9_]{1,32}`.

응답 `Info`: `provider_name`, `enabled`, `priority`, `config`(비밀 아님), `has_secret`, `key_version`, `credential_source`(`db`, `env`, `none`, `db_undecryptable`), `usable`, `updated_by`, `tm_update`, `secret_updated_by`, `tm_secret_update`(쓰기는 비어 있지 않은 actor_id를 요구하므로 변경 이력이 있는 행의 값은 항상 유효한 UUID, 변경 이력이 없으면 NULL이고 응답은 null 필드 생략). 검증: 알 수 없는 provider 404, priority 0 이상 상한 1000000, config 키는 Descriptor 허용 키만, secret 비어 있거나 1024자 초과이면 400, 키 미설정 시 비밀 저장은 400 계열 에러(저장 거부).

api-manager(`/number_providers` 최상위 어드민 표면, 상담사용 `service_agents` 경로는 만들지 않음): `GET /number_providers`, `GET /number_providers/{provider_name}`, `PUT /number_providers/{provider_name}`, `PUT /number_providers/{provider_name}/secret`, `DELETE /number_providers/{provider_name}/secret`. servicehandler는 `a.IsDirect()` 거부 후 `hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin)`만 허용하고 `a.AgentID()`를 actor_id로 전달합니다. 접근키(accesskey) 인증은 `AgentID()`가 Nil이므로 쓰기 거부(감사 불가). 응답은 `ConvertWebhookMessage()`를 거치며 비밀 입력 요청 본문은 로그에 남기지 않습니다(server 핸들러가 본문을 로그 필드에 넣지 않음). OpenAPI: `openapi/paths/number_providers/` 신규, 스키마 `NumberManagerNumberProvider*`, 비밀 입력 필드는 `writeOnly: true`.

bin-common-handler `requesthandler`: `NumberV1NumberProviderList/Get/Update/UpdateSecret` 추가(암호화 코드 없음). 이 메서드는 api-manager와 number-manager 둘만 사용하는 기존 requesthandler 패턴을 따릅니다.

## 8. 환경변수와 배포 선언

신규 환경변수 이름: `NUMBER_SECRET_ENCRYPTION_KEYS` (flag `number_secret_encryption_keys`). `bin-number-manager/komodo/docker-compose.yml`에 `NUMBER_SECRET_ENCRYPTION_KEYS=[[BIN_MANAGER__NUMBER_SECRET_ENCRYPTION_KEYS]]`, `k8s/deployment.yml`에 secret 참조(`optional: true`로 하여 값 주입 전에도 기동)를 선언합니다. 값은 넣지 않습니다. 값이 비어 있어도 기동하며 배포 직후 동작은 환경변수 자격의 Telnyx 단독으로 현재와 같습니다. 운영 값 주입은 infra 범위입니다.

## 9. 보안 점검 항목 (credential 노출 스윕)

구조적 보장: `Credentials`와 `Provider` 모델은 `String()`과 `GoString()`을 구현해 비밀을 `[REDACTED]`로 출력하므로 `%v`/`%+v`/`%#v`로 찍혀도 노출되지 않으며 이를 테스트로 단언합니다.

응답(JSON 태그, WebhookMessage), 로그(요청 본문, provider 객체 `%v`/`%+v` 출력, 기존 `log.WithField("number", number)` 같은 필드에 provider 구조체가 들어가지 않는지), 이벤트 payload, 캐시(provider 설정은 캐시하지 않음), 에러 메시지(복호화 에러에 평문 미포함), 테스트 픽스처(가짜 값만), RST 예시(가짜 값), CLI(`number-control`)에 비밀 출력 경로가 없는지를 코드 리뷰 필수 항목으로 둡니다. Telnyx 요청 코드가 token을 URL/로그에 남기지 않는지도 확인합니다.

## 10. 테스트와 검증 기준 (verify 항목)

- secretcrypto: 형식 오류 각각, 최신 버전 선택, 이전 버전 복호화, AAD 불일치 실패, 키 없음 암호화 거부, 변조 실패.
- providerhandler: Resolve 우선순위 표(행 유무 x 암호문 유무 x 환경변수 유무 x 복호화 실패), usable 판정, 후보 정렬(동률 포함).
- numberhandler.Create: 1순위 성공, 1순위 재고 없음 후 2순위 성공, 전부 실패, 후보 없음, `ErrPurchaseUnconfirmed` 폴백 중단, Register 실패 시 구입한 provider로 롤백 release, 기존 Telnyx 단독 경로 회귀.
- Delete: 비활성 provider 번호 반환, 자격 없음 에러, provider_name별 해석.
- dbhandler: sqlite 스키마로 upsert와 조회, 암호문 json 비노출.
- listenhandler와 api-manager: 정상, 404, 400, 권한 거부(비 슈퍼어드민, direct, accesskey), 응답에 암호문 필드 없음 단언, 비밀 입력 요청 본문이 로그 출력(hook으로 캡처)에 나타나지 않는다는 단언, PUT 행 생성 기본값, config 병합, DELETE secret 복구, `Credentials` 출력 마스킹.
- 마이그레이션: `go test ./migrations/`, `scripts/check-service-migrations.sh`, 두 엔진 upgrade/downgrade/upgrade.
- 전체 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`을 number-manager, common-handler, openapi-manager, api-manager에서 실행. RST 변경 후 clean rebuild와 build 산출물 포함.

못 하는 것(명시): 실제 Telnyx, Twilio 구입 호출(비용, 하드룰 9), 운영 DB 적용.
