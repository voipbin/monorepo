# VOIP-1492 이슈 확인/분석

작성: 2026-09-08. 모든 수치는 프로덕션 DB 직접 조회로 확인.

## 1. 이슈 유효성: 유효하나 **범위를 재정의해야 하고, 티켓에 없던 급한 항목이 하나 있다**

티켓 원문은 "만료 계정 소속 고아 자격증명/리소스를 정리한다"이다. 조사 결과:

- 삭제 대상 대부분은 위험하지 않다
- 티켓이 경고만 해두었던 "실사용 계정 혼입"은 **실재한다**
- 티켓에 없던 **billing top-up 잡이 만료 계정에 매달 토큰을 계속 지급하고 있다.** 이것이 가장 급하다

## 2. 실측 현황

| 항목 | 건수 |
|---|---|
| `status=expired` 고객 | 193 (참고: `active`는 132. **만료가 활성보다 많다**) |
| 살아있는 agent | 92 |
| 살아있고 미만료인 accesskey | 97 (전체 98, 1건 삭제됨) |
| 살아있는 번호 | 10 |
| 살아있는 flow | 2 |
| 기타 고아 | billing_accounts 94, storage_accounts 92, direct_directs 94, call_outbound_configs 63, registrar_customer_domains 2 |

**자가 복구 가능 여부로 193건이 갈린다.** `resendTarget`(customerhandler/signup.go:396-455)이
살아있는 agent가 없으면 `nil`을 반환하므로, VOIP-1490의 재발송으로 스스로 복구할 수 있는 것은
**agent가 살아있는 92건뿐**이다. **나머지 101건은 자가 복구 경로가 아예 없다.**
(재가입은 가능하다. `tm_delete`가 설정된 기존 행이라 고객 중복 검사에 걸리지 않고 agent도 없기 때문이다.)
이 101건은 되살아날 여지가 없다는 점에서 오히려 정리의 근거가 되는 쪽이다.
반대로 92건은 `validateCreate`의 agent 중복 검사(customerhandler/customer.go:92-95)에 걸려
재가입이 막힌다. 즉 **193건 각각은 복구 경로를 정확히 하나씩만 갖는다.** 0개도, 2개도 아니다.

## 3. 가장 급한 것: billing top-up이 만료 계정에 매달 지급되고 있다

`billing_billings`에 만료 이후 생성된 행이 **140건, 76개 계정**에 걸쳐 존재한다.
`reference_type='monthly_allowance'` + `transaction_type='top_up'` 형태의 단일 행 유형이다.

| 월 | 지급 건수 | 대상 계정 |
|---|---|---|
| 2026-05 | 12 | 12 |
| 2026-06 | 12 | 12 |
| 2026-07 | 14 | 14 |
| 2026-08 | 26 | 26 |
| 2026-09 | **76** | **76** |

**정적인 문제가 아니라 가속 중이다.** 만료 계정 수가 늘면서 매달 지급 대상도 늘어난다.
`billing_accounts` 중 **78건이 `tm_next_topup = 2026-10-01`** 이며 3주 뒤 다시 실행된다.

**원인**: `bin-billing-manager/pkg/accounthandler/topup.go`의 `TopUpDue`가
`account.FieldDeleted: false`만 필터하고 고객 상태를 보지 않는다. 그리고
`bin-customer-manager/models/customer/event.go:4-11`에 **`customer_expired` 이벤트가 아예 없어서**,
`customer_frozen`처럼 billing을 동결시킬 신호 자체가 존재하지 않는다.

**VOIP-1491로는 막히지 않는다.** 내부 cron이지 HTTP 경로가 아니다.

**이 항목을 먼저 처리해야 하는 이유는 심각도가 아니라 성격이다.** 개별 행은 `amount_credit=0`이라
현금이 움직이지 않는다. 그러나 (i) 지금 이 순간에도 진행 중이고, (ii) 만료 계정 수에 비례해
커지며, (iii) 3주 뒤 다시 실행되고, (iv) 수정이 비파괴적이고 작다.
방치하면 빌링 원장이 죽은 계정에 대한 지급 기록으로 계속 오염되고, 나중에 VOIP-1504로
보존 정책을 세울 때 정리 대상이 그만큼 늘어난다.

**주의: 이 잡을 막는다고 8절의 유료 경로 노출이 줄지는 않는다.** 8절 (a)의 `speakings`는
잔액이 0이어도 그대로 호출된다(사전 검사가 없고 credit이 음수를 허용한다). 오히려 잔여 토큰
7,800은 원장에 손실이 드러나는 시점을 **늦출 뿐**이지 상한이 아니다. 두 문제는 독립적이며
각각 따로 처리해야 한다.
따라서 이 항목만은 VOIP-1504(보존 정책)를 기다리면 안 된다. 별도로 즉시 처리해야 한다.
수정은 비파괴적이고 작다(`TopUpDue`에 고객 상태 조건 추가, 또는 `customer_expired` 이벤트 도입).

