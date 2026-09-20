# Recording Inline Playback + Waveform — Issue Analysis

Status: Analysis APPROVED (2 consecutive rounds, iter-4/iter-5)
Date: 2026-09-19
Author: CPO (Hermes)

## 1. Request

square-admin 레코딩 화면에서 "바로 플레이"(인라인 오디오 재생)와 함께 음성 파형(waveform) 그래프를 함께 볼 수 있게 한다.

Locked decisions (대표님 확정, 2026-09-19):
- D1. 재생 UI 위치: 레코딩 상세(detail) 페이지 인라인 플레이어. (list 행 재생 아님)
- D2. 재생용 wav URL 확보 방식: 신규 백엔드 엔드포인트 `GET /recordings/{id}/playfiles` (개별 파일 + signed URL 목록).
- D3. 파형: 인라인 플레이어에 음성 파형 시각화 포함.

## 2. Current state (실측)

### 2.1 Frontend
- `square-admin/src/views/recordings/recordings_detail.js`: Download 버튼만 존재. 재생 UI 없음.
  - Download는 `recordingfiles/{id}`를 zip으로 저장(`ProviderDownload`, 파일명 `{id}.{format|zip}`).
- `provider.js`: `Download()`(saveAs로 브라우저 저장, line 254), `FetchBinary()`(ArrayBuffer 반환, 저장 다이얼로그 없음, line 291) 존재.
- 오디오/파형 라이브러리 없음(`package.json`에 wavesurfer/howler/audio 계열 없음). React 18.2.

### 2.2 Backend
- `GET /recordings/{id}` → `CallManagerRecording` (재생 URL 필드 없음).
- `GET /recordingfiles/{id}` → `RecordingfileGet` → `StorageV1CompressfileCreate(...)` → **항상 ZIP** signed URL 반환. 브라우저 인라인 재생 불가.
  - 파일: `bin-api-manager/pkg/servicehandler/recordingfile.go:46`.
- Recording 포맷: `wav` 단일 (`bin-call-manager/models/recording/recording.go:63`, `FormatWAV`).
- 통화 녹음은 방향별 2개 wav 생성: `_in.wav`, `_out.wav`
  (`bin-call-manager/pkg/recordinghandler/recording.go:73` in/out 루프, line 75 `%s_%s.%s` 포맷). 그래서 다운로드가 zip.
  - conference/confbridge 녹음은 단일 파일 경로도 존재(`recording.go:182`, `%s_in` 단일).
- 개별 wav는 storage File로 등록됨:
  `bin-call-manager/pkg/recordinghandler/stop.go:98`
  `StorageV1FileCreate(..., smfile.ReferenceTypeRecording, r.ID, smfile.TypeRecording, "recording file", "", filename, ...)`.
- storage File 엔티티는 개별 signed URL을 가짐:
  - `bin-storage-manager/models/file/webhook.go`: `uri_download`, `tm_download_expire`, `reference_type`, `reference_id`, `filename`, `filesize`.
  - `ReferenceTypeRecording = "recording"` (`models/file/main.go:47`), `File.ReferenceID = recording.ID`.
- storage File 조회 RPC 이미 존재:
  - `StorageV1FileList(ctx, pageToken, pageSize, filters map[smfile.Field]any)` (`bin-common-handler/pkg/requesthandler/storage_files.go:120`).
  - filters의 `reference_type`/`reference_id` 지원 근거: `bin-storage-manager/models/file/field.go:12-13` (`FieldReferenceType`/`FieldReferenceID` 상수) + `bin-storage-manager/pkg/dbhandler/files.go:179` (`ApplyFields(sb, filters)`).
  - `StorageV1FileGet`(`storage_files.go:144`), `StorageV1FileDownloadURIRefresh`(`storage_files.go:182`) 존재.

