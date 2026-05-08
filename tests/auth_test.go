package api_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func register(t *testing.T, name, password string) int {
	t.Helper()
	body := fmt.Sprintf(`{"name":%q,"password":%q}`, name, password)
	resp, err := client.Post(address+"/api/register", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

func userLogin(t *testing.T, name, password string) (string, int) {
	t.Helper()
	body := fmt.Sprintf(`{"name":%q,"password":%q}`, name, password)
	resp, err := client.Post(address+"/api/user/login", "application/json", bytes.NewBufferString(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	token, _ := io.ReadAll(resp.Body)
	return string(token), resp.StatusCode
}

// Успешная регистрация нового пользователя
func TestUserRegister(t *testing.T) {
	name := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	code := register(t, name, "password123")
	require.Equal(t, http.StatusCreated, code)
}

// Повторная регистрация с тем же именем возвращает 409
func TestUserRegisterDuplicate(t *testing.T) {
	name := fmt.Sprintf("dupuser_%d", time.Now().UnixNano())
	require.Equal(t, http.StatusCreated, register(t, name, "password123"))
	require.Equal(t, http.StatusConflict, register(t, name, "password123"))
}

// Регистрация с пустыми полями возвращает 400
func TestUserRegisterEmptyFields(t *testing.T) {
	require.Equal(t, http.StatusBadRequest, register(t, "", "password123"))
	require.Equal(t, http.StatusBadRequest, register(t, "someuser", ""))
}

// Зарегистрированный пользователь может войти и получить токен
func TestUserLoginSuccess(t *testing.T) {
	name := fmt.Sprintf("loginuser_%d", time.Now().UnixNano())
	require.Equal(t, http.StatusCreated, register(t, name, "mypassword"))

	token, code := userLogin(t, name, "mypassword")
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, token)
}

// Неверный пароль возвращает 401
func TestUserLoginWrongPassword(t *testing.T) {
	name := fmt.Sprintf("wrongpass_%d", time.Now().UnixNano())
	require.Equal(t, http.StatusCreated, register(t, name, "correctpassword"))

	_, code := userLogin(t, name, "wrongpassword")
	require.Equal(t, http.StatusUnauthorized, code)
}

// Логин несуществующего пользователя возвращает 401
func TestUserLoginUnknown(t *testing.T) {
	_, code := userLogin(t, "nonexistent_user_xyz", "password")
	require.Equal(t, http.StatusUnauthorized, code)
}

// Ттокен имеет формат JWT
func TestUserTokenIsJWT(t *testing.T) {
	name := fmt.Sprintf("jwtuser_%d", time.Now().UnixNano())
	require.Equal(t, http.StatusCreated, register(t, name, "password123"))

	token, code := userLogin(t, name, "password123")
	require.Equal(t, http.StatusOK, code)

	parts := bytes.Split([]byte(token), []byte("."))
	require.Len(t, parts, 3, "JWT must have 3 parts separated by dots")
}

// Ппоиск доступен без авторизации
func TestSearchPublic(t *testing.T) {
	resp, err := client.Get(address + "/api/search?phrase=linux")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
}
