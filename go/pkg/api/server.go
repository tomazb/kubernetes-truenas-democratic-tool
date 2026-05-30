package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/k8s"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/orphan"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	defaultOrphanAgeThreshold     = 24 * time.Hour
	defaultOrphanAgeThresholdQuery = "24h"
)

// Server represents the API server
type Server struct {
	server                  *http.Server
	k8sClient               k8s.Client
	truenasClient           truenas.Client
	logger                  *zap.Logger
	orphanDetector *orphan.Detector
}

// Config holds the server configuration
type Config struct {
	Port          int
	K8sClient     k8s.Client
	TruenasClient truenas.Client
	Logger        *zap.Logger
}

// NewServer creates a new API server with comprehensive middleware
func NewServer(config Config) (*Server, error) {
	if config.K8sClient == nil {
		return nil, fmt.Errorf("k8sClient is required")
	}
	if config.TruenasClient == nil {
		return nil, fmt.Errorf("truenasClient is required")
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(corsMiddleware())

	// Add request ID middleware for tracing
	router.Use(requestIDMiddleware())

	// Add logging middleware
	router.Use(loggingMiddleware(logger))

	// Add rate limiting middleware
	router.Use(rateLimitMiddleware())

	orphanDetector, err := orphan.NewDetector(config.K8sClient, config.TruenasClient, orphan.Config{
		AgeThreshold:      defaultOrphanAgeThreshold,
		SnapshotRetention: 30 * 24 * time.Hour,
		DryRun:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create orphan detector: %w", err)
	}

	server := &Server{
		k8sClient:      config.K8sClient,
		truenasClient:  config.TruenasClient,
		logger:         logger,
		orphanDetector: orphanDetector,
	}

	// Setup routes
	server.setupRoutes(router)

	// Create HTTP server with enhanced configuration
	server.server = &http.Server{
		Addr:           fmt.Sprintf(":%d", config.Port),
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	return server, nil
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting API server", zap.String("addr", s.server.Addr))

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("API server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the API server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping API server")
	return s.server.Shutdown(ctx)
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", s.healthHandler)
	router.GET("/ready", s.readyHandler)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Orphaned resources
		v1.GET("/orphans", s.listOrphansHandler)
		v1.GET("/orphans/pvs", s.listOrphanedPVsHandler)
		v1.GET("/orphans/pvcs", s.listOrphanedPVCsHandler)
		v1.GET("/orphans/snapshots", s.listOrphanedSnapshotsHandler)

		// Storage analysis
		v1.GET("/analysis", s.storageAnalysisHandler)
		v1.GET("/analysis/usage", s.storageUsageHandler)
		v1.GET("/analysis/trends", s.storageTrendsHandler)

		// Resources
		v1.GET("/resources/pvs", s.listPVsHandler)
		v1.GET("/resources/pvcs", s.listPVCsHandler)
		v1.GET("/resources/snapshots", s.listSnapshotsHandler)
		v1.GET("/resources/storageclasses", s.listStorageClassesHandler)

		// TrueNAS resources
		v1.GET("/truenas/volumes", s.listTrueNASVolumesHandler)
		v1.GET("/truenas/snapshots", s.listTrueNASSnapshotsHandler)
		v1.GET("/truenas/pools", s.listTrueNASPoolsHandler)
		v1.GET("/truenas/info", s.getTrueNASInfoHandler)

		// Validation
		v1.GET("/validate", s.validateHandler)
		v1.GET("/validate/config", s.validateConfigHandler)
		v1.GET("/validate/connectivity", s.validateConnectivityHandler)

		// Reports
		v1.GET("/reports/summary", s.summaryReportHandler)
		v1.GET("/reports/detailed", s.detailedReportHandler)
	}
}

func (s *Server) parseAgeThreshold(c *gin.Context) (time.Duration, string, bool) {
	ageThresholdRaw, ok := c.GetQuery("age_threshold")
	if !ok {
		return defaultOrphanAgeThreshold, defaultOrphanAgeThresholdQuery, true
	}

	parsed, err := time.ParseDuration(ageThresholdRaw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid age_threshold format",
		})
		return 0, ageThresholdRaw, false
	}
	if parsed <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "age_threshold must be greater than 0",
		})
		return 0, ageThresholdRaw, false
	}
	return parsed, ageThresholdRaw, true
}

func (s *Server) runOrphanDetection(ctx context.Context, namespace string, ageThreshold time.Duration) (*orphan.DetectionResult, error) {
	return s.orphanDetector.WithAgeThreshold(ageThreshold).DetectOrphanedResources(ctx, namespace)
}

func (s *Server) runOrphanPVDetection(ctx context.Context, ageThreshold time.Duration) (*orphan.DetectionResult, error) {
	return s.orphanDetector.WithAgeThreshold(ageThreshold).DetectOrphanedPVs(ctx)
}

func notImplemented(c *gin.Context, endpoint string) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":    "not_implemented",
		"message":  "endpoint not implemented",
		"endpoint": endpoint,
	})
}

// healthHandler handles health check requests
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "0.1.0",
	})
}

// readyHandler handles readiness check requests
func (s *Server) readyHandler(c *gin.Context) {
	ctx := c.Request.Context()

	// Test Kubernetes connection
	if err := s.k8sClient.TestConnection(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "kubernetes connection failed",
			"error":  err.Error(),
		})
		return
	}

	// Test TrueNAS connection
	if err := s.truenasClient.TestConnection(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "truenas connection failed",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now().UTC(),
	})
}

