package chunker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/fumiama/go-docx"
)

// covWriteFile writes content into a new file under a per-test temp dir.
func covWriteFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatalf("could not write fixture: %v", err)
	}
	return p
}

// covBuildDOCX builds a docx whose body has one paragraph per entry; an empty entry is an empty paragraph.
func covBuildDOCX(t *testing.T, paragraphs []string, withTable bool) string {
	t.Helper()

	d := docx.New().WithDefaultTheme()
	for _, text := range paragraphs {
		p := d.AddParagraph()
		if text != "" {
			p.AddText(text)
		}
	}
	if withTable {
		d.AddTable(1, 1, 0, nil)
	}

	var buf bytes.Buffer
	if _, err := d.WriteTo(&buf); err != nil {
		t.Fatalf("could not build docx: %v", err)
	}
	return covWriteFile(t, "fixture.docx", buf.Bytes())
}

// covBuildPDF builds a minimal valid PDF with one page per entry, using the Helvetica base font.
// Page text must not contain parentheses or backslashes.
func covBuildPDF(t *testing.T, pages []string) string {
	t.Helper()

	var buf bytes.Buffer
	var offsets []int
	addObj := func(body string) {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", len(offsets), body)
	}

	buf.WriteString("%PDF-1.4\n")

	// object numbers: 1 catalog, 2 pages, 3 font, then (page, content) pairs starting at 4.
	var kids []string
	for i := range pages {
		kids = append(kids, fmt.Sprintf("%d 0 R", 4+2*i))
	}
	addObj("<< /Type /Catalog /Pages 2 0 R >>")
	addObj(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)))
	addObj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	for i, text := range pages {
		addObj(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", 5+2*i))
		stream := ""
		if text != "" {
			stream = fmt.Sprintf("BT /F1 12 Tf 72 720 Td (%s) Tj ET", text)
		}
		addObj(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}

	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xrefOffset)

	return covWriteFile(t, "fixture.pdf", buf.Bytes())
}

func covTitles(chunks []Chunk) []string {
	res := make([]string, 0, len(chunks))
	for _, c := range chunks {
		res = append(res, c.SectionTitle)
	}
	return res
}

func covTexts(chunks []Chunk) []string {
	res := make([]string, 0, len(chunks))
	for _, c := range chunks {
		res = append(res, c.Text)
	}
	return res
}

func Test_docxChunker_Chunk(t *testing.T) {
	tests := []struct {
		name string

		paragraphs []string
		withTable  bool
		maxTokens  int

		expectTitles []string
		expectTexts  []string
	}{
		{
			name:         "paragraphs are joined with newlines",
			paragraphs:   []string{"alpha one", "beta two"},
			maxTokens:    800,
			expectTitles: []string{"Part 1"},
			expectTexts:  []string{"alpha one\nbeta two\n"},
		},
		{
			name:         "non paragraph body items are ignored",
			paragraphs:   []string{"alpha one"},
			withTable:    true,
			maxTokens:    800,
			expectTitles: []string{"Part 1"},
			expectTexts:  []string{"alpha one\n"},
		},
		{
			name:         "blank paragraph splits when over the token limit",
			paragraphs:   []string{"aaaaaaaaaa", "", "bbbbbbbbbb"},
			maxTokens:    3,
			expectTitles: []string{"Part 1", "Part 2"},
			expectTexts:  []string{"aaaaaaaaaa", "bbbbbbbbbb\n"},
		},
		{
			name:         "text exactly at the limit is not split",
			paragraphs:   []string{"aaaaaaaaaa", "", "bbbbbbbbbbb"},
			maxTokens:    6,
			expectTitles: []string{"Part 1"},
			expectTexts:  []string{"aaaaaaaaaa\n\nbbbbbbbbbbb\n"},
		},
		{
			name:         "only blank paragraphs yields nothing",
			paragraphs:   []string{"", ""},
			maxTokens:    800,
			expectTitles: nil,
			expectTexts:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := covBuildDOCX(t, tt.paragraphs, tt.withTable)

			res, err := NewDOCXChunker().Chunk(path, tt.maxTokens)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if tt.expectTexts == nil {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}

			if !reflect.DeepEqual(covTitles(res), tt.expectTitles) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectTitles, covTitles(res))
			}
			if !reflect.DeepEqual(covTexts(res), tt.expectTexts) {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectTexts, covTexts(res))
			}

			ids := map[string]bool{}
			for _, c := range res {
				if c.SourceFile != path {
					t.Errorf("Wrong match. expect: %v, got: %v", path, c.SourceFile)
				}
				if c.DocType != DocTypeDevDoc {
					t.Errorf("Wrong match. expect: %v, got: %v", DocTypeDevDoc, c.DocType)
				}
				if c.ID == "" || ids[c.ID] {
					t.Errorf("Wrong match. expect: unique non-empty id, got: %q", c.ID)
				}
				ids[c.ID] = true
			}
		})
	}
}

