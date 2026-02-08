package chunker

import (
	"os"
	"testing"
)

func TestMarkdownChunker_Chunk(t *testing.T) {
	content := `# Main Title

Some intro text.

## Getting Started

How to get started with the project.

## API Reference

API endpoint documentation.

### Sub-section

This is a sub-section under API Reference.
`

	tmpFile, err := os.CreateTemp("", "test_*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewMarkdownChunker(DocTypeDesign)
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify doc type
	for _, chunk := range chunks {
		if chunk.DocType != DocTypeDesign {
			t.Errorf("expected doc type design, got %s", chunk.DocType)
		}
	}

	// Verify section titles
	titles := make(map[string]bool)
	for _, chunk := range chunks {
		titles[chunk.SectionTitle] = true
	}
	if !titles["Getting Started"] {
		t.Error("expected 'Getting Started' section")
	}
	if !titles["API Reference"] {
		t.Error("expected 'API Reference' section")
	}
}

func TestMarkdownChunker_GuidelineDocType(t *testing.T) {
	content := `# CLAUDE.md

## Overview

This is a guideline document.
`

	tmpFile, err := os.CreateTemp("", "test_*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewMarkdownChunker(DocTypeGuideline)
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	for _, chunk := range chunks {
		if chunk.DocType != DocTypeGuideline {
			t.Errorf("expected doc type guideline, got %s", chunk.DocType)
		}
	}
}