### 2.3 결론적 사실
개별 재생 wav의 signed URL은 **이미 storage File에 존재**하고, reference_id(=recording id)로 **기존 RPC(StorageV1FileList)로 조회 가능**하다. 신규 엔드포인트는 이 RPC를 recording 스코프로 감싸 노출하기만 하면 되고, storage-manager/call-manager 신규 로직은 불필요하다.

## 3. Feasibility of "바로 플레이"

- 재생 자체: wav 단일 포맷이라 브라우저 재생 가능.
- 소스 URL: `StorageV1FileList(reference_type=recording, reference_id=id)`가 반환하는 각 File의 `uri_download`.
- 방향(in/out): 통화 녹음은 2개 파일 → 플레이어를 2개(In/Out) 또는 방향 토글로 처리(설계에서 확정). conference는 1개.

### 3.1 재생/파형 리스크

- **R1. CORS (파형 fetch 한정, BLOCKING 후보)**: 파형은 브라우저가 GCS signed URL의 wav 바이트를 cross-origin fetch 해야 peak 계산 가능. GCS 버킷(`gcp_bucket_name_media`)에 CORS 허용이 없으면 파형 fetch 실패. **단, 재생(네이티브 `<audio src>` 스트리밍)은 CORS 무관하게 동작**(§5 설계 방향의 핵심). → 파형 소스 전략은 설계 단계 실검증.
- **R2. disposition=attachment**: recording File은 `filename`이 세팅되어(`stop.go:98`) signed URL에 `response-content-disposition=attachment`가 붙는다. `<audio>` 재생과 fetch 바이트에는 무해(다운로드 트리거 아님). 신규 엔드포인트가 이 URL을 그대로 내려도 재생 목적에는 문제 없음.
- **R3. signed URL 만료**: File의 `tm_download_expire` 존재. 재생 시점 만료 시 실패 → `StorageV1FileDownloadURIRefresh` 또는 엔드포인트 호출 시점 재발급 고려(설계).
- **R3b. 긴 녹음 재생 중 signed URL 중도 만료 (R3와 별개)**: 네이티브 `<audio>`는 재생/seek 시 URL을 반복 요청한다. 1시간 녹음을 재생하면 재생 소요시간이 URL TTL을 넘겨 재생 도중이나 일시정지 후 seek 시 403으로 끊길 수 있다. 시작 시점 재발급만으로는 커버되지 않는다. → 설계에서 (a) 녹음 최대 길이를 커버하는 충분한 TTL을 playfiles 응답 시점에 확보하거나 (b) 재생 중 403 감지 시 재발급 후 currentTime 복원. TTL vs 재생 소요시간 비교를 설계 입력으로 명기.
- **R4. 긴 녹음 클라이언트 디코딩 (중대, 파형 한정)**: wavesurfer의 WebAudio 백엔드는 파형 peak 계산을 위해 wav 전체를 브라우저로 fetch + 디코딩한다. VoIPBin wav는 무압축(대략 8kHz/16bit mono 기준 약 1MB/분, 샘플레이트/채널에 따라 증가) → 1시간 통화면 수십~수백 MB. 모바일/저사양에서 로딩 지연·메모리 폭증·크래시 위험. **이 리스크가 파형 소스 전략을 좌우하는 핵심 요인**이며, "재생은 스트리밍, 파형은 크기 임계치 게이트"라는 설계 방향(§5)의 근거다.
- **R5. GCS 이그레스 비용 (운영)**: WebAudio 파형은 재생 스트림과 별개로 wav를 한 번 더 내려받아 GCS 이그레스가 사실상 2배. 긴 녹음에서 셀프호스터 실비용 증가. 임계치 게이트로 상한 관리(§5).
- **R6. 모바일 재생 정책 (운영/UX)**: iOS Safari/모바일 크롬은 사용자 제스처 없는 자동재생 차단, 터치 인터랙션·메모리 제약. audio-only 콘솔이 모바일에서 열릴 수 있으므로 네이티브 컨트롤 기반이 안전.
- **R7. 접근성**: 풀 커스텀 wavesurfer 플레이어는 네이티브 `<audio controls>`와 달리 키보드 조작·ARIA·포커스를 직접 구현해야 함. → 네이티브 컨트롤을 재생 주체로 두고 파형을 시각 보조로 얹는 조합이 접근성·모바일·성능 모두에 유리.
- **R8. direction 도출의 취약성**: storage File에는 direction 필드가 **없음**(확인: `models/file/*.go`에 direction 없음). in/out 구분은 `filename`의 `_in`/`_out` 접미 파싱에 의존할 수밖에 없다(`recording.go:75` 생성 규칙). 파일명 규칙 변경에 취약 → 파싱 실패 시 방향 라벨 없이(파일명 그대로) 표기하는 폴백 필요. 설계에서 명시.

