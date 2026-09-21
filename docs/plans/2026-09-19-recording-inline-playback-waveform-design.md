# Recording Inline Playback + Waveform — Design

Status: Design APPROVED (2 consecutive rounds, iter-2/iter-3)
Date: 2026-09-19
Author: CPO (Hermes)
Analysis: `2026-09-19-recording-inline-playback-waveform-analysis.md` (Analysis APPROVED, iter-4/5)

## 1. Problem statement

square-admin 레코딩 상세 페이지에는 Download(zip) 버튼만 있어 통화 녹음을 바로 들을 수 없다. 재생하려면 zip을 내려받아 압축을 풀고 외부 플레이어로 열어야 한다. 상담 품질 검토·분쟁 확인 등 콜센터 운영에서 녹음 확인은 빈번한 작업이므로, 다운로드 없이 상세 페이지에서 바로 재생하고 음성 파형으로 구간을 시각 확인할 수 있어야 한다.

## 2. Goals (numbered, testable)

- G1. 레코딩 상세 페이지에서 다운로드 없이 녹음을 인라인 재생할 수 있다(네이티브 오디오 컨트롤).
- G2. 통화 녹음의 방향별(in/out) 파일을 각각 재생할 수 있다. conference 녹음은 단일 파일.
- G3. 재생과 함께 음성 파형을 표시한다. **녹음 길이와 무관하게 모든 녹음에서 파형을 표시한다**(대표님 결정, 분석 §5.4).
- G4. 파형 데이터는 백엔드가 계산해 제공한다. 클라이언트는 wav 원본을 파형 목적으로 내려받지 않는다(CORS·이그레스·모바일 메모리 문제 회피).
- G5. 녹음이 아직 없거나(진행중/등록지연) 파일이 없을 때 재생 UI가 빈/생성중 상태를 구분해 안내한다.

## 3. Non-goals (scope cuts)

- N1. list 행에서의 바로 재생(분석 D1: 상세 페이지 한정).
- N2. 파형 위 구간 지정/클립 편집/주석. v1은 재생 + 파형 표시 + seek만.
- N3. 재생 속도 조절·트랜스크립트 동기 하이라이트. (별도 후속.)
- N4. recordingfiles/{id}(zip) 다운로드 경로 변경. 기존 Download 버튼 유지.

## 4. Decisions locked (대표님 확정)

- D1. 재생 UI = 레코딩 상세 페이지 인라인.
- D2. 재생용 wav URL = 신규 `GET /recordings/{id}/playfiles`.
- D3. 파형 = 인라인 플레이어에 포함, **전체 녹음(길이 무관)**.
- D4 (분석 §5.4). 파형 데이터 = 백엔드 peak 계산(옵션 B) 채택. 클라이언트 wav 원본 파형 fetch 없음.

## 5. Architecture

```
square-admin recordings_detail
   │  GET /recordings/{id}/playfiles          (신규)
   ▼
bin-api-manager (게이트웨이, 권한 확인만)
   │  RPC: StorageV1RecordingPeaks(recording_id)   (신규, 온디맨드+캐시)
   │  RPC: StorageV1FileList(reference_type=recording, reference_id=id)  (기존)
   ▼
bin-storage-manager
   │  - StorageV1FileList: 재생용 File 목록(uri_download 포함) 반환 (기존)
   │  - Recording peaks: File별 wav를 GCS에서 읽어 peak 배열 계산, Redis 캐시 (신규)
   ▼
GCS (wav)  +  Redis (peak 캐시)
```

- 재생 스트림: 프론트 네이티브 `<audio src={uri_download}>`. api-manager/storage-manager 경유 없음(GCS signed URL 직접). 길이 무관 스트리밍.
- 파형: 프론트가 playfiles 응답의 `peaks` 배열을 wavesurfer에 주입(`peaks` 옵션). wav를 fetch하지 않음 → CORS/이그레스/메모리 무관.

### 5.1 peak 계산: 온디맨드 + 캐시 (채택) vs 사전계산 + 저장 (기각)

