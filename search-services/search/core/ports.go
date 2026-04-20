package core

import "context"

type Searcher interface {
	Search(ctx context.Context, phrase string, limit int) ([]Comics, error)
	ISearch(ctx context.Context, phrase string, limit int) ([]Comics, error)
}

type IndexBuilder interface {
	BuildIndex(ctx context.Context) error
}

type IndexResetter interface {
	ResetIndex()
}

type DB interface {
	Search(ctx context.Context, keywords []string, limit int) ([]Comics, error)
	AllComics(ctx context.Context) ([]IndexComic, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}
