-- 000002_add_document_ingestion_fields.up.sql
ALTER TABLE rag_documents ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE rag_documents ADD COLUMN IF NOT EXISTS tm_processing TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_rag_documents_status ON rag_documents(status) WHERE tm_delete IS NULL;
