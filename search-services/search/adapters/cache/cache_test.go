package cache_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
	"yadro.com/course/search/adapters/cache"
	"yadro.com/course/search/core"
)

func newTestRedis(t *testing.T) *cache.Redis {
	t.Helper()
	mr := miniredis.RunT(t)
	r, err := cache.New(mr.Addr())

	require.NoError(t, err)
	return r
}

// Cache miss возвращает false без ошибки
func TestRedis_GetMiss(t *testing.T) {
	r := newTestRedis(t)
	comics, ok, err := r.Get(context.Background(), "missing")

	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, comics)
}

// Set и Get возвращают те же данные
func TestRedis_SetAndGet(t *testing.T) {
	r := newTestRedis(t)
	expected := []core.Comics{{ID: 1, URL: "url1"}, {ID: 2, URL: "url2"}}

	require.NoError(t, r.Set(context.Background(), "key1", expected))

	got, ok, err := r.Get(context.Background(), "key1")

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, expected, got)
}

// После Flush ключи недоступны
func TestRedis_Flush(t *testing.T) {
	r := newTestRedis(t)
	comics := []core.Comics{{ID: 1, URL: "url1"}}

	require.NoError(t, r.Set(context.Background(), "key1", comics))
	require.NoError(t, r.Flush(context.Background()))

	_, ok, err := r.Get(context.Background(), "key1")
	require.NoError(t, err)
	require.False(t, ok)
}

// Set с пустым списком корректно сохраняется и возвращается
func TestRedis_SetEmpty(t *testing.T) {
	r := newTestRedis(t)
	require.NoError(t, r.Set(context.Background(), "empty", []core.Comics{}))

	got, ok, err := r.Get(context.Background(), "empty")

	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, got)
}

// Noop всегда возвращает cache miss и не падает
func TestNoop(t *testing.T) {
	n := cache.Noop{}
	ctx := context.Background()

	got, ok, err := n.Get(ctx, "key")

	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, got)

	require.NoError(t, n.Set(ctx, "key", []core.Comics{{ID: 1}}))
	require.NoError(t, n.Flush(ctx))
}
