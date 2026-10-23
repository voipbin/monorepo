# Remove the build-approval gate on PR branch pipelines (VOIP-1645)

Follows `2026-10-13-main-auto-deploy-design.md`, `2026-10-14-main-auto-deploy-phase2-design.md`, `2026-10-14-timeline-rag-auto-deploy-design.md` and `2026-10-15-auto-run-verify-workflows-design.md`. Those changes removed the approval click on `main` for 35 service workflows but kept `build-approval` on every other branch. This change removes it there too, so a push to a PR branch runs test and build without a click.

## Findings (measured 2026-10-24, read-only, origin/main 4d38975de)

- F1. `config_work.yml` has 44 workflows: 40 service workflows and 4 non-service ones (`shell-tests`, `lint-config-check`, `check-test-conventions`, `check-structure-conventions`). 39 workflows contain a job named `build-approval`.
  - 35 carry it with `non_main_filters` (`ignore: main`): 33 in `auto-deploy-services.txt` and 2 in `auto-run-services.txt`. These are the target of this change.
  - 4 carry an unfiltered `build-approval` that also gates `main`: `bin-trigger-sender`, `voip-asterisk-proxy`, `voip-rtpengine-proxy`, `voip-kamailio-proxy`. They are exactly the checker's `DENY` list and are not changed.
  - 5 have no `build-approval`: `bin-dbscheme-manager` (its only approval is the main-only `migration-applied-checkpoint`) and the 4 non-service workflows.
- F2. The gate was introduced in `639271c76` (credit optimization, 2026-02-05, about 252,000 credits per month at the time) with the stated goal "zero credits until approved". That commit's design names no registry or security reason, and its rollback plan says to revert the approval-first flow if it causes problems.
- F3. `<svc>-build` pushes `<repo>:$CIRCLE_SHA1` on every branch and `:latest` only when `CIRCLE_BRANCH == main`. `<svc>-deploy` is `only: main`. `<svc>-build` and `<svc>-deploy` run with the `production` context on every branch they run on. `<svc>-test` has no context. The `docker-build` command only uses `CC_DOCKERHUB_*`.
- F4. The pipeline config (setup config and continuation config) is loaded from the pushed commit. A branch can therefore edit any job, filter or `requires` of its own pipeline, including deleting the approval job or the `only: main` filter of `deploy`. The approval gate has never been a technical security boundary (the header comment of `config_work.yml` already says `deploy_filters` is "Not a security boundary"). What the click gave was a human look at the diff of that branch (including its `.circleci` changes) before anything ran. This change removes that human look; it does not add a new way for a malicious pusher to reach production secrets, because editing the config already was one.
- F5. The `production` context holds the production secrets (Komodo API credentials, deploy SSH key, database DSNs, third-party API keys; names only were read) and its only restriction is the group "All members" (no project or branch expression restriction). Fork PR builds are off and forks receive no secrets (`build_fork_prs=false`, `forks_receive_secret_env_vars=false`). So any organization member who can push a branch can run a job with the full `production` context, with or without this change. Narrowing that (a separate context with only the Docker Hub credentials for `build`, and a branch-expression restriction on `production`) would be a real boundary, but it is a context change outside the repository and is not part of this change.
- F6. CircleCI Insights, last 30 days (GET only, service workflows only, de-duplicated by workflow id; two independent measurements agree within about 15% on counts and a few percent on credits): non-main about 320 to 380 finished runs and about 13,300 to 14,000 credits, average about 39 to 42 credits per successful run; main about 384 to 408 runs and about 16,500 to 16,900 credits. Non-main runs include about 41 with status `canceled` and 0 credits (workflows that waited for the gate and were cancelled), so part of the waiting load is visible. Workflows that were never approved and never cancelled are not in Insights. In a sample of 7 runs, `test` took about 250 s and `build` about 80 s.
- F7. The project setting "auto-cancel redundant workflows" is off (`autocancel_builds=false`). Today that is harmless, because a superseded push usually sits at the gate. Without the gate, every push of a branch runs the full workflow even if the next push arrives a minute later. CircleCI applies this setting to non-default branches only, so main deploy serialization is not affected (to be confirmed on the first branch after enabling).
- F8. `main` already runs the target shape. In recent main pipelines the service workflows show only `test-auto`, `build`, `serial-start`, `deploy`, `serial-end`, all successful, while `build-approval` and `test` are filtered out and `build` still runs with `requires: [test, test-auto]`. The non-main shape after this change is the mirror image of that.

## Change

1. `config_work.yml`: in the 35 workflows delete the `build-approval` job and the `requires: [build-approval]` of `<svc>-test` (auto-run: of the verify job). Everything else stays: `<svc>-test` keeps `non_main_filters`, `<svc>-test-auto` keeps `deploy_filters`, `build` still requires both (the filtered-out one is skipped, as F8 shows on main), `deploy` is unchanged. Update the header comment.
2. `check-auto-deploy-config.rb`:
   - Auto-deploy shape: exactly `<svc>-test`, `<svc>-test-auto`, `<svc>-build`, `<svc>-deploy`. A job named `build-approval` is reported as its own explicit error before the size check, so a restored approval is not only a size mismatch. `<svc>-test` must not have a `requires` key at all (so `requires: []` also fails, same as the existing `test-auto` rule).
   - Auto-run shape: exactly the verify job and its `-auto` copy, with the same two rules (explicit `build-approval` error, no `requires` key on the non-main verify job). The size condition and the `bases` computation change accordingly and the error message becomes "jobs must be exactly one verify job and its -auto copy".
   - Any other approval-type job in these workflows is rejected by the exact job-id set.
   - Remove the auto-shape rules about `build-approval` type and filters (dead code after the change). The approval-kept rules (`check_kept_workflow`) are not touched.
