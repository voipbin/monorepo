package confighandler

//go:generate mockgen -package confighandler -destination ./mock_store.go -source store.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
)

// Row is one stored key. A row holds either a value (ValueJSON) or a secret (SecretCiphertext).
// The type has no db tags: a Store converts it explicitly.
type Row struct {
	Key              string
	ValueType        string // "secret" for KindSecret, the registry Type otherwise
	ValueJSON        []byte
	SecretCiphertext []byte // nonce || ciphertext || tag
	KeyVersion       int    // 0 for a row that is not a secret
	Revision         uint32
	UpdatedBy        *uuid.UUID
	TMCreate         time.Time
	TMUpdate         time.Time
}

// History is one change. For a secret or sensitive key IsSecret is true and both value fields are
// nil: the library has no code path that writes a value there.
type History struct {
	ID            uuid.UUID
	Key           string
	Action        string
	IsSecret      bool
	OldValueJSON  []byte
	NewValueJSON  []byte
	OldKeyVersion int
	NewKeyVersion int
	ActorID       uuid.UUID
	Source        string
	TMCreate      time.Time
}

// HistoryCursor marks the last history row of a page. The zero value means the first page.
type HistoryCursor struct {
	TMCreate time.Time
	ID       uuid.UUID
}

// Store is implemented by each service on its own configs tables.
// Every write stores the row change and the history row in one transaction.
type Store interface {
	// Get returns ErrNotFound when the row does not exist.
	Get(ctx context.Context, key string) (*Row, error)

	List(ctx context.Context) ([]*Row, error)

	// Insert returns ErrConflict when the key exists already.
	Insert(ctx context.Context, row *Row, hist *History) error

	// Update changes the row when its revision is expectRevision and sets the revision to
	// expectRevision+1. It returns ErrConflict when no row matches.
	Update(ctx context.Context, row *Row, expectRevision uint32, hist *History) error

	// Delete removes the row when its revision is expectRevision. It returns ErrConflict when no
	// row matches.
	Delete(ctx context.Context, key string, expectRevision uint32, hist *History) error

	// ListHistories returns the history rows of the key, ordered by TMCreate and ID descending,
	// that come after the cursor (strictly older, or the same time with a smaller ID). At most
	// limit rows. A zero cursor starts at the newest row.
	ListHistories(ctx context.Context, key string, after HistoryCursor, limit int) ([]*History, error)
}
