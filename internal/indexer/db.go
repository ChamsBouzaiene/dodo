package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// IndexStatus represents the indexing state of a file.
type IndexStatus string

const (
	StatusPending  IndexStatus = "pending"  // Needs indexing
	StatusIndexing IndexStatus = "indexing" // Currently being indexed
	StatusIndexed  IndexStatus = "indexed"  // Successfully indexed
	StatusFailed   IndexStatus = "failed"   // Indexing failed
)

// FileRecord represents a file entry in the database.
type FileRecord struct {
	FileID      int64
	RepoID      string
	Path        string
	Lang        string
	Hash        string
	SizeBytes   int64
	MtimeUnix   int64
	Deleted     bool
	IndexStatus IndexStatus
	IndexedAt   int64  // Unix timestamp when successfully indexed
	IndexError  string // Error message if indexing failed
}

// DB provides database operations for file indexing.
type DB struct {
	db *sql.DB
}

// NewDB creates a new database connection and initializes the schema.
func NewDB(ctx context.Context, dbPath string) (*DB, error) {
	// Enable WAL mode for better concurrency and set busy timeout
	// WAL mode allows multiple readers and one writer simultaneously
	dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=5000"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings for better concurrency
	db.SetMaxOpenConns(1) // SQLite doesn't support multiple writers well
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	d := &DB{db: db}
	if err := d.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return d, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// initSchema creates the database tables if they don't exist.
func (d *DB) initSchema(ctx context.Context) error {
	schema := `
	-- Repository metadata
	CREATE TABLE IF NOT EXISTS repos (
		repo_id    TEXT PRIMARY KEY,
		root_path  TEXT NOT NULL,
		is_git     INTEGER NOT NULL,
		git_root   TEXT,
		created_at INTEGER NOT NULL
	);

	-- File tracking
	CREATE TABLE IF NOT EXISTS files (
		file_id      INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id      TEXT NOT NULL,
		path         TEXT NOT NULL,
		lang         TEXT NOT NULL,
		hash         TEXT NOT NULL,
		size_bytes   INTEGER NOT NULL,
		mtime_unix   INTEGER NOT NULL,
		deleted      INTEGER NOT NULL DEFAULT 0,
		index_status TEXT NOT NULL DEFAULT 'pending',
		indexed_at   INTEGER,
		index_error  TEXT,
		UNIQUE (repo_id, path),
		FOREIGN KEY (repo_id) REFERENCES repos(repo_id)
	);

	-- Symbols (functions, classes, methods, etc.)
	CREATE TABLE IF NOT EXISTS symbols (
		symbol_id  TEXT PRIMARY KEY,
		repo_id    TEXT NOT NULL,
		file_id    INTEGER NOT NULL,
		file_path  TEXT NOT NULL,
		lang       TEXT NOT NULL,
		name       TEXT NOT NULL,
		kind       TEXT NOT NULL,
		signature  TEXT NOT NULL,
		start_line INTEGER NOT NULL,
		end_line   INTEGER NOT NULL,
		docstring  TEXT,
		FOREIGN KEY (repo_id) REFERENCES repos(repo_id),
		FOREIGN KEY (file_id) REFERENCES files(file_id) ON DELETE CASCADE
	);

	-- Chunks (text segments for semantic search)
	CREATE TABLE IF NOT EXISTS chunks (
		chunk_id    TEXT PRIMARY KEY,
		repo_id     TEXT NOT NULL,
		file_id     INTEGER NOT NULL,
		file_path   TEXT NOT NULL,
		lang        TEXT NOT NULL,
		symbol_id   TEXT,
		symbol_name TEXT,
		kind        TEXT NOT NULL,
		start_line  INTEGER NOT NULL,
		end_line    INTEGER NOT NULL,
		text        TEXT NOT NULL,
		FOREIGN KEY (repo_id) REFERENCES repos(repo_id),
		FOREIGN KEY (file_id) REFERENCES files(file_id) ON DELETE CASCADE,
		FOREIGN KEY (symbol_id) REFERENCES symbols(symbol_id) ON DELETE CASCADE
	);

	-- Embeddings (vector representations)
	CREATE TABLE IF NOT EXISTS embeddings (
		chunk_id TEXT PRIMARY KEY,
		repo_id  TEXT NOT NULL,
		dim      INTEGER NOT NULL,
		vector   BLOB NOT NULL,
		FOREIGN KEY (chunk_id) REFERENCES chunks(chunk_id) ON DELETE CASCADE,
		FOREIGN KEY (repo_id) REFERENCES repos(repo_id)
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_files_repo ON files(repo_id);
	CREATE INDEX IF NOT EXISTS idx_files_deleted ON files(deleted);
	CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);
	CREATE INDEX IF NOT EXISTS idx_files_status ON files(index_status);
	
	CREATE INDEX IF NOT EXISTS idx_symbols_repo ON symbols(repo_id);
	CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);
	CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
	
	CREATE INDEX IF NOT EXISTS idx_chunks_repo ON chunks(repo_id);
	CREATE INDEX IF NOT EXISTS idx_chunks_file ON chunks(file_id);
	CREATE INDEX IF NOT EXISTS idx_chunks_symbol ON chunks(symbol_id);
	
	CREATE INDEX IF NOT EXISTS idx_embeddings_repo ON embeddings(repo_id);
	`

	_, err := d.db.ExecContext(ctx, schema)
	return err
}

// UpsertFile inserts or updates a file record.
// Returns true if the file is new or the hash changed (needs indexing).
// Sets index_status to 'pending' when needsIndexing is true.
func (d *DB) UpsertFile(ctx context.Context, repoID, path, lang, hash string, sizeBytes, mtimeUnix int64) (bool, error) {
	// Check if file exists and if hash changed
	var existingHash string
	var existingStatus string
	checkQuery := `SELECT hash, index_status FROM files WHERE repo_id = ? AND path = ?`
	err := d.db.QueryRowContext(ctx, checkQuery, repoID, path).Scan(&existingHash, &existingStatus)

	needsIndexing := false
	newStatus := existingStatus

	if err == sql.ErrNoRows {
		// New file - needs indexing
		needsIndexing = true
		newStatus = string(StatusPending)
	} else if err != nil {
		return false, fmt.Errorf("failed to check existing file: %w", err)
	} else if existingHash != hash {
		// Hash changed - needs re-indexing
		needsIndexing = true
		newStatus = string(StatusPending)
	} else if existingStatus == string(StatusFailed) {
		// Previous indexing failed - retry
		needsIndexing = true
		newStatus = string(StatusPending)
	}

	// Upsert the file record
	query := `
		INSERT INTO files (repo_id, path, lang, hash, size_bytes, mtime_unix, deleted, index_status, indexed_at, index_error)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?, NULL, NULL)
		ON CONFLICT(repo_id, path) DO UPDATE SET
			lang = excluded.lang,
			hash = excluded.hash,
			size_bytes = excluded.size_bytes,
			mtime_unix = excluded.mtime_unix,
			deleted = 0,
			index_status = ?,
			indexed_at = CASE WHEN ? = 'pending' THEN NULL ELSE indexed_at END,
			index_error = CASE WHEN ? = 'pending' THEN NULL ELSE index_error END
	`

	_, err = d.db.ExecContext(ctx, query, repoID, path, lang, hash, sizeBytes, mtimeUnix, newStatus, newStatus, newStatus, newStatus)
	if err != nil {
		return false, fmt.Errorf("failed to upsert file: %w", err)
	}

	return needsIndexing, nil
}

// MarkDeleted marks a file as deleted.
func (d *DB) MarkDeleted(ctx context.Context, repoID, path string) error {
	query := `UPDATE files SET deleted = 1 WHERE repo_id = ? AND path = ?`
	_, err := d.db.ExecContext(ctx, query, repoID, path)
	return err
}

// GetFilesNeedingIndex returns all files with status='pending' that need indexing.
func (d *DB) GetFilesNeedingIndex(ctx context.Context, repoID string) ([]FileRecord, error) {
	query := `
		SELECT file_id, repo_id, path, lang, hash, size_bytes, mtime_unix, deleted, index_status, indexed_at, index_error
		FROM files
		WHERE repo_id = ? AND deleted = 0 AND index_status = ?
		ORDER BY path
	`

	rows, err := d.db.QueryContext(ctx, query, repoID, string(StatusPending))
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var f FileRecord
		var deleted int
		var indexedAt sql.NullInt64
		var indexError sql.NullString
		err := rows.Scan(&f.FileID, &f.RepoID, &f.Path, &f.Lang, &f.Hash, &f.SizeBytes, &f.MtimeUnix, &deleted, &f.IndexStatus, &indexedAt, &indexError)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		f.Deleted = deleted == 1
		if indexedAt.Valid {
			f.IndexedAt = indexedAt.Int64
		}
		if indexError.Valid {
			f.IndexError = indexError.String
		}
		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating files: %w", err)
	}

	return files, nil
}

