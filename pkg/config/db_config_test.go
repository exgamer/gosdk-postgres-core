package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreconfig "github.com/exgamer/gosdk-core/pkg/config"
)

var postgresEnvKeys = []string{
	"POSTGRES_DB_HOST",
	"POSTGRES_DB_USER",
	"POSTGRES_DB_PASSWORD",
	"POSTGRES_DB_NAME",
	"POSTGRES_DB_PORT",
	"POSTGRES_DB_MAX_OPEN_CONNECTIONS",
	"POSTGRES_DB_MAX_IDLE_CONNECTIONS",
	"POSTGRES_DB_LOGGING",
	"POSTGRES_DB_LOG_LEVEL",
	"POSTGRES_DB_THRESHOLD",
}

func TestInitConfig_OSEnvWithoutEnvFile(t *testing.T) {
	clearPostgresEnv(t)
	loadEnvFile(t, "")
	t.Setenv("POSTGRES_DB_HOST", "db.local")
	t.Setenv("POSTGRES_DB_USER", "app")
	t.Setenv("POSTGRES_DB_PASSWORD", "secret")
	t.Setenv("POSTGRES_DB_NAME", "main")
	t.Setenv("POSTGRES_DB_PORT", "5433")
	t.Setenv("POSTGRES_DB_MAX_OPEN_CONNECTIONS", "20")
	t.Setenv("POSTGRES_DB_MAX_IDLE_CONNECTIONS", "5")
	t.Setenv("POSTGRES_DB_LOGGING", "true")
	t.Setenv("POSTGRES_DB_LOG_LEVEL", "info")
	t.Setenv("POSTGRES_DB_THRESHOLD", "0.5")

	cfg := &PostgresDbConfig{}
	if err := coreconfig.InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	want := PostgresDbConfig{
		Host:               "db.local",
		User:               "app",
		Password:           "secret",
		Db:                 "main",
		Port:               "5433",
		MaxOpenConnections: 20,
		MaxIdleConnections: 5,
		Logging:            true,
		DbLogLevel:         "info",
		Threshold:          0.5,
	}
	if *cfg != want {
		t.Errorf("got %+v, want %+v", *cfg, want)
	}
}

func TestInitConfig_OSEnvOverridesEnvFile(t *testing.T) {
	clearPostgresEnv(t)
	loadEnvFile(t, "POSTGRES_DB_HOST=file\nPOSTGRES_DB_NAME=file_db\nPOSTGRES_DB_MAX_OPEN_CONNECTIONS=10\nPOSTGRES_DB_LOGGING=true\n")
	t.Setenv("POSTGRES_DB_HOST", "env")
	t.Setenv("POSTGRES_DB_MAX_OPEN_CONNECTIONS", "30")
	t.Setenv("POSTGRES_DB_LOGGING", "false")

	cfg := &PostgresDbConfig{}
	if err := coreconfig.InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if cfg.Host != "env" {
		t.Errorf("Host = %q, want env", cfg.Host)
	}
	if cfg.MaxOpenConnections != 30 {
		t.Errorf("MaxOpenConnections = %d, want 30", cfg.MaxOpenConnections)
	}
	if cfg.Logging {
		t.Error("Logging = true, want false from env")
	}
	if cfg.Db != "file_db" {
		t.Errorf("Db = %q, want file_db from .env", cfg.Db)
	}
}

func TestInitConfig_EmptyOSEnvOverridesEnvFile(t *testing.T) {
	clearPostgresEnv(t)
	loadEnvFile(t, "POSTGRES_DB_PASSWORD=file_pass\nPOSTGRES_DB_MAX_OPEN_CONNECTIONS=10\n")
	t.Setenv("POSTGRES_DB_PASSWORD", "")
	t.Setenv("POSTGRES_DB_MAX_OPEN_CONNECTIONS", "")

	cfg := &PostgresDbConfig{}
	if err := coreconfig.InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if cfg.Password != "" || cfg.MaxOpenConnections != 0 {
		t.Errorf("got Password=%q MaxOpenConnections=%d, want empty and 0", cfg.Password, cfg.MaxOpenConnections)
	}
}

func TestInitConfig_InvalidOSEnvValue(t *testing.T) {
	cases := []struct {
		name string
		key  string
		val  string
	}{
		{name: "int", key: "POSTGRES_DB_MAX_OPEN_CONNECTIONS", val: "abc"},
		{name: "bool", key: "POSTGRES_DB_LOGGING", val: "yes"},
		{name: "float", key: "POSTGRES_DB_THRESHOLD", val: "1,5"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearPostgresEnv(t)
			loadEnvFile(t, "")
			t.Setenv(tc.key, tc.val)

			err := coreconfig.InitConfig(&PostgresDbConfig{})
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.key) {
				t.Errorf("error %q does not name %s", err, tc.key)
			}
		})
	}
}

func TestInitConfig_EnvFileOnly(t *testing.T) {
	clearPostgresEnv(t)
	loadEnvFile(t, "POSTGRES_DB_HOST=file\n"+
		"POSTGRES_DB_USER=file_user\n"+
		"POSTGRES_DB_PASSWORD=file_pass\n"+
		"POSTGRES_DB_NAME=file_db\n"+
		"POSTGRES_DB_PORT=5432\n"+
		"POSTGRES_DB_MAX_OPEN_CONNECTIONS=10\n"+
		"POSTGRES_DB_MAX_IDLE_CONNECTIONS=2\n"+
		"POSTGRES_DB_LOGGING=true\n"+
		"POSTGRES_DB_LOG_LEVEL=errors\n"+
		"POSTGRES_DB_THRESHOLD=0.2\n")

	cfg := &PostgresDbConfig{}
	if err := coreconfig.InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	want := PostgresDbConfig{
		Host:               "file",
		User:               "file_user",
		Password:           "file_pass",
		Db:                 "file_db",
		Port:               "5432",
		MaxOpenConnections: 10,
		MaxIdleConnections: 2,
		Logging:            true,
		DbLogLevel:         "errors",
		Threshold:          0.2,
	}
	if *cfg != want {
		t.Errorf("got %+v, want %+v", *cfg, want)
	}
}

// clearPostgresEnv Убирает POSTGRES_DB_* из окружения процесса на время теста.
func clearPostgresEnv(t *testing.T) {
	t.Helper()

	for _, key := range postgresEnvKeys {
		t.Setenv(key, "")

		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
}

// loadEnvFile Загружает .env с содержимым content; пустой content — файла нет, только окружение.
func loadEnvFile(t *testing.T, content string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".env")
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write env file: %v", err)
		}
	}

	if _, err := coreconfig.LoadEnv(path); err != nil {
		t.Fatalf("load env: %v", err)
	}
}
