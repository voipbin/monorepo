# VOIP-1545 PR-A: asterisk-proxy recording upload fix (VOIP-1546) and docs refresh

Date: 2026-09-30. Tickets: VOIP-1545 (proxy rebuild ahead of Asterisk 23.5.0), VOIP-1546 (Bug).
Branch / PR: `VOIP-1545-Refresh-asterisk-proxy-readme` / monorepo #1353 (branch name kept: renaming the head branch of an open PR closes it per GitHub docs).
Status: Revision 2. Design review finished (rounds 3 and 4 approved); implemented in this PR.

## 1. Problem

1. Production asterisk-proxy (digest `86b07fba`, monorepo 61224fe40, 2026-09-14) must be rebuilt from current main before the Asterisk 23.5.0 upgrade (CEO decision: proxy first). The monorepo `voip-asterisk-proxy` workflow runs only when a file under `voip-asterisk-proxy/` changes (`.circleci/config.yml`, `base-revision: main`).
2. VOIP-1546 defect in `pkg/servicehandler/recording.go` `recordingFileUpload`: after `io.Copy` into the GCS writer, `wc.Close()` is deferred with its error ignored, and `os.Remove` of the local file runs before it. storage v1.61.3 reports the upload result only from `Writer.Close` (multipart upload for files up to the 16 MiB chunk; the request is sent at Close). Any GCS rejection at commit (permissions, revoked key, missing bucket, network) returns success while the local file is already deleted: recording lost. Verified with a fake GCS server: 403 gives `io.Copy` nil and `Close` `googleapi: Error 403`, also for 0-byte files. Also on `io.Copy` error the writer is never closed and 2 goroutines leak per writer permanently (ctx is `context.Background()`).
   CEO decision 2026-09-30: fix VOIP-1546 in this PR so it ships with the rebuild.
3. Proxy docs drift against code: README lacks `--google_application_credentials_json` (init.go:89; 21 of 22 flags documented); `--recording_bucket_directory` reads like a mount but is a GCS object-name prefix used verbatim (`fmt.Sprintf("%s/%s", dir, filename)`); `docs/domain.md` says ADC only; `docs/architecture.md` names the handler `MoveRecordingFile` (real: `RecordingFileMove`); `docs/operations.md` troubleshooting lacks the upload failure symptoms; `docs/subsystems.md` shows an `emptyDir` share without the retention implication.

Issue analysis (9 review rounds, closed with 2 consecutive approvals): `~/.hermes/cache/scratch/ast/VOIP-1546-analysis.md`.

## 2. Goals

1. Recording upload deletes the local file only after GCS confirms the commit; any upload failure is returned as an RPC error with the local file kept.
2. A failed copy never commits a partial object and never leaks the writer.
3. Delete failure after a confirmed upload: log an error and return success (CEO decision (b)).
4. Unit tests with a fake GCS server cover every path, runnable as root.
5. README table matches all 22 flags; README/domain/architecture/operations/subsystems describe the fixed behavior.
6. Merge produces `voipbin/voip-asterisk-proxy:<merge sha>` via the main pipeline; digest captured per §7.

## 3. Non-goals

- No change to the RPC contract, listenhandler, call-manager, or go.mod (httptest, logrus hooks/test, storage, option are already available).
- No retry, no per-file result reporting, no orphan-object reconciliation (residual risks, §8).
- No CI change (proxy workflow has no test job; tests are verified locally).
- README footer `<!-- Updated dependencies: 2026-02-20 -->` kept (repo-wide dependency marker; dependencies unchanged).

## 4. Affected files

| File | Change |
|---|---|
| `voip-asterisk-proxy/pkg/servicehandler/recording.go` | New upload sequence (§5.1) |
| `voip-asterisk-proxy/pkg/servicehandler/recording_test.go` | New tests (§5.2) |
| `voip-asterisk-proxy/README.md` | Table rows (§5.3), Recording upload section (§5.4) |
| `voip-asterisk-proxy/docs/domain.md` | Recording file paragraph (§5.5) |
| `voip-asterisk-proxy/docs/architecture.md` | Handler name `RecordingFileMove` (§5.6) |
| `voip-asterisk-proxy/docs/operations.md` | Line 11 troubleshooting row, new row, line 89 wording (§5.7) |
| `voip-asterisk-proxy/docs/subsystems.md` | Retention sentence in Recording file sharing (§5.8) |
| `docs/plans/2026-09-30-voip-1545-asterisk-proxy-readme-refresh-design.md` | This document |

