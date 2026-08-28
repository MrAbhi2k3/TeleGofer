package client

import "log/slog"

// Logger is the interface for diagnostic output. Its method signatures
// match *slog.Logger, so standard library slog works without an adapter.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// NopLogger discards all output.
type NopLogger struct{}

func (NopLogger) Debug(string, ...any) {}
func (NopLogger) Info(string, ...any)  {}
func (NopLogger) Warn(string, ...any)  {}
func (NopLogger) Error(string, ...any) {}

// SlogLogger wraps an *slog.Logger as a client Logger.
func SlogLogger(l *slog.Logger) Logger { return slogWrap{l} }

type slogWrap struct{ l *slog.Logger }

func (w slogWrap) Debug(msg string, args ...any) { w.l.Debug(msg, args...) }
func (w slogWrap) Info(msg string, args ...any)  { w.l.Info(msg, args...) }
func (w slogWrap) Warn(msg string, args ...any)  { w.l.Warn(msg, args...) }
func (w slogWrap) Error(msg string, args ...any) { w.l.Error(msg, args...) }
