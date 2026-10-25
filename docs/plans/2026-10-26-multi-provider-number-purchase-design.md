# VOIP-1654 Multi-provider number purchase (bin-number-manager)

## 1. 착수 전 티켓 검토 (적대적 재확인) : 부분 유효

origin/main 6bbe6b7fd 기준으로 직접 읽어 확인했습니다.

- 유효(반증 못함): `numberhandler.Create`(`pkg/numberhandler/number.go`)는 구입을 `h.numberHandlerTelnyx.NumberPurchase`로 고정하고, 롤백 release와 태그 갱신도 Telnyx 고정입니다. 반환(`Delete`)만 번호의 `provider_name`으로 분기합니다(telnyx, twilio). provider 설정 테이블 없음(`migrations/`는 `number_numbers` 베이스라인 하나). 열린 PR과 중복 티켓 없음(`gh pr list --state all --search "number provider"`에 무관한 PR만).
- 확인하지 못했던 것의 결과: Twilio 구입은 구현되어 있지 않습니다(`numberhandlertwilio.CreateNumber`와 `ReleaseNumber`는 "unimplemented" 에러 반환, 검색 `GetAvailableNumbers`도 unimplemented: `numberhandlertwilio/available.go`, `Test_unimplemented`). 따라서 이번 카드의 Twilio는 인터페이스 적합화까지이며 `Purchasable=false`로 구입 후보에서 제외됩니다(기본 비활성). api-manager 어드민 모델은 `hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin)`과 `a.IsDirect()` 거부 패턴입니다(`pkg/servicehandler/provider.go`). 최상위 `/providers`는 이미 route-manager의 SIP 캐리어 리소스이므로 이름 충돌을 피해 `/number_providers`를 씁니다.
- 틀린 주장: 없음. 암호화 코드는 ai-manager와 bin-common-handler를 바꾸지 않고 number-manager 안에 둡니다(공용화는 세 번째 사용처가 생길 때). api-manager가 number-manager를 호출하려면 `bin-common-handler/pkg/requesthandler`에 RPC 메서드 추가가 불가피합니다(기존 모든 리소스와 동일한 방식, 암호화 코드는 포함하지 않음).
- 숨은 가정: (a) Telnyx 주문 성공 후 확인 조회가 실패해도 폴백하면 이중 구입이 됩니다. 5절에서 폴백 중단 조건으로 처리합니다. (b) 구입 후 DB 등록 실패 시 롤백 release는 구입한 provider로 해야 합니다.

## 2. 범위와 비범위

PR 분할: 이 문서는 두 PR 전체를 다루며 PR1에 포함해 머지합니다.
- PR1(이 카드, VOIP-1654 브랜치): number-manager 단독. provider 레지스트리, 설정 테이블 마이그레이션, 라우팅, 암호화, number-manager RPC(listenhandler, 7절 표), 오류 분류, `number-control` 조립, komodo와 k8s 환경변수 이름, number-manager 문서(domain, operations, architecture, CLAUDE.md, schema-ownership.md).
- PR2(별도 카드): bin-common-handler requesthandler 메서드, bin-openapi-manager 계약, bin-api-manager 어드민 엔드포인트, docsdev RST와 build HTML. PR1의 RPC 계약이 main에 있어야 합니다.
비범위: square-admin, 신규 provider 구현체, 고객 대상 API, 운영 값 주입, 재암호화 배치, 운영 DB 쓰기. PR1 단독 기간에는 어드민 API가 없어 DB 행 입력 수단이 없고 환경변수 자격으로 Telnyx 단독이 유지됩니다.

## 3. Provider 인터페이스와 레지스트리

`pkg/providerhandler`(신규)가 모든 provider 접근을 소유합니다. 순환 import를 피하기 위해 리프 패키지 `models/provider`에 `Provider` 행 모델, 응답 도메인 타입 `Info`(계산 필드 `usable`, `credential_source` 포함, json 태그를 가진 도메인 타입으로 listenhandler가 직접 마샬링: 루트 CLAUDE.md layering 스타일 A), `Credentials`, 센티널 에러(`ErrPurchaseRejected`, `ErrPurchaseUnconfirmed`, `ErrSecretKeyNotConfigured`, `ErrNotImplemented`, `ErrInvalidArgument`)를 둡니다. `numberhandlertelnyx`와 `numberhandlertwilio`는 `models/provider`만 import하고(생성자는 현행처럼 평문 문자열 인자), `providerhandler`의 레지스트리가 Factory 클로저에서 `Credentials`를 풀어 생성자를 호출합니다.

```go
type Provider interface {
    GetAvailableNumbers(countryCode string, limit uint) ([]*availablenumber.AvailableNumber, error)
    NumberPurchase(num string) (*providernumber.ProviderNumber, error)
    NumberRelease(ctx context.Context, num *number.Number) error
    NumberUpdateTags(ctx context.Context, num *number.Number, tags []string) error
}
type Credentials struct { Secret string; Config map[string]string } // String/GoString 은 [REDACTED]
type Factory func(c Credentials) (Provider, error) // reqHandler, db 는 레지스트리 생성 시 클로저로 캡처
```

