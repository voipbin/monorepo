# VOIP-1545 PR-A: Refresh voip-asterisk-proxy docs (rebuild trigger)

Date: 2026-09-30. Ticket: VOIP-1545. Branch: `VOIP-1545-Refresh-asterisk-proxy-readme`.
Status: APPROVED (rounds 3 and 4 consecutive APPROVED).

## 1. Problem

The production asterisk-proxy sidecar (digest `86b07fba`, monorepo 61224fe40, 2026-09-14) has not been rebuilt since. The later proxy change (1e333b264) is stuck at CircleCI `build-approval` (pipeline #4964, on_hold). VOIP-1545 needs a fresh proxy image from current main before the Asterisk 23.5.0 upgrade (CEO decision 2026-09-30: proxy first, triggered by a docs PR). The monorepo `voip-asterisk-proxy` workflow only runs when a file under `voip-asterisk-proxy/` changes (path filter `voip-asterisk-proxy/.*`, `base-revision: main` in `.circleci/config.yml`).

The proxy docs have real drift against the code:
- README configuration table lacks `--google_application_credentials_json` / `GOOGLE_APPLICATION_CREDENTIALS_JSON` (`cmd/asterisk-proxy/init.go:89`, added by VOIP-1526 61224fe40). README has 21 rows; init.go has 22 flags.
- README `--recording_bucket_directory` "Bucket recording directory" reads like a filesystem mount (gcsfuse-era wording). It is a GCS object-name prefix: `pkg/servicehandler/recording.go` builds the object name as `fmt.Sprintf("%s/%s", recordingBucketDirectory, filename)` with no slash normalization. The default `/mnt/media/recording` therefore yields an object name with a leading slash; VoIPBin deployments that record set `RECORDING_BUCKET_DIRECTORY=recording` (4 komodo compose files: call and conference `docker-compose.yml` and `docker-compose-2.yml`, plus 2 GKE-era k8s `deployment.yml`; registrar compose sets no `RECORDING_*` env because it does not record), matching bin-storage-manager's expected recording prefix.
- `docs/domain.md:71` says "Authentication uses Application Default Credentials (ADC)" (stale since VOIP-1526: credential JSON first, ADC fallback) and "then removes the local copy" (overstates guarantees, see §1.1).
- `docs/architecture.md:56` names the handler `serviceHandler.MoveRecordingFile`; the real method is `RecordingFileMove`.
- `docs/operations.md` recording rows are already accurate (no change).

### 1.1 Discovered defect (out of scope, tracked as VOIP-1546)
`recordingFileUpload` runs `io.Copy` into the GCS writer, then `os.Remove` of the local file, and only then the deferred `wc.Close()` whose error is discarded. storage v1.61.3 commits the upload in `Close` (files under the 16 MiB default chunk are actually sent there). A GCS failure at Close therefore returns success while the local file is already deleted. Filed as VOIP-1546 (Bug). Not fixed here (CEO decision pending; this PR stays docs-only so the image equals current main). The docs added in this PR describe the current order of operations without promising "delete only after confirmed upload".

## 2. Goals

1. README configuration table matches all 22 flags/env/defaults in `init.go`.
2. Recording upload semantics documented accurately in README and `docs/domain.md` (GCS Go client, object-name composition, credential behavior, current order of operations and failure paths).
3. Fix `docs/architecture.md` handler name.
4. Merge to main starts the `voip-asterisk-proxy` workflow; CEO approves `build-approval` on the MAIN merge pipeline; CI pushes `voipbin/voip-asterisk-proxy:<merge sha>` and `:latest`; digest captured per §7.

## 3. Non-goals

- No Go code change, no go.mod change (image content = current main). The VOIP-1546 defect is not fixed here.
- No CI change. No digest bump (PR-B in monorepo-voip).
- README footer `<!-- Updated dependencies: 2026-02-20 -->` is kept: it is the repo-wide "dependencies updated" marker (26 of 39 service READMEs) and dependencies are unchanged.

## 4. Affected files

| File | Change |
|---|---|
| `docs/plans/2026-09-30-voip-1545-asterisk-proxy-readme-refresh-design.md` | This design doc |
| `voip-asterisk-proxy/README.md` | Add credential row; reword 3 recording rows; add "Recording upload" section |
| `voip-asterisk-proxy/docs/domain.md` | Rewrite the "Recording file" paragraph (line ~71) |
| `voip-asterisk-proxy/docs/architecture.md` | `serviceHandler.MoveRecordingFile` -> `serviceHandler.RecordingFileMove` (line ~56) |

The design doc path (`docs/plans/`) maps to no workflow except one unrelated single-file entry, so it triggers nothing extra.

## 5. Exact changes

### 5.1 README.md table rows (defaults from init.go:40-44, 85-91)

```
| `--recording_bucket_name` | `RECORDING_BUCKET_NAME` | `` | GCS bucket name for recording uploads (required for uploads) |
| `--recording_asterisk_directory` | `RECORDING_ASTERISK_DIRECTORY` | `/var/spool/asterisk/recording` | Local Asterisk recording directory (shared with the Asterisk container) |
| `--recording_bucket_directory` | `RECORDING_BUCKET_DIRECTORY` | `/mnt/media/recording` | GCS object-name prefix, used verbatim as `<prefix>/<filename>` (not a filesystem mount). The default yields object names starting with `/`; VoIPBin deployments set `recording`, the prefix storage-manager expects |
| `--google_application_credentials_json` | `GOOGLE_APPLICATION_CREDENTIALS_JSON` | `` | GCP service-account key JSON content (not a file path) for GCS uploads. Empty means Application Default Credentials |
```
(`--kubernetes_disabled` row unchanged; the new credential row is inserted before it.)

### 5.2 README.md new section (after the whole `# Run` section, i.e. after its environment-variable example block, and before `# RabbitMQ RPC`)

```
# Recording upload

The proxy uploads recordings itself with the Google Cloud Storage Go client when it receives
the `/proxy/recording_file_move` RPC. The bucket is not mounted (no gcsfuse); only the local
recording directory is shared with the Asterisk container.

For each filename, in order:
1. Open `<recording_asterisk_directory>/<filename>`.
2. Copy it into a GCS object writer for object `<recording_bucket_directory>/<filename>` in bucket
   `<recording_bucket_name>`. The object name is the two values joined with `/`, without normalization.
3. Delete the local file.
4. Close the object writer. This is where GCS commits the upload (for typical recording sizes the
   data is actually sent here).

Current limitation (VOIP-1546): step 4 runs after the local file is deleted and its error is not
reported. If GCS rejects the upload at that point (permissions, network, revoked key), the RPC
still returns success and the recording is lost.

Other failure paths:
- Open or copy error (steps 1-2): the local file is kept and the RPC returns an error.
- Local delete error (step 3): the RPC returns an error and the local file is kept, but step 4 still
  runs and commits the object, so a retry overwrites the same object.
- Files are processed in request order and processing stops at the first error, so earlier files
  in the same request may already be uploaded and deleted.

If the GCS client cannot be created at startup (for example, invalid credential JSON, or no
credential JSON and no Application Default Credentials), the proxy still starts and only recording
upload returns an error.
```

### 5.3 docs/domain.md "Recording file" paragraph

```
A call recording generated by Asterisk, stored on the local filesystem at `--recording_asterisk_directory` (default `/var/spool/asterisk/recording`). On a `/proxy/recording_file_move` request, the proxy streams each file to GCS bucket `--recording_bucket_name` as object `<recording_bucket_directory>/<filename>` and deletes the local file; the upload is committed when the object writer closes, after the local delete, and a commit failure is currently not reported (VOIP-1546). See README "Recording upload" for the exact order and failure behavior. Authentication uses the service-account key JSON in `--google_application_credentials_json` when set, otherwise Application Default Credentials.
```

### 5.4 docs/architecture.md
`serviceHandler.MoveRecordingFile` -> `serviceHandler.RecordingFileMove`.

### 5.5 Code-claim checklist

| Claim | Source |
|---|---|
| Object name `<dir>/<filename>`, no normalization | `pkg/servicehandler/recording.go` `destinationFilepath := fmt.Sprintf("%s/%s", ...)` |
| Order: open, copy, remove, then deferred Close (error discarded) | same function: `defer func() { _ = wc.Close() }()` registered before `os.Remove`, runs at return |
| Upload committed at Writer.Close | cloud.google.com/go/storage v1.61.3 `Writer.Close` doc; go.mod line 70 |
| Remove failure -> error return, deferred Close still runs | `recording.go` `if errRemove := os.Remove(...)` return path |
| Open/copy failure keeps local file, returns error | same, early `return errors.Wrapf` before `os.Remove` (copy failure path also runs the deferred source-file close only) |
| Stops at first failing file | `RecordingFileMove` loop returns on first error |
| RPC error -> 500 | `pkg/listenhandler/proxy_handler.go` `simpleResponse(500)` |
| Client creation fails -> handler alive, upload errors | `pkg/servicehandler/main.go` nil-client path; `recording.go` guard |
| Credential JSON content, empty -> ADC | `main.go` `option.WithAuthCredentialsJSON` vs `storage.NewClient(ctx)` |
| Route | `pkg/listenhandler/main.go:43` |
| Production prefix `recording` | monorepo-voip `voip-asterisk-k8s-{call,conference}/komodo/docker-compose{,-2}.yml` (4 files) `RECORDING_BUCKET_DIRECTORY=recording`; registrar has no RECORDING_* env |

Wording note: the README lists the real 4-step order including the deferred Close and states the silent-loss limitation explicitly (VOIP-1546), so readers do not infer that a failed upload keeps the local file.

## 6. Verification

- Flag parity: script parses init.go for every `pflag.String|Int|Bool("name", <default>, ...)` (resolving `default*` constants from the const block, and literals such as `false` for `kubernetes_disabled`; `defaultRedisDB` is an int) plus `viper.BindEnv("name", "ENV")`, and checks each (flag, env, default) triple appears in the README table. Expect 22/22.
- Local build/test (proxy CI has no test job), requires Go 1.27.1 (GOTOOLCHAIN=auto): in `voip-asterisk-proxy`: `go mod tidy`, `go mod vendor`, `go generate ./...`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`; in `bin-common-handler`: `go test ./pkg/requesthandler/...`. Round-1 reviewer run: tidy/generate 0 tracked diffs, build/vet ok, 5 packages ok, lint 0 issues, requesthandler ok. `go mod vendor` creates git-ignored `vendor/`; remove it after verification. Known flake `pkg/listenhandler` `Test_Run_single_consumer_error` noted if seen.
- `git diff --stat origin/main`: exactly the 4 files in §4.
- No em/en dashes in changed text.

## 7. Rollout and digest capture

1. PR review loop; CEO merge instruction (squash).
2. Identify the pipeline to approve: `branch=main` AND checkout sha == merge sha (`git rev-parse origin/main` after merge), and confirm it contains a `voip-asterisk-proxy` workflow (expected per `base-revision: main`; precedents: #4893 for 61224fe40 ran and succeeded, #4964 for 1e333b264 created on_hold). CEO approves its `build-approval`.
   - Do NOT approve the PR-branch pipeline's build-approval (it pushes a branch-sha tag that PR-B must not use).
   - Do NOT approve pipeline #4964's on_hold `build-approval` (would overwrite `:latest` with an older 1e333b264 build). Cancel it if convenient.
3. Capture the digest from the single successful build run:
   (a) merge SHA (40 chars) as above;
   (b) `curl -s https://hub.docker.com/v2/repositories/voipbin/voip-asterisk-proxy/tags/<merge sha>` -> `digest` field (single-arch amd64 image, same format as current pin `86b07fba`);
   (c) cross-check with the build job log `docker push` line `digest: sha256:...` and with the `latest` tag digest right after push;
   (d) record tag, digest, pipeline number, build job URL, timestamp in the VOIP-1545 ticket and PR-B body.
4. Do not rerun the build job. Base images (`golang:1.27.1-alpine`, `distroless/static-debian12`) float, so a rerun re-pushes the same tag with a different digest. If a rerun is unavoidable, redo step 3 entirely and use only the last successful run's digest.

Risk: none at runtime from this PR (docs only). The produced image equals current main, reviewed in the VOIP-1545 analysis.

## 8. Approval status
v3 after round 2. Round 3 APPROVED (non-blocking notes applied). Round 4 APPROVED. Design loop closed.

## Iter-1 review response summary
A1 upload order / failure paths / defect -> §1.1 (VOIP-1546), §5.2, §5.5. A2 object name + leading slash + prod `recording` -> §1, §5.1, §5.2. A3 domain.md/architecture.md drift -> included, §4, §5.3, §5.4. A4 footer kept -> §3. A5 client-creation conditions + bucket required -> §5.1, §5.2. A6 #4964 do-not-approve -> §7.2. A7 script handles literals/Int/Bool -> §6.
B1 digest capture procedure -> §7.3. B2 no-rerun rule -> §7.4. B3 pipeline identification + #4964 -> §7.2. B4 measured local results + vendor cleanup + toolchain -> §6. B5 credential condition wording -> §5.2.

## Iter-2 review response summary
R2-1 real 4-step order + silent-loss limitation, narrowed open/copy failure scope -> §5.2, §5.3, §5.5 wording note. R2-2 os.Remove failure path -> §5.2 other failure paths, §5.5 rows. R2-3 compose file count corrected (call/conference 4 files; registrar none) -> §1, §5.5. Non-blocking: "for example" added to client-creation conditions -> §5.2.

## Iter-3 review response summary
Round 3: APPROVED. Non-blocking notes applied: status header v3; bucket-not-mounted wording (§5.2); default prefix leading-slash consequence (§5.1); delete-error wording "step 4 still runs and commits" (§5.2). Zero-byte edge noted on VOIP-1546 ticket (not in docs).

## Iter-4 review response summary
Round 4: APPROVED (2nd consecutive). Non-blocking notes applied: §1 file-count wording; §5.2 insertion position clarified. Design review loop CLOSED (R1 CR x2 reviewers, R2 CR, R3 APPROVED, R4 APPROVED).

## PR review notes
PR round 1 (2 reviewers) APPROVED, round 2 APPROVED. Round-2 non-blocking note applied: domain.md object placeholder `<recording_bucket_directory>/<filename>` (matches README). Other notes (operations.md VOIP-1546 hint, subsystems.md emptyDir example) out of scope, deferred to VOIP-1546.
