# Update RAG RST Documentation

**Date:** 2026-03-20
**Branch:** NOJIRA-Update-rag-rst-docs

## Problem

The RAG RST documentation in `bin-api-manager/docsdev/source/` is stale. Several API changes have landed since the docs were written:

1. Source struct gained `id` and `customer_id` fields (via `commonidentity.Identity` embed)
2. `DELETE /rags/{id}/sources/{source_id}` endpoint was added (remove source)
3. `GET /rags` (list RAGs) endpoint exists but was never documented in the tutorial
4. Duplicate source rejection (409) was added in commit `e60eed6cc`
5. Best practices incorrectly state documents are immutable with no delete endpoint

## Approach

Update four existing RST files — no new files needed.

### Files to Change

1. **`rag_struct_rag.rst`** — Add `id` and `customer_id` fields to the Source struct section. Update the Source example to include these fields.

2. **`rag_tutorial.rst`** — Add two new tutorial steps:
   - "List RAGs" (`GET /rags`) with paginated response example
   - "Remove a Source" (`DELETE /rags/{id}/sources/{source_id}`) with response example
   - Update all existing response examples so `sources[]` entries include `id` and `customer_id`
   - Add 409 Conflict to troubleshooting for duplicate sources

3. **`rag_overview.rst`** — Fix "Document Management" best practice: replace the "immutable" statement with guidance on using `DELETE /rags/{id}/sources/{source_id}` to remove and re-add updated sources.

4. **No changes to `rag.rst`** (index file) — it already includes all three sub-files.

### Trade-offs

- Renumbering tutorial steps (currently Steps 1-5) to accommodate two new steps. Chose to insert "List RAGs" after Create and "Remove Source" after Add Sources, keeping the logical flow: Create → List → Add Sources → Remove Source → Check Status → Update → Delete.

## Verification

- Rebuild HTML: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`
- Verify no Sphinx warnings
- Spot-check rendered HTML for the RAG pages
- Commit both RST source and built HTML
