# voip-kamailio-proxy 단위 테스트 커버리지 추가 설계 (VOIP-1657)

기준(origin/main f47ad27b)의 수기 코드 커버리지는 76.97% (254/330 statement) 이고 이 서비스에는 생성 코드와 mock 이 없어 원본과 같다. 제품 코드는 바꾸지 않고 `_test.go` 파일만 추가하며 go.mod/go.sum 은 바꾸지 않는다. 목표는 80.5% 이상(최소 80.0%)이다.

## 추가할 테스트

1. `pkg/siphandler` 의 기존 health 테스트 파일에 UDP dial 오류 분기 테스트를 추가한다. `SendOptionsCheck` 에 scope 가 존재하지 않는 인터페이스를 가리키는 IPv6 링크로컬 호스트명 `[fe80::1%nonexistent0]` 를 준다(함수가 `:5060` 을 붙이므로 대괄호가 필요). 주소 해석은 통과하고 dial 이 실패해 `Healthy=false`, `ResponseCode="timeout"`, 오류 nil 이어야 한다. 분기 통과는 시간이 아니라 logrus test hook 의 디버그 로그 `Could not dial UDP`(`hostname` 필드 포함)로 단언한다. 전역 logger 는 레벨 Debug 와 `ReplaceHooks` 후 `test.NewGlobal()` 을 쓰고 t.Cleanup 에서 복원하며 t.Parallel 을 쓰지 않는다. 에러 문자열은 단언하지 않고, dial 이 성공하는 환경이면 테스트가 실패한다(skip 하지 않음). timeout 은 100ms.
2. `cmd/kamailio-proxy` 에 `fakebroker_test.go` 와 `main_run_test.go` 를 추가한다. loopback 의 최소 AMQP 0-9-1 가짜 브로커(`127.0.0.1:0`, 외부 네트워크 없음, 자격 증명은 `test:test` 같은 가짜 값)를 만들고 `rabbitMQAddress` 를 그 주소로 바꿔 `main` 을 goroutine 으로 실행한다. 브로커가 처리할 메서드는 실제 호출 경로(`QueueCreate` 의 declare 와 `queueConfig`, `ConsumeRPC`)에서 확인한 다음으로 확정한다: Connection Start/StartOk, Tune/TuneOk, Open/OpenOk, Close/CloseOk; Channel Open/OpenOk, Close/CloseOk; Exchange Declare/DeclareOk; Queue Declare/DeclareOk, Bind/BindOk; Basic Qos/QosOk, Consume/ConsumeOk. 선례(bin-trigger-sender 의 fakebroker_test.go)는 Exchange Declare, Queue Bind, Basic Qos 응답이 없어 그대로 쓰지 않고 이 서비스에 필요한 최소분만 새로 쓴다.
   - 브로커 요구사항: heartbeat 프레임(type 8)은 무시하고, Exchange.Declare 와 Queue.Declare 의 field-table 은 `S`(문자열)와 `I`(int32) 타입을 디코딩하고(Exchange 는 durable 플래그도 단언), Queue.Declare 플래그는 비트(passive=0x01, durable=0x02, exclusive=0x04, auto-delete=0x08, no-wait=0x10)별로 디코딩해 각각 단언한다(passive=0, no-wait=0 포함). 채널 번호는 닫히면 재사용되므로 채널 상태(마지막 Qos 값)를 Channel.Open 에서 초기화하고 Consume 프레임의 큐 이름과 같은 채널의 가장 최근 Qos 로 묶어 큐별 prefetch=10 을 단언하며(Exchange.Declare 플래그 비트는 passive=1, durable=2, auto-delete=4, internal=8, no-wait=16 으로 Queue.Declare 와 다르다), Tune 의 heartbeat 는 0 으로 협상하고, Exchange 이름 `bin-manager.delay` 종류 `x-delayed-message`, Bind 의 routing key 가 큐 이름임도 단언한다. 기존 `isolateLogger`, `waitLogEntry`(init_test.go)를 재사용한다.
   - 재연결 고루틴 대비: `main` 은 `Close` 하지 않아 클라이언트의 재연결 고루틴이 프로세스 끝까지 남는다. 그래서 Cleanup 에서 브로커 리스너를 닫지 않고, 먼저 거부 상태(새 연결을 accept 즉시 끊음)로 전환한 다음 기존 연결을 모두 끊고, 리스너는 프로세스 종료까지 열어 둔다(포트가 재사용되지 않아 다음 실행의 브로커에 옛 클라이언트가 붙지 않음).
   - 브로커 고루틴 안에서는 `t.Errorf` 를 쓰지 않고 오류를 mutex 로 보호하는 필드에 모아 테스트 본문에서 단언한다. `main` 이 `Close` 하지 않아 클라이언트의 재연결 고루틴이 남을 수 있으므로 단언은 개수가 아니라 이름과 값의 포함 여부로 하고 "오류 로그 0건" 같은 단언은 쓰지 않는다.
   - 단언(정확한 값): 영구 큐(테스트용 고유 이름) declare 는 durable=1, auto-delete=0, exclusive=0 이고 x-expires 가 없다. 휘발 큐 `voip.kamailio.<mac>.request` declare 는 durable=0, auto-delete=1, exclusive=0 이고 `x-expires` 가 1800000(int32) 이다. Exchange Declare 1회 이상과 두 큐 각각의 Queue Bind 를 수신한다. 두 큐 각각에 consumer tag `kamailio-proxy` 의 Basic.Consume 이 있고 Basic.Qos prefetch 가 10 이다.
   - MAC 이 있는 인터페이스는 `net.Interfaces()` 에서 `HardwareAddr` 가 비어 있지 않은 첫 항목으로 고르며 없으면 skip 한다(환경 전제를 만들 수 없는 경우뿐).
   - 종료: 두 큐의 consume 기록을 상한 있는 폴링(5초)으로 기다린 뒤 `chSigs` 에 신호를 넣고 main 반환을 기다리며, `Terminating kamailio-proxy` 로그에 신호 값 문자열(`interrupt`)이 남았는지 부분 문자열로 단언한다. 폴링이 5초 안에 끝나지 않아도 먼저 `chSigs` 에 신호를 넣어 main 반환을 확인한 뒤에 Fatal 하며, 그래도 반환하지 않을 때만 전역을 건드리지 않고 Fatal 한다. 반환한 뒤에만 `rabbitMQAddress`, `interfaceName`, `rabbitMQQueueListen` 을 복원하고 시작 시와 Cleanup 시 `chSigs` 를 비운다. 
