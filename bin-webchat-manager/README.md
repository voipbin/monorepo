# webchat-manager

## Deployment

bin-webchat-manager deploys via Komodo (VOIP-1347 Tier 1 rollout, following the
VOIP-1342/bin-call-manager pilot pattern) instead of the older SSH +
`versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-webchat-manager/komodo/docker-compose.yml` (git is
  the source of truth for structure; Komodo only executes it on request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes the built
  image tag, then `.circleci/scripts/komodo-api-deploy.sh` pushes the file's
  content to Komodo and triggers a deploy, gated by the
  `bin-webchat-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md)
  (in the monorepo root, not this service's own `docs/`).

Note: bin-webchat-manager does not have a `docs/operations.md` file (only
`docs/plans/`) — see [VOIP-1352](https://voipbin.atlassian.net/browse/VOIP-1352)
for generating the full architecture/operations/domain/dependencies doc
suite that `docs/reference/extractor.sh` produces for other services. Once
that ticket lands, this `## Deployment` section should move to the new
`docs/operations.md` and be removed from here.
