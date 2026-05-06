package db

import (
	"context"
	"embed"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {
	conn, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}
	return &DB{log: log, conn: conn}, nil
}

func (db *DB) Migrate() error {
	db.log.Debug("running migration")
	files, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return err
	}
	driver, err := pgx.WithInstance(db.conn.DB, &pgx.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", files, "pgx", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		db.log.Error("migration failed", "error", err)
		return err
	}
	db.log.Debug("migration finished")
	return nil
}

func (db *DB) CreateUser(ctx context.Context, username, password string) error {
	_, err := db.conn.ExecContext(ctx,
		`INSERT INTO users (username, password) VALUES ($1, $2)`,
		username, password,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (db *DB) GetPassword(ctx context.Context, username string) (string, error) {
	var password string
	err := db.conn.QueryRowContext(ctx,
		`SELECT password FROM users WHERE username = $1`,
		username,
	).Scan(&password)
	if err != nil {
		return "", fmt.Errorf("get password: %w", err)
	}
	return password, nil
}
