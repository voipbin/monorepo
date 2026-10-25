# VOIP-1652 bin-hook-manager 단위 테스트 커버리지 75% 계획

## 1. 티켓 검토 (유효)

- 기준선(origin/main accc03e, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`): 수기 합계 422/577 = 73.14%, 원본 74.42%. 목표는 수기 합계 75.0% 이상(433개 이상 커버, 약 11개 추가, 권장 76% 는 439개 이상).
- 미커버 중 도달 가능분: servicehandler 7 statement(본문 읽기 오류 4, 서명 헤더 필드 누락, 본문 크기 초과, 대화 GET 전달 오류), `writeBase64` 쓰기 오류 2, `initProm` 리스너 실패 재시도 3(리스너가 오류 없이 끝나는 분기는 `ListenAndServe` 가 항상 오류를 반환하므로 도달 불가), `runService` 의 DB 연결 실패 반환 지점까지 21(설정 값에 닫힌 루프백 포트를 지정하면 `commondatabasehandler.Connect` 가 재시도 없이 오류를 반환하고 소켓 연결 전에 종료됨). 개인키 기록 실패 반환 2를 더하면 합계 약 +35, 예상 약 457/577 = 79%(임시 테스트로 `runService` 만 추가해 443/577 = 76.8% 를 확인).
- 제외(비용 대비 제외 포함): 두 `main`, `runService` 의 DB 연결 이후(소켓 연결은 공용 rabbitmqhandler 의 무한 재시도), `initHandler` 와 `runSend*` 성공 경로(같은 이유), `Bootstrap` 과 `PersistentPreRunE` 의 바인딩 오류 분기(직전에 플래그를 정의하므로 도달 불가), `writeBase64` 의 디코드 치명 오류 경로(프로세스 종료).
- 제품 코드 변경 없음. 루프백 닫힌 포트 연결 거부는 기존 `Test_initProm` 의 루프백 사용과 같은 수준이며 외부망에 의존하지 않는다.

## 2. 테스트 계획 (새 함수는 `Test_` 접두사, 테스트 파일만 추가, 합성값만 사용)

| 대상 | 케이스 |
|---|---|
| `Email`, `Message`, `Conversation`, `Billing` 본문 읽기 오류 | `iotest.ErrReader` 와 일부를 읽은 뒤 실패하는 reader 로 호출. 래핑 메시지 단언, mock 에 EXPECT 없음(하위 호출 없음). `Billing` 은 `errors.As` 로 `ValidationError` 확인. `Conversation` 은 GET 과 POST 모두 시험(읽기 오류가 GET 분기 앞에서 막힘) |
| `Billing` 본문 크기 | secret 설정 후 정확히 1MB 는 크기 검사를 통과해 전체 문자열 `missing Paddle-Signature header` 로 거부, 1MB+1 은 `exceeds maximum size` 로 거부(메시지와 `ValidationError` 단언) |
| `verifyPaddleSignature` 누락 | 세그먼트 2개 이상에서 `ts=<유효>;h1=`, `ts=;h1=<서명>`, `x=1;y=2` 각각 `missing ts or h1` 메시지를 단언(기존 케이스는 세그먼트 1개라 이 분기를 타지 않고 `err != nil` 만 보므로 구분 필요) |
| `Conversation` GET | `ConversationV1HookGet` 오류가 래핑되어 전달되고 빈 challenge 반환 |
| `writeBase64` 쓰기 오류 | `/dev/full` 에 비어 있지 않은 유효 base64 를 기록, 오류 반환(`/dev/full` 이 없으면 skip) |
| `runService` DB 연결 실패 | 포트는 `net.Listen("127.0.0.1:0")` 로 받아 즉시 닫은 값, DSN 은 자격 증명 없는 `tcp(<addr>)/test` 형태. 전역 설정 객체 필드를 바꾸고 `t.Cleanup` 으로 원복(`*c = orig`), 메트릭 리슨 주소는 `127.0.0.1:0` 으로 고정. 인증서/개인키는 중립 문자열의 유효 base64(PEM 형태 리터럴 금지), 비밀번호 리터럴 없음. 호출마다 고유 메트릭 엔드포인트(패키지 전역 atomic 카운터). `/tmp` 상수 경로(`/tmp/ssl_cert.pem`, `/tmp/ssl_privkey.pem`)는 시험 전 존재 여부를 확인해 하나라도 이미 있으면 건드리지 않고 skip, 없었으면 `t.Cleanup` 으로 두 경로 모두 제거(권한 오류도 skip). 테스트는 직렬 실행(`t.Parallel` 금지)이며 호출 직전에 `ReplaceHooks` 로 새 훅을 만들고 `t.Cleanup` 으로 복원(기존 `Test_writeBase64_logs` 패턴), `Data["func"] == "runService"` 인 엔트리만 판정. 판정 기준: `Could not access to database` 있음, `Could not write the ssl` 없음, 두 파일이 기록됨(기록 직후 즉시 읽어 단언, 불일치는 `t.Errorf`). 반환은 별도 고루틴과 timeout 으로 감시하고 실패 시 `t.Fatalf`(실패 시 고루틴이 소켓 연결 재시도로 남는 점은 허용). 닫은 포트를 다른 테스트가 재사용할 낮은 확률의 경합은 수용 |
| `runService` 기록 실패 | 같은 전제에서 개인키 경로를 디렉터리로 선점해 개인키 기록 실패 반환을 시험: `Could not write the ssl private key` 로그 있음, `Could not access to database` 없음, 인증서 파일은 기록됨(순서 변경 변이 포착). 정리 시 디렉터리 제거, 선점이 불가하면 skip |
| `initProm` 재시도 | 점유 중인 루프백 포트로 호출, 훅을 해당 `listen`/`endpoint` 필드로 필터해 리스너 실패 로그를 deadline polling 으로 확인. 점유 해제 후 `GET http://<addr><고유 endpoint>` 200 확인. 리스너는 종료되지 않고 남으며, 포트 해제 후 재바인드 사이의 경합은 낮은 확률로 존재함을 수용 |

## 3. verify 항목

- 수기 합계 75.0% 이상, 목표 76% 이상(gens, mock_, DO NOT EDIT 제외, 동일 블록 최대 count 병합). 패키지별 전후 표와 수기, 원본 합계를 PR 본문에 기록.
- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`, `go test -race -count=2 -shuffle=on ./...`, `bash scripts/check-test-conventions.sh`, go.mod/go.sum 무변경, 테스트 후 `/tmp/ssl_*.pem` 이 테스트 전 상태와 같음.
- 변이 검사(scratch 복사본): 본문 읽기 오류 래핑 제거(4곳 전수), `Conversation` 읽기를 GET 분기 뒤로 이동, GET 오류 시 challenge 동반 반환, `ValidationError` 래핑 제거, 크기 비교 `>` 를 `>=` 로, `maxWebhookBodySize+1` 을 `maxWebhookBodySize` 로, 누락 검사의 `||` 를 `&&` 로, GET 오류 반환 제거, `writeBase64` 의 Write 오류를 `return nil` 로 변경, `runService` 의 DB 오류 시 `return` 제거, 인증서/개인키 기록 순서 변경, 개인키 기록 실패 반환 제거 시 새 테스트가 실패해야 함.
- `git diff --stat origin/main...HEAD` 가 bin-hook-manager/ 안의 `_test.go` 와 이 문서만 포함.
