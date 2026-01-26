package logger

import (
	"log/slog"
	"os"
)

// LoggerWrapper хранит logger и уровень логирования.
type LoggerWrapper struct {
	Logger *slog.Logger
	lvl    *slog.LevelVar
}

// func() Setup устанавливает уровень логирования
func Setup(initialLevel string) *LoggerWrapper {
	lvl := new(slog.LevelVar)
	setStartLevel(lvl, initialLevel)

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})

	l := slog.New(handler)
	slog.SetDefault(l)

	return &LoggerWrapper{
		Logger: l,
		lvl:    lvl,
	}
}

// func() SetLevel позволяет менять уровень логирования в real-time.
func (lw *LoggerWrapper) SetLevel(levelStr string) {
	var l slog.Level
	switch levelStr {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	lw.lvl.Set(l)
}

// func() setStartLevel для установки стартового уровня логирования.
func setStartLevel(lvl *slog.LevelVar, levelStr string) {
	var l slog.Level
	switch levelStr {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	lvl.Set(l)
}