| 항목 | A. 온디맨드+캐시 (채택) | B. 사전계산+저장 (기각) |
|---|---|---|
| 계산 시점 | playfiles 첫 호출 시 storage-manager가 계산, Redis 캐시(24h) | 녹음 등록(stop.go) 시 계산해 DB 저장 |
| DB 스키마 | 무변경 | files 테이블에 peak 컬럼 추가(Alembic) |
| call-manager | 무변경 | 등록 흐름 수정 |
| 기존 녹음 | 자동 동작(backfill 불필요) | backfill 마이그레이션 필요 |
| 첫 요청 지연 | wav 디코딩 1회(캐시 후 없음) | 없음 |
| 스코프 | openapi + api-manager + storage-manager | + dbscheme + call-manager + backfill |

**채택 = A.** 스코프가 작고(3서비스), DB/등록흐름 무변경, 기존 녹음이 backfill 없이 동작. 첫 요청 지연은 Redis 캐시로 1회에 국한. 서버측 디코딩은 스트리밍이라 클라이언트 대비 안전. B는 오버엔지니어링 지양 원칙(신규 스키마·마이그레이션·다서비스 확장)과 충돌하며, 실측 신호(첫 요청 지연이 실제 문제) 없이 선제 도입하지 않는다.

### 5.2 peak 계산 상세

- 대상: `StorageV1FileList(reference_type=recording, reference_id=id)`가 반환한 각 File(방향별 wav).
- 계산: storage-manager가 `bucketFile.NewReader`(`bucketfile.go:102`)로 wav 스트림을 읽어 고정 개수 버킷(예: 1000 샘플)으로 다운샘플, 각 버킷 절대값 max 배열 생성(0.0~1.0). wavesurfer `peaks` 포맷(float 배열, 0.0~1.0은 유효 부분집합) 준수.
- 포맷: VoIPBin wav = Asterisk `format=wav` 기본 8kHz/16bit mono(실측은 구현 시 wav 헤더로 확정). 헤더의 samplerate/channels/bits를 파싱해 하드코딩하지 않는다.
- 캐시: Redis key `recording_peaks:{recording_id}`, TTL 24h(기존 파일 캐시와 동일 관례, `cachehandler/handler.go:36`). 녹음은 종료 후 불변이므로 캐시 안전.
- 캐시 스탬피드(백엔드 리뷰 5): 긴 녹음 첫 요청이 동시에 오면 각 요청이 wav를 중복 디코딩할 수 있다. v1은 계산 자체가 idempotent하고 결과가 동일하므로 정합성 문제는 없으며, 중복 디코딩 비용은 실측 신호(동시 요청 폭주) 확인 시 singleflight 도입으로 대응한다(현재 선제 도입 안 함, 오버엔지니어링 지양). 이 유보를 §9 리스크에 명기.
- 신규 의존성: Go wav 디코딩 라이브러리(예: `github.com/go-audio/wav`). vendor에 없음 → 추가. 경량, 표준 wav PCM만 파싱.
- graceful degrade(백엔드 리뷰 4): wav 헤더 파싱 실패/디코딩 오류 시 해당 파일의 `peaks`를 빈 배열로 폴백하고 로그만 남긴다. peak 실패가 playfiles 전체(재생 포함)를 실패시키지 않는다. 재생은 peak과 독립(uri_download).

### 5.3 신규 엔드포인트: GET /recordings/{id}/playfiles

- api-manager server: `server/recordings.go`(또는 신규 파일)에 핸들러 추가. path `/recordings/{id}/playfiles`.
- servicehandler `RecordingPlayfilesGet(ctx, a, id)`:
  1. `recordingGet(ctx, id)` (기존 private helper, no perm) → recording. `recording.reference_type`(call/confbridge) 확보.
  2. `hasPermission(ctx, a, r.CustomerID, PermissionCustomerAdmin|PermissionCustomerManager)` (기존 top-level recording과 동일 권한).
  3. `StorageV1FileList(reference_type=recording, reference_id=id)` → File 목록.
  4. `StorageV1RecordingPeaks(ctx, id)` → **파일명(또는 file_id)을 키로 하는 맵** `{ filename: {peaks: [...], duration: n} }` 반환(신규 RPC). 실패 시 빈 맵(각 파일 graceful degrade). **조인 키(백엔드 후속 1/2)**: FileList 결과와 peaks 결과는 배열 순서가 아니라 **filename(고유)으로 상관**한다. in/out 2파일에서 순서 의존 오정렬을 방지. duration도 이 맵에서 파일별로 가져와 §5.3 duration 권위 출처와 일원화(peak과 동일 구조).
  5. File(filename 키) + peaks 맵(filename 키) 병합해 응답 항목 배열 반환. 맵에 없는 파일은 peaks=[], duration=0.