3. `check-auto-deploy-config.bats` and `fixtures/`:
   - Delete the auto-shape cases that mutate a `build-approval` that no longer exists (about: filter missing, `only: main` instead of ignore, test losing the requires, type no longer approval, the two auto-run approval cases) and update the message-dependent auto-run case.
   - Keep every approval-kept case (the kept fixture cases and the four cases that keep an unfiltered approval).
   - Add cases: approval job restored in an auto-deploy workflow, `requires` restored on `<svc>-test` (also `requires: []`), approval restored in an auto-run workflow, `requires` restored on the auto-run verify job, an approval-type job under another name.
4. Not changed: the 5 workflows without a gate, the 4 approval-kept workflows, `deploy_filters`, `<svc>-test-auto`, the Komodo deploy scripts, `config.yml` path filtering.

Why keep the `test` / `test-auto` split instead of one unfiltered `test`: the review tooling distinguishes an auto-deploy service from an approval-kept one by the presence of `<svc>-test-auto`, the checker and bats already pin it, F8 shows it works on main, and the diff stays a pure deletion of one job and one `requires` line per workflow. One unfiltered `test` would also change what runs on `main`.

## Cost

- The rise cannot be derived from the unapproved-but-pending workflows, because they are not measurable. Upper bound scenario: if as many pushes were waiting at the gate as were approved, non-main load doubles, that is about +13,000 to +14,000 credits per month on a current total of about 30,000 (the same service-workflow total as in F6). The real number is lower to the extent that waiting workflows were abandoned (about 41 of about 360 observed ended `canceled`). This is a bound, not an estimate.
- The multiplier is pushes times mapped services, not approvals. With `autocancel_builds=false` rapid pushes each run to the end.
- Registry: one `:$CIRCLE_SHA1` image per service per push. Docker Hub retention for these repositories has not been checked (known unknown).
- Dependabot: `.github/dependabot.yml` exists; dependency update branches that map to a service now run test and build without a click. Their effect on cost is part of the post-merge measurement.
- Measurement after merge: compare the 30 days after the merge with the 30 days before (non-main runs and credits per workflow, counted the same way as F6).
- Companion setting (not a repository change, decision needed): enable "auto-cancel redundant workflows" in the CircleCI project before or together with the merge. It is the cheapest cost lever and is recommended as a precondition.
- Push discipline: a branch is pushed once, after local test, local checker run and reviews are finished. Everything that can be done before the push (tests, the checker, `circleci config validate` and `process`, the structural diff, the mutation sweep, the reviews) is done locally. Repeated pushes become the main cost driver after this change.

## Risks and mitigations

- Human look removed (F4, F5): a branch with an unreviewed change now runs test and build with the `production` context at once, where before a person decided at the click. Real exposure to a malicious pusher is unchanged (F4), accidental or unreviewed runs are not caught any more. Accepted by the decision-maker for this change. Behaviors that remain in the normal config (not defenses against a branch that edits its config): `deploy` is `only: main`, `:latest` is main-only, forks get no secrets. A real boundary would need the context change described in F5 and is a separate decision.
- A broken `.circleci` change now runs at once. Local `circleci config validate`/`process` and the bats suite run before the push.
- Rollback: revert the commit. The shape returns to the approval-gated one and the checker returns with it. Already spent credits and pushed `:$CIRCLE_SHA1` images are not recoverable.

## Out of scope

- The 4 approval-kept workflows and the 5 workflows without a gate (F1).
- Rewriting `build-approval` mentions in service READMEs (16 files; the ones for the 4 approval-kept services stay correct): touching `bin-*/` files arms that service's workflow, which would run test and build on this very PR. The statements in the other READMEs become outdated and are fixed whenever each README is next edited.
- The `production` context restriction and a Docker-Hub-only context for `build` (F5).
- Any change to review or approval rules outside this repository.

## Verification plan (all before the single push)

1. `bats .circleci/tests/*.bats` passes on the host and in the CI image (`cimg/base:2024.02`, ruby 3.0, bats 1.2); results are recorded separately. The checker passes on the real config.
2. Mutation sweep with `checker-rule-mutation.py` on the host bats: one entry per new or changed checker rule in `mutations.json`, no unprobed rule; table recorded for review.
3. `circleci config validate` and `circleci config process` (docker `circleci/circleci-cli`) on the edited config_work.yml.
4. Structural diff script (parsed old versus new YAML): exactly the 35 target workflows differ, each only by the removed `build-approval` job and the removed `requires` entry; the other 9 workflows are identical; anchors, `jobs:`, `commands:`, parameters and every other top-level key are identical. Script and output are recorded in the PR description.
5. Design review loop (2 consecutive approvals) and PR review loop (3 rounds, 2 consecutive approvals) are finished before the push.
6. After the push: this change touches only `.circleci/` and `docs/`, so the pipeline arms `shell-tests` and `lint-config-check` only (plus the convention checks that run on every pipeline); no service workflow runs. After the merge, the first merge that touches a service shows `test-auto`, `build`, `deploy` running by themselves (unchanged mechanism, F8) and the first non-main push that touches a service shows `test` and `build` running without a click.
