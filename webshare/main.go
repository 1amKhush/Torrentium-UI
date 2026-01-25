// Package main implements the Torrentium Web Share Portal server.
// This server allows users to publish their file magnet links for easy sharing
// through a browsable web interface.
package main

import (
	"flag"
	"os"

	"github.com/1amkhush/torrentium/webshare/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuration flags
	port := flag.String("port", getEnvOrDefault("PORT", "8080"), "Server port")
	dbPath := flag.String("db", getEnvOrDefault("WEBSHARE_DB", "./webshare.db"), "Database path")
	corsOrigins := flag.String("cors", getEnvOrDefault("CORS_ORIGINS", "*"), "Allowed CORS origins (comma-separated)")
	apiKey := flag.String("api-key", getEnvOrDefault("API_KEY", ""), "Optional API key for publishing (leave empty for open publishing)")
	flag.Parse()

	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Initialize and run server
	cfg := server.Config{
		Port:        *port,
		DBPath:      *dbPath,
		CORSOrigins: *corsOrigins,
		APIKey:      *apiKey,
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create server")
	}

	log.Info().Str("port", cfg.Port).Msg("Starting Torrentium Web Share Portal")
	if err := srv.Run(); err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
