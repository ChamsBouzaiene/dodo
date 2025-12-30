package indexer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

// DefaultChunker provides basic chunking for various languages.
type DefaultChunker struct{}

// NewDefaultChunker creates a new default chunker.
func NewDefaultChunker() *DefaultChunker {
	return &DefaultChunker{}
}

// Chunk splits a file into semantic chunks based on language.
func (c *DefaultChunker) Chunk(ctx context.Context, file FileInfo, content []byte) ([]Chunk, []Symbol, error) {
	switch file.Lang {
	case LangGo:
		return c.chunkGo(file, content)
	case LangPython:
		return c.chunkPython(file, content)
	case LangTypeScript, LangJavaScript:
		return c.chunkJavaScript(file, content)
	case LangMarkdown:
		return c.chunkMarkdown(file, content)
	default:
		// Fallback: paragraph-based chunking
		return c.chunkParagraphs(file, content)
	}
}

// chunkGo uses Go's AST parser to extract functions and types.
func (c *DefaultChunker) chunkGo(file FileInfo, content []byte) ([]Chunk, []Symbol, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file.Path, content, parser.ParseComments)
	if err != nil {
		// If parsing fails, fall back to paragraph chunking
		return c.chunkParagraphs(file, content)
	}
	
	var chunks []Chunk
	var symbols []Symbol
	lines := bytes.Split(content, []byte("\n"))
	
	// Extract top-level declarations
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Function declaration
			startPos := fset.Position(d.Pos())
			endPos := fset.Position(d.End())
			
			funcName := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				// Method: prepend receiver type
				recv := d.Recv.List[0].Type
				funcName = fmt.Sprintf("(%s).%s", formatType(recv), d.Name.Name)
			}
			
			// Extract function signature
			signature := extractLines(lines, startPos.Line, startPos.Line)
			
			// Extract docstring
			docstring := ""
			if d.Doc != nil {
				docstring = d.Doc.Text()
			}
			
			// Create symbol
			symbolID := fmt.Sprintf("%s:%s:%d", file.Path, funcName, startPos.Line)
			symbol := Symbol{
				SymbolID:  symbolID,
				RepoID:    "", // Will be set by caller
				FileID:    0,  // Will be set by caller
				FilePath:  file.Path,
				Lang:      string(file.Lang),
				Name:      funcName,
				Kind:      "function",
				Signature: signature,
				StartLine: startPos.Line,
				EndLine:   endPos.Line,
				Docstring: strings.TrimSpace(docstring),
			}
			symbols = append(symbols, symbol)
			
			// Create chunk
			text := extractLines(lines, startPos.Line, endPos.Line)
			chunkID := hashChunk(file.Path, startPos.Line, endPos.Line)
			chunk := Chunk{
				ChunkID:    chunkID,
				RepoID:     "", // Will be set by caller
				FileID:     0,  // Will be set by caller
				FilePath:   file.Path,
				Lang:       string(file.Lang),
				SymbolID:   symbolID,
				SymbolName: funcName,
				Kind:       "function",
				StartLine:  startPos.Line,
				EndLine:    endPos.Line,
				Text:       text,
			}
			chunks = append(chunks, chunk)
			
		case *ast.GenDecl:
			// Type, const, var, or import declaration
			if d.Tok == token.TYPE {
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						startPos := fset.Position(ts.Pos())
						endPos := fset.Position(d.End())
						
						typeName := ts.Name.Name
						signature := extractLines(lines, startPos.Line, startPos.Line)
						
						// Extract docstring
						docstring := ""
						if d.Doc != nil {
							docstring = d.Doc.Text()
						}
						
						// Create symbol
						symbolID := fmt.Sprintf("%s:%s:%d", file.Path, typeName, startPos.Line)
						symbol := Symbol{
							SymbolID:  symbolID,
							RepoID:    "",
							FileID:    0,
							FilePath:  file.Path,
							Lang:      string(file.Lang),
							Name:      typeName,
							Kind:      "type",
							Signature: signature,
							StartLine: startPos.Line,
							EndLine:   endPos.Line,
							Docstring: strings.TrimSpace(docstring),
						}
						symbols = append(symbols, symbol)
						
						// Create chunk
						text := extractLines(lines, startPos.Line, endPos.Line)
						chunkID := hashChunk(file.Path, startPos.Line, endPos.Line)
						chunk := Chunk{
							ChunkID:    chunkID,
							RepoID:     "",
							FileID:     0,
							FilePath:   file.Path,
							Lang:       string(file.Lang),
							SymbolID:   symbolID,
							SymbolName: typeName,
							Kind:       "type",
							StartLine:  startPos.Line,
							EndLine:    endPos.Line,
							Text:       text,
						}
						chunks = append(chunks, chunk)
					}
				}
			}
		}
	}
	
	return chunks, symbols, nil
}

