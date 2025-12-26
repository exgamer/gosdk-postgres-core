package app

import (
	"github.com/exgamer/gosdk-core/pkg/app"
	"github.com/exgamer/gosdk-core/pkg/di"
	"github.com/exgamer/gosdk-postgres-core/pkg/config"
	"gorm.io/gorm"
)

// GetDefaultPostgresConnection возвращает основной connection postgres.
func GetDefaultPostgresConnection(a *app.App) (*gorm.DB, error) {
	r, err := di.Resolve[*PostgresGormRegistry](a.Container)

	if err != nil {
		return nil, err
	}

	connection, err := r.GetDefaultConnection()
	if err != nil {
		return nil, err
	}

	return connection, nil
}

// AddPostgresConnection добавить соединение с БД
func AddPostgresConnection(a *app.App, name string, config *config.PostgresDbConfig) error {
	r, err := di.Resolve[*PostgresGormRegistry](a.Container)

	if err != nil {
		return err
	}

	r.Add(name, config)

	return nil
}

// GetPostgresConnection возвращает connection postgres.
func GetPostgresConnection(a *app.App, name string) (*gorm.DB, error) {
	r, err := di.Resolve[*PostgresGormRegistry](a.Container)

	if err != nil {
		return nil, err
	}

	connection, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	return connection, nil
}