## 5. Changes

### 5.1 recording.go

`RecordingFileMove` unchanged (nil-client guard, request-order loop, stop at first error).

```go
func (h *serviceHandler) recordingFileUpload(ctx context.Context, filename string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "recordingFileUpload",
		"filename": filename,
	})

	sourceFilepath := fmt.Sprintf("%s/%s", h.recordingAsteriskDirectory, filename)
	destinationFilepath := fmt.Sprintf("%s/%s", h.recordingBucketDirectory, filename)

	sourceFile, err := os.Open(sourceFilepath)
	if err != nil {
		return errors.Wrapf(err, "failed to open source file. source_filepath: %s", sourceFilepath)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	// The writer context is cancelled on a copy failure so that Close aborts the upload
	// instead of committing a partial object. Cancelling after a successful Close is a no-op.
	writerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	wc := h.client.Bucket(h.recordingBucketName).Object(destinationFilepath).NewWriter(writerCtx)
	if _, errCopy := io.Copy(wc, sourceFile); errCopy != nil {
		cancel()
		_ = wc.Close() // abort; the copy error is the one returned
		return errors.Wrapf(errCopy, "failed to copy data. source_filepath: %s, destination_filepath: %s", sourceFilepath, destinationFilepath)
	}

	// GCS commits the upload in Close; its error is the upload result.
	if errClose := wc.Close(); errClose != nil {
		return errors.Wrapf(errClose, "failed to upload the file to the bucket. source_filepath: %s, destination_filepath: %s", sourceFilepath, destinationFilepath)
	}
	log.Debugf("Uploaded the file to bucket. source_filepath: %s, destination_filepath: %s", sourceFilepath, destinationFilepath)

	// Delete the local file only after the upload is committed. A delete failure does not
	// fail the request: the recording is safely stored and must still be registered.
	if errRemove := os.Remove(sourceFilepath); errRemove != nil {
		log.WithFields(logrus.Fields{
			"source_filepath":      sourceFilepath,
			"destination_filepath": destinationFilepath,
		}).Errorf("Uploaded the recording file but could not delete the local file. Remove it manually. err: %v", errRemove)
	}

	return nil
}
```

Evidence for the abort choice (scratch, storage v1.61.3, fake server): plain `Close` after a copy error returns nil and commits the partial data (1 request); `CloseWithError` also sends 1 request; `cancel()` then `Close()` returns `context canceled` with 0 requests (500 iterations under `-race`, no failure); `cancel()` after a successful `Close` has no effect.

### 5.2 recording_test.go (package `servicehandler`)

Helper: `httptest.Server` fake of the GCS JSON upload endpoint (`POST /upload/storage/v1/b/<bucket>/o`, multipart), recording hit count, object name (`name` query param) and body; configurable status. Client: `storage.NewClient(ctx, option.WithEndpoint(srv.URL+"/storage/v1/"), option.WithoutAuthentication())`, injected via `&serviceHandler{client: ..., recordingBucketName: "bucket", recordingAsteriskDirectory: t.TempDir(), recordingBucketDirectory: "recording"}` (same package; no production change). Log assertions via `github.com/sirupsen/logrus/hooks/test` (`test.NewGlobal()`).