func Test_docxChunker_Chunk_notADocx(t *testing.T) {
	path := covWriteFile(t, "fixture.docx", []byte("this is not a zip archive"))

	res, err := NewDOCXChunker().Chunk(path, 800)
	if err == nil || !strings.Contains(err.Error(), "could not parse DOCX") {
		t.Errorf("Wrong match. expect: could not parse DOCX error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_pdfChunker_Chunk(t *testing.T) {
	tests := []struct {
		name string

		pages     []string
		maxTokens int

		expectNil    bool
		expectTitles []string
		expectHas    [][]string
	}{
		{
			name:         "single page text is extracted",
			pages:        []string{"Hello world"},
			maxTokens:    800,
			expectTitles: []string{"Page group 1"},
			expectHas:    [][]string{{"Hello world"}},
		},
		{
			name:         "pages are concatenated into one group under the limit",
			pages:        []string{"First page", "Second page"},
			maxTokens:    800,
			expectTitles: []string{"Page group 1"},
			expectHas:    [][]string{{"First page", "Second page"}},
		},
		{
			name:         "pages split into groups over the limit",
			pages:        []string{"First page", "Second page"},
			maxTokens:    3,
			expectTitles: []string{"Page group 1", "Page group 2"},
			expectHas:    [][]string{{"First page"}, {"Second page"}},
		},
		{
			name:      "page without text yields nothing",
			pages:     []string{""},
			maxTokens: 800,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := covBuildPDF(t, tt.pages)

			res, err := NewPDFChunker().Chunk(path, tt.maxTokens)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if tt.expectNil {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}

			if !reflect.DeepEqual(covTitles(res), tt.expectTitles) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectTitles, covTitles(res))
			}
			for i, c := range res {
				for _, want := range tt.expectHas[i] {
					if !strings.Contains(c.Text, want) {
						t.Errorf("Wrong match. expect chunk %d to contain: %q, got: %q", i, want, c.Text)
					}
				}
				if c.SourceFile != path || c.DocType != DocTypeDevDoc {
					t.Errorf("Wrong match. expect: %v/%v, got: %v/%v", path, DocTypeDevDoc, c.SourceFile, c.DocType)
				}
			}
			// groups must not leak text of other groups
			if len(res) == 2 && strings.Contains(res[0].Text, "Second") {
				t.Errorf("Wrong match. expect: group 1 without the second page, got: %q", res[0].Text)
			}
		})
	}
}

