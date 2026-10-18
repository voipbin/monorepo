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
- 쓰기 직후의 *UpdateToCache(ctx)(Create/Update/Delete류 공통)는 DB 재조회와 redis Set 이 모두 ctx 에 묶인다. ctx 가 만료/취소되면 갱신이 건너뛰어져 캐시가 stale 로 남을 수 있고(에러는 _ = 로 무시), Delete 후에는 삭제된 행이 캐시 TTL 24h(cachehandler/handler.go:37) 동안 캐시 우선 조회(AIcallGet 등)에 남고, 캐시 우선 AIcallGet 으로 읽은 뒤 FieldMetadata 전체를 다시 쓰는 경로(UpdateListenState, stopListening, persistToolMap)가 stale 스냅샷을 쓸 수 있다. 다만 변경 전에도 만료된 ctx 에서는 redis Set(ctx)가 실패해 같은 stale 결과였으므로 stale 자체는 새로 생기는 결과 종류가 아니고(DB 재조회 단계가 ctx 에 묶이며 창이 약간 넓어질 뿐), 쓰기 후 갱신을 WithoutCancel 로 분리하는 것은 범위 확대이므로 하지 않고 수용한다. 분리가 필요하면 별도 후속 판단.
- 행 순회 중 취소(QueryContext): ctx 가 쿼리 반환 뒤 행을 읽는 중 취소되면 rows.Next() 가 false 를 돌려주고 원인은 rows.Err() 로만 나온다. 이 변경 전에는 DB 가 ctx 를 무시해 불가능했던 새 실패 모드(단건 조회가 ErrNotFound 로 오판, 목록이 에러 없이 부분 결과 반환)이므로 이 PR 범위로 처리한다(대표 default 결정 2026-10-18). 이 PR 이 Query 에서 QueryContext 로 바꾼 17곳에 한해 get-by-ID 의 !rows.Next() 분기는 rows.Err() 를 먼저 확인해 에러를 반환하고 없을 때만 ErrNotFound, 목록은 순회 뒤 rows.Err() 확인으로 에러를 반환한다. 이 PR 이전부터 QueryContext 였던 사이트(mcpserver.go, participant.go, mcpoauthstate.go, MessageAssistantReplyExists)는 건드리지 않고 후속 제안으로만 남긴다.
- 쓰기 Exec 도중 취소되면 드라이버가 에러를 반환하지만 서버에는 이미 반영됐을 수 있음(결과 모호). 변경 전에는 완료까지 대기해 성공/실패가 확정됐음. 모호한 경로: Create류(AIcallCreate 등)와 rows-affected 판정(AIcallUpdateIfActive, SummaryUpdateStatusDoneIfNotDone). 호출부는 에러를 실패로 반환한다(AIcallCreate 호출처 aicallhandler/db.go:83,158). RPC 경로는 context.Background() 기반이라 이 모호성은 위 deadline 경로나 호출자 ctx 가 실제로 만료된 경우에만 발생하고, 그 경우 호출자는 이미 결과를 포기한 상태이므로 수용(자동 재시도 로직은 이 변경으로 추가하지 않음).

