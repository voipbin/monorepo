# VOIP-1612 ai-manager dbhandler: pass request context to DB calls

## 티켓 검토
유효. noctx 35건(h.db.Exec/Query), 표준 lint 설정은 0건. 30곳은 감싼 함수가 ctx 보유, 5곳은 private *GetFromDB 에 ctx 없음.

## 변경
- bin-ai-manager/pkg/dbhandler 의 h.db.Exec -> ExecContext(ctx, ...), h.db.Query -> QueryContext(ctx, ...).
- private 5개(aicallGetFromDB, summaryGetFromDB, aiauditGetFromDB, aiPromptHistoryGetFromDB, aipromptproposalGetFromDB)에 ctx 매개변수 추가. 호출처는 모두 dbhandler 내부이며 ctx 보유 함수 안.
- DBHandler 인터페이스와 mock 은 변경 없음(재생성 불필요).

## 취소 부작용 판단
- 캐시 갱신 defer(ai.go AIActivateInsight, aipromptproposal.go AIAcceptProposal)의 context.Background() 는 의도(커밋 뒤 캐시 갱신이 취소로 건너뛰지 않도록)이므로 그대로 유지.
- BeginTx 는 이미 ctx 사용. Rollback 은 ctx 인자 없는 API 이며 database/sql 이 ctx 취소 시 tx 를 자동 롤백. 변경 없음.
- public 경로의 aicallUpdateToCache(ctx) 등은 redis 호출에 이미 ctx 사용. 취소 시 DB 재조회도 함께 중단되어 일관됨(캐시 미갱신은 TTL 과 다음 조회가 복구). 쓰기 자체는 Exec 가 이미 끝난 뒤라 영향 없음.
- 쓰기 Exec 도중 취소되면 드라이버가 쿼리를 중단하고 에러를 반환(기존에는 완료까지 대기). 호출부는 이미 에러 경로를 처리.

- AIActivateInsight: 커밋 뒤 두 번의 aiGetFromDB 는 context.WithoutCancel(ctx) 로 분리. 커밋된 활성화가 취소 때문에 에러로 보이면 호출부가 update 이벤트 발행을 건너뛰기 때문.
- aicallhandler.rollbackListenState: 이 함수는 만료된 deadline ctx 에서도 호출되고 결과가 버려진다. DB 읽기와 쓰기를 WithTimeout(WithoutCancel(ctx), 기존 락 해제 타임아웃) 으로 분리(같은 파일의 락 해제 패턴). 변경 전에는 DB 가 ctx 를 무시해 롤백이 성공했으므로 동작 보존. 이 파일 변경은 dbhandler 변경의 직접 결과이므로 범위 안.
- 호출 경로 조사: listenhandler, subscribehandler 는 context.Background() 를 사용해 RPC 취소가 전파되지 않음. 파생 WithTimeout 은 start.go 5s fetchCtx, mcp 2s/mcpCtx 등 읽기 전용(best-effort)뿐.

## 범위 밖
goroutine 과 context.Background 8곳, 거대 함수 분리, %w 전환, 다른 서비스, 마이그레이션.

## 검증
- noctx 임시 설정으로 35건 -> 0건, 표준 lint 0건.
- 신규 rollback_listen_ctx_test.go: 만료된 ctx 에서도 롤백이 살아있는 ctx 로 DB 호출(수정 전 실패 확인).
- 신규 ctxcancel_test.go: 취소된 ctx 로 read/write/list 가 중단됨(변경 전 코드에서 실패 확인).
- go mod tidy/vendor/generate/test/lint 5단계 통과.
