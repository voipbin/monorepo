# VOIP-1654 PR2: number provider admin API (requesthandler, OpenAPI, api-manager, docsdev)

PR1(`2026-10-26-multi-provider-number-purchase-design.md`)이 머지한 number-manager RPC(`/v1/number_providers...`)를 외부에 어드민 표면으로 노출합니다. number-manager 로직과 RPC 응답은 바뀌지 않으며, 응답 모델에 `WebhookMessage` 정의만 더합니다(3절 0항).

## 1. 착수 전 검토 (착수 전 검토: 유효)

- 유효(반증 못함): main 에 `/number_providers` 관련 requesthandler 메서드, OpenAPI 경로, servicehandler, server 핸들러가 없음(grep 0건). PR1 RPC(`bin-number-manager/pkg/listenhandler/v1_number_providers.go`)는 main(d0588f5e5)에 있고 요청 DTO는 `request.V1DataNumberProvidersIDPut/IDSecretPut/IDSecretDelete`(모두 `actor_id` 포함), 응답은 `provider.Info`를 직접 마샬링(스타일 A).
- 틀린 주장/보정: 카드의 "ConvertWebhookMessage 를 거친다"는 설계 문서 주장에 대해 `provider.Info`에는 `WebhookMessage`/`ConvertWebhookMessage()`가 없습니다. 모든 외부 리소스가 따르는 컨벤션(루트 CLAUDE.md, RST는 WebhookMessage 기준)을 위해 `bin-number-manager/models/provider/webhook.go`에 `WebhookMessage` 타입과 `func (i *Info) ConvertWebhookMessage() *WebhookMessage`를 추가합니다(선례 `models/number/webhook.go`. Info 와 필드 동일, 비밀 필드 없음, RPC 응답과 number-manager 로직 무변경). PR1 설계 서술의 두 가지를 이 PR 에서 보정합니다: requesthandler 메서드에 `DeleteSecret`이 더해지고, 변환은 api-manager servicehandler 에서 `ConvertWebhookMessage()`로 합니다.
- 최상위 `/providers`는 route-manager SIP 캐리어 리소스이므로 `/number_providers`를 씁니다. `service_agents` 경로는 만들지 않습니다.

## 2. 공개 API (모두 SuperAdmin 전용)

| 메서드 | 경로 | 본문 | RPC |
|---|---|---|---|
| GET | `/number_providers` | 없음 | GET `/v1/number_providers` |
| GET | `/number_providers/{provider_name}` | 없음 | GET `/v1/number_providers/{name}` |
| PUT | `/number_providers/{provider_name}` | `enabled?`, `priority?`, `config?`(map string) | PUT 동일 경로 |
| PUT | `/number_providers/{provider_name}/secret` | `secret`(writeOnly) | PUT `.../secret` |
| DELETE | `/number_providers/{provider_name}/secret` | 없음(공개 API) | DELETE `.../secret`, RPC 본문은 `{"actor_id":...}` 필수(number-manager 가 `json.Unmarshal(m.Data)` 실패 시 400, `v1_number_providers.go`의 SecretDelete 핸들러). 기존 `NumberV1NumberDelete`의 nil 본문 패턴을 복사하지 않음 |

