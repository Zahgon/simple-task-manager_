package logger

import (
	"time"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
	"github.com/rs/zerolog"
)

// ZerologAdapter wraps zerolog.Logger so it implements the logger port.
// Swap this for another adapter (e.g. ZapAdapter) to change the logging library.
type ZerologAdapter struct {
	l zerolog.Logger
}

// Ensure ZerologAdapter implements the port at compile time.
var _ logger.Logger = (*ZerologAdapter)(nil)

// NewZerologAdapter returns a Logger that uses the given zerolog.Logger.
func NewZerologAdapter(l zerolog.Logger) *ZerologAdapter {
	return &ZerologAdapter{l: l}
}

func (z *ZerologAdapter) Info(msg string, keysAndValues ...any) {
	z.l.Info().Fields(fieldsFromKV(keysAndValues...)).Msg(msg)
}

func (z *ZerologAdapter) Error(msg string, keysAndValues ...any) {
	z.l.Error().Fields(fieldsFromKV(keysAndValues...)).Msg(msg)
}

func (z *ZerologAdapter) Warn(msg string, keysAndValues ...any) {
	z.l.Warn().Fields(fieldsFromKV(keysAndValues...)).Msg(msg)
}

func (z *ZerologAdapter) Debug(msg string, keysAndValues ...any) {
	z.l.Debug().Fields(fieldsFromKV(keysAndValues...)).Msg(msg)
}

func (z *ZerologAdapter) Fatal(msg string, keysAndValues ...any) {
	z.l.Fatal().Fields(fieldsFromKV(keysAndValues...)).Msg(msg)
}

func (z *ZerologAdapter) With(keysAndValues ...any) logger.Logger {
	return &ZerologAdapter{l: z.l.With().Fields(fieldsFromKV(keysAndValues...)).Logger()}
}

// fieldsFromKV builds a map suitable for zerolog's Fields from alternating key-value pairs.
// Supports string, int, int64, error, time.Duration, and any (Interface).
func fieldsFromKV(keysAndValues ...any) map[string]any {
	if len(keysAndValues)%2 != 0 {
		return nil
	}
	m := make(map[string]any, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		k, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		v := keysAndValues[i+1]
		if err, ok := v.(error); ok {
			m[k] = err
			continue
		}
		if d, ok := v.(time.Duration); ok {
			m[k] = d
			continue
		}
		m[k] = v
	}
	return m
}
