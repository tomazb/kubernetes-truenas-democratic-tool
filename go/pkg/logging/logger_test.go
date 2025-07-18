package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
			name: "json format",
			config: Config{
				Level:  "info",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "console format",
			config: Config{
				Level:  "debug",
				Format: "console",
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
			wantErr: true,
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

func TestLoggerWithContext(t *testing.T) {
	logger, err := NewLogger(Config{Level: "info"})
	require.NoError(t, err)

	// Test with correlation ID
	ctx := NewContextWithCorrelationID(context.Background(), "test-correlation-id")
	contextLogger := logger.WithContext(ctx)
	assert.NotNil(t, contextLogger)

	// Test with component
	ctx = NewContextWithComponent(ctx, "test-component")
	contextLogger = logger.WithContext(ctx)
	assert.NotNil(t, contextLogger)

	// Test with operation
	ctx = NewContextWithOperation(ctx, "test-operation")
	contextLogger = logger.WithContext(ctx)
	assert.NotNil(t, contextLogger)

	// Test with empty context
	emptyContextLogger := logger.WithContext(context.Background())
	assert.NotNil(t, emptyContextLogger)
}

func TestLoggerWithMethods(t *testing.T) {
	logger, err := NewLogger(Config{Level: "info"})
	require.NoError(t, err)

	// Test WithComponent
	componentLogger := logger.WithComponent("test-component")
	assert.NotNil(t, componentLogger)

	// Test WithOperation
	operationLogger := logger.WithOperation("test-operation")
	assert.NotNil(t, operationLogger)

	// Test WithCorrelationID
	correlationLogger := logger.WithCorrelationID("test-correlation-id")
	assert.NotNil(t, correlationLogger)

	// Test WithError
	testErr := assert.AnError
	errorLogger := logger.WithError(testErr)
	assert.NotNil(t, errorLogger)

	// Test WithFields
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	fieldsLogger := logger.WithFields(fields)
	assert.NotNil(t, fieldsLogger)
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Test correlation ID
	correlationID := "test-correlation-id"
	ctx = NewContextWithCorrelationID(ctx, correlationID)
	assert.Equal(t, correlationID, GetCorrelationID(ctx))

	// Test component
	component := "test-component"
	ctx = NewContextWithComponent(ctx, component)
	assert.Equal(t, component, ctx.Value(ComponentKey))

	// Test operation
	operation := "test-operation"
	ctx = NewContextWithOperation(ctx, operation)
	assert.Equal(t, operation, ctx.Value(OperationKey))

	// Test empty context
	emptyCtx := context.Background()
	assert.Empty(t, GetCorrelationID(emptyCtx))
}

func TestLogOperations(t *testing.T) {
	logger, err := NewLogger(Config{Level: "info"})
	require.NoError(t, err)

	// Test LogHTTPRequest
	logger.LogHTTPRequest("GET", "/api/v1/orphans", "test-agent", "127.0.0.1", 200, 150)

	// Test LogK8sOperation
	logger.LogK8sOperation("list", "persistentvolumes", "", 10, 250)
	logger.LogK8sOperation("list", "persistentvolumeclaims", "default", 5, 100)

	// Test LogTrueNASOperation
	logger.LogTrueNASOperation("list_volumes", "/api/v2.0/pool/dataset", 200, 300)

	// Test LogScanResult
	logger.LogScanResult(2, 1, 0, 5000)
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
			name: "invalid level",
			config: Config{Level: "invalid"},
			wantErr: true,
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

func TestLoggerChaining(t *testing.T) {
	logger, err := NewLogger(Config{Level: "info"})
	require.NoError(t, err)

	// Test method chaining
	chainedLogger := logger.
		WithComponent("test-component").
		WithOperation("test-operation").
		WithCorrelationID("test-correlation-id")

	assert.NotNil(t, chainedLogger)

	// Test logging with chained logger
	chainedLogger.Info("Test message with chained context")
}