## 4. Scope / 진행 타당성

- 크로스 리포지토리 작업(구조적 필연): 백엔드 monorepo 1 PR + 프론트 monorepo-javascript 1 PR.
- 선행 의존성: 백엔드 엔드포인트가 프론트보다 먼저 존재해야 프론트가 실데이터로 검증 가능.
- 백엔드 변경은 api-manager(+openapi-manager) 한정. storage-manager/call-manager 무변경(기존 RPC 재사용).
- 운영 비용: R5(이그레스 2배)는 파형 임계치 게이트로 상한 관리. 재생 자체는 스트리밍이라 추가 비용 없음.
- 진행 타당성: 유효. 이미 해결된 문제 아님(재생 UI 부재 실측 확인). 요구 명확, 기존 인프라로 저비용 구현 가능.

## 5. Recommended approach (설계 입력)

핵심 설계 방향: **재생과 파형을 분리**한다. 재생은 항상 동작하는 저비용/접근성 경로로, 파형은 조건부 시각 보조로 둔다.

1. 백엔드: `GET /recordings/{id}/playfiles`
   - api-manager servicehandler: recording 소유권/권한 확인(기존 `recordingGet` + `hasPermission`, `PermissionCustomerAdmin|PermissionCustomerManager`) 후 `StorageV1FileList(reference_type=recording, reference_id=id)` 호출, 각 File을 재생용 항목으로 매핑.
   - 응답 항목당 최소: `filename`, `direction`(filename 파싱, 실패 시 빈 값), `uri_download`, `tm_download_expire`, `filesize`.
2. 프론트: recordings_detail에 인라인 플레이어 카드 추가
   - **재생 주체 = 네이티브 `<audio controls src=uri_download>`**: 스트리밍이라 전체 디코딩 불필요, CORS 무관, 모바일/접근성(키보드·ARIA·자동재생 정책) 기본 제공, 이그레스 1배.
   - **파형 = 시각 보조 오버레이(조건부)**: `filesize`가 임계치(설계에서 확정) 이하일 때만 wavesurfer로 파형 렌더. 초과 시 파형 생략 또는 "too large to render waveform" 안내 + 네이티브 재생은 유지.
   - 파일 2개면 In/Out 각각(또는 토글). CORS 검증 결과에 따라 파형 소스 전략 최종 확정(R1).
3. 상태 처리는 §5.3 참조.

### 5.1 오버엔지니어링 판단 (리뷰어 권고 2에 대한 CPO 응답)
분석 리뷰어(Task 2)는 "백엔드 peak 사전계산(옵션 B)"을 기본안으로 권고했다. **부분 수용한다.** B는 긴 녹음 클라이언트 부담(R4)과 이그레스(R5)를 해소하지만, 서버측 wav 디코딩 + peak 저장 + (가능하면) 스키마의 **신규 백엔드 체계 구축**이며, 본 조직의 오버엔지니어링 지양 원칙(실측 신호 없는 선제적 인프라 확장 반대, 기존 도구 재사용 우선)과 충돌한다. 또한 B는 "기존 RPC 재사용으로 저비용"이라는 본 작업의 핵심 이점을 상쇄한다.
대안으로 채택한 "네이티브 스트리밍 재생 + 크기 임계치 게이트 파형"은 신규 백엔드 체계 없이 R4/R5/R6/R7을 동시에 완화한다. B(백엔드 peak 사전계산)는 **트리거 조건(예: 임계치 이하 녹음에서도 파형 로딩이 실측 문제로 확인)이 발생하면 후속 설계**로 남긴다. 이는 열린 질문이 아니라 유보된 후속안이다.

