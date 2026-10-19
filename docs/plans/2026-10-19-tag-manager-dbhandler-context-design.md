# VOIP-1619 tag-manager dbhandler: pass request context to DB calls and check rows.Err

줄 번호는 origin/main b5af852d1 기준.

## 티켓 검토
유효. bin-tag-manager/pkg/dbhandler/tag.go 에 noctx 5건(47 TagCreate Exec, 104 tagGetFromDB Query, 164 TagList Query, 204 TagUpdate Exec, 247 TagDelete Exec), 모두 ctx 보유 함수 안. rows.Err() 확인 0건. 트랜잭션 없음. 인터페이스(6개 메서드)와 mock 은 이미 ctx 를 받으므로 변경 없음. 다른 서비스가 이 dbhandler 를 import 하지 않음. 열린 PR 중 tag-manager 관련 없음.

## 변경
- tag.go 의 h.db.Exec -> ExecContext(ctx, ...) 3곳, h.db.Query -> QueryContext(ctx, ...) 2곳.
- tagGetFromDB: `!rows.Next()` 분기 안에서 rows.Err() 를 먼저 확인해 `%w` 로 감싸 반환, 없을 때만 ErrNotFound. TagList: 루프 뒤 rows.Err() 확인, `%w`.
- 기존 `%v` 줄은 변경하지 않음(.golangci-errorlint.yml 은 변경된 줄만 검사). go.mod 에 go-sqlmock v1.5.2 추가(테스트 전용).
- TagSetBasicInfo 는 TagUpdate 를 호출하므로 별도 변경 없음.

## 취소 부작용 판단
- 진입 ctx 는 listenhandler/main.go:148, subscribehandler/main.go:136, cmd/tag-control 모두 context.Background(). tag-manager 안에 WithTimeout/WithCancel/WithDeadline 로 파생한 ctx 가 없으므로 취소로 인한 즉시 동작 변화는 없음. 변화 경로는 순회 중 DB 오류(rows.Err)가 드러나는 경우뿐.
- ErrNotFound 오판: 호출부(listenhandler/main.go:101,209 는 404 매핑, taghandler/tag.go:45 는 TAG_NOT_FOUND 변환)뿐이고 없으면-생성이나 멱등 분기는 없어 금전, 중복 생성 위험 없음. rows.Err() 확인으로 순회 중 오류는 404 대신 에러로 반환되며 listenhandler/main.go:205~213 의 기본 핸들러가 ErrNotFound 와 VoipbinError 이외를 400 으로 응답하므로 실제 변화는 404 -> 400.
- 커밋 뒤 구간: 트랜잭션이 없고 TagCreate/Update/Delete 는 Exec 뒤 `_ = h.tagUpdateToCache(ctx, id)` 로 오류를 버리고 nil 반환하므로 커밋된 쓰기를 에러로 반환하는 경로가 dbhandler 안에 없음. WithoutCancel 분리는 해당 없음. 쓰기 Exec 도중 취소 시 서버에는 반영됐을 수 있는 모호성이 새로 생기나 호출자는 실패로 처리하고 진입 ctx 가 Background 라 수용.

## 범위 밖(보고서에 한 줄 기록)
- taghandler/db.go 의 dbCreate, dbUpdateInfo, dbDelete 가 쓰기 뒤 TagGet 을 같은 ctx 로 실행해 실패 시 에러만 반환하고 PublishEvent 를 건너뛰는 패턴.
- Exec 뒤 `_ = tagUpdateToCache` 로 취소 ctx 에서 캐시 갱신이 생략되면 캐시(TTL 24시간)가 stale 로 남을 수 있는 성질(이 PR 이전부터 있던 노출, 진입 ctx 가 Background 라 잠재).
- taghandler/event.go:14 EventCustomerDeleted(로그는 29): dbList 오류를 빈 메시지 log.Errorf("") 로 남기고 계속하며 nil 반환. TagList 가 rows.Err 시 nil 을 반환하면 이전의 부분 목록 삭제가 전체 미삭제로 바뀜(진입 ctx 가 Background 라 잠재). 빈 로그 메시지도 이 PR 에서 고치지 않음.
- 호출부(taghandler, listenhandler, subscribehandler), 다른 서비스, 마이그레이션.

## 검증(verify)
- noctx, rowserrcheck 만 켠 임시 설정(워크트리 밖 yml, default none, vendor 제외)으로 non-test 0건, 표준 설정 0 issues, .golangci-errorlint.yml --new-from-merge-base=origin/main 통과.
- 신규 테스트(수정 전 RED 확인 후 통과): tagGetFromDB 와 TagList 에 sqlmock RowError 주입 시 반환 에러가 ErrNotFound 가 아니고 원인이 errors.Is 로 보존됨(TagList 는 에러 반환과 nil 결과). Exec 3종(TagCreate, TagUpdate, TagDelete)과 Query 2종(TagGet 의 DB 경로, TagList)에 이미 취소된 ctx 를 주면 DB 호출이 즉시 실패(쿼리 도중 중단을 검증하는 것이 아님). 취소 ctx 테스트는 sqlite(dbTest) 로 하며 고유 ID 를 쓰고 행을 먼저 삽입한 뒤 취소한다(행이 없으면 수정 전에도 ErrNotFound 로 err != nil 이 되어 RED 가 무의미). sqlmock 으로 하게 되면 ExpectExec/ExpectQuery 를 반드시 등록한다. ctx 오류는 기존 `%v` 줄 때문에 errors.Is 로 단언하지 않고 err != nil 로만, 원인 보존 단언은 신규 `%w` 두 곳(RowError)에만 적용. TagCreate, TagUpdate, TagDelete 는 TimeNow gomock 기대를 두고 취소 시 캐시 갱신이 호출되지 않음. TagSetBasicInfo 가 TagUpdate 를 거치므로 TagUpdate 로 대표.
- 변이 시험(별도 $TMPDIR 복사본에서만, 호출 5곳과 rows.Err 2곳을 하나씩 개별 변이): 각 변이가 신규 테스트를 실패시켜야 함.
- 기존 sqlite 테스트가 그대로 통과하는지 확인하고 최소 수정.
- go mod tidy, vendor(추적 대상 아님), generate(mock diff 없음), test, lint 5단계. go.mod/go.sum 은 tidy 뒤 git diff 로 sqlmock 추가 외 변경 없음을 확인.