// listOrphansHandler handles requests for all orphaned resources
func (s *Server) listOrphansHandler(c *gin.Context) {
	namespace := c.Query("namespace")
	ageThreshold, ageThresholdRaw, ok := s.parseAgeThreshold(c)
	if !ok {
		return
	}

	result, err := s.runOrphanDetection(c.Request.Context(), namespace, ageThreshold)
	if err != nil {
		s.logger.Error("Failed to detect orphaned resources", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "orphan detection failed",
		})
		return
	}

	totalOrphans := len(result.OrphanedPVs) + len(result.OrphanedPVCs) + len(result.OrphanedSnapshots)

	c.JSON(http.StatusOK, gin.H{
		"timestamp":          result.Timestamp,
		"namespace":          namespace,
		"age_threshold":      ageThresholdRaw,
		"orphaned_pvs":       result.OrphanedPVs,
		"orphaned_pvcs":      result.OrphanedPVCs,
		"orphaned_snapshots": result.OrphanedSnapshots,
		"total_pvs":          result.TotalPVs,
		"total_pvcs":         result.TotalPVCs,
		"total_snapshots":    result.TotalSnapshots,
		"scan_duration":      result.ScanDuration.String(),
		"total_orphans":      totalOrphans,
	})
}

// listOrphanedPVsHandler handles requests for orphaned PVs
func (s *Server) listOrphanedPVsHandler(c *gin.Context) {
	ageThreshold, ageThresholdRaw, ok := s.parseAgeThreshold(c)
	if !ok {
		return
	}

	result, err := s.runOrphanPVDetection(c.Request.Context(), ageThreshold)
	if err != nil {
		s.logger.Error("Failed to detect orphaned PVs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "orphan detection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"timestamp":     result.Timestamp,
		"age_threshold": ageThresholdRaw,
		"total_pvs":     result.TotalPVs,
		"orphaned_pvs":  result.OrphanedPVs,
		"total_orphans": len(result.OrphanedPVs),
	})
}

// listPVsHandler handles requests for all PVs
func (s *Server) listPVsHandler(c *gin.Context) {
	ctx := c.Request.Context()

	pvs, err := s.k8sClient.ListPersistentVolumes(ctx)
	if err != nil {
		s.logger.Error("Failed to list PVs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list persistent volumes",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now().UTC(),
		"count":     len(pvs),
		"items":     pvs,
	})
}

// listTrueNASVolumesHandler handles requests for TrueNAS volumes
func (s *Server) listTrueNASVolumesHandler(c *gin.Context) {
	ctx := c.Request.Context()

	volumes, err := s.truenasClient.ListVolumes(ctx)
	if err != nil {
		s.logger.Error("Failed to list TrueNAS volumes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list truenas volumes",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now().UTC(),
		"count":     len(volumes),
		"items":     volumes,
	})
}

// validateHandler handles validation requests
func (s *Server) validateHandler(c *gin.Context) {
	ctx := c.Request.Context()

	results := make(map[string]interface{})

	// Test Kubernetes connection
	if err := s.k8sClient.TestConnection(ctx); err != nil {
		results["kubernetes"] = gin.H{
			"status": "failed",
			"error":  err.Error(),
		}
	} else {
		results["kubernetes"] = gin.H{
			"status": "passed",
		}
	}

	// Test TrueNAS connection
	if err := s.truenasClient.TestConnection(ctx); err != nil {
		results["truenas"] = gin.H{
			"status": "failed",
			"error":  err.Error(),
		}
	} else {
		results["truenas"] = gin.H{
			"status": "passed",
		}
	}

	// Determine overall status
	allPassed := true
	for _, result := range results {
		if result.(gin.H)["status"] != "passed" {
			allPassed = false
			break
		}
	}

	status := http.StatusOK
	if !allPassed {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"timestamp":      time.Now().UTC(),
		"overall_status": allPassed,
		"checks":         results,
	})
}

func (s *Server) listOrphanedPVCsHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/orphans/pvcs")
}

func (s *Server) listOrphanedSnapshotsHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/orphans/snapshots")
}

func (s *Server) storageAnalysisHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/analysis")
}

func (s *Server) storageUsageHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/analysis/usage")
}

func (s *Server) storageTrendsHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/analysis/trends")
}

func (s *Server) listPVCsHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/resources/pvcs")
}

func (s *Server) listSnapshotsHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/resources/snapshots")
}

func (s *Server) listStorageClassesHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/resources/storageclasses")
}

func (s *Server) listTrueNASSnapshotsHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/truenas/snapshots")
}

func (s *Server) listTrueNASPoolsHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/truenas/pools")
}

func (s *Server) getTrueNASInfoHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/truenas/info")
}

func (s *Server) validateConfigHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/validate/config")
}

func (s *Server) validateConnectivityHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/validate/connectivity")
}

func (s *Server) summaryReportHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/reports/summary")
}

func (s *Server) detailedReportHandler(c *gin.Context) {
	notImplemented(c, "/api/v1/reports/detailed")
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.Info("HTTP request",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
		)
		return ""
	})
}

// requestIDMiddleware adds a unique request ID to each request
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// rateLimitMiddleware implements rate limiting
func rateLimitMiddleware() gin.HandlerFunc {
	// Create a rate limiter: 100 requests per minute
	limiter := rate.NewLimiter(rate.Every(time.Minute/100), 100)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			retryAfter := time.Second
			if reservation := limiter.Reserve(); reservation.OK() {
				retryAfter = reservation.Delay()
				reservation.Cancel()
			}
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter.String(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
