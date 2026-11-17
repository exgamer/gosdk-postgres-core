package database

import (
	"context"
	"fmt"
	"strings"
	"sync"

	config2 "github.com/exgamer/gosdk-core/pkg/config"
	"github.com/exgamer/gosdk-core/pkg/logger"
	database "github.com/exgamer/gosdk-db-core/pkg/helpers"
	"github.com/exgamer/gosdk-postgres-core/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// sslModes Значения sslmode, которые понимает libpq
var sslModes = map[string]bool{
	"disable": true, "allow": true, "prefer": true, "require": true, "verify-ca": true, "verify-full": true,
}

// unencryptedSslModes Значения sslmode, при которых соединение может быть без TLS; пусто — disable
var unencryptedSslModes = map[string]bool{"": true, "disable": true, "allow": true, "prefer": true}

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

	WarnSslMode(dbConfig)

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

// sslModeWarned Режим и окружение, о которых WARN уже был: старт и каждое подключение не повторяют его
var sslModeWarned sync.Map

// checkSslMode Режим sslMode без обязательного TLS при APP_ENV не local (пустой и неизвестный — не local): WARN, один
// раз на режим и окружение. Подключение не меняется: БД на том же хосте или в доверенной сети работает как раньше
func checkSslMode(sslMode, appEnv string) {
	mode := strings.ToLower(strings.TrimSpace(sslMode))
	if !unencryptedSslModes[mode] || strings.EqualFold(strings.TrimSpace(appEnv), "local") {
		return
	}

	if mode == "" {
		mode = "disable"
	}

	if _, warned := sslModeWarned.LoadOrStore(mode+"|"+appEnv, true); warned {
		return
	}

	logger.Warning(context.Background(), fmt.Sprintf("POSTGRES_DB_SSLMODE: %q on APP_ENV=%q, "+
		"the connection to postgres may be unencrypted; use require, verify-ca or verify-full", mode, appEnv))
}

// WarnSslMode WARN о POSTGRES_DB_SSLMODE без обязательного TLS при APP_ENV не local. PostgresKernel вызывает её при
// старте, InitPostgresGormConnection — при подключении; WARN — один раз
func WarnSslMode(dbConfig *config.PostgresDbConfig) {
	checkSslMode(dbConfig.SslMode, appEnv())
}

// appEnv APP_ENV из конфига gosdk-core (BaseConfig.AppEnv): окружение процесса и загруженный .env
func appEnv() string {
	baseConfig := &config2.BaseConfig{}
	if err := config2.InitConfig(baseConfig); err != nil {
		return ""
	}

	return baseConfig.AppEnv
}

// quoteDSNValue Значение keyword/value libpq: в одинарных кавычках, `\` и `'` — через `\`
func quoteDSNValue(v string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(v) + "'"
}
