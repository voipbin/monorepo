# bin-conference-manager unit test coverage to 75% (VOIP-1652)

## 1. 티켓 검토 (분석)

판정: 유효. origin/main accc03e00 에서 재측정했고 카드 수치와 일치합니다.

- 수기 합계 71.76% (1959/2730 statement), 원본 63.56%. 75.0% 는 2048, 76% 는 2075 이므로 각각 89개, 116개를 더 커버해야 합니다.
- 제외 방법: 경로에 `/gens/` 가 있거나 파일명이 `mock_` 로 시작하거나 파일 헤더(package 줄 이전)에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일. 같은 블록의 중복은 최대 count 로 합칩니다.
- 패키지별 전(수기): cmd/conference-control 0.0 (510), cmd/conference-manager 0.0 (162), internal/config 97.8 (135), models/conference 97.1 (35), models/conferencecall 90.0 (20), pkg/cachehandler 100.0 (59), pkg/conferencecallhandler 98.5 (271), pkg/conferencehandler 98.1 (467), pkg/dbhandler 89.9 (425), pkg/listenhandler 93.7 (569), pkg/subscribehandler 98.7 (77).
- 구조 확인: cmd 를 제외한 statement 는 2058개이고 그중 1959 가 커버되어 미커버는 99개뿐입니다. cmd 672개를 전부 미커버로 두면 분모 2730 기준 최대 2058/2730 = 75.38% 라서 75.0% 는 99 중 89 를 채워야 하고 76% 는 불가능합니다. 99 중 도달 가능분은 리뷰어 실측으로 약 75개입니다: dbhandler 43 중 인자가 고정된 `ToSql`/`PrepareFields` 약 13개만 도달 불가이고 나머지는 sqlmock 과 cachehandler mock, `Data` 필드에 직렬화 불가 값(chan)을 넣어 `PrepareFields` 를 실패시키는 방식, `NotEq{Value: <Value() 가 오류인 값>}` 로 `ToSql` 을 실패시키는 방식으로 도달합니다. listenhandler 36 중 `json.Marshal` 실패 34개는 `Conference.Data`(map) 에 chan 을 넣으면 conferences 쪽 약 24개가 도달하고, conferencecalls 6개와 count 2개 정도만 도달 불가입니다. 도달 가능분만으로는 약 75/89 라 부족하므로 cmd/conference-control 의 일부가 필요합니다.
- cmd/conference-control (510 statement): 12개 `cmd*` 생성자(flag 등록, 약 69), `initCommand`, `resolveUUID`, `printJSON`(chan 입력으로 오류 경로 도달), `runCreate` 의 입력 검증(핸들러 초기화 이전), 각 `run*` 의 `initHandler` 실패 전파는 제품 코드 변경 없이 테스트할 수 있습니다. 리뷰어 실측으로 실패 전파만으로 이 패키지가 약 56% 가 됩니다. `initHandler` 가 성공해야 도달하는 본문(DB, Redis, RabbitMQ 연결)은 단위 테스트 대상이 아닙니다.
- 함정: `config.Get()` 은 `LoadGlobalConfig()` 이전에 호출하면 panic 이고 `cfg` 는 비공개이며 `sync.Once` 라 프로세스당 한 번만 값이 고정됩니다. 따라서 `TestMain` 에서 `viper.Set("database_dsn", <형식이 잘못된 DSN>)` 후 `config.LoadGlobalConfig()` 를 한 번 호출합니다. viper 는 전역이므로 각 테스트가 설정한 키를 정리합니다. `runGet` 등은 `initHandler` 가 `resolveUUID` 보다 먼저라 입력 검증이 `initHandler` 실패에 가려지고, 입력 검증을 칠 수 있는 것은 `runCreate` 뿐입니다.
- 네트워크 금지: 형식이 잘못된 DSN(슬래시 없음 등)은 mysql 드라이버가 `sql.Open` 에서 파싱 오류로 끝내 접속을 시도하지 않습니다. 빈 DSN 과 기본 DSN 은 로컬 포트로 Ping 하므로 사용하지 않습니다.
- cmd/conference-manager (162): `runMain` 이 외부 연결을 직접 맺는 구조이며 이번 범위에서 제외합니다. 산술 부족분이 생길 때만 `run`/`runSubscribe`/`runListen` 을 재검토합니다.
- 중복 여부: 겹치는 열린 PR 없음(VOIP-1647, 1648, 1651 모두 머지됨). VOIP-1416 이 다루는 패키지(dbhandler 등)와 겹침은 PR 본문에 명시만 합니다.
- 틀린 주장: 없음. 카드의 수치는 모두 재현됩니다(작업 worktree 가 오래된 base 여서 처음 69.4% 로 측정됐으나 origin/main 으로 갱신 후 일치).
