package db

import (
	"context"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"yadro.com/course/search/core"
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
	return &DB{log: log, conn: db}, nil
}

type dbComic struct {
	ID      int    `db:"id"`
	ImgURL  string `db:"img_url"`
	Matches int    `db:"matches"`
}


func (db *DB) Search(ctx context.Context, keywords []string, limit int) ([]core.Comics, error) {
	rows, err := db.conn.QueryxContext(ctx, `
		SELECT id, img_url,
			(SELECT count(*) FROM unnest(keywords) k WHERE k = ANY($1)) AS matches
		FROM comics
		WHERE keywords && $1
		ORDER BY matches DESC
		LIMIT $2
	`, keywords, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []core.Comics
	for rows.Next() {
		var row dbComic
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		result = append(result, core.Comics{ID: row.ID, URL: row.ImgURL})
	}
	if result == nil {
		result = []core.Comics{}
	}
	return result, rows.Err()
}

func (db *DB) AllComics(ctx context.Context) ([]core.IndexComic, error) {
	rows, err := db.conn.QueryContext(ctx, `SELECT id, img_url, keywords FROM comics`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []core.IndexComic
	for rows.Next() {
		var ic core.IndexComic
		var keywords pq.StringArray
		if err := rows.Scan(&ic.ID, &ic.URL, &keywords); err != nil {
			return nil, err
		}
		ic.Keywords = []string(keywords)
		result = append(result, ic)
	}
	return result, rows.Err()
}
