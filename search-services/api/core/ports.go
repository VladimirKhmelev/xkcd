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
	GetPassword(ctx context.Context, username string) (string, error)
}

type Updater interface {
	Update(context.Context) error
	Stats(context.Context) (UpdateStats, error)
	Status(context.Context) (UpdateStatus, error)
	Drop(context.Context) error
}

type Searcher interface {
	Search(context.Context, string, int) ([]Comics, error)
	SearchIndex(context.Context, string, int) ([]Comics, error)
}