- 응답 스키마 `RecordingPlayfile` (openapi 신규 컴포넌트; **call-manager 도메인 모델이 아니라 api-manager가 합성하는 신규 DTO이므로 `CallManager*` 접두를 쓰지 않는다** — 백엔드 리뷰 6, 프론트 리뷰 4):
  - `filename` (string)
  - `direction` (string: `in`/`out`/빈값)
  - `uri_download` (string: 재생용 signed URL, 기존 File.uri_download 재사용. 분석 R2: attachment disposition이나 `<audio>` 재생 무해)
  - `tm_download_expire` (date-time: 만료. 분석 R3/R3b)
  - `filesize` (integer)
  - `duration` (number, seconds: 파형/재생 duration의 권위 출처. **파일별 wav 헤더에서 서버가 산출**(프론트 리뷰 4). in/out 파일 길이가 다를 수 있어 recording 전체 duration이 아닌 파일별 duration을 제공. peak 실패로 빈 peaks면 0 또는 헤더 실패 시 0.)
  - `peaks` (array[number]: 절대값 크기 파형 0.0~1.0. 전체 파일. 실패 시 빈 배열.)
- **direction 규칙 (재작성 — 프론트 리뷰 1/3, confbridge `_in` 단일 오태깅 방지)**: direction은 **`recording.reference_type == call` 이고 파일이 2개일 때만** filename 접미(`_in`/`_out`)로 부여한다. `reference_type == confbridge`(단일 파일, 파일명이 `_in`이지만 방향 의미 없음)나 파일이 1개면 direction=빈값으로 두고 단일 플레이어로 렌더한다. 즉 `_in` 접미 존재만으로 direction을 부여하지 않는다.
- 정렬: direction 있으면 in→out, 없으면 filename 순.

### 5.4 signed URL 만료 (분석 R3/R3b)

- playfiles 응답의 uri_download는 File의 기존 signed URL. 만료 시 재생 실패.
- **1차 방어선(서버 선제 재발급)**: playfiles 호출 시점에 `tm_download_expire` 잔여가 임계(OQ4) 미만인 File은 `StorageV1FileDownloadURIRefresh`로 재발급 후 반환. 재생 시작~긴 녹음 재생 중(R3b) 만료 위험을 줄인다. **부수효과 명기(프론트 후속 4)**: 이 GET 엔드포인트는 재발급 시 File 레코드에 write 부수효과가 있다(멱등적: 반복 호출해도 유효 URL 유지). recording_*.rst 엔드포인트 문서에 이 부수효과/멱등성을 명기한다.
- **2차 방어선(프론트, 403 직접감지 불가 — 프론트 리뷰 2)**: HTML `<audio>`는 HTTP 상태코드를 노출하지 않고 `MediaError`(error/stalled/abort 이벤트)만 준다. 따라서 프론트는 (a) `tm_download_expire` 기반 시간제어로 만료 임박 시 선제적으로 playfiles 재호출, (b) audio `error`/`stalled`/`abort` 이벤트 발생 시 playfiles 재호출 → 새 uri_download로 교체 후 `currentTime` 복원. "403 감지"에 의존하지 않는다.

## 6. Frontend design (square-admin)

- 신규 컴포넌트 `recordings/RecordingPlayerCard.js`(가칭): recordings_detail의 Recording Information 카드 아래, AISummary/Transcript 위에 배치.
- 데이터: `useDetailResource` 또는 provider `Get('recordings/{id}/playfiles')`로 playfiles 독립 로드. **status prop 계약(프론트 후속 3)**: 상태 분기가 `recording.status`에 의존하나 카드는 playfiles를 독립 로드하므로, 부모 recordings_detail이 `recordingData.status`를 prop으로 전달한다(`recordingStatus` prop). 카드가 recording을 재조회하지 않는다.
- 재생: 항목별 네이티브 `<audio controls src={uri_download}>`.
  - direction 있는 2파일(call): In/Out 라벨 + 각각 플레이어(나란히 2개, 단순·명확). direction 빈값(confbridge/단일): 라벨 없이 단일 플레이어(프론트 리뷰 1/3).
