package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"yadro.com/course/api/adapters/rest"
	"yadro.com/course/api/core"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

// Имитирует сервис с методом Ping, возвращая заданную ошибку
type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(_ context.Context) error {
	return m.err
}

// Имитирует сервис аутентификации: Login возвращает заданный токен или ошибку
type mockAuth struct {
	token string
	err   error
}

func (m *mockAuth) Login(_, _ string) (string, error) { return m.token, m.err }

// Имитирует сервис обновления базы комиксов
type mockUpdater struct {
	updateErr error
	stats     core.UpdateStats
	statsErr  error
	status    core.UpdateStatus
	statusErr error
	dropErr   error
}

func (m *mockUpdater) Update(_ context.Context) error {
	return m.updateErr
}

func (m *mockUpdater) Stats(_ context.Context) (core.UpdateStats, error) {
	return m.stats, m.statsErr
}

func (m *mockUpdater) Status(_ context.Context) (core.UpdateStatus, error) {
	return m.status, m.statusErr
}

func (m *mockUpdater) Drop(_ context.Context) error { return m.dropErr }

// Имитирует поисковый сервис, возвращая заданный список комиксов и ошибку
type mockSearcher struct {
	result []core.Comics
	err    error
}

func (m *mockSearcher) Search(_ context.Context, _ string, _ int) ([]core.Comics, error) {
	return m.result, m.err
}
func (m *mockSearcher) SearchIndex(_ context.Context, _ string, _ int) ([]core.Comics, error) {
	return m.result, m.err
}

// Проверяет, что хендлер отвечает 200 OK и возвращает статус "ok" для каждого из зарегистрированных сервисов
func TestPingHandler_AllOK(t *testing.T) {
	h := rest.NewPingHandler(log, map[string]core.Pinger{
		"words":  &mockPinger{},
		"update": &mockPinger{},
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "ok", body["replies"]["words"])
	require.Equal(t, "ok", body["replies"]["update"])
}

// Проверяет, что недоступный сервис отображается как "error" в теле ответа, при этом HTTP-статус остаётся 200 OK
func TestPingHandler_ServiceError(t *testing.T) {
	h := rest.NewPingHandler(log, map[string]core.Pinger{
		"words": &mockPinger{err: fmt.Errorf("down")},
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Contains(t, body["replies"]["words"], "error")
}

// Проверяет успешный вход: хендлер возвращает 200 OK и тело с токеном
func TestLoginHandler_Success(t *testing.T) {
	h := rest.NewLoginHandler(log, &mockAuth{token: "mytoken"})
	body := bytes.NewBufferString(`{"name":"admin","password":"password"}`)
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodPost, "/api/login", body))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "mytoken", w.Body.String())
}

// Проверяет, что неверные учётные данные приводят к 401 Unauthorized
func TestLoginHandler_BadCredentials(t *testing.T) {
	h := rest.NewLoginHandler(log, &mockAuth{err: fmt.Errorf("invalid")})
	body := bytes.NewBufferString(`{"name":"admin","password":"wrong"}`)
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodPost, "/api/login", body))
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// Проверяет, что невалидный JSON в теле запроса приводит к 400 Bad Request
func TestLoginHandler_BadBody(t *testing.T) {
	h := rest.NewLoginHandler(log, &mockAuth{})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(`not json`)))
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// Проверяет успешный запуск обновления базы комиксов: ожидается 200 OK
func TestUpdateHandler_OK(t *testing.T) {
	h := rest.NewUpdateHandler(log, &mockUpdater{})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodPost, "/api/db/update", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

// Проверяет, что повторный запрос обновления когда оно уже выполняется, возвращает 202 Accepted
func TestUpdateHandler_AlreadyRunning(t *testing.T) {
	h := rest.NewUpdateHandler(log, &mockUpdater{updateErr: core.ErrAlreadyExists})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodPost, "/api/db/update", nil))
	require.Equal(t, http.StatusAccepted, w.Code)
}

// Проверяет, что внутренняя ошибка обновления возвращает 500 Internal Server Error
func TestUpdateHandler_Error(t *testing.T) {
	h := rest.NewUpdateHandler(log, &mockUpdater{updateErr: fmt.Errorf("db error")})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodPost, "/api/db/update", nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// Проверяет, что хендлер возвращает 200 OK и корректное значение comics_total
func TestStatsHandler_OK(t *testing.T) {
	h := rest.NewUpdateStatsHandler(log, &mockUpdater{stats: core.UpdateStats{ComicsTotal: 52}})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/api/db/stats", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]int
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, 52, body["comics_total"])
}

// Проверяет, что хендлер возвращает 200 OK и статус "idle", когда обновление не выполняется
func TestStatusHandler_OK(t *testing.T) {
	h := rest.NewUpdateStatusHandler(log, &mockUpdater{status: core.StatusUpdateIdle})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/api/db/status", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "idle", body["status"])
}

// Проверяет, что успешное удаление базы комиксов возвращает 200 OK
func TestDropHandler_OK(t *testing.T) {
	h := rest.NewDropHandler(log, &mockUpdater{})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodDelete, "/api/db", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

// Проверяет, что запрос без параметра phrase возвращает 400 Bad Request
func TestSearchHandler_NoPhrase(t *testing.T) {
	h := rest.NewSearchHandler(log, &mockSearcher{})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/api/search", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// Проверяет успешный поиск: ожидается 200 OK и корректное поле total в ответе
func TestSearchHandler_OK(t *testing.T) {
	h := rest.NewSearchHandler(log, &mockSearcher{result: []core.Comics{{ID: 1, URL: "url1"}}})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/api/search?phrase=linux", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, float64(1), body["total"])
}

// Проверяет, что нечисловое значение параметра limit возвращает 400 Bad Request
func TestSearchHandler_BadLimit(t *testing.T) {
	h := rest.NewSearchHandler(log, &mockSearcher{})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/api/search?phrase=linux&limit=abc", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// Проверяет, что индексный поиск без параметра phrase возвращает 400 Bad Request
func TestISearchHandler_NoPhrase(t *testing.T) {
	h := rest.NewSearchIndexHandler(log, &mockSearcher{})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/api/isearch", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// Проверяет успешный индексный поиск: ожидается 200 OK
func TestISearchHandler_OK(t *testing.T) {
	h := rest.NewSearchIndexHandler(log, &mockSearcher{result: []core.Comics{{ID: 2, URL: "url2"}}})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/api/isearch?phrase=linux", nil))
	require.Equal(t, http.StatusOK, w.Code)
}
