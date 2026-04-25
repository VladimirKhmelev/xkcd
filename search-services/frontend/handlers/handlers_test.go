package handlers

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Минимальные шаблоны чтобы render() не падал
const testTemplates = `
{{define "search.html"}}search:{{.Phrase}}:{{.Limit}}:{{.Error}}:{{.Total}}{{end}}
{{define "admin.html"}}admin:{{.LoggedIn}}:{{.Error}}:{{.Message}}:{{.Status}}{{end}}
`

func newHandler(t *testing.T, apiURL string) *Handler {
	t.Helper()
	tmpl := template.Must(template.New("").Parse(testTemplates))
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return New(log, apiURL, tmpl)
}

// Без фразы возвращает пустую страницу с лимитом по умолчанию 10
func TestSearchPage_NoPhrase(t *testing.T) {
	h := newHandler(t, "")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.SearchPage(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "search::10::")
}

// С фразой делает запрос к API и отображает результаты
func TestSearchPage_WithPhrase(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/search", r.URL.Path)
		assert.Equal(t, "linux", r.URL.Query().Get("phrase"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"comics": []map[string]any{{"id": 1, "url": "http://img"}},
			"total":  1,
		}))
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	r := httptest.NewRequest(http.MethodGet, "/?phrase=linux", nil)
	w := httptest.NewRecorder()

	h.SearchPage(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "linux")
	assert.Contains(t, w.Body.String(), ":1")
}

// Пользовательский лимит передаётся в API как есть
func TestSearchPage_CustomLimit(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"comics": []any{}, "total": 0}))
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	r := httptest.NewRequest(http.MethodGet, "/?phrase=test&limit=50", nil)
	w := httptest.NewRecorder()

	h.SearchPage(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Невалидный лимит (не число) заменяется на 10
func TestSearchPage_InvalidLimit_FallsBackTo10(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"comics": []any{}, "total": 0}))
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	r := httptest.NewRequest(http.MethodGet, "/?phrase=test&limit=abc", nil)
	w := httptest.NewRecorder()

	h.SearchPage(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Ошибка API отображается в шаблоне, страница всё равно 200
func TestSearchPage_APIError(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	r := httptest.NewRequest(http.MethodGet, "/?phrase=linux", nil)
	w := httptest.NewRecorder()

	h.SearchPage(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "search error")
}

// AdminPage (панель администратора)

// Без куки токена отдаёт страницу с LoggedIn=false
func TestAdminPage_NotLoggedIn(t *testing.T) {
	h := newHandler(t, "")
	r := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()

	h.AdminPage(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "admin:false")
}

// С валидным токеном запрашивает stats и status у API
func TestAdminPage_LoggedIn(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/db/stats":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]int{"comics_fetched": 100}))
		case "/api/db/status":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"status": "idle"}))
		}
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	r := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: "mytoken"})
	w := httptest.NewRecorder()

	h.AdminPage(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "admin:true")
	assert.Contains(t, w.Body.String(), "idle")
}

// flash-сообщения из query-параметров msg и err передаются в шаблон
func TestAdminPage_FlashMessages(t *testing.T) {
	h := newHandler(t, "")
	r := httptest.NewRequest(http.MethodGet, "/admin?msg=hello&err=oops", nil)
	w := httptest.NewRecorder()

	h.AdminPage(w, r)

	body := w.Body.String()
	assert.Contains(t, body, "hello")
	assert.Contains(t, body, "oops")
}

// GET-запрос на логин перенаправляет на /admin без обработки
func TestLogin_WrongMethod(t *testing.T) {
	h := newHandler(t, "")
	r := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	w := httptest.NewRecorder()

	h.Login(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/admin", w.Header().Get("Location"))
}

// Успешный логин устанавливает куку с токеном и редиректит на /admin
func TestLogin_Success(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("tok123"))
		require.NoError(t, err)
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	form := url.Values{"name": {"admin"}, "password": {"pass"}}
	r := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Login(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	require.NotEmpty(t, w.Header().Get("Set-Cookie"))
	assert.Contains(t, w.Header().Get("Set-Cookie"), "token=tok123")
	assert.Contains(t, w.Header().Get("Location"), "/admin")
}

// Неверные credentials редиректят на /admin с err в query
func TestLogin_Failure(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	form := url.Values{"name": {"bad"}, "password": {"wrong"}}
	r := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Login(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "err=")
}

// Сбрасывает куку и редиректит на /admin
func TestLogout(t *testing.T) {
	h := newHandler(t, "")
	r := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: "tok"})
	w := httptest.NewRecorder()

	h.Logout(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/admin", w.Header().Get("Location"))
	assert.Contains(t, w.Header().Get("Set-Cookie"), "token=;")
}

// GET-запрос на update перенаправляет без обработки
func TestUpdate_WrongMethod(t *testing.T) {
	h := newHandler(t, "")
	r := httptest.NewRequest(http.MethodGet, "/admin/update", nil)
	w := httptest.NewRecorder()

	h.Update(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
}

// Запрос без токена редиректит с ошибкой
func TestUpdate_NoToken(t *testing.T) {
	h := newHandler(t, "")
	r := httptest.NewRequest(http.MethodPost, "/admin/update", nil)
	w := httptest.NewRecorder()

	h.Update(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "err=")
}

// Успешный запрос передаёт токен в заголовке и редиректит с msg
func TestUpdate_Success(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Token mytoken", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	r := httptest.NewRequest(http.MethodPost, "/admin/update", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: "mytoken"})
	w := httptest.NewRecorder()

	h.Update(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "msg=")
}

// API вернул 202 — обновление уже идёт, сообщаем об этом
func TestUpdate_AlreadyRunning(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	r := httptest.NewRequest(http.MethodPost, "/admin/update", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: "tok"})
	w := httptest.NewRecorder()

	h.Update(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "already+running")
}

// Ошибка API редиректит с err в query
func TestUpdate_APIError(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	r := httptest.NewRequest(http.MethodPost, "/admin/update", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: "tok"})
	w := httptest.NewRecorder()

	h.Update(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "err=")
}

// GET-запрос на drop перенаправляет без обработки
func TestDrop_WrongMethod(t *testing.T) {
	h := newHandler(t, "")
	r := httptest.NewRequest(http.MethodGet, "/admin/drop", nil)
	w := httptest.NewRecorder()

	h.Drop(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
}

// Запрос без токена редиректит с ошибкой
func TestDrop_NoToken(t *testing.T) {
	h := newHandler(t, "")
	r := httptest.NewRequest(http.MethodPost, "/admin/drop", nil)
	w := httptest.NewRecorder()

	h.Drop(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "err=")
}

// Успешный drop отправляет DELETE к API и редиректит с msg
func TestDrop_Success(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	r := httptest.NewRequest(http.MethodPost, "/admin/drop", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: "tok"})
	w := httptest.NewRecorder()

	h.Drop(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "msg=")
}

// Ошибка API редиректит с err в query
func TestDrop_APIError(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer api.Close()

	h := newHandler(t, api.URL)
	r := httptest.NewRequest(http.MethodPost, "/admin/drop", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: "tok"})
	w := httptest.NewRecorder()

	h.Drop(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "err=")
}
