package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
)

//go:embed migration/*.sql
var migrations embed.FS

// RunMigrations применяет все pending-миграции при старте приложения.
func RunMigrations(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	goose.SetLogger(gooseLogger{logger})
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, "migration"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	logger.Info("Migrations applied successfully")
	return nil
}

type gooseLogger struct{ l *slog.Logger }

func (g gooseLogger) Fatalf(format string, v ...interface{}) {
	g.l.Error(fmt.Sprintf(format, v...))
}
func (g gooseLogger) Printf(format string, v ...interface{}) {
	g.l.Info(fmt.Sprintf(format, v...))
}
