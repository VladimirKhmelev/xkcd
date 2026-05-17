package cache

import (
	"context"

	"yadro.com/course/search/core"
)

type Noop struct{}

func (Noop) Get(_ context.Context, _ string) (core.CachedResult, bool, error) {
	return core.CachedResult{}, false, nil
}
func (Noop) Set(_ context.Context, _ string, _ core.CachedResult) error { return nil }
func (Noop) Flush(_ context.Context) error                               { return nil }
