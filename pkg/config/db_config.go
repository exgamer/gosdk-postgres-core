package config

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
}
