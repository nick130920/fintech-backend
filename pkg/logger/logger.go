package logger

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var once sync.Once
var log zerolog.Logger

// InitLogger initializes the global logger with environment-specific settings.
// It configures a human-friendly console logger for development and a JSON logger for production.
func InitLogger(logLevel, ginMode string) {
	once.Do(func() {
		var level zerolog.Level
		switch strings.ToLower(logLevel) {
		case "debug":
			level = zerolog.DebugLevel
		case "info":
			level = zerolog.InfoLevel
		case "warn":
			level = zerolog.WarnLevel
		case "error":
			level = zerolog.ErrorLevel
		default:
			level = zerolog.InfoLevel
		}
		zerolog.SetGlobalLevel(level)

		// For local development, use a console writer for readability.
		// For production (e.g., Railway), use a JSON writer for structured logging.
		if ginMode == "debug" || ginMode == "" {
			output := zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC3339,
				NoColor:    false,
			}
			log = zerolog.New(output).With().Timestamp().Logger()
		} else {
			log = zerolog.New(os.Stdout).With().
				Timestamp().
				Logger()
		}
	})
}

// Get returns the configured global logger instance.
func Get() zerolog.Logger {
	return log
}