| Test | Setup | Assertions |
|---|---|---|
| success | file with content, server 200 | nil error; 1 hit; object name `recording/<file>`; body equals content; local file gone |
| commit rejected (403) | server 403 | error; local file kept |
| commit rejected, 0-byte file | empty file, server 403 | error; local file kept |
| copy failure | source path is a directory (`os.Open` succeeds, `io.Copy` fails "is a directory") | error; 0 hits (no partial object); path still exists |
| multi-file stop at first error | files a, b, c; server 200 for a, 403 for b | error mentions b; a uploaded and deleted; b kept; c kept and never requested (no request with object name `recording/c`); exactly 2 hits |
| delete failure after commit | server handler, before responding 200, removes the source file and creates a non-empty directory at the same path, so `os.Remove` fails with ENOTEMPTY | nil error; 1 hit; an Error-level log entry with `source_filepath` and `destination_filepath` |
| nil client | existing `Test_RecordingFileMove_nilClient_returnsErrorNoPanic` in main_test.go | unchanged |

Determinism of the delete-failure test: the handler completes the swap before writing its response and multipart `Close` returns only after the response, so `os.Remove` always runs after the swap. Works as root (a read-only directory would not, root bypasses it; not used).
Mutation checks during implementation: (1) revert to deferred ignored Close -> "commit rejected" tests fail; (2) replace `cancel(); Close()` with plain `Close()` in the copy-error branch -> "copy failure" test fails on hits (0-byte object committed); (3) return the remove error -> "delete failure" test fails; (4) make the loop continue after an error -> "multi-file" test fails (c gets requested, 3 hits).
Test naming follows the repo pattern `Test_RecordingFileMove_<case>`, one test per row, in a single `recording_test.go`. Test conventions: the fake server rejects with a non-retryable 4xx (403) and a JSON error body (5xx could trigger retry policy); tests using the global logrus hook do not call `t.Parallel()` and call `hook.Reset()` before acting. The copy-failure test covers the writer-never-opened path; the opened-writer-then-cancel path relies on the scratch evidence in §5.1.

### 5.3 README table rows (defaults from init.go)

```
| `--recording_bucket_name` | `RECORDING_BUCKET_NAME` | `` | GCS bucket name for recording uploads (required for uploads) |
| `--recording_asterisk_directory` | `RECORDING_ASTERISK_DIRECTORY` | `/var/spool/asterisk/recording` | Local Asterisk recording directory (shared with the Asterisk container) |
| `--recording_bucket_directory` | `RECORDING_BUCKET_DIRECTORY` | `/mnt/media/recording` | GCS object-name prefix, used verbatim as `<prefix>/<filename>` (not a filesystem mount). The default yields object names starting with `/`; VoIPBin deployments set `recording`, the prefix call-manager uses when it registers the file with storage-manager |
| `--google_application_credentials_json` | `GOOGLE_APPLICATION_CREDENTIALS_JSON` | `` | GCP service-account key JSON content (not a file path) for GCS uploads. Empty means Application Default Credentials |
```

### 5.4 README "Recording upload" section (on this branch it replaces the Revision 1 section, README lines 60-87, keeping the blank line before `# RabbitMQ RPC`; relative to main, which has no such section, it is a new section inserted before `# RabbitMQ RPC`)

```
# Recording upload

The proxy uploads recordings itself with the Google Cloud Storage Go client when it receives
the `/proxy/recording_file_move` RPC. The bucket is not mounted (no gcsfuse); only the local
recording directory is shared with the Asterisk container.

For each filename, in order:
1. Open `<recording_asterisk_directory>/<filename>`.
2. Copy it into a GCS object writer for object `<recording_bucket_directory>/<filename>` in bucket
   `<recording_bucket_name>`. The object name is the two values joined with `/`, without normalization.
3. Close the object writer. GCS commits the upload here, and the result is checked.
4. Delete the local file, only after the upload is committed.

Failure behavior:
- Open or copy error: the upload is aborted (no partial object), the local file is kept, and the
  RPC returns an error.
- Upload rejected at commit (permissions, network, revoked key, missing bucket): the local file is
  kept and the RPC returns an error. The caller does not register the recording.
- Local delete error after a committed upload: the RPC returns success (the recording is stored
  and gets registered) and the proxy logs an error with the source and destination paths. When the
  recording is registered, storage-manager moves the object from `<recording_bucket_directory>/<filename>`
  to its own `bin/<file id>` path, so it is no longer at the logged destination. Remove the local
  file manually once the recording's storage file record exists.
- Files are processed in request order and processing stops at the first error, so earlier files
  in the same request may already be uploaded and deleted.

Kept local files survive only as long as the recording volume does: a named Docker volume (as in
the Komodo deployment) survives container recreation but not volume deletion or stack
migration; a Kubernetes `emptyDir` is lost when the pod is deleted.

If the GCS client cannot be created at startup (for example, invalid credential JSON, or no
credential JSON and no Application Default Credentials), the proxy still starts and only recording
upload returns an error.
```