## 4. 실사용 고객 1건이 섞여 있다

`tgolebiowski@magicbeansoftware.com` (customer `6F3106C1B494424F85F03568AC2BA4B5`)

| 시각 | 사건 |
|---|---|
| 2026-06-09 16:04 | 가입 |
| 2026-06-09 17:07 | **1시간 뒤 자동 만료** |
| 2026-06-10 16:09 | 만료 하루 뒤 agent 갱신 |
| 2026-06-15 15:31 | **만료 6일 뒤 flow 생성** |
| 2026-06-15 15:37 | **번호 `+899392992871` 취득** |
| (만료 후 누적) | accesskey 2개, direct 1개 추가 생성 |

simocosta317과 동일 패턴이며 3개월 먼저 발생했고 **아무도 신고하지 않았다.**
기업 도메인이므로 평가 중이던 잠재 고객으로 보인다. 무조건 삭제했다면 이 계정을 파괴했을 것이다.

### 조사 방법 (전수 여부)

`customer_id` 컬럼을 가진 테이블 **66개를 전수 열거**해 그중 `tm_create`를 가진 65개를 `tm_create`와
`tm_update` 양쪽 기준으로 만료 시각과 대조했다. 만료 이후 **사용자 활동**은 전부 이 1건으로 수렴한다.
(만료 시각 프록시로 `c.tm_update`를 썼다. 193건 중 161건은 `tm_update`와 `tm_delete`가 정확히
같지는 않으나 **최대 차이가 마이크로초 단위**이므로 이 집단에 한해서는 안전한 프록시다.)

단, 위 3절의 billing top-up은 **시스템이 만든 행**이므로 이 집계에서 분리해야 한다.
"192건은 활동 흔적이 전혀 없다"가 아니라 **"192건은 사용자 활동 흔적이 없다"**가 정확한 서술이다.

## 5. 비용: 번호는 무료가 맞으나 근거가 틀렸다

**결론은 유효하다.** 만료 계정 소속 `number` / `number_renew` 빌링 행이 **0건**이다
(시스템 전체로는 각각 13건, 39건 존재).

**단 근거를 정정한다.** `provider_name`은 빌링이 참조하지 않는 컬럼이며 번호 웹훅 페이로드에도 없다
(`bin-number-manager/models/number/webhook.go:13-38`). 실제 과금 판별은
`type == TypeVirtual`이다(`bin-billing-manager/pkg/billinghandler/event.go:152-156, :173-177`).
해당 10건은 전부 `type='virtual'`이라 무과금이다.

번호는 구매 시 $5, 약 28일 갱신마다 $5가 과금된다(`cost_type.go:45`).
`type` 컬럼은 2026-02-10 이전 행에 대해 `'normal'`로 백필됐으므로
**`provider_name=''` + `type='normal'` 조합은 실제로 과금 대상이다.**
향후 정리 작업에서 `provider_name`을 판별 기준으로 쓰면 안 된다.

## 6. 통화: 0건이 아니다

`call_calls`는 0건이 맞다. 그러나 `call_groupcalls`에 **17건**이 4개 만료 계정에 걸쳐 존재하며
**전부 `status='progressing'`으로 종료되지 않았다.** 전부 만료 이전 생성이고 테스트 계정 소속이라
4절 결론은 바뀌지 않으나, 종료되지 않은 groupcall은 그 자체로 별도 위생 항목이다.

또한 `call_calls = 0`을 "시도가 없었다"는 근거로 쓸 수 없다. 계정 상태 거부는 call row 생성 이전에
발생하므로(VOIP-1496) 거부된 시도는 DB에 남지 않는다. 로그 기준으로는 보존 창(약 41시간) 내에
거부 시도를 한 만료 계정이 없었다(유일한 건은 이미 복구한 simocosta317).

## 7. 자격증명 노출: VOIP-1491로 다 닫히지 않는다

**accesskey 97개** — 최단 만료 2026-12-31, 최장 2027-09-06. **90일 내 자연 만료 0건**
(180일 내 2건). 아무것도 하지 않으면 4~12개월 더 살아있다.

**agent 92개** — `password_hash`는 가입 시 부여된 랜덤값이라 사용자가 모르지만
(`agenthandler/event.go:151`), agent row가 살아있으므로 `/auth/password-forgot` ->
`/auth/password-reset`으로 비밀번호를 새로 설정해 로그인할 수 있다.

