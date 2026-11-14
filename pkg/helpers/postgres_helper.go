package database

import (
	"fmt"
	"strings"

	config2 "github.com/exgamer/gosdk-core/pkg/config"
	database "github.com/exgamer/gosdk-db-core/pkg/helpers"
	"github.com/exgamer/gosdk-postgres-core/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// sslModes Значения sslmode, которые понимает libpq
var sslModes = map[string]bool{
	"disable": true, "allow": true, "prefer": true, "require": true, "verify-ca": true, "verify-full": true,
}

// InitPostgresDbConfig Инициализация конфига БД c переменок окружения
func InitPostgresDbConfig() (*config.PostgresDbConfig, error) {
	dbConfig := &config.PostgresDbConfig{}
	err := config2.InitConfig(dbConfig)

	if err != nil {
		return nil, err
	}

	return dbConfig, nil
}

// InitPostgresGormConnection инициализирует клиент для postgres
func InitPostgresGormConnection(dbConfig *config.PostgresDbConfig) (*gorm.DB, error) {
	c := &database.DbConfig{

		Host:               dbConfig.Host,
		User:               dbConfig.User,
		Password:           dbConfig.Password,
		Db:                 dbConfig.Db,
		Port:               dbConfig.Port,
		SslMode:            false,
		Logging:            dbConfig.Logging,
		DbLogLevel:         dbConfig.DbLogLevel,
		MaxOpenConnections: dbConfig.MaxOpenConnections,
		MaxIdleConnections: dbConfig.MaxIdleConnections,
		Threshold:          dbConfig.Threshold,
	}

	dsn, err := postgresDSN(c, dbConfig.SslMode)
	if err != nil {
		return nil, err
	}

	c.Dialector = postgres.Open(dsn)

	dbClient, err := database.GetGormConnection(c)

	if err != nil {
		return nil, err
	}

	return dbClient, nil
}

// postgresDSN DSN в формате keyword/value libpq: значения в одинарных кавычках, `\` и `'` экранированы —
// пароль с пробелом, кавычкой или обратной косой чертой доходит до сервера как есть. sslMode — POSTGRES_DB_SSLMODE;
// пусто — по DbConfig.SslMode (true — require, false — disable); неизвестное значение — ошибка
func postgresDSN(dbConfig *database.DbConfig, sslMode string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(sslMode))

	switch {
	case mode == "" && dbConfig.SslMode:
		mode = "require"
	case mode == "":
		mode = "disable"
	case !sslModes[mode]:
		return "", fmt.Errorf("POSTGRES_DB_SSLMODE: unknown value %q (disable, allow, prefer, require, verify-ca, verify-full)", sslMode)
	}

	parts := make([]string, 0, 6)

	for _, kv := range [][2]string{
		{"host", dbConfig.Host}, {"port", dbConfig.Port}, {"user", dbConfig.User},
		{"password", dbConfig.Password}, {"dbname", dbConfig.Db},
	} {
		// пустой пароль — явно password='': без ключа libpq взял бы PGPASSWORD или .pgpass
		if kv[1] == "" && kv[0] != "password" {
			continue
		}

		parts = append(parts, kv[0]+"="+quoteDSNValue(kv[1]))
	}

	parts = append(parts, "sslmode="+mode)

	return strings.Join(parts, " "), nil
}

// quoteDSNValue Значение keyword/value libpq: в одинарных кавычках, `\` и `'` — через `\`
func quoteDSNValue(v string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(v) + "'"
}