`providerhandler.ProviderHandler` 인터페이스(풀어낸 인자, request/response DTO 미사용): `Resolve(ctx, name) (*Resolved, error)`, `PurchaseCandidates(ctx) ([]Candidate, error)`, `ForRelease(ctx, name) (Provider, error)`(enabled 무관), `ForSearch(ctx) (Provider, error)`, `List(ctx) ([]*provider.Info, error)`, `Get(ctx, name) (*provider.Info, error)`, `Update(ctx, name, actor uuid.UUID, enabled *bool, priority *int, config map[string]string) (*provider.Info, error)`, `UpdateSecret(ctx, name, actor uuid.UUID, secret string) (*provider.Info, error)`, `DeleteSecret(ctx, name, actor uuid.UUID) (*provider.Info, error)`. `actor`가 `uuid.Nil`이거나 파싱에 실패하면 `ErrInvalidArgument`(400).

- 레지스트리는 `map[number.ProviderName]Descriptor{Factory, AllowedConfig []string, RequiredConfig []string, Purchasable bool, DefaultEnabled bool, DefaultPriority int, EnvCredentials func() Credentials}`. `Purchasable=false`(twilio)이면 구입 후보에서 항상 제외하고 반환 해석에는 쓰입니다(반환도 unimplemented 에러가 되며 기존과 동일). 코드에 등록된 provider만 DB 행이 의미를 가집니다(알 수 없는 provider_name은 API가 404).
- telnyx: 비밀=token, config=`connection_id`,`profile_id`. 기존 `numberhandlertelnyx`를 Provider에 맞춰 생성자가 Credentials를 받도록 바꿉니다(외부 HTTP 코드는 그대로).
- twilio: 비밀=token, config=`sid`. `Purchasable=false`. Twilio용 어댑터(`Provider` 구현)가 기존 `numberhandlertwilio`의 시그니처 차이(`CreateNumber`, `ReleaseNumber`만 있음)를 감싸, 검색, 구입, 반환, 태그 네 메서드 모두 `ErrNotImplemented`를 반환합니다. `RequiredConfig`는 비어 있고 `AllowedConfig`=`sid`.
- telnyx: `AllowedConfig`=`connection_id`,`profile_id`, `RequiredConfig`=`connection_id`(db 소스 한정, env 소스는 현행처럼 token만 있으면 usable이며 profile_id 필수화는 하지 않음: 배포 직후 구입 중단 방지).
- 기본값(행이 없을 때): telnyx enabled=true priority=10, twilio enabled=false priority=20. 어드민 PUT으로 행이 처음 만들어질 때 요청에 없는 필드는 DDL 기본값이 아니라 이 레지스트리 기본값으로 채웁니다(예: 행 없는 telnyx에 `priority`만 보내도 `enabled=true` 유지). usable provider가 0개가 되는 변경은 허용하되 warn 로그를 남깁니다. 따라서 배포 직후 환경변수만 있으면 Telnyx 단독으로 기존과 동일하게 동작합니다. 시드 마이그레이션(데이터 INSERT)은 두지 않습니다.
- 새 provider 추가 = 레지스트리에 Descriptor 한 건과 구현체 추가(마이그레이션 불필요).

### 자격 해석(Resolve)

provider 하나에 대해: 행이 있으면 enabled, priority는 행 값, 없으면 기본값. 비밀과 config의 소스는 아래 '소스 단위 원자' 규칙으로 함께 정해집니다(키별 폴백 없음). 비밀은 행에 `secret_ciphertext`가 있으면 복호화한 값(DB 우선), 없으면 환경변수. 행에 암호문이 있는데 복호화 실패(키 버전 없음, 변조)이면 환경변수로 폴백하지 않고 "사용 불가"로 처리하며 provider_name과 key_version만 로그에 남깁니다(다른 계정으로 조용히 바뀌는 것을 막기 위함, fail closed). `usable = enabled && 비밀이 있음 && (소스가 db이면 RequiredConfig 충족)`. 구입 후보는 추가로 `Purchasable`이어야 합니다.

자격은 소스 단위로 원자적으로 해석합니다(혼합 금지): 행에 `secret_ciphertext`가 있으면 소스는 `db`이며 config도 행의 `config`에서만 읽고 RequiredConfig가 하나라도 없으면 사용 불가입니다. 행에 비밀이 없으면 소스는 `env`이며 비밀과 config 모두 환경변수에서만 읽습니다(행의 config는 무시). 이로써 DB의 새 계정 token과 환경변수의 옛 connection_id가 섞이지 않습니다. 복호화 실패 정책의 영향 범위: 해당 provider의 구입, 반환(`Delete`: 잔액 부족 정리 `renew.go`, 고객 삭제 정리 `event.go`), 검색이 모두 실패합니다. 이는 의도된 fail closed이며 복구 경로는 키 목록 복원 또는 `DELETE /number_providers/{name}/secret`(DB 비밀 삭제, 환경변수 소스로 복귀)입니다. 이 영향과 복구 절차를 `docs/operations.md`에 적습니다. 키가 없는 provider는 enabled여도 구입 후보에서 제외됩니다.