### 5.5 docs/domain.md "Recording file" paragraph

```
A call recording generated by Asterisk, stored on the local filesystem at `--recording_asterisk_directory` (default `/var/spool/asterisk/recording`). On a `/proxy/recording_file_move` request, the proxy streams each file to GCS bucket `--recording_bucket_name` as object `<recording_bucket_directory>/<filename>`, checks the commit result, and deletes the local file only after the upload is committed. An upload failure is returned as an RPC error with the local file kept. See README "Recording upload" for the full failure behavior. Authentication uses the service-account key JSON in `--google_application_credentials_json` when set, otherwise Application Default Credentials.
```

### 5.6 docs/architecture.md
`serviceHandler.MoveRecordingFile` -> `serviceHandler.RecordingFileMove`.

### 5.7 docs/operations.md

- Line 11 row replaced:
```
| Recording upload fails (proxy logs `Could not move the recording files` with the GCS cause, e.g. `googleapi: Error 403`, and returns a bodiless 500 to `/proxy/recording_file_move`; call-manager logs `Could not stopped the recording`, whose error chain carries only `Internal Server Error` as the cause, so look for the cause in the proxy log; no storage file record is created; the local file stays in the recording directory) | GCS bucket not configured, credentials not available or lacking permission, or GCS unreachable | Set `--recording_bucket_name`; provide credentials via `GOOGLE_APPLICATION_CREDENTIALS_JSON` (service-account key JSON) or ensure ADC/workload-identity grants `roles/storage.objectCreator` on the bucket. The kept local file is the only copy. Uploading it manually does not register the recording (no storage file record, `on_end_flow` was not run, no automatic retry); a separate registration step is needed (no such tool exists yet; follow-up candidate) |
```
- New row after it:
```
| Error log `Uploaded the recording file but could not delete the local file` | Local filesystem error after a committed upload | The recording is stored and registered; storage-manager moves the object to `bin/<file id>`, so it is no longer at `destination_filepath`. Confirm the recording's storage file record exists, then delete the file at `source_filepath` |
```
- Line 89 description `GCS path prefix for uploads` -> ``GCS object-name prefix for uploads, used verbatim as `<prefix>/<filename>` `` (the placeholder must be in backticks, otherwise markdown drops it as an HTML tag).

### 5.8 docs/subsystems.md
Append to the "Recording file sharing" paragraph: "Files whose upload fails are kept in this directory for recovery, so it should outlive the pod if they are to be recovered: `emptyDir` is lost on pod deletion, `hostPath` stays on the node, and a named Docker volume (the Komodo deployment) survives container recreation." Example YAML unchanged.

## 6. Verification

- In `voip-asterisk-proxy` (Go 1.27.1, `GOTOOLCHAIN=auto`): `go mod tidy` (no diff), `go mod vendor`, `go generate ./...` (no diff), `go build ./...`, `go vet ./...`, `go test ./...`, `go test -race -count=5 ./pkg/servicehandler/...`, `golangci-lint run -v --timeout 5m`; remove `vendor/` afterwards. Package count and results re-measured and recorded in the PR body. Known unrelated flake: `pkg/listenhandler` `Test_Run_single_consumer_error`.
- `bin-common-handler`: `go test ./pkg/requesthandler/...`.
- All four mutation checks of §5.2 performed and recorded in the PR body.
- README flag parity script: 22/22.
- `git diff --stat origin/main...HEAD`: exactly the 8 files of §4.
- Gate is run again after writing the docs (current expected hits until replaced: README lines 70, 74, 81 and domain.md line 71).
- Stale-statement gate (case-insensitive): the 26-phrase pattern defined in the VOIP-1546 issue analysis §3-6 (kept outside the repo so this document does not match itself; working copy `~/.hermes/cache/scratch/ast/gate.pat`) is run with `grep -niE -f` on each changed file (this document with the Revision 1 history section cut) and on the PR #1353 body, per line and newline-joined. It targets Revision 1 statements such as the documentation-only scope claim, the unreported-commit limitation paragraph, and the old step order where local deletion preceded the commit. Must return no hits. Code comments avoid these phrases.
- No em/en dashes in added lines.