- PUT `{provider_name}`의 `config`는 키 단위 병합이며 값이 빈 문자열이면 그 키를 삭제합니다(PR1 7절). OpenAPI 설명과 RST 에 이 의미를 적습니다. 본문에는 `enabled`, `priority`, `config` 중 최소 하나가 있어야 하며 비어 있으면 number-manager 가 400('nothing to update')을 반환합니다(설명과 RST 에 명시, 서버 테스트에 `{}` 케이스). 생성 타입은 `secret`이 `*string`, `config`가 `*map[string]string`, `enabled`/`priority`가 포인터이므로 server 핸들러가 nil 을 검사합니다(`secret` nil 이면 400 `INVALID_JSON_BODY`, `config` nil 은 생략으로 변환하고 nil 포인터 역참조 금지). server 테스트에 `{}`, `{"secret":null}`, `enabled=false`/`priority=0` 전달(requesthandler 까지 값 보존) 케이스를 둡니다. `credential_source`는 `x-enum-varnames`를 지정한 최상위 enum 스키마로 두고 생성 후 상수 이름 충돌이 없는지 확인합니다. DELETE secret 은 등록된 provider 라도 DB 행이 없으면 404 이며(환경변수 자격만 쓰는 상태), 같은 provider config 의 동시 PUT 은 마지막 쓰기가 이깁니다(OpenAPI 설명과 RST 에 적음). RST 오류 절에는 400 의 대표 원인(허용되지 않은 config 키, priority 범위, 필수 config 선행 입력, secret 길이, 빈 본문)을 나열합니다. number-manager 는 상태 코드만 전달하므로 응답 메시지로는 원인이 구분되지 않습니다. servicehandler 시그니처는 `enabled *bool`, `priority *int`, `config map[string]string`으로 포인터 의미를 requesthandler 까지 유지합니다. `secret`은 number-manager 가 앞뒤 공백을 제거한 뒤 바이트(1024 이하) 기준으로 검사하므로 OpenAPI 설명에 이를 적고(`maxLength`는 문자 기준이라 참고용), 공백만 있는 값과 멀티바이트 값을 테스트합니다. `priority` 범위(0 이상 1000000 이하), config 키 허용 목록과 길이 제한은 number-manager 가 검증하며 400 으로 돌아옵니다.
- 목록 응답은 플랫폼 관례 `{result:[...], next_page_token:""}`(`GenerateListResponse`, provider 수가 한 자리라 페이지네이션 파라미터 없음). 단건, PUT, DELETE 는 `NumberManagerNumberProvider` 객체.
- 스키마 `NumberManagerNumberProvider`(= `WebhookMessage` 필드): `provider_name`(string, 패턴 `^[a-z0-9_]{1,32}$`), `enabled`, `priority`, `config`, `has_secret`, `key_version`, `credential_source`(enum db/env/none/db_undecryptable), `usable`, `updated_by`, `secret_updated_by`(uuid, 선택), `tm_update`, `tm_secret_update`(선택). 비밀 값, 암호문 필드는 어디에도 없음. 요청 스키마의 `secret`은 `writeOnly: true`, `minLength: 1`, `maxLength: 1024`.
- 상태 코드: 400(검증), 401, 403, 404(알 수 없는 provider), 503(암호화 키 미설정, number-manager 가 503 반환), 500. 모두 기존 `translateToVoipbinError`의 requesthandler 센티널 매핑(ErrBadRequest, ErrNotFound, ErrServiceUnavailable)을 재사용하며 신규 매핑 없음.

## 3. 계층별 변경

