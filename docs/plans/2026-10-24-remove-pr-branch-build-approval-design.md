# Remove the build-approval gate on PR branch pipelines (VOIP-1645)

Follows `2026-10-13-main-auto-deploy-design.md`, `2026-10-14-main-auto-deploy-phase2-design.md`, `2026-10-14-timeline-rag-auto-deploy-design.md` and `2026-10-15-auto-run-verify-workflows-design.md`. Those changes removed the approval click on `main` for 35 service workflows but kept `build-approval` on every other branch. This change removes it there too, so a push to a PR branch runs test and build without a click.

## Background

- `config_work.yml` has 44 workflows: 40 service workflows and 4 non-service ones. 39 of them contain a job named `build-approval`.
  - 35 carry it with `non_main_filters` (`ignore: main`): 33 in `auto-deploy-services.txt` and 2 in `auto-run-services.txt`. These are the target of this change.
  - 4 carry an unfiltered `build-approval` that also gates `main`: `bin-trigger-sender`, `voip-asterisk-proxy`, `voip-rtpengine-proxy`, `voip-kamailio-proxy`. They are exactly the checker's `DENY` list and are not changed.
  - 5 have no `build-approval`: `bin-dbscheme-manager` (its only approval is the main-only `migration-applied-checkpoint`) and the 4 non-service workflows.
- The gate was introduced in `639271c76` (CI credit optimization, 2026-02-05) with the goal "zero credits until approved". That change names no registry or review reason, and its rollback plan says to revert the approval-first flow if it causes problems.
- `<svc>-build` pushes `<repo>:$CIRCLE_SHA1` on every branch and `:latest` only when `CIRCLE_BRANCH == main`. `<svc>-deploy` is `only: main`. `<svc>-test` has no context.
- The approval was the point where a person looked at what a branch was about to run. The header comment of `config_work.yml` already says `deploy_filters` is "Not a security boundary".
- `main` already runs the target shape: in recent main pipelines the service workflows show only `test-auto`, `build` and `deploy`, while `build-approval` and `test` are filtered out and `build` still runs with `requires: [test, test-auto]`. The non-main shape after this change is the mirror image of that.

## Change

1. `config_work.yml`: in the 35 workflows delete the `build-approval` job and the `requires: [build-approval]` of `<svc>-test` (auto-run: of the verify job). Everything else stays: `<svc>-test` keeps `non_main_filters`, `<svc>-test-auto` keeps `deploy_filters`, `build` still requires both (the filtered-out one is skipped, as on main today), `deploy` is unchanged. Update the header comments.
2. `check-auto-deploy-config.rb`:
   - Auto-deploy shape: exactly `<svc>-test`, `<svc>-test-auto`, `<svc>-build`, `<svc>-deploy`. A job named `build-approval` is reported as its own explicit error before the size check. `<svc>-test` must not have a `requires` key at all (so `requires: []` also fails, same as the existing `test-auto` rule).
   - Auto-run shape: exactly the verify job and its `-auto` copy, with the same two rules (explicit `build-approval` error, no `requires` key on the non-main verify job). `bases = ids - autos` and `autos.size == 1 && bases.size == 1` (the former `ids.size == 3` condition is implied by these two counts, so it is dropped rather than kept as a rule no case can probe). The error message becomes "jobs must be exactly one verify job and its -auto copy".
   - A `type` key on any job of an auto-deploy or auto-run workflow is rejected, evaluated before any early return of the size check; otherwise `type: approval` attached to `<svc>-test` or to the verify job would bring the click back without changing the job ids. Any other approval-type job is also rejected by the exact job-id set.
   - The auto-shape rules about the `build-approval` type and filters are removed (dead code). The approval-kept rules (`check_kept_workflow`) are not touched.
   - Comments that mention the old shape are updated (`config_work.yml` header, `.circleci/auto-run-services.txt` line 3).
3. `check-auto-deploy-config.bats`:
   - Delete the 6 cases whose mutation target disappears (approval without filter, `only: main` instead of ignore, test losing the approval requires, approval no longer of type approval, auto-run approval filter, auto-run verify losing the approval requires).
   - Change the expected message of the extra-job case in the auto-run group, and rewrite the three cases that referenced the removed job.
   - Keep every approval-kept case; they pass unchanged.
   - Add cases: approval job restored in an auto-deploy workflow, an approval job under another name, `type: approval` on `<svc>-test`, a `requires` and `requires: []` on `<svc>-test`, approval restored in an auto-run workflow, `type: approval` on the verify job, a `requires` and `requires: []` on the verify job, a duplicated verify job.
