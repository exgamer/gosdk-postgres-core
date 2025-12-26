package database

import (
	"fmt"
	config2 "github.com/exgamer/gosdk-core/pkg/config"
	database "github.com/exgamer/gosdk-db-core/pkg/helpers"
	"github.com/exgamer/gosdk-postgres-core/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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
		MaxOpenConnections: dbConfig.MaxOpenConnections,
		MaxIdleConnections: dbConfig.MaxIdleConnections,
		Threshold:          dbConfig.Threshold,
	}

	c.Dialector = postgres.Open(getPostgresConnectionString(c))

	dbClient, err := GetGormConnection(c)

	if err != nil {
		return nil, err
	}

	return dbClient, nil
}

// getPostgresConnectionString Возвращает строку (DSN) для создания соединения с Postgres
func getPostgresConnectionString(dbConfig *database.DbConfig) string {
	sslMode := "disable"

	if dbConfig.SslMode {
		sslMode = "enable"
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.Db, sslMode)
}