- 파형: 각 오디오에 wavesurfer 인스턴스, `peaks` 배열 + 서버 제공 `duration` 주입, `media` 옵션으로 네이티브 audio 엘리먼트와 연결(backend=MediaElement). wavesurfer가 `peaks`+`duration` 제공 시 wav를 다시 fetch/decode하지 않음 → G4 충족. seek/진행 표시는 wavesurfer가 native audio와 동기.
  - `peaks`가 빈 배열(백엔드 graceful degrade)이면 파형 생략, 네이티브 재생만 표시.
- **duration 라벨 혼선 방지(프론트 후속 2)**: Recording Information 카드의 duration(tm_start/tm_end 기반 `computeDurationSec`, 통화 전체)과 플레이어의 duration(wav 헤더 파일별)이 다를 수 있다. 혼선 방지를 위해 라벨을 구분한다(예: 정보 카드는 "Call duration", 플레이어는 "File duration" 또는 무라벨).
- 상태(분석 §5.3):
  - playfiles 로딩중: 스피너.
  - 빈 배열 + recording.status ∈ {initiating,recording,stopping}: "Recording in progress" 안내(재생 불가).
  - 빈 배열 + status=ended: "No playable files"(등록 지연 가능) 안내.
  - 정상: 플레이어 + 파형.
- 라이브러리: `wavesurfer.js`(v7, React 18 호환) 신규 의존성. package.json 추가.
- 접근성(분석 R7): 재생 주체가 네이티브 `<audio controls>`라 키보드·ARIA·모바일 자동재생 정책 기본 충족. 파형은 시각 보조(aria-hidden), 조작은 audio 컨트롤로.

## 7. Affected files

### Backend (monorepo) — 1 PR
| File | Why |
|---|---|
| `bin-openapi-manager/openapi/paths/recordings/id_playfiles.yaml` | 신규 path `GET /recordings/{id}/playfiles` |
| `bin-openapi-manager/openapi/openapi.yaml` | path $ref 등록 + `RecordingPlayfile` 스키마 |
| `bin-openapi-manager/gens/models/gen.go` | 생성물(go generate) |
| `bin-api-manager/gens/openapi_server/gen.go` (+ redoc gens) | **api-manager 자체 서버 인터페이스 스텁 재생성**(백엔드 리뷰 2). `bin-api-manager/openapi/config_server/generate.go`가 openapi.yaml 소비. 이게 있어야 구현할 `GetRecordingsIdPlayfiles` 인터페이스 메서드가 생긴다. |
| `bin-storage-manager/models/...` (peak 요청/응답 도메인 타입) | peak RPC 도메인 타입 |
| `bin-storage-manager/pkg/filehandler/*` (+ `mock_main.go`) | wav 읽어 peak 계산. 신규 인터페이스 메서드 → mock 재생성(백엔드 리뷰 3) |
| `bin-storage-manager/pkg/storagehandler/*` (+ `mock_main.go`) | peak 계산 오케스트레이션 + 캐시. 신규 인터페이스 메서드 → mock 재생성(백엔드 리뷰 3) |
| `bin-storage-manager/pkg/cachehandler/*` | recording_peaks get/set |
| `bin-storage-manager/pkg/listenhandler/*` | **라우팅 충돌 해소(백엔드 리뷰 1)**: 기존 catch-all `regV1RecordingsID = "/v1/recordings/(.*)"`(`main.go:60`)가 `/peaks`까지 greedy 흡수. (a) 기존 정규식을 `/v1/recordings/` + regUUID + `$`로 앵커링, (b) 신규 `regV1RecordingsIDPeaks = "/v1/recordings/" + regUUID + "/peaks$"` 케이스를 catch-all보다 **앞에** 배치. |
| `bin-common-handler/pkg/requesthandler/storage_recordings.go` | `StorageV1RecordingPeaks` 클라이언트 |
| `bin-api-manager/pkg/servicehandler/recording.go` (또는 신규) | `RecordingPlayfilesGet` |
| `bin-api-manager/pkg/servicehandler/main.go` + `mock_main.go` | 인터페이스 + mock |
| `bin-api-manager/server/recordings.go` (또는 신규) | HTTP 핸들러 |
| `bin-storage-manager` go.mod/go.sum, vendor | wav 디코딩 의존성 |
| `bin-api-manager/docsdev/source/recording_*.rst` | 신규 엔드포인트 문서 |