// GetAllRepoFiles returns all files (including deleted) for a repo.
func (d *DB) GetAllRepoFiles(ctx context.Context, repoID string) ([]FileRecord, error) {
	query := `
		SELECT file_id, repo_id, path, lang, hash, size_bytes, mtime_unix, deleted, index_status, indexed_at, index_error
		FROM files
		WHERE repo_id = ?
		ORDER BY path
	`

	rows, err := d.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var f FileRecord
		var deleted int
		var indexedAt sql.NullInt64
		var indexError sql.NullString
		err := rows.Scan(&f.FileID, &f.RepoID, &f.Path, &f.Lang, &f.Hash, &f.SizeBytes, &f.MtimeUnix, &deleted, &f.IndexStatus, &indexedAt, &indexError)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		f.Deleted = deleted == 1
		if indexedAt.Valid {
			f.IndexedAt = indexedAt.Int64
		}
		if indexError.Valid {
			f.IndexError = indexError.String
		}
		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating files: %w", err)
	}

	return files, nil
}

// MarkIndexing marks a file as currently being indexed.
// This prevents duplicate work if multiple workers are running.
func (d *DB) MarkIndexing(ctx context.Context, repoID, path string) error {
	query := `UPDATE files SET index_status = ? WHERE repo_id = ? AND path = ?`
	_, err := d.db.ExecContext(ctx, query, string(StatusIndexing), repoID, path)
	if err != nil {
		return fmt.Errorf("failed to mark file as indexing: %w", err)
	}
	return nil
}

