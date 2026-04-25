package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"yadro.com/course/api/config"
)

// MustLoad читает конфиг из yaml-файла и возвращает корректную структуру
func TestMustLoad_FromFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	require.NoError(t, err)
	_, err = f.WriteString("log_level: INFO\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cfg := config.MustLoad(f.Name())
	require.Equal(t, "INFO", cfg.LogLevel)
}

// Значения по умолчанию применяются если поля не заданы в yaml
func TestMustLoad_Defaults(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	require.NoError(t, err)
	_, err = f.WriteString("log_level: DEBUG\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cfg := config.MustLoad(f.Name())
	require.NotEmpty(t, cfg.HTTPConfig.Address)
	require.Greater(t, cfg.SearchConcurrency, 0)
}