주의: `bin-common-handler` admission rule — `StorageV1RecordingPeaks`는 requesthandler(RPC 클라이언트)라 예외 대상(공유 라이브러리 자체 plumbing, `storage_recordings.go`가 이미 `smbucketfile` 모델 import하는 선례 있음). wav 디코딩 로직은 storage-manager 내부에만 둔다(공유 금지). transport DTO layering: peak 도메인 타입은 storage-manager `models/`에 두고, listenhandler만 wire request/response DTO를 다룬다.

### Frontend (monorepo-javascript) — 1 PR
| File | Why |
|---|---|
| `square-admin/src/views/recordings/RecordingPlayerCard.js` | 신규 인라인 플레이어+파형 |
| `square-admin/src/views/recordings/recordings_detail.js` | 카드 삽입 |
| `square-admin/src/types/api.ts` | playfile 타입 |
| `square-admin/package.json`, lock | wavesurfer 추가 |
| `square-admin/src/views/recordings/__tests__/*` | 테스트 |

## 8. Verification plan

- Backend codegen 순서: (1) `bin-openapi-manager`에서 `go generate ./...`(모델), (2) **`bin-api-manager`에서 `go generate ./...`(서버 인터페이스 스텁 `gens/openapi_server/gen.go` + redoc) — 백엔드 리뷰 2**, (3) `bin-storage-manager`에서 `go generate ./...`(신규 인터페이스 mock 재생성 — 백엔드 리뷰 3).
- Backend: 각 서비스에서 `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run` (bin-openapi-manager → bin-storage-manager → bin-api-manager 순). peak 계산 유닛테스트(고정 wav fixture → 알려진 peak). wav 헤더 파싱 실패 시 빈 peaks 폴백 테스트(graceful degrade). 캐시 hit/miss 테스트. **라우팅 테스트: `/v1/recordings/{id}/peaks`가 신규 핸들러로, `/v1/recordings/{id}`가 기존 핸들러로 분기되는지(catch-all 앵커링 회귀 방지)**. servicehandler 권한 테스트(Admin/Manager 허용, 그 외 거부). confbridge 단일 파일이 direction 빈값으로 처리되는지 테스트.
- Frontend: `npm test` 통과(baseline 대비 신규 실패 0). **`npm run build` — wavesurfer v7 ESM 패키지가 react-scripts 5.0.1 webpack에서 정상 resolve/build되는지 확인(프론트 리뷰 5, CJS fallback 존재하나 실빌드 확인 필수)**. wavesurfer `peaks`+`duration` 주입 시 wav fetch 미발생 확인(네트워크 mock).
- 시각 검증 게이트(필수): 워크트리 프로덕션 빌드 + serve, Playwright로 recordings_detail 렌더 스크린샷, 파형 실제 표시 + 재생 컨트롤 확인. call(In/Out 2파일) + confbridge(단일) 레이아웃 각각 확인.
- 비용 안전: 실제 통화 생성 없이, 기존 완료 녹음으로 검증(CLAUDE.md 비용 규칙 준수).

## 9. Rollout / risk

- 선행 의존성: 백엔드 PR이 먼저 머지되어야 프론트가 실데이터로 검증 가능.
- R(신규 의존성): wav 디코딩 라이브러리 — 경량·표준 PCM만. Asterisk wav 헤더 호환 구현 시 실검증. wavesurfer v7 ESM/CRA 빌드는 §8 게이트로 확인.
- R(첫 요청 지연): 긴 녹음 첫 playfiles에서 서버 wav 디코딩. 스트리밍 디코딩 + Redis 캐시로 완화. 실측 지연이 문제로 확인되면 사전계산(§5.1 B) 후속 검토.
- R(캐시 스탬피드): 긴 녹음 첫 요청 동시 다발 시 중복 디코딩. 정합성 문제 없음(idempotent). 실측 신호 확인 시 singleflight 도입(현재 유보, 오버엔지니어링 지양).
- R(peak 정확도): 다운샘플 버킷 수(1000) 고정. 매우 긴 녹음도 고정 길이 배열이라 프론트 부담 일정.

