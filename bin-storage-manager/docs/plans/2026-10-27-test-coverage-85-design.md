# bin-storage-manager 단위 테스트 커버리지 85% 상향 (VOIP-1659)

## 1. 티켓 검토와 기준선

- 티켓 검토: 유효. 비테스트 코드는 바꾸지 않고 단위 테스트만 추가한다.
- 수기 코드: `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` 헤더 파일을 제외하고 같은 블록의 중복은 최대 count 로 합친다.
- 측정: 서비스 디렉터리에서 `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` 후 위 규칙으로 집계한다.
- 기준선(origin/main a47f5a2a1): 수기 84.58% (2150/2542 statement), 원본 74.2%. 85.0% 에는 11개(2161 필요), 85.5% 에는 24개(2174 필요)가 더 필요하다.

| 패키지 | 미커버 statement |
|---|---|
| cmd/storage-control, cmd/storage-manager | 161, 115 |
| pkg/filehandler | 44 (file.go 13, peak.go 11, bucketfile.go 10, main.go 7, compress.go 2, signing.go 1) |
| pkg/listenhandler | 31 |
| pkg/dbhandler | 30 (accounts.go 15, files.go 15) |
| 그 밖(config 3, models 4, storagehandler 1, subscribehandler 3) | 11 |

## 2. 도달 가능성

- cmd/storage-control: init 이후 코드와 `runService`, `createDBHandler` 의 연결 이후는 DB 연결 성공이나 브로커 연결(무한 재시도)이 필요해 제외한다. 연결 전 실패 경로는 `Test_run_handlerInitFailure`, `Test_initHandlers_connectFailure` 가 이미 덮는다. `initCache` 의 연결 실패(3 statement)는 닿지 않는 주소로 직접 호출하면 도달하므로 조건부로 둔다.
- cmd/storage-manager `run()`: 잘못된 DSN 으로 호출하면 DB 연결 전에 실패해 약 23~27 statement 가 도달 가능하다(실측, 표의 26~30 은 initCache 3 을 합친 값). `initProm` 이 전역 `http.Handle` 을 등록하고 loopback listen goroutine 을 띄우므로 한 번만 호출할 수 있다. 조건부 항목이며 확정분이 모자랄 때만 추가한다.
- 도달 가능하나 제외하는 것(가치 대비 비용): listenhandler 와 models 의 json.Marshal 실패는 time.Time 연도 범위 밖 값이나 NaN 을 mock 이 돌려주면 일부 도달하지만 마샬 실패를 일으키려고 비정상 값을 주입하는 것이라 제외한다. goroutine 경합이 있는 Run goroutine(listenhandler 1, subscribehandler 2)은 동기화 비용과 flaky 위험으로 제외하며 subscribehandler 114행은 기준선이 ±1 흔들릴 수 있다.
- 도달 불가: 설정 바인딩 에러(config 3, `Lookup` 이 항상 non-nil), dbhandler 의 Create/Get/Delete 쪽 PrepareFields, ToSql(입력이 고정), `peak.go` 의 `frameSize == 0`(bytesPerSample 2, numChannels 1 이상이라 2 이상)과 `framesPerBucket <= 0`(bucketCount 가 1 이상 totalFrames 이하로 잘림), `NewFileHandler` 의 `storage.NewClient` 실패 분기(137행, 생성 시 네트워크를 쓰지 않아 실패시키는 입력이 없음).
- 도달 가능 집계:

| 영역 | statement | 방법 |
|---|---|---|
| listenhandler | 7 | URI 경로에 `%zz` 를 넣어 url.Parse 실패 4곳(`v1AccountsGet`, `v1AccountsIDGet`, `v1FilesGet`, `v1FilesIDGet`), recording id `uriItems[3]` 에 `%zz` 를 넣어 QueryUnescape 실패 3곳. 쿼리의 `%zz` 는 파싱에 실패하지 않으므로 경로에 둔다. 라우팅 정규식은 시작이 고정되지 않아 `/%zz/v1/accounts` 같은 접두 경로나 `/v1/recordings/%zz/v1/recordings/<uuid>` 같은 URI 가 핸들러에 도달한다(핸들러 직접 호출도 가능). 단언은 에러 반환과 nil 응답 |
| filehandler file.go | 13 | Create 실패 경로 10, Get 3. 의존성은 gomock 과 기존 `storageFake` |
| filehandler peak.go | 9 | `storageFake` 의 미디어 GET 으로 NewReader 실패와 정상 WAV |
| filehandler compress.go | 2 | 파일이 있는 `compressCreate`, 메서드 래퍼 |
| storagehandler, subscribehandler, models | 4 | reference id 조회 실패 1, 이벤트 처리 실패 로그 경로 1, `ConvertStringMapToFieldMap` 에러 2 |
| (확정 소계) | 35 | |
| filehandler bucketfile.go | 최대 10 | `storageFake` 는 삭제와 rewrite 가 항상 성공이므로 실패 스위치를 테스트 파일 안에서 확장한다. 쓰기 실패는 업로드 상태 코드, 175행(SignedURL 실패)은 잘못된 privateKey 와 accessID 로 도달. 블록마다 구현 중 도달 여부를 확인 |
| filehandler main.go, signing.go | 최대 7 | 파싱 불가 JSON(120, 122행), 테스트에서 만든 RSA 키가 든 서비스 계정 JSON(125, 131행, `NewClient` 는 네트워크를 쓰지 않음), 서명 거부 로그(signing.go) |
| dbhandler | 최대 20 | 파일마다 10곳: Create 후 캐시 재조회/설정 실패(gomock), 행 스캔 실패와 GetFromDB/Gets 의 행 변환 실패(컬럼 타입이 어긋난 행), RowsAffected 에러(`*sql.DB` 를 바꿔 끼울 수 없으므로 `ctx_rows_test.go` 의 `failAfterConnector` 처럼 테스트 파일 안에서 database/sql/driver 를 구현하고 `sql.OpenDB` 로 여는 connector 신설), List 의 미지원 필터로 ApplyFields 실패, Update 의 직렬화 불가 값으로 PrepareFields 실패, 에러를 내는 Valuer/Sqlizer 값으로 List/Update 의 ToSql 실패. 래퍼는 테스트 파일 안에서만 만든다. 신설이 불가하면 go-sqlmock 은 하드룰 조건(다른 서비스와 같은 버전)을 만족할 때만 쓴다 |
| cmd/storage-manager run(), storage-control initCache | 약 26~30 | 위 조건부 항목 |