4. Documentation: the 16 `bin-*` READMEs that describe the `build-approval` gate (all of them auto-deploy services) are updated in the same PR to say that test and build run without an approval step (where a README also mentions deploy, that deploy runs on main only). Only the sentence about the gate is changed; other statements in those READMEs are not touched. `voip-kamailio-proxy/komodo/README.md` belongs to an approval-kept service and stays correct.
5. Not changed: the 5 workflows without a gate, the 4 approval-kept workflows, `deploy_filters`, `<svc>-test-auto`, the Komodo deploy scripts, `config.yml` path filtering.

Why keep the `test` / `test-auto` split instead of one unfiltered `test`: the review tooling distinguishes an auto-deploy service from an approval-kept one by the presence of `<svc>-test-auto`, the checker and bats already pin it, it works on `main` today, and the diff stays a pure deletion of one job and one `requires` line per workflow. One unfiltered `test` would also change what runs on `main`.

## Cost and operation

- Every push to a branch that touches a mapped path now runs the full workflow where it used to wait for a click. The load grows with pushes times mapped services. It cannot be derived from past data (workflows that waited and were never approved are not recorded), so it is measured after the merge by comparing the 30 days after with the 30 days before.
- One `:$CIRCLE_SHA1` image per service per push is pushed for non-main branches.
- This PR touches files under 16 service directories (the READMEs), which arms those 16 service workflows. On the PR branch their test and build run once without a click (about 16 times a typical test and build run). On the merge to `main`, the 16 services run `test-auto`, `build` and `deploy`, so all 16 are redeployed with unchanged code (containers are recreated). The 16 deploys run in parallel, one stack per service. After the deploys, the services on the call and registration path get an extra health and registration check. The merge is scheduled for a low-traffic window and each service's deploy is verified afterwards.
- A CircleCI project setting that cancels superseded workflows of the same branch is recommended before or together with the merge. It is a project setting, not a repository change.
- A branch is pushed once, after local tests, the checker run, `circleci config validate`/`process`, the structural diff, the mutation sweep and the reviews are done. Repeated pushes become the main cost driver after this change.

## Risks and rollback

- The pre-run look at a branch's pipeline-config changes is gone. Behaviors that remain in the normal config: `deploy` is `only: main` and `:latest` is main-only.
- A broken `.circleci` change now runs at once. Local `circleci config validate`/`process` and the bats suite run before the push.
- Rollback is a revert of the commit: the shape returns to the approval-gated one and the checker returns with it. Credits already spent and images already pushed are not recoverable.

## Out of scope

- The 4 approval-kept workflows and the 5 workflows without a gate.
- Other outdated statements inside the touched READMEs (for example about a removed deploy job); only the gate sentence is changed.
- The `CLAUDE.md` Merge responsibility sentence about PR-branch approvals: it becomes partly outdated and is changed separately, not in this PR.
- Review and approval rules kept outside this repository.

## Verification plan (local steps before the single push)

1. `bats .circleci/tests/*.bats docs/reference/tests/*.bats` passes on the host and in the CI image (`cimg/base:2024.02` with the same `apt-get install bats mawk ruby` step as the `shell-tests` job: bats 1.2.1, ruby 3.0); results are recorded separately. The checker passes on the real config.
2. Mutation sweep with `checker-rule-mutation.py` (host bats): one entry per new or changed checker rule, no unprobed rule. The case that runs the checker against the real tree fails in the scratch copy of the sweep for every entry and is not counted as a probe.
3. `circleci config validate` and `circleci config process --pipeline-parameters` (docker `circleci/circleci-cli`) on the edited config for one auto-deploy and one auto-run workflow.
4. Structural diff script (parsed old from `origin/main` versus new YAML), output kept in the PR description: exactly the 35 target workflows differ, each only by the removed `build-approval` job and the removed `requires` entry whose value was exactly `build-approval`; the other 9 workflows, anchors, `jobs:`, `commands:`, parameters and every other top-level key are identical.
5. After the push the PR arms `shell-tests`, `lint-config-check`, the convention checks that run on every pipeline and the 16 service workflows of the touched READMEs (test and build without a click, no deploy on a branch). After the merge, the first merge that touches a service shows `test-auto`, `build`, `deploy` running by themselves (unchanged mechanism) and the first non-main push that touches a service shows `test` and `build` running without a click.
