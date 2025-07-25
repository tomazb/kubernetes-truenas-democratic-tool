package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "default config",
			config: Config{},
			wantErr: false,
		},
		{
			name: "json encoding",
			config: Config{
				Level:    "info",
				Encoding: "json",
			},
			wantErr: false,
		},
		{
			name: "console encoding",
			config: Config{
				Level:    "debug",
				Encoding: "console",
			},
			wantErr: false,
		},
		{
			name: "development mode",
			config: Config{
				Level:       "debug",
				Development: true,
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: Config{
				Level: "invalid",
			},
			wantErr: false, // Invalid level defaults to info
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
				assert.NotNil(t, logger.Logger)
			}
		})
	}
}

func TestLoggerWithMethods(t *testing.T) {
	logger, err := NewLogger(Config{Level: "info"})
	require.NoError(t, err)

	// Test WithComponent
	componentLogger := logger.WithComponent("test-component")
	assert.NotNil(t, componentLogger)

	// Test WithRequestID
	requestLogger := logger.WithRequestID("test-request-id")
	assert.NotNil(t, requestLogger)

	// Test WithNamespace
	namespaceLogger := logger.WithNamespace("test-namespace")
	assert.NotNil(t, namespaceLogger)

	// Test WithError
	testErr := assert.AnError
	errorLogger := logger.WithError(testErr)
	assert.NotNil(t, errorLogger)
}

func TestLogOperations(t *testing.T) {
	logger, err := NewLogger(Config{Level: "info"})
	require.NoError(t, err)

	// Test LogAPIRequest
	logger.LogAPIRequest("GET", "/api/v1/orphans", "127.0.0.1", 200, 150)

	// Test LogK8sOperation
	logger.LogK8sOperation("list", "persistentvolumes", "", "", nil)
	logger.LogK8sOperation("list", "persistentvolumeclaims", "default", "", nil)

	// Test LogTrueNASOperation
	logger.LogTrueNASOperation("list", "datasets", 200, nil)

	// Test LogOrphanDetection
	logger.LogOrphanDetection("volumes", 10, 2, 5000)

	// Test LogSecurityEvent
	logger.LogSecurityEvent("access", "test-user", "volumes", true)
}

func TestLogLevels(t *testing.T) {
	logger, err := NewLogger(Config{Level: "debug"})
	require.NoError(t, err)

	// Test all log levels
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")

	// Test with fields
	logger.Info("Message with fields",
		zap.String("key1", "value1"),
		zap.Int("key2", 42),
	)
}

func TestLoggerLevelControl(t *testing.T) {
	logger, err := NewLogger(Config{Level: "warn"})
	require.NoError(t, err)

	// Test getting current level
	assert.Equal(t, "warn", logger.GetLevel())

	// Test setting level
	err = logger.SetLevel("debug")
	assert.NoError(t, err)
	assert.Equal(t, "debug", logger.GetLevel())

	// Test invalid level
	err = logger.SetLevel("invalid")
	assert.Error(t, err)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid info level",
			config: Config{Level: "info"},
			wantErr: false,
		},
		{
			name: "valid debug level",
			config: Config{Level: "debug"},
			wantErr: false,
		},
		{
			name: "valid warn level",
			config: Config{Level: "warn"},
			wantErr: false,
		},
		{
			name: "valid error level",
			config: Config{Level: "error"},
			wantErr: false,
		},
		{
			name: "invalid level defaults to info",
			config: Config{Level: "invalid"},
			wantErr: false,
		},
		{
			name: "empty level defaults to info",
			config: Config{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLogger(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}