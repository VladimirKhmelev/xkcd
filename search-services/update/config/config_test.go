package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"yadro.com/course/update/config"
)

// MustLoad читает конфиг из несуществующего файла и берёт значения из env
func TestMustLoad_FromEnv(t *testing.T) {
	require.NoError(t, os.Setenv("UPDATE_ADDRESS", ":9082"))
	require.NoError(t, os.Setenv("WORDS_ADDRESS", "words:8080"))
	require.NoError(t, os.Setenv("XKCD_URL", "https://xkcd.com"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("UPDATE_ADDRESS"))
		require.NoError(t, os.Unsetenv("WORDS_ADDRESS"))
		require.NoError(t, os.Unsetenv("XKCD_URL"))
	})

	cfg := config.MustLoad("nonexistent.yaml")
	require.Equal(t, ":9082", cfg.Address)
	require.Equal(t, "words:8080", cfg.WordsAddress)
	require.Equal(t, "https://xkcd.com", cfg.XKCD.URL)
}

// Значения по умолчанию применяются если env не задан
func TestMustLoad_Defaults(t *testing.T) {
	cfg := config.MustLoad("nonexistent.yaml")
	require.NotEmpty(t, cfg.Address)
	require.Greater(t, cfg.XKCD.Concurrency, 0)
}
