# Run verify-only workflows on main without approval (VOIP-1597)

Follows `2026-10-13-main-auto-deploy-design.md`, `2026-10-14-main-auto-deploy-phase2-design.md` and `2026-10-14-timeline-rag-auto-deploy-design.md`. Those changes removed the approval click from every service that deploys on `main` and is not kept approval-gated on purpose. Two workflows still wait for a click on `main` although they never deploy anything: `bin-common-handler` and `bin-openapi-manager`. This change lets their verification job run on `main` by itself.

## Findings (measured 2026-10-15, read-only)

- F1. `bin-common-handler` runs one job, `bin-common-handler-test` (`go-test` on the module), behind `build-approval`. `bin-openapi-manager` runs one job, `bin-openapi-manager-validate` (spec syntax check and a check that the committed generated models match), behind `build-approval`. Neither workflow has a deploy job, a build job, a push, a `context` or a production secret.
- F2. On `main` the approval stays on hold until a human clicks it, so the verification either waits or never runs: the recent `main` pipelines held 9 on-hold `bin-common-handler` workflows and 8 on-hold `bin-openapi-manager` workflows.
- F3. `check-auto-deploy-config.rb` accepts only two shapes: the auto-deploy shape (listed in `auto-deploy-services.txt`) and the approval-kept shape. It rejects a `-test-auto` job and an `ignore: main` filter outside the allowlist, so the verify-only shape needs its own, explicit rule.
- F4. `bin-openapi-manager` is also a trigger of the `bin-api-manager` workflow (`when: or [...]`, guard path list). That is unaffected: this change touches only the `bin-openapi-manager` workflow itself.

## Change

1. `config_work.yml`: both workflows get the same three jobs. `build-approval` and the verify job use `non_main_filters` (as before, branches still wait for an approval), and a copy of the verify job named `<job>-auto` uses `deploy_filters` (only `main`) with no `requires`. No deploy job is created.
2. `auto-run-services.txt` (new): the two workflow names. Separate from `auto-deploy-services.txt` on purpose, so a verify-only workflow can never be mistaken for a deploying one.
3. `check-auto-deploy-config.rb`: a new rule set for listed auto-run workflows. The jobs must be exactly `build-approval` (type approval, ignore main), one verify job named `<svc>-test` or `<svc>-validate` (ignore main, requires only `build-approval`) and its `-auto` copy (only main, no requires, same job key). The rule fails on any `context`, any job named or keyed `-deploy` or `-build`, any step other than `checkout`, `run`, `go-test` and `store_test_results` (so an orb or custom command such as `docker-build` or `setup_remote_docker` is rejected), and any `run` line that calls `komodo-api-deploy.sh`, `docker push`, `login`, `build` or `tag`, `buildx`, `kubectl`, `skopeo` or `gcloud`. Entries must not be in the deny list or in the auto-deploy allowlist, must not repeat, and must have a workflow. A `-auto` job in a workflow that is on neither list still fails.
4. `config.yml`: the new list file arms `shell-tests`, like the existing list.
5. `check-auto-deploy-config.bats`: the checker takes the new list as an optional fifth argument; eighteen cases pin the new rules.

## Out of scope

- `bin-trigger-sender`, `voip-asterisk-proxy`, `voip-rtpengine-proxy`: the approval is the production release because the build pushes `:latest`. `voip-kamailio-proxy` stays approval-gated. `bin-dbscheme-manager` keeps its approval, which records a manual database step.
- The `go install ...@latest` of the OpenAPI generator inside the validate job is existing behavior and is not changed here.

## Risks and mitigations

- A verify job now runs on every `main` merge that touches these directories. A failure is a red check on `main`, not a production change. The job definitions are scanned by the checker for anything that could publish or deploy; the allowed `go-test` command is a fixed definition in this file and changes to it are reviewed.
- Hiding a deploy behind the verify shape: blocked by the checker rules above and by the separate list.
- Rollback: remove the two names from `auto-run-services.txt` and revert the two workflow blocks. Later pipelines are approval-gated again.

## Verification plan

1. `bats .circleci/tests/*.bats` and the checker pass; each new rule fails when it is switched off (mutation sweep on a copy).
2. A throwaway branch that touches a file in `bin-common-handler/` and in `bin-openapi-manager/` shows `build-approval` on hold, the verify jobs blocked and no `-auto` job (no approval is clicked, the branch is deleted).
3. This change touches only `.circleci/` and `docs/`, so the merge itself runs `shell-tests` and `lint-config-check` only. At the first later merge that touches each directory, the `-auto` job runs by itself and succeeds.
4. Right after the merge, cancel the old on-hold workflows of the two services.
