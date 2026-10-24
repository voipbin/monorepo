# voip-kamailio-proxy 단위 테스트 커버리지 65% 설계 (VOIP-1649)

기준 커버리지는 수기 코드 64.8% (214/330 statement) 이고, 이 서비스에는 생성 코드와 mock 이 없어 원본과 같다. 목표는 65% 이상이고, CI 와의 소수점 차이 여유를 위해 66% 이상을 권장한다. 65% 판정은 siphandler 테스트(5060 바인드 불가 시 skip)에 의존하지 않고 listenhandler 테스트만으로 성립해야 한다(로그 줄 2개 제외 후 약 70.9%, 기존 siphandler 테스트 포함). 제품 코드는 바꾸지 않고 `_test.go` 파일만 추가한다.

## 대상과 검증 항목 (verify)

1. pkg/listenhandler/main_test.go (sockhandler.MockSockHandler 사용, 네트워크 없음)
   - NewListenHandler: 같은 패키지 테스트에서 반환값을 `*listenHandler` 로 단언하고 전달한 큐 이름 2개, 타임아웃, sockHandler 를 필드로 보존하는지 확인한다. 체커는 func 값이라 비교할 수 없으므로 보존된 sipChecker 를 호출해 센티널 결과가 돌아오는지 확인한다.
   - listenRun 성공: permanent 큐는 타입 "normal", volatile 큐는 타입 "volatile" 로 생성하고, 두 큐 모두에 대해 consumer 이름 "kamailio-proxy", exclusive/noLocal/noWait 모두 false, 워커 수 10 으로 ConsumeRPC 를 호출한다(불리언 인자는 gomock.Any 로 뭉개지 않고 명시). Run 처럼 백그라운드 고루틴이 부르는 mock 기대값(QueueCreate 2회, ConsumeRPC 2회)은 모두 ctrl.Finish 전에 설정하며, QueueCreate 는 ConsumeRPC 보다 먼저 호출되므로 ConsumeRPC 완료 대기로 충분히 동기화된다. 큐별로 기대값을 따로 설정하고 gomock.InOrder 는 쓰지 않는다. 두 큐의 ConsumeRPC 호출이 모두 끝날 때까지(WaitGroup 카운트 2) 기다린 뒤 컨트롤러를 종료한다. 소비 콜백은 DoAndReturn 안에서 채널(또는 mutex)로 전달하고 WaitGroup 완료 후 메인 고루틴에서 호출해 확인한다(-race 에서 데이터 레이스가 없어야 함). 두 큐의 콜백이 모두 processRequest 로 라우팅되는지 큐별로 확인한다: 스텁 체커를 둔 핸들러에 `POST /v1/providers/health` 를 보내 200/healthy 가 나오는지(라우팅까지 도달) 확인하고 404 확인은 보조로 둔다.
   - listenRun 오류: permanent 큐 생성 실패 시 오류를 반환하고 volatile 생성과 ConsumeRPC 를 호출하지 않는다(기대값 미설정으로 검증). volatile 큐 생성 실패 시 오류를 반환하고 ConsumeRPC 를 호출하지 않는다.
   - ConsumeRPC 가 오류를 반환해도 listenRun 은 nil 을 반환하고 두 큐 모두 소비를 시도한다(큐별 호출 완료 대기, 위와 동일).
   - Run: 즉시 nil 을 반환하고 백그라운드에서 listenRun 이 두 큐 생성과 두 ConsumeRPC 호출까지 수행한다. 테스트는 두 ConsumeRPC 호출 완료(WaitGroup 카운트 2)까지 기다린 뒤 종료한다. ConsumeRPC 기대값이 없는 상태로 끝나지 않게 한다.
2. processRequest 경계
   - 기존 routing 테스트 테이블에 케이스를 추가한다. 같은 URI 에 GET 이면 404, 접두사/접미사가 붙은 URI(`/v1/providers/health/x`, `/x/v1/providers/health`) 는 404 이며 sipChecker 는 호출되지 않는다.
   - unhealthy 결과(Healthy=false, 코드 "timeout") 가 200 응답의 status "unhealthy", result_code "timeout" 으로 매핑된다. 스텁 체커 안에서 hostname 과 timeout 을 캡처해 비교하며, 설정 타임아웃은 기본값(5s)과 다른 3s 로 두어 자기비교가 되지 않게 한다. 호스트명은 `192.0.2.x`/`example.invalid` 같은 예약 값을 쓴다.
3. pkg/siphandler/health_responder_test.go (루프백 UDP 응답기)
   - 127.0.0.1:5060 바인드가 실패하면 t.Skip. SendOptionsCheck 에 넘기는 timeout 은 2s 이상으로 하고(느린 CI 와 -race 대비), 응답기가 결과를 보내는 채널은 버퍼 1 이상으로 둔다. 응답기 고루틴은 읽기 데드라인을 두고 수신한 요청 첫 줄과 헤더 검증 결과를 채널로 전달하며, 테스트 종료 전에 소켓을 닫고 고루틴을 join 한다. 같은 포트를 쓰므로 t.Parallel 을 쓰지 않고 케이스를 직렬로 실행한다. 응답기가 받은 첫 줄이 `OPTIONS sip:healthcheck@<host> SIP/2.0` 이고 Via, CSeq, Max-Forwards 가 있는지 확인한다.
   - 응답 첫 줄 `SIP/2.0 404 Not Found` -> Healthy=true, 코드 "404". `SIP/2.0 200 OK` -> Healthy=true, 코드 "200". 형식 불량 첫 줄(공백 없음) -> Healthy=true, 코드 "unknown".
4. 제외(추가): ConsumeRPC 오류 로그 줄과 Run 내부 listenRun 오류 로그 줄은 완료 신호 이후 실행되어 종료와 경합하므로 커버 목표에서 제외한다(65% 판정에 영향 없음). 기존 Test_healthResponseFormat 은 구조체를 직접 marshal 하는 테스트이므로 새 unhealthy 테스트(processRequest 경유 JSON 디코드)가 대체하지 않고 보완한다.
5. 제외: main(), initVariable 의 파싱 실패 분기, initProm, 도달 불가 분기(handler 오류 -> 400, json.Marshal 오류), OS 오류 주입이 필요한 UDP 오류 분기.

새 테스트는 실제 업체 호스트명을 쓰지 않는다. 5060 skip 여부는 로컬 실측으로 완료 보고에 남긴다.

## 변이 확인 계획
큐 타입 문자열, permanent/volatile 큐 이름 맞바꿈, consumer 이름, 워커 수, 정규식 앵커, Healthy 플래그와 status 매핑, 코드 파싱 인덱스를 임시로 바꾸었을 때 새 테스트가 실패하는지 표본 확인하고 되돌린다. 되돌린 뒤 `git diff --stat` 에 `_test.go` 와 이 문서 외 변경이 없음을 확인한다(제품 코드 비변경 검증). 변이 중에는 커밋하지 않고 `git checkout -- <file>` 로 되돌린다(추적 중인 제품 코드에만 해당하며 새 `_test.go` 는 대상이 아니다).

## 완료 기준
수기 합계 66% 이상 예상(siphandler 포함 약 73%, 5060 skip 시 약 71%이며 65% 판정에는 영향 없음), 기존 테스트 통과, golangci-lint 통과, PR 파일은 voip-kamailio-proxy/ 안에만 있다.
