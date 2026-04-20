package db

import (
	"context"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/update/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Add(ctx context.Context, comics core.Comics) error {
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO comics (id, img_url, keywords)
		VALUES ($1, $2, $3)`,
		comics.ID, comics.URL, comics.Words,
	)
	return err
}

type dbStats struct {
	WordsTotal    int `db:"words_total"`
	WordsUnique   int `db:"words_unique"`
	ComicsFetched int `db:"comics_fetched"`
}

func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {
	var s dbStats
	err := db.conn.GetContext(ctx, &s, `
		SELECT
			(SELECT count(*) FROM comics) as comics_fetched,
			(SELECT COALESCE(SUM(array_length(keywords, 1)), 0) FROM comics) as words_total,
			(SELECT count(DISTINCT word) FROM comics, unnest(keywords) as word) as words_unique
	`)
	return core.DBStats{
		WordsTotal:    s.WordsTotal,
		WordsUnique:   s.WordsUnique,
		ComicsFetched: s.ComicsFetched,
	}, err
}

func (db *DB) IDs(ctx context.Context) ([]int, error) {
	var ids []int
	err := db.conn.SelectContext(ctx, &ids, "SELECT id FROM comics ORDER BY id")
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (db *DB) Drop(ctx context.Context) error {
	_, err := db.conn.ExecContext(ctx, "TRUNCATE TABLE comics")
	return err
}