## 7. Rollout and digest capture

1. PR review loop; CEO merge instruction (squash). Squash body = PR body.
2. Approve only the `build-approval` job of the `voip-asterisk-proxy` workflow on the main pipeline whose checkout sha equals the merge sha (`git rev-parse origin/main` after merge). Do not approve branch pipelines (#5002, #5003 and later ones for this branch) or the old on_hold #4964 (would overwrite `:latest` with an older build).
3. Digest: `curl -s https://hub.docker.com/v2/repositories/voipbin/voip-asterisk-proxy/tags/<merge sha>` `digest` field; cross-check with the build log `docker push` digest and the `latest` digest; record tag, digest, pipeline number, job URL, time in VOIP-1545 and the PR-B body.
4. Do not rerun the build job (floating base images give a new digest for the same tag); if unavoidable, redo step 3 and use only the last successful run.
5. Post-deploy (with PR-B): smoke (e) recording upload to GCS succeeds and the local file is deleted; after the first recordings, confirm no proxy `Could not move the recording files` logs, no call-manager `Could not stopped the recording` logs, and no files accumulating in the recording volumes (a previously silent GCS failure would now show up here).
6. Rollback: if PR-B's deploy misbehaves, re-pin the proxy to the previous digest `sha256:86b07fba...` (Komodo re-pin, then a revert PR so git matches).
7. Tickets: after merge, transition VOIP-1546 to Done; VOIP-1545 stays open until PR-D.

## 8. Risks

- Behavior change on a GCS commit failure (intended). Before: the proxy returned 200 with the local file already deleted; call-manager then failed at `StorageV1FileCreate` because storage-manager could not find the object (`file does not exist`, bin-storage-manager `filehandler/file.go` attrs check), logged `Could not send the request for the storing the recording correctly`, and `Stopped` returned an error; the recording content was unrecoverable. After: the failure happens at the move step (proxy 500, cause in the proxy log, call-manager chain `Could not move the recording files. err: Internal Server Error`), and the local file is kept, so the recording is recoverable. Customer-visible outcome is the same in both cases: the `recording_finished` webhook was already published (before the upload), no storage file record exists, and `on_end_flow_id` is not executed. For the post-deploy check (§7.5), the baseline log for this failure changes from `Could not send the request for the storing the recording correctly` to `Could not move the recording files`.
- Residual, pre-existing, out of scope: (a) an upload longer than the 30 s RPC timeout (large file or resumable retries above 16 MiB) is treated as failed by call-manager while the proxy commits and deletes: orphan object; (b) multi-file request with a later failure: earlier files become orphan objects; (c) during a long GCS outage kept files accumulate on the recording volume; (d) Close error although GCS committed (lost response): local file kept, a re-upload with the same name needs overwrite permission beyond `roles/storage.objectCreator`.
- Latency unchanged: before this fix the writer was closed by a defer just before the handler returned, so the commit already happened inside the RPC; now it happens explicitly before the delete.

## 9. Approval status
Revision 2. Design round 1: A CHANGES_REQUESTED (2), B CHANGES_REQUESTED (8). Round 2: CHANGES_REQUESTED (2). Round 3: APPROVED. Round 4: APPROVED. Design review loop finished (2 consecutive approvals).

## Revision 2 review response summary
R1-A1 multi-file test a/b/c, c never requested -> §5.2. R1-A2 mutation (4) -> §5.2. A non-blocking: 4xx + JSON body, no t.Parallel + hook.Reset, opened-writer cancel evidence -> §5.2.
R1-B1 delete-failure confirmation via storage file record; object moved to bin/<id> (verified bin-storage-manager filehandler/file.go, bucketfile.go) -> §5.4, §5.7. R1-B2 manual upload does not register -> §5.7. R1-B3 log wording (proxy vs call-manager arieventhandler) -> §5.7. R1-B4 backticks line 89 -> §5.7. R1-B5 prefix owner = call-manager -> §5.3. R1-B6 customer impact -> §8. R1-B7 post-deploy checks, rollback digest, VOIP-1546 Done -> §7. R1-B8 README replace range + gate rerun -> §5.4, §6.
R2-1 call-manager log chain corrected (bodiless 500 -> ErrInternal "Internal Server Error"; cause only in proxy log; verified listenhandler simpleResponse and requesthandler common.go) -> §5.7. R2-2 before/after behavior corrected (pre-fix failure at StorageV1FileCreate, same customer outcome, recoverability is the gain, baseline log change) -> §8. Non-blocking: gate hit list, `origin/main...HEAD`, registration follow-up note -> §6, §5.7.
R3 APPROVED. Non-blocking applied: README blank line 88 kept (§5.4), test naming/single file (§5.2), all 4 mutations recorded (§6). The long §5.7 symptom cell is kept as is (one table row per failure mode is the file's convention).
R4 APPROVED (mutation (1) confirmed). Non-blocking applied: R2/R3 summaries moved out of the history section; §5.7 wording narrowed to "cause".
PR round 1: A, B, C APPROVED. Non-blocking applied: error-message assertions in commitRejected/copyFailure, logrus hook cleanup, §5.4 heading clarified (main has no such section).
PR round 2: APPROVED. Non-blocking applied: status header updated.

## Revision 1 history

Revision 1 was a docs-only version of this PR (design rounds 1-4, PR rounds 1-3). It was superseded by Revision 2 when the CEO folded the VOIP-1546 fix into this PR. The review records below describe Revision 1 and are kept for history only.

### Iter-1 review response summary
A1 upload order / failure paths / defect -> §1.1 (VOIP-1546), §5.2, §5.5. A2 object name + leading slash + prod `recording` -> §1, §5.1, §5.2. A3 domain.md/architecture.md drift -> included, §4, §5.3, §5.4. A4 footer kept -> §3. A5 client-creation conditions + bucket required -> §5.1, §5.2. A6 #4964 do-not-approve -> §7.2. A7 script handles literals/Int/Bool -> §6.
B1 digest capture procedure -> §7.3. B2 no-rerun rule -> §7.4. B3 pipeline identification + #4964 -> §7.2. B4 measured local results + vendor cleanup + toolchain -> §6. B5 credential condition wording -> §5.2.

### Iter-2 review response summary
R2-1 real 4-step order + silent-loss limitation, narrowed open/copy failure scope -> §5.2, §5.3, §5.5 wording note. R2-2 os.Remove failure path -> §5.2 other failure paths, §5.5 rows. R2-3 compose file count corrected (call/conference 4 files; registrar none) -> §1, §5.5. Non-blocking: "for example" added to client-creation conditions -> §5.2.

### Iter-3 review response summary
Round 3: APPROVED. Non-blocking notes applied: status header v3; bucket-not-mounted wording (§5.2); default prefix leading-slash consequence (§5.1); delete-error wording "step 4 still runs and commits" (§5.2). Zero-byte edge noted on VOIP-1546 ticket (not in docs).

### Iter-4 review response summary
Round 4: APPROVED (2nd consecutive). Non-blocking notes applied: §1 file-count wording; §5.2 insertion position clarified. Design review loop CLOSED (R1 CR x2 reviewers, R2 CR, R3 APPROVED, R4 APPROVED).

### PR review notes
PR round 1 (2 reviewers) APPROVED, round 2 APPROVED. Round-2 non-blocking note applied: domain.md object placeholder `<recording_bucket_directory>/<filename>` (matches README). Other notes (operations.md VOIP-1546 hint, subsystems.md emptyDir example) out of scope, deferred to VOIP-1546.
