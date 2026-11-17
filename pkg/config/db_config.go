package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PostgresDbConfig Данные для соединения с БД
type PostgresDbConfig struct {
	Host               string  `mapstructure:"POSTGRES_DB_HOST" json:"POSTGRES_DB_HOST"`
	User               string  `mapstructure:"POSTGRES_DB_USER" json:"POSTGRES_DB_USER"`
	Password           string  `mapstructure:"POSTGRES_DB_PASSWORD" json:"POSTGRES_DB_PASSWORD"`
	Db                 string  `mapstructure:"POSTGRES_DB_NAME" json:"POSTGRES_DB_NAME"`
	Port               string  `mapstructure:"POSTGRES_DB_PORT" json:"POSTGRES_DB_PORT"`
	MaxOpenConnections int     `mapstructure:"POSTGRES_DB_MAX_OPEN_CONNECTIONS" json:"POSTGRES_DB_MAX_OPEN_CONNECTIONS"`
	MaxIdleConnections int     `mapstructure:"POSTGRES_DB_MAX_IDLE_CONNECTIONS" json:"POSTGRES_DB_MAX_IDLE_CONNECTIONS"`
	Logging            bool    `mapstructure:"POSTGRES_DB_LOGGING" json:"POSTGRES_DB_LOGGING"`
	DbLogLevel         string  `mapstructure:"POSTGRES_DB_LOG_LEVEL" json:"POSTGRES_DB_LOG_LEVEL"`
	Threshold          float64 `mapstructure:"POSTGRES_DB_THRESHOLD" json:"POSTGRES_DB_THRESHOLD"`
	// SslMode Режим TLS libpq: disable, allow, prefer, require, verify-ca, verify-full; пусто — disable (как раньше)
	SslMode string `mapstructure:"POSTGRES_DB_SSLMODE" json:"POSTGRES_DB_SSLMODE"`
}

// postgresDbConfigAlias PostgresDbConfig без методов: вывод маски без рекурсии в String и MarshalJSON
type postgresDbConfigAlias PostgresDbConfig

// masked Копия конфига с паролем "***"; пустой пароль остаётся пустым
func (c PostgresDbConfig) masked() postgresDbConfigAlias {
	if c.Password != "" {
		c.Password = "***"
	}

	return postgresDbConfigAlias(c)
}

// String Конфиг для %v и %+v без пароля
func (c PostgresDbConfig) String() string {
	return fmt.Sprintf("%+v", c.masked())
}

// GoString Конфиг для %#v без пароля
func (c PostgresDbConfig) GoString() string {
	return strings.Replace(fmt.Sprintf("%#v", c.masked()), "postgresDbConfigAlias", "PostgresDbConfig", 1)
}

// MarshalJSON Конфиг в JSON без пароля
func (c PostgresDbConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.masked())
}
