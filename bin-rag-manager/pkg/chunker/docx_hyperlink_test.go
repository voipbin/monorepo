package chunker

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fumiama/go-docx"
)

func Test_DOCXChunker_Chunk_nonRunParagraphChild(t *testing.T) {
	d := docx.New().WithDefaultTheme()
	p := d.AddParagraph()
	p.AddText("before ")
	p.AddLink("linkword", "https://example.test/page")
	p.AddText(" after")

	var buf bytes.Buffer
	if _, err := d.WriteTo(&buf); err != nil {
		t.Fatalf("could not build docx: %v", err)
	}
	path := covWriteFile(t, "link.docx", buf.Bytes())

	chunks, err := NewDOCXChunker().Chunk(path, 800)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("Wrong match. expect one chunk, got: %d", len(chunks))
	}

	// the paragraph children that are not text runs are skipped, the runs around them are kept in order
	if got := strings.TrimSpace(chunks[0].Text); got != "before  after" {
		t.Errorf("Wrong match. expect: %q, got: %q", "before  after", got)
	}
}
