package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/1amkhush/torrentium/pkg/config"
	"github.com/1amkhush/torrentium/pkg/logger"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type LocalFile struct {
	ID        string
	CID       string
	Filename  string
	FileSize  int64
	FilePath  string
	FileHash  string
	CreatedAt time.Time
}

type Download struct {
	ID           string
	CID          string
	Filename     string
	FileSize     int64
	DownloadPath string
	DownloadedAt time.Time
	Status       string
}

// Piece tracks chunked pieces for resume and verification
type Piece struct {
	ID        string
	CID       string
	Index     int64
	Offset    int64
	Size      int64
	Hash      string
	Have      bool
	UpdatedAt time.Time
}

// PeerScore stores reputation
type PeerScore struct {
	PeerID string
	Score  float64
	SeenAt time.Time
}

// FileEntry represents a single file within a multi-file torrent
type FileEntry struct {
	ID       string `json:"id,omitempty"`
	CID      string `json:"cid,omitempty"`
	Path     string `json:"path"`   // Relative path within torrent
	Size     int64  `json:"size"`   // File size in bytes
	Offset   int64  `json:"offset"` // Byte offset in the concatenated data
	FileHash string `json:"file_hash,omitempty"`
}

// UploadRecord tracks upload statistics per CID/peer
type UploadRecord struct {
	ID            string
	CID           string
	PeerID        string
	BytesUploaded int64
	ChunksServed  int
	UploadedAt    time.Time
}

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository { return &Repository{DB: db} }

