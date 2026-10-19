# VOIP-1625 campaign-manager dbhandler: pass request context to DB calls and check rows.Err

## 티켓 검토
부분 유효. 컨텍스트 없는 DB 호출은 3건뿐: campaign.go:349 CampaignUpdate, campaigncall.go:375 CampaigncallUpdate, outplan.go:272 OutplanUpdate 의 `h.db.Exec`. 세 함수 모두 ctx 를 이미 받으므로 인터페이스와 mock 은 변경 없음. rows.Err 확인은 단건 getter 5곳과 목록 5곳에 이미 있으나 반환 10줄이 `%v` 라서 `errors.Is` 로 원인을 확인할 수 없음.

## 변경
- 3곳 `h.db.Exec(sqlStr, args...)` -> `h.db.ExecContext(ctx, sqlStr, args...)`.
- rows.Err 반환 10줄(campaign.go:128/248/301, campaigncall.go:125/179/215/270/346, outplan.go:126/233)의 `%v` -> `%w`. 로직 변화 없음. 이 외 기존 `%v` 줄(non-test 50건 중 40건)은 유지(서비스에 errors.Is(context.*) 호출부 0건). QueryContext 실패의 `could not query ... %v` 도 유지하므로 취소 ctx 의 목록/단건 조회를 errors.Is 로 확인하는 테스트는 쓰지 않음(범위 밖).
- go.mod, go.sum 에 go-sqlmock v1.5.2 추가(bin-billing-manager 선례).

## 취소 부작용 판단
- 진입 ctx 는 listenhandler/main.go:173, subscribehandler/main.go:149, cmd/campaign-control 모두 Background. 유일한 파생 취소는 campaignhandler/reconcile.go:105 WithTimeout(90s) 이며 dbhandler 로는 이미 QueryContext 인 CampaignListDeletedSince 만 닿음. 따라서 즉시 동작 변화 없음.
- 트랜잭션 0건. 세 Update 는 Exec 뒤 `_ = xUpdateToCache` 로 에러를 무시하므로 커밋된 쓰기를 실패로 보고하는 경로 없음(WithoutCancel 분리 불필요).
- 쓰기 중 취소 시 서버 반영 여부가 모호해질 수 있으나 호출자는 에러를 실패로 처리하고, 진입 ctx 가 Background 라 수용.
- ErrNotFound 호출부(campaigncallhandler/campaigncall.go:107, campaignhandler/campaign.go:193, outplanhandler/outplan.go:88, listenhandler/main.go:120,315)는 404 매핑뿐, create-if-missing 없음.

## 범위 밖
호출부 변경, 기존 `%v` 줄, Exec 뒤 `_ = *UpdateToCache` 캐시 stale(TTL 24h, cachehandler/handler.go:37), 다른 서비스, 마이그레이션.

## 검증(verify)
- 신규 `ctx_rows_test.go`: (1) sqlmock RowError 주입으로 단건 getter 5곳과 목록 5곳이 `errors.Is(err, errBroken)` 참, `errors.Is(err, ErrNotFound)` 거짓. (2) 취소된 ctx 에서 세 Update 가 중단되고 `context.Canceled` 가 `%w` 체인에 있음(sqlite dbTest 사용: database/sql 이 드라이버 호출 전에 ctx.Err() 를 확인하므로 취소 ctx 에서 즉시 실패하며, 수정 전 Exec 는 ctx 를 무시해 성공하므로 RED 성립. sqlmock 은 RowError 주입에만 사용. `-count=2` 와 shared 메모리 DB 상호작용은 구현 때 확인). 수정 전 코드에서 RED 확인.
- 변이 시험(scratch worktree): ExecContext 를 Exec 로, 각 rows.Err 확인 제거 시 해당 테스트 실패.
- noctx+rowserrcheck 임시 설정 non-test 0건, 표준 설정 0 issues, `.golangci-errorlint.yml --new-from-merge-base=origin/main` 통과, pkg/dbhandler `-count=2` 를 origin/main 과 비교, 5단계(tidy/vendor/generate/test/lint).
