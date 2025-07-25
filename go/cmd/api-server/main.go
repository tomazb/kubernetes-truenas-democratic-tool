package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/api"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/config"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/k8s"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "/app/config.yaml", "Path to configuration file")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	port       = flag.Int("port", 8080, "Server port")
	healthCmd  = flag.Bool("health", false, "Run health check and exit")
)

func main() {
	flag.Parse()

	// Handle health check command
	if *healthCmd {
		os.Exit(healthCheck())
	}

	// Initialize logger
	logger, err := initLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting TrueNAS API Server",
		zap.String("version", "0.1.0"),
		zap.String("config", *configPath),
		zap.Int("port", *port))

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient(k8s.Config{
		Kubeconfig: cfg.Kubernetes.Kubeconfig,
		Namespace:  cfg.Kubernetes.Namespace,
		InCluster:  cfg.Kubernetes.InCluster,
	})
	if err != nil {
		logger.Fatal("Failed to initialize Kubernetes client", zap.Error(err))
	}

	// Initialize TrueNAS client
	timeout, err := time.ParseDuration(cfg.TrueNAS.Timeout)
	if err != nil {
		logger.Fatal("Failed to parse TrueNAS timeout", zap.Error(err))
	}
	
	truenasClient, err := truenas.NewClient(truenas.Config{
		URL:      cfg.TrueNAS.URL,
		Username: cfg.TrueNAS.Username,
		Password: cfg.TrueNAS.Password,
		Timeout:  timeout,
	})
	if err != nil {
		logger.Fatal("Failed to initialize TrueNAS client", zap.Error(err))
	}

	// Initialize API server
	apiServer, err := api.NewServer(api.Config{
		Port:          *port,
		K8sClient:     k8sClient,
		TruenasClient: truenasClient,
		Logger:        logger,
	})
	if err != nil {
		logger.Fatal("Failed to initialize API server", zap.Error(err))
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start API server
	if err := apiServer.Start(ctx); err != nil {
		logger.Fatal("Failed to start API server", zap.Error(err))
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("API server started successfully", zap.Int("port", *port))
	<-sigChan

	logger.Info("Shutting down API server...")
	cancel()

	// Give server time to shutdown gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := apiServer.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("API server stopped successfully")
}

func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zap.AtomicLevel
	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	config := zap.NewProductionConfig()
	config.Level = zapLevel
	config.DisableStacktrace = true

	return config.Build()
}

func healthCheck() int {
	// Simple health check - verify we can start
	logger, err := initLogger("error")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		return 1
	}
	defer logger.Sync()

	logger.Info("Health check passed")
	return 0
}