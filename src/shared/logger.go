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
	logFileName          = "relay.log"
)

// Init initializes the global logger.
// It reads configuration from environment variables.
// 'serviceName' is used to tag log entries (e.g., "ngit-relay-khatru").
func Init(serviceName string) {
	logDir := getEnv("LOG_DIR", defaultLogDir)
	logLevelStr := strings.ToUpper(getEnv("LOG_LEVEL", defaultLogLevel))
	maxSizeMB := getEnvInt("LOG_MAX_SIZE_MB", defaultLogMaxSizeMB)
	maxBackups := getEnvInt("LOG_MAX_BACKUPS", defaultLogMaxBackups)
	maxAgeDays := getEnvInt("LOG_MAX_AGE_DAYS", defaultLogMaxAgeDays)

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Fallback to home directory if default/provided is not writable
		if logDir == defaultLogDir || !isWritable(logDir) {
			homeDir, homeErr := os.UserHomeDir()
			if homeErr == nil {
				fallbackLogDir := filepath.Join(homeDir, ".ngit-relay", "logs")
				if err := os.MkdirAll(fallbackLogDir, 0700); err == nil {
					logDir = fallbackLogDir
				} else {
					// If home also fails, log to stdout only
					initStdOutLogger(serviceName, logLevelStr, "Failed to create primary log dir: "+err.Error()+", and fallback dir: "+err.Error())
					return
				}
			} else {
				// If home dir lookup fails, log to stdout only
				initStdOutLogger(serviceName, logLevelStr, "Failed to create primary log dir: "+err.Error()+", and failed to get home dir: "+homeErr.Error())
				return
			}
		} else {
			// If a custom logDir was provided and it fails, log to stdout only
			initStdOutLogger(serviceName, logLevelStr, "Failed to create custom log directory '"+logDir+"': "+err.Error())
			return
		}
	}

	logFilePath := filepath.Join(logDir, logFileName)

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

	// Core for writing to file
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(lj),
		atomicLevel,
	)

	// Core for writing to stdout (for `docker logs`)
	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig), // Or NewConsoleEncoder for more human-readable stdout
		zapcore.Lock(os.Stdout),
		atomicLevel,
	)

	// Tee core to write to both file and stdout
	teeCore := zapcore.NewTee(fileCore, stdoutCore)

	// Add common fields
	fields := zap.Fields(
		zap.String("service", serviceName),
		zap.Int("pid", os.Getpid()),
	)

	_logger = zap.New(teeCore, zap.AddCaller(), zap.ErrorOutput(zapcore.AddSync(lj)), fields)

	// Redirect standard Go `log` package to Zap
	zap.RedirectStdLog(_logger)

	_logger.Info("Logger initialized",
		zap.String("logDir", logDir),
		zap.String("logLevel", logLevelStr),
		zap.Int("maxSizeMB", maxSizeMB),
		zap.Int("maxBackups", maxBackups),
		zap.Int("maxAgeDays", maxAgeDays),
	)
}

// initStdoutLogger is a fallback for when file logging cannot be established.
func initStdOutLogger(serviceName, logLevelStr, reason string) {
	atomicLevel := zap.NewAtomicLevel()
	if err := atomicLevel.UnmarshalText([]byte(strings.ToLower(logLevelStr))); err != nil {
		atomicLevel.SetLevel(zap.InfoLevel)
	}
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.Lock(os.Stdout),
		atomicLevel,
	)
	fields := zap.Fields(
		zap.String("service", serviceName),
		zap.Int("pid", os.Getpid()),
	)
	_logger = zap.New(stdoutCore, zap.AddCaller(), fields)
	zap.RedirectStdLog(_logger)
	_logger.Warn("File logging failed, falling back to stdout only.", zap.String("reason", reason))
	_logger.Info("Logger initialized (stdout only)", zap.String("logLevel", logLevelStr))
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

// isWritable checks if the given path is writable.
func isWritable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		// If path doesn't exist, try to create a test file.
		// This handles cases where the directory exists but we're checking a potential file path.
		// For a directory, os.MkdirAll above would have failed if not writable.
		testFile := filepath.Join(path, ".writetest")
		if f, err := os.Create(testFile); err == nil {
			f.Close()
			os.Remove(testFile)
			return true
		}
		return false
	}
	return info.Mode().Perm()&0200 != 0 // Check for write permission for owner or group or others
}
