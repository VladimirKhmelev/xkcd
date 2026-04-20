package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPingAllServices verifies that all downstream services are reachable via the API.
// Fast, no side effects — safe to run on every push.
func TestPingAllServices(t *testing.T) {
	resp, err := client.Get(address + "/api/ping")
	require.NoError(t, err, "cannot reach API")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var reply PingResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reply))
	for _, svc := range []string{"words", "update", "search"} {
		require.Equalf(t, "ok", reply.Replies[svc], "service %q is not healthy", svc)
	}
}

// TestUpdateRequiresAuth checks that the update endpoint rejects requests without a token.
func TestUpdateRequiresAuth(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, address+"/api/db/update", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestStatsPublic checks that the stats endpoint is publicly accessible.
func TestStatsPublic(t *testing.T) {
	resp, err := client.Get(address + "/api/db/stats")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var s struct {
		WordsTotal    int `json:"words_total"`
		WordsUnique   int `json:"words_unique"`
		ComicsFetched int `json:"comics_fetched"`
		ComicsTotal   int `json:"comics_total"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&s))
}
