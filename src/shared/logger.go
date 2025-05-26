package shared

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var _logger *zap.Logger

const (
	defaultLogDir        = "/var/log/ngit-relay"
	defaultLogLevel      = "INFO"
	defaultLogMaxSizeMB  = 20
	defaultLogMaxBackups = 10
	defaultLogMaxAgeDays = 30
)

// Init initializes the global logger.
// It reads configuration from environment variables.
// 'serviceName' is used to tag log entries (e.g., "ngit-relay-khatru").
func Init(serviceName string, logToStdoutStderr bool, divertStandardLog bool) {
	logDir := getEnv("LOG_DIR", defaultLogDir)
	logLevelStr := strings.ToUpper(getEnv("LOG_LEVEL", defaultLogLevel))
	maxSizeMB := getEnvInt("LOG_MAX_SIZE_MB", defaultLogMaxSizeMB)
	maxBackups := getEnvInt("LOG_MAX_BACKUPS", defaultLogMaxBackups)
	maxAgeDays := getEnvInt("LOG_MAX_AGE_DAYS", defaultLogMaxAgeDays)

	logFilePath := filepath.Join(logDir, serviceName+".log")

	// Configure lumberjack for log rotation
	lj := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    maxSizeMB,  // megabytes
		MaxBackups: maxBackups, // number of old log files to retain
		MaxAge:     maxAgeDays, // days
		Compress:   true,       // compress old log files
	}

	// Configure Zap logger
	atomicLevel := zap.NewAtomicLevel()
	if err := atomicLevel.UnmarshalText([]byte(strings.ToLower(logLevelStr))); err != nil {
		atomicLevel.SetLevel(zap.InfoLevel) // Default to INFO on parse error
		log.Printf("Invalid LOG_LEVEL '%s', defaulting to INFO. Error: %v", logLevelStr, err)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Add common fields
	fields := zap.Fields(
		zap.String("service", serviceName),
		zap.Int("pid", os.Getpid()),
	)

	// Core for writing to file
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(lj),
		atomicLevel,
	)

	if logToStdoutStderr {
		// Core for writing to stdout (for `docker logs`)
		stdoutCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig), // Or NewConsoleEncoder for more human-readable stdout
			zapcore.Lock(os.Stdout),
			zapcore.WarnLevel,
		)
		stderrCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
				MessageKey:    "msg",
				LevelKey:      "level",
				TimeKey:       "time",
				CallerKey:     "caller",
				StacktraceKey: "stacktrace",
				EncodeTime:    zapcore.ISO8601TimeEncoder,
				EncodeLevel:   zapcore.CapitalLevelEncoder,
				EncodeCaller:  zapcore.ShortCallerEncoder,
			}),
			zapcore.AddSync(os.Stderr),
			zapcore.WarnLevel,
		)

		teeCore := zapcore.NewTee(fileCore, stdoutCore, stderrCore)
		_logger = zap.New(teeCore, zap.AddCaller(), zap.ErrorOutput(zapcore.AddSync(lj)), fields)
	} else {
		teeCore := zapcore.NewTee(fileCore)
		_logger = zap.New(teeCore, zap.AddCaller(), zap.ErrorOutput(zapcore.AddSync(lj)), fields)
	}

	if divertStandardLog {
		// Redirect standard Go `log` package to Zap
		zap.RedirectStdLog(_logger)
	}
}

// L returns the global zap logger instance.
// It panics if Init() has not been called.
func L() *zap.Logger {
	if _logger == nil {
		// This basic logger will ensure the panic message itself is visible.
		// It's a last resort if Init() was somehow skipped entirely.
		basicLogger, _ := zap.NewProduction()
		defer basicLogger.Sync()
		basicLogger.Panic("Logger accessed before Init() was called. Ensure shared.InitLogger() is called in main().")
	}
	return _logger
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
		log.Printf("Warning: Invalid integer value for env var %s: '%s'. Using default %d. \n", key, valueStr, fallback)
	}
	return fallback
}
