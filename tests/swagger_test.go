package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// Swagger UI доступен
func TestSwaggerUIAvailable(t *testing.T) {
	resp, err := client.Get(address + "/swagger/index.html")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// swagger.json корректный и содержит все нужные эндпоинты
func TestSwaggerSpecValid(t *testing.T) {
	resp, err := client.Get(address + "/swagger/doc.json")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var spec struct {
		Info  map[string]any    `json:"info"`
		Paths map[string]any    `json:"paths"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&spec))

	require.NotEmpty(t, spec.Info["title"], "swagger title must not be empty")

	required := []string{"/api/search", "/api/login", "/api/register", "/api/user/login", "/api/db/stats"}
	for _, path := range required {
		require.Contains(t, spec.Paths, path, "swagger spec missing path: %s", path)
	}
}