**중요 정정: VOIP-1491은 API 표면을 닫을 뿐 자격증명 자체를 닫지 않는다.**

| 경로 | 상태 |
|---|---|
| `POST /auth/login` | 고객 상태 미확인 (`agenthandler/agent.go:254`). 만료 계정 agent도 유효 JWT 발급 |
| `/auth/password-forgot`, `/auth/password-reset` | 공개 그룹, 고객 상태·agent 상태·`email_verified` 전부 미확인 |
| `/auth/unregister` | `isBlockedAccountStatus`에서 **명시적 예외**(`bin-api-manager/lib/middleware/authenticate.go:238-241`) |
| 고객 조회 RPC 실패 시 | **fail open**으로 통과(`bin-api-manager/lib/middleware/authenticate.go:251-254`) |
| direct token | 검사 자체를 건너뜀(`bin-api-manager/lib/middleware/authenticate.go:227-230`) |
| `PermissionProjectSuperAdmin` 보유자 | 검사 면제. 단 가입 agent는 `PermissionCustomerAdmin`이므로(`agenthandler/event.go:167`) 이 집단에는 무해 |

## 8. 게이트가 없는 유료 경로 (잠재 위험, 현재 미실현)

만료 계정이 살아있는 accesskey로 지금 접근 가능하며 **고객 상태 검사가 어느 층에도 없는** 경로.
과금 여부로 둘로 나뉜다.

**(a) 원장에 기록됨 — 그러나 차단되지 않는다**
- `POST /v1.0/speakings` -> Google TTS.
  `ReferenceTypeSpeaking`이 존재하며(`bin-billing-manager/models/billing/billing.go:80`)
  `rate_token_per_unit=3`(`cost_type.go:53`)으로 `balance_token`을 차감한다.
  **그러나 이것은 사후 기록이지 상한이 아니다.** tts-manager가 `speaking_started`/`speaking_stopped`를
  발행하면 billing-manager가 구독해 `BillingStart`/`BillingEnd`를 호출한다
  (`subscribehandler/main.go:219-223`, `billinghandler/event_tts.go:16,:41`). 과금은 Google TTS를
  이미 호출한 **뒤에** 기록된다. `bin-api-manager/pkg/servicehandler/speaking.go`에도,
  `bin-tts-manager`에도 사전 잔액 검사가 없다(call, SMS, email, number/number_renew, recording은 전부
  `BillingV1AccountIsValidBalanceByCustomerID`로 사전 검사한다. **speaking만 하지 않는다.**)
  더 결정적으로, `ReferenceTypeSpeaking`은 `IsValidBalance`(`accounthandler/balance.go:78-136`)의
  case에 **아예 존재하지 않는다.** `default`로 떨어져 `UNSUPPORTED_BILLING_TYPE`을 반환하므로,
  이 경로는 현재 구현 상태로는 사전 검사를 붙일 수 없다(`ReferenceTypeSpeaking` case를 추가하면 가능해진다).
  토큰이 0이 되면 비용은 credit으로 넘어가는데, `CalculateTokenCreditDeduction`
  (`dbhandler/billing.go:372-384`)이 토큰 잔액을 초과한 단위마다 credit을 차감하고
  `:415-418`은 credit이 음수가 될 수 있음을 명시한다. 그 주석은 "사전 검사가 낙관적이라
  동시 호출에서 초과할 수 있다"는 제한적 서술이지만, speaking 경로에는 **사전 검사 자체가 없으므로**
  그 전제가 성립하지 않고 이탈 폭에 경계가 없다.

**(b) 아예 기록되지 않음**
- `bin-ai-manager` — `validate.go`가 없고 `CustomerV1CustomerGet`을 호출하지 않음.
  `POST /v1.0/aisummaries`, `/aiaudits`, `/aipromptproposals`가 **플랫폼 소유 키로** OpenAI/Gemini 호출
- `POST /v1.0/rags`, `/rags/{id}/sources` -> Vertex AI 임베딩. `bin-rag-manager`는 customer-manager
  의존이 전혀 없고, 5분 주기 티커로 pending ingestion을 인증 컨텍스트 없이 재구동

(b)에는 LLM/임베딩에 대한 빌링 `ReferenceType`이 아예 없다.

**정리하면 (a)와 (b)의 차이는 "안전한 쪽과 위험한 쪽"이 아니라 "원장에 남는 쪽과 안 남는 쪽"이다.
어느 쪽에도 지출을 막는 상한이 없다.** 잔액은 원장이지 한도가 아니다.

