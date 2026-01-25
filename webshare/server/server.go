package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Server represents the web share portal server
type Server struct {
	cfg    Config
	db     *Database
	router *gin.Engine
}

// New creates a new server instance
func New(cfg Config) (*Server, error) {
	// Initialize database
	db, err := NewDatabase(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(requestLogger())

	// CORS
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	if cfg.CORSOrigins == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = strings.Split(cfg.CORSOrigins, ",")
	}
	router.Use(cors.New(corsConfig))

	srv := &Server{
		cfg:    cfg,
		db:     db,
		router: router,
	}

	// Register routes
	srv.registerRoutes()

	// Start cleanup goroutine
	go srv.cleanupTask()

	return srv, nil
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Debug().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Msg("request")
	}
}

func (s *Server) registerRoutes() {
	// API v1
	api := s.router.Group("/api/v1")
	{
		// Public endpoints
		api.GET("/files", s.listFiles)
		api.GET("/files/:cid", s.getFile)
		api.GET("/categories", s.getCategories)
		api.GET("/stats", s.getStats)
		api.POST("/files/:cid/download", s.trackDownload)
		api.POST("/report/:cid", s.reportFile)

		// Publishing endpoints (optionally protected by API key)
		publish := api.Group("/publish")
		if s.cfg.APIKey != "" {
			publish.Use(s.apiKeyAuth())
		}
		{
			publish.POST("", s.publishFile)
			publish.DELETE("/:cid", s.unpublishFile)
		}
	}

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Serve static files for frontend (in production)
	s.router.StaticFS("/portal", http.Dir("./frontend/dist"))
	s.router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/portal/")
	})
}

func (s *Server) apiKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey != s.cfg.APIKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing API key"})
			return
		}
		c.Next()
	}
}

// Run starts the server
func (s *Server) Run() error {
	return s.router.Run(":" + s.cfg.Port)
}

// Close closes the server resources
func (s *Server) Close() error {
	return s.db.Close()
}

func (s *Server) cleanupTask() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if removed, err := s.db.CleanupExpired(); err == nil && removed > 0 {
			log.Info().Int("removed", removed).Msg("Cleaned up expired files")
		}
	}
}

// ========== Handlers ==========

func (s *Server) listFiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	category := c.Query("category")
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "newest")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	files, total, err := s.db.ListPublicFiles(page, limit, category, search, sortBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files": files,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + limit - 1) / limit,
		},
	})
}

func (s *Server) getFile(c *gin.Context) {
	cid := c.Param("cid")

	file, err := s.db.GetFileByCID(cid)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch file"})
		return
	}

	// Check expiration
	if file.ExpiresAt != nil && file.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File has expired"})
		return
	}

	// Increment views
	s.db.IncrementViews(cid)

	c.JSON(http.StatusOK, file)
}

func (s *Server) publishFile(c *gin.Context) {
	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate category
	if req.Category != "" && !ValidCategory(req.Category) {
		req.Category = "other"
	}

	file, err := s.db.PublishFile(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish file"})
		return
	}

	log.Info().Str("cid", req.CID).Str("filename", req.Filename).Msg("File published")

	c.JSON(http.StatusCreated, gin.H{
		"message":    "File published successfully",
		"file":       file,
		"magnetLink": file.MagnetLink,
		"shareUrl":   fmt.Sprintf("/portal/#/file/%s", file.CID),
	})
}

func (s *Server) unpublishFile(c *gin.Context) {
	cid := c.Param("cid")

	if err := s.db.UnpublishFile(cid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unpublish file"})
		return
	}

	log.Info().Str("cid", cid).Msg("File unpublished")

	c.JSON(http.StatusOK, gin.H{"message": "File unpublished successfully"})
}

func (s *Server) getCategories(c *gin.Context) {
	categories, err := s.db.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"categories":  categories,
		"allCategories": Categories,
	})
}

func (s *Server) getStats(c *gin.Context) {
	stats, err := s.db.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (s *Server) trackDownload(c *gin.Context) {
	cid := c.Param("cid")

	if err := s.db.IncrementDownloads(cid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track download"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Download tracked"})
}

func (s *Server) reportFile(c *gin.Context) {
	cid := c.Param("cid")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get file by CID first
	file, err := s.db.GetFileByCID(cid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	reporterIP := c.ClientIP()
	if err := s.db.ReportFile(file.ID, req.Reason, reporterIP); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
		return
	}

	log.Warn().Str("cid", cid).Str("reason", req.Reason).Msg("File reported")

	c.JSON(http.StatusOK, gin.H{"message": "Report submitted"})
}
