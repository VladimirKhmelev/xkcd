package xkcd_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"yadro.com/course/update/adapters/xkcd"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

// Создает тестовый сервер для мокирования API-запросов
func makeServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// Проверяет, что создание клиента с пустым URL возвращает ошибку
func TestNewClient_EmptyURL(t *testing.T) {
	_, err := xkcd.NewClient("", time.Second, log)
	require.Error(t, err)
}

// Проверяет, что клиент успешно создаётся с валидным URL
func TestNewClient_OK(t *testing.T) {
	c, err := xkcd.NewClient("http://example.com", time.Second, log)
	require.NoError(t, err)
	require.NotNil(t, c)
}

// Проверяет успешное получение информации о комиксе
func TestGet_OK(t *testing.T) {
	srv := makeServer(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"num":        52,
			"img":        "https://imgs.xkcd.com/comics/52.png",
			"title":      "Title",
			"alt":        "Alt text",
			"transcript": "Transcript",
		}))
	})
	defer srv.Close()

	c, _ := xkcd.NewClient(srv.URL, time.Second, log)
	info, err := c.Get(context.Background(), 52)
	require.NoError(t, err)
	require.Equal(t, 52, info.ID)
	require.Equal(t, "https://imgs.xkcd.com/comics/52.png", info.URL)
	require.Contains(t, info.Description, "Title")
}

// Проверяет, что ответ сервера 404 Not Found приводит к ошибке на стороне клиента
func TestGet_NotFound(t *testing.T) {
	srv := makeServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer srv.Close()

	c, _ := xkcd.NewClient(srv.URL, time.Second, log)
	_, err := c.Get(context.Background(), 404)
	require.Error(t, err)
}

// Проверяет, что ответ сервера 500 Internal Server Error приводит к ошибке клиента
func TestGet_ServerError(t *testing.T) {
	srv := makeServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	c, _ := xkcd.NewClient(srv.URL, time.Second, log)
	_, err := c.Get(context.Background(), 1)
	require.Error(t, err)
}

// Проверяет, что клиент корректно извлекает номер последнего комикса из ответа сервера
func TestLastID_OK(t *testing.T) {
	srv := makeServer(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"num": 3000}))
	})
	defer srv.Close()

	c, _ := xkcd.NewClient(srv.URL, time.Second, log)
	id, err := c.LastID(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3000, id)
}

// Проверяет, что ошибка сервера при запросе последнего ID возвращает ошибку клиента
func TestLastID_ServerError(t *testing.T) {
	srv := makeServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	c, _ := xkcd.NewClient(srv.URL, time.Second, log)
	_, err := c.LastID(context.Background())
	require.Error(t, err)
}
