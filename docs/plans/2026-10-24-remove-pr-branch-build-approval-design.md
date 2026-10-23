# Remove the build-approval gate on PR branch pipelines (VOIP-1645)

Follows `2026-10-13-main-auto-deploy-design.md`, `2026-10-14-main-auto-deploy-phase2-design.md`, `2026-10-14-timeline-rag-auto-deploy-design.md` and `2026-10-15-auto-run-verify-workflows-design.md`. Those changes removed the approval click on `main` for 35 service workflows but kept `build-approval` on every other branch. This change removes it there too, so a push to a PR branch runs test and build without a click.

## Findings (measured 2026-10-24, read-only, origin/main 4d38975de)

- F1. 35 of the 42 service workflows (33 in `auto-deploy-services.txt`, 2 in `auto-run-services.txt`) carry a `build-approval` job with `non_main_filters`. `<svc>-test` requires it. The other 7 workflows keep an unfiltered `build-approval` that also gates `main`; they are not part of this change.
- F2. The gate was introduced in `639271c76` (credit optimization, 2026-02-05, about 252,000 credits per month at the time) with the stated goal "zero credits until approved". The design of that commit names no registry or security reason. Its rollback plan says to revert the approval-first flow if it causes problems.
- F3. `<svc>-build` pushes `<repo>:$CIRCLE_SHA1` on every branch and `:latest` only when `CIRCLE_BRANCH == main`. `<svc>-deploy` is `only: main`. `<svc>-build` runs with the `production` context on every branch. `<svc>-test` has no context.
- F4. The pipeline config is loaded from the pushed branch (dynamic config). Without the click, a branch that edits `.circleci/` runs its own edited jobs, including the production context of `build`, without a review step. Until now the click was the point where a person looked at the `.circleci` diff of the branch. The project does not build fork PRs and does not pass secrets to forks (project settings: `build_fork_prs=false`, `forks_receive_secret_env_vars=false`), so the exposure is limited to people with push access to the repository.
- F5. CircleCI Insights, last 30 days: 403 non-main pipelines, 379 finished non-main service workflow runs (about 14,000 credits, 42 credits per successful run, all of them approved ones), 408 main runs (about 16,900 credits). Workflows that waited at the gate and were never approved are not counted by Insights, so the extra load cannot be read directly. In a sample of 7 runs, `test` took about 250 s and `build` about 80 s.
- F6. The project setting "auto-cancel redundant workflows" is off (`autocancel_builds=false`). Today that is harmless, because a superseded push usually sits at the gate. Without the gate, every push of a branch runs the full workflow even if the next push arrives a minute later.

## Change

1. `config_work.yml`: in the 35 workflows delete the `build-approval` job and the `requires: [build-approval]` of `<svc>-test` (auto-run: of the verify job). Everything else stays as it is: `<svc>-test` keeps `non_main_filters`, `<svc>-test-auto` keeps `deploy_filters`, `build` still requires both (the filtered-out one is skipped, which is the mechanism in use today), `deploy` is unchanged. Update the header comment.
2. `check-auto-deploy-config.rb`: the auto-deploy shape becomes exactly `<svc>-test`, `<svc>-test-auto`, `<svc>-build`, `<svc>-deploy`. The auto-run shape becomes exactly the verify job and its `-auto` copy. Both reject a `build-approval` job and any `requires` on the non-main verify job. The approval-kept rules are unchanged.
3. `check-auto-deploy-config.bats` and `fixtures/`: update the cases that expect `build-approval` in the auto shapes; add cases that fail when an approval job or a `requires` comes back, and keep every approval-kept case.
4. Not changed: the 7 approval-kept workflows, `deploy_filters`, `<svc>-test-auto`, the Komodo deploy scripts, `config.yml` path filtering.

Why keep the `test` / `test-auto` split instead of one unfiltered `test`: the split is how the review tooling tells an auto-deploy service from an approval-kept one (`<svc>-test-auto` present), the checker and bats already pin it, and the diff stays a pure deletion of one job and one `requires` line per workflow. One unfiltered `test` would also change what runs on `main`.

## Cost

- Expected rise: every push to a branch that touches a mapped path now runs the full workflow. From F5 the sustained load is bounded by pushes, not by clicks. Assumption: unapproved workflows are 0.5 to 1 times the approved ones, which gives roughly +7,000 to +14,000 credits per month (current total about 31,000). This is an estimate, not a measurement.
- Registry: one `:$CIRCLE_SHA1` image per service per push. No retention policy has been checked.
- Measurement after merge: compare the 30 days after the merge with the 30 days before (non-main runs and credits per workflow). If the rise is above the estimate, the first levers are the push discipline below and auto-cancel.
- Companion setting (not a repository change, decision needed): enable "auto-cancel redundant workflows" in the CircleCI project. It cancels older running workflows of the same branch when a newer push arrives, which removes the waste of F6. It is a project setting and is applied separately.
- Push discipline: a branch should be pushed once, after local test, local checker run and review are finished. Work that can be done before the push (tests, the checker, `circleci config validate` and `process`, reviews) is done locally. Repeated pushes are the main cost driver after this change.

## Risks and mitigations

- F4 above: a branch can run edited jobs with the production context without a review step. Accepted risk (decision of the maintainers), limited to people with push access. Mitigations that remain: `deploy` stays `only: main`, `:latest` stays main-only, forks receive no secrets, review of any `.circleci` change before merge, and the branch-name and commit evidence in the pipeline.
- A broken `.circleci` change now runs at once. Local `circleci config validate`/`process` and the bats suite run before the push.
- Rollback: revert the commit. The shape returns to the approval-gated one and the checker returns with it. Already spent credits and pushed `:$CIRCLE_SHA1` images are not recoverable.

## Out of scope

- The 7 approval-kept workflows (they also gate `main`).
- Rewriting `build-approval` mentions in service READMEs: touching `bin-*/` files arms that service's workflow, which would now run test and build on this very PR. The statements there become outdated; they are fixed whenever each README is next edited.
- Any change to review or approval rules outside this repository.

## Verification plan (all before the single push)

1. `bats .circleci/tests/*.bats` in the same image as CI (`cimg/base`, ruby 3.0, bats 1.2) passes, and the checker passes on the real config.
2. Mutation sweep: each new checker rule fails when it is switched off (`scripts/checker-rule-mutation.py`); table recorded for review.
3. `circleci config validate` and `circleci config process` on the edited `config_work.yml` (docker `circleci/circleci-cli`); processed config shows no `build-approval` job in the 35 workflows and the 7 kept workflows byte-identical to origin/main.
4. Structural diff: parse old and new YAML, confirm the only differences are the removed approval job and the removed `requires` entries in the 35 workflows (script, recorded in the PR).
5. Design review loop (2 consecutive approvals) and PR review loop (3 rounds, 2 consecutive approvals) are finished before the push.
6. After the push: this change touches only `.circleci/` and `docs/`, so the pipeline arms `shell-tests` and `lint-config-check` only. After the merge, the first merge that touches a service shows `test-auto`, `build`, `deploy` running by themselves (unchanged mechanism), and the first non-main push shows `test` and `build` running without a click.
