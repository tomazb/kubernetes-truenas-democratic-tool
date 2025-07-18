package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger with additional functionality
type Logger struct {
	*zap.Logger
	level zap.AtomicLevel
}

// Config holds logger configuration
type Config struct {
	Level       string `yaml:"level" json:"level"`
	Development bool   `yaml:"development" json:"development"`
	Encoding    string `yaml:"encoding" json:"encoding"` // json or console
}

// NewLogger creates a new structured logger
func NewLogger(config Config) (*Logger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	atomicLevel := zap.NewAtomicLevelAt(level)

	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	if config.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Choose encoder
	var encoder zapcore.Encoder
	if config.Encoding == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		atomicLevel,
	)

	// Create logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{
		Logger: logger,
		level:  atomicLevel,
	}, nil
}

// SetLevel dynamically changes the log level
func (l *Logger) SetLevel(level string) error {
	zapLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return err
	}
	l.level.SetLevel(zapLevel)
	return nil
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() string {
	return l.level.Level().String()
}

// WithRequestID adds a request ID to the logger context
func (l *Logger) WithRequestID(requestID string) *zap.Logger {
	return l.With(zap.String("request_id", requestID))
}

// WithComponent adds a component name to the logger context
func (l *Logger) WithComponent(component string) *zap.Logger {
	return l.With(zap.String("component", component))
}

// WithNamespace adds a Kubernetes namespace to the logger context
func (l *Logger) WithNamespace(namespace string) *zap.Logger {
	return l.With(zap.String("namespace", namespace))
}

// WithError adds an error to the logger context
func (l *Logger) WithError(err error) *zap.Logger {
	return l.With(zap.Error(err))
}

// LogAPIRequest logs an API request with standard fields
func (l *Logger) LogAPIRequest(method, path, clientIP string, statusCode int, duration int64) {
	l.Info("API request",
		zap.String("method", method),
		zap.String("path", path),
		zap.String("client_ip", clientIP),
		zap.Int("status_code", statusCode),
		zap.Int64("duration_ms", duration),
	)
}

// LogK8sOperation logs a Kubernetes operation
func (l *Logger) LogK8sOperation(operation, resource, namespace, name string, err error) {
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.String("resource", resource),
		zap.String("namespace", namespace),
		zap.String("name", name),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		l.Error("Kubernetes operation failed", fields...)
	} else {
		l.Debug("Kubernetes operation completed", fields...)
	}
}

// LogTrueNASOperation logs a TrueNAS API operation
func (l *Logger) LogTrueNASOperation(operation, endpoint string, statusCode int, err error) {
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.String("endpoint", endpoint),
		zap.Int("status_code", statusCode),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		l.Error("TrueNAS operation failed", fields...)
	} else {
		l.Debug("TrueNAS operation completed", fields...)
	}
}

// LogOrphanDetection logs orphan detection results
func (l *Logger) LogOrphanDetection(resourceType string, totalResources, orphanedCount int, scanDuration int64) {
	l.Info("Orphan detection completed",
		zap.String("resource_type", resourceType),
		zap.Int("total_resources", totalResources),
		zap.Int("orphaned_count", orphanedCount),
		zap.Int64("scan_duration_ms", scanDuration),
	)
}

// LogSecurityEvent logs security-related events
func (l *Logger) LogSecurityEvent(event, user, resource string, allowed bool) {
	fields := []zap.Field{
		zap.String("event", event),
		zap.String("user", user),
		zap.String("resource", resource),
		zap.Bool("allowed", allowed),
	}

	if allowed {
		l.Info("Security event", fields...)
	} else {
		l.Warn("Security event - access denied", fields...)
	}
}