// MarkIndexed marks a file as successfully indexed.
func (d *DB) MarkIndexed(ctx context.Context, repoID, path string) error {
	now := time.Now().Unix()
	query := `UPDATE files SET index_status = ?, indexed_at = ?, index_error = NULL WHERE repo_id = ? AND path = ?`
	_, err := d.db.ExecContext(ctx, query, string(StatusIndexed), now, repoID, path)
	if err != nil {
		return fmt.Errorf("failed to mark file as indexed: %w", err)
	}
	return nil
}

// MarkFailed marks a file as failed to index with an error message.
func (d *DB) MarkFailed(ctx context.Context, repoID, path, errorMsg string) error {
	query := `UPDATE files SET index_status = ?, index_error = ? WHERE repo_id = ? AND path = ?`
	_, err := d.db.ExecContext(ctx, query, string(StatusFailed), errorMsg, repoID, path)
	if err != nil {
		return fmt.Errorf("failed to mark file as failed: %w", err)
	}
	return nil
}

// ResetStuckIndexing resets files stuck in 'indexing' state back to 'pending'.
// This is useful for recovering from crashes where files were marked as indexing but never completed.
func (d *DB) ResetStuckIndexing(ctx context.Context, repoID string, olderThan time.Duration) (int, error) {
	// Files stuck in indexing state with mtime older than the threshold
	cutoff := time.Now().Add(-olderThan).Unix()
	query := `UPDATE files SET index_status = ? WHERE repo_id = ? AND index_status = ? AND mtime_unix < ?`
	result, err := d.db.ExecContext(ctx, query, string(StatusPending), repoID, string(StatusIndexing), cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to reset stuck indexing: %w", err)
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// CleanupDeleted removes deleted files older than the specified duration.
func (d *DB) CleanupDeleted(ctx context.Context, repoID string, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan).Unix()
	query := `DELETE FROM files WHERE repo_id = ? AND deleted = 1 AND mtime_unix < ?`
	_, err := d.db.ExecContext(ctx, query, repoID, cutoff)
	return err
}

// RepoRecord represents a repository entry.
type RepoRecord struct {
	RepoID    string
	RootPath  string
	IsGit     bool
	GitRoot   string
	CreatedAt int64
}

// UpsertRepo inserts or updates a repository record.
func (d *DB) UpsertRepo(ctx context.Context, repoID, rootPath string, isGit bool, gitRoot string) error {
	now := time.Now().Unix()
	query := `
		INSERT INTO repos (repo_id, root_path, is_git, git_root, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(repo_id) DO UPDATE SET
			root_path = excluded.root_path,
			is_git = excluded.is_git,
			git_root = excluded.git_root
	`
	isGitInt := 0
	if isGit {
		isGitInt = 1
	}
	_, err := d.db.ExecContext(ctx, query, repoID, rootPath, isGitInt, gitRoot, now)
	return err
}

// GetRepo retrieves a repository record.
func (d *DB) GetRepo(ctx context.Context, repoID string) (*RepoRecord, error) {
	query := `SELECT repo_id, root_path, is_git, git_root, created_at FROM repos WHERE repo_id = ?`
	var r RepoRecord
	var isGitInt int
	var gitRoot sql.NullString
	err := d.db.QueryRowContext(ctx, query, repoID).Scan(&r.RepoID, &r.RootPath, &isGitInt, &gitRoot, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	r.IsGit = isGitInt == 1
	if gitRoot.Valid {
		r.GitRoot = gitRoot.String
	}
	return &r, nil
}

// Symbol represents a code symbol (function, class, method, etc.)
type Symbol struct {
	SymbolID  string
	RepoID    string
	FileID    int64
	FilePath  string
	Lang      string
	Name      string
	Kind      string
	Signature string
	StartLine int
	EndLine   int
	Docstring string
}

// InsertSymbol inserts a new symbol.
func (d *DB) InsertSymbol(ctx context.Context, s *Symbol) error {
	query := `
		INSERT INTO symbols (symbol_id, repo_id, file_id, file_path, lang, name, kind, signature, start_line, end_line, docstring)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(symbol_id) DO UPDATE SET
			name = excluded.name,
			signature = excluded.signature,
			start_line = excluded.start_line,
			end_line = excluded.end_line,
			docstring = excluded.docstring
	`
	_, err := d.db.ExecContext(ctx, query, s.SymbolID, s.RepoID, s.FileID, s.FilePath, s.Lang, s.Name, s.Kind, s.Signature, s.StartLine, s.EndLine, s.Docstring)
	return err
}

// DeleteSymbolsByFile deletes all symbols for a file.
func (d *DB) DeleteSymbolsByFile(ctx context.Context, fileID int64) error {
	query := `DELETE FROM symbols WHERE file_id = ?`
	_, err := d.db.ExecContext(ctx, query, fileID)
	return err
}

// Chunk represents a text chunk for semantic search.
type Chunk struct {
	ChunkID    string
	RepoID     string
	FileID     int64
	FilePath   string
	Lang       string
	SymbolID   string
	SymbolName string
	Kind       string
	StartLine  int
	EndLine    int
	Text       string
}

// InsertChunk inserts a new chunk.
func (d *DB) InsertChunk(ctx context.Context, c *Chunk) error {
	query := `
		INSERT INTO chunks (chunk_id, repo_id, file_id, file_path, lang, symbol_id, symbol_name, kind, start_line, end_line, text)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chunk_id) DO UPDATE SET
			text = excluded.text,
			start_line = excluded.start_line,
			end_line = excluded.end_line
	`
	_, err := d.db.ExecContext(ctx, query, c.ChunkID, c.RepoID, c.FileID, c.FilePath, c.Lang, c.SymbolID, c.SymbolName, c.Kind, c.StartLine, c.EndLine, c.Text)
	return err
}

// DeleteChunksByFile deletes all chunks for a file.
func (d *DB) DeleteChunksByFile(ctx context.Context, fileID int64) error {
	query := `DELETE FROM chunks WHERE file_id = ?`
	_, err := d.db.ExecContext(ctx, query, fileID)
	return err
}

// GetChunksByFile retrieves all chunks for a file.
func (d *DB) GetChunksByFile(ctx context.Context, fileID int64) ([]Chunk, error) {
	query := `
		SELECT chunk_id, repo_id, file_id, file_path, lang, symbol_id, symbol_name, kind, start_line, end_line, text
		FROM chunks WHERE file_id = ?
		ORDER BY start_line
	`
	rows, err := d.db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		var symbolID, symbolName sql.NullString
		err := rows.Scan(&c.ChunkID, &c.RepoID, &c.FileID, &c.FilePath, &c.Lang, &symbolID, &symbolName, &c.Kind, &c.StartLine, &c.EndLine, &c.Text)
		if err != nil {
			return nil, err
		}
		if symbolID.Valid {
			c.SymbolID = symbolID.String
		}
		if symbolName.Valid {
			c.SymbolName = symbolName.String
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

// Embedding represents a vector embedding for a chunk.
type Embedding struct {
	ChunkID string
	RepoID  string
	Dim     int
	Vector  []byte
}

// InsertEmbedding inserts or updates an embedding.
func (d *DB) InsertEmbedding(ctx context.Context, e *Embedding) error {
	query := `
		INSERT INTO embeddings (chunk_id, repo_id, dim, vector)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(chunk_id) DO UPDATE SET
			vector = excluded.vector,
			dim = excluded.dim
	`
	_, err := d.db.ExecContext(ctx, query, e.ChunkID, e.RepoID, e.Dim, e.Vector)
	return err
}

// DeleteEmbeddingsByFile deletes all embeddings for chunks belonging to a file.
func (d *DB) DeleteEmbeddingsByFile(ctx context.Context, fileID int64) error {
	query := `
		DELETE FROM embeddings 
		WHERE chunk_id IN (SELECT chunk_id FROM chunks WHERE file_id = ?)
	`
	_, err := d.db.ExecContext(ctx, query, fileID)
	return err
}
