package logger

import (
	"log/slog"
	"os"
)

// Init sets up the global structured JSON logger.
// All logs across the entire service will be JSON formatted.
func Init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(handler))
	slog.Info("logger initialized", "format", "json", "level", "debug")
}