0. `bin-number-manager/models/provider/webhook.go`(+테스트), `bin-number-manager/docs/domain.md`(기존 'Provider Settings' 절에 WebhookMessage 서술 추가)와 `docs/architecture.md`의 Models 표(`models/provider` 행 추가). `dependencies.md`, 라우팅 표는 변경 없음. number-manager 서비스의 검증 워크플로(tidy, vendor, generate, test, lint)를 실행합니다.
1. `bin-common-handler/pkg/requesthandler/number_provider.go`(신규, 기존 `nunmber_*` 오타 파일명은 건드리지 않음): `NumberV1NumberProviderList(ctx)`, `Get(ctx, name)`, `Update(ctx, name, actorID, enabled *bool, priority *int, config map[string]string)`, `UpdateSecret(ctx, name, actorID, secret string)`, `DeleteSecret(ctx, name, actorID)`(RPC 본문 `{actor_id}` 필수). 반환은 `*nmprovider.Info`(목록은 `[]*nmprovider.Info`). 경로는 `url.PathEscape(name)`. 타임아웃 15000ms. `main.go` 인터페이스와 mock 갱신. 에러 문자열에 본문을 넣지 않음(`sendRequest`의 에러는 queue, method, uri 만 포함함을 코드로 확인).
2. `bin-openapi-manager`: `openapi/paths/number_providers/main.yaml`, `id.yaml`, `id_secret.yaml`, 루트 `openapi.yaml`의 `tags:`와 `paths:` `$ref` 등록, 스키마, `NumberProvider` 태그(`externalDocs.url`은 `.../docs/number_provider.html`, RST 의 API Reference 링크는 `redoc/#tag/NumberProvider`). `provider_name`은 기존 enum `NumberManagerNumberProviderName`(telnyx, twilio, messagebird, messagebird 는 number-manager 에 없는 값)을 재사용하지 않고 패턴 문자열로 둡니다: 레지스트리에 provider 가 추가될 때 spec 변경 없이 동작하게 하고 서버 쪽 경계는 servicehandler 검사가 맡습니다. 두 패턴은 같은 문자열 상수를 쓰고 경계값(길이 33, 대문자, `/`, `..`)을 테스트합니다. 생성물은 `bin-openapi-manager`에서 `go generate ./...`만 사용합니다(`configs/config_model/generate.go`의 `go tool`이 `go.mod` 고정 버전을 쓰며 CI `bin-openapi-manager-validate`의 'Verify generated models match committed' 단계가 같은 명령 후 `git diff --exit-code gens/models/gen.go`로 검사. PATH 의 `oapi-codegen` 바이너리나 `@latest`로 gen.go 를 만들지 않음). CI 의 별도 '문법 검증' 단계는 `@latest` 로 spec 을 파싱하므로 로컬에서 `go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`로 spec 파싱이 통과하는지도 확인합니다(출력은 버림). 스키마는 openapi-manager 규칙 1~5 를 따릅니다: UUID 는 `format: uuid` + `x-go-type: string`, `updated_by`, `secret_updated_by` 설명에 출처 엔드포인트, 모든 leaf 에 현실적 `example`, 시각은 `date-time`.
3. `bin-api-manager`: `bin-api-manager`에서 `go generate ./...`(openapi-manager 생성 다음 순서)로 `gens/openapi_server/gen.go`와 redoc 산출물 재생성. redoc 생성은 `npx` 가 없으면 조용히 건너뛰므로 `npx` 존재를 먼저 확인하고 `gens/openapi_redoc/openapi.json`과 `api.html`에 `number_providers` 변경이 생겼는지 diff 로 확인합니다(api-manager 쪽은 CI 에 gen diff 검사가 없어 로컬이 유일한 안전망). `server/number_providers.go`(5개 핸들러. 기존 `providers.go`의 로그 필드 `"request_address": c.ClientIP`는 함수값을 호출하지 않는 결함이라 복사하지 않고 `c.ClientIP()`를 쓰거나 필드를 생략. 두 PUT 은 `http.MaxBytesReader`(8 KiB)로 본문을 제한해 API 레이어에 전역 크기 제한과 요청 스키마 검증 미들웨어가 없는 점을 보완하며, 초과도 다른 바인딩 실패와 같은 400 `INVALID_JSON_BODY`로 통일), `pkg/servicehandler/number_provider.go`(5개 메서드), 인터페이스와 mock, `docs/routing.md`.
4. `docsdev/source`: 파일명 `number_provider.rst`(label `number-provider-main`, 기존 `provider-main`과 구분), `number_provider_overview.rst`, `number_provider_struct_number_provider.rst`, `number_provider_tutorial.rst`(예시는 명백한 가짜 값), `index.rst`의 `number` 다음에 등록, `number_overview.rst`에 상호 참조 한 줄. 기존 SIP 캐리어 `provider*.rst`와 혼동되지 않게 첫 문단에 구분을 적습니다. 클린 빌드 후 `build/` 를 force add. Sphinx 경고(미등록 문서, 중복 label, 깨진 `:ref:`)가 새로 생기지 않았는지 확인하고, 구조 문서의 필드 목록은 `WebhookMessage` 와 일치시킵니다.

### 검증 범위

`bin-common-handler`의 `RequestHandler` 인터페이스와 mock 변경은 모든 서비스에 영향을 주므로 `docs/workflows/special-cases.md` 절차를 따릅니다. (a) `bin-common-handler`, `bin-openapi-manager`, `bin-api-manager`, `bin-number-manager` 각각에서 5단계 검증(`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run`)을 실행합니다. `bin-openapi-manager`는 생성 뒤 `cd ../bin-api-manager && go build ./...`도 확인합니다. (b) 나머지 서비스는 `go clean -testcache` 후 `go build ./... && go test ./...`를 돌리고(서비스별 `go mod tidy`/`vendor` 후, 새 패키지 `bin-number-manager/models/provider`가 vendor 에 들어오는 영향 확인) 서비스 수와 결과를 PR 본문에 적습니다. (c) 생성 뒤 `gofmt`와 `git status`로 의도하지 않은 산출물 변경이 없는지 확인합니다.

## 4. 권한과 actor

- servicehandler 모든 메서드: ① `a.IsDirect()`이면 `ErrDirectAccessNotSupported` ② `hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin)` 아니면 `ErrPermissionDenied` ③ 쓰기(PUT 두 개, DELETE)는 `actor_id = a.AgentID()`를 전달합니다. SuperAdmin 통과는 Agent 가 있어야만 가능하므로(접근키, delegate, Agent nil 은 `HasPermission`이 ②에서 이미 거부) 별도 Nil 검사는 두지 않고, 만에 하나 Nil 이 전달돼도 number-manager 가 400 으로 거부합니다. 테스트는 접근키와 delegate 거부를 ②에서 검증합니다. customer_id 주입 없음(어드민 전역 리소스).
- `provider_name`은 요청 경로에서 온 값이므로 servicehandler 가 `^[a-z0-9_]{1,32}$`를 검사해 어긋나면 `ErrInvalidArgument`(RPC URI 오염, 의도치 않은 라우트 매칭 방지. 검사는 권한 확인 뒤).
- 읽기(GET)도 SuperAdmin 전용입니다(config 와 상태가 운영 정보).

