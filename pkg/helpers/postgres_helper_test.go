package database

import (
	"strings"
	"testing"
	"time"

	database "github.com/exgamer/gosdk-db-core/pkg/helpers"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresDSN_PasswordAndSslMode(t *testing.T) {
	cases := []struct {
		name     string
		password string
		boolSSL  bool
		sslMode  string
		wantTLS  bool
		wantErr  bool
	}{
		{name: "простой пароль, по умолчанию prefer", password: "secret", wantTLS: true},
		{name: "пароль с пробелом", password: "my secret", wantTLS: true},
		{name: "пароль с кавычкой и обратной косой", password: `it's\a"b`, wantTLS: true},
		{name: "пароль с = и пробелом в конце", password: "a=b sslmode=require ", wantTLS: true},
		{name: "SslMode true — require (раньше было неверное enable)", password: "x", boolSSL: true, wantTLS: true},
		{name: "POSTGRES_DB_SSLMODE=verify-full", password: "x", sslMode: "verify-full", wantTLS: true},
		{name: "POSTGRES_DB_SSLMODE=disable важнее SslMode true", password: "x", boolSSL: true, sslMode: "disable"},
		{name: "неизвестный sslmode — ошибка", password: "x", sslMode: "enable", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dsn, err := postgresDSN(&database.DbConfig{
				Host: "db.local", Port: "5432", User: "app", Password: tc.password, Db: "app_db", SslMode: tc.boolSSL,
			}, tc.sslMode)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got DSN %q", dsn)
				}

				return
			}

			if err != nil {
				t.Fatalf("postgresDSN: %v", err)
			}

			cfg, err := pgconn.ParseConfig(dsn)
			if err != nil {
				t.Fatalf("pgconn.ParseConfig(%q): %v", dsn, err)
			}

			if cfg.Password != tc.password || cfg.User != "app" || cfg.Database != "app_db" || cfg.Host != "db.local" || cfg.Port != 5432 {
				t.Fatalf("parsed %+v from %q", cfg, dsn)
			}

			if (cfg.TLSConfig != nil) != tc.wantTLS {
				t.Fatalf("TLS = %v, want %v; DSN %q", cfg.TLSConfig != nil, tc.wantTLS, dsn)
			}
		})
	}
}

func TestPostgresDSN_EmptyPassword(t *testing.T) {
	t.Setenv("PGPASSWORD", "from-env")

	dsn, err := postgresDSN(&database.DbConfig{Host: "db.local", Port: "5432", User: "app", Db: "app_db"}, "")
	if err != nil {
		t.Fatalf("postgresDSN: %v", err)
	}

	cfg, err := pgconn.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("pgconn.ParseConfig(%q): %v", dsn, err)
	}

	// пустой пароль остаётся пустым (без подстановки PGPASSWORD/.pgpass), dbname — указанный
	if cfg.Password != "" || cfg.Database != "app_db" {
		t.Fatalf("password %q, database %q from %q", cfg.Password, cfg.Database, dsn)
	}
}

func TestPostgresDSN_DefaultPreferFallsBackToPlain(t *testing.T) {
	dsn, err := postgresDSN(&database.DbConfig{Host: "db.local", Port: "5432", User: "app", Password: "x", Db: "app_db"}, "")
	if err != nil {
		t.Fatalf("postgresDSN: %v", err)
	}

	cfg, err := pgconn.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("pgconn.ParseConfig(%q): %v", dsn, err)
	}

	// prefer: сначала TLS без проверки сертификата, затем без TLS
	if cfg.TLSConfig == nil || len(cfg.Fallbacks) == 0 || cfg.Fallbacks[len(cfg.Fallbacks)-1].TLSConfig != nil {
		t.Fatalf("TLS %v, fallbacks %+v from %q", cfg.TLSConfig != nil, cfg.Fallbacks, dsn)
	}
}

func TestPostgresDSN_ConnectTimeout(t *testing.T) {
	t.Setenv("PGCONNECT_TIMEOUT", "")

	dsn, err := postgresDSN(&database.DbConfig{Host: "db.local", Port: "5432", User: "app", Password: "x", Db: "app_db"}, "disable")
	if err != nil {
		t.Fatalf("postgresDSN: %v", err)
	}

	cfg, err := pgconn.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("pgconn.ParseConfig(%q): %v", dsn, err)
	}

	if cfg.ConnectTimeout != 5*time.Second {
		t.Fatalf("ConnectTimeout = %v, want 5s; DSN %q", cfg.ConnectTimeout, dsn)
	}

	t.Setenv("PGCONNECT_TIMEOUT", "12")

	dsn, err = postgresDSN(&database.DbConfig{Host: "db.local", Port: "5432", User: "app", Password: "x", Db: "app_db"}, "disable")
	if err != nil {
		t.Fatalf("postgresDSN: %v", err)
	}

	if strings.Contains(dsn, "connect_timeout") {
		t.Fatalf("PGCONNECT_TIMEOUT set, DSN has connect_timeout: %q", dsn)
	}

	if cfg, err = pgconn.ParseConfig(dsn); err != nil || cfg.ConnectTimeout != 12*time.Second {
		t.Fatalf("ConnectTimeout = %v (err %v), want 12s from PGCONNECT_TIMEOUT", cfg.ConnectTimeout, err)
	}
}
