package core_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"yadro.com/course/search/core"
)

var testLog = slog.New(slog.NewTextHandler(os.Stderr, nil))

type mockDB struct {
	searchResult []core.Comics
	allComics    []core.IndexComic
	searchErr    error
	allErr       error
}

func (m *mockDB) Search(_ context.Context, _ []string, _ int) ([]core.Comics, error) {
	return m.searchResult, m.searchErr
}
func (m *mockDB) AllComics(_ context.Context) ([]core.IndexComic, error) {
	return m.allComics, m.allErr
}

type mockWords struct {
	result []string
	err    error
}

func (m *mockWords) Norm(_ context.Context, _ string) ([]string, error) {
	return m.result, m.err
}

// Gустые ключевые слова после нормализации — возвращаем пустой результат без запроса к БД
func TestSearch_EmptyKeywords(t *testing.T) {
	svc := core.NewService(testLog, &mockDB{}, &mockWords{result: nil})
	result, err := svc.Search(context.Background(), "the an", 10)
	require.NoError(t, err)
	require.Empty(t, result)
}

// Нрмальный поиск возвращает то что пришло из БД
func TestSearch_ReturnsDBResults(t *testing.T) {
	expected := []core.Comics{{ID: 1, URL: "url1"}, {ID: 2, URL: "url2"}}
	svc := core.NewService(testLog, &mockDB{searchResult: expected}, &mockWords{result: []string{"linux"}})
	result, err := svc.Search(context.Background(), "linux", 10)
	require.NoError(t, err)
	require.Equal(t, expected, result)
}

// Ошибка нормализации пробрасывается наружу
func TestSearch_WordsError(t *testing.T) {
	svc := core.NewService(testLog, &mockDB{}, &mockWords{err: errors.New("words down")})
	_, err := svc.Search(context.Background(), "linux", 10)
	require.Error(t, err)
}

// Ошибка БД пробрасывается наружу
func TestSearch_DBError(t *testing.T) {
	svc := core.NewService(testLog, &mockDB{searchErr: errors.New("db down")}, &mockWords{result: []string{"linux"}})
	_, err := svc.Search(context.Background(), "linux", 10)
	require.Error(t, err)
}

// После построения индекса isearch находит комиксы по ключевому слову
func TestBuildIndex_AndISearch(t *testing.T) {
	comics := []core.IndexComic{
		{ID: 1, URL: "url1", Keywords: []string{"linux", "kernel"}},
		{ID: 2, URL: "url2", Keywords: []string{"linux", "windows"}},
		{ID: 3, URL: "url3", Keywords: []string{"macos"}},
	}
	svc := core.NewService(testLog, &mockDB{allComics: comics}, &mockWords{result: []string{"linux"}})
	require.NoError(t, svc.BuildIndex(context.Background()))

	result, err := svc.ISearch(context.Background(), "linux", 10)
	require.NoError(t, err)
	require.Len(t, result, 2)
}

// limit обрезает результаты isearch
func TestISearch_LimitApplied(t *testing.T) {
	comics := []core.IndexComic{
		{ID: 1, URL: "u1", Keywords: []string{"linux"}},
		{ID: 2, URL: "u2", Keywords: []string{"linux"}},
		{ID: 3, URL: "u3", Keywords: []string{"linux"}},
	}
	svc := core.NewService(testLog, &mockDB{allComics: comics}, &mockWords{result: []string{"linux"}})
	require.NoError(t, svc.BuildIndex(context.Background()))

	result, err := svc.ISearch(context.Background(), "linux", 2)
	require.NoError(t, err)
	require.Len(t, result, 2)
}

// Комикс с большим количеством совпадений по ключевым словам идёт первым
func TestISearch_SortedByScore(t *testing.T) {
	comics := []core.IndexComic{
		{ID: 1, URL: "u1", Keywords: []string{"linux"}},
		{ID: 2, URL: "u2", Keywords: []string{"linux", "kernel"}},
	}
	svc := core.NewService(testLog, &mockDB{allComics: comics}, &mockWords{result: []string{"linux", "kernel"}})
	require.NoError(t, svc.BuildIndex(context.Background()))

	result, err := svc.ISearch(context.Background(), "linux kernel", 10)
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, 2, result[0].ID)
}

// После сброса индекса isearch не находит ничего
func TestResetIndex_ClearsResults(t *testing.T) {
	comics := []core.IndexComic{
		{ID: 1, URL: "u1", Keywords: []string{"linux"}},
	}
	svc := core.NewService(testLog, &mockDB{allComics: comics}, &mockWords{result: []string{"linux"}})
	require.NoError(t, svc.BuildIndex(context.Background()))

	svc.ResetIndex()

	result, err := svc.ISearch(context.Background(), "linux", 10)
	require.NoError(t, err)
	require.Empty(t, result)
}

// Пустые ключевые слова в isearch — возвращаем пустой результат
func TestISearch_EmptyKeywords(t *testing.T) {
	svc := core.NewService(testLog, &mockDB{}, &mockWords{result: nil})
	result, err := svc.ISearch(context.Background(), "", 10)
	require.NoError(t, err)
	require.Empty(t, result)
}

// Ошибка нормализации в isearch пробрасывается наружу
func TestISearch_WordsError(t *testing.T) {
	svc := core.NewService(testLog, &mockDB{}, &mockWords{err: errors.New("words down")})
	_, err := svc.ISearch(context.Background(), "linux", 10)
	require.Error(t, err)
}

// Ошибка при построении индекса пробрасывается наружу
func TestBuildIndex_DBError(t *testing.T) {
	svc := core.NewService(testLog, &mockDB{allErr: errors.New("db down")}, &mockWords{})
	err := svc.BuildIndex(context.Background())
	require.Error(t, err)
}
