package server

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// SharedFile represents a published file in the database
type SharedFile struct {
	ID           string    `json:"id"`
	CID          string    `json:"cid"`
	Filename     string    `json:"filename"`
	FileSize     int64     `json:"fileSize"`
	SizeHuman    string    `json:"sizeHuman"`
	Description  string    `json:"description,omitempty"`
	Category     string    `json:"category,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	PublisherID  string    `json:"publisherId,omitempty"`
	Visibility   string    `json:"visibility"` // "public" or "unlisted"
	MagnetLink   string    `json:"magnetLink"`
	Downloads    int       `json:"downloads"`
	Views        int       `json:"views"`
	PublishedAt  time.Time `json:"publishedAt"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
}

// PublishRequest is the request body for publishing a file
type PublishRequest struct {
	CID         string   `json:"cid" binding:"required"`
	Filename    string   `json:"filename" binding:"required"`
	FileSize    int64    `json:"fileSize" binding:"required"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	PublisherID string   `json:"publisherId"`
	Visibility  string   `json:"visibility"` // "public" or "unlisted", defaults to "public"
	ExpiresIn   int      `json:"expiresIn"`  // Hours until expiration, 0 = never
}

// Database wraps the SQL database with repository methods
type Database struct {
	db *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(path string) (*Database, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &Database{db: db}, nil
}

func createSchema(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS shared_files (
			id TEXT PRIMARY KEY,
			cid TEXT UNIQUE NOT NULL,
			filename TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			description TEXT DEFAULT '',
			category TEXT DEFAULT 'other',
			tags TEXT DEFAULT '',
			publisher_id TEXT DEFAULT '',
			visibility TEXT DEFAULT 'public',
			downloads INTEGER DEFAULT 0,
			views INTEGER DEFAULT 0,
			published_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME DEFAULT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_shared_files_cid ON shared_files(cid);`,
		`CREATE INDEX IF NOT EXISTS idx_shared_files_visibility ON shared_files(visibility);`,
		`CREATE INDEX IF NOT EXISTS idx_shared_files_category ON shared_files(category);`,
		`CREATE INDEX IF NOT EXISTS idx_shared_files_published_at ON shared_files(published_at);`,
		`CREATE TABLE IF NOT EXISTS reports (
			id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL,
			reason TEXT NOT NULL,
			reporter_ip TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'pending',
			FOREIGN KEY (file_id) REFERENCES shared_files(id)
		);`,
	}

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// PublishFile adds a new shared file to the database
func (d *Database) PublishFile(req PublishRequest) (*SharedFile, error) {
	id := uuid.New().String()
	visibility := req.Visibility
	if visibility == "" {
		visibility = "public"
	}

	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
		expiresAt = &t
	}

	// Convert tags to comma-separated string
	tagsStr := ""
	for i, tag := range req.Tags {
		if i > 0 {
			tagsStr += ","
		}
		tagsStr += tag
	}

	category := req.Category
	if category == "" {
		category = "other"
	}

	query := `INSERT INTO shared_files (id, cid, filename, file_size, description, category, tags, publisher_id, visibility, expires_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			  ON CONFLICT(cid) DO UPDATE SET 
			  	filename = excluded.filename,
			  	file_size = excluded.file_size,
			  	description = excluded.description,
			  	category = excluded.category,
			  	tags = excluded.tags,
			  	visibility = excluded.visibility,
			  	expires_at = excluded.expires_at`

	_, err := d.db.Exec(query, id, req.CID, req.Filename, req.FileSize, req.Description, category, tagsStr, req.PublisherID, visibility, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to publish file: %w", err)
	}

	return d.GetFileByCID(req.CID)
}

// GetFileByCID retrieves a file by its CID
func (d *Database) GetFileByCID(cid string) (*SharedFile, error) {
	row := d.db.QueryRow(`SELECT id, cid, filename, file_size, description, category, tags, publisher_id, visibility, downloads, views, published_at, expires_at
						  FROM shared_files WHERE cid = ?`, cid)

	return scanSharedFile(row)
}

// GetFileByID retrieves a file by its ID
func (d *Database) GetFileByID(id string) (*SharedFile, error) {
	row := d.db.QueryRow(`SELECT id, cid, filename, file_size, description, category, tags, publisher_id, visibility, downloads, views, published_at, expires_at
						  FROM shared_files WHERE id = ?`, id)

	return scanSharedFile(row)
}

func scanSharedFile(row *sql.Row) (*SharedFile, error) {
	var f SharedFile
	var tagsStr string
	var expiresAt sql.NullTime

	err := row.Scan(&f.ID, &f.CID, &f.Filename, &f.FileSize, &f.Description, &f.Category, &tagsStr, &f.PublisherID, &f.Visibility, &f.Downloads, &f.Views, &f.PublishedAt, &expiresAt)
	if err != nil {
		return nil, err
	}

	// Parse tags
	if tagsStr != "" {
		f.Tags = splitTags(tagsStr)
	}

	if expiresAt.Valid {
		f.ExpiresAt = &expiresAt.Time
	}

	// Generate magnet link
	f.MagnetLink = GenerateMagnetLink(f.CID, f.Filename, f.FileSize)
	f.SizeHuman = humanizeBytes(f.FileSize)

	return &f, nil
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	var tags []string
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				tags = append(tags, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		tags = append(tags, current)
	}
	return tags
}

