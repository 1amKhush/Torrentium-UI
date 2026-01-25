package server

// Config holds the server configuration
type Config struct {
	// Port to listen on
	Port string
	// DBPath is the path to the SQLite database
	DBPath string
	// CORSOrigins is a comma-separated list of allowed origins
	CORSOrigins string
	// APIKey is an optional key for publishing (empty = open publishing)
	APIKey string
}