3. 두 테스트가 전역을 바꾸므로 t.Parallel 은 쓰지 않는다.

## 검증 항목 (verify)

1. `go test -count=1 -coverprofile=cp.out ./...`(vendor 가 어긋나 있으면 `GOFLAGS=-mod=mod`, 이 경우 측정 직후 `git diff --exit-code -- go.mod go.sum`) 의 수기 합계(블록별 최대 count)가 80.5% 이상이다(모자랄 때만 예비로 `init.go` 의 BindPFlag 오류 분기 테스트를 추가한다). `go test -v` 에서 SKIP 이 0건임을 확인한다(MAC 인터페이스 부재로 main 테스트가 skip 되면 main 쪽 증가분이 빠지며, skip 시 siphandler 증가분만의 수치(약 77.6%)를 별도로 확인하고, skip 이 발생한 측정은 완료 근거로 쓰지 않는다). 패키지별 전후 수치를 측정해 PR 본문에 쓴다.
2. 기존 테스트 통과, `-race -shuffle=on -count=3` 통과, golangci-lint 통과, `git diff --stat origin/main...HEAD` 의 파일이 voip-kamailio-proxy/ 안의 `_test.go` 와 이 문서뿐이며 go.mod/go.sum 무변경이다(측정 직후 `git diff --exit-code -- go.mod go.sum`).
3. 변이 확인(scratch 복사본에서만 적용하며 모두 테스트가 실패해야 함): (a) health.go dial 오류 분기의 `return` 제거, (b) 그 분기의 `ResponseCode` 변경, (c) main.go 휘발 큐 이름 형식 변경, (d) main.go 의 `Run()` 호출을 no-op 으로 대체, (e) 종료 로그 메시지 문자열 변경(줄 삭제는 미사용 변수로 컴파일 오류가 되므로 쓰지 않음), (f) main.go 의 영구/휘발 큐 이름 인자 맞바꿈.
4. flaky 확인: `-race -shuffle=on -count=20` 을 두 패키지에 돌려 실패 0 건이다.
