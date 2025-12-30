# 🧪 Testing Guide for Semantic Search Tools

This guide helps you validate that the `codebase_search` and `read_span` tools are working correctly.

---

## 🚀 Quick Start: Manual Testing

### Step 1: Choose a Test Repository

Pick a repository you know well (preferably one with Go/Python code):

```bash
cd /path/to/your/test-repo
```

### Step 2: Initial Indexing (One Time)

```bash
# Set OpenAI API key if you want semantic search
export OPENAI_API_KEY="your-key-here"

# Run initial indexing
dodo --repo . --task "test" --index
```

**What to expect:**
- ✅ Logs showing file discovery
- ✅ "Starting initial full indexing..."
- ✅ Progress updates as files are chunked and embedded
- ✅ "BM25 index ready"
- ✅ Database created at `.dodo/index.db`
- ✅ BM25 index created at `.dodo/index.db.bleve`

**Time estimate:**
- Small repo (< 100 files): ~30 seconds
- Medium repo (1k files): ~2-5 minutes
- Large repo (10k+ files): ~10-30 minutes

### Step 3: Test with a Real Query

```bash
# Ask a question about your codebase
dodo --repo . --task "Find where we handle authentication"
```

**What to expect:**
- ✅ Agent uses `codebase_search` tool
- ✅ Returns relevant code locations
- ✅ Agent uses `read_span` to read full context
- ✅ Agent provides answer based on found code

---

## 🔍 Validation Checklist

### ✅ Indexing Validation

1. **Check database exists:**
   ```bash
   ls -lh .dodo/index.db
   ```

2. **Check BM25 index exists:**
   ```bash
   ls -lh .dodo/index.db.bleve/
   ```

3. **Verify files are indexed:**
   ```bash
   sqlite3 .dodo/index.db "SELECT COUNT(*) FROM files WHERE index_status = 'indexed';"
   ```

4. **Verify chunks are created:**
   ```bash
   sqlite3 .dodo/index.db "SELECT COUNT(*) FROM chunks;"
   ```

5. **Verify embeddings exist (if OpenAI key set):**
   ```bash
   sqlite3 .dodo/index.db "SELECT COUNT(*) FROM embeddings;"
   ```

### ✅ Tool Registration Validation

1. **Check agent logs for tool registration:**
   - Look for logs showing tools being registered
   - Should see `codebase_search` and `read_span` in the tool list

2. **Verify system prompt includes search guidance:**
   - Check logs for: "To find relevant code or documentation, use the 'codebase_search' tool..."

### ✅ Search Functionality Validation

Test these scenarios:

#### Test 1: Keyword Search (BM25)
```bash
dodo --repo . --task "Find functions that use JWT"
```
**Expected:** Should find code with "JWT" keyword even if query doesn't match semantically.

#### Test 2: Semantic Search (Embeddings)
```bash
dodo --repo . --task "Where do we validate user credentials"
```
**Expected:** Should find authentication/validation code even if exact keywords don't match.

#### Test 3: Hybrid Search (Both)
```bash
dodo --repo . --task "Show me error handling patterns"
```
**Expected:** Should combine keyword matches (BM25) with semantic matches (embeddings).

#### Test 4: File Filtering
```bash
dodo --repo . --task "Find test files that use mocks"
```
**Expected:** Should filter to test files (if globs are used).

---

## 🛠️ Advanced Testing

### Test with Debug Logging

Add verbose logging to see what's happening:

```bash
# Enable Go debug logging
export GODEBUG=gctrace=1

# Run with verbose output
dodo --repo . --task "test" 2>&1 | tee dodo.log
```

### Inspect the Index

```bash
# Check indexed files
sqlite3 .dodo/index.db "SELECT path, lang, index_status FROM files LIMIT 10;"

# Check chunks
sqlite3 .dodo/index.db "SELECT file_path, start_line, end_line, kind FROM chunks LIMIT 10;"

# Check symbols
sqlite3 .dodo/index.db "SELECT name, kind, file_path FROM symbols LIMIT 10;"
```

### Test ReadSpan Directly

Create a simple test script:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/yourorg/dodo/internal/indexer"
)

