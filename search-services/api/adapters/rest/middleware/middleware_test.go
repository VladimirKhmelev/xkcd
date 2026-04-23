package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"yadro.com/course/api/adapters/rest/middleware"
)

// --- helpers ---

func okHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
}

func recorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

// --- Auth ---

type mockVerifier struct {
	err error
}

func (m mockVerifier) Verify(token string) error { return m.err }

/* Запрос без заголовка Authorization возвращает 401 */
func TestAuth_NoHeader(t *testing.T) {
	h := middleware.Auth(okHandler(), mockVerifier{})
	w, r := recorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	h(w, r)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

/* Заголовок с неверной схемой (Bearer вместо Token) возвращает 401 */
func TestAuth_BadHeaderFormat(t *testing.T) {
	h := middleware.Auth(okHandler(), mockVerifier{})
	w, r := recorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer sometoken")
	h(w, r)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

/* Невалидный токен (verifier возвращает ошибку) возвращает 401 */
func TestAuth_InvalidToken(t *testing.T) {
	h := middleware.Auth(okHandler(), mockVerifier{err: fmt.Errorf("bad token")})
	w, r := recorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Token badtoken")
	h(w, r)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

/* Валидный токен пропускает запрос к следующему handler'у с 200 */
func TestAuth_ValidToken(t *testing.T) {
	h := middleware.Auth(okHandler(), mockVerifier{})
	w, r := recorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Token validtoken")
	h(w, r)
	require.Equal(t, http.StatusOK, w.Code)
}

// --- Concurrency ---

/* До лимита все запросы проходят, лишний получает 503 */
func TestConcurrency_AllowsUpToLimit(t *testing.T) {
	const limit = 3
	ready := make(chan struct{})
	done := make(chan struct{})
	h := middleware.Concurrency(func(w http.ResponseWriter, r *http.Request) {
		ready <- struct{}{}
		<-done
	}, limit)

	var wg sync.WaitGroup
	codes := make([]int, limit)
	for i := range limit {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := recorder()
			h(w, httptest.NewRequest(http.MethodGet, "/", nil))
			codes[i] = w.Code
		}(i)
		<-ready
	}

	w := recorder()
	h(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	close(done)
	wg.Wait()
	for _, c := range codes {
		require.Equal(t, http.StatusOK, c)
	}
}

/* При занятом слоте новый запрос сразу получает 503 */
func TestConcurrency_RejectsOverLimit(t *testing.T) {
	block := make(chan struct{})
	h := middleware.Concurrency(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}, 1)

	ready := make(chan struct{}, 1)
	go func() {
		ready <- struct{}{}
		h(recorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}()
	<-ready

	w := recorder()
	h(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	close(block)
}

// --- WithMetrics ---

/* Метрики не меняют статус ответа handler-а */
func TestWithMetrics_PassesThrough(t *testing.T) {
	h := middleware.WithMetrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	w := recorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/ping", nil))
	require.Equal(t, http.StatusCreated, w.Code)
}

/* При отсутствии явного WriteHeader статус по умолчанию 200 */
func TestWithMetrics_DefaultStatus200(t *testing.T) {
	h := middleware.WithMetrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok")) //nolint
	}))
	w := recorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

// --- Rate ---

/* Одиночный запрос при ненулевом лимите проходит и получает 200 */
func TestRate_AllowsSingleRequest(t *testing.T) {
	h := middleware.Rate(okHandler(), 100)
	w := recorder()
	h(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

/* Rate пропускает запрос к следующему handler'у и тот может вернуть любой статус */
func TestRate_PassesStatusThrough(t *testing.T) {
	inner := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) }
	h := middleware.Rate(inner, 100)
	w := recorder()
	h(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusCreated, w.Code)
}