- AIActivateInsight: 커밋 뒤 두 번의 aiGetFromDB 는 WithTimeout(WithoutCancel(ctx), 5s) 로 분리. 커밋된 활성화가 취소 때문에 에러로 보이면 호출부가 update 이벤트 발행을 건너뛰기 때문.
- aicallhandler.rollbackListenState: 이 함수는 만료된 deadline ctx 에서도 호출되고 결과가 버려진다. DB 읽기와 쓰기를 WithTimeout(WithoutCancel(ctx), AIcallListenStartLockReleaseTimeoutSeconds) 으로 분리(같은 파일의 락 해제 패턴). 읽기와 쓰기가 한 예산을 공유하며 기본 3초(config/main.go) 의존, 설정 0 이하는 기동 시 config.Validate()(validateListenConfig positives)가 거부. 롤백의 읽기, 쓰기, 쓰기 후 캐시 갱신이 모두 이 3초 예산을 쓰므로 DB 지연이 3초를 넘으면 롤백이 전에 없던 방식으로 실패할 수 있으며 이는 수용(전에는 상한 없음, 락 해제와 같은 패턴). 변경 전에는 DB 가 ctx 를 무시해 만료 ctx 에서도 DB 롤백이 성공했으므로 그 경우의 동작을 보존(ListenAIcallIDRemove(ctx)는 전과 같이 만료 ctx 에서 실패). 이 파일 변경은 dbhandler 변경의 직접 결과이므로 범위 안.
- 호출 경로 조사: listenhandler, subscribehandler 는 context.Background() 를 사용해 RPC 취소가 전파되지 않음. deadline 을 가진 파생 ctx 아래의 DB 호출은 다음과 같고, 변경 후 만료 시 중단된다(전에는 DB 가 ctx 를 무시). 해당 timeout 들은 후처리가 무한히 계속되지 않도록 묶는 목적이므로(tool.go B25 주석) 의도된 동작으로 수용하고 분리하지 않는다.
  - 쓰기 포함: listen_trigger.go:72 runListenStart(AIcallListenEnsureGoroutineTimeoutSeconds) 안의 UpdateListenState(:531 선쓰기, :507 재사용, :591 승자 채택)와 AIcallGetSkipCache(:479). :531, :591 실패는 rollbackListenState 로 보정되고 :507 실패는 롤백 없이 "failed" 반환(DB 반영 뒤 redis SADD 가 없는 상태가 남을 수 있으며 기존 SADD 실패 경로와 같은 종류). listen.go:547,561 stopListening 의 AIcallGet + AIcallUpdateNoTouchTMUpdate(2분 turnCtx 안, 턴 초반 호출, redis 호출도 같은 ctx 라 위험 낮음).
  - 읽기만: listen.go:126 messageHandler.List 등 turnCtx 읽기, tool.go:178 mcpCtx 아래 resolveActiveAIForMcp 와 mcpServerHandler.Get(fillSuccess/fillFailed 는 메모리 내 함수이고 결과 메시지 toolCreateResultMessage 는 바깥 ctx 로 생성), mcp_tool.go:85 discoverCtx, start.go:103 5s fetchCtx, aiaudithandler/main.go:194 geminiCtx 의 MessageList, aipromptproposalhandler/main.go:162 의 loadAuditBlocks(최종 쓰기는 별도 writeCtx). 만료 시 대체로 해당 작업이 실패 처리되며, 순회 중 만료는 위 rows.Err 항목처럼 조용한 부분 결과가 가능.

## 범위 밖
goroutine 과 dbhandler, aicallhandler 의 기존 context.Background() 호출(건수 미특정), 거대 함수 분리, %w 전환, 다른 서비스, 마이그레이션.

## 검증
- noctx 임시 설정으로 non-test 코드 35건 -> 0건(_test.go 12건은 남으며 표준 lint 가 테스트를 제외), 표준 lint 0건.
- 신규 rollback_listen_ctx_test.go: 만료된 ctx 에서도 롤백이 살아있는 ctx 로 DB 호출(수정 전 실패 확인).
- 신규 ctxcancel_test.go: 이미 취소된 ctx 로 read/write/list 가 중단됨. 쓰기/목록 케이스는 변경 전 코드에서 동작상 실패, private get 두 케이스는 변경 전 시그니처 불일치로 컴파일 불가. 순회 중 취소는 rowserr_test.go 가 대체.
- 신규 rowserr_test.go: 행 순회가 즉시 실패하는 가짜 드라이버로 17곳 모두 ErrNotFound 오판이나 빈 성공 없이 에러를 반환함(수정 전 17건 모두 실패 확인). 가짜 드라이버는 첫 Next 에서 실패하므로 "일부 행을 읽은 뒤 실패"는 별도 검증하지 않음(같은 rows.Err 경로).
- go mod tidy/vendor/generate/test/lint 5단계 통과.
- deadline 경로(runListenStart, turnCtx, mcpCtx)의 만료 시 중단과 쓰기 결과 모호성은 위 판단으로 수용하며 전용 테스트는 없음(결정적 주입 어려움).
- 미검증: AIActivateInsight 커밋 뒤 읽기의 WithoutCancel 분리는 전용 테스트가 없음(단일 커넥션 sqlite 에서 커밋 직후 취소를 결정적으로 주입하기 어려움). 코드 리뷰와 호출 경로 조사로 대체.