### 5.2 파형 가치 역전 트레이드오프 + D3 조건부 이행 (대표님 의사결정 항목)
임계치 게이트(§5-2)와 CORS(R1)는 파형에 두 가지 제품 비용을 만든다. 게이트 이득(R4/R5/R6/R7 완화)과 나란히 표면화한다.
- **가치 역전**: 스크러빙·구간 탐색 가치가 가장 큰 것은 긴 녹음인데, 임계치 게이트는 정작 긴 녹음에서 파형을 억제하고 짧은 녹음에서만 렌더한다. 즉 파형이 "가장 필요한 곳에 없고 가장 덜 필요한 곳에 있는" 역전이 생긴다.
- **D3 조건부 이행**: Q1(CORS)이 "막히면 파형 보류"로 귀결되면, 대표님이 확정한 D3(파형)가 v1에서 아예 미실현되거나 임계치 이하 짧은 녹음의 일부 케이스에서만 보일 수 있다. 즉 D3의 실현이 CORS·임계치에 조건부다.
- **대표님 결정 필요**: (i) 짧은 녹음 한정 파형이라도 실사용 가치가 있어 v1에 포함할지, (ii) 긴 녹음 파형이 핵심이라면 §5.1의 유보안 B(백엔드 peak 사전계산)를 v1 스코프로 끌어올릴지(오버엔지니어링 트레이드오프 재검토), (iii) CORS 실검증 결과를 보고 결정할지. 설계 단계 진입 전 이 항목을 확정 대상으로 올린다.

### 5.3 빈/생성중 상태 처리
playfiles가 빈 배열(녹음 진행중/File 미등록/등록 지연)일 때 empty vs generating을 구분한다. `recording.status`를 힌트로 사용(실제 enum: `initiating`/`recording`/`stopping`/`ended`, `bin-call-manager/models/recording/recording.go:52-55`): status가 진행중(`initiating`/`recording`/`stopping`)이면 "recording in progress" 안내 + (선택)폴링/WebSocket 갱신, `ended`인데도 파일이 없으면 empty(또는 등록 지연) 처리.

## 5.4 대표님 결정 (2026-09-19, §5.2 재확정 — 방향 변경)
대표님은 "길이와 무관하게 모든 녹음에서 파형 표시"를 확정했다. 이로써 §5의 "임계치 게이트 파형"은 폐기되고, §5.1에서 유보했던 **옵션 B(백엔드 peak 사전계산)를 v1 스코프로 채택**한다.
- 이 결정은 클라이언트가 wav 원본을 받지 않으므로 R1(CORS)/R4(긴녹음 디코딩)/R5(이그레스 2배)/R6(모바일)을 동시에 소멸시킨다.
- 트레이드오프: 백엔드가 wav 디코딩→peak 계산 필요. 스코프가 api-manager 한정에서 storage-manager로 확대. 신규 wav 디코딩 의존성 1개.
- 실현 가능성(확인): storage-manager가 wav 바이트 접근 가능(`bucketfile.go:102` `NewReader`). File 엔티티에 peak 필드 없음(저장 시 DB 스키마 변경 필요). Go wav 디코딩 라이브러리 vendor에 없음(신규 의존성).
- 설계 기본안(설계 리뷰로 확정): **온디맨드 계산 + 캐시**. api-manager playfiles → storage-manager peak 계산 RPC(Redis 캐시). DB 스키마/call-manager 무변경, 기존 녹음 backfill 불필요. 대안(사전계산+저장: DB 스키마 + 등록 흐름 + backfill)은 설계 §에서 트레이드오프로 비교.
- 재생(네이티브 `<audio>`)은 이 결정과 무관하게 모든 녹음에서 항상 동작(변경 없음).

