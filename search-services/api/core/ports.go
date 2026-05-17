package core

import "context"

type Normalizer interface {
	Norm(context.Context, string) ([]string, error)
}

type Pinger interface {
	Ping(context.Context) error
}

type UserStorage interface {
	CreateUser(ctx context.Context, username, password string) error
	CheckPassword(ctx context.Context, username, password string) error
}

type Updater interface {
	Update(context.Context) error
	Stats(context.Context) (UpdateStats, error)
	Status(context.Context) (UpdateStatus, error)
	Drop(context.Context) error
}

type Searcher interface {
	Search(ctx context.Context, phrase string, limit, page int) ([]Comics, int, error)
	SearchIndex(ctx context.Context, phrase string, limit int) ([]Comics, error)
}
