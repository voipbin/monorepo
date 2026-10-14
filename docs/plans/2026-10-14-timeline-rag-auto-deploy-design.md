# Auto deploy for bin-timeline-manager and bin-rag-manager (VOIP-1595)

Follows `2026-10-14-main-auto-deploy-phase2-design.md`. Those two services were kept out of the auto-deploy list because they apply database migrations at startup. They now deploy automatically as well; the human approval click is replaced by a stronger review of database schema changes (see F3).

## Findings (measured 2026-10-14, read-only)

- F1. Both workflows in `config_work.yml` still have the approval-gated shape `build-approval -> test -> build -> deploy` (deploy job `<svc>-deploy` with `deploy_filters`, no `serial-group`, no stale-deploy guard).
- F2. Newest deployed commit per service (Komodo stack image tag): bin-timeline-manager `d82c03073`, bin-rag-manager `ec3d0790e`. Commits that touch each service directory after that revision on `main`: 0 for both, and 0 of them touch `<svc>/migrations`. Both services also build the shared module `bin-common-handler` (`replace` in `go.mod`): 0 new commits for bin-timeline-manager, 9 for bin-rag-manager (2026-09-27 to 2026-10-10: mostly new request-handler client methods for the AI builder and contact layering fixes, 890 lines added and 33 changed in 22 files, all merged to `main` with their own tests). bin-rag-manager only uses `NewRequestHandler` and the handler type, and does not reference the changed methods by name, so its first automatic deploy rebuilds against that shared code without a behavior change we know of. This is exactly what an approval click on today's `main` would deploy; so the first deploy should not be described as carrying no new code. Re-check both ranges (service directory and shared module) immediately before merge.
- F3. Both services run `migrate.Up` at startup, so a merge that adds a migration becomes a production schema change right after the merge. That risk is handled by the schema change verification described in `CLAUDE.md` (written impact analysis, independent re-verification, upgrade and downgrade run on a scratch database, destructive operations stay with a human, and a check for earlier unapplied migrations), not by an approval click.
- F4. `check-auto-deploy-config.rb` hard-codes both services in `DENY`, and `check-auto-deploy-config.bats` uses `bin-timeline-manager` as its approval-kept example in about ten cases.
- F5. Old `on_hold` approval workflows of these services exist in older main pipelines (2 for bin-timeline-manager within the 90 day window of the API). Approving one later would run the old, unguarded job definitions for its old revision.

## Change

1. `config_work.yml`: convert both workflows to the pilot shape used by the other auto-deploy services (`build-approval` and `<svc>-test` use `non_main_filters`, `<svc>-test-auto` uses `deploy_filters`, build requires both, deploy uses `deploy_filters` plus `serial-group`), and add `DEPLOY_REQUIRE_LATEST_FOR_PATHS="<svc>"` to each deploy command. Nothing else in the file changes (verified by parsing both versions and comparing every workflow and job).
2. `auto-deploy-services.txt`: add both names, keep the list sorted, and drop them from the never-allowed comment.
3. `check-auto-deploy-config.rb`: remove both names from `DENY`. The remaining deny list is `bin-trigger-sender`, `voip-asterisk-proxy`, `voip-rtpengine-proxy`, `voip-kamailio-proxy`.
4. `CLAUDE.md`: the human-confirmation paragraph now matches the new behavior (destructive or ambiguous schema changes and `monorepo-monitoring` stay with a human; other migration PRs are merged by QA after the strengthened verification; production apply of `bin-dbscheme-manager` stays manual).
5. `check-auto-deploy-config.bats`: no real service remains approval-kept in the standard shape, so the approval-kept cases run on a fixture workflow (`tests/fixtures/keep-workflow.yml`, `keep-deploy-job.yml`, inserted by `add_keep_fixture`). Two cases are added: removing either service from the allowlist fails, and removing either guard line fails.

## Out of scope

- Migration logic of the two services and the production apply step of `bin-dbscheme-manager` (a separate design).
- The four services that stay approval-kept.

## Risks and mitigations

- First automatic deploy ships unexpected code: F2 shows 0 pending service commits and, for bin-rag-manager, 9 shared-module commits that do not touch anything it calls. Re-check immediately before merge; any change in either range must be listed and accepted.
- A later migration reaches production without a human: covered by the schema change verification in `CLAUDE.md`; destructive or ambiguous changes still need a human.
- Stale old `on_hold` workflows: cancel them right after the merge (F5).
- Rollback of this change: remove the two names from the allowlist and revert the workflow blocks. Later pipelines are approval-gated again; a pipeline already running is not stopped.
- Rolling back a bad deployment: redeploying an older image restores the code only. The schema stays at the newer revision because the down scripts are never run automatically, so the older image has to work with the newer schema. This is why the verification requires compatibility with the previous code in both directions.
- Failed or interrupted migration: golang-migrate marks the version dirty and the service cannot start until it is repaired by a human (fix the schema by hand, then force the version). The deploy script's running check (three consecutive running samples after Komodo reports the deploy complete) catches a crash loop that begins at startup; a failure that happens later than that window is not detected by the pipeline and is found by the post-merge verification and service health checks.

## Verification plan

1. `bats .circleci/tests/*.bats` and the checker pass; mutations of the new rules fail as expected.
2. A branch pipeline on a throwaway branch that touches one of the two service directories shows `build-approval` on hold, `test` and `build` blocked, and no `test-auto` or `deploy` job (no approval is clicked, the branch is deleted).
3. After the merge: cancel old `on_hold` workflows; the first merge of each service shows `test-auto`, `build`, `deploy` all success and the stack image tag equals the merge revision.