// chunkPython extracts functions and classes using regex (basic implementation).
func (c *DefaultChunker) chunkPython(file FileInfo, content []byte) ([]Chunk, []Symbol, error) {
	lines := bytes.Split(content, []byte("\n"))
	
	// Regex patterns for Python
	funcPattern := regexp.MustCompile(`^def\s+(\w+)\s*\(`)
	classPattern := regexp.MustCompile(`^class\s+(\w+)`)
	
	var chunks []Chunk
	var symbols []Symbol
	
	currentSymbol := ""
	currentKind := ""
	startLine := 0
	
	for i, line := range lines {
		lineNum := i + 1
		lineStr := string(line)
		indent := len(lineStr) - len(strings.TrimLeft(lineStr, " \t"))
		
		// Check for function definition
		if matches := funcPattern.FindStringSubmatch(lineStr); len(matches) > 1 && indent == 0 {
			// Save previous symbol if any
			if currentSymbol != "" {
				c.savePythonSymbol(file, lines, currentSymbol, currentKind, startLine, lineNum-1, &chunks, &symbols)
			}
			currentSymbol = matches[1]
			currentKind = "function"
			startLine = lineNum
		} else if matches := classPattern.FindStringSubmatch(lineStr); len(matches) > 1 && indent == 0 {
			// Save previous symbol if any
			if currentSymbol != "" {
				c.savePythonSymbol(file, lines, currentSymbol, currentKind, startLine, lineNum-1, &chunks, &symbols)
			}
			currentSymbol = matches[1]
			currentKind = "class"
			startLine = lineNum
		}
	}
	
	// Save last symbol
	if currentSymbol != "" {
		c.savePythonSymbol(file, lines, currentSymbol, currentKind, startLine, len(lines), &chunks, &symbols)
	}
	
	// If no symbols found, fall back to paragraph chunking
	if len(chunks) == 0 {
		return c.chunkParagraphs(file, content)
	}
	
	return chunks, symbols, nil
}

func (c *DefaultChunker) savePythonSymbol(file FileInfo, lines [][]byte, name, kind string, startLine, endLine int, chunks *[]Chunk, symbols *[]Symbol) {
	text := extractLines(lines, startLine, endLine)
	signature := extractLines(lines, startLine, startLine)
	
	symbolID := fmt.Sprintf("%s:%s:%d", file.Path, name, startLine)
	symbol := Symbol{
		SymbolID:  symbolID,
		FilePath:  file.Path,
		Lang:      string(file.Lang),
		Name:      name,
		Kind:      kind,
		Signature: signature,
		StartLine: startLine,
		EndLine:   endLine,
	}
	*symbols = append(*symbols, symbol)
	
	chunkID := hashChunk(file.Path, startLine, endLine)
	chunk := Chunk{
		ChunkID:    chunkID,
		FilePath:   file.Path,
		Lang:       string(file.Lang),
		SymbolID:   symbolID,
		SymbolName: name,
		Kind:       kind,
		StartLine:  startLine,
		EndLine:    endLine,
		Text:       text,
	}
	*chunks = append(*chunks, chunk)
}

// chunkJavaScript extracts functions and classes using regex (basic implementation).
func (c *DefaultChunker) chunkJavaScript(file FileInfo, content []byte) ([]Chunk, []Symbol, error) {
	// For now, use paragraph chunking for JS/TS
	// A full implementation would use a proper parser (like tree-sitter)
	return c.chunkParagraphs(file, content)
}

