package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
)

// New returns a Logger implementation (currently zerolog). Swap the adapter
// in this package to use another library without changing call sites.
func New(level string) logger.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	zl := zerolog.New(
		zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		},
	).Level(lvl).With().Timestamp().Caller().Logger()

	return NewZerologAdapter(zl)
}