func Test_pdfChunker_Chunk_notAPDF(t *testing.T) {
	path := covWriteFile(t, "fixture.pdf", []byte("this is not a pdf"))

	res, err := NewPDFChunker().Chunk(path, 800)
	if err == nil || !strings.Contains(err.Error(), "could not open PDF") {
		t.Errorf("Wrong match. expect: could not open PDF error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

// covJSONBoundaryObject returns a single-key object file whose indented key text is exactly 4*tokens bytes long.
func covJSONBoundaryObject(t *testing.T) (path string, tokens int, textLen int) {
	t.Helper()

	for n := 1; n <= 4; n++ {
		b, err := json.MarshalIndent(map[string]any{"k": strings.Repeat("x", n)}, "", "  ")
		if err != nil {
			t.Fatalf("could not marshal: %v", err)
		}
		if len(b)%4 == 0 {
			raw, _ := json.Marshal(map[string]any{"k": strings.Repeat("x", n)})
			return covWriteFile(t, "fixture.json", raw), len(b) / 4, len(b)
		}
	}
	t.Fatal("could not build a boundary fixture")
	return "", 0, 0
}

func Test_jsonChunker_Chunk_objectBoundary(t *testing.T) {
	path, tokens, textLen := covJSONBoundaryObject(t)

	tests := []struct {
		name         string
		maxTokens    int
		expectTitles []string
	}{
		{"text exactly at the limit stays whole", tokens, []string{"Key: k"}},
		{"text one token over the limit is routed to the split path", tokens - 1, []string{"Key: k (part 1)"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NewJSONChunker().Chunk(path, tt.maxTokens)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(covTitles(res), tt.expectTitles) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectTitles, covTitles(res))
			}
			// the whole indented key text is kept in a single chunk in both cases
			if len(res) != 1 || len(res[0].Text) != textLen {
				t.Errorf("Wrong match. expect: one chunk of %d bytes, got: %v", textLen, covTexts(res))
			}
			if res[0].DocType != DocTypeDevDoc || res[0].SourceFile != path {
				t.Errorf("Wrong match. expect: %v/%v, got: %v/%v", DocTypeDevDoc, path, res[0].DocType, res[0].SourceFile)
			}
		})
	}
}

func Test_jsonChunker_Chunk_objectKeys(t *testing.T) {
	path := covWriteFile(t, "fixture.json", []byte(`{"alpha":1,"beta":{"x":2}}`))

	res, err := NewJSONChunker().Chunk(path, 800)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// map iteration order is random, so compare as a sorted set.
	titles := covTitles(res)
	sort.Strings(titles)
	expectTitles := []string{"Key: alpha", "Key: beta"}
	if !reflect.DeepEqual(titles, expectTitles) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectTitles, titles)
	}

	texts := covTexts(res)
	sort.Strings(texts)
	expectTexts := []string{"{\n  \"alpha\": 1\n}", "{\n  \"beta\": {\n    \"x\": 2\n  }\n}"}
	if !reflect.DeepEqual(texts, expectTexts) {
		t.Errorf("Wrong match. expect: %q, got: %q", expectTexts, texts)
	}
}

func Test_jsonChunker_Chunk_arraySplit(t *testing.T) {
	tests := []struct {
		name string

		content   string
		maxTokens int

		expectTitles []string
		expectTexts  []string
	}{
		{
			// each item marshals to 6 bytes; the first two fill the 12 byte limit exactly and the third starts a new part.
			name:         "items fill the limit exactly before the next part starts",
			content:      `["aaaa","bbbb","cccc"]`,
			maxTokens:    3,
			expectTitles: []string{"Items (part 1)", "Items (part 2)"},
			expectTexts:  []string{"[\n  \"aaaa\",\n  \"bbbb\"\n]", "[\n  \"cccc\"\n]"},
		},
		{
			name:         "all items fit in one part",
			content:      `["aaaa","bbbb"]`,
			maxTokens:    800,
			expectTitles: []string{"Items (part 1)"},
			expectTexts:  []string{"[\n  \"aaaa\",\n  \"bbbb\"\n]"},
		},
		{
			name:         "one item larger than the limit still becomes its own part",
			content:      `["aaaaaaaaaaaaaaaaaaaa","b"]`,
			maxTokens:    3,
			expectTitles: []string{"Items (part 1)", "Items (part 2)"},
			expectTexts:  []string{"[\n  \"aaaaaaaaaaaaaaaaaaaa\"\n]", "[\n  \"b\"\n]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := covWriteFile(t, "fixture.json", []byte(tt.content))

			res, err := NewJSONChunker().Chunk(path, tt.maxTokens)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(covTitles(res), tt.expectTitles) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectTitles, covTitles(res))
			}
			if !reflect.DeepEqual(covTexts(res), tt.expectTexts) {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectTexts, covTexts(res))
			}
		})
	}
}

func Test_jsonChunker_Chunk_notObjectOrArray(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		maxTokens    int
		expectTitles []string
		expectTexts  []string
	}{
		{"scalar json falls back to plain text", `42`, 800, []string{"Part 1"}, []string{"42"}},
		{"invalid json falls back to plain text", "not json at all", 800, []string{"Part 1"}, []string{"not json at all"}},
		{
			name:         "plain text fallback splits on blank lines",
			content:      "aaaaaaaaaa\n\nbbbbbbbbbb",
			maxTokens:    3,
			expectTitles: []string{"Part 1", "Part 2"},
			expectTexts:  []string{"aaaaaaaaaa", "bbbbbbbbbb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := covWriteFile(t, "fixture.json", []byte(tt.content))

			res, err := NewJSONChunker().Chunk(path, tt.maxTokens)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(covTitles(res), tt.expectTitles) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectTitles, covTitles(res))
			}
			if !reflect.DeepEqual(covTexts(res), tt.expectTexts) {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectTexts, covTexts(res))
			}
		})
	}
}