현재 잔액이 음수인 계정은 없다(5,002건 중 최솟값 0). 단 credit이 음수가 되도록 설계된 이상
이 수치는 **한도가 작동한다는 증거가 아니라 아직 아무도 쓰지 않았다는 증거**로만 읽어야 한다.

**다만 실측상 193개 계정 전체에서 `ai_*`, `transcribe_*`, `tts_manager_speaking`, `storage_files`
행이 하나도 없다.** 능력은 열려 있으나 행사된 적이 없다. 실현된 손실이 아니라 잠재 위험이다.

따라서 "VOIP-1491 이후엔 보안 조치가 아니라 정리 작업"이라는 서술은 **성립하지 않는다.**
다만 실현된 손실이 아니라 잠재 위험이라는 점은 유지된다.

## 9. 진행 타당성 판단

진행해야 하나 **티켓 범위를 재정의**한다. 원문대로 실행하면 실사용 고객 1건을 파괴하고,
비용도 안 드는 테스트 번호를 지우느라 프로덕션 데이터를 대량으로 건드린다. 위험 대비 이득이 맞지 않는다.

권장 순서:

**순서는 심각도 순이 아니라 (확실성 x 즉시성) 순이다.** 1번은 심각도는 낮으나 확실하고 지금
진행 중이며 수정이 싸다. 3번은 심각도가 높으나 아직 실현되지 않았고 판단이 필요하다. 혼동하지 말 것.

1. **billing top-up 차단 (최우선, VOIP-1504 대기 금지)**
   유일하게 **지금도 계속 진행 중이고 커지고 있는** 문제다. 3주 뒤 78건 재실행 예정.
   방치하면 빌링 원장이 죽은 계정에 대한 지급 기록으로 계속 오염되고, VOIP-1504로 보존 정책을
   세울 때 정리 대상이 그만큼 늘어난다. 비파괴적이고 작은 수정이다. 별도 티켓으로 분리해 즉시 처리한다.
   (3절 참고: 이 잡을 막는다고 8절의 유료 경로 노출이 줄지는 않는다. 두 문제는 독립적이다.)

2. **tgolebiowski 계정 복구**
   실사용 고객이 3개월째 갇혀 있다. VOIP-1490 배포 이후라 재발송 자가 복구도 가능하나
   (agent가 살아있으므로 대상에 해당), 3개월 방치는 우리 책임이므로 능동 조치가 맞다.

3. **accesskey 97개 처리 방식 결정**
   - (a) VOIP-1491 우선 배포 — 데이터 무변경, 되돌리기 쉬움. **단 7절대로 자격증명 자체는 남는다**
   - (b) accesskey 즉시 폐기 — 즉효이나 되돌리기 어렵고, 복구된 고객은 재발급 필요
   VOIP-1491을 곧 낼 수 있다면 (a)로 시작하되, **login/password-forgot 경로와 fail-open까지
   VOIP-1491 범위에 포함**시켜야 실제로 닫힌다.

   **이 항목의 무게를 낮게 보지 말 것.** 8절에서 열거한 경로 어디에도 지출을 막는 상한이 없다
   (call/SMS/email/number/recording은 사전 검사가 있으므로 여기 해당하지 않는다).
   speakings는 원장에 남지만 차단되지 않고, ai/rag 경로는 기록조차 되지 않는다.
   실현된 손실이 없는 것은 통제가 작동해서가 아니라 아직 아무도 시도하지 않았기 때문이다.
   97개 키가 2027년까지 유효하다는 점을 함께 놓고 판단해야 한다.

4. **고아 데이터 정리는 마지막**
   VOIP-1504(보존 정책)에서 기간을 정한 뒤 그 정책으로 일괄 처리한다.
   지금 별도 절차로 지우면 나중에 정책과 충돌한다.
   정리 대상 인벤토리에 `call_groupcalls` 17건(미종료)과 2절의 기타 고아를 포함시킨다.

## 10. 미검증 사항

- 로그 보존 창이 약 41시간이라 그 이전의 **조회성** 사용(리소스를 만들지 않은 접근)은 확인 불가.
  DB 기반 리소스 생성은 전 기간 조사했다.
- api-manager 접근 로그로 만료 계정 accesskey의 실사용 여부를 전 기간 확인하지 못했다.
- tgolebiowski가 지금도 복구를 원하는지 알 수 없다. 3개월 경과했으므로 이미 이탈했을 수 있다.
- `registrar_extensions` 9건이 2026-08-23 16:00:49에 일괄 soft-delete 됐다. 의도된 작업인지 미확인.
- 만료 고객 2건이 `registrar_customer_domains.domain_label`(UNIQUE, 만료 없음)을 점유 중.
  4자 SIP 네임스페이스는 희소 자원이므로 정리 대상에 포함 검토.
