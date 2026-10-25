# bin-conference-manager unit test coverage to 75% (VOIP-1652)

## 1. 티켓 검토 (분석)

판정: 유효. origin/main accc03e00 에서 재측정했고 카드 수치와 일치합니다.

- 수기 합계 71.76% (1959/2730 statement), 원본 63.56%. 75.0% 는 2048, 76% 는 2075 이므로 각각 89개, 116개를 더 커버해야 합니다.
- 제외 방법: 경로에 `/gens/` 가 있거나 파일명이 `mock_` 로 시작하거나 파일 헤더(package 줄 이전)에 `Code generated` 와 `DO NOT EDIT` 가 있는 파일. 같은 블록의 중복은 최대 count 로 합칩니다.
- 패키지별 전(수기): cmd/conference-control 0.0 (510), cmd/conference-manager 0.0 (162), internal/config 97.8 (135), models/conference 97.1 (35), models/conferencecall 90.0 (20), pkg/cachehandler 100.0 (59), pkg/conferencecallhandler 98.5 (271), pkg/conferencehandler 98.1 (467), pkg/dbhandler 89.9 (425), pkg/listenhandler 93.7 (569), pkg/subscribehandler 98.7 (77).
- 구조 확인: cmd 를 제외한 미커버는 99 statement 뿐입니다(2058 중 1959 커버). cmd 를 제외하면 최대 75.38% 라서 75.0% 는 99 중 89 를 모두 채워야 하고 76% 는 불가능합니다. 그 99 는 대부분 도달 불가입니다: dbhandler 의 squirrel `ToSql`/`PrepareFields` 실패 분기(약 41), listenhandler 의 `json.Marshal` 실패 분기(약 32). 따라서 cmd/conference-control 의 일부를 테스트로 확보해야 합니다.
- cmd/conference-control (510 statement): 12개 `cmd*` 생성자(flag 등록, 약 69), `initCommand`, `resolveUUID`, `printJSON`, `runCreate` 의 입력 검증(핸들러 초기화 이전), 각 `run*` 의 `initHandler` 실패 전파는 제품 코드 변경 없이 테스트할 수 있습니다. `initHandler` 가 성공해야 도달하는 본문(DB, Redis, RabbitMQ 연결)은 단위 테스트 대상이 아닙니다.
- `initHandler` 실패는 형식이 잘못된 DSN 이 `sql.Open` 단계에서 로컬 파싱 오류로 끝나므로 네트워크 접속 없이 결정적으로 재현됩니다.
- cmd/conference-manager (162): `runMain` 이 외부 연결을 직접 맺는 구조이며 이번 범위에서 제외합니다. 산술 부족분이 생길 때만 `run`/`runSubscribe`/`runListen` 을 재검토합니다.
- 중복 여부: 겹치는 열린 PR 없음(VOIP-1647, 1648, 1651 모두 머지됨). VOIP-1416 이 다루는 패키지(dbhandler 등)와 겹침은 PR 본문에 명시만 합니다.
- 틀린 주장: 카드의 "수기 71.8%" 는 맞습니다(별도 worktree 가 오래된 base 여서 처음 69.4% 로 측정됐으나 origin/main 으로 갱신 후 일치).
