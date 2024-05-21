package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"monorepo/bin-storage-manager/models/file"
	"strconv"

	"github.com/gofrs/uuid"
)

const (
	// select query for flow get
	fileSelect = `
	select
		id,
		customer_id,
		owner_id,

		reference_type,
		reference_id,

		name,
		detail,
		filename,

		bucket_name,
		filepath,

		uri_bucket,
		uri_download,

		tm_download_expire,
		tm_create,
		tm_update,
		tm_delete
	from
		storage_files
	`
)

// fileGetFromRow gets the file from the row.
func (h *handler) fileGetFromRow(row *sql.Rows) (*file.File, error) {

	res := &file.File{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.OwnerID,

		&res.ReferenceType,
		&res.ReferenceID,

		&res.Name,
		&res.Detail,
		&res.Filename,

		&res.BucketName,
		&res.Filepath,

		&res.URIBucket,
		&res.URIDownload,

		&res.TMDownloadExpire,
		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. fileGetFromRow. err: %v", err)
	}

	return res, nil
}

// FileCreate creates a new file row
func (h *handler) FileCreate(ctx context.Context, f *file.File) error {

	q := `insert into storage_files(
		id,
		customer_id,
		owner_id,

		reference_type,
		reference_id,

		name,
		detail,
		filename,

		bucket_name,
		filepath,

		uri_bucket,
		uri_download,

		tm_download_expire,
        tm_create,
        tm_update,
        tm_delete
	) values(
		?, ?, ?,
		?, ?,
		?, ?, ?,
		?, ?,
		?, ?,
		?, ?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. FileCreate. err: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		f.ID.Bytes(),
		f.CustomerID.Bytes(),
		f.OwnerID.Bytes(),

		f.ReferenceType,
		f.ReferenceID.Bytes(),

		f.Name,
		f.Detail,
		f.Filename,

		f.BucketName,
		f.Filepath,

		f.URIBucket,
		f.URIDownload,

		f.TMDownloadExpire,
		h.util.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. FileCreate. err: %v", err)
	}

	_ = h.fileUpdateToCache(ctx, f.ID)

	return nil
}

// fileUpdateToCache gets the flow from the DB and update the cache.
func (h *handler) fileUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.fileGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.fileSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// fileSetToCache sets the given file to the cache
func (h *handler) fileSetToCache(ctx context.Context, f *file.File) error {
	if err := h.cache.FileSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// fileGetFromCache returns file from the cache if possible.
func (h *handler) fileGetFromCache(ctx context.Context, id uuid.UUID) (*file.File, error) {

	// get from cache
	res, err := h.cache.FileGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// fileDeleteCache deletes cache
func (h *handler) fileDeleteCache(ctx context.Context, id uuid.UUID) error {

	// delete from cache
	err := h.cache.FileDel(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

// fileGetFromDB gets the file info from the db.
func (h *handler) fileGetFromDB(ctx context.Context, id uuid.UUID) (*file.File, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", fileSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. fileGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. fileGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.fileGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// FileGet returns file.
func (h *handler) FileGet(ctx context.Context, id uuid.UUID) (*file.File, error) {

	res, err := h.fileGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.fileGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.fileSetToCache(ctx, res)

	return res, nil
}

// FileGets returns files.
func (h *handler) FileGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*file.File, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, fileSelect)

	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "owner_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))
	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. FileGets. err: %v", err)
	}
	defer rows.Close()

	res := []*file.File{}
	for rows.Next() {
		u, err := h.fileGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. FileGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// FileUpdate updates the most of file information.
// except permenant info(i.e. id, timestamp, etc)
func (h *handler) FileUpdate(ctx context.Context, id uuid.UUID, name, detail string) error {
	q := `
	update storage_files set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, name, detail, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. FileUpdate. err: %v", err)
	}

	// set to the cache
	_ = h.fileUpdateToCache(ctx, id)

	return nil
}

// FileDelete deletes the given flow
func (h *handler) FileDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update storage_files set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.util.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. FileDelete. err: %v", err)
	}

	// delete cache
	_ = h.fileDeleteCache(ctx, id)

	return nil
}

// FileUpdateDownloadInfo updates the file's download info.
func (h *handler) FileUpdateDownloadInfo(ctx context.Context, id uuid.UUID, uriDownload string, tmDownloadExpire string) error {
	q := `
	update storage_files set
		uri_download = ?,
		tm_download_expire = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, uriDownload, tmDownloadExpire, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. FileUpdateDownloadInfo. err: %v", err)
	}

	// set to the cache
	_ = h.fileUpdateToCache(ctx, id)

	return nil
}
