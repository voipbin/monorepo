# CircleCI Credit Optimization Design

**Date:** 2026-02-05
**Branch:** NOJIRA-circleci-credit-optimization
**Status:** Draft

## Problem Statement

CircleCI credit usage has increased significantly, causing frequent credit warnings.

**Current usage:**
- Total: ~252,000 credits/month
- `docker-large`: 156,728 credits (62%)
- `remote-docker-large`: 94,492 credits (37%)
- `docker-small`: 1,410 credits (1%)

## Goals

- Reduce monthly credit consumption by 80%+
- Maintain build functionality
- Allow rollback if issues occur

## Solution Overview

Four optimization strategies:

1. **Resource class optimization** - Use `small` for all jobs
2. **Remote docker optimization** - Use `small` size for docker builds
3. **Remove unnecessary remote_docker** - Remove from release jobs
4. **Workflow restructuring** - Move approval to beginning

## Detailed Changes

### Change 1: Add resource_class: small to all jobs

All test, build, and release jobs will explicitly set `resource_class: small`.

**Example:**
```yaml
# Before
bin-agent-manager-test:
  docker: *go_image
  steps:
    - go-test:
        source-directory: bin-agent-manager

# After
bin-agent-manager-test:
  docker: *go_image
  resource_class: small
  steps:
    - go-test:
        source-directory: bin-agent-manager
```

**Affected jobs:** ~82 jobs (all test, build, release, validate jobs)

### Change 2: Modify docker-build command

Add explicit `size: small` and disable Docker Layer Caching.

```yaml
# Before
docker-build:
  steps:
    - setup_remote_docker
    - checkout
    ...

# After
docker-build:
  steps:
    - setup_remote_docker:
        docker_layer_caching: false
        size: small
    - checkout
    ...
```

### Change 3: Remove setup_remote_docker from docker-release

The release job only runs `kubectl apply` - no Docker commands needed.

```yaml
# Before
docker-release:
  steps:
    - setup_remote_docker    # Unnecessary - remove
    - checkout
    - run: Config service account
    - run: Release (kubectl apply)

# After
docker-release:
  steps:
    - checkout
    - run: Config service account
    - run: Release (kubectl apply)
```

### Change 4: Move approval to beginning of workflow

Run test and build only after manual approval.

```yaml
# Before
test → build → approval → release

# After
approval → test → build → release
```

**Example workflow:**
```yaml
# Before
bin-agent-manager:
  jobs:
    - bin-agent-manager-test
    - bin-agent-manager-build:
        requires:
          - bin-agent-manager-test
    - release-approval:
        type: approval
        requires:
          - bin-agent-manager-build
    - bin-agent-manager-release:
        requires:
          - bin-agent-manager-build
          - release-approval

# After
bin-agent-manager:
  jobs:
    - build-approval:
        type: approval
    - bin-agent-manager-test:
        requires:
          - build-approval
    - bin-agent-manager-build:
        <<: *context_production
        requires:
          - bin-agent-manager-test
    - bin-agent-manager-release:
        <<: *context_production
        requires:
          - bin-agent-manager-build
```

**Affected workflows:** ~28 workflows

## Expected Results

| Optimization | Impact |
|-------------|--------|
| resource_class: small | ~75% reduction |
| remote_docker size: small | ~75% reduction |
| Remove remote_docker from release | Additional ~50% reduction |
| Approval at start | **Zero credits until approved** |

**Final result:** Only approved builds consume credits, and at minimum resource levels.

## Resource Class Reference

### Docker Executor
| Class | vCPU | RAM | Credits/min |
|-------|------|-----|-------------|
| small | 1 | 2 GB | 5 |
| medium | 2 | 4 GB | 10 |
| large | 4 | 8 GB | 20 |

### Remote Docker
| Size | vCPU | RAM | Credits/min |
|------|------|-----|-------------|
| small | 2 | 8 GB | 5 |
| medium | 4 | 15 GB | 10 |
| large | 8 | 32 GB | 20 |

## Rollback Plan

If builds fail or become too slow:

1. Increase specific job's `resource_class` to `medium`
2. Increase `remote_docker size` to `medium`
3. If approval-first workflow causes issues, revert to original flow

## File Changes

**Target file:** `.circleci/config_work.yml`

**Summary:**
- Modify `docker-build` command (add size: small, disable DLC)
- Modify `docker-release` command (remove setup_remote_docker)
- Add `resource_class: small` to all jobs (~82 jobs)
- Restructure all workflows to approval-first (~28 workflows)
