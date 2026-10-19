# VOIP-1616 registrar-manager dbhandler: pass request context to DB calls and check rows.Err

줄 번호는 origin/main b5af852d1 기준.

## 티켓 검토
유효. noctx 11건(h.db.Exec/Query), 모두 ctx 를 받는 함수 안. rows.Err 확인이 없는 조회 6곳. 인터페이스와 mock 은 이미 ctx 를 받으므로 변경 없음. 트랜잭션과 Prepare 는 없음. extension.go, trunks.go, sipauths.go, customerdomains.go 는 이미 *Context 호출.

## 변경 (bin-registrar-manager/pkg/dbhandler)
- Exec -> ExecContext(ctx, ...) 7곳: ast_aor.go 100/221, ast_auth.go 84/206/228, ast_endpoint.go 60/170. Query -> QueryContext(ctx, ...) 4곳: ast_aor.go 136, ast_auth.go 115, ast_contact.go 113, ast_endpoint.go 85.
- rows.Err 6곳. 단건 getter 5곳(astAORGetFromDB 144, astAuthGetFromDB 123, astEndpointGetFromDB 93, ExtensionGetByEndpointID extension.go 253, ExtensionGetByExtension 305)은 `!row.Next()` 분기 안에서 row.Err() 를 먼저 확인해 에러를 반환하고 없을 때만 ErrNotFound. AstContactGetsByEndpoint(122)는 순회 뒤, AstContactsSetToCache 앞에서 확인. 새 에러는 `%w`, 기존 `%v` 줄은 그대로.
- 이미 rows.Err 가 있는 8곳과 AstContactDeleteByEndpoint 는 건드리지 않음.

## 취소 부작용 판단
- 진입 ctx 가 Background(listenhandler/main.go:179, subscribehandler/main.go:138, registrar-control)이고 WithTimeout/WithCancel 파생이 없어 취소로 인한 동작 변화는 없고, rows.Err 로 인한 의도된 동작 변화가 2건(아래 두 항목) 있음. 효과는 noctx 일관성, 향후 취소 도입 대비, 순회 중 오류의 오분류 차단.
- 가장 큰 효과: AstContactGetsByEndpoint 가 순회 중 에러에서 부분 목록을 TTL 3분 캐시에 저장하고 정상 반환하던 것(호출부 contacthandler/contact.go:24)을 에러 반환으로 바꿈.
- extension 253/305 의 일시 오류가 ErrNotFound 로 오분류되던 것은 untyped 에러가 됨. GetByExtension(extensionhandler/extension.go:190~192)에서 typed NotFound(404)로 바뀌던 경로가 listenhandler/main.go default 분기의 400 이 됨(의도된 응답 코드 변화, PR 본문에도 기록). ExtensionGetByEndpointID 의 dbhandler 밖 호출자는 없음. ps_aors/ps_auths/ps_endpoints Get 은 dbhandler 밖 호출자가 없고 내부 UpdateToCache 가 결과를 `_ =` 로 무시(관측 효과 거의 없음).
- 쓰기 Exec 뒤 캐시 갱신과 삭제는 모두 `_ =` 로 에러를 무시하고 트랜잭션이 없어, 전환 대상 Exec 뒤에는 이미 커밋된 쓰기를 실패로 반환하는 경로가 없음. 따라서 WithTimeout(WithoutCancel) 분리는 하지 않음(VOIP-1614 의 BillingConsumeAndRecord 와 다름). 쓰기 Exec 도중 취소 시 서버 반영 여부가 모호해지는 점은 호출자가 에러를 실패로 처리하며, 진입 ctx 가 Background 라 수용.

## 범위 밖 (보고서와 PR 본문에 한 줄씩)
- 쓰기 뒤 `_ =` 캐시 갱신 실패로 stale 이 남는 것(ast_aor/auth/endpoint 의 Create/Update/Delete 와 extension.go 97/349/382, trunks.go 92/221/355 등. 이 PR 이전부터 있던 노출이며 진입 ctx 가 Background 라 취소가 없어 현재 영향 없음. 향후 취소 도입 시 AstAuthUpdate 등에서 stale 가능성이 커짐).
- AstContactDeleteByEndpoint(86~101행)만 DELETE 뒤 캐시 삭제 실패를 반환(호출부 domain_migrate.go:567). 전환 대상 아님.
- 호출부(handler 계층), 다른 서비스, 마이그레이션.

