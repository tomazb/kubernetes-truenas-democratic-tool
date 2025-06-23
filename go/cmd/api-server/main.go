package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	version   = "dev"
	gitCommit = "unknown"
	buildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "truenas-api-server",
	Short: "API server for TrueNAS storage monitoring",
	Long:  `RESTful API server providing access to TrueNAS storage monitoring data and operations.`,
	Run:   run,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default: ./config.yaml)")
	rootCmd.PersistentFlags().StringP("listen", "l", ":8080", "listen address")
	rootCmd.PersistentFlags().StringP("log-level", "", "info", "log level (debug, info, warn, error)")

	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("api.listen", rootCmd.PersistentFlags().Lookup("listen"))
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
}

func initConfig() {
	if cfgFile := viper.GetString("config"); cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/truenas-monitor/")
	}

	viper.SetEnvPrefix("TRUENAS_MONITOR")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}
	}
}

func run(cmd *cobra.Command, args []string) {
	logger, err := setupLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting TrueNAS API Server",
		zap.String("version", version),
		zap.String("commit", gitCommit),
		zap.String("buildDate", buildDate),
	)

	// Setup Gin
	if viper.GetString("log.level") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := setupRouter(logger)

	srv := &http.Server{
		Addr:         viper.GetString("api.listen"),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("API server listening", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down API server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("API server stopped")
}

func setupRouter(logger *zap.Logger) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(ginLogger(logger))

	// Health check
	router.GET("/health", healthCheck)
	router.GET("/ready", readinessCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/version", versionHandler)
		// TODO: Add more routes
	}

	return router
}

func ginLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("HTTP request",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", time.Since(start)),
			zap.String("user-agent", c.Request.UserAgent()),
		)
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

func readinessCheck(c *gin.Context) {
	// TODO: Check actual readiness (k8s connection, truenas connection, etc.)
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"time":   time.Now().UTC(),
	})
}

func versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":   version,
		"gitCommit": gitCommit,
		"buildDate": buildDate,
	})
}

func setupLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()

	level := viper.GetString("log.level")
	if err := config.Level.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	return config.Build()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}