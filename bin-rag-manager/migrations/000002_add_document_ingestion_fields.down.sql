-- 000002_add_document_ingestion_fields.down.sql
DROP INDEX IF EXISTS idx_rag_documents_status;
ALTER TABLE rag_documents DROP COLUMN IF EXISTS tm_processing;
ALTER TABLE rag_documents DROP COLUMN IF EXISTS retry_count;