## 5. 비밀값 취급

- 입력은 PUT secret 본문의 `secret` 하나이며 server 핸들러는 `c.BindJSON`(실패 시 바인딩 에러를 `c.Errors`에 남기고 헤더를 먼저 씀) 대신 `c.ShouldBindJSON`을 쓰고, 실패 시 `err`를 로그, 응답, `c.Errors` 어디에도 넘기지 않으며(디코더 메시지가 입력을 인용할 수 있음) 고정 메시지 `INVALID_JSON_BODY`로만 응답합니다. 핸들러, servicehandler, requesthandler 어디서도 `req` 구조체나 secret 을 로그 필드, 에러, 이벤트에 넣지 않습니다. 이벤트 발행 없음.
- gin 접근 로그는 본문을 기록하지 않음을 확인함(`logmask`는 query/cookie 만 다룸). secret 은 URL 이 아닌 본문으로만 전달.
- 응답은 `has_secret`, `key_version`, `credential_source` 만 노출. RabbitMQ RPC 구간에서 secret 이 평문 JSON 으로 내부 브로커를 통과하는 점은 기존 `POST /providers/setup`의 `api_key`와 동일한 신뢰 경계이며 이 PR 에서 바꾸지 않습니다(브로커 TLS 여부는 인프라 몫이라 확인하지 못함).
- 검증: 실제 gin 라우터에 `logmask.Middlewares`(접근 로그와 recovery)를 붙이고 로그 hook 을 달아, 성공과 실패 경로(바인딩 실패, 크기 초과, RPC 오류, mock servicehandler 가 panic 을 내는 경우의 recovery 출력) 모두에서 secret 이 접근 로그, 로그 hook, 응답 어디에도 없음을 단언합니다. 추가로 로그 hook 으로 server, servicehandler, requesthandler 테스트가 secret 이 출력에 없음을 단언(실패 경로 포함), 응답 JSON 에 `secret`, `ciphertext` 키가 없음을 단언.

## 6. 테스트 계획(코딩 단계에서 작성)

규칙: 신규 테스트 함수는 `Test_<대상>[_<시나리오>]`, gomock 컨트롤러 변수는 `mc`(scripts/check-test-conventions.sh 는 추가된 줄을 검사). 픽스처는 `test-secret-value`처럼 명백한 가짜 값과 `550e8400-e29b-41d4-a716-446655440000` 류 예시 UUID 만 씁니다.


- requesthandler: 5개 메서드(DeleteSecret 본문 `{actor_id}` 포함)의 URI/메서드/본문(actor_id 포함, 미지정 포인터 생략) 매칭, 응답 파싱, 404/400/503 센티널, 전송 실패 에러에 secret 미포함.
- servicehandler: 정상, direct 거부, 비 SuperAdmin 거부, 접근키 쓰기 거부, actor 전달(`gomock` 인자 정확 매칭), provider_name 형식 위반, RPC 에러 전파, 로그 hook secret 미포함.
- server: 5개 경로 HTTP 라우팅(실제 gin 라우터), 본문 바인딩 실패 400, 목록 래퍼 `result`, 응답에 secret 키 없음, 인증 정보 없음 401.
- 모델: `Info.ConvertWebhookMessage()`가 모든 필드를 옮기고 직렬화에 secret 필드가 없음.

## 7. 영향도

DB 스키마 변경 없음, 마이그레이션 없음(스키마 변경 검증 절차 해당 없음). 신규 공개 API 추가만이며 기존 엔드포인트 무변경. 구 코드와 신 코드 호환: api-manager 가 number-manager 의 provider RPC 보다 먼저 배포되면 신규 경로만 실패(upstream 404 또는 오류)하고 기존 경로는 영향이 없으며, 반대 순서는 신규 경로가 없을 뿐입니다. 서비스별 CI 트리거는 자기 디렉터리 기준이라 `bin-common-handler` 변경이 다른 서비스를 자동 재배포하지 않고, `bin-openapi-manager` 변경은 api-manager 파이프라인도 실행하며, `bin-number-manager/models/provider/webhook.go` 추가로 number-manager 파이프라인도 실행됩니다(런타임 동작 무변경). PR 본문에 `swag init`은 이 저장소 구성에 해당 없음을 적습니다. 롤백은 되돌림 PR.

## 8. 범위 밖

square-admin 화면, 신규 provider 구현체, 운영 값 주입, 고객 대상 API 변경, number-manager 로직 변경.