## 6. Open questions (설계에서 결정)
- Q1. 파형 fetch용 CORS: 버킷 CORS 룰 추가(A)로 임계치 이하 파형을 그릴지, 아니면 CORS가 막히면 파형을 아예 보류할지. 실검증 후 결정. (B=백엔드 peak 사전계산은 §5.1대로 유보된 후속안.)
- Q2. 파형 렌더 크기 임계치(filesize) 구체값.
- Q3. In/Out 2파일 UI: 나란히 2 플레이어 vs 방향 토글 1 플레이어.
- Q4. 신규 엔드포인트 응답 스키마 이름/필드 (openapi 컴포넌트).
- Q5. 재생 URL을 기존 attachment-disposition File URL 재사용 vs 재생 전용(inline) 재발급(R2는 재사용해도 무해로 판단, 확정 필요).

## Iter-1 review response summary

Task 1 (기술 사실 정확성):
- [수정] 2.2의 `filehandler/file.go:38-39` 인용(로그 필드 선언, 필터 근거 아님) → `models/file/field.go:12-13` + `dbhandler/files.go:179`로 교체. (§2.2)

Task 2 (스코프/제품/운영):
- [수정 1] 긴 녹음 클라이언트 디코딩 리스크 추가 + wav 무압축 크기 추정. (§3.1 R4)
- [수정 2] Q1 재작성: peak 사전계산(B)을 §5.1에서 오버엔지니어링 근거로 유보된 후속안으로 명시, 기본안은 네이티브 스트리밍 재생 + 임계치 게이트 파형. (부분 수용, 근거 기술)
- [수정 3] GCS 이그레스 2배 비용 추가. (§3.1 R5, §4)
- [수정 4] 모바일 재생 정책 리스크 추가. (§3.1 R6)
- [수정 5] 접근성 항목 + 네이티브 컨트롤 vs 풀 커스텀 트레이드오프. (§3.1 R7, §5)
- [수정 6] playfiles 빈 배열 empty vs generating 상태 처리 추가. (§5.3)
- [수정 7] direction을 filename 파싱으로 도출하는 취약성 + direction 필드 부재 확인 + 폴백. (§3.1 R8, §5)

## Iter-2 review response summary

Task 1 (기술 사실 정확성): VERDICT APPROVED (수정 요청 없음, iter-1 오인용 교정 및 신규 사실 전수 확인됨).

Task 2 (스코프/제품/운영): CHANGES_REQUESTED 3건 반영:
- [수정 1] §5.3 교차참조 정합화: empty/generating을 독립 §5.3 헤더로 승격, §5 item3은 참조로 변경. (§5.3)
- [수정 2] 파형 가치 역전 + D3 조건부 이행 트레이드오프를 대표님 의사결정 항목으로 표면화. (§5.2)
- [수정 3] 긴 녹음 재생 중 signed URL 중도 만료(재생 소요>TTL, seek/pause 후 403)를 R3와 별개 리스크로 추가. (§3.1 R3b)

## Iter-3 review response summary

Task 2 (스코프/제품/운영): VERDICT APPROVED (iter-2 3건 실질 반영 확인, 신규 블로킹 없음). Non-blocking으로 지적된 Status 헤더 갱신 반영.

Task 1 (기술 사실 정확성): CHANGES_REQUESTED 1건 반영:
- [수정 1] §5.3의 `recording.status` 예시값 "processing/completed"(코드베이스에 없는 값)를 실제 enum `initiating`/`recording`/`stopping`/`ended`(`bin-call-manager/models/recording/recording.go:52-55`)로 교정. (§5.3)
