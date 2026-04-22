package aaa_test

import (
	"crypto/rand"
	"crypto/rsa"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"yadro.com/course/api/adapters/aaa"
)

// Создаёт экземпляр AAA с тестовыми учётными данными через переменные окружения
func makeAAA(t *testing.T, ttl time.Duration) aaa.AAA {
	t.Helper()
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASSWORD", "password")
	a, err := aaa.New(ttl, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.NoError(t, err)
	return a
}

// Проверяет, что создание AAA завершается ошибкой, если переменная окружения ADMIN_USER не задана
func TestNew_MissingUser(t *testing.T) {
	os.Unsetenv("ADMIN_USER")
	os.Unsetenv("ADMIN_PASSWORD")
	_, err := aaa.New(time.Minute, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.Error(t, err)
}

// Проверяет, что создание AAA завершается ошибкой, если переменная окружения ADMIN_PASSWORD не задана
func TestNew_MissingPassword(t *testing.T) {
	t.Setenv("ADMIN_USER", "admin")
	os.Unsetenv("ADMIN_PASSWORD")
	_, err := aaa.New(time.Minute, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.Error(t, err)
}

// Проверяет успешную аутентификацию с правильными учётными данными и ожидается, что в ответ придёт непустой токен
func TestLogin_Success(t *testing.T) {
	a := makeAAA(t, time.Minute)
	token, err := a.Login("admin", "password")
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

// Проверяет, что вход с неверным паролем возвращает ошибку
func TestLogin_WrongPassword(t *testing.T) {
	a := makeAAA(t, time.Minute)
	_, err := a.Login("admin", "wrong")
	require.Error(t, err)
}

// Проверяет, что вход с несуществующим пользователем возвращает ошибку
func TestLogin_UnknownUser(t *testing.T) {
	a := makeAAA(t, time.Minute)
	_, err := a.Login("hacker", "password")
	require.Error(t, err)
}

// Проверяет, что свежевыданный токен успешно проходит верификацию
func TestVerify_ValidToken(t *testing.T) {
	a := makeAAA(t, time.Minute)
	token, err := a.Login("admin", "password")
	require.NoError(t, err)
	require.NoError(t, a.Verify(token))
}

// Проверяет, что истёкший токен не проходит верификацию
// Токен выдаётся с TTL в 1 мс, после чего ждём 10 мс для гарантированного истечения
func TestVerify_ExpiredToken(t *testing.T) {
	a := makeAAA(t, time.Millisecond)
	token, err := a.Login("admin", "password")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	require.Error(t, a.Verify(token))
}

// Проверяет, что произвольная строка(не JWT) не проходит
func TestVerify_InvalidToken(t *testing.T) {
	a := makeAAA(t, time.Minute)
	require.Error(t, a.Verify("not.a.token"))
}

// Проверяет, что токен, подписанный алгоритмом RS256 вместо HS256. Отклоняется ,даже если структура токена валидна
func TestVerify_WrongAlgorithm(t *testing.T) {
	a := makeAAA(t, time.Minute)
	// подписываем RS256 вместо HS256
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Subject:   "superuser",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	})
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	require.Error(t, a.Verify(signed))
}

// Проверяет, что токен с корректным алгоритмом, но чужим subject-полем отклоняется, даже если подпись математически верна
func TestVerify_WrongSubject(t *testing.T) {
	a := makeAAA(t, time.Minute)
	// подписываем правильным алгоритмом, но с неверным subject
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "notadmin",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	})
	signed, err := token.SignedString([]byte("something secret here"))
	require.NoError(t, err)
	require.Error(t, a.Verify(signed))
}
