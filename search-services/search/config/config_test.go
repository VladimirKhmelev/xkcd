package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"yadro.com/course/search/config"
)

// MustLoad читает конфиг из несуществующего файла и берёт значения из env
func TestMustLoad_FromEnv(t *testing.T) {
	require.NoError(t, os.Setenv("SEARCH_ADDRESS", ":9083"))
	require.NoError(t, os.Setenv("WORDS_ADDRESS", "words:8080"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("SEARCH_ADDRESS"))
		require.NoError(t, os.Unsetenv("WORDS_ADDRESS"))
	})

	cfg := config.MustLoad("nonexistent.yaml")
	require.Equal(t, ":9083", cfg.Address)
	require.Equal(t, "words:8080", cfg.WordsAddress)
}

// Значения по умолчанию применяются если env не задан
func TestMustLoad_Defaults(t *testing.T) {
	cfg := config.MustLoad("nonexistent.yaml")
	require.NotEmpty(t, cfg.Address)
	require.NotEmpty(t, cfg.IndexTTL)
}
