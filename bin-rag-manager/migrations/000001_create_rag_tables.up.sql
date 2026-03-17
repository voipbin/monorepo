-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create rag_rags table
CREATE TABLE IF NOT EXISTS rag_rags (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    tm_create TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    tm_update TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    tm_delete TIMESTAMP WITH TIME ZONE
);

-- Create rag_documents table
-- Foreign keys intentionally omit ON DELETE CASCADE — the application layer
-- controls deletion order: chunks first, then documents, then rags.
CREATE TABLE IF NOT EXISTS rag_documents (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    rag_id UUID NOT NULL REFERENCES rag_rags(id),
    name TEXT NOT NULL,
    doc_type TEXT NOT NULL DEFAULT 'uploaded',
    storage_file_id UUID,
    source_url TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    status_message TEXT NOT NULL DEFAULT '',
    tm_create TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    tm_update TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    tm_delete TIMESTAMP WITH TIME ZONE
);

-- Create rag_chunks table
CREATE TABLE IF NOT EXISTS rag_chunks (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES rag_documents(id),
    rag_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    chunk_index INTEGER NOT NULL DEFAULT 0,
    text TEXT NOT NULL,
    section_title TEXT NOT NULL DEFAULT '',
    embedding vector(768),
    token_count INTEGER DEFAULT 0,
    tm_create TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    tm_delete TIMESTAMP WITH TIME ZONE
);

-- Partial indexes (exclude soft-deleted rows)
CREATE INDEX IF NOT EXISTS idx_rag_rags_customer_id ON rag_rags(customer_id) WHERE tm_delete IS NULL;
CREATE INDEX IF NOT EXISTS idx_rag_documents_rag_id ON rag_documents(rag_id) WHERE tm_delete IS NULL;
CREATE INDEX IF NOT EXISTS idx_rag_documents_customer_id ON rag_documents(customer_id) WHERE tm_delete IS NULL;
CREATE INDEX IF NOT EXISTS idx_rag_chunks_rag_id ON rag_chunks(rag_id) WHERE tm_delete IS NULL;
CREATE INDEX IF NOT EXISTS idx_rag_chunks_document_id ON rag_chunks(document_id) WHERE tm_delete IS NULL;

-- HNSW index for vector similarity search (cosine distance)
CREATE INDEX IF NOT EXISTS idx_rag_chunks_embedding ON rag_chunks
USING hnsw (embedding vector_cosine_ops);
