package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	Use:   "truenas-monitor",
	Short: "Monitor TrueNAS storage in Kubernetes/OpenShift",
	Long: `TrueNAS Monitor continuously watches for orphaned resources,
configuration issues, and storage problems in your Kubernetes cluster
using TrueNAS Scale storage via democratic-csi.`,
	Run: run,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default: ./config.yaml)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("json-logs", "j", false, "output logs in JSON format")

	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log.json", rootCmd.PersistentFlags().Lookup("json-logs"))
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

	logger.Info("Starting TrueNAS Monitor",
		zap.String("version", version),
		zap.String("commit", gitCommit),
		zap.String("buildDate", buildDate),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// TODO: Initialize and start monitor service
	logger.Info("Monitor service would start here")

	<-ctx.Done()
	logger.Info("Shutting down monitor service")
}

func setupLogger() (*zap.Logger, error) {
	var config zap.Config

	if viper.GetBool("log.json") {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

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