func Test_chunkers_readErrors(t *testing.T) {
	// a directory can be opened but not read, so each chunker reports its own read or parse failure.
	tests := []struct {
		name      string
		chunker   Chunker
		expectErr string
	}{
		{"json", NewJSONChunker(), "could not read file"},
		{"markdown", NewMarkdownChunker(DocTypeDevDoc), "could not read file"},
		{"rst", NewRSTChunker(), "could not read file"},
		{"html", NewHTMLChunker(), "could not parse HTML"},
		{"csv", NewCSVChunker(), "could not parse CSV"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.chunker.Chunk(t.TempDir(), 800)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %q error, got: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_csvChunker_Chunk(t *testing.T) {
	tests := []struct {
		name string

		content   string
		maxTokens int

		expectErr    string
		expectNil    bool
		expectTitles []string
		expectTexts  []string
	}{
		{
			name:      "unterminated quote is a parse error",
			content:   "a,\"b\n",
			maxTokens: 800,
			expectErr: "could not parse CSV",
		},
		{
			name:      "empty file yields nothing",
			content:   "",
			maxTokens: 800,
			expectNil: true,
		},
		{
			name:         "header only yields no chunks",
			content:      "id,name\n",
			maxTokens:    800,
			expectTitles: []string{},
			expectTexts:  []string{},
		},
		{
			// header "h" is 2 bytes with the newline and each row 5 bytes, so two rows fill the 12 byte limit exactly.
			name:         "rows split into parts with the header repeated",
			content:      "h\naaaa\nbbbb\ncccc\n",
			maxTokens:    3,
			expectTitles: []string{"Rows (part 1)", "Rows (part 2)"},
			expectTexts:  []string{"h\naaaa\nbbbb\n", "h\ncccc\n"},
		},
		{
			name:         "all rows in one part",
			content:      "h\naaaa\nbbbb\n",
			maxTokens:    800,
			expectTitles: []string{"Rows (part 1)"},
			expectTexts:  []string{"h\naaaa\nbbbb\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := covWriteFile(t, "fixture.csv", []byte(tt.content))

			res, err := NewCSVChunker().Chunk(path, tt.maxTokens)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect: %q error, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if tt.expectNil {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}
			if !reflect.DeepEqual(covTitles(res), tt.expectTitles) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectTitles, covTitles(res))
			}
			if !reflect.DeepEqual(covTexts(res), tt.expectTexts) {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectTexts, covTexts(res))
			}
		})
	}
}

func Test_rstChunker_Chunk_sections(t *testing.T) {
	tests := []struct {
		name string

		content string

		expectTitles []string
		expectTexts  []string
	}{
		{
			name:         "overline title closes the previous section",
			content:      "intro\n\n=====\nTitle\n=====\ntext body\n",
			expectTitles: []string{"Introduction", "Title"},
			expectTexts:  []string{"intro\n", "text body\n"},
		},
		{
			name:         "section with only blank lines is skipped",
			content:      "A\n===\n\nB\n===\ntext\n",
			expectTitles: []string{"B"},
			expectTexts:  []string{"text\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := covWriteFile(t, "fixture.rst", []byte(tt.content))

			res, err := NewRSTChunker().Chunk(path, 800)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(covTitles(res), tt.expectTitles) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectTitles, covTitles(res))
			}
			if !reflect.DeepEqual(covTexts(res), tt.expectTexts) {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectTexts, covTexts(res))
			}
		})
	}
}