- 합계는 확정 35 + 조건부는 최대 약 65 이다(블록별 도달 여부는 구현 중 확인). 24개 필요 대비 확정분만으로 11개 여유가 있어 상한은 85.5% 를 넘는다. block 하지 않는다.

## 3. verify 항목

- listenhandler: 위 7곳이 에러를 반환하고 하위 핸들러(mock)가 호출되지 않음.
- Create: 원본 속성 조회 실패, 계정 검증 실패, 이동 실패(목적지 이미 존재), `FileCreate` 실패, 생성 후 `FileGet` 실패가 에러를 반환하고 이후 단계(이벤트 발행, 계정 증가)를 호출하지 않음. `IncreaseFileInfo` 실패는 Create 를 실패시키지 않음. Get 은 NotFound 를 `FILE_NOT_FOUND` 로, 그 밖의 에러를 감싸 반환.
- PeaksComputeByFile: 읽기 열기 실패는 에러, 정상 WAV 는 peaks 와 duration, bucketCount 0 은 기본 개수로 대체.
- compressCreate: 파일이 있으면 파일 ID 목록으로 해시를 만들고, 래퍼는 기존 zip 이 있으면 압축을 호출하지 않음.
- storagehandler: reference id 파일 조회가 실패하면 CompressfileCreate 가 에러를 감싸 반환하고 압축을 호출하지 않음.
- subscribehandler: customer_deleted 이벤트의 하위 핸들러가 에러를 반환해도 `processEvent` 가 반환하고 mock 호출이 소진됨(패닉 없음, 호출 횟수는 gomock 이 검증).
- models: 필드 타입에 맞지 않는 값에서 `ConvertStringMapToFieldMap` 이 에러를 반환.
- 조건부 항목의 단언: dbhandler 는 반환 에러가 해당 연산 이름을 포함하고 ErrNotFound 로 오분류되지 않으며 캐시에 아무것도 쓰지 않음. bucketfile 은 실패한 단계 이후의 객체 상태(소스 삭제 여부, 불완전 zip 삭제 여부)를 fake 가 기록한 요청으로 단언. main.go 는 반환 에러 문구와 서명 가능 여부 필드(accessID, privateKey) 값을 단언. signing 로그 경로는 Create 가 성공하고 URIDownload 가 비며 TMDownloadExpire 가 nil 임을 함께 단언. subscribehandler 는 성공 대조 케이스에서 하위 핸들러 호출 인자를 단언하고 실패 케이스와 비교.
- 추가 단언: Create 의 이동, `FileCreate`, `FileGet` 실패 케이스는 `PublishEvent` 와 `IncreaseFileInfo` 가 호출되지 않음을 gomock 으로 단언한다. peaks 의 기본 개수 대체는 프레임 1000개 이상인 WAV 로 `len(peaks)` 가 기본 개수와 같고 `bucketCount` 를 준 경우와 다름을 단언하며, peak 값은 바이트에서 손으로 계산한 기댓값(진폭 / 32768)과 비교한다. dbhandler 는 에러를 버리는 호출(`_ =`)과 반환하는 호출을 블록마다 구분해 단언한다. 변이 대상은 peaks 의 `bucketCount <= 0` 분기와 NewReader 에러 처리, compress 의 `isExistFunc` 조기 반환을 포함한다.
- 격리: 외부 네트워크와 시간 의존이 없고 httptest 서버(`STORAGE_EMULATOR_HOST` 는 `t.Setenv`, `t.Parallel` 미사용)와 gomock 만 쓴다. go.mod, go.sum 은 가능하면 불변.
- 변이 검사: 새로 추가한 테스트 군(listenhandler, Create, Get, peaks, compress, storagehandler, models, 그리고 추가한 조건부 군)마다 대응하는 에러 반환이나 분기를 최소 하나씩 제거해(변이는 scratch 복사본에서 한다) 해당 테스트가 실패함을 확인한다.
- 완료 기준: 수기 합계 85.5% 이상, 기존 테스트 전부 통과, golangci-lint 통과, PR 파일이 bin-storage-manager/ 안에만 있음. VOIP-1416 과 겹치는 패키지는 PR 본문에 명시만 한다.