func main() {
    // Load manager (same setup as main.go)
    manager := setupManager()
    
    // Test ReadSpan
    ctx := context.Background()
    code, err := manager.ReadSpan(ctx, "internal/indexer/manager.go", 100, 120)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(code)
}
```

---

## 🐛 Troubleshooting

### Problem: "No results found"

**Possible causes:**
1. Indexing didn't complete
2. Files not indexed yet (check `index_status`)
3. Query too specific or too vague

**Solutions:**
- Check indexing status: `sqlite3 .dodo/index.db "SELECT COUNT(*) FROM chunks;"`
- Re-run indexing: `dodo --repo . --task "test" --index`
- Try simpler queries first

### Problem: "BM25 search failed"

**Possible causes:**
1. BM25 index not created
2. Index corrupted

**Solutions:**
- Check BM25 index exists: `ls .dodo/index.db.bleve/`
- Delete and re-index: `rm -rf .dodo/ && dodo --repo . --task "test" --index`

### Problem: "Embedding search failed"

**Possible causes:**
1. No OpenAI API key
2. API key invalid
3. Network issues

**Solutions:**
- Check API key: `echo $OPENAI_API_KEY`
- Test API key: `curl https://api.openai.com/v1/models -H "Authorization: Bearer $OPENAI_API_KEY"`
- System will fall back to BM25-only if embeddings fail

### Problem: Tools not registered

**Possible causes:**
1. Manager not passed to workflow
2. Retrieval interface not implemented

**Solutions:**
- Check `workflow.go` passes manager
- Check `agent.go` receives retrieval
- Check logs for tool registration

---

## 📊 Expected Results

### Good Results ✅

- **Fast search**: < 1 second for most queries
- **Relevant results**: Top 3 results should be highly relevant
- **Hybrid working**: Results show `"reason": "rrf(bm25+vec)"`
- **Agent uses tools**: Logs show `codebase_search` and `read_span` calls

### Warning Signs ⚠️

- **Slow search**: > 2 seconds (might need optimization)
- **Irrelevant results**: Top results don't match query
- **Only BM25**: Results show `"reason": "embedding_only"` (embeddings not working)
- **No results**: Empty array returned (index might be empty)

---

## 🎯 Test Scenarios

### Scenario 1: Find a Specific Function

```bash
dodo --repo . --task "Find the function that validates JWT tokens"
```

**Expected behavior:**
1. Agent calls `codebase_search` with query
2. Returns function locations
3. Agent calls `read_span` on most relevant result
4. Agent provides answer with function name and location

### Scenario 2: Understand Code Pattern

```bash
dodo --repo . --task "How do we handle database errors in this codebase"
```

**Expected behavior:**
1. Agent searches for error handling patterns
2. Returns multiple relevant locations
3. Agent reads multiple spans to understand pattern
4. Agent provides comprehensive answer

### Scenario 3: Find Test Files

```bash
dodo --repo . --task "Show me test files for the authentication module"
```

**Expected behavior:**
1. Agent searches with file filtering (if implemented)
2. Returns test files matching query
3. Agent reads relevant test code
4. Agent lists test files and their purpose

---

## 🔬 Unit Testing (For Developers)

See `internal/indexer/` for unit test examples:

```bash
# Run all tests
go test ./internal/indexer/...

# Run specific test
go test ./internal/indexer/ -run TestSearch

# With coverage
go test ./internal/indexer/ -cover
```

---

## 📝 Next Steps

Once basic testing passes:

1. **Test on larger repos** (10k+ files)
2. **Test with different languages** (Go, Python, TypeScript, etc.)
3. **Test edge cases** (empty repos, very large files, binary files)
4. **Measure performance** (indexing time, search latency)
5. **Gather feedback** (are results relevant? is search fast enough?)

---

## 💡 Tips

1. **Start small**: Test on a small, familiar repo first
2. **Use known queries**: Ask about code you know exists
3. **Check logs**: Watch for errors or warnings
4. **Be patient**: First indexing takes time
5. **Iterate**: Try different queries to understand behavior

---

Happy testing! 🎉