## 검증(verify)
- rowserrcheck 는 이 단건 getter 누락을 잡지 못하므로(수정 전후 동일하게 0건) rows.Err 의 증거는 RED 테스트뿐. go mod tidy 뒤 go.mod 는 github.com/DATA-DOG/go-sqlmock v1.5.2 직접 의존 추가와 go.sum 갱신만 있어야 함.
- 임시 설정(워크트리 밖 yml, linters default none, noctx 와 rowserrcheck 만, vendor 제외, --max-same-issues=0 --max-issues-per-linter=0)으로 bin-registrar-manager 비테스트 0건, 표준 설정 0 issues, .golangci-errorlint.yml --new-from-merge-base=origin/main 통과.
- 신규 테스트(수정 전 코드에서 RED 확인): sqlmock(bin-billing-manager/go.mod 의 v1.5.2 선례, RowError 주입에 필요하며 sqlite 로는 순회 중 오류를 만들 수 없음. go.mod/go.sum 변경은 이 때문)으로 단건 getter 5곳에서 반환 에러가 ErrNotFound 가 아니고 errors.Is 로 원인 보존, AstContactGetsByEndpoint 는 에러 반환과 AstContactsSet 미호출(gomock 에 기대 미등록) 단언. 취소된 ctx 에서 쿼리 중단: 전환한 11곳 전부를 table-driven 으로 한 사이트씩 검사(Query 4곳: astAORGetFromDB, astAuthGetFromDB, AstContactGetsByEndpoint, astEndpointGetFromDB / Exec 7곳: AstAORCreate, AstAORDelete, AstAuthCreate, AstAuthDelete, AstAuthUpdate, AstEndpointCreate, AstEndpointDelete)(sqlite dbTest 사용, SetMaxOpenConns(1)에서 rows 미종료 교착 주의). Exec 계열은 에러뿐 아니라 행이 생성/삭제되지 않았음도 단언하고 테스트 끝에 행을 정리. sqlmock 은 Query 와 QueryContext 를 구분하지 못하므로 Context 전환 검증은 취소 ctx 테스트가 전담하고, 여러 줄 SQL 은 regexp.QuoteMeta 또는 부분 패턴으로 매칭.
- 테스트 작성 주의: extension.go 253/305 는 DB 앞에 cache mock(ExtensionGetByEndpointID, ExtensionGetByCustomerIDANDExtension)이 먼저 호출되므로 미스 에러를 반환하는 gomock 기대를 둠(에러 경로라 extensionSetToCache 는 호출되지 않음). AstContactGetsByEndpoint 는 cache AstContactsGet 미스 후 sqlmock 쿼리, 행 15개 컬럼을 갖춘 첫 행을 AddRow 하고 두 번째 행 이상에 RowError 를 걸어 부분 목록 시나리오를 만듦(AddRow 2개 이상, RowError(1, ...))(행 1개뿐이면 EOF 가 먼저). ast_* 쿼리는 소문자 select 여러 줄, extension 은 squirrel 대문자라 sqlmock 정규식을 사이트별로 다르게(예: (?is)select.*from ps_aors).
- 취소 ctx 테스트 단언: 기존 `%v` 래핑 때문에 errors.Is 가 아니라 strings.Contains(err.Error(), context.Canceled.Error()) 로 단언(선례와 동일, 기존 `%v` 줄은 바꾸지 않음). AstContactGetsByEndpoint 취소 케이스는 cache AstContactsGet 미스 mock 이 먼저 필요. 순회 오류의 부분 목록 대신 에러 반환은 PR 본문에 의도된 동작 변화로 한 줄 기록.
- 변이 시험: rows.Err 확인 제거, 캐시 저장 앞 확인 제거, Context 호출을 되돌림 각각에서 해당 테스트 실패 확인.
- 기존 sqlite 테스트의 기대가 Context 호출에서도 맞는지 확인하고 최소 수정.
- go mod tidy, vendor, generate, test, lint 5단계.