## 10. Open questions (설계 리뷰에서 확정)
- OQ1. peak 배열 버킷 수(해상도) 기본값. 1000 제안.
- OQ2. storage-manager peak RPC 경로/도메인 타입 네이밍(`/v1/recordings/{id}/peaks`).
- OQ3. wav 디코딩 라이브러리 최종 선택(go-audio/wav 후보).
- OQ4. uri_download 만료 임박 재발급 임계 기준(§5.4 1차 방어선).
- OQ5. In/Out UI 최종(나란히 2 vs 토글). v1 나란히 2 제안. **동기/믹스 재생(In+Out을 시간축 정렬해 대화로 듣기)은 v1 비포함, 후속(프론트 후속 1)**. v1은 방향별 독립 재생.

## Iter-2 review response summary

백엔드 아키텍처 리뷰: VERDICT APPROVED (iter-1 6건 전부 정확 반영 확인, 블로킹 없음). 비차단 후속 2건 반영:
- [수정] peaks RPC 응답을 filename 키 맵 `{filename:{peaks,duration}}`으로 명문화, FileList와 filename 조인(순서 의존 오정렬 방지). (§5.3)
- [수정] duration을 peaks와 동일 맵 구조에 포함(권위 출처 일원화). (§5.3)

프론트/제품/계약 리뷰: VERDICT APPROVED (iter-1 5건 전부 정확 반영 확인, 블로킹 없음). 비차단 후속 4건 반영:
- [수정 1] 동기/믹스 재생 v1 비포함 명기. (OQ5)
- [수정 2] duration 라벨 혼선 방지(Call duration vs File duration). (§6)
- [수정 3] status prop 계약(부모가 recordingStatus 전달, 카드 재조회 안 함). (§6)
- [수정 4] playfiles GET의 URI 재발급 write 부수효과/멱등성 문서화. (§5.4)

## Iter-1 review response summary

백엔드 아키텍처 리뷰 (CHANGES_REQUESTED 6건):
- [수정 1 BLOCKING] storage-manager catch-all 라우팅 충돌: 기존 `regV1RecordingsID` 앵커링 + 신규 peaks 케이스 우선 배치. (§7, §8)
- [수정 2 BLOCKING] api-manager 자체 gens(`gens/openapi_server/gen.go`) 재생성 단계 추가. (§7, §8)
- [수정 3] storage-manager storagehandler/filehandler mock 재생성 명시. (§7)
- [수정 4] peak 계산 실패 시 빈 peaks 폴백 + 재생 경로 유지(graceful degrade). (§5.2, §5.3)
- [수정 5] 캐시 스탬피드 처리 방침(유보, singleflight 후속) 리스크 추가. (§5.2, §9)
- [수정 6] DTO 네이밍 `CallManagerRecordingPlayfile` → `RecordingPlayfile`(api-manager 합성 DTO). (§5.3, §7)

프론트/제품/계약 리뷰 (CHANGES_REQUESTED 5건):
- [수정 1 필수] confbridge `_in` 단일 파일 오태깅 방지: direction은 reference_type=call + 2파일일 때만 부여. (§5.3, §6)
- [수정 2 필수] R3b 만료 폴백 재기술: `<audio>`는 403 감지 불가 → tm_download_expire 시간제어 + error/stalled 이벤트 기반. (§5.4)
- [수정 3] direction 데드브랜치 정리: 실제 파일명 규약(call 2파일 / confbridge _in 단일)에 맞게 규칙 단순화. (§5.3)
- [수정 4] duration 권위 출처: 서버가 파일별 wav 헤더에서 산출, 응답 스키마에 duration 필드 추가. (§5.3)
- [수정 5] wavesurfer v7 ESM/CRA 빌드 확인 게이트 명문화. (§8)
