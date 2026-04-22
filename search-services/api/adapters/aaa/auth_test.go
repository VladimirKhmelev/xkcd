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

func makeAAA(t *testing.T, ttl time.Duration) aaa.AAA {
	t.Helper()
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASSWORD", "password")
	a, err := aaa.New(ttl, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.NoError(t, err)
	return a
}

func TestNew_MissingUser(t *testing.T) {
	os.Unsetenv("ADMIN_USER")
	os.Unsetenv("ADMIN_PASSWORD")
	_, err := aaa.New(time.Minute, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.Error(t, err)
}

func TestNew_MissingPassword(t *testing.T) {
	t.Setenv("ADMIN_USER", "admin")
	os.Unsetenv("ADMIN_PASSWORD")
	_, err := aaa.New(time.Minute, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.Error(t, err)
}

func TestLogin_Success(t *testing.T) {
	a := makeAAA(t, time.Minute)
	token, err := a.Login("admin", "password")
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestLogin_WrongPassword(t *testing.T) {
	a := makeAAA(t, time.Minute)
	_, err := a.Login("admin", "wrong")
	require.Error(t, err)
}

func TestLogin_UnknownUser(t *testing.T) {
	a := makeAAA(t, time.Minute)
	_, err := a.Login("hacker", "password")
	require.Error(t, err)
}

func TestVerify_ValidToken(t *testing.T) {
	a := makeAAA(t, time.Minute)
	token, err := a.Login("admin", "password")
	require.NoError(t, err)
	require.NoError(t, a.Verify(token))
}

func TestVerify_ExpiredToken(t *testing.T) {
	a := makeAAA(t, time.Millisecond)
	token, err := a.Login("admin", "password")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	require.Error(t, a.Verify(token))
}

func TestVerify_InvalidToken(t *testing.T) {
	a := makeAAA(t, time.Minute)
	require.Error(t, a.Verify("not.a.token"))
}

func TestVerify_WrongAlgorithm(t *testing.T) {
	a := makeAAA(t, time.Minute)
	// sign with RS256 instead of HS256
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

func TestVerify_WrongSubject(t *testing.T) {
	a := makeAAA(t, time.Minute)
	// sign with correct algorithm but wrong subject
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "notadmin",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	})
	signed, err := token.SignedString([]byte("something secret here"))
	require.NoError(t, err)
	require.Error(t, a.Verify(signed))
}
