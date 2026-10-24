# bin-route-manager unit test coverage 60% (VOIP-1647)

## 1. 이슈 분석 (티켓 검토)

판정: 유효. 기준선 `origin/main d68e695b5`, 수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외, 같은 블록 중복은 최대 count) 합계를 직접 재측정했다.

- 수기 statement 2502개 중 1340개 커버, 53.6%. 원본(생성 코드와 mock 포함) 47.8%. 60%까지 161개, 61%까지 약 187개가 더 필요하다. 티켓 수치와 일치한다.
- 관련 PR/브랜치: `gh pr list --search VOIP-1647` 결과 없음. 이 경로의 최근 커밋(VOIP-1642 lint, VOIP-1637, VOIP-1601)은 테스트 범위에 영향을 주지 않는다.
- 제품 코드 변경은 하지 않는다. 테스트 불가 구조는 없고 모든 대상이 기존 mock/인터페이스 또는 sqlite 인메모리로 주입된다.

미커버 상위 파일(미커버 statement / 전체):

| 파일 | 미커버/전체 | 성격 |
|---|---|---|
| cmd/route-control/main.go | 315/315 | CLI, DB/Redis 연결 필요. 제외 |
| cmd/route-manager/main.go | 171/190 | 데몬 기동, 외부 의존. 제외 |
| pkg/listenhandler/v1_providercalls.go | 128/128 | 핸들러 4개 전부 무테스트 |
| pkg/dbhandler/providercall.go | 114/114 | 전부 무테스트, sqlite 스키마에 테이블 없음 |
| pkg/telnyxclient/telnyx.go | 62/186 | 오류 경로, parseRejectedCountries 0% |
| internal/config/config.go | 60/60 | Bootstrap, LoadGlobalConfig |
| pkg/cachehandler | 59/59 (handler.go 46, main.go 13) | 실제 Redis 클라이언트 필요. miniredis 가 이 모듈 go.mod 에 없어 의존성 추가가 필요하므로 후순위 |
| pkg/listenhandler/main.go, v1_routes.go, v1_providers.go | 37, 36, 28 | processRequest 분기, 입력 오류 경로 |
| pkg/dbhandler/provider.go, route.go | 31, 27 | UpdateHealthStatus, 오류 경로 |
| pkg/healthcheckhandler/health.go | 23/90 | Run 루프 |
| pkg/providercallhandler/providercall.go | 21/97 | Get/List/Delete 오류 경로 |
| models/providercall/webhook.go | 9/9 | ConvertWebhookMessage, CreateWebhookEvent |

후순위 대상(cmd/ 전체 statement 505개, cachehandler 59개, 합 564개)의 미커버 약 548개를 빼도 나머지 미커버가 약 600개이므로 필요량(약 187개)은 산술적으로 달성 가능하다. 재측정 시 cmd/route-manager 의 커버 수가 실행마다 3개 내외 달라질 수 있다.

반증하지 못한 주장: 기준선 수치, 60%까지의 필요량. 틀린 주장: 없음. 모호한 부분: "cmd/ 의 main" 은 테스트하기 어려우므로 다른 패키지로 채운다(카드 본문이 허용).