// chunkMarkdown splits by headers.
func (c *DefaultChunker) chunkMarkdown(file FileInfo, content []byte) ([]Chunk, []Symbol, error) {
	lines := bytes.Split(content, []byte("\n"))
	headerPattern := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	
	var chunks []Chunk
	var symbols []Symbol
	
	currentHeader := ""
	startLine := 1
	
	for i, line := range lines {
		lineNum := i + 1
		lineStr := string(line)
		
		if matches := headerPattern.FindStringSubmatch(lineStr); len(matches) > 2 {
			// Save previous section
			if currentHeader != "" {
				c.saveMarkdownSection(file, lines, currentHeader, startLine, lineNum-1, &chunks, &symbols)
			}
			currentHeader = matches[2]
			startLine = lineNum
		}
	}
	
	// Save last section
	if currentHeader != "" {
		c.saveMarkdownSection(file, lines, currentHeader, startLine, len(lines), &chunks, &symbols)
	}
	
	// If no headers found, treat whole file as one chunk
	if len(chunks) == 0 {
		text := string(content)
		chunkID := hashChunk(file.Path, 1, len(lines))
		chunk := Chunk{
			ChunkID:   chunkID,
			FilePath:  file.Path,
			Lang:      string(file.Lang),
			Kind:      "document",
			StartLine: 1,
			EndLine:   len(lines),
			Text:      text,
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks, symbols, nil
}

func (c *DefaultChunker) saveMarkdownSection(file FileInfo, lines [][]byte, header string, startLine, endLine int, chunks *[]Chunk, symbols *[]Symbol) {
	text := extractLines(lines, startLine, endLine)
	
	symbolID := fmt.Sprintf("%s:%s:%d", file.Path, header, startLine)
	symbol := Symbol{
		SymbolID:  symbolID,
		FilePath:  file.Path,
		Lang:      string(file.Lang),
		Name:      header,
		Kind:      "section",
		Signature: header,
		StartLine: startLine,
		EndLine:   endLine,
	}
	*symbols = append(*symbols, symbol)
	
	chunkID := hashChunk(file.Path, startLine, endLine)
	chunk := Chunk{
		ChunkID:    chunkID,
		FilePath:   file.Path,
		Lang:       string(file.Lang),
		SymbolID:   symbolID,
		SymbolName: header,
		Kind:       "section",
		StartLine:  startLine,
		EndLine:    endLine,
		Text:       text,
	}
	*chunks = append(*chunks, chunk)
}

// chunkParagraphs splits content by blank lines (fallback strategy).
func (c *DefaultChunker) chunkParagraphs(file FileInfo, content []byte) ([]Chunk, []Symbol, error) {
	lines := bytes.Split(content, []byte("\n"))
	
	var chunks []Chunk
	startLine := 1
	paragraphLines := [][]byte{}
	
	for i, line := range lines {
		lineNum := i + 1
		trimmed := bytes.TrimSpace(line)
		
		if len(trimmed) == 0 {
			// Blank line - save current paragraph
			if len(paragraphLines) > 0 {
				endLine := lineNum - 1
				text := string(bytes.Join(paragraphLines, []byte("\n")))
				chunkID := hashChunk(file.Path, startLine, endLine)
				chunk := Chunk{
					ChunkID:   chunkID,
					FilePath:  file.Path,
					Lang:      string(file.Lang),
					Kind:      "paragraph",
					StartLine: startLine,
					EndLine:   endLine,
					Text:      text,
				}
				chunks = append(chunks, chunk)
				paragraphLines = [][]byte{}
			}
			startLine = lineNum + 1
		} else {
			paragraphLines = append(paragraphLines, line)
		}
	}
	
	// Save last paragraph
	if len(paragraphLines) > 0 {
		endLine := len(lines)
		text := string(bytes.Join(paragraphLines, []byte("\n")))
		chunkID := hashChunk(file.Path, startLine, endLine)
		chunk := Chunk{
			ChunkID:   chunkID,
			FilePath:  file.Path,
			Lang:      string(file.Lang),
			Kind:      "paragraph",
			StartLine: startLine,
			EndLine:   endLine,
			Text:      text,
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil, nil
}

// Helper functions

func extractLines(lines [][]byte, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return ""
	}
	return string(bytes.Join(lines[start-1:end], []byte("\n")))
}

func formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + formatType(t.X)
	case *ast.SelectorExpr:
		return formatType(t.X) + "." + t.Sel.Name
	default:
		return "?"
	}
}

func hashChunk(filePath string, startLine, endLine int) string {
	key := fmt.Sprintf("%s:%d:%d", filePath, startLine, endLine)
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}