func Test_htmlChunker_Chunk(t *testing.T) {
	tests := []struct {
		name string

		content string

		expectNil   bool
		expectTexts []string
	}{
		{
			name:        "script and style content is dropped",
			content:     "<html><body><p>Hello</p><script>var secret = 1;</script><style>p { color: red; }</style><p>World</p></body></html>",
			expectTexts: []string{"Hello\nWorld\n"},
		},
		{
			name:      "document with only a script yields nothing",
			content:   "<html><body><script>var a = 1;</script></body></html>",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := covWriteFile(t, "fixture.html", []byte(tt.content))

			res, err := NewHTMLChunker().Chunk(path, 800)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if tt.expectNil {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}
			if !reflect.DeepEqual(covTexts(res), tt.expectTexts) {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectTexts, covTexts(res))
			}
		})
	}
}

func Test_markdownChunker_Chunk_docTypeAndParts(t *testing.T) {
	// the section body is two 10 byte paragraphs; with a 12 byte limit they land in separate parts.
	path := covWriteFile(t, "fixture.md", []byte("# Title\n\naaaaaaaaaa\n\nbbbbbbbbbb\n"))

	res, err := NewMarkdownChunker(DocTypeDesign).Chunk(path, 3)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) < 2 {
		t.Fatalf("Wrong match. expect: at least two parts, got: %q", covTexts(res))
	}
	for i, c := range res {
		expectTitle := fmt.Sprintf("Title (part %d)", i+1)
		if c.SectionTitle != expectTitle {
			t.Errorf("Wrong match. expect: %v, got: %v", expectTitle, c.SectionTitle)
		}
		if c.DocType != DocTypeDesign {
			t.Errorf("Wrong match. expect: %v, got: %v", DocTypeDesign, c.DocType)
		}
	}
}

func Test_textChunker_Chunk_blank(t *testing.T) {
	path := covWriteFile(t, "fixture.txt", []byte(" \n\t\n"))

	res, err := NewTextChunker().Chunk(path, 800)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_openapiChunker_Chunk_nonMapPathItem(t *testing.T) {
	spec := "openapi: 3.0.0\npaths:\n  /broken: not-a-map\n  /ok:\n    get:\n      summary: List things\n"
	path := covWriteFile(t, "fixture.yaml", []byte(spec))

	res, err := NewOpenAPIChunker().Chunk(path, 800)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual(covTitles(res), []string{"GET /ok"}) {
		t.Errorf("Wrong match. expect: %v, got: %v", []string{"GET /ok"}, covTitles(res))
	}
	if len(res) == 1 && (!strings.Contains(res[0].Text, "Summary: List things") || res[0].DocType != DocTypeOpenAPI) {
		t.Errorf("Wrong match. expect: openapi chunk with the summary, got: %v/%q", res[0].DocType, res[0].Text)
	}
}

func Test_rstChunker_Chunk_tokenLimit(t *testing.T) {
	tests := []struct {
		name string

		content   string
		maxTokens int

		expectTitles []string
		expectTexts  []string
	}{
		{
			// the section text is 22 bytes, which is within the 24 byte limit.
			name:         "section within the limit keeps its title",
			content:      "A\n===\naaaaaaaaaa\n\nbbbbbbbbbb",
			maxTokens:    6,
			expectTitles: []string{"A"},
			expectTexts:  []string{"aaaaaaaaaa\n\nbbbbbbbbbb"},
		},
		{
			// the text is 22 bytes and the limit is 20, so the second paragraph starts a new part.
			name:         "section over the limit is split into numbered parts",
			content:      "A\n===\naaaaaaaaaa\n\nbbbbbbbbbb",
			maxTokens:    5,
			expectTitles: []string{"A (part 1)", "A (part 2)"},
			expectTexts:  []string{"aaaaaaaaaa", "bbbbbbbbbb"},
		},
		{
			// three 4 byte paragraphs plus separators fill the 16 byte limit exactly; the fourth starts a new part.
			name:         "paragraphs filling the limit exactly stay together",
			content:      "A\n===\naaaa\n\nbbbb\n\ncccc\n\ndddd",
			maxTokens:    4,
			expectTitles: []string{"A (part 1)", "A (part 2)"},
			expectTexts:  []string{"aaaa\n\nbbbb\n\ncccc", "dddd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := covWriteFile(t, "fixture.rst", []byte(tt.content))

			res, err := NewRSTChunker().Chunk(path, tt.maxTokens)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(covTitles(res), tt.expectTitles) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectTitles, covTitles(res))
			}
			if !reflect.DeepEqual(covTexts(res), tt.expectTexts) {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectTexts, covTexts(res))
			}
		})
	}
}