// InitDB initializes and returns a database connection with proper configuration.
// Returns the database connection and any error encountered.
func InitDB(cfg *config.DatabaseConfig) (*sql.DB, error) {
	dbpath := cfg.Path
	if dbpath == "" {
		dbpath = os.Getenv("SQLITE_DB_PATH")
		if dbpath == "" {
			dbpath = "./peer.db"
		}
	}

	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Verify connection
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	if err := createTables(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	logger.Info().Str("path", dbpath).Msg("Successfully connected to peer database")
	return db, nil
}

// InitDBWithDefaults initializes database with default configuration
func InitDBWithDefaults() (*sql.DB, error) {
	cfg := config.DefaultConfig()
	return InitDB(&cfg.Database)
}

func createTables(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS local_files (
			id TEXT PRIMARY KEY,
			cid TEXT UNIQUE NOT NULL,
			filename TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			file_path TEXT NOT NULL,
			file_hash TEXT NOT NULL,
			is_directory INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS downloads (
			id TEXT PRIMARY KEY,
			cid TEXT UNIQUE NOT NULL,
			filename TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			download_path TEXT NOT NULL,
			downloaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'completed'
		);`,
		`CREATE TABLE IF NOT EXISTS pieces (
			id TEXT PRIMARY KEY,
			cid TEXT NOT NULL,
			idx INTEGER NOT NULL,
			offset INTEGER NOT NULL,
			size INTEGER NOT NULL,
			hash TEXT NOT NULL,
			have INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (cid, idx)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_pieces_cid ON pieces(cid);`,
		`CREATE TABLE IF NOT EXISTS peer_scores (
			peer_id TEXT PRIMARY KEY,
			score REAL NOT NULL,
			seen_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS metadata_index (
			cid TEXT PRIMARY KEY,
			filename TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			file_hash TEXT NOT NULL
		);`,
		// New table for multi-file support
		`CREATE TABLE IF NOT EXISTS file_entries (
			id TEXT PRIMARY KEY,
			cid TEXT NOT NULL,
			path TEXT NOT NULL,
			size INTEGER NOT NULL,
			offset INTEGER NOT NULL,
			file_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (cid, path)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_file_entries_cid ON file_entries(cid);`,
		// New table for upload tracking
		`CREATE TABLE IF NOT EXISTS uploads (
			id TEXT PRIMARY KEY,
			cid TEXT NOT NULL,
			peer_id TEXT NOT NULL,
			bytes_uploaded INTEGER NOT NULL DEFAULT 0,
			chunks_served INTEGER NOT NULL DEFAULT 0,
			uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (cid, peer_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_uploads_cid ON uploads(cid);`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("schema error: %w", err)
		}
	}
	return nil
}

func (r *Repository) AddLocalFile(ctx context.Context, cid, filename string, fileSize int64, filePath, fileHash string) error {
	q := `INSERT INTO local_files (id, cid, filename, file_size, file_path, file_hash, created_at)
	      VALUES (?, ?, ?, ?, ?, ?, ?)
	      ON CONFLICT(cid) DO UPDATE SET filename=excluded.filename, file_path=excluded.file_path, file_size=excluded.file_size, file_hash=excluded.file_hash`
	_, err := r.DB.ExecContext(ctx, q, uuid.New().String(), cid, filename, fileSize, filePath, fileHash, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add local file: %w", err)
	}
	// update metadata index for search
	if _, err := r.DB.ExecContext(ctx, `INSERT INTO metadata_index (cid, filename, file_size, file_hash)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(cid) DO UPDATE SET filename=excluded.filename, file_size=excluded.file_size, file_hash=excluded.file_hash`,
		cid, filename, fileSize, fileHash); err != nil {
		logger.Warn().Err(err).Str("cid", cid).Msg("Failed to update metadata index")
	}
	return nil
}

func (r *Repository) GetLocalFiles(ctx context.Context) ([]LocalFile, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, cid, filename, file_size, file_path, file_hash, created_at FROM local_files ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []LocalFile
	for rows.Next() {
		var f LocalFile
		if err := rows.Scan(&f.ID, &f.CID, &f.Filename, &f.FileSize, &f.FilePath, &f.FileHash, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (r *Repository) GetLocalFileByCID(ctx context.Context, cid string) (*LocalFile, error) {
	var f LocalFile
	err := r.DB.QueryRowContext(ctx, `SELECT id, cid, filename, file_size, file_path, file_hash, created_at FROM local_files WHERE cid = ?`, cid).
		Scan(&f.ID, &f.CID, &f.Filename, &f.FileSize, &f.FilePath, &f.FileHash, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repository) DeleteLocalFile(ctx context.Context, cid string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM local_files WHERE cid=?`, cid)
	return err
}

func (r *Repository) AddDownload(ctx context.Context, cid, filename string, fileSize int64, downloadPath string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO downloads (id, cid, filename, file_size, download_path, downloaded_at, status)
		VALUES (?, ?, ?, ?, ?, ?, 'completed') ON CONFLICT(cid) DO UPDATE SET status='completed', downloaded_at=excluded.downloaded_at, download_path=excluded.download_path`,
		uuid.New().String(), cid, filename, fileSize, downloadPath, time.Now())
	return err
}

func (r *Repository) GetDownloads(ctx context.Context) ([]Download, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, cid, filename, file_size, download_path, downloaded_at, status FROM downloads ORDER BY downloaded_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var downloads []Download
	for rows.Next() {
		var d Download
		if err := rows.Scan(&d.ID, &d.CID, &d.Filename, &d.FileSize, &d.DownloadPath, &d.DownloadedAt, &d.Status); err != nil {
			return nil, err
		}
		downloads = append(downloads, d)
	}
	return downloads, rows.Err()
}

func (r *Repository) UpsertPiece(ctx context.Context, cid string, idx int64, offset, size int64, hash string, have bool) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO pieces (id, cid, idx, offset, size, hash, have, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cid, idx) DO UPDATE SET have=excluded.have, updated_at=excluded.updated_at`,
		uuid.New().String(), cid, idx, offset, size, hash, boolToInt(have), time.Now())
	return err
}

func (r *Repository) GetPieces(ctx context.Context, cid string) ([]Piece, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, cid, idx, offset, size, hash, have, updated_at FROM pieces WHERE cid=? ORDER BY idx ASC`, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Piece
	for rows.Next() {
		var p Piece
		var haveInt int
		if err := rows.Scan(&p.ID, &p.CID, &p.Index, &p.Offset, &p.Size, &p.Hash, &haveInt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Have = haveInt == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) MissingPieces(ctx context.Context, cid string) ([]Piece, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, cid, idx, offset, size, hash, have, updated_at FROM pieces WHERE cid=? AND have=0 ORDER BY idx ASC`, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Piece
	for rows.Next() {
		var p Piece
		var haveInt int
		if err := rows.Scan(&p.ID, &p.CID, &p.Index, &p.Offset, &p.Size, &p.Hash, &haveInt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Have = false
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) SetPeerScore(ctx context.Context, peerID string, delta float64) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var score float64
	err = tx.QueryRowContext(ctx, `SELECT score FROM peer_scores WHERE peer_id=?`, peerID).Scan(&score)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_, err = tx.ExecContext(ctx, `INSERT INTO peer_scores (peer_id, score, seen_at) VALUES (?, ?, ?)`, peerID, 10.0+delta, time.Now())
		} else {
			return err
		}
	} else {
		score += delta
		if score < -50 {
			score = -50
		}
		if score > 100 {
			score = 100
		}
		_, err = tx.ExecContext(ctx, `UPDATE peer_scores SET score=?, seen_at=? WHERE peer_id=?`, score, time.Now(), peerID)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) GetPeerScore(ctx context.Context, peerID string) (float64, error) {
	var s float64
	err := r.DB.QueryRowContext(ctx, `SELECT score FROM peer_scores WHERE peer_id=?`, peerID).Scan(&s)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return s, nil
}

func (r *Repository) SearchByFilename(ctx context.Context, q string) ([]LocalFile, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT cid, filename, file_size, '' as file_path, '' as file_hash, CURRENT_TIMESTAMP FROM metadata_index WHERE filename LIKE ? ORDER BY filename`, "%"+q+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []LocalFile
	for rows.Next() {
		var lf LocalFile
		if err := rows.Scan(&lf.CID, &lf.Filename, &lf.FileSize, &lf.FilePath, &lf.FileHash, &lf.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, lf)
	}
	return res, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// AddFileEntry adds a file entry for multi-file torrent support
func (r *Repository) AddFileEntry(ctx context.Context, cid, path string, size, offset int64, fileHash string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO file_entries (id, cid, path, size, offset, file_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cid, path) DO UPDATE SET size=excluded.size, offset=excluded.offset, file_hash=excluded.file_hash`,
		uuid.New().String(), cid, path, size, offset, fileHash, time.Now())
	return err
}

// GetFileEntries returns all file entries for a CID
func (r *Repository) GetFileEntries(ctx context.Context, cid string) ([]FileEntry, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, cid, path, size, offset, file_hash FROM file_entries WHERE cid=? ORDER BY offset ASC`, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []FileEntry
	for rows.Next() {
		var e FileEntry
		if err := rows.Scan(&e.ID, &e.CID, &e.Path, &e.Size, &e.Offset, &e.FileHash); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// RecordUpload records or updates upload statistics for a CID/peer combination
func (r *Repository) RecordUpload(ctx context.Context, cid, peerID string, bytesUploaded int64, chunksServed int) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO uploads (id, cid, peer_id, bytes_uploaded, chunks_served, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(cid, peer_id) DO UPDATE SET 
			bytes_uploaded = uploads.bytes_uploaded + excluded.bytes_uploaded,
			chunks_served = uploads.chunks_served + excluded.chunks_served,
			uploaded_at = excluded.uploaded_at`,
		uuid.New().String(), cid, peerID, bytesUploaded, chunksServed, time.Now())
	return err
}

// GetUploadStats returns upload statistics for all CIDs or a specific CID
func (r *Repository) GetUploadStats(ctx context.Context, cid string) ([]UploadRecord, error) {
	var rows *sql.Rows
	var err error
	if cid == "" {
		rows, err = r.DB.QueryContext(ctx, `SELECT id, cid, peer_id, bytes_uploaded, chunks_served, uploaded_at FROM uploads ORDER BY uploaded_at DESC`)
	} else {
		rows, err = r.DB.QueryContext(ctx, `SELECT id, cid, peer_id, bytes_uploaded, chunks_served, uploaded_at FROM uploads WHERE cid=? ORDER BY uploaded_at DESC`, cid)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []UploadRecord
	for rows.Next() {
		var r UploadRecord
		if err := rows.Scan(&r.ID, &r.CID, &r.PeerID, &r.BytesUploaded, &r.ChunksServed, &r.UploadedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetTotalUploadStats returns aggregated upload statistics
func (r *Repository) GetTotalUploadStats(ctx context.Context) (totalBytes int64, totalChunks int, uniquePeers int, err error) {
	err = r.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(bytes_uploaded), 0), COALESCE(SUM(chunks_served), 0), COUNT(DISTINCT peer_id) FROM uploads`).Scan(&totalBytes, &totalChunks, &uniquePeers)
	return
}

// GetUploadStatsByCID returns aggregated upload statistics for a specific CID
func (r *Repository) GetUploadStatsByCID(ctx context.Context, cid string) (totalBytes int64, totalChunks int, uniquePeers int, err error) {
	err = r.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(bytes_uploaded), 0), COALESCE(SUM(chunks_served), 0), COUNT(DISTINCT peer_id) FROM uploads WHERE cid=?`, cid).Scan(&totalBytes, &totalChunks, &uniquePeers)
	return
}
