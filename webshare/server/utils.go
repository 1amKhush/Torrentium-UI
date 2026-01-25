package server

import (
	"fmt"
	"net/url"
)

// GenerateMagnetLink creates a Torrentium magnet link
func GenerateMagnetLink(cid, filename string, fileSize int64) string {
	// Format: torrentium://<cid>?dn=<filename>&sz=<size>
	return fmt.Sprintf("torrentium://%s?dn=%s&sz=%d",
		cid,
		url.QueryEscape(filename),
		fileSize,
	)
}

// humanizeBytes converts bytes to human readable format
func humanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Categories for file classification
var Categories = []string{
	"documents",
	"images",
	"videos",
	"audio",
	"archives",
	"software",
	"code",
	"other",
}

// ValidCategory checks if a category is valid
func ValidCategory(cat string) bool {
	for _, c := range Categories {
		if c == cat {
			return true
		}
	}
	return false
}
