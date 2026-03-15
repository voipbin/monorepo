"""rag_rags rag_documents rag_chunks create tables

Revision ID: 001_create_rag_tables
Revises:
Create Date: 2026-03-15 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '001_create_rag_tables'
down_revision = None
branch_labels = None
depends_on = None


def upgrade():
    # Enable pgvector extension
    op.execute("CREATE EXTENSION IF NOT EXISTS vector")

    # Create rag_rags table
    op.execute("""
    CREATE TABLE rag_rags (
        id UUID PRIMARY KEY,
        customer_id UUID NOT NULL,
        name TEXT NOT NULL,
        description TEXT DEFAULT '',
        tm_create TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        tm_update TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        tm_delete TIMESTAMP WITH TIME ZONE
    )
    """)

    # Create rag_documents table
    op.execute("""
    CREATE TABLE rag_documents (
        id UUID PRIMARY KEY,
        customer_id UUID NOT NULL,
        rag_id UUID NOT NULL REFERENCES rag_rags(id),
        name TEXT NOT NULL,
        doc_type TEXT NOT NULL DEFAULT 'uploaded',
        storage_file_id UUID,
        source_url TEXT,
        status TEXT NOT NULL DEFAULT 'pending',
        status_message TEXT,
        tm_create TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        tm_update TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        tm_delete TIMESTAMP WITH TIME ZONE
    )
    """)

    # Create rag_chunks table
    op.execute("""
    CREATE TABLE rag_chunks (
        id UUID PRIMARY KEY,
        document_id UUID NOT NULL REFERENCES rag_documents(id),
        rag_id UUID NOT NULL,
        customer_id UUID NOT NULL,
        chunk_index INTEGER NOT NULL DEFAULT 0,
        text TEXT NOT NULL,
        section_title TEXT DEFAULT '',
        embedding vector(1536),
        token_count INTEGER DEFAULT 0,
        tm_create TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        tm_delete TIMESTAMP WITH TIME ZONE
    )
    """)

    # Partial indexes (exclude soft-deleted rows)
    op.execute("CREATE INDEX idx_rag_rags_customer_id ON rag_rags(customer_id) WHERE tm_delete IS NULL")
    op.execute("CREATE INDEX idx_rag_documents_rag_id ON rag_documents(rag_id) WHERE tm_delete IS NULL")
    op.execute("CREATE INDEX idx_rag_documents_customer_id ON rag_documents(customer_id) WHERE tm_delete IS NULL")
    op.execute("CREATE INDEX idx_rag_chunks_rag_id ON rag_chunks(rag_id) WHERE tm_delete IS NULL")
    op.execute("CREATE INDEX idx_rag_chunks_document_id ON rag_chunks(document_id) WHERE tm_delete IS NULL")

    # HNSW index for vector similarity search (cosine distance)
    op.execute("""
    CREATE INDEX idx_rag_chunks_embedding ON rag_chunks
    USING hnsw (embedding vector_cosine_ops)
    """)


def downgrade():
    op.execute("DROP TABLE IF EXISTS rag_chunks")
    op.execute("DROP TABLE IF EXISTS rag_documents")
    op.execute("DROP TABLE IF EXISTS rag_rags")
    op.execute("DROP EXTENSION IF EXISTS vector")
