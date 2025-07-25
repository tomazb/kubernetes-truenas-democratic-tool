package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/config"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/k8s"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/logging"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/metrics"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/monitor"
	"github.com/tomazb/kubernetes-truenas-democratic-tool/pkg/truenas"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "/app/config.yaml", "Path to configuration file")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
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

	logger.Info("Starting TrueNAS Monitor Service",
		zap.String("version", "0.1.0"),
		zap.String("config", *configPath),
	)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient(k8s.Config{
		Kubeconfig: cfg.Kubernetes.Kubeconfig,
		Namespace:  cfg.Kubernetes.Namespace,
		InCluster:  cfg.Kubernetes.InCluster,
	})
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Kubernetes client")
	}

	// Initialize TrueNAS client
	timeout, err := time.ParseDuration(cfg.TrueNAS.Timeout)
	if err != nil {
		logger.WithError(err).Fatal("Failed to parse TrueNAS timeout")
	}
	
	truenasClient, err := truenas.NewClient(truenas.Config{
		URL:      cfg.TrueNAS.URL,
		Username: cfg.TrueNAS.Username,
		Password: cfg.TrueNAS.Password,
		Timeout:  timeout,
	})
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize TrueNAS client")
	}

	// Initialize metrics exporter
	metricsExporter := metrics.NewExporter(metrics.Config{
		Enabled: cfg.Metrics.Enabled,
		Port:    cfg.Metrics.Port,
		Path:    cfg.Metrics.Path,
	})

	// Initialize monitor service
	monitorService, err := monitor.NewService(monitor.Config{
		K8sClient:       k8sClient,
		TruenasClient:   truenasClient,
		MetricsExporter: metricsExporter,
		Logger:          logger,
		ScanInterval:    cfg.Monitor.ScanInterval,
	})
	if err != nil {
		logger.WithError(err).Fatal("Failed to create monitor service")
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start monitor service
	if err := monitorService.Start(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to start monitor service")
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Monitor service started successfully")
	<-sigChan

	logger.Info("Shutting down monitor service...")
	cancel()

	// Give services time to shutdown gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := monitorService.Stop(shutdownCtx); err != nil {
		logger.WithError(err).Error("Error during shutdown")
		os.Exit(1)
	}

	logger.Info("Monitor service stopped successfully")
}

func initLogger(level string) (*logging.Logger, error) {
	config := logging.Config{
		Level:       level,
		Development: false,
		Encoding:    "json",
	}
	
	return logging.NewLogger(config)
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