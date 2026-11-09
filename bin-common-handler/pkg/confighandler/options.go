package confighandler

// writeOpts is the setting of one write.
type writeOpts struct {
	expectSet bool
	expectRev uint32
}

// WriteOption configures one write of Set, SetSecret or Clear.
type WriteOption func(*writeOpts)

// ExpectRevision makes the write succeed only when the stored revision is rev (0 means: the key
// has no row). The write is tried once and never retried; a mismatch, or a change between the read
// and the write, returns ErrStale.
//
// The revision restarts at 1 when a key is cleared and set again, so a writer that holds an
// old revision 1 is not detected after such a recreate. Both changes stay in the history.
func ExpectRevision(rev uint32) WriteOption {
	return func(o *writeOpts) {
		o.expectSet = true
		o.expectRev = rev
	}
}

func applyWriteOptions(opts []WriteOption) writeOpts {
	var o writeOpts
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}

	return o
}
