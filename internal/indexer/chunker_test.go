package indexer

import (
	"context"
	"testing"
)

func TestDefaultChunker_ChunkGo(t *testing.T) {
	chunker := NewDefaultChunker()
	ctx := context.Background()

	content := `package main

import "fmt"

// Hello says hello
func Hello() {
	fmt.Println("Hello")
}

type User struct {
	Name string
}

func (u *User) Greet() {
	fmt.Printf("Hi, I'm %s\n", u.Name)
}
`
	file := FileInfo{
		Path: "main.go",
		Lang: LangGo,
	}

	chunks, symbols, err := chunker.Chunk(ctx, file, []byte(content))
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	// Expected: Hello func, User type, User.Greet method
	expectedSymbols := []string{"Hello", "User", "(*User).Greet"}

	if len(symbols) != len(expectedSymbols) {
		t.Errorf("Got %d symbols, want %d", len(symbols), len(expectedSymbols))
	}

	for i, sym := range symbols {
		if i < len(expectedSymbols) && sym.Name != expectedSymbols[i] {
			t.Errorf("Symbol[%d] name = %s, want %s", i, sym.Name, expectedSymbols[i])
		}
	}

	if len(chunks) != len(expectedSymbols) {
		t.Errorf("Got %d chunks, want %d", len(chunks), len(expectedSymbols))
	}
}

func TestDefaultChunker_ChunkPython(t *testing.T) {
	chunker := NewDefaultChunker()
	ctx := context.Background()

	content := `
def hello():
    print("Hello")

class User:
    def __init__(self, name):
        self.name = name

    def greet(self):
        print(f"Hi, I'm {self.name}")
`
	file := FileInfo{
		Path: "main.py",
		Lang: LangPython,
	}

	chunks, symbols, err := chunker.Chunk(ctx, file, []byte(content))
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Got 0 chunks, expected some")
	}
	// It checks indent == 0. So nested methods won't be picked up as top-level chunks.

	expectedSymbols := []string{"hello", "User"}

	if len(symbols) != len(expectedSymbols) {
		t.Errorf("Got %d symbols, want %d", len(symbols), len(expectedSymbols))
	}

	for i, sym := range symbols {
		if i < len(expectedSymbols) && sym.Name != expectedSymbols[i] {
			t.Errorf("Symbol[%d] name = %s, want %s", i, sym.Name, expectedSymbols[i])
		}
	}
}

func TestDefaultChunker_ChunkMarkdown(t *testing.T) {
	chunker := NewDefaultChunker()
	ctx := context.Background()

	content := `# Title

Introduction text.

## Section 1
Content 1.

## Section 2
Content 2.
`
	file := FileInfo{
		Path: "README.md",
		Lang: LangMarkdown,
	}

	chunks, symbols, err := chunker.Chunk(ctx, file, []byte(content))
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Got 0 chunks, expected some")
	}

	// Expected: Title, Section 1, Section 2
	expectedSymbols := []string{"Title", "Section 1", "Section 2"}

	if len(symbols) != len(expectedSymbols) {
		t.Errorf("Got %d symbols, want %d", len(symbols), len(expectedSymbols))
	}

	for i, sym := range symbols {
		if i < len(expectedSymbols) && sym.Name != expectedSymbols[i] {
			t.Errorf("Symbol[%d] name = %s, want %s", i, sym.Name, expectedSymbols[i])
		}
	}
}

func TestDefaultChunker_ChunkParagraphs(t *testing.T) {
	chunker := NewDefaultChunker()
	ctx := context.Background()

	content := `Para 1 line 1.
Para 1 line 2.

Para 2 line 1.

Para 3.
`
	file := FileInfo{
		Path: "text.txt",
		Lang: "text",
	}

	chunks, _, err := chunker.Chunk(ctx, file, []byte(content))
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks) != 3 {
		t.Errorf("Got %d chunks, want 3", len(chunks))
	}
}
