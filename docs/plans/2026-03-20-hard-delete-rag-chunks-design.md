# Hard-Delete RAG Chunks After Soft-Delete

**Date:** 2026-03-20
**Status:** Approved

## Problem

When a document or RAG is deleted, chunks are soft-deleted (tm_delete timestamp set). This makes them invisible to search queries but they remain in the database, consuming storage — especially significant because each chunk carries a 768-dimension vector embedding.

There is no need to restore deleted documents or chunks. If a customer needs a document again, they create a new one.

## Decision

- **Chunks**: Hard-delete after soft-delete (two-phase)
- **Documents**: Keep soft-delete only (lightweight metadata, useful for audit)

## Approach

After the existing soft-delete cascade completes, fire a background goroutine that hard-deletes the chunk rows. The soft-delete ensures chunks are immediately invisible to search; the goroutine reclaims storage asynchronously.

### RagRemoveSource (single document)

1. Soft-delete chunks by document ID (existing `ChunkSoftDeleteByDocumentID`) — chunks disappear from search
2. Soft-delete the document (existing `DocumentDelete`)
3. `go h.chunkHardDeleteByDocumentID(documentID)` — background hard-delete

### RagDelete (full RAG)

1. Soft-delete chunks by RAG ID (existing `ChunkSoftDeleteByRagID`) — all chunks disappear from search
2. Soft-delete documents by RAG ID (existing `DocumentDeleteByRagID`)
3. Soft-delete the RAG (existing `RagDelete`)
4. `go h.chunkHardDeleteByRagID(ragID)` — background hard-delete

## Implementation Details

### New private methods in raghandler

Two thin wrapper methods that call existing DB hard-delete methods with `context.Background()` and error logging:

- `chunkHardDeleteByDocumentID(documentID uuid.UUID)` — calls `dbHandler.ChunkDeleteByDocumentID`
- `chunkHardDeleteByRagID(ragID uuid.UUID)` — calls `dbHandler.ChunkDeleteByRagID`

### No new DB methods

The hard-delete methods already exist in `pkg/dbhandler/chunk.go`:
- `ChunkDeleteByDocumentID` — `DELETE FROM rag_chunks WHERE document_id = ?`
- `ChunkDeleteByRagID` — `DELETE FROM rag_chunks WHERE rag_id = ?`

### No schema changes

No migration needed.

### Context handling

Background goroutines use `context.Background()` since the request context may be cancelled after the response is sent. This follows the existing pattern used by `documentIngest` goroutines.

## Error Handling

If the background hard-delete fails, chunks remain soft-deleted — invisible to search but still in the database. This is a safe degradation. Errors are logged for observability.

## Files Changed

- `bin-rag-manager/pkg/raghandler/rag.go` — Add goroutine calls after soft-delete in `RagDelete` and `RagRemoveSource`; add `chunkHardDeleteByDocumentID` and `chunkHardDeleteByRagID` private methods

## Trade-offs

- **No batching**: Large RAGs could trigger a single large DELETE. Acceptable for now since it runs in the background; batching can be added later if needed.
- **No retry**: If the hard-delete fails, there is no automatic retry. The data remains soft-deleted (safe). A periodic cleanup sweep could be added later if this becomes an issue.
