package database

import (
	"database/sql"
	"log/slog"
)

// PostgresRepository - универсальная структура для доступа к базе данных для работы со сценриями, резльтатами и тестами.
type PostgresRepository struct {
	logger *slog.Logger
	db     *sql.DB
}

// NewPostgresRepository - функция для создания структуры доступа до repository.
func NewPostgresRepository(logger *slog.Logger, db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		logger: logger,
		db:     db}
}
