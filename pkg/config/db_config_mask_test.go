package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	coreconfig "github.com/exgamer/gosdk-core/pkg/config"
)

func TestPostgresDbConfig_PasswordMasked(t *testing.T) {
	cfg := PostgresDbConfig{Host: "db.local", User: "app", Password: "s3cr3t-pass", Db: "main", SslMode: "require"}

	jsonValue, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	jsonPointer, err := json.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal pointer: %v", err)
	}

	outputs := map[string]string{
		"%v":           fmt.Sprintf("%v", cfg),
		"%+v":          fmt.Sprintf("%+v", cfg),
		"%#v":          fmt.Sprintf("%#v", cfg),
		"%s":           fmt.Sprintf("%s", cfg),
		"%v pointer":   fmt.Sprintf("%v", &cfg),
		"%+v pointer":  fmt.Sprintf("%+v", &cfg),
		"%#v pointer":  fmt.Sprintf("%#v", &cfg),
		"json":         string(jsonValue),
		"json pointer": string(jsonPointer),
	}

	for name, out := range outputs {
		if strings.Contains(out, "s3cr3t-pass") {
			t.Errorf("%s shows the password: %s", name, out)
		}

		if !strings.Contains(out, "***") || !strings.Contains(out, "db.local") {
			t.Errorf("%s = %s, want *** and other fields", name, out)
		}
	}

	if cfg.Password != "s3cr3t-pass" {
		t.Errorf("Password = %q, the value itself must not change", cfg.Password)
	}

	if got := fmt.Sprintf("%#v", cfg); !strings.HasPrefix(got, "config.PostgresDbConfig{") {
		t.Errorf("%%#v = %s, want config.PostgresDbConfig{...}", got)
	}

	var decoded map[string]any
	if err := json.Unmarshal(jsonValue, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded["POSTGRES_DB_PASSWORD"] != "***" || decoded["POSTGRES_DB_HOST"] != "db.local" {
		t.Errorf("json = %s, want the same keys with POSTGRES_DB_PASSWORD=***", jsonValue)
	}
}

func TestPostgresDbConfig_EmptyPasswordStaysEmpty(t *testing.T) {
	cfg := PostgresDbConfig{Host: "db.local"}

	out, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if strings.Contains(string(out), "***") || !strings.Contains(string(out), `"POSTGRES_DB_PASSWORD":""`) {
		t.Errorf("json = %s, want an empty password", out)
	}

	if s := fmt.Sprintf("%+v", cfg); strings.Contains(s, "***") {
		t.Errorf("%%+v = %s, want an empty password", s)
	}
}

// Загрузка через mapstructure/viper и окружение не меняется: пароль читается как есть, несмотря на MarshalJSON.
func TestInitConfig_PasswordLoaded(t *testing.T) {
	clearPostgresEnv(t)
	loadEnvFile(t, "POSTGRES_DB_PASSWORD=file_pass\n")

	cfg := &PostgresDbConfig{}
	if err := coreconfig.InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if cfg.Password != "file_pass" {
		t.Errorf("from .env: Password=%q, want file_pass", cfg.Password)
	}

	t.Setenv("POSTGRES_DB_PASSWORD", "env_pass")

	cfg = &PostgresDbConfig{}
	if err := coreconfig.InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if cfg.Password != "env_pass" {
		t.Errorf("from env: Password=%q, want env_pass", cfg.Password)
	}
}