환경변수 `TELNYX_*`, `TWILIO_*`는 그대로 유지합니다.

## 4. 데이터 모델과 마이그레이션

`migrations/000002_number_providers_create_table_all.up.sql`/`.down.sql` (명명 규칙 `NNNNNN_<table>_<verb>_<type>_<items>`, items=all) (파일당 DDL 한 문장, 비파괴, 데이터 INSERT 없음):

```sql
CREATE TABLE IF NOT EXISTS number_providers (
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

down은 `DROP TABLE IF EXISTS number_providers`로 데이터를 복원하지 못하므로 첫 줄에 `-- DOWN-LOSSY:`를 씁니다(up은 비파괴이므로 `[DESTRUCTIVE]` 대상 아님). 조회는 항상 컬럼을 명시합니다. 테스트용 sqlite 스키마는 기존 관례대로 `scripts/database_scripts_test/table_number_providers.sql`(디렉터리 전체가 로드됨, `dbhandler/main_test.go`)에 추가합니다. `verify-service-migrations.sh`의 E1은 'Alembic head에 서비스 마이그레이션 적용', E2는 '빈 DB에 서비스 마이그레이션 적용'이며 새 테이블은 양쪽에 모두 생기므로 diff 0이 정상입니다(`E1_ALLOWED_DIFF` 불필요). 기존 데이터가 있는 DB 적용 검증에는 `number_numbers` 행 INSERT를 담은 `EXISTING_ROWS_SQL` 파일을 직접 준비합니다(스크립트 기본값은 다른 서비스용). `docs/schema-ownership.md`와 `bin-number-manager/CLAUDE.md`의 Database Tables 절에 `number_providers`를 추가합니다. `config`는 비밀이 아닌 값만 담으며 API가 비밀 키 이름을 받지 않도록 허용 키를 Descriptor의 `AllowedConfig`로 제한합니다. sqlite 테스트 스키마는 `varbinary` 대신 `blob`을 씁니다.

모델 타입: `UpdatedBy`, `SecretUpdatedBy`는 `*uuid.UUID`(`db:"...,uuid"`), `TMSecretUpdate`와 `TMUpdate`는 `*time.Time`으로 하여 NULL을 저장/조회하고(`databasehandler`는 `uuid.UUID` 값 타입 Nil을 16바이트 0으로 쓰기 때문), 응답 JSON은 `omitempty`입니다. `has_secret`은 DB에 암호문이 있을 때만 true이며 env 소스일 때는 false입니다.

감사: `updated_by`/`tm_update`는 enabled, priority, config 변경의 마지막 변경자와 시각, `secret_updated_by`/`tm_secret_update`는 비밀 변경의 변경자와 시각입니다. 변경마다 info 로그(변경자, provider_name, 바뀐 필드 이름만)를 남깁니다.

### 영향도 분석

- 파괴적 연산: DROP, TRUNCATE, UPDATE, DELETE, INSERT SELECT, 타입 변경, 기본값 제거, 이름 변경, 기존 데이터에 대한 NOT NULL, UNIQUE, FK, CHECK 추가 모두 해당 없음. upgrade는 신규 테이블 CREATE 한 문장뿐입니다(기존 테이블 무변경, `provider_name`(PK)을 제외한 NOT NULL 컬럼은 모두 DEFAULT가 있고 PK는 항상 값을 지정하는 신규 빈 테이블이므로 기존 데이터와 무관). enum 값 제거 해당 없음.
- 구/신 호환: 구 코드는 새 테이블을 참조하지 않으므로 신 스키마에서 동작합니다. 신 코드는 기동 시 마이그레이션이 리스닝 전에 실행되어 구 스키마를 보는 구간이 없습니다. `number_providers` 참조 코드는 신규뿐입니다(main에서 grep 결과 0건).
- 락과 소요: 신규 테이블 CREATE는 기존 테이블 락이 없고 즉시 끝납니다. 행 수는 provider 종류 수(한 자리)입니다.
- 롤백: 마이그레이터는 전진 호환이며 이전 코드는 신규 테이블을 사용하지 않으므로 이미지를 되돌려도 동작합니다. 테이블은 남습니다. 필요하면 후속 마이그레이션으로 DROP.
- 데이터 손실: upgrade는 없음. down은 provider 설정 행(비밀 암호문 포함)을 잃으므로 DOWN-LOSSY.
- scratch의 MariaDB와 MySQL 8.0에서 upgrade, downgrade, upgrade와 기존 데이터가 있는 DB 적용, Alembic head와의 덤프 비교(`scripts/verify-service-migrations.sh`)를 실행하고 PR 본문에 남깁니다.

## 5. 구입 라우팅

`numberHandler.Create`는 잔액 검사 후 `providerHandler.PurchaseCandidates(ctx)`로 usable provider를 priority 오름차순(동률은 provider_name 순)으로 받아 순서대로 시도합니다.

- `Create`는 후보 루프 사이에 ctx 취소를 확인합니다. provider 행은 한 자리 수라 호출마다 DB에서 읽으며 캐시를 두지 않습니다.
- 구입 전에 `FieldDeleted=false, FieldNumber=num` 중복 검사를 먼저 합니다(이미 등록된 번호를 구입한 뒤 Register가 실패해 누수되는 것을 막음).
- 폴백 전제: 고객이 검색에서 고른 번호(telnyx 재고)를 다른 provider가 가지고 있다는 보장은 없습니다. 폴백은 같은 번호를 보유한 provider가 있을 때만 의미가 있고, 다른 provider에 그 번호가 없으면 정상적인 확정 실패로 보고 다음 후보로 넘어갑니다. 이 한계는 문서화하고(검색의 provider 통합은 비범위) 테스트에 넣습니다.
- 오류 분류 표(이중 구입 방지, 보수적 기본값): provider는 구입 에러를 두 종류로 반환합니다. 확정 실패 `ErrPurchaseRejected`(요청이 수신되어 4xx로 거절됨: 재고 없음, 번호 오류, 주문 status가 실패로 확정)만 다음 후보로 폴백합니다. 현재 `requestexternal`은 비-200 응답을 상태 코드가 없는 문자열 에러로만 반환하므로, `requestexternal`에 상태 코드를 담은 타입 오류(`*HTTPStatusError{StatusCode}`)를 추가하고 `TelnyxNumberOrdersPost`가 이를 반환하게 하며(`TelnyxNumberOrdersPost`는 이미 `error`를 반환하므로 인터페이스와 mock 변경은 없음). Telnyx 분류 표: HTTP 400, 401, 403, 404, 422는 `ErrPurchaseRejected`, 그 밖의 모든 상태(408, 409, 429, 5xx 포함)와 전송 오류는 `ErrPurchaseUnconfirmed`. HTTP 200이면 `OrderNumber.Status`를 읽어 `failure`는 `ErrPurchaseRejected`, `pending`/`success`와 알 수 없는 값은 이후 `TelnyxPhoneNumbersGetByNumber` 확인으로 진행하며 그 조회 실패는 `ErrPurchaseUnconfirmed`입니다. 그 밖의 모든 에러(네트워크 오류, 타임아웃(Telnyx 클라이언트 30초), 5xx, 주문 접수 후 조회 실패, 응답 파싱 실패)는 주문이 접수됐을 수 있는 불명확 실패로 보고 `ErrPurchaseUnconfirmed`로 감싸 폴백 없이 중단합니다(로그에 provider와 번호만, 운영자 확인). 불명확 실패는 provider 장애 때 폴백이 안 되는 단점이 있으나 이중 구입 비용 위험을 우선합니다(어드민이 provider를 비활성화하여 우회). 후보가 없으면 `could not purchase: no usable provider` 에러(일반 error, 현행 `processV1NumbersPost`의 실패 응답 경로를 그대로 따르며 별도 상태 코드를 만들지 않음).
- `requestexternal/telnyx.go`가 `http.Response` 전체를 Debug 로그로 남기는 곳(Authorization 헤더가 노출될 수 있음)을 삭제하고 로그 출력에 헤더가 없다는 테스트를 추가합니다.
- `Register` 실패 시 롤백 release는 실제로 구입한 provider로 합니다. 롤백 release가 실패하면(예: provider가 release 미구현) error 레벨로 provider와 번호, `ROLLBACK_RELEASE_FAILED`를 남겨 운영 확인 대상으로 합니다. 태그 갱신도 같은 provider입니다. 번호의 `provider_name`에는 실제 구입한 provider 이름이 들어갑니다.
- `Delete`(반환)는 번호의 `provider_name`으로 레지스트리에서 provider를 해석하되 enabled 여부와 무관하게 자격만 있으면 해석합니다(비활성 provider 번호도 반환 가능). 자격이 없으면 에러(기존과 동일하게 반환 실패). `ProviderNameNone`(virtual)과 알 수 없는 이름의 기존 동작은 유지합니다.
- `RenewNumbers`(`renew.go`)는 provider를 직접 호출하지 않고 잔액 부족 시 `h.Delete`만 호출하므로(확인함) Delete 규칙을 따릅니다. 단 fail closed로 `Delete`가 실패할 가능성이 커지므로, 현재 `renew.go`에서 삭제 실패에도 `processed++`가 실행되어 실패한 번호가 `tm_renew` 조회에 계속 다시 나와 과금 RPC와 DB 조회가 반복되는 문제를 막습니다: 삭제에 실패한 번호는 처리 건수에 세지 않고 이번 실행의 제외 ID 목록에 넣으며, 다음 페이지 조회는 그 목록을 쿼리에서 제외합니다(`dbListByTMRenew`와 `NumberGetsByTMRenew`에 `excludeIDs []uuid.UUID` 인자를 추가하고 `squirrel.NotEq{id: 바이트}`; 인터페이스와 mock 변경). 조회 한 페이지(100건)가 전부 실패해도 다음 조회가 나머지를 가져오므로 굶지 않으며, 종료 조건은 '이번 페이지에서 `processed==0`이고 새로 제외된 ID도 0건'으로 바꿉니다(삭제 외의 `continue` 경로인 과금 RPC 오류와 `dbUpdate` 오류는 제외하지 않으며 기존처럼 processed==0이면 종료). `excludeIDs`는 비어 있으면 조건을 붙이지 않고, 여러 건이면 `squirrel.Expr("id NOT IN (?,?,...)")` 형태의 바이트 인자 `NOT IN`을 `Where`로 직접 붙입니다(`ApplyFields` 비경유). 100건 초과 실패 케이스를 테스트합니다. `event.go`의 고객 삭제 정리도 `Delete`를 호출합니다.
- `cmd/number-control/main.go`도 `NewNumberHandler`를 조립하고 `number delete`가 provider를 호출하므로, 같은 secretcrypto와 providerhandler 조립을 적용하고 키 환경변수는 동일한 `NUMBER_SECRET_ENCRYPTION_KEYS`를 읽습니다. 두 `NewNumberHandler` 시그니처 변경과 mock 재생성이 따릅니다.
- 검색(`GetAvailableNumbers`, 고객 대상 API)은 telnyx 고정을 유지하되 해석된 telnyx 자격을 씁니다(DB 비밀 반영). DB 조회가 필요하므로 `NumberHandler.GetAvailableNumbers`에 `ctx`를 추가하고 listenhandler 호출부(`v1_available_numbers.go`), `cmd/number-control/main.go`의 호출부, mock, 테스트를 함께 바꿉니다. providerHandler는 `NewNumberHandler`와 `NewListenHandler`(어드민 RPC용)에 각각 주입합니다. 고객 대상 응답은 변경하지 않습니다. 단 telnyx가 db 소스에서 복호화 실패이면 검색도 실패합니다(위 영향 범위).
- provider 설정은 구입과 반환 시점에 DB에서 읽습니다(호출 빈도가 낮아 Redis 캐시를 두지 않으므로 캐시에 비밀이 남지 않고 어드민 변경이 즉시 반영). 복호화한 비밀은 해당 요청의 Provider 인스턴스 안에만 있고 로그와 에러 메시지에 넣지 않습니다.

## 6. 비밀값 암호화

`pkg/secretcrypto`(number-manager 안, ai-manager의 `SecretCrypto` 패턴을 별도 구현). 공용화는 세 번째 사용처가 생길 때(루트 CLAUDE.md 승인 규칙 3곳 이상).

- 환경변수 `NUMBER_SECRET_ENCRYPTION_KEYS`: `<버전>:<base64 32바이트>` 쉼표 나열. 최신(최대) 버전으로 암호화하고 행의 `key_version`으로 복호화. 형식 오류, 중복 버전, 버전<=0, 32바이트 아님은 파싱 에러입니다. number-manager는 call-manager 등 통화 경로가 의존하므로 파싱 오류로 부팅 실패시키지 않고 error 로그(원인만, 값 제외) 후 키 없음으로 격하하여 기동합니다(여전히 fail closed: 암호화는 항상 에러이므로 비밀 저장 거부, DB 비밀 복호화 불가, 환경변수 자격으로만 동작). `number-control`은 같은 파서를 쓰되 파싱 오류를 즉시 에러로 반환합니다. 빈 값은 허용(키 없음).
- 암호화: AES-256-GCM, 12바이트 랜덤 nonce, AAD=provider_name(행 바꿔치기 방지). 저장 형식은 `nonce || ciphertext`를 `secret_ciphertext` 한 컬럼에 담습니다. 평문 상한 1024바이트(바이트 기준).
- 계정 교체 주의(PUT과 DELETE secret 모두 해당: DELETE는 DB 계정에서 환경변수 계정으로 바꿈): DB 비밀을 다른 계정의 token으로 교체하면 이전 계정에서 구입한 번호의 반환(`Delete`)이 실패하므로 기존 번호가 있는 provider의 계정 교체는 문서 경고만 두고 강제하지 않습니다(같은 계정의 token을 DB로 옮기는 정상 이전 경로를 막지 않기 위함) `docs/operations.md`에 적습니다. `number-control`은 `migrate up` 선행이 필요합니다(없으면 provider 조회 실패, fail closed). 키 파서 에러 메시지는 `ai-manager/secret.go`처럼 키 항목을 `%q`로 찍지 않고 값 제외를 테스트로 단언합니다. `ErrPurchaseUnconfirmed` 로그에도 `PURCHASE_UNCONFIRMED` 고정 마커를 둡니다.
- 모델 `Provider.SecretCiphertext []byte`는 `json:"-"`. 조회 응답은 `has_secret`, `key_version`만 노출. 에러 메시지와 로그에 평문, 암호문, 키를 넣지 않습니다. 이벤트는 발행하지 않습니다(provider 설정은 이벤트 대상이 아님). Config 설정 로그(`logStartup`)에 신규 키를 출력하지 않으며 `cmd/number-manager/startup_log_test.go`의 필드 분류에서 `NumberSecretEncryptionKeys`를 `secrets`에 추가합니다. 오류 상태 매핑: 어드민 핸들러는 `errorResponse(err)`를 직접 호출하며(기본 500은 DB/내부 암호화 오류), 센티널 `ErrInvalidArgument`(400, 검증 위반)와 `ErrSecretKeyNotConfigured`(503) 분기를 `errorResponse`에 추가하고 알 수 없는 provider는 기존 `ErrNotFound`(404)를 씁니다. `processRequest` 말미의 기본 분기(400)로 흘러가지 않게 합니다. 센티널은 `models/provider`에 정의합니다.
- 키 교체 절차(재암호화 배치 불필요 판단 근거: 행이 provider 종류 수뿐이고 비밀은 어드민이 가진 외부 값을 다시 입력하면 되며, 서버가 평문을 일괄로 다루는 경로를 만드는 것이 오히려 노출면을 키움): (1) 새 버전을 목록에 추가하고 이전 버전을 유지하여 배포, (2) 어드민이 provider별로 비밀을 다시 입력(새 버전으로 재암호화, 응답의 `key_version`으로 확인), (3) 모든 provider의 key_version이 최신인 것을 확인한 뒤 이전 버전을 제거. 이전 버전을 먼저 빼면 그 행은 복호화되지 않아 provider가 사용 불가가 되고(환경변수 폴백 없음) 로그에 provider_name과 key_version이 남습니다. 절차는 `docs/operations.md`에 문서화합니다.

## 7. 어드민 API

number-manager RPC (listenhandler, 모두 `/v1/number_providers...`):

| 메서드 | 경로 | 동작 |
|---|---|---|
| GET | `/v1/number_providers` | 레지스트리의 모든 provider 해석 결과 목록(행이 없어도 기본값으로 표시) |
| GET | `/v1/number_providers/{name}` | 단건 |
| PUT | `/v1/number_providers/{name}` | `enabled`, `priority`, `config` 부분 갱신(포인터 필드, 행이 없으면 레지스트리 기본값으로 생성, `config`는 키 단위 병합이며 값이 빈 문자열이면 그 키 삭제. 단 행의 소스가 db(암호문 있음)이고 병합 결과가 RequiredConfig를 깨뜨리면 enabled 여부와 무관하게 400으로 거부), 요청에 `actor_id` |
| PUT | `/v1/number_providers/{name}/secret` | 비밀 쓰기 전용(`secret`, `actor_id`). 키 없음이면 거부, 행의 config가 RequiredConfig를 충족하지 않으면 거부(enabled 여부와 무관: 비활성 provider에도 비밀과 config를 먼저 입력할 수 있음)(먼저 PUT `{name}`으로 config 입력). 행이 없으면 레지스트리 기본값으로 생성 |
| DELETE | `/v1/number_providers/{name}/secret` | DB 비밀 삭제(환경변수 소스로 복귀, 복구용, `actor_id`). 행이 없으면 404 |

쓰기 방식: 행 보장은 insert-if-absent로 합니다. 방식은 저장소 선례(`bin-agent-manager/pkg/dbhandler/agent_address.go`)와 같이 평범한 INSERT(레지스트리 기본값, `tm_create` 포함) 후 MySQL 1062 또는 sqlite `UNIQUE constraint failed`를 중복으로 보고 성공 처리합니다(엔진 전용 문법 `INSERT IGNORE`, `OR IGNORE`, `ON CONFLICT`는 쓰지 않음). 그 뒤 변경된 컬럼만 UPDATE하며 비밀 PUT은 `secret_ciphertext`, `key_version`, `secret_updated_by`, `tm_secret_update`를 하나의 UPDATE로 씁니다. (병행 호출이 서로의 컬럼을 지우지 않음). config 키 병합만 읽기-수정-쓰기이며 동시에 같은 provider config를 바꾸는 어드민은 마지막 쓰기가 이기고 이를 문서화합니다. 행 조회의 DB 오류는 기본값으로 취급하지 않고 요청을 실패시킵니다.

**로그 마스킹(필수)**: `listenhandler/main.go`의 공통 요청 로그(`data` Debug 필드)와 에러 경로, 각 `processV1*`의 `"request": m` 로그는 본문을 기록하므로 `processRequest`는 원본 `m`을 핸들러에 그대로 넘기고(그렇지 않으면 비밀 PUT이 `[REDACTED]`를 파싱함), 로그에는 `Data`를 `[REDACTED]`로 치환한 사본 `logM`만 씁니다(마스킹 판정은 라우팅 정규식과 독립으로 URI가 `/secret`으로 끝나거나 `/v1/number_providers`로 시작하는지로 하며 대소문자 변형과 잘못된 메서드, notfound 분기도 포함. 다음은 `/secret` 대상 로그 지점: `main.go`의 `data` 필드와 에러 경로, 비밀 핸들러의 `"request"` 필드와 unmarshal 실패 로그). 비밀 핸들러는 `(ctx, m, logM)`을 받으며 `m.Data`를 로그에 직접 쓰지 않습니다. 테스트는 비밀이 실제로 저장(암호화 호출에 평문이 전달)되는 동시에 로그 hook 출력에 평문이 없음을 한 케이스에서 단언하며, 로그 hook으로 캡처해 평문이 나타나지 않는다는 테스트를 둡니다.

listenhandler 정규식은 `^`/`$` 앵커를 두고(`/v1/number_providers$`, `/v1/number_providers/<name>$`, `/v1/number_providers/<name>/secret$`) 서로 겹치지 않게 합니다. `{name}`은 소문자 영숫자 `[a-z0-9_]{1,32}`.

응답 `Info`: `provider_name`, `enabled`, `priority`, `config`(비밀 아님), `has_secret`, `key_version`, `credential_source`(`db`, `env`, `none`, `db_undecryptable`), `usable`, `updated_by`, `tm_update`, `secret_updated_by`, `tm_secret_update`(쓰기는 비어 있지 않은 actor_id를 요구하므로 변경 이력이 있는 행의 값은 항상 유효한 UUID, 변경 이력이 없으면 NULL이고 응답은 null 필드 생략). 검증: 알 수 없는 provider 404, priority 0 이상 상한 1000000, config 키는 Descriptor 허용 키만, secret 비어 있거나 1024바이트 초과이면 400, 키 미설정 시 비밀 저장은 서버 설정 문제이므로 `ErrSecretKeyNotConfigured`로 거부하고 listenhandler가 `errors.Is`로 식별해 503 응답으로 변환합니다(기본 에러 분기는 `VoipbinError`/`ErrNotFound` 외에는 400이므로 명시적 분기 추가). 알 수 없는 provider는 `ErrNotFound`(404), 검증 위반(priority 범위, 허용 외 config 키, RequiredConfig 위반, 길이)은 400. config는 최대 8키, 키 32바이트, 값 256바이트 이하.

PR2 범위 참고용 계약: api-manager(`/number_providers` 최상위 어드민 표면, 상담사용 `service_agents` 경로는 만들지 않음): `GET /number_providers`, `GET /number_providers/{provider_name}`, `PUT /number_providers/{provider_name}`, `PUT /number_providers/{provider_name}/secret`, `DELETE /number_providers/{provider_name}/secret`. servicehandler는 `a.IsDirect()` 거부 후 `hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin)`만 허용하고 `a.AgentID()`를 actor_id로 전달합니다. 접근키(accesskey) 인증은 `AgentID()`가 Nil이므로 쓰기 거부(감사 불가). 응답은 `ConvertWebhookMessage()`를 거치며 비밀 입력 요청 본문은 로그에 남기지 않습니다(server 핸들러가 본문을 로그 필드에 넣지 않음). OpenAPI: `openapi/paths/number_providers/` 신규, 스키마 `NumberManagerNumberProvider*`, 비밀 입력 필드는 `writeOnly: true`.

PR2: (RPC 비밀 본문이 내부 RabbitMQ를 평문으로 통과하는 점은 PR2 설계에서 다룸) bin-common-handler `requesthandler`: `NumberV1NumberProviderList/Get/Update/UpdateSecret` 추가(암호화 코드 없음). 이 메서드는 api-manager와 number-manager 둘만 사용하는 기존 requesthandler 패턴을 따릅니다.

## 8. 환경변수와 배포 선언

신규 환경변수 이름: `NUMBER_SECRET_ENCRYPTION_KEYS` (flag `number_secret_encryption_keys`). `bin-number-manager/komodo/docker-compose.yml`에 `NUMBER_SECRET_ENCRYPTION_KEYS=[[BIN_MANAGER__NUMBER_SECRET_ENCRYPTION_KEYS]]`, `k8s/deployment.yml`에 secret 참조(`optional: true`로 하여 값 주입 전에도 기동)를 선언합니다. 값은 넣지 않습니다. 주의: Komodo 변수 `BIN_MANAGER__NUMBER_SECRET_ENCRYPTION_KEYS`가 배포 전에 만들어져 있지 않으면 compose 치환이 실패할 수 있으므로 변수 생성(빈 값)이 이 PR의 배포 선행 조건입니다(배포 선행 조건으로 PR 본문에 명시). 값이 비어 있어도 기동하며 배포 직후 동작은 환경변수 자격의 Telnyx 단독으로 현재와 같습니다. 운영 값 주입은 infra 범위입니다.

## 9. 보안 점검 항목 (credential 노출 스윕)

구조적 보장: `Credentials`와 `Provider` 모델은 `String()`과 `GoString()`을 구현해 비밀을 `[REDACTED]`로 출력하므로 `%v`/`%+v`/`%#v`로 찍혀도 노출되지 않으며 이를 테스트로 단언합니다.

