# bin-flow-manager 단위 테스트 커버리지 85% 상향 (VOIP-1659)

## 1. 티켓 검토 (이슈 분석)

- 판정: 유효. origin/main a47f5a2a 에서 수기 코드(mock_*.go 와 `Code generated ... DO NOT EDIT` 헤더 파일 제외, 같은 블록은 최대 count) 3567/4252 = 83.89%. 이 서비스에는 gens/ 가 없다. 카드의 기준선(83.90%)과 0.01%p 차이이며 반올림 차이다.
- 85.0% 까지 47 statement(covered 3614 이상), 권장 85.5% 까지 69 statement(covered 3636 이상)가 필요하다.
- 미커버 685 statement 의 분포: pkg/activeflowhandler 173, cmd/flow-control 141, cmd/flow-manager 133, pkg/listenhandler 63, pkg/dbhandler 61, pkg/flowhandler 51, pkg/stackmaphandler 25, pkg/subscribehandler 16, internal/config 8, 그 외 소수.
- 직전 작업(VOIP-1651, 1652, 1657)이 cmd/flow-control 의 헬퍼와 생성자, activeflowhandler 의 db.go 오류 전파를 이미 다뤘다. 남은 미커버 중 도달 가능한 것은 하위 의존성(RPC, stackmap, 핸들러) 오류의 전파 분기와 stackmaphandler 의 경계 입력이다.
- 도달 가능 상한은 필요량(69)을 크게 넘는다. 확인한 도달 가능 후보: activeflowhandler/actionhandle.go 의 하위 RPC, stackmap 오류 반환 약 60 블록, flowhandler/db.go 의 mock 오류 전파 약 30 statement, stackmaphandler/action.go 의 경계 입력 약 19 statement. 도달 불가로 분류해 제외하는 것: cmd/flow-manager 의 main 과 runDaemon, initFlowHandler 와 initCache, listenhandler 의 json.Marshal 실패, dbhandler 의 ToSql 과 Scan 계열. 따라서 차단 없이 진행한다.
- VOIP-1416 과 겹치는 패키지: 80% 미만인 pkg/subscribehandler(72.9%)와 pkg/flowhandler(79.6%), cmd 두 패키지가 해당할 수 있으며 흡수하지 않고 PR 본문에 명시만 한다.
- 열린 PR 중 bin-flow-manager 를 건드리는 것은 없고, 최근 머지된 VOIP-1651, 1652, 1657 은 위 범위와 겹치지 않는 구간을 다뤘다.
