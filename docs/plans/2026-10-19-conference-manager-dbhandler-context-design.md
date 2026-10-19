# VOIP-1624 conference-manager dbhandler: pass request context to DB calls and check rows.Err

## 티켓 검토
유효. 컨텍스트 없는 h.db.Exec 4건(conference.go:337, 369, 442, 468), 모두 ctx 보유 함수 안. `!row.Next()` 단건 getter 중 rows.Err 확인이 없는 곳 2곳(conference.go:278, conferencecall.go:187). 인터페이스와 mock 은 이미 ctx 를 받으므로 변경 없음.

## 변경
- bin-conference-manager/pkg/dbhandler/conference.go: h.db.Exec -> h.db.ExecContext(ctx, ...) 4곳(ConferenceAddConferencecallID, ConferenceRemoveConferencecallID, ConferenceAddRecordingIDs, ConferenceAddTranscribeIDs).
- ConferenceGetByConfbridgeID, ConferencecallGetByReferenceID: `!row.Next()` 분기 안에서 row.Err() 를 먼저 확인해 `%w` 로 반환하고, 없을 때만 ErrNotFound. 구조는 기존 :152, :102 와 같고 래핑 동사만 `%w`(errorlint 게이트가 새 줄의 `%v` 를 막고 기존 `%v` 줄은 변경하지 않으므로 같은 파일 안에서 동사가 섞이는 것은 의도). 기존 `%v` 줄은 건드리지 않음.
- 쓰기 뒤 `_ = *UpdateToCache` 는 에러를 무시하고 nil 을 반환하며 트랜잭션이 없으므로 커밋 뒤 구간 분리(WithoutCancel)는 하지 않음.

## 취소 부작용 판단
- 진입 ctx 는 listenhandler/main.go:185, subscribehandler/main.go:141, conference-control 모두 Background 이고 WithTimeout/WithCancel/WithDeadline 파생이 없어 현재 동작 변화는 거의 없음. 효과는 noctx 일관성과 향후 취소 도입 대비.
- ConferencecallGetByReferenceID 호출부(subscribehandler/callmanager.go:27, 56)는 err != nil 을 종류와 무관하게 삼켜 rows.Err 반환의 가시 효과 없음. ConferenceGetByConfbridgeID 호출부(conferencehandler/conference.go:227, CLI)에는 create-if-missing 이나 멱등 분기 없음.

## 범위 밖(보고서에 한 줄)
- callmanager.go:27, 56 에러 삼킴, 쓰기 뒤 UpdateToCache 실패 시 캐시 stale(TTL 24시간), 비멱등 json_array_append/json_remove 의 취소 시 모호성(Background 라 현재 발생 불가), 호출부(conferencehandler, conferencecallhandler, subscribehandler) 코드 변경, 다른 서비스, 마이그레이션.

## 검증(verify)
- 테스트 도구: github.com/DATA-DOG/go-sqlmock v1.5.2 신규 require (go.mod 에 없음. billing-manager 와 같은 버전). 기존 sqlite 인메모리는 RowError 주입이 안 되고 MySQL JSON 함수를 쓰는 4개 Exec 를 실행할 수 없음. go.mod/go.sum 은 이 의존성 추가분만 변경하고 vendor/ 는 커밋하지 않음.
- 신규 ctx_rows_test.go(기존 sqlite TestMain 과 별개로 테스트마다 sqlmock.New() 와 handler{db: ...} 를 직접 조립하고 전역 dbTest 는 건드리지 않음): 단건 getter 4곳과 목록 2곳(ConferenceList, ConferencecallList) RowError 주입(NewRows(cols).AddRow(...).RowError(0, e) 로 행 1개를 넣고 컬럼 수를 GetDBFields 와 맞춤) 시 errors.Is(err, ErrNotFound) 거짓. 원인 판정은 구분: 신규 2곳은 `%w` 이므로 errors.Is(err, rowErr), 기존 2곳(conferenceGetFromDB, conferencecallGetFromDB)과 목록 2곳은 기존 `%v` 래핑이므로 에러 문자열에 주입한 메시지 포함 여부로 판정(기존 줄은 변경하지 않음. 신규 줄은 메시지 형식은 기존과 같고 동사만 `%w`). 4개 Exec 함수는 ExpectExec 를 등록하고 취소된 ctx 에서 에러 문자열에 "context canceled" 포함 여부로 판정한다. 기존 4개 Exec 의 에러 반환은 `%v` 이고 변경 줄이 아니므로 `%w` 로 바꾸지 않아 errors.Is 는 쓰지 않음(billing 선례도 문자열 판정). ExpectationsWereMet 는 호출하지 않음(취소 경로에서 기대값이 소진되지 않음). ExpectQuery 는 정규식 "SELECT" 같은 짧은 패턴으로 매칭. 핸들러 조립에 utilHandler.TimeNow 기대값(Exec 함수)과, 캐시를 먼저 읽는 ConferencecallGetByReferenceID(conferencecall.go:162 cache.ConferencecallGetByReferenceID)와 ConferenceGet/ConferencecallGet 용 캐시 miss 기대값을 둠(ConferenceGetByConfbridgeID 는 캐시를 읽지 않아 불필요).
- 수정 전 RED 는 신규 getter 2곳(rows.Err)과 Exec 4건(취소 ctx). 기존 getter 2곳(:152, :102)과 목록 2곳은 수정 전에도 통과하는 회귀 보호용. 변이 시험: Exec 4건 각각 Context 복원, 신규 rows.Err 2곳 각각 제거에서 테스트 실패 확인(ConferencecallGetByReferenceID 는 캐시 miss 입력으로).
- 파생 ctx 없음 근거: grep WithTimeout|WithCancel|WithDeadline bin-conference-manager/pkg bin-conference-manager/cmd 결과 0건.
- 테스트 규약(check-test-conventions): 이름 Test_ 접두, gomock 컨트롤러 변수 mc, testify 미사용. 캐시 miss 는 mockCache.EXPECT().<메서드>(gomock.Any(), id).Return(nil, fmt.Errorf("not found")) 로 테스트에서 이 방식으로 구현함.
- noctx, rowserrcheck 임시 설정 0건, 표준 설정 0 issues, .golangci-errorlint.yml --new-from-merge-base=origin/main 통과, go mod tidy/vendor/generate/test -race/lint, check-structure-conventions, check-test-conventions.
