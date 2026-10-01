# Assistant Builder 구현 확인 결과 (T0, VOIP-1558)

설계 7절 2단계의 코드 확인 결과다. 기준 커밋은 main `4a14812da`이다. 저장소 밖 인프라(LB, ingress, Loki 수집 소스)는 확인할 수 없어 "미확인, 활성화 전 선결"로 둔다.

| 항목 | 결과 | 근거 | 조치 |
|---|---|---|---|
| `ai_ais.init_prompt` 컬럼 | `text`이다. 한도는 글자 수가 아니라 약 65535바이트이며, 8000 rune의 한국어는 약 24000바이트라 들어간다. | `bin-dbscheme-manager/.../c74277ac6ab8_ai_ais_update_table.py:85`(`init_prompt text`), 원 컬럼은 `08fa3628f652_add_column_chatbot.py:25`(`text`) | T1의 상한 8000자를 유지한다. |
| 트레이스 사용 | `go.opentelemetry.io`는 api-manager `go.mod`에 `// indirect`로만 있고, ai-manager는 `auto/sdk`만 indirect이다. `cmd`, `lib`, `server`, `pkg` 코드에서 otel과 sentry를 쓰는 곳이 없다. | `go.mod` grep, `*.go` grep 0건 | 스팬에 본문이 실릴 경로가 없다. 추가 조치 없음. |
| api-manager 본문 로깅 | gin 접근 로거(`gin.LoggerWithConfig`)는 요청 본문을 기록하지 않는다(`cmd/api-manager/main.go:224-250`의 주석이 "JSON body is never logged by gin's access logger"를 전제로 한다). `lib/middleware`에 본문을 읽는 로깅은 없고 `ShouldBindBodyWith`, `GetRawData`, `httputil.Dump` 사용처도 없다. | grep | 서버 핸들러(T10)가 본문과 `BindJSON` 실패 문자열을 로그에 싣지 않으면 된다. |
| `CustomerRateLimit` tier | agent와 accesskey는 같은 tier `v1_customer`이며 기본 16.7 rps, burst 33이다(고객 단위 전 라우트 공유 버킷). Builder 전용 tier가 없다. | `internal/config/main.go:146-147`, `lib/middleware/customer_ratelimit.go:28` | **부족하다.** 분당 약 1000회까지 통과하므로 LLM 비용 방어는 일일 200회 카운터가 유일한 상한이다. 활성화 전 필수 조건의 추가분 2로 올린다(분당 폭주 방어가 필요하면 Builder 전용 tier를 설계한다). |
| 알림 규칙의 `consume.go` 에러 문자열 매칭 | 저장소 안 yml, yaml, json에서 `could not parse the message`, `could not reply the message`, `could not marshal the response` 매칭 0건 | grep | 저장소 밖 인프라는 미확인. |
| LB와 ingress timeout 65초 이상 | 미확인, 활성화 전 선결 | 저장소 밖 | 추가분 2. |
| Loki 수집 소스 | 미확인, 활성화 전 선결 | 저장소 밖 | 추가분 2. |
