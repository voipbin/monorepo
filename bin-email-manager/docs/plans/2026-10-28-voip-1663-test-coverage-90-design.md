# VOIP-1663: bin-email-manager unit test coverage 90%

## 배경
수기 코드(gens, mock, DO NOT EDIT 제외) 커버리지는 origin/main 에서 85.69% 이고, 미커버는 대부분 cmd/ 의 연결 생성 구간입니다. bin-common-handler 에 머지된 테스트 주입 지점(`sockhandler.SetRabbitFactory`, `databasehandler.SetConnectFunc`, `migrationhandler.SetEngineFunc`)으로 이 구간을 외부 연결 없이 시험합니다.

## 범위
- bin-email-manager 의 `_test.go` 와, 필요 시 go.mod/go.sum(테스트 전용 모듈, 같은 버전이 다른 서비스에 존재)만 변경합니다. 비테스트 코드는 바꾸지 않습니다.

## 계획
1. cmd/email-manager: `createDBHandler` 의 마이그레이션 실패, 캐시 연결 실패, 성공 경로. `run` 은 mock sock 으로 Connect 와 listen 큐 위임 확인. `runService` 는 모든 시임을 가짜로 바꾸고 `chSigs` 로 종료 신호를 보내 정상 종료를 확인.
2. cmd/email-control: `initHandler` 와 `initEmailHandler` 성공/실패 경로. `runGet`, `runList`, `runDelete` 는 sqlite 인메모리 DB(서비스 스키마 스크립트)와 miniredis 로 성공, 조회 실패, 입력 오류(UUID 누락, 형식 오류)를 확인하고 표준출력 JSON 을 검증.
3. 격리: 주입 지점은 프로세스 전역이므로 `t.Parallel` 없이 복원 함수를 defer 또는 `t.Cleanup` 으로 호출. 고정 포트, 호스트 파일, 실제 네트워크는 쓰지 않고 Prometheus 설정은 비워 둠. `config.Get()` 필드와 viper 는 저장 후 복원. `chSigs`/`chDone` 은 시작 시 비우고 `signal.Reset` 으로 정리하며 `runService` 는 타임아웃 있는 래퍼로 실행. 가짜 sock 은 `TopicCreateWithKind` 를 nil 로 반환하고 `PublishWebhookEvent` 의 비동기 발행은 기대 호출과 완료 대기(WaitGroup)로 받음. `requesthandler.NewRequestHandler` 는 호출마다 Prometheus 수집기를 기본 registerer 에 등록해 두 번째 호출에서 panic 하므로, 해당 경로를 부르는 시험마다 `prometheus.DefaultRegisterer` 를 새 registry 로 바꾸고 복원. `initHandler` 가 `resolveUUID` 보다 먼저 실행되므로 입력 오류 시험도 DB, 캐시, sock, registry 격리를 모두 갖춤. sqlite 는 시험마다 고유 이름의 인메모리 DB(`SetMaxOpenConns(1)`).

## 검증 기준 (verify)
- 수기 합계 90.5% 이상 (같은 플래그로 최종 HEAD 에서 측정), 기존 테스트 전부 통과, `-race -count=3 -shuffle=on` 통과, lint 통과.
- 변이 표본: `runGet` 의 UUID 검증, `createDBHandler` 의 마이그레이션 오류 반환, `run` 의 Connect 호출을 각각 깨뜨려 새 테스트가 실패함을 확인.
- 자기비교식 테스트 금지: 기대값은 입력 상수에서 독립적으로 작성.

## 영향도
스키마 변경 없음, 운영 영향 없음. PR 파일은 bin-email-manager/ 안에만 존재.
