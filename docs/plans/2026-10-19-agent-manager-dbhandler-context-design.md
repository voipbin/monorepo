# VOIP-1630 agent-manager dbhandler: pass request context to DB calls and pin rows.Err

줄 번호는 origin/main b5af852d1 기준입니다.

## 티켓 검토
부분 유효. 컨텍스트 없는 DB 호출은 프로덕션 1건(pkg/dbhandler/agent.go:549, agentUpdate의 `h.db.Exec`)뿐입니다(git grep 및 noctx 실측). rows.Err() 확인은 조회 사이트 전부에 이미 있습니다(agent.go:213, 359, agent_address.go:151, 220, 255, agent_reserve.go:189). 다만 rows.Err 반환 5곳이 `%v` 로 감싸므로 원인 에러는 errors.Is 로 식별되지 않습니다(agent_address.go:220 의 agentAddressListByAgentIDs 만 raw 반환).

## 변경
- agent.go:549 `h.db.Exec(sqlStr, args...)` -> `h.db.ExecContext(ctx, sqlStr, args...)` (1줄). 시그니처가 이미 ctx 를 받으므로 인터페이스와 mock 은 불변.
- 프로덕션 코드의 그 밖의 변경 없음(rows.Err 보호는 이미 존재, `%v` 줄은 건드리지 않음).
- 테스트 추가(pkg/dbhandler/ctx_rows_test.go, sqlmock 별도 DB). go.mod/go.sum 에는 go-sqlmock v1.5.2(billing-manager 와 같은 버전)만 추가.

## 테스트 설계
- RowError 주입(첫 행 읽기 실패, Next()==false 이고 원인은 rows.Err 로만 보고됨): agentGetFromDB, agentAddressLookupOwnerAgentID 와 AgentGetByCustomerIDAndAddress 경로, AgentList, agentAddressListByAgentID, agentAddressListByAgentIDs, AgentReserveSweep. 단언: error 반환, errors.Is(err, ErrNotFound) 거짓, 원인 문구 포함(문자열). agentAddressListByAgentIDs 는 직접 호출할 때만 errors.Is(원인) 추가 단언(AgentList 경유는 agent.go:374 에서 `%v` 로 다시 감쌈). RowError 는 AddRow 로 행을 먼저 만든 뒤 RowError(0, err) 를 걸어야 Next() 가 false 가 됨. agentAddressListByAgentID 는 직접 호출 테스트로 분리.
- 취소 ctx: agentUpdate(AgentUpdate 경유)가 취소된 ctx 에서 context.Canceled 를 errors.Is 로 반환. ExpectExec 를 반드시 설정(변경 전 Exec 가 ctx 를 무시하고 이를 소비해 성공 반환하는 것이 RED, 설정이 없으면 sqlmock 불일치로 실패하는 가짜 RED). 수정 후에는 database/sql 이 드라이버 호출 전에 ctx 를 검사하므로 ExpectExec 가 소비되지 않아 ExpectationsWereMet 를 호출하지 않음(주석으로 사유 명시).
- 현재 동작 고정(커밋 뒤 갱신 실패): ExpectBegin, 쓰기 ExpectExec, ExpectCommit 뒤 캐시 갱신용 조회(agentGetFromDB)에 ExpectQuery(...).WillReturnError(context.Canceled) 를 걸어도 AgentCreate 는 nil 반환, AgentSetAddresses 는 nil 반환하고 cache.AgentDel 호출을 기대. sqlmock 에는 Commit 직후 ctx 를 취소하는 훅이 없어 ctx 취소 자체가 아니라 '커밋 뒤 갱신 오류가 호출자에게 전파되지 않음'을 고정하며 테스트 이름도 그렇게 붙임.
- 변이 시험: ExecContext 를 Exec 로 되돌리면 취소 테스트 실패, 각 getter 의 rows.Err 분기 제거(agentAddressListByAgentIDs 는 `return rows.Err()` 를 `return nil` 로 변경) 시 해당 RowError 테스트 실패. 단건 getter(agent.go:213, agent_address.go:255)의 확인은 `!Next()` 분기 안에 있음.

- 구현 주의: utilHandler.TimeNow() mock 은 *time.Time 을 반환(DoAndReturn 사용). AgentList 용 정상 행은 GetDBFields(&agent.Agent{}) 열 목록에 id 만 채우고 나머지 NULL. 취소 테스트의 RED 원인은 'nil 반환'이며 성공 시 이어지는 agentUpdateToCache 오류는 무시됨.

## 취소 부작용
진입 ctx 는 Background(listenhandler/main.go:150, subscribehandler/main.go:142, cmd/agent-control)이고 파생 취소가 없어 즉시 동작 변화는 없습니다. dbhandler 안에서 커밋된 쓰기를 실패로 보고하는 경로가 없어(AgentCreate, AgentDelete, AgentSetAddresses, agentUpdate 모두 커밋 뒤 오류를 무시) WithoutCancel 분리는 하지 않습니다. agentUpdate 는 단발 Exec 라 취소 시 서버 반영과 클라이언트 에러가 어긋날 수 있으나 호출자는 에러를 실패로 처리합니다. 이 경우 agentUpdateToCache 에 도달하지 못해 캐시가 무효화되지 않고(TTL 24시간 stale) 메트릭 DBOperationTotal{update,agent} 는 failure 로 집계됩니다(취소가 없는 현재는 도달하지 않음).

## 범위 밖(보고서 한 줄)
- agenthandler/db.go 의 쓰기 성공 뒤 `h.db.AgentGet(ctx)` 재조회(AgentDelete :182, SetPasswordHash :280, SetStatus :386)가 취소 ctx 에서 커밋된 쓰기를 실패로 반환할 잠재 위험.
- 캐시 갱신/삭제 실패 시 캐시 stale(TTL 24시간, cachehandler/handler.go:36).
- 호출부의 `err != dbhandler.ErrNotFound` 직접 비교, 기존 `%v` 스타일(rows.Err 반환 5곳 포함).

## 검증
go mod tidy 후 go.mod/go.sum diff 가 sqlmock 추가 외에 없는지 확인. noctx+rowserrcheck 임시 설정으로 프로덕션 0건, 표준 설정 0 issues, .golangci-errorlint.yml --new-from-merge-base=origin/main 통과, go mod tidy/vendor/generate/test/lint 5단계.
