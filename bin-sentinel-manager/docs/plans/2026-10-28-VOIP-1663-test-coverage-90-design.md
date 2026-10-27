# VOIP-1663 bin-sentinel-manager 단위 테스트 커버리지 90% 이상

수치는 수기 코드 기준입니다(gens/, mock_*.go, DO NOT EDIT 헤더 파일 제외, 같은 블록은 최대 count 로 합침). 기준선은 917/1058 = 86.67% 이며 목표는 90% 이상입니다. 측정 명령은 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 입니다.

## 현황

미커버의 대부분은 cmd/sentinel-manager 의 `run`, `runService`, `buildBackend` 이고, 나머지는 cachehandler, dockerwatchhandler, k8swatchhandler 의 오류 경로입니다. bin-common-handler 에 `sockhandler.SetRabbitFactory` 가 추가되어 RabbitMQ 연결 없이 서비스 기동 경로를 실행할 수 있습니다. 제품 코드는 바꾸지 않고 테스트만 추가합니다.

## 접근

- cmd/sentinel-manager (`run_test.go`): `SetRabbitFactory` 로 `MockSockHandler` 를 주입합니다(`Connect` 1회, `TopicCreateWithKind` 1회). docker 백엔드는 loopback `httptest` 가짜 Docker 서버(`/_ping`, `/containers/json`, `/events`)와 miniredis 로 구성합니다. `requesthandler` 가 호출마다 metrics 를 등록하므로 테스트마다 `prometheus.DefaultRegisterer` 를 새 레지스트리로 바꿉니다. `run` 은 호출마다 다른 metrics endpoint 경로와 `127.0.0.1:0` listen 주소를 쓰고, 신호는 `chSigs` 에 직접 주입하며, 신호 고루틴이 신호를 읽은 것을 확인한 뒤에만 `signal.Stop` 과 정리를 합니다. 모든 설정은 `t.Setenv` 로 명시하고 `KUBERNETES_SERVICE_HOST` 와 `KUBERNETES_SERVICE_PORT` 는 비웁니다.
- cachehandler: miniredis 의 `SetPreHook` 으로 GET 과 TTL 에 null 과 오류 응답을 주입합니다.
- dockerwatchhandler: Run 의 초기 refresh 경고, 이벤트 루프 오류 전파, 사전 취소된 ctx, 재연결 대기 중 취소(로그 호출 지점에서 cancel), refresh 도중 항목 소멸.
- k8swatchhandler: fake clientset 의 watch reactor 로 selector 별 `watch.FakeWatcher` 를 주입합니다. 초기 목록은 `Added` 이후 `k8s.io/initial-events-end` 주석의 `Bookmark` 로 끝내고, `Modify`(UID 변경)와 `Delete` 로 콜백을 실행합니다. watch 오류로 예산을 소진하는 경우와 목록 실패로 동기화가 실패하는 경우도 실행합니다.
- sentinel-control: `runContainerGet` 의 클라이언트 생성 오류와 `main` 의 help 경로.
- 모든 대기는 채널과 timeout 으로 동기화하고 sleep 에 의존하지 않으며 `t.Parallel` 을 쓰지 않습니다. 전역 상태는 `t.Cleanup` 으로 복원합니다.

## 검증

- `runService`: 이벤트 스트림이 열린 뒤 ctx 취소는 nil, 부팅 목록 실패와 Redis 연결 실패와 kubernetes 설정 실패는 각각의 래핑 오류를 반환합니다.
- `run`: 유효 설정에서 기동되어 신호로 nil 종료하고, backend 실패는 래핑된 오류를 반환합니다.
- cachehandler: GET null 과 TTL null 은 해당 키를 건너뛰고, GET 오류와 TTL 오류는 키가 포함된 오류를 반환합니다.
- dockerwatchhandler 와 k8swatchhandler: 각 경로의 반환 오류 문구, 발행 순서(died, started, died), 상태 표 결과를 단언합니다.
- 새 테스트가 결함을 잡는지는 작성 중 제품 코드 변이를 별도 복사본에서 적용해 확인합니다.