// ListPublicFiles returns paginated public files
func (d *Database) ListPublicFiles(page, limit int, category, search, sortBy string) ([]SharedFile, int, error) {
	offset := (page - 1) * limit

	// Build query conditions
	conditions := "WHERE visibility = 'public' AND (expires_at IS NULL OR expires_at > datetime('now'))"
	args := []interface{}{}

	if category != "" && category != "all" {
		conditions += " AND category = ?"
		args = append(args, category)
	}

	if search != "" {
		conditions += " AND (filename LIKE ? OR description LIKE ? OR tags LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) FROM shared_files " + conditions
	if err := d.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Sort order
	orderBy := "published_at DESC"
	switch sortBy {
	case "downloads":
		orderBy = "downloads DESC"
	case "views":
		orderBy = "views DESC"
	case "size":
		orderBy = "file_size DESC"
	case "name":
		orderBy = "filename ASC"
	case "oldest":
		orderBy = "published_at ASC"
	}

	// Fetch files
	query := fmt.Sprintf(`SELECT id, cid, filename, file_size, description, category, tags, publisher_id, visibility, downloads, views, published_at, expires_at
						  FROM shared_files %s ORDER BY %s LIMIT ? OFFSET ?`, conditions, orderBy)
	args = append(args, limit, offset)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []SharedFile
	for rows.Next() {
		var f SharedFile
		var tagsStr string
		var expiresAt sql.NullTime

		if err := rows.Scan(&f.ID, &f.CID, &f.Filename, &f.FileSize, &f.Description, &f.Category, &tagsStr, &f.PublisherID, &f.Visibility, &f.Downloads, &f.Views, &f.PublishedAt, &expiresAt); err != nil {
			return nil, 0, err
		}

		if tagsStr != "" {
			f.Tags = splitTags(tagsStr)
		}
		if expiresAt.Valid {
			f.ExpiresAt = &expiresAt.Time
		}
		f.MagnetLink = GenerateMagnetLink(f.CID, f.Filename, f.FileSize)
		f.SizeHuman = humanizeBytes(f.FileSize)

		files = append(files, f)
	}

	return files, total, nil
}

// IncrementDownloads increments the download counter for a file
func (d *Database) IncrementDownloads(cid string) error {
	_, err := d.db.Exec("UPDATE shared_files SET downloads = downloads + 1 WHERE cid = ?", cid)
	return err
}

// IncrementViews increments the view counter for a file
func (d *Database) IncrementViews(cid string) error {
	_, err := d.db.Exec("UPDATE shared_files SET views = views + 1 WHERE cid = ?", cid)
	return err
}

// UnpublishFile removes a file from the portal
func (d *Database) UnpublishFile(cid string) error {
	_, err := d.db.Exec("DELETE FROM shared_files WHERE cid = ?", cid)
	return err
}

// ReportFile creates a report for a file
func (d *Database) ReportFile(fileID, reason, reporterIP string) error {
	id := uuid.New().String()
	_, err := d.db.Exec("INSERT INTO reports (id, file_id, reason, reporter_ip) VALUES (?, ?, ?, ?)",
		id, fileID, reason, reporterIP)
	return err
}

// GetCategories returns all distinct categories
func (d *Database) GetCategories() ([]string, error) {
	rows, err := d.db.Query("SELECT DISTINCT category FROM shared_files WHERE visibility = 'public' ORDER BY category")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

// GetStats returns portal statistics
func (d *Database) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total files
	var totalFiles int
	d.db.QueryRow("SELECT COUNT(*) FROM shared_files WHERE visibility = 'public'").Scan(&totalFiles)
	stats["totalFiles"] = totalFiles

	// Total downloads
	var totalDownloads int
	d.db.QueryRow("SELECT COALESCE(SUM(downloads), 0) FROM shared_files").Scan(&totalDownloads)
	stats["totalDownloads"] = totalDownloads

	// Total size
	var totalSize int64
	d.db.QueryRow("SELECT COALESCE(SUM(file_size), 0) FROM shared_files WHERE visibility = 'public'").Scan(&totalSize)
	stats["totalSize"] = totalSize
	stats["totalSizeHuman"] = humanizeBytes(totalSize)

	return stats, nil
}

// CleanupExpired removes expired files
func (d *Database) CleanupExpired() (int, error) {
	result, err := d.db.Exec("DELETE FROM shared_files WHERE expires_at IS NOT NULL AND expires_at < datetime('now')")
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}
