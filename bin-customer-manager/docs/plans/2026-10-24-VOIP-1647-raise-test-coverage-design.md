# VOIP-1647: bin-customer-manager unit test coverage

## 목표와 범위
수기 코드(gens/, mock_*.go, `Code generated ... DO NOT EDIT` 제외) 커버리지를 59.1%에서 61% 이상으로 올린다. 테스트 파일만 추가하며 제품 코드, 시그니처, 노출 범위는 바꾸지 않는다. cmd/ 의 main 693 statements 는 프로세스 기동 코드라 제외하고 다른 패키지로 채운다. 값은 합성값만, 외부 호출과 네트워크 의존 없음.

## 기준선 (수기, base d68e695b5)
합계 59.06% (1828/3095). 60% 까지 29개, 61% 까지 60개 부족.

## 추가할 테스트와 verify 항목
1. listenhandler/v1_customers_freeze_test.go (미커버 84): processRequest 로 `/freeze`, `/freeze_and_delete`, `/recover` 라우팅. 각 성공(200, 본문의 customer id 확인), 핸들러 오류(400) 를 검증. 짧은 URI 400 은 정규식으로 도달 불가이므로 핸들러 직접 호출로 검증. Marshal 실패 500 분기는 도달 불가라 제외.
2. customerhandler/identity_verification_test.go (미커버 36): 잘못된 status 는 DB 호출 없이 오류, CustomerGet 실패(h.Get 이 db.CustomerGet 호출) 오류 전달, 동일 status 는 Update 와 Publish 없이 기존 customer 반환, CustomerUpdate 실패, 갱신 후 재조회(CustomerGet 두 번째 호출) 실패, 성공 시 올바른 필드 맵으로 Update 하고 `EventTypeCustomerIdentityVerificationUpdated` 발행.
3. dbhandler/customer_test.go 추가 (sqlite 하네스, 미커버 약 70): CustomerFreeze 는 active 만 frozen 으로 바꾸고 tm_deletion_scheduled 를 설정(호출 후 같은 패키지의 customerGetFromDB 로 DB 값을 직접 단언하며, UPDATE 뒤 customerUpdateToCache 가 부르는 TimeNow 와 mockCache.CustomerSet 기대를 둔다), 이미 frozen 이거나 없는 id 는 ErrNotFound 이며 행 불변. CustomerRecover 는 frozen 만 active 로 복구하고 tm_deletion_scheduled 를 비우며 active 는 ErrNotFound. 같은 customer 에 Freeze 를 여러 고루틴으로 호출하면 정확히 1건 성공, 나머지 ErrNotFound (하네스가 단일 연결이라 직렬화되므로 가드된 UPDATE 의 단일 성공을 검증하는 것이며 진짜 병렬성 검증이 아님을 주석에 밝힘). CustomerListFrozenExpired 는 시각을 모두 UTC 로 고정하고 경계값(예정시각 이전, 같음, 이후), 비 frozen, tm_delete 설정 행 제외를 검증. sqlite 인메모리 DB 는 테스트 간 공유되므로 단언은 결과 전체 동치가 아니라 자기가 만든 id 의 포함 여부로 하고, '같음' 경계가 제외되는지 구현 단계에서 실제 통과를 확인한다.
4. models/customer 에 IsValid 표 테스트(유효 4종, 빈 문자열, 미정의 값) 추가. identityverificationhandler 에 noop provider 테스트(CreateSession 의 ID 접두와 customer id, GetResult 와 HandleWebhook 의 verified 상태와 sessionID 전달).
5. cachehandler/handler_test.go 추가 (miniredis, 기존 의존): Customer/Accesskey Set 후 Get 왕복 및 TTL 설정(24h), 미존재 키 오류, 손상된 JSON 오류, EmailVerifyToken Set/Get/Delete 와 만료(miniredis FastForward), 미존재 토큰 오류, 비 UUID 값 오류, VerifyLock 획득 중복 거부와 해제 후 재획득, NewHandler 필드와 Connect 성공 및 서버 종료 후 실패.
6. 부족하면 여유분: dbhandler/accesskey.go, listenhandler v1_customers.go, v1_accesskeys.go, main.go 의 오류 경로, customerhandler.Delete/db.go.

## 검증 계획
- 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile` 후 수기 합계(계산 스크립트는 PR 본문에 명령으로 기재해 분모 3095 를 재현 가능하게 함)를 gens/ mock_*.go DO NOT EDIT 제외, 같은 블록은 최대 count 로 합쳐 계산. 목표 61% 이상. 패키지별 전후 표를 PR 본문에 첨부.
- 변이 검사 표본: Freeze 의 status 가드 제거, ListFrozenExpired 의 비교 연산자 변경, 동일 status 조기 반환 제거, 핸들러의 400 반환 변경, cachehandler TTL 값 변경, IsValid 의 목록 항목 제거를 각각 scratch 복사본에서 적용해 새 테스트가 실패하는지 확인.
- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, `scripts/check-test-conventions.sh`(Test_ 이름, gomock 컨트롤러명 mc, Rule 2: reflect.DeepEqual 과 t.Errorf 사용, testify 미사용), `git diff --stat origin/main...HEAD` 가 bin-customer-manager/ 안에만 있음을 확인.
- 테스트 중 발견한 결함은 고치지 않고 완료 요약에만 기록.

## 구현 시 주의 (설계 2회차 반영)
- dbhandler 테스트는 CustomerCreate 의 TimeNow 와 CustomerSet 기대도 명시하고, 가드 불일치(ErrNotFound) 케이스에서는 캐시 미호출을 별도로 단언한다.
- ListFrozenExpired 와 캐시 왕복 테스트의 시각은 UTC, 소수점 이하 없는 정초 값 또는 nil 로 둔다.