응답(JSON 태그, WebhookMessage), 로그(요청 본문, provider 객체 `%v`/`%+v` 출력, 기존 `log.WithField("number", number)` 같은 필드에 provider 구조체가 들어가지 않는지), 이벤트 payload, 캐시(provider 설정은 캐시하지 않음), 에러 메시지(복호화 에러에 평문 미포함), 테스트 픽스처(가짜 값만), RST 예시(가짜 값), CLI(`number-control`)에 비밀 출력 경로가 없는지를 코드 리뷰 필수 항목으로 둡니다. Telnyx 요청 코드가 token을 URL/로그에 남기지 않는지도 확인합니다.

## 10. 테스트와 검증 기준 (verify 항목)

- secretcrypto: 형식 오류 각각, 최신 버전 선택, 이전 버전 복호화, AAD 불일치 실패, 키 없음 암호화 거부, 변조 실패.
- providerhandler: Resolve 우선순위 표(행 유무 x 암호문 유무 x 환경변수 유무 x 복호화 실패), usable 판정, 후보 정렬(동률 포함).
- numberhandler.Create: 1순위 성공, 1순위 재고 없음 후 2순위 성공, 전부 실패, 후보 없음, `RenewNumbers`가 삭제 실패 번호에 대해 반복하지 않음, `ErrPurchaseUnconfirmed` 폴백 중단, Register 실패 시 구입한 provider로 롤백 release, 기존 Telnyx 단독 경로 회귀.
- Delete: 비활성 provider 번호 반환, 자격 없음 에러, provider_name별 해석.
- dbhandler: sqlite 스키마로 upsert와 조회, 암호문 json 비노출.
- listenhandler와 api-manager: 정상, 404, 400, 권한 거부(비 슈퍼어드민, direct, accesskey), 응답에 암호문 필드 없음 단언, 비밀 입력 요청 본문이 로그 출력(hook으로 캡처)에 나타나지 않는다는 단언, PUT 행 생성 기본값, config 병합, DELETE secret 복구, `Credentials` 출력 마스킹.
- 마이그레이션: `go test ./migrations/`, `scripts/check-service-migrations.sh`, 두 엔진 upgrade/downgrade/upgrade.
- PR1: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run`을 bin-number-manager에서 실행(common-handler 무변경).
- 관측: `number_manager_provider_unusable_total{provider,reason}`(reason=`no_secret`,`db_undecryptable`,`required_config`)와 `number_manager_provider_purchase_total{provider,result}`(result=`success`,`rejected`,`unconfirmed`) 카운터를 추가하고 `docs/operations.md`에 경보 후보로 적습니다.
- 테스트 PR 구분: 이 PR1은 number-manager 테스트만이며 api-manager/requesthandler 테스트는 PR2입니다(위 목록 중 api-manager 권한 거부 등은 PR2).
- listenhandler: secret PUT 본문이 로그 hook에 나타나지 않음, requestexternal 로그에 Authorization 헤더 없음, 4xx/5xx/네트워크 오류 분류.

못 하는 것(명시): 실제 Telnyx, Twilio 구입 호출(비용이 발생하는 호출), 운영 DB 적용.

## 11. 구현 시 확정 사항

- 키 파싱 즉시 에러는 `number-control`의 `initNumberHandler`에서만 반환합니다(루트 PreRun에 두면 선행 필수인 `migrate up`이 막힘).
- `ForRelease`와 `ForSearch`는 token(비밀)만 요구하며 RequiredConfig는 구입 후보 판정에만 적용합니다(현행 반환/검색이 token만 쓰기 때문). enabled=false여도 반환과 검색은 허용합니다.
- config PUT과 secret PUT의 동시 실행으로 RequiredConfig 결손이 생길 수 있으나 마지막 쓰기 우선이며, 결손 상태는 `credential_source=db`, `usable=false`로 보이고 구입 후보에서 제외됩니다(복구: config 재입력 또는 DELETE secret). `docs/operations.md`에 적습니다.
- `Info.config`는 행의 config를 그대로 보여 주며, 소스가 env이면 행의 config는 무시된다고 문서화합니다(`credential_source`로 구분).
- 메트릭 `provider_unusable_total`은 enabled인 provider에 대해 해석이 실패한 호출마다 증가합니다(상태 전이 아님).
- `Credentials`는 `Secret`에 `json:"-"`를 붙이고 `String`/`GoString`은 값 수신자로 구현합니다.
- `Provider` 인터페이스는 `providerhandler`에 둡니다(`models/provider`는 모델, 센티널, `Credentials`만).
- `excludeIDs`는 `id.Bytes()`로 전달합니다(`gofrs/uuid`의 `Value()`는 문자열이므로 binary(16) 비교에 부적합).
- 영향도 분석의 기준 문서는 `docs/workflows/service-owned-migrations.md`의 Impact analysis와 `bin-number-manager/CLAUDE.md`의 migrations 절입니다. PR 본문에 두 레플리카가 000002를 동시에 적용하는 경로를 `migrationhandler`의 named lock이 직렬화함을 한 줄 남깁니다.
- 비밀 입력은 앞뒤 공백과 개행을 제거해 저장합니다(붙여 넣기로 섞인 개행이 Authorization 헤더를 깨는 것을 막기 위함). 공백만 있으면 400입니다.
