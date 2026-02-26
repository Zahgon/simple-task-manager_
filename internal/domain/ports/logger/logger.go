package logger

// Logger is the application logging port. Depend on this interface so the
// concrete implementation (zerolog, zap, stdlog, etc.) can be swapped.
type Logger interface {
	Info(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Debug(msg string, keysAndValues ...any)
	Fatal(msg string, keysAndValues ...any)
	With(keysAndValues ...any) Logger
}
