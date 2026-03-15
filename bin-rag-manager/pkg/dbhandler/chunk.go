package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/chunk"
)

const tableRagChunks = "rag_chunks"

// formatEmbedding converts a []float32 slice to the pgvector string format: [0.1,0.2,0.3]
func formatEmbedding(embedding []float32) string {
	parts := make([]string, len(embedding))
	for i, v := range embedding {
		parts[i] = fmt.Sprintf("%g", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// chunkColumns returns the column names for the rag_chunks table (excluding embedding).
func chunkColumns() []string {
	return []string{
		"id",
		"document_id",
		"rag_id",
		"customer_id",
		"chunk_index",
		"text",
		"section_title",
		"token_count",
		"tm_create",
		"tm_delete",
	}
}

// chunkScanRow scans a single row into a chunk.Chunk struct.
func chunkScanRow(row *sql.Row) (*chunk.Chunk, error) {
	var c chunk.Chunk
	var idBytes, documentIDBytes, ragIDBytes, customerIDBytes []byte

	err := row.Scan(
		&idBytes,
		&documentIDBytes,
		&ragIDBytes,
		&customerIDBytes,
		&c.ChunkIndex,
		&c.Text,
		&c.SectionTitle,
		&c.TokenCount,
		&c.TMCreate,
		&c.TMDelete,
	)
	if err != nil {
		return nil, err
	}

	c.ID, _ = uuid.FromBytes(idBytes)
	c.DocumentID, _ = uuid.FromBytes(documentIDBytes)
	c.RagID, _ = uuid.FromBytes(ragIDBytes)
	c.CustomerID, _ = uuid.FromBytes(customerIDBytes)

	return &c, nil
}

// chunkScanRows scans multiple rows into chunk.Chunk structs.
func chunkScanRows(rows *sql.Rows) ([]*chunk.Chunk, error) {
	res := []*chunk.Chunk{}

	for rows.Next() {
		var c chunk.Chunk
		var idBytes, documentIDBytes, ragIDBytes, customerIDBytes []byte

		err := rows.Scan(
			&idBytes,
			&documentIDBytes,
			&ragIDBytes,
			&customerIDBytes,
			&c.ChunkIndex,
			&c.Text,
			&c.SectionTitle,
			&c.TokenCount,
			&c.TMCreate,
			&c.TMDelete,
		)
		if err != nil {
			return nil, err
		}

		c.ID, _ = uuid.FromBytes(idBytes)
		c.DocumentID, _ = uuid.FromBytes(documentIDBytes)
		c.RagID, _ = uuid.FromBytes(ragIDBytes)
		c.CustomerID, _ = uuid.FromBytes(customerIDBytes)

		res = append(res, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

// ChunkCreate inserts a new chunk with its embedding vector
func (h *handler) ChunkCreate(ctx context.Context, c *chunk.Chunk, embedding []float32) error {
	embStr := formatEmbedding(embedding)

	q := psql.Insert(tableRagChunks).
		Columns(
			"id",
			"document_id",
			"rag_id",
			"customer_id",
			"chunk_index",
			"text",
			"section_title",
			"embedding",
			"token_count",
			"tm_create",
		).
		Values(
			c.ID.Bytes(),
			c.DocumentID.Bytes(),
			c.RagID.Bytes(),
			c.CustomerID.Bytes(),
			c.ChunkIndex,
			c.Text,
			c.SectionTitle,
			// pgvector accepts a string literal cast to vector type
			sq.Expr("?::vector", embStr),
			c.TokenCount,
			sq.Expr("NOW()"),
		)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build chunk insert query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute chunk insert: %w", err)
	}

	return nil
}

// ChunkCreateBatch inserts multiple chunks with their embedding vectors within a transaction
func (h *handler) ChunkCreateBatch(ctx context.Context, chunks []*chunk.Chunk, embeddings [][]float32) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunks and embeddings length mismatch: %d != %d", len(chunks), len(embeddings))
	}

	if len(chunks) == 0 {
		return nil
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for i, c := range chunks {
		embStr := formatEmbedding(embeddings[i])

		q := psql.Insert(tableRagChunks).
			Columns(
				"id",
				"document_id",
				"rag_id",
				"customer_id",
				"chunk_index",
				"text",
				"section_title",
				"embedding",
				"token_count",
				"tm_create",
			).
			Values(
				c.ID.Bytes(),
				c.DocumentID.Bytes(),
				c.RagID.Bytes(),
				c.CustomerID.Bytes(),
				c.ChunkIndex,
				c.Text,
				c.SectionTitle,
				sq.Expr("?::vector", embStr),
				c.TokenCount,
				sq.Expr("NOW()"),
			)

		sqlStr, args, err := q.ToSql()
		if err != nil {
			return fmt.Errorf("could not build chunk batch insert query for index %d: %w", i, err)
		}

		_, err = tx.ExecContext(ctx, sqlStr, args...)
		if err != nil {
			return fmt.Errorf("could not execute chunk batch insert for index %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit chunk batch insert: %w", err)
	}

	return nil
}

// ChunkSearchByRagID performs vector similarity search within a rag.
// It returns the matching chunks and their relevance scores (1 - cosine_distance).
// Raw SQL is used here because squirrel cannot express the pgvector <=> (cosine distance) operator.
func (h *handler) ChunkSearchByRagID(ctx context.Context, ragID uuid.UUID, queryEmbedding []float32, topK int) ([]*chunk.Chunk, []float64, error) {
	embStr := formatEmbedding(queryEmbedding)

	// Raw SQL required: squirrel cannot express the pgvector <=> operator for cosine distance.
	query := `SELECT id, document_id, rag_id, customer_id, chunk_index, text, section_title, token_count, tm_create,
                     embedding <=> $1::vector AS distance
              FROM rag_chunks
              WHERE rag_id = $2 AND tm_delete IS NULL
              ORDER BY embedding <=> $1::vector
              LIMIT $3`

	rows, err := h.db.QueryContext(ctx, query, embStr, ragID.Bytes(), topK)
	if err != nil {
		return nil, nil, fmt.Errorf("could not execute chunk search query: %w", err)
	}
	defer rows.Close()

	chunks := []*chunk.Chunk{}
	scores := []float64{}

	for rows.Next() {
		var c chunk.Chunk
		var idBytes, documentIDBytes, ragIDBytes, customerIDBytes []byte
		var distance float64

		err := rows.Scan(
			&idBytes,
			&documentIDBytes,
			&ragIDBytes,
			&customerIDBytes,
			&c.ChunkIndex,
			&c.Text,
			&c.SectionTitle,
			&c.TokenCount,
			&c.TMCreate,
			&distance,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("could not scan chunk search row: %w", err)
		}

		c.ID, _ = uuid.FromBytes(idBytes)
		c.DocumentID, _ = uuid.FromBytes(documentIDBytes)
		c.RagID, _ = uuid.FromBytes(ragIDBytes)
		c.CustomerID, _ = uuid.FromBytes(customerIDBytes)

		chunks = append(chunks, &c)
		scores = append(scores, 1-distance)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("could not iterate chunk search rows: %w", err)
	}

	return chunks, scores, nil
}

// ChunkDeleteByDocumentID hard-deletes all chunks for a document
func (h *handler) ChunkDeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error {
	q := psql.Delete(tableRagChunks).
		Where(sq.Eq{"document_id": documentID.Bytes()})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build chunk delete by document query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute chunk delete by document: %w", err)
	}

	return nil
}

// ChunkDeleteByRagID hard-deletes all chunks for a rag
func (h *handler) ChunkDeleteByRagID(ctx context.Context, ragID uuid.UUID) error {
	q := psql.Delete(tableRagChunks).
		Where(sq.Eq{"rag_id": ragID.Bytes()})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build chunk delete by rag query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute chunk delete by rag: %w", err)
	}

	return nil
}

// ChunkSoftDeleteByDocumentID soft-deletes all chunks for a document
func (h *handler) ChunkSoftDeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error {
	q := psql.Update(tableRagChunks).
		Set("tm_delete", sq.Expr("NOW()")).
		Where(sq.Eq{"document_id": documentID.Bytes()}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build chunk soft delete by document query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute chunk soft delete by document: %w", err)
	}

	return nil
}

// ChunkSoftDeleteByRagID soft-deletes all chunks for a rag
func (h *handler) ChunkSoftDeleteByRagID(ctx context.Context, ragID uuid.UUID) error {
	q := psql.Update(tableRagChunks).
		Set("tm_delete", sq.Expr("NOW()")).
		Where(sq.Eq{"rag_id": ragID.Bytes()}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build chunk soft delete by rag query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute chunk soft delete by rag: %w", err)
	}

	return nil
}
