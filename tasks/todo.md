# VOIP-1391 Extension QR Provisioning — TODO

Plan: docs/plans/2026-08-24-voip-1391-extension-qr-provisioning-plan.md (리뷰 2연속 Approval)
Design: docs/plans/2026-08-24-voip-1391-extension-qr-provisioning-design.md (리뷰 2연속 Approval)

## Phase 1: Backend (이 워크트리)

- [ ] 1.1 bin-openapi-manager 스펙 (paths 2건 + 스키마 + go generate)
- [ ] 1.2 bin-api-manager config (rate limit 2키 + public_base_url, env-var-only) + operations.md
- [ ] 1.3 cachehandler ProvisioningTokenSet/Get + miniredis 테스트 + mock
- [ ] 1.4 servicehandler (생성자 cache 주입, TokenCreate, XMLGet, XML 렌더러 + 골든 테스트)
- [ ] 1.5 HTTP 계층 (server 핸들러+테스트, 404 스텁, lib/service 공개 핸들러, main.go 로거/그룹,
      routing/architecture/operations docs)
- [ ] 1.6 RST 문서 (overview/tutorial/struct/quickstart) + Sphinx 클린 빌드 + build force-add
- [ ] 1.7 검증 (5단계 x 2서비스) → 코드 리뷰 루프 (min 3, 2연속 Approval) → 충돌 확인 → PR

## Phase 2: Frontend (monorepo-javascript 워크트리)

- [ ] 2.1 워크트리 + npm install + 테스트 기준선
- [ ] 2.2 ProvisioningQrSection + qrcode.react + 테스트
- [ ] 2.3 extensions_detail 통합 + 테스트
- [ ] 2.4 테스트 게이트 → 빌드 → 코드 리뷰 루프 → 충돌 확인 → PR

## Phase 3: 배포 후 실기기 검증 (머지 후 별도)

- [ ] Linphone 실기기 스캔 → REGISTER 확인, TTL 만료 400, 로그 토큰 미노출, VOIP-1391 코멘트

## Working Notes

- 빌드 순서 함정: 1.1 이후 bin-api-manager 전체 go generate 금지 (1.5에서 수행). 1.2~1.4는 ./pkg/ 스코프.
- 발급 시그니처: *auth.AuthIdentity. XML은 ConvertWebhookMessage 경유 (DomainName 원시 사용 금지).
- 머지는 대표님 명시 승인 후에만. PR 제목 = 브랜치명 = VOIP-1391-extension-qr-provisioning.
