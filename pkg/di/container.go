package di

import (
	"github.com/exgamer/gosdk-core/pkg/di"
	app2 "github.com/exgamer/gosdk-postgres-core/pkg/app"
	"github.com/exgamer/gosdk-postgres-core/pkg/config"
	"gorm.io/gorm"
)

// GetDefaultPostgresConnection возвращает основной connection postgres.
func GetDefaultPostgresConnection(c *di.Container) (*gorm.DB, error) {
	r, err := di.Resolve[*app2.PostgresGormRegistry](c)

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
func AddPostgresConnection(c *di.Container, name string, config *config.PostgresDbConfig) error {
	r, err := di.Resolve[*app2.PostgresGormRegistry](c)

	if err != nil {
		return err
	}

	r.Add(name, config)

	return nil
}

// GetPostgresConnection возвращает connection postgres.
func GetPostgresConnection(c *di.Container, name string) (*gorm.DB, error) {
	r, err := di.Resolve[*app2.PostgresGormRegistry](c)

	if err != nil {
		return nil, err
	}

	connection, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	return connection, nil